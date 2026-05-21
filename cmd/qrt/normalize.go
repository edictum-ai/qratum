package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxTranscriptLineBytes = 10 << 20

type qratumSession struct {
	SchemaVersion             string                `json:"schema_version"`
	SessionID                 string                `json:"session_id"`
	Source                    string                `json:"source"`
	AgentModel                string                `json:"agent_model,omitempty"`
	WorkspaceID               string                `json:"workspace_id,omitempty"`
	RepoID                    string                `json:"repo_id,omitempty"`
	StartedAt                 string                `json:"started_at,omitempty"`
	EndedAt                   string                `json:"ended_at,omitempty"`
	Turns                     []qratumTurn          `json:"turns"`
	ToolCalls                 []qratumToolCall      `json:"tool_calls"`
	FileChanges               []qratumFileChange    `json:"file_changes"`
	Commands                  []qratumCommand       `json:"commands"`
	Git                       *qratumGitInfo        `json:"git,omitempty"`
	Workspace                 *captureWorkspaceRef  `json:"workspace,omitempty"`
	SourceEventID             string                `json:"source_event_id,omitempty"`
	SourceEventType           string                `json:"source_event_type,omitempty"`
	SourceEventTimestamp      string                `json:"source_event_timestamp,omitempty"`
	SourceTranscriptSessionID string                `json:"source_transcript_session_id,omitempty"`
	TranscriptPath            string                `json:"transcript_path,omitempty"`
	PipelineStatus            string                `json:"pipeline_status,omitempty"`
	ArtifactPaths             *daemonArtifactPaths  `json:"artifact_paths,omitempty"`
	BusinessMetrics           qratumBusinessMetrics `json:"business_metrics"`
	Provenance                map[string]any        `json:"provenance"`
}

type qratumTurn struct {
	Role      string `json:"role"`
	Timestamp string `json:"timestamp,omitempty"`
	Content   string `json:"content"`
}

type qratumToolCall struct {
	ToolCallID     string         `json:"tool_call_id"`
	Name           string         `json:"name"`
	Timestamp      string         `json:"timestamp,omitempty"`
	Input          map[string]any `json:"input"`
	Success        *bool          `json:"success,omitempty"`
	Result         string         `json:"result,omitempty"`
	ResultTime     string         `json:"result_timestamp,omitempty"`
	SourceID       string         `json:"source_id,omitempty"`
	ResultSourceID string         `json:"result_source_id,omitempty"`
}

type qratumFileChange struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	Timestamp string `json:"timestamp,omitempty"`
}

type qratumCommand struct {
	Command   string `json:"command"`
	Timestamp string `json:"timestamp,omitempty"`
	Success   *bool  `json:"success,omitempty"`
	Output    string `json:"output,omitempty"`
}

type qratumGitInfo struct {
	Remote  string `json:"remote,omitempty"`
	Branch  string `json:"branch,omitempty"`
	HeadSHA string `json:"head_sha,omitempty"`
}

type qratumBusinessMetrics struct {
	DurationSeconds int `json:"duration_seconds"`
	ToolCalls       int `json:"tool_calls"`
	FilesChanged    int `json:"files_changed"`
	CommandsRun     int `json:"commands_run"`
	TestsRun        int `json:"tests_run"`
}

type normalizeSessionContext struct {
	SessionID            string
	TranscriptPath       string
	Workspace            *captureWorkspaceRef
	SourceEventID        string
	SourceEventType      string
	SourceEventTimestamp string
	ArtifactPaths        *daemonArtifactPaths
	PipelineStatus       string
}

type claudeTranscriptParser struct {
	session             qratumSession
	transcriptSessionID string
	firstTimestamp      string
	lastTimestamp       string
	lineCount           int
	pendingByToolID     map[string]int
	pendingByName       map[string][]int
	commandByToolIndex  map[int]int
}

func normalize(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing transcript path")
		return 2
	}
	if len(args) != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: normalize accepts exactly one transcript path")
		return 2
	}

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve current project: %v\n", err)
		return 1
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: resolve current project absolute path: %v\n", err)
		return 1
	}

	transcriptPath, err := resolveTranscriptPath(projectRoot, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid transcript path: %v\n", err)
		return 1
	}
	if err := requireTranscriptFile(transcriptPath, projectRoot, args[0]); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	session, err := normalizeClaudeTranscriptFile(transcriptPath, normalizeSessionContext{})
	if err != nil {
		fmt.Fprintf(stderr, "error: normalize transcript %s: %v\n", displayPath(projectRoot, transcriptPath), err)
		return 1
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "error: encode normalized session: %v\n", err)
		return 1
	}
	_, _ = stdout.Write(append(data, '\n'))
	return 0
}

func normalizeClaudeTranscriptFile(path string, context normalizeSessionContext) (qratumSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return qratumSession{}, fmt.Errorf("read transcript: %w", err)
	}
	defer file.Close()

	return normalizeClaudeTranscript(file, context)
}

func normalizeClaudeTranscript(reader io.Reader, context normalizeSessionContext) (qratumSession, error) {
	parser := newClaudeTranscriptParser(context)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), maxTranscriptLineBytes)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := parser.parseLine(lineNo, line); err != nil {
			return qratumSession{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return qratumSession{}, fmt.Errorf("read transcript JSONL: %w", err)
	}

	return parser.finish()
}

func newClaudeTranscriptParser(context normalizeSessionContext) *claudeTranscriptParser {
	session := qratumSession{
		SchemaVersion:        qratumSessionSchemaVersion,
		SessionID:            strings.TrimSpace(context.SessionID),
		Source:               claudeCodeSource,
		Turns:                []qratumTurn{},
		ToolCalls:            []qratumToolCall{},
		FileChanges:          []qratumFileChange{},
		Commands:             []qratumCommand{},
		SourceEventID:        strings.TrimSpace(context.SourceEventID),
		SourceEventType:      strings.TrimSpace(context.SourceEventType),
		SourceEventTimestamp: strings.TrimSpace(context.SourceEventTimestamp),
		TranscriptPath:       strings.TrimSpace(context.TranscriptPath),
		PipelineStatus:       strings.TrimSpace(context.PipelineStatus),
		ArtifactPaths:        context.ArtifactPaths,
		Provenance:           map[string]any{},
	}
	if context.Workspace != nil {
		workspace := *context.Workspace
		workspace.CWD = strings.TrimSpace(workspace.CWD)
		if workspace.CWD != "" {
			session.Workspace = &workspace
		}
	}

	return &claudeTranscriptParser{
		session:            session,
		pendingByToolID:    map[string]int{},
		pendingByName:      map[string][]int{},
		commandByToolIndex: map[int]int{},
	}
}

func (p *claudeTranscriptParser) parseLine(lineNo int, line []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil {
		return fmt.Errorf("line %d: invalid JSON: %w", lineNo, err)
	}
	if len(fields) == 0 {
		return fmt.Errorf("line %d: empty transcript record", lineNo)
	}

	recordType, err := requiredStringField(fields, "type")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	timestamp, err := optionalTimestampField(fields, "timestamp")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	p.observeTimestamp(timestamp)
	p.lineCount++

	if err := p.captureCommonContext(fields, lineNo); err != nil {
		return err
	}

	switch recordType {
	case "session_start":
		return p.parseSessionStart(lineNo, fields, timestamp)
	case "session_end":
		return p.parseSessionEnd(lineNo, fields, timestamp)
	case "user", "assistant":
		return p.parseTurn(lineNo, fields, recordType, timestamp)
	case "tool_use":
		return p.parseToolUse(lineNo, fields, timestamp)
	case "tool_result":
		return p.parseToolResult(lineNo, fields, timestamp)
	default:
		return fmt.Errorf("line %d: unsupported transcript record type %q", lineNo, recordType)
	}
}

func (p *claudeTranscriptParser) parseSessionStart(lineNo int, fields map[string]json.RawMessage, timestamp string) error {
	if err := p.rememberTranscriptSessionID(lineNo, fields); err != nil {
		return err
	}
	if timestamp != "" {
		p.session.StartedAt = timestamp
	}
	model, err := optionalStringField(fields, "model")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if model != "" {
		p.session.AgentModel = model
	}
	return nil
}

func (p *claudeTranscriptParser) parseSessionEnd(lineNo int, fields map[string]json.RawMessage, timestamp string) error {
	if err := p.rememberTranscriptSessionID(lineNo, fields); err != nil {
		return err
	}
	if timestamp != "" {
		p.session.EndedAt = timestamp
	}
	return nil
}

func (p *claudeTranscriptParser) parseTurn(lineNo int, fields map[string]json.RawMessage, role string, timestamp string) error {
	content, err := contentField(fields, "content")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	p.session.Turns = append(p.session.Turns, qratumTurn{
		Role:      role,
		Timestamp: timestamp,
		Content:   content,
	})
	return nil
}

func (p *claudeTranscriptParser) parseToolUse(lineNo int, fields map[string]json.RawMessage, timestamp string) error {
	name, err := requiredStringField(fields, "name")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	input, err := objectField(fields, "input")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}

	sourceID, err := optionalFirstStringField(fields, "tool_call_id", "tool_use_id", "id")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	toolCallID := sourceID
	if toolCallID == "" {
		toolCallID = fmt.Sprintf("tool_%04d", len(p.session.ToolCalls)+1)
	}

	toolCall := qratumToolCall{
		ToolCallID: toolCallID,
		Name:       name,
		Timestamp:  timestamp,
		Input:      input,
	}
	if sourceID != "" && sourceID != toolCallID {
		toolCall.SourceID = sourceID
	}

	toolIndex := len(p.session.ToolCalls)
	p.session.ToolCalls = append(p.session.ToolCalls, toolCall)
	p.pendingByToolID[toolCallID] = toolIndex
	if sourceID != "" {
		p.pendingByToolID[sourceID] = toolIndex
	}
	p.pendingByName[name] = append(p.pendingByName[name], toolIndex)

	if op, ok := fileOperationForTool(name); ok {
		path, err := stringFromAny(input, "file_path", "path")
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
		if path == "" {
			return fmt.Errorf("line %d: %s tool input is missing file_path", lineNo, name)
		}
		p.session.FileChanges = append(p.session.FileChanges, qratumFileChange{
			Path:      path,
			Operation: op,
			Timestamp: timestamp,
		})
	}

	if strings.EqualFold(name, "Bash") {
		command, err := stringFromAny(input, "command")
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
		if command == "" {
			return fmt.Errorf("line %d: Bash tool input is missing command", lineNo)
		}
		commandIndex := len(p.session.Commands)
		p.session.Commands = append(p.session.Commands, qratumCommand{
			Command:   command,
			Timestamp: timestamp,
		})
		p.commandByToolIndex[toolIndex] = commandIndex
	}

	return nil
}

func (p *claudeTranscriptParser) parseToolResult(lineNo int, fields map[string]json.RawMessage, timestamp string) error {
	resultID, err := optionalFirstStringField(fields, "tool_call_id", "tool_use_id", "id")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	name, err := optionalStringField(fields, "name")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if resultID == "" && name == "" {
		return fmt.Errorf("line %d: tool_result is missing name or tool_use_id", lineNo)
	}

	toolIndex, ok := p.matchPendingToolResult(resultID, name)
	if !ok {
		if resultID != "" {
			return fmt.Errorf("line %d: tool_result for %q has no matching tool_use", lineNo, resultID)
		}
		return fmt.Errorf("line %d: tool_result for %q has no matching tool_use", lineNo, name)
	}

	success, err := optionalBoolField(fields, "success")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	result, err := contentField(fields, "content")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if result == "" {
		result, err = contentField(fields, "output")
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
	}

	p.session.ToolCalls[toolIndex].Success = success
	p.session.ToolCalls[toolIndex].Result = result
	p.session.ToolCalls[toolIndex].ResultTime = timestamp
	if resultID != "" && resultID != p.session.ToolCalls[toolIndex].ToolCallID {
		p.session.ToolCalls[toolIndex].ResultSourceID = resultID
	}

	if commandIndex, ok := p.commandByToolIndex[toolIndex]; ok {
		p.session.Commands[commandIndex].Success = success
		p.session.Commands[commandIndex].Output = result
	}

	return nil
}

func (p *claudeTranscriptParser) finish() (qratumSession, error) {
	if p.lineCount == 0 {
		return qratumSession{}, fmt.Errorf("transcript has no JSON records")
	}
	if p.session.SessionID == "" {
		p.session.SessionID = p.transcriptSessionID
	}
	if p.session.SessionID == "" {
		return qratumSession{}, fmt.Errorf("transcript is missing session_id")
	}
	if p.transcriptSessionID != "" && p.session.SessionID != p.transcriptSessionID {
		p.session.SourceTranscriptSessionID = p.transcriptSessionID
	}
	if p.session.StartedAt == "" {
		p.session.StartedAt = p.firstTimestamp
	}
	if p.session.EndedAt == "" {
		p.session.EndedAt = p.lastTimestamp
	}

	durationSeconds, err := durationSeconds(p.session.StartedAt, p.session.EndedAt)
	if err != nil {
		return qratumSession{}, err
	}
	p.session.BusinessMetrics = qratumBusinessMetrics{
		DurationSeconds: durationSeconds,
		ToolCalls:       len(p.session.ToolCalls),
		FilesChanged:    len(p.session.FileChanges),
		CommandsRun:     len(p.session.Commands),
		TestsRun:        countTestCommands(p.session.Commands),
	}

	return p.session, nil
}

func (p *claudeTranscriptParser) observeTimestamp(timestamp string) {
	if timestamp == "" {
		return
	}
	if p.firstTimestamp == "" {
		p.firstTimestamp = timestamp
	}
	p.lastTimestamp = timestamp
}

func (p *claudeTranscriptParser) rememberTranscriptSessionID(lineNo int, fields map[string]json.RawMessage) error {
	sessionID, err := optionalStringField(fields, "session_id")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if sessionID == "" {
		return nil
	}
	if p.transcriptSessionID != "" && p.transcriptSessionID != sessionID {
		return fmt.Errorf("line %d: conflicting transcript session_id %q, previously %q", lineNo, sessionID, p.transcriptSessionID)
	}
	p.transcriptSessionID = sessionID
	return nil
}

func (p *claudeTranscriptParser) captureCommonContext(fields map[string]json.RawMessage, lineNo int) error {
	workspaceID, err := optionalStringField(fields, "workspace_id")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if workspaceID != "" {
		p.session.WorkspaceID = workspaceID
	}
	repoID, err := optionalStringField(fields, "repo_id")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if repoID != "" {
		p.session.RepoID = repoID
	}
	if err := p.captureWorkspace(fields, lineNo); err != nil {
		return err
	}
	return p.captureGit(fields, lineNo)
}

func (p *claudeTranscriptParser) captureWorkspace(fields map[string]json.RawMessage, lineNo int) error {
	if p.session.Workspace != nil && p.session.Workspace.CWD != "" {
		return nil
	}
	cwd, err := optionalStringField(fields, "cwd")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if cwd != "" {
		p.session.Workspace = &captureWorkspaceRef{CWD: cwd}
		return nil
	}

	raw, ok := fields["workspace"]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var workspace captureWorkspaceRef
	if err := json.Unmarshal(raw, &workspace); err != nil {
		return fmt.Errorf("line %d: workspace must be an object: %w", lineNo, err)
	}
	workspace.CWD = strings.TrimSpace(workspace.CWD)
	if workspace.CWD != "" {
		p.session.Workspace = &workspace
	}
	return nil
}

func (p *claudeTranscriptParser) captureGit(fields map[string]json.RawMessage, lineNo int) error {
	var git qratumGitInfo
	if p.session.Git != nil {
		git = *p.session.Git
	}

	raw, ok := fields["git"]
	if ok && len(bytes.TrimSpace(raw)) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		var incoming qratumGitInfo
		if err := json.Unmarshal(raw, &incoming); err != nil {
			return fmt.Errorf("line %d: git must be an object: %w", lineNo, err)
		}
		if git.Remote == "" {
			git.Remote = strings.TrimSpace(incoming.Remote)
		}
		if git.Branch == "" {
			git.Branch = strings.TrimSpace(incoming.Branch)
		}
		if git.HeadSHA == "" {
			git.HeadSHA = strings.TrimSpace(incoming.HeadSHA)
		}
	}

	remote, err := optionalStringField(fields, "git_remote")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	branch, err := optionalStringField(fields, "git_branch")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	headSHA, err := optionalFirstStringField(fields, "git_head_sha", "head_sha")
	if err != nil {
		return fmt.Errorf("line %d: %w", lineNo, err)
	}
	if git.Remote == "" {
		git.Remote = remote
	}
	if git.Branch == "" {
		git.Branch = branch
	}
	if git.HeadSHA == "" {
		git.HeadSHA = headSHA
	}
	if git.Remote != "" || git.Branch != "" || git.HeadSHA != "" {
		p.session.Git = &git
	}
	return nil
}

func (p *claudeTranscriptParser) matchPendingToolResult(resultID string, name string) (int, bool) {
	if resultID != "" {
		if index, ok := p.pendingByToolID[resultID]; ok {
			delete(p.pendingByToolID, resultID)
			p.removePendingName(index)
			return index, true
		}
	}
	if name == "" {
		return 0, false
	}
	queue := p.pendingByName[name]
	if len(queue) == 0 {
		return 0, false
	}
	index := queue[0]
	if len(queue) == 1 {
		delete(p.pendingByName, name)
	} else {
		p.pendingByName[name] = queue[1:]
	}
	delete(p.pendingByToolID, p.session.ToolCalls[index].ToolCallID)
	return index, true
}

func (p *claudeTranscriptParser) removePendingName(index int) {
	name := p.session.ToolCalls[index].Name
	queue := p.pendingByName[name]
	for i, queued := range queue {
		if queued == index {
			queue = append(queue[:i], queue[i+1:]...)
			break
		}
	}
	if len(queue) == 0 {
		delete(p.pendingByName, name)
		return
	}
	p.pendingByName[name] = queue
}

func requiredStringField(fields map[string]json.RawMessage, name string) (string, error) {
	value, err := optionalStringField(fields, name)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("missing required field %s", name)
	}
	return value, nil
}

func optionalFirstStringField(fields map[string]json.RawMessage, names ...string) (string, error) {
	for _, name := range names {
		value, err := optionalStringField(fields, name)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
	}
	return "", nil
}

func optionalStringField(fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", name)
	}
	return strings.TrimSpace(value), nil
}

func optionalTimestampField(fields map[string]json.RawMessage, name string) (string, error) {
	value, err := optionalStringField(fields, name)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return "", fmt.Errorf("%s must be RFC3339: %w", name, err)
	}
	return value, nil
}

func optionalBoolField(fields map[string]json.RawMessage, name string) (*bool, error) {
	raw, ok := fields[name]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a boolean", name)
	}
	return &value, nil
}

func objectField(fields map[string]json.RawMessage, name string) (map[string]any, error) {
	raw, ok := fields[name]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}

func contentField(fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", nil
	}
	return contentFromRaw(raw)
}

func contentFromRaw(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, block := range blocks {
			if block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
		return strings.Join(parts, "\n"), nil
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return "", fmt.Errorf("content must be valid JSON")
	}
	return compact.String(), nil
}

func stringFromAny(input map[string]any, names ...string) (string, error) {
	for _, name := range names {
		value, ok := input[name]
		if !ok || value == nil {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("input.%s must be a string", name)
		}
		return strings.TrimSpace(text), nil
	}
	return "", nil
}

func fileOperationForTool(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "write":
		return "write", true
	case "edit", "multiedit", "notebookedit":
		return "edit", true
	default:
		return "", false
	}
}

func durationSeconds(startedAt string, endedAt string) (int, error) {
	if startedAt == "" || endedAt == "" {
		return 0, nil
	}
	started, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return 0, fmt.Errorf("started_at must be RFC3339: %w", err)
	}
	ended, err := time.Parse(time.RFC3339, endedAt)
	if err != nil {
		return 0, fmt.Errorf("ended_at must be RFC3339: %w", err)
	}
	if ended.Before(started) {
		return 0, fmt.Errorf("ended_at %q is before started_at %q", endedAt, startedAt)
	}
	return int(ended.Sub(started).Seconds()), nil
}

func countTestCommands(commands []qratumCommand) int {
	count := 0
	for _, command := range commands {
		if strings.Contains(strings.ToLower(command.Command), "test") {
			count++
		}
	}
	return count
}

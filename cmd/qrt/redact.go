package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	qschema "github.com/acartag7/qratum/internal/schema"
	"github.com/acartag7/qratum/internal/workspace"
)

const (
	redactionStatus               = "redacted"
	qratumRedactionSummaryVersion = "qratum.redaction_summary.v1"
)

var (
	privateKeyBlockPattern    = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	credentialURLPattern      = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s\"'<>@:/]+:[^\s\"'<>@]+@[^\s\"'<>]+`)
	jwtLikePattern            = regexp.MustCompile(`\b[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{6,}\b`)
	apiKeyPattern             = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9][A-Za-z0-9_-]{20,}|(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{20,}|xox[baprs]-[A-Za-z0-9-]{20,})\b`)
	sshRemotePattern          = regexp.MustCompile(`\bgit@[A-Za-z0-9._-]+:[A-Za-z0-9._~/%+-]+(?:\.git)?\b`)
	secretAssignmentPattern   = regexp.MustCompile(`(?i)\b((?:[A-Z0-9_]*(?:API[_-]?KEY|ACCESS[_-]?TOKEN|AUTH[_-]?TOKEN|TOKEN|SECRET|PASSWORD|PASSWD|JWT)[A-Z0-9_]*)\s*(?:[:=]\s*)+(?:>\s*)?)([^\s\"',;]+)`)
	secretSpacePattern        = regexp.MustCompile(`(?i)\b((?:API[_-]?KEY|ACCESS[_-]?TOKEN|AUTH[_-]?TOKEN|TOKEN|PASSWORD|PASSWD|JWT)\s+)([^\s\"',;]+)`)
	highEntropyPattern        = regexp.MustCompile(`\b[A-Za-z0-9+/=_-]{32,}\b`)
	posixLocalPathPattern     = regexp.MustCompile(`/(?:Users|home|tmp|private|var|opt|Volumes)(?:/[A-Za-z0-9._@%+=:,~-]+)+`)
	relativeSecretPathPattern = regexp.MustCompile(`(?:~|\.)/[A-Za-z0-9._@%+=:,\-/]*(?:\.env|credentials?|secrets?|secret|token|key)[A-Za-z0-9._@%+=:,\-/]*`)
	windowsLocalPathPattern   = regexp.MustCompile(`[A-Za-z]:\\(?:Users|Temp|tmp|Windows|ProgramData)(?:\\[^\s\"'<>|;]+)+`)
	uuidPattern               = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

type qratumRedactionSummary struct {
	SchemaVersion      string                   `json:"schema_version"`
	DataClass          string                   `json:"data_class"`
	Status             string                   `json:"status"`
	Findings           []qratumRedactionFinding `json:"findings"`
	SecretPlaceholders int                      `json:"secret_placeholders"`
	PathPlaceholders   int                      `json:"path_placeholders"`
}

type qratumRedactionFinding struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Field            string   `json:"field"`
	ReplacementCount int      `json:"replacement_count"`
	Placeholders     []string `json:"placeholders"`
}

type deterministicRedactor struct {
	secretPlaceholders map[string]string
	pathPlaceholders   map[string]string
	secretOrder        []string
	pathOrder          []string
	findings           []qratumRedactionFinding
}

type fieldRedactionTracker struct {
	secretCount        int
	pathCount          int
	secretPlaceholders map[string]struct{}
	pathPlaceholders   map[string]struct{}
}

func redact(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing session path")
		return 2
	}
	if len(args) != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: redact accepts exactly one session path")
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

	sessionPath, err := resolveProjectFilePath(projectRoot, args[0], "session")
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid session path: %v\n", err)
		return 1
	}
	session, err := readQratumSessionFile(sessionPath, projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	redacted, err := redactQratumSession(session)
	if err != nil {
		fmt.Fprintf(stderr, "error: redact session %s: %v\n", displayPath(projectRoot, sessionPath), err)
		return 1
	}
	data, err := json.MarshalIndent(redacted, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "error: encode redacted session: %v\n", err)
		return 1
	}
	_, _ = stdout.Write(append(data, '\n'))
	return 0
}

func readQratumSessionFile(path string, projectRoot string) (qratumSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return qratumSession{}, fmt.Errorf("read session %s: %w", displayPath(projectRoot, path), err)
	}
	var session qratumSession
	if err := json.Unmarshal(data, &session); err != nil {
		return qratumSession{}, fmt.Errorf("invalid session JSON %s: %w", displayPath(projectRoot, path), err)
	}
	if err := validateQratumSession(session, displayPath(projectRoot, path)); err != nil {
		return qratumSession{}, err
	}
	return session, nil
}

func validateQratumSession(session qratumSession, label string) error {
	if strings.TrimSpace(session.SchemaVersion) != qratumSessionSchemaVersion {
		return fmt.Errorf("session %s has unsupported schema_version %q", label, session.SchemaVersion)
	}
	if strings.TrimSpace(session.SessionID) == "" {
		return fmt.Errorf("session %s is missing session_id", label)
	}
	if strings.TrimSpace(session.Source) != claudeCodeSource {
		return fmt.Errorf("session %s has unsupported source %q", label, session.Source)
	}
	if session.Turns == nil {
		return fmt.Errorf("session %s is missing turns", label)
	}
	if session.ToolCalls == nil {
		return fmt.Errorf("session %s is missing tool_calls", label)
	}
	if session.FileChanges == nil {
		return fmt.Errorf("session %s is missing file_changes", label)
	}
	if session.Commands == nil {
		return fmt.Errorf("session %s is missing commands", label)
	}
	for i, turn := range session.Turns {
		switch turn.Role {
		case "user", "assistant":
		default:
			return fmt.Errorf("session %s has unsupported turns[%d].role %q", label, i, turn.Role)
		}
	}
	for i, toolCall := range session.ToolCalls {
		if strings.TrimSpace(toolCall.ToolCallID) == "" {
			return fmt.Errorf("session %s is missing tool_calls[%d].tool_call_id", label, i)
		}
		if strings.TrimSpace(toolCall.Name) == "" {
			return fmt.Errorf("session %s is missing tool_calls[%d].name", label, i)
		}
		if toolCall.Input == nil {
			return fmt.Errorf("session %s is missing tool_calls[%d].input", label, i)
		}
	}
	return nil
}

func resolveProjectFilePath(projectRoot string, inputPath string, label string) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", fmt.Errorf("missing %s path", label)
	}

	insideRoot := func(root string, path string) (bool, error) {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return false, err
		}
		return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel), nil
	}

	var candidates []string
	if filepath.IsAbs(inputPath) {
		resolved := filepath.Clean(inputPath)
		insideProject, err := insideRoot(projectRoot, resolved)
		if err != nil {
			return "", fmt.Errorf("resolve %s path: %w", label, err)
		}
		insideQratumHome := false
		if !insideProject {
			qratumHome, homeErr := workspace.Resolve()
			if homeErr != nil {
				return "", homeErr
			}
			insideQratumHome, err = insideRoot(qratumHome.Root, resolved)
			if err != nil {
				return "", fmt.Errorf("resolve %s path: %w", label, err)
			}
		}
		if !insideProject && !insideQratumHome {
			return "", fmt.Errorf("%s path %q escapes current project or qratum home", label, inputPath)
		}
		candidates = append(candidates, resolved)
	} else {
		projectCandidate := filepath.Clean(filepath.Join(projectRoot, inputPath))
		insideProject, err := insideRoot(projectRoot, projectCandidate)
		if err != nil {
			return "", fmt.Errorf("resolve %s path: %w", label, err)
		}
		if !insideProject {
			return "", fmt.Errorf("%s path %q escapes current project", label, inputPath)
		}
		if info, err := os.Stat(projectCandidate); err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("%s %s is a directory", label, displayPath(projectRoot, projectCandidate))
			}
			return projectCandidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect %s %s: %w", label, displayPath(projectRoot, projectCandidate), err)
		}
		candidates = append(candidates, projectCandidate)

		if shouldResolveQratumHomeCandidate(inputPath) {
			qratumHome, homeErr := workspace.Resolve()
			if homeErr != nil {
				return "", homeErr
			}
			homeCandidate := filepath.Clean(filepath.Join(qratumHome.Root, filepath.FromSlash(inputPath)))
			insideQratumHome, err := insideRoot(qratumHome.Root, homeCandidate)
			if err != nil {
				return "", fmt.Errorf("resolve %s path: %w", label, err)
			}
			if insideQratumHome {
				candidates = append(candidates, homeCandidate)
			}
		}
	}

	var firstMissing string
	for _, resolved := range candidates {
		info, err := os.Stat(resolved)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("%s %s is a directory", label, displayPath(projectRoot, resolved))
			}
			return resolved, nil
		}
		if os.IsNotExist(err) {
			if firstMissing == "" {
				firstMissing = resolved
			}
			continue
		}
		return "", fmt.Errorf("inspect %s %s: %w", label, displayPath(projectRoot, resolved), err)
	}
	if firstMissing != "" {
		return "", fmt.Errorf("missing %s %s", label, displayPath(projectRoot, firstMissing))
	}
	return "", fmt.Errorf("missing %s path", label)
}

func shouldResolveQratumHomeCandidate(inputPath string) bool {
	slash := filepath.ToSlash(strings.TrimSpace(inputPath))
	return strings.HasPrefix(slash, "sessions/") || strings.HasPrefix(slash, "events/")
}

func redactQratumSession(session qratumSession) (qratumSession, error) {
	if err := validateQratumSession(session, session.SessionID); err != nil {
		return qratumSession{}, err
	}
	redactor := newDeterministicRedactor()
	redacted := session
	redacted.DataClass = qschema.DataClassRedacted
	redacted.PipelineStatus = redactionStatus
	redacted.Redaction = nil

	redacted.AgentModel = redactor.redactString("agent_model", redacted.AgentModel)
	redacted.WorkspaceID = redactor.redactString("workspace_id", redacted.WorkspaceID)
	redacted.RepoID = redactor.redactString("repo_id", redacted.RepoID)
	redacted.TranscriptPath = redactor.redactString("transcript_path", redacted.TranscriptPath)
	redacted.SourceTranscriptSessionID = redactor.redactString("source_transcript_session_id", redacted.SourceTranscriptSessionID)
	if redacted.Workspace != nil {
		workspace := *redacted.Workspace
		workspace.CWD = redactor.redactString("workspace.cwd", workspace.CWD)
		redacted.Workspace = &workspace
	}

	redacted.Turns = make([]qratumTurn, len(session.Turns))
	for i, turn := range session.Turns {
		redacted.Turns[i] = turn
		redacted.Turns[i].Content = redactor.redactString(fmt.Sprintf("turns[%d].content", i), turn.Content)
	}

	redacted.ToolCalls = make([]qratumToolCall, len(session.ToolCalls))
	for i, toolCall := range session.ToolCalls {
		redacted.ToolCalls[i] = toolCall
		input, err := redactAny(redactor, fmt.Sprintf("tool_calls[%d].input", i), toolCall.Input)
		if err != nil {
			return qratumSession{}, err
		}
		redactedInput, ok := input.(map[string]any)
		if !ok {
			return qratumSession{}, fmt.Errorf("tool_calls[%d].input redacted to %T, want object", i, input)
		}
		redacted.ToolCalls[i].Input = redactedInput
		redacted.ToolCalls[i].Result = redactor.redactString(fmt.Sprintf("tool_calls[%d].result", i), toolCall.Result)
	}

	redacted.FileChanges = make([]qratumFileChange, len(session.FileChanges))
	for i, fileChange := range session.FileChanges {
		redacted.FileChanges[i] = fileChange
		redacted.FileChanges[i].Path = redactor.redactString(fmt.Sprintf("file_changes[%d].path", i), fileChange.Path)
	}

	redacted.Commands = make([]qratumCommand, len(session.Commands))
	for i, command := range session.Commands {
		redacted.Commands[i] = command
		redacted.Commands[i].Command = redactor.redactString(fmt.Sprintf("commands[%d].command", i), command.Command)
		redacted.Commands[i].Output = redactor.redactString(fmt.Sprintf("commands[%d].output", i), command.Output)
	}

	redacted.StartedAt = redactor.redactSensitiveMetadata("started_at", redacted.StartedAt)
	redacted.EndedAt = redactor.redactSensitiveMetadata("ended_at", redacted.EndedAt)
	redacted.SourceEventID = redactor.redactSensitiveMetadata("source_event_id", redacted.SourceEventID)

	if session.Git != nil {
		git := *session.Git
		git.Remote = redactor.redactSensitiveMetadata("git.remote", git.Remote)
		git.Branch = redactor.redactSensitiveMetadata("git.branch", git.Branch)
		git.HeadSHA = redactor.redactSensitiveMetadata("git.head_sha", git.HeadSHA)
		redacted.Git = &git
	}
	if session.Provenance != nil {
		provenance, err := redactAny(redactor, "provenance", session.Provenance)
		if err != nil {
			return qratumSession{}, err
		}
		redactedProvenance, ok := provenance.(map[string]any)
		if !ok {
			return qratumSession{}, fmt.Errorf("provenance redacted to %T, want object", provenance)
		}
		redacted.Provenance = redactedProvenance
	}

	summary := redactor.summary()
	redacted.Redaction = &summary
	return redacted, nil
}

func newDeterministicRedactor() *deterministicRedactor {
	return &deterministicRedactor{
		secretPlaceholders: map[string]string{},
		pathPlaceholders:   map[string]string{},
	}
}

func redactAny(redactor *deterministicRedactor, field string, value any) (any, error) {
	switch v := value.(type) {
	case string:
		return redactor.redactString(field, v), nil
	case map[string]any:
		out := make(map[string]any, len(v))
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			keyField := fmt.Sprintf("%s.[key_%03d]", field, i+1)
			redactedKey := redactor.redactString(keyField, key)
			if _, exists := out[redactedKey]; exists {
				return nil, fmt.Errorf("%s redacts to duplicate key %q", keyField, redactedKey)
			}
			childField := redactionMapChildField(field, key, redactedKey, i)
			redactedValue, err := redactAny(redactor, childField, v[key])
			if err != nil {
				return nil, err
			}
			out[redactedKey] = redactedValue
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			redactedValue, err := redactAny(redactor, fmt.Sprintf("%s[%d]", field, i), item)
			if err != nil {
				return nil, err
			}
			out[i] = redactedValue
		}
		return out, nil
	case nil, bool, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v, nil
	default:
		return nil, fmt.Errorf("%s has unsupported redaction value type %T", field, value)
	}
}

func redactionMapChildField(field string, key string, redactedKey string, index int) string {
	if key == redactedKey && isSafeRedactionFieldKey(key) {
		return field + "." + key
	}
	return fmt.Sprintf("%s.[key_%03d]", field, index+1)
}

func isSafeRedactionFieldKey(key string) bool {
	if key == "" || len(key) > 80 {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func (r *deterministicRedactor) redactString(field string, value string) string {
	if value == "" {
		return value
	}
	tracker := fieldRedactionTracker{
		secretPlaceholders: map[string]struct{}{},
		pathPlaceholders:   map[string]struct{}{},
	}

	redacted := value
	redacted = r.replacePattern(redacted, privateKeyBlockPattern, "secret", &tracker)
	redacted = r.replacePattern(redacted, credentialURLPattern, "secret", &tracker)
	redacted = r.replacePattern(redacted, jwtLikePattern, "secret", &tracker)
	redacted = r.replacePattern(redacted, apiKeyPattern, "secret", &tracker)
	redacted = r.replacePattern(redacted, sshRemotePattern, "secret", &tracker)
	redacted = r.replaceSecretAssignments(redacted, &tracker)
	redacted = r.replaceSecretSpaceAssignments(redacted, &tracker)
	redacted = r.replaceHighEntropy(redacted, &tracker)
	redacted = r.replacePattern(redacted, posixLocalPathPattern, "path", &tracker)
	redacted = r.replacePattern(redacted, relativeSecretPathPattern, "path", &tracker)
	redacted = r.replacePattern(redacted, windowsLocalPathPattern, "path", &tracker)

	r.recordFindings(field, tracker)
	return redacted
}

func (r *deterministicRedactor) redactSensitiveMetadata(field string, value string) string {
	redacted := r.redactString(field, value)
	if redacted != value || isAlreadyRedactedCandidate(value) || strings.TrimSpace(value) == "" {
		return redacted
	}
	tracker := fieldRedactionTracker{
		secretPlaceholders: map[string]struct{}{},
		pathPlaceholders:   map[string]struct{}{},
	}
	placeholder := r.placeholder("secret", value)
	tracker.add("secret", placeholder)
	r.recordFindings(field, tracker)
	return placeholder
}

func (r *deterministicRedactor) replacePattern(value string, pattern *regexp.Regexp, kind string, tracker *fieldRedactionTracker) string {
	return pattern.ReplaceAllStringFunc(value, func(match string) string {
		candidate, suffix := splitTrailingPunctuation(match)
		if shouldSkipCandidate(candidate) {
			return match
		}
		placeholder := r.placeholder(kind, candidate)
		tracker.add(kind, placeholder)
		return placeholder + suffix
	})
}

func (r *deterministicRedactor) replaceSecretAssignments(value string, tracker *fieldRedactionTracker) string {
	return r.replaceSecretPattern(value, secretAssignmentPattern, tracker)
}

func (r *deterministicRedactor) replaceSecretSpaceAssignments(value string, tracker *fieldRedactionTracker) string {
	return r.replaceSecretPattern(value, secretSpacePattern, tracker)
}

func (r *deterministicRedactor) replaceSecretPattern(value string, pattern *regexp.Regexp, tracker *fieldRedactionTracker) string {
	matches := pattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		valueStart, valueEnd := match[4], match[5]
		if valueStart < 0 || valueEnd < 0 {
			continue
		}
		secretValue := value[valueStart:valueEnd]
		if shouldSkipCandidate(secretValue) {
			continue
		}
		b.WriteString(value[last:valueStart])
		placeholder := r.placeholder("secret", secretValue)
		tracker.add("secret", placeholder)
		b.WriteString(placeholder)
		last = valueEnd
	}
	b.WriteString(value[last:])
	return b.String()
}

func (r *deterministicRedactor) replaceHighEntropy(value string, tracker *fieldRedactionTracker) string {
	return highEntropyPattern.ReplaceAllStringFunc(value, func(match string) string {
		if shouldSkipCandidate(match) || !looksHighEntropy(match) {
			return match
		}
		placeholder := r.placeholder("secret", match)
		tracker.add("secret", placeholder)
		return placeholder
	})
}

func (r *deterministicRedactor) placeholder(kind string, raw string) string {
	switch kind {
	case "path":
		if placeholder, ok := r.pathPlaceholders[raw]; ok {
			return placeholder
		}
		placeholder := fmt.Sprintf("[REDACTED_PATH_%03d]", len(r.pathOrder)+1)
		r.pathPlaceholders[raw] = placeholder
		r.pathOrder = append(r.pathOrder, raw)
		return placeholder
	default:
		if placeholder, ok := r.secretPlaceholders[raw]; ok {
			return placeholder
		}
		placeholder := fmt.Sprintf("[REDACTED_SECRET_%03d]", len(r.secretOrder)+1)
		r.secretPlaceholders[raw] = placeholder
		r.secretOrder = append(r.secretOrder, raw)
		return placeholder
	}
}

func (t *fieldRedactionTracker) add(kind string, placeholder string) {
	if kind == "path" {
		t.pathCount++
		t.pathPlaceholders[placeholder] = struct{}{}
		return
	}
	t.secretCount++
	t.secretPlaceholders[placeholder] = struct{}{}
}

func (r *deterministicRedactor) recordFindings(field string, tracker fieldRedactionTracker) {
	if tracker.secretCount > 0 {
		r.findings = append(r.findings, qratumRedactionFinding{
			ID:               fmt.Sprintf("redaction.secret_detected.%04d", len(r.findings)+1),
			Type:             "redaction.secret_detected",
			Field:            field,
			ReplacementCount: tracker.secretCount,
			Placeholders:     sortedPlaceholderKeys(tracker.secretPlaceholders),
		})
	}
	if tracker.pathCount > 0 {
		r.findings = append(r.findings, qratumRedactionFinding{
			ID:               fmt.Sprintf("redaction.path_redacted.%04d", len(r.findings)+1),
			Type:             "redaction.path_redacted",
			Field:            field,
			ReplacementCount: tracker.pathCount,
			Placeholders:     sortedPlaceholderKeys(tracker.pathPlaceholders),
		})
	}
}

func (r *deterministicRedactor) summary() qratumRedactionSummary {
	findings := make([]qratumRedactionFinding, len(r.findings))
	copy(findings, r.findings)
	return qratumRedactionSummary{
		SchemaVersion:      qratumRedactionSummaryVersion,
		DataClass:          qschema.DataClassRedacted,
		Status:             redactionStatus,
		Findings:           findings,
		SecretPlaceholders: len(r.secretOrder),
		PathPlaceholders:   len(r.pathOrder),
	}
}

func sortedPlaceholderKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func shouldSkipCandidate(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return true
	}
	return isAlreadyRedactedCandidate(candidate) ||
		isHexDigestCandidate(candidate) ||
		uuidPattern.MatchString(candidate)
}

func isAlreadyRedactedCandidate(candidate string) bool {
	return strings.Contains(candidate, "[REDACTED_") ||
		strings.Contains(candidate, "REDACTED_SECRET_") ||
		strings.Contains(candidate, "REDACTED_PATH_")
}

func isHexDigestCandidate(candidate string) bool {
	switch len(candidate) {
	case 40, 64:
	default:
		return false
	}
	for _, r := range candidate {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func splitTrailingPunctuation(candidate string) (string, string) {
	trimmed := strings.TrimRight(candidate, ".,;:!?")
	return trimmed, candidate[len(trimmed):]
}

func looksHighEntropy(candidate string) bool {
	if len(candidate) < 32 {
		return false
	}
	if strings.Contains(candidate, "\\") {
		return false
	}
	if strings.Count(candidate, "/") >= 2 || strings.Contains(candidate, "/.") || strings.Contains(candidate, "./") {
		return false
	}
	classes := 0
	for _, test := range []func(rune) bool{isLower, isUpper, isDigit, isEntropySymbol} {
		for _, r := range candidate {
			if test(r) {
				classes++
				break
			}
		}
	}
	if classes < 2 {
		return false
	}
	return shannonEntropy(candidate) >= 3.5
}

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }
func isEntropySymbol(r rune) bool {
	switch r {
	case '+', '/', '=', '_', '-':
		return true
	default:
		return false
	}
}

func shannonEntropy(value string) float64 {
	counts := map[rune]int{}
	for _, r := range value {
		counts[r]++
	}
	length := float64(len([]rune(value)))
	var entropy float64
	for _, count := range counts {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}
	return entropy
}

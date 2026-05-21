package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	qratumSessionSchemaVersion  = "qratum.session.v1"
	qratumEvidenceSchemaVersion = "qratum.evidence.v1"
	qratumReviewSchemaVersion   = "qratum.review_card.v1"
	qratumAPIErrorSchemaVersion = "qratum.ui.api_error.v1"
)

type daemonRunSummary struct {
	Events    int
	Processed int
	Skipped   int
}

type daemonArtifactPaths struct {
	Event    string `json:"event"`
	Session  string `json:"session"`
	Redacted string `json:"redacted"`
	Evidence string `json:"evidence"`
	Review   string `json:"review"`
	Report   string `json:"report"`
	Export   string `json:"export"`
}

type daemonArtifactFile struct {
	rel string
	abs string
}

type pipelineSessionPlaceholder struct {
	SchemaVersion        string               `json:"schema_version"`
	SessionID            string               `json:"session_id"`
	Source               string               `json:"source"`
	Turns                []any                `json:"turns"`
	ToolCalls            []any                `json:"tool_calls"`
	FileChanges          []any                `json:"file_changes"`
	Commands             []any                `json:"commands"`
	SourceEventID        string               `json:"source_event_id"`
	SourceEventType      string               `json:"source_event_type"`
	SourceEventTimestamp string               `json:"source_event_timestamp"`
	TranscriptPath       string               `json:"transcript_path,omitempty"`
	Workspace            *captureWorkspaceRef `json:"workspace,omitempty"`
	PipelineStatus       string               `json:"pipeline_status"`
	ArtifactPaths        daemonArtifactPaths  `json:"artifact_paths"`
}

type evidencePlaceholder struct {
	SchemaVersion   string              `json:"schema_version"`
	SessionID       string              `json:"session_id"`
	Summary         evidenceSummary     `json:"summary"`
	Findings        []any               `json:"findings"`
	MissingEvidence []string            `json:"missing_evidence"`
	SourceEventID   string              `json:"source_event_id"`
	ArtifactPaths   daemonArtifactPaths `json:"artifact_paths"`
}

type evidenceSummary struct {
	Status               string `json:"status"`
	Source               string `json:"source"`
	SourceEventID        string `json:"source_event_id"`
	SourceEventType      string `json:"source_event_type"`
	SourceEventTimestamp string `json:"source_event_timestamp"`
}

type reviewPlaceholder struct {
	SchemaVersion      string   `json:"schema_version"`
	SessionID          string   `json:"session_id"`
	Verdict            string   `json:"verdict"`
	MainFinding        string   `json:"main_finding"`
	Evidence           []string `json:"evidence"`
	SuggestedNextHabit string   `json:"suggested_next_habit"`
	Warnings           []string `json:"warnings"`
	SourceEventID      string   `json:"source_event_id"`
}

type adpPlaceholderRecord struct {
	RecordType  string `json:"record_type"`
	SessionID   string `json:"session_id"`
	Source      string `json:"source"`
	EventType   string `json:"event_type"`
	Timestamp   string `json:"timestamp"`
	Placeholder bool   `json:"placeholder"`
}

type apiErrorResponse struct {
	SchemaVersion string       `json:"schema_version"`
	Error         apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func daemon(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing daemon command")
		return 2
	}
	if args[0] != "run-once" {
		fmt.Fprintf(stderr, "error: unsupported daemon command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: daemon run-once does not accept arguments")
		printUsage(stderr)
		return 2
	}

	summary, err := runDaemonOnce()
	if err != nil {
		writeAPIError(stderr, "daemon.run_once_failed", err.Error())
		return 1
	}

	fmt.Fprintln(stdout, "qratum daemon run-once")
	fmt.Fprintf(stdout, "events: %d\n", summary.Events)
	fmt.Fprintf(stdout, "processed: %d\n", summary.Processed)
	fmt.Fprintf(stdout, "skipped: %d\n", summary.Skipped)
	return 0
}

func runDaemonOnce() (daemonRunSummary, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return daemonRunSummary{}, fmt.Errorf("resolve current project: %w", err)
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return daemonRunSummary{}, fmt.Errorf("resolve current project absolute path: %w", err)
	}

	eventPaths, err := listEventFiles(filepath.Join(projectRoot, ".qratum", "events"), projectRoot)
	if err != nil {
		return daemonRunSummary{}, err
	}

	summary := daemonRunSummary{Events: len(eventPaths)}
	for _, eventPath := range eventPaths {
		event, err := readCaptureEventFile(eventPath, projectRoot)
		if err != nil {
			return summary, err
		}

		artifacts := artifactPathsForEvent(event)
		completed, empty, missing, existing, err := inspectArtifactCompletion(artifactFilesForPaths(projectRoot, artifacts))
		if err != nil {
			return summary, err
		}
		if completed {
			summary.Skipped++
			continue
		}
		if !empty {
			return summary, fmt.Errorf("partial artifacts for event %s: missing %s; existing %s", event.EventID, strings.Join(missing, ", "), strings.Join(existing, ", "))
		}

		transcriptPath, err := resolveTranscriptPath(projectRoot, event.SessionRef.TranscriptPath)
		if err != nil {
			return summary, fmt.Errorf("event %s has invalid transcript_path: %w", event.EventID, err)
		}
		if err := requireTranscriptFile(transcriptPath, projectRoot, event.SessionRef.TranscriptPath); err != nil {
			return summary, fmt.Errorf("event %s: %w", event.EventID, err)
		}

		if err := writePipelinePlaceholders(projectRoot, event, artifacts); err != nil {
			return summary, err
		}
		summary.Processed++
	}

	return summary, nil
}

func listEventFiles(eventsDir string, projectRoot string) ([]string, error) {
	info, err := os.Stat(eventsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("event spool %s does not exist", displayPath(projectRoot, eventsDir))
		}
		return nil, fmt.Errorf("inspect event spool %s: %w", displayPath(projectRoot, eventsDir), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("event spool %s is not a directory", displayPath(projectRoot, eventsDir))
	}

	paths, err := filepath.Glob(filepath.Join(eventsDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list event spool %s: %w", displayPath(projectRoot, eventsDir), err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("event spool %s has no event JSON files", displayPath(projectRoot, eventsDir))
	}
	return paths, nil
}

func readCaptureEventFile(path string, projectRoot string) (captureEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return captureEvent{}, fmt.Errorf("read capture event %s: %w", displayPath(projectRoot, path), err)
	}

	var event captureEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return captureEvent{}, fmt.Errorf("invalid capture event JSON %s: %w", displayPath(projectRoot, path), err)
	}

	if err := validateCaptureEventFile(&event, path); err != nil {
		return captureEvent{}, fmt.Errorf("invalid capture event %s: %w", displayPath(projectRoot, path), err)
	}
	return event, nil
}

func validateCaptureEventFile(event *captureEvent, path string) error {
	event.SchemaVersion = strings.TrimSpace(event.SchemaVersion)
	event.EventID = strings.TrimSpace(event.EventID)
	event.Source = strings.TrimSpace(event.Source)
	event.EventType = strings.TrimSpace(event.EventType)
	event.Timestamp = strings.TrimSpace(event.Timestamp)
	event.SessionRef.SessionID = strings.TrimSpace(event.SessionRef.SessionID)
	event.SessionRef.TranscriptPath = strings.TrimSpace(event.SessionRef.TranscriptPath)
	event.Workspace.CWD = strings.TrimSpace(event.Workspace.CWD)

	if event.SchemaVersion != captureEventSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", event.SchemaVersion)
	}
	if event.EventID == "" {
		return fmt.Errorf("missing event_id")
	}
	if !validArtifactStem(event.EventID) {
		return fmt.Errorf("event_id %q is not a safe artifact id", event.EventID)
	}
	if got, want := filepath.Base(path), event.EventID+".json"; got != want {
		return fmt.Errorf("filename %q does not match event_id %q", got, event.EventID)
	}
	if event.Source != claudeCodeSource {
		return fmt.Errorf("unsupported source %q", event.Source)
	}
	switch event.EventType {
	case "session_start", "session_end":
	default:
		return fmt.Errorf("unsupported event_type %q", event.EventType)
	}
	if event.Timestamp == "" {
		return fmt.Errorf("missing timestamp")
	}
	if event.SessionRef.SessionID == "" {
		return fmt.Errorf("missing session_ref.session_id")
	}
	if event.SessionRef.TranscriptPath == "" {
		return fmt.Errorf("missing session_ref.transcript_path")
	}
	if event.Workspace.CWD == "" {
		return fmt.Errorf("missing workspace.cwd")
	}
	return nil
}

func validArtifactStem(stem string) bool {
	if stem == "" || stem == "." || stem == ".." {
		return false
	}
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func artifactPathsForEvent(event captureEvent) daemonArtifactPaths {
	stem := event.EventID
	return daemonArtifactPaths{
		Event:    slashPath(filepath.Join(".qratum", "events", stem+".json")),
		Session:  slashPath(filepath.Join(".qratum", "sessions", stem+".normalized.json")),
		Redacted: slashPath(filepath.Join(".qratum", "redacted", stem+".redacted.json")),
		Evidence: slashPath(filepath.Join(".qratum", "evidence", stem+".evidence.json")),
		Review:   slashPath(filepath.Join(".qratum", "reviews", stem+".review.json")),
		Report:   slashPath(filepath.Join(".qratum", "reports", stem+".html")),
		Export:   slashPath(filepath.Join(".qratum", "exports", stem+".adp.jsonl")),
	}
}

func artifactFilesForPaths(projectRoot string, paths daemonArtifactPaths) []daemonArtifactFile {
	return []daemonArtifactFile{
		{rel: paths.Session, abs: filepath.Join(projectRoot, filepath.FromSlash(paths.Session))},
		{rel: paths.Redacted, abs: filepath.Join(projectRoot, filepath.FromSlash(paths.Redacted))},
		{rel: paths.Evidence, abs: filepath.Join(projectRoot, filepath.FromSlash(paths.Evidence))},
		{rel: paths.Review, abs: filepath.Join(projectRoot, filepath.FromSlash(paths.Review))},
		{rel: paths.Report, abs: filepath.Join(projectRoot, filepath.FromSlash(paths.Report))},
		{rel: paths.Export, abs: filepath.Join(projectRoot, filepath.FromSlash(paths.Export))},
	}
}

func inspectArtifactCompletion(files []daemonArtifactFile) (completed bool, empty bool, missing []string, existing []string, err error) {
	for _, file := range files {
		info, statErr := os.Stat(file.abs)
		if statErr == nil {
			if info.IsDir() {
				return false, false, nil, nil, fmt.Errorf("artifact path %s is a directory", file.rel)
			}
			existing = append(existing, file.rel)
			continue
		}
		if errors.Is(statErr, os.ErrNotExist) {
			missing = append(missing, file.rel)
			continue
		}
		return false, false, nil, nil, fmt.Errorf("inspect artifact %s: %w", file.rel, statErr)
	}
	return len(missing) == 0, len(existing) == 0, missing, existing, nil
}

func resolveTranscriptPath(projectRoot string, transcriptPath string) (string, error) {
	transcriptPath = strings.TrimSpace(transcriptPath)
	if transcriptPath == "" {
		return "", fmt.Errorf("missing transcript path")
	}
	if filepath.IsAbs(transcriptPath) {
		return filepath.Clean(transcriptPath), nil
	}

	resolved := filepath.Clean(filepath.Join(projectRoot, transcriptPath))
	rel, err := filepath.Rel(projectRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve relative transcript path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("relative path %q escapes current project", transcriptPath)
	}
	return resolved, nil
}

func requireTranscriptFile(path string, projectRoot string, original string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing transcript %s resolved from %q", displayPath(projectRoot, path), original)
		}
		return fmt.Errorf("inspect transcript %s resolved from %q: %w", displayPath(projectRoot, path), original, err)
	}
	if info.IsDir() {
		return fmt.Errorf("transcript %s resolved from %q is a directory", displayPath(projectRoot, path), original)
	}
	return nil
}

func writePipelinePlaceholders(projectRoot string, event captureEvent, paths daemonArtifactPaths) error {
	files := artifactFilesForPaths(projectRoot, paths)
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.abs), 0o755); err != nil {
			return fmt.Errorf("create artifact directory for %s: %w", file.rel, err)
		}
	}

	writes := []struct {
		path string
		data []byte
	}{
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Session)), mustJSON(buildSessionPlaceholder(event, paths, true))},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Redacted)), mustJSON(buildSessionPlaceholder(event, paths, false))},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Evidence)), mustJSON(buildEvidencePlaceholder(event, paths))},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Review)), mustJSON(buildReviewPlaceholder(event, paths))},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Report)), buildReportPlaceholder(event, paths)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Export)), buildADPPlaceholder(event)},
	}

	for _, write := range writes {
		if err := writeFileAtomic(write.path, write.data, 0o644); err != nil {
			return fmt.Errorf("write artifact %s: %w", displayPath(projectRoot, write.path), err)
		}
	}
	return nil
}

func buildSessionPlaceholder(event captureEvent, paths daemonArtifactPaths, normalized bool) pipelineSessionPlaceholder {
	status := "redaction_placeholder_pending"
	transcriptPath := ""
	var workspace *captureWorkspaceRef
	if normalized {
		status = "normalize_placeholder_pending"
		transcriptPath = event.SessionRef.TranscriptPath
		workspace = &event.Workspace
	}

	return pipelineSessionPlaceholder{
		SchemaVersion:        qratumSessionSchemaVersion,
		SessionID:            event.SessionRef.SessionID,
		Source:               event.Source,
		Turns:                []any{},
		ToolCalls:            []any{},
		FileChanges:          []any{},
		Commands:             []any{},
		SourceEventID:        event.EventID,
		SourceEventType:      event.EventType,
		SourceEventTimestamp: event.Timestamp,
		TranscriptPath:       transcriptPath,
		Workspace:            workspace,
		PipelineStatus:       status,
		ArtifactPaths:        paths,
	}
}

func buildEvidencePlaceholder(event captureEvent, paths daemonArtifactPaths) evidencePlaceholder {
	return evidencePlaceholder{
		SchemaVersion: qratumEvidenceSchemaVersion,
		SessionID:     event.SessionRef.SessionID,
		Summary: evidenceSummary{
			Status:               "placeholder_pending",
			Source:               event.Source,
			SourceEventID:        event.EventID,
			SourceEventType:      event.EventType,
			SourceEventTimestamp: event.Timestamp,
		},
		Findings: []any{},
		MissingEvidence: []string{
			"normalize.not_implemented",
			"redaction.not_implemented",
			"evidence.not_implemented",
		},
		SourceEventID: event.EventID,
		ArtifactPaths: paths,
	}
}

func buildReviewPlaceholder(event captureEvent, paths daemonArtifactPaths) reviewPlaceholder {
	return reviewPlaceholder{
		SchemaVersion: qratumReviewSchemaVersion,
		SessionID:     event.SessionRef.SessionID,
		Verdict:       "needs_attention",
		MainFinding:   "Pipeline shell generated placeholder artifacts; review content is pending.",
		Evidence: []string{
			paths.Event,
			paths.Evidence,
		},
		SuggestedNextHabit: "Run the completed local pipeline before using this review.",
		Warnings: []string{
			"pipeline.placeholder",
		},
		SourceEventID: event.EventID,
	}
}

func buildReportPlaceholder(event captureEvent, paths daemonArtifactPaths) []byte {
	var b strings.Builder
	b.WriteString("<!doctype html>\n")
	b.WriteString("<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Qratum Session ")
	b.WriteString(html.EscapeString(event.SessionRef.SessionID))
	b.WriteString("</title>\n")
	b.WriteString("</head>\n<body>\n")
	b.WriteString("<h1>Qratum Session ")
	b.WriteString(html.EscapeString(event.SessionRef.SessionID))
	b.WriteString("</h1>\n")
	b.WriteString("<p>Pipeline shell placeholder generated for event ")
	b.WriteString(html.EscapeString(event.EventID))
	b.WriteString(".</p>\n")
	b.WriteString("<p>Event timestamp: ")
	b.WriteString(html.EscapeString(event.Timestamp))
	b.WriteString("</p>\n")
	b.WriteString("<h2>Artifact paths</h2>\n<ul>\n")
	for _, item := range []struct {
		label string
		path  string
	}{
		{"Event", paths.Event},
		{"Session", paths.Session},
		{"Redacted", paths.Redacted},
		{"Evidence", paths.Evidence},
		{"Review", paths.Review},
		{"Report", paths.Report},
		{"Export", paths.Export},
	} {
		b.WriteString("<li>")
		b.WriteString(html.EscapeString(item.label))
		b.WriteString(": ")
		b.WriteString(html.EscapeString(item.path))
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n")
	b.WriteString("</body>\n</html>\n")
	return []byte(b.String())
}

func buildADPPlaceholder(event captureEvent) []byte {
	record := adpPlaceholderRecord{
		RecordType:  "placeholder",
		SessionID:   event.SessionRef.SessionID,
		Source:      event.Source,
		EventType:   event.EventType,
		Timestamp:   event.Timestamp,
		Placeholder: true,
	}
	data, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func mustJSON(value any) []byte {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	removeTmp = false
	return nil
}

func writeAPIError(w io.Writer, code string, message string) {
	response := apiErrorResponse{
		SchemaVersion: qratumAPIErrorSchemaVersion,
		Error: apiErrorBody{
			Code:    code,
			Message: message,
		},
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "error: %s\n", message)
		return
	}
	fmt.Fprintln(w, string(data))
}

func displayPath(projectRoot string, path string) string {
	if rel, err := filepath.Rel(projectRoot, path); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." && !filepath.IsAbs(rel) {
		return slashPath(rel)
	}
	return slashPath(path)
}

func slashPath(path string) string {
	return filepath.ToSlash(path)
}

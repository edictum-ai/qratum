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
	Event    string `json:"event,omitempty"`
	Session  string `json:"session,omitempty"`
	Redacted string `json:"redacted,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Review   string `json:"review,omitempty"`
	Report   string `json:"report,omitempty"`
	Export   string `json:"export,omitempty"`
}

type daemonArtifactFile struct {
	rel string
	abs string
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

		session, err := normalizeClaudeTranscriptFile(transcriptPath, normalizeSessionContext{
			SessionID:            event.SessionRef.SessionID,
			TranscriptPath:       event.SessionRef.TranscriptPath,
			Workspace:            &event.Workspace,
			SourceEventID:        event.EventID,
			SourceEventType:      event.EventType,
			SourceEventTimestamp: event.Timestamp,
			ArtifactPaths:        &artifacts,
			PipelineStatus:       "normalized",
		})
		if err != nil {
			return summary, fmt.Errorf("event %s normalize transcript %s: %w", event.EventID, displayPath(projectRoot, transcriptPath), err)
		}

		if err := writePipelineArtifacts(projectRoot, event, artifacts, session); err != nil {
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
	return artifactPathsForStem(event.EventID)
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

func writePipelineArtifacts(projectRoot string, event captureEvent, paths daemonArtifactPaths, session qratumSession) error {
	redactedSession, err := redactQratumSession(session)
	if err != nil {
		return fmt.Errorf("redact session %s: %w", session.SessionID, err)
	}
	evidenceBundle, err := buildEvidenceBundle(redactedSession, paths)
	if err != nil {
		return fmt.Errorf("build evidence for session %s: %w", session.SessionID, err)
	}
	reviewCard, err := buildReviewCard(evidenceBundle)
	if err != nil {
		return fmt.Errorf("build review for session %s: %w", session.SessionID, err)
	}

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
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Session)), mustJSON(session)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Redacted)), mustJSON(redactedSession)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Evidence)), mustJSON(evidenceBundle)},
		{filepath.Join(projectRoot, filepath.FromSlash(paths.Review)), mustJSON(reviewCard)},
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

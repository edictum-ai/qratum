package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/edictum-ai/qratum/internal/workspace"
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
	Skips     []daemonSkippedEvent
}

type daemonSkippedEvent struct {
	EventID string
	Reason  string
}

const skipReasonAlreadyProcessed = "already_processed"
const (
	skipReasonRawMissing    = "raw_missing"
	skipReasonRawCopyFailed = "raw_copy_failed"
)

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
	if len(summary.Skips) > 0 {
		fmt.Fprintln(stdout, "skipped_events:")
		for _, skip := range summary.Skips {
			fmt.Fprintf(stdout, "- event_id: %s\n", skip.EventID)
			fmt.Fprintf(stdout, "  reason: %s\n", skip.Reason)
		}
	}
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

	qratumHome, err := workspace.Resolve()
	if err != nil {
		return daemonRunSummary{}, err
	}

	eventPaths, err := listEventFiles(qratumHome.EventsDir(), projectRoot)
	if err != nil {
		return daemonRunSummary{}, err
	}

	summary := daemonRunSummary{Events: len(eventPaths)}
	for _, eventPath := range eventPaths {
		event, err := readCaptureEventFile(eventPath, projectRoot)
		if err != nil {
			return summary, err
		}

		artifacts, err := artifactPathsForEvent(event)
		if err != nil {
			return summary, fmt.Errorf("event %s has invalid artifact stem: %w", event.EventID, err)
		}
		completed, empty, missing, existing, err := inspectArtifactCompletion(artifactFilesForPaths(projectRoot, artifacts))
		if err != nil {
			return summary, err
		}
		if completed {
			summary.Skipped++
			summary.Skips = append(summary.Skips, daemonSkippedEvent{EventID: event.EventID, Reason: skipReasonAlreadyProcessed})
			continue
		}
		if !empty {
			return summary, fmt.Errorf("partial artifacts for event %s: missing %s; existing %s", event.EventID, strings.Join(missing, ", "), strings.Join(existing, ", "))
		}

		if event.Raw != nil && event.Raw.RawMissing {
			summary.Skipped++
			summary.Skips = append(summary.Skips, daemonSkippedEvent{EventID: event.EventID, Reason: skipReasonRawMissing})
			continue
		}
		if event.Raw != nil && event.Raw.CopyStatus == "failed" {
			summary.Skipped++
			summary.Skips = append(summary.Skips, daemonSkippedEvent{EventID: event.EventID, Reason: skipReasonRawCopyFailed})
			continue
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
		if applySourceEventTimestampFallback(&event, session) {
			eventPath := artifactAbsolutePath(projectRoot, artifacts.Event)
			if err := writeFileAtomic(eventPath, mustJSON(event), 0o600); err != nil {
				return summary, fmt.Errorf("update capture event %s: %w", displayPath(projectRoot, eventPath), err)
			}
		}
		session.SourceEventTimestamp = event.Timestamp

		if err := writePipelineArtifacts(projectRoot, artifacts, session); err != nil {
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
	event.TimestampSource = strings.TrimSpace(event.TimestampSource)
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
	if _, err := time.Parse(time.RFC3339Nano, event.Timestamp); err != nil {
		return fmt.Errorf("timestamp must be RFC3339")
	}
	switch event.TimestampSource {
	case "", hookTimestampSourceHookPayload, hookTimestampSourceCaptureTime, hookTimestampSourceTranscriptEnd:
	default:
		return fmt.Errorf("unsupported timestamp_source %q", event.TimestampSource)
	}
	if event.SessionRef.SessionID == "" {
		return fmt.Errorf("missing session_ref.session_id")
	}
	if event.SessionRef.TranscriptPath == "" && (event.Raw == nil || !event.Raw.RawMissing) {
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

func artifactPathsForEvent(event captureEvent) (daemonArtifactPaths, error) {
	stem, err := artifactStemForSession(event.SessionRef.SessionID)
	if err != nil {
		return daemonArtifactPaths{}, err
	}
	paths := artifactPathsForStem(stem)
	paths.Event = filepath.ToSlash(filepath.Join("events", event.EventID+".json"))
	return paths, nil
}

func applySourceEventTimestampFallback(event *captureEvent, session qratumSession) bool {
	if event.EventType == "session_end" && strings.TrimSpace(session.EndedAt) != "" && event.Timestamp != session.EndedAt {
		event.Timestamp = session.EndedAt
		event.TimestampSource = hookTimestampSourceTranscriptEnd
		return true
	}
	return false
}

func artifactFilesForPaths(projectRoot string, paths daemonArtifactPaths) []daemonArtifactFile {
	return []daemonArtifactFile{
		{rel: paths.Session, abs: artifactAbsolutePath(projectRoot, paths.Session)},
		{rel: paths.Redacted, abs: artifactAbsolutePath(projectRoot, paths.Redacted)},
		{rel: paths.Evidence, abs: artifactAbsolutePath(projectRoot, paths.Evidence)},
		{rel: paths.Review, abs: artifactAbsolutePath(projectRoot, paths.Review)},
		{rel: paths.Report, abs: artifactAbsolutePath(projectRoot, paths.Report)},
		{rel: paths.Export, abs: artifactAbsolutePath(projectRoot, paths.Export)},
	}
}

func artifactAbsolutePath(projectRoot string, path string) string {
	resolved := filepath.FromSlash(path)
	if filepath.IsAbs(resolved) {
		return filepath.Clean(resolved)
	}
	qratumHome, err := workspace.Resolve()
	if err != nil {
		return filepath.Join(projectRoot, resolved)
	}
	return filepath.Join(qratumHome.Root, resolved)
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

func writePipelineArtifacts(projectRoot string, paths daemonArtifactPaths, session qratumSession) error {
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
		if err := os.MkdirAll(filepath.Dir(file.abs), 0o700); err != nil {
			return fmt.Errorf("create artifact directory for %s: %w", file.rel, err)
		}
	}

	adpExport, err := buildADPStrictJSONL(redactedSession)
	if err != nil {
		return fmt.Errorf("build ADP strict export for session %s: %w", session.SessionID, err)
	}

	writes := []struct {
		path string
		data []byte
	}{
		{artifactAbsolutePath(projectRoot, paths.Session), mustJSON(session)},
		{artifactAbsolutePath(projectRoot, paths.Redacted), mustJSON(redactedSession)},
		{artifactAbsolutePath(projectRoot, paths.Evidence), mustJSON(evidenceBundle)},
		{artifactAbsolutePath(projectRoot, paths.Review), mustJSON(reviewCard)},
		{artifactAbsolutePath(projectRoot, paths.Export), adpExport},
	}

	for _, write := range writes {
		if err := writeFileAtomic(write.path, write.data, 0o600); err != nil {
			return fmt.Errorf("write artifact %s: %w", displayPath(projectRoot, write.path), err)
		}
	}

	reportData, err := buildReportDocument(projectRoot, reportContext{
		session:     session,
		redacted:    redactedSession,
		evidence:    evidenceBundle,
		review:      reviewCard,
		paths:       paths,
		sessionPath: paths.Session,
		artifactsAt: uiArtifactCreatedAt(session, evidenceBundle),
	})
	if err != nil {
		return fmt.Errorf("build report for session %s: %w", session.SessionID, err)
	}
	reportPath := artifactAbsolutePath(projectRoot, paths.Report)
	if err := writeFileAtomic(reportPath, reportData, 0o600); err != nil {
		return fmt.Errorf("write artifact %s: %w", displayPath(projectRoot, reportPath), err)
	}
	return nil
}

func mustJSON(value any) []byte {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
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

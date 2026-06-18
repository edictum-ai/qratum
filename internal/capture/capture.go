// Package capture owns the fast Claude Code hook capture path.
package capture

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	qschema "github.com/edictum-ai/qratum/internal/schema"
	"github.com/edictum-ai/qratum/internal/vault"
	"github.com/edictum-ai/qratum/internal/workspace"
)

const (
	// EventSchemaVersion is the capture event schema version.
	EventSchemaVersion = "qratum.event.v1"
	// ClaudeCodeSource labels Claude Code capture events.
	ClaudeCodeSource = "claude-code"
	// DeprecatedUnixZeroHookTimestamp is the legacy invalid hook timestamp sentinel.
	DeprecatedUnixZeroHookTimestamp = "1970-01-01T00:00:00Z"
	// HookTimestampSourceHookPayload means the hook payload supplied the event timestamp.
	HookTimestampSourceHookPayload = "hook_payload"
	// HookTimestampSourceCaptureTime means qrt supplied the event timestamp at capture time.
	HookTimestampSourceCaptureTime = "capture_time"
	// HookTimestampSourceTranscriptEnd means the daemon later aligned the event timestamp to transcript end.
	HookTimestampSourceTranscriptEnd = "transcript_end"
	// MaxHookPayloadBytes caps hook stdin so the hook remains fast.
	MaxHookPayloadBytes = 1 << 20
)

// ClaudeCodeHookPayload is the small JSON object Claude Code sends to the hook.
type ClaudeCodeHookPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      string `json:"timestamp"`
}

// Event is the capture event written to the qratum event spool.
type Event struct {
	SchemaVersion   string       `json:"schema_version"`
	DataClass       string       `json:"data_class"`
	EventID         string       `json:"event_id"`
	Source          string       `json:"source"`
	EventType       string       `json:"event_type"`
	Timestamp       string       `json:"timestamp"`
	TimestampSource string       `json:"timestamp_source,omitempty"`
	SessionRef      SessionRef   `json:"session_ref"`
	Workspace       WorkspaceRef `json:"workspace"`
	Raw             *EventRaw    `json:"raw,omitempty"`
}

// EventRaw records the outcome of copying the source transcript into the vault.
type EventRaw struct {
	RawMissing bool   `json:"raw_missing,omitempty"`
	CopyStatus string `json:"copy_status,omitempty"`
	RawRefID   string `json:"raw_ref_id,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Kind       string `json:"kind,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
	CopyError  string `json:"copy_error,omitempty"`
}

// SessionRef identifies the captured source session.
type SessionRef struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// WorkspaceRef records the source workspace reported by the hook payload.
type WorkspaceRef struct {
	CWD string `json:"cwd"`
}

type captureError string

func (e captureError) Error() string { return string(e) }

// SpoolClaudeCodeHook reads stdin, archives the transcript if allowed, writes one
// capture event with O_EXCL, and updates vault state. captureTime must be an
// RFC3339 timestamp supplied by the caller when the payload omitted one.
func SpoolClaudeCodeHook(stdin io.Reader, captureTime string) (Event, error) {
	paths, err := workspace.Resolve()
	if err != nil {
		return Event{}, err
	}
	return SpoolClaudeCodeHookToStore(stdin, vault.New(paths), captureTime)
}

// SpoolClaudeCodeHookToStore is the store-injected form used by tests.
func SpoolClaudeCodeHookToStore(stdin io.Reader, store vault.Store, captureTime string) (Event, error) {
	payload, err := readClaudeCodeHookPayload(stdin)
	if err != nil {
		return Event{}, err
	}

	eventType, err := claudeCodeEventType(payload.HookEventName)
	if err != nil {
		return Event{}, err
	}

	event := newEvent(payload, eventType, captureTime)
	if eventType == "session_end" {
		event.Raw = eventRawForPayload(payload, store, event.Timestamp)
	}
	if err := writeEvent(store.Paths.EventsDir(), event); err != nil {
		return Event{}, err
	}
	if err := store.UpdateState(func(state *vault.State) {
		state.LastCaptureAt = event.Timestamp
		if event.Raw != nil {
			if event.Raw.RawMissing {
				state.RawMissingCount++
			}
			if event.Raw.CopyStatus == "failed" {
				state.CopyFailureCount++
			}
		}
	}); err != nil {
		return Event{}, err
	}
	return event, nil
}

// HookWarning returns the operator-visible degraded-capture warning for an event.
func HookWarning(event Event) string {
	if event.Raw == nil {
		return ""
	}
	if event.Raw.RawMissing {
		return "capture recorded without transcript_path; preservation degraded"
	}
	if event.Raw.CopyStatus == "failed" {
		return "capture recorded but transcript copy failed: " + event.Raw.CopyError
	}
	return ""
}

// ResolveHookTranscriptPath confines transcript_path to ~/.claude/projects, the
// hook process cwd, or the cwd reported by the hook payload. Existing non-symlink
// paths are resolved through filepath.EvalSymlinks before containment so an
// intermediate symlink cannot redirect capture outside the allowed roots; a final
// symlink is left unresolved so vault.ArchiveFile rejects it without following it.
func ResolveHookTranscriptPath(payload ClaudeCodeHookPayload) (string, error) {
	path := strings.TrimSpace(payload.TranscriptPath)
	if path == "" {
		return "", captureError("missing transcript_path")
	}

	roots := allowedTranscriptRoots(payload)
	if filepath.IsAbs(path) {
		candidate, err := resolveTranscriptCandidate(roots, path)
		if err != nil {
			return "", err
		}
		return candidate.path, nil
	}

	bases := candidateBases(payload)
	fallback := ""
	for _, base := range bases {
		candidate, err := resolveTranscriptCandidate(roots, filepath.Join(base, path))
		if err != nil {
			continue
		}
		if fallback == "" {
			fallback = candidate.path
		}
		if candidate.exists && !candidate.isDir {
			return candidate.path, nil
		}
	}
	if fallback == "" {
		return "", captureError("transcript_path escapes allowed capture roots")
	}
	return fallback, nil
}

// RawKindForTranscriptPath maps Claude transcript paths to raw vault kinds.
func RawKindForTranscriptPath(path string) string {
	if strings.Contains(filepath.ToSlash(path), "/subagents/") {
		return vault.KindSubagentTranscript
	}
	return vault.KindMainTranscript
}

func eventRawForPayload(payload ClaudeCodeHookPayload, store vault.Store, observedAt string) *EventRaw {
	if strings.TrimSpace(payload.TranscriptPath) == "" {
		return &EventRaw{RawMissing: true, CopyStatus: "missing"}
	}

	archivePath, err := ResolveHookTranscriptPath(payload)
	if err != nil {
		return &EventRaw{CopyStatus: "failed", CopyError: err.Error(), Kind: RawKindForTranscriptPath(payload.TranscriptPath)}
	}
	minFreeBytes, err := store.ConfiguredDiskFreeMinBytes()
	if err != nil {
		return &EventRaw{CopyStatus: "failed", CopyError: err.Error(), Kind: RawKindForTranscriptPath(archivePath)}
	}
	result, err := store.ArchiveFile(vault.ArchiveRequest{
		Source:          vault.SourceClaudeCode,
		SourceSessionID: payload.SessionID,
		Kind:            RawKindForTranscriptPath(archivePath),
		OriginalPath:    archivePath,
		ObservedAt:      observedAt,
		MinFreeBytes:    minFreeBytes,
	})
	if err != nil {
		return &EventRaw{CopyStatus: "failed", CopyError: err.Error(), Kind: RawKindForTranscriptPath(archivePath)}
	}
	status := "copied"
	if !result.BlobCreated {
		status = "deduped"
	}
	return &EventRaw{
		CopyStatus: status,
		RawRefID:   result.RawRef.RawRefID,
		Digest:     result.RawRef.Digest,
		Kind:       result.RawRef.Kind,
		SizeBytes:  result.RawRef.SizeBytes,
	}
}

func readClaudeCodeHookPayload(stdin io.Reader) (ClaudeCodeHookPayload, error) {
	data, err := io.ReadAll(io.LimitReader(stdin, MaxHookPayloadBytes+1))
	if err != nil {
		return ClaudeCodeHookPayload{}, captureError("read hook JSON: " + err.Error())
	}
	if len(data) > MaxHookPayloadBytes {
		return ClaudeCodeHookPayload{}, captureError("hook JSON exceeds 1048576 bytes")
	}
	if strings.TrimSpace(string(data)) == "" {
		return ClaudeCodeHookPayload{}, captureError("missing hook JSON on stdin")
	}

	var payload ClaudeCodeHookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return ClaudeCodeHookPayload{}, captureError("invalid hook JSON: " + err.Error())
	}
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	payload.TranscriptPath = strings.TrimSpace(payload.TranscriptPath)
	payload.CWD = strings.TrimSpace(payload.CWD)
	payload.HookEventName = strings.TrimSpace(payload.HookEventName)
	payload.Timestamp = strings.TrimSpace(payload.Timestamp)

	if payload.SessionID == "" {
		return ClaudeCodeHookPayload{}, captureError("missing required hook field session_id")
	}
	if payload.CWD == "" {
		return ClaudeCodeHookPayload{}, captureError("missing required hook field cwd")
	}
	if payload.HookEventName == "" {
		return ClaudeCodeHookPayload{}, captureError("missing required hook field hook_event_name")
	}
	if payload.Timestamp != "" && !looksRFC3339(payload.Timestamp) {
		return ClaudeCodeHookPayload{}, captureError("invalid hook field timestamp: must be RFC3339")
	}
	return payload, nil
}

func claudeCodeEventType(hookEventName string) (string, error) {
	switch hookEventName {
	case "SessionStart", "session_start":
		return "session_start", nil
	case "SessionEnd", "session_end":
		return "session_end", nil
	default:
		return "", captureError("unsupported Claude Code hook_event_name \"" + hookEventName + "\"")
	}
}

func newEvent(payload ClaudeCodeHookPayload, eventType string, captureTime string) Event {
	timestamp := payload.Timestamp
	timestampSource := HookTimestampSourceHookPayload
	if timestamp == "" {
		timestamp = captureTime
		timestampSource = HookTimestampSourceCaptureTime
	}

	event := Event{
		SchemaVersion:   EventSchemaVersion,
		DataClass:       qschema.DataClassRaw,
		Source:          ClaudeCodeSource,
		EventType:       eventType,
		Timestamp:       timestamp,
		TimestampSource: timestampSource,
		SessionRef: SessionRef{
			SessionID:      payload.SessionID,
			TranscriptPath: payload.TranscriptPath,
		},
		Workspace: WorkspaceRef{CWD: payload.CWD},
	}
	event.EventID = deterministicEventID(event)
	return event
}

func deterministicEventID(event Event) string {
	hash := sha256.New()
	for _, part := range []string{
		event.Source,
		event.EventType,
		event.SessionRef.SessionID,
		event.SessionRef.TranscriptPath,
		event.Workspace.CWD,
	} {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return "evt_" + hex.EncodeToString(hash.Sum(nil))[:32]
}

func writeEvent(eventsDir string, event Event) error {
	if err := os.MkdirAll(eventsDir, 0o700); err != nil {
		return captureError("create event spool " + filepath.ToSlash(eventsDir) + ": " + err.Error())
	}

	baseEventID := event.EventID
	for attempt := 1; ; attempt++ {
		eventID := baseEventID
		if attempt > 1 {
			eventID = baseEventID + "_" + strconv.Itoa(attempt)
		}
		event.EventID = eventID
		path := filepath.Join(eventsDir, eventID+".json")
		// #nosec G304 -- capture event paths are derived from the resolved qratum workspace root and sanitized event ids.
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return captureError("create capture event " + filepath.ToSlash(path) + ": " + err.Error())
		}
		data, err := marshalEvent(event)
		if err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return captureError("encode capture event: " + err.Error())
		}
		if err := file.Chmod(0o600); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return captureError("set capture event permissions: " + err.Error())
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return captureError("write capture event: " + err.Error())
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return captureError("close capture event: " + err.Error())
		}
		return nil
	}
}

func marshalEvent(event Event) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(event); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type transcriptCandidate struct {
	path   string
	exists bool
	isDir  bool
}

func resolveTranscriptCandidate(roots []string, path string) (transcriptCandidate, error) {
	cleaned := filepath.Clean(path)
	candidate := transcriptCandidate{path: cleaned}
	containmentPath := cleaned

	info, err := os.Lstat(cleaned)
	if err == nil {
		candidate.exists = true
		candidate.isDir = info.IsDir()
		if info.Mode()&os.ModeSymlink == 0 {
			if evaluated, evalErr := filepath.EvalSymlinks(cleaned); evalErr == nil {
				candidate.path = filepath.Clean(evaluated)
				containmentPath = candidate.path
			}
		}
	}

	if !containedInAnyRoot(roots, containmentPath) {
		return transcriptCandidate{}, captureError("transcript_path escapes allowed capture roots")
	}
	return candidate, nil
}

func allowedTranscriptRoots(payload ClaudeCodeHookPayload) []string {
	roots := []string{}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		roots = appendCanonicalRoot(roots, wd)
	}
	if cwd := strings.TrimSpace(payload.CWD); cwd != "" {
		roots = appendCanonicalRoot(roots, absoluteFromProcessCWD(cwd))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = appendCanonicalRoot(roots, filepath.Join(home, ".claude", "projects"))
	}
	return uniqueNonEmptyStrings(roots)
}

func appendCanonicalRoot(roots []string, root string) []string {
	cleaned := filepath.Clean(root)
	roots = append(roots, cleaned)
	if evaluated, err := filepath.EvalSymlinks(cleaned); err == nil {
		roots = append(roots, filepath.Clean(evaluated))
	}
	return roots
}

func candidateBases(payload ClaudeCodeHookPayload) []string {
	bases := []string{}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		bases = append(bases, filepath.Clean(wd))
	}
	if cwd := strings.TrimSpace(payload.CWD); cwd != "" {
		bases = append(bases, absoluteFromProcessCWD(cwd))
	}
	return uniqueNonEmptyStrings(bases)
}

func absoluteFromProcessCWD(path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return filepath.Clean(filepath.Join(wd, path))
	}
	return filepath.Clean(path)
}

func containedInAnyRoot(roots []string, path string) bool {
	for _, root := range roots {
		if pathContainedIn(root, path) {
			return true
		}
	}
	return false
}

func pathContainedIn(root string, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel))
}

func uniqueNonEmptyStrings(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = filepath.Clean(value)
		if value == "" || value == "." {
			continue
		}
		seen := false
		for _, existing := range out {
			if existing == value {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, value)
		}
	}
	return out
}

func looksRFC3339(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

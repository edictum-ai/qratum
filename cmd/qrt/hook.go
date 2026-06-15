package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/edictum-ai/qratum/internal/vault"
	"github.com/edictum-ai/qratum/internal/workspace"
)

const (
	captureEventSchemaVersion        = "qratum.event.v1"
	claudeCodeSource                 = "claude-code"
	deprecatedUnixZeroHookTimestamp  = "1970-01-01T00:00:00Z"
	hookTimestampSourceHookPayload   = "hook_payload"
	hookTimestampSourceCaptureTime   = "capture_time"
	hookTimestampSourceTranscriptEnd = "transcript_end"
	maxHookPayloadBytes              = 1 << 20
)

type claudeCodeHookPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      string `json:"timestamp"`
}

type captureEvent struct {
	SchemaVersion   string              `json:"schema_version"`
	EventID         string              `json:"event_id"`
	Source          string              `json:"source"`
	EventType       string              `json:"event_type"`
	Timestamp       string              `json:"timestamp"`
	TimestampSource string              `json:"timestamp_source,omitempty"`
	SessionRef      captureSessionRef   `json:"session_ref"`
	Workspace       captureWorkspaceRef `json:"workspace"`
	Raw             *captureEventRaw    `json:"raw,omitempty"`
}

type captureEventRaw struct {
	RawMissing bool   `json:"raw_missing,omitempty"`
	CopyStatus string `json:"copy_status,omitempty"`
	RawRefID   string `json:"raw_ref_id,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Kind       string `json:"kind,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
	CopyError  string `json:"copy_error,omitempty"`
}

type captureSessionRef struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

type captureWorkspaceRef struct {
	CWD string `json:"cwd"`
}

func hook(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	_ = stdout

	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing hook adapter")
		return 2
	}

	switch args[0] {
	case claudeCodeSource:
		if len(args) != 1 {
			fmt.Fprintln(stderr, "error: hook claude-code does not accept arguments")
			printUsage(stderr)
			return 2
		}
		paths, err := workspace.Resolve()
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		event, err := spoolClaudeCodeHook(stdin, vault.New(paths))
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		if warning := hookWarning(event); warning != "" {
			fmt.Fprintf(stderr, "warning: %s\n", warning)
		}
		return 0
	case "install":
		return hookInstall(args[1:], stdin, stdout, stderr)
	case "status":
		return hookStatus(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unsupported hook adapter %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func spoolClaudeCodeHook(stdin io.Reader, store vault.Store) (captureEvent, error) {
	payload, err := readClaudeCodeHookPayload(stdin)
	if err != nil {
		return captureEvent{}, err
	}

	eventType, err := claudeCodeEventType(payload.HookEventName)
	if err != nil {
		return captureEvent{}, err
	}

	event := newCaptureEvent(payload, eventType)
	if eventType == "session_end" {
		event.Raw = captureEventRawForPayload(payload, store, event.Timestamp)
	}
	if err := writeCaptureEvent(store.Paths.EventsDir(), event); err != nil {
		return captureEvent{}, err
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
		return captureEvent{}, err
	}
	return event, nil
}

func hookWarning(event captureEvent) string {
	if event.Raw == nil {
		return ""
	}
	switch {
	case event.Raw.RawMissing:
		return "capture recorded without transcript_path; preservation degraded"
	case event.Raw.CopyStatus == "failed":
		return fmt.Sprintf("capture recorded but transcript copy failed: %s", event.Raw.CopyError)
	default:
		return ""
	}
}

func captureEventRawForPayload(payload claudeCodeHookPayload, store vault.Store, observedAt string) *captureEventRaw {
	if strings.TrimSpace(payload.TranscriptPath) == "" {
		return &captureEventRaw{
			RawMissing: true,
			CopyStatus: "missing",
		}
	}

	archivePath, err := resolveHookTranscriptPath(payload)
	if err != nil {
		return &captureEventRaw{
			CopyStatus: "failed",
			CopyError:  err.Error(),
			Kind:       rawKindForTranscriptPath(payload.TranscriptPath),
		}
	}
	result, err := store.ArchiveFile(vault.ArchiveRequest{
		Source:          vault.SourceClaudeCode,
		SourceSessionID: payload.SessionID,
		Kind:            rawKindForTranscriptPath(archivePath),
		OriginalPath:    archivePath,
		ObservedAt:      observedAt,
	})
	if err != nil {
		return &captureEventRaw{
			CopyStatus: "failed",
			CopyError:  err.Error(),
			Kind:       rawKindForTranscriptPath(archivePath),
		}
	}
	status := "copied"
	if !result.BlobCreated {
		status = "deduped"
	}
	return &captureEventRaw{
		CopyStatus: status,
		RawRefID:   result.RawRef.RawRefID,
		Digest:     result.RawRef.Digest,
		Kind:       result.RawRef.Kind,
		SizeBytes:  result.RawRef.SizeBytes,
	}
}

func resolveHookTranscriptPath(payload claudeCodeHookPayload) (string, error) {
	path := strings.TrimSpace(payload.TranscriptPath)
	if path == "" {
		return "", fmt.Errorf("missing transcript_path")
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	candidates := make([]string, 0, 2)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if cwd := strings.TrimSpace(payload.CWD); cwd != "" {
		candidates = append(candidates, cwd)
	}
	var fallback string
	for _, base := range candidates {
		resolved := filepath.Clean(filepath.Join(base, path))
		if fallback == "" {
			fallback = resolved
		}
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			return resolved, nil
		}
	}
	if fallback == "" {
		fallback = filepath.Clean(path)
	}
	return fallback, nil
}

func rawKindForTranscriptPath(path string) string {
	if strings.Contains(filepath.ToSlash(path), "/subagents/") {
		return vault.KindSubagentTranscript
	}
	return vault.KindMainTranscript
}

func readClaudeCodeHookPayload(stdin io.Reader) (claudeCodeHookPayload, error) {
	data, err := io.ReadAll(io.LimitReader(stdin, maxHookPayloadBytes+1))
	if err != nil {
		return claudeCodeHookPayload{}, fmt.Errorf("read hook JSON: %w", err)
	}
	if len(data) > maxHookPayloadBytes {
		return claudeCodeHookPayload{}, fmt.Errorf("hook JSON exceeds %d bytes", maxHookPayloadBytes)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return claudeCodeHookPayload{}, fmt.Errorf("missing hook JSON on stdin")
	}

	var payload claudeCodeHookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return claudeCodeHookPayload{}, fmt.Errorf("invalid hook JSON: %w", err)
	}

	payload.SessionID = strings.TrimSpace(payload.SessionID)
	payload.TranscriptPath = strings.TrimSpace(payload.TranscriptPath)
	payload.CWD = strings.TrimSpace(payload.CWD)
	payload.HookEventName = strings.TrimSpace(payload.HookEventName)
	payload.Timestamp = strings.TrimSpace(payload.Timestamp)

	if payload.SessionID == "" {
		return claudeCodeHookPayload{}, fmt.Errorf("missing required hook field session_id")
	}
	if payload.CWD == "" {
		return claudeCodeHookPayload{}, fmt.Errorf("missing required hook field cwd")
	}
	if payload.HookEventName == "" {
		return claudeCodeHookPayload{}, fmt.Errorf("missing required hook field hook_event_name")
	}
	if payload.Timestamp != "" {
		if _, err := time.Parse(time.RFC3339Nano, payload.Timestamp); err != nil {
			return claudeCodeHookPayload{}, fmt.Errorf("invalid hook field timestamp: must be RFC3339")
		}
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
		return "", fmt.Errorf("unsupported Claude Code hook_event_name %q", hookEventName)
	}
}

func newCaptureEvent(payload claudeCodeHookPayload, eventType string) captureEvent {
	timestamp := payload.Timestamp
	timestampSource := hookTimestampSourceHookPayload
	if timestamp == "" {
		timestamp = currentTimestamp()
		timestampSource = hookTimestampSourceCaptureTime
	}

	event := captureEvent{
		SchemaVersion:   captureEventSchemaVersion,
		Source:          claudeCodeSource,
		EventType:       eventType,
		Timestamp:       timestamp,
		TimestampSource: timestampSource,
		SessionRef: captureSessionRef{
			SessionID:      payload.SessionID,
			TranscriptPath: payload.TranscriptPath,
		},
		Workspace: captureWorkspaceRef{
			CWD: payload.CWD,
		},
	}
	event.EventID = deterministicEventID(event)
	return event
}

func currentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func deterministicEventID(event captureEvent) string {
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
	return "evt_" + fmt.Sprintf("%x", hash.Sum(nil))[:32]
}

func writeCaptureEvent(eventsDir string, event captureEvent) error {
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return fmt.Errorf("create event spool %s: %w", filepath.ToSlash(eventsDir), err)
	}

	eventID, path, err := nextCaptureEventPath(eventsDir, event.EventID)
	if err != nil {
		return err
	}
	event.EventID = eventID

	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("encode capture event: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(eventsDir, "."+event.EventID+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary capture event: %w", err)
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
		return fmt.Errorf("write capture event: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set capture event permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close capture event: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("commit capture event %s: %w", filepath.ToSlash(path), err)
	}
	removeTmp = false

	return nil
}

func nextCaptureEventPath(eventsDir string, baseEventID string) (string, string, error) {
	for attempt := 1; ; attempt++ {
		eventID := baseEventID
		if attempt > 1 {
			eventID = fmt.Sprintf("%s_%d", baseEventID, attempt)
		}
		path := filepath.Join(eventsDir, eventID+".json")
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return eventID, path, nil
		}
		if err != nil {
			return "", "", fmt.Errorf("inspect capture event path %s: %w", filepath.ToSlash(path), err)
		}
	}
}

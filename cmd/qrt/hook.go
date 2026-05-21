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
)

const (
	captureEventSchemaVersion = "qratum.event.v1"
	claudeCodeSource          = "claude-code"
	defaultHookTimestamp      = "1970-01-01T00:00:00Z"
	maxHookPayloadBytes       = 1 << 20
)

type claudeCodeHookPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      string `json:"timestamp"`
}

type captureEvent struct {
	SchemaVersion string              `json:"schema_version"`
	EventID       string              `json:"event_id"`
	Source        string              `json:"source"`
	EventType     string              `json:"event_type"`
	Timestamp     string              `json:"timestamp"`
	SessionRef    captureSessionRef   `json:"session_ref"`
	Workspace     captureWorkspaceRef `json:"workspace"`
}

type captureSessionRef struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
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
	if args[0] != claudeCodeSource {
		fmt.Fprintf(stderr, "error: unsupported hook adapter %q\n", args[0])
		printUsage(stderr)
		return 2
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: hook claude-code does not accept arguments")
		printUsage(stderr)
		return 2
	}

	if err := spoolClaudeCodeHook(stdin, ".qratum/events"); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

func spoolClaudeCodeHook(stdin io.Reader, eventsDir string) error {
	payload, err := readClaudeCodeHookPayload(stdin)
	if err != nil {
		return err
	}

	eventType, err := claudeCodeEventType(payload.HookEventName)
	if err != nil {
		return err
	}

	event := newCaptureEvent(payload, eventType)
	return writeCaptureEvent(eventsDir, event)
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
	if payload.TranscriptPath == "" {
		return claudeCodeHookPayload{}, fmt.Errorf("missing required hook field transcript_path")
	}
	if payload.CWD == "" {
		return claudeCodeHookPayload{}, fmt.Errorf("missing required hook field cwd")
	}
	if payload.HookEventName == "" {
		return claudeCodeHookPayload{}, fmt.Errorf("missing required hook field hook_event_name")
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
	if timestamp == "" {
		timestamp = defaultHookTimestamp
	}

	event := captureEvent{
		SchemaVersion: captureEventSchemaVersion,
		Source:        claudeCodeSource,
		EventType:     eventType,
		Timestamp:     timestamp,
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

func deterministicEventID(event captureEvent) string {
	hash := sha256.New()
	for _, part := range []string{
		event.Source,
		event.EventType,
		event.Timestamp,
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
		return fmt.Errorf("create event spool %s: %w", eventsDir, err)
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
		return fmt.Errorf("commit capture event %s: %w", path, err)
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
			return "", "", fmt.Errorf("inspect capture event path %s: %w", path, err)
		}
	}
}

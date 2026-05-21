package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionPrintsFixedDevelopmentVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "qrt dev\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStatusWorksWithoutQratumDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"qratum status\n",
		"milestone: A\n",
		"version: dev\n",
		"qratum_dir: missing\n",
		"ready: true\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, missing %q", output, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStatusReportsPresentQratumDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir(".qratum", 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "qratum_dir: present\n") {
		t.Fatalf("stdout = %q, want qratum_dir present", stdout.String())
	}
}

func TestStatusFailsWhenQratumPathIsInvalid(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".qratum", []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: .qratum exists but is not a directory") {
		t.Fatalf("stderr = %q, want invalid .qratum error", stderr.String())
	}
}

func TestMissingCommandFailsVisibly(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(nil, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"usage: qrt", "error: missing command"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestUnsupportedCommandFailsVisibly(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"frobnicate"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `error: unsupported command "frobnicate"`) {
		t.Fatalf("stderr = %q, want unsupported command error", stderr.String())
	}
}

func TestVersionRejectsExtraArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--version", "extra"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: --version does not accept arguments") {
		t.Fatalf("stderr = %q, want extra argument error", stderr.String())
	}
}

func TestStatusRejectsExtraArguments(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	code := run([]string{"status", "extra"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: status does not accept arguments") {
		t.Fatalf("stderr = %q, want extra argument error", stderr.String())
	}
}

func TestHookClaudeCodeSpoolsCaptureEventFromFixture(t *testing.T) {
	input := readFixture(t, "hook-session-end.json")
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	code := runWithIO([]string{"hook", "claude-code"}, bytes.NewReader(input), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	files, err := filepath.Glob(".qratum/events/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("event files = %v, want exactly one", files)
	}

	var event captureEvent
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode event JSON: %v\n%s", err, data)
	}

	if got, want := filepath.Base(files[0]), event.EventID+".json"; got != want {
		t.Fatalf("event filename = %q, want %q", got, want)
	}
	if event.EventID == "" {
		t.Fatal("event_id is empty")
	}
	if got, want := event.SchemaVersion, "qratum.event.v1"; got != want {
		t.Fatalf("schema_version = %q, want %q", got, want)
	}
	if got, want := event.Source, "claude-code"; got != want {
		t.Fatalf("source = %q, want %q", got, want)
	}
	if got, want := event.EventType, "session_end"; got != want {
		t.Fatalf("event_type = %q, want %q", got, want)
	}
	if got, want := event.Timestamp, defaultHookTimestamp; got != want {
		t.Fatalf("timestamp = %q, want deterministic fallback %q", got, want)
	}
	if got, want := event.SessionRef.SessionID, "claude-session-0001"; got != want {
		t.Fatalf("session_ref.session_id = %q, want %q", got, want)
	}
	if got, want := event.SessionRef.TranscriptPath, "fixtures/claude-code/transcript-verification-gap.jsonl"; got != want {
		t.Fatalf("session_ref.transcript_path = %q, want %q", got, want)
	}
	if got, want := event.Workspace.CWD, "/Users/acartagena/project/qratum"; got != want {
		t.Fatalf("workspace.cwd = %q, want %q", got, want)
	}
}

func TestHookClaudeCodeToleratesUnknownFields(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer
	input := `{
		"session_id": "claude-session-unknown-fields",
		"transcript_path": "fixtures/claude-code/transcript-basic.jsonl",
		"cwd": "/tmp/qratum",
		"hook_event_name": "SessionStart",
		"unexpected": {"still": "accepted"}
	}`

	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	files, err := filepath.Glob(".qratum/events/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("event files = %v, want exactly one", files)
	}

	var event captureEvent
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode event JSON: %v", err)
	}
	if got, want := event.EventType, "session_start"; got != want {
		t.Fatalf("event_type = %q, want %q", got, want)
	}
	if got, want := event.SessionRef.TranscriptPath, "fixtures/claude-code/transcript-basic.jsonl"; got != want {
		t.Fatalf("transcript_path = %q, want %q", got, want)
	}
}

func TestHookClaudeCodeWritesOneNewEventPerCall(t *testing.T) {
	input := readFixture(t, "hook-session-end.json")
	t.Chdir(t.TempDir())

	for i := 0; i < 2; i++ {
		var stdout, stderr bytes.Buffer
		code := runWithIO([]string{"hook", "claude-code"}, bytes.NewReader(input), &stdout, &stderr)
		if code != 0 {
			t.Fatalf("call %d exit code = %d, want 0; stderr = %q", i+1, code, stderr.String())
		}
	}

	files, err := filepath.Glob(".qratum/events/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("event files = %v, want two files after two calls", files)
	}

	seen := map[string]bool{}
	for _, file := range files {
		var event captureEvent
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatalf("decode %s: %v", file, err)
		}
		if seen[event.EventID] {
			t.Fatalf("duplicate event_id %q in files %v", event.EventID, files)
		}
		seen[event.EventID] = true
		if got, want := filepath.Base(file), event.EventID+".json"; got != want {
			t.Fatalf("event filename = %q, want %q", got, want)
		}
	}
}

func TestHookClaudeCodeFailsOnInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "missing stdin",
			input:     "",
			wantError: "error: missing hook JSON on stdin",
		},
		{
			name:      "invalid JSON",
			input:     "{not json",
			wantError: "error: invalid hook JSON:",
		},
		{
			name: "missing transcript path",
			input: `{
				"session_id": "claude-session-0001",
				"cwd": "/tmp/qratum",
				"hook_event_name": "SessionEnd"
			}`,
			wantError: "error: missing required hook field transcript_path",
		},
		{
			name: "unsupported hook event",
			input: `{
				"session_id": "claude-session-0001",
				"transcript_path": "fixtures/claude-code/transcript-basic.jsonl",
				"cwd": "/tmp/qratum",
				"hook_event_name": "UnknownHook"
			}`,
			wantError: `error: unsupported Claude Code hook_event_name "UnknownHook"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			var stdout, stderr bytes.Buffer

			code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(tt.input), &stdout, &stderr)

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantError) {
				t.Fatalf("stderr = %q, missing %q", stderr.String(), tt.wantError)
			}
			if files, _ := filepath.Glob(".qratum/events/*.json"); len(files) != 0 {
				t.Fatalf("event files = %v, want none", files)
			}
		})
	}
}

func TestHookClaudeCodeRejectsBadArguments(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "missing adapter",
			args:      []string{"hook"},
			wantError: "error: missing hook adapter",
		},
		{
			name:      "unsupported adapter",
			args:      []string{"hook", "codex"},
			wantError: `error: unsupported hook adapter "codex"`,
		},
		{
			name:      "extra args",
			args:      []string{"hook", "claude-code", "extra"},
			wantError: "error: hook claude-code does not accept arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := runWithIO(tt.args, strings.NewReader(`{}`), &stdout, &stderr)

			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantError) {
				t.Fatalf("stderr = %q, missing %q", stderr.String(), tt.wantError)
			}
		})
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "claude-code", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/edictum-ai/qratum/internal/vault"
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

func TestStatusCreatesSecuredQratumDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"qratum status\n",
		"milestone: vault-first\n",
		"version: dev\n",
		"qratum_home: " + filepath.ToSlash(qratumHome) + "\n",
		"qratum_home_state: present\n",
		"vault_blobs: 0\n",
		"vault_refs: 0\n",
		"copy_failures: 0\n",
		"ready: true\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, missing %q", output, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertPathMode(t, qratumHome, 0o700)
}

func TestStatusReportsPresentQratumDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	if err := os.MkdirAll(qratumHome, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "qratum_home_state: present\n") {
		t.Fatalf("stdout = %q, want qratum_home present", stdout.String())
	}
	assertPathMode(t, qratumHome, 0o700)
}

func TestStatusFailsWhenQratumPathIsInvalid(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	if err := os.WriteFile(qratumHome, []byte("not a directory\n"), 0o644); err != nil {
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
	if !strings.Contains(stderr.String(), "not a directory") {
		t.Fatalf("stderr = %q, want invalid qratum home error", stderr.String())
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
	setTestQratumHome(t)
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
	t.Chdir(repoRoot(t))
	qratumHome := setTestQratumHome(t)
	var stdout, stderr bytes.Buffer
	started := time.Now().UTC().Add(-time.Second)

	code := runWithIO([]string{"hook", "claude-code"}, bytes.NewReader(input), &stdout, &stderr)
	ended := time.Now().UTC().Add(time.Second)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(qratumHome, "events", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("event files = %v, want exactly one", files)
	}
	assertPathMode(t, filepath.Join(qratumHome, "events"), 0o700)
	assertPathMode(t, files[0], 0o600)

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
	timestamp, err := time.Parse(time.RFC3339Nano, event.Timestamp)
	if err != nil {
		t.Fatalf("timestamp = %q, want RFC3339 timestamp: %v", event.Timestamp, err)
	}
	if timestamp.Before(started) || timestamp.After(ended) {
		t.Fatalf("timestamp = %q, want between %q and %q", event.Timestamp, started.Format(time.RFC3339Nano), ended.Format(time.RFC3339Nano))
	}
	if got, want := event.TimestampSource, hookTimestampSourceCaptureTime; got != want {
		t.Fatalf("timestamp_source = %q, want %q", got, want)
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
	if event.Raw == nil {
		t.Fatal("raw capture details are missing")
	}
	if got, want := event.Raw.CopyStatus, "copied"; got != want {
		t.Fatalf("raw.copy_status = %q, want %q", got, want)
	}
	if event.Raw.Digest == "" || event.Raw.RawRefID == "" {
		t.Fatalf("raw digest/ref missing: %#v", event.Raw)
	}
	if got, want := event.Raw.Kind, vault.KindMainTranscript; got != want {
		t.Fatalf("raw.kind = %q, want %q", got, want)
	}
	rawRefPath := filepath.Join(qratumHome, "raw", "refs", event.Raw.RawRefID+".json")
	if _, err := os.Stat(rawRefPath); err != nil {
		t.Fatalf("raw ref stat %s: %v", rawRefPath, err)
	}
	blobDigest := strings.TrimPrefix(event.Raw.Digest, "sha256:")
	blobPath := filepath.Join(qratumHome, "raw", "blobs", "sha256", blobDigest[:2], blobDigest)
	if _, err := os.Stat(blobPath); err != nil {
		t.Fatalf("blob stat %s: %v", blobPath, err)
	}
}

func TestHookClaudeCodeToleratesUnknownFields(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	var stdout, stderr bytes.Buffer
	input := `{
		"session_id": "claude-session-unknown-fields",
		"transcript_path": "fixtures/claude-code/transcript-basic.jsonl",
		"cwd": "/tmp/qratum",
		"hook_event_name": "SessionStart",
		"timestamp": "2026-05-21T20:00:00Z",
		"unexpected": {"still": "accepted"}
	}`

	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	files, err := filepath.Glob(filepath.Join(qratumHome, "events", "*.json"))
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
	if got, want := event.Timestamp, "2026-05-21T20:00:00Z"; got != want {
		t.Fatalf("timestamp = %q, want %q", got, want)
	}
	if got, want := event.TimestampSource, hookTimestampSourceHookPayload; got != want {
		t.Fatalf("timestamp_source = %q, want %q", got, want)
	}
	if got, want := event.SessionRef.TranscriptPath, "fixtures/claude-code/transcript-basic.jsonl"; got != want {
		t.Fatalf("transcript_path = %q, want %q", got, want)
	}
}

func TestHookClaudeCodeWritesOneNewEventPerCall(t *testing.T) {
	input := readFixture(t, "hook-session-end.json")
	t.Chdir(repoRoot(t))
	qratumHome := setTestQratumHome(t)

	for i := 0; i < 2; i++ {
		var stdout, stderr bytes.Buffer
		code := runWithIO([]string{"hook", "claude-code"}, bytes.NewReader(input), &stdout, &stderr)
		if code != 0 {
			t.Fatalf("call %d exit code = %d, want 0; stderr = %q", i+1, code, stderr.String())
		}
	}

	files, err := filepath.Glob(filepath.Join(qratumHome, "events", "*.json"))
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
	if refs, err := filepath.Glob(filepath.Join(qratumHome, "raw", "refs", "*.json")); err != nil {
		t.Fatal(err)
	} else if len(refs) != 1 {
		t.Fatalf("raw refs = %v, want one deduped ref", refs)
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
			name: "unsupported hook event",
			input: `{
				"session_id": "claude-session-0001",
				"transcript_path": "fixtures/claude-code/transcript-basic.jsonl",
				"cwd": "/tmp/qratum",
				"hook_event_name": "UnknownHook"
			}`,
			wantError: `error: unsupported Claude Code hook_event_name "UnknownHook"`,
		},
		{
			name:      "invalid timestamp",
			input:     `{"session_id":"claude-session-0001","transcript_path":"fixtures/claude-code/transcript-basic.jsonl","cwd":"/tmp/qratum","hook_event_name":"SessionEnd","timestamp":"not-time"}`,
			wantError: "error: invalid hook field timestamp: must be RFC3339",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			qratumHome := setTestQratumHome(t)
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
			if files, _ := filepath.Glob(filepath.Join(qratumHome, "events", "*.json")); len(files) != 0 {
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

func TestNormalizeClaudeCodeBasicFixtureMatchesGolden(t *testing.T) {
	t.Chdir(repoRoot(t))
	var stdout, stderr bytes.Buffer

	code := run([]string{"normalize", "fixtures/claude-code/transcript-basic.jsonl"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertJSONEqual(t, stdout.Bytes(), readFixture(t, "transcript-basic.normalized.golden.json"))
}

func TestNormalizeClaudeCodeVerificationGapFixtureMatchesGolden(t *testing.T) {
	t.Chdir(repoRoot(t))
	var stdout, stderr bytes.Buffer

	code := run([]string{"normalize", "fixtures/claude-code/transcript-verification-gap.jsonl"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertJSONEqual(t, stdout.Bytes(), readFixture(t, "transcript-verification-gap.normalized.golden.json"))
}

func TestRedactSecretSessionFixtureMatchesGolden(t *testing.T) {
	t.Chdir(repoRoot(t))
	var stdout, stderr bytes.Buffer

	code := run([]string{"redact", "fixtures/redaction/secret-session.input.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertJSONEqual(t, stdout.Bytes(), readRedactionFixture(t, "secret-session.redacted.golden.json"))

	output := stdout.String()
	for _, raw := range []string{
		"sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890",
		"supersecret",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0In0.signature",
		"/Users/acartagena/project/qratum/.env",
		"-----BEGIN PRIVATE KEY-----",
		"MIIEvQIBADANBgkqhkiG9w0BAQEFAASC",
		"fA9sD8f7Gh6Jk5Lm4Np3Qr2St1Uv0WxY",
	} {
		if strings.Contains(output, raw) {
			t.Fatalf("redacted output leaked %q:\n%s", raw, output)
		}
	}
	for _, want := range []string{
		`"type": "redaction.secret_detected"`,
		`"type": "redaction.path_redacted"`,
		`"pipeline_status": "redacted"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("redacted output = %s, missing %s", output, want)
		}
	}
	if got := strings.Count(output, "[REDACTED_PATH_001]"); got < 2 {
		t.Fatalf("[REDACTED_PATH_001] count = %d, want stable reuse in content and input", got)
	}
}

func TestRedactRejectsMissingAndInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T)
		wantCode  int
		wantError string
	}{
		{
			name:      "missing session argument",
			args:      []string{"redact"},
			wantCode:  2,
			wantError: "error: missing session path",
		},
		{
			name:      "extra argument",
			args:      []string{"redact", "one.json", "two.json"},
			wantCode:  2,
			wantError: "error: redact accepts exactly one session path",
		},
		{
			name:      "missing file",
			args:      []string{"redact", "missing.json"},
			wantCode:  1,
			wantError: "missing session missing.json",
		},
		{
			name:      "relative path escaping project",
			args:      []string{"redact", "../outside.json"},
			wantCode:  1,
			wantError: `session path "../outside.json" escapes current project`,
		},
		{
			name: "invalid JSON",
			args: []string{"redact", "bad.json"},
			setup: func(t *testing.T) {
				if err := os.WriteFile("bad.json", []byte("{not json"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: "invalid session JSON bad.json",
		},
		{
			name: "unsupported schema",
			args: []string{"redact", "bad-schema.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.session.v2","session_id":"ses","source":"claude-code","turns":[],"tool_calls":[],"file_changes":[],"commands":[]}`
				if err := os.WriteFile("bad-schema.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: `unsupported schema_version "qratum.session.v2"`,
		},
		{
			name: "missing session id",
			args: []string{"redact", "missing-session-id.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.session.v1","source":"claude-code","turns":[],"tool_calls":[],"file_changes":[],"commands":[]}`
				if err := os.WriteFile("missing-session-id.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: "is missing session_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tt.setup != nil {
				tt.setup(t)
			}
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
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

func TestEvidenceVerificationGapFixtureWritesBundle(t *testing.T) {
	root := t.TempDir()
	writeEvidenceFixture(t, root, "verification-gap.input.json")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	evidencePath := qratumSessionArtifact(qratumHome, "ses_0001", "evidence.json")
	var stdout, stderr bytes.Buffer

	code := run([]string{"evidence", "fixtures/evidence/verification-gap.input.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got, want := stdout.String(), "wrote "+filepath.ToSlash(evidencePath)+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	var bundle evidenceBundle
	evidenceData := []byte(readTextFile(t, evidencePath))
	assertJSONEqual(t, evidenceData, readEvidenceFixture(t, "verification-gap.evidence.golden.json"))
	if err := json.Unmarshal(evidenceData, &bundle); err != nil {
		t.Fatalf("decode evidence bundle: %v", err)
	}
	if got, want := bundle.SessionID, "ses_0001"; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if got, want := bundle.Summary.Status, evidenceStatusComplete; got != want {
		t.Fatalf("summary.status = %q, want %q", got, want)
	}
	if got, want := bundle.ArtifactPaths.Evidence, "sessions/ses_0001/evidence.json"; got != want {
		t.Fatalf("artifact_paths.evidence = %q, want %q", got, want)
	}
	assertFindingTypes(t, bundle.Findings, []string{
		findingOnlyFailedVerification,
		findingMissingFinalVerification,
		findingFinalEditAfterLastTest,
		findingRepeatedFailingCommand,
		findingSourceChangedWithoutTest,
	})
	if got, want := bundle.Findings[3].Summary, `"go test ./..." failed 2 times in this session.`; got != want {
		t.Fatalf("repeated command summary = %q, want %q", got, want)
	}
	if len(bundle.Findings[3].Evidence) != 2 {
		t.Fatalf("repeated command evidence count = %d, want 2", len(bundle.Findings[3].Evidence))
	}
	if !containsString(bundle.MissingEvidence, "successful verification command after 2026-05-21T21:55:00Z") {
		t.Fatalf("missing_evidence = %v, want final verification gap", bundle.MissingEvidence)
	}
}

func TestReviewVerificationGapEvidenceWritesReviewCard(t *testing.T) {
	root := t.TempDir()
	writeEvidenceFixture(t, root, "verification-gap.input.json")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	evidencePath := qratumSessionArtifact(qratumHome, "ses_0001", "evidence.json")
	reviewPath := qratumSessionArtifact(qratumHome, "ses_0001", "review.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"evidence", "fixtures/evidence/verification-gap.input.json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("evidence exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"review", "sessions/ses_0001/evidence.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got, want := stdout.String(), "wrote "+filepath.ToSlash(reviewPath)+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	var card reviewCard
	if _, err := os.Stat(evidencePath); err != nil {
		t.Fatalf("expected evidence path %s: %v", evidencePath, err)
	}
	reviewData := []byte(readTextFile(t, reviewPath))
	assertJSONEqual(t, reviewData, readReviewFixture(t, "verification-gap.review.golden.json"))
	if err := json.Unmarshal(reviewData, &card); err != nil {
		t.Fatalf("decode review card: %v", err)
	}
	if got, want := card.SessionID, "ses_0001"; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if got, want := card.Verdict, "needs_attention"; got != want {
		t.Fatalf("verdict = %q, want %q", got, want)
	}
	if !strings.Contains(card.MainFinding, "verification command(s) ran in this session and none succeeded") {
		t.Fatalf("main_finding = %q, want strongest verification finding", card.MainFinding)
	}
	if got := card.SuggestedNextHabit; strings.Contains(strings.ToLower(got), "score") || got == "" {
		t.Fatalf("suggested_next_habit = %q, want non-score habit", got)
	}
	if got, want := card.SuggestedSkill, "final-verification-loop"; got != want {
		t.Fatalf("suggested_skill = %q, want %q", got, want)
	}
	if len(card.Evidence) < 6 {
		t.Fatalf("review evidence = %v, want explicit evidence for all findings", card.Evidence)
	}
	if !containsString(card.Warnings, "successful verification command after 2026-05-21T21:55:00Z") {
		t.Fatalf("warnings = %v, want missing final verification warning", card.Warnings)
	}
	reviewText := readTextFile(t, reviewPath)
	for _, banned := range []string{"score", "ranking", "shame"} {
		if strings.Contains(strings.ToLower(reviewText), banned) {
			t.Fatalf("review contains banned language %q: %s", banned, reviewText)
		}
	}
}

func TestExportADPStrictFixtureMatchesGolden(t *testing.T) {
	root := t.TempDir()
	sessionPath := writeADPExportSessionFixture(t, root)
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	exportPath := qratumSessionArtifact(qratumHome, "ses_0001", "session.adp.jsonl")
	var stdout, stderr bytes.Buffer

	code := run([]string{"export", sessionPath, "--profile", "adp-strict"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got, want := stdout.String(), "wrote "+filepath.ToSlash(exportPath)+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	got := []byte(readTextFile(t, exportPath))
	if !bytes.Equal(got, readADPFixture(t, "session.adp-strict.golden.jsonl")) {
		t.Fatalf("ADP strict export mismatch\n got:\n%s\nwant:\n%s", got, readADPFixture(t, "session.adp-strict.golden.jsonl"))
	}

	output := string(got)
	for _, banned := range []string{
		`"session_id"`,
		`"artifact_paths"`,
		`"provenance"`,
		`"redaction"`,
		`"pipeline_status"`,
		`"source_event_id"`,
		`"x-qratum-`,
		`"secret_map"`,
		`"content":"package redaction\n\nfunc Redact`,
		`"old_string"`,
		`"new_string"`,
	} {
		if strings.Contains(output, banned) {
			t.Fatalf("ADP strict export contains banned Qratum-only or raw tool field %q: %s", banned, output)
		}
	}
}

func TestExportADPStrictRejectsMissingAndInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T, root string)
		wantCode  int
		wantError string
	}{
		{
			name:      "missing session argument",
			args:      []string{"export"},
			wantCode:  2,
			wantError: "error: missing session path",
		},
		{
			name:      "missing profile",
			args:      []string{"export", ".qratum/sessions/ses_0001.normalized.json"},
			wantCode:  2,
			wantError: "error: export accepts exactly one session path and --profile adp-strict",
		},
		{
			name:      "unsupported profile",
			args:      []string{"export", ".qratum/sessions/ses_0001.normalized.json", "--profile", "full-adp"},
			wantCode:  2,
			wantError: `error: unsupported export profile "full-adp"`,
		},
		{
			name:      "missing session file",
			args:      []string{"export", ".qratum/sessions/missing.normalized.json", "--profile", "adp-strict"},
			wantCode:  1,
			wantError: "missing session .qratum/sessions/missing.normalized.json",
		},
		{
			name: "invalid session JSON",
			args: []string{"export", ".qratum/sessions/bad.normalized.json", "--profile", "adp-strict"},
			setup: func(t *testing.T, root string) {
				path := filepath.Join(root, ".qratum", "sessions", "bad.normalized.json")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("{not json\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: "invalid session JSON .qratum/sessions/bad.normalized.json",
		},
		{
			name: "invalid timestamp",
			args: []string{"export", ".qratum/sessions/ses_0001.normalized.json", "--profile", "adp-strict"},
			setup: func(t *testing.T, root string) {
				sessionPath := writeADPExportSessionFixture(t, root)
				var session qratumSession
				readJSONFile(t, sessionPath, &session)
				session.Turns[0].Timestamp = "not-rfc3339"
				if err := os.WriteFile(sessionPath, mustJSON(session), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: "turns[0].timestamp: must be RFC3339",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			setTestQratumHome(t)
			if tt.setup != nil {
				tt.setup(t, root)
			}
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr = %q", code, tt.wantCode, stderr.String())
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

func TestEvidenceRejectsMissingAndInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T)
		wantCode  int
		wantError string
	}{
		{
			name:      "missing session argument",
			args:      []string{"evidence"},
			wantCode:  2,
			wantError: "error: missing redacted session path",
		},
		{
			name:      "extra argument",
			args:      []string{"evidence", "one.json", "two.json"},
			wantCode:  2,
			wantError: "error: evidence accepts exactly one redacted session path",
		},
		{
			name:      "missing file",
			args:      []string{"evidence", "missing.json"},
			wantCode:  1,
			wantError: "missing redacted session missing.json",
		},
		{
			name: "invalid timestamp",
			args: []string{"evidence", "bad-timestamp.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.session.v1","session_id":"ses_bad","source":"claude-code","turns":[],"tool_calls":[],"file_changes":[{"path":"a.go","operation":"edit","timestamp":"not-time"}],"commands":[],"business_metrics":{},"provenance":{}}`
				if err := os.WriteFile("bad-timestamp.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: "file_changes[0].timestamp must be RFC3339",
		},
		{
			name: "unsafe session id",
			args: []string{"evidence", "unsafe-session.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.session.v1","session_id":"bad/session","source":"claude-code","turns":[],"tool_calls":[],"file_changes":[],"commands":[],"business_metrics":{},"provenance":{}}`
				if err := os.WriteFile("unsafe-session.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: `session_id "bad/session" is not a safe artifact id`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setTestQratumHome(t)
			if tt.setup != nil {
				tt.setup(t)
			}
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
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

func TestEvidenceIgnoresInputArtifactPathsForOutput(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	data := `{"schema_version":"qratum.session.v1","session_id":"ses_bad","source":"claude-code","turns":[],"tool_calls":[],"file_changes":[],"commands":[],"artifact_paths":{"evidence":"../escape.evidence.json"},"business_metrics":{},"provenance":{}}`
	if err := os.WriteFile("repo-local-artifact-path.json", []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"evidence", "repo-local-artifact-path.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "wrote "+filepath.ToSlash(qratumSessionArtifact(qratumHome, "ses_bad", "evidence.json"))+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(qratumHome, "sessions", "ses_bad", "evidence.json")); err != nil {
		t.Fatalf("expected central evidence artifact: %v", err)
	}
	if _, err := os.Stat("../escape.evidence.json"); !os.IsNotExist(err) {
		t.Fatalf("repo-local escape artifact exists or stat failed unexpectedly: %v", err)
	}
}

func TestReviewRejectsMissingAndInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T)
		wantCode  int
		wantError string
	}{
		{
			name:      "missing evidence argument",
			args:      []string{"review"},
			wantCode:  2,
			wantError: "error: missing evidence path",
		},
		{
			name:      "extra argument",
			args:      []string{"review", "one.json", "two.json"},
			wantCode:  2,
			wantError: "error: review accepts exactly one evidence path",
		},
		{
			name:      "missing file",
			args:      []string{"review", "missing.json"},
			wantCode:  1,
			wantError: "missing evidence missing.json",
		},
		{
			name: "unsupported finding type",
			args: []string{"review", "bad-finding.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.evidence.v1","session_id":"ses_bad","summary":{"status":"complete","source":"claude-code"},"findings":[{"finding_id":"future.0001","type":"tool_risk.future","title":"future","summary":"future","evidence":[],"missing_evidence":[]}],"missing_evidence":[],"artifact_paths":{"review":".qratum/reviews/ses_bad.review.json"}}`
				if err := os.WriteFile("bad-finding.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: `unsupported findings[0].type "tool_risk.future"`,
		},
		{
			name: "unsupported status",
			args: []string{"review", "bad-status.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.evidence.v1","session_id":"ses_bad","summary":{"status":"placeholder_pending","source":"claude-code"},"findings":[],"missing_evidence":[]}`
				if err := os.WriteFile("bad-status.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: `unsupported summary.status "placeholder_pending"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setTestQratumHome(t)
			if tt.setup != nil {
				tt.setup(t)
			}
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
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

func TestNormalizeToleratesUnknownFieldsAndMissingOptionalTimestamps(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	transcript := strings.Join([]string{
		`{"type":"session_start","session_id":"claude-session-tolerant","model":"claude-sonnet-4-6","unknown":{"ignored":true}}`,
		`{"type":"user","content":"Run the tests.","extra":"ignored"}`,
		`{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."},"unexpected":[1,2,3]}`,
		`{"type":"tool_result","name":"Bash","success":true,"content":"ok ./...","timestamp":"2026-05-21T21:23:00Z"}`,
		`{"type":"session_end","session_id":"claude-session-tolerant","timestamp":"2026-05-21T21:24:00Z"}`,
	}, "\n") + "\n"
	if err := os.WriteFile("transcript.jsonl", []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"normalize", "transcript.jsonl"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var session qratumSession
	if err := json.Unmarshal(stdout.Bytes(), &session); err != nil {
		t.Fatalf("decode normalized session: %v\n%s", err, stdout.String())
	}
	if got, want := session.SessionID, "claude-session-tolerant"; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if got, want := session.AgentModel, "claude-sonnet-4-6"; got != want {
		t.Fatalf("agent_model = %q, want %q", got, want)
	}
	if got, want := len(session.Turns), 1; got != want {
		t.Fatalf("turn count = %d, want %d", got, want)
	}
	if got, want := session.Turns[0].Timestamp, ""; got != want {
		t.Fatalf("turn timestamp = %q, want missing optional timestamp to remain empty", got)
	}
	if got, want := len(session.Commands), 1; got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}
	if session.Commands[0].Success == nil || !*session.Commands[0].Success {
		t.Fatalf("command success = %v, want true", session.Commands[0].Success)
	}
	if got, want := session.BusinessMetrics.TestsRun, 1; got != want {
		t.Fatalf("tests_run = %d, want %d", got, want)
	}
}

func TestNormalizeFailsOnMalformedJSONL(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	transcript := strings.Join([]string{
		`{"type":"session_start","session_id":"claude-session-bad"}`,
		`{not json}`,
	}, "\n") + "\n"
	if err := os.WriteFile("bad.jsonl", []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"normalize", "bad.jsonl"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "line 2: invalid JSON") {
		t.Fatalf("stderr = %q, want malformed line error", stderr.String())
	}
}

func TestNormalizeToleratesUnsupportedRecordType(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	transcript := `{"type":"summary","session_id":"claude-session-drift","content":"future drift"}`
	if err := os.WriteFile("drift.jsonl", []byte(transcript+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"normalize", "drift.jsonl"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var session qratumSession
	if err := json.Unmarshal(stdout.Bytes(), &session); err != nil {
		t.Fatalf("decode normalized session: %v\n%s", err, stdout.String())
	}
	if got, want := session.SessionID, "claude-session-drift"; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if got := len(session.Turns); got != 0 {
		t.Fatalf("turn count = %d, want unknown record not to create turns", got)
	}
}

func TestNormalizeRejectsMissingAndInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantCode  int
		wantError string
	}{
		{
			name:      "missing transcript argument",
			args:      []string{"normalize"},
			wantCode:  2,
			wantError: "error: missing transcript path",
		},
		{
			name:      "extra argument",
			args:      []string{"normalize", "one.jsonl", "two.jsonl"},
			wantCode:  2,
			wantError: "error: normalize accepts exactly one transcript path",
		},
		{
			name:      "missing file",
			args:      []string{"normalize", "missing.jsonl"},
			wantCode:  1,
			wantError: `missing transcript missing.jsonl`,
		},
		{
			name:      "relative path escaping project",
			args:      []string{"normalize", "../outside.jsonl"},
			wantCode:  1,
			wantError: `relative path "../outside.jsonl" escapes current project`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
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

func TestDaemonRunOnceGeneratesPipelineArtifactsFromHookEvent(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"qratum daemon run-once\n",
		"events: 1\n",
		"processed: 1\n",
		"skipped: 0\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}

	eventPath := singleGlob(t, filepath.Join(qratumHome, "events", "*.json"))
	sessionPath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "normalized.json"))
	redactedPath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "redacted.json"))
	evidencePath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "evidence.json"))
	reviewPath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "review.json"))
	reportPath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "report.html"))
	exportPath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "session.adp.jsonl"))
	wantArtifactPaths := artifactPathsForStem("claude-session-0001")
	if got, want := filepath.ToSlash(sessionPath), filepath.ToSlash(filepath.Join(qratumHome, filepath.FromSlash(wantArtifactPaths.Session))); got != want {
		t.Fatalf("session artifact path = %q, want %q", got, want)
	}
	if got, want := filepath.ToSlash(redactedPath), filepath.ToSlash(filepath.Join(qratumHome, filepath.FromSlash(wantArtifactPaths.Redacted))); got != want {
		t.Fatalf("redacted artifact path = %q, want %q", got, want)
	}
	if got, want := filepath.ToSlash(evidencePath), filepath.ToSlash(filepath.Join(qratumHome, filepath.FromSlash(wantArtifactPaths.Evidence))); got != want {
		t.Fatalf("evidence artifact path = %q, want %q", got, want)
	}
	if got, want := filepath.ToSlash(reviewPath), filepath.ToSlash(filepath.Join(qratumHome, filepath.FromSlash(wantArtifactPaths.Review))); got != want {
		t.Fatalf("review artifact path = %q, want %q", got, want)
	}
	if got, want := filepath.ToSlash(reportPath), filepath.ToSlash(filepath.Join(qratumHome, filepath.FromSlash(wantArtifactPaths.Report))); got != want {
		t.Fatalf("report artifact path = %q, want %q", got, want)
	}
	if got, want := filepath.ToSlash(exportPath), filepath.ToSlash(filepath.Join(qratumHome, filepath.FromSlash(wantArtifactPaths.Export))); got != want {
		t.Fatalf("export artifact path = %q, want %q", got, want)
	}

	var event captureEvent
	readJSONFile(t, eventPath, &event)
	if event.Timestamp == "" {
		t.Fatal("event timestamp is empty")
	}
	if got := event.TimestampSource; got == "" {
		t.Fatal("event timestamp_source is empty")
	}

	var session qratumSession
	readJSONFile(t, sessionPath, &session)
	if got, want := session.SchemaVersion, qratumSessionSchemaVersion; got != want {
		t.Fatalf("session schema_version = %q, want %q", got, want)
	}
	if got, want := session.SessionID, "claude-session-0001"; got != want {
		t.Fatalf("session_id = %q, want source hook session id %q", got, want)
	}
	if session.SessionID == "claude-session-verify-gap" {
		t.Fatal("session_id came from transcript fixture instead of preserving hook source id")
	}
	if got, want := session.Source, claudeCodeSource; got != want {
		t.Fatalf("source = %q, want %q", got, want)
	}
	if got, want := session.SourceEventTimestamp, "2026-05-21T22:10:00Z"; got != want {
		t.Fatalf("source_event_timestamp = %q, want transcript end timestamp %q", got, want)
	}
	if got, want := session.SourceTranscriptSessionID, "claude-session-verify-gap"; got != want {
		t.Fatalf("source_transcript_session_id = %q, want %q", got, want)
	}
	if got, want := session.TranscriptPath, "fixtures/claude-code/transcript-verification-gap.jsonl"; got != want {
		t.Fatalf("transcript_path = %q, want %q", got, want)
	}
	if got, want := session.PipelineStatus, "normalized"; got != want {
		t.Fatalf("pipeline_status = %q, want %q", got, want)
	}
	if got, want := session.AgentModel, "claude-sonnet-4-6"; got != want {
		t.Fatalf("agent_model = %q, want %q", got, want)
	}
	if got, want := session.StartedAt, "2026-05-21T21:20:00Z"; got != want {
		t.Fatalf("started_at = %q, want %q", got, want)
	}
	if got, want := session.EndedAt, "2026-05-21T22:10:00Z"; got != want {
		t.Fatalf("ended_at = %q, want %q", got, want)
	}
	if got, want := len(session.Turns), 3; got != want {
		t.Fatalf("turn count = %d, want %d", got, want)
	}
	if got, want := len(session.ToolCalls), 5; got != want {
		t.Fatalf("tool call count = %d, want %d", got, want)
	}
	if got, want := len(session.FileChanges), 2; got != want {
		t.Fatalf("file change count = %d, want %d", got, want)
	}
	if got, want := session.FileChanges[1].Operation, "edit"; got != want {
		t.Fatalf("file_changes[1].operation = %q, want %q", got, want)
	}
	if got, want := len(session.Commands), 2; got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}
	if session.Commands[0].Success == nil || *session.Commands[0].Success {
		t.Fatalf("commands[0].success = %v, want false", session.Commands[0].Success)
	}
	if got, want := session.BusinessMetrics.DurationSeconds, 3000; got != want {
		t.Fatalf("duration_seconds = %d, want %d", got, want)
	}
	if session.ArtifactPaths == nil {
		t.Fatal("artifact_paths is nil, want generated artifact paths")
	}
	wantEventPath := filepath.ToSlash(filepath.Join("events", strings.TrimSuffix(filepath.Base(eventPath), ".json")+".json"))
	if got, want := session.ArtifactPaths.Event, wantEventPath; got != want {
		t.Fatalf("artifact_paths.event = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Session, wantArtifactPaths.Session; got != want {
		t.Fatalf("artifact_paths.session = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Redacted, wantArtifactPaths.Redacted; got != want {
		t.Fatalf("artifact_paths.redacted = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Evidence, wantArtifactPaths.Evidence; got != want {
		t.Fatalf("artifact_paths.evidence = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Review, wantArtifactPaths.Review; got != want {
		t.Fatalf("artifact_paths.review = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Report, wantArtifactPaths.Report; got != want {
		t.Fatalf("artifact_paths.report = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Export, wantArtifactPaths.Export; got != want {
		t.Fatalf("artifact_paths.export = %q, want %q", got, want)
	}

	var redacted qratumSession
	readJSONFile(t, redactedPath, &redacted)
	if got, want := redacted.PipelineStatus, "redacted"; got != want {
		t.Fatalf("redacted pipeline_status = %q, want %q", got, want)
	}
	if got, want := redacted.SessionID, session.SessionID; got != want {
		t.Fatalf("redacted session_id = %q, want %q", got, want)
	}
	if redacted.Redaction == nil {
		t.Fatal("redacted redaction summary is nil")
	}
	if got, want := redacted.Redaction.Status, "redacted"; got != want {
		t.Fatalf("redaction status = %q, want %q", got, want)
	}
	if got := redacted.Redaction.PathPlaceholders; got == 0 {
		t.Fatal("redaction path_placeholders = 0, want workspace cwd path redacted")
	}
	if strings.Contains(readTextFile(t, redactedPath), "/Users/acartagena/project/qratum") {
		t.Fatalf("redacted daemon artifact leaked raw workspace path: %s", readTextFile(t, redactedPath))
	}
	if got, want := redacted.ArtifactPaths.Redacted, wantArtifactPaths.Redacted; got != want {
		t.Fatalf("redacted artifact_paths.redacted = %q, want %q", got, want)
	}

	var evidence evidenceBundle
	readJSONFile(t, evidencePath, &evidence)
	if got, want := evidence.SchemaVersion, qratumEvidenceSchemaVersion; got != want {
		t.Fatalf("evidence schema_version = %q, want %q", got, want)
	}
	if got, want := evidence.SessionID, "claude-session-0001"; got != want {
		t.Fatalf("evidence session_id = %q, want %q", got, want)
	}
	if got, want := evidence.ArtifactPaths.Report, wantArtifactPaths.Report; got != want {
		t.Fatalf("evidence artifact_paths.report = %q, want %q", got, want)
	}
	if got, want := evidence.Summary.Status, evidenceStatusComplete; got != want {
		t.Fatalf("evidence summary.status = %q, want %q", got, want)
	}
	if got, want := evidence.Summary.SourceEventTimestamp, "2026-05-21T22:10:00Z"; got != want {
		t.Fatalf("evidence source_event_timestamp = %q, want %q", got, want)
	}
	assertFindingTypes(t, evidence.Findings, []string{
		findingOnlyFailedVerification,
		findingMissingFinalVerification,
		findingFinalEditAfterLastTest,
		findingRepeatedFailingCommand,
		findingSourceChangedWithoutTest,
	})
	if got, want := evidence.Summary.LastFileChangeAt, "2026-05-21T21:55:00Z"; got != want {
		t.Fatalf("last_file_change_at = %q, want %q", got, want)
	}
	if got, want := evidence.Summary.LastTestCommandAt, "2026-05-21T21:41:00Z"; got != want {
		t.Fatalf("last_test_command_at = %q, want %q", got, want)
	}
	if !containsString(evidence.MissingEvidence, "successful verification command after 2026-05-21T21:55:00Z") {
		t.Fatalf("missing_evidence = %v, want missing final verification evidence", evidence.MissingEvidence)
	}
	if got := readTextFile(t, evidencePath); strings.Contains(got, "score") || strings.Contains(got, "rank") {
		t.Fatalf("evidence contains score-like language: %s", got)
	}

	var review reviewCard
	readJSONFile(t, reviewPath, &review)
	if got, want := review.SchemaVersion, qratumReviewSchemaVersion; got != want {
		t.Fatalf("review schema_version = %q, want %q", got, want)
	}
	if got, want := review.Verdict, "needs_attention"; got != want {
		t.Fatalf("review verdict = %q, want %q", got, want)
	}
	if !strings.Contains(review.MainFinding, "verification command(s) ran in this session and none succeeded") {
		t.Fatalf("review main_finding = %q, want strongest verification finding", review.MainFinding)
	}
	if len(review.Evidence) == 0 {
		t.Fatal("review evidence is empty")
	}
	reviewText := readTextFile(t, reviewPath)
	for _, banned := range []string{"score", "rank", "shame"} {
		if strings.Contains(strings.ToLower(reviewText), banned) {
			t.Fatalf("review contains banned score-like language %q: %s", banned, reviewText)
		}
	}
	if !strings.Contains(reviewText, "successful verification command after 2026-05-21T21:55:00Z") {
		t.Fatalf("review = %s, want missing verification evidence", reviewText)
	}
	if got, want := review.ArtifactPaths.Evidence, wantArtifactPaths.Evidence; got != want {
		t.Fatalf("review artifact_paths.evidence = %q, want %q", got, want)
	}

	report := readTextFile(t, reportPath)
	for _, want := range []string{
		"<h2>Session summary</h2>",
		"<h2>Review card</h2>",
		"<h2>Evidence findings</h2>",
		"<h2>Missing evidence</h2>",
		"<h2>Redaction summary</h2>",
		"<h2>Artifacts</h2>",
		"<h2>Provenance digests</h2>",
		"sha256:",
		"successful verification command after 2026-05-21T21:55:00Z",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report = %q, missing %q", report, want)
		}
	}
	for _, banned := range []string{
		"Pipeline shell placeholder generated",
		"Implement deterministic redaction for obvious secrets.",
		`"turns": [`,
		`"tool_calls": [`,
		`"secret_map"`,
		"<script",
		"<link",
		`href="javascript:`,
	} {
		if strings.Contains(report, banned) {
			t.Fatalf("report contains banned content %q: %q", banned, report)
		}
	}

	line := strings.TrimSpace(readTextFile(t, exportPath))
	var adp adpStrictTrajectory
	if err := json.Unmarshal([]byte(line), &adp); err != nil {
		t.Fatalf("decode ADP strict JSONL: %v\n%s", err, line)
	}
	if got, want := adp.SchemaVersion, adpStrictSchemaVersion; got != want {
		t.Fatalf("adp schema_version = %q, want %q", got, want)
	}
	if got, want := adp.ID, "claude-session-0001"; got != want {
		t.Fatalf("adp id = %q, want %q", got, want)
	}
	if got, want := adp.Details.Source, claudeCodeSource; got != want {
		t.Fatalf("adp details.source = %q, want %q", got, want)
	}
	if got := line; strings.Contains(got, "placeholder") || strings.Contains(got, "x-qratum-") || strings.Contains(got, "secret_map") || strings.Contains(got, "provenance") || strings.Contains(got, "artifact_paths") {
		t.Fatalf("adp export contains placeholder or Qratum-only internals: %s", got)
	}
	for _, want := range []string{`"class_":"TextObservation"`, `"class_":"ApiAction"`, `"class_":"CodeAction"`, `"source":"agent"`, `"source":"environment"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("adp export = %s, missing %s", line, want)
		}
	}
}

func TestDaemonRunOnceIsIdempotentForCompletedEvents(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("first run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	before := qratumFiles(t)

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("second run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"processed: 0\n", "skipped: 1\n", "skipped_events:\n", "reason: already_processed\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
	after := qratumFiles(t)
	if got, want := strings.Join(after, "\n"), strings.Join(before, "\n"); got != want {
		t.Fatalf("qratum files changed after idempotent second run:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDaemonRunOnceSilentSkippedEventsWhenZero(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "skipped_events:") {
		t.Fatalf("stdout = %q, must not include skipped_events when skipped is 0", stdout.String())
	}
}

func TestHookClaudeCodeNeverWritesUnixZeroTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "missing timestamp uses capture time",
			input: `{
				"session_id": "claude-session-no-ts",
				"transcript_path": "fixtures/claude-code/transcript-basic.jsonl",
				"cwd": "/tmp/qratum",
				"hook_event_name": "SessionEnd"
			}`,
		},
		{
			name: "empty string timestamp uses capture time",
			input: `{
				"session_id": "claude-session-empty-ts",
				"transcript_path": "fixtures/claude-code/transcript-basic.jsonl",
				"cwd": "/tmp/qratum",
				"hook_event_name": "SessionEnd",
				"timestamp": ""
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(repoRoot(t))
			qratumHome := setTestQratumHome(t)
			var stdout, stderr bytes.Buffer
			code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(tt.input), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
			}
			files, err := filepath.Glob(filepath.Join(qratumHome, "events", "*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 1 {
				t.Fatalf("event files = %v, want exactly one", files)
			}
			var event captureEvent
			readJSONFile(t, files[0], &event)
			if event.Timestamp == "1970-01-01T00:00:00Z" || strings.HasPrefix(event.Timestamp, "1970-") {
				t.Fatalf("hook wrote Unix zero timestamp %q", event.Timestamp)
			}
			parsed, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil {
				t.Fatalf("invalid timestamp %q: %v", event.Timestamp, err)
			}
			if parsed.Year() < 2020 {
				t.Fatalf("timestamp %q is suspiciously old (year %d)", event.Timestamp, parsed.Year())
			}
		})
	}
}

func TestCleanReviewIncludesVerificationProof(t *testing.T) {
	t.Chdir(t.TempDir())
	bundle := evidenceBundle{
		SchemaVersion: qratumEvidenceSchemaVersion,
		SessionID:     "ses-clean-0001",
		Summary: evidenceBundleSummary{
			Status:                 evidenceStatusComplete,
			Source:                 claudeCodeSource,
			FilesChanged:           2,
			CommandsRun:            3,
			TestsRun:               1,
			LastFileChangeAt:       "2026-05-21T19:00:00Z",
			LastTestCommandAt:      "2026-05-21T19:05:00Z",
			LastSuccessfulVerifyAt: "2026-05-21T19:05:00Z",
		},
		Findings:        []evidenceFinding{},
		MissingEvidence: []string{},
	}
	card, err := buildReviewCard(bundle)
	if err != nil {
		t.Fatalf("buildReviewCard: %v", err)
	}
	if got, want := card.Verdict, "clean"; got != want {
		t.Fatalf("verdict = %q, want %q", got, want)
	}
	wantEvidence := []string{
		"files_changed: 2",
		"commands_run: 3",
		"tests_run: 1",
		"final_file_edit_at: 2026-05-21T19:00:00Z",
		"last_test_command_at: 2026-05-21T19:05:00Z",
		"last_successful_verification_at: 2026-05-21T19:05:00Z",
		"verification_after_final_edit: true",
	}
	joined := strings.Join(card.Evidence, "\n")
	for _, want := range wantEvidence {
		if !strings.Contains(joined, want) {
			t.Fatalf("clean review evidence missing %q:\n%s", want, joined)
		}
	}
	lowered := strings.ToLower(joined)
	for _, banned := range []string{"score", "rank", "shame"} {
		if strings.Contains(lowered, banned) {
			t.Fatalf("clean review contains banned language %q: %s", banned, joined)
		}
	}
}

func TestNewArtifactsAreSessionIDBased(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	expect := artifactPathsForStem("claude-session-0001")
	for _, path := range []string{expect.Session, expect.Redacted, expect.Evidence, expect.Review, expect.Report, expect.Export} {
		absPath := artifactAbsolutePath(root, path)
		if _, err := os.Stat(absPath); err != nil {
			t.Fatalf("expected session-id-based artifact %s: %v", path, err)
		}
		if strings.Contains(path, "evt_") {
			t.Fatalf("artifact path %s contains event-id prefix", path)
		}
	}
	assertPathMode(t, qratumSessionArtifact(qratumHome, "claude-session-0001", "normalized.json"), 0o600)
}

func TestExistingEventIDArtifactsRemainReadable(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	// Simulate legacy on-disk layout: rename artifacts to event-id-based paths,
	// rewrite session.ArtifactPaths to point at those legacy names, and confirm
	// the read pipeline still resolves them via the `evidence` command.
	eventPath := singleGlob(t, filepath.Join(qratumHome, "events", "*.json"))
	var event captureEvent
	readJSONFile(t, eventPath, &event)
	legacyStem := event.EventID

	legacyPaths := artifactPathsForStem(legacyStem)
	legacyPaths.Event = filepath.ToSlash(filepath.Join("events", legacyStem+".json"))

	currentPaths := artifactPathsForStem("claude-session-0001")
	moveRename := func(from string, to string) {
		t.Helper()
		fromPath := artifactAbsolutePath(root, from)
		toPath := artifactAbsolutePath(root, to)
		if err := os.MkdirAll(filepath.Dir(toPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(fromPath, toPath); err != nil {
			t.Fatalf("rename %s -> %s: %v", from, to, err)
		}
	}
	moveRename(currentPaths.Session, legacyPaths.Session)
	moveRename(currentPaths.Redacted, legacyPaths.Redacted)
	moveRename(currentPaths.Evidence, legacyPaths.Evidence)
	moveRename(currentPaths.Review, legacyPaths.Review)
	moveRename(currentPaths.Report, legacyPaths.Report)
	moveRename(currentPaths.Export, legacyPaths.Export)

	// Patch the session.normalized.json to point ArtifactPaths at the legacy filenames.
	var session qratumSession
	readJSONFile(t, artifactAbsolutePath(root, legacyPaths.Session), &session)
	session.ArtifactPaths = &legacyPaths
	if err := os.WriteFile(artifactAbsolutePath(root, legacyPaths.Session), mustJSON(session), 0o600); err != nil {
		t.Fatal(err)
	}

	// `qrt evidence` rebuilds the bundle from the redacted artifact; it should resolve
	// the legacy paths from session.ArtifactPaths without crashing.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"evidence", legacyPaths.Redacted}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("evidence on legacy paths exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
}

func TestDaemonRunOnceFailsOnMissingTranscriptWithAPIError(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	writeMissingTranscriptEvent(t, qratumHome)
	var stdout, stderr bytes.Buffer

	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`"code": "daemon.run_once_failed"`,
		`missing transcript fixtures/claude-code/transcript-verification-gap.jsonl`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
	if files, _ := filepath.Glob(filepath.Join(qratumHome, "sessions", "*", "normalized.json")); len(files) != 0 {
		t.Fatalf("session artifacts = %v, want none", files)
	}
}

func TestDaemonRunOnceRejectsInvalidEventJSONWithAPIError(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	eventsDir := filepath.Join(qratumHome, "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	badPath := filepath.Join(eventsDir, "bad.json")
	if err := os.WriteFile(badPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`invalid capture event JSON ` + filepath.ToSlash(badPath),
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestDaemonRunOnceFailsWhenEventSpoolIsMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	qratumHome := setTestQratumHome(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`event spool ` + filepath.ToSlash(filepath.Join(qratumHome, "events")) + ` does not exist`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestDaemonRunOnceRejectsRelativeTranscriptPathEscapingProject(t *testing.T) {
	t.Chdir(t.TempDir())
	setTestQratumHome(t)
	var hookStdout, hookStderr bytes.Buffer
	input := `{
		"session_id": "escape-session",
		"transcript_path": "../outside.jsonl",
		"cwd": "/tmp/qratum",
		"hook_event_name": "SessionEnd"
	}`
	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &hookStdout, &hookStderr)
	if code != 0 {
		t.Fatalf("hook exit code = %d, want 0; stderr = %q", code, hookStderr.String())
	}

	var stdout, stderr bytes.Buffer
	code = run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"processed: 0\n",
		"skipped: 1\n",
		"reason: raw_copy_failed\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestDaemonRunOnceRejectsPartialArtifactSet(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("first run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	evidencePath := singleGlob(t, filepath.Join(qratumHome, "sessions", "*", "evidence.json"))
	if err := os.Remove(evidencePath); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"daemon", "run-once"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`partial artifacts for event`,
		artifactPathsForStem("claude-session-0001").Evidence,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestSessionsListPrintsGeneratedSessionArtifacts(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("daemon exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"sessions", "list"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("sessions list exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	wantPath := filepath.ToSlash(qratumSessionArtifact(qratumHome, "claude-session-0001", "normalized.json"))
	if !strings.Contains(stdout.String(), "claude-session-0001\t"+wantPath) {
		t.Fatalf("stdout = %q, want generated session entry", stdout.String())
	}
}

func TestDaemonRunOnceRejectsBadArguments(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "missing daemon command",
			args:      []string{"daemon"},
			wantError: "error: missing daemon command",
		},
		{
			name:      "unsupported daemon command",
			args:      []string{"daemon", "forever"},
			wantError: `error: unsupported daemon command "forever"`,
		},
		{
			name:      "extra args",
			args:      []string{"daemon", "run-once", "extra"},
			wantError: "error: daemon run-once does not accept arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

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
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "claude-code", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readRedactionFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "redaction", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readEvidenceFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "evidence", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readReviewFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "review", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func writeClaudeFixture(t *testing.T, root string, name string) {
	t.Helper()
	target := filepath.Join(root, "fixtures", "claude-code", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, readFixture(t, name), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeEvidenceFixture(t *testing.T, root string, name string) {
	t.Helper()
	target := filepath.Join(root, "fixtures", "evidence", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, readEvidenceFixture(t, name), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeADPExportSessionFixture(t *testing.T, root string) string {
	t.Helper()
	var session qratumSession
	if err := json.Unmarshal(readFixture(t, "transcript-verification-gap.normalized.golden.json"), &session); err != nil {
		t.Fatalf("decode normalized fixture: %v", err)
	}
	session.SessionID = "ses_0001"
	session.ArtifactPaths = nil
	session.SourceEventID = ""
	session.SourceEventType = ""
	session.SourceEventTimestamp = ""
	session.SourceTranscriptSessionID = ""
	session.TranscriptPath = ""
	session.PipelineStatus = ""
	path := filepath.Join(root, ".qratum", "sessions", "ses_0001.normalized.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, mustJSON(session), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(path)
}

func spoolHookFixture(t *testing.T, fixture string) {
	t.Helper()
	setTestQratumHome(t)
	var stdout, stderr bytes.Buffer
	code := runWithIO([]string{"hook", "claude-code"}, bytes.NewReader(readFixture(t, fixture)), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("hook exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("hook stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("hook stderr = %q, want empty", stderr.String())
	}
}

func writeMissingTranscriptEvent(t *testing.T, qratumHome string) {
	t.Helper()
	event := captureEvent{
		SchemaVersion:   captureEventSchemaVersion,
		EventID:         "evt_missing_transcript",
		Source:          claudeCodeSource,
		EventType:       "session_end",
		Timestamp:       "2026-06-15T00:00:00Z",
		TimestampSource: hookTimestampSourceCaptureTime,
		SessionRef: captureSessionRef{
			SessionID:      "missing-transcript-session",
			TranscriptPath: "fixtures/claude-code/transcript-verification-gap.jsonl",
		},
		Workspace: captureWorkspaceRef{CWD: "/tmp/qratum"},
	}
	eventPath := filepath.Join(qratumHome, "events", event.EventID+".json")
	if err := os.MkdirAll(filepath.Dir(eventPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventPath, mustJSON(event), 0o644); err != nil {
		t.Fatal(err)
	}
}

func singleGlob(t *testing.T, pattern string) string {
	t.Helper()
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("%s = %v, want exactly one file", pattern, files)
	}
	return files[0]
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode %s: %v\n%s", path, err, data)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertJSONEqual(t *testing.T, gotData []byte, wantData []byte) {
	t.Helper()
	var got any
	if err := json.Unmarshal(gotData, &got); err != nil {
		t.Fatalf("decode got JSON: %v\n%s", err, string(gotData))
	}
	var want any
	if err := json.Unmarshal(wantData, &want); err != nil {
		t.Fatalf("decode want JSON: %v\n%s", err, string(wantData))
	}
	if !reflect.DeepEqual(got, want) {
		gotPretty, _ := json.MarshalIndent(got, "", "  ")
		wantPretty, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("JSON mismatch\n got:\n%s\nwant:\n%s", gotPretty, wantPretty)
	}
}

func assertFindingTypes(t *testing.T, findings []evidenceFinding, want []string) {
	t.Helper()
	got := make([]string, len(findings))
	for i, finding := range findings {
		got[i] = finding.Type
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("finding types = %v, want %v", got, want)
	}
}

func setTestQratumHome(t *testing.T) string {
	t.Helper()
	if root := os.Getenv("QRATUM_HOME"); strings.TrimSpace(root) != "" {
		return root
	}
	root := filepath.Join(t.TempDir(), "qratum-home")
	t.Setenv("QRATUM_HOME", root)
	return root
}

func qratumSessionArtifact(qratumHome string, sessionID string, name string) string {
	return filepath.Join(qratumHome, "sessions", sessionID, name)
}

func assertPathMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}

func setTestHome(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", root)
	return root
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func qratumFiles(t *testing.T) []string {
	t.Helper()
	root := ".qratum"
	if qratumHome := strings.TrimSpace(os.Getenv("QRATUM_HOME")); qratumHome != "" {
		root = qratumHome
	}
	var files []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

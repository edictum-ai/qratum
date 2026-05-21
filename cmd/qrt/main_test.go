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
	var stdout, stderr bytes.Buffer

	code := run([]string{"evidence", "fixtures/evidence/verification-gap.input.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got, want := stdout.String(), "wrote .qratum/evidence/ses_0001.evidence.json\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	var bundle evidenceBundle
	evidenceData := []byte(readTextFile(t, ".qratum/evidence/ses_0001.evidence.json"))
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
	if got, want := bundle.ArtifactPaths.Evidence, ".qratum/evidence/ses_0001.evidence.json"; got != want {
		t.Fatalf("artifact_paths.evidence = %q, want %q", got, want)
	}
	assertFindingTypes(t, bundle.Findings, []string{
		findingFinalEditAfterLastTest,
		findingMissingFinalVerification,
		findingRepeatedFailingCommand,
	})
	if got, want := bundle.Findings[2].Summary, `"go test ./..." failed 2 times in this session.`; got != want {
		t.Fatalf("repeated command summary = %q, want %q", got, want)
	}
	if len(bundle.Findings[2].Evidence) != 2 {
		t.Fatalf("repeated command evidence count = %d, want 2", len(bundle.Findings[2].Evidence))
	}
	if !containsString(bundle.MissingEvidence, "successful verification command after 2026-05-21T21:55:00Z") {
		t.Fatalf("missing_evidence = %v, want final verification gap", bundle.MissingEvidence)
	}
}

func TestReviewVerificationGapEvidenceWritesReviewCard(t *testing.T) {
	root := t.TempDir()
	writeEvidenceFixture(t, root, "verification-gap.input.json")
	t.Chdir(root)

	var stdout, stderr bytes.Buffer
	code := run([]string{"evidence", "fixtures/evidence/verification-gap.input.json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("evidence exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"review", ".qratum/evidence/ses_0001.evidence.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if got, want := stdout.String(), "wrote .qratum/reviews/ses_0001.review.json\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	var card reviewCard
	reviewData := []byte(readTextFile(t, ".qratum/reviews/ses_0001.review.json"))
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
	if !strings.Contains(card.MainFinding, "internal/redaction/redactor.go changed via edit") {
		t.Fatalf("main_finding = %q, want final edit finding", card.MainFinding)
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
	reviewText := readTextFile(t, ".qratum/reviews/ses_0001.review.json")
	for _, banned := range []string{"score", "ranking", "shame"} {
		if strings.Contains(strings.ToLower(reviewText), banned) {
			t.Fatalf("review contains banned language %q: %s", banned, reviewText)
		}
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
		{
			name: "artifact path escapes project",
			args: []string{"evidence", "bad-artifact-path.json"},
			setup: func(t *testing.T) {
				data := `{"schema_version":"qratum.session.v1","session_id":"ses_bad","source":"claude-code","turns":[],"tool_calls":[],"file_changes":[],"commands":[],"artifact_paths":{"evidence":"../escape.evidence.json"},"business_metrics":{},"provenance":{}}`
				if err := os.WriteFile("bad-artifact-path.json", []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: `evidence output path "../escape.evidence.json" escapes current project`,
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

func TestNormalizeRejectsUnsupportedRecordType(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	transcript := `{"type":"summary","session_id":"claude-session-drift","content":"future drift"}`
	if err := os.WriteFile("drift.jsonl", []byte(transcript+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"normalize", "drift.jsonl"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `unsupported transcript record type "summary"`) {
		t.Fatalf("stderr = %q, want unsupported record type error", stderr.String())
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

func TestDaemonRunOnceGeneratesPlaceholderArtifactsFromHookEvent(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
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

	sessionPath := singleGlob(t, ".qratum/sessions/*.normalized.json")
	redactedPath := singleGlob(t, ".qratum/redacted/*.redacted.json")
	evidencePath := singleGlob(t, ".qratum/evidence/*.evidence.json")
	reviewPath := singleGlob(t, ".qratum/reviews/*.review.json")
	reportPath := singleGlob(t, ".qratum/reports/*.html")
	exportPath := singleGlob(t, ".qratum/exports/*.adp.jsonl")

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
	if got, want := session.SourceEventTimestamp, defaultHookTimestamp; got != want {
		t.Fatalf("source_event_timestamp = %q, want deterministic hook timestamp %q", got, want)
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
	if got, want := session.ArtifactPaths.Session, filepath.ToSlash(sessionPath); got != want {
		t.Fatalf("artifact_paths.session = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Redacted, filepath.ToSlash(redactedPath); got != want {
		t.Fatalf("artifact_paths.redacted = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Evidence, filepath.ToSlash(evidencePath); got != want {
		t.Fatalf("artifact_paths.evidence = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Review, filepath.ToSlash(reviewPath); got != want {
		t.Fatalf("artifact_paths.review = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Report, filepath.ToSlash(reportPath); got != want {
		t.Fatalf("artifact_paths.report = %q, want %q", got, want)
	}
	if got, want := session.ArtifactPaths.Export, filepath.ToSlash(exportPath); got != want {
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
	if got, want := redacted.ArtifactPaths.Redacted, filepath.ToSlash(redactedPath); got != want {
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
	if got, want := evidence.ArtifactPaths.Report, filepath.ToSlash(reportPath); got != want {
		t.Fatalf("evidence artifact_paths.report = %q, want %q", got, want)
	}
	if got, want := evidence.Summary.Status, evidenceStatusComplete; got != want {
		t.Fatalf("evidence summary.status = %q, want %q", got, want)
	}
	if got, want := evidence.Summary.SourceEventTimestamp, defaultHookTimestamp; got != want {
		t.Fatalf("evidence source_event_timestamp = %q, want %q", got, want)
	}
	assertFindingTypes(t, evidence.Findings, []string{
		findingFinalEditAfterLastTest,
		findingMissingFinalVerification,
		findingRepeatedFailingCommand,
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
	if !strings.Contains(review.MainFinding, "after the last test command") {
		t.Fatalf("review main_finding = %q, want final edit finding", review.MainFinding)
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
	if got, want := review.ArtifactPaths.Evidence, filepath.ToSlash(evidencePath); got != want {
		t.Fatalf("review artifact_paths.evidence = %q, want %q", got, want)
	}

	report := readTextFile(t, reportPath)
	if !strings.Contains(report, "Pipeline shell placeholder generated") {
		t.Fatalf("report = %q, want placeholder message", report)
	}
	if strings.Contains(report, "FAIL ./internal/redaction") {
		t.Fatalf("report rendered raw transcript content: %q", report)
	}

	var adp adpPlaceholderRecord
	line := strings.TrimSpace(readTextFile(t, exportPath))
	if err := json.Unmarshal([]byte(line), &adp); err != nil {
		t.Fatalf("decode ADP placeholder JSONL: %v\n%s", err, line)
	}
	if got, want := adp.SessionID, "claude-session-0001"; got != want {
		t.Fatalf("adp session_id = %q, want %q", got, want)
	}
	if !adp.Placeholder {
		t.Fatal("adp placeholder flag = false, want true")
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
	for _, want := range []string{"processed: 0\n", "skipped: 1\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
	after := qratumFiles(t)
	if got, want := strings.Join(after, "\n"), strings.Join(before, "\n"); got != want {
		t.Fatalf("qratum files changed after idempotent second run:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDaemonRunOnceFailsOnMissingTranscriptWithAPIError(t *testing.T) {
	t.Chdir(t.TempDir())
	spoolHookFixture(t, "hook-session-end.json")
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
	if files, _ := filepath.Glob(".qratum/sessions/*.normalized.json"); len(files) != 0 {
		t.Fatalf("session artifacts = %v, want none", files)
	}
}

func TestDaemonRunOnceRejectsInvalidEventJSONWithAPIError(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".qratum/events", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".qratum/events/bad.json", []byte("{not json"), 0o644); err != nil {
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
		`invalid capture event JSON .qratum/events/bad.json`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestDaemonRunOnceFailsWhenEventSpoolIsMissing(t *testing.T) {
	t.Chdir(t.TempDir())
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
		`event spool .qratum/events does not exist`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestDaemonRunOnceRejectsRelativeTranscriptPathEscapingProject(t *testing.T) {
	t.Chdir(t.TempDir())
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

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`relative path \"../outside.jsonl\" escapes current project`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestDaemonRunOnceRejectsPartialArtifactSet(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	spoolHookFixture(t, "hook-session-end.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("first run exit code = %d, want 0; stderr = %q", code, stderr.String())
	}

	evidencePath := singleGlob(t, ".qratum/evidence/*.evidence.json")
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
		filepath.ToSlash(evidencePath),
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
	if !strings.Contains(stdout.String(), "claude-session-0001\t.qratum/sessions/") {
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

func spoolHookFixture(t *testing.T, fixture string) {
	t.Helper()
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
	var files []string
	if err := filepath.Walk(".qratum", func(path string, info os.FileInfo, err error) error {
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

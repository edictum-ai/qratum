package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDogfoodImportRealShapedFixtureWritesArtifacts(t *testing.T) {
	root := t.TempDir()
	writeDogfoodFixture(t, root, "real-shaped-transcript.jsonl")
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"dogfood", "import", "fixtures/dogfood/real-shaped-transcript.jsonl"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"qratum dogfood import\n",
		"session_id: dogfood-session-0001\n",
		"review_path: .qratum/reviews/dogfood-session-0001.review.json\n",
		"html_report_path: .qratum/reports/dogfood-session-0001.html\n",
		"adp_export_path: .qratum/exports/dogfood-session-0001.adp.jsonl\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}

	for _, path := range []string{
		".qratum/sessions/dogfood-session-0001.normalized.json",
		".qratum/redacted/dogfood-session-0001.redacted.json",
		".qratum/evidence/dogfood-session-0001.evidence.json",
		".qratum/reviews/dogfood-session-0001.review.json",
		".qratum/reports/dogfood-session-0001.html",
		".qratum/exports/dogfood-session-0001.adp.jsonl",
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("artifact %s is empty", path)
		}
	}

	review := []byte(readTextFile(t, ".qratum/reviews/dogfood-session-0001.review.json"))
	assertJSONEqual(t, review, readDogfoodFixture(t, "real-shaped-transcript.golden.review.json"))

	var session qratumSession
	readJSONFile(t, ".qratum/sessions/dogfood-session-0001.normalized.json", &session)
	if got, want := session.AgentModel, "claude-sonnet-4-20250514"; got != want {
		t.Fatalf("agent_model = %q, want %q", got, want)
	}
	if got, want := len(session.Turns), 6; got != want {
		t.Fatalf("turns = %d, want %d", got, want)
	}
	if got, want := len(session.ToolCalls), 4; got != want {
		t.Fatalf("tool_calls = %d, want %d", got, want)
	}
	if got, want := session.ToolCalls[0].ToolCallID, "toolu_read_readme"; got != want {
		t.Fatalf("first tool_call_id = %q, want real tool_use id %q", got, want)
	}
	if !strings.Contains(session.ToolCalls[0].Result, "Local trust pipeline.") {
		t.Fatalf("first tool result = %q, want preserved tool result content", session.ToolCalls[0].Result)
	}
	if got, want := len(session.FileChanges), 1; got != want {
		t.Fatalf("file_changes = %d, want %d", got, want)
	}
	if got, want := session.FileChanges[0].Operation, "edit"; got != want {
		t.Fatalf("file_changes[0].operation = %q, want %q", got, want)
	}
	if got, want := len(session.Commands), 2; got != want {
		t.Fatalf("commands = %d, want %d", got, want)
	}
	if session.Commands[0].Success == nil || *session.Commands[0].Success {
		t.Fatalf("first command success = %v, want false from tool_result is_error", session.Commands[0].Success)
	}
}

func TestDogfoodImportDoesNotCopyRawTranscriptContent(t *testing.T) {
	root := t.TempDir()
	writeDogfoodFixture(t, root, "real-shaped-transcript.jsonl")
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"dogfood", "import", "fixtures/dogfood/real-shaped-transcript.jsonl"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "RAW_TRANSCRIPT_COPY_CANARY") || strings.Contains(stderr.String(), "RAW_TRANSCRIPT_COPY_CANARY") {
		t.Fatalf("dogfood output printed raw transcript canary; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	assertQratumTreeDoesNotContain(t, ".qratum", []string{
		"RAW_TRANSCRIPT_COPY_CANARY",
		"/Users/example/dev/qratum",
	})
	if matches, err := filepath.Glob(".qratum/**/*.jsonl"); err != nil {
		t.Fatal(err)
	} else {
		for _, match := range matches {
			if strings.Contains(filepath.ToSlash(match), "real-shaped-transcript") {
				t.Fatalf("raw transcript appears to have been copied into .qratum: %s", match)
			}
		}
	}
}

func TestDogfoodLatestPrintsCompactReview(t *testing.T) {
	root := t.TempDir()
	writeDogfoodFixture(t, root, "real-shaped-transcript.jsonl")
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	code := run([]string{"dogfood", "import", "fixtures/dogfood/real-shaped-transcript.jsonl"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("dogfood import exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"dogfood", "latest"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"session_id: dogfood-session-0001\n",
		"verdict: needs_attention\n",
		"main_finding: [REDACTED_PATH_003] changed via edit",
		"suggested_next_habit: After the last edit, run the project's verification command and keep the result in the session.\n",
		"html_report_path: .qratum/reports/dogfood-session-0001.html\n",
		"adp_export_path: .qratum/exports/dogfood-session-0001.adp.jsonl\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestDogfoodImportMissingTranscriptFails(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	code := run([]string{"dogfood", "import", "missing.jsonl"}, &stdout, &stderr)

	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing transcript missing.jsonl") {
		t.Fatalf("stderr = %q, want missing transcript error", stderr.String())
	}
}

func TestDogfoodRejectsBadArguments(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "missing command",
			args:      []string{"dogfood"},
			wantError: "error: missing dogfood command",
		},
		{
			name:      "missing import path",
			args:      []string{"dogfood", "import"},
			wantError: "error: missing transcript path",
		},
		{
			name:      "unsupported command",
			args:      []string{"dogfood", "sync"},
			wantError: `error: unsupported dogfood command "sync"`,
		},
		{
			name:      "latest extra arg",
			args:      []string{"dogfood", "latest", "extra"},
			wantError: "error: dogfood latest does not accept arguments",
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

func readDogfoodFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "dogfood", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeDogfoodFixture(t *testing.T, root string, name string) {
	t.Helper()
	target := filepath.Join(root, "fixtures", "dogfood", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, readDogfoodFixture(t, name), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertQratumTreeDoesNotContain(t *testing.T, root string, banned []string) {
	t.Helper()
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, value := range banned {
			if strings.Contains(text, value) {
				t.Fatalf("%s contains banned raw transcript content %q:\n%s", path, value, text)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

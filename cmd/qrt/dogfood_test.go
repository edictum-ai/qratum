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
		"review_path: sessions/dogfood-session-0001/review.json\n",
		"html_report_path: sessions/dogfood-session-0001/report.html\n",
		"adp_export_path: sessions/dogfood-session-0001/session.adp.jsonl\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}

	for _, path := range []string{
		"sessions/dogfood-session-0001/normalized.json",
		"sessions/dogfood-session-0001/redacted.json",
		"sessions/dogfood-session-0001/evidence.json",
		"sessions/dogfood-session-0001/review.json",
		"sessions/dogfood-session-0001/report.html",
		"sessions/dogfood-session-0001/session.adp.jsonl",
	} {
		info, err := os.Stat(artifactAbsolutePath(root, path))
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("artifact %s is empty", path)
		}
	}

	review := []byte(readTextFile(t, artifactAbsolutePath(root, "sessions/dogfood-session-0001/review.json")))
	assertJSONEqual(t, review, readDogfoodFixture(t, "real-shaped-transcript.golden.review.json"))

	var session qratumSession
	readJSONFile(t, artifactAbsolutePath(root, "sessions/dogfood-session-0001/normalized.json"), &session)
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
	assertQratumTreeDoesNotContain(t, setTestQratumHome(t), []string{
		"RAW_TRANSCRIPT_COPY_CANARY",
		"/Users/example/dev/qratum",
	})
	if matches, err := filepath.Glob(filepath.Join(setTestQratumHome(t), "sessions", "*", "*.jsonl")); err != nil {
		t.Fatal(err)
	} else {
		for _, match := range matches {
			if strings.Contains(filepath.ToSlash(match), "real-shaped-transcript") {
				t.Fatalf("raw transcript appears to have been copied into qratum home: %s", match)
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
	cloneDogfoodArtifacts(t, artifactPathsForStem("normal-session-newer"), func(data []byte) []byte {
		s := string(data)
		s = strings.ReplaceAll(s, "dogfood-session-0001", "normal-session-newer")
		s = strings.ReplaceAll(s, `"pipeline_status": "dogfood_imported"`, `"pipeline_status": "normalized"`)
		s = strings.ReplaceAll(s, "2026-05-21T18:15:00Z", "2026-05-22T18:15:00Z")
		return []byte(s)
	})
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"dogfood", "latest"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if strings.Contains(stdout.String(), "normal-session-newer") {
		t.Fatalf("dogfood latest included a non-dogfood session:\n%s", stdout.String())
	}
	for _, want := range []string{
		"session_id: dogfood-session-0001\n",
		"verdict: needs_attention\n",
		"main_finding: 2 verification command(s) ran in this session and none succeeded.\n",
		"top_findings:\n",
		"- verification.only_failed_verification:",
		"- verification.missing_final_verification:",
		"- verification.final_edit_after_last_test:",
		"- reliability.repeated_failing_command:",
		"evidence:\n",
		"suggested_next_habit: After the last edit, run the project's verification command and keep the result in the session.\n",
		"html_report_path: sessions/dogfood-session-0001/report.html\n",
		"adp_export_path: sessions/dogfood-session-0001/session.adp.jsonl\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestDogfoodListShowsImportedSessionsNewestFirst(t *testing.T) {
	root := t.TempDir()
	writeDogfoodFixture(t, root, "real-shaped-transcript.jsonl")
	t.Chdir(root)

	importDogfood := func(t *testing.T) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		code := run([]string{"dogfood", "import", "fixtures/dogfood/real-shaped-transcript.jsonl"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("dogfood import exit code = %d, want 0; stderr = %q", code, stderr.String())
		}
	}
	importDogfood(t)

	// Add a second imported session with a different session_id and earlier ended_at so we can verify ordering.
	cloneDogfoodArtifacts(t, artifactPathsForStem("dogfood-session-older"), func(data []byte) []byte {
		s := string(data)
		s = strings.ReplaceAll(s, "dogfood-session-0001", "dogfood-session-older")
		s = strings.ReplaceAll(s, "2026-05-21T18:15:00Z", "2026-05-20T10:15:00Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:12:10Z", "2026-05-20T10:00:10Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:12:00Z", "2026-05-20T10:00:00Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:07:30Z", "2026-05-20T09:55:30Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:07:00Z", "2026-05-20T09:55:00Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:05:30Z", "2026-05-20T09:50:30Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:05:00Z", "2026-05-20T09:50:00Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:01:02Z", "2026-05-20T09:31:02Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:01:00Z", "2026-05-20T09:31:00Z")
		s = strings.ReplaceAll(s, "2026-05-21T18:00:00Z", "2026-05-20T09:30:00Z")
		return []byte(s)
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"dogfood", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("dogfood list exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	output := stdout.String()
	firstIdx := strings.Index(output, "session_id: dogfood-session-0001")
	secondIdx := strings.Index(output, "session_id: dogfood-session-older")
	if firstIdx < 0 || secondIdx < 0 {
		t.Fatalf("dogfood list output missing entries:\n%s", output)
	}
	if firstIdx > secondIdx {
		t.Fatalf("expected newest session first, got:\n%s", output)
	}
	for _, want := range []string{
		"source: claude-code\n",
		"verdict: needs_attention\n",
		"files_changed: 1\n",
		"commands_run: 2\n",
		"tests_run: 2\n",
		"report_path: sessions/dogfood-session-0001/report.html\n",
		"report_path: sessions/dogfood-session-older/report.html\n",
		"started_at:",
		"ended_at: 2026-05-21T18:15:00Z\n",
		"ended_at: 2026-05-20T10:15:00Z\n",
		"main_finding:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("dogfood list output missing %q:\n%s", want, output)
		}
	}
}

func TestDogfoodShowPrintsDetailWithArtifactPaths(t *testing.T) {
	root := t.TempDir()
	writeDogfoodFixture(t, root, "real-shaped-transcript.jsonl")
	t.Chdir(root)

	var stdout, stderr bytes.Buffer
	code := run([]string{"dogfood", "import", "fixtures/dogfood/real-shaped-transcript.jsonl"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"dogfood", "show", "dogfood-session-0001"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("show exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	for _, want := range []string{
		"session_id: dogfood-session-0001\n",
		"source: claude-code\n",
		"agent_model: claude-sonnet-4-20250514\n",
		"verdict: needs_attention\n",
		"main_finding:",
		"evidence:\n",
		"suggested_next_habit: After the last edit, run the project's verification command",
		"files_changed: 1\n",
		"commands_run: 2\n",
		"tests_run: 2\n",
		"last_file_change_at: 2026-05-21T18:12:00Z\n",
		"last_test_command_at: 2026-05-21T18:07:00Z\n",
		"last_successful_verification_at: -\n",
		"html_report_path: sessions/dogfood-session-0001/report.html\n",
		"adp_export_path: sessions/dogfood-session-0001/session.adp.jsonl\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("dogfood show output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestDogfoodShowMissingSessionFails(t *testing.T) {
	root := t.TempDir()
	writeDogfoodFixture(t, root, "real-shaped-transcript.jsonl")
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	code := run([]string{"dogfood", "import", "fixtures/dogfood/real-shaped-transcript.jsonl"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = run([]string{"dogfood", "show", "nope"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code for unknown session")
	}
	if !strings.Contains(stderr.String(), `session "nope" not found`) {
		t.Fatalf("stderr = %q, want not found error", stderr.String())
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
		{
			name:      "list extra arg",
			args:      []string{"dogfood", "list", "extra"},
			wantError: "error: dogfood list does not accept arguments",
		},
		{
			name:      "show missing session_id",
			args:      []string{"dogfood", "show"},
			wantError: "error: missing session_id",
		},
		{
			name:      "show extra arg",
			args:      []string{"dogfood", "show", "ses_0001", "extra"},
			wantError: "error: dogfood show accepts exactly one session_id",
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
	setTestQratumHome(t)
	target := filepath.Join(root, "fixtures", "dogfood", name)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, readDogfoodFixture(t, name), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cloneDogfoodArtifacts(t *testing.T, paths daemonArtifactPaths, mutate func([]byte) []byte) {
	t.Helper()
	projectRoot, err := currentProjectRoot()
	if err != nil {
		t.Fatal(err)
	}
	sourcePaths := artifactPathsForStem("dogfood-session-0001")
	for _, item := range []struct {
		source string
		target string
	}{
		{source: sourcePaths.Session, target: paths.Session},
		{source: sourcePaths.Redacted, target: paths.Redacted},
		{source: sourcePaths.Evidence, target: paths.Evidence},
		{source: sourcePaths.Review, target: paths.Review},
		{source: sourcePaths.Report, target: paths.Report},
		{source: sourcePaths.Export, target: paths.Export},
	} {
		sourcePath := artifactAbsolutePath(projectRoot, item.source)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		if mutate != nil {
			data = mutate(data)
		}
		targetPath := artifactAbsolutePath(projectRoot, item.target)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(targetPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
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

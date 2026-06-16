package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edictum-ai/qratum/internal/vault"
)

func TestHookClaudeCodeMissingTranscriptPathRecordsRawMissing(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	input := `{"session_id":"ses-missing-raw","cwd":"/tmp/qratum","hook_event_name":"SessionEnd"}`
	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning: capture recorded without transcript_path") {
		t.Fatalf("stderr = %q, want raw_missing warning", stderr.String())
	}

	eventPath := singleGlob(t, filepath.Join(qratumHome, "events", "*.json"))
	var event captureEvent
	readJSONFile(t, eventPath, &event)
	if event.Raw == nil || !event.Raw.RawMissing {
		t.Fatalf("event raw = %#v, want raw_missing", event.Raw)
	}
	if got, want := event.Raw.CopyStatus, "missing"; got != want {
		t.Fatalf("raw.copy_status = %q, want %q", got, want)
	}
}

func TestHookClaudeCodeCopyFailureRecordedAndSurfacedByDoctor(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	home := setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)
	var stdout, stderr bytes.Buffer

	input := `{"session_id":"ses-copy-fail","transcript_path":"missing.jsonl","cwd":"` + filepath.ToSlash(projectRoot) + `","hook_event_name":"SessionEnd"}`
	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "warning: capture recorded but transcript copy failed") {
		t.Fatalf("stderr = %q, want copy-failed warning", stderr.String())
	}

	eventPath := singleGlob(t, filepath.Join(qratumHome, "events", "*.json"))
	var event captureEvent
	readJSONFile(t, eventPath, &event)
	if event.Raw == nil || event.Raw.CopyStatus != "failed" {
		t.Fatalf("event raw = %#v, want failed copy status", event.Raw)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doctor exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"copy_failures: 1\n",
		"cloud_sessions: sessions that start and end on vendor infra are not captured in vault v1\n",
		"- global SessionEnd hook is not installed\n",
		"- copy failures recorded: 1\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output = %q, missing %q (home %s)", out, want, home)
		}
	}
}

func TestHookInstallShowsDiffAndIsIdempotent(t *testing.T) {
	home := setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, readVaultFixture(t, "global-settings.empty.json"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runWithIO([]string{"hook", "install"}, strings.NewReader("y\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"qratum hook install\n",
		"diff:\n",
		"+    \"SessionEnd\": [\n",
		"Apply changes? [y/N]: ",
		"installed: true\n",
		"changed: true\n",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
	assertJSONEqual(t, []byte(readTextFile(t, settingsPath)), readVaultFixture(t, "global-settings.installed.golden.json"))

	stdout.Reset()
	stderr.Reset()
	code = runWithIO([]string{"hook", "install"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second install exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "changed: false\n") {
		t.Fatalf("stdout = %q, want changed false", stdout.String())
	}
}

func TestHookInstallRejectsProjectLocalDoubleCapture(t *testing.T) {
	home := setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, readVaultFixture(t, "global-settings.empty.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	localSettingsPath := filepath.Join(projectRoot, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(localSettingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localSettingsPath, readVaultFixture(t, "project-settings.local-hook.json"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runWithIO([]string{"hook", "install"}, strings.NewReader("y\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "project-local SessionEnd hook already installed") {
		t.Fatalf("stderr = %q, want double-capture warning", stderr.String())
	}
	assertJSONEqual(t, []byte(readTextFile(t, settingsPath)), readVaultFixture(t, "global-settings.empty.json"))
}

func TestVaultBackfillIsIdempotent(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	home := setTestHome(t)
	projectsRoot := filepath.Join(home, ".claude", "projects", "demo")
	mainPath := filepath.Join(projectsRoot, "session-main.jsonl")
	subagentPath := filepath.Join(projectsRoot, "subagents", "session-subagent.jsonl")
	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(subagentPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, readFixture(t, "transcript-basic.jsonl"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subagentPath, readFixture(t, "transcript-verification-gap.jsonl"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "backfill"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("first backfill exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"transcripts_seen: 2\n", "archived: 2\n", "deduped: 0\n", "failed: 0\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("first backfill stdout = %q, missing %q", stdout.String(), want)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "backfill"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second backfill exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"transcripts_seen: 2\n", "archived: 0\n", "deduped: 2\n", "failed: 0\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("second backfill stdout = %q, missing %q", stdout.String(), want)
		}
	}

	refs, err := filepath.Glob(filepath.Join(qratumHome, "raw", "refs", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("raw refs = %v, want two refs", refs)
	}
}

func TestVaultArchiveSupportsKindsAndWritesRawRefFixture(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)

	sourcePath := filepath.Join(projectRoot, "source-metadata.json")
	if err := os.WriteFile(sourcePath, readVaultFixture(t, "source-metadata.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "archive", sourcePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("archive exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"kind: source_metadata\n", "files_seen: 1\n", "archived: 1\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}

	refPath := singleGlob(t, filepath.Join(qratumHome, "raw", "refs", "*.json"))
	var ref vault.RawRef
	readJSONFile(t, refPath, &ref)
	assertPathMode(t, filepath.Join(qratumHome, "raw", "refs"), 0o700)
	assertPathMode(t, refPath, 0o600)
	assertPathMode(t, filepath.Dir(filepath.FromSlash(ref.ArchivedPath)), 0o700)
	assertPathMode(t, filepath.FromSlash(ref.ArchivedPath), 0o600)
	assertPathMode(t, filepath.Join(qratumHome, "state"), 0o700)
	assertPathMode(t, filepath.Join(qratumHome, "state", "vault.json"), 0o600)
	if got, want := ref.Kind, vault.KindSourceMetadata; got != want {
		t.Fatalf("raw ref kind = %q, want %q", got, want)
	}
	if got, want := ref.Source, vault.SourceManual; got != want {
		t.Fatalf("raw ref source = %q, want %q", got, want)
	}

	wantTemplate := string(readVaultFixture(t, "raw-ref.source-metadata.golden.json.tmpl"))
	wantTemplate = strings.ReplaceAll(wantTemplate, "__RAW_REF_ID__", ref.RawRefID)
	wantTemplate = strings.ReplaceAll(wantTemplate, "__DIGEST__", ref.Digest)
	wantTemplate = strings.ReplaceAll(wantTemplate, "__ORIGINAL_PATH__", filepath.ToSlash(sourcePath))
	wantTemplate = strings.ReplaceAll(wantTemplate, "__ARCHIVED_PATH__", ref.ArchivedPath)
	wantTemplate = strings.ReplaceAll(wantTemplate, "\"observed_at\": \"2026-06-15T12:00:00Z\"", "\"observed_at\": \""+ref.ObservedAt+"\"")
	wantTemplate = strings.ReplaceAll(wantTemplate, "\"size_bytes\": 118", fmt.Sprintf("\"size_bytes\": %d", ref.SizeBytes))
	assertJSONEqual(t, mustJSON(ref), []byte(wantTemplate))
}

func TestVaultBackupVerifyCopiesWorkspace(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)

	sourcePath := filepath.Join(projectRoot, "source-metadata.json")
	if err := os.WriteFile(sourcePath, readVaultFixture(t, "source-metadata.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"vault", "archive", sourcePath}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("archive setup failed with exit code %d", code)
	}

	dest := filepath.Join(t.TempDir(), "backup")
	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "backup", "--verify", dest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("backup exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"verified: yes\n", "files_copied: "} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "raw", "refs")); err != nil {
		t.Fatalf("backup raw refs missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "state", "vault.json")); err != nil {
		t.Fatalf("backup state missing: %v", err)
	}
	if qratumHome == dest {
		t.Fatal("backup destination must differ from qratum home")
	}
}

func readVaultFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "vault", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

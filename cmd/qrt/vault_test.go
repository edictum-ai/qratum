package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/edictum-ai/qratum/internal/capture"
	"github.com/edictum-ai/qratum/internal/schedule"
	"github.com/edictum-ai/qratum/internal/vault"
	"github.com/edictum-ai/qratum/internal/workspace"
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

func TestHookClaudeCodeLowDiskRecordsLoudFailureAndDoctorWarning(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	home := setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)
	if err := os.MkdirAll(qratumHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(qratumHome, "config.toml"), []byte("[worker]\ndisk_free_min_gb = 999999999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	transcriptPath := filepath.Join(projectRoot, "transcript.jsonl")
	if err := os.WriteFile(transcriptPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	input := `{"session_id":"ses-low-disk","transcript_path":"transcript.jsonl","cwd":"` + filepath.ToSlash(projectRoot) + `","hook_event_name":"SessionEnd"}`
	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "warning: capture recorded but transcript copy failed: disk free below configured minimum") {
		t.Fatalf("stderr = %q, want low-disk warning", stderr.String())
	}
	eventPath := singleGlob(t, filepath.Join(qratumHome, "events", "*.json"))
	var event captureEvent
	readJSONFile(t, eventPath, &event)
	if event.Raw == nil || event.Raw.CopyStatus != "failed" || !strings.Contains(event.Raw.CopyError, "disk free below configured minimum") {
		t.Fatalf("event raw = %#v, want recorded low-disk failure", event.Raw)
	}
	assertNoVaultBlobsOrRefs(t, qratumHome)

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doctor exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"disk_free_status: low\n",
		"copy_failures: 1\n",
		"- disk free below configured minimum:",
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

func TestVaultInstallSchedulePrintWritesNothing(t *testing.T) {
	scheduleDir := t.TempDir()
	t.Setenv(schedule.DirEnv, scheduleDir)
	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "install-schedule", "--print", "--platform", "darwin"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("install-schedule --print exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"qratum vault install-schedule\n",
		"dry_run: yes\n",
		"changed: no\n",
		"command: ",
		" vault backfill\n",
		"<key>StartInterval</key>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout = %q, missing %q", out, want)
		}
	}
	entries, err := os.ReadDir(scheduleDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("schedule dir entries = %v, want dry-run to write nothing", entries)
	}
}

func TestVaultInstallScheduleFakeDirIdempotentAndUninstallClean(t *testing.T) {
	scheduleDir := t.TempDir()
	t.Setenv(schedule.DirEnv, scheduleDir)
	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "install-schedule", "--platform", "darwin"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("install-schedule exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "changed: yes\n") {
		t.Fatalf("first install stdout = %q, want changed yes", stdout.String())
	}
	path := filepath.Join(scheduleDir, schedule.Label+".plist")
	firstBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "install-schedule", "--platform", "darwin"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second install-schedule exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "changed: no\n") || !strings.Contains(stdout.String(), "already_installed: yes\n") {
		t.Fatalf("second install stdout = %q, want no change", stdout.String())
	}
	secondBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("schedule file changed across idempotent reinstall")
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "uninstall-schedule", "--platform", "darwin"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("uninstall-schedule exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed: 1\n") {
		t.Fatalf("uninstall stdout = %q, want removed 1", stdout.String())
	}
	entries, err := os.ReadDir(scheduleDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("schedule dir entries after uninstall = %v, want empty", entries)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "uninstall-schedule", "--platform", "darwin"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second uninstall-schedule exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed: 0\n") || !strings.Contains(stdout.String(), "nothing_to_remove: yes\n") {
		t.Fatalf("second uninstall stdout = %q, want clean no-op", stdout.String())
	}
}

func TestVaultDoctorHealthyUsesEventDriftHeuristicAndInstalledSchedule(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	scheduleDir := t.TempDir()
	t.Setenv(schedule.DirEnv, scheduleDir)
	home := setTestHome(t)
	projectRoot := t.TempDir()
	t.Chdir(projectRoot)
	fixedNow := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	restoreNow := setNowForTest(t, fixedNow)
	defer restoreNow()

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, readVaultFixture(t, "global-settings.installed.golden.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := schedule.Install(schedule.Options{ScheduleDir: scheduleDir}); err != nil {
		t.Fatal(err)
	}
	paths, err := workspace.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	store := vault.New(paths)
	for i, body := range []string{"first\n", "second\n"} {
		source := filepath.Join(t.TempDir(), fmt.Sprintf("transcript-%d.jsonl", i))
		if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		status := "copied"
		if i == 1 {
			status = "deduped"
		}
		result, err := store.ArchiveFile(vault.ArchiveRequest{
			Source:          vault.SourceClaudeCode,
			SourceSessionID: fmt.Sprintf("session-%d", i),
			Kind:            vault.KindMainTranscript,
			OriginalPath:    source,
			ObservedAt:      fixedNow.Format(time.RFC3339Nano),
		})
		if err != nil {
			t.Fatal(err)
		}
		writeCaptureEventForDoctor(t, qratumHome, fmt.Sprintf("evt_%d", i), fmt.Sprintf("session-%d", i), result.RawRef.RawRefID, result.RawRef.Digest, status)
	}
	if err := store.SaveState(vault.State{
		LastBackfillAt:         fixedNow.Add(-time.Hour).Format(time.RFC3339Nano),
		LastBackupVerifiedAt:   fixedNow.Add(-time.Hour).Format(time.RFC3339Nano),
		LastBackupVerifiedDest: "/tmp/backup",
	}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doctor exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"hook_installed: yes\n",
		"schedule_installed: yes\n",
		"transcript_drift (heuristic): +0 (expected=2 archived=2)\n",
		"cloud_sessions: sessions that start and end on vendor infra are not captured in vault v1\n",
		"warnings: none\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output = %q, missing %q", out, want)
		}
	}
}

func TestTrustJSONUsesConfiguredQRTBinary(t *testing.T) {
	t.Chdir(repoRoot(t))
	t.Setenv("QRATUM_QRT_BIN", writeFakeTrustQRT(t))
	var stdout, stderr bytes.Buffer
	code := run([]string{"trust", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("trust exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		`"schema_version": "qratum.trust_scorecard.v1"`,
		`"headline": "TRUSTED-WITH-NAMED-GAPS"`,
		`"state": "KNOWN-RED"`,
		`"honest_residual"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("trust JSON = %q, missing %q", out, want)
		}
	}
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

func TestVaultArchiveMemoryImportReceiptRoundTripPinsKind(t *testing.T) {
	qratumHome := setTestQratumHome(t)
	setTestHome(t)
	t.Chdir(repoRoot(t))
	receiptPath := filepath.Join(repoRoot(t), "fixtures", "memory-import", "synthetic-receipt.json")
	receiptBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "archive", "--kind", vault.KindMemoryImport, receiptPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("archive exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "kind: memory_import_receipt\n") {
		t.Fatalf("stdout = %q, want pinned memory_import_receipt kind", stdout.String())
	}
	refPath := singleGlob(t, filepath.Join(qratumHome, "raw", "refs", "*.json"))
	var ref vault.RawRef
	readJSONFile(t, refPath, &ref)
	if ref.Kind != vault.KindMemoryImport {
		t.Fatalf("raw ref kind = %q, want %q", ref.Kind, vault.KindMemoryImport)
	}
	blobBytes, err := os.ReadFile(filepath.FromSlash(ref.ArchivedPath))
	if err != nil {
		t.Fatal(err)
	}
	if string(blobBytes) != string(receiptBytes) {
		t.Fatalf("archived receipt blob changed\n got:\n%s\nwant:\n%s", blobBytes, receiptBytes)
	}
}

func TestVaultArchiveMemoryImportReceiptRejectsSchemaViolations(t *testing.T) {
	setTestQratumHome(t)
	setTestHome(t)
	t.Chdir(repoRoot(t))
	tests := []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{
			name: "unknown-content-class",
			edit: func(doc map[string]any) { doc["content_class"] = "invented_class" },
			want: "is not in enum",
		},
		{
			name: "out-of-vocabulary-outcome",
			edit: func(doc map[string]any) { doc["outcome"] = "stored" },
			want: "is not in enum",
		},
		{
			name: "out-of-vocabulary-error-class",
			edit: func(doc map[string]any) {
				doc["decision"] = "denied"
				doc["error_class"] = "fabricated_gateway_state"
				delete(doc, "outcome")
			},
			want: "is not in enum",
		},
		{
			name: "namespace-forbidden",
			edit: func(doc map[string]any) {
				doc["decision"] = "denied"
				doc["error_class"] = "namespace_forbidden"
				delete(doc, "outcome")
			},
			want: "namespace_forbidden",
		},
		{
			name: "malformed-shape-fail-closed",
			edit: func(doc map[string]any) {
				delete(doc, "memory_ids")
			},
			want: `missing required property "memory_ids"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := readJSONMapFromFile(t, filepath.Join(repoRoot(t), "fixtures", "memory-import", "synthetic-receipt.json"))
			tt.edit(receipt)
			path := filepath.Join(t.TempDir(), tt.name+".receipt.json")
			if err := os.WriteFile(path, mustJSON(receipt), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := run([]string{"vault", "archive", "--kind", vault.KindMemoryImport, path}, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("archive exit code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, missing %q", stderr.String(), tt.want)
			}
		})
	}
}

func TestVaultArchiveMemoryImportReceiptGatewayRoundTripGated(t *testing.T) {
	t.Skip("GATED: requires a deployed personal-memory gateway producer; synthetic receipts are contract fixtures, not leak proofs")
}

func TestVaultArchiveMemoryImportReceiptSupersedesIdempotentGatewayGated(t *testing.T) {
	t.Skip("GATED: idempotent supersedes[] proof requires real memory_import_receipt files from personal-memory Phase 1+")
}

func TestVaultArchiveMemoryImportReceiptDefaultKindWarning(t *testing.T) {
	setTestQratumHome(t)
	setTestHome(t)
	t.Chdir(repoRoot(t))
	receiptPath := filepath.Join(repoRoot(t), "fixtures", "memory-import", "synthetic-receipt.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "archive", receiptPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("archive exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "receipt-shaped input archived as source_metadata") {
		t.Fatalf("stderr = %q, want default-kind warning", stderr.String())
	}
	if !strings.Contains(stdout.String(), "kind: source_metadata\n") {
		t.Fatalf("stdout = %q, want default kind still visible", stdout.String())
	}
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
	code := run([]string{"vault", "backup", "--verify", "--allow-raw-egress", dest}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("backup exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	for _, want := range []string{"verified: yes\n", "files_copied: ", "raw_egress_ack: raw vault bytes copied by explicit operator request\n"} {
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

func TestVaultBackupRefusesRawWithoutEgressAck(t *testing.T) {
	setTestQratumHome(t)
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

	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "backup", filepath.Join(t.TempDir(), "backup")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("backup exit code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "backup includes raw vault bytes; rerun with --allow-raw-egress") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestVaultGCEraseLifecycleCommands(t *testing.T) {
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
	refPath := singleGlob(t, filepath.Join(qratumHome, "raw", "refs", "*.json"))
	var ref vault.RawRef
	readJSONFile(t, refPath, &ref)
	orphanDigest := strings.Repeat("b", 64)
	orphanPath := filepath.Join(qratumHome, "raw", "blobs", "sha256", "bb", orphanDigest)
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphanPath, []byte("orphan\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"vault", "gc"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("gc exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"orphans_removed: 1\n", "referenced_kept: 1\n"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("gc stdout = %q, missing %q", stdout.String(), want)
		}
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("orphan stat err = %v, want not exist", err)
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"vault", "erase", "--reason", "test deletion request", ref.RawRefID}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("erase exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"qratum vault erase\n", "blob_removed: yes\n", "tombstone: "} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("erase stdout = %q, missing %q", stdout.String(), want)
		}
	}
	if _, err := os.Stat(filepath.FromSlash(ref.ArchivedPath)); !os.IsNotExist(err) {
		t.Fatalf("erased blob stat err = %v, want not exist", err)
	}
	tombstonePath := filepath.Join(qratumHome, "raw", "tombstones", ref.RawRefID+".json")
	if _, err := os.Stat(tombstonePath); err != nil {
		t.Fatalf("tombstone missing: %v", err)
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

func readJSONMapFromFile(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode %s: %v", filepath.ToSlash(path), err)
	}
	return value
}

func setNowForTest(t *testing.T, fixed time.Time) func() {
	t.Helper()
	previous := nowUTC
	nowUTC = func() time.Time { return fixed }
	return func() {
		nowUTC = previous
	}
}

func writeCaptureEventForDoctor(t *testing.T, qratumHome string, eventID string, sessionID string, rawRefID string, digest string, copyStatus string) {
	t.Helper()
	event := captureEvent{
		SchemaVersion:   captureEventSchemaVersion,
		DataClass:       "raw",
		EventID:         eventID,
		Source:          claudeCodeSource,
		EventType:       "session_end",
		Timestamp:       "2026-06-17T12:00:00Z",
		TimestampSource: hookTimestampSourceCaptureTime,
		SessionRef: captureSessionRef{
			SessionID:      sessionID,
			TranscriptPath: sessionID + ".jsonl",
		},
		Workspace: captureWorkspaceRef{CWD: "/tmp/qratum"},
		Raw: &capture.EventRaw{
			CopyStatus: copyStatus,
			RawRefID:   rawRefID,
			Digest:     digest,
			Kind:       vault.KindMainTranscript,
			SizeBytes:  1,
		},
	}
	path := filepath.Join(qratumHome, "events", eventID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, mustJSON(event), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeFakeTrustQRT(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "qrt")
	script := `#!/bin/sh
case "$1" in
  status) echo "qratum status"; exit 0 ;;
  vault) echo "qratum vault doctor"; exit 0 ;;
  dogfood) echo "qratum dogfood"; exit 0 ;;
  evidence) echo "qratum evidence"; exit 0 ;;
  review) echo "qratum review"; exit 0 ;;
  report) echo "qratum report"; exit 0 ;;
  export) echo "qratum export"; exit 0 ;;
  --version) echo "qrt test"; exit 0 ;;
esac
echo "unsupported $*" >&2
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- fake qrt script must be executable for the trust smoke test.
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/edictum-ai/qratum/internal/vault"
)

func TestRedactArrowSeparatorLeavesNoResidualSecret(t *testing.T) {
	variants := []struct {
		input  string
		secret string
	}{
		{input: "PASSWORD => hunter2pass", secret: "hunter2pass"},
		{input: "API_KEY ==> arrowsecret123", secret: "arrowsecret123"},
		{input: "TOKEN : => tokenvalue456", secret: "tokenvalue456"},
	}
	for _, tt := range variants {
		t.Run(tt.input, func(t *testing.T) {
			redactor := newDeterministicRedactor()
			got := redactor.redactString("test", tt.input)
			if strings.Contains(got, tt.secret) {
				t.Fatalf("redacted output leaked residual secret %q: %q", tt.secret, got)
			}
			if !strings.Contains(got, "[REDACTED_SECRET_") {
				t.Fatalf("redacted output %q has no secret placeholder", got)
			}
		})
	}
}

func TestSSHRemoteRedactionPattern(t *testing.T) {
	redactor := newDeterministicRedactor()
	got := redactor.redactString("git.remote", "git@example.invalid:org/repo.git")
	if strings.Contains(got, "git@example.invalid:org/repo.git") || !strings.Contains(got, "[REDACTED_SECRET_") {
		t.Fatalf("SSH remote redaction = %q, want placeholder and no raw remote", got)
	}
}

func TestGitFieldHybridRedactsJSONAndDropsShareableSurfaces(t *testing.T) {
	session := minimalSecuritySession()
	session.StartedAt = "2026-06-16T11:00:00Z"
	session.EndedAt = "2026-06-16T11:05:00Z"
	session.SourceEventID = "evt_customer_acme_prod_secret"
	session.Git = &qratumGitInfo{
		Remote:  "git@example.invalid:secret-org/private-repo.git",
		Branch:  "feature/customer-acme-prod-keys",
		HeadSHA: "0123456789abcdef0123456789abcdef01234567",
	}

	redacted, err := redactQratumSession(session)
	if err != nil {
		t.Fatal(err)
	}
	redactedJSON := string(mustJSON(redacted))
	for _, raw := range []string{session.StartedAt, session.EndedAt, session.SourceEventID, session.Git.Remote, session.Git.Branch, session.Git.HeadSHA} {
		if strings.Contains(redactedJSON, raw) {
			t.Fatalf("redacted session JSON leaked raw metadata %q:\n%s", raw, redactedJSON)
		}
	}
	for _, value := range []string{redacted.StartedAt, redacted.EndedAt, redacted.SourceEventID, redacted.Git.Remote, redacted.Git.Branch, redacted.Git.HeadSHA} {
		if !strings.HasPrefix(value, "[REDACTED_SECRET_") {
			t.Fatalf("metadata value = %q, want secret placeholder", value)
		}
	}

	sshRedactor := newDeterministicRedactor()
	sshOut := sshRedactor.redactString("git.remote", "remote git@example.invalid:org/repo.git")
	if strings.Contains(sshOut, "git@example.invalid:org/repo.git") || !strings.Contains(sshOut, "[REDACTED_SECRET_") {
		t.Fatalf("SSH remote was not redacted: %q", sshOut)
	}

	detail := buildUISessionDetail(uiSessionContext{
		session:  session,
		redacted: redacted,
		evidence: evidenceBundle{Summary: evidenceBundleSummary{Status: evidenceStatusComplete}},
		review:   reviewCard{Verdict: "clean", MainFinding: "ok", Evidence: []string{}, SuggestedNextHabit: "ok", Warnings: []string{}},
	})
	dto := string(mustJSON(detail))
	for _, banned := range []string{"git_remote", "git_branch", "git_head_sha", "started_at", "ended_at", "source_event_id"} {
		if strings.Contains(dto, banned) {
			t.Fatalf("UI DTO contains dropped field %q:\n%s", banned, dto)
		}
	}

	var report strings.Builder
	writeSessionSummary(&report, detail, "sessions/ses_security/normalized.json")
	for _, banned := range []string{"Git remote", "Git branch", "Git head SHA", "Started at", "Ended at", "Source event ID"} {
		if strings.Contains(report.String(), banned) {
			t.Fatalf("report summary contains dropped label %q:\n%s", banned, report.String())
		}
	}

	adp, err := buildADPStrictJSONL(session)
	if err != nil {
		t.Fatal(err)
	}
	adpText := string(adp)
	for _, banned := range []string{"git", "started_at", "ended_at", "source_event_id", session.Git.Remote, session.Git.Branch, session.Git.HeadSHA} {
		if strings.Contains(adpText, banned) {
			t.Fatalf("ADP contains dropped metadata %q:\n%s", banned, adpText)
		}
	}
}

func TestADPAllowlistDropsUnknownNestedInternalKeys(t *testing.T) {
	session := minimalSecuritySession()
	unknownKey := "internal_random_unknown_7f3d9c"
	session.ToolCalls = []qratumToolCall{{
		ToolCallID: "tool_fetch",
		Name:       "Fetch",
		Timestamp:  "2026-06-16T10:00:00Z",
		Input: map[string]any{
			"url":        "https://example.invalid/data.json",
			unknownKey:   "must-not-export",
			"metadata":   map[string]any{unknownKey: "nested-must-not-export"},
			"x-qratum-z": "denylist-alone-would-catch-this",
		},
	}}
	data, err := buildADPStrictJSONL(session)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, "https://example.invalid/data.json") {
		t.Fatalf("ADP output dropped allowed url kwarg: %s", out)
	}
	for _, banned := range []string{unknownKey, "must-not-export", "nested-must-not-export", "metadata", "x-qratum-z"} {
		if strings.Contains(out, banned) {
			t.Fatalf("ADP allowlist leaked %q:\n%s", banned, out)
		}
	}
}

func TestDaemonArtifactsStayInQratumHomeAndLeaveGitWorktreeUntouched(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture repo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeClaudeFixture(t, root, "transcript-verification-gap.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	before := snapshotWorktree(t, root)
	hookInput := fmt.Sprintf(`{"session_id":"claude-session-0001","transcript_path":"fixtures/claude-code/transcript-verification-gap.jsonl","cwd":%q,"hook_event_name":"SessionEnd"}`, filepath.ToSlash(root))
	var stdout, stderr bytes.Buffer
	code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(hookInput), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("hook exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"daemon", "run-once"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("daemon exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	after := snapshotWorktree(t, root)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("git worktree changed\n before=%v\n after=%v", before, after)
	}
	if _, err := os.Stat(filepath.Join(root, ".qratum")); !os.IsNotExist(err) {
		t.Fatalf("repo-local .qratum exists or stat failed: %v", err)
	}
	for _, pattern := range []string{
		filepath.Join(qratumHome, "sessions", "*", "normalized.json"),
		filepath.Join(qratumHome, "sessions", "*", "redacted.json"),
		filepath.Join(qratumHome, "sessions", "*", "evidence.json"),
		filepath.Join(qratumHome, "sessions", "*", "review.json"),
		filepath.Join(qratumHome, "sessions", "*", "report.html"),
		filepath.Join(qratumHome, "sessions", "*", "session.adp.jsonl"),
	} {
		path := singleGlob(t, pattern)
		if !pathIsInside(qratumHome, path) {
			t.Fatalf("artifact %s escaped QRATUM_HOME %s", path, qratumHome)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"sessions", "list", "--repo", filepath.ToSlash(root)}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("sessions list --repo exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "claude-session-0001") {
		t.Fatalf("repo-filtered sessions list missing generated session: %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"sessions", "list", "--repo", "other_repo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("sessions list --repo other exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("repo-filtered sessions list for other repo = %q, want empty", stdout.String())
	}
}

func TestHookConcurrentCapturesCreateDistinctEvents(t *testing.T) {
	root := t.TempDir()
	writeClaudeFixture(t, root, "transcript-basic.jsonl")
	t.Chdir(root)
	qratumHome := setTestQratumHome(t)
	input := fmt.Sprintf(`{"session_id":"race-session","transcript_path":"fixtures/claude-code/transcript-basic.jsonl","cwd":%q,"hook_event_name":"SessionEnd","timestamp":"2026-06-16T10:00:00Z"}`, filepath.ToSlash(root))

	const n = 32
	var wg sync.WaitGroup
	errs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var stdout, stderr bytes.Buffer
			code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)
			if code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				errs <- fmt.Sprintf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(qratumHome, "events", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != n {
		t.Fatalf("event files = %d %v, want %d", len(files), files, n)
	}
	seen := map[string]struct{}{}
	for _, file := range files {
		var event captureEvent
		readJSONFile(t, file, &event)
		if _, ok := seen[event.EventID]; ok {
			t.Fatalf("duplicate event_id %q", event.EventID)
		}
		seen[event.EventID] = struct{}{}
		if got, want := filepath.Base(file), event.EventID+".json"; got != want {
			t.Fatalf("event filename = %q, want %q", got, want)
		}
	}
}

func TestHookPathConfinementRejectsHostilePayloads(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string) string
	}{
		{
			name: "symlink-to-secret",
			setup: func(t *testing.T, root string) string {
				secret := filepath.Join(t.TempDir(), "id_rsa")
				if err := os.WriteFile(secret, []byte("PRIVATE KEY"), 0o600); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(root, "transcript-link.jsonl")
				if err := os.Symlink(secret, link); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
				return "transcript-link.jsonl"
			},
		},
		{
			name: "dotdot-traversal",
			setup: func(t *testing.T, root string) string {
				outside := filepath.Join(filepath.Dir(root), "outside.jsonl")
				if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
					t.Fatal(err)
				}
				return "../outside.jsonl"
			},
		},
		{
			name: "non-regular-directory",
			setup: func(t *testing.T, root string) string {
				dir := filepath.Join(root, "transcript-dir")
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				return "transcript-dir"
			},
		},
		{
			name: "oversized",
			setup: func(t *testing.T, root string) string {
				path := filepath.Join(root, "too-large.jsonl")
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Truncate(vault.MaxArchiveFileBytes + 1); err != nil {
					_ = file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				return "too-large.jsonl"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			path := tt.setup(t, root)
			t.Chdir(root)
			qratumHome := setTestQratumHome(t)
			input := fmt.Sprintf(`{"session_id":"hostile-%s","transcript_path":%q,"cwd":%q,"hook_event_name":"SessionEnd","timestamp":"2026-06-16T10:00:00Z"}`, tt.name, filepath.ToSlash(path), filepath.ToSlash(root))
			var stdout, stderr bytes.Buffer
			code := runWithIO([]string{"hook", "claude-code"}, strings.NewReader(input), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("hook exit code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), "warning: capture recorded but transcript copy failed") {
				t.Fatalf("stderr = %q, want degraded capture warning", stderr.String())
			}
			eventPath := singleGlob(t, filepath.Join(qratumHome, "events", "*.json"))
			var event captureEvent
			readJSONFile(t, eventPath, &event)
			if event.Raw == nil || event.Raw.CopyStatus != "failed" || event.Raw.CopyError == "" {
				t.Fatalf("event raw = %#v, want recorded failed capture", event.Raw)
			}
			assertNoVaultBlobsOrRefs(t, qratumHome)
		})
	}
}

func TestNoSecretInGolden(t *testing.T) {
	t.Run("working-tree", func(t *testing.T) {
		findings, err := scanWorkingTreeGoldensForSecrets(repoRoot(t))
		if err != nil {
			t.Fatal(err)
		}
		if len(findings) > 0 {
			t.Fatalf("working tree golden secret lint failed:\n%s", strings.Join(findings, "\n"))
		}
	})
	t.Run("git-history-known-red", func(t *testing.T) {
		findings, err := scanFixtureHistoryForSecrets(repoRoot(t))
		if err != nil {
			t.Fatal(err)
		}
		if len(findings) > 0 {
			t.Fatalf("KNOWN-RED: fixture git history contains internal identifiers; maintainer history rewrite/relocation required, not attempted by this task. Findings:\n%s", strings.Join(findings, "\n"))
		}
	})
}

func minimalSecuritySession() qratumSession {
	ok := true
	return qratumSession{
		SchemaVersion: qratumSessionSchemaVersion,
		SessionID:     "ses_security",
		Source:        claudeCodeSource,
		Turns: []qratumTurn{{
			Role:      "user",
			Timestamp: "2026-06-16T10:00:00Z",
			Content:   "hello",
		}},
		ToolCalls: []qratumToolCall{{
			ToolCallID: "tool_read",
			Name:       "Read",
			Timestamp:  "2026-06-16T10:00:01Z",
			Input:      map[string]any{"file_path": "README.md"},
			Success:    &ok,
			Result:     "ok",
		}},
		FileChanges: []qratumFileChange{},
		Commands:    []qratumCommand{},
		BusinessMetrics: qratumBusinessMetrics{
			DurationSeconds: 1,
			ToolCalls:       1,
		},
		Provenance: map[string]any{},
	}
}

type worktreeFileSnapshot struct {
	Mode os.FileMode
	Data string
}

func snapshotWorktree(t *testing.T, root string) map[string]worktreeFileSnapshot {
	t.Helper()
	out := map[string]worktreeFileSnapshot{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = worktreeFileSnapshot{Mode: info.Mode().Perm(), Data: string(data)}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func pathIsInside(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func assertNoVaultBlobsOrRefs(t *testing.T, qratumHome string) {
	t.Helper()
	for _, pattern := range []string{
		filepath.Join(qratumHome, "raw", "refs", "*.json"),
		filepath.Join(qratumHome, "raw", "blobs", "sha256", "*", "*"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("vault raw files for %s = %v, want none", pattern, matches)
		}
	}
}

type lintPattern struct {
	name string
	re   *regexp.Regexp
}

var fixtureSecretPatterns = []lintPattern{
	{name: "internal qratum repo", re: regexp.MustCompile(`edictum-ai/qratum\.git`)},
	{name: "ssh remote", re: regexp.MustCompile(`git@[^\s"'<>]+:[^\s"'<>]+`)},
	{name: "OpenAI/Anthropic shaped token", re: regexp.MustCompile(`\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\b`)},
	{name: "GitHub token", re: regexp.MustCompile(`\bghp_[A-Za-z0-9_]{20,}\b`)},
	{name: "40-hex head sha", re: regexp.MustCompile(`\b[0-9a-f]{40}\b`)},
}

func scanWorkingTreeGoldensForSecrets(root string) ([]string, error) {
	var findings []string
	fixturesRoot := filepath.Join(root, "fixtures")
	err := filepath.WalkDir(fixturesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		findings = append(findings, scanTextForFixtureSecrets(root, path, string(data), "working-tree")...)
		return nil
	})
	return findings, err
}

func scanFixtureHistoryForSecrets(root string) ([]string, error) {
	cmd := exec.Command("git", "log", "--format=commit:%H", "-p", "--", "fixtures/")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log -p failed: %w\n%s", err, string(exitErr.Stderr))
		}
		return nil, err
	}
	var findings []string
	commit := "unknown"
	path := "fixtures/"
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "commit:") {
			commit = strings.TrimPrefix(line, "commit:")
			continue
		}
		if strings.HasPrefix(line, "diff --git ") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				path = strings.TrimPrefix(fields[3], "b/")
			}
			continue
		}
		findings = append(findings, scanTextForFixtureSecrets(root, path, line, commit)...)
	}
	sort.Strings(findings)
	return uniqueStrings(findings), nil
}

func scanTextForFixtureSecrets(root string, path string, text string, location string) []string {
	var findings []string
	for _, pattern := range fixtureSecretPatterns {
		matches := pattern.re.FindAllString(text, -1)
		for _, match := range matches {
			if pattern.name == "40-hex head sha" && isAllowedHexMatch(text, match) {
				continue
			}
			rel := path
			if r, err := filepath.Rel(root, path); err == nil {
				rel = filepath.ToSlash(r)
			}
			findings = append(findings, fmt.Sprintf("%s %s %s matched %q", location, rel, pattern.name, match))
		}
	}
	return findings
}

func isAllowedHexMatch(line string, match string) bool {
	if match == strings.Repeat("0", 40) {
		return true
	}
	return strings.Contains(line, "sha256:"+match)
}

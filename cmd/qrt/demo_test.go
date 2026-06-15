package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeDemoProducesFullMilestoneSlice(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("make", "demo")
	cmd.Dir = root

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make demo failed: %v\n%s", err, output)
	}
	out := string(output)

	for _, want := range []string{
		"Running Milestone A demo with fixture input...",
		"QRATUM_HOME=",
		"qratum daemon run-once",
		"processed: 1",
		"claude-session-0001\t.qratum/sessions/",
		"Verified UI DTOs for session claude-session-0001",
		"Generated artifacts:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("make demo output missing %q:\n%s", want, out)
		}
	}

	var qratumHome string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "QRATUM_HOME=") {
			qratumHome = strings.TrimPrefix(line, "QRATUM_HOME=")
			break
		}
	}
	if qratumHome == "" {
		t.Fatalf("make demo output missing QRATUM_HOME line:\n%s", out)
	}

	for _, expected := range []struct {
		label   string
		pattern string
	}{
		{label: "vault event", pattern: filepath.Join(qratumHome, "events", "*.json")},
		{label: "vault raw ref", pattern: filepath.Join(qratumHome, "raw", "refs", "*.json")},
		{label: "vault raw blob", pattern: filepath.Join(qratumHome, "raw", "blobs", "sha256", "*", "*")},
		{label: "normalized session", pattern: ".qratum/sessions/*.normalized.json"},
		{label: "redacted session", pattern: ".qratum/redacted/*.redacted.json"},
		{label: "evidence", pattern: ".qratum/evidence/*.evidence.json"},
		{label: "review", pattern: ".qratum/reviews/*.review.json"},
		{label: "HTML report", pattern: ".qratum/reports/*.html"},
		{label: "ADP strict export", pattern: ".qratum/exports/*.adp.jsonl"},
	} {
		pattern := expected.pattern
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(root, filepath.FromSlash(pattern))
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob %s: %v", pattern, err)
		}
		if len(matches) != 1 {
			t.Fatalf("%s artifacts for %s = %v, want exactly one", expected.label, pattern, matches)
		}
		info, err := os.Stat(matches[0])
		if err != nil {
			t.Fatalf("stat %s: %v", matches[0], err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s artifact %s is empty", expected.label, matches[0])
		}
		wantPath := filepath.ToSlash(matches[0])
		if rel, err := filepath.Rel(root, matches[0]); err == nil && !strings.HasPrefix(rel, "..") {
			wantPath = filepath.ToSlash(rel)
		}
		if !strings.Contains(out, wantPath) {
			t.Fatalf("make demo output did not print %s artifact path %s:\n%s", expected.label, wantPath, out)
		}
	}
}

func TestDemoArtifactVerifierRejectsMissingArtifacts(t *testing.T) {
	root := t.TempDir()
	qratumHome := filepath.Join(root, "qratum-home")
	if err := os.MkdirAll(filepath.Join(qratumHome, "events"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(qratumHome, "raw", "refs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(qratumHome, "raw", "blobs", "sha256", "aa"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(qratumHome, "events", "demo.json"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(qratumHome, "raw", "refs", "demo.json"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(qratumHome, "raw", "blobs", "sha256", "aa", "aablob"), []byte("demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".qratum/sessions/demo.normalized.json",
		".qratum/redacted/demo.redacted.json",
		".qratum/reviews/demo.review.json",
		".qratum/reports/demo.html",
		".qratum/exports/demo.adp.jsonl",
	} {
		abs := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("demo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".qratum", "evidence"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", filepath.Join(repoRoot(t), "scripts", "demo.sh"), "--verify-only")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "QRATUM_HOME="+qratumHome)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("demo verifier succeeded, want missing evidence failure\n%s", output)
	}
	if !strings.Contains(string(output), "missing evidence artifact") {
		t.Fatalf("demo verifier output = %q, want missing evidence artifact error", output)
	}
}

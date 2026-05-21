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

	for _, expected := range []struct {
		label   string
		pattern string
	}{
		{label: "event", pattern: ".qratum/events/*.json"},
		{label: "normalized session", pattern: ".qratum/sessions/*.normalized.json"},
		{label: "redacted session", pattern: ".qratum/redacted/*.redacted.json"},
		{label: "evidence", pattern: ".qratum/evidence/*.evidence.json"},
		{label: "review", pattern: ".qratum/reviews/*.review.json"},
		{label: "HTML report", pattern: ".qratum/reports/*.html"},
		{label: "ADP strict export", pattern: ".qratum/exports/*.adp.jsonl"},
	} {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(expected.pattern)))
		if err != nil {
			t.Fatalf("glob %s: %v", expected.pattern, err)
		}
		if len(matches) != 1 {
			t.Fatalf("%s artifacts for %s = %v, want exactly one", expected.label, expected.pattern, matches)
		}
		info, err := os.Stat(matches[0])
		if err != nil {
			t.Fatalf("stat %s: %v", matches[0], err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s artifact %s is empty", expected.label, matches[0])
		}
		rel, err := filepath.Rel(root, matches[0])
		if err != nil {
			t.Fatalf("rel %s: %v", matches[0], err)
		}
		if !strings.Contains(out, filepath.ToSlash(rel)) {
			t.Fatalf("make demo output did not print %s artifact path %s:\n%s", expected.label, filepath.ToSlash(rel), out)
		}
	}
}

func TestDemoArtifactVerifierRejectsMissingArtifacts(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{
		".qratum/events/demo.json",
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
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("demo verifier succeeded, want missing evidence failure\n%s", output)
	}
	if !strings.Contains(string(output), "missing evidence artifact") {
		t.Fatalf("demo verifier output = %q, want missing evidence artifact error", output)
	}
}

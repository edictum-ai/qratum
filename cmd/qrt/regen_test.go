package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRegenerateUIGoldens is a manually-invoked helper used to refresh the
// fixtures/ui/*.golden.json files after intentional contract changes. It is
// guarded by an env var so it never runs in normal CI/test runs.
//
//	QRATUM_REGENERATE_UI_GOLDENS=1 go test ./cmd/qrt -run TestRegenerateUIGoldens
func TestRegenerateUIGoldens(t *testing.T) {
	if os.Getenv("QRATUM_REGENERATE_UI_GOLDENS") != "1" {
		t.Skip("set QRATUM_REGENERATE_UI_GOLDENS=1 to regenerate UI goldens")
	}
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	t.Chdir(root)

	for _, spec := range []struct {
		args []string
		dest string
	}{
		{[]string{"ui", "sessions", "--json"}, "sessions.golden.json"},
		{[]string{"ui", "session", "ses_0001", "--json"}, "session-detail.golden.json"},
		{[]string{"ui", "review", "ses_0001", "--json"}, "review.golden.json"},
	} {
		var stdout, stderr bytes.Buffer
		code := run(spec.args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("regen %v exit code = %d; stderr = %q", spec.args, code, stderr.String())
		}
		dest := filepath.Join(repoRoot(t), "fixtures", "ui", spec.dest)
		if err := os.WriteFile(dest, stdout.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

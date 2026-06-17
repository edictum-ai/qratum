package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCreatesAndSecuresRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "qratum-home")
	// #nosec G301 -- deliberately creates a loose directory to verify Resolve tightens it.
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- deliberately creates a loose directory to verify Resolve tightens it.
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(HomeEnv, root)

	paths, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if paths.Root != root {
		t.Fatalf("root = %q, want %q", paths.Root, root)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("root mode = %#o, want %#o", got, want)
	}
}

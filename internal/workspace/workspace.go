// Package workspace resolves the local Qratum home layout.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HomeEnv overrides the default ~/.qratum workspace root.
const HomeEnv = "QRATUM_HOME"

// Paths holds the resolved local Qratum workspace paths.
type Paths struct {
	Root string
}

// Resolve returns the effective Qratum workspace root.
func Resolve() (Paths, error) {
	root := strings.TrimSpace(os.Getenv(HomeEnv))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve user home: %w", err)
		}
		root = filepath.Join(home, ".qratum")
	} else if strings.HasPrefix(root, "~"+string(os.PathSeparator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve user home: %w", err)
		}
		root = filepath.Join(home, strings.TrimPrefix(root, "~"+string(os.PathSeparator)))
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve qratum home %q: %w", root, err)
	}
	clean := filepath.Clean(abs)
	if err := os.MkdirAll(clean, 0o700); err != nil {
		return Paths{}, fmt.Errorf("create qratum home %s: %w", filepath.ToSlash(clean), err)
	}
	if err := os.Chmod(clean, 0o700); err != nil {
		return Paths{}, fmt.Errorf("secure qratum home %s: %w", filepath.ToSlash(clean), err)
	}
	return Paths{Root: clean}, nil
}

// EventsDir returns the capture event spool path.
func (p Paths) EventsDir() string {
	return filepath.Join(p.Root, "events")
}

// RawDir returns the raw archive root.
func (p Paths) RawDir() string {
	return filepath.Join(p.Root, "raw")
}

// RawBlobsDir returns the content-addressed blob directory.
func (p Paths) RawBlobsDir() string {
	return filepath.Join(p.RawDir(), "blobs", "sha256")
}

// RawBlobsTempDir returns the temporary blob staging directory.
func (p Paths) RawBlobsTempDir() string {
	return filepath.Join(p.RawDir(), "blobs", ".tmp")
}

// RawRefsDir returns the raw-ref metadata directory.
func (p Paths) RawRefsDir() string {
	return filepath.Join(p.RawDir(), "refs")
}

// StateDir returns the workspace state directory.
func (p Paths) StateDir() string {
	return filepath.Join(p.Root, "state")
}

// VaultStatePath returns the vault state file path.
func (p Paths) VaultStatePath() string {
	return filepath.Join(p.StateDir(), "vault.json")
}

// BlobPathForDigest returns the blob path for a sha256 hex digest.
func (p Paths) BlobPathForDigest(digest string) string {
	prefix := digest
	if len(prefix) > 2 {
		prefix = prefix[:2]
	}
	return filepath.Join(p.RawBlobsDir(), prefix, digest)
}

// RawRefIDForDigest returns the stable raw-ref identifier for a digest.
func (p Paths) RawRefIDForDigest(digest string) string {
	return "raw_" + digest
}

// RawRefPathForDigest returns the raw-ref file path for a digest.
func (p Paths) RawRefPathForDigest(digest string) string {
	return filepath.Join(p.RawRefsDir(), p.RawRefIDForDigest(digest)+".json")
}

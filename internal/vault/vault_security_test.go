package vault

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edictum-ai/qratum/internal/workspace"
)

func TestRawRefIDCollisionUsesFullDigest(t *testing.T) {
	paths := workspace.Paths{Root: t.TempDir()}
	store := New(paths)
	digestA := "aaaaaaaaaaaa1111111111111111111111111111111111111111111111111111"
	digestB := "aaaaaaaaaaaa2222222222222222222222222222222222222222222222222222"
	if digestA[:12] != digestB[:12] || digestA == digestB {
		t.Fatalf("test digests must be distinct with matching 12-char prefix")
	}

	for _, digest := range []string{digestA, digestB} {
		ref := RawRef{SchemaVersion: RawRefSchemaVersion, RawRefID: paths.RawRefIDForDigest(digest), Source: SourceClaudeCode, Kind: KindMainTranscript, Digest: "sha256:" + digest, OriginalPath: "/tmp/source.jsonl", ArchivedPath: paths.BlobPathForDigest(digest), LocalOnly: true}
		created, err := store.writeRawRef(ref, digest)
		if err != nil {
			t.Fatalf("write ref %s: %v", digest, err)
		}
		if !created {
			t.Fatalf("write ref %s created=false, want true", digest)
		}
	}

	refs, err := filepath.Glob(filepath.Join(paths.RawRefsDir(), "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("raw refs = %v, want two distinct refs", refs)
	}
	if got := paths.RawRefIDForDigest(digestA); got != "raw_"+digestA || strings.Contains(got, "raw_"+digestA[:12]+".json") {
		t.Fatalf("raw ref id = %q, want full digest", got)
	}
}

func TestVaultAtRestPermsAreOwnerOnly(t *testing.T) {
	// This test enforces local file permissions only. At-rest encryption is out of
	// scope for P1; the threat model relies on OS permissions here.
	qratumHome := filepath.Join(t.TempDir(), "qratum-home")
	t.Setenv(workspace.HomeEnv, qratumHome)
	paths, err := workspace.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	store := New(paths)

	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(source, []byte("secret transcript"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"}); err != nil {
		t.Fatalf("archive file: %v", err)
	}
	if err := store.SaveState(State{LastCaptureAt: "2026-06-16T00:00:00Z"}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := os.MkdirAll(paths.EventsDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(paths.EventsDir(), "evt_test.json")
	if err := os.WriteFile(eventPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := checkVaultAtRestPerms(paths.Root); err != nil {
		t.Fatalf("clean vault permissions failed: %v", err)
	}
	// #nosec G302 -- deliberately flips a vault file world-readable to prove the permission gate goes red.
	if err := os.Chmod(eventPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkVaultAtRestPerms(paths.Root); err == nil {
		t.Fatal("permission check stayed green after flipping an event file to 0644")
	}
}

func checkVaultAtRestPerms(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		want := os.FileMode(0o600)
		if d.IsDir() {
			want = 0o700
		}
		if got := info.Mode().Perm(); got != want {
			return fmt.Errorf("%s mode = %#o, want %#o", path, got, want)
		}
		return nil
	})
}

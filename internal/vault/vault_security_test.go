package vault

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestArchiveFileRecordsIndependentSourceDigestAndDedups(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := []byte("independent source bytes\n")
	if err := os.WriteFile(source, body, 0o600); err != nil {
		t.Fatal(err)
	}
	sourceDigest := sha256.Sum256(body)
	wantDigest := fmt.Sprintf("sha256:%x", sourceDigest)

	first, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"})
	if err != nil {
		t.Fatalf("first archive: %v", err)
	}
	second, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:01Z"})
	if err != nil {
		t.Fatalf("second archive: %v", err)
	}
	if first.RawRef.Digest != wantDigest {
		t.Fatalf("raw ref digest = %q, want independent source digest %q", first.RawRef.Digest, wantDigest)
	}
	if !first.BlobCreated || !first.RefCreated {
		t.Fatalf("first archive result = %#v, want blob+ref created", first)
	}
	if second.BlobCreated || second.RefCreated {
		t.Fatalf("second archive result = %#v, want digest dedup", second)
	}
	blobs, err := filepath.Glob(filepath.Join(store.Paths.RawBlobsDir(), "*", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 {
		t.Fatalf("blob files = %v, want one deduped blob", blobs)
	}
}

func TestUpdateStateSerializesConcurrentCounters(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	const workers = 64
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.UpdateState(func(state *State) {
				state.CopyFailureCount++
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("update state: %v", err)
		}
	}
	state, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if got := state.CopyFailureCount; got != workers {
		t.Fatalf("copy_failure_count = %d, want %d", got, workers)
	}
}

func TestArchiveFileRefusesConfiguredLowDiskWithoutBlobOrRef(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(source, []byte("transcript\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := store.ArchiveFile(ArchiveRequest{
		Source:       SourceClaudeCode,
		Kind:         KindMainTranscript,
		OriginalPath: source,
		ObservedAt:   "2026-06-16T00:00:00Z",
		MinFreeBytes: 1 << 62,
	})
	if err == nil || !strings.Contains(err.Error(), "disk free below configured minimum") {
		t.Fatalf("archive error = %v, want low-disk refusal", err)
	}
	if refs, err := filepath.Glob(filepath.Join(store.Paths.RawRefsDir(), "*.json")); err != nil || len(refs) != 0 {
		t.Fatalf("raw refs = %v, err = %v; want none", refs, err)
	}
	if blobs, err := filepath.Glob(filepath.Join(store.Paths.RawBlobsDir(), "*", "*")); err != nil || len(blobs) != 0 {
		t.Fatalf("raw blobs = %v, err = %v; want none", blobs, err)
	}
}

func TestConfiguredDiskFreeMinBytesReadsWorkerConfig(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	if err := os.MkdirAll(store.Paths.Root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := "[raw]\narchive = true\n\n[worker]\nmax_jobs = 4\ndisk_free_min_gb = 3 # loud guard\n"
	if err := os.WriteFile(filepath.Join(store.Paths.Root, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := store.ConfiguredDiskFreeMinBytes()
	if err != nil {
		t.Fatal(err)
	}
	const want int64 = 3 * 1024 * 1024 * 1024
	if got != want {
		t.Fatalf("configured disk free bytes = %d, want %d", got, want)
	}
}

func TestBackupRoundTripRestoreSummaryMatchesSource(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	for name, body := range map[string]string{"first.jsonl": "first\n", "second.jsonl": "second\n"} {
		source := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"}); err != nil {
			t.Fatalf("archive %s: %v", name, err)
		}
	}
	if err := store.SaveState(State{LastCaptureAt: "2026-06-16T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	sourceSummary, err := store.Summary()
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "backup")
	if _, err := store.Backup(dest, true, true); err != nil {
		t.Fatalf("backup: %v", err)
	}
	restoreSummary, err := New(workspace.Paths{Root: dest}).Summary()
	if err != nil {
		t.Fatal(err)
	}
	if restoreSummary.BlobCount != sourceSummary.BlobCount || restoreSummary.RefCount != sourceSummary.RefCount {
		t.Fatalf("restore summary = %#v, source = %#v", restoreSummary, sourceSummary)
	}
	if restoreSummary.LastState.LastCaptureAt != sourceSummary.LastState.LastCaptureAt {
		t.Fatalf("restored state = %#v, source state = %#v", restoreSummary.LastState, sourceSummary.LastState)
	}
}

func TestBackupRawEgressWritesAuditEvent(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(source, []byte("raw vault bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"}); err != nil {
		t.Fatalf("archive file: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "backup")
	result, err := store.Backup(dest, true, true)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if !result.RawEgress {
		t.Fatalf("RawEgress=false, want true")
	}
	auditPath := filepath.Join(store.Paths.StateDir(), "raw-egress-audit.jsonl")
	// #nosec G304 -- test reads the audit file written under its temp qratum home.
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var event map[string]string
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &event); err != nil {
		t.Fatal(err)
	}
	if got, want := event["event"], "raw_egress_ack"; got != want {
		t.Fatalf("audit event = %q, want %q", got, want)
	}
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := event["destination"], filepath.ToSlash(destAbs); got != want {
		t.Fatalf("audit destination = %q, want %q", got, want)
	}
	if strings.TrimSpace(event["observed_at"]) == "" {
		t.Fatalf("audit event missing observed_at: %#v", event)
	}
}

func TestSweepStaleTempBlobsKeepsFreshFiles(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	tempDir := store.Paths.RawBlobsTempDir()
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(tempDir, ".blob.stale.tmp")
	fresh := filepath.Join(tempDir, ".blob.fresh.tmp")
	for _, path := range []string{stale, fresh} {
		if err := os.WriteFile(path, []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	if err := os.Chtimes(stale, now.Add(-2*DefaultTempBlobStaleAfter), now.Add(-2*DefaultTempBlobStaleAfter)); err != nil {
		t.Fatal(err)
	}

	removed, err := store.SweepStaleTempBlobs(DefaultTempBlobStaleAfter, now)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale temp stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("fresh temp missing: %v", err)
	}
}

func TestBackupExcludesTempBlobs(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(source, []byte("vault bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"}); err != nil {
		t.Fatalf("archive file: %v", err)
	}
	tempPath := filepath.Join(store.Paths.RawBlobsTempDir(), ".blob.leftover.tmp")
	if err := os.WriteFile(tempPath, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "backup")
	if _, err := store.Backup(dest, true, true); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "raw", "blobs", ".tmp")); !os.IsNotExist(err) {
		t.Fatalf("backup temp dir stat err = %v, want not exist", err)
	}
}

func TestBackupVerifyUsesRecordedDigestAndDetectsCorruption(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(source, []byte("vault bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"})
	if err != nil {
		t.Fatalf("archive file: %v", err)
	}
	dest := filepath.Join(t.TempDir(), "backup")
	if _, err := store.Backup(dest, true, true); err != nil {
		t.Fatalf("backup: %v", err)
	}
	rel, err := filepath.Rel(store.Paths.Root, filepath.FromSlash(result.RawRef.ArchivedPath))
	if err != nil {
		t.Fatal(err)
	}
	backupBlob := filepath.Join(dest, rel)
	if err := os.WriteFile(backupBlob, []byte("corrupt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = verifyTree(store.Paths.Root, dest)
	if err == nil || !strings.Contains(err.Error(), "against recorded digest") {
		t.Fatalf("verifyTree error = %v, want recorded digest mismatch", err)
	}
}

func TestSyntheticVaultBlobUnionIsDedupClean(t *testing.T) {
	storeA := New(workspace.Paths{Root: filepath.Join(t.TempDir(), "a")})
	storeB := New(workspace.Paths{Root: filepath.Join(t.TempDir(), "b")})
	shared := []byte("shared\n")
	onlyA := []byte("only-a\n")
	onlyB := []byte("only-b\n")
	for _, item := range []struct {
		store Store
		name  string
		body  []byte
	}{
		{storeA, "shared-a.jsonl", shared},
		{storeA, "only-a.jsonl", onlyA},
		{storeB, "shared-b.jsonl", shared},
		{storeB, "only-b.jsonl", onlyB},
	} {
		source := filepath.Join(t.TempDir(), item.name)
		if err := os.WriteFile(source, item.body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := item.store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"}); err != nil {
			t.Fatalf("archive %s: %v", item.name, err)
		}
	}

	union := New(workspace.Paths{Root: filepath.Join(t.TempDir(), "union")})
	for _, sourceRoot := range []string{storeA.Paths.RawBlobsDir(), storeB.Paths.RawBlobsDir()} {
		if err := copyBlobTreeForTest(sourceRoot, union.Paths.RawBlobsDir()); err != nil {
			t.Fatal(err)
		}
	}
	blobs, err := filepath.Glob(filepath.Join(union.Paths.RawBlobsDir(), "*", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 3 {
		t.Fatalf("union blobs = %v, want three unique content-addressed blobs", blobs)
	}
	got := map[string]bool{}
	for _, path := range blobs {
		// #nosec G304 -- blob paths come from the test's synthetic QRATUM_HOME glob.
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got[string(data)] = true
	}
	for _, want := range [][]byte{shared, onlyA, onlyB} {
		if !got[string(want)] {
			t.Fatalf("union missing blob content %q; got %v", want, got)
		}
	}
}

func TestGarbageCollectRemovesOnlyOrphanBlobs(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	source := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(source, []byte("referenced\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: source, ObservedAt: "2026-06-16T00:00:00Z"})
	if err != nil {
		t.Fatalf("archive file: %v", err)
	}
	orphanDigest := strings.Repeat("a", 64)
	orphanPath := store.Paths.BlobPathForDigest(orphanDigest)
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphanPath, []byte("orphan\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	gc, err := store.GarbageCollectOrphanBlobs()
	if err != nil {
		t.Fatal(err)
	}
	if got := gc.OrphansRemoved; got != 1 {
		t.Fatalf("orphans_removed = %d, want 1", got)
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("orphan stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.FromSlash(result.RawRef.ArchivedPath)); err != nil {
		t.Fatalf("referenced blob missing: %v", err)
	}
}

func TestEraseRawRefWritesTombstoneAndRemovesOnlyTarget(t *testing.T) {
	store := New(workspace.Paths{Root: t.TempDir()})
	first := filepath.Join(t.TempDir(), "first.jsonl")
	second := filepath.Join(t.TempDir(), "second.jsonl")
	if err := os.WriteFile(first, []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstResult, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: first, ObservedAt: "2026-06-16T00:00:00Z"})
	if err != nil {
		t.Fatalf("archive first: %v", err)
	}
	secondResult, err := store.ArchiveFile(ArchiveRequest{Source: SourceClaudeCode, Kind: KindMainTranscript, OriginalPath: second, ObservedAt: "2026-06-16T00:00:01Z"})
	if err != nil {
		t.Fatalf("archive second: %v", err)
	}

	erase, err := store.EraseRawRef(firstResult.RawRef.RawRefID, "test erasure", "2026-06-17T00:00:00Z")
	if err != nil {
		t.Fatalf("erase: %v", err)
	}
	if !erase.BlobRemoved || !erase.Tombstone.BlobRemoved {
		t.Fatalf("erase result = %#v, want blob removed", erase)
	}
	if _, err := os.Stat(filepath.FromSlash(firstResult.RawRef.ArchivedPath)); !os.IsNotExist(err) {
		t.Fatalf("first blob stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.FromSlash(secondResult.RawRef.ArchivedPath)); err != nil {
		t.Fatalf("second blob missing: %v", err)
	}
	tombstonePath := store.Paths.RawTombstonePathForDigest(strings.TrimPrefix(firstResult.RawRef.Digest, "sha256:"))
	var tombstone RawTombstone
	// #nosec G304 -- tombstone path is derived from the test-created raw ref digest.
	data, err := os.ReadFile(tombstonePath)
	if err != nil {
		t.Fatalf("read tombstone: %v", err)
	}
	if err := json.Unmarshal(data, &tombstone); err != nil {
		t.Fatalf("decode tombstone: %v", err)
	}
	if tombstone.RawRefID != firstResult.RawRef.RawRefID || tombstone.Reason != "test erasure" {
		t.Fatalf("tombstone = %#v", tombstone)
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

func copyBlobTreeForTest(source string, dest string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		// #nosec G304,G122 -- test copies files from a synthetic vault blob tree.
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// #nosec G304 -- target is under the test destination blob tree.
		existing, err := os.ReadFile(target)
		if err == nil {
			if !bytes.Equal(existing, data) {
				return fmt.Errorf("content-addressed collision at %s", filepath.ToSlash(target))
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		// #nosec G703 -- target is the content-addressed path under the test vault root.
		return os.WriteFile(target, data, 0o600)
	})
}

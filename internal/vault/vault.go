// Package vault manages the local raw transcript vault.
package vault

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	qschema "github.com/acartag7/qratum/internal/schema"
	"github.com/acartag7/qratum/internal/workspace"
)

const (
	// RawRefSchemaVersion is the schema version stored in raw ref records.
	RawRefSchemaVersion = "qratum.raw_ref.v1"
	// RawTombstoneSchemaVersion is the schema version stored in erasure tombstones.
	RawTombstoneSchemaVersion = "qratum.raw_tombstone.v1"
	// VaultStateSchemaVersion is the schema version stored in vault state.
	VaultStateSchemaVersion = "qratum.vault_state.v1"

	// SourceClaudeCode labels raw items captured from Claude Code.
	SourceClaudeCode = "claude-code"
	// SourceCodex labels raw items captured from Codex-managed files.
	SourceCodex = "codex"
	// SourceManual labels raw items archived from manual local paths.
	SourceManual = "manual"

	// KindMainTranscript labels a primary session transcript.
	KindMainTranscript = "main_transcript"
	// KindSubagentTranscript labels a subagent transcript.
	KindSubagentTranscript = "subagent_transcript"
	// KindFileHistory labels a captured file-history snapshot.
	KindFileHistory = "file_history_snapshot"
	// KindInsightReport labels a source-generated insight report.
	KindInsightReport = "source_insight_report"
	// KindSourceMetadata labels source metadata files.
	KindSourceMetadata = "source_metadata"
	// KindSourceExportBundle labels source export bundle files.
	KindSourceExportBundle = "source_export_bundle"
	// KindSourceMemory labels source memory exports.
	KindSourceMemory = "source_memory"
	// KindVendorMemoryDir labels archived vendor memory directories.
	KindVendorMemoryDir = "vendor_memory_dir"
	// KindVendorInsight labels archived vendor insight reports.
	KindVendorInsight = "vendor_insight_report"
	// KindMemoryImport labels archived memory import receipts.
	KindMemoryImport = "memory_import_receipt"
	// KindUnknown labels uncategorized raw archive items.
	KindUnknown = "unknown"
	// MaxArchiveFileBytes caps a single raw archive copy. This is a permissions/
	// containment guard, not at-rest encryption.
	MaxArchiveFileBytes int64 = 50 << 20
	// DefaultTempBlobStaleAfter is the grace period before abandoned temp blobs
	// are considered crash leftovers.
	DefaultTempBlobStaleAfter = 10 * time.Minute
)

// Store wraps access to the local vault workspace.
type Store struct {
	Paths workspace.Paths
}

// RawRef records one content-addressed raw archive item.
type RawRef struct {
	SchemaVersion   string `json:"schema_version"`
	DataClass       string `json:"data_class"`
	RawRefID        string `json:"raw_ref_id"`
	Source          string `json:"source"`
	SourceSessionID string `json:"source_session_id,omitempty"`
	Kind            string `json:"kind"`
	Digest          string `json:"digest"`
	OriginalPath    string `json:"original_path"`
	ArchivedPath    string `json:"archived_path"`
	SizeBytes       int64  `json:"size_bytes"`
	ObservedAt      string `json:"observed_at"`
	LocalOnly       bool   `json:"local_only"`
}

// State stores lightweight operational vault state.
type State struct {
	SchemaVersion          string `json:"schema_version"`
	DataClass              string `json:"data_class"`
	LastCaptureAt          string `json:"last_capture_at,omitempty"`
	LastBackfillAt         string `json:"last_backfill_at,omitempty"`
	LastArchiveAt          string `json:"last_archive_at,omitempty"`
	LastBackupAt           string `json:"last_backup_at,omitempty"`
	LastBackupDest         string `json:"last_backup_dest,omitempty"`
	LastBackupVerifiedAt   string `json:"last_backup_verified_at,omitempty"`
	LastBackupVerifiedDest string `json:"last_backup_verified_dest,omitempty"`
	CopyFailureCount       int    `json:"copy_failure_count,omitempty"`
	RawMissingCount        int    `json:"raw_missing_count,omitempty"`
}

// ArchiveRequest describes one file to archive into the vault.
type ArchiveRequest struct {
	Source          string
	SourceSessionID string
	Kind            string
	OriginalPath    string
	ObservedAt      string
	MinFreeBytes    int64
}

// ArchiveResult reports what changed while archiving one file.
type ArchiveResult struct {
	RawRef      RawRef
	BlobCreated bool
	RefCreated  bool
}

// Summary reports high-level vault counts and state.
type Summary struct {
	Root      string
	Present   bool
	BlobCount int
	RefCount  int
	LastState State
}

// BackupResult reports the outcome of a local backup run.
type BackupResult struct {
	Destination string
	FileCount   int
	Verified    bool
	RawEgress   bool
}

// RawTombstone records an explicit raw blob erasure.
type RawTombstone struct {
	SchemaVersion string `json:"schema_version"`
	DataClass     string `json:"data_class"`
	RawRefID      string `json:"raw_ref_id"`
	Digest        string `json:"digest"`
	Reason        string `json:"reason"`
	ErasedAt      string `json:"erased_at"`
	BlobRemoved   bool   `json:"blob_removed"`
}

// GCResult reports orphan-blob garbage collection.
type GCResult struct {
	OrphansRemoved int
	ReferencedKept int
	TombstonedKept int
}

// EraseResult reports one explicit raw erasure.
type EraseResult struct {
	Tombstone   RawTombstone
	BlobPath    string
	RefPath     string
	BlobRemoved bool
}

// New creates a Store for the resolved workspace paths.
func New(paths workspace.Paths) Store {
	return Store{Paths: paths}
}

// ValidKinds returns the supported raw kinds in sorted order.
func ValidKinds() []string {
	kinds := make([]string, 0, len(validKinds))
	for kind := range validKinds {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// IsValidKind reports whether a raw kind is supported.
func IsValidKind(kind string) bool {
	_, ok := validKinds[strings.TrimSpace(kind)]
	return ok
}

var validKinds = map[string]struct{}{
	KindMainTranscript:     {},
	KindSubagentTranscript: {},
	KindFileHistory:        {},
	KindInsightReport:      {},
	KindSourceMetadata:     {},
	KindSourceExportBundle: {},
	KindSourceMemory:       {},
	KindVendorMemoryDir:    {},
	KindVendorInsight:      {},
	KindMemoryImport:       {},
	KindUnknown:            {},
}

// ArchiveFile copies one file into the content-addressed vault.
func (s Store) ArchiveFile(req ArchiveRequest) (ArchiveResult, error) {
	kind := strings.TrimSpace(req.Kind)
	if !IsValidKind(kind) {
		return ArchiveResult{}, fmt.Errorf("unsupported raw kind %q", req.Kind)
	}

	originalPath, err := filepath.Abs(strings.TrimSpace(req.OriginalPath))
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("resolve archive path %q: %w", req.OriginalPath, err)
	}
	info, err := os.Lstat(originalPath)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("inspect archive path %s: %w", filepath.ToSlash(originalPath), err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ArchiveResult{}, fmt.Errorf("archive path %s is a symlink", filepath.ToSlash(originalPath))
	}
	if !info.Mode().IsRegular() {
		return ArchiveResult{}, fmt.Errorf("archive path %s is not a regular file", filepath.ToSlash(originalPath))
	}
	if info.Size() > MaxArchiveFileBytes {
		return ArchiveResult{}, fmt.Errorf("archive path %s exceeds %d byte capture cap", filepath.ToSlash(originalPath), MaxArchiveFileBytes)
	}

	if err := os.MkdirAll(s.Paths.RawBlobsTempDir(), 0o700); err != nil {
		return ArchiveResult{}, fmt.Errorf("create blob temp directory: %w", err)
	}
	if req.MinFreeBytes > 0 {
		if err := CheckMinFreeSpace(s.Paths.RawBlobsTempDir(), req.MinFreeBytes); err != nil {
			return ArchiveResult{}, err
		}
	}
	// #nosec G304 -- archive targets are explicit local filesystem paths chosen by the user or hook payload.
	src, err := openFileNoFollowRead(originalPath)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("open archive path %s: %w", filepath.ToSlash(originalPath), err)
	}
	defer func() {
		_ = src.Close()
	}()
	openedInfo, err := src.Stat()
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("inspect opened archive path %s: %w", filepath.ToSlash(originalPath), err)
	}
	if !openedInfo.Mode().IsRegular() {
		return ArchiveResult{}, fmt.Errorf("archive path %s is not a regular file", filepath.ToSlash(originalPath))
	}
	if openedInfo.Size() > MaxArchiveFileBytes {
		return ArchiveResult{}, fmt.Errorf("archive path %s exceeds %d byte capture cap", filepath.ToSlash(originalPath), MaxArchiveFileBytes)
	}

	tmp, err := os.CreateTemp(s.Paths.RawBlobsTempDir(), ".blob.*.tmp")
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("create temporary blob: %w", err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(src, MaxArchiveFileBytes+1))
	if err != nil {
		_ = tmp.Close()
		return ArchiveResult{}, fmt.Errorf("copy archive path %s: %w", filepath.ToSlash(originalPath), err)
	}
	if size > MaxArchiveFileBytes {
		_ = tmp.Close()
		return ArchiveResult{}, fmt.Errorf("archive path %s exceeds %d byte capture cap", filepath.ToSlash(originalPath), MaxArchiveFileBytes)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return ArchiveResult{}, fmt.Errorf("set blob permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return ArchiveResult{}, fmt.Errorf("close temporary blob: %w", err)
	}

	digestHex := fmt.Sprintf("%x", hash.Sum(nil))
	digest := "sha256:" + digestHex
	blobPath := s.Paths.BlobPathForDigest(digestHex)
	if err := os.MkdirAll(filepath.Dir(blobPath), 0o700); err != nil {
		return ArchiveResult{}, fmt.Errorf("create blob directory: %w", err)
	}

	blobCreated := false
	if _, err := os.Stat(blobPath); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(tmpPath, blobPath); err != nil {
			return ArchiveResult{}, fmt.Errorf("commit blob %s: %w", filepath.ToSlash(blobPath), err)
		}
		removeTmp = false
		blobCreated = true
	} else if err != nil {
		return ArchiveResult{}, fmt.Errorf("inspect blob %s: %w", filepath.ToSlash(blobPath), err)
	}

	ref := RawRef{
		SchemaVersion:   RawRefSchemaVersion,
		DataClass:       qschema.DataClassRaw,
		RawRefID:        s.Paths.RawRefIDForDigest(digestHex),
		Source:          defaultSource(req.Source),
		SourceSessionID: strings.TrimSpace(req.SourceSessionID),
		Kind:            kind,
		Digest:          digest,
		OriginalPath:    filepath.ToSlash(originalPath),
		ArchivedPath:    filepath.ToSlash(blobPath),
		SizeBytes:       size,
		ObservedAt:      normalizeObservedAt(req.ObservedAt),
		LocalOnly:       true,
	}

	refCreated, err := s.writeRawRef(ref, digestHex)
	if err != nil {
		return ArchiveResult{}, err
	}
	return ArchiveResult{RawRef: ref, BlobCreated: blobCreated, RefCreated: refCreated}, nil
}

func (s Store) writeRawRef(ref RawRef, digestHex string) (bool, error) {
	if err := os.MkdirAll(s.Paths.RawRefsDir(), 0o700); err != nil {
		return false, fmt.Errorf("create raw refs directory: %w", err)
	}
	path := s.Paths.RawRefPathForDigest(digestHex)
	// #nosec G304 -- raw ref paths are derived from the local workspace root.
	if existing, err := os.ReadFile(path); err == nil {
		var current RawRef
		if err := json.Unmarshal(existing, &current); err != nil {
			return false, fmt.Errorf("invalid existing raw ref %s: %w", filepath.ToSlash(path), err)
		}
		if current.Digest != ref.Digest {
			return false, fmt.Errorf("raw ref collision at %s: %s vs %s", filepath.ToSlash(path), current.Digest, ref.Digest)
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect raw ref %s: %w", filepath.ToSlash(path), err)
	}

	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode raw ref %s: %w", ref.RawRefID, err)
	}
	data = append(data, '\n')
	if err := writeFileAtomic(path, data, 0o600); err != nil {
		return false, fmt.Errorf("write raw ref %s: %w", filepath.ToSlash(path), err)
	}
	return true, nil
}

// LoadState reads the current vault state file if it exists.
func (s Store) LoadState() (State, error) {
	// #nosec G304 -- the state path is derived from the local workspace root.
	data, err := os.ReadFile(s.Paths.VaultStatePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{SchemaVersion: VaultStateSchemaVersion, DataClass: qschema.DataClassRaw}, nil
		}
		return State{}, fmt.Errorf("read vault state %s: %w", filepath.ToSlash(s.Paths.VaultStatePath()), err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode vault state %s: %w", filepath.ToSlash(s.Paths.VaultStatePath()), err)
	}
	if state.SchemaVersion == "" {
		state.SchemaVersion = VaultStateSchemaVersion
	}
	if state.DataClass == "" {
		state.DataClass = qschema.DataClassRaw
	}
	return state, nil
}

// SaveState writes the current vault state file atomically.
func (s Store) SaveState(state State) error {
	state.SchemaVersion = VaultStateSchemaVersion
	state.DataClass = qschema.DataClassRaw
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode vault state: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(s.Paths.StateDir(), 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := writeFileAtomic(s.Paths.VaultStatePath(), data, 0o600); err != nil {
		return fmt.Errorf("write vault state %s: %w", filepath.ToSlash(s.Paths.VaultStatePath()), err)
	}
	return nil
}

// UpdateState loads, mutates, and saves the vault state atomically.
func (s Store) UpdateState(update func(*State)) error {
	return s.withStateLock(func() error {
		state, err := s.LoadState()
		if err != nil {
			return err
		}
		update(&state)
		return s.SaveState(state)
	})
}

func (s Store) withStateLock(fn func() error) error {
	if err := os.MkdirAll(s.Paths.StateDir(), 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	lockPath := filepath.Join(s.Paths.StateDir(), ".vault.lock")
	// #nosec G304 -- the lock path is under the resolved qratum state directory.
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open vault state lock %s: %w", filepath.ToSlash(lockPath), err)
	}
	defer func() {
		_ = lock.Close()
	}()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock vault state %s: %w", filepath.ToSlash(lockPath), err)
	}
	defer func() {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	}()
	return fn()
}

// ConfiguredDiskFreeMinBytes reads the supported worker disk-free guard from
// config.toml. The parser intentionally supports only the shipped key.
func (s Store) ConfiguredDiskFreeMinBytes() (int64, error) {
	path := filepath.Join(s.Paths.Root, "config.toml")
	// #nosec G304 -- config.toml lives under the resolved qratum workspace root.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read qratum config %s: %w", filepath.ToSlash(path), err)
	}
	return parseDiskFreeMinGB(data)
}

func parseDiskFreeMinGB(data []byte) (int64, error) {
	section := ""
	for lineNumber, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		if section != "worker" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "disk_free_min_gb" {
			continue
		}
		value = strings.TrimSpace(value)
		gb, err := strconv.ParseInt(value, 10, 64)
		if err != nil || gb < 0 {
			return 0, fmt.Errorf("invalid config worker.disk_free_min_gb on line %d", lineNumber+1)
		}
		const bytesPerGiB int64 = 1024 * 1024 * 1024
		if gb > (1<<63-1)/bytesPerGiB {
			return 0, fmt.Errorf("config worker.disk_free_min_gb on line %d is too large", lineNumber+1)
		}
		return gb * bytesPerGiB, nil
	}
	return 0, nil
}

// CheckMinFreeSpace fails when path's filesystem has less free space than min.
func CheckMinFreeSpace(path string, minFreeBytes int64) error {
	if minFreeBytes <= 0 {
		return nil
	}
	free, err := FreeSpaceBytes(path)
	if err != nil {
		return err
	}
	// #nosec G115 -- callers pass a validated non-negative byte threshold.
	required := uint64(minFreeBytes)
	if free < required {
		return fmt.Errorf("disk free below configured minimum: available=%d bytes required=%d bytes", free, minFreeBytes)
	}
	return nil
}

// FreeSpaceBytes returns free bytes available to the current user for path.
func FreeSpaceBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("inspect free disk space for %s: %w", filepath.ToSlash(path), err)
	}
	blockSize := uint64(stat.Bsize) // #nosec G115 -- a filesystem block size is always non-negative
	return stat.Bavail * blockSize, nil
}

// Summary scans the vault workspace and returns high-level counts.
func (s Store) Summary() (Summary, error) {
	info, err := os.Stat(s.Paths.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			state, stateErr := s.LoadState()
			if stateErr != nil {
				return Summary{}, stateErr
			}
			return Summary{Root: filepath.ToSlash(s.Paths.Root), Present: false, LastState: state}, nil
		}
		return Summary{}, fmt.Errorf("inspect qratum home %s: %w", filepath.ToSlash(s.Paths.Root), err)
	}
	if !info.IsDir() {
		return Summary{}, fmt.Errorf("qratum home %s is not a directory", filepath.ToSlash(s.Paths.Root))
	}

	blobCount, err := countFiles(s.Paths.RawBlobsDir())
	if err != nil {
		return Summary{}, err
	}
	refCount, err := countFiles(s.Paths.RawRefsDir())
	if err != nil {
		return Summary{}, err
	}
	state, err := s.LoadState()
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		Root:      filepath.ToSlash(s.Paths.Root),
		Present:   true,
		BlobCount: blobCount,
		RefCount:  refCount,
		LastState: state,
	}, nil
}

// ListRawRefs loads every raw ref record in the vault.
func (s Store) ListRawRefs() ([]RawRef, error) {
	dir := s.Paths.RawRefsDir()
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect raw refs %s: %w", filepath.ToSlash(dir), err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list raw refs %s: %w", filepath.ToSlash(dir), err)
	}
	sort.Strings(paths)
	refs := make([]RawRef, 0, len(paths))
	for _, path := range paths {
		// #nosec G304 -- raw ref paths are derived from the local workspace root.
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read raw ref %s: %w", filepath.ToSlash(path), err)
		}
		var ref RawRef
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, fmt.Errorf("decode raw ref %s: %w", filepath.ToSlash(path), err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// GarbageCollectOrphanBlobs removes only blobs with no live raw ref.
func (s Store) GarbageCollectOrphanBlobs() (GCResult, error) {
	refs, err := s.ListRawRefs()
	if err != nil {
		return GCResult{}, err
	}
	referenced := map[string]struct{}{}
	for _, ref := range refs {
		digestHex := strings.TrimPrefix(ref.Digest, "sha256:")
		if digestHex != "" {
			referenced[digestHex] = struct{}{}
		}
	}
	tombstoned, err := s.tombstonedDigests()
	if err != nil {
		return GCResult{}, err
	}
	result := GCResult{}
	root := s.Paths.RawBlobsDir()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		digestHex := filepath.Base(path)
		if _, ok := referenced[digestHex]; ok {
			result.ReferencedKept++
			return nil
		}
		if _, ok := tombstoned[digestHex]; ok {
			result.TombstonedKept++
			return nil
		}
		// #nosec G122 -- WalkDir is constrained to the vault blob root and only removes unreferenced blob paths.
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove orphan blob %s: %w", filepath.ToSlash(path), err)
		}
		result.OrphansRemoved++
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return result, nil
		}
		return GCResult{}, fmt.Errorf("garbage collect raw blobs %s: %w", filepath.ToSlash(root), err)
	}
	return result, nil
}

// EraseRawRef records and performs an explicit raw blob erasure.
func (s Store) EraseRawRef(rawRefID string, reason string, erasedAt string) (EraseResult, error) {
	rawRefID = strings.TrimSpace(rawRefID)
	reason = strings.TrimSpace(reason)
	if rawRefID == "" {
		return EraseResult{}, fmt.Errorf("missing raw_ref_id")
	}
	if reason == "" {
		return EraseResult{}, fmt.Errorf("missing erasure reason")
	}
	digestHex, ok := strings.CutPrefix(rawRefID, "raw_")
	if !ok || !isHexDigest(digestHex) {
		return EraseResult{}, fmt.Errorf("invalid raw_ref_id %q", rawRefID)
	}
	refPath := s.Paths.RawRefPathForDigest(digestHex)
	// #nosec G304 -- raw ref path is derived from a validated raw_ref_id.
	data, err := os.ReadFile(refPath)
	if err != nil {
		return EraseResult{}, fmt.Errorf("read raw ref %s: %w", filepath.ToSlash(refPath), err)
	}
	var ref RawRef
	if err := json.Unmarshal(data, &ref); err != nil {
		return EraseResult{}, fmt.Errorf("decode raw ref %s: %w", filepath.ToSlash(refPath), err)
	}
	if ref.RawRefID != rawRefID {
		return EraseResult{}, fmt.Errorf("raw ref file %s contains raw_ref_id %q", filepath.ToSlash(refPath), ref.RawRefID)
	}
	if erasedAt = strings.TrimSpace(erasedAt); erasedAt == "" {
		erasedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	tombstone := RawTombstone{
		SchemaVersion: RawTombstoneSchemaVersion,
		DataClass:     qschema.DataClassRaw,
		RawRefID:      rawRefID,
		Digest:        ref.Digest,
		Reason:        reason,
		ErasedAt:      erasedAt,
		BlobRemoved:   false,
	}
	tombstonePath := s.Paths.RawTombstonePathForDigest(digestHex)
	if err := writeFileAtomic(tombstonePath, mustMarshalTombstone(tombstone), 0o600); err != nil {
		return EraseResult{}, fmt.Errorf("write raw tombstone %s: %w", filepath.ToSlash(tombstonePath), err)
	}
	blobPath := s.Paths.BlobPathForDigest(digestHex)
	if err := os.Remove(blobPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return EraseResult{}, fmt.Errorf("remove raw blob %s: %w", filepath.ToSlash(blobPath), err)
		}
	} else {
		tombstone.BlobRemoved = true
		if err := writeFileAtomic(tombstonePath, mustMarshalTombstone(tombstone), 0o600); err != nil {
			return EraseResult{}, fmt.Errorf("update raw tombstone %s: %w", filepath.ToSlash(tombstonePath), err)
		}
	}
	return EraseResult{
		Tombstone:   tombstone,
		BlobPath:    filepath.ToSlash(blobPath),
		RefPath:     filepath.ToSlash(refPath),
		BlobRemoved: tombstone.BlobRemoved,
	}, nil
}

func (s Store) tombstonedDigests() (map[string]struct{}, error) {
	dir := s.Paths.RawTombstonesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]struct{}{}, nil
		}
		return nil, fmt.Errorf("read raw tombstones %s: %w", filepath.ToSlash(dir), err)
	}
	result := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		// #nosec G304 -- tombstone paths come from walking the tombstone dir.
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read raw tombstone %s: %w", filepath.ToSlash(path), err)
		}
		var tombstone RawTombstone
		if err := json.Unmarshal(data, &tombstone); err != nil {
			return nil, fmt.Errorf("decode raw tombstone %s: %w", filepath.ToSlash(path), err)
		}
		digestHex := strings.TrimPrefix(tombstone.Digest, "sha256:")
		if digestHex != "" {
			result[digestHex] = struct{}{}
		}
	}
	return result, nil
}

func isHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

func mustMarshalTombstone(tombstone RawTombstone) []byte {
	data, err := json.MarshalIndent(tombstone, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

// SweepStaleTempBlobs removes abandoned temp blobs older than staleAfter.
func (s Store) SweepStaleTempBlobs(staleAfter time.Duration, now time.Time) (int, error) {
	dir := s.Paths.RawBlobsTempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read blob temp directory %s: %w", filepath.ToSlash(dir), err)
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return removed, fmt.Errorf("inspect temp blob %s: %w", filepath.ToSlash(path), err)
		}
		if now.Sub(info.ModTime()) < staleAfter {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, fmt.Errorf("remove stale temp blob %s: %w", filepath.ToSlash(path), err)
		}
		removed++
	}
	return removed, nil
}

// Backup copies the full vault workspace to a destination directory.
func (s Store) Backup(dest string, verify bool, allowRawEgress bool) (BackupResult, error) {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return BackupResult{}, fmt.Errorf("missing backup destination")
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return BackupResult{}, fmt.Errorf("resolve backup destination %q: %w", dest, err)
	}
	if absDest == s.Paths.Root {
		return BackupResult{}, fmt.Errorf("backup destination must differ from qratum home")
	}
	rawBearing, err := hasFiles(s.Paths.RawDir())
	if err != nil {
		return BackupResult{}, err
	}
	if rawBearing && !allowRawEgress {
		return BackupResult{}, fmt.Errorf("backup includes raw vault bytes; rerun with --allow-raw-egress after confirming the destination is approved")
	}
	if rawBearing {
		if err := s.appendRawEgressAudit(absDest); err != nil {
			return BackupResult{}, err
		}
	}

	fileCount, err := copyTree(s.Paths.Root, absDest)
	if err != nil {
		return BackupResult{}, err
	}
	result := BackupResult{Destination: filepath.ToSlash(absDest), FileCount: fileCount, RawEgress: rawBearing}
	if verify {
		if err := verifyTree(s.Paths.Root, absDest); err != nil {
			return BackupResult{}, err
		}
		result.Verified = true
	}
	return result, nil
}

func (s Store) appendRawEgressAudit(dest string) error {
	path := filepath.Join(s.Paths.StateDir(), "raw-egress-audit.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create raw egress audit directory: %w", err)
	}
	line := fmt.Sprintf("{\"event\":\"raw_egress_ack\",\"destination\":%q,\"observed_at\":%q}\n", filepath.ToSlash(dest), time.Now().UTC().Format(time.RFC3339Nano))
	// #nosec G304 -- the audit path is under the resolved qratum state directory.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open raw egress audit %s: %w", filepath.ToSlash(path), err)
	}
	if _, err := file.WriteString(line); err != nil {
		_ = file.Close()
		return fmt.Errorf("write raw egress audit %s: %w", filepath.ToSlash(path), err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close raw egress audit %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

// DetectSource infers a source label from a local filesystem path.
func DetectSource(path string) string {
	slash := filepath.ToSlash(path)
	switch {
	case strings.Contains(slash, "/.claude/"):
		return SourceClaudeCode
	case strings.Contains(slash, "/.codex/"):
		return SourceCodex
	default:
		return SourceManual
	}
}

func defaultSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return SourceManual
	}
	return source
}

func normalizeObservedAt(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func countFiles(root string) (int, error) {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("inspect directory %s: %w", filepath.ToSlash(root), err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("path %s is not a directory", filepath.ToSlash(root))
	}
	count := 0
	err = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			count++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk directory %s: %w", filepath.ToSlash(root), err)
	}
	return count, nil
}

func hasFiles(root string) (bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("inspect directory %s: %w", filepath.ToSlash(root), err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("path %s is not a directory", filepath.ToSlash(root))
	}
	found := false
	err = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}

func copyTree(source string, dest string) (int, error) {
	info, err := os.Stat(source)
	if err != nil {
		return 0, fmt.Errorf("inspect qratum home %s: %w", filepath.ToSlash(source), err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("qratum home %s is not a directory", filepath.ToSlash(source))
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return 0, fmt.Errorf("create backup destination %s: %w", filepath.ToSlash(dest), err)
	}

	count := 0
	err = filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if isPathWithin(path, filepath.Join(source, "raw", "blobs", ".tmp")) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := copyFileAtomic(path, target, info.Mode().Perm()); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("copy qratum home to %s: %w", filepath.ToSlash(dest), err)
	}
	return count, nil
}

func verifyTree(source string, dest string) error {
	blobDigests, err := rawRefBlobDigests(filepath.Join(source, "raw", "refs"), source)
	if err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if isPathWithin(path, filepath.Join(source, "raw", "blobs", ".tmp")) {
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
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		destHash, err := fileHash(target)
		if err != nil {
			return err
		}
		if digest, ok := blobDigests[filepath.ToSlash(rel)]; ok {
			if "sha256:"+destHash != digest {
				return fmt.Errorf("backup verify mismatch for %s against recorded digest", filepath.ToSlash(rel))
			}
			return nil
		}
		sourceHash, err := fileHash(path)
		if err != nil {
			return err
		}
		if sourceHash != destHash {
			return fmt.Errorf("backup verify mismatch for %s", filepath.ToSlash(rel))
		}
		return nil
	})
}

func rawRefBlobDigests(refsDir string, sourceRoot string) (map[string]string, error) {
	digests := map[string]string{}
	paths, err := filepath.Glob(filepath.Join(refsDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list raw refs %s: %w", filepath.ToSlash(refsDir), err)
	}
	for _, path := range paths {
		// #nosec G304 -- raw refs are inside the walked vault root.
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read raw ref %s: %w", filepath.ToSlash(path), err)
		}
		var ref RawRef
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, fmt.Errorf("decode raw ref %s: %w", filepath.ToSlash(path), err)
		}
		rel, err := filepath.Rel(sourceRoot, filepath.FromSlash(ref.ArchivedPath))
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			digestHex := strings.TrimPrefix(ref.Digest, "sha256:")
			if len(digestHex) < 2 {
				return nil, fmt.Errorf("raw ref %s has invalid digest %q", ref.RawRefID, ref.Digest)
			}
			rel, err = filepath.Rel(sourceRoot, filepath.Join(sourceRoot, "raw", "blobs", "sha256", digestHex[:2], digestHex))
			if err != nil {
				return nil, fmt.Errorf("resolve blob path for raw ref %s: %w", ref.RawRefID, err)
			}
		}
		digests[filepath.ToSlash(rel)] = ref.Digest
	}
	return digests, nil
}

func fileHash(path string) (string, error) {
	// #nosec G304 -- verification paths come from walking the local workspace tree.
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filepath.ToSlash(path), err)
	}
	defer func() {
		_ = file.Close()
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", filepath.ToSlash(path), err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func copyFileAtomic(source string, target string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	// #nosec G304 -- backup source paths come from walking the local vault root.
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		_ = src.Close()
	}()
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return err
	}
	removeTmp = false
	return nil
}

func isPathWithin(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	removeTmp = false
	return nil
}

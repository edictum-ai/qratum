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
	"strings"
	"time"

	"github.com/edictum-ai/qratum/internal/workspace"
)

const (
	// RawRefSchemaVersion is the schema version stored in raw ref records.
	RawRefSchemaVersion = "qratum.raw_ref.v1"
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
)

// Store wraps access to the local vault workspace.
type Store struct {
	Paths workspace.Paths
}

// RawRef records one content-addressed raw archive item.
type RawRef struct {
	SchemaVersion   string `json:"schema_version"`
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
			return State{SchemaVersion: VaultStateSchemaVersion}, nil
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
	return state, nil
}

// SaveState writes the current vault state file atomically.
func (s Store) SaveState(state State) error {
	state.SchemaVersion = VaultStateSchemaVersion
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
	state, err := s.LoadState()
	if err != nil {
		return err
	}
	update(&state)
	return s.SaveState(state)
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

// Backup copies the full vault workspace to a destination directory.
func (s Store) Backup(dest string, verify bool) (BackupResult, error) {
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

	fileCount, err := copyTree(s.Paths.Root, absDest)
	if err != nil {
		return BackupResult{}, err
	}
	result := BackupResult{Destination: filepath.ToSlash(absDest), FileCount: fileCount}
	if verify {
		if err := verifyTree(s.Paths.Root, absDest); err != nil {
			return BackupResult{}, err
		}
		result.Verified = true
	}
	return result, nil
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
		// #nosec G304,G122 -- backup copy paths come from walking the local workspace tree inside the vault root.
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := writeFileAtomic(target, data, info.Mode().Perm()); err != nil {
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
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
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
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		sourceHash, err := fileHash(path)
		if err != nil {
			return err
		}
		destHash, err := fileHash(target)
		if err != nil {
			return err
		}
		if sourceHash != destHash {
			return fmt.Errorf("backup verify mismatch for %s", filepath.ToSlash(rel))
		}
		return nil
	})
}

func fileHash(path string) (string, error) {
	// #nosec G304 -- verification paths come from walking the local workspace tree.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
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

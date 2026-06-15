package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	claudecfg "github.com/edictum-ai/qratum/internal/claude"
	"github.com/edictum-ai/qratum/internal/vault"
	"github.com/edictum-ai/qratum/internal/workspace"
)

const (
	backfillStaleAfter = 7 * 24 * time.Hour
	backupStaleAfter   = 7 * 24 * time.Hour
)

func vaultCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing vault command")
		return 2
	}
	switch args[0] {
	case "doctor":
		return vaultDoctor(args[1:], stdout, stderr)
	case "backfill":
		return vaultBackfill(args[1:], stdout, stderr)
	case "archive":
		return vaultArchive(args[1:], stdout, stderr)
	case "backup":
		return vaultBackup(args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		fmt.Fprintf(stderr, "error: unsupported vault command %q\n", args[0])
		return 2
	}
}

func vaultDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault doctor does not accept arguments")
		return 2
	}

	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	store := vault.New(qratumHome)
	summary, err := store.Summary()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	hookStatus, err := claudecfg.HookStatusForProject(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	refs, err := store.ListRawRefs()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	transcripts, err := claudecfg.ListTranscriptFiles()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	transcriptRefCount := 0
	for _, ref := range refs {
		switch ref.Kind {
		case vault.KindMainTranscript, vault.KindSubagentTranscript:
			transcriptRefCount++
		}
	}

	warnings := make([]string, 0, 6)
	if !hookStatus.GlobalInstalled {
		warnings = append(warnings, "global SessionEnd hook is not installed")
	}
	backfillStatus := "ok"
	if stale(summary.LastState.LastBackfillAt, backfillStaleAfter) {
		backfillStatus = "stale"
		warnings = append(warnings, "backfill is missing or stale")
	}
	backupStatus := "verified"
	if strings.TrimSpace(summary.LastState.LastBackupVerifiedAt) == "" {
		backupStatus = "missing_or_unverified"
		warnings = append(warnings, "backup has never been verified")
	} else if stale(summary.LastState.LastBackupVerifiedAt, backupStaleAfter) {
		backupStatus = "stale"
		warnings = append(warnings, "verified backup is stale")
	}
	if summary.LastState.CopyFailureCount > 0 {
		warnings = append(warnings, fmt.Sprintf("copy failures recorded: %d", summary.LastState.CopyFailureCount))
	}
	driftStatus := "unavailable"
	driftValue := 0
	if transcripts != nil {
		driftValue = len(transcripts) - transcriptRefCount
		driftStatus = fmt.Sprintf("%+d (known=%d archived=%d)", driftValue, len(transcripts), transcriptRefCount)
		if driftValue != 0 {
			warnings = append(warnings, "blob-vs-known-transcript drift detected")
		}
	}

	fmt.Fprintln(stdout, "qratum vault doctor")
	fmt.Fprintf(stdout, "qratum_home: %s\n", summary.Root)
	fmt.Fprintf(stdout, "hook_installed: %s\n", yesNo(hookStatus.GlobalInstalled))
	fmt.Fprintf(stdout, "last_capture_at: %s\n", dashIfEmpty(summary.LastState.LastCaptureAt))
	fmt.Fprintf(stdout, "last_backfill_at: %s\n", dashIfEmpty(summary.LastState.LastBackfillAt))
	fmt.Fprintf(stdout, "backfill_status: %s\n", backfillStatus)
	fmt.Fprintf(stdout, "copy_failures: %d\n", summary.LastState.CopyFailureCount)
	fmt.Fprintf(stdout, "raw_missing: %d\n", summary.LastState.RawMissingCount)
	fmt.Fprintf(stdout, "blob_count: %d\n", summary.BlobCount)
	fmt.Fprintf(stdout, "raw_ref_count: %d\n", summary.RefCount)
	fmt.Fprintf(stdout, "transcript_drift: %s\n", driftStatus)
	fmt.Fprintf(stdout, "backup_verified_at: %s\n", dashIfEmpty(summary.LastState.LastBackupVerifiedAt))
	fmt.Fprintf(stdout, "backup_status: %s\n", backupStatus)
	fmt.Fprintln(stdout, "cloud_sessions: sessions that start and end on vendor infra are not captured in vault v1")
	if len(warnings) == 0 {
		fmt.Fprintln(stdout, "warnings: none")
	} else {
		fmt.Fprintln(stdout, "warnings:")
		for _, warning := range warnings {
			fmt.Fprintf(stdout, "- %s\n", warning)
		}
	}
	return 0
}

func vaultBackfill(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault backfill does not accept arguments")
		return 2
	}
	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	store := vault.New(qratumHome)
	transcripts, err := claudecfg.ListTranscriptFiles()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if transcripts == nil {
		fmt.Fprintf(stderr, "error: Claude projects directory %s does not exist\n", filepath.ToSlash(filepath.Join(mustClaudeRoot(stderr), "projects")))
		return 1
	}

	archived, deduped, failed := 0, 0, 0
	for _, transcript := range transcripts {
		result, err := store.ArchiveFile(vault.ArchiveRequest{
			Source:       vault.SourceClaudeCode,
			Kind:         transcript.Kind,
			OriginalPath: transcript.Path,
			ObservedAt:   currentTimestamp(),
		})
		if err != nil {
			failed++
			fmt.Fprintf(stderr, "error: backfill %s: %v\n", transcript.Path, err)
			continue
		}
		if result.RefCreated {
			archived++
		} else {
			deduped++
		}
	}
	if err := store.UpdateState(func(state *vault.State) {
		state.LastBackfillAt = currentTimestamp()
	}); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum vault backfill")
	fmt.Fprintf(stdout, "transcripts_seen: %d\n", len(transcripts))
	fmt.Fprintf(stdout, "archived: %d\n", archived)
	fmt.Fprintf(stdout, "deduped: %d\n", deduped)
	fmt.Fprintf(stdout, "failed: %d\n", failed)
	if failed > 0 {
		return 1
	}
	return 0
}

func vaultArchive(args []string, stdout io.Writer, stderr io.Writer) int {
	kind, target, err := parseArchiveArgs(args)
	if err != nil {
		if errors.Is(err, errUsage) {
			printUsage(stderr)
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 2
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	store := vault.New(qratumHome)
	paths, err := collectArchivePaths(target)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	archived, deduped := 0, 0
	for _, path := range paths {
		result, err := store.ArchiveFile(vault.ArchiveRequest{
			Source:       vault.DetectSource(path),
			Kind:         kind,
			OriginalPath: path,
			ObservedAt:   currentTimestamp(),
		})
		if err != nil {
			fmt.Fprintf(stderr, "error: archive %s: %v\n", filepath.ToSlash(path), err)
			return 1
		}
		if result.RefCreated {
			archived++
		} else {
			deduped++
		}
	}
	if err := store.UpdateState(func(state *vault.State) {
		state.LastArchiveAt = currentTimestamp()
	}); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum vault archive")
	fmt.Fprintf(stdout, "kind: %s\n", kind)
	fmt.Fprintf(stdout, "files_seen: %d\n", len(paths))
	fmt.Fprintf(stdout, "archived: %d\n", archived)
	fmt.Fprintf(stdout, "deduped: %d\n", deduped)
	return 0
}

func vaultBackup(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("vault backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	verify := fs.Bool("verify", false, "")
	if err := fs.Parse(args); err != nil {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault backup usage: vault backup [--verify] <dest>")
		return 2
	}
	if fs.NArg() != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault backup usage: vault backup [--verify] <dest>")
		return 2
	}

	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	store := vault.New(qratumHome)
	result, err := store.Backup(fs.Arg(0), *verify)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	now := currentTimestamp()
	if err := store.UpdateState(func(state *vault.State) {
		state.LastBackupAt = now
		state.LastBackupDest = result.Destination
		if result.Verified {
			state.LastBackupVerifiedAt = now
			state.LastBackupVerifiedDest = result.Destination
		}
	}); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum vault backup")
	fmt.Fprintf(stdout, "destination: %s\n", result.Destination)
	fmt.Fprintf(stdout, "files_copied: %d\n", result.FileCount)
	fmt.Fprintf(stdout, "verified: %s\n", yesNo(result.Verified))
	return 0
}

var errUsage = errors.New("invalid usage")

func parseArchiveArgs(args []string) (string, string, error) {
	kind := vault.KindSourceMetadata
	target := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--kind":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("%w: vault archive requires a value after --kind", errUsage)
			}
			kind = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return "", "", fmt.Errorf("%w: unsupported archive flag %q", errUsage, args[i])
			}
			if target != "" {
				return "", "", fmt.Errorf("%w: vault archive accepts exactly one path", errUsage)
			}
			target = args[i]
		}
	}
	if target == "" {
		return "", "", fmt.Errorf("%w: missing archive path", errUsage)
	}
	if !vault.IsValidKind(kind) {
		return "", "", fmt.Errorf("unsupported raw kind %q", kind)
	}
	return kind, target, nil
}

func collectArchivePaths(root string) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("missing archive path")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve archive path %q: %w", root, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("inspect archive path %s: %w", filepath.ToSlash(abs), err)
	}
	if !info.IsDir() {
		return []string{abs}, nil
	}
	paths := make([]string, 0, 16)
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode().IsRegular() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk archive path %s: %w", filepath.ToSlash(abs), err)
	}
	sort.Strings(paths)
	return paths, nil
}

func stale(timestamp string, limit time.Duration) bool {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		return true
	}
	parsed, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return true
	}
	return time.Since(parsed) > limit
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func mustClaudeRoot(stderr io.Writer) string {
	root, err := claudecfg.Root()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ".claude"
	}
	return root
}

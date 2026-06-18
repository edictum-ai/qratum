package main

import (
	"encoding/json"
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
	"github.com/edictum-ai/qratum/internal/schedule"
	qschema "github.com/edictum-ai/qratum/internal/schema"
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
	case "gc":
		return vaultGC(args[1:], stdout, stderr)
	case "erase":
		return vaultErase(args[1:], stdout, stderr)
	case "install-schedule":
		return vaultInstallSchedule(args[1:], stdout, stderr)
	case "uninstall-schedule":
		return vaultUninstallSchedule(args[1:], stdout, stderr)
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
	swept, err := store.SweepStaleTempBlobs(vault.DefaultTempBlobStaleAfter, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
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
	scheduleInstalled, err := schedule.IsInstalled(schedule.Options{})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	refs, err := store.ListRawRefs()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	minFreeBytes, err := store.ConfiguredDiskFreeMinBytes()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	expectedCapturedTranscripts, err := countCapturedTranscriptEvents(qratumHome.EventsDir())
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
	if !scheduleInstalled {
		warnings = append(warnings, "preservation freshness depends on a schedule that is not installed")
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
	diskFreeStatus := "unconfigured"
	if minFreeBytes > 0 {
		freeBytes, err := vault.FreeSpaceBytes(qratumHome.Root)
		switch {
		case err != nil:
			diskFreeStatus = "unknown"
			warnings = append(warnings, err.Error())
		case freeBytes < uint64(minFreeBytes):
			diskFreeStatus = "low"
			warnings = append(warnings, fmt.Sprintf("disk free below configured minimum: available=%d bytes required=%d bytes", freeBytes, minFreeBytes))
		default:
			diskFreeStatus = "ok"
		}
	}
	driftValue := expectedCapturedTranscripts - transcriptRefCount
	driftStatus := fmt.Sprintf("%+d (expected=%d archived=%d)", driftValue, expectedCapturedTranscripts, transcriptRefCount)
	if driftValue > 0 {
		warnings = append(warnings, "transcript drift heuristic indicates captured transcripts without refs")
	}

	fmt.Fprintln(stdout, "qratum vault doctor")
	fmt.Fprintf(stdout, "qratum_home: %s\n", summary.Root)
	fmt.Fprintf(stdout, "hook_installed: %s\n", yesNo(hookStatus.GlobalInstalled))
	fmt.Fprintf(stdout, "schedule_installed: %s\n", yesNo(scheduleInstalled))
	fmt.Fprintf(stdout, "last_capture_at: %s\n", dashIfEmpty(summary.LastState.LastCaptureAt))
	fmt.Fprintf(stdout, "last_backfill_at: %s\n", dashIfEmpty(summary.LastState.LastBackfillAt))
	fmt.Fprintf(stdout, "backfill_status: %s\n", backfillStatus)
	fmt.Fprintf(stdout, "copy_failures: %d\n", summary.LastState.CopyFailureCount)
	fmt.Fprintf(stdout, "raw_missing: %d\n", summary.LastState.RawMissingCount)
	fmt.Fprintf(stdout, "blob_count: %d\n", summary.BlobCount)
	fmt.Fprintf(stdout, "raw_ref_count: %d\n", summary.RefCount)
	fmt.Fprintf(stdout, "stale_temp_blobs_swept: %d\n", swept)
	fmt.Fprintf(stdout, "disk_free_status: %s\n", diskFreeStatus)
	fmt.Fprintf(stdout, "transcript_drift (heuristic): %s\n", driftStatus)
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
	if _, err := store.SweepStaleTempBlobs(vault.DefaultTempBlobStaleAfter, time.Now()); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	minFreeBytes, err := store.ConfiguredDiskFreeMinBytes()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
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
			MinFreeBytes: minFreeBytes,
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
	minFreeBytes, err := store.ConfiguredDiskFreeMinBytes()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	paths, err := collectArchivePaths(target)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	archived, deduped := 0, 0
	for _, path := range paths {
		if kind == vault.KindMemoryImport {
			if err := validateMemoryImportReceipt(path); err != nil {
				fmt.Fprintf(stderr, "error: archive %s: %v\n", filepath.ToSlash(path), err)
				return 1
			}
		} else if kind == vault.KindSourceMetadata && looksMemoryImportReceipt(path) {
			fmt.Fprintln(stderr, "warning: receipt-shaped input archived as source_metadata; rerun with --kind memory_import_receipt to pin the kind")
		}
		result, err := store.ArchiveFile(vault.ArchiveRequest{
			Source:       vault.DetectSource(path),
			Kind:         kind,
			OriginalPath: path,
			ObservedAt:   currentTimestamp(),
			MinFreeBytes: minFreeBytes,
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

func vaultGC(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault gc does not accept arguments")
		return 2
	}
	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	result, err := vault.New(qratumHome).GarbageCollectOrphanBlobs()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "qratum vault gc")
	fmt.Fprintf(stdout, "orphans_removed: %d\n", result.OrphansRemoved)
	fmt.Fprintf(stdout, "referenced_kept: %d\n", result.ReferencedKept)
	fmt.Fprintf(stdout, "tombstoned_kept: %d\n", result.TombstonedKept)
	return 0
}

func vaultErase(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("vault erase", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reason := fs.String("reason", "", "")
	if err := fs.Parse(args); err != nil {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault erase usage: vault erase --reason <reason> <raw_ref_id>")
		return 2
	}
	if fs.NArg() != 1 || strings.TrimSpace(*reason) == "" {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault erase usage: vault erase --reason <reason> <raw_ref_id>")
		return 2
	}
	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	result, err := vault.New(qratumHome).EraseRawRef(fs.Arg(0), *reason, currentTimestamp())
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "qratum vault erase")
	fmt.Fprintf(stdout, "raw_ref_id: %s\n", result.Tombstone.RawRefID)
	fmt.Fprintf(stdout, "blob_removed: %s\n", yesNo(result.BlobRemoved))
	fmt.Fprintf(stdout, "tombstone: %s\n", filepath.ToSlash(qratumHome.RawTombstonePathForDigest(strings.TrimPrefix(result.Tombstone.RawRefID, "raw_"))))
	return 0
}

func vaultInstallSchedule(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("vault install-schedule", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	printOnly := fs.Bool("print", false, "")
	platform := fs.String("platform", "", "")
	if err := fs.Parse(args); err != nil {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault install-schedule usage: vault install-schedule [--print]")
		return 2
	}
	if fs.NArg() != 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault install-schedule does not accept arguments")
		return 2
	}
	options := schedule.Options{Platform: *platform}
	if *printOnly {
		plan, err := schedule.BuildPlan(options)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		printScheduleInstall(stdout, plan, true, false)
		return 0
	}
	result, err := schedule.Install(options)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	printScheduleInstall(stdout, result.Plan, false, result.Changed)
	return 0
}

func vaultUninstallSchedule(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("vault uninstall-schedule", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	platform := fs.String("platform", "", "")
	if err := fs.Parse(args); err != nil {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault uninstall-schedule usage: vault uninstall-schedule")
		return 2
	}
	if fs.NArg() != 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault uninstall-schedule does not accept arguments")
		return 2
	}
	result, err := schedule.Uninstall(schedule.Options{Platform: *platform})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "qratum vault uninstall-schedule")
	fmt.Fprintf(stdout, "removed: %d\n", result.Removed)
	if result.Removed == 0 {
		fmt.Fprintln(stdout, "nothing_to_remove: yes")
	} else {
		fmt.Fprintln(stdout, "nothing_to_remove: no")
	}
	for _, path := range result.Paths {
		fmt.Fprintf(stdout, "path: %s\n", filepath.ToSlash(path))
	}
	return 0
}

func printScheduleInstall(stdout io.Writer, plan schedule.Plan, dryRun bool, changed bool) {
	fmt.Fprintln(stdout, "qratum vault install-schedule")
	fmt.Fprintf(stdout, "platform: %s\n", plan.Platform)
	fmt.Fprintf(stdout, "dry_run: %s\n", yesNo(dryRun))
	fmt.Fprintf(stdout, "changed: %s\n", yesNo(changed))
	if !changed && !dryRun {
		fmt.Fprintln(stdout, "already_installed: yes")
	}
	fmt.Fprintf(stdout, "command: %s\n", strings.Join(plan.Command, " "))
	for _, file := range plan.Files {
		fmt.Fprintf(stdout, "write: %s\n", filepath.ToSlash(file.Path))
		fmt.Fprintf(stdout, "--- %s ---\n", filepath.Base(file.Path))
		fmt.Fprint(stdout, string(file.Content))
	}
}

func vaultBackup(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("vault backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	verify := fs.Bool("verify", false, "")
	allowRawEgress := fs.Bool("allow-raw-egress", false, "")
	if err := fs.Parse(args); err != nil {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault backup usage: vault backup [--verify] [--allow-raw-egress] <dest>")
		return 2
	}
	if fs.NArg() != 1 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: vault backup usage: vault backup [--verify] [--allow-raw-egress] <dest>")
		return 2
	}

	qratumHome, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	store := vault.New(qratumHome)
	result, err := store.Backup(fs.Arg(0), *verify, *allowRawEgress)
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
	if result.RawEgress {
		fmt.Fprintln(stdout, "raw_egress_ack: raw vault bytes copied by explicit operator request")
	}
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

func validateMemoryImportReceipt(path string) error {
	// #nosec G304 -- receipt paths are explicit operator archive inputs.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read memory import receipt: %w", err)
	}
	schemaData, err := readSchemaFile("qratum-memory-import-receipt.v1.schema.json")
	if err != nil {
		return err
	}
	if err := qschema.Validate(schemaData, data); err != nil {
		return fmt.Errorf("invalid memory import receipt: %w", err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(data, &receipt); err != nil {
		return fmt.Errorf("decode memory import receipt: %w", err)
	}
	if receipt["error_class"] == "namespace_forbidden" {
		return fmt.Errorf("memory import receipt has namespace_forbidden and is not archived")
	}
	return nil
}

func looksMemoryImportReceipt(path string) bool {
	// #nosec G304 -- receipt sniffing reads an explicit operator archive input.
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	return doc["schema_version"] == "qratum.memory_import_receipt.v1"
}

func readSchemaFile(name string) ([]byte, error) {
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "schemas", name))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "schemas", name))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "schemas", name))
	}
	for _, candidate := range candidates {
		// #nosec G304 -- schema candidates are under the repo or qrt install directory.
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read schema %s: %w", filepath.ToSlash(candidate), err)
		}
	}
	return nil, fmt.Errorf("schema %s not found; run from the qratum repo or install schemas beside qrt", name)
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
	return nowUTC().Sub(parsed) > limit
}

func countCapturedTranscriptEvents(eventsDir string) (int, error) {
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read capture events %s: %w", filepath.ToSlash(eventsDir), err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(eventsDir, entry.Name())
		// #nosec G304 -- event files come from the resolved qratum event spool.
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, fmt.Errorf("read capture event %s: %w", filepath.ToSlash(path), err)
		}
		var event captureEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return 0, fmt.Errorf("decode capture event %s: %w", filepath.ToSlash(path), err)
		}
		if event.EventType != "session_end" || event.Raw == nil {
			continue
		}
		switch event.Raw.CopyStatus {
		case "copied", "deduped":
			count++
		}
	}
	return count, nil
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

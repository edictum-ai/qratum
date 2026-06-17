package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/edictum-ai/qratum/internal/vault"
	"github.com/edictum-ai/qratum/internal/workspace"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via -ldflags "-X main.version={{.Version}}".
var version = "dev"

func main() {
	os.Exit(runWithIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWithIO(args, strings.NewReader(""), stdout, stderr)
}

func runWithIO(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing command")
		return 2
	}

	switch args[0] {
	case "--version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "error: --version does not accept arguments")
			printUsage(stderr)
			return 2
		}
		fmt.Fprintf(stdout, "qrt %s\n", version)
		return 0
	case "status":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "error: status does not accept arguments")
			printUsage(stderr)
			return 2
		}
		return status(stdout, stderr)
	case "hook":
		return hook(args[1:], stdin, stdout, stderr)
	case "vault":
		return vaultCommand(args[1:], stdout, stderr)
	case "daemon":
		return daemon(args[1:], stdout, stderr)
	case "dogfood":
		return dogfood(args[1:], stdout, stderr)
	case "normalize":
		return normalize(args[1:], stdout, stderr)
	case "redact":
		return redact(args[1:], stdout, stderr)
	case "evidence":
		return evidence(args[1:], stdout, stderr)
	case "review":
		return review(args[1:], stdout, stderr)
	case "report":
		return report(args[1:], stdout, stderr)
	case "export":
		return exportCommand(args[1:], stdout, stderr)
	case "sessions":
		return sessions(args[1:], stdout, stderr)
	case "ui":
		return ui(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unsupported command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func status(stdout io.Writer, stderr io.Writer) int {
	paths, err := workspace.Resolve()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	state, err := qratumDirState(paths.Root)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	summary, err := vault.New(paths).Summary()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum status")
	fmt.Fprintln(stdout, "milestone: vault-first")
	fmt.Fprintf(stdout, "version: %s\n", version)
	fmt.Fprintf(stdout, "qratum_home: %s\n", filepath.ToSlash(paths.Root))
	fmt.Fprintf(stdout, "qratum_home_state: %s\n", state)
	fmt.Fprintf(stdout, "vault_blobs: %d\n", summary.BlobCount)
	fmt.Fprintf(stdout, "vault_refs: %d\n", summary.RefCount)
	fmt.Fprintf(stdout, "last_backfill_at: %s\n", dashIfEmpty(summary.LastState.LastBackfillAt))
	fmt.Fprintf(stdout, "copy_failures: %d\n", summary.LastState.CopyFailureCount)
	fmt.Fprintln(stdout, "ready: true")
	return 0
}

func qratumDirState(path string) (string, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("%s exists but is not a directory", path)
		}
		return "present", nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "missing", nil
	}
	return "", fmt.Errorf("cannot inspect %s: %w", path, err)
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: qrt --version | status | hook claude-code | hook install | hook status | vault doctor | vault backfill | vault archive <path> [--kind K] | vault backup [--verify] <dest> | daemon run-once | dogfood import <transcript_path> | dogfood latest | dogfood list | dogfood show <session_id> | normalize <transcript> | redact <session> | evidence <redacted-session> | review <evidence> | report <session> | export <session> --profile adp-strict | sessions list [--repo <repo_id>] | ui sessions --json | ui session <session_id> --json | ui review <session_id> --json")
}

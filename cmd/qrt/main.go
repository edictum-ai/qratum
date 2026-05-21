package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
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
	default:
		fmt.Fprintf(stderr, "error: unsupported command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func status(stdout io.Writer, stderr io.Writer) int {
	state, err := qratumDirState(".qratum")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum status")
	fmt.Fprintln(stdout, "milestone: A")
	fmt.Fprintf(stdout, "version: %s\n", version)
	fmt.Fprintf(stdout, "qratum_dir: %s\n", state)
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
	fmt.Fprintln(w, "usage: qrt --version | status")
}

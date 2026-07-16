package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	claudecfg "github.com/acartag7/qratum/internal/claude"
	"github.com/acartag7/qratum/internal/textdiff"
)

func hookInstall(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "error: hook install does not accept arguments")
		printUsage(stderr)
		return 2
	}

	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	status, err := claudecfg.HookStatusForProject(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if status.ProjectLocalInstalled {
		fmt.Fprintf(stderr, "error: project-local SessionEnd hook already installed at %s; remove it before installing the global hook to avoid double capture\n", displayPath(projectRoot, status.ProjectLocalSettingsPath))
		return 1
	}
	if status.GlobalInstalled {
		fmt.Fprintln(stdout, "qratum hook install")
		fmt.Fprintf(stdout, "global_settings: %s\n", filepath.ToSlash(status.GlobalSettingsPath))
		fmt.Fprintln(stdout, "installed: true")
		fmt.Fprintln(stdout, "changed: false")
		return 0
	}

	doc, before, err := claudecfg.LoadSettings(status.GlobalSettingsPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	changed, err := claudecfg.EnsureSessionEndHook(doc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if !changed {
		fmt.Fprintln(stdout, "qratum hook install")
		fmt.Fprintf(stdout, "global_settings: %s\n", filepath.ToSlash(status.GlobalSettingsPath))
		fmt.Fprintln(stdout, "installed: true")
		fmt.Fprintln(stdout, "changed: false")
		return 0
	}

	after, err := claudecfg.EncodeSettings(doc)
	if err != nil {
		fmt.Fprintf(stderr, "error: encode updated settings: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum hook install")
	fmt.Fprintf(stdout, "global_settings: %s\n", filepath.ToSlash(status.GlobalSettingsPath))
	fmt.Fprintln(stdout, "diff:")
	fmt.Fprint(stdout, textdiff.Unified(filepath.ToSlash(status.GlobalSettingsPath), ensureTrailingNewline(before), filepath.ToSlash(status.GlobalSettingsPath), after))
	if !confirmApply(stdin, stdout) {
		fmt.Fprintln(stderr, "error: hook install aborted")
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(status.GlobalSettingsPath), 0o755); err != nil {
		fmt.Fprintf(stderr, "error: create Claude settings directory: %v\n", err)
		return 1
	}
	if err := writeFileAtomic(status.GlobalSettingsPath, after, 0o644); err != nil {
		fmt.Fprintf(stderr, "error: write global Claude settings: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "installed: true")
	fmt.Fprintln(stdout, "changed: true")
	return 0
}

func hookStatus(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "error: hook status does not accept arguments")
		printUsage(stderr)
		return 2
	}
	projectRoot, err := currentProjectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	status, err := claudecfg.HookStatusForProject(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "qratum hook status")
	fmt.Fprintf(stdout, "global_settings: %s\n", filepath.ToSlash(status.GlobalSettingsPath))
	fmt.Fprintf(stdout, "installed: %s\n", yesNo(status.GlobalInstalled))
	if status.ProjectLocalInstalled {
		fmt.Fprintf(stdout, "project_local_hook: %s\n", displayPath(projectRoot, status.ProjectLocalSettingsPath))
	} else {
		fmt.Fprintln(stdout, "project_local_hook: none")
	}
	return 0
}

func confirmApply(stdin io.Reader, stdout io.Writer) bool {
	fmt.Fprint(stdout, "Apply changes? [y/N]: ")
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func ensureTrailingNewline(data []byte) []byte {
	if len(data) == 0 {
		return []byte{}
	}
	if data[len(data)-1] == '\n' {
		return data
	}
	return append(append([]byte(nil), data...), '\n')
}

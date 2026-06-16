package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/edictum-ai/qratum/internal/workspace"
)

type sessionListEntry struct {
	SessionID string
	Path      string
}

func sessions(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		fmt.Fprintln(stderr, "error: missing sessions command")
		return 2
	}
	if args[0] != "list" {
		fmt.Fprintf(stderr, "error: unsupported sessions command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: sessions list does not accept arguments")
		printUsage(stderr)
		return 2
	}

	entries, err := listSessions()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	for _, entry := range entries {
		fmt.Fprintf(stdout, "%s\t%s\n", entry.SessionID, entry.Path)
	}
	return 0
}

func listSessions() ([]sessionListEntry, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current project: %w", err)
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve current project absolute path: %w", err)
	}

	qratumHome, err := workspace.Resolve()
	if err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(qratumHome.Root, "sessions")
	info, err := os.Stat(sessionsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("sessions directory %s does not exist", displayPath(projectRoot, sessionsDir))
		}
		return nil, fmt.Errorf("inspect sessions directory %s: %w", displayPath(projectRoot, sessionsDir), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("sessions path %s is not a directory", displayPath(projectRoot, sessionsDir))
	}

	paths, err := filepath.Glob(filepath.Join(sessionsDir, "*", "normalized.json"))
	if err != nil {
		return nil, fmt.Errorf("list sessions directory %s: %w", displayPath(projectRoot, sessionsDir), err)
	}
	sort.Strings(paths)

	entries := make([]sessionListEntry, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read session %s: %w", displayPath(projectRoot, path), err)
		}
		var session struct {
			SchemaVersion string `json:"schema_version"`
			SessionID     string `json:"session_id"`
		}
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("invalid session JSON %s: %w", displayPath(projectRoot, path), err)
		}
		if session.SchemaVersion != qratumSessionSchemaVersion {
			return nil, fmt.Errorf("session %s has unsupported schema_version %q", displayPath(projectRoot, path), session.SchemaVersion)
		}
		if session.SessionID == "" {
			return nil, fmt.Errorf("session %s is missing session_id", displayPath(projectRoot, path))
		}
		entries = append(entries, sessionListEntry{
			SessionID: session.SessionID,
			Path:      displayPath(projectRoot, path),
		})
	}
	return entries, nil
}

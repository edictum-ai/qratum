// Package claude handles local Claude Code filesystem conventions.
package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SessionEndCommand is the global Claude Code hook command Qratum installs.
const SessionEndCommand = "qrt hook claude-code"

// TranscriptFile is a Claude transcript discovered during backfill.
type TranscriptFile struct {
	Path string
	Kind string
}

// HookStatus describes whether the global and project-local hooks are present.
type HookStatus struct {
	GlobalSettingsPath       string
	GlobalInstalled          bool
	ProjectLocalSettingsPath string
	ProjectLocalInstalled    bool
}

// Root returns the Claude home directory under the current user home.
func Root() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// GlobalSettingsPath returns the global Claude settings file path.
func GlobalSettingsPath() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "settings.json"), nil
}

// ProjectsDir returns the Claude transcript projects directory.
func ProjectsDir() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "projects"), nil
}

// LoadSettings reads a Claude settings JSON file or returns an empty document.
func LoadSettings(path string) (map[string]any, []byte, error) {
	// #nosec G304 -- settings paths are resolved from the local Claude home or local project fixtures.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil, nil
		}
		return nil, nil, fmt.Errorf("read settings %s: %w", filepath.ToSlash(path), err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("decode settings %s: %w", filepath.ToSlash(path), err)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return doc, data, nil
}

// EncodeSettings formats a Claude settings document with stable indentation.
func EncodeSettings(doc map[string]any) ([]byte, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// EnsureSessionEndHook adds the Qratum SessionEnd hook if it is missing.
func EnsureSessionEndHook(doc map[string]any) (bool, error) {
	if doc == nil {
		return false, fmt.Errorf("settings document is nil")
	}
	hooksAny, ok := doc["hooks"]
	if !ok || hooksAny == nil {
		hooksAny = map[string]any{}
		doc["hooks"] = hooksAny
	}
	hooks, ok := hooksAny.(map[string]any)
	if !ok {
		return false, fmt.Errorf("settings field hooks must be an object")
	}
	entriesAny, ok := hooks["SessionEnd"]
	if !ok || entriesAny == nil {
		entriesAny = []any{}
	}
	entries, ok := entriesAny.([]any)
	if !ok {
		return false, fmt.Errorf("settings field hooks.SessionEnd must be an array")
	}
	if hasSessionEndHookEntries(entries) {
		return false, nil
	}
	entries = append(entries, sessionEndEntry())
	hooks["SessionEnd"] = entries
	return true, nil
}

// HasSessionEndHook reports whether the Qratum SessionEnd hook is installed.
func HasSessionEndHook(doc map[string]any) bool {
	hooksAny, ok := doc["hooks"]
	if !ok || hooksAny == nil {
		return false
	}
	hooks, ok := hooksAny.(map[string]any)
	if !ok {
		return false
	}
	entriesAny, ok := hooks["SessionEnd"]
	if !ok || entriesAny == nil {
		return false
	}
	entries, ok := entriesAny.([]any)
	if !ok {
		return false
	}
	return hasSessionEndHookEntries(entries)
}

// HookStatusForProject inspects the global settings and current project settings.
func HookStatusForProject(projectRoot string) (HookStatus, error) {
	globalPath, err := GlobalSettingsPath()
	if err != nil {
		return HookStatus{}, err
	}
	globalDoc, _, err := LoadSettings(globalPath)
	if err != nil {
		return HookStatus{}, err
	}

	status := HookStatus{
		GlobalSettingsPath: globalPath,
		GlobalInstalled:    HasSessionEndHook(globalDoc),
	}
	for _, candidate := range []string{
		filepath.Join(projectRoot, ".claude", "settings.local.json"),
		filepath.Join(projectRoot, ".claude", "settings.json"),
	} {
		doc, _, err := LoadSettings(candidate)
		if err != nil {
			return HookStatus{}, err
		}
		if HasSessionEndHook(doc) {
			status.ProjectLocalSettingsPath = candidate
			status.ProjectLocalInstalled = true
			break
		}
	}
	return status, nil
}

// ListTranscriptFiles inventories Claude transcript files under ~/.claude/projects.
func ListTranscriptFiles() ([]TranscriptFile, error) {
	projectsDir, err := ProjectsDir()
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(projectsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect Claude projects directory %s: %w", filepath.ToSlash(projectsDir), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("claude projects path %s is not a directory", filepath.ToSlash(projectsDir))
	}

	files := make([]TranscriptFile, 0, 32)
	err = filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "memory" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		kind := "main_transcript"
		if strings.Contains(filepath.ToSlash(path), "/subagents/") {
			kind = "subagent_transcript"
		}
		files = append(files, TranscriptFile{Path: filepath.ToSlash(path), Kind: kind})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk Claude projects directory %s: %w", filepath.ToSlash(projectsDir), err)
	}
	sort.Slice(files, func(i int, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func hasSessionEndHookEntries(entries []any) bool {
	for _, entryAny := range entries {
		entry, ok := entryAny.(map[string]any)
		if !ok {
			continue
		}
		hooksAny, ok := entry["hooks"]
		if !ok || hooksAny == nil {
			continue
		}
		hooks, ok := hooksAny.([]any)
		if !ok {
			continue
		}
		for _, hookAny := range hooks {
			hook, ok := hookAny.(map[string]any)
			if !ok {
				continue
			}
			command, _ := hook["command"].(string)
			if strings.TrimSpace(command) == SessionEndCommand {
				return true
			}
		}
	}
	return false
}

func sessionEndEntry() map[string]any {
	return map[string]any{
		"hooks": []map[string]any{
			{
				"type":    "command",
				"command": SessionEndCommand,
			},
		},
	}
}

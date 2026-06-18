// Package schedule generates and installs the passive backfill OS timer.
package schedule

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// DirEnv overrides the real launchd/systemd user schedule directory.
	DirEnv = "QRATUM_SCHEDULE_DIR"
	// Label is the stable OS timer label/unit stem.
	Label = "dev.qratum.backfill"
	// BackfillIntervalSeconds is the stable one-shot cadence.
	BackfillIntervalSeconds = 6 * 60 * 60
)

// Options controls schedule generation and installation.
type Options struct {
	BinaryPath  string
	ScheduleDir string
	Platform    string
}

// File is one schedule file to write.
type File struct {
	Path    string
	Content []byte
}

// Plan is the generated schedule plan.
type Plan struct {
	Platform string
	Command  []string
	Files    []File
}

// InstallResult reports the result of a fake-safe or real install.
type InstallResult struct {
	Plan    Plan
	Changed bool
}

// UninstallResult reports schedule removal.
type UninstallResult struct {
	Paths   []string
	Removed int
}

// BuildPlan returns the timer files for the current or requested platform.
func BuildPlan(options Options) (Plan, error) {
	platform := strings.TrimSpace(options.Platform)
	if platform == "" {
		platform = runtime.GOOS
	}
	binaryPath := strings.TrimSpace(options.BinaryPath)
	if binaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return Plan{}, fmt.Errorf("resolve qrt executable: %w", err)
		}
		binaryPath = exe
	}
	absBinary, err := filepath.Abs(binaryPath)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve qrt executable %q: %w", binaryPath, err)
	}
	scheduleDir, err := ResolveDir(platform, options.ScheduleDir)
	if err != nil {
		return Plan{}, err
	}
	command := []string{filepath.ToSlash(absBinary), "vault", "backfill"}
	switch platform {
	case "darwin":
		path := filepath.Join(scheduleDir, Label+".plist")
		return Plan{Platform: platform, Command: command, Files: []File{{Path: path, Content: launchdPlist(command)}}}, nil
	case "linux":
		servicePath := filepath.Join(scheduleDir, Label+".service")
		timerPath := filepath.Join(scheduleDir, Label+".timer")
		return Plan{Platform: platform, Command: command, Files: []File{
			{Path: servicePath, Content: systemdService(command)},
			{Path: timerPath, Content: systemdTimer()},
		}}, nil
	default:
		return Plan{}, fmt.Errorf("unsupported schedule platform %q", platform)
	}
}

// ResolveDir returns the schedule directory, honoring QRATUM_SCHEDULE_DIR.
func ResolveDir(platform string, override string) (string, error) {
	dir := strings.TrimSpace(override)
	if dir == "" {
		dir = strings.TrimSpace(os.Getenv(DirEnv))
	}
	if dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("resolve schedule dir %q: %w", dir, err)
		}
		return abs, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	switch platform {
	case "darwin":
		return filepath.Join(home, "Library", "LaunchAgents"), nil
	case "linux":
		if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
			return filepath.Join(xdg, "systemd", "user"), nil
		}
		return filepath.Join(home, ".config", "systemd", "user"), nil
	default:
		return "", fmt.Errorf("unsupported schedule platform %q", platform)
	}
}

// Install writes the generated schedule files, preserving byte-identical files.
func Install(options Options) (InstallResult, error) {
	plan, err := BuildPlan(options)
	if err != nil {
		return InstallResult{}, err
	}
	changed := false
	for _, file := range plan.Files {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0o700); err != nil {
			return InstallResult{}, fmt.Errorf("create schedule dir %s: %w", filepath.ToSlash(filepath.Dir(file.Path)), err)
		}
		existing, err := os.ReadFile(file.Path)
		if err == nil && bytes.Equal(existing, file.Content) {
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return InstallResult{}, fmt.Errorf("read schedule file %s: %w", filepath.ToSlash(file.Path), err)
		}
		if err := writeFileAtomic(file.Path, file.Content, 0o600); err != nil {
			return InstallResult{}, fmt.Errorf("write schedule file %s: %w", filepath.ToSlash(file.Path), err)
		}
		changed = true
	}
	return InstallResult{Plan: plan, Changed: changed}, nil
}

// Uninstall removes only the generated schedule files.
func Uninstall(options Options) (UninstallResult, error) {
	plan, err := BuildPlan(options)
	if err != nil {
		return UninstallResult{}, err
	}
	result := UninstallResult{Paths: make([]string, 0, len(plan.Files))}
	for _, file := range plan.Files {
		result.Paths = append(result.Paths, file.Path)
		if err := os.Remove(file.Path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return UninstallResult{}, fmt.Errorf("remove schedule file %s: %w", filepath.ToSlash(file.Path), err)
		}
		result.Removed++
	}
	return result, nil
}

// IsInstalled reports whether every generated schedule file exists.
func IsInstalled(options Options) (bool, error) {
	plan, err := BuildPlan(options)
	if err != nil {
		return false, err
	}
	for _, file := range plan.Files {
		info, err := os.Stat(file.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}
			return false, fmt.Errorf("inspect schedule file %s: %w", filepath.ToSlash(file.Path), err)
		}
		if info.IsDir() {
			return false, nil
		}
	}
	return true, nil
}

func launchdPlist(command []string) []byte {
	var out bytes.Buffer
	out.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	out.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	out.WriteString(`<plist version="1.0">` + "\n")
	out.WriteString("<dict>\n")
	out.WriteString("  <key>Label</key>\n")
	out.WriteString("  <string>" + xmlEscaped(Label) + "</string>\n")
	out.WriteString("  <key>ProgramArguments</key>\n")
	out.WriteString("  <array>\n")
	for _, arg := range command {
		out.WriteString("    <string>" + xmlEscaped(arg) + "</string>\n")
	}
	out.WriteString("  </array>\n")
	out.WriteString("  <key>StartInterval</key>\n")
	fmt.Fprintf(&out, "  <integer>%d</integer>\n", BackfillIntervalSeconds)
	out.WriteString("</dict>\n")
	out.WriteString("</plist>\n")
	return out.Bytes()
}

func systemdService(command []string) []byte {
	return []byte(fmt.Sprintf(`[Unit]
Description=Qratum vault backfill

[Service]
Type=oneshot
ExecStart=%s
`, strings.Join(command, " ")))
}

func systemdTimer() []byte {
	return []byte(`[Unit]
Description=Run Qratum vault backfill every 6 hours

[Timer]
OnUnitActiveSec=6h
Persistent=true

[Install]
WantedBy=timers.target
`)
}

func xmlEscaped(value string) string {
	var out bytes.Buffer
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
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

package schedule

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlanDarwinInstallsBackfillCommand(t *testing.T) {
	plan, err := BuildPlan(Options{BinaryPath: "/tmp/qrt", ScheduleDir: t.TempDir(), Platform: "darwin"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(plan.Command[1:], " "), "vault backfill"; got != want {
		t.Fatalf("command tail = %q, want %q", got, want)
	}
	if len(plan.Files) != 1 || !strings.HasSuffix(plan.Files[0].Path, Label+".plist") {
		t.Fatalf("darwin files = %#v", plan.Files)
	}
	content := string(plan.Files[0].Content)
	for _, want := range []string{
		"<key>ProgramArguments</key>",
		"<string>/tmp/qrt</string>",
		"<string>vault</string>",
		"<string>backfill</string>",
		"<key>StartInterval</key>",
		"<integer>21600</integer>",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("plist missing %q:\n%s", want, content)
		}
	}
	args, interval := parseLaunchdPlist(t, plan.Files[0].Content)
	if got, want := strings.Join(args, " "), "/tmp/qrt vault backfill"; got != want {
		t.Fatalf("launchd ProgramArguments = %q, want %q", got, want)
	}
	if interval != "21600" {
		t.Fatalf("launchd StartInterval = %q, want 21600", interval)
	}
}

func TestBuildPlanLinuxInstallsBackfillCommand(t *testing.T) {
	plan, err := BuildPlan(Options{BinaryPath: "/tmp/qrt", ScheduleDir: t.TempDir(), Platform: "linux"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(plan.Command[1:], " "), "vault backfill"; got != want {
		t.Fatalf("command tail = %q, want %q", got, want)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("linux files = %#v, want service+timer", plan.Files)
	}
	content := string(plan.Files[0].Content) + string(plan.Files[1].Content)
	for _, want := range []string{"ExecStart=/tmp/qrt vault backfill", "OnUnitActiveSec=6h"} {
		if !strings.Contains(content, want) {
			t.Fatalf("systemd content missing %q:\n%s", want, content)
		}
	}
	service := parseSystemdUnit(plan.Files[0].Content)
	timer := parseSystemdUnit(plan.Files[1].Content)
	if got, want := service["ExecStart"], "/tmp/qrt vault backfill"; got != want {
		t.Fatalf("systemd ExecStart = %q, want %q", got, want)
	}
	if got, want := timer["OnUnitActiveSec"], "6h"; got != want {
		t.Fatalf("systemd OnUnitActiveSec = %q, want %q", got, want)
	}
}

func TestInstallIsIdempotentAndUninstallIsClean(t *testing.T) {
	dir := t.TempDir()
	options := Options{BinaryPath: "/tmp/qrt", ScheduleDir: dir, Platform: "darwin"}
	first, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("first install changed=false, want true")
	}
	path := filepath.Join(dir, Label+".plist")
	// #nosec G304 -- the test reads the schedule file it just installed in t.TempDir.
	firstBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Fatal("second install changed=true, want byte-identical no-op")
	}
	// #nosec G304 -- the test reads the schedule file it just installed in t.TempDir.
	secondBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("schedule changed across reinstall")
	}
	installed, err := IsInstalled(options)
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Fatal("IsInstalled=false after install")
	}

	removed, err := Uninstall(options)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Removed != 1 {
		t.Fatalf("removed = %d, want 1", removed.Removed)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("schedule dir entries after uninstall = %v, want empty", entries)
	}
	removedAgain, err := Uninstall(options)
	if err != nil {
		t.Fatal(err)
	}
	if removedAgain.Removed != 0 {
		t.Fatalf("second removed = %d, want clean no-op", removedAgain.Removed)
	}
}

func parseLaunchdPlist(t *testing.T, content []byte) ([]string, string) {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var args []string
	var interval string
	var lastKey string
	inProgramArguments := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatal(err)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "key":
				var value string
				if err := decoder.DecodeElement(&value, &tok); err != nil {
					t.Fatal(err)
				}
				lastKey = value
			case "array":
				inProgramArguments = lastKey == "ProgramArguments"
			case "string":
				var value string
				if err := decoder.DecodeElement(&value, &tok); err != nil {
					t.Fatal(err)
				}
				if inProgramArguments {
					args = append(args, value)
				}
			case "integer":
				var value string
				if err := decoder.DecodeElement(&value, &tok); err != nil {
					t.Fatal(err)
				}
				if lastKey == "StartInterval" {
					interval = value
				}
			}
		case xml.EndElement:
			if tok.Name.Local == "array" && inProgramArguments {
				inProgramArguments = false
			}
		}
	}
	return args, interval
}

func parseSystemdUnit(content []byte) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	return values
}

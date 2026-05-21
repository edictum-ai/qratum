package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestVersionPrintsFixedDevelopmentVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "qrt dev\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStatusWorksWithoutQratumDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"qratum status\n",
		"milestone: A\n",
		"version: dev\n",
		"qratum_dir: missing\n",
		"ready: true\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, missing %q", output, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStatusReportsPresentQratumDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir(".qratum", 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "qratum_dir: present\n") {
		t.Fatalf("stdout = %q, want qratum_dir present", stdout.String())
	}
}

func TestStatusFailsWhenQratumPathIsInvalid(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".qratum", []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"status"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: .qratum exists but is not a directory") {
		t.Fatalf("stderr = %q, want invalid .qratum error", stderr.String())
	}
}

func TestMissingCommandFailsVisibly(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(nil, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"usage: qrt", "error: missing command"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestUnsupportedCommandFailsVisibly(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"hook", "claude-code"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `error: unsupported command "hook"`) {
		t.Fatalf("stderr = %q, want unsupported command error", stderr.String())
	}
}

func TestVersionRejectsExtraArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--version", "extra"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: --version does not accept arguments") {
		t.Fatalf("stderr = %q, want extra argument error", stderr.String())
	}
}

func TestStatusRejectsExtraArguments(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	code := run([]string{"status", "extra"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error: status does not accept arguments") {
		t.Fatalf("stderr = %q, want extra argument error", stderr.String())
	}
}

package main

import (
	"bytes"
	"encoding/json"
	htmlpkg "html"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportWritesStaticHTMLFromArtifacts(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	qratumHome := setTestQratumHome(t)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"report", "sessions/ses_0001/normalized.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	reportPath := qratumSessionArtifact(qratumHome, "ses_0001", "report.html")
	if got, want := stdout.String(), "wrote "+filepath.ToSlash(reportPath)+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	report := readTextFile(t, reportPath)
	for _, want := range []string{
		"<!doctype html>",
		"<h2>Session summary</h2>",
		"<h2>Review card</h2>",
		"<h2>Evidence findings</h2>",
		"<h2>Missing evidence</h2>",
		"<h2>Redaction summary</h2>",
		"<h2>Artifacts</h2>",
		"<h2>Provenance digests</h2>",
		"needs_attention",
		"successful verification command after 2026-05-21T21:55:00Z",
		"sha256:",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
	assertNoUnsafeReportHTML(t, report)
	assertNoRawReportInternals(t, report)
}

func TestReportEscapesHTMLAndKeepsArtifactHrefsLocal(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	qratumHome := setTestQratumHome(t)

	evidencePath := qratumSessionArtifact(qratumHome, "ses_0001", "evidence.json")
	var bundle evidenceBundle
	readJSONFile(t, evidencePath, &bundle)
	maliciousScript := `<script>alert("x")</script>`
	maliciousImage := `<img src=x onerror=alert(1)>`
	bundle.Findings[0].Title = maliciousScript
	bundle.Findings[0].Summary = maliciousImage
	bundle.Findings[0].Evidence[0].Path = maliciousImage
	writeJSONFile(t, evidencePath, bundle)

	reviewPath := qratumSessionArtifact(qratumHome, "ses_0001", "review.json")
	var card reviewCard
	readJSONFile(t, reviewPath, &card)
	card.MainFinding = maliciousScript
	card.Evidence = append(card.Evidence, maliciousImage)
	writeJSONFile(t, reviewPath, card)

	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	code := run([]string{"report", "sessions/ses_0001/normalized.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	report := readTextFile(t, qratumSessionArtifact(qratumHome, "ses_0001", "report.html"))
	assertNoUnsafeReportHTML(t, report)
	for _, want := range []string{
		htmlpkg.EscapeString(maliciousScript),
		htmlpkg.EscapeString(maliciousImage),
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing escaped value %q:\n%s", want, report)
		}
	}
}

func TestReportDoesNotLeakFixtureSecrets(t *testing.T) {
	root := t.TempDir()
	seedSecretReportArtifacts(t, root)
	qratumHome := setTestQratumHome(t)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"report", "sessions/ses_0001/normalized.json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	report := readTextFile(t, qratumSessionArtifact(qratumHome, "ses_0001", "report.html"))
	for _, raw := range []string{
		"qratumSECRETtoken1234567890abcdef",
		"supersecret",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0In0.signature",
		"/Users/acartagena/project/qratum/.env",
		"-----BEGIN PRIVATE KEY-----",
		"MIIEvQIBADANBgkqhkiG9w0BAQEFAASC",
		"fA9sD8f7Gh6Jk5Lm4Np3Qr2St1Uv0WxY",
	} {
		if strings.Contains(report, raw) {
			t.Fatalf("report leaked raw secret/path %q:\n%s", raw, report)
		}
	}
	for _, want := range []string{
		"[REDACTED_SECRET_001]",
		"[REDACTED_PATH_001]",
		"Redaction summary",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
	assertNoUnsafeReportHTML(t, report)
}

func TestReportRejectsMissingAndInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(t *testing.T)
		wantCode  int
		wantError string
	}{
		{
			name:      "missing session argument",
			args:      []string{"report"},
			wantCode:  2,
			wantError: "error: missing session path",
		},
		{
			name:      "extra argument",
			args:      []string{"report", "one.json", "two.json"},
			wantCode:  2,
			wantError: "error: report accepts exactly one session path",
		},
		{
			name:      "missing file",
			args:      []string{"report", "missing.json"},
			wantCode:  1,
			wantError: "missing session missing.json",
		},
		{
			name: "missing redacted artifact",
			args: []string{"report", "sessions/ses_0001/normalized.json"},
			setup: func(t *testing.T) {
				seedUIFixtureArtifacts(t, ".")
				if err := os.Remove(qratumSessionArtifact(setTestQratumHome(t), "ses_0001", "redacted.json")); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:  1,
			wantError: "missing redacted session",
		},
		{
			name: "unsupported review verdict",
			args: []string{"report", "sessions/ses_0001/normalized.json"},
			setup: func(t *testing.T) {
				seedUIFixtureArtifacts(t, ".")
				reviewPath := qratumSessionArtifact(setTestQratumHome(t), "ses_0001", "review.json")
				var card reviewCard
				readJSONFile(t, reviewPath, &card)
				card.Verdict = "future_score"
				writeJSONFile(t, reviewPath, card)
			},
			wantCode:  1,
			wantError: `unsupported verdict "future_score"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setTestQratumHome(t)
			if tt.setup != nil {
				tt.setup(t)
			}
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr = %q", code, tt.wantCode, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantError) {
				t.Fatalf("stderr = %q, missing %q", stderr.String(), tt.wantError)
			}
		})
	}
}

func seedSecretReportArtifacts(t *testing.T, root string) {
	t.Helper()
	setTestQratumHome(t)
	var session qratumSession
	if err := json.Unmarshal(readRedactionFixture(t, "secret-session.input.json"), &session); err != nil {
		t.Fatalf("decode secret session fixture: %v", err)
	}
	session.PipelineStatus = "normalized"
	paths := artifactPathsForStem(session.SessionID)
	session.ArtifactPaths = &paths

	redacted, err := redactQratumSession(session)
	if err != nil {
		t.Fatalf("redact secret report fixture: %v", err)
	}
	evidence, err := buildEvidenceBundle(redacted, paths)
	if err != nil {
		t.Fatalf("build secret report evidence: %v", err)
	}
	review, err := buildReviewCard(evidence)
	if err != nil {
		t.Fatalf("build secret report review: %v", err)
	}

	writeJSONFile(t, artifactAbsolutePath(root, paths.Session), session)
	writeJSONFile(t, artifactAbsolutePath(root, paths.Redacted), redacted)
	writeJSONFile(t, artifactAbsolutePath(root, paths.Evidence), evidence)
	writeJSONFile(t, artifactAbsolutePath(root, paths.Review), review)
}

func assertNoUnsafeReportHTML(t *testing.T, report string) {
	t.Helper()
	lower := strings.ToLower(report)
	for _, banned := range []string{
		"<script",
		"<link",
		"<img",
		"<iframe",
		"<object",
		"<embed",
		`href="javascript:`,
		`href="http://`,
		`href="https://`,
		`href="//`,
	} {
		if strings.Contains(lower, banned) {
			t.Fatalf("report contains unsafe HTML %q:\n%s", banned, report)
		}
	}
}

func assertNoRawReportInternals(t *testing.T, report string) {
	t.Helper()
	for _, banned := range []string{
		"Implement deterministic redaction for obvious secrets.",
		`"turns": [`,
		`"tool_calls": [`,
		`"commands": [`,
		`"provenance"`,
		`"secret_map"`,
	} {
		if strings.Contains(report, banned) {
			t.Fatalf("report leaks raw/internal field %q:\n%s", banned, report)
		}
	}
}

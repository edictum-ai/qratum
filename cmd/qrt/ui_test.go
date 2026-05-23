package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUISessionsJSONEmitsListDTOFixture(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"ui", "sessions", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertJSONEqual(t, stdout.Bytes(), readUIFixture(t, "sessions.golden.json"))
	assertNoRawUIInternals(t, stdout.String())

	var items []uiSessionListItem
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		t.Fatalf("decode UI sessions JSON: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("session items = %d, want 1", len(items))
	}
	if got, want := items[0].SchemaVersion, qratumUISessionListItemSchemaVersion; got != want {
		t.Fatalf("schema_version = %q, want %q", got, want)
	}
	if got, want := items[0].SessionID, "ses_0001"; got != want {
		t.Fatalf("session_id = %q, want %q", got, want)
	}
	if len(items[0].Artifacts) != 5 {
		t.Fatalf("artifacts = %d, want 5", len(items[0].Artifacts))
	}
}

func TestUISessionJSONEmitsDetailDTOFixture(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"ui", "session", "ses_0001", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertJSONEqual(t, stdout.Bytes(), readUIFixture(t, "session-detail.golden.json"))
	assertNoRawUIInternals(t, stdout.String())

	var detail uiSessionDetail
	if err := json.Unmarshal(stdout.Bytes(), &detail); err != nil {
		t.Fatalf("decode UI session detail JSON: %v", err)
	}
	if got, want := detail.SchemaVersion, qratumUISessionDetailSchemaVersion; got != want {
		t.Fatalf("schema_version = %q, want %q", got, want)
	}
	if got, want := len(detail.Findings), 5; got != want {
		t.Fatalf("findings = %d, want %d", got, want)
	}
	if got, want := detail.Findings[0].Severity, "high"; got != want {
		t.Fatalf("first finding severity = %q, want %q", got, want)
	}
}

func TestBuildUISessionDetailUsesRedactedSourceMetadata(t *testing.T) {
	detail := buildUISessionDetail(uiSessionContext{
		session: qratumSession{
			SessionID:                 "ses_redaction_surface",
			SourceEventID:             "raw-event-id",
			SourceEventType:           "raw-event-type",
			SourceEventTimestamp:      "2026-05-21T10:00:00Z",
			SourceTranscriptSessionID: "raw-transcript-session-id",
			Turns:                     []qratumTurn{{Role: "assistant", Content: "ok"}},
			BusinessMetrics:           qratumBusinessMetrics{DurationSeconds: 12},
		},
		redacted: qratumSession{
			SourceEventID:             "redacted-event-id",
			SourceEventType:           "redacted-event-type",
			SourceEventTimestamp:      "2026-05-21T11:00:00Z",
			SourceTranscriptSessionID: "[REDACTED_SOURCE_TRANSCRIPT_SESSION_ID_001]",
		},
		evidence: evidenceBundle{
			Summary: evidenceBundleSummary{Status: evidenceStatusComplete},
		},
	})

	if got, want := detail.Time.SourceEventTimestamp, "2026-05-21T11:00:00Z"; got != want {
		t.Fatalf("time source_event_timestamp = %q, want %q", got, want)
	}
	if got, want := detail.Summary.SourceEventID, "redacted-event-id"; got != want {
		t.Fatalf("summary source_event_id = %q, want %q", got, want)
	}
	if got, want := detail.Summary.SourceEventType, "redacted-event-type"; got != want {
		t.Fatalf("summary source_event_type = %q, want %q", got, want)
	}
	if got, want := detail.Summary.SourceEventTimestamp, "2026-05-21T11:00:00Z"; got != want {
		t.Fatalf("summary source_event_timestamp = %q, want %q", got, want)
	}
	if got, want := detail.Summary.SourceTranscriptSessionID, "[REDACTED_SOURCE_TRANSCRIPT_SESSION_ID_001]"; got != want {
		t.Fatalf("summary source_transcript_session_id = %q, want %q", got, want)
	}
}

func TestUIReviewJSONEmitsReviewDTOFixture(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"ui", "review", "ses_0001", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	assertJSONEqual(t, stdout.Bytes(), readUIFixture(t, "review.golden.json"))
	assertNoRawUIInternals(t, stdout.String())

	var card uiReviewCardDTO
	if err := json.Unmarshal(stdout.Bytes(), &card); err != nil {
		t.Fatalf("decode UI review JSON: %v", err)
	}
	if got, want := card.SchemaVersion, qratumUIReviewCardSchemaVersion; got != want {
		t.Fatalf("schema_version = %q, want %q", got, want)
	}
	if len(card.Findings) == 0 {
		t.Fatal("review findings are empty")
	}
}

func TestUIRejectsMissingArtifact(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	if err := os.Remove(filepath.Join(root, ".qratum", "evidence", "ses_0001.evidence.json")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"ui", "session", "ses_0001", "--json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		"missing evidence .qratum/evidence/ses_0001.evidence.json",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestUIRejectsMissingSession(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"ui", "session", "missing-session", "--json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`session \"missing-session\" not found in .qratum/sessions`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestUIRejectsUnsupportedEvidenceFindingType(t *testing.T) {
	root := t.TempDir()
	seedUIFixtureArtifacts(t, root)
	evidencePath := filepath.Join(root, ".qratum", "evidence", "ses_0001.evidence.json")
	var bundle evidenceBundle
	readJSONFile(t, evidencePath, &bundle)
	bundle.Findings[0].Type = "tool_risk.future"
	writeJSONFile(t, evidencePath, bundle)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer

	code := run([]string{"ui", "review", "ses_0001", "--json"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{
		`"schema_version": "qratum.ui.api_error.v1"`,
		`unsupported findings[0].type \"tool_risk.future\"`,
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
	}
}

func TestUIRejectsBadArguments(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "missing ui command",
			args:      []string{"ui"},
			wantError: "error: missing ui command",
		},
		{
			name:      "unsupported ui command",
			args:      []string{"ui", "trends", "--json"},
			wantError: `error: unsupported ui command "trends"`,
		},
		{
			name:      "sessions missing json",
			args:      []string{"ui", "sessions"},
			wantError: "error: ui sessions requires --json",
		},
		{
			name:      "session missing id",
			args:      []string{"ui", "session", "--json"},
			wantError: "error: ui session accepts <session_id> --json",
		},
		{
			name:      "review missing json",
			args:      []string{"ui", "review", "ses_0001"},
			wantError: "error: ui review accepts <session_id> --json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr)

			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
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

func seedUIFixtureArtifacts(t *testing.T, root string) {
	t.Helper()
	var session qratumSession
	if err := json.Unmarshal(readEvidenceFixture(t, "verification-gap.input.json"), &session); err != nil {
		t.Fatalf("decode session fixture: %v", err)
	}
	session.PipelineStatus = "normalized"
	paths := artifactPathsForStem(session.SessionID)
	session.ArtifactPaths = &paths

	redacted, err := redactQratumSession(session)
	if err != nil {
		t.Fatalf("redact UI fixture session: %v", err)
	}
	evidence, err := buildEvidenceBundle(redacted, paths)
	if err != nil {
		t.Fatalf("build UI fixture evidence: %v", err)
	}
	review, err := buildReviewCard(evidence)
	if err != nil {
		t.Fatalf("build UI fixture review: %v", err)
	}

	writeJSONFile(t, filepath.Join(root, filepath.FromSlash(paths.Session)), session)
	writeJSONFile(t, filepath.Join(root, filepath.FromSlash(paths.Redacted)), redacted)
	writeJSONFile(t, filepath.Join(root, filepath.FromSlash(paths.Evidence)), evidence)
	writeJSONFile(t, filepath.Join(root, filepath.FromSlash(paths.Review)), review)
	writeBytesFile(t, filepath.Join(root, filepath.FromSlash(paths.Report)), []byte("<!doctype html>\n<html lang=\"en\"><body><h1>Qratum Session ses_0001</h1></body></html>\n"))
	writeBytesFile(t, filepath.Join(root, filepath.FromSlash(paths.Export)), readADPFixture(t, "session.adp-strict.golden.jsonl"))
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := mustJSON(value)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeBytesFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readUIFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "ui", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readADPFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "fixtures", "adp", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertNoRawUIInternals(t *testing.T, output string) {
	t.Helper()
	for _, banned := range []string{
		"Implement deterministic redaction for obvious secrets.",
		`"turns": [`,
		`"tool_calls": [`,
		`"commands": [`,
		`"provenance"`,
		`"secret_map"`,
	} {
		if strings.Contains(output, banned) {
			t.Fatalf("UI output leaks raw/internal field %q: %s", banned, output)
		}
	}
}

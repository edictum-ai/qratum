package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceFlagsFinalVerificationFailed(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "final-verification-failed.input.json")
	requireFindingType(t, bundle, findingFinalVerificationFailed)
	requireNoFindingType(t, bundle, findingMissingFinalVerification)
	requireNoFindingType(t, bundle, findingOnlyFailedVerification)

	finding := pickFindingType(t, bundle, findingFinalVerificationFailed)
	if !strings.Contains(finding.Summary, "2026-05-21T21:40:00Z") {
		t.Fatalf("summary missing failed verification timestamp: %s", finding.Summary)
	}
	if !strings.Contains(finding.Summary, "go test ./...") {
		t.Fatalf("summary missing failed command text: %s", finding.Summary)
	}
}

func TestEvidenceFlagsOnlyFailedVerification(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "only-failed-verification.input.json")
	requireFindingType(t, bundle, findingOnlyFailedVerification)

	finding := pickFindingType(t, bundle, findingOnlyFailedVerification)
	if !strings.Contains(finding.Summary, "2 verification command(s)") {
		t.Fatalf("only-failed summary missing command count: %s", finding.Summary)
	}
	if len(finding.Evidence) != 2 {
		t.Fatalf("only-failed evidence count = %d, want 2", len(finding.Evidence))
	}
	for _, fact := range finding.Evidence {
		if fact.Timestamp == "" {
			t.Fatalf("only-failed evidence fact missing timestamp: %+v", fact)
		}
		if fact.Command == "" {
			t.Fatalf("only-failed evidence fact missing command text: %+v", fact)
		}
	}
}

func TestEvidenceFlagsDestructiveCommand(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "destructive-command.input.json")
	requireFindingType(t, bundle, findingDestructiveCommand)
	requireNoFindingType(t, bundle, findingOnlyFailedVerification)
	requireNoFindingType(t, bundle, findingMissingFinalVerification)

	finding := pickFindingType(t, bundle, findingDestructiveCommand)
	if !strings.Contains(finding.Summary, "rm -rf build") {
		t.Fatalf("destructive summary missing command text: %s", finding.Summary)
	}
	if len(finding.Evidence) == 0 || finding.Evidence[0].Timestamp == "" {
		t.Fatalf("destructive evidence missing timestamp: %+v", finding.Evidence)
	}
}

func TestEvidenceFlagsNetworkCallWithoutNeed(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "network-call-without-need.input.json")
	requireFindingType(t, bundle, findingNetworkCallWithoutNeed)

	finding := pickFindingType(t, bundle, findingNetworkCallWithoutNeed)
	if !strings.Contains(finding.Summary, "curl") {
		t.Fatalf("network summary missing command text: %s", finding.Summary)
	}
}

func TestEvidenceDoesNotFlagNetworkCallWhenManifestChanged(t *testing.T) {
	setTestQratumHome(t)
	var session qratumSession
	if err := json.Unmarshal(readEvidenceFixture(t, "network-call-without-need.input.json"), &session); err != nil {
		t.Fatal(err)
	}
	session.FileChanges = append(session.FileChanges, qratumFileChange{
		Path:      "go.mod",
		Operation: "edit",
		Timestamp: "2026-05-21T21:09:00Z",
	})
	bundle, err := buildEvidenceBundle(session, artifactPathsForStem(session.SessionID))
	if err != nil {
		t.Fatalf("buildEvidenceBundle: %v", err)
	}
	requireNoFindingType(t, bundle, findingNetworkCallWithoutNeed)
}

func TestEvidenceFlagsSourceChangedWithoutTestChange(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "source-without-test.input.json")
	requireFindingType(t, bundle, findingSourceChangedWithoutTest)

	finding := pickFindingType(t, bundle, findingSourceChangedWithoutTest)
	if len(finding.Evidence) == 0 || finding.Evidence[0].Path != "src/handler.go" {
		t.Fatalf("source-without-test evidence wrong: %+v", finding.Evidence)
	}
}

func TestEvidenceDoesNotFlagSourceWithoutTestWhenTestFileChanged(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "destructive-command.input.json")
	requireNoFindingType(t, bundle, findingSourceChangedWithoutTest)
}

func TestEvidenceFlagsBroadFileChangeWithoutVerification(t *testing.T) {
	bundle := buildEvidenceFromFixture(t, "broad-file-change.input.json")
	requireFindingType(t, bundle, findingBroadFileChange)

	finding := pickFindingType(t, bundle, findingBroadFileChange)
	if !strings.Contains(finding.Summary, "8 files") {
		t.Fatalf("broad change summary missing file count: %s", finding.Summary)
	}
}

func buildEvidenceFromFixture(t *testing.T, name string) evidenceBundle {
	t.Helper()
	root := t.TempDir()
	writeEvidenceFixture(t, root, name)
	t.Chdir(root)
	setTestQratumHome(t)
	data, err := os.ReadFile(filepath.Join(root, "fixtures", "evidence", name))
	if err != nil {
		t.Fatal(err)
	}
	var session qratumSession
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	bundle, err := buildEvidenceBundle(session, artifactPathsForStem(session.SessionID))
	if err != nil {
		t.Fatalf("buildEvidenceBundle %s: %v", name, err)
	}
	return bundle
}

func requireFindingType(t *testing.T, bundle evidenceBundle, findingType string) {
	t.Helper()
	for _, f := range bundle.Findings {
		if f.Type == findingType {
			return
		}
	}
	t.Fatalf("missing finding type %q in %v", findingType, findingTypeList(bundle.Findings))
}

func requireNoFindingType(t *testing.T, bundle evidenceBundle, findingType string) {
	t.Helper()
	for _, f := range bundle.Findings {
		if f.Type == findingType {
			t.Fatalf("unexpected finding type %q present in %v", findingType, findingTypeList(bundle.Findings))
		}
	}
}

func pickFindingType(t *testing.T, bundle evidenceBundle, findingType string) evidenceFinding {
	t.Helper()
	for _, f := range bundle.Findings {
		if f.Type == findingType {
			return f
		}
	}
	t.Fatalf("missing finding type %q in %v", findingType, findingTypeList(bundle.Findings))
	return evidenceFinding{}
}

func findingTypeList(findings []evidenceFinding) []string {
	out := make([]string, len(findings))
	for i, f := range findings {
		out[i] = f.Type
	}
	return out
}

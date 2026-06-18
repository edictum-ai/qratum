package trust

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/edictum-ai/qratum/internal/schema"
)

func TestEvaluateEmitsSchemaValidLeakFreeScorecard(t *testing.T) {
	root := repoRoot(t)
	qrt := fakeQRT(t)
	scorecard, ci, err := Evaluate(Options{
		RepoRoot:    root,
		QRTPath:     qrt,
		Now:         time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
		BuildCommit: "testcommit",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ci.Pass {
		t.Fatalf("ci = %#v, want pass", ci)
	}
	if scorecard.Headline != HeadlineTrustedWithGap || scorecard.GapCount != 1 {
		t.Fatalf("headline=%s gap_count=%d", scorecard.Headline, scorecard.GapCount)
	}
	d4 := dimensionByID(scorecard.Dimensions, "D4")
	if d4 == nil {
		t.Fatalf("scorecard missing D4 dimension: %#v", scorecard.Dimensions)
	}
	for _, want := range []string{
		"dogfood latest",
		"evidence sessions/dogfood-session-0001/redacted.json",
		"review sessions/dogfood-session-0001/evidence.json",
		"report sessions/dogfood-session-0001/redacted.json",
		"export sessions/dogfood-session-0001/redacted.json --profile adp-strict",
	} {
		if !dimensionEvidenceContains(*d4, want) {
			t.Fatalf("D4 evidence missing %q: %#v", want, d4.Evidence)
		}
	}
	data, err := Marshal(scorecard)
	if err != nil {
		t.Fatal(err)
	}
	if err := NoLeakCheck(data); err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- test reads a committed schema under the resolved repo root.
	schemaData, err := os.ReadFile(filepath.Join(root, "schemas", "qratum-trust-scorecard.v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(schemaData, data); err != nil {
		t.Fatalf("scorecard rejected by schema: %v\n%s", err, data)
	}
	for _, want := range []string{
		"PII / third-party content is NOT redacted",
		"At-rest disk encryption is out of scope",
		"Cloud / web sessions are uncaptured",
		"`vault backup` of raw is the sanctioned, consent-gated exception",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("scorecard missing residual %q:\n%s", want, data)
		}
	}
}

func TestEvaluateCIThreeStateRules(t *testing.T) {
	base := Scorecard{
		Dimensions: []Dimension{{ID: "D1", State: StateGreen, Summary: "ok"}},
		KnownRed: []KnownRed{{
			ID:       "D10",
			Note:     "gateway gated",
			Owner:    "maintainer",
			Deadline: "2026-12-31",
		}},
		ExtendedRecall: ExtendedRecall{
			ClassCount:            BaselineRecallClasses,
			BaselineClassCount:    BaselineRecallClasses,
			RecallPercent:         100,
			BaselineRecallPercent: 100,
		},
	}
	now := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	if ci := EvaluateCI(base, now); !ci.Pass {
		t.Fatalf("future known-red ci = %#v, want pass", ci)
	}
	blocking := base
	blocking.Dimensions = []Dimension{{ID: "D1", State: StateBlockingRed, Summary: "bad"}}
	if ci := EvaluateCI(blocking, now); ci.Pass || !containsReason(ci.Reasons, "BLOCKING-RED") {
		t.Fatalf("blocking ci = %#v, want fail", ci)
	}
	past := base
	past.KnownRed[0].Deadline = "2026-01-01"
	if ci := EvaluateCI(past, now); ci.Pass || !containsReason(ci.Reasons, "past deadline") {
		t.Fatalf("past ci = %#v, want deadline fail", ci)
	}
	increased := base
	increased.KnownRed = append(increased.KnownRed, KnownRed{ID: "D99", Note: "new", Owner: "maintainer", Deadline: "2026-12-31"})
	if ci := EvaluateCI(increased, now); ci.Pass || !containsReason(ci.Reasons, "count increased") {
		t.Fatalf("increased ci = %#v, want count fail", ci)
	}
	shrunk := base
	shrunk.ExtendedRecall.ClassCount = BaselineRecallClasses - 1
	if ci := EvaluateCI(shrunk, now); ci.Pass || !containsReason(ci.Reasons, "class count shrank") {
		t.Fatalf("shrunk ci = %#v, want corpus fail", ci)
	}
	regressed := base
	regressed.ExtendedRecall.RecallPercent = 99
	if ci := EvaluateCI(regressed, now); ci.Pass || !containsReason(ci.Reasons, "recall regressed") {
		t.Fatalf("regressed ci = %#v, want recall fail", ci)
	}
}

func TestREADMEStatesShippedReality(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "searches every session") {
		t.Fatal("README still claims qratum searches every session")
	}
	for _, want := range []string{"Claude Code sessions", "archive-only", "no redaction path"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing shipped-reality phrase %q", want)
		}
	}
}

func fakeQRT(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "qrt")
	script := `#!/bin/sh
case "$1" in
  status) echo "qratum status"; exit 0 ;;
  vault) echo "qratum vault doctor"; exit 0 ;;
  dogfood) echo "qratum dogfood"; exit 0 ;;
  evidence) echo "qratum evidence"; exit 0 ;;
  review) echo "qratum review"; exit 0 ;;
  report) echo "qratum report"; exit 0 ;;
  export) echo "qratum export"; exit 0 ;;
  --version) echo "qrt test"; exit 0 ;;
esac
echo "unsupported $*" >&2
exit 2
`
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	// #nosec G302 -- fake qrt script must be executable for the trust smoke test.
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, want) {
			return true
		}
	}
	return false
}

func dimensionByID(dimensions []Dimension, id string) *Dimension {
	for i := range dimensions {
		if dimensions[i].ID == id {
			return &dimensions[i]
		}
	}
	return nil
}

func dimensionEvidenceContains(dimension Dimension, want string) bool {
	for _, item := range dimension.Evidence {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

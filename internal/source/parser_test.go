package source

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	qschema "github.com/edictum-ai/qratum/internal/schema"
)

const fixtureEvidenceDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestParseClaudeCodeSupportedResumedTranscript(t *testing.T) {
	result := parseClaudeFixture(t, "main-resumed.jsonl", ParseContext{
		StreamID:               "main",
		EvidenceRevisionDigest: fixtureEvidenceDigest,
	})
	if result.Coverage != CoverageComplete || result.SupportedRecords != 4 || result.UnsupportedRecords != 0 {
		t.Fatalf("coverage/counts = %s/%d/%d", result.Coverage, result.SupportedRecords, result.UnsupportedRecords)
	}
	if result.SourceVersion != SupportedClaudeCodeVersion || result.RootSessionID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("source identity = %q/%q", result.SourceVersion, result.RootSessionID)
	}
	if result.SourceStreamSessionID != result.RootSessionID {
		t.Fatalf("main source stream identity = %q", result.SourceStreamSessionID)
	}
	if len(result.UsageRecords) != 2 {
		t.Fatalf("usage records = %d, want 2", len(result.UsageRecords))
	}
	first := result.UsageRecords[0]
	second := result.UsageRecords[1]
	if first.Model != "claude-sonnet-4-5" || second.Model != "claude-opus-4-6" {
		t.Fatalf("models = %q/%q", first.Model, second.Model)
	}
	if first.Tokens.CacheCreationFiveMin != 10 || first.Tokens.CacheCreationOneHour != 20 || first.Tokens.Total != 200 {
		t.Fatalf("first token classes = %#v", first.Tokens)
	}
	if first.TotalBasis != "derived_sum" {
		t.Fatalf("Claude total basis = %q", first.TotalBasis)
	}
	if second.Tokens.Total != 240 || second.SourceEventID != "req_synthetic_002" {
		t.Fatalf("second usage = %#v", second)
	}
	if first.UsageID == second.UsageID {
		t.Fatal("distinct Claude messages produced the same usage ID")
	}
	validateParsedUsage(t, result.UsageRecords)

	repeated := parseClaudeFixture(t, "main-resumed.jsonl", ParseContext{
		StreamID:               "main",
		EvidenceRevisionDigest: fixtureEvidenceDigest,
	})
	if !reflect.DeepEqual(result, repeated) {
		t.Fatal("repeated Claude parse was not deterministic")
	}
}

func TestParseClaudeCodeChildStreamKeepsStreamIdentity(t *testing.T) {
	main := parseClaudeFixture(t, "main-resumed.jsonl", ParseContext{StreamID: "main", EvidenceRevisionDigest: fixtureEvidenceDigest})
	child := parseClaudeFixture(t, "main-resumed.jsonl", ParseContext{
		RootSessionID:          "99999999-9999-4999-8999-999999999999",
		StreamID:               "agent-synthetic",
		EvidenceRevisionDigest: fixtureEvidenceDigest,
	})
	if main.UsageRecords[0].UsageID == child.UsageRecords[0].UsageID {
		t.Fatal("main and child streams produced the same usage ID")
	}
	if main.UsageRecords[0].SourceEventID != child.UsageRecords[0].SourceEventID {
		t.Fatal("source evidence identity changed across duplicate streams")
	}
	if child.RootSessionID != "99999999-9999-4999-8999-999999999999" || child.SourceStreamSessionID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("child root/source identities = %q/%q", child.RootSessionID, child.SourceStreamSessionID)
	}
	if child.UsageRecords[0].RootSessionID != child.RootSessionID {
		t.Fatalf("child usage root = %q", child.UsageRecords[0].RootSessionID)
	}
}

func TestParseClaudeCodeReportsUnknownRecord(t *testing.T) {
	result := parseClaudeFixture(t, "unknown-record.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if result.Coverage != CoverageUnsupported || result.UnsupportedRecords != 1 || len(result.Issues) != 1 {
		t.Fatalf("unknown record result = %#v", result)
	}
	if result.Issues[0].Code != "unknown_record_type" || result.Issues[0].RecordType != "future-record" {
		t.Fatalf("unknown record issue = %#v", result.Issues[0])
	}
}

func TestUnknownRecordCoverageRemainsUnsupportedAfterStructuralIssue(t *testing.T) {
	data := strings.Join([]string{
		`{"type":"started","version":"2.1.207","sessionId":"11111111-1111-4111-8111-111111111111"}`,
		`{"type":"future-record","version":"2.1.207","sessionId":"11111111-1111-4111-8111-111111111111"}`,
		`{"type":"assistant","version":"2.1.207","sessionId":"11111111-1111-4111-8111-111111111111","message":{}}`,
	}, "\n")
	result, err := ParseClaudeCode(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage != CoverageUnsupported {
		t.Fatalf("coverage = %q, want unsupported", result.Coverage)
	}
}

func TestParseClaudeCodeAcceptsPinnedNonUsageRecordTypes(t *testing.T) {
	result := parseClaudeFixture(t, "known-non-usage-records.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if result.Coverage != CoverageComplete || result.SupportedRecords != 8 || result.UnsupportedRecords != 0 {
		t.Fatalf("known non-usage result = %#v", result)
	}
	if len(result.UsageRecords) != 0 {
		t.Fatalf("known non-usage records produced usage: %#v", result.UsageRecords)
	}
}

func TestParseClaudeCodeFailsClosedOnVersionAndTypeDrift(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		_, err := ParseClaudeCode(bytes.NewReader(readClaudeFixture(t, "version-drift.jsonl")), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		var versionErr *UnsupportedVersionError
		if !errors.As(err, &versionErr) || versionErr.Observed != "2.2.0" {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("usage type", func(t *testing.T) {
		_, err := ParseClaudeCode(bytes.NewReader(readClaudeFixture(t, "wrong-usage-type.jsonl")), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		var formatErr *FormatError
		if !errors.As(err, &formatErr) || !strings.Contains(err.Error(), "input_tokens must be a non-negative integer") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		_, err := ParseClaudeCode(strings.NewReader("{not-json}\n"), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		var formatErr *FormatError
		if !errors.As(err, &formatErr) || !strings.Contains(err.Error(), "invalid JSON") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestParseCodexSupportedResumedTranscript(t *testing.T) {
	result := parseCodexFixture(t, "main-resumed.jsonl", ParseContext{
		StreamID:               "main",
		EvidenceRevisionDigest: fixtureEvidenceDigest,
	})
	if result.Coverage != CoverageComplete || result.SupportedRecords != 7 || result.UnsupportedRecords != 0 {
		t.Fatalf("coverage/counts = %s/%d/%d", result.Coverage, result.SupportedRecords, result.UnsupportedRecords)
	}
	if result.SourceVersion != SupportedCodexVersion || result.RootSessionID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("source identity = %q/%q", result.SourceVersion, result.RootSessionID)
	}
	if result.SourceStreamSessionID != result.RootSessionID {
		t.Fatalf("main source stream identity = %q", result.SourceStreamSessionID)
	}
	if len(result.UsageRecords) != 2 {
		t.Fatalf("usage records = %d, want 2", len(result.UsageRecords))
	}
	first := result.UsageRecords[0]
	second := result.UsageRecords[1]
	if first.Tokens.Total != 100 || second.Tokens.Total != 50 {
		t.Fatalf("incremental totals = %d/%d", first.Tokens.Total, second.Tokens.Total)
	}
	if first.TotalBasis != "source_reported" || second.TotalBasis != "source_reported" {
		t.Fatalf("Codex total basis = %q/%q", first.TotalBasis, second.TotalBasis)
	}
	if first.Model != "gpt-5.4" || second.Model != "gpt-5.4-mini" {
		t.Fatalf("models = %q/%q", first.Model, second.Model)
	}
	if first.CounterEpoch != 0 || second.CounterEpoch != 0 || first.ReconciliationStatus != "matched" || second.ReconciliationStatus != "matched" {
		t.Fatalf("reconciliation = %#v / %#v", first, second)
	}
	validateParsedUsage(t, result.UsageRecords)

	repeated := parseCodexFixture(t, "main-resumed.jsonl", ParseContext{StreamID: "main", EvidenceRevisionDigest: fixtureEvidenceDigest})
	if !reflect.DeepEqual(result, repeated) {
		t.Fatal("repeated Codex parse was not deterministic")
	}
}

func TestParseCodexChildStreamUsesHookRootIdentity(t *testing.T) {
	result := parseCodexFixture(t, "main-resumed.jsonl", ParseContext{
		RootSessionID:          "99999999-9999-4999-8999-999999999999",
		StreamID:               "agent-synthetic",
		EvidenceRevisionDigest: fixtureEvidenceDigest,
	})
	if result.RootSessionID != "99999999-9999-4999-8999-999999999999" || result.SourceStreamSessionID != "22222222-2222-4222-8222-222222222222" {
		t.Fatalf("child root/source identities = %q/%q", result.RootSessionID, result.SourceStreamSessionID)
	}
	for _, usage := range result.UsageRecords {
		if usage.RootSessionID != result.RootSessionID || usage.StreamID != "agent-synthetic" {
			t.Fatalf("child usage identity = %#v", usage)
		}
	}
}

func TestParseCodexStartsNewCounterEpochAfterReset(t *testing.T) {
	result := parseCodexFixture(t, "counter-reset.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if result.Coverage != CoverageComplete || len(result.UsageRecords) != 2 {
		t.Fatalf("reset result = %#v", result)
	}
	if result.UsageRecords[0].CounterEpoch != 0 || result.UsageRecords[1].CounterEpoch != 1 {
		t.Fatalf("counter epochs = %d/%d", result.UsageRecords[0].CounterEpoch, result.UsageRecords[1].CounterEpoch)
	}
	if result.UsageRecords[1].ReconciliationStatus != "matched" {
		t.Fatalf("second epoch reconciliation = %q", result.UsageRecords[1].ReconciliationStatus)
	}
}

func TestParseCodexReportsReconciliationMismatch(t *testing.T) {
	result := parseCodexFixture(t, "reconciliation-mismatch.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if result.Coverage != CoverageIncomplete || len(result.UsageRecords) != 1 || result.UsageRecords[0].ReconciliationStatus != "mismatch" {
		t.Fatalf("mismatch result = %#v", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].Code != "usage_reconciliation_mismatch" {
		t.Fatalf("mismatch issues = %#v", result.Issues)
	}
}

func TestParseCodexKeepsDirtyEpochRecordsMismatched(t *testing.T) {
	result := parseCodexFixture(t, "reconciliation-dirty-epoch.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if len(result.UsageRecords) != 3 {
		t.Fatalf("usage records = %d, want 3", len(result.UsageRecords))
	}
	if result.UsageRecords[1].ReconciliationStatus != "mismatch" {
		t.Fatalf("mismatching record status = %q", result.UsageRecords[1].ReconciliationStatus)
	}
	if result.UsageRecords[2].ReconciliationStatus == "matched" {
		t.Fatalf("recovering dirty-epoch record status = %q, must not be matched", result.UsageRecords[2].ReconciliationStatus)
	}
}

func TestParseCodexReportsUnknownShapes(t *testing.T) {
	result := parseCodexFixture(t, "unknown-record.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if result.Coverage != CoverageUnsupported || result.UnsupportedRecords != 3 || len(result.Issues) != 3 {
		t.Fatalf("unknown result = %#v", result)
	}
	wantCodes := []string{"unknown_record_type", "unknown_event_type", "unknown_response_item_type"}
	for i, want := range wantCodes {
		if result.Issues[i].Code != want {
			t.Fatalf("issue %d = %#v, want %s", i, result.Issues[i], want)
		}
	}
}

func TestParseCodexAcceptsPinnedNonUsageRecordTypes(t *testing.T) {
	result := parseCodexFixture(t, "known-non-usage-records.jsonl", ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	if result.Coverage != CoverageComplete || result.SupportedRecords != 5 || result.UnsupportedRecords != 0 {
		t.Fatalf("known non-usage result = %#v", result)
	}
	if len(result.UsageRecords) != 0 {
		t.Fatalf("known non-usage records produced usage: %#v", result.UsageRecords)
	}
}

func TestParseCodexFailsClosedOnVersionAndTypeDrift(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		_, err := ParseCodex(bytes.NewReader(readCodexFixture(t, "version-drift.jsonl")), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		var versionErr *UnsupportedVersionError
		if !errors.As(err, &versionErr) || versionErr.Observed != "0.145.0" {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("token type", func(t *testing.T) {
		_, err := ParseCodex(bytes.NewReader(readCodexFixture(t, "wrong-token-type.jsonl")), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		var formatErr *FormatError
		if !errors.As(err, &formatErr) || !strings.Contains(err.Error(), "input_tokens must be a non-negative integer") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("token before context", func(t *testing.T) {
		data := strings.Join([]string{
			`{"timestamp":"2026-07-12T09:00:00Z","type":"session_meta","payload":{"id":"22222222-2222-4222-8222-222222222222","session_id":"22222222-2222-4222-8222-222222222222","cli_version":"0.144.1","model_provider":"openai"}}`,
			`{"timestamp":"2026-07-12T09:00:01Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1,"reasoning_output_tokens":0,"total_tokens":2},"last_token_usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1,"reasoning_output_tokens":0,"total_tokens":2}}}}`,
		}, "\n")
		_, err := ParseCodex(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		if err == nil || !strings.Contains(err.Error(), "requires preceding session_meta and turn_context") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		_, err := ParseCodex(strings.NewReader("{not-json}\n"), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
		var formatErr *FormatError
		if !errors.As(err, &formatErr) || !strings.Contains(err.Error(), "invalid JSON") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestParsersRejectMultipleJSONValuesOnOneLine(t *testing.T) {
	tests := []struct {
		name  string
		parse func(string) error
	}{
		{name: "Claude", parse: func(data string) error {
			_, err := ParseClaudeCode(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
			return err
		}},
		{name: "Codex", parse: func(data string) error {
			_, err := ParseCodex(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse(`{"type":"future-record"} {"type":"second-record"}` + "\n")
			var formatErr *FormatError
			if !errors.As(err, &formatErr) || !strings.Contains(err.Error(), "multiple JSON values") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestParsersRejectLineOverRecordCap(t *testing.T) {
	data := strings.Repeat("x", maxSourceRecordBytes+1) + "\n"
	for _, tt := range []struct {
		name  string
		parse func() error
	}{
		{name: "Claude", parse: func() error {
			_, err := ParseClaudeCode(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
			return err
		}},
		{name: "Codex", parse: func() error {
			_, err := ParseCodex(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse()
			if err == nil || !strings.Contains(err.Error(), "token too long") {
				t.Fatalf("over-cap error = %v", err)
			}
		})
	}
}

func TestTokenCountsEnforceHardCapInBothAdapters(t *testing.T) {
	values := []struct {
		name    string
		value   int64
		wantErr bool
	}{
		{name: "near max int64", value: int64(^uint64(0) >> 1), wantErr: true},
		{name: "just over ceiling", value: maxTokenCount + 1, wantErr: true},
		{name: "just under ceiling", value: maxTokenCount - 1, wantErr: false},
	}
	for _, adapter := range []struct {
		name  string
		parse func(int64) error
	}{
		{name: "Claude", parse: parseClaudeTokenLimitFixture},
		{name: "Codex", parse: parseCodexTokenLimitFixture},
	} {
		for _, value := range values {
			t.Run(adapter.name+"/"+value.name, func(t *testing.T) {
				err := adapter.parse(value.value)
				var formatErr *FormatError
				if value.wantErr && (!errors.As(err, &formatErr) || !strings.Contains(err.Error(), "token-count limit")) {
					t.Fatalf("error = %v", err)
				}
				if !value.wantErr && err != nil {
					t.Fatalf("just-under-ceiling value rejected: %v", err)
				}
			})
		}
	}
}

func TestParseCodexRejectsCumulativeEpochSumOverCap(t *testing.T) {
	count := strconv.FormatInt(maxTokenCount, 10)
	data := codexTokenFixture(
		fmt.Sprintf(`{"input_tokens":%s,"cached_input_tokens":0,"output_tokens":0,"reasoning_output_tokens":0,"total_tokens":%s}`, count, count),
		fmt.Sprintf(`{"input_tokens":%s,"cached_input_tokens":0,"output_tokens":0,"reasoning_output_tokens":0,"total_tokens":%s}`, count, count),
	) + "\n" + codexTokenEvent(
		`{"input_tokens":1,"cached_input_tokens":0,"output_tokens":0,"reasoning_output_tokens":0,"total_tokens":1}`,
		fmt.Sprintf(`{"input_tokens":%s,"cached_input_tokens":0,"output_tokens":0,"reasoning_output_tokens":0,"total_tokens":%s}`, count, count),
		"2026-07-12T09:00:03Z",
	)
	_, err := ParseCodex(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	var formatErr *FormatError
	if !errors.As(err, &formatErr) || !strings.Contains(err.Error(), "cumulative epoch sum exceeds") {
		t.Fatalf("error = %v", err)
	}
}

func parseClaudeTokenLimitFixture(value int64) error {
	count := strconv.FormatInt(value, 10)
	data := strings.Join([]string{
		`{"type":"started","version":"2.1.207","sessionId":"11111111-1111-4111-8111-111111111111"}`,
		fmt.Sprintf(`{"type":"assistant","version":"2.1.207","sessionId":"11111111-1111-4111-8111-111111111111","uuid":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2","timestamp":"2026-07-12T09:00:01Z","message":{"id":"msg_synthetic_001","model":"claude-test","usage":{"input_tokens":%s,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`, count),
	}, "\n")
	_, err := ParseClaudeCode(strings.NewReader(data), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	return err
}

func parseCodexTokenLimitFixture(value int64) error {
	count := strconv.FormatInt(value, 10)
	tokens := fmt.Sprintf(`{"input_tokens":%s,"cached_input_tokens":0,"output_tokens":0,"reasoning_output_tokens":0,"total_tokens":%s}`, count, count)
	_, err := ParseCodex(strings.NewReader(codexTokenFixture(tokens, tokens)), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest})
	return err
}

func codexTokenFixture(last string, total string) string {
	return strings.Join([]string{
		`{"timestamp":"2026-07-12T09:00:00Z","type":"session_meta","payload":{"id":"22222222-2222-4222-8222-222222222222","session_id":"22222222-2222-4222-8222-222222222222","cli_version":"0.144.1","model_provider":"openai"}}`,
		`{"timestamp":"2026-07-12T09:00:01Z","type":"turn_context","payload":{"turn_id":"33333333-3333-4333-8333-333333333331","cwd":"/tmp/project","model":"gpt-test"}}`,
		codexTokenEvent(last, total, "2026-07-12T09:00:02Z"),
	}, "\n")
}

func codexTokenEvent(last string, total string, timestamp string) string {
	return fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":%s,"last_token_usage":%s}}}`, timestamp, total, last)
}

func TestParsersRequireEvidenceDigest(t *testing.T) {
	if _, err := ParseClaudeCode(bytes.NewReader(readClaudeFixture(t, "main-resumed.jsonl")), ParseContext{}); err == nil {
		t.Fatal("Claude parser accepted empty evidence digest")
	}
	if _, err := ParseCodex(bytes.NewReader(readCodexFixture(t, "main-resumed.jsonl")), ParseContext{}); err == nil {
		t.Fatal("Codex parser accepted empty evidence digest")
	}
	invalid := ParseContext{EvidenceRevisionDigest: "sha256:not-hex"}
	if _, err := ParseClaudeCode(bytes.NewReader(readClaudeFixture(t, "main-resumed.jsonl")), invalid); err == nil {
		t.Fatal("Claude parser accepted invalid evidence digest")
	}
	if _, err := ParseCodex(bytes.NewReader(readCodexFixture(t, "main-resumed.jsonl")), invalid); err == nil {
		t.Fatal("Codex parser accepted invalid evidence digest")
	}
}

func TestParsersRejectInvalidUsageTimestamps(t *testing.T) {
	claudeData := strings.Replace(string(readClaudeFixture(t, "main-resumed.jsonl")), "2026-07-12T09:00:01Z", "not-a-time", 1)
	if _, err := ParseClaudeCode(strings.NewReader(claudeData), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest}); err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("Claude timestamp error = %v", err)
	}
	codexData := strings.Replace(string(readCodexFixture(t, "main-resumed.jsonl")), "2026-07-12T09:00:03Z", "not-a-time", 1)
	if _, err := ParseCodex(strings.NewReader(codexData), ParseContext{EvidenceRevisionDigest: fixtureEvidenceDigest}); err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("Codex timestamp error = %v", err)
	}
}

func TestMainStreamRejectsMismatchedHookRootIdentity(t *testing.T) {
	context := ParseContext{
		RootSessionID:          "99999999-9999-4999-8999-999999999999",
		StreamID:               "main",
		EvidenceRevisionDigest: fixtureEvidenceDigest,
	}
	if _, err := ParseClaudeCode(bytes.NewReader(readClaudeFixture(t, "main-resumed.jsonl")), context); err == nil || !strings.Contains(err.Error(), "does not match root context") {
		t.Fatalf("Claude mismatch error = %v", err)
	}
	if _, err := ParseCodex(bytes.NewReader(readCodexFixture(t, "main-resumed.jsonl")), context); err == nil || !strings.Contains(err.Error(), "does not match root context") {
		t.Fatalf("Codex mismatch error = %v", err)
	}
}

func parseClaudeFixture(t *testing.T, name string, context ParseContext) ParseResult {
	t.Helper()
	result, err := ParseClaudeCode(bytes.NewReader(readClaudeFixture(t, name)), context)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func parseCodexFixture(t *testing.T, name string, context ParseContext) ParseResult {
	t.Helper()
	result, err := ParseCodex(bytes.NewReader(readCodexFixture(t, name)), context)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func readClaudeFixture(t *testing.T, name string) []byte {
	t.Helper()
	return readFixture(t, "fixtures", "wave1", "sources", "claude-code", SupportedClaudeCodeVersion, name)
}

func readCodexFixture(t *testing.T, name string) []byte {
	t.Helper()
	return readFixture(t, "fixtures", "wave1", "sources", "codex", SupportedCodexVersion, name)
}

func readFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	// #nosec G304 -- all path components are fixed test fixture names.
	data, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func validateParsedUsage(t *testing.T, records []UsageRecord) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	schemaPath, ok := qschema.RegistryFile(UsageRecordSchemaVersion)
	if !ok {
		t.Fatal("usage schema is not registered")
	}
	// #nosec G304 -- schemaPath comes from the in-process registry.
	schemaData, err := os.ReadFile(filepath.Join(root, schemaPath))
	if err != nil {
		t.Fatal(err)
	}
	for i, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := qschema.Validate(schemaData, data); err != nil {
			t.Fatalf("usage record %d failed schema: %v\n%s", i, err, data)
		}
	}
}

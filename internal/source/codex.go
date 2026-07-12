package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// SupportedCodexVersion is the exact source shape pinned by Wave 1 fixtures.
const SupportedCodexVersion = "0.144.1"

var knownCodexEventTypes = map[string]struct{}{
	"agent_message":           {},
	"collab_agent_spawn_end":  {},
	"collab_close_end":        {},
	"collab_waiting_end":      {},
	"context_compacted":       {},
	"entered_review_mode":     {},
	"exec_command_end":        {},
	"exited_review_mode":      {},
	"mcp_tool_call_end":       {},
	"patch_apply_end":         {},
	"sub_agent_activity":      {},
	"task_complete":           {},
	"task_started":            {},
	"thread_settings_applied": {},
	"token_count":             {},
	"turn_aborted":            {},
	"user_message":            {},
	"web_search_end":          {},
}

var knownCodexResponseTypes = map[string]struct{}{
	"agent_message":           {},
	"custom_tool_call":        {},
	"custom_tool_call_output": {},
	"function_call":           {},
	"function_call_output":    {},
	"local_shell_call":        {},
	"message":                 {},
	"reasoning":               {},
	"web_search_call":         {},
}

type codexParser struct {
	context       ParseContext
	result        ParseResult
	provider      string
	currentTurnID string
	currentModel  string
	tokenOrdinal  int64
	counterEpoch  int64
	epochSum      TokenCounts
	previousTotal *TokenCounts
	epochIndexes  []int
	epochMismatch bool
}

// ParseCodex extracts stable session identity and incremental token events.
func ParseCodex(reader io.Reader, context ParseContext) (ParseResult, error) {
	if err := validateParseContext(context); err != nil {
		return ParseResult{}, err
	}
	parser := codexParser{
		context: context,
		result: ParseResult{
			Source:       SourceCodex,
			StreamID:     streamID(context),
			Coverage:     CoverageComplete,
			Issues:       []FormatIssue{},
			UsageRecords: []UsageRecord{},
		},
		epochIndexes: []int{},
	}
	if err := scanJSONLines(reader, parser.parseLine); err != nil {
		return ParseResult{}, err
	}
	if parser.result.SourceVersion == "" {
		return ParseResult{}, &UnsupportedVersionError{Source: SourceCodex, Supported: SupportedCodexVersion, Observed: "missing"}
	}
	if parser.result.RootSessionID == "" {
		return ParseResult{}, fmt.Errorf("codex transcript has no source session ID")
	}
	return parser.result, nil
}

func (p *codexParser) parseLine(lineNo int, line []byte) error {
	fields, err := decodeObject(lineNo, line)
	if err != nil {
		return err
	}
	recordType, err := requiredString(lineNo, fields, "type")
	if err != nil {
		return err
	}

	switch recordType {
	case "session_meta":
		p.result.SupportedRecords++
		return p.parseSessionMeta(lineNo, fields)
	case "turn_context":
		p.result.SupportedRecords++
		return p.parseTurnContext(lineNo, fields)
	case "event_msg":
		return p.parseEvent(lineNo, fields, line)
	case "response_item":
		return p.parseResponseItem(lineNo, fields)
	default:
		addUnsupportedIssue(&p.result, FormatIssue{
			Line:       lineNo,
			Code:       "unknown_record_type",
			RecordType: recordType,
			Detail:     "record was preserved but not interpreted",
		})
		return nil
	}
}

func (p *codexParser) parseSessionMeta(lineNo int, fields map[string]json.RawMessage) error {
	payload, err := requiredObject(lineNo, fields, "payload")
	if err != nil {
		return err
	}
	version, err := requiredString(lineNo, payload, "cli_version")
	if err != nil {
		return err
	}
	if version != SupportedCodexVersion {
		return &UnsupportedVersionError{Source: SourceCodex, Supported: SupportedCodexVersion, Observed: version}
	}
	sessionID, err := requiredString(lineNo, payload, "session_id")
	if err != nil {
		return err
	}
	id, err := requiredString(lineNo, payload, "id")
	if err != nil {
		return err
	}
	if id != sessionID {
		return &FormatError{Line: lineNo, Detail: "session_meta id and session_id differ"}
	}
	if p.result.SourceStreamSessionID != "" && p.result.SourceStreamSessionID != sessionID {
		return &FormatError{Line: lineNo, Detail: "conflicting Codex session IDs"}
	}
	provider, err := requiredString(lineNo, payload, "model_provider")
	if err != nil {
		return err
	}
	p.result.SourceVersion = version
	p.result.SourceStreamSessionID = sessionID
	rootSessionID, err := effectiveRootSessionID(p.context, sessionID)
	if err != nil {
		return &FormatError{Line: lineNo, Detail: err.Error()}
	}
	p.result.RootSessionID = rootSessionID
	p.provider = provider
	return nil
}

func (p *codexParser) parseTurnContext(lineNo int, fields map[string]json.RawMessage) error {
	if p.result.RootSessionID == "" {
		return &FormatError{Line: lineNo, Detail: "turn_context appeared before session_meta"}
	}
	payload, err := requiredObject(lineNo, fields, "payload")
	if err != nil {
		return err
	}
	turnID, err := requiredString(lineNo, payload, "turn_id")
	if err != nil {
		return err
	}
	model, err := requiredString(lineNo, payload, "model")
	if err != nil {
		return err
	}
	if _, err := requiredString(lineNo, payload, "cwd"); err != nil {
		return err
	}
	p.currentTurnID = turnID
	p.currentModel = model
	return nil
}

func (p *codexParser) parseEvent(lineNo int, fields map[string]json.RawMessage, rawLine []byte) error {
	payload, err := requiredObject(lineNo, fields, "payload")
	if err != nil {
		return err
	}
	eventType, err := requiredString(lineNo, payload, "type")
	if err != nil {
		return err
	}
	if _, known := knownCodexEventTypes[eventType]; !known {
		addUnsupportedIssue(&p.result, FormatIssue{
			Line:       lineNo,
			Code:       "unknown_event_type",
			RecordType: eventType,
			Detail:     "event payload was preserved but not interpreted",
		})
		return nil
	}
	p.result.SupportedRecords++
	if eventType != "token_count" {
		return nil
	}
	return p.parseTokenCount(lineNo, fields, payload, rawLine)
}

func (p *codexParser) parseResponseItem(lineNo int, fields map[string]json.RawMessage) error {
	payload, err := requiredObject(lineNo, fields, "payload")
	if err != nil {
		return err
	}
	itemType, err := requiredString(lineNo, payload, "type")
	if err != nil {
		return err
	}
	if _, known := knownCodexResponseTypes[itemType]; !known {
		addUnsupportedIssue(&p.result, FormatIssue{
			Line:       lineNo,
			Code:       "unknown_response_item_type",
			RecordType: itemType,
			Detail:     "response item was preserved but not interpreted",
		})
		return nil
	}
	p.result.SupportedRecords++
	return nil
}

func (p *codexParser) parseTokenCount(lineNo int, fields map[string]json.RawMessage, payload map[string]json.RawMessage, rawLine []byte) error {
	info, hasInfo, err := optionalObject(lineNo, payload, "info")
	if err != nil {
		return err
	}
	if !hasInfo {
		return nil
	}
	if p.result.RootSessionID == "" || p.currentTurnID == "" || p.currentModel == "" || p.provider == "" {
		return &FormatError{Line: lineNo, Detail: "token_count requires preceding session_meta and turn_context"}
	}
	lastRaw, err := requiredObject(lineNo, info, "last_token_usage")
	if err != nil {
		return err
	}
	totalRaw, err := requiredObject(lineNo, info, "total_token_usage")
	if err != nil {
		return err
	}
	last, err := parseCodexTokenCounts(lineNo, lastRaw)
	if err != nil {
		return err
	}
	total, err := parseCodexTokenCounts(lineNo, totalRaw)
	if err != nil {
		return err
	}

	if p.previousTotal != nil && tokenCountsDecreased(total, *p.previousTotal) {
		p.counterEpoch++
		p.epochSum = TokenCounts{}
		p.epochIndexes = nil
		p.epochMismatch = false
	}
	p.epochSum, err = addTokenCounts(lineNo, p.epochSum, last)
	if err != nil {
		return err
	}
	p.tokenOrdinal++
	rawDigest := sha256.Sum256(rawLine)
	eventDigest := hex.EncodeToString(rawDigest[:])
	sourceEventID := strconv.FormatInt(p.tokenOrdinal, 10) + ":" + eventDigest
	timestamp, err := requiredTimestamp(lineNo, fields, "timestamp")
	if err != nil {
		return err
	}

	record := UsageRecord{
		SchemaVersion:          UsageRecordSchemaVersion,
		DataClass:              DataClassRaw,
		UsageID:                stableUsageID(SourceCodex, p.result.RootSessionID, p.result.StreamID, strconv.FormatInt(p.tokenOrdinal, 10), eventDigest),
		Source:                 SourceCodex,
		SourceVersion:          p.result.SourceVersion,
		RootSessionID:          p.result.RootSessionID,
		StreamID:               p.result.StreamID,
		SourceEventID:          sourceEventID,
		TurnID:                 p.currentTurnID,
		Model:                  p.currentModel,
		Provider:               p.provider,
		Tokens:                 last,
		TotalBasis:             "source_reported",
		OccurredAt:             timestamp,
		TimeBasis:              "source_reported",
		Semantics:              "incremental",
		Reliability:            "exact_source_reported",
		EvidenceRevisionDigest: evidenceDigest(p.context),
		DuplicateStatus:        "accepted",
		ReconciliationStatus:   "matched",
		CounterEpoch:           p.counterEpoch,
	}
	p.result.UsageRecords = append(p.result.UsageRecords, record)
	p.epochIndexes = append(p.epochIndexes, len(p.result.UsageRecords)-1)

	switch {
	case !equalTokenCounts(p.epochSum, total):
		p.epochMismatch = true
		for _, index := range p.epochIndexes {
			p.result.UsageRecords[index].ReconciliationStatus = "mismatch"
		}
		addIssue(&p.result, FormatIssue{
			Line:       lineNo,
			Code:       "usage_reconciliation_mismatch",
			RecordType: "token_count",
			Detail:     "sum of incremental usage does not match cumulative usage for the counter epoch",
		})
	case !p.epochMismatch:
		for _, index := range p.epochIndexes {
			p.result.UsageRecords[index].ReconciliationStatus = "matched"
		}
	default:
		p.result.UsageRecords[len(p.result.UsageRecords)-1].ReconciliationStatus = "mismatch"
	}
	p.previousTotal = &total
	return nil
}

func parseCodexTokenCounts(lineNo int, fields map[string]json.RawMessage) (TokenCounts, error) {
	input, err := requiredTokenCount(lineNo, fields, "input_tokens")
	if err != nil {
		return TokenCounts{}, err
	}
	output, err := requiredTokenCount(lineNo, fields, "output_tokens")
	if err != nil {
		return TokenCounts{}, err
	}
	cacheRead, err := requiredTokenCount(lineNo, fields, "cached_input_tokens")
	if err != nil {
		return TokenCounts{}, err
	}
	reasoning, err := requiredTokenCount(lineNo, fields, "reasoning_output_tokens")
	if err != nil {
		return TokenCounts{}, err
	}
	total, err := requiredTokenCount(lineNo, fields, "total_tokens")
	if err != nil {
		return TokenCounts{}, err
	}
	return TokenCounts{
		Input:           input,
		Output:          output,
		CacheRead:       cacheRead,
		ReasoningOutput: reasoning,
		Total:           total,
	}, nil
}

func addTokenCounts(lineNo int, left TokenCounts, right TokenCounts) (TokenCounts, error) {
	values := []struct {
		name        string
		left, right int64
	}{
		{"input_tokens", left.Input, right.Input},
		{"output_tokens", left.Output, right.Output},
		{"cached_input_tokens", left.CacheRead, right.CacheRead},
		{"cache_creation_input_tokens", left.CacheCreation, right.CacheCreation},
		{"cache_creation_5m_input_tokens", left.CacheCreationFiveMin, right.CacheCreationFiveMin},
		{"cache_creation_1h_input_tokens", left.CacheCreationOneHour, right.CacheCreationOneHour},
		{"reasoning_output_tokens", left.ReasoningOutput, right.ReasoningOutput},
		{"total_tokens", left.Total, right.Total},
	}
	sums := make([]int64, len(values))
	for i, value := range values {
		if value.left > maxTokenCount-value.right {
			return TokenCounts{}, &FormatError{Line: lineNo, Detail: value.name + " cumulative epoch sum exceeds the token-count limit"}
		}
		sums[i] = value.left + value.right
	}
	return TokenCounts{
		Input: sums[0], CacheRead: sums[2], Output: sums[1], CacheCreation: sums[3],
		CacheCreationFiveMin: sums[4], CacheCreationOneHour: sums[5],
		ReasoningOutput: sums[6], Total: sums[7],
	}, nil
}

func equalTokenCounts(left TokenCounts, right TokenCounts) bool {
	return left == right
}

func tokenCountsDecreased(current TokenCounts, previous TokenCounts) bool {
	return current.Input < previous.Input ||
		current.Output < previous.Output ||
		current.CacheRead < previous.CacheRead ||
		current.CacheCreation < previous.CacheCreation ||
		current.CacheCreationFiveMin < previous.CacheCreationFiveMin ||
		current.CacheCreationOneHour < previous.CacheCreationOneHour ||
		current.ReasoningOutput < previous.ReasoningOutput ||
		current.Total < previous.Total
}

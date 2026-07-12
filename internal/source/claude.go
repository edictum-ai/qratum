package source

import (
	"encoding/json"
	"fmt"
	"io"
)

// SupportedClaudeCodeVersion is the exact source shape pinned by Wave 1 fixtures.
const SupportedClaudeCodeVersion = "2.1.207"

var knownClaudeRecordTypes = map[string]struct{}{
	"agent-setting":         {},
	"ai-title":              {},
	"assistant":             {},
	"attachment":            {},
	"compacted":             {},
	"custom-title":          {},
	"file-history-snapshot": {},
	"frame-link":            {},
	"last-prompt":           {},
	"mode":                  {},
	"permission-mode":       {},
	"progress":              {},
	"pr-link":               {},
	"queue-operation":       {},
	"relocated":             {},
	"result":                {},
	"started":               {},
	"summary":               {},
	"system":                {},
	"user":                  {},
	"worktree-state":        {},
}

// ParseClaudeCode extracts stable source identity and per-message usage.
func ParseClaudeCode(reader io.Reader, context ParseContext) (ParseResult, error) {
	result := ParseResult{
		Source:       SourceClaudeCode,
		StreamID:     streamID(context),
		Coverage:     CoverageComplete,
		Issues:       []FormatIssue{},
		UsageRecords: []UsageRecord{},
	}
	if err := validateParseContext(context); err != nil {
		return ParseResult{}, err
	}

	err := scanJSONLines(reader, func(lineNo int, line []byte) error {
		fields, err := decodeObject(lineNo, line)
		if err != nil {
			return err
		}
		recordType, err := requiredString(lineNo, fields, "type")
		if err != nil {
			return err
		}

		version, err := optionalString(lineNo, fields, "version")
		if err != nil {
			return err
		}
		if version != "" {
			if version != SupportedClaudeCodeVersion {
				return &UnsupportedVersionError{Source: SourceClaudeCode, Supported: SupportedClaudeCodeVersion, Observed: version}
			}
			result.SourceVersion = version
		}

		sessionID, err := optionalString(lineNo, fields, "sessionId")
		if err != nil {
			return err
		}
		if sessionID != "" {
			if result.SourceStreamSessionID != "" && result.SourceStreamSessionID != sessionID {
				return &FormatError{Line: lineNo, Detail: "conflicting sessionId values"}
			}
			result.SourceStreamSessionID = sessionID
			rootSessionID, err := effectiveRootSessionID(context, sessionID)
			if err != nil {
				return &FormatError{Line: lineNo, Detail: err.Error()}
			}
			result.RootSessionID = rootSessionID
		}

		if _, known := knownClaudeRecordTypes[recordType]; !known {
			addUnsupportedIssue(&result, FormatIssue{
				Line:       lineNo,
				Code:       "unknown_record_type",
				RecordType: recordType,
				Detail:     "record was preserved but not interpreted",
			})
			return nil
		}
		result.SupportedRecords++
		if recordType != "assistant" {
			return nil
		}
		if version == "" {
			return &FormatError{Line: lineNo, Detail: "assistant record is missing version"}
		}
		if sessionID == "" {
			return &FormatError{Line: lineNo, Detail: "assistant record is missing sessionId"}
		}

		usageRecord, ok, issue, err := parseClaudeAssistantUsage(lineNo, fields, result.RootSessionID, result.StreamID, context)
		if err != nil {
			return err
		}
		if issue != nil {
			addIssue(&result, *issue)
			return nil
		}
		if ok {
			result.UsageRecords = append(result.UsageRecords, usageRecord)
		}
		return nil
	})
	if err != nil {
		return ParseResult{}, err
	}
	if result.SourceVersion == "" {
		return ParseResult{}, &UnsupportedVersionError{Source: SourceClaudeCode, Supported: SupportedClaudeCodeVersion, Observed: "missing"}
	}
	if result.RootSessionID == "" {
		return ParseResult{}, fmt.Errorf("claude-code transcript has no source session ID")
	}
	return result, nil
}

func parseClaudeAssistantUsage(lineNo int, fields map[string]json.RawMessage, rootSessionID string, stream string, context ParseContext) (UsageRecord, bool, *FormatIssue, error) {
	message, err := requiredObject(lineNo, fields, "message")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	usage, ok, err := optionalObject(lineNo, message, "usage")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	if !ok {
		return UsageRecord{}, false, &FormatIssue{
			Line:       lineNo,
			Code:       "assistant_usage_missing",
			RecordType: "assistant",
			Detail:     "assistant record had no source-reported usage",
		}, nil
	}

	messageUUID, err := requiredString(lineNo, fields, "uuid")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	messageID, err := requiredString(lineNo, message, "id")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	model, err := requiredString(lineNo, message, "model")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	timestamp, err := requiredTimestamp(lineNo, fields, "timestamp")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	requestID, err := optionalString(lineNo, fields, "requestId")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	if requestID == "" {
		requestID = messageID
	}

	// service_tier is intentionally preserved only in the raw blob in T1.1;
	// capturing it as pricing input is deferred to T1.4.
	input, err := requiredTokenCount(lineNo, usage, "input_tokens")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	output, err := requiredTokenCount(lineNo, usage, "output_tokens")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	cacheCreation, err := requiredTokenCount(lineNo, usage, "cache_creation_input_tokens")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	cacheRead, err := requiredTokenCount(lineNo, usage, "cache_read_input_tokens")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}

	var cacheFiveMin int64
	var cacheOneHour int64
	cacheDetails, hasCacheDetails, err := optionalObject(lineNo, usage, "cache_creation")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	if hasCacheDetails {
		cacheFiveMin, err = requiredTokenCount(lineNo, cacheDetails, "ephemeral_5m_input_tokens")
		if err != nil {
			return UsageRecord{}, false, nil, err
		}
		cacheOneHour, err = requiredTokenCount(lineNo, cacheDetails, "ephemeral_1h_input_tokens")
		if err != nil {
			return UsageRecord{}, false, nil, err
		}
		if cacheFiveMin+cacheOneHour != cacheCreation {
			return UsageRecord{}, false, nil, &FormatError{Line: lineNo, Detail: "cache_creation token details do not match cache_creation_input_tokens"}
		}
	}

	version, err := requiredString(lineNo, fields, "version")
	if err != nil {
		return UsageRecord{}, false, nil, err
	}
	tokens := TokenCounts{
		Input:                input,
		Output:               output,
		CacheRead:            cacheRead,
		CacheCreation:        cacheCreation,
		CacheCreationFiveMin: cacheFiveMin,
		CacheCreationOneHour: cacheOneHour,
		Total:                input + output + cacheRead + cacheCreation,
	}

	return UsageRecord{
		SchemaVersion:          UsageRecordSchemaVersion,
		DataClass:              DataClassRaw,
		UsageID:                stableUsageID(SourceClaudeCode, rootSessionID, stream, messageUUID, messageID),
		Source:                 SourceClaudeCode,
		SourceVersion:          version,
		RootSessionID:          rootSessionID,
		StreamID:               stream,
		SourceEventID:          requestID,
		MessageID:              messageID,
		Model:                  model,
		Provider:               "anthropic",
		Tokens:                 tokens,
		TotalBasis:             "derived_sum",
		OccurredAt:             timestamp,
		TimeBasis:              "source_reported",
		Semantics:              "per_message",
		Reliability:            "exact_source_reported",
		EvidenceRevisionDigest: evidenceDigest(context),
		DuplicateStatus:        "accepted",
		ReconciliationStatus:   "not_applicable",
		CounterEpoch:           0,
	}, true, nil, nil
}

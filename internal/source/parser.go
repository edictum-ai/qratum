package source

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// maxSourceRecordBytes is 10 MiB: far above observed source records while
// bounding memory used by one untrusted JSONL value. Scanner rejects an
// over-cap line and aborts the parse because it cannot safely resume at the
// next record without replacing the scanner with a bounded discarding reader.
const maxSourceRecordBytes = 10 << 20

// maxTokenCount is a per-field hard cap for untrusted source-reported counts.
// One trillion tokens is well above current model context and session counters,
// while leaving ample int64 headroom for derived per-record totals.
const maxTokenCount int64 = 1_000_000_000_000

// Parse coverage values distinguish complete interpretation from visible drift.
const (
	CoverageComplete    = "complete"
	CoverageIncomplete  = "incomplete"
	CoverageUnsupported = "unsupported"
)

// ParseContext supplies preservation facts that are not part of a transcript line.
type ParseContext struct {
	RootSessionID          string
	StreamID               string
	EvidenceRevisionDigest string
}

// FormatIssue is a visible unsupported source shape that did not require guessing.
type FormatIssue struct {
	Line       int    `json:"line"`
	Code       string `json:"code"`
	RecordType string `json:"record_type,omitempty"`
	Detail     string `json:"detail"`
}

// ParseResult contains only source identity and usage needed by Wave 1.
type ParseResult struct {
	Source                string        `json:"source"`
	SourceVersion         string        `json:"source_version"`
	RootSessionID         string        `json:"root_session_id"`
	SourceStreamSessionID string        `json:"source_stream_session_id"`
	StreamID              string        `json:"stream_id"`
	Coverage              string        `json:"coverage"`
	SupportedRecords      int64         `json:"supported_records"`
	UnsupportedRecords    int64         `json:"unsupported_records"`
	Issues                []FormatIssue `json:"issues"`
	UsageRecords          []UsageRecord `json:"usage_records"`
}

// UnsupportedVersionError reports source-version drift without best-effort parsing.
type UnsupportedVersionError struct {
	Source    string
	Supported string
	Observed  string
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("%s source version %q is unsupported; expected %q", e.Source, e.Observed, e.Supported)
}

// FormatError reports malformed or wrongly typed source input.
type FormatError struct {
	Line   int
	Detail string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Detail)
}

func scanJSONLines(reader io.Reader, visit func(lineNo int, line []byte) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), maxSourceRecordBytes)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := visit(lineNo, bytes.Clone(line)); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read source JSONL: %w", err)
	}
	return nil
}

func decodeObject(lineNo int, data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value map[string]json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return nil, &FormatError{Line: lineNo, Detail: "invalid JSON: " + err.Error()}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, &FormatError{Line: lineNo, Detail: "multiple JSON values are not allowed"}
		}
		return nil, &FormatError{Line: lineNo, Detail: "invalid trailing JSON: " + err.Error()}
	}
	if value == nil {
		return nil, &FormatError{Line: lineNo, Detail: "record must be an object"}
	}
	return value, nil
}

func requiredString(lineNo int, fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok {
		return "", &FormatError{Line: lineNo, Detail: fmt.Sprintf("missing required string %q", name)}
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be a string", name)}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must not be empty", name)}
	}
	return value, nil
}

func requiredTimestamp(lineNo int, fields map[string]json.RawMessage, name string) (string, error) {
	value, err := requiredString(lineNo, fields, name)
	if err != nil {
		return "", err
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return "", &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be an RFC3339 timestamp", name)}
	}
	return value, nil
}

func optionalString(lineNo int, fields map[string]json.RawMessage, name string) (string, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be a string or null", name)}
	}
	return strings.TrimSpace(value), nil
}

func requiredObject(lineNo int, fields map[string]json.RawMessage, name string) (map[string]json.RawMessage, error) {
	raw, ok := fields[name]
	if !ok {
		return nil, &FormatError{Line: lineNo, Detail: fmt.Sprintf("missing required object %q", name)}
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be an object", name)}
	}
	return value, nil
}

func optionalObject(lineNo int, fields map[string]json.RawMessage, name string) (map[string]json.RawMessage, bool, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, false, nil
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, false, &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be an object or null", name)}
	}
	return value, true, nil
}

func requiredNonNegativeInt(lineNo int, fields map[string]json.RawMessage, name string) (int64, error) {
	raw, ok := fields[name]
	if !ok {
		return 0, &FormatError{Line: lineNo, Detail: fmt.Sprintf("missing required integer %q", name)}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return 0, &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be an integer", name)}
	}
	value, ok := decoded.(json.Number)
	if !ok {
		return 0, &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be a non-negative integer", name)}
	}
	integer, err := value.Int64()
	if err != nil || integer < 0 {
		return 0, &FormatError{Line: lineNo, Detail: fmt.Sprintf("%s must be a non-negative integer", name)}
	}
	return integer, nil
}

func requiredTokenCount(lineNo int, fields map[string]json.RawMessage, name string) (int64, error) {
	value, err := requiredNonNegativeInt(lineNo, fields, name)
	if err != nil {
		return 0, err
	}
	if value > maxTokenCount {
		return 0, &FormatError{
			Line:   lineNo,
			Detail: fmt.Sprintf("%s exceeds the token-count limit of %d", name, maxTokenCount),
		}
	}
	return value, nil
}

func streamID(context ParseContext) string {
	value := strings.TrimSpace(context.StreamID)
	if value == "" {
		return "main"
	}
	return value
}

func effectiveRootSessionID(context ParseContext, sourceStreamSessionID string) (string, error) {
	root := strings.TrimSpace(context.RootSessionID)
	if root == "" {
		return sourceStreamSessionID, nil
	}
	if streamID(context) == "main" && root != sourceStreamSessionID {
		return "", fmt.Errorf("main stream source session ID %q does not match root context %q", sourceStreamSessionID, root)
	}
	return root, nil
}

func evidenceDigest(context ParseContext) string {
	return strings.TrimSpace(context.EvidenceRevisionDigest)
}

func validateParseContext(context ParseContext) error {
	digest := evidenceDigest(context)
	const prefix = "sha256:"
	if !strings.HasPrefix(digest, prefix) {
		return fmt.Errorf("evidence revision digest must use sha256:<64 hex>")
	}
	hexValue := strings.TrimPrefix(digest, prefix)
	if len(hexValue) != 64 {
		return fmt.Errorf("evidence revision digest must use sha256:<64 hex>")
	}
	decoded, err := hex.DecodeString(hexValue)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("evidence revision digest must use sha256:<64 hex>")
	}
	return nil
}

func addIssue(result *ParseResult, issue FormatIssue) {
	result.Issues = append(result.Issues, issue)
	result.UnsupportedRecords++
	if result.Coverage != CoverageUnsupported {
		result.Coverage = CoverageIncomplete
	}
}

func addUnsupportedIssue(result *ParseResult, issue FormatIssue) {
	result.Issues = append(result.Issues, issue)
	result.UnsupportedRecords++
	result.Coverage = CoverageUnsupported
}

func stableUsageID(source string, parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(hash, "%d:", len(part))
		_, _ = hash.Write([]byte(part))
	}
	return source + ":" + hex.EncodeToString(hash.Sum(nil))
}

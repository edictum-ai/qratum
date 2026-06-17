// Package capture owns the fast Claude Code hook capture path.
package capture

import (
	"crypto/sha256"
	"io"
	"os"

	"github.com/edictum-ai/qratum/internal/vault"
	"github.com/edictum-ai/qratum/internal/workspace"
)

const (
	// EventSchemaVersion is the capture event schema version.
	EventSchemaVersion = "qratum.event.v1"
	// ClaudeCodeSource labels Claude Code capture events.
	ClaudeCodeSource = "claude-code"
	// DeprecatedUnixZeroHookTimestamp is the legacy invalid hook timestamp sentinel.
	DeprecatedUnixZeroHookTimestamp = "1970-01-01T00:00:00Z"
	// HookTimestampSourceHookPayload means the hook payload supplied the event timestamp.
	HookTimestampSourceHookPayload = "hook_payload"
	// HookTimestampSourceCaptureTime means qrt supplied the event timestamp at capture time.
	HookTimestampSourceCaptureTime = "capture_time"
	// HookTimestampSourceTranscriptEnd means the daemon later aligned the event timestamp to transcript end.
	HookTimestampSourceTranscriptEnd = "transcript_end"
	// MaxHookPayloadBytes caps hook stdin so the hook remains fast.
	MaxHookPayloadBytes = 1 << 20
)

// ClaudeCodeHookPayload is the small JSON object Claude Code sends to the hook.
type ClaudeCodeHookPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      string `json:"timestamp"`
}

// Event is the capture event written to the qratum event spool.
type Event struct {
	SchemaVersion   string       `json:"schema_version"`
	EventID         string       `json:"event_id"`
	Source          string       `json:"source"`
	EventType       string       `json:"event_type"`
	Timestamp       string       `json:"timestamp"`
	TimestampSource string       `json:"timestamp_source,omitempty"`
	SessionRef      SessionRef   `json:"session_ref"`
	Workspace       WorkspaceRef `json:"workspace"`
	Raw             *EventRaw    `json:"raw,omitempty"`
}

// EventRaw records the outcome of copying the source transcript into the vault.
type EventRaw struct {
	RawMissing bool   `json:"raw_missing,omitempty"`
	CopyStatus string `json:"copy_status,omitempty"`
	RawRefID   string `json:"raw_ref_id,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Kind       string `json:"kind,omitempty"`
	SizeBytes  int64  `json:"size_bytes,omitempty"`
	CopyError  string `json:"copy_error,omitempty"`
}

// SessionRef identifies the captured source session.
type SessionRef struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// WorkspaceRef records the source workspace reported by the hook payload.
type WorkspaceRef struct {
	CWD string `json:"cwd"`
}

type captureError string

func (e captureError) Error() string { return string(e) }

// SpoolClaudeCodeHook reads stdin, archives the transcript if allowed, writes one
// capture event with O_EXCL, and updates vault state. captureTime must be an
// RFC3339 timestamp supplied by the caller when the payload omitted one.
func SpoolClaudeCodeHook(stdin io.Reader, captureTime string) (Event, error) {
	paths, err := workspace.Resolve()
	if err != nil {
		return Event{}, err
	}
	return SpoolClaudeCodeHookToStore(stdin, vault.New(paths), captureTime)
}

// SpoolClaudeCodeHookToStore is the store-injected form used by tests.
func SpoolClaudeCodeHookToStore(stdin io.Reader, store vault.Store, captureTime string) (Event, error) {
	payload, err := readClaudeCodeHookPayload(stdin)
	if err != nil {
		return Event{}, err
	}

	eventType, err := claudeCodeEventType(payload.HookEventName)
	if err != nil {
		return Event{}, err
	}

	event := newEvent(payload, eventType, captureTime)
	if eventType == "session_end" {
		event.Raw = eventRawForPayload(payload, store, event.Timestamp)
	}
	if err := writeEvent(store.Paths.EventsDir(), event); err != nil {
		return Event{}, err
	}
	if err := store.UpdateState(func(state *vault.State) {
		state.LastCaptureAt = event.Timestamp
		if event.Raw != nil {
			if event.Raw.RawMissing {
				state.RawMissingCount++
			}
			if event.Raw.CopyStatus == "failed" {
				state.CopyFailureCount++
			}
		}
	}); err != nil {
		return Event{}, err
	}
	return event, nil
}

// HookWarning returns the operator-visible degraded-capture warning for an event.
func HookWarning(event Event) string {
	if event.Raw == nil {
		return ""
	}
	if event.Raw.RawMissing {
		return "capture recorded without transcript_path; preservation degraded"
	}
	if event.Raw.CopyStatus == "failed" {
		return "capture recorded but transcript copy failed: " + event.Raw.CopyError
	}
	return ""
}

// ResolveHookTranscriptPath confines transcript_path to ~/.claude/projects, the
// hook process cwd, or the cwd reported by the hook payload.
func ResolveHookTranscriptPath(payload ClaudeCodeHookPayload) (string, error) {
	path := trimSpace(payload.TranscriptPath)
	if path == "" {
		return "", captureError("missing transcript_path")
	}

	roots := allowedTranscriptRoots(payload)
	if isAbsPath(path) {
		resolved := cleanPath(path)
		if !containedInAnyRoot(roots, resolved) {
			return "", captureError("transcript_path escapes allowed capture roots")
		}
		return resolved, nil
	}

	bases := candidateBases(payload)
	fallback := ""
	for _, base := range bases {
		resolved := cleanPath(joinPath(base, path))
		if !containedInAnyRoot(roots, resolved) {
			continue
		}
		if fallback == "" {
			fallback = resolved
		}
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			return resolved, nil
		}
	}
	if fallback == "" {
		return "", captureError("transcript_path escapes allowed capture roots")
	}
	return fallback, nil
}

// RawKindForTranscriptPath maps Claude transcript paths to raw vault kinds.
func RawKindForTranscriptPath(path string) string {
	if containsString(toSlash(path), "/subagents/") {
		return vault.KindSubagentTranscript
	}
	return vault.KindMainTranscript
}

func eventRawForPayload(payload ClaudeCodeHookPayload, store vault.Store, observedAt string) *EventRaw {
	if trimSpace(payload.TranscriptPath) == "" {
		return &EventRaw{RawMissing: true, CopyStatus: "missing"}
	}

	archivePath, err := ResolveHookTranscriptPath(payload)
	if err != nil {
		return &EventRaw{CopyStatus: "failed", CopyError: err.Error(), Kind: RawKindForTranscriptPath(payload.TranscriptPath)}
	}
	result, err := store.ArchiveFile(vault.ArchiveRequest{
		Source:          vault.SourceClaudeCode,
		SourceSessionID: payload.SessionID,
		Kind:            RawKindForTranscriptPath(archivePath),
		OriginalPath:    archivePath,
		ObservedAt:      observedAt,
	})
	if err != nil {
		return &EventRaw{CopyStatus: "failed", CopyError: err.Error(), Kind: RawKindForTranscriptPath(archivePath)}
	}
	status := "copied"
	if !result.BlobCreated {
		status = "deduped"
	}
	return &EventRaw{
		CopyStatus: status,
		RawRefID:   result.RawRef.RawRefID,
		Digest:     result.RawRef.Digest,
		Kind:       result.RawRef.Kind,
		SizeBytes:  result.RawRef.SizeBytes,
	}
}

func readClaudeCodeHookPayload(stdin io.Reader) (ClaudeCodeHookPayload, error) {
	data, err := io.ReadAll(io.LimitReader(stdin, MaxHookPayloadBytes+1))
	if err != nil {
		return ClaudeCodeHookPayload{}, captureError("read hook JSON: " + err.Error())
	}
	if len(data) > MaxHookPayloadBytes {
		return ClaudeCodeHookPayload{}, captureError("hook JSON exceeds 1048576 bytes")
	}
	if trimSpaceBytes(data) == "" {
		return ClaudeCodeHookPayload{}, captureError("missing hook JSON on stdin")
	}

	payload, err := parseHookPayload(data)
	if err != nil {
		return ClaudeCodeHookPayload{}, captureError("invalid hook JSON: " + err.Error())
	}
	payload.SessionID = trimSpace(payload.SessionID)
	payload.TranscriptPath = trimSpace(payload.TranscriptPath)
	payload.CWD = trimSpace(payload.CWD)
	payload.HookEventName = trimSpace(payload.HookEventName)
	payload.Timestamp = trimSpace(payload.Timestamp)

	if payload.SessionID == "" {
		return ClaudeCodeHookPayload{}, captureError("missing required hook field session_id")
	}
	if payload.CWD == "" {
		return ClaudeCodeHookPayload{}, captureError("missing required hook field cwd")
	}
	if payload.HookEventName == "" {
		return ClaudeCodeHookPayload{}, captureError("missing required hook field hook_event_name")
	}
	if payload.Timestamp != "" && !looksRFC3339(payload.Timestamp) {
		return ClaudeCodeHookPayload{}, captureError("invalid hook field timestamp: must be RFC3339")
	}
	return payload, nil
}

func claudeCodeEventType(hookEventName string) (string, error) {
	switch hookEventName {
	case "SessionStart", "session_start":
		return "session_start", nil
	case "SessionEnd", "session_end":
		return "session_end", nil
	default:
		return "", captureError("unsupported Claude Code hook_event_name \"" + hookEventName + "\"")
	}
}

func newEvent(payload ClaudeCodeHookPayload, eventType string, captureTime string) Event {
	timestamp := payload.Timestamp
	timestampSource := HookTimestampSourceHookPayload
	if timestamp == "" {
		timestamp = captureTime
		timestampSource = HookTimestampSourceCaptureTime
	}

	event := Event{
		SchemaVersion:   EventSchemaVersion,
		Source:          ClaudeCodeSource,
		EventType:       eventType,
		Timestamp:       timestamp,
		TimestampSource: timestampSource,
		SessionRef: SessionRef{
			SessionID:      payload.SessionID,
			TranscriptPath: payload.TranscriptPath,
		},
		Workspace: WorkspaceRef{CWD: payload.CWD},
	}
	event.EventID = deterministicEventID(event)
	return event
}

func deterministicEventID(event Event) string {
	hash := sha256.New()
	for _, part := range []string{
		event.Source,
		event.EventType,
		event.SessionRef.SessionID,
		event.SessionRef.TranscriptPath,
		event.Workspace.CWD,
	} {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return "evt_" + hexString(hash.Sum(nil))[:32]
}

func writeEvent(eventsDir string, event Event) error {
	if err := os.MkdirAll(eventsDir, 0o700); err != nil {
		return captureError("create event spool " + toSlash(eventsDir) + ": " + err.Error())
	}

	baseEventID := event.EventID
	for attempt := 1; ; attempt++ {
		eventID := baseEventID
		if attempt > 1 {
			eventID = baseEventID + "_" + intString(int64(attempt))
		}
		event.EventID = eventID
		path := joinPath(eventsDir, eventID+".json")
		// #nosec G304 -- capture event paths are derived from the resolved qratum workspace root and sanitized event ids.
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return captureError("create capture event " + toSlash(path) + ": " + err.Error())
		}
		data := marshalEvent(event)
		if err := file.Chmod(0o600); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return captureError("set capture event permissions: " + err.Error())
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return captureError("write capture event: " + err.Error())
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return captureError("close capture event: " + err.Error())
		}
		return nil
	}
}

func marshalEvent(event Event) []byte {
	out := []byte{'{', '\n'}
	first := true
	out, first = appendJSONField(out, first, "schema_version", event.SchemaVersion)
	out, first = appendJSONField(out, first, "event_id", event.EventID)
	out, first = appendJSONField(out, first, "source", event.Source)
	out, first = appendJSONField(out, first, "event_type", event.EventType)
	out, first = appendJSONField(out, first, "timestamp", event.Timestamp)
	if event.TimestampSource != "" {
		out, first = appendJSONField(out, first, "timestamp_source", event.TimestampSource)
	}
	out = appendComma(out, first)
	first = false
	out = append(out, []byte("  \"session_ref\": {")...)
	out = append(out, '\n')
	sf := true
	out, sf = appendJSONField(out, sf, "session_id", event.SessionRef.SessionID)
	if event.SessionRef.TranscriptPath != "" {
		out, _ = appendJSONField(out, sf, "transcript_path", event.SessionRef.TranscriptPath)
	}
	out = append(out, []byte("\n  }")...)
	out = appendComma(out, first)
	out = append(out, []byte("  \"workspace\": {")...)
	out = append(out, '\n')
	wf := true
	out, _ = appendJSONField(out, wf, "cwd", event.Workspace.CWD)
	out = append(out, []byte("\n  }")...)
	if event.Raw != nil {
		out = append(out, ',')
		out = append(out, '\n')
		out = append(out, []byte("  \"raw\": {")...)
		out = append(out, '\n')
		rf := true
		if event.Raw.RawMissing {
			out, rf = appendJSONBoolField(out, rf, "raw_missing", true)
		}
		if event.Raw.CopyStatus != "" {
			out, rf = appendJSONField(out, rf, "copy_status", event.Raw.CopyStatus)
		}
		if event.Raw.RawRefID != "" {
			out, rf = appendJSONField(out, rf, "raw_ref_id", event.Raw.RawRefID)
		}
		if event.Raw.Digest != "" {
			out, rf = appendJSONField(out, rf, "digest", event.Raw.Digest)
		}
		if event.Raw.Kind != "" {
			out, rf = appendJSONField(out, rf, "kind", event.Raw.Kind)
		}
		if event.Raw.SizeBytes != 0 {
			out, rf = appendJSONIntField(out, rf, "size_bytes", event.Raw.SizeBytes)
		}
		if event.Raw.CopyError != "" {
			out, _ = appendJSONField(out, rf, "copy_error", event.Raw.CopyError)
		}
		out = append(out, []byte("\n  }")...)
	}
	out = append(out, '\n', '}', '\n')
	return out
}

func appendJSONField(out []byte, first bool, key string, value string) ([]byte, bool) {
	out = appendComma(out, first)
	out = append(out, []byte("  ")...)
	out = appendJSONString(out, key)
	out = append(out, []byte(": ")...)
	out = appendJSONString(out, value)
	return out, false
}

func appendJSONBoolField(out []byte, first bool, key string, value bool) ([]byte, bool) {
	out = appendComma(out, first)
	out = append(out, []byte("  ")...)
	out = appendJSONString(out, key)
	out = append(out, []byte(": ")...)
	if value {
		out = append(out, []byte("true")...)
	} else {
		out = append(out, []byte("false")...)
	}
	return out, false
}

func appendJSONIntField(out []byte, first bool, key string, value int64) ([]byte, bool) {
	out = appendComma(out, first)
	out = append(out, []byte("  ")...)
	out = appendJSONString(out, key)
	out = append(out, []byte(": ")...)
	out = append(out, []byte(intString(value))...)
	return out, false
}

func appendComma(out []byte, first bool) []byte {
	if !first {
		out = append(out, ',')
	}
	out = append(out, '\n')
	return out
}

func appendJSONString(out []byte, value string) []byte {
	out = append(out, '"')
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch c {
		case '\\', '"':
			out = append(out, '\\', c)
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		default:
			if c < 0x20 {
				out = append(out, []byte("\\u00")...)
				out = append(out, hexDigit(c>>4), hexDigit(c&0x0f))
			} else {
				out = append(out, c)
			}
		}
	}
	out = append(out, '"')
	return out
}

type jsonParser struct {
	data []byte
	pos  int
}

func parseHookPayload(data []byte) (ClaudeCodeHookPayload, error) {
	p := &jsonParser{data: data}
	payload, err := p.parsePayloadObject()
	if err != nil {
		return ClaudeCodeHookPayload{}, err
	}
	p.skipSpace()
	if p.pos != len(p.data) {
		return ClaudeCodeHookPayload{}, captureError("trailing data")
	}
	return payload, nil
}

func (p *jsonParser) parsePayloadObject() (ClaudeCodeHookPayload, error) {
	var payload ClaudeCodeHookPayload
	p.skipSpace()
	if !p.take('{') {
		return payload, captureError("expected object")
	}
	p.skipSpace()
	if p.take('}') {
		return payload, nil
	}
	for {
		key, err := p.parseString()
		if err != nil {
			return payload, err
		}
		p.skipSpace()
		if !p.take(':') {
			return payload, captureError("expected colon")
		}
		if key == "session_id" || key == "transcript_path" || key == "cwd" || key == "hook_event_name" || key == "timestamp" {
			value, err := p.parseStringValue()
			if err != nil {
				return payload, err
			}
			switch key {
			case "session_id":
				payload.SessionID = value
			case "transcript_path":
				payload.TranscriptPath = value
			case "cwd":
				payload.CWD = value
			case "hook_event_name":
				payload.HookEventName = value
			case "timestamp":
				payload.Timestamp = value
			}
		} else if err := p.skipValue(); err != nil {
			return payload, err
		}
		p.skipSpace()
		if p.take('}') {
			return payload, nil
		}
		if !p.take(',') {
			return payload, captureError("expected comma")
		}
		p.skipSpace()
	}
}

func (p *jsonParser) parseStringValue() (string, error) {
	p.skipSpace()
	if p.peek() != '"' {
		return "", captureError("expected string")
	}
	return p.parseString()
}

func (p *jsonParser) parseString() (string, error) {
	p.skipSpace()
	if !p.take('"') {
		return "", captureError("expected string")
	}
	out := []byte{}
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		p.pos++
		if c == '"' {
			return string(out), nil
		}
		if c == '\\' {
			if p.pos >= len(p.data) {
				return "", captureError("unterminated escape")
			}
			esc := p.data[p.pos]
			p.pos++
			switch esc {
			case '"', '\\', '/':
				out = append(out, esc)
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'u':
				if p.pos+4 > len(p.data) || !isHex(p.data[p.pos]) || !isHex(p.data[p.pos+1]) || !isHex(p.data[p.pos+2]) || !isHex(p.data[p.pos+3]) {
					return "", captureError("invalid unicode escape")
				}
				out = append(out, '?')
				p.pos += 4
			default:
				return "", captureError("invalid escape")
			}
			continue
		}
		if c < 0x20 {
			return "", captureError("control character in string")
		}
		out = append(out, c)
	}
	return "", captureError("unterminated string")
}

func (p *jsonParser) skipValue() error {
	p.skipSpace()
	switch p.peek() {
	case '"':
		_, err := p.parseString()
		return err
	case '{':
		p.pos++
		p.skipSpace()
		if p.take('}') {
			return nil
		}
		for {
			if _, err := p.parseString(); err != nil {
				return err
			}
			p.skipSpace()
			if !p.take(':') {
				return captureError("expected colon")
			}
			if err := p.skipValue(); err != nil {
				return err
			}
			p.skipSpace()
			if p.take('}') {
				return nil
			}
			if !p.take(',') {
				return captureError("expected comma")
			}
			p.skipSpace()
		}
	case '[':
		p.pos++
		p.skipSpace()
		if p.take(']') {
			return nil
		}
		for {
			if err := p.skipValue(); err != nil {
				return err
			}
			p.skipSpace()
			if p.take(']') {
				return nil
			}
			if !p.take(',') {
				return captureError("expected comma")
			}
			p.skipSpace()
		}
	case 't':
		return p.takeLiteral("true")
	case 'f':
		return p.takeLiteral("false")
	case 'n':
		return p.takeLiteral("null")
	default:
		return p.skipNumber()
	}
}

func (p *jsonParser) skipNumber() error {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
	}
	for isDigit(p.peek()) {
		p.pos++
	}
	if p.peek() == '.' {
		p.pos++
		for isDigit(p.peek()) {
			p.pos++
		}
	}
	if p.peek() == 'e' || p.peek() == 'E' {
		p.pos++
		if p.peek() == '+' || p.peek() == '-' {
			p.pos++
		}
		for isDigit(p.peek()) {
			p.pos++
		}
	}
	if p.pos == start {
		return captureError("expected value")
	}
	return nil
}

func (p *jsonParser) takeLiteral(value string) error {
	if p.pos+len(value) > len(p.data) {
		return captureError("expected " + value)
	}
	for i := 0; i < len(value); i++ {
		if p.data[p.pos+i] != value[i] {
			return captureError("expected " + value)
		}
	}
	p.pos += len(value)
	return nil
}

func (p *jsonParser) skipSpace() {
	for p.pos < len(p.data) && isSpace(p.data[p.pos]) {
		p.pos++
	}
}

func (p *jsonParser) take(c byte) bool {
	if p.pos < len(p.data) && p.data[p.pos] == c {
		p.pos++
		return true
	}
	return false
}

func (p *jsonParser) peek() byte {
	if p.pos >= len(p.data) {
		return 0
	}
	return p.data[p.pos]
}

func allowedTranscriptRoots(payload ClaudeCodeHookPayload) []string {
	roots := []string{}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		roots = append(roots, cleanPath(wd))
	}
	if cwd := trimSpace(payload.CWD); cwd != "" {
		roots = append(roots, absoluteFromProcessCWD(cwd))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots, cleanPath(joinPath(home, ".claude/projects")))
	}
	return uniqueNonEmptyStrings(roots)
}

func candidateBases(payload ClaudeCodeHookPayload) []string {
	bases := []string{}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		bases = append(bases, cleanPath(wd))
	}
	if cwd := trimSpace(payload.CWD); cwd != "" {
		bases = append(bases, absoluteFromProcessCWD(cwd))
	}
	return uniqueNonEmptyStrings(bases)
}

func absoluteFromProcessCWD(path string) string {
	path = trimSpace(path)
	if isAbsPath(path) {
		return cleanPath(path)
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return cleanPath(joinPath(wd, path))
	}
	return cleanPath(path)
}

func containedInAnyRoot(roots []string, path string) bool {
	for _, root := range roots {
		if pathContainedIn(root, path) {
			return true
		}
	}
	return false
}

func pathContainedIn(root string, path string) bool {
	root = cleanPath(root)
	path = cleanPath(path)
	if root == path {
		return true
	}
	if root == "/" {
		return isAbsPath(path)
	}
	return hasPrefix(path, root+"/")
}

func cleanPath(path string) string {
	path = trimSpace(toSlash(path))
	if path == "" {
		return "."
	}
	abs := isAbsPath(path)
	parts := []string{}
	start := 0
	for start <= len(path) {
		end := start
		for end < len(path) && path[end] != '/' {
			end++
		}
		part := path[start:end]
		if part != "" && part != "." {
			if part == ".." {
				if len(parts) > 0 && parts[len(parts)-1] != ".." {
					parts = parts[:len(parts)-1]
				} else if !abs {
					parts = append(parts, part)
				}
			} else {
				parts = append(parts, part)
			}
		}
		if end == len(path) {
			break
		}
		start = end + 1
	}
	joined := joinParts(parts)
	if abs {
		if joined == "" {
			return "/"
		}
		return "/" + joined
	}
	if joined == "" {
		return "."
	}
	return joined
}

func joinPath(base string, path string) string {
	if isAbsPath(path) {
		return cleanPath(path)
	}
	if base == "" || base == "." {
		return cleanPath(path)
	}
	return cleanPath(base + "/" + path)
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "/" + parts[i]
	}
	return out
}

func isAbsPath(path string) bool {
	path = toSlash(path)
	return len(path) > 0 && path[0] == '/'
}

func toSlash(path string) string {
	out := []byte(path)
	for i, c := range out {
		if c == '\\' {
			out[i] = '/'
		}
	}
	return string(out)
}

func uniqueNonEmptyStrings(values []string) []string {
	out := []string{}
	for _, value := range values {
		if value == "" || value == "." {
			continue
		}
		seen := false
		for _, existing := range out {
			if existing == value {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, value)
		}
	}
	return out
}

func looksRFC3339(value string) bool {
	if len(value) < len("2006-01-02T15:04:05Z") {
		return false
	}
	if !isNDigits(value, 0, 4) || value[4] != '-' || !isNDigits(value, 5, 2) || value[7] != '-' || !isNDigits(value, 8, 2) {
		return false
	}
	if value[10] != 'T' && value[10] != 't' {
		return false
	}
	if !isNDigits(value, 11, 2) || value[13] != ':' || !isNDigits(value, 14, 2) || value[16] != ':' || !isNDigits(value, 17, 2) {
		return false
	}
	pos := 19
	if pos < len(value) && value[pos] == '.' {
		pos++
		start := pos
		for pos < len(value) && isDigit(value[pos]) {
			pos++
		}
		if pos == start {
			return false
		}
	}
	if pos == len(value)-1 && (value[pos] == 'Z' || value[pos] == 'z') {
		return true
	}
	if pos+6 == len(value) && (value[pos] == '+' || value[pos] == '-') && isNDigits(value, pos+1, 2) && value[pos+3] == ':' && isNDigits(value, pos+4, 2) {
		return true
	}
	return false
}

func isNDigits(value string, start int, count int) bool {
	if start+count > len(value) {
		return false
	}
	for i := 0; i < count; i++ {
		if !isDigit(value[start+i]) {
			return false
		}
	}
	return true
}

func trimSpaceBytes(value []byte) string { return trimSpace(string(value)) }

func trimSpace(value string) string {
	start := 0
	for start < len(value) && isSpace(value[start]) {
		start++
	}
	end := len(value)
	for end > start && isSpace(value[end-1]) {
		end--
	}
	return value[start:end]
}

func containsString(value string, needle string) bool {
	if needle == "" {
		return true
	}
	if len(needle) > len(value) {
		return false
	}
	for i := 0; i <= len(value)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if value[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func hasPrefix(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func isSpace(c byte) bool { return c == ' ' || c == '\n' || c == '\r' || c == '\t' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isHex(c byte) bool   { return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') }

func intString(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	if negative {
		digits = append(digits, '-')
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

func hexString(bytes []byte) string {
	out := make([]byte, len(bytes)*2)
	for i, b := range bytes {
		out[i*2] = hexDigit(b >> 4)
		out[i*2+1] = hexDigit(b & 0x0f)
	}
	return string(out)
}

func hexDigit(value byte) byte {
	if value < 10 {
		return '0' + value
	}
	return 'a' + value - 10
}

// Package source defines Wave 1 source records and strict transcript parsers.
package source

// Stored schema versions, supported source names, and data classes used by Wave 1.
const (
	CaptureEventSchemaVersion         = "qratum.capture_event.v2"
	SessionRevisionSchemaVersion      = "qratum.session_revision.v1"
	UsageRecordSchemaVersion          = "qratum.usage_record.v1"
	SessionTombstoneSchemaVersion     = "qratum.session_tombstone.v1"
	CaptureStateSchemaVersion         = "qratum.capture_state.v1"
	PriceCatalogManifestSchemaVersion = "qratum.price_catalog_manifest.v1"

	SourceClaudeCode = "claude-code"
	SourceCodex      = "codex"

	DataClassRaw       = "raw"
	DataClassPublished = "published"
)

// RecordProvenance identifies the code that interpreted a source record.
type RecordProvenance struct {
	Adapter        string `json:"adapter"`
	AdapterVersion string `json:"adapter_version"`
}

// GitObservation records bounded repository facts captured with a source event.
type GitObservation struct {
	Status          string `json:"status"`
	RepositoryRoot  string `json:"repository_root,omitempty"`
	CommonDirectory string `json:"common_directory,omitempty"`
	Branch          string `json:"branch,omitempty"`
	HeadCommit      string `json:"head_commit,omitempty"`
	SanitizedRemote string `json:"sanitized_remote,omitempty"`
	FailureReason   string `json:"failure_reason,omitempty"`
}

// HarnessMarker is accepted only when a harness explicitly supplies every field.
type HarnessMarker struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	RunID      string `json:"run_id"`
	ObservedAt string `json:"observed_at"`
}

// CaptureEvent is the small owner-only event written by a source hook.
type CaptureEvent struct {
	SchemaVersion  string           `json:"schema_version"`
	DataClass      string           `json:"data_class"`
	EventID        string           `json:"event_id"`
	Source         string           `json:"source"`
	SourceVersion  string           `json:"source_version"`
	EventType      string           `json:"event_type"`
	RootSessionID  string           `json:"root_session_id"`
	AgentID        string           `json:"agent_id,omitempty"`
	AgentType      string           `json:"agent_type,omitempty"`
	TranscriptKind string           `json:"transcript_kind"`
	TranscriptPath string           `json:"transcript_path"`
	CWD            string           `json:"cwd"`
	Model          string           `json:"model,omitempty"`
	EventTime      string           `json:"event_time,omitempty"`
	ObservedAt     string           `json:"observed_at"`
	Git            GitObservation   `json:"git"`
	Harness        *HarnessMarker   `json:"harness,omitempty"`
	Provenance     RecordProvenance `json:"provenance"`
}

// FileFacts are checked before and after a stable streaming copy.
type FileFacts struct {
	Identity   string `json:"identity"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at"`
}

// SessionRevision maps a logical stream and exact digest to a preserved raw ref.
type SessionRevision struct {
	SchemaVersion  string    `json:"schema_version"`
	DataClass      string    `json:"data_class"`
	Source         string    `json:"source"`
	SourceVersion  string    `json:"source_version"`
	RootSessionID  string    `json:"root_session_id"`
	StreamID       string    `json:"stream_id"`
	StreamKind     string    `json:"stream_kind"`
	AgentID        string    `json:"agent_id,omitempty"`
	Digest         string    `json:"digest"`
	RawRefID       string    `json:"raw_ref_id"`
	Revision       int64     `json:"revision"`
	SourcePath     string    `json:"source_path"`
	Before         FileFacts `json:"before"`
	After          FileFacts `json:"after"`
	ObservedAt     string    `json:"observed_at"`
	PreservedAt    string    `json:"preserved_at"`
	AdapterVersion string    `json:"adapter_version"`
	ParserVersion  string    `json:"parser_version"`
	State          string    `json:"state"`
}

// TokenCounts keeps token classes separate so later pricing does not guess.
type TokenCounts struct {
	Input                int64 `json:"input"`
	Output               int64 `json:"output"`
	CacheRead            int64 `json:"cache_read"`
	CacheCreation        int64 `json:"cache_creation"`
	CacheCreationFiveMin int64 `json:"cache_creation_5m"`
	CacheCreationOneHour int64 `json:"cache_creation_1h"`
	ReasoningOutput      int64 `json:"reasoning_output"`
	Total                int64 `json:"total"`
}

// UsageRecord is one accepted incremental source usage record.
type UsageRecord struct {
	SchemaVersion          string      `json:"schema_version"`
	DataClass              string      `json:"data_class"`
	UsageID                string      `json:"usage_id"`
	Source                 string      `json:"source"`
	SourceVersion          string      `json:"source_version"`
	RootSessionID          string      `json:"root_session_id"`
	StreamID               string      `json:"stream_id"`
	SourceEventID          string      `json:"source_event_id"`
	TurnID                 string      `json:"turn_id,omitempty"`
	MessageID              string      `json:"message_id,omitempty"`
	Model                  string      `json:"model"`
	Provider               string      `json:"provider"`
	Tokens                 TokenCounts `json:"tokens"`
	TotalBasis             string      `json:"total_basis"`
	OccurredAt             string      `json:"occurred_at"`
	TimeBasis              string      `json:"time_basis"`
	Semantics              string      `json:"semantics"`
	Reliability            string      `json:"reliability"`
	EvidenceRevisionDigest string      `json:"evidence_revision_digest"`
	DuplicateStatus        string      `json:"duplicate_status"`
	ReconciliationStatus   string      `json:"reconciliation_status"`
	CounterEpoch           int64       `json:"counter_epoch"`
}

// RemovalCounts reports which Wave 1 representations deletion removed.
type RemovalCounts struct {
	Blobs     int64 `json:"blobs"`
	RawRefs   int64 `json:"raw_refs"`
	Revisions int64 `json:"revisions"`
	Usage     int64 `json:"usage"`
	Events    int64 `json:"events"`
	State     int64 `json:"state"`
	Temporary int64 `json:"temporary"`
}

// SessionTombstone prevents a deleted source session from being gathered again.
type SessionTombstone struct {
	SchemaVersion         string        `json:"schema_version"`
	DataClass             string        `json:"data_class"`
	Source                string        `json:"source"`
	SessionIdentityDigest string        `json:"session_identity_digest"`
	ErasedAt              string        `json:"erased_at"`
	RemovalStatus         string        `json:"removal_status"`
	RemovalCounts         RemovalCounts `json:"removal_counts"`
}

// CaptureSourceState reports truthful capture health for one source.
type CaptureSourceState struct {
	Source                 string   `json:"source"`
	SourceVersion          string   `json:"source_version,omitempty"`
	Status                 string   `json:"status"`
	ConfiguredHookEvents   []string `json:"configured_hook_events"`
	HookTrustState         string   `json:"hook_trust_state"`
	LastHookEventAt        string   `json:"last_hook_event_at,omitempty"`
	PendingEventCount      int64    `json:"pending_event_count"`
	OldestPendingEventAt   string   `json:"oldest_pending_event_at,omitempty"`
	LastRefreshStartedAt   string   `json:"last_refresh_started_at,omitempty"`
	LastRefreshFinishedAt  string   `json:"last_refresh_finished_at,omitempty"`
	LastRefreshResult      string   `json:"last_refresh_result,omitempty"`
	LastSnapshotAt         string   `json:"last_snapshot_at,omitempty"`
	RootStreamCount        int64    `json:"root_stream_count"`
	ChildStreamCount       int64    `json:"child_stream_count"`
	SupportedRecordCount   int64    `json:"supported_record_count"`
	UnsupportedRecordCount int64    `json:"unsupported_record_count"`
	UsageCoverage          string   `json:"usage_coverage"`
	UsageReconciliation    string   `json:"usage_reconciliation"`
	ScheduleState          string   `json:"schedule_state"`
	Warnings               []string `json:"warnings"`
}

// StreamCaptureState records retry and preservation state for one logical stream.
type StreamCaptureState struct {
	Source          string `json:"source"`
	RootSessionID   string `json:"root_session_id"`
	StreamID        string `json:"stream_id"`
	State           string `json:"state"`
	LatestDigest    string `json:"latest_digest,omitempty"`
	LastObservedAt  string `json:"last_observed_at,omitempty"`
	LastPreservedAt string `json:"last_preserved_at,omitempty"`
	RetryCount      int64  `json:"retry_count"`
	LastFailure     string `json:"last_failure,omitempty"`
}

// CaptureState is the file-backed source and stream status used by doctor.
type CaptureState struct {
	SchemaVersion string               `json:"schema_version"`
	DataClass     string               `json:"data_class"`
	UpdatedAt     string               `json:"updated_at"`
	Sources       []CaptureSourceState `json:"sources"`
	Streams       []StreamCaptureState `json:"streams"`
}

// PriceCatalogManifest identifies an immutable bundled, fetched, or imported catalog.
type PriceCatalogManifest struct {
	SchemaVersion    string `json:"schema_version"`
	DataClass        string `json:"data_class"`
	CatalogID        string `json:"catalog_id"`
	Upstream         string `json:"upstream"`
	ResolvedCommit   string `json:"resolved_commit,omitempty"`
	Digest           string `json:"digest"`
	RetrievedAt      string `json:"retrieved_at"`
	RetrievalMethod  string `json:"retrieval_method"`
	SourceURL        string `json:"source_url,omitempty"`
	Currency         string `json:"currency"`
	EffectiveAt      string `json:"effective_at,omitempty"`
	TransformVersion string `json:"transform_version"`
	EntryCount       int64  `json:"entry_count"`
}

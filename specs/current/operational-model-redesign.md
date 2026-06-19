# Qratum Operational Model v2

Status: stale base architecture reference. Do not implement directly.

Supersession: `specs/current/ui-first-onboarding.md` is authoritative for
first-run onboarding, the local app, the public `qrt` command contract, export
gates, background behavior, and AI boundaries. Where this document conflicts
with that spec, `ui-first-onboarding.md` wins.

## Plain-language glossary

These words show up throughout. Here is what they mean in everyday terms.

- **Raw / transcript:** the original conversation file a source tool wrote
  (for example, a Claude Code session log). It can contain anything the user
  typed or the agent saw.
- **Vault / raw archive:** Qratum's safe local copy of those raw files, kept so
  they survive even after the source tool deletes them.
- **Redaction:** removing or masking secrets (API keys, tokens, etc.) from data
  before it can move anywhere riskier.
- **Blob:** a single stored file, named by its content hash so identical files
  are stored once. "World-readable blob" means a stored file other users on the
  machine could open.
- **Digest / hash (sha256):** a fingerprint of some bytes. Same bytes give the
  same fingerprint; any change gives a different one.
- **Content-addressed:** stored and looked up by that fingerprint, not by a
  filename or path.
- **Idempotent (runs every time without making changes):** you can run it again
  and again and it does not duplicate work or cause new side effects.
- **Monotonic:** only moves in one direction; never silently goes backward.
- **Data class:** a sensitivity label on a piece of data (raw, redacted,
  review, metrics, lesson, corpus, publishable, published).
- **Consent:** a recorded human approval for a sensitive action.
- **Provenance:** the recorded history of how an object was produced — which
  inputs, which transform versions, which model.
- **ADP:** an export format family (ADP-style JSONL) Qratum can write.
- **DTO (data transfer object):** a clean shaped object the UI consumes, instead
  of Qratum's raw internal pipeline objects.
- **Source adapter:** the code that knows how to read and normalize one specific
  source tool (Claude Code is the first one).
- **Facet:** an AI- or classifier-assigned label on a session (its goal,
  outcome, friction, and so on).
- **Signal:** a low-level observation (deterministic or AI-generated) that may
  feed findings, metrics, lessons, or corpus scoring.
- **Finding:** a concrete, user-relevant issue or observation with evidence.
- **Corpus:** cleaned, scored, provenance-bound trajectory data prepared for
  export to some downstream consumer.
- **Trajectory:** the ordered sequence of what happened in a session
  (messages plus tool calls).
- **Recall / precision (used for the redaction benchmark):** recall is "of all
  the secrets present, how many did we catch"; precision is "of everything we
  flagged, how much was actually a secret."
- **Planted-secret corpus:** a test set seeded with known secrets so we can
  measure how many the redactor catches (recall).
- **TOCTOU (time-of-check to time-of-use):** a bug class where something changes
  between when you check it and when you act on it.

## One-Sentence Product

Qratum is a local-first AI session librarian. It preserves your AI coding
sessions first, then offers review, search, and curation on demand — only when
those features earn their keep.

The first user is the person running Qratum on their own machine. The first
valuable loop is:

```txt
capture session -> preserve it safely -> recover it after the source tool deletes it
```

Qratum should not be a git checkpoint system or a raw pile with no refinery,
but it also should not assume review, search, lessons, or corpus work before
the vault exists.

Qratum's input is bounded AI coding sessions. It can later produce
session-derived lessons, but preservation and browseable session history come
first.

## Locked Product Decisions

These are the decisions already taken. They were unlocked and rewritten on
2026-06-14 after the accepted vault-first review.

For context, here is what the machine actually looked like when re-measured at
2026-06-14T15:33:55Z: repo-local capture had 80 event files; 43 of the
transcript paths they referenced were already missing; and
`~/.claude/projects/**/*.jsonl` held 305 local transcripts totaling about 99.1
MB. These are volatile observations, not targets. They are here because they
justify the "preserve first" priority.

- Long-term shape: local-first personal librarian for AI session data, with
  on-demand refinery work that must earn itself.
- First user: the developer using it on their own AI coding sessions.
- Day-one value: preserve transcripts before source tools delete them, then
  make the preserved archive inspectable and reusable.
- Output priority:
  1. Preservation.
  2. Lessons-to-memory.
  3. Insights-harvest.
  4. Search.
  5. Review.
  6. Corpus.
- Primary onboarding surface: local Qratum app opened by the CLI. The CLI
  bootstraps, reports status, imports external paths, and handles recovery.
- CLI role: init, open, status, doctor, import, sessions, session detail, and
  explicit export. Lower-level
  hook/vault/backfill/pipeline commands should be removed from the public
  runtime as the onboarding replacements land.
- State location: central user workspace under `~/.qratum`.
- Human config: one autogenerated TOML file.
- Runtime data: JSON/JSONL with versioned schemas.
- Source data safety: raw data stays local. External services and shareable
  outputs may use only prepared/redacted artifacts unless a future accepted
  contract explicitly changes that rule.
- Architecture style: lightweight DDD plus ports and adapters.
- Filesystem object store is the source of truth.
- SQLite-backed search is the default local projection when search lands, but
  it becomes Qratum's first third-party Go runtime dependency and therefore
  needs an explicit supply-chain decision under `docs/supply-chain.md`.
- Review, insights, and corpus stay on-demand or future until real pull
  exists.
- Normal UX only shows sources that are detected and usable.
- No generic JSONL source. Build real source adapters when needed.
- Remote/private-perimeter paths are future scope, not part of the personal
  MVP.

## What The Current Milestone A Proves

This section sorts the existing Milestone A work into things worth keeping and
things to rebuild.

Keep:

- Go single binary.
- Fixture-driven tests and golden outputs.
- Fast hook that captures a small event and exits.
- Claude Code transcript parser as the first source adapter implementation.
- Deterministic local redaction as the first redactor.
- Deterministic evidence checks as the first signal extractors.
- ADP strict export as one export boundary.
- UI DTO idea: product surfaces consume DTOs, not raw internal pipeline objects.

Redo:

- Public CLI is too pipeline-shaped.
- `dogfood` is the wrong product noun.
- Artifact paths are too coupled to source session IDs.
- Generic session validation is Claude-only.
- `pipeline_status` belongs in job state, not in the session object.
- The current report is safe but not useful as the main product surface.
- Schemas are too thin for extensibility.
- Source capture, import, storage, review, lessons, and publishing need explicit
  contracts.

## Core Primitives

These are the building-block objects Qratum works with. Each line names the
object and says, in plain terms, what it represents.

- Workspace: the local Qratum environment under `~/.qratum`.
- Source: an AI coding tool or import format.
- Capture: a small event or import record that points to raw source data.
- RawRef: a local-only reference to raw data, optionally with an archived copy.
- Session: Qratum's normalized source-neutral trajectory.
- SessionRevision: a versioned normalized view of a source session at an
  observed point in time.
- SessionMeta: deterministic, source-neutral metadata and metrics derived from
  a session revision.
- Facet: AI-assisted or classifier-assisted labels over a session revision:
  goal, outcome, friction, satisfaction, usefulness, and reusable instructions.
- Review: a review envelope over a session revision or full source session.
- Finding: a concrete issue or observation with evidence and provenance.
- Signal: a deterministic or AI-generated observation that may feed findings,
  metrics, lessons, or corpus scoring.
- Insight: cross-session analysis built from session metadata, facets, metrics,
  and summaries.
- Lesson: curated reusable context, habit, rule, or session-derived lesson.
- CorpusItem: cleaned, scored, provenance-bound trajectory data for export.
- Artifact: generated file such as report, export, bundle, or manifest.
- Publisher: destination connector for approved bundles.
- PublishBundle: approved delivery package for a publisher.
- Consent: recorded approval for a sensitive action.
- Tombstone: safe deletion marker without raw/redacted content.
- Failure: structured failure record with retry and recovery metadata.
- Job: worker-owned processing state. Jobs are operational state, not product
  lifecycle objects.
- Provenance: input digests, transform versions, scorer versions, and data class
  history.

Object hierarchy (which object sits on top of which):

```txt
SessionRevision = normalized source data
SessionMeta = deterministic metadata
Signal = low-level observation
Finding = user-relevant observation
Review = envelope over findings, metrics, and signals
Facet = AI/classifier label
Insight = cross-session analysis
Lesson = curated reusable lesson/context
CorpusCandidate = candidate for future export
CorpusItem = finalized item inside an export
Artifact = managed generated file
PublishBundle = approved delivery package
Job = operational work/retry state
```

UI language can be simpler than internal object names. For example, `Facet`
can appear as "labels" and `Signal` can appear as "detected evidence".

## Architecture Style

In short: keep the core business objects clean and stable, and push everything
that talks to the outside world behind clear boundaries.

Qratum uses lightweight DDD plus ports and adapters.

Rules:

- Domain objects are stable and infrastructure-independent.
- Use cases orchestrate domain behavior.
- Ports define real external boundaries.
- Adapters implement ports.
- Interfaces exist at boundaries where implementations are expected to vary.
- Internal orchestration can use concrete types.
- Versioned JSON/JSONL schemas are the cross-process contracts.
- Adapters must have fixture or contract tests.
- Do not introduce abstractions until the boundary is real.

## Schema Registry

The contracts that different processes rely on live as schema files. This
section lists them and flags which names ship today versus which are the target.

Qratum's cross-process contracts live in `schemas/`. Go types may be handwritten
or generated, but JSON Schema files are the source of truth for stored objects,
exports, fixtures, and adapter contracts.

Current shipped schema layout:

```txt
schemas/
  qratum-event.v1.schema.json
  qratum-session.v1.schema.json
  qratum-review-card.v1.schema.json
  qratum-evidence.v1.schema.json
  qratum-raw-ref.v1.schema.json
  qratum-provenance.v1.schema.json
  ui/
```

> **Reconciliation notes (2026-06-15):** the list above is what is on disk
> today. The earlier dot-named target layout (`qratum.session.v1.schema.json`,
> `qratum.raw_ref.v1.schema.json`, etc.) is still target design, not shipped
> reality. The rename/migration is P0 contract work tracked by the
> schema-naming-and-validator decision (`verification-and-trust-gate.md` D9),
> which also adds the missing `additionalProperties:false` and a validator.
> - `qratum.lesson.v1`, `qratum.publish_manifest.v1`, and any
>   `memory_export`/`memory_bundle` schema are **deferred/dead** per
>   `qratum-vault-first.md` §Dead — do not author them at P0.
>   `qratum.consent.v1` is **future-only** (see §Consent Records below).

Rules:

```txt
Every persisted object declares schema_version.
Every schema has fixture examples and validation tests.
Migration plans reference source and target schema versions.
UI DTOs have their own schemas when they become stable contracts.
```

Use cases (the named operations Qratum performs):

```txt
SetupWorkspace
CaptureSourceEvent
InventorySource
ImportSource
ArchiveRaw
NormalizeSession
ReviewRevision
ExtractFacets
GenerateInsights
StageLesson
ExportCorpus
PublishBundle
DeleteObject
ApplyRetention
RetryJob
RepairObject
```

Ports (the boundaries to the outside world):

```txt
ObjectStore
SourceAdapter
RawArchive
Redactor
SignalDetector
ReviewEngine
AIProvider
SearchBackend
Renderer
Exporter
Publisher
PolicyEvaluator
JobQueue
Clock
IDGenerator
```

Expected adapters across milestones (the concrete implementations behind those
ports):

```txt
FilesystemObjectStore
ContentAddressedRawArchive
ClaudeCodeSourceAdapter
DeterministicRedactor
DeterministicReviewEngine
OpenAICompatibleLocalProvider
OpenRouterProvider
SQLiteSearchBackend
HTMLRenderer
ADPExporter
LocalFolderPublisher
```

## Object Lifecycle

This section traces one session from capture through to a published bundle, and
spells out which steps are automatic versus gated.

Product lifecycle (the path data travels):

```txt
Capture
  -> RawRef
  -> Session
  -> SessionRevision
  -> SessionMeta
  -> Signals
  -> Review
  -> Findings
  -> Facets
  -> Insights
  -> Lesson candidates
  -> Corpus candidates
  -> Artifacts
  -> Publish bundles
```

Transition rules (what triggers each step, and whether it needs approval):

```txt
Capture -> RawRef
  automatic, unless raw archive policy blocks it

RawRef -> Session/SessionRevision
  automatic normalization

SessionRevision -> SessionMeta
  automatic deterministic extraction

SessionRevision -> Signals
  automatic deterministic checks

Signals + SessionMeta -> Review/Findings
  automatic for local deterministic review

SessionRevision/Review/SessionMeta -> Facets
  AI/classifier gated by policy and consent

Facets + SessionMeta + Review -> Insights
  manual or scheduled, async only

Facets/Review -> Lesson candidates
  AI-gated when AI is used; stored locally

Review/SessionRevision -> Corpus candidates
  automatic candidate marking is allowed; export requires explicit approval

Review/Insight/Corpus -> Artifacts
  automatic local rendering/export after explicit action or configured policy

Artifacts -> Publish bundles
  manual approval first
```

Jobs track work on transitions. They are not the product lifecycle. A failed
job can be retried without rerunning the whole import or pipeline.

## Central Workspace Layout

Everything Qratum manages lives in one folder under the user's home directory.
Users should not have to understand the internal folder tree during onboarding.
The product shows simple states: found, preserved, prepared, open.

All Qratum-managed state lives under the user's workspace.

User-facing workspace summary:

```txt
~/.qratum/
  config.toml
  raw archive
  prepared sessions
  local app/search state
  backups/exports when explicitly requested
```

Internal target layout:

```txt
~/.qratum/
  config.toml
  config.schema.json
  state/
    workspace.json
    repos.json
    sources.json
    app_auth.json
    consent/
    tombstones/
    failures/
  captures/
    pending/
    running/
    complete/
    failed/
  raw/
    blobs/
      sha256/
    refs/
  jobs/
    pending/
    running/
    complete/
    failed/
  sessions/
  derived/
    session_meta/
    facets/
    aggregates/
    ai_cache/
  reviews/
  insights/
  lessons/
  corpus/
  artifacts/
    reports/
    exports/
    bundles/
  publish/
    outbox/
    history/
  indexes/
    qratum.sqlite
  backend/
  observability/
    events.jsonl
    metrics.json
  manifests/
    migrations/
```

> **Shipped vs target (2026-06-15):** the tree above is the *full v2* layout. The
> shipped vault-minimum workspace is much smaller — only `raw/blobs/`,
> `raw/refs/`, `events/`, and `state/vault.json` (per `qratum-vault-first.md`).
> UI-first onboarding does not require a resident daemon. If a local service is
> accepted later, its pid/lock/log state must be added under a service-specific
> path instead of making the first app depend on `daemon/`. Note: derived
> refinery artifacts currently land in repo-local `./.qratum/`, not under
> `~/.qratum/` — moving them here is target state tracked by the
> centralize-workspace fix (trust-gate D11/FIX-4).

Repositories are metadata and context only. Qratum stores repo path, branch,
remote, head SHA, working tree status, and source discovery hints centrally.

If a user exports or publishes a bundle into a folder, bucket, endpoint, or
repository, that destination is a publish target, not Qratum working state.

## Config Shape

There is one config file, and it is generated for the user, not hand-written.
This section shows the target full config and flags which blocks are not
day-one defaults.

`~/.qratum/config.toml` is autogenerated on first use. First use can be `qrt`,
`qrt init`, `qrt import`, or `qrt open`.

The generated file should be understandable in 30 seconds. It should contain
only common options and detected values. Advanced options are discoverable
through CLI help, the local app settings page, and `config.schema.json`.

Example generated personal config:

> **UI-first onboarding update (2026-06-17):** the example below shows target
> config ideas only. The local app is now the accepted onboarding direction, but
> the runtime implementation is not shipped yet. SQLite/search is not an
> onboarding default and still needs separate acceptance. Background worker
> behavior remains opt-in.
>
> `qrt init` should generate the file with all stable user-facing options
> populated, using defaults derived from the user's answers. Users should not
> have to guess which keys exist.

```toml
[raw]
archive = true
retention = "forever"

[sources]
claude_code = true

[ai]
local = true
external = "ask"

[ai.local]
endpoint = "http://localhost:1234/v1"
model = ""

[ai.openrouter]
enabled = false
model = ""
api_key_env = "OPENROUTER_API_KEY"

# SQLite/search is future scope, not generated by onboarding by default.
# [backend]
# mode = "sqlite"


[app]
host = "127.0.0.1"
port = 9473
idle_timeout = "30m"
raw_routes = false

[observability]
enabled = true
exporter = "local"

[observability.otlp]
enabled = false
endpoint = ""

[worker]
max_jobs = 4
max_ai_jobs = 1
disk_free_min_gb = 10

[publish]
mode = "manual"

[publish.local_folder]
path = "~/QratumPublished"
```

Config best practices:

- One primary file.
- Strong defaults.
- Generate all stable user-facing options with explicit defaults.
- No required policy language for personal use.
- No plain-text API keys if avoidable. Prefer environment variables first, OS
  keychain later.
- Backend connection strings should use environment variables by default.
- `qrt config describe` explains every supported key.
- `qrt config set <key> <value>` edits simple values safely.
- `qrt config edit` opens the config file directly for users who prefer editing
  the file.
- Invalid config errors point to exact keys and expected values.

Useful config commands:

```txt
qrt config
qrt config edit
qrt config get raw.archive
qrt config set raw.archive true
qrt config describe
qrt config describe ai.external
```

Backend modes:

```txt
sqlite
  default local projection for search and local library views

none
  debug/emergency mode; no search backend, basic file-backed views only
```

## Raw Archive And Retention

The point here is simple: copy the original transcripts somewhere safe before
the source tool can delete them, and never throw those copies away unless the
user says so.

Compression:

- The archive records the original uncompressed sha256 digest.
- Compression may be added as a storage optimization, but it must not change the
  identity of the raw object.
- If compression is enabled, Qratum stores enough metadata to restore the exact
  original bytes and verify them against the original digest.
- Compression should be a storage policy, not something the user has to
  understand during onboarding.

`qrt init` asks whether to archive raw transcripts. The default answer can be
references only, but the personal configuration should support raw copy.

When raw archive is enabled:

- Copy raw source files as soon as possible after session end.
- Archive main transcripts, subagent transcripts, file-history snapshots, and
  source-generated insight reports when available.
- Store forever unless manually deleted.
- Use content-addressed blobs plus metadata refs.
- Tag every raw item by kind.

Suggested raw storage:

```txt
~/.qratum/raw/
  blobs/
    sha256/
      ab/
        abc123...jsonl
  refs/
    raw_abc123.json
```

Raw ref shape (a small JSON record pointing at one archived raw file):

```json
{
  "schema_version": "qratum.raw_ref.v1",
  "raw_ref_id": "raw_...",
  "source": "claude-code",
  "source_session_id": "456d8fee-...",
  "kind": "main_transcript",
  "digest": "sha256:...",
  "original_path": "/Users/.../.claude/projects/.../session.jsonl",
  "archived_path": "~/.qratum/raw/blobs/sha256/ab/abc123.jsonl",
  "size_bytes": 12345,
  "observed_at": "2026-05-23T17:07:00Z",
  "local_only": true
}
```

Raw kinds (what type of raw file each blob is):

```txt
main_transcript
subagent_transcript
file_history_snapshot
source_insight_report
source_metadata
unknown
```

The local app must show data class badges so the user can always see how
sensitive a piece of data is:

```txt
raw
redacted
review
metrics
lesson
corpus
publishable
published
```

## Trust Boundaries

Trust boundaries are the gates that stop data from quietly moving somewhere more
dangerous than where it started. This section lists each gate and its default.

Trust boundaries prevent accidental data movement.

Hard rule:

```txt
No boundary may silently upgrade access to a more sensitive data class.
Raw transcripts must not be sent to external services or rendered into
shareable reports. External AI, export, and publish paths operate on
prepared/redacted artifacts unless a future accepted contract explicitly changes
that rule.
```

Boundary defaults:

| Boundary | Default |
| --- | --- |
| Source tool | Read only configured or user-approved sources. Hooks capture pointers/events, not full processing. |
| Raw archive | Local-only. Never exported, published, or sent to external AI. |
| Redaction | First boundary that can downgrade raw into safer data. Redaction uncertainty blocks export/publish unless approved. |
| Derived data | Reviews, metrics, findings, facets, lessons, and insights inherit data class from inputs unless explicitly downgraded. |
| Search index | Metadata, review, and redacted indexes are allowed. Raw indexes are off by default. |
| Local AI | Uses prepared/redacted artifacts by default. Any raw-local-AI mode is future scope and requires a verified local-only boundary. Provenance is recorded. |
| External AI | Redacted/prepared artifacts only, with consent. No raw transcript path. |
| Export | Explicit command/approval only. Eligibility checked first. |
| Publish | Manual first. Raw forbidden. Destination recorded in manifest. |

> **Warning — redaction is best-effort alpha (2026-06-15), with known leaks.** The
> Redaction boundary above is the *intended* contract, not a proven guarantee.
> The shipped deterministic redactor only matches an enumerated set of secret
> classes, and it currently leaks in several ways:
> - a `=>` assignment edge case (the real value survives after the placeholder);
> - the fields `git.branch`, `git.head_sha`, `started_at`, `ended_at`, and
>   `source_event_id` are not routed through redaction at all;
> - SSH-style git remotes (`git@host:org/repo.git`) are not caught;
> - and a committed golden fixture ships some of these unredacted.
>
> Closing these gaps — and proving it with a planted-secret recall benchmark — is
> the proposed verification-and-trust-gate work (P2-VERIFY-TRUST-GATE), namely
> the redaction-completeness and benchmark decisions
> (`verification-and-trust-gate.md` D3/D4, FIX-1/FIX-2). Do not rely on redaction
> as the sole barrier until that gate is green.

Every boundary should answer:

```txt
What data enters?
What data leaves?
Who/what can read it?
Can it be exported?
Can it be indexed?
Can AI consume it?
Is user approval required?
```

## Consent Records

When the user approves something sensitive, Qratum records that approval. For
the MVP this record is deliberately tiny: a one-line audit event. A richer
consent record exists only as documented future work.

Config is default policy. For the vault-first MVP, consent storage stays
intentionally small: config carries the defaults, and each sensitive action
writes a one-line audit event describing what was approved, by whom, for what
scope, and when.

This deliberately mirrors Edictum's "policy default + audited override"
semantics, but it is a documented resemblance only. Qratum does not depend on
Edictum runtime or storage to do it.

MVP audit event example (this is the shipped MVP behavior):

```json
{"event_type":"consent_audit","scope":"external_ai_redacted","decision":"approved","approved_by":"local_user","approved_at":"2026-05-24T10:10:00Z","object_selector":{"type":"current_workspace","object_ids":[]},"provider":"openrouter","destination":null,"created_from":"local_cli"}
```

Future full consent record shape (documented here, **not** MVP behavior):

```json
{
  "schema_version": "qratum.consent.v1",
  "consent_id": "con_...",
  "scope": "external_ai_redacted",
  "decision": "approved",
  "approved_by": "local_user",
  "approved_at": "2026-05-24T10:10:00Z",
  "expires_at": "2026-05-25T10:10:00Z",
  "data_classes": ["redacted", "review", "metrics"],
  "provider": "openrouter",
  "destination": null,
  "object_selector": {
    "type": "current_workspace",
    "object_ids": []
  },
  "revoked_at": null,
  "revocation_behavior": "blocks_future_use_only",
  "created_from": "local_app"
}
```

Stored duration choices and revocation records belong to that future full
shape, not the vault-first MVP.

MVP consent defaults (what Qratum asks, and when, for each sensitive scope):

| Scope | Default |
| --- | --- |
| raw_archive | Ask in `qrt init`. |
| index_raw | Off. |
| local_ai_raw | Future scope; off in onboarding. Any later version must prove the endpoint is local-only. |
| external_ai_redacted | Ask per use, allow remember in config. |
| external_ai_raw | Not allowed. Raw transcripts must not be sent to external services. |
| export_corpus | Ask every export. |
| export_adp | Ask every export. |
| publish_review_bundle | Ask every publish batch. |

Changing config or refusing a prompt blocks future use. It does not delete
existing raw data, derived outputs, external provider copies, or published
destinations. Deletion and purge are separate actions.

## Sources

A "source" is one AI coding tool Qratum can read from. The rule is to only show
the user sources Qratum can actually use right now.

Normal UX only shows sources Qratum can actually use.

MVP source:

```txt
Claude Code
```

Compatibility sources are added only when there is a concrete fixture and use
case. ADP can remain an export boundary and may become an import source when
needed. There is no generic JSONL source.

User-facing source states:

```txt
ready
import_only
needs_setup
error
```

Normal wizard and source list show only detected, usable sources. Debug/doctor
may show known adapters with `--all` and explain why they are hidden.

Every source adapter supports some subset of these operations:

```txt
discover
install_capture
capture_event
inventory_existing
archive_raw
normalize
```

SourceAdapter contract (the methods an adapter must provide):

```txt
Name() string
Capabilities() SourceCapabilities
Discover(ctx) SourceDiscovery
InstallCapture(ctx, opts) InstallResult
Inventory(ctx, opts) InventoryResult
Normalize(ctx, rawRefs) SessionRevision
```

Source capability flags (what a given adapter is able to do):

```txt
inventory_supported
capture_supported
archive_supported
normalize_supported
review_supported
facet_supported
export_supported
```

Known Claude discovery targets (where the Claude Code adapter looks for data):

```txt
~/.claude/projects/**/*.jsonl
~/.claude/projects/**/subagents/*.jsonl
~/.claude/file-history/**
~/.claude/usage-data/*.html
```

Do not make hardcoded source paths the primary capture mechanism. Runtime
capture still uses hook payload paths when available. Discovery/backfill uses
known locations with user consent.

## Future Bulk Import Wizard

This section is future scope. The accepted onboarding contract only exposes
`qrt import <file-or-folder>` with an explicit plan. It does not accept
`qrt import --all`.

If a later contract accepts bulk import, it should feel like a guided wizard,
not a silent bulk scan. The user sees what is available and chooses before any
heavy work runs.

Future shape:

Flow:

1. Inventory known source stores and configured folders.
2. Show what is available by source, count, size, date range, and support level.
3. Let the user choose what to import.
4. Write an import plan.
5. Run the plan idempotently (re-running it does not duplicate work).
6. Archive raw files if configured.
7. Normalize into sessions/revisions.
8. Redact, review, index, and mark corpus candidates.
9. Show success, skipped duplicates, unsupported files, and failures.

Large imports are planned, checkpointed, pausable, resumable, and rate-limited.

Backpressure controls (the levers that keep a big import from overwhelming the
machine):

```txt
max concurrent jobs
max concurrent AI jobs
external AI rate limit
disk free guard
raw archive size warning
pause/resume import
cancel import
progress checkpoint
retry individual failed jobs
skip unsupported files
```

Import plan shape (the plan written before a bulk import runs):

```json
{
  "schema_version": "qratum.import_plan.v1",
  "plan_id": "imp_...",
  "created_at": "2026-05-23T19:30:00Z",
  "sources": [
    {
      "source": "claude-code",
      "items": 3313,
      "bytes": 651000000,
      "date_min": "2026-01-11T11:30:00Z",
      "date_max": "2026-05-23T19:15:00Z",
      "actions": ["archive_raw", "normalize", "review", "index"]
    }
  ],
  "limits": {
    "max_jobs": 4,
    "max_ai_jobs": 1,
    "max_bytes": null,
    "disk_free_min_gb": 10
  },
  "status": "planned"
}
```

Before running large AI batches, Qratum estimates the work first so the user
isn't surprised by cost or time:

```txt
items to process
estimated input tokens
estimated output tokens
provider
model
local_or_external
estimated external cost when known
estimated time when known
data classes included
consent needed
```

AI job plan shape (the estimate written before an AI batch runs):

```json
{
  "schema_version": "qratum.ai_job_plan.v1",
  "job_plan_id": "aijp_...",
  "kind": "extract_facets",
  "provider": "openrouter",
  "model": "anthropic/...",
  "local_or_external": "external",
  "items": 842,
  "input_data_classes": ["redacted", "review", "metrics"],
  "estimated_input_tokens": 3200000,
  "estimated_output_tokens": 80000,
  "estimated_cost": {
    "currency": "USD",
    "min": 8.20,
    "max": 14.70
  },
  "consent_required": true,
  "raw_included": false
}
```

External AI batch work requires confirmation if cost is non-zero or unknown.
External AI stops when a configured budget cap is reached. AI jobs checkpoint
after each item or batch and can be retried individually.

## Session Revisions And Resumes

Source tools let a user reopen an old conversation and add to it. When that
happens, Qratum must not overwrite what it already processed — it records a new
revision instead.

Source tools can resume old sessions and append new turns. Qratum should not
overwrite prior processing output.

Model (three layers of identity):

```txt
source session = source tool conversation identity
qratum session = stable Qratum identity for that source session
session revision = normalized state observed at a specific digest/time
```

Session identity (what makes two observations "the same session"):

```txt
source + source_session_id
```

`workspace_context` is annotation, not identity. It describes where the session
happened and helps filtering, display, and future topology integration.

Workspace context shape:

```txt
cwd
git_root
git_remotes
branch
head_sha
dirty_state
source_project_path
optional_topology_node_id
```

If a source does not provide globally unique session IDs, that source adapter
must define a source-specific `source_session_key`.

Library default (how sessions and revisions are shown):

- Main list shows source sessions.
- A secondary view shows recent revisions.
- Session detail shows revisions/timeline.

Worker logic (step by step, what the worker does for each captured item):

1. Observe capture or import item.
2. Compute raw digest.
3. Match by source + source session ID, or source adapter's session key.
4. If no match, create a new Qratum session.
5. If digest already exists, mark duplicate/no-op.
6. If digest changed, create a new session revision.
7. Diff previous normalized revision vs new normalized revision.
8. Create a revision review.
9. Update full-session rollup review.

Review types:

```txt
revision_review
session_rollup_review
insight
```

A revision review answers:

```txt
What changed since the last Qratum observation?
What new risks/findings appeared?
What was fixed?
What remains unresolved?
```

A session rollup answers:

```txt
What is the current overall state of the source session?
What should the user inspect next?
Is this trajectory a corpus candidate?
```

## Retention And Deletion

Deleting things is mostly done through the app, where the user can see the blast
radius first. The CLI keeps a small set of delete commands for automation and
emergencies.

Deletion is UI-first. CLI deletion exists for automation and emergencies.

Public CLI should stay small:

```txt
qrt delete <object_id>
qrt raw purge
qrt retention apply
qrt workspace wipe
```

The local app shows impact before deletion:

```txt
object being deleted
affected revisions/reviews/findings/lessons/corpus/artifacts/indexes/raw refs
deletion mode
tombstone behavior
whether original source files may still exist outside Qratum
```

Deletion modes (how far the delete reaches):

```txt
object_only
object_and_derived
object_derived_and_raw
```

Tombstone shape (the marker left behind after a delete — it records that
something was removed, without keeping the removed content):

```json
{
  "schema_version": "qratum.tombstone.v1",
  "object_id": "ses_...",
  "object_kind": "session",
  "deleted_at": "2026-05-24T10:30:00Z",
  "deleted_by": "local_user",
  "deletion_mode": "object_derived_and_raw",
  "digests": ["sha256:..."],
  "reason": "user_requested"
}
```

Retention config:

```toml
[retention]
raw = "forever"
redacted = "forever"
reviews = "forever"
indexes = "sync_with_objects"
tombstones = "forever"
```

Any object deletion must remove or invalidate related index/backend entries.
Indexes and backends are derived projections, never source of truth.

`publish revoke` is deferred. Qratum can later revoke local publish records and
delete reachable local published bundles, but it cannot guarantee recall from
external destinations.

## Failure Taxonomy And Retry

When anything fails, Qratum writes down exactly what failed and whether it can
be retried. Retrying one failed step should never force a whole re-import.

Every failed capture, import, job, export, publish, AI run, or index update
produces a structured failure record.

Outcomes (the three ways an action can end short of success):

```txt
failed
  work did not complete and needs user/action/retry

blocked
  policy or consent prevented the action

skipped
  expected no-op, such as duplicate capture
```

Initial failure codes:

```txt
RAW_MISSING
RAW_UNREADABLE
RAW_TOO_LARGE
SOURCE_UNSUPPORTED
SOURCE_DISCOVERY_FAILED
CAPTURE_INVALID
DUPLICATE_CAPTURE
NORMALIZATION_FAILED
NORMALIZATION_UNSUPPORTED
REDACTION_FAILED
REDACTION_UNCERTAIN
AI_PROVIDER_UNAVAILABLE
AI_PROVIDER_FAILED
AI_POLICY_BLOCKED
AI_OUTPUT_INVALID
INDEX_FAILED
EXPORT_INELIGIBLE
EXPORT_FAILED
PUBLISH_FAILED
CONFIG_INVALID
STORE_LOCKED
STORE_CORRUPT
DISK_LIMIT_REACHED
PERMISSION_DENIED
INPUT_DELETED
```

Failure record shape:

```json
{
  "schema_version": "qratum.failure.v1",
  "failure_id": "fail_...",
  "code": "RAW_MISSING",
  "message": "Raw transcript path no longer exists.",
  "retryable": false,
  "recovery_hint": "Run import inventory again or enable raw archive earlier.",
  "object_refs": ["cap_...", "raw_..."],
  "source": "claude-code",
  "path": "/Users/.../.claude/projects/.../session.jsonl",
  "created_at": "2026-05-24T10:45:00Z"
}
```

Individual retry is first-class. Each job targets one object transition.

Job record shape (one unit of worker processing state):

```json
{
  "schema_version": "qratum.job.v1",
  "job_id": "job_...",
  "kind": "extract_facets",
  "status": "failed",
  "target_object_id": "revn_003",
  "input_object_ids": ["ses_123", "revn_003", "red_123"],
  "output_object_ids": [],
  "attempts": 2,
  "max_attempts": 5,
  "last_failure_id": "fail_...",
  "idempotency_key": "extract_facets:revn_003:prompt_v1:model_x:red_123",
  "trace_id": "...",
  "created_at": "2026-05-24T10:40:00Z",
  "updated_at": "2026-05-24T10:45:00Z"
}
```

Retry rules:

```txt
Only retry jobs with retryable=true unless the user forces it.
Retry creates a new attempt record.
Retry reuses the same idempotency key unless config/input changed.
If output already exists and matches expected digest, mark complete.
If input object was deleted, mark blocked INPUT_DELETED.
If policy changed, re-evaluate policy before running.
```

`repair <object_id>` is different from retry. It inspects an object and
schedules missing downstream jobs.

## Versioning And Migrations

Every stored object carries version stamps so Qratum can read old data and
migrate it safely. The guiding rule is: never silently destroy or rewrite the
original raw data.

Every stored object has version metadata.

Required object fields:

```json
{
  "schema_version": "qratum.review.v1",
  "producer": "qratum",
  "producer_version": "0.2.0",
  "transform_version": "review_deterministic.v1",
  "migration_version": 0
}
```

Rules:

```txt
Never silently rewrite raw refs or raw blobs.
Never destroy old objects without a backup manifest.
Old objects remain readable for at least one major version.
Migrations are idempotent.
Migrations write a migration report.
Dangerous migrations require confirmation.
If an object can be regenerated from raw/session inputs, prefer regeneration
over mutation.
```

Migration commands:

```txt
qrt migrate status
qrt migrate plan
qrt migrate apply
```

Migration manifests live under:

```txt
~/.qratum/manifests/migrations/
```

## Deduplication Policy

Qratum should not store the same thing twice or re-process unchanged data. It
matches on exact fingerprints first; detecting "near-duplicates" is left for
later.

Dedup is exact first. Near-duplicate detection is deferred.

Dedup levels (for each object type, the key used to decide "this is the same
thing I already have"):

| Level | Key |
| --- | --- |
| Raw blob | `sha256(raw bytes)` |
| RawRef | `source + source_session_id + raw_digest + kind` |
| Capture | `source + source_event_id`, else `source + source_session_id + raw_digest + observed_at bucket` |
| Session | `source + source_session_id`, or source adapter's `source_session_key` |
| SessionRevision | `session_id + normalized_digest` |
| SessionMeta | `revision_id + extractor_version` |
| Signal | `revision_id + signal_type + evidence_digest + detector_version` |
| Review | `revision_id + review_type + scorer_config_digest` |
| Facet | `revision_id + input_data_digest + provider + model + prompt_version` |
| Artifact | `input_object_ids + input_digest + renderer_version + artifact_kind` |
| CorpusItem | `source_review_id + export_digest + corpus_schema_version` |
| PublishBundle | `publisher_id + bundle_digest + destination` |

Rules (common situations and what dedup does in each):

```txt
same raw digest, different import path
  same blob; add extra source location/provenance

same source session, changed raw digest
  new SessionRevision

same source session, same normalized digest
  duplicate/no-op, even if raw formatting changed

same review content, different renderer
  same Review, different Artifact

same artifact content, different destination
  same Artifact, different PublishBundle or publish event

same resumed session
  same Session, new Revision if digest changed
```

Dedup never deletes source history automatically. It marks duplicates/skips and
preserves provenance paths.

## Artifact Manifests

Every file Qratum writes is tracked by a manifest record — there are no loose,
unmanaged files lying around. The file's identity is its ID, not its path, so
moving a file doesn't change what it is.

Every file Qratum writes is managed by an artifact manifest. No loose reports,
exports, bundles, or diagnostics.

Artifact record:

```json
{
  "schema_version": "qratum.artifact.v1",
  "artifact_id": "art_...",
  "kind": "html_report",
  "data_class": "review",
  "path": "~/.qratum/artifacts/reports/rev_123.html",
  "media_type": "text/html",
  "digest": "sha256:...",
  "size_bytes": 43892,
  "producer": "qratum-renderer-html",
  "producer_version": "0.2.0",
  "renderer_version": "html_report.v2",
  "input_object_ids": ["rev_123"],
  "input_digest": "sha256:...",
  "created_at": "2026-05-24T11:10:00Z",
  "export_eligibility": "not_exportable",
  "publish_eligibility": "review_bundle_allowed",
  "qratum_uri": "qratum://artifact/art_..."
}
```

Artifact kinds:

```txt
html_report
json_report
adp_jsonl
corpus_jsonl
publish_bundle
redaction_report
import_plan
migration_plan
migration_report
diagnostic_bundle
```

Eligibility values (whether an artifact may leave the machine, and under what
condition):

```txt
not_exportable
exportable_redacted
exportable_review
exportable_corpus
exportable_raw_requires_explicit_approval
```

Artifact rules:

```txt
Every artifact has a manifest.
Every artifact has a digest.
Artifact paths are not identity. artifact_id is identity.
Moving an artifact updates path, not artifact_id.
Deleting an artifact writes a tombstone.
Regenerating with a new renderer creates a new artifact.
```

Reports link to Qratum object refs internally. Human-facing UI renders those
refs as labels, app routes, or relative bundle links.

```txt
qratum://session/ses_...
qratum://revision/revn_...
qratum://review/rev_...
qratum://artifact/art_...
qratum://raw/raw_...
qratum://lesson/les_...
qratum://corpus/cor_...
```

## Review, Metrics, Lessons, And Insights

A "review" looks at one session and says what happened in it. "Insights" zoom
out across many sessions to describe the user's overall AI coding practice. This
section also covers the metrics and the lessons Qratum can suggest.

Replace `ReviewCard` as the central product object with `ReviewEnvelope`.

Review envelope shape (the central object that wraps everything known about one
review):

```json
{
  "schema_version": "qratum.review.v1",
  "review_id": "rev_...",
  "session_id": "ses_...",
  "revision_id": "r002",
  "review_type": "revision_review",
  "status": "complete",
  "verdict": "needs_attention",
  "summary": "...",
  "metrics": {},
  "findings": [],
  "signals": [],
  "lesson_candidates": [],
  "corpus_candidate": {},
  "artifact_refs": [],
  "provenance": {}
}
```

Metrics should include:

```txt
duration
tokens
estimated_cost
tool_calls
tools_used
commands_run
verification_commands
failed_verification_commands
retry_loops
files_touched
time_to_first_test
final_verification_status
external_network_calls
```

Deterministic review must treat tool-call sequences as first-class input, not
just message text.

Signal classes (the categories of observation the review can produce):

```txt
message_text_signals
tool_sequence_signals
file_change_signals
verification_signals
cost_time_signals
```

Important tool-call findings (specific patterns worth flagging from the tool
calls):

```txt
edits after last verification
no final verification
failed verification after claimed success
retry loops
file thrashing
repeated failed edits
tool errors ignored
user rejected action repeated
large change without inspection
external network call during sensitive task
```

Lesson suggestions are future scope, not part of first-run onboarding. If a
later contract accepts them, they should include:

- Short reusable lessons.
- Repo/project-specific habits.
- Failure patterns.
- Good trajectory examples.
- Suggested rules, hooks, or skills.

AI-generated lesson behavior:

- Auto-save low-risk suggestions.
- Move higher-risk suggestions through a factory-curated, human-sampled, batch-approved lane when recurring volume exists; no persistent per-item approval queue.
- Record provider, model, prompt version, input data class, and input digest.

Insight output is cross-session and should be closer to a usage-insights report
than the current Qratum report.

Insight sections:

```txt
At a glance
What you work on
How you use agents
Wins
Where things go wrong
Suggested habits/features
New usage patterns
On the horizon
Model/tool feedback
```

Insights are not the same product as per-session review. A review tells the user
what happened in one session or revision. An insight report tells the user what
their AI coding practice looks like across many sessions.

Lessons from reviewed usage-insights implementations (design lessons learned
from other people's insight tools):

- Keep lightweight inventory separate from transcript parsing.
- Cache deterministic session metadata separately from AI facets.
- Extract facets per session, then aggregate across sessions.
- Generate insight sections independently, then generate "At a glance" from the
  section outputs.
- Track repeated user instructions because those become lessons/rule candidates.
- Track friction, tool errors, interruptions, retry loops, response time, token
  usage, tool usage, files touched, language mix, and overlapping sessions.
- Detect and suppress meta-sessions created by insight generation itself.
- Do not copy monolithic command shapes, provider-specific model calls, hidden
  remote collection, or automatic external publishing.

Qratum Insights pipeline (the ordered steps that produce an insight report):

1. Inventory source sessions using file metadata and source indexes first.
2. Parse only new or changed raw refs.
3. Write `SessionMeta` for deterministic metrics.
4. Select the canonical revision for aggregate analysis.
5. Write `Facet` records for AI/classifier labels when allowed by data policy.
6. Build aggregate datasets from session metadata and facets.
7. Generate insight sections from the aggregate dataset.
8. Generate the at-a-glance summary from the completed sections.
9. Render local app DTOs and optional HTML artifacts.

Canonical aggregate rules (which revision counts when summarizing across
sessions — and how to keep the rest without losing it):

- Keep every raw ref, session, and revision.
- Aggregate on the latest complete revision per source session by default.
- For source tools that create branches, mark non-canonical branches instead of
  deleting them.
- For resumed sessions, aggregate the newest observed revision and show the
  revision timeline separately.
- Let advanced users switch an insight report to "all revisions" when they want
  churn analysis.

SessionMeta should include at least:

```txt
session_id
revision_id
source
workspace_context
started_at
ended_at
duration
human_message_count
assistant_message_count
tool_counts
tokens
estimated_cost
commands_run
verification_commands
tool_errors
user_interruptions
retry_loops
files_touched
languages
lines_added
lines_removed
message_hours
overlap_markers
```

Facet should include at least:

```txt
session_id
revision_id
source
underlying_goal
goal_categories
outcome
user_satisfaction
agent_helpfulness
session_type
friction_categories
friction_detail
primary_success
brief_summary
user_instructions_to_agent
lesson_candidates
corpus_candidate_reason
```

Facet cache keys must include (so a cached facet is reused only when all of
these match):

```txt
revision_digest
redaction_digest
provider
model
prompt_version
input_data_classes
```

Facet extraction must be source-neutral. A Claude Code adapter can populate
Claude-specific tool names and transcript details, but the facet schema cannot
require Claude Code.

## AI Providers And Data Policy

Qratum talks to AI models through one provider interface, so local and external
models are interchangeable behind it. Local models are preferred; external calls
are gated by data policy.

Qratum should support local AI and external AI behind a provider interface.

Provider priority (which local runtimes to prefer, in order):

1. OpenAI-compatible local endpoint.
2. Ollama through compatible endpoint or adapter.
3. LM Studio through compatible endpoint.
4. llama.cpp server.
5. MLX direct later.
6. OpenRouter external provider.

For Mac, MLX-backed models are likely the best local runtime. Qratum should not
hard-require MLX at first. It should integrate through OpenAI-compatible local
endpoints first and use `llmfit` as an optional setup advisor.

Future `llmfit` integration:

- If installed, a future accepted AI setup flow can run `llmfit system --json`.
- It can suggest local coding/reasoning models.
- It is optional and never required for core Qratum.

Data policy (what each kind of AI is allowed to see by default):

```txt
local AI default: may see redacted, review, metrics, lessons
local AI raw input: future scope; off for onboarding unless a later contract
  proves the endpoint is local-only
external AI default: ask every time
external AI recommended input: redacted session + review + metrics
external AI raw input: forbidden
```

Every AI run records:

```txt
provider
model
input_data_classes
input_digests
prompt_version
output_digest
local_or_external
approved_by
```

## Search And Derived Projections

Search is a convenience layer built *on top of* the files, never a second copy
of the truth. You can always rebuild it from the filesystem objects.

Qratum always has a filesystem object store. That store is the source of
truth. Search is a derived local projection, not a second store.

Projection rule:

```txt
Backends are projections, not source of truth.
Search stores source object IDs, digests, data_class, and provenance.
Projections must support delete-by-object-id.
Projections must respect consent, retention, and tombstones.
Projections must be rebuildable from filesystem objects where possible.
```

Search interface (what any search backend must support):

```txt
SearchBackend
  upsert index entries
  delete by object ID
  search query + filters
  rebuild scope
```

Default local search:

```txt
SearchBackend
  SQLite FTS5 when search is unlocked
```

The first search implementation uses SQLite FTS as the default local
projection. That will also be Qratum's first third-party Go runtime
dependency, so it needs an explicit supply-chain decision under
`docs/supply-chain.md` when it ships.

Search rules:

```txt
Search indexes are derived artifacts.
Each index has data_class.
Raw indexes are disabled by default.
Redacted/review indexes are enabled only when indexing is explicitly turned on.
Index entries store source object IDs and spans.
Index purge follows object purge.
```

Search modes:

```txt
metadata
review_text
redacted_text
raw_scan_once
```

Persistent raw indexes are deferred. If richer retrieval or
remote/private-perimeter projections ever earn implementation, they need a
fresh spec and measured demand rather than this draft assuming them in advance.

## Corpus Export

A "corpus" is cleaned training-style data prepared for some downstream consumer.
Producing it is a separate, explicitly-approved act — having reviewed a session
does not make it safe to export.

Qratum should produce two export families:

- ADP-style JSONL.
- Qratum-native corpus JSONL.

Corpus export is separate from review publishing.

Corpus bundle (the files written for one corpus export):

```txt
corpus_export/
  manifest.json
  corpus.jsonl
  provenance.json
  redaction_report.json
```

Corpus item shape (one entry in a corpus export):

```json
{
  "schema_version": "qratum.corpus_item.v1",
  "corpus_item_id": "cor_...",
  "source_session_id": "ses_...",
  "source_review_id": "rev_...",
  "trajectory": {},
  "facets": {
    "outcome": "mostly_achieved",
    "primary_success": "good_debugging",
    "signals": ["verification_present", "clear_user_goal"],
    "risks": ["license_or_export_rights_unknown"]
  },
  "data_class": "redacted",
  "export_eligibility": "needs_review",
  "provenance": {
    "session_digest": "sha256:...",
    "redaction_version": "...",
    "export_version": "..."
  }
}
```

Corpus export eligibility is explicit. A review existing does not mean the item
is safe to export.

Corpus quality uses sparse facets/signals/risks, not a required score matrix.
Absence of a signal means unknown, not bad. Failed sessions can be valuable if
they include useful failure, debugging, or repair evidence.

There is no License/IP model in MVP. Any corpus export or publish outside the
machine requires explicit user approval and defaults to `needs_review` when
provenance/export rights are unknown.

## Publishing

Publishing is how approved outputs leave Qratum for a destination the user
chose. Today that destination is a local folder, by manual approval only.
Private-perimeter publishing is future scope.

Publishing exists so the user can move approved Qratum outputs to an explicit
destination. Private-perimeter publishing is future scope.

Store, exporter, and publisher are three separate things:

```txt
Store: where Qratum keeps working data.
Exporter: transforms data into portable formats.
Publisher: delivers approved bundles to a destination.
```

MVP publisher:

```txt
local_folder
```

Future non-MVP publishers:

```txt
s3_compatible_bucket
private_http_endpoint
scp_or_rsync
git_branch
database_projector
queue_or_topic
```

Publishing modes:

```txt
manual
suggested_auto
policy_auto
```

MVP mode is manual. `suggested_auto` and `policy_auto` are future modes. No
automatic publishing is part of the normal operating model.

First publishable bundle (the files in the first kind of bundle Qratum can
publish):

```txt
review_bundle/
  manifest.json
  objects/
    session.redacted.json
    review.json
    metrics.json
    findings.json
    provenance.json
```

Raw data is forbidden.

Publish manifest (the record of one publish run):

```json
{
  "schema_version": "qratum.publish_manifest.v1",
  "publish_id": "pubrun_...",
  "publisher_id": "local_folder",
  "created_at": "2026-05-23T19:45:00Z",
  "items": [
    {
      "object_id": "rev_...",
      "object_type": "review",
      "data_class": "review",
      "digest": "sha256:...",
      "destination_ref": "~/QratumPublished/pubrun_..."
    }
  ],
  "policy": {
    "raw_allowed": false,
    "redaction_required": true,
    "approval_mode": "manual"
  }
}
```

## Public CLI

This is the command surface the user sees. It is kept deliberately small.
Heavier internals should move behind implementation use cases, not hidden CLI
aliases.

First-run public commands:

```txt
qrt init
qrt open
qrt status
qrt doctor
qrt import <file-or-folder>
qrt sessions
qrt session <session_id>
qrt export
```

`qrt export` is explicit egress. It must show scope, destination, data classes,
item counts, estimated size, and confirmation before data leaves Qratum.

Future commands beyond this list require separate acceptance before they become
public. Do not carry them forward from Milestone A just because code already
exists.

Future/destructive/emergency candidates:

```txt
qrt failed
qrt retry <id>
qrt logs
qrt delete <object_id>
qrt raw purge
qrt retention apply
qrt workspace wipe
```

These commands must show impact and require confirmation unless an explicit
automation flag is used.

`qrt` with no args:

- Does not create state or run work.
- Shows concise help/status and points to `qrt init`, `qrt open`, and
  `qrt status`.

Do not keep Milestone A commands as hidden compatibility aliases by default.
When a replacement lands, delete the old public command path. Internal package
functions can remain only when the new use cases call them.

Diagnostics stay CLI-first where useful. Public CLI exposes status, doctor,
failed, retry, and logs.

## Local App

The UI-first onboarding direction supersedes the older "app later" posture.
`qrt open` is the product entrypoint after install, and the CLI remains the
bootstrap/recovery surface. The runtime implementation is not shipped yet; the
contract lives in `specs/current/ui-first-onboarding.md`.

Initial app sections:

```txt
Status
Sessions
Reviews
Insights
Lessons
Corpus
Publish
Import
Failures
Settings
```

The app should always show data class badges and whether external AI was used.

Local app security (it binds to localhost only; no LAN access in the MVP):

```txt
default host: 127.0.0.1
default port: 9473
remote LAN mode: unsupported for MVP
```

MVP auth (how the local app authenticates the user on first launch):

```txt
Use a one-time bootstrap nonce on first local launch.
Immediately exchange it for an HttpOnly SameSite cookie.
Redirect to a clean URL without the nonce.
Store token hash, not plaintext token.
Store auth state under ~/.qratum/state/app_auth.json with 0600 permissions.
```

Bootstrap example (the first-launch handshake, step by step):

```txt
open http://127.0.0.1:9473/bootstrap?nonce=...
server validates nonce
server sets cookie
server rotates/deletes nonce
server redirects to http://127.0.0.1:9473/
```

Cookie:

```txt
Set-Cookie: qratum_app_token=...; HttpOnly; SameSite=Strict
```

Do not set `Secure` for localhost HTTP unless local HTTPS is added.

Route/security rules:

```txt
CSRF/same-origin checks for mutating routes.
Route capability classes:
  public_bootstrap
  read_safe
  read_sensitive
  mutate_safe
  mutate_sensitive
  raw_access
  external_action

No raw routes in first app shell.
No CORS wildcard.
CSP header enabled.
frame-ancestors 'none'.
No logging paths, tokens, raw content, or prompt content.
```

Suggested headers:

```txt
Content-Security-Policy: default-src 'self'; frame-ancestors 'none'
X-Frame-Options: DENY
Referrer-Policy: no-referrer
X-Content-Type-Options: nosniff
```

## Setup / Init

The older `qrt setup` wording is superseded by the UI-first onboarding contract.
Implementation should use `qrt init` for first-run bootstrap and `qrt open` for
the local app.

`qrt init` shows exactly what it will write before it writes anything. It
preserves existing local sessions and prepares the latest 10 for viewing only
after confirmation. It does not configure AI, publishing, remote backends, raw
indexing, or full-history processing as happy-path onboarding.

Setup should not show planned/unavailable sources. It should only show detected
sources that are ready, import-only, need setup, or have an error.

## No Hidden Processing

Qratum must never quietly do anything expensive, sensitive, or visible to the
outside world. If it costs money, touches raw data, or leaves the machine, the
user asked for it.

Qratum must not perform expensive, sensitive, or externally visible work unless
the user explicitly asked for it or enabled a background policy.

No hidden actions (these never happen on their own):

```txt
large scans
raw archive changes
raw indexing
external AI calls
corpus export
publishing
deletion/purge
migration apply
backend projection to remote targets
```

Allowed implicit/light actions (small, safe things Qratum may do without
asking):

```txt
create ~/.qratum/config.toml on first use
read small state files
show status
read already-created indexes
start local app when user runs qrt open
resume background preservation schedule if user enabled it
```

> **UI-first update (2026-06-17):** the local app is the accepted onboarding
> direction. A resident daemon is not required for the first implementation;
> background preservation should use a source hook or OS schedule first, stay
> preserve-only, and remain visible in `qrt status` and `qrt doctor`.

Command rules (per command, how much work it is allowed to do):

```txt
qrt
  status only; no big work

qrt status
  status only; no big work

qrt init
  may preserve all discovered local sessions and prepare latest 10 after showing
  a plan and receiving confirmation; prepare all requires a separate estimate
  and confirmation; no AI, upload, publish, or raw preview

qrt open
  starts local app; no import/AI/publish automatically

qrt import <file-or-folder>
  may inventory after confirmation; import plan before heavy work

qrt export
  starts with an export plan; writes or sends only after scope, destination,
  data classes, and confirmation are explicit
```

A configured policy counts as explicit permission. For example, `raw.archive =
true` allows the worker to archive raw refs from captured sessions, but
`ai.external = "ask"` does not allow background external AI calls.

## Performance Principles

These are guidelines for keeping Qratum responsive, not hard service-level
guarantees.

These are principles, not strict SLAs:

```txt
Hooks do the minimum durable capture work and return quickly.
qrt with no args performs only lightweight status reads.
Inventory starts with metadata/file scans before parsing full transcripts.
Large imports run as background jobs with visible progress.
AI work is async and never blocks app startup, status, or navigation.
Raw archive, redaction, indexing, and export process large files incrementally
where practical.
Search uses an opt-in backend for advanced scale.
External provider calls are rate-limited and retryable.
Worker concurrency is configurable.
```

One hard rule stands: source hooks must not parse full transcripts, call AI,
generate reports, or perform network calls.

## Observability

Qratum emits traces and metrics from day one, using OpenTelemetry APIs, but it
does not require a collector to run. Crucially, sensitive content never goes
into telemetry.

Qratum uses OpenTelemetry APIs internally from the start. A collector is not
required to run Qratum.

Default:

```txt
observability.enabled = true
observability.exporter = "local"
observability.otlp.enabled = false
```

Exporter modes:

```txt
noop
local
otlp
```

Every major use case gets a trace span:

```txt
setup.workspace
source.capture
source.inventory
raw.archive
session.normalize
session.revision.create
review.generate
redaction.run
facet.extract
insight.generate
index.update
backend.project
artifact.render
export.run
publish.run
retention.apply
delete.object
job.retry
```

Metrics:

```txt
qratum.jobs.total
qratum.job.duration
qratum.failures.total
qratum.captures.total
qratum.raw.archive.bytes
qratum.objects.total
qratum.ai.calls.total
qratum.backend.operations.total
qratum.artifacts.total
qratum.exports.total
qratum.publish.total
```

Safe attributes (low-sensitivity labels that may appear on spans/metrics):

```txt
source
operation
status
failure_code
data_class
backend_type
provider_type
object_kind
```

Never put these in logs, spans, or metrics:

```txt
raw content
prompt content
tool result content
API keys
auth tokens
full transcript paths
full command bodies
full file contents
```

Public diagnostics stay small:

```txt
qrt status
qrt failed
qrt retry <id>
qrt logs
```

Detailed diagnostics live in `qrt debug` now and can later gain a local app surface.

## Minimal Implementation Milestones

### P0-SPEC-AND-CONTRACTS

Goal: lock the redesign before any runtime changes.

Deliverables:

- Accepted operational model aligned with `qratum-vault-first.md`.
- Architecture style: lightweight DDD plus ports and adapters.
- Object lifecycle.
- Trust boundaries.
- Consent defaults plus audit-event contract, with the full consent record
  shape documented as future work.
- Failure/job retry contract.
- Dedup/versioning/migration rules.
- Artifact manifest contract.
- Search projection contract.
- JSON schemas for core runtime objects.
- TOML config schema.
- Migration notes from Milestone A.
- ADR 0010 recording the vault-first direction.

Acceptance:

- Design doc has no unresolved storage/config contradictions.
- Generic JSONL and persistent approval queues are explicitly out of scope.
- Vault-first sequencing is recorded without unlocking runtime work.
- Public CLI stays small.

### P1-VAULT-FIRST

Status: **SHIPPED** (merged, test-backed; `qrt` self-reports
`milestone: vault-first`).

Goal: preservation before product surface.

Deliverables:

- Central `~/.qratum` workspace.
- Global Claude Code hook install/status.
- Copy-on-capture raw archive with `qratum.raw_ref.v1`.
- Backfill over existing local transcripts.
- `qrt vault archive`, `qrt vault doctor`, and `qrt vault backup --verify`.
- Status output for vault health, copy failures, and backup freshness.
- Raw kind additions for vendor exports, vendor memory dirs, insight reports,
  and import receipts.
- The explicit supply-chain note for the later SQLite search dependency.

Acceptance:

- A deleted source transcript remains recoverable from the vault. (Met for the
  **raw blob** — the content-addressed copy survives. NOT yet met for *derived
  artifacts*: the refinery reads the live `transcript_path` (`daemon.go`) and
  does not fall back to the blob, so re-deriving review/report from a deleted
  source fails. Tracked by the blob-fallback fix in
  `verification-and-trust-gate.md` FIX-3 / D6a.)
- Hook install is idempotent and global.
- Backfill re-runs cleanly with digest dedup.
- Doctor reports missing hook, copy failures, stale backfill, missing or
  unverified backup, and the cloud-session limitation.
- No raw content leaves the machine.

### Later unlocks (demonstrated pull only)

Nothing after P1 is pre-committed to a fixed numbered ladder. Each later phase
needs measured demand and a narrow contract before implementation.

Possible later unlocks, in earned order:

1. Lessons-to-memory: direct gateway calls or thin local scripts once recurring
   curated candidates exist, with a factory-curated, human-sampled,
   batch-approved flow.
2. Insights-harvest: on-demand aggregate analysis once preserved data is being
   inspected.
3. Search: SQLite FTS local projection once real use shows repeated need.
4. Review surface: CLI first, app later if preserved sessions create sustained
   pull for richer inspection.
5. Corpus: only after a real downstream consumer exists.

## Explicit Non-Goals For The Next Build Phase

These are deliberately out of scope for the next build phase:

- Hosted service.
- Enterprise dashboard.
- Marketplace backend.
- PR bot.
- Git checkpoint/rewind.
- Multi-user auth.
- Required external database/server.
- Required `llmfit`.
- Required external AI.
- LLM redaction as the default safety layer.
- Generic JSONL/qratum-jsonl source.
- Persistent approval/pending item queues.
- License/IP workflow beyond `needs_review`.
- Persistent raw search indexes.
- Remote LAN app mode.

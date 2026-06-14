# Qratum Memory Curation Pipeline

Status: SUPERSEDED on 2026-06-12. Do not implement anything from this
document. An adversarial review
(`docs/reviews/2026-06-12-memory-architecture/`) invalidated its core
mechanisms: the bundle/importer/receipt bridge, the `duplicate` and
`blocked_sensitive` receipt outcomes (the gateway cannot produce them), the
heading-based Tier-0 split (the real data contains no headings), and the
lesson staging queue (violates the operational model's own non-goal).
Replacement: `specs/current/qratum-vault-first.md` and
`personal-memory-gateway/docs/gateway-verbs-plan.md`. Kept for the historical
record only.

## Purpose

Turn historical AI conversation data into curated, durable, atomic memories and
deliver approved memories to the external Personal Memory gateway.

Concrete motivating inputs:

- A Claude.ai account data export (conversations, projects, curated memories).
- Local Claude Code JSONL transcripts (already the MVP Qratum source).

Concrete consumer:

- `personal-memory-gateway`, whose only write path is the MCP `memory_store`
  tool and whose roadmap explicitly gates Qratum integration on Qratum
  publishing source-artifact contracts.

This pipeline is the first real consumer of the `Lesson` primitive and the
first publish destination beyond review bundles.

## Relationship To The Operational Model

This spec adds no new core primitive. It reuses:

```txt
Source / SourceAdapter   one new import-only adapter: claude-ai-export
RawRef                   export files archived as content-addressed blobs
Session / SessionRevision  one per export conversation
Lesson                   the memory candidate object
Consent                  one new scope for memory publishing
Artifact / PublishBundle one new bundle kind: memory_bundle
Publisher                MVP local_folder; future direct HTTP publisher
```

The operational model says Qratum produces session-derived Lessons and a future
consumer turns durable Lessons into longer-lived knowledge. Personal Memory is
that consumer. Qratum curates; Personal Memory stores.

## Locked Decisions

- Memory candidates are `Lesson` objects. No separate `memory_candidate`
  primitive.
- Raw transcript content never leaves Qratum. Only short, curated,
  contract-shaped lesson content is exported. This matches the Personal Memory
  roadmap hard rule: never ingest raw transcripts.
- Handoff is bundle-mediated (pull), not direct push, for MVP. Qratum publishes
  an approved `memory_bundle` to a local folder. A small importer owned by the
  `personal-memory` repo reads the bundle and issues one `memory_store` call
  per item. Qratum holds no gateway credentials. A direct
  `personal_memory_http` publisher is future scope.
- Approval lives in Qratum, before export. The gateway receives only approved
  memories. Qratum does not depend on the gateway's proposed (unimplemented)
  `staging:` namespaces.
- Export of a memory bundle requires explicit per-bundle approval, like every
  other publish path.

## Source: claude-ai-export

A Claude.ai account data export is a folder:

```txt
export/
  conversations.json   all conversations, one array
  memories.json        curated account + project memory strings
  projects/*.json      project metadata (uuid, name, description, docs,
                       prompt_template)
  design_chats/*.json  conversation-shaped design chats (uuid, title, project,
                       messages)
  users.json           account identity
```

Observed reference export (not a fixture; fixtures must be synthetic):

```txt
conversations.json   187 MB, 556 conversations, 11,962 messages,
                     447 with non-empty summary
memories.json        1 account memory (10,097 chars),
                     7 project memories (2,274 - 5,158 chars)
projects/            13 projects
design_chats/        5 files
```

Adapter contract:

```txt
name                 claude-ai-export
state                import_only
capabilities         inventory_supported, archive_supported,
                     normalize_supported
not supported        capture (no hooks; exports are point-in-time snapshots)
input                path to an export folder
source_session_key   conversation uuid (also design chat uuid)
```

Raw kinds mapping:

```txt
conversations.json   source_export_bundle (new raw kind)
memories.json        source_memory (new raw kind)
projects/*.json      source_metadata
users.json           source_metadata
design_chats/*.json  source_export_bundle
```

Normalization:

- Each conversation in `conversations.json` becomes one Session with one
  SessionRevision per observed export digest. Re-importing a newer export of
  the same account creates new revisions only for changed conversations.
- Messages map `sender: human|assistant` to the source-neutral trajectory
  roles. `attachments`, `files`, and `parent_message_uuid` are preserved in
  the revision, not dropped.
- The conversation's project uuid (when present) resolves to a project name
  via `projects/*.json` and lands in `workspace_context` as annotation, not
  identity.
- Design chats normalize the same way with `kind: design_chat` annotation.

Scope note: Claude.ai conversations are bounded AI sessions but not all are
coding sessions. Import-only sources may contain general sessions. The session
library, facets, and goal categories absorb this. This does not widen capture
scope or add a generic JSONL source.

## Source: claude-code

No new adapter work. Claude Code JSONL transcripts already flow through
capture/import, normalization, redaction, review, and (P4) facet extraction.
Facet `lesson_candidates` from those sessions enter this pipeline at the
Lesson stage, identically to export-derived candidates.

## Extraction Tiers

Tier 0 - deterministic memory parse (no AI, local, cheap):

```txt
input    memories.json
output   Lesson candidates
rules
  - account memory string splits on markdown structure (headings, bold
    section markers, bullet groups) into atomic candidates
  - each project memory string splits the same way; candidates carry
    scope {type: project, project: <slugified project name>}
  - account candidates carry scope {type: account}
  - splitting is mandatory: the gateway caps content at 8000 chars and the
    observed account memory is 10,097 chars
  - default kind: project memories -> project_context,
    account memory -> profile; reviewer can reclassify
  - transform_version: memory_parse.v1
```

Tier 1 - summary mining (deferred decision):

```txt
input    conversations[].summary (447 non-empty in reference export)
output   Lesson candidates
status   optional; AI-assisted or skipped in v1
```

Tier 2 - full conversation mining (P4, AI, consent-gated):

```txt
input    redacted SessionRevisions
path     existing facet extraction -> lesson_candidates
gating   AI job plan with item/token/cost estimates is mandatory before
         batch runs; external AI requires consent per existing data policy
```

Tier 0 alone delivers most of the immediate value: the export's memory strings
are already curated by the source tool and only need atomization, scoping,
review, and delivery.

## Lesson Object

`qratum.lesson.v1` (already in the planned schema registry) gains its concrete
shape from this pipeline:

```json
{
  "schema_version": "qratum.lesson.v1",
  "lesson_id": "les_...",
  "status": "suggested",
  "kind": "project_context",
  "content": "Short durable memory text.",
  "reason": "Why this is worth keeping.",
  "confidence": "high",
  "data_class": "lesson",
  "scope": {
    "type": "project",
    "project": "ai-contracts"
  },
  "source": {
    "source": "claude-ai-export",
    "source_session_id": null,
    "raw_ref_id": "raw_...",
    "source_digest": "sha256:...",
    "transform_version": "memory_parse.v1"
  },
  "review": {
    "reviewed_by": null,
    "reviewed_at": null,
    "edited": false
  },
  "export": {
    "exported_at": null,
    "bundle_id": null
  },
  "provenance": {}
}
```

Status lifecycle:

```txt
suggested -> staged -> approved -> exported
any non-exported status -> rejected
```

Kind enum (deliberately aligned with the gateway contentClass enum so the
export mapping is near-identity):

```txt
note
profile
preference
project_context
coding_context
workflow_rule
decision
relationship
reference
project_checkpoint
```

Constraints:

```txt
content length      hard max 8000 chars (gateway cap); recommended <= 1200
content quality     atomic, self-contained, no raw transcript spans, no
                    tool-call output, no file contents
data_class          always lesson; lessons derived from raw inputs are a
                    deliberate data-class downgrade performed only by the
                    extraction transform
editability         reviewer may edit content, kind, and scope before
                    approval; edits set review.edited = true
```

Dedup (extends the operational-model dedup table):

```txt
Lesson    scope + sha256(normalized content)
```

Re-running extraction over the same input digests produces no duplicate
candidates. Rejected lessons stay on file so re-extraction does not resurface
them as new suggestions.

## Pre-Flight Content Policy

The gateway hard-rejects, with no bypass:

```txt
empty content
secret-like content   PEM private keys, AWS key IDs, key:value secrets,
                      seed/recovery phrases
high-sensitivity      passport/SSN/tax/government ID, medical terms,
                      bank/card/IBAN, legal-dispute terms
```

Qratum mirrors these checks as a deterministic pre-flight at two points:

```txt
candidate creation    failing candidates are created with a blocking finding
                      attached, so the reviewer sees why
export                approved lessons failing pre-flight are excluded from
                      the bundle and marked blocked, never silently dropped
```

The gateway remains the authority; the mirror is best-effort and exists so
rejects surface at review time instead of import time. Residual gateway
rejects are handled by the import receipt (below).

Expected real-world consequence: memories about medical, financial, or legal
topics will be blocked by the gateway by design. The reviewer can rephrase
them or accept that they stay local in Qratum.

## Review And Approval

- Lessons are filesystem objects under `~/.qratum/lessons/`, projected into
  the SQLite LessonBackend.
- The local app (P3+) is the primary review surface: list suggested lessons,
  edit, approve, reject.
- `qrt export memories` is the CLI path: it shows approved-but-unexported
  lessons grouped by scope, runs pre-flight, displays exactly what will leave
  the machine, and asks for confirmation. Without the app, review can happen
  entirely through this command plus lesson file edits.
- New consent scope:

```txt
publish_memory_bundle    ask every publish batch (same default as
                         publish_review_bundle)
```

## Memory Bundle Contract

New artifact/bundle kind: `memory_bundle`.

```txt
memory_bundle/
  manifest.json      qratum.publish_manifest.v1, items typed memory_export_item
  memories.jsonl     one qratum.memory_export_item.v1 per line
  provenance.json    source digests, transform versions, extraction tiers
```

`qratum.memory_export_item.v1` is wire-shaped for `memory_store`:

```json
{
  "schema_version": "qratum.memory_export_item.v1",
  "lesson_id": "les_...",
  "content_sha256": "...",
  "store": {
    "namespace": "project:ai-contracts",
    "content": "Short durable memory text.",
    "contentClass": "project_context",
    "metadata": {
      "origin": "qratum",
      "lesson_id": "les_...",
      "bundle_id": "pubrun_...",
      "source": "claude-ai-export",
      "source_session_id": null,
      "source_digest": "sha256:...",
      "transform_version": "memory_parse.v1"
    }
  }
}
```

Export mapping rules:

```txt
namespace
  scope {type: account}            -> personal (default, overridable)
  scope {type: project, project:p} -> project:<p>
  mapping overrides live in exporter config, not in lesson objects;
  emitted namespaces must match the gateway allowlist or project:<slug>
  pattern, validated at export

contentClass
  lesson.kind maps 1:1 onto the gateway enum; any kind outside the enum
  blocks export of that item (the gateway silently coerces unknown classes
  to "note", which would destroy classification - Qratum must never rely on
  that fallback)

metadata
  serialized metadata must stay under the gateway's 4000-char cap,
  validated at export; the gateway nests it under metadata.clientMetadata

content
  hard max 8000 chars, validated at export
```

Publishing uses the existing manual flow: `local_folder` publisher, publish
manifest, per-bundle approval, digest-verified output.

## Personal Memory Importer Contract

The importer lives in the `personal-memory` repo, not in Qratum. Qratum's
contract with it:

```txt
input        a memory_bundle folder
auth         a credential with memory:write scope (OAuth client or local
             header mode); never stored in or known to Qratum
behavior     for each memories.jsonl item, issue one memory_store call with
             the store object verbatim
ordering     none required; items are independent
idempotency  the gateway dedupes on (namespace, subject, content_hash) and
             upserts, so re-running a bundle is safe; the importer must also
             skip items already recorded as stored in the receipt
receipt      the importer writes import_receipt.jsonl next to the bundle:
             one line per item with lesson_id, content_sha256, and outcome
outcomes     stored | duplicate | blocked_secret | blocked_sensitive |
             content_too_large | failed
retry        failed items may be retried; blocked items must not be
             retried unedited
```

Receipts close the loop: Qratum can later ingest a receipt to mark lessons
`exported` versus blocked, and re-stage blocked lessons for editing.

This resolves the gateway roadmap's open push-versus-pull question in favor of
pull (bundle plus importer) for MVP.

## Trust Boundary Additions

| Boundary | Default |
| --- | --- |
| Lesson extraction | Tier 0 reads raw memory strings locally and emits lesson data class. Tier 2 reads redacted revisions only, under existing AI data policy. |
| Memory bundle export | Approved lessons only. Pre-flight enforced. Explicit consent per bundle. |
| Personal Memory gateway | External destination. Receives lesson data class only. Never raw, never redacted transcripts, never session content. |

## Failure Handling

Existing failure codes cover this pipeline (`NORMALIZATION_FAILED`,
`EXPORT_INELIGIBLE`, `EXPORT_FAILED`, `PUBLISH_FAILED`). One addition:

```txt
LESSON_BLOCKED_BY_POLICY    pre-flight content policy blocked a lesson at
                            candidate creation or export; not retryable
                            without an edit
```

## Milestone Mapping

```txt
P0 (now)   this spec; qratum.lesson.v1 schema; qratum.memory_export_item.v1
           schema; memory_bundle manifest shape; synthetic claude-ai-export
           fixtures (small fake export folder); schema validation tests

P2         claude-ai-export adapter (inventory, archive, normalize);
           Tier 0 deterministic memory parse (it is deterministic extraction,
           the same class as evidence checks)

P3         lesson review surface in the local app

P4         Tier 2 AI mining via facet lesson_candidates; Tier 1 decision

P5         memory_bundle export + local_folder publish; importer lands in
           the personal-memory repo against the frozen P0 contracts
```

A thin vertical slice (claude-ai-export import, Tier 0 parse, CLI review,
manual bundle export, importer) is deliverable without any AI work and covers
the highest-value content in the reference export. Sequencing that slice ahead
of full P2/P3 is a milestone decision, not a spec change.

## Non-Goals

- Direct HTTP push from Qratum to the gateway (future publisher).
- Automatic or scheduled memory sync.
- Embedding generation in Qratum (the gateway owns embeddings).
- Gateway-side staging namespaces as a Qratum dependency.
- Near-duplicate or semantic dedup of lessons (exact dedup only, consistent
  with the operational model).
- A generic conversation-export source. `claude-ai-export` is one concrete
  adapter with fixtures; a future ChatGPT export adapter would be its own
  adapter with its own fixtures.

## Fixture Plan

```txt
fixtures/claude-ai-export/
  export/
    conversations.json    2-3 tiny synthetic conversations, one with a
                          project, one with a summary, one resumed
    memories.json         synthetic account memory (with markdown structure
                          and one >8000-char string to exercise splitting)
                          plus 2 project memories
    projects/*.json       2 synthetic projects
    design_chats/*.json   1 synthetic design chat
    users.json            synthetic identity
  expected/
    lessons/*.json        expected Tier 0 candidates
    memory_bundle/        expected export output for an approved subset
```

Real export data must never be committed as fixtures.

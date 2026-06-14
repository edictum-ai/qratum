# Qratum Vault-First Revision

Status: Accepted 2026-06-14 (Arnold). Supersedes
`specs/current/memory-curation-pipeline.md` (now marked superseded) and
revises parts of `specs/current/operational-model-redesign.md` (exact edits
listed under "Spec hygiene"). Source analysis:
`docs/reviews/2026-06-12-memory-architecture/`.

Decisions already taken by Arnold (2026-06-12): all gateway verb work proceeds
in phases; import namespaces are `personal` + `coding` with an extensible
override map; unknown contentClass becomes a hard reject gateway-side; this
session produces specs only — implementation requires an explicit milestone
unlock.

## What Qratum Is

Qratum is the local librarian for AI session data. It captures, preserves,
normalizes, redacts, and — on demand — reviews and searches every AI session
across vendors. It is the system of record for **where session data came
from**: raw history, provenance, and deterministic derivations.

What Qratum is NOT:

- not a knowledge store (personal-memory owns durable knowledge)
- not a memory curation queue (no standing human approval lane; the
  operational model's own non-goal stands)
- not an insights generator (Claude Code `/insights` is native; Qratum
  archives its output instead of rebuilding it)
- not a publisher of governed bundles between one person's own tools
- not a corpus product, until a real corpus consumer exists

## Ecosystem Roles

```txt
edictum          the company and product: runtime process enforcement for AI
                 agents — approval boundaries, blocked actions, evidence-gated
                 progression. Governance patterns belong HERE when they are
                 product-shaped.

ductum           the internal agent factory: dispatches and orchestrates agent
                 work across repos. A producer of sessions Qratum captures and
                 a consumer of curation work (agents curate, human samples).

qratum           the librarian: vault + refinery for AI session data, local
                 first. System of record for where session data came from.

personal-memory  the living store: cross-agent durable knowledge, serving
                 (memory_search) plus stewardship (lifecycle verbs + hygiene).
                 System of record for what memories are now.
```

One home per job. The only bridge between qratum and personal-memory is the
gateway API, called directly by small scripts or commands holding a
keychain/env credential. No bundle/importer/receipt protocol between them.

## Why This Revision

The adversarial review (see `docs/reviews/2026-06-12-memory-architecture/`)
established, with repo evidence:

- The vault is the only component with an irreversibility clock: capture
  currently stores pointers only, covers ~15-20% of real session volume
  (ductum-only hook), and one captured transcript has already been deleted by
  Claude Code's 30-day cleanup.
- The review/lesson machinery had a 0% engagement baseline (18 real capture
  events, none ever processed; the only full run was a fixture demo).
- The bundle -> importer -> receipt bridge was a one-party trust protocol with
  invented counterparty behavior (a `duplicate` outcome the gateway cannot
  emit, an error vocabulary that doesn't match the gateway's).
- `internal/` is empty scaffolding; all Milestone A code lives flat in
  `cmd/qrt` and stores into repo-local `./.qratum/`, not a central workspace.

## The Vault (build first)

Goal: a transcript Claude Code deletes tomorrow is recoverable. Passive value;
survives abandonment gracefully.

### Workspace

Central workspace at `~/.qratum/` (migrating off repo-local `./.qratum/`).
Vault-minimum layout only — do not build the full v2 workspace:

```txt
~/.qratum/
  raw/
    blobs/sha256/ab/abc123....jsonl    content-addressed, immutable
    refs/raw_<digest12>.json           qratum.raw_ref.v1 records
  events/                              capture events (existing shape)
  state/vault.json                     last backfill cursor, stats
```

`qratum.raw_ref.v1` is reused from the operational model draft as-is. Raw
kinds gain: `source_export_bundle`, `source_memory`, `vendor_memory_dir`,
`vendor_insight_report`, `memory_import_receipt` — joining the operational
model's existing kinds (`main_transcript`, `subagent_transcript`,
`file_history_snapshot`, `source_insight_report`, `source_metadata`,
`unknown`). `qrt vault archive` uses the existing `source_metadata` for export
metadata files such as `projects/*.json` and `users.json`, and
`memory_import_receipt` for the curated-import receipt (see the companion
gateway spec) so the record of what curated content left the machine is
preserved next to the export blob it came from.

### Capture (global, copy-on-capture)

1. Global SessionEnd hook in `~/.claude/settings.json` (not per-project):
   `qrt hook claude-code`.
2. The hook reads the hook JSON from stdin, writes one capture event, and
   **copies the transcript file into the blob store** (stream-hash sha256,
   skip if blob exists, tmp+rename). A file copy is within the fast-hook rule;
   parsing, network, and LLM calls remain forbidden.
3. Degraded cases: missing `transcript_path` -> event recorded with
   `raw_missing: true`; copy failure -> event recorded, visible in
   `qrt status` — never silently swallowed.

### Operational ownership

The vault is worthless if installing and running it is manual tribal
knowledge. Capture, backfill, and backup are first-class commands, not
post-merge instructions. (Review finding: vault capture was operationally
unowned — global hook install, doctor, and backup verification were left as
out-of-band manual steps.)

- `qrt hook install` — idempotently add the SessionEnd hook to
  `~/.claude/settings.json` (the GLOBAL settings, not project-local); show the
  exact diff and confirm; detect and report an existing project-local hook so
  capture is not double-counted. `qrt hook status` reports whether the global
  hook is installed.
- `qrt vault doctor` — one command answering "is preservation actually
  working right now": global hook installed yes/no, last capture time, last
  backfill time and staleness, copy-failure count, blob count vs known
  transcript count drift, backup freshness, and any known machine not reporting.
- `qrt vault backup --verify` — back up and then prove the backup is
  restorable (digest-check a sample, or full round-trip for small vaults).
  A backup that has never been verified is not a backup.

### Backfill and archiving

- `qrt vault backfill` — idempotent inventory of
  `~/.claude/projects/**/*.jsonl` (and subagent transcripts) into blobs.
  Dedup by digest. First run captures the existing local transcripts (~267 at
  time of writing). Re-runnable; intended to be run periodically, not once
  (new sessions accrue; cloud/other-machine sessions may need a manual sync
  in first).
- `qrt vault archive <path>` — generic archiver for files/folders into blobs
  with a kind tag. First uses: the Claude.ai data export folder
  (`source_export_bundle` / `source_memory` / `source_metadata`), vendor
  memory dirs (`~/.claude/projects/*/memory/`, `~/.codex/memories/`), and
  `/insights` HTML output (`vendor_insight_report`). Vendors do the expensive
  mining; Qratum preserves their output as input.
- `qrt vault backup [--verify] <dest>` — wrapper (rclone/restic or plain copy)
  over `~/.qratum/`. Local-first must not mean single-disk.

### Multi-machine and cloud scope (stated, not solved)

Capture is a local-machine sensor. Two gaps are real and must be named, not
silently assumed away:

- **Second machine**: the ecosystem already runs agents on more than one host
  (e.g. a Mac mini running Hermes). Each machine needs its own
  `qrt hook install` and `qrt vault backfill`; merging two machines' vaults is
  blob-dedup-clean (content-addressed) but the install is per-machine. A
  second-machine runbook is a P2 deliverable.
- **Cloud sessions**: sessions that start and end on vendor infra (Claude Code
  on the web) never touch a local hook. v1 scope explicitly does NOT capture
  these; `qrt vault doctor` should state this limitation rather than implying
  full coverage. A pull-based cloud inventory is future scope.

### Acceptance

- Fresh session on any repo -> blob + ref + event exist within seconds of
  session end.
- `qrt hook install` is idempotent and shows its diff before writing.
- `qrt vault backfill` twice -> second run is a no-op (digest dedup).
- Deleting the source transcript does not lose data.
- `qrt vault doctor` warns on: no global hook, stale backfill, copy failures,
  unverified/missing backup, and states the cloud-session limitation.
- `qrt vault backup --verify` proves restorability, not just copy success.
- `qrt status` shows vault counts, last backfill, and copy failures.
- No raw content in logs or events (paths and digests only).

## The Refinery (kept, on demand)

The Milestone A pipeline (normalize -> deterministic redaction -> evidence ->
review -> report -> ADP export) is retained as on-demand tooling reading from
vault blobs. No daemon obligation, no automatic review, no review queue. It
runs when explicitly invoked and is allowed to stay unused.

## Later, Evidence-Gated (do not build now)

- **Local search (SQLite FTS)** over redacted/normalized sessions. Trigger:
  Arnold actually greps the vault twice. Note: this is the repo's first
  third-party Go dependency (CGO `mattn` vs `modernc.org/sqlite` tree) — an
  explicit supply-chain decision under `docs/supply-chain.md`, not ambient.
- **Thin claude-ai-export normalizer**: only if summary/conversation mining is
  ever wanted. Tier 1 (the 447 ready-made summaries) is preferred over
  full-conversation normalization; full Tier 2 mining is gated on a real
  consent + cost decision and must never feed a standing human queue.
- **Git-native candidate lane** (memory candidates as files, factory agents
  curate via PR, human samples and merges, push script delivers, commit
  history is the receipt). Trigger: real recurring candidates exist. Spec'd in
  `docs/reviews/2026-06-12-memory-architecture/PROPOSAL.md` (W5).

## Dead (do not resurrect in implementation prompts)

- bundle -> local_folder publisher -> importer -> receipt -> re-ingestion
  bridge between qratum and personal-memory
- `claude-ai-export` SourceAdapter as a product subsystem
- lesson staging lifecycle as a standing human approval queue
- `qratum.lesson.v1` / `qratum.memory_export_item.v1` wire schemas,
  `memory_bundle` artifact kind, `publish_memory_bundle` consent scope
- LessonBackend, VectorBackend / sqlite-vec / embedding policy, TiDB-direct
  projection, DuckDB analytics
- rebuilding insights; corpus export (until a consumer exists); per-session
  review as a primary output

## Spec Hygiene (edits to perform on acceptance)

`operational-model-redesign.md` must be edited in place — not overlaid:

1. "Locked Product Decisions": explicitly unlock and rewrite — output priority
   becomes preservation -> lessons-to-memory -> insights-harvest -> search ->
   review -> corpus; "Primary surface: local app" becomes "Primary surface:
   CLI + vault; app earned later"; "SQLite-backed search is the default local
   projection" gains the dependency caveat.
2. Cut: LessonBackend (ports list, adapters list, backend stack),
   VectorBackend/sqlite-vec, `tidb_remote` backend mode and config example,
   DuckDB references.
3. Consent section: full record schema becomes documented future shape;
   MVP behavior = config defaults + one-line audit event. Note explicitly that
   the schema deliberately mirrors Edictum semantics.
4. Honor the existing non-goal: no persistent approval/pending item queues —
   remove the latent contradiction with lesson candidates ("higher-risk
   suggestions stored for user review" becomes "factory-curated,
   human-sampled, batch-approved").
5. Milestones: replace the current P1-P5 ladder with vault-first sequencing
   (P1 = this vault spec; later milestones only on demonstrated pull).
6. `SPEC.md`: point to this file; current milestone remains
   P0-SPEC-AND-CONTRACTS until Arnold unlocks vault implementation.
7. New ADR 0010: vault-first; no one-person publish ceremony; the store owns
   its own curation; direct gateway calls with a locally-held credential are
   the integration mechanism.

## Companion Spec

The personal-memory side (gateway verb phases + the one-shot import script) is
specced in `personal-memory-gateway/docs/gateway-verbs-plan.md`. The two specs
are designed to be independently shippable; only the import script (W2)
depends on gateway Phase 1.

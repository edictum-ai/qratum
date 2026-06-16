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

## Plain-language glossary

A few terms recur throughout. Plain meaning first, exact term in parentheses.

- **Blob** — a single file of saved content stored under a name derived from
  its own contents, so identical content is never stored twice
  (content-addressed).
- **Content-addressed** — the file's storage name is a hash of its bytes
  (sha256). Same bytes in, same name out; that is how we deduplicate.
- **Immutable** — once written, a blob is never changed.
- **Runs every time without making changes if nothing is new** (idempotent) —
  you can run the command repeatedly and a second run does nothing extra.
- **Digest** — the sha256 hash that identifies a blob.
- **Redaction** — removing secrets (tokens, keys, etc.) from content before it
  leaves the raw stage.
- **Refinery** — the on-demand tooling that takes raw blobs and produces
  cleaned, normalized, redacted, reviewed output.
- **ADP export** — the refinery's final export format (carried over from the
  Milestone A pipeline).
- **Ref** — a small JSON record that points at a blob and records what it is
  (its kind, digest, source path).
- **Backfill** — scanning existing transcript files already on disk and pulling
  them into the vault.
- **Gateway** — the personal-memory API; the only way qratum content reaches
  personal-memory.
- **Hook** — a Claude Code callback that fires on an event (here, when a
  session ends).

## What Qratum Is

In one line: Qratum is the local librarian for your AI session data. It saves
every AI session from every vendor, keeps the originals safe, cleans them up,
strips secrets, and — only when you ask — lets you review and search them.

Put differently: Qratum is the system of record for **where session data came
from**. That means three things — the raw history, the provenance, and the
deterministic derivations of it.

What Qratum is NOT:

- not a knowledge store (personal-memory owns durable knowledge)
- not a memory curation queue (no standing human approval lane; the
  operational model's own non-goal stands)
- not an insights generator (Claude Code `/insights` is native; Qratum
  archives its output instead of rebuilding it)
- not a publisher of governed bundles between one person's own tools
- not a corpus product, until a real corpus consumer exists

## Ecosystem Roles

There are four systems, and each one has a single job. The table below names
each system and what it owns.

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

Each job lives in exactly one home. Qratum and personal-memory connect through
one channel only: the gateway API. Small scripts or commands call that API
directly, using a credential held in the keychain or an environment variable.
There is no bundle/importer/receipt protocol between the two systems.

## Why This Revision

The short version: only one part of the system is on a clock. That part is the
vault — wait too long and its data is lost forever. Everything else can be
added later. The adversarial review proved this with evidence from the repo
(see `docs/reviews/2026-06-12-memory-architecture/`):

- **The vault is the only part that loses data if we wait.** Capture today only
  stores pointers, not the content itself. It covers just ~15-20% of real
  session volume (the hook is ductum-only). And one captured transcript has
  already been deleted by Claude Code's 30-day cleanup.
- **The review/lesson machinery was never actually used.** Its engagement
  baseline was 0%: of 18 real capture events, none were ever processed. The
  only full run was a fixture demo.
- **The old bridge was a fake handshake.** The bundle -> importer -> receipt
  bridge was a one-party trust protocol with invented counterparty behavior. It
  assumed a `duplicate` outcome the gateway cannot emit, and used an error
  vocabulary that doesn't match the gateway's.
- **The code was not where the design said it was.** `internal/` was empty
  scaffolding. All Milestone A code lived flat in `cmd/qrt` and stored into
  repo-local `./.qratum/`, not a central workspace. That was the state at
  review time. Since then, some of it has shipped: `internal/{vault,workspace,
  claude,textdiff}` now exist, and capture/events use central `~/.qratum/`. One
  gap remains: the **derived refinery artifacts still land repo-local** in
  `./.qratum/`. Moving them to `~/.qratum/` is tracked as the
  derived-artifact-location fix (FIX-4 / D11 in `verification-and-trust-gate.md`).

## The Vault (build first)

Plain goal: if Claude Code deletes a transcript tomorrow, you can still get it
back. The vault delivers value passively, just by running. And it survives
neglect — even if you stop using everything else, the vault still protects your
data.

### Workspace

Everything lives in one central folder, `~/.qratum/` (we are moving off the old
repo-local `./.qratum/`). Build only the minimum layout the vault needs — do
NOT build the full v2 workspace:

```txt
~/.qratum/
  raw/
    blobs/sha256/ab/abc123....jsonl    content-addressed, immutable
    refs/raw_<digest12>.json           qratum.raw_ref.v1 records
  events/                              capture events (existing shape)
  state/vault.json                     last backfill cursor, stats
```

We reuse the existing `qratum.raw_ref.v1` ref format from the operational model
draft as-is.

Each blob needs a label saying what it is. That label is its "kind." We add new
raw kinds for the content the vault now handles: `source_export_bundle`,
`source_memory`, `vendor_memory_dir`, `vendor_insight_report`, and
`memory_import_receipt`. These join the kinds the operational model already
defines: `main_transcript`, `subagent_transcript`, `file_history_snapshot`,
`source_insight_report`, `source_metadata`, and `unknown`.

Two of these kinds get reused for archiving, so the record of what left the
machine sits right next to the export it came from. The `qrt vault archive`
command uses the existing `source_metadata` kind for export metadata files
(such as `projects/*.json` and `users.json`). It uses `memory_import_receipt`
for the curated-import receipt (see the companion gateway spec).

### Capture (global, copy-on-capture)

Plain idea: the moment any Claude Code session ends, copy its transcript into
the vault before anything can delete it.

1. Install one global SessionEnd hook in `~/.claude/settings.json` (global, NOT
   per-project): `qrt hook claude-code`.
2. When a session ends, the hook does three things. It reads the hook JSON from
   stdin, writes one capture event, and **copies the transcript file into the
   blob store**. The copy streams the file and hashes it with sha256, skips the
   write if that blob already exists, and writes to a temp file before renaming
   it into place. A plain file copy is allowed under the fast-hook rule.
   Parsing, network calls, and LLM calls remain forbidden in the hook.
3. When something goes wrong, record it — never swallow it silently. If
   `transcript_path` is missing, the event is still recorded, with
   `raw_missing: true`. If the copy fails, the event is still recorded, and the
   failure shows up in `qrt status`.

### Operational ownership

Plain problem: a vault nobody knows how to install or run is useless. So
installing, checking, backfilling, and backing up are real commands you can
run — not tribal knowledge buried in post-merge instructions. (Review finding:
vault capture was operationally unowned. Global hook install, doctor, and
backup verification were all left as out-of-band manual steps.)

- `qrt hook install` — adds the SessionEnd hook to `~/.claude/settings.json`
  (the GLOBAL settings, not project-local). It runs every time without making
  changes if the hook is already there (idempotent). It shows the exact diff
  and asks you to confirm before writing. It also detects and reports an
  existing project-local hook, so capture is not double-counted.
  `qrt hook status` reports whether the global hook is installed.
- `qrt vault doctor` — one command that answers "is preservation actually
  working right now?" It reports: whether the global hook is installed
  (yes/no), last capture time, last backfill time and whether it is stale,
  copy-failure count, drift between blob count and known transcript count,
  backup freshness, and any known machine that is not reporting.
- `qrt vault backup --verify` — backs up, then proves the backup can actually
  be restored. It digest-checks a sample, or does a full round-trip for small
  vaults. A backup that has never been verified is not a backup.

### Backfill and archiving

These commands pull existing files on disk into the vault.

- `qrt vault backfill` — inventories `~/.claude/projects/**/*.jsonl` (and
  subagent transcripts) into blobs, deduplicating by digest. It runs every time
  without making changes for files it has already captured (idempotent). The
  first run captures the existing local transcripts (~267 at time of writing).
  It is meant to be re-run periodically, not just once: new sessions keep
  accruing, and cloud or other-machine sessions may first need a manual sync
  into place.
- `qrt vault archive <path>` — a generic archiver that pulls any files or
  folders into blobs with a kind tag. First uses: the Claude.ai data export
  folder (`source_export_bundle` / `source_memory` / `source_metadata`), vendor
  memory dirs (`~/.claude/projects/*/memory/`, `~/.codex/memories/`), and
  `/insights` HTML output (`vendor_insight_report`). The vendors do the
  expensive mining; Qratum just preserves their output as input.
- `qrt vault backup [--verify] <dest>` — a wrapper (rclone/restic or a plain
  copy) over `~/.qratum/`. Local-first must not mean single-disk.

### Multi-machine and cloud scope (stated, not solved)

Plain limit: capture only sees the machine it runs on. Two gaps are real, and
we name them rather than pretend they don't exist:

- **Second machine**: the ecosystem already runs agents on more than one host
  (for example, a Mac mini running Hermes). Each machine needs its own
  `qrt hook install` and `qrt vault backfill`. Merging two machines' vaults is
  clean, because blobs are content-addressed and dedupe automatically — but the
  install itself is per-machine. A second-machine runbook is a P2 deliverable.
- **Cloud sessions**: sessions that start and end on vendor infrastructure
  (Claude Code on the web) never touch a local hook. v1 scope explicitly does
  NOT capture these. `qrt vault doctor` should state this limitation rather
  than imply full coverage. A pull-based cloud inventory is future scope.

### Acceptance

These are the conditions the vault must meet.

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

> **Acceptance status (2026-06-15):** 7 of these met by shipped code. Two gaps:
> (1) "Deleting the source transcript does not lose data" holds for the **raw
> blob** but not for *derived artifacts* — the refinery reads the live
> `transcript_path`, not the blob (FIX-3/D6a). (2) "`backup --verify` proves
> restorability" is not actually met: `verifyTree` re-hashes the live source
> instead of comparing against the recorded `ref.Digest`, and there is no
> round-trip restore (D8). Both owned by `verification-and-trust-gate.md`.

## The Refinery (kept, on demand)

Plain idea: the cleanup-and-review tooling stays, but only as something you run
by hand. It never runs on its own, and it's allowed to sit unused.

The Milestone A pipeline (normalize -> deterministic redaction -> evidence ->
review -> report -> ADP export) is retained as on-demand tooling that reads
from vault blobs. There is no daemon obligation, no automatic review, and no
review queue. It runs only when explicitly invoked, and is allowed to stay
unused.

> **Not yet wired (2026-06-15):** "reading from vault blobs" is the intended
> design but is NOT built — the refinery resolves and requires the live
> `transcript_path` (`daemon.go`). Wiring blob fallback is FIX-3 / D6a in
> `verification-and-trust-gate.md`.

## Later, Evidence-Gated (do not build now)

Plain rule: do not build any of these until there is real evidence someone
needs it. Each has a named trigger.

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

When this is accepted, edit `operational-model-redesign.md` in place — do not
overlay it:

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

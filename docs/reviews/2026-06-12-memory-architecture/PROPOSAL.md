# Proposed Shape (post-review)

Status: Accepted 2026-06-14 (Arnold). Synthesized from the seven
adversarial reviews in this directory, plus an independent GPT-5.5 meta-review
(2026-06-12) that re-measured the evidence and stress-tested the replacement.
Nothing here is implemented or spec-canonical yet.

## The spine: four things, then stop

The meta-review's sharpest finding is that even this plan risks treating a
founder-attention problem as an architecture problem. The genuinely urgent
object is narrow. In priority order:

1. **Stop losing transcripts** — vault capture + backfill + backup (W1).
2. **Make memory writes truthful and reversible** — gateway Phase 1, plus the
   minimum Phase 2 correction verbs (W3). Enough that adding facts can't
   corrupt the store.
3. **Import the 17 curated strings once** — the one-shot script (W2), with a
   durable receipt.
4. **Stop.** Everything else (gateway Phase 3-4, search, refinery expansion,
   the git-native lane, insights) must earn its next hour with measured need —
   it is recorded, not queued.

Every item below maps to that spine. If a workstream isn't one of the four,
it is parked by default.

## The system, redrawn

```
            agents / sessions (Claude Code, Codex, Ductum, Claude.ai)
                 |  produce                            ^  read at runtime
                 v                                     |
  +--------------------------------+      +---------------------------------+
  | QRATUM — the librarian          |      | PERSONAL-MEMORY — the living    |
  | system of record for where      |      | store; system of record for     |
  | session data CAME FROM          |      | what memories ARE NOW           |
  |                                 |      |                                 |
  | vault: global capture +         |      | gains verbs: delete, supersede, |
  |   content-addressed raw archive |      |   created/updated discriminator,|
  | refinery: normalize/redact/     |      |   write-time near-dup check,    |
  |   evidence (on demand)          |      |   per-namespace grants          |
  | local FTS search (later,        |      | owns live-store curation        |
  |   evidence-gated)               |      |   (roadmap curation track)      |
  +--------------------------------+      +---------------------------------+
                 |                                     ^
                 |  small scripts / commands calling   |
                 +--- the gateway API directly --------+
                      (git-native candidate lane when
                       ongoing volume justifies it)

  EDICTUM: "governed memory writes" recorded as a product pattern
           (approval boundaries for what agents may persist) — not built here.
```

One home per data class survives, with one correction: **the store is alive.**
Curated knowledge lives in personal-memory AND personal-memory must be able to
curate it (verbs), or the home is a landfill.

## What is explicitly dead

Killed by the review (do not resurrect in implementation prompts):

- The bundle -> local_folder publisher -> foreign importer -> receipt ->
  receipt-re-ingestion bridge (one-party trust protocol; ceremony as control).
- `claude-ai-export` SourceAdapter as a P2 product subsystem (the export is
  archived as blobs; an adapter is built thin and later, only if Tier-1/2
  mining ever earns it).
- Tier-0-as-pipeline: the 17 memory strings are a one-shot script, not a
  permanent subsystem.
- Lesson staging lifecycle as a standing human approval queue (violates the
  op model's own non-goal; the curation math is fatal).
- `qratum.lesson.v1` / `qratum.memory_export_item.v1` wire schemas,
  `memory_bundle` artifact kind, `publish_memory_bundle` consent scope.
- LessonBackend, VectorBackend/sqlite-vec/embedding policy, TiDB-direct,
  DuckDB (already decided; the spec prose must actually be cut — see W4).
- Rebuilding insights (Claude Code `/insights` is native; archive its output).
- Corpus export until a real consumer exists.
- Per-session review as a primary output (0% engagement baseline; Edictum
  already enforces process at runtime).

## Workstreams

### W1 — Stop the bleeding: the vault (days; highest urgency)

The only component with an irreversibility clock. One captured transcript is
already gone; capture covers ~15-20% of real session volume (ductum only).

1. Global SessionEnd hook in `~/.claude/settings.json` (not per-project).
2. Copy-on-capture: hook copies the transcript to a content-addressed blob
   under `~/.qratum/raw/blobs/sha256/...` immediately (a file copy is within
   the fast-hook rule). Pointer events alone are proven insufficient.
3. Backfill: one-shot inventory copying the existing ~263 transcripts in
   `~/.claude/projects` into the vault.
4. Archive the Claude.ai export folder as blobs (an hour; archive != normalize).
5. Also archive vendor memory dirs as raw kinds: `~/.claude/projects/*/memory/`,
   `~/.codex/memories/`, `/insights` HTML output — vendors do the mining free.
6. `qrt backup` (restic/rclone wrapper or documented Time Machine coverage).
   "Local-first" must stop meaning "single-disk".

Acceptance: a transcript Claude Code deletes tomorrow is recoverable.
The vault survives abandonment gracefully — archives appreciate, queues rot.

### W2 — The 17 memory strings (1-2 days; after or alongside W3 grants)

One-shot script in the personal-memory repo (`scripts/import-claude-memories.ts`):

1. Parse `memories.json` + `projects/*.json`.
2. Split on **bold markers + blank-line blocks** (the real structure — there
   are zero headings), propagating the bold header as a prefix onto children,
   with a minimum-size merge for orphan labels. The 5,629-char section gets a
   second-level blank-block split.
3. Emit `candidates.md`; review/edit in `$EDITOR` — one sitting (~30-60 items).
   The editor pass, not the parser, produces atomicity.
4. Push via `memory_store`, calling `evaluateContentPolicy` in-process first
   (same code, zero drift, no Go mirror, no parity fixture). Items tripping
   the sensitivity patterns (the account memory trips legal, one project
   memory trips medical) get rephrased or stay local.
5. Record outcomes incl. the returned `memoryIds` to `receipt.jsonl` —
   grounded in real responses.
6. Namespaces: collapse to `personal` + `coding` (+ at most 1-2 obvious
   project slugs that match repo names agents actually query). Thirteen
   `project:<claude.ai-name>` namespaces = rows nobody ever retrieves.

Idempotent for free via gateway dedup. Throw the script away after, or keep it
as the template for the W5 push step.

### W3 — Gateway verbs (personal-memory; small, high-leverage; jumps the roadmap queue)

In order:

1. **Per-namespace grants** (security first: `private` is currently readable
   by any connected identity; static client->namespace map, ~30 lines in
   `authorizeTool`).
2. **Soft `memory_delete`** (column and scope already exist; the prerequisite
   for every other lifecycle feature).
3. **Store discriminator**: read `affectedRows`, return
   `created | updated | resurrected`; skip re-embedding on exact duplicates
   (also fixes updated_at pollution and double Bedrock spend).
4. **Write-time near-dup check** (~50 lines: the embedding is already computed
   at store time; nearest-neighbor before insert; within-threshold writes
   refresh `updated_at` instead of creating rows). The one pattern all prior
   art converged on; kills the dominant degradation mode at the source.
5. Provenance: upsert merges metadata instead of replacing; resurrect becomes
   explicit, not a side effect of byte-identical re-store.
6. Search returns `contentClass`; stop the silent coercion to `note` (reject
   or warn).
7. Re-embed backfill script + startup warning when rows exist under a
   non-current embedding provider/model/dims triple (silent-invisibility bomb).
8. Docs honesty: `contracts.md` promises a confirmation flow for
   `confirmation_required` that does not exist — implement a `confirmed: true`
   audited override or rename the error class.

### W4 — Spec hygiene (qratum; half a day, factory-dispatchable)

The decisions currently exist in nobody's repo; three sources of truth disagree.

1. Delete `specs/current/memory-curation-pipeline.md` (contains verified-wrong
   contracts: heading split, `duplicate` outcome, `blocked_sensitive`,
   LessonBackend reference) — superseded by this proposal.
2. Revise `operational-model-redesign.md` as **edits, not an overlay**:
   explicitly unlock and rewrite the Locked Product Decisions (priority order
   becomes preservation -> lessons-to-memory -> insights-harvest -> search ->
   review -> corpus; primary surface is CLI + vault, app demoted); cut
   LessonBackend/VectorBackend/tidb_remote/DuckDB prose; honor the
   "no persistent approval queues" non-goal; consent = documented future shape
   + audit log line; note SQLite = first third-party Go dependency (explicit
   supply-chain decision when FTS lands).
3. Update `SPEC.md` pointers; add ADR 0010 recording this review's outcome
   (vault-first; no one-person publish ceremony; the store owns its own
   curation; direct gateway calls with keychain token are the integration).

### W5 — Ongoing lesson lane (later; evidence-gated; do not build yet)

Trigger: real recurring candidates exist (e.g., vendor memory harvest or
factory sessions produce material worth promoting) — not before.

Shape (steelmanned winner for this user): **git-native curation.**
Candidates land as files in a small git repo (frontmatter: namespace, class,
provenance; body: content). Ductum agents curate — split, dedup, classify —
as PRs; Arnold samples and merges; a push script (the W2 script generalized)
delivers merged candidates via `memory_store` with `supersedes` once W3 ships
it; the commit history is the receipt and the audit trail. Never a standing
per-item human queue: the human is policy-author + sampler, not approval
station. Historical mining (Tier 1's 447 summaries, Tier 2 conversations) only
topic-scoped/on-demand, never full-corpus batch, and only into durably-true
classes until the ranker is recency-aware.

### W6 — Strategic note (record, don't build)

"Governed memory writes" — approval boundaries, provenance requirements, and
staged writes for what agents may persist to shared memory — is an
Edictum-shaped product pattern, just validated by Anthropic's Managed Agents
memory (per-write audit logs, immutable versions) inside one vendor's walls.
Record it in edictum-harness as a wedge candidate Edictum can serve across
vendors. Stop re-deriving it in personal repos.

## Decision points for Arnold

1. Accept this shape (and the deaths list) as the direction?
2. Proceed with W1 + W2 immediately (W1 is losing data weekly)?
3. W3 jumps the personal-memory roadmap queue — yes/no, and how much of 1-8?
4. Namespace collapse choice in W2 (`personal`+`coding` vs keeping a few
   project slugs)?
5. Gateway breaking-change appetite (reject unknown contentClass vs warn)?

# Dispatch Prompts (post-review execution)

Copy-paste prompts for Ductum/Claude Code sessions. Each starts with a
mandatory review phase and carries its own abort conditions.

Pre-dispatch checklist (Arnold, manual — agents enforce both):

1. Acceptance gate: flip the Status line to "Accepted (date)" in PROPOSAL.md,
   specs/current/qratum-vault-first.md, and
   personal-memory-gateway/docs/gateway-verbs-plan.md. Until then, every
   prompt's Phase A aborts.
2. Commit first: the spec and review files each prompt reads must be committed
   in both repos (`git status --short` clean for those paths). Agents abort on
   dirty or untracked inputs.

Scope: this plan touches exactly two repos — qratum and personal-memory. It is
personal infrastructure and is deliberately kept separate from edictum (the
product).

Dispatch order: P1 (spec hygiene) first or parallel with P3 (gateway Phase 1);
P2 (vault) after P1; P4 (import script) only after BOTH P3 is deployed AND the
Claude.ai export is archived into the vault (P2's `qrt vault archive`) —
unless Arnold explicitly waives the archive precondition.

---

## PROMPT 1 — qratum: spec hygiene (docs only)

```
You are working in /Users/acartagena/project/qratum. Documentation-only task:
make the on-disk specs match the post-review direction (acceptance-gated
below). Read AGENTS.md first and follow it.

PHASE A — REVIEW FIRST (no edits):
1. Read docs/reviews/2026-06-12-memory-architecture/README.md and PROPOSAL.md
   (review record + direction). Verify PROPOSAL.md's Status line says
   Accepted — ABORT if it still says proposal.
2. Read specs/current/qratum-vault-first.md (the revision spec; verify its
   Status line says Accepted, abort otherwise),
   SPEC.md, specs/current/operational-model-redesign.md IN FULL, and the
   SUPERSEDED banner in specs/current/memory-curation-pipeline.md.
3. Produce a written edit plan: for every change required by the "Spec
   Hygiene" section of qratum-vault-first.md, list the exact sections/lines of
   operational-model-redesign.md you will modify or cut and what replaces
   them. Cross-check the undead references listed in review findings
   06-mechanics.md #11 and 07-reality-audit.md F4 (LessonBackend in
   ports/adapters/backend stack, sqlite-vec/VectorBackend, tidb_remote config,
   DuckDB, the Locked Decisions list, consent as P0 deliverable, the
   approval-queue contradiction).
4. ABORT with a findings report instead of editing if the spec files changed
   materially since 2026-06-12 (git log/diff), if qratum-vault-first.md
   contradicts the review record anywhere, or if any file you depend on is
   uncommitted/untracked (`git status --short`).

PHASE B — IMPLEMENT (markdown edits only):
1. Edit specs/current/operational-model-redesign.md IN PLACE:
   - rewrite "Locked Product Decisions" with an explicit unlock note dated
     2026-06-12: output priority preservation -> lessons-to-memory ->
     insights-harvest -> search -> review -> corpus; primary surface CLI +
     vault (local app demoted to earned-later); SQLite caveated as the first
     third-party dependency decision
   - cut LessonBackend, VectorBackend/sqlite-vec/embedding policy, the
     tidb_remote backend mode + config example, and DuckDB everywhere
   - consent: full record schema = documented future shape (deliberately
     mirroring Edictum semantics); MVP behavior = config defaults + one-line
     audit event
   - resolve the contradiction with the "no persistent approval/pending item
     queues" non-goal: lesson candidates become factory-curated,
     human-sampled, batch-approved
   - replace the P1-P5 milestone ladder with vault-first sequencing per
     qratum-vault-first.md
2. Update SPEC.md pointers (canonical = operational-model-redesign.md +
   qratum-vault-first.md; milestone stays P0-SPEC-AND-CONTRACTS).
3. Write docs/decisions/0010-vault-first-and-direct-gateway-integration.md in
   the ADR format of 0001-0009: vault-first; no one-person publish ceremony;
   the store owns its own curation; direct gateway calls with a locally-held
   credential are the integration mechanism.
4. Do NOT touch Go code, fixtures, or schemas. Do NOT resurrect anything on
   the Dead list in qratum-vault-first.md.

VERIFY: run `make build && make test` to prove nothing broke (docs-only, but
prove it); then grep EXACTLY these files — SPEC.md,
specs/current/operational-model-redesign.md,
docs/decisions/0010-vault-first-and-direct-gateway-integration.md — for
"LessonBackend|sqlite-vec|VectorBackend|tidb_remote|DuckDB" and show zero
hits. qratum-vault-first.md, the superseded pipeline spec, and docs/reviews/
intentionally retain those terms in their Dead/history sections — do not
"fix" them.
```

---

## PROMPT 2 — qratum: vault build (W1; runtime unlock)

```
You are working in /Users/acartagena/project/qratum. Arnold explicitly unlocks
runtime implementation for this task: the milestone for this work is the Vault
section of specs/current/qratum-vault-first.md. Everything else in AGENTS.md
still applies (supply-chain rules, fast-hook rule, fixture-driven tests,
make verify).

PHASE A — REVIEW FIRST:
1. Read specs/current/qratum-vault-first.md (the contract; verify its Status
   line says Accepted — ABORT if it still says proposal), AGENTS.md,
   docs/supply-chain.md, and docs/reviews/2026-06-12-memory-architecture/README.md
   for context.
2. Audit current reality: cmd/qrt/hook.go and daemon.go (storage is repo-local
   ./.qratum; internal/ is empty directories), the capture event shape,
   fixtures/claude-code/. Confirm the review's claims still hold; report deltas.
3. Produce an implementation plan covering: workspace migration (repo-local ->
   ~/.qratum with a QRATUM_HOME override for tests/fixtures), hook
   copy-on-capture, backfill, archive, backup, status additions — and which
   Milestone A behaviors (make demo) must keep working.
4. ABORT with a report if the spec conflicts with existing code in a way the
   spec does not anticipate, or if the spec/review files you depend on are
   uncommitted (`git status --short`).

PHASE B — IMPLEMENT (scope = the Vault section, nothing more):
1. Central workspace ~/.qratum (env override QRATUM_HOME) with vault-minimum
   layout: raw/blobs/sha256/, raw/refs/, events/, state/vault.json. Migrate
   hook + daemon paths; keep make demo green (fixtures may set
   QRATUM_HOME=./.qratum).
2. Hook copy-on-capture: stream-sha256 the file at transcript_path, copy into
   blobs (skip if digest exists; tmp+rename), write a qratum.raw_ref.v1 ref.
   Degraded cases per spec: missing transcript_path -> event with
   raw_missing=true; copy failure -> recorded and visible in qrt status; never
   silently swallowed. Hook stays fast: file copy yes; parsing/network/LLM no.
3. Operational ownership (the vault must install and self-check itself, not
   rely on manual post-merge steps):
   - `qrt hook install` / `qrt hook status` — idempotently add the SessionEnd
     hook to GLOBAL ~/.claude/settings.json, show the diff before writing,
     detect an existing project-local hook to avoid double-capture.
   - `qrt vault doctor` — report: global hook installed?, last capture, last
     backfill + staleness, copy-failure count, blob-vs-transcript drift,
     backup freshness; and STATE the cloud-session limitation rather than
     implying full coverage.
   - `qrt vault backup [--verify] <dest>` — --verify proves restorability
     (sample digest-check or small-vault round-trip), not just copy success.
4. `qrt vault backfill` — idempotent inventory of ~/.claude/projects/**/*.jsonl
   (and subagent transcripts) into blobs; second run is a no-op; re-runnable
   (periodic, not one-shot).
5. `qrt vault archive <path> [--kind K]` — file/folder archiver with kind tag
   (source_export_bundle, source_memory, source_metadata, vendor_memory_dir,
   vendor_insight_report, memory_import_receipt).
6. `qrt status` gains vault counts, last backfill time, copy failures.
7. Multi-machine/cloud scope: write a short second-machine runbook
   (per-machine hook install + backfill; vaults merge blob-dedup-clean) and
   state explicitly that cloud-only sessions are out of v1 scope.
8. Schemas + fixtures: add/extend qratum.raw_ref.v1 schema with the new raw
   kinds + fixture examples; golden tests for copy-on-capture, backfill
   idempotency, archive kinds, and hook-install idempotency (diff shown, no
   double-write).
NON-GOALS: no SQLite, no new third-party dependencies, no normalizer changes,
no lessons, no publishers, nothing from the Dead list.

VERIFY: make verify green (full CI mirror); make demo green; manual
end-to-end with output shown: pipe fixtures/claude-code/hook-session-end.json
through the hook under a temp QRATUM_HOME, prove blob+ref+event exist, delete
the source transcript, prove the data survives. Do not claim done without
showing these outputs.

POST-MERGE (Arnold, manual): add the global SessionEnd hook to
~/.claude/settings.json and run `qrt vault backfill` once.
```

---

## PROMPT 3 — personal-memory: gateway Phase 1

```
You are working in /Users/acartagena/project/personal-memory/personal-memory-gateway.
This is a deployed, security-critical service: OWASP rigor, fail closed, no
shortcuts. pnpm always.

PHASE A — REVIEW FIRST:
1. Read docs/gateway-verbs-plan.md — this task is Phase 1 ONLY; verify its
   Status line says Accepted, ABORT if it still says proposal. Also read
   docs/contracts.md, docs/roadmap.md, docs/threat-model.md.
2. Read the code you will change: src/domain/access.ts,
   src/domain/content-policy.ts,
   src/application/use-cases/persistent-memory.ts,
   src/infrastructure/tidb/memory-repository.ts,
   src/interfaces/mcp/phase0-server.ts, src/config.ts. For evidence context,
   read (read-only, sibling repo)
   /Users/acartagena/project/qratum/docs/reviews/2026-06-12-memory-architecture/05-lifecycle.md
   and 06-mechanics.md and 07-reality-audit.md.
3. Verify the review claims against current code: grants absent in
   authorizeTool; upsert discards affectedRows and clears deleted_at;
   contentClass silently coerces to "note". Report drift.
4. Inventory current writers/clients (config, OAuth client records) to assess
   the blast radius of the contentClass breaking change, and confirm whether a
   unit-test harness exists (the review found smoke scripts only).
5. ABORT with a report if the plan conflicts with code reality, or if the
   proposed grant map would lock out an active production client.

PHASE B — IMPLEMENT PHASE 1 ONLY:
1.1 Per-namespace grants: config-driven client -> namespace map;
    deny-by-default for non-default namespaces on unknown clients; "private"
    matched only by explicit grant, never by wildcard; new errorClass
    namespace_forbidden; audited like other denials. Update docs/contracts.md.
1.2 Store outcome discriminator + duplicate short-circuit: existence check on
    (namespace, subject, content_hash) BEFORE embedding. Live duplicate ->
    no-op: outcome "duplicate", no re-embed (no Bedrock call), no updated_at
    bump. Soft-deleted duplicate -> denied with errorClass duplicate_of_deleted
    including the existing memoryId (ends silent resurrection).
    structuredContent gains `outcome: "created"|"duplicate"` and always
    returns memoryIds.
1.3 Reject unknown contentClass: keep normalization (lowercase/underscore),
    then unknown -> denied, errorClass invalid_content_class with the valid
    enum in the message. Hard reject — Arnold's decision; no coercion.
2. Tests: introduce a real unit-test harness for these paths (content policy,
   grants, store outcomes) if none exists. Phase 1 does not ship untested.
3. PR description: state the contentClass change is breaking by design; list
   known clients and observed classes from audit data if available.
NON-GOALS: no delete/restore/supersedes (Phase 2), no near-dup check
(Phase 3), no staging namespaces, no bulk endpoint, no memory_records schema
changes.

VERIFY: pnpm test green (including the new harness); existing smoke scripts
pass against a local run; demonstrate with actual request/response output:
namespace_forbidden, duplicate no-op (same memoryId, no updated_at change),
duplicate_of_deleted, invalid_content_class. Note: Phase 1 ships no delete
path and no MCP route sets deleted_at — seed a soft-deleted row directly in
the test database to exercise duplicate_of_deleted. Do not claim done without
showing these.
```

---

## PROMPT 4 — personal-memory: one-shot import script (after Phase 1 deploys)

```
You are working in /Users/acartagena/project/personal-memory/personal-memory-gateway.
PRECONDITIONS — verify both; ABORT if either fails and Arnold has not
explicitly waived it: (a) gateway Phase 1 (grants, outcome discriminator,
contentClass reject) is merged and deployed; (b) the Claude.ai export has been
archived into the Qratum vault (`qrt vault archive`), so the script's
source_digest provenance can reference the archived blob rather than a loose
Downloads folder.

PHASE A — REVIEW FIRST:
1. Read docs/gateway-verbs-plan.md, section "The Curated Import Script" — that
   section is the contract; verify the doc's Status line says Accepted, ABORT
   otherwise.
2. Read /Users/acartagena/project/qratum/docs/reviews/2026-06-12-memory-architecture/07-reality-audit.md
   finding F3: the real structure is ZERO headings; bold pairs 4-9 per string;
   account-memory bold sections of roughly 495/716/974/5630/2282 chars —
   these digits are delimiter-accounting-sensitive, so re-measure yourself and
   document your length accounting; do not hardcode them. Your splitter
   targets THIS shape, not generic markdown.
3. Inspect the export ONLY via jq (lengths, keys, marker counts). NEVER print,
   log, or commit memory content — it is personal data. Path:
   /Users/acartagena/Downloads/data-9b2793dd-8fb2-4d38-908e-de973f9257ee-1781210321-00ab4a5c-batch-0000
4. Confirm in-process reuse of evaluateContentPolicy from scripts/ with the
   gateway's exact policy subject (content + "\n" + JSON.stringify(metadata)).

PHASE B — IMPLEMENT scripts/import-claude-memories.ts with three subcommands:
- parse: memories.json + projects/*.json -> bold-marker sectioning with
  header-prefix propagation onto blank-line child blocks; orphan labels under
  ~40 chars merge forward; sections over 8000 chars get a second-level
  blank-block split. Emit candidates.md (editable blocks: namespace,
  contentClass, content) with inline flags for items tripping sensitivity
  patterns. Namespace map: account -> personal, projectDefault -> coding,
  plus an empty-but-present overrides table.
- push: read the edited candidates.md; per item run evaluateContentPolicy
  in-process; assert the authenticated subject equals the expected subject
  constant BEFORE the first store; call memory_store with
  metadata.origin="claude-ai-export" and metadata.source_digest; append to
  receipt.jsonl: candidate id, content sha256, gateway memoryIds, outcome/
  errorClass VERBATIM (gateway success outcomes are created | duplicate),
  namespace used, subject used, source export digest, and namespace_map_version.
  Re-running push skips items whose receipt records created or duplicate.
- status: summarize receipt vs candidates (created/duplicate/blocked/remaining).
receipt.jsonl is a TESTED format (schema + round-trip test), not a scratch log:
a malformed receipt fails loudly, never silently re-pushes. After a push run,
archive the receipt into the Qratum vault
(`qrt vault archive <receipt> --kind memory_import_receipt`) next to the
export blob it references — this is the only durable record of what curated
content left the machine and what id it became. The script is NOT disposable
until `status` after `push` is boring.
Tests use SYNTHETIC fixtures shaped like the real data (bold + blank lines,
zero headings, one section over 8000 chars). Never commit real export content.

VERIFY: pnpm test green; dry-run parse against the real export printing ONLY
counts and a section-size histogram. The actual push run is Arnold's to
execute by hand.
```

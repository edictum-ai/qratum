# Backlog — per project

Status: nothing started. Derived from PROPOSAL.md (the spine), the two specs,
and the 8 reviews. The spine: (1) stop losing transcripts, (2) make memory
writes truthful + reversible, (3) import the 17 strings once, (4) stop.

Legend: [spine N] = which pillar · (P#) = dispatch prompt · gate = precondition.

---

## Pre-flight (Arnold, manual — before any dispatch)

- [ ] Decide the fork: is qratum *insurance* (preserve + import + stop) or *the
      dream* (preserve as foundation, then design the curation lane)? This
      only changes what happens AFTER the foundation; the foundation is
      identical either way.
- [ ] Flip Status lines to "Accepted (date)" in: PROPOSAL.md,
      qratum-vault-first.md, gateway-verbs-plan.md. (Every dispatch prompt
      aborts until this is done.)
- [ ] Commit the spec + review files in both repos (agents abort on dirty
      inputs).

---

## QRATUM (local Go librarian)

### A. Spec hygiene — docs only (P1) · [spine 4: keeps the map honest]
- [ ] Edit operational-model-redesign.md IN PLACE: unlock & rewrite "Locked
      Product Decisions" (priority -> preservation/lessons/insights-harvest/
      search/review/corpus; primary surface CLI+vault; SQLite dependency
      caveat).
- [ ] Cut from that spec: LessonBackend, VectorBackend/sqlite-vec, tidb_remote
      mode+config, DuckDB.
- [ ] Consent -> documented future shape + one-line audit event (note: mirrors
      Edictum semantics).
- [ ] Resolve the "no persistent approval queues" contradiction (lesson
      candidates = factory-curated, human-sampled, batch-approved).
- [ ] Replace P1-P5 ladder with vault-first sequencing.
- [ ] Replace stale counts (18/263/1-missing) with dated measurements.
- [ ] Update SPEC.md pointers; write ADR 0010 (vault-first; no one-person
      publish ceremony; store owns its curation; direct gateway calls).
- [ ] Verify: make build && make test green; grep SPEC.md +
      operational-model-redesign.md + ADR 0010 show zero dead-term hits.

### B. Vault build — runtime (P2, after A) · [spine 1: THE urgent one]
- [ ] Central workspace ~/.qratum (env override QRATUM_HOME); migrate off
      repo-local ./.qratum.
- [ ] Hook copy-on-capture: stream-sha256 transcript -> blob (skip-if-exists,
      tmp+rename) + qratum.raw_ref.v1 ref. Degraded cases recorded, never
      swallowed. Hook stays fast.
- [ ] `qrt hook install` / `qrt hook status` — idempotent GLOBAL settings
      edit, shows diff, detects project-local double-capture.
- [ ] `qrt vault doctor` — hook installed?, last capture, backfill staleness,
      copy failures, blob-vs-transcript drift, backup freshness; STATES the
      cloud-session limitation.
- [ ] `qrt vault backfill` — idempotent, re-runnable (the ~267 existing
      transcripts; second run no-op).
- [ ] `qrt vault archive <path> [--kind]` — kinds: source_export_bundle,
      source_memory, source_metadata, vendor_memory_dir, vendor_insight_report,
      memory_import_receipt.
- [ ] `qrt vault backup [--verify] <dest>` — --verify proves restorability.
- [ ] `qrt status` gains vault counts / last backfill / copy failures.
- [ ] Second-machine runbook (per-machine install+backfill; vaults merge
      blob-dedup-clean); state cloud-only sessions out of v1 scope.
- [ ] Schemas + golden tests: raw_ref.v1 + new kinds; copy-on-capture,
      backfill idempotency, archive kinds, hook-install idempotency.
- [ ] Verify: make verify + make demo green; show end-to-end (capture ->
      delete source -> data survives).
- [ ] POST-MERGE (Arnold, manual): `qrt hook install`; run `qrt vault backfill`
      once; archive the Claude.ai export.

### C. Parked — do NOT build until a trigger fires · [spine 4]
- [ ] Local SQLite FTS search — trigger: you grep the vault twice (first Go
      third-party dep = explicit supply-chain decision).
- [ ] Thin claude-ai-export normalizer (Tier-1 summaries preferred) — trigger:
      summary/conversation mining actually wanted.
- [ ] Git-native curation lane — trigger: real recurring candidates exist.
      THIS is the "dream" design target if the fork lands on curation.

---

## PERSONAL-MEMORY (deployed AWS gateway)

### D. Gateway Phase 1 — security + truthful writes (P3) · [spine 2: runway]
- [ ] Per-namespace grants: client->namespace map; deny-by-default for
      non-default namespaces; `private` only by explicit grant (never
      wildcard); new errorClass namespace_forbidden; audited. Update
      contracts.md.
- [ ] Store outcome discriminator: existence check before embed; live dup ->
      no-op (outcome "duplicate", no re-embed, no updated_at bump);
      soft-deleted dup -> denied duplicate_of_deleted with existing memoryId;
      structuredContent gains outcome + always returns memoryIds.
- [ ] Reject unknown contentClass (hard reject, errorClass
      invalid_content_class) — Arnold's decision; breaking by design.
- [ ] Introduce a real unit-test harness (currently smoke-only) for these
      paths.
- [ ] Verify: pnpm test green; demonstrate namespace_forbidden, duplicate
      no-op, duplicate_of_deleted (seed a soft-deleted row), invalid_content_
      class with real request/response output.

### E. Gateway Phase 2 — correction verbs (after D) · [spine 2: runway]
- [ ] memory_delete (soft) — scope memory:delete; decide issuance (proposed:
      dedicated memory-admin OAuth client).
- [ ] memory_restore (explicit; replaces upsert resurrection).
- [ ] supersedes[] on memory_store (transactional replace; the update
      primitive).
- [ ] Provenance merge-not-replace on dup/update (preserve first-writer
      origin; record last_source_client).

### F. One-shot import script (P4, gate: D deployed + export archived in vault)
      · [spine 3: the 17 strings]
- [ ] scripts/import-claude-memories.ts — parse (bold-marker split, header
      prefix, orphan merge, >8000 second-level split) -> candidates.md.
- [ ] review = Arnold edits candidates.md in $EDITOR (one sitting); sensitivity
      hits flagged inline.
- [ ] push: in-process evaluateContentPolicy over the exact gateway subject;
      subject pinning assert before first store; memory_store with origin +
      source_digest.
- [ ] receipt.jsonl = TESTED contract (schema + round-trip): candidate id,
      sha256, memoryIds, outcome verbatim (created|duplicate), namespace,
      subject, source digest, namespace_map_version. Re-run skips
      created/duplicate.
- [ ] Archive receipt into the vault (`--kind memory_import_receipt`) next to
      the export blob.
- [ ] Synthetic fixtures only (bold+blank, zero headings, one >8000 section);
      never commit real content.
- [ ] Verify: pnpm test green; dry-run parse prints counts + size histogram
      only. Actual push run = Arnold by hand.

### G. Gateway Phase 3-4 — EVIDENCE-GATED, parked · [spine 4]
- [ ] Write-time near-dup check — trigger: measured duplicate incident OR hot
      namespace > ~200 rows.
- [ ] contentClass + createdAt in search results — trigger: classification
      actually used in retrieval.
- [ ] Re-embed backfill — trigger: embedding provider/model change (ship the
      startup-warning tripwire with Phase 2).
- [ ] confirmed:true override — trigger: a real curated item is blocked by the
      sensitivity pattern (doc fix in contracts.md is enough until then).
- [ ] Retrieval report (audit-events join) — trigger: weeks of read signal /
      deciding whether Phase 3 is needed.
- [ ] Content-policy fixture export — trigger: a second (e.g. Go) policy
      implementation is actually written.

---

## Scope note

This plan touches exactly two repos — qratum and personal-memory. Both are
personal infrastructure and are kept deliberately separate from edictum (the
product). An earlier draft added an edictum-harness "governed memory writes"
note; it was removed as scope creep — that pattern is the same governance-DNA
leakage the review warned about, just in reverse. If the idea ever matters, it
will resurface naturally while working on Edictum; it does not belong in this
plan.

---

## Dispatch order

```
Pre-flight (accept + commit)
   |
   +--> P1 qratum spec hygiene  ──┐   (parallel)
   +--> P3 gateway Phase 1       ─┘
          |
   P2 qratum vault  (after P1)
          |
   E gateway Phase 2  (after P3, minimal)
          |
   P4 import script  (after P3 deployed AND export archived by P2)
          |
   STOP — G / C only on a measured trigger
```

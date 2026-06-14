# Adversarial Review: Memory Curation Architecture (2026-06-12)

Seven independent adversarial reviewers (mixed Opus/Fable models, separate
contexts, distinct lenses) attacked the proposed Qratum -> personal-memory
memory-curation architecture: the draft in
`specs/current/memory-curation-pipeline.md` plus the ecosystem split
(vault/refinery/staging in Qratum, bundle -> importer -> receipt bridge,
curated store in personal-memory).

## Reports

| # | Lens | File |
|---|------|------|
| 1 | Alternative shapes steelman | [01-alternatives-steelman.md](01-alternatives-steelman.md) |
| 2 | Prior art (memory systems, data engineering, PKM) | [02-prior-art.md](02-prior-art.md) |
| 3 | Adoption realism / opportunity cost | [03-adoption-realism.md](03-adoption-realism.md) |
| 4 | Platform trajectory (2026 memory wave) | [04-platform-trajectory.md](04-platform-trajectory.md) |
| 5 | Lifecycle and missing pieces | [05-lifecycle.md](05-lifecycle.md) |
| 6 | Pipeline mechanics / contracts | [06-mechanics.md](06-mechanics.md) |
| 7 | Reality audit (claims vs repos) | [07-reality-audit.md](07-reality-audit.md) |
| 8 | Independent meta-review (GPT-5.5, different lineage) | [08-gpt-meta-review.md](08-gpt-meta-review.md) |

The consolidated recommendation is in [PROPOSAL.md](PROPOSAL.md). Dispatch
prompts in [DISPATCH-PROMPTS.md](DISPATCH-PROMPTS.md).

**Measurement correction (2026-06-12, from report 08):** several counts in
reports 01-07 were point-in-time and have drifted. Current measured values:
22 capture events (not 18), 2 missing transcript paths (not 1), 267 local
transcripts (not 263), ~76 MB. `internal/` is 15 empty dirs (not 16). These
are volatile by nature; spec hygiene (Prompt 1) replaces them with dated
observations. The structural claims (pointer-only capture, no gateway
mutation verbs, silent contentClass coercion, exposed `private` namespace,
export shape) were all re-verified and hold.

Process note: reports 5-7 were first launched on Opus, killed mid-run at the
operator's request, and relaunched on Fable. The Opus kill-time partial
findings (zero markdown headings in the real memory strings; no `duplicate`
outcome in the gateway response; no supersede/retract lifecycle transition)
were independently re-derived by the Fable runs — useful cross-validation.

## Convergent verdict

All seven reviewers independently converged on the same core diagnosis:

1. **One-party trust protocol.** The bundle -> foreign importer -> receipt
   bridge is an inter-organizational integration pattern applied between two
   processes owned by one person on one laptop. The "Qratum never holds
   credentials" boundary has no enforcement mechanism (same UNIX user, same
   keychain). The receipt loop exists only to patch the blindness the design
   itself created.
2. **Wrong asset class.** Memories are mutable state needing stewardship
   (write -> retrieve -> reconcile -> rewrite), not artifacts to ship. The
   design is a one-way ADD line into a store with no mutation verb anywhere
   (no delete/update in the gateway; no supersede/retract in Qratum).
   "Qratum is the system of record for where memories came from; nothing is
   the system of record for what memories are now."
3. **Root cause:** Edictum's governed-handoff idiom (evidence-gated
   progression, approval boundaries, publish manifests) self-applied to a
   personal data chore. The governance pattern is an Edictum product
   opportunity, not Qratum personal machinery.

## Key disproven claims (blockers)

- The review pipeline is behaviorally dead: only one end-to-end run ever (a
  fixture demo); 18 real capture events sat unprocessed for 10 days; hook
  wired in one project only (~15-20% session coverage).
- The vault does not exist in code: `internal/` is 16 empty directories;
  capture stores pointers, archives nothing; one captured transcript was
  already deleted by Claude Code's 30-day cleanup.
- The curation queue math is fatal: Tier 2 = 30-90 hours of solo triage; the
  operational model's own non-goal ("no persistent approval queues") is
  violated by the lesson staging lane.
- Contract fictions: the `duplicate` receipt outcome is unimplementable; the
  spec invented `blocked_sensitive` (gateway says `confirmation_required`);
  receipt ingestion is in no milestone; receipts drop the gateway memoryIds;
  Tier-0 "heading split" targets structure absent from the real data (zero
  headings; bold markers + blank lines only).

## What survived attack (keep)

Medallion shape (content-addressed raw -> deterministic transforms -> curated
export); ETL-over-ELT because raw must never reach the destination;
vault-first sequencing; refusal to rely on the gateway's silent contentClass
coercion; mandatory 8000-char splitting; pre-flight at review time (the real
dataset trips the legal and medical patterns); raw-never-leaves boundary;
exactness of every load-bearing number about the export and gateway (~80% of
the proposal verified grounded; hallucination concentrated at integration
seams).

## personal-memory bugs surfaced (flagged upstream)

- `private` namespace readable/writable by any identity with memory scopes
  (no per-namespace grants).
- Upsert overwrites `source_client` + `metadata` wholesale: provenance is
  destructible by any byte-identical writer.
- Re-store of identical content resurrects soft-deleted rows.
- Embedding provider/model env change makes the whole store silently
  invisible to search; no re-embed job exists.
- `confirmation_required` names a confirmation mechanism that does not exist;
  `contracts.md` documents it as if it did.
- Search never returns or ranks on `contentClass` (write-only field).

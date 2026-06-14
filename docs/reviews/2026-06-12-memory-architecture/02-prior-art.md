# Review 2: Prior Art

- Lens: test the design against systems and disciplines that already solved this problem class
- Model: Fable
- Date: 2026-06-12

---

Grounding read: `specs/current/memory-curation-pipeline.md`, `specs/current/operational-model-redesign.md` (dedup, backends, review sections), `personal-memory-gateway/docs/{architecture,contracts,roadmap}.md`, `personal-memory-gateway/src/domain/content-policy.ts`. Web-verified the named memory systems.

## Findings

### F1 — No write-time reconciliation: the one thing every shipped memory system converged on is absent
**VERIFIED / blocker for the bulk-import use, serious for trickle writes.**
Prior art is unanimous. mem0's core contribution is not extraction, it's the *update phase*: every candidate is compared against semantically similar existing memories and an LLM picks ADD / UPDATE / DELETE / NOOP. Zep/Graphiti made supersession the primitive: edges carry `valid_at`/`invalid_at`, contradictions invalidate old facts without deleting history. LangMem's memory manager does consolidation ("this fact about Bob is no longer true -> overwrite"). Letta's sleep-time agents exist to rewrite messy memory. Even ChatGPT's consumer memory updates/combines/removes saved memories. The proposal's pipeline only ADDs, into a gateway whose `memory_update`/`memory_delete` are unimplemented, with dedup on exact `(namespace, subject, content_hash)` — and `subject` is constant (one user), so it's effectively exact-content dedup. "Prefers pnpm" vs "always uses pnpm" = two rows forever. The gateway roadmap *knows* this (its Curation Track lists semantic dedupe, contradiction, staleness) but defers it to an external daily optimizer that does not exist. Net: the architecture ships the bulk-write path before any reconciliation path exists anywhere.
**Fix (cheapest first):** the importer already authenticates to the gateway — have it call `memory_search` per item before `memory_store` and emit a new receipt outcome `near_duplicate_suspected` with the neighbor IDs, routed back into Qratum staging for human triage. The embedding infra already exists; this is one extra call per item. Medium-term: `supersedes` field in `qratum.memory_export_item.v1` + gateway `memory_update`.

### F2 — Historical backfill into recency-blind retrieval
**VERIFIED / serious.** Zep's entire thesis is that conversation-mined facts are temporally scoped. The gateway's search is brute-force cosine filtered by namespace/model — no recency or salience weighting (roadmap defers decay). Bulk-importing facts mined from a years-spanning export makes a 2024 fact rank equal to a current one. The two decisions are individually defensible and jointly toxic: backfill *specifically* punishes a recency-less ranker. The pipeline dutifully carries `source_digest` and timestamps in provenance metadata, but nothing ranks on them.
**Fix:** before any Tier-2 backfill, either add `updated_at`-aware ranking gateway-side, or restrict historical mining to durably-true classes (`preference`, `profile`, `workflow_rule`) and exclude time-bound classes (`project_checkpoint`, `decision`) from the historical tiers.

### F3 — Asymmetric quality gates: five gates on one door, zero on the other
**VERIFIED / serious.** The Qratum path has: deterministic extraction -> pre-flight policy -> human review -> per-bundle consent -> receipt. Meanwhile the gateway's own MCP instructions tell *every connected agent* to `memory_store` after work — uncurated runtime writes flowing into the **same namespaces** (`personal`, `project:<slug>`) as the hand-approved imports. Data-warehouse practice puts the quality gate at the layer boundary, not on one feeder; a gold layer's quality is the min() of its write paths. Your carefully curated rows will sit beside, and be retrieved alongside, junk-drawer agent writes — the exact failure the gateway roadmap names.
**Fix:** either implement the roadmap's write-time bar at the gateway before the backfill lands, or partition by provenance (`metadata.origin: qratum` is already there — but origin lives in metadata, not in anything retrieval can rank or filter on; promote curated-vs-scratch to a first-class column or namespace convention).

### F4 — Two extraction tiers over overlapping sources + exact dedup = guaranteed near-dupes
**VERIFIED / serious.** `memories.json` is Claude's own synthesis of the same conversations Tier 2 will later mine. The same fact will arrive twice with different wording and pass exact dedup both times — mem0's update phase exists precisely because reworded-equivalent facts are the *common case* in conversation-mined data. The spec's non-goal "near-duplicate or semantic dedup… exact only" is consistent with the operational model but deviates from all prior art with no stated reason beyond consistency.
**Fix:** same mechanism as F1; additionally, semantic self-dedup *within* a bundle at export time (Qratum-side, no AI needed if you route it through the importer's search call).

### F5 — No retrieval feedback loop: curation effort has no value signal
**SUSPECTED / serious for effort allocation.** No component measures whether a stored memory is ever retrieved. You will hand-approve hundreds of items and never learn which twelve mattered. Notably, the data already exists: `memory_audit_events` records `memory_ids` per `memory_search` call. This is a missing report, not missing capture.
**Fix:** periodic join of audit retrievals against `memory_records`; feed hit-rates back into what classes/scopes are worth mining. This is the cheapest possible "salience" implementation and de-risks F10.

### F6 — Human review gate pre-write
**DESIGN-VINDICATED.** No shipped system has a pre-write human gate (ChatGPT and Claude offer only post-hoc review/edit/delete; mem0/Zep/Letta/LangMem are fully automatic). For a single-user system whose owner's product thesis is literally "approval before risky transition, evidence-gated progression," this is a coherent differentiator, not a smell. The cost it imports is F10.

### F7 — The vault->refinery->curated shape itself
**DESIGN-VINDICATED.** This is a medallion architecture done by the book: content-addressed immutable bronze, versioned deterministic transforms (`transform_version`), schema-on-read raw / schema-on-write at the export boundary, backends-as-rebuildable-projections, reprocess-from-bronze as the recovery primitive, tombstones, digest-keyed idempotency. Also: ETL (curate-before-load) over ELT is *correct* here, because the binding constraint is data class — raw must never reach the destination. Privacy-driven ETL is the recognized legitimate exception to the ELT trend. No deviation found; this layer is the strongest part of the design.

### F8 — Human decisions keyed to content hashes do not survive transform-version bumps
**VERIFIED / serious.** Lesson identity = `scope + sha256(normalized content)`. The spec guarantees "re-running extraction over the same input digests produces no duplicate candidates" — true only for the *same* `transform_version`. Ship `memory_parse.v2` with different split boundaries and every prior approve/reject decision orphans; the entire queue resurfaces. This is the known MDM/data-labeling problem of steward decisions surviving match-rule changes; the discipline's answer is to key human judgments to stable semantic identity (source span) rather than derived-content hash.
**Fix:** record review decisions against `(raw_ref digest, source span)` or keep a cross-version rejected-fingerprint set consulted at candidate creation. Cheap now, expensive after the first v2.

### F9 — Deleting the LessonBackend projection
**SUSPECTED / annoyance now, serious at Tier-2 scale.** Filesystem-only staging works for Tier 0 (tens of atoms from 8 memory strings). It does not work for triaging thousands of Tier-2 candidates — no filter/sort/group surface. The operational model itself says projections are cheap and rebuildable; deleting this one saves almost nothing.
**Fix:** keep files as truth; rebuild the SQLite projection when candidate count crosses ~100. Defer, don't delete.

### F10 — Inbox bankruptcy
**VERIFIED / blocker if Tier 2 runs as a batch; currently mitigated by sequencing.** Decades of PKM evidence (GTD inbox failure, the Zettelkasten "collector's fallacy," Evernote/Readwise graveyards) predict exactly what happens when a capture queue fed by 11,962 messages of backlog plus ongoing sessions terminates in mandatory per-atom review by one human: the queue grows monotonically, review becomes aversive, the system is abandoned at staging, and the deployed store receives only the Tier-0 trickle. The designs that survived are *process-on-demand*: evergreen notes written when working on the topic, progressive summarization touching notes only on revisit — never process-all.
**Fix:** make Tier 2 retrieval-triggered or topic-scoped ("mine conversations about X when project X is active"), never a full-corpus batch into the human queue; add a hard queue cap (mining halts while `suggested > N`). The spec's Tier-0-first sequencing is correct — the danger is the implicit promise that Tier 2 eventually runs as a batch.

### F11 — Atomizing a living document into an append-only ledger
**VERIFIED / serious.** Claude.ai's account/project memories are *maintained summary documents* — regenerated and editable, the closest prior art being Letta's memory blocks (named, bounded, rewritten-in-place). Tier 0 snapshots one version, shatters it into atoms, and appends them to a store with no supersede path. The next export (the source document keeps evolving) re-splits into mostly-overlapping, differently-worded atoms: exact dedup misses them (F4), the queue refills with near-dupes of approved items, and stale atoms from the old snapshot persist. This is the slowly-changing-dimension problem handled as pure inserts — no Type-2 versioning, no current-flag. Zettelkasten adds the second half: atomic notes earn value through *linking and revisitation*; these atoms have provenance back-links but no lateral structure and no revisit trigger. Atomization without linking is a shoebox of index cards.
**Fix:** treat `profile`/account-memory content as document-shaped: one row per topic block, superseded on re-import (re-import of same scope+block replaces, via F1's mechanism). Atomize only genuinely independent facts.

### F12 — Bundle -> importer -> receipt is EDI between one person's two laps
**PARTIALLY VINDICATED / annoyance.** Integration practice names this precisely: Hohpe & Woolf "File Transfer" style plus a functional acknowledgment (EDI 997). It is the pattern for crossing *trust and organizational* boundaries. Here producer, reviewer, approver, importer, and consumer are the same human on (mostly) the same machine. Two of its justifications are real, not cargo-culted: credential isolation (Qratum holds no gateway creds — a genuine security boundary) and the data-class firewall (raw never leaves). But it imports file-transfer's known costs: dual validation logic (F13) and manual reconciliation. The phrase "Qratum can *later* ingest a receipt" is the weak joint — exported-state truth lives in three places (lesson status, receipt JSONL, gateway rows) with no automatic reconciliation; outbox/saga practice says one system of record plus mandatory reconciliation.
**Fix:** make receipt ingestion a required step of the export flow (`qrt export memories` is incomplete until the receipt is consumed; block re-export of a scope with an unconsumed receipt).

### F13 — Cross-engine regex mirror
**SUSPECTED (future) / annoyance — checked the actual code.** `content-policy.ts` patterns are currently RE2-safe (non-capturing groups only; no lookaround/backreferences), so the Go mirror is feasible *today*. The trap: TS is the authority and JS regex permits lookbehind/backreferences that Go's RE2 `regexp` cannot compile at all. One convenience-motivated lookahead in the authority and the mirror silently can't mirror.
**Fix:** the fixture contract must pin the RE2-compatible subset, and both repos must run a dual-engine conformance test over shared match/non-match cases — your own "parity by fixture, not by memory" rule, applied here.

### F14 — Flagged in passing (standing flag-bugs rule, gateway side)
- `content-policy.ts:56-60`: invalid `contentClass` silently coerces to `"note"`. The Qratum spec rightly refuses to rely on this, but every *other* writer hits it silently, eroding classification. Return an error instead.
- `confirmation_required` denial has no confirmation mechanism anywhere in the contract — high-sensitivity content is a hard block in practice. Implement the confirm path or rename the error class to match reality.
- Keyword patterns (`\bmedical\b`, `\bcontract dispute\b`) will false-positive on benign project memory ("medical-imaging project"); fail-closed is right, but expect review-time noise at Tier-2 scale.
- Namespace migration hazard: the gateway roadmap plans reclassifying `personal` -> `global`/`coding`/`project:*` and explicitly says "do not bulk rename." Qratum's exporter maps `account -> personal`. Run the backfill after the namespace redesign or the curated import lands in a namespace scheduled for reclassification — a second manual migration of your own curated data.

---

## (a) The one pattern the proposal most needs to adopt

**mem0's update phase / Graphiti's invalidation: write-time reconciliation against semantic neighbors.** Every system that has operated conversation-derived memory at any scale independently converged on it; it is the load-bearing wall of the category, and its absence here amplifies four other findings (F2, F4, F10, F11). The minimum viable version needs no new infrastructure: the importer calls `memory_search` before each `memory_store`, and high-similarity hits become a receipt outcome that flows back into the human staging queue — which converts your existing review gate from an add-only filter into a reconciliation surface. That single change is the difference between a memory system and a sediment layer.

## (b) The unnamed "odd shape"

**The design is a supply chain bolted onto something that needs a metabolism.** Every shipped memory system is a *loop*: write -> retrieve -> reconcile -> rewrite. This design is a *line*: mine -> review -> approve -> bundle -> receipt -> done. All of the system's considerable rigor — manifests, consent, digests, receipts, dual-engine policy fixtures — sits on the producer side of the wire, and **no component owns the data after it lands**: gateway-side curation is an unstarted roadmap item, and Qratum's responsibility ends at the receipt. The shape exists because the designer reached for his own native idiom — Qratum is an artifact-publishing pipeline, Edictum is governed handoff, so memory delivery got modeled as a governed publish (one-person EDI, with producer, approver, importer, and consumer all the same human). That idiom is *right* for the trust boundary (credential isolation, raw-data firewall — keep those) and *wrong* for the asset class: memories are mutable state to be maintained, not artifacts to be shipped. The fix is not to redraw the pipeline; it is to close the loop at both ends — reconciliation at write (a), retrieval feedback to curation (F5) — so the line becomes the loop's bottom half rather than the whole picture.

## Sources

- Mem0: Building Production-Ready AI Agents with Scalable Long-Term Memory (arXiv 2504.19413)
- mem0 ADD-only conflict-resolution issue #4896
- Zep: A Temporal Knowledge Graph Architecture for Agent Memory (arXiv 2501.13956)
- Graphiti: Knowledge graph memory for an agentic world (Neo4j blog)
- Letta: Memory Blocks / Agent Memory
- LangMem SDK launch (LangChain) / background quickstart
- OpenAI: Memory FAQ / Reference saved memories
- Anthropic: Memory announcement

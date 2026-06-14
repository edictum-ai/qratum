# Review 3: Adoption Realism / Opportunity Cost

- Lens: will this actually get used, and should it be built now
- Model: Fable
- Date: 2026-06-12

---

## FINDINGS

### F1. The review pipeline is already dead, and the repo proves it — VERIFIED / BLOCKER

The single most important evidence in this review: **no real session has ever been processed through Qratum.**

- `qratum/.qratum/` contains exactly one full artifact chain (session, redacted, evidence, review, report, export), all named `claude-session-0001`, all dated June 2. That session ID and transcript path (`fixtures/claude-code/transcript-verification-gap.jsonl`) come from `fixtures/claude-code/hook-session-end.json`. **The only end-to-end run in the tool's life was a fixture demo.**
- Meanwhile 18 *real* capture events accumulated in `.qratum/events/` between June 2 and June 12 (one per day-ish). Zero were ever normalized, reviewed, or reported. The user had `qrt debug daemon run-once` available for three weeks and never pointed it at his own data.
- The capture hook is wired in exactly one place: `ductum/.claude/settings.local.json` (SessionEnd -> `cd .../qratum && ./bin/qrt hook claude-code`). His actual session volume is 67 transcripts in the last 7 days across `~/.claude/projects` (263 total, 71MB). Capture covers maybe 15-20% of his real activity — and it's mostly the *factory's* sessions, not his interactive work.

Severity is blocker because the proposed architecture's recurring-value loop (capture -> review -> lessons) depends on a behavior — opening review output — for which there is now a 3-week, 18-sample, 0% engagement baseline. The design treats "review" as proven; it is disproven.

**Fix:** anything whose value requires him to *open* something on a schedule must be cut or made push-based and passive. Sequencing already demotes review surfaces — finish the thought and remove them from the critical path entirely.

### F2. The curation math kills the lesson queue, and the spec already knows it — VERIFIED / BLOCKER

Touch-count per candidate: read, judge kind/scope, maybe edit, approve/reject — 30s for pre-curated Tier-0 strings, 1-3 min for AI-mined candidates (those require remembering the source conversation to judge accuracy/staleness).

Queue arithmetic from the specs' own numbers (`specs/current/memory-curation-pipeline.md`):
- **Tier 0:** 10,097-char account memory + 7 project memories -> 50-150 candidates -> **25-75 min, one sitting. Survivable exactly once.**
- **Tier 1:** 447 summaries -> 100-400 candidates if mined.
- **Tier 2:** 556 conversations x 3-5 -> 1,700-2,800 candidates -> **30-90 hours of solo human triage. Dead on arrival.** No solo founder with a startup has ever drained a 2,000-item personal review queue. The first skipped week is terminal; this queue starts pre-skipped.
- **Ongoing:** at his real volume (~9-10 sessions/day if capture went global), 70-200 candidates/week, forever. The threshold where a solo founder stops opening the queue is roughly "more than one sitting clears" (~20-30 items). He'd cross it in week one.

The damning detail: `operational-model-redesign.md` (~line 2280) lists **"Persistent approval/pending item queues"** as an explicit non-goal — and the lesson staging lifecycle (`suggested -> staged -> approved -> exported`, statuses on the filesystem, receipt reconciliation re-staging blocked items) **is precisely a persistent approval queue.** The design's own immune system flagged this and was overridden one spec later.

**Fix:** there must never be a standing human queue. One-time Tier-0 batch reviewed in `$EDITOR` (the candidates file *is* the review surface). For any future mining: he owns an agent factory — the factory curates, the human samples (approve batch after reviewing a random 10%; trust tiers: `memories.json` content is *already curated by Claude.ai* and should be near-auto-approve; AI-mined gets factory review + human spot-check). Receipt reconciliation must be a command, never a task.

### F3. The immediate want is being used to justify a pipeline it doesn't need — VERIFIED / SERIOUS

The actual want: ~17 curated memory strings inside personal-memory. The gateway's `memory_store` MCP tool is live **on this machine right now** (the stdio bridge is connected in this very environment; `docs/contracts.md` confirms namespaces `personal`, `project:<repo>` and dedupe on `(namespace, subject, content_hash)` are shipped — re-running is already idempotent, gateway-side).

A 1-day path exists: script splits `memories.json` on markdown structure -> 50-150 lines of JSONL -> he reviews in a text editor for an hour -> script calls `memory_store` per line -> log file as receipt. Done. Idempotency, dedupe, and content policy are *already enforced by the gateway* — Qratum's pre-flight mirror, bundle manifests, and consent scope re-implement protections the destination already has.

The proposed path: accept a 2,283-line operational model + 470-line pipeline spec (both currently **untracked working-tree files** — `git status` shows `?? specs/current/operational-model-redesign.md`), freeze P0 schemas (19 of them) + fixtures, build a claude-ai-export adapter, Tier-0 extractor with `transform_version`, lesson lifecycle, new consent scope, `memory_bundle` format, publish manifest, importer in a second repo, receipt format, reconciliation loop. Via the factory that's realistically 5-10 dispatched prompts across 2 repos over 1-3 weeks — and every merge consumes the one resource that matters, his review attention. Milestone A took ~10 prompts over 3 days *plus* hardening commits; this is bigger.

End-state comparison: after the pipeline, he has the same ~17 memories in the gateway as after the script — plus seven new contracts whose recurring justification (ongoing session mining) is the thing F1/F2 show he won't do.

**Fix:** ship the script (or a single throwaway `qrt` subcommand if it must be in-repo). Let the pipeline earn existence later from a demonstrated recurring need.

### F4. Vault-first is right, and the repo just proved why — DESIGN-VINDICATED (with a serious caveat)

Of the 19 captured events, **one transcript is already gone**: `~/.claude/projects/-Users-acartagena-project-ductum/cd83084c-9103-44d7-89b6-a0526378c73d.jsonl` no longer exists. Capture stores pointers, not copies; Claude Code rotates history; ~5% loss in 10 days, empirically. The vault thesis ("source tools may delete history") is vindicated by the design's own data — and the current pointer-only capture is indicted by it. Every week spent freezing P0 contracts is a week of silent transcript loss.

The vault is also the only component that survives abandonment gracefully: a content-addressed blob pile needs no triage, no queue, no app, and is still valuable in three years. Review queues rot; archives appreciate.

**Caveat (serious):** the vault as deployed archives a biased ~15-20% slice (ductum only). If vault is the payoff, the highest-value change is not a schema — it's a *global* SessionEnd hook plus copy-on-capture. That's days, not milestones.

### F5. Qratum's review duplicates what Edictum already does for him, live — VERIFIED / SERIOUS

The one fixture review's findings — "no successful verification after the final file change," "edits after last verification," failed `go test` ignored — are exactly the class of process violations his own product enforces *at runtime* (read-before-edit, verify-before-push, no-done-before-CI-green). His agents run under Edictum workflow gates; the factory's merge gates demand evidence. A post-hoc HTML report telling him yesterday's session skipped verification is strictly weaker than the runtime gate that already blocked it. For *this user*, deterministic review has negative pull — which is why the 18 events sat unprocessed. (For other users it might matter. There are no other users.)

### F6. Two curation layers are being designed for the same stream — SUSPECTED / SERIOUS

`personal-memory-gateway/docs/roadmap.md` lists "Memory curation design" as the approved next product theme (write-time salience, semantic dedupe, consolidation, contradiction handling — "design not started") plus planned `staging:<target>` namespaces. Qratum's spec simultaneously plans human staging on its side. Two curation layers, both specced, neither built, same memory stream, same one human. One of them will die unused; the coordination cost gets paid regardless. The Qratum spec's decision not to depend on gateway staging is correct but doesn't resolve which side owns curation.

### F7. Dead-weight inventory in the v2 spec — VERIFIED / ANNOYANCE-to-SERIOUS

Fraction of `operational-model-redesign.md` with a real forcing function for an audience of one: maybe 25%. Specifically dead-on-arrival for this user: local app auth machinery (bootstrap nonce, HttpOnly cookies, CSRF, route capability classes, CSP — for localhost, single user), consent records with durations and revocation semantics (he is asking himself for permission), tombstones, migration manifests, OTel span catalog, corpus/ADP export (a training-data play with no consumer today), publishers beyond `local_folder`, insights section taxonomy. Each is well-designed; none has a user. If Qratum ends 20%-built-then-abandoned, the right 20% is: global hook + content-addressed vault + the one-shot memory export. That abandonment is harmless — the archive keeps working at zero attention.

Vindicated spec decisions worth keeping on record: bundle-mediated pull with no gateway credentials in Qratum (correct trust boundary in intent); mandatory 8000-char splitting; refusing the gateway's silent contentClass->"note" coercion; Tier-0-first with the spec's own admission that "Tier 0 alone delivers most of the immediate value"; hooks-stay-fast; no-hidden-processing; the buried "thin vertical slice" paragraph, which is the correct product hiding inside the wrong one.

---

## (a) MINIMUM DESIGN THAT SURVIVES THIS USER'S ACTUAL BEHAVIOR

1. **Global capture + copy-on-capture vault, this week.** SessionEnd hook in `~/.claude/settings.json` (not per-project), hook copies the transcript to a content-addressed blob immediately (it's a file copy — within the fast-hook rule). No P0 schema freeze as a prerequisite; the RawRef shape from the spec is fine as-is. This is the only component with passive value, and F4 shows data is being lost *now*.
2. **One-shot memories export, ~1-2 factory days.** Script or single subcommand: `memories.json` -> split -> JSONL candidates -> human edits the file in an editor (one sitting, F2 math says it fits) -> importer script in the personal-memory repo calls `memory_store` per line -> log receipt. Gateway dedupe makes it idempotent for free. No lesson lifecycle, no consent scope, no bundle manifest, no schemas.
3. **Stop.** No refinery, facets, insights, corpus, app, or review surfaces until the vault generates a real pull. The first genuine recurring want will be *search over the vault* ("find that session from March"), not review. Build that when he actually greps the blob pile twice.
4. **If session mining ever happens:** factory-curated, human-sampled, batch-approved. Never a standing per-item human queue.

Weekly-open honesty: of the six outputs, the realistic answer is **none weekly**. Lessons-to-memory is one-time-plus-occasional; insights is maybe-monthly-novelty; reviews/library/corpus/metrics/governance-evidence have zero demonstrated pull. Design for "opens nothing, loses nothing."

## (b) THE UNNAMED ODD SHAPE

**It is a two-party trust protocol with only one party.** Producer, refinery, staging, approval, consent records, publish manifests, bundle-mediated pull, foreign importer, receipts, reconciliation — that is the architecture of a governed data supply chain between organizations that don't trust each other. Both ends of this chain are the same person on the same laptop, moving his own notes into his own memory service. The one real trust boundary (raw transcripts must never reach the gateway) is worth ~50 lines of exporter code, not seven contracts.

The shape has a traceable origin: it is **Edictum's conceptual machinery unconsciously self-applied**. Evidence-gated progression, approval boundaries, consent, auditability, governed handoff — the founder's product vocabulary, and his Edictum-marinated agent factory's house style, auto-stamped onto a personal tool. That's why a 17-string import grew a nine-contract pipeline, why the spec's own non-goals list ("no persistent approval queues") got overridden by a staging queue one document later, and why the design's most degrading move went unnoticed: it hires its sole user — a founder whose scarce resource is review attention — as the full-time approval station inside his own refinery. The repo already recorded his answer to that job offer: 18 events, 10 days, zero opened.

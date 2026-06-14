# Review 1: Alternative Shapes Steelman

- Lens: beat the proposal with a simpler design that does the same job
- Model: Opus (first wave; completed before the relaunch)
- Date: 2026-06-12

---

## What is actually true right now (ground truth, not spec fiction)

Before scoring anything, the single most important fact: **almost none of this exists.** Qratum has a working Milestone-A pipeline (capture/normalize/redact/evidence/report) and a 2,283-line v2 spec that is 0% implemented (current milestone is P0, "lock contracts"). The gateway is real and deployed. The Claude.ai export is real and on disk. The 17 curated memory strings are real.

So the proposal is not "refactor a system." It is "decide how much new machinery to build to move ~17 already-curated strings (plus a future trickle of coding-session lessons) into a store that accepts them via one function call: `memory_store(namespace, content, contentClass, metadata)`."

That framing is the whole game. Hold it the entire way down.

Key confirmed facts from code:

- **Gateway write path is exactly one MCP tool: `memory_store(namespace, content, contentClass, metadata)`.** Dedup is real: natural key is `(namespace, subject, content_hash)` with `ON DUPLICATE KEY UPDATE` (true upsert, un-soft-deletes). So re-running is genuinely idempotent.
- **Namespace allowlist is enforced today**: `personal`, `coding`, `hermes`, `claude-code`, `codex`, `chatgpt`, `claude-browser`, `private`, plus `project:<slug>`. The staging namespaces in the roadmap are **unimplemented** (correctly noted in the proposal).
- **Content policy is a tiny pure function** (`evaluateContentPolicy`, ~50 LOC of regexes). Mirroring it in Go is trivial and low-maintenance — but calling it in-process is better.
- **`contentClass` silently coerces unknown values to `"note"`** — confirmed in `normalizeContentClass`. The spec's worry is real.
- The gateway roadmap explicitly says **curation should be an external process, not inline gateway behavior**, and lists an Admin UI as "Later / roadmap only."
- Qratum's whole v2 is **unimplemented spec** (P0). Milestone A is the only working code. The proposal *deletes* things from a spec, not from a codebase.

---

## ALTERNATIVE 1 — "No Qratum for this" (one-off script in personal-memory)

**Concrete design.** A single file in `personal-memory-gateway/scripts/import-claude-memories.ts` (TypeScript, runs in the repo that already has the MCP client, the OAuth token plumbing, and `evaluateContentPolicy` in-process):

1. `parse`: read `memories.json` + `projects/*.json` from a path arg. Split the 10,097-char account string on markdown structure into atomic candidates; each project string -> candidates scoped to `project:<slug>`. Emit `candidates.jsonl` + a human-readable `candidates.md` (one block per candidate: namespace, contentClass guess, content).
2. *(human step)* open `candidates.md` in your editor, delete/edit lines, fix classes.
3. `push`: read the edited file, call `evaluateContentPolicy` locally first (free, same code the gateway runs), then `memory_store` per surviving line with `metadata.origin = "claude-ai-export"`, `metadata.source_digest`. Append outcome to `receipt.jsonl`. Re-runnable because the gateway upserts on content hash.

**Effort:** ~200-250 LOC, one afternoon, zero new schemas, zero Go, zero fixtures-as-contracts. Reuses the gateway's own content-policy code in-process (no mirror to drift).

| Axis | Score |
|---|---|
| Time-to-value (17 strings) | **Best.** Hours. The strings are already curated by Claude.ai; they need atomization + scoping + a glance, which an editor pass gives you. |
| Moving parts | **Lowest.** One script, one editable file, one store. No bundle, no importer-handoff, no receipt re-ingest, no adapter, no Lesson schema. |
| Failure modes | Few and obvious. A bad split = you edit the file. A policy block = the gateway tells you, you rephrase. |
| Lock-in / reversibility | **Best.** It's a script. Throw it away after the one-time import. Nothing to maintain, nothing to migrate. |
| Fit with repo direction | Strong on the gateway side (roadmap literally says curation is an *external process*). The cost: the export never enters a durable archive, and **ongoing coding-session lessons get no home from this** — that lane stays a TODO in Qratum. |

**Verdict on A1:** For the *immediate* need it is the winner by a wide margin. Its only real gap is the ongoing lane: it does nothing for coding-session lessons. But notice — *neither does the proposal yet*. The proposal's ongoing lane (Tier-2 AI mining via facets) is P4, gated behind P1/P2/P3 that don't exist. So the proposal's claimed advantage on "ongoing lessons" is **a roadmap promise, not a shipped capability.** A1 ties the proposal on the thing the proposal is supposed to be better at.

---

## ALTERNATIVE 2 — "Direct push, no bundle" (`qrt export memories --push`)

**Concrete design.** Inside Qratum: keep Tier-0 split + a lesson file + CLI review, but the export command holds an OAuth token from env/keychain and calls the gateway MCP endpoint directly, recording `stored | duplicate | blocked | failed` synchronously into the lesson file. Delete: `memory_bundle` schema, `local_folder` publisher hop, the separate importer in personal-memory, `import_receipt.jsonl`, and the receipt re-ingest loop.

**What the bundle+importer+receipt actually buys, examined honestly:**

The proposal's stated reason for the pull/bundle/importer shape is *"Qratum holds no gateway credentials."* I will name this plainly: **for a single operator, on one Mac, where Qratum and the importer run as the same Unix user with read access to the same keychain and the same `~`, this boundary is theater.** The importer in personal-memory reads the token from the same keychain Qratum could read. There is no privilege separation, no second principal, no blast-radius reduction. An attacker who owns the Qratum process owns the importer process. The "credential boundary" is a property of *who typed the token into which repo*, not an enforced security control. It buys a *narrative* ("the local tool never touches prod creds") without buying a *mechanism*.

The receipt loop *does* buy one real thing: a durable record in Qratum of what was stored vs blocked, so Qratum can re-stage blocked items. But A2 gets that for free — synchronous outcomes written straight to the lesson file are strictly simpler than "write bundle -> importer writes receipt -> Qratum ingests receipt to learn the same outcomes it could have observed directly."

| Axis | Score |
|---|---|
| Time-to-value | Better than proposal (no second repo round-trip), worse than A1 (still needs Tier-0 + lesson schema + CLI built in Go first). |
| Moving parts | Roughly half the proposal's: no bundle, no publisher, no importer, no receipt re-ingest. |
| Failure modes | Fewer hops = fewer partial-failure states. Synchronous per-item outcome is the simplest possible error model. |
| Lock-in | Same as proposal otherwise. |
| Fit | Good. The gateway already supports `local-header` auth and per-client OAuth; a direct push is its native shape. |

**Verdict on A2:** A2 beats the proposal cleanly *on its own terms*. The bundle/importer/receipt triangle is the proposal's most over-engineered region, justified by a boundary that doesn't exist for this user. If you keep curation in Qratum at all, push directly.

---

## ALTERNATIVE 3 — "Curation lives with the store"

**Concrete design.** A `candidates` table in TiDB; gateway gains `candidate_submit` / `candidate_list` / `candidate_resolve` (approve -> `memory_store`, reject, edit). A tiny admin surface (CLI subcommand or one minimal page) to review. Qratum and any other producer just POST candidates. Curation, supersede, retire, dedup all happen where the memories live.

**Honest read:** Conceptually this is the *most correct* long-term shape, and the gateway roadmap is already walking toward it (namespace model, `memory_delete`, `memory_update`, contradiction/staleness detection, reversible curation log, "Admin UI — Later"). Curating in a tool that then *forgets what it exported* (the proposal's receipt loop exists precisely to paper over that amnesia) is genuinely awkward, and A3 dissolves that awkwardness: the store remembers because the store is the curator.

**But the cost is brutal for *now*.** This requires: a new TiDB table + migration, three new authenticated MCP tools or admin endpoints, an admin surface with its own auth, and it expands the attack surface of a *deployed, internet-reachable, OAuth-protected prod service* — in a repo whose CLAUDE.md says security is non-negotiable and whose own roadmap explicitly defers the Admin UI to "Later" and gates curation tooling behind an unbuilt namespace/grant model. You'd be pulling a "Later" item forward and bolting write-amplifying curation endpoints onto prod to move 17 strings.

| Axis | Score |
|---|---|
| Time-to-value | **Worst.** Days-to-weeks, plus prod deploy + threat-model review for new endpoints. |
| Moving parts | New table, new tools, new admin auth — all in the security-critical service. |
| Failure modes | Highest-stakes: bugs here are bugs in deployed prod, not a local tool. |
| Lock-in | High. Curation logic now lives in the cloud service; hard to walk back. |
| Fit | **Best long-term**, premature short-term. It's the right destination, wrong sprint. |

**Verdict on A3:** Right idea, wrong time. It resolves the conceptual smell but pays for it with prod risk and weeks of work to ship something a 200-line script ships in an afternoon. Revisit when there are hundreds of memories from many producers and curation is a recurring chore — not for the one-time 17.

---

## ALTERNATIVE 4 — "Git-native curation"

**Concrete design.** A `memory-candidates/` git repo. Producers (the Tier-0 splitter, agents, future exports) write one file per candidate (`<slug>.md` with YAML frontmatter: namespace, contentClass, source_digest; body = content). Review = edit/delete files in your editor or dispatch a Ductum agent to triage. Approval = merge to `main`. A post-merge local script (or CI) diffs merged candidates and pushes them via `memory_store`, then writes results back as a commit (`receipt/<hash>.json`). Versioned, diffable, PR-reviewable.

**Honest read for *this specific user*:** This is the sneakily-good one, and here's why it fits *him* and almost nobody else. He runs an agent factory (Ductum). "Curation" in git means he can **dispatch agents to curate** — split, classify, deduplicate, propose edits — as PRs he merges. Review becomes `git diff`. History is free. No new UI, no prod changes, no Go schema work. The "Qratum forgets what it exported" problem (which the receipt loop exists to solve) vanishes because **git is the durable memory of what was approved and pushed** — the receipt commit *is* the provenance.

The cost: a git repo per-candidate-file is slightly odd ergonomically (many tiny files), and you still need the push script (~= A1's `push` step). But the push script is the same ~80 LOC either way.

| Axis | Score |
|---|---|
| Time-to-value | Near A1. The splitter + push script is the same work; "review = edit files / merge" needs zero new code. |
| Moving parts | Low: a repo + one push script. The repo *is* the queue, the audit log, and the provenance store. |
| Failure modes | Low and inspectable; every state is a commit. |
| Lock-in | **Excellent.** It's files in git. Maximally reversible and portable. |
| Fit | **Uniquely strong for this user.** Agent-dispatchable curation, diffable approval, zero UI — this is the Ductum-shaped answer. |

**Verdict on A4:** The best *ongoing* answer for a solo founder with an agent factory. It generalizes past the one-time import (any producer drops files; agents curate; merge ships) without any of the proposal's schema/bundle/importer weight.

---

## Declared winner: a hybrid — **A1 now, A4 as the durable lane. The proposal does not survive intact.**

- **For the immediate 17 strings: Alternative 1.** A ~200-line script in `personal-memory`, reusing the gateway's own `evaluateContentPolicy` in-process, editor-as-review, direct `memory_store`, append-only receipt. Ships in an afternoon, idempotent for free (the gateway upserts on `(namespace, subject, content_hash)` — confirmed in `memory-repository.ts`). This captures essentially all of the proposal's *immediate* value (the strings are already curated; they need atomization + scoping + a glance) at ~3% of the effort.

- **For the ongoing coding-session lane: Alternative 4 (git-native), feeding the same direct-push script (Alternative 2's push, minus the bundle).** When Qratum's deterministic evidence/lesson extraction starts producing candidates from real coding sessions, have it *write candidate files to a git repo*, not stage them in a SQLite LessonBackend behind a local app that doesn't exist. Agents curate via PR; merge triggers direct push; the commit is the receipt. This gives the ongoing lane a home **without building P1-P5 of the v2 spec first.**

- **Keep from the proposal, because they're genuinely right:** (a) the content-policy *mirror* concept is sound, but **don't mirror — call the same code in-process** if the script lives in the gateway repo (A1), or vendor the regex list as a tiny shared fixture if it lives in Qratum (A4); (b) the `contentClass` enum-exactness rule (never let the gateway's silent `"note"` coercion happen) — keep that validation, it's a real, confirmed footgun; (c) **never send raw transcripts** — non-negotiable and all alternatives honor it.

- **Kill from the proposal:** the `claude-ai-export` SourceAdapter, `source_export_bundle`/`source_memory` raw kinds, the `memory_bundle` artifact, the `local_folder` publisher hop *for memories*, the separate importer-in-personal-memory, the `import_receipt.jsonl` re-ingest loop, the new `publish_memory_bundle` consent scope, and the `qratum.memory_export_item.v1` + `qratum.lesson.v1` *wire* schemas. They are P0 contract weight for an afternoon's worth of value.

---

## The single biggest thing the proposal gets WRONG

**It treats a one-time, 17-item data migration as a permanent product subsystem, and lets that one-time job drag a 0%-implemented v2 spec onto its critical path.** The proposal's milestone mapping puts the *immediate* value (Tier-0 split of `memories.json`) at P2, behind P0 schemas (Lesson + memory_export_item + memory_bundle manifest + synthetic export fixtures) and ahead of a P5 importer — i.e., the 17 curated strings can't move until you've designed three schemas, a bundle format, a publisher, a cross-repo importer contract, and synthetic fixtures. **The export is already curated. The destination is one function call.** Building a content-addressed adapter + Lesson primitive + bundle/importer/receipt pipeline to deliver pre-curated strings to a single `memory_store` endpoint is using a shipping-container architecture to mail a postcard. The proposal optimizes for "a clean general pipeline for all future memory sources" when the user has *one* source, *once*, and *17 items*.

## The unnamed "odd shape"

**The data flows in a circle to launder a boundary that isn't real.** Watch the path: Qratum extracts -> stages a Lesson -> exports a `memory_bundle` to a folder -> a *separate* importer in *another repo* reads the folder -> calls `memory_store` -> writes a `receipt` -> and then **Qratum re-ingests the receipt to learn the outcome of an action it initiated.** That loop exists almost entirely to preserve the claim *"Qratum never holds gateway credentials."* But both processes run as the same user on the same laptop against the same keychain — so the boundary is **organizational, not enforced**. The architecture is contorted into a pull-shaped, bundle-mediated, round-trip handoff to honor a credential boundary that has no mechanism behind it. That's the odd shape: **ceremony standing in for a control.** The system performs the *motions* of a trust boundary (separate repo, separate importer, no shared creds, receipt handshake) between two halves of one operator on one machine, and pays for that performance in schemas, hops, and a state-reconciliation loop. Strip the ceremony and the honest shape is: *split locally -> glance -> push -> record the result inline.* Everything past that is a costume.

A secondary, quieter oddity worth naming: **the proposal duplicates the gateway's content policy into Go as a "mirror" and then immediately admits the mirror is best-effort and the gateway is the real authority** — i.e., it creates a guaranteed drift surface (two copies of the same security regexes in two languages, the user's own CLAUDE.md flags exactly this rename/drift risk) to move a check a few seconds earlier in a flow that, in A1, runs *the actual gateway code* with zero duplication.

# Fable Review: Qratum Project Intelligence

Status: product-owner disposition recorded; not implementation authority

Date: 2026-07-12

Source: prior Fable review session supplied by the product owner

Reviewed proposal:

```txt
docs/reviews/2026-07-12-project-intelligence-user-stories.md
```

## Product-Owner Disposition

F1 is downgraded from critical to a footnote. The ledger's source data is
validated in practice: per-message usage fields in Claude Code JSONL and
delta-from-cumulative accounting for Codex rollouts are sufficient to reconcile
usage. `ccusage` demonstrates the Claude side daily on the same local files
Qratum will read.

The retained residual is deliberately cheap and already required by Qratum's
fixture discipline:

- pin the supported Claude Code and Codex field semantics with Tranche 1
  fixtures;
- treat supported Claude Code and Codex records as `exact/source-reported` by
  default;
- retain a usage-record reliability field for future sources and format drift;
- fail closed and reduce visible coverage when an unknown source shape appears;
- reconcile the Claude implementation against known-working `ccusage`-style
  totals during development.

The pricing-source decision is also resolved:

- use a LiteLLM-style model-price catalog;
- vendor a pinned snapshot, including its upstream version or commit identity;
- refresh the bundled snapshot through Qratum binary releases;
- optionally accept a separate, explicit user-initiated catalog import;
- never fetch or update the catalog silently at runtime;
- treat missing model pricing as `unknown`, not zero.

F2 through F9 stand unchanged. Their severities and recommendations remain the
reshaping input for this proposal.

## Verdict

**Accept with specific reshaping.**

The identity model and cost-truth machinery strengthen the accepted product
thesis. Paths are observations, Git common-directory identity correctly joins
worktrees, Unassigned is a valid state, cost totals expose coverage, pricing is
versioned, and cumulative-versus-incremental usage is treated explicitly.

The proposal nevertheless expands the product thesis and release scope before
the accepted core has shipped. It also risks recreating standing triage queues
through project suggestions, workstream suggestions, proposed decisions, open
loops, and roadmap candidates.

The recommended shape is:

1. Accept capture-time identity, usage semantics, repository grouping, Projects,
   and the deterministic ledger with the corrections in this review.
2. Keep the accepted daily session spine ahead of the new Project product.
3. Prove Projects and costs in daily use before adding Workstreams.
4. Gate decisions, open loops, and roadmap synthesis behind demonstrated use of
   the layer below them.

## Product-Critical Findings

### F1 — Fixture-lock usage semantics

Severity: footnote.

Claude Code per-message usage and Codex delta-from-cumulative usage are accepted
as exact/source-reported inputs for the initial ledger. The tranche contract
only needs to lock those field semantics with source-version fixtures so future
format drift becomes a visible coverage failure rather than silent bad totals.

The reliability field remains useful for future sources and observed drift. It
is not a day-one warning label on supported Claude Code or Codex records.

### F2 — Do not create standing suggestion queues

Severity: critical.

The proposed design can create five persistent triage lanes:

- suggested project membership;
- suggested workstreams;
- proposed decisions;
- suggested open loops;
- suggested roadmap items.

Qratum already has evidence that a standing review queue does not earn user
attention. The intelligence layer must not dominate the application with unread
suggestions.

Required correction:

- generate suggestions only on explicit request or show them inline where they
  are relevant;
- do not display unread or pending-curation counters;
- show accepted or confirmed items on overview surfaces;
- bound extraction to a selected project, time range, or session set;
- treat Unassigned as a healthy permanent state, not debt.

### F3 — Separate capture-time facts from re-derivable analysis

Severity: high.

The proposal overloads Tranches 1 and 2 by treating all identity and ledger work
as capture-time foundation. Only facts that disappear with an ephemeral working
copy must be captured immediately:

- Git common directory or repository identity;
- observed working-copy path;
- machine identity;
- sanitized remote identity;
- branch and commit at observation time;
- source-provided surface, agent, model, and harness evidence;
- assignment provenance when the source supplies it.

Usage parsing, pricing, project assignment, aggregation, and most attribution
interpretation are re-derivable from preserved exact history. They should not
delay the accepted reader, library, search, deletion, and export spine.

### F4 — Workstream overlaps with Continuation

Severity: high.

The accepted product already groups confirmed continuations into one body of
work. The proposal adds Workstream as another grouping of sessions without
defining how the two concepts relate.

Recommended rule:

- a confirmed continuation thread is a candidate workstream;
- naming or promoting that thread creates a workstream;
- one workstream may join multiple continuation threads;
- every project surface remains useful with zero workstreams.

This gives Workstreams a cold start without forcing a second assignment system.

### F5 — Forge references are not live issue or PR state

Severity: high.

Locally observed issue, PR, and commit identifiers can be stored as evidence.
Current GitHub, GitLab, or Jira state requires a network integration with its
own consent, privacy, authentication, freshness, and failure contract.

Required correction:

- treat forge references as inert identifiers and links in this extension;
- do not claim current issue or PR state;
- defer any live forge read or synchronization to a separate opt-in contract.

### F6 — Terminal deletion and an auditable ledger conflict

Severity: high.

Deleting a session can either remove its usage records, causing historical
project totals to change, or retain anonymous usage, weakening the promise that
all session-derived data is removed.

Recommended default:

- privacy wins: usage records are deleted with the session;
- aggregates are recomputed;
- affected totals disclose that they exclude erased sessions, using tombstone
  counts without retaining the erased content or usage;
- deletion preview lists affected project totals, summaries, decisions, open
  loops, workstreams, and roadmap evidence;
- user-owned statements may survive only with their evidence clearly marked as
  erased and with that behavior approved in the product contract.

### F7 — Subscription users need API-equivalent framing

Severity: medium.

Claude Code and Codex are often used through flat-rate subscriptions. A
calculated API price is not the amount billed to the user.

Required correction:

- call calculated dollars `API-equivalent usage value` for subscription use;
- record billing mode per source or account: API, subscription, or unknown;
- make tokens and coverage primary evidence;
- show source-reported billed cost only when a source actually provides it;
- never label an estimate as spend.

### F8 — Git identity fields remain sensitive

Severity: medium.

Branch names, remotes, and commit metadata are needed for owner-only project
identity, but can contain customer names, internal project names, credentials,
or other sensitive strings. Earlier trust work removed these fields from
shareable artifacts for that reason.

Required correction:

- keep exact Git identity in an owner-only metadata store;
- sanitize remotes before display or comparison;
- apply credential masking to branch names and other displayed Git fields;
- exclude exact Git metadata from share-oriented output unless the egress plan
  explicitly lists and handles it.

### F9 — The roadmap state model is too large initially

Severity: medium.

`Suggested`, `Accepted`, `In progress`, `Blocked`, `Done`, `Rejected`, and
`Superseded`, combined with Now/Next/Later and dependencies, becomes a manually
maintained task tracker.

Recommended first shape:

- `Suggested`, `Accepted`, `Done`, and `Rejected` only;
- Qratum may suggest a change when evidence appears but never changes state;
- add `In progress`, `Blocked`, or richer planning states only after actual use
  proves they will be maintained.

## Project And Repository Identity Review

The core hierarchy remains sound:

```txt
Project
  -> Repository
    -> Working-copy observations
      -> Sessions
```

Required additions:

- Repository grouping works automatically and does not require creating a
  Project.
- Project is an optional, user-named grouping above one or more repositories.
- The library remains useful when every session is Unassigned.
- Cold start supports batch assignment by repository or remote pattern. It must
  not require one confirmation per session.
- Assignment provenance distinguishes current working directory, observed Git
  identity, and multi-repository file activity when available.
- Project rename, archive, merge, split, and delete semantics are explicit.
- Deleting a Project does not delete sessions; sessions return to repository or
  Unassigned views unless the user separately erases them.
- A rejected grouping suggestion is remembered and not repeated without new
  evidence, which must be shown.
- Machine identity is provenance only while cross-machine merge remains
  deferred.
- Non-Git folders and ambiguous clones remain valid, honestly labeled cases.

One product decision remains necessary: whether a repository belongs to only
one Project, or can be shared by several Projects with one primary assignment
per session for cost accounting.

## Usage And Cost Review

The ledger should retain separate dimensions for:

- input tokens;
- output tokens;
- cache-read tokens;
- cache-write tokens, including TTL class when provided;
- reasoning tokens;
- source-reported cost, if any;
- calculated API-equivalent value;
- model and provider;
- source version and usage semantics;
- billing mode;
- reliability class;
- coverage state.

Additional rules:

- A single merged token headline is misleading. Show token classes and explain
  that repeated context is billed repeatedly.
- Historical prices come from a pinned LiteLLM-style catalog snapshot and are
  keyed by provider, model, token class, and effective date.
- Unknown or custom models can use an explicit user-supplied price without
  rewriting historical source evidence.
- The price catalog snapshot rides Qratum binary releases. An explicit
  user-initiated import may refresh it out of band; silent runtime fetching is
  forbidden.
- Claude re-ingestion deduplicates by stable message or request evidence rather
  than transcript digest alone.
- Codex cumulative counters are converted to increments and reconciled against
  the final cumulative total. A reset or mismatch becomes a coverage problem,
  not a guessed number.
- Resumed-session behavior is fixture-tested for every source.
- Imports without usage evidence remain `unknown`; Qratum does not invent
  tokens or cost.
- Qratum's own optional AI-analysis usage is a separate ledger and is never
  mixed into the project's source-session usage.
- Repricing events are journaled so a changed estimate can be explained.

The proposal's drill-down and coverage-denominator rules should remain: every
aggregate must lead to the usage records and exact sessions behind it.

## Workstream And Code-Link Review

Workstreams are not yet proven to justify their curation cost. The first useful
shape is a saved, user-named bundle of continuation threads, sessions, and code
evidence. It is not a taxonomy the user must maintain.

Recommended constraints:

- ship Projects, costs, and search before Workstreams;
- bootstrap manual Workstreams from confirmed continuation threads;
- allow one-action assignment from a session or search result;
- keep all views functional with zero Workstreams;
- measure use before building clustering or LLM naming;
- give generated Workstream suggestions a kill path if acceptance stays low.

Code-link evidence must distinguish:

1. transcript-proven creation, such as a `git commit` action and its resulting
   SHA;
2. transcript mention, such as reading a SHA from history;
3. user-confirmed association;
4. weaker path, branch, time, or semantic correlation.

Only transcript-proven creation may answer causal questions such as which
session produced a commit. Correlation links are capped, shown on demand, and
never phrased as causation.

## LLM And Information-Quality Review

The deterministic product is independently useful: repository identity,
Projects, token accounting, API-equivalent estimates, coverage, timelines,
manual organization, lexical search, and evidence drill-down do not require a
generative model.

The five information classes remain the correct provenance model, with a
simpler display rule:

- exact content is the unbadged default;
- deterministic derivation, semantic relation, LLM-generated content, and
  user-confirmed statements carry visible labels;
- every non-exact claim links back to evidence and transform provenance.

Generated artifacts require:

- input digests and transform or prompt versions;
- visible freshness and staleness;
- an index of every source session consumed, reused for deletion propagation;
- visible refresh cost before an external or metered model is invoked;
- deduplication so one decision discussed in several sessions becomes one
  candidate with several evidence links;
- owner-only handling because project names, summaries, decisions, open loops,
  and context packs are raw-derived artifacts;
- an egress preview that includes repository names, machine identifiers, and
  project metadata as well as transcript content.

Decision and open-loop extraction should be on request. Open-loop candidates
also need evidence age, bulk dismissal, and an expiry or archive rule so stale
suggestions do not become a generated task queue.

## Direction And Roadmap Review

The user must remain the only authority that accepts direction or changes
roadmap state.

Minimum safe first interaction:

- accepted decisions and confirmed open loops form the evidence base;
- Qratum proposes a candidate only when the user asks;
- candidate sources are limited initially to user-authored turns, accepted
  decisions, confirmed loops, and explicit user requests;
- assistant boilerplate and repeated assistant themes do not create roadmap
  candidates;
- every candidate shows evidence, age, the newest relevant evidence, and a
  contradiction or supersession check;
- stale or superseded evidence suppresses rather than reinforces a candidate;
- suggestions live outside the user's Now/Next/Later columns;
- accepting requires reviewing the evidence;
- Qratum never moves an item, including to Done;
- generated estimates remain excluded;
- forge references remain inert until a separate integration is accepted.

If decisions and open loops do not earn real use, roadmap synthesis has no
trusted input and should be shelved rather than fed weaker evidence.

## Missing Or Broken User Journeys

### 1. Bulk cold-start assignment

Hundreds of sessions are grouped by repository evidence and can be assigned in
one confirmation per group. Unassigned remains a valid permanent outcome.

### 2. Context-pack round trip

A context pack carries provenance. When a later source session exposes that
provenance, Qratum can confirm its Project or Workstream relationship instead
of leaving the new session unrelated.

### 3. Project lifecycle

Rename, archive, merge, split, and delete all preserve audit history. Project
deletion removes project-generated artifacts but does not imply session
erasure.

### 4. Erasure inside a Project

Deletion preview lists affected aggregates and generated artifacts. After
deletion, totals and generated views update or become stale according to the
accepted deletion policy.

### 5. Re-suggestion after rejection

A rejected grouping or relationship is not proposed again unless new evidence
appears; the changed evidence is shown.

### 6. Usage-format drift

An unrecognized source shape reduces visible coverage and reports a diagnostic.
It never becomes silent zeros or fabricated cost.

### 7. Organization export

Project names, assignments, corrections, user-confirmed relationships,
decisions, and roadmap state are included in a recoverable owner export because
they cannot be reconstructed entirely from source transcripts.

### 8. Gather lag

Project overviews and cost ranges disclose whether an in-progress source
session has not yet been captured. Today's total must not imply live coverage
when gathering occurs only at session end.

## Scope And Tranche Review

### Tranche 1 — Source correctness

Keep the accepted Tranche 1 work and add only facts that can disappear before a
later scan:

- capture-time repository and working-copy observation;
- machine, branch, commit, and sanitized remote observations;
- source, surface, agent, model, and proven harness evidence;
- usage-semantic fixtures and source-version reliability tests.

Usage parsing and Project aggregation remain re-derivable and need not block
capture unless a source exposes usage only ephemerally.

### Tranche 2 — Accepted daily spine

Keep the accepted daily spine ahead of the new Project product. Automatic
repository grouping and a repository filter may join it because they improve
the library without requiring curation.

### Tranche 2.5 — Deterministic Project product

- optional multi-repository Projects;
- Unassigned and repository views;
- batch assignment and correction;
- project overview;
- project-scoped lexical search;
- usage, coverage, API-equivalent value, and evidence drill-down;
- bookmarks and notes;
- project-organization export.

### Immediate follow-up

- timeline;
- transcript-proven code links;
- deterministic cost anomalies;
- manual Workstreams bootstrapped from continuation threads;
- deterministic context-pack selection and round-trip provenance.

### Later accepted tranches

- semantic search and related-session evidence remain in the accepted semantic
  tranche;
- imports gain project assignment when imports ship;
- optional AI summaries remain behind the accepted local/external AI boundary.

### Acceptance-gated intelligence tranche

Only after Projects and manual organization demonstrate actual use:

1. Project and Workstream summaries;
2. decisions and open loops, generated only on request;
3. user-owned direction and roadmap proposals;
4. context-pack compression;
5. Workstream clustering, only if manual Workstreams prove valuable.

Each layer must earn use before the next one becomes implementation scope.

## Blocking Product Decisions

The pricing source and initial usage reliability defaults are resolved above.
The remaining owner decisions are:

1. Which Git signals may assign automatically, and which require confirmation?
2. Can one repository belong to multiple Projects, or is assignment one-to-one
   with per-session exceptions?
3. When a session is erased, do its usage records disappear and totals disclose
   excluded erasures? The review recommends yes.
4. Do user-owned decisions or roadmap items survive erasure of their only
   evidence with a tombstoned evidence marker?
5. What evidence proves a harness such as Ductum, and should harness attribution
   wait until that marker exists?
6. Are forge references inert in this extension? The review recommends yes.
7. Are decision, loop, and roadmap extractions strictly on request and bounded
   in scope?
8. May roadmap candidates use assistant-authored content, or only user-touched
    and user-confirmed evidence? The review recommends the latter initially.
9. What is the exact relationship between a continuation thread and a
    Workstream?
10. Is Project organization part of owner export before backup exists? The
    review recommends yes.

## Final Consistency Check

- Qratum remains coherent if Projects and costs are organizing and accounting
  views over the accepted searchable-memory thesis. It becomes less coherent if
  roadmap and task-management behavior become the headline before the core
  ships.
- Project is the right organizing layer if repository grouping is automatic and
  Project creation remains optional.
- Workstream value is unproven. Bootstrap it from Continuation and measure it.
- Project cost is viable, but trust depends on verified source semantics,
  visible coverage, correct billing framing, and deletion behavior.
- Evidence drill-down and digest-keyed provenance are the proposal's strongest
  mechanisms.
- Roadmap ownership is credible only when candidate inputs exclude assistant
  boilerplate and the user alone accepts or changes state.
- The deterministic app remains valuable without a generative LLM.
- Deletion is incomplete until aggregate, generated-artifact, and user-owned
  statement behavior is accepted explicitly.
- The release becomes credible when the accepted daily spine ships before the
  Project product and the intelligence tier is acceptance-gated.

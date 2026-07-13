# Qratum Product Direction

Status: accepted product contract

Accepted by product owner: 2026-07-11

Project Intelligence extension accepted by product owner: 2026-07-12

Technical implementation status: not accepted; tranche contracts pending

## Authority

This file is the canonical product authority for Qratum. It governs product
intent, user journeys, user-facing language, public surfaces, release order,
and the session-reference information the product must preserve and show.

The complete reviewed user stories are incorporated by reference:

```txt
docs/reviews/2026-07-10-product-user-stories.md
docs/reviews/2026-07-12-project-intelligence-user-stories.md
docs/reviews/2026-07-12-project-intelligence-owner-decisions.md
```

Those reviewed documents are normative for user needs and product behavior.
This file owns their place in the repository authority chain, tranche order,
and the minimum session-reference contract.

Where these files conflict with the following older documents on product
shape, this contract wins:

```txt
specs/current/ui-first-onboarding.md
specs/current/operational-model-redesign.md
specs/current/qratum-vault-first.md
specs/current/verification-and-trust-gate.md
docs/reviews/2026-07-10-runtime-rebootstrap.md
```

Those files remain evidence about shipped v0.1.0, the local runtime candidate,
preservation architecture, security findings, and previously accepted design.
They do not authorize implementation of the new product direction.

## Product Thesis

Qratum is not backup software.

> Qratum is the private, searchable memory of my AI work. It continuously
> gathers my Claude and Codex sessions and their associated context, keeps the
> exact history available locally, and helps me find and understand past work
> through a clean UI. When useful, it also gives me a direct path to continue
> that work in its source tool.

## Accepted Product Shape

The accepted shape is:

- the local UI is the primary product;
- exact local history is readable by its owner;
- lexical and semantic search are core product capabilities;
- search results identify passages or turns and group them by session;
- lexical and semantic indexes are raw-derived, owner-only, local-only by
  default, rebuildable, and delete-coupled;
- Claude Code and Codex are separate source adapters with separately verified
  capture, identity, parsing, and failure behavior;
- a source hook is preferred when its lifecycle and identity semantics are
  sufficient, but the mechanism is not assumed before verification;
- source-confirmed and user-confirmed continuations are distinct from weaker
  related-work suggestions;
- understanding what was said and what happened is a complete user journey;
- continuing in the source tool is an optional, source-native action;
- observed source context, derived insights, and durable memories are distinct;
- durable approved memories belong in the user's personal-memory project and
  require an explicit, consented handoff;
- a session view includes exact history, date, source, model when known,
  summary/outcome, and honestly calculated cost;
- actual billed cost and an API-equivalent usage value are different values;
- repository identity joins moved clones and Git worktrees without treating
  paths as identity;
- repository grouping works without requiring a Project;
- a Project is an optional user-owned grouping over repositories and sessions;
- Project costs aggregate exact/source-reported usage records and drill down to
  their sessions;
- `Unassigned` is a healthy state, not curation debt;
- suggestions are contextual or explicitly requested and never become standing
  inboxes or unread counters;
- Workstreams are optional saved evidence bundles that begin from confirmed
  continuation threads;
- forge references are inert identifiers unless a separate opt-in integration
  is accepted;
- deletion covers local raw and derived representations and cannot silently
  resurrect through later capture or import; and
- import, export, delete, external processing, and health status must execute
  real behavior and report it truthfully.

## Superseded Product Rules

The following older rules are no longer authoritative for new work:

- the global `found -> preserved -> prepared -> open` product state machine;
- `Viewable` as a separate product state;
- `prepare` or a raw queue as standing user workflow;
- the prohibition on the owner reading exact raw history locally;
- raw indexes being disabled as the product default;
- the eight-command UI-first public CLI as accepted product surface;
- the local API shell being described as a shipped UI product;
- Claude-only capture/refinery as sufficient for the first user's product;
- review cards as the primary session payoff;
- preservation, backup, lessons, insights, review, or corpus ordering ahead of
  search and exact reading; and
- the 2026-07-10 rebootstrap slices as the active implementation roadmap.

## Standing Constraints Retained

This cutover does not weaken the following accepted constraints:

- one Go binary named `qrt`; no Python runtime;
- raw transcripts are sensitive and owner-only;
- raw transcript content is not silently sent to external services;
- raw transcript content is not rendered into shareable reports;
- external processing is explicit and reports provider, locality, input class,
  and cost;
- source adapters and other untrusted-input boundaries fail closed;
- source hooks stay small, local, fast, and network-free;
- the UI consumes typed DTOs rather than parsing source transcripts or storage
  internals;
- stored and exported objects carry provenance;
- fixtures and golden outputs lock intentional contracts;
- the supply-chain rules in `docs/supply-chain.md` apply; and
- no shipped or trusted claim may be inferred from a green unit test alone.

## Accepted Project And Cost Layer

The Project layer organizes the accepted searchable session memory; it does not
replace it or turn Qratum into a task tracker.

### Identity and assignment

- Paths and machine locations are observations, not repository or Project
  identity.
- Working copies sharing one Git common directory are automatically one
  repository.
- A previously confirmed repository remains the same after it moves or appears
  through another worktree.
- Matching sanitized remotes across independent clones create a suggestion,
  not an automatic Project assignment.
- Forks, ambiguous remotes, monorepo subprojects, repositories without remotes,
  and non-Git folders require confirmation.
- Once a repository is assigned to a Project, later sessions with that
  confirmed identity inherit the assignment.
- One repository may appear in several Projects for discovery, but each session
  has exactly one primary Project for cost; secondary relationships never copy
  usage.
- Batch confirmation by repository is the cold-start path. Per-session triage
  is the exception.
- Exact Git identity is owner-only. Remotes are sanitized, displayed Git fields
  receive credential masking, and share-oriented output excludes exact Git
  metadata unless its egress plan lists it.

### Usage and cost

- Supported Claude Code per-message usage and Codex delta-from-cumulative usage
  default to `exact/source-reported`.
- Tranche 1 fixtures pin supported source-version field semantics; unknown
  shapes reduce coverage visibly and fail closed.
- A pinned LiteLLM-style price-catalog snapshot rides Qratum releases. An
  explicit user import may update it; silent runtime fetching is forbidden.
- Source-reported billed cost, API-equivalent usage value, and unknown cost are
  separate states.
- Subscription calculations are API-equivalent value, never spend.
- Every total shows coverage and drills down to usage records and sessions.
- Qratum's own optional analysis cost is separate from source-session usage.

### Curation, deletion, and external ownership

- Suggestions appear only on explicit request or in immediate context. They do
  not accumulate in queues or home-screen debt counters.
- Rejected suggestions persist and are reconsidered only when new evidence is
  shown.
- Workstreams remain optional and begin by promoting confirmed continuation
  threads; all Project surfaces work with zero Workstreams.
- Only an explicit harness marker containing harness name, version, run
  identity, and observation time proves Ductum or another harness. Otherwise
  harness attribution is `unknown`.
- Session deletion removes its usage, recomputes aggregates, and discloses that
  totals exclude erased sessions without retaining anonymous usage.
- A user-confirmed decision or roadmap statement may survive deletion with an
  `evidence erased` marker. Generated drafts and other session-derived artifacts
  do not survive.
- Project organization and user-confirmed relationships are included in owner
  export.
- Personal-memory receives only an explicit user-approved editable statement,
  records the returned identifier, and retains its own correction/deletion
  authority.

### Optional intelligence

- Deterministic Project views require no generative model.
- A proven-local model may read exact history only after its technical contract
  verifies that boundary.
- External providers receive no exact history by default. Every request shows
  provider, model, locality, input class, Project/repository metadata,
  destination, and estimated cost before confirmation.
- Generation is initiated for a bounded user-selected scope. Changed inputs
  make cached output visibly stale; there is no automatic corpus-wide
  extraction or background candidate generation.
- The first roadmap states are `Suggested`, `Accepted`, `Done`, and `Rejected`.
  Suggestions remain outside user-owned `Now`, `Next`, and `Later` columns, and
  Qratum never changes an item's state.

## Release Tranches

Tranches are dependency order. A later tranche does not begin until the prior
tranche has an accepted technical contract and passes its executable proof.

### Tranche 0: Product truth

Status: complete 2026-07-11

- establish this authority chain;
- correct shipped-versus-candidate wording;
- mark conflicting older product rules as superseded; and
- define the minimum reference information every session must expose.

### Tranche 1: Source correctness

- separate Claude Code and Codex adapter contracts and fixtures;
- verify the capture mechanism for each source;
- gather exact local history with correct source identity;
- update resumed sessions without duplicates;
- capture repository/Git-common-directory, working-copy, machine, sanitized
  remote, branch, commit, and proven source/harness observations that may
  disappear before a later scan;
- fixture-lock Claude Code per-message usage and Codex
  delta-from-cumulative usage semantics; and
- prove provenance, containment, owner-only storage, and installed behavior.

### Tranche 2: Daily product spine

- UI-first onboarding and normal operation;
- unified library and exact transcript reader;
- passage-level lexical search;
- automatic repository grouping across worktrees and clones;
- repository filters in the library and lexical search;
- honest date, source, summary/outcome, and cost presentation;
- manual rename and pin/favorite;
- truthful source-native continue action;
- confirmed continuation handling;
- terminal local deletion;
- real exact export; and
- truthful capture, storage, and lexical-index health.

### Tranche 2.5: Deterministic Project product

- optional multi-repository Projects plus repository and `Unassigned` views;
- batch assignment, correction, and audit history;
- Project Overview and project-scoped library/search;
- token usage, coverage, API-equivalent value, and evidence drill-down;
- bookmarks and notes; and
- owner export of Project organization.

After Tranche 2.5 proves useful, deterministic timeline, transcript-proven code
links, cost anomalies, manual Workstreams from continuation threads, and
deterministic context-pack selection may receive a separately accepted
follow-up contract.

### Tranche 3: Semantic retrieval

- local-default embeddings;
- hybrid lexical and semantic ranking;
- partial-coverage disclosure;
- resumable model-version re-indexing; and
- installed privacy, deletion, and retrieval verification.

Project-scoped semantic search and related-session evidence are included.
Workstream clustering is not.

### Tranche 4: Context and imports

- allowlisted observed source context for Claude and Codex;
- explicit personal-memory handoff contract;
- strict Claude.ai and ChatGPT export adapters; and
- truthful context timing and provenance.

Project-scoped source context, personal-memory handoff, and imported-session
Project assignment use the same accepted boundaries.

### Tranche 5: Optional enrichment and sharing

- clearly labeled session-level narrative AI summaries, outcomes, and memory
  candidates;
- editable or rejectable generated fields;
- weaker related-work suggestions;
- real redacted/share-oriented export; and
- complete provider, locality, input-class, cost, and provenance disclosure.

### Tranche 6: Acceptance-gated Project intelligence

This tranche exists only after Projects and manual organization demonstrate
real use. Its internal order is:

1. Project and Workstream summaries;
2. decisions and open loops generated only on explicit bounded request;
3. user-owned roadmap proposals;
4. context-pack compression; and
5. Workstream clustering only if manual Workstreams prove valuable.

Each layer must earn use before the next receives a technical contract. Low
acceptance or non-use is a kill signal, not permission to generate more.

## Contract Readiness

Only the product direction and Tranche 0 are accepted. Tranches 1 through 6,
including Tranche 2.5, are not implementation-ready.

Before code begins for a tranche, its technical contract must:

1. resolve every product decision that changes externally visible behavior;
2. define source schemas, stored schemas, and DTOs used by the tranche;
3. define failure and recovery behavior;
4. define privacy and deletion boundaries;
5. name executable acceptance tests and fixtures;
6. name the installed-artifact user flow used to check the behavior; and
7. receive explicit product-owner acceptance.

Existing code is donor material and evidence. Its presence does not settle an
open contract decision.

## Minimum Session Reference

The sessions already exist. Qratum's first job is to preserve enough truthful
reference information that the user can find one, understand it, and open its
exact history without guessing.

Every session must expose, when the source provides it:

### Where

- source (`claude-code`, `codex`, `claude-ai`, or `chatgpt`);
- source session identity;
- repository, workspace, and working directory;
- confirmed repository identity and primary Project assignment when available;
- branch and commit when observed;
- local machine scope; and
- the capture or import record that brought it into Qratum.

### When

- source-reported start and end times;
- last activity time;
- Qratum observation and capture times; and
- whether each time is source-reported, observed, or inferred.

Unknown times remain `unknown`; Qratum does not invent precision.

### How much it cost

- model identity;
- recorded input, output, and cached token usage;
- source-reported billed cost when available;
- otherwise an explicitly labeled `API-equivalent usage value` calculated from
  a pinned LiteLLM-style price-catalog snapshot;
- price source, effective date, currency, and calculation basis; and
- `unknown` when the usage, model, or applicable price is missing.

The catalog snapshot records its upstream version or commit and rides Qratum
binary releases. A separate explicit user-initiated import may refresh it.
Qratum never fetches or updates pricing silently at runtime.

An API-equivalent value is never presented as the user's actual subscription
charge or spend. Supported Claude Code per-message usage and supported Codex
delta-from-cumulative usage are treated as exact/source-reported. Tranche 1
fixtures pin those semantics so future source-format drift reduces visible
coverage instead of silently changing totals.

### What happened

- exact ordered history;
- a digest tying the displayed history to the captured bytes;
- concise summary and outcome when available;
- whether the summary/outcome is source-reported, deterministic, or
  AI-generated; and
- files, commands, verification results, and unresolved work when derivable.

### Relationships and context

- source-confirmed or user-confirmed continuations;
- weaker related-work suggestions kept separate;
- observed source-context versions and observation times; and
- durable memories sent to personal-memory, including the returned identifier.

The Tranche 1, Tranche 2, and Tranche 2.5 technical contracts will define the
concrete stored shape and UI DTOs for this reference. They should not invent a
larger metadata system unless a user story requires it.

## Release Truth

The purpose of release checking is narrow: do not describe a behavior as
available until the installed `qrt` artifact performs that user flow with its
accepted fixtures.

For each tranche:

- acceptance checks cover only the behavior the tranche builds;
- checks use an isolated `QRATUM_HOME`, never the user's real sessions;
- missing or unsupported reference fields display `unknown` or `unsupported`;
- documentation names exactly what the installed artifact can do;
- candidate code is not described as published; and
- a failed security, deletion, containment, or data-integrity check blocks the
  affected release claim.

Tranche 0 is complete when `SPEC.md` and `AGENTS.md` select this contract, the
conflicting older specs and dispatch plans carry supersession notices, and the
published v0.1.0 baseline is clearly separated from the local candidate.

## Blocking Decisions Before Tranche Contracts

The accepted product review leaves these decisions open for the relevant
technical contract:

1. local semantic model acquisition, versioning, updating, and execution;
2. AI enrichment defaults and whether proven-local AI may read exact history;
3. the exact Claude and Codex context-source allowlists;
4. the source identifiers that prove a resume;
5. per-source capture cadence and whether hooks alone meet continuous-gathering
   expectations;
6. the explicit personal-memory handoff/correction/deletion mechanism.

The session-pricing authority is resolved: use a pinned LiteLLM-style catalog
snapshot that rides Qratum releases, with an optional explicit user-initiated
import and no silent runtime fetch. Only an explicit source-reported charge is
actual billed cost; calculated values are API-equivalent usage values.

These are not permission to design by implementation. Resolve them in the
tranche contract before code.

# Qratum Product User Stories

Status: accepted product direction; technical contract pending

Date: 2026-07-10

Last updated: 2026-07-11

Accepted by product owner: 2026-07-11

## Purpose

This document defines Qratum from the user's actual needs before choosing
commands, schemas, storage structures, models, or implementation phases. It is
not an accepted implementation contract yet.

The repository audit and Fable review are evidence about the old shape. They do
not decide the new product. The product decisions in this document come from
the user's direct review feedback.

## Product Thesis

Qratum is not backup software.

> Qratum is the private, searchable memory of my AI work. It continuously
> gathers my Claude and Codex sessions and their associated context, keeps the
> exact history available locally, and helps me find and understand past work
> through a clean UI. When useful, it also gives me a direct path to continue
> that work in its source tool.

Preservation is essential infrastructure for that promise. It is not the whole
product and should not dominate the user experience.

## First User

The first user is a developer who:

- regularly works with Claude Code and Codex;
- already has hundreds of sessions;
- has decisions, debugging trails, implementation context, instructions, and
  memories distributed across those systems;
- restarts or resumes work across multiple sessions;
- needs to find previous work by meaning, error, file, decision, or topic;
- wants to read the exact local history, not only metadata about it;
- does not want raw work history silently sent to external services; and
- wants the product to work day to day without repeating onboarding.

Claude.ai and ChatGPT web history are also important, but enter through explicit
vendor-export import because they are not continuously visible to a local
capture process.

## Product Principles

### UI first

The main product is the local UI. The CLI bootstraps, diagnoses, and supports
automation; it is not where the normal library workflow lives.

### Always available after onboarding

Initialization happens once. After that, Qratum continuously gathers supported
history, updates resumed sessions, and keeps the library searchable.

### Exact history first

The owner can read the exact locally stored transcript inside Qratum. A cleaned
presentation, summary, review, or memory view must never replace or distort the
source history.

Detected credentials can be masked by default in the UI with an explicit local
reveal action. This is a display safeguard, not deletion or a claim that the
content is privacy-safe.

### Search is core

Search is not a post-product enhancement. The user already has hundreds of
sessions. Qratum must support both exact retrieval and semantic retrieval in the
first genuinely usable product.

### Continuity matters

A resumed or restarted conversation must remain connected to the work it
continues. Qratum should represent source sessions, confirmed continuations,
related work, and associated context without forcing the user to reconstruct
the chain.

### Understanding is useful without resuming

Many visits end after the user learns what was said and what the outcome was.
Continuing a session is an available action, not a required end state for every
search or reading journey.

### AI enriches; it does not gate access

Exact transcript viewing, organization, lexical search, capture, and deletion
must work without external AI. AI may add titles, summaries, embeddings,
topics, decisions, and memory candidates, but its provider, locality, inputs,
and cost must be explicit.

### User control is real

Delete, import, export, and any external processing must execute real behavior.
Qratum must not present plans, toggles, or status claims for capabilities that
do not exist.

## Day-to-Day Model

### First run

1. The user launches Qratum.
2. Onboarding opens in the UI.
3. Qratum discovers supported local Claude Code and Codex history without
   displaying transcript contents prematurely.
4. The UI shows the sources, session counts, associated memory/context files,
   date ranges, and estimated local storage.
5. The user approves local gathering and ongoing capture.
6. Qratum builds the initial library and search index.
7. The library opens immediately and visibly progresses while indexing
   continues.

### Normal day

1. The developer works in Claude Code or Codex.
2. Qratum preserves new transcript bytes and relevant source memory/context
   changes locally.
3. The existing library record is updated or a new record appears.
4. Search and recent-work views update automatically.
5. Failures appear in the UI instead of remaining silent.

The user does not rerun initialization to refresh the library.

Gathering is defined separately for every source. A source lifecycle hook is
the preferred candidate when it provides sufficient, stable events and source
identity, but the product contract does not assume that every source has the
same hook semantics. Each adapter must define its capture trigger, in-progress
visibility, expected staleness, retry behavior, and fallback refresh path. The
UI states when an in-progress session has not yet been gathered.

### Restarting or resuming work

When the source tool resumes the same session, Qratum updates the existing
logical session without creating a duplicate.

When the source tool creates a new session that continues earlier work, Qratum
may call it a continuation only when source metadata proves the relationship or
the user confirms it. The user can link, unlink, accept, or reject continuation
relationships.

Repository, working directory, time proximity, shared files, and semantic
similarity may produce a weaker `Related work` suggestion. Related work is kept
separate from the confirmed continuation timeline and never presented as fact.

### When source history disappears

The session remains readable, searchable, and attributable from Qratum's local
copy. The user should not notice that the source tool deleted its copy unless
they inspect provenance.

## Core Object Model, in User Language

### Session

A captured conversation from Claude Code, Codex, Claude.ai, or ChatGPT.

### Continuation

A source-confirmed or user-confirmed relationship between sessions or resumed
segments that belong to the same body of work.

### Related work

A weaker, explicitly suggested relationship based on shared subject matter,
files, repositories, or timing. It is not a continuation unless the source or
the user confirms it.

### Source context

Observed context associated with work, including:

- source-owned memory or instruction files;
- session summaries or checkpoints emitted by Claude or Codex;
- observed versions and observation timestamps; and
- provenance showing exactly which source supplied the context.

Qratum claims that context was active during a session only when source metadata
proves that relationship. Otherwise it says the version was observed near the
session and shows the observation time.

### Derived insight

A summary, outcome, decision, topic, or memory candidate derived from a session.
It remains visibly separate from exact history and observed source context.

### Durable memory

A user-approved item intended for use across agents and sessions. Durable
memory belongs in the user's existing personal-memory project, not in a second
knowledge store hidden inside Qratum.

Qratum may gather source context and organize memory candidates. A later,
explicit integration may send an approved candidate to personal-memory with
consent, provenance, and a returned identifier. Qratum must not silently write
anything into Claude, Codex, personal-memory, or another shared system.

### Session state

Content availability and search freshness are independent properties, not a
single progression.

Content state is per session:

- `Gathering`: Qratum is receiving or updating the session.
- `Available`: exact local history is readable.
- `Failed`: the latest gather or processing operation needs attention.
- `Erased`: the user deliberately removed the session.

Search separately reports whether that session's lexical and semantic indexes
are current, pending, failed, or unavailable. The UI normally hides both when
everything is healthy.

## User Story 1: See My AI Work in One Place

> As a developer, I want one clean library for my Claude and Codex work so that
> I can understand what I have and return to it without navigating tool-specific
> storage.

### Why I need it

My history is fragmented across tools, projects, dates, local files, and memory
systems. The main entry point must answer “what have I been working on?” in the
UI.

### How it works

The default home screen contains:

- a prominent search field;
- recent and resumed work;
- source and repository filters;
- meaningful titles or first-request fallbacks;
- user rename and pin/favorite controls;
- last activity time;
- source identity;
- compact indicators for failures or incomplete indexing; and
- no raw paths, JSON filenames, or pipeline vocabulary.

The library combines available and still-indexing sessions. There is no
separate “raw queue” that the user must drain.

## User Story 2: Read the Exact Session in a Great UI

> As a developer, I want to read the exact conversation in a clean, highly
> usable interface so that the history is valuable, not merely archived.

### Why I need it

Metadata, counts, and synthetic review cards cannot replace the conversation.
Reading the work is the primary payoff.

### Session-detail organization

The session page should feel deliberately designed rather than generated from
internal artifacts.

#### Header

- useful title;
- exact source;
- model when known;
- repository/workspace;
- start and last-activity time;
- session cost with its basis and pricing date;
- continuation/resume relationships; and
- local-only/privacy state.

#### Main conversation

- readable user and assistant turns;
- tool calls collapsed by default but expandable;
- commands, outputs, and file changes rendered intentionally;
- navigation between important turns;
- exact source order;
- search-term highlighting; and
- detected credentials masked by default with deliberate local reveal.

#### Context panel

- concise summary when available;
- stated outcome and unresolved work when available;
- files and commands involved;
- verification signals;
- source context proven active during the session or explicitly labeled as
  merely observed near it;
- extracted decisions or memory candidates with provenance; and
- related or continued sessions.

#### Provenance and safety

- exact source and capture time;
- whether content is exact, deterministically derived, or AI-generated;
- AI provider/model when enrichment exists; and
- export/share eligibility kept separate from local readability.

Raw local readability is allowed. Raw content must never be confused with a
share-safe artifact or silently sent over the network.

### Session cost

The session page shows cost only when it can be calculated honestly:

- use source-reported billed cost when the source provides it;
- otherwise calculate an `API-equivalent usage value` from recorded model
  identity, input/output/cached token usage, and a pinned LiteLLM-style
  price-catalog snapshot;
- show the price source, effective date, calculation basis, and currency;
- never describe an API-equivalent value as the user's actual subscription
  charge; and
- show `unknown` rather than `0` when usage, model identity, or applicable
  pricing is missing.

The bundled catalog records its upstream version or commit and rides Qratum
binary releases. A separate explicit user-initiated import may update it.
Qratum never fetches or updates pricing silently at runtime. Historical entries
remain versioned so a past session uses the price that applied at the session
time, not whichever price is current when the page is opened.

Supported Claude Code per-message usage and supported Codex
delta-from-cumulative usage default to `exact/source-reported`. Tranche 1
fixtures pin those source-version semantics so format drift reduces visible
coverage instead of silently changing totals.

## User Story 3: Continue Work When Useful

> As a developer, I want a direct path from a useful past session back into its
> source tool so that I can continue the work without reconstructing its
> identity or repository context.

### Why I need it

Sometimes finding and understanding the outcome is enough. When it is not,
Qratum should remove the friction between the historical record and the source
tool's native continuation mechanism.

### How it works

- The session page shows `Continue in Claude` or `Continue in Codex` only when
  the source adapter can construct a truthful native continuation action.
- Before launch, Qratum shows the source session identity, repository/working
  directory, and action it will invoke.
- Qratum invokes the source-native resume path directly when the local platform
  supports it and provides a copyable command as a fallback.
- If the original source session can no longer be resumed, Qratum says so and
  keeps the exact history readable. It does not pretend that starting a new
  session is the same operation.
- A newly created source session is linked as a continuation only when the
  source proves it or the user confirms it.

## User Story 4: Find Past Work Reliably

> As a developer, I want to find a session by exact text or by meaning so that I
> can recover decisions and solutions from hundreds of sessions.

### Why I need it

The library already contains hundreds of sessions. Metadata filtering alone is
not enough for the first usable release.

### Search shape

Qratum should use hybrid retrieval:

1. Lexical search for exact errors, commands, filenames, symbols, URLs, and
   quoted phrases.
2. Semantic search for concepts such as “the auth bug with profile routing”
   when the exact wording is unknown.
3. Filters for source, repository, date, model, session state, and continuation.
4. Ranking that combines lexical score, semantic score, recency, and explicit
   filters without hiding why a result matched.

Retrieval happens at passage or turn level, not only at whole-session level.
Results are grouped by session, and selecting a result opens the reader at the
matching place.

Every result shows:

- a useful title;
- source and repository;
- date;
- a matching snippet;
- why it matched, such as exact text, filename, memory, or semantic similarity;
- whether the snippet is exact source or generated summary; and
- its place in a resumed/continued thread.

### Metadata sources

Metadata must not be invented accidentally. It comes from named sources:

- source session ID and timestamps;
- Claude/Codex source identity and model fields;
- working directory and repository identity;
- Git remote, branch, and commit when supplied;
- tool calls, commands, and file paths parsed from the transcript;
- associated source memory/instruction files;
- first user request as a deterministic title fallback; and
- optional AI-derived title, topics, and summary, clearly labeled generated.

### Embeddings decision

Semantic search is a product requirement, but the embedding boundary is still
an implementation decision that must be resolved before the technical spec:

- local embeddings should be the default target;
- external embedding providers must be explicit opt-in;
- the UI must state what content is embedded and where processing occurs;
- embeddings must be rebuildable derived data, not the source of truth; and
- exact lexical search must continue to work without an AI provider.

The technical design must decide how a single-binary product obtains and
versions a local embedding model without pretending that model distribution is
free or invisible.

Both lexical and semantic indexes are raw-derived local artifacts. They must be
owner-only, local-only by default, rebuildable, and covered by session deletion.
Search queries also stay local unless the user explicitly opts into an external
provider after seeing what will be sent.

Search reports partial coverage honestly. Lexical results never disappear
because semantic indexing is incomplete, and the UI shows semantic coverage
while initial indexing or a model-version rebuild is in progress. Re-indexing
must be resumable after interruption.

## User Story 5: Keep My Searchable Memory Current

> As a developer, I want Qratum to gather new and resumed sessions throughout
> the day so that the searchable memory stays current without manual refreshes.

### Why I need it

One-time import is not a working memory. Continuity is part of the product, even
though preservation itself is background infrastructure.

### How it works

- Onboarding enables one real, verified capture mechanism for each accepted
  source.
- Source hooks are the preferred starting mechanism where their lifecycle and
  identity guarantees are sufficient, but this must be proven separately for
  Claude and Codex before the mechanism is accepted.
- The UI shows whether each source is connected and when it last gathered data.
- Resumed sessions update the existing logical history when source identity
  proves continuity.
- Restarted sessions become continuations only when source-confirmed or
  user-confirmed. Weaker matches appear as related work.
- The UI offers a manual “Check for new history” action.
- Capture failure is visible and actionable.

No primary UI exposes hook, backfill, daemon, event spool, or raw-ref language.

## User Story 6: Gather Memories and Context

> As a developer, I want Qratum to gather the memories and instructions around
> my Claude and Codex sessions so that I can recover not only the conversation,
> but the context that shaped it.

### Why I need it

Important context may live outside the transcript in source memory files,
instructions, checkpoints, summaries, or prior-session references.

### How it works

Qratum inventories accepted Claude and Codex memory/context sources separately
from transcripts and records:

- source;
- path or source identifier;
- observed version/digest;
- observation timestamps;
- related repository/workspace; and
- linked sessions when that relationship is known.

The session view distinguishes:

- source context proven active during the session;
- source context merely observed near the session;
- Qratum-extracted decisions or facts;
- AI-generated memory candidates; and
- durable memories explicitly sent to personal-memory.

This requires a separate source contract for Claude and Codex memory artifacts.
Qratum must not scan arbitrary files or infer that every instruction file is a
memory.

## User Story 7: Understand What Was Said and What Happened

> As a developer, I want a concise account of what was discussed, attempted,
> decided, and achieved so that I can recover the outcome without always
> reading the whole session.

### Why I need it

Long sessions are difficult to scan, but an inaccurate summary cannot become
the only representation of the work.

### Two-layer behavior

The always-available deterministic layer:

- parses and organizes turns;
- identifies tools, commands, files, timestamps, and source metadata;
- provides an at-a-glance outline from exact facts such as the first request,
  final response, source-reported summary, commands, and verification results;
- renders the exact conversation cleanly;
- enables lexical search; and
- never requires an AI provider.

The optional AI enrichment layer may produce:

- title;
- narrative summary;
- topics;
- goal and outcome;
- decisions;
- unresolved questions;
- semantic embeddings; and
- memory candidates.

Generated fields must carry provider, model, input class, transform version,
and source provenance. AI output is editable or rejectable and never replaces
the exact conversation.

Whether AI summaries are enabled by default with a local provider remains a
blocking product decision.

## User Story 8: Delete Sensitive History

> As a developer, I want to erase a session and everything Qratum derived from
> it so that Qratum does not retain data I deliberately removed.

### Why I need it

This is mandatory. Sessions can contain credentials, customer data, private
code, or conversations that should not remain indefinitely.

### How it works

The primary deletion flow is in the UI and is addressed by session, not hash or
internal ref ID.

Before confirmation, Qratum shows the affected transcript occurrence, search
index, embeddings, summaries, memory candidates, reports, known continuations,
durable memories previously sent to personal-memory, and known prior exports.
It states clearly that copies already exported outside Qratum are outside the
local deletion operation.

After confirmation:

- the session occurrence is terminally tombstoned;
- raw bytes are removed when no other legitimate occurrence references them;
- indexes, embeddings, summaries, memories, and prepared artifacts are removed;
- storage engines compact or rewrite deleted content where their accepted
  deletion contract requires it;
- ongoing capture and later imports cannot silently resurrect the session;
- the audit trail retains only the minimal opaque identity and deletion proof
  required to prevent resurrection, never transcript content; and
- repeated deletion cannot overwrite the original record.

Qratum must not claim forensic erasure from storage hardware unless a later
encrypted-storage design can prove it. Backup-copy deletion and deletion from
personal-memory are separate authorized operations and must not be falsely
claimed by the local deletion flow.

## User Story 9: Import Claude.ai and ChatGPT History

> As a developer, I want to import exports from Claude.ai and ChatGPT so that my
> web conversations can join the same searchable memory as local Claude Code
> and Codex work.

### Why I need it

Some important work exists only on vendor-managed web products and is invisible
to local continuous capture.

### How it works

Import is initiated and reviewed in the UI:

1. Choose a Claude.ai or ChatGPT export.
2. Qratum identifies the vendor and supported export version.
3. The UI shows conversation counts, date range, attachments or unsupported
   data, estimated storage, and privacy implications.
4. The user confirms local import.
5. Imported conversations enter the same library and hybrid search.
6. Provenance always shows that the source was a vendor export.

Each vendor/version requires fixtures and a strict adapter. Arbitrary JSON or
JSONL is not accepted as a supported conversation export. An unrecognized or
changed vendor format fails closed without partially mutating the library.

## User Story 10: Export My Work

> As a developer, I want to take selected sessions and memories out of Qratum so
> that the product never becomes the only usable copy of my work.

### How it works

The UI lets the user select sessions, destination, and content:

- exact local transcript export;
- cleaned/redacted transcript;
- summaries, outcomes, cost breakdowns, and memory candidates;
- deterministic provenance/evidence; and
- machine-readable provenance.

The UI distinguishes exact raw export from share-oriented redacted export.
Every export shows what will leave Qratum, requires confirmation, executes the
copy, records provenance, and verifies the result. The session deletion view
can name known export destinations but cannot claim to delete those external
copies.

## User Story 11: Know Whether Qratum Is Working

> As a developer, I want the UI to tell me whether gathering, indexing, search,
> and deletion controls are healthy so that I can trust Qratum day to day.

### How it works

The normal UI shows a quiet status indicator. It becomes prominent only when a
source is stale, capture fails, indexing is incomplete, storage is low, or a
privacy-sensitive action needs attention.

A diagnostic view explains:

- connected sources;
- last successful gather;
- sessions gathered and indexed;
- in-progress sessions not yet available under the source's capture cadence;
- lexical and semantic coverage separately;
- index/model versions;
- failed jobs;
- local storage use;
- per-component storage use and the available reduction actions;
- an explicit `this machine only` boundary while cross-machine merge remains
  deferred;
- deletion/tombstone consistency; and
- concrete recovery actions.

No status is hard-coded. No proof is green unless it ran.

## Deferred User Story: Protect Against Disk Loss

Verified backup and restore are important, but the product is already losing
searchable work today. Backup/restore needs a deliberate design after the core
UI, continuous gathering, reading, search, memory, and deletion flows are
correct.

Until verified backup ships, Qratum must describe itself as a local searchable
memory, not guarantee protection against disk failure. The on-disk format should
remain documented and recoverable without requiring Qratum.

## UI Information Architecture

The first polished application should contain fewer, better surfaces.

### Search/Home

- dominant search box;
- recent and resumed work;
- source/repository/date filters;
- clear indexing progress; and
- useful result snippets.

### Session

- exact conversation as the main content;
- date, source, model, repository, and honestly calculated session cost;
- summary, outcome, and unresolved work;
- context and memories alongside it;
- generated summaries clearly labeled;
- continuation timeline;
- related work kept separate from continuations;
- source-native continue action when available;
- provenance and privacy controls; and
- delete/export actions.

### Imports

- Claude.ai and ChatGPT export intake;
- plan and confirmation;
- progress and failures; and
- resulting library entries.

### Settings and Health

- Claude and Codex source connections;
- capture status;
- search/index model settings;
- local versus external AI disclosure;
- storage use;
- diagnostics; and
- advanced trust evidence.

There should not be separate primary pages for raw queue, refinery, artifacts,
jobs, DTOs, vault operations, or data classes.

## Minimal Public CLI

UI-first means the public CLI should be smaller than previously proposed.

Proposed minimum:

```text
qrt              # start or open Qratum; show onboarding on first run
qrt init         # automation/headless bootstrap path
qrt doctor       # recovery when the UI cannot start or is unhealthy
qrt trust        # only if it executes real self-contained proof
```

Source-tool capture adapters may require hidden machine entrypoints. They are
not user workflow commands.

Import, export, delete, source enablement, search, and session reading belong in
the UI first. CLI equivalents may be added later only for demonstrated
automation needs.

## Normal User Journey

```text
Install and launch Qratum
    -> complete onboarding in the UI
    -> approve Claude/Codex gathering and local indexing
    -> land in a populated library while indexing continues
    -> return later to the same always-available library
    -> search by exact text or meaning
    -> open the exact session in a polished reader
    -> see what was said, what happened, its cost, context, and relationships
    -> leave with the answer, or optionally continue, export, or delete
```

During daily work, Qratum gathers and indexes in the background. Restarting
Qratum or resuming a Claude/Codex session does not repeat onboarding.

## Ordered Release Tranches

The following list is dependency order, not a promise that everything ships as
one release. Work on a later tranche does not begin until the preceding tranche
passes its contract and installed-artifact verification. The daily loop is not
called usable before Tranche 2, the product does not satisfy its promised search
experience before Tranche 3, and no capability is called shipped before its
proof runs.

### Tranche 0: Product truth

- accept the product contract;
- update `SPEC.md` and `AGENTS.md` to point at it;
- mark conflicting state, raw-viewing, CLI, and shipped-claim sections of older
  specs as superseded; and
- define the minimum reference information every session must expose.

### Tranche 1: Source correctness

- separate Claude Code and Codex adapter contracts and fixtures;
- verify the accepted capture mechanism for each source;
- gather exact local history with correct source identity;
- update resumed sessions without duplicates;
- preserve named provenance and owner-only storage; and
- prove behavior from the installed artifact.

This tranche is foundation work and is not presented as the usable product.

### Tranche 2: Daily product spine

- UI-first onboarding and normal operation;
- one unified library;
- polished exact transcript reading with credential masking and local reveal;
- truthful date, source, deterministic at-a-glance outline, available
  source-reported summary/outcome, and session-cost presentation;
- turn/passage-level lexical search grouped by session;
- rename and pin/favorite controls;
- source-native continue action where truthful;
- source-confirmed and user-confirmed continuation handling;
- terminal session deletion covering local raw and derived data;
- real exact transcript export;
- truthful capture, storage, and lexical-index health; and
- the minimal public CLI.

### Tranche 3: Semantic retrieval

- local-default embeddings;
- hybrid lexical and semantic ranking;
- partial-coverage disclosure;
- resumable model-version re-indexing; and
- installed privacy, deletion, and retrieval verification.

Completion of this tranche is the first point at which Qratum satisfies the
document's required hybrid-search experience for hundreds of sessions.

### Tranche 4: Context and imports

- allowlisted observed source context for Claude and Codex;
- explicit personal-memory handoff contract for approved durable memories;
- strict Claude.ai and ChatGPT export import; and
- truthful source-context timing and provenance.

### Tranche 5: Optional enrichment and sharing

- clearly labeled narrative AI summaries, outcomes, decisions, and memory
  candidates;
- editable/rejectable generated fields;
- weaker related-work suggestions;
- real redacted/share-oriented export; and
- provider, locality, input-class, cost, and provenance disclosure.

### Out until separately designed

- verified backup and restore product flow;
- retention policies;
- cross-machine merge;
- revision/checkpoint history as a user-facing feature;
- per-line or per-turn permanent redaction while retaining the rest of a
  session;
- corpus generation and publishing;
- automatic memory writeback to Claude, Codex, or shared stores;
- enterprise control plane;
- MCP;
- marketplace;
- skill mining; and
- claims that deterministic review proves correctness.

## Blocking Decisions Before Technical Specification

### Decision 1: Semantic search runtime

How will a single-binary local product obtain, version, update, and run a local
embedding model? External embeddings cannot be the default without changing the
privacy promise.

### Decision 2: AI enrichment default

Should local AI summaries, embeddings, and memory candidates run automatically
when a local provider is available, or only after explicit user enablement? May
a proven-local model read exact raw transcripts, or only masked text?

### Decision 3: Claude and Codex memory sources

Which exact memory, instruction, checkpoint, and summary artifacts are accepted
from each source? This must be an allowlist with fixtures, not broad filesystem
scanning.

### Decision 4: Continuation identity

Which source identifiers prove a resume for Claude and Codex? Continuations
require source or user confirmation; repository, time, file, and semantic
overlap may only suggest related work.

### Decision 5: Always-available local app

Should Qratum run only when opened, use source hooks plus on-demand UI, or
install a small local background service? The answer must support continuous
gathering without hiding a resident service from the user. Hooks are the
preferred candidate, but their event timing, identity, retry, and mid-session
coverage must be verified separately for Claude and Codex.

### Decision 6: Personal-memory handoff

Which explicit mechanism sends an approved durable memory to the user's
personal-memory project, returns its identifier, and supports later correction
or deletion? This is not assumed to be the same mechanism as transcript capture.

### Decision 7: Session pricing authority

Resolved on 2026-07-12: use a pinned LiteLLM-style model-price catalog snapshot
that rides Qratum binary releases, with an optional explicit user-initiated
import and no silent runtime fetch. Only an explicit source-reported charge is
actual billed cost; calculated values are API-equivalent usage values.

## Acceptance Boundary

The product owner accepted the user needs and product direction in this
document on 2026-07-11. This acceptance does not accept the existing
implementation, schemas, command signatures, model/provider choices, storage
backend, or release plan. Those require later contracts after the blocking
decisions are resolved.

The Tranche 0 authority update is recorded in
`specs/current/product-direction.md`, `SPEC.md`, and `AGENTS.md`. Tranches 1
through 5 still require separately accepted technical contracts; no agent
should implement them from this review document alone.

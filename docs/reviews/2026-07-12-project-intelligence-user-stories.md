# Qratum Project And Workstream Intelligence

Status: accepted product extension; incorporated into canonical direction

Date: 2026-07-12

Accepted by product owner: 2026-07-12

## Purpose

This document puts the Project, Workstream, analytics, summaries, decisions,
open work, and direction/roadmap ideas into user stories before they change the
accepted Qratum contract.

It extends and is incorporated by:

```txt
specs/current/product-direction.md
docs/reviews/2026-07-10-product-user-stories.md
```

The accepted core remains exact local history, search, continuity, truthful
cost, source context, deletion, and user control. This extension asks whether
Qratum should organize that evidence around projects and workstreams and use
optional AI to explain what the work means and where it appears to be going.

This document is accepted product evidence, not implementation authority. Its
accepted rules are incorporated into the canonical product-direction spec;
every implementation wave still requires a separate technical contract.

The Fable review and the product owner's disposition of its usage-reliability
and pricing findings are recorded in:

```txt
docs/reviews/2026-07-12-project-intelligence-fable-review.md
```

The accepted owner choices and answers are recorded in:

```txt
docs/reviews/2026-07-12-project-intelligence-owner-decisions.md
```

## Product Bet

Qratum can add project organization and accounting without becoming a generic
project-management product or replacing its accepted searchable-memory thesis.

> Qratum is the private searchable memory of AI-assisted work, with optional
> project views that explain where the work happened, what it cost, and which
> exact evidence supports the user's organization and direction.

The existing thesis still holds. Projects and workstreams are organizing views
over exact session evidence, not replacements for that evidence.

## Product Boundary

Qratum owns:

- exact AI session history and its provenance;
- project and workstream organization over that history;
- deterministic usage and cost accounting;
- links between sessions and observed Git/code activity;
- source-reported, deterministic, AI-generated, and user-confirmed information
  as separately labeled classes;
- optional summaries, decisions, open-loop suggestions, and roadmap candidates;
  and
- drill-down from every aggregate or generated statement to supporting
  evidence.

Qratum does not replace:

- Git as the owner of commits, branches, and worktrees;
- GitHub or another forge as the owner of issues and pull requests;
- Jira or another task system as the owner of formal team planning;
- personal-memory as the owner of durable cross-agent memory;
- source tools as the owner of native session continuation; or
- billing providers as the owner of actual invoices and subscription charges.

Qratum does not score developer productivity, claim one model caused a better
outcome, or turn an assistant suggestion into an accepted plan without the
user.

## Information Classes

The application must keep four layers structurally distinct:

```txt
Exact source evidence
  transcripts, timestamps, usage, model, repository, commands, Git metadata

Deterministic derivations
  cost calculations, totals, coverage, exact Git links, timelines

Semantic projections
  embeddings, similarity, suggested related sessions and workstreams

LLM-generated interpretations
  narrative summaries, outcomes, decisions, open loops, roadmap candidates
```

User confirmation is a fifth state applied to a proposed relationship or
interpretation. Confirmation does not make the original generated text exact;
it records that the user accepted or edited it.

Exact content is the unbadged default in the UI. Deterministic derivations,
semantic relationships, generated interpretations, and user-confirmed
statements carry consistent visible markers and link to their evidence.

## No Standing Curation Queues

Qratum does not turn suggestions into inbox debt.

- `Unassigned` is a healthy permanent state, not work the user owes Qratum.
- Suggestions are generated on explicit request or shown inline where they are
  immediately relevant.
- The home and project overviews do not show unread-suggestion or
  pending-curation counters.
- Overview surfaces show accepted or confirmed items, not accumulated
  candidates.
- Generated extraction is bounded to a user-selected project, time range,
  session set, or workstream.
- Rejected suggestions are not repeated unless new evidence appears, and that
  new evidence is shown.

## Core Object Model, In User Language

### Project

A user-owned grouping representing one product or body of work. A project may
contain multiple repositories, clones, worktrees, and explicitly included
folders.

Project identity is stable and user-controlled. It is never derived solely
from a filesystem path.

### Repository

A logical Git repository identity. Qratum may recognize several clones or
worktrees as the same repository.

### Working copy

One observed local checkout or Git worktree at a particular path and machine.
Paths may move or disappear without changing historical project membership.

### Workstream

A user-confirmed feature, bug, initiative, migration, or other coherent body of
work within a project. A workstream may span sessions, repositories, branches,
commits, and time periods.

A confirmed continuation thread is the cold-start unit for a workstream. Naming
or promoting that thread creates a workstream, and one workstream may join
several continuation threads. Project views remain complete with zero
workstreams.

### Usage record

A source-provided or explicitly estimated record of model/token consumption.
Project and workstream costs aggregate usage records, not re-tokenized
transcript text when source usage is available.

### Code link

A relationship between a session or workstream and a file, commit, branch,
pull request, issue, or release. Every link records whether it is
transcript-proven creation, transcript mention, user-confirmed association, or
weaker deterministic/semantic correlation.

Forge references are inert identifiers and links in this extension. Qratum does
not fetch or claim current issue or pull-request state without a separate
opt-in network-integration contract.

### Decision

A concise project or workstream choice linked to the exact evidence where it
was discussed. A decision may be proposed, accepted, rejected, or superseded.

### Open loop

An unresolved question, failed verification, blocker, deferred task, or
unfinished outcome linked to its evidence.

### Roadmap item

A user-owned direction item. On explicit request, Qratum may propose one from
accepted decisions, confirmed open loops, explicit user requests, user-authored
turns, transcript-proven code evidence, and inert issue/PR references. Only the
user can accept it or change its state.

### Project summary

A time-bounded explanation of what changed, what completed, current
workstreams, important decisions, unresolved work, usage, cost, and coverage.
Every generated statement links to evidence.

### Context pack

A user-selected package of exact excerpts, summaries, accepted decisions,
open loops, repository context, and approved memories used to start or continue
work in a source tool.

## Project And Repository Identity

Paths are observations, not identity.

The relationship is:

```txt
Project
  -> Repository
      -> Working copy / worktree
          -> Sessions observed there
```

For every Git-backed session Qratum records, when available:

- repository root;
- Git common directory;
- working-copy/worktree path;
- sanitized remote identities;
- branch and commit;
- machine identity;
- observation time; and
- how the project assignment was made.

Exact Git identity lives in owner-only metadata, separate from share-safe or
redacted artifacts. Remotes are sanitized before comparison or display. Branch
names and other displayed Git fields receive credential masking, and exact Git
metadata appears in an export only when the egress plan lists it explicitly.

### Same clone, different worktrees

Worktrees that share a Git common directory are occurrences of one repository,
even when their paths are unrelated.

### Separate clones

Independent clones may be suggested as the same repository when normalized
remote identities match. For example:

```txt
git@github.com:edictum-ai/qratum.git
https://github.com/edictum-ai/qratum.git

normalize to:

github.com/edictum-ai/qratum
```

Remote credentials, tokens, and user-info must be removed before storage or
display.

### Forks and ambiguous remotes

Matching an upstream remote does not prove two repositories are the same
project. Forks, vendored copies, repositories without remotes, and ambiguous
folder rules require user confirmation.

### Moved or deleted working copies

Historical sessions remain attached to their repository and project. The UI
marks the old working-copy path unavailable and may learn a new occurrence.

### Multiple repositories in one project

A project may explicitly group related repositories:

```txt
Project: Edictum

Repositories:
  edictum
  edictum-ts
  edictum-go
  edictum-schemas
  edictum-api
  edictum-app
```

### Assignment safety

- Repository grouping works automatically without requiring a Project.
- A Project is an optional user-named grouping above repositories.
- Qratum may suggest Project membership but does not merge Projects silently.
- Cold start groups suggestions by repository or normalized remote and supports
  one batch confirmation per group; it never requires one decision per session.
- The user can add, move, split, or remove a repository or working copy.
- Uncertain sessions remain `Unassigned`.
- A session gets one primary project for cost aggregation unless the source
  supplies defensible per-project usage segments.
- Qratum does not divide a session's cost between projects by guesswork.
- Moving a session changes its project attribution with an audit record; it
  does not duplicate the session or usage.
- Assignment provenance distinguishes working-directory observation, captured
  Git identity, source evidence, user confirmation, and multi-repository
  activity when available.
- Machine identity is provenance only while cross-machine merge is deferred.
- A rejected grouping is remembered and is suggested again only when new
  evidence is available.

### Project lifecycle and recoverability

- Rename and archive do not change session evidence.
- Merge and split preserve an audit record and visibly recalculate affected
  aggregates.
- Deleting a Project removes its generated Project artifacts but never erases
  sessions implicitly; sessions return to repository or `Unassigned` views.
- Project organization, assignments, corrections, and user-confirmed
  relationships are included in owner export because they cannot all be
  reconstructed from transcripts.

## Usage And Cost Ledger

Project cost must be reconcilable to usage records and sessions.

### Usage dimensions

When supplied by the source, Qratum records:

- input tokens;
- output tokens;
- cached input/read/write tokens;
- reasoning tokens;
- model identity;
- source event or usage-record identity;
- usage time;
- source session identity;
- whether the record is exact, cumulative, incremental, or estimated;
- a source-version reliability class; and
- billing mode: API, subscription, or unknown.

Supported Claude Code per-message usage and supported Codex
delta-from-cumulative usage default to `exact/source-reported`. The reliability
field exists for future sources and format drift, not as a warning on the
initial supported sources. Wave 1 fixtures pin the accepted field semantics
for each supported source version. An unrecognized shape reduces visible
coverage and reports a diagnostic instead of producing silent zeros.

### Attribution dimensions

These are separate fields, not one overloaded `agent` label:

```txt
source    Claude, Codex, Claude.ai, or ChatGPT
surface   Claude Code, Codex CLI, Codex app, or another client
harness   Ductum or another orchestrator when proven
agent     named agent, subagent, or role when reported
model     exact reported model
```

Missing attribution displays `unknown`. Qratum does not infer that a session
used Ductum merely because it occurred in a repository Ductum also touches.

### Cost truth

- Use source-reported billed cost when available.
- Otherwise calculate an `API-equivalent usage value` from usage, model, and a
  versioned price catalog.
- Use a pinned snapshot of a LiteLLM-style model-price catalog, including its
  upstream version or commit identity.
- Refresh the bundled snapshot through Qratum binary releases. A separate,
  explicit user action may refresh it from Qratum's allowlisted pricing source
  or import it from a file.
- Never fetch or update the price catalog silently at runtime.
- Record price source, effective date, currency, and formula.
- Use the price applicable at the usage time only when the catalog evidence
  proves it; otherwise show which catalog date was used and do not call the
  value historical billed cost.
- A recalculation using today's price may exist only as a separately labeled
  comparison.
- Subscription charges are not API usage costs.
- For subscription use, calculated dollars are labeled `API-equivalent usage
  value`, never spend or billed cost.
- Missing usage/model/price produces `unknown`, never zero.
- The UI shows how much of the project has calculable versus unknown cost.

### No double counting

- Resumed session observations aggregate incremental usage rather than summing
  repeated cumulative totals.
- Imports and local capture deduplicate the same source usage record.
- Several worktrees pointing at one repository do not duplicate usage.
- A session moved between projects retains one usage ledger; attribution
  changes, usage does not clone.
- Generated Qratum AI cost is tracked separately from the cost of the source
  session being analyzed.

## User Story 1: Create A Project Across Repositories And Worktrees

> As a developer, I want one project to contain all its repositories, clones,
> folders, and worktrees so that path layout does not fragment my history.

### Why I need it

One product may span many repositories and each repository may have several
worktrees or clones in unrelated locations. Folder-based identity would split
one project into false duplicates.

### How it works

1. Create or select a project.
2. Review repository and working-copy suggestions grouped by repository or
   normalized remote.
3. Confirm, reject, or move a whole group, with per-session correction only
   when needed.
4. See why Qratum suggested the relationship.
5. Keep uncertain sessions and repositories in `Unassigned`.
6. Add explicit folders for non-Git work only after confirmation.

The project remains stable when paths move or worktrees are removed.

## User Story 2: See A Project Overview

> As a developer, I want one page that explains the current shape of a project
> so that I can understand its activity without reconstructing it from sessions.

The overview shows:

- repositories and working copies;
- session count and date range;
- current user-confirmed workstreams, when any exist;
- accepted outcomes and confirmed open work;
- token and cost totals with coverage;
- source, surface, harness, agent, and model breakdowns;
- accepted decisions;
- user-owned roadmap status, when enabled; and
- links from every number or generated statement to its evidence.

Deterministic totals render without an LLM. Generated summaries are clearly
optional and labeled.

## User Story 3: Know What A Project Cost

> As a developer, I want to see token usage and cost for one project or time
> period so that I understand what the AI work consumed.

The Project Costs page supports:

- lifetime, day, week, month, and custom ranges;
- tokens and cost by repository;
- source, surface, harness, agent, and model;
- workstream and session;
- actual billed cost, API-equivalent usage value, and unknown cost as separate
  totals;
- pricing catalog and effective-date disclosure;
- usage/cost coverage percentages; and
- drill-down to every contributing usage record and session.

The headline never presents a partial total without its coverage.

## User Story 4: Compare Models, Sources, And Harnesses

> As a developer, I want factual comparisons of how models and harnesses were
> used so that I can understand spending and workflow patterns.

Safe comparisons include:

- session and usage count;
- token and cost totals;
- cache usage;
- recorded verification activity;
- source-reported or user-confirmed outcomes; and
- unknown/coverage rates.

Qratum does not claim that a model or harness is more productive, higher
quality, or causally responsible for an outcome without an accepted evaluation
contract.

## User Story 5: Search And Browse Within A Project

> As a developer, I want project-scoped search and recent history so that
> unrelated repositories and sessions do not overwhelm retrieval.

Project search uses the accepted hybrid retrieval contract. Results still
identify passage-level evidence, exact versus generated text, and why they
matched. Project scope is an explicit filter backed by confirmed project
membership, not path-prefix matching.

## User Story 6: Organize Work Into Workstreams

> As a developer, I want sessions grouped into meaningful features, bugs, and
> initiatives so that I can understand what the project actually spent effort
> on.

### Manual core

After Projects and costs have demonstrated daily use, the user can promote a
confirmed continuation thread into a named workstream. Without AI, the user
can then:

- create and name a workstream;
- attach or detach sessions;
- attach repositories, branches, commits, PRs, or issues;
- merge, split, complete, abandon, or reopen workstreams; and
- see tokens, cost, outcomes, and open loops for the workstream.

Every Project, cost, search, session, and timeline surface works without any
workstream. Workstreams are saved evidence bundles, not a required taxonomy.

### Suggested workstreams

Workstream suggestions are acceptance-gated later work. When explicitly
requested, embeddings may cluster a selected session set and an enabled LLM may
name or summarize the cluster. Suggested membership is not fact and never
accumulates in an inbox.

The user can confirm, rename, merge, split, or reject every suggestion. The UI
shows which sessions and evidence caused the suggestion.

## User Story 7: Link Sessions To Code And Delivery Evidence

> As a developer, I want to know which sessions relate to commits, files, pull
> requests, issues, and releases so that I can recover why code exists.

Links may come from:

- source-reported Git metadata;
- commands and tool calls recorded in the session;
- a commit-producing tool call and its resulting identifier;
- exact commit, PR, or issue identifiers merely mentioned in the transcript;
- deterministic time/branch/file correlation;
- user confirmation; or
- AI suggestion.

The UI labels the evidence level. Only transcript-proven creation may answer
which session produced a commit. A mentioned identifier proves only that the
session encountered it. Correlation is capped, shown on demand, and never
phrased as causation. Shared files or timestamps alone do not prove that a
session produced a commit.

PR and issue references are inert links. This extension does not fetch their
current state.

Useful questions include:

- Which transcript-proven session created this commit?
- Which sessions mentioned or were user-associated with this PR?
- Why was this design chosen?
- Which workstream produced this release change?
- How much AI usage is attributable to this confirmed body of work?

## User Story 8: Read A Project Summary

> As a developer, I want a concise, time-bounded summary of a project so that I
> can recover what changed without opening every session.

A generated summary may include:

- what changed;
- completed and abandoned work;
- active workstreams;
- important decisions;
- unresolved questions and blockers;
- releases and code changes;
- usage, cost, and coverage; and
- notable changes since the previous summary.

Every statement links to exact sessions, accepted decisions, commits, PRs, or
usage records. The summary records time range, provider, model, input class,
input digests, prompt/transform version, generation cost, and generated time.

## User Story 9: Maintain A Project Decision Log

> As a developer, I want important decisions collected with their reasoning and
> evidence so that I do not have to remember which conversation settled them.

Decision states are:

```txt
Proposed
Accepted
Rejected
Superseded
```

On explicit request, an LLM may propose a concise decision from a bounded
evidence set. Repeated discussion produces one candidate with several evidence
links, not duplicate candidates. Only the user can accept it. The user can edit
wording, reject it, or supersede it with a linked replacement. The exact
supporting turns remain accessible. Proposed decisions do not accumulate in a
standing inbox.

Accepted durable decisions may be explicitly promoted to personal-memory under
the separately accepted handoff contract.

## User Story 10: Track Open Loops

> As a developer, I want unresolved questions, failed verification, blockers,
> and deferred work collected so that useful unfinished work does not disappear
> inside old sessions.

On explicit request over a bounded scope, Qratum can deterministically identify
some signals, such as a final failing test. An enabled LLM is needed to
interpret broader unresolved questions and deferred intent.

Open loops remain suggestions until confirmed. Users can accept, edit, dismiss,
resolve, or attach them to a workstream or roadmap item.

Every candidate shows evidence age. Stale candidates may expire into an archive
and can be dismissed in bulk. Open-loop suggestions never create unread counts
or a persistent task inbox.

## User Story 11: Build A User-Owned Direction And Roadmap

> As a developer, I want Qratum to propose project direction from session and
> code evidence so that repeated decisions and unfinished work can become an
> explicit plan.

The roadmap is user-owned. On explicit request, Qratum may propose items from:

- explicit user requests;
- accepted decisions;
- confirmed open loops;
- user-confirmed workstreams;
- transcript-proven code evidence; and
- user-authored turns in the selected evidence scope.

Assistant-authored boilerplate, repeated assistant themes, abandoned plans,
unconfirmed suggestions, and unfetched forge state do not become roadmap work.

Roadmap states are:

```txt
Suggested
Accepted
Done
Rejected
```

Only the user can move an item from `Suggested` to `Accepted`. Qratum may
suggest a status change from later evidence but cannot move any item, including
to `Done`.

Accepted items may be organized by the user into `Now`, `Next`, and `Later`.
Suggestions remain in a separate review surface and never render inside those
columns. Generated delivery dates and engineering estimates are out unless
entered or accepted by the user.

Every roadmap candidate shows:

- why it was suggested;
- supporting sessions, decisions, workstreams, and code evidence;
- dependencies;
- cost already associated with the workstream;
- confidence/evidence class; and
- what remains unresolved.

The evidence panel also shows the newest relevant evidence, evidence age, and a
contradiction/supersession check. Opening that evidence is required before
acceptance. Superseded or newer contrary evidence suppresses a candidate rather
than appearing beside it as equal support.

## User Story 12: See A Project Timeline

> As a developer, I want a chronological view of sessions, decisions,
> workstreams, code changes, releases, costs, and memories so that I can answer
> what happened over time.

The timeline is assembled from exact and clearly labeled derived events. It can
filter by repository, workstream, source, model, harness, event type, and date.
Generated project summaries may reference the timeline but never replace it.

## User Story 13: Create A Context Pack

> As a developer, I want to select the relevant project evidence for new work
> so that I can continue with good context without reopening a stale or enormous
> session.

A context pack may contain:

- selected exact excerpts;
- session and workstream summaries;
- accepted decisions;
- confirmed open loops;
- repository, branch, and recent verification context;
- relevant code/PR links; and
- approved personal memories.

Selection works without an LLM. Optional LLM compression or rewriting is a
separate generated artifact. Before sending anything to a source tool or
external model, Qratum shows exactly what will leave and where it will go.

Every pack carries Project, Workstream, and source-evidence provenance. If a
later source session reports that provenance, Qratum can source-confirm the
resulting relationship instead of guessing from repository or time proximity.

## User Story 14: Understand Coverage And Data Quality

> As a developer, I want to know which project totals and summaries are
> complete so that I do not mistake partial data for truth.

The project shows:

```txt
sessions discovered
exact history available
project assigned
repository identified
token usage available
cost calculable
usage source version supported
surface/harness/agent identified
lexical indexed
semantic indexed
summarized
unknown or unassigned
```

Every aggregate carries its denominator and unknown count. A cost total without
coverage is not a valid headline.

Project and cost views also disclose source gather lag. An in-progress session
that is not captured until session end does not silently appear as complete
coverage for today.

## User Story 15: Correct And Curate The Project

> As a developer, I want simple manual controls so that Qratum remains useful
> when automatic organization is wrong.

Manual controls include:

- rename and pin sessions;
- bookmark important turns;
- add notes;
- correct project or repository assignment;
- create, merge, split, and rename workstreams;
- confirm or reject links;
- accept, edit, reject, or supersede decisions;
- accept or dismiss open loops; and
- accept, reorder, complete, or reject roadmap items.

Corrections and rejections persist and outrank future generated suggestions.
A rejected suggestion is reconsidered only when new evidence appears, and the
changed evidence is shown.

## User Story 16: Explain Cost Anomalies

> As a developer, I want unusual spending and missing coverage called out so
> that I can understand why a project total changed.

Deterministic anomaly examples include:

- a session costing several times the project median;
- a sudden cache-utilization change;
- most spending concentrated in a few sessions or workstreams;
- a pricing change affecting estimates;
- many sessions with unknown usage; and
- duplicate or cumulative usage records rejected from totals.

An LLM may explain the surrounding workstream context, but the anomaly and the
numbers remain deterministic and drill down to usage records.

## User Story 17: Erase A Session Without Hiding The Project Impact

> As a developer, I want session deletion to remove its usage and derived data
> while explaining how Project totals and intelligence will change.

Privacy wins over preserving a historically unchanged ledger:

- deletion removes the session's usage records;
- project and workstream aggregates are recomputed;
- affected totals disclose that they exclude erased sessions using tombstone
  counts, without retaining erased usage or content;
- deletion preview lists affected totals, summaries, workstreams, decisions,
  open loops, roadmap evidence, embeddings, context packs, and memory handoffs;
- generated outputs consuming the session are removed or marked stale before
  they can be shown again; and
- an accepted user-owned statement may survive only under the separately
  accepted rule for evidence that has been erased.

Deletion never leaves an anonymous usage record behind merely to keep an old
total reproducible.

## AI Requirement Matrix

The application must be useful with generative AI disabled.

| Capability | Required intelligence |
| --- | --- |
| Project/repository/worktree grouping | Git metadata plus user confirmation; no LLM |
| Token and cost ledger | Deterministic; no LLM |
| Source/model/harness/agent attribution | Source metadata; no LLM |
| Project timeline and coverage | Deterministic; no LLM |
| Manual workstreams, decisions, roadmap, notes | No LLM; later and acceptance-gated |
| Lexical project search | No LLM |
| Semantic project search | Embedding model |
| Related-session/workstream clustering | Embeddings; later and on request; LLM optional for naming |
| Narrative session/workstream/project summaries | Generative LLM unless source-supplied |
| Automatic decision and open-loop extraction | Generative LLM for useful semantic interpretation |
| Roadmap candidates and project-direction synthesis | Generative LLM |
| Context-pack selection | No LLM |
| Context-pack compression/rewrite | Generative LLM optional |
| Cost anomaly detection | Deterministic statistics; LLM optional explanation |
| Memory-candidate generation | Generative LLM |

No core project, usage, cost, exact-history, manual-organization, or lexical
search behavior requires a generative LLM.

## AI Synthesis Pipeline

The optional AI pipeline is incremental:

```txt
Exact session, Git, usage, and project evidence
  -> changed-session summary/outcome
  -> affected-workstream cluster/summary
  -> affected-project summary
  -> decision and open-loop candidates
  -> roadmap candidates
  -> user acceptance/edit/rejection
```

This pipeline runs only for a user-selected scope after the corresponding
feature has earned implementation scope. It does not run merely because a page
opens and does not accumulate candidates in a standing inbox.

Generated outputs cache by:

- exact input object IDs and digests;
- provider and model;
- local or external execution;
- input data class;
- prompt/transform version;
- output digest;
- generation time; and
- token/cost usage for Qratum's own analysis.

The input-object index is also the deletion-propagation index: Qratum must be
able to enumerate every generated output that consumed a deleted session.

When a session changes, only its dependent workstream and project outputs
become stale. Stale generated summaries remain visibly stale until refreshed;
they do not silently masquerade as current.

## AI Safety And Privacy

- Exact and deterministic project views work without external AI.
- Embeddings are local by default under the accepted search boundary.
- A proven-local model may read exact history only after its wave contract
  defines and verifies the local boundary.
- External AI is opt-in and receives only the explicitly displayed input class.
- External raw input remains blocked unless a later accepted contract changes
  that rule.
- Generated statements link to evidence and remain editable/rejectable.
- Project names, Workstream names, summaries, decisions, open loops, roadmaps,
  and context packs are owner-only raw-derived artifacts.
- Before any external request, the egress preview lists repository, Project,
  Workstream, machine, and Git metadata as well as transcript content.
- Refresh cost is shown before invoking an external or metered model.
- Generated roadmaps cannot create issues, mutate repositories, or dispatch
  agents without a separate explicit action and contract.
- Qratum tracks the cost of its own AI analysis separately from project source
  session cost.

## Application Information Architecture

### Projects

- project list;
- unassigned sessions/repositories;
- create/edit project;
- contextual repository and working-copy suggestions, with no pending counter;
- project coverage and recent activity.

### Project Overview

- recent exact activity and an optional time-bounded generated summary;
- deterministic usage/cost cards with coverage;
- user-confirmed workstreams, when enabled;
- accepted decisions and confirmed open loops, when enabled;
- user-owned roadmap snapshot, when enabled;
- source/model/harness breakdown;
- recent sessions and code changes.

### Workstreams

- manual workstreams promoted from continuation threads;
- suggestions only after explicit user request;
- sessions and code links;
- goals, summaries, outcomes, decisions, and open loops;
- cost and usage;
- merge/split/confirm/reject controls.

### Roadmap

- user-owned `Now`, `Next`, and `Later` columns for accepted items;
- suggested items in a separate evidence-review surface;
- evidence and dependencies;
- edit, reorder, accept, reject, and complete controls.

### Decisions

- accepted, rejected, and superseded decisions;
- proposed decisions only in an explicit bounded review;
- exact supporting evidence;
- project/workstream filters;
- explicit personal-memory promotion.

### Costs

- tokens and cost by time, repository, workstream, source, surface, harness,
  agent, model, and session;
- actual/API-equivalent/unknown separation;
- coverage and pricing basis;
- anomaly explanations;
- drill-down to usage records.

### Sessions

- project-scoped library and hybrid search;
- exact reader;
- project/workstream assignment;
- code links, decisions, open loops, and bookmarks.

### Repositories

- logical repositories;
- clones and worktrees by machine/path;
- normalized sanitized remotes;
- project membership and ambiguity explanations;
- missing/moved working-copy state.

### Timeline

- sessions, code links, decisions, workstreams, roadmap changes, releases,
  usage/cost, and durable-memory handoffs.

## Proposed Wave Placement

This extension does not change the accepted waves until approved. If
accepted, the smallest coherent placement is:

### Wave 1 additions: capture-time facts and usage fixtures

Add only facts that may disappear before a later scan:

- repository and Git common-directory identity observed while a working copy
  still exists;
- working-copy/worktree path, machine, sanitized remotes, branch, and commit at
  observation time;
- source-reported surface, agent, model, and harness evidence;
- assignment provenance supplied by the source; and
- source-version fixtures that pin Claude Code per-message usage and Codex
  delta-from-cumulative semantics.

Project assignment, price-catalog interpretation, usage aggregation, and cost
views are re-derivable from preserved history. They do not delay the accepted
daily spine.

### Wave 2 additions: repository awareness only

Keep the accepted daily library, reader, lexical search, continuation,
deletion, export, health, and CLI spine ahead of the new Project product.

The only additions are:

- automatic repository grouping across worktrees and clones; and
- repository filters in the library and lexical search.

These improve the core without requiring the user to create or curate a
Project.

### Wave 2.5: deterministic Project product

- optional multi-repository Projects;
- repository and `Unassigned` views;
- batch assignment, correction, and audit history;
- Project Overview;
- project-scoped library and lexical search;
- tokens, coverage, API-equivalent value, and evidence drill-down;
- bookmarks and notes; and
- owner export of Project organization.

### Immediate follow-up after Wave 2.5 proves useful

- deterministic Project timeline;
- transcript-proven code links;
- deterministic cost anomalies;
- manual Workstreams bootstrapped from continuation threads; and
- deterministic context-pack selection with round-trip provenance.

### Existing later waves

- Wave 3 adds project-scoped semantic search and related-session evidence,
  but not Workstream clustering.
- Wave 4 adds project-scoped source context, personal-memory handoff, and
  vendor-import Project assignment.
- Wave 5 keeps the already accepted optional enrichment work behind the
  accepted local/external AI boundary.

### Acceptance-gated intelligence wave

Only after Projects and manual organization demonstrate actual use:

1. narrative Project and Workstream summaries;
2. decisions and open loops, generated only on explicit bounded request;
3. user-owned direction and roadmap proposals;
4. context-pack compression; and
5. Workstream clustering, only if manual Workstreams prove valuable.

Each layer must earn use before the next becomes implementation scope. Low
acceptance or continued non-use is a kill signal, not permission to generate
more suggestions.

## Explicitly Deferred

- automatic project merges;
- arbitrary fractional cost allocation across projects;
- team billing and chargeback;
- budgets, alerts, and enforcement;
- developer productivity scores;
- causal model/harness quality rankings;
- generated delivery dates or engineering estimates;
- live GitHub, GitLab, Jira, or other forge-state reads;
- automatic GitHub/Jira issue creation;
- automatic agent dispatch from roadmap items;
- bidirectional project-management synchronization;
- persistent suggestion inboxes or unread-curation counters;
- Workstream clustering before manual Workstreams demonstrate use;
- cross-machine Project merging;
- hosted/multi-user project analytics; and
- organization-wide surveillance or employee evaluation.

## Blocking Product Decisions

1. Which Project-membership signals may assign automatically, beyond automatic
   repository/worktree grouping?
2. Can one repository belong to several Projects, while each session retains
   one primary Project for cost?
3. What explicit source marker proves harness identity such as Ductum?
4. Does an accepted user-owned decision or roadmap item survive erasure of its
   only evidence with a visible `evidence erased` marker?
5. Which local AI boundary permits exact-history summaries, and what may an
   external provider receive?
6. What refresh and consent cadence applies to optional Project intelligence?
7. Which exact Project artifacts are included in the personal-memory handoff?

Resolved decisions:

- Claude Code per-message usage and Codex delta-from-cumulative usage are the
  accepted initial ledger semantics and default to `exact/source-reported`;
- a pinned LiteLLM-style price-catalog snapshot rides Qratum releases, with an
  optional explicit user-initiated online refresh or file import and no silent
  runtime fetch;
- suggestions are on-request or contextual, never standing queues;
- Workstreams build from confirmed continuation threads and remain optional;
- forge references are inert identifiers in this extension;
- deleting a session deletes its usage, recalculates totals, and discloses
  excluded tombstones;
- calculated subscription usage is API-equivalent value, not spend;
- exact Git identity remains owner-only and excluded from share output unless
  an egress plan lists it; and
- the first roadmap has only `Suggested`, `Accepted`, `Done`, and `Rejected`,
  with suggestions outside user-owned `Now`/`Next`/`Later` columns.

## Review Questions

1. Does Project become the correct primary organizing layer without weakening
   the session library?
2. Is Workstream the missing layer between Project and Session, or is it too
   much manual/AI curation?
3. Can repository/worktree identity remain understandable and correct across
   clones, forks, moved paths, and multiple machines?
4. Are token and cost totals auditable enough to trust?
5. Is the source/surface/harness/agent/model taxonomy useful or over-modeled?
6. Which code links are valuable enough to justify their ambiguity handling?
7. Do decisions and open loops remain evidence-backed rather than becoming an
   AI task queue?
8. Is a user-owned AI-proposed roadmap a natural extension or a step into
   project-management scope?
9. Does the LLM boundary keep the deterministic product useful on its own?
10. Is the proposed wave placement credible, or does it overload the daily
    product spine?

## Acceptance Boundary

The product owner accepted this extension on 2026-07-12, and its approved rules
are incorporated into `specs/current/product-direction.md`. That acceptance does
not accept existing implementation, select concrete model binaries or provider
versions, or make a wave implementation-ready by itself. Wave 1 was later
accepted in `specs/current/wave-1-reliable-session-capture.md`; every later wave
still needs a separately accepted technical contract.

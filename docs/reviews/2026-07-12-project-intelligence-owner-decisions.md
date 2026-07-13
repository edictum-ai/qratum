# Qratum Project Intelligence: Remaining Owner Decisions

Status: accepted; incorporated into canonical direction

Date: 2026-07-12

Accepted by product owner: 2026-07-12

This document contains only the choices still capable of changing the Project
Intelligence product shape. Resolved rules live in:

```txt
docs/reviews/2026-07-12-project-intelligence-user-stories.md
docs/reviews/2026-07-12-project-intelligence-fable-review.md
specs/current/product-direction.md
```

Nothing here authorizes implementation. These accepted choices are folded into
the canonical product direction; the Tranche 1 technical contract is the next
contract step.

## Already Resolved

- Repository grouping works without creating a Project.
- `Unassigned` is healthy, not an inbox.
- Suggestions are contextual or explicitly requested; there are no standing
  curation queues or unread counts.
- Tranche 1 captures facts that can disappear and fixture-locks source usage
  semantics.
- Tranche 2 keeps the accepted daily session spine and adds only repository
  awareness.
- The deterministic Project product is a separate Tranche 2.5.
- Workstreams are optional and begin from confirmed continuation threads.
- Forge references are inert until a separate opt-in integration is accepted.
- Session deletion removes usage records, recomputes aggregates, and discloses
  excluded tombstones.
- Subscription calculations are labeled `API-equivalent usage value`, not
  spend.
- The price catalog is a pinned LiteLLM-style snapshot delivered with releases
  or updated by explicit user import; there is no silent runtime fetch.
- Exact Git identity is owner-only and excluded from share output unless an
  egress plan lists it.
- The first roadmap uses `Suggested`, `Accepted`, `Done`, and `Rejected`, with
  suggestions outside the user's `Now`/`Next`/`Later` columns.
- Project organization is included in owner export.

## Decision 1: Automatic Project Assignment

### Question

Which evidence may place a repository or session into a Project without asking
again?

### Recommended answer

- Automatically unify working copies that share one Git common directory.
- Automatically recognize a previously confirmed repository identity after it
  moves or appears in another worktree.
- Once the user assigns a repository to a Project, later sessions with that
  confirmed repository identity inherit the Project assignment.
- A matching sanitized remote across independent clones creates a repository
  suggestion, not an automatic Project assignment.
- Forks, ambiguous remotes, monorepo subprojects, repositories without remotes,
  and non-Git folders require confirmation.
- Batch confirmation is the primary cold-start interaction.

### Why

This automates repeat observations without silently merging things the user has
never related.

## Decision 2: Repositories Shared Across Projects

### Question

May one logical repository appear in more than one Project?

### Recommended answer

Yes. Shared infrastructure and schema repositories legitimately support several
products. Repository membership may be many-to-many for discovery, but each
session has exactly one primary Project for cost aggregation. Additional
Project relationships are labeled secondary and never duplicate usage.

### Why

A strict one-repository-to-one-Project rule cannot represent shared repositories
without cloning identity or misattributing costs.

## Decision 3: Harness Attribution

### Question

What proves that a session ran under Ductum or another harness?

### Recommended answer

Only an explicit source or harness marker counts. The marker contains the
harness name, version, run identity, and observation time and is captured with
the session evidence. Repository path, process ancestry observed after the
fact, prompt text, or the presence of Ductum files does not prove attribution.

Until Ductum defines and emits this marker, `harness` remains `unknown` and the
application does not promote harness comparisons as a primary view.

### Why

This makes the dimension useful when proven and harmless when absent.

## Decision 4: User-Owned Statements After Evidence Erasure

### Question

When the user erases the only session supporting an accepted decision or
roadmap item, does the user-owned statement survive?

### Recommended answer

Yes, but only the user-confirmed statement survives. It displays `evidence
erased`, loses the inaccessible evidence link, and appears in the deletion
preview. Generated drafts, summaries, embeddings, usage, excerpts, and other
session-derived artifacts are deleted.

An external personal-memory item is not silently deleted by Qratum. The preview
lists the handoff identifier and explains that correction or deletion uses the
personal-memory mechanism.

### Why

The user owns the statement after accepting or editing it. Retaining generated
source material would violate terminal deletion; retaining the user's explicit
decision with an honest tombstone does not pretend the evidence still exists.

## Decision 5: AI Access To Exact History

### Question

Which models may read exact session history for Project summaries, decisions,
open loops, and context compression?

### Recommended answer

- Deterministic Project features never require a model.
- A proven-local model may read exact history only after the local boundary is
  fixture- and runtime-verified in its tranche contract.
- External providers receive no exact history by default.
- An external request requires a per-action egress preview naming provider,
  model, locality, input class, repository/Project metadata, estimated cost,
  and destination.
- Share-safe or explicitly selected content may be sent only after confirmation.
- The approved input object IDs and digests are recorded for provenance and
  deletion propagation.

### Why

This keeps the deterministic product useful while preserving an explicit path
to optional high-quality synthesis.

## Decision 6: Intelligence Refresh And Consent

### Question

When do generated Project summaries, decisions, open loops, and roadmap
candidates refresh?

### Recommended answer

- Generation is initiated explicitly for a selected Project, Workstream, time
  range, or session set.
- Cached output remains until its inputs change, then becomes visibly stale.
- Qratum shows the provider, model, input class, and estimated cost before an
  external or metered refresh.
- There is no corpus-wide automatic extraction and no standing background
  candidate generation.
- A later standing budget or schedule requires a separate accepted contract
  after on-request use proves value.

### Why

This prevents both surprise egress/cost and the suggestion-queue failure mode.

## Decision 7: Project Intelligence Handoff To Personal Memory

### Question

Which Project artifacts may become durable personal memory?

### Recommended answer

Only a user-approved, editable statement is handed off: normally an accepted
decision, durable preference, constraint, or lesson. Qratum does not send whole
Project summaries, open-loop queues, roadmap state, or raw evidence by default.

Every handoff shows the exact content and destination, records the returned
personal-memory identifier, and exposes the separate correction/deletion path.

### Why

Personal memory receives durable knowledge, not a duplicate Project database or
an unreviewed generated backlog.

## Acceptance Effect

Acceptance closed the Project Intelligence product decisions without accepting
implementation details. The resulting authority change:

1. folded the deterministic Project layer and tranche placement into
   `specs/current/product-direction.md`;
2. left the acceptance-gated intelligence tranche explicitly conditional; and
3. authorized a Tranche 1 contract covering source capture, ephemeral Git
   observations, usage fixtures, deletion propagation, and installed-artifact
   verification.

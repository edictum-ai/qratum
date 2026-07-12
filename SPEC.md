# Qratum SPEC

This is the source-of-truth index. It separates accepted product direction,
published history, local candidate code, and future technical contracts.

## Source Of Truth

Read these in order:

```txt
specs/current/product-direction.md
  Canonical accepted product authority, including the Project Intelligence
  extension accepted 2026-07-12. Governs product
  intent, user journeys, user-facing language, release tranches, supersession,
  and the minimum session-reference contract.

docs/reviews/2026-07-10-product-user-stories.md
docs/reviews/2026-07-12-project-intelligence-user-stories.md
docs/reviews/2026-07-12-project-intelligence-owner-decisions.md
  Complete product-owner-reviewed user stories and Project decisions,
  incorporated by reference by the canonical product-direction spec.

specs/current/qratum-vault-first.md
  Published v0.1.0 preservation-baseline design and historical evidence.

specs/current/verification-and-trust-gate.md
  Published v0.1.0 P2 trust-gate design and security findings. It does not
  prove later commits or future product claims without re-execution.

specs/current/ui-first-onboarding.md
  Superseded product contract. Retained as evidence for the local UI-first
  CLI/API candidate and loopback security design.

specs/current/operational-model-redesign.md
  Superseded product model. Retained as architecture background and donor
  design where the canonical product direction does not conflict.
```

Files under `specs/current/milestone-a/` and
`specs/current/memory-curation-pipeline.md` are historical only.

Schemas under `schemas/`, accepted decisions under `docs/decisions/`, and
fixtures/goldens remain executable contracts for the behavior they describe.
They do not override the accepted product direction.

## Current Reality

| Layer | State | What it means |
| --- | --- | --- |
| Published `v0.1.0` (`f7e21dc`) | published baseline | Vault-first runtime, old pipeline-shaped CLI, and P2 trust-gate implementation. |
| Remote `main` at the 2026-07-10 audit | no UI-first runtime | Published baseline plus UI-first documentation; not the candidate runtime. |
| Current worktree candidate | local, not shipped | Contains a UI-first CLI/API shell and prepare-from-preserved-blob work with correctness and trust gaps. |
| Product direction | accepted 2026-07-11; Project extension 2026-07-12 | Private searchable memory for Claude and Codex work, with exact local reading, hybrid search, continuity, context, repository/Project organization, and truthful usage accounting. |
| Tranche 0 | complete 2026-07-11 | Authority, supersession, truthful status, and the minimum session-reference contract. No runtime feature work. |
| Tranches 1-6, including 2.5 | blocked | Require separately accepted technical contracts before implementation. |

The old statement `UI-FIRST-ONBOARDING-RUNTIME — SHIPPED WITH NAMED
RESIDUALS` is retracted. Candidate code existing in a branch is not a published
or accepted product capability.

## Accepted Product Thesis

Qratum is not backup software.

> Qratum is the private, searchable memory of my AI work. It continuously
> gathers my Claude and Codex sessions and their associated context, keeps the
> exact history available locally, and helps me find and understand past work
> through a clean UI. When useful, it also gives me a direct path to continue
> that work in its source tool.

The complete accepted shape and user stories are in the canonical direction and
the three incorporated review documents listed above.

## Accepted Release Order

```txt
Tranche 0  product truth
Tranche 1  source correctness for Claude Code and Codex
Tranche 2  daily UI, exact reader, lexical search, repository awareness,
           continuation, deletion, exact export, honest cost, and health
Tranche 2.5 deterministic Projects, project-scoped search, usage/cost, and export
Tranche 3  semantic retrieval
Tranche 4  source context, personal-memory handoff, and vendor imports
Tranche 5  optional session enrichment and share-oriented export
Tranche 6  acceptance-gated Project intelligence
```

This ordering is a dependency graph, not a commitment to implement all
tranches in one release. A later tranche starts only after the prior tranche's
technical contract is accepted and its proof runs.

## Contract Readiness

The product direction and Tranche 0 are accepted and complete. Existing
candidate code does not make Tranches 1-6, including 2.5,
implementation-ready.

Before implementation, each tranche must resolve its blocking decisions and
define:

- source and storage contracts;
- UI DTOs and user-visible failure/recovery behavior;
- privacy, local-raw, egress, and deletion boundaries;
- fixture-driven acceptance tests;
- an installed-artifact user flow for the behavior; and
- the exact session reference fields it stores and displays.

The minimum session reference is normative:

```txt
specs/current/product-direction.md#minimum-session-reference
```

Missing session information reports `unknown` or `unsupported`, never a
fabricated value. A unit-test pass alone is not evidence that a user-facing
capability shipped.

## Superseded Product Rules

The canonical product direction supersedes older rules for:

- the found/preserved/prepared/viewable/open state machine;
- `prepare` and a raw queue as user workflows;
- the prohibition on owner-only exact local reading;
- raw indexes being off by default;
- the eight-command UI-first CLI contract;
- Claude-only support as the first usable source shape;
- review cards as the primary payoff;
- search being deferred behind lessons or insights; and
- the 2026-07-10 runtime rebootstrap roadmap.

## Standing Constraints

These remain accepted:

- Go single binary named `qrt`; no Python runtime.
- Raw transcripts are sensitive and owner-only.
- Raw content is not silently sent to external services.
- Raw content is not rendered into shareable reports.
- External processing is explicit and provenance-bound.
- Source hooks stay small, local, fast, and network-free.
- Untrusted source and import inputs fail closed.
- Product surfaces consume typed DTOs rather than parsing raw internals.
- Fixture/golden contracts and `docs/supply-chain.md` apply.
- Published and trusted claims require proof against the exact artifact.

> Redaction remains best-effort and credentials-oriented. It is not a PII,
> third-party-content, or share-safety guarantee. Local exact reading and
> external/share-oriented output are separate boundaries.

## Published Baseline Versus New Work

The `v0.1.0` tag remains historical release evidence. Preserve its fixtures,
schemas, and security findings unless an accepted migration intentionally
changes them.

The current worktree's UI-first shell is donor code. Do not describe it as
shipped, do not use its command surface as product authority, and do not assume
its DTOs or state model survive the new tranche contracts.

## Explicitly Deferred

These remain outside the accepted tranches until separately designed:

- verified backup/restore product flow;
- retention policies and automatic deletion;
- cross-machine merge;
- revision/checkpoint history as a user-facing feature;
- per-line permanent redaction while retaining the rest of a session;
- corpus generation and publishing;
- automatic memory writeback;
- enterprise control plane;
- MCP and marketplace behavior;
- skill mining; and
- claims that deterministic review proves correctness.

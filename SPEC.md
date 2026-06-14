# Qratum SPEC

## Source Of Truth

The canonical product and architecture spec is:

```txt
specs/current/operational-model-redesign.md
```

Proposed revisions of the operational model live alongside it:

```txt
specs/current/qratum-vault-first.md         (proposal: vault-first revision;
                                            supersedes the memory curation
                                            pipeline draft; see
                                            docs/reviews/2026-06-12-memory-architecture/)
specs/current/memory-curation-pipeline.md   (SUPERSEDED 2026-06-12; historical)
```

Milestone A is complete and historical. Its implementation, commands, fixtures,
and generated artifacts may remain as compatibility/debug behavior, but
Milestone A is no longer the product model.

Historical Milestone A notes live under:

```txt
specs/current/milestone-a/
```

## Current Milestone

Current milestone:

```txt
P0-SPEC-AND-CONTRACTS
```

P0 closes the redesign before runtime implementation.

## P0 Goal

Turn the operational model into executable contracts:

- schema registry under `schemas/`
- core object JSON Schemas
- config schema
- fixture examples for core objects
- schema validation tests
- migration notes from Milestone A to the operational model
- updated source-of-truth documentation

## P0 Non-Goals

Do not implement P1+ runtime behavior yet:

- workspace creation behavior
- setup wizard behavior
- raw archive implementation
- import wizard implementation
- session revision worker
- local app
- SQLite projection
- AI providers
- lesson/insight generation
- corpus export changes
- publisher behavior
- daemon behavior changes beyond compatibility fixes

## Standing Constraints

- Go single binary.
- No Python runtime in Qratum.
- Local-first raw transcript safety.
- Do not send raw transcripts to external services.
- Do not render raw transcripts into shareable reports.
- Source hooks must stay fast and only do durable capture work.
- Fixture/golden tests remain the contract where practical.
- Supply-chain rules in `docs/supply-chain.md` still apply.

## Compatibility

Milestone A commands can remain as hidden or debug compatibility aliases while
the new public model is designed and implemented.

Current compatibility behavior should keep working unless an accepted P0/P1+
contract intentionally replaces it.

# Qratum SPEC

## Source Of Truth

The current source-of-truth spec set is:

```txt
specs/current/operational-model-redesign.md  (base operational model; keep it
                                             aligned with the accepted
                                             vault-first edits)
specs/current/qratum-vault-first.md          (accepted 2026-06-14 vault-first
                                             revision and sequencing contract)
specs/current/memory-curation-pipeline.md    (SUPERSEDED 2026-06-12;
                                             historical only)
```

The accepted review spine and dispatch context live under:

```txt
docs/reviews/2026-06-12-memory-architecture/
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

P0 closes the redesign before runtime implementation. The next runtime unlock,
if Arnold approves it, is the vault-first P1 sequence described in
`specs/current/qratum-vault-first.md`.

## P0 Goal

Turn the accepted operational direction into executable contracts:

- schema registry under `schemas/`
- core object JSON Schemas
- config schema
- fixture examples for core objects
- schema validation tests
- migration notes from Milestone A to the operational model
- updated source-of-truth documentation
- ADRs that record accepted product decisions

## P0 Non-Goals

Do not implement runtime behavior yet. P0 is still docs/contracts only:

- workspace creation behavior
- setup wizard behavior
- import wizard implementation
- session revision worker
- local app
- AI providers
- lesson/insight generation
- corpus export changes
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

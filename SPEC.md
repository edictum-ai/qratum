# Qratum SPEC

This is a short index. It points to the real specs and records what has shipped, what is proposed, and what is deliberately not built. The detailed design lives in the files this page links to.

## Plain-language glossary

- **vault**: the central workspace where captured transcripts and processed artifacts live (on disk at `~/.qratum/`).
- **blob store**: a file store that names each file by its content, so identical content is stored once (content-addressed).
- **idempotent**: an operation you can run repeatedly and it makes no further change after the first run.
- **redaction**: automatically removing secrets (tokens, keys, etc.) from text before it is shown or shared.
- **golden / fixture test**: a test that compares output against a saved, known-good file; the saved file is the contract.

## Source Of Truth

Start here. These are the specs that currently define Qratum, so read them first.

```txt
specs/current/operational-model-redesign.md  (base operational model; keep it
                                             aligned with the accepted
                                             vault-first edits)
specs/current/qratum-vault-first.md          (accepted 2026-06-14 vault-first
                                             revision and sequencing contract)
specs/current/memory-curation-pipeline.md    (SUPERSEDED 2026-06-12;
                                             historical only)
```

One more spec exists, but Arnold has not signed it off yet. It changes nothing on this page until he accepts it.

Proposed (not yet accepted):

```txt
specs/current/verification-and-trust-gate.md (proposed 2026-06-15;
                                             P2-VERIFY-TRUST-GATE; awaiting
                                             Arnold's acceptance — does not
                                             change this milestone until then)
```

The accepted review spine and dispatch context live under:

```txt
docs/reviews/2026-06-12-memory-architecture/
```

Milestone A is an earlier product model. It is finished and kept only for history. Its code, commands, fixtures, and generated artifacts may stay around as compatibility/debug behavior, but Milestone A is no longer the product model.

Historical Milestone A notes live under:

```txt
specs/current/milestone-a/
```

## Current Milestone

Where things stand right now, in one line: the vault-first phase (P1-VAULT-FIRST) has shipped, and the verify/trust-gate phase (P2-VERIFY-TRUST-GATE) is only proposed.

```txt
P1-VAULT-FIRST — SHIPPED (merged, test-backed)
P2-VERIFY-TRUST-GATE — PROPOSED (specs/current/verification-and-trust-gate.md;
                       awaiting Arnold's acceptance)
```

The vault-first phase (P1-VAULT-FIRST) has shipped. It is merged, it runs from the `qrt` binary, and it is test-backed. The binary self-reports `milestone: vault-first` (Details: `specs/current/qratum-vault-first.md`).

The earlier spec-and-contracts phase (P0-SPEC-AND-CONTRACTS — schemas, ADRs, source-of-truth cleanup) is substantially complete. Its remaining gaps are tracked under "Known P0 gaps" below.

The next phase to unlock is the verify/trust-gate phase (P2-VERIFY-TRUST-GATE), but only if Arnold approves it. That phase is the verification benchmark plus the confirmed-defect fixes (Details: `specs/current/verification-and-trust-gate.md`).

## Shipped runtime (vault-first P1)

This is what already works today. These features are built, registered in the `qrt` binary, and test-backed. They are no longer "do not implement" items:

- the central workspace, the vault, at `~/.qratum/` (override with `QRATUM_HOME`)
- a hook that copies each transcript as it is captured (`qrt hook claude-code`), plus a content-addressed blob store (a file store that names each file by its content, so identical content is stored once)
- hook setup and status (`qrt hook install` / `qrt hook status`) — these edit global settings and can be run repeatedly without making further changes (idempotent)
- vault maintenance: `qrt vault backfill` / `archive` / `backup [--verify]` / `doctor`
- a status view (`qrt status`) showing vault counts, last backfill, and copy failures

The Milestone A refinery still ships too, but only as tooling you invoke by hand: `normalize`/`redact`/`evidence`/`review`/`report`/`export`, plus `daemon run-once` and `dogfood`.

## Known P0 gaps (still contracts/docs work)

Two pieces promised in the spec-and-contracts phase (P0-SPEC-AND-CONTRACTS) are not finished yet:

- the config schema (`config.schema.json`) is advertised but **not yet delivered**
- the JSON schemas (`schemas/*.json`) have two problems: they do not set `additionalProperties`, and no Go test checks them. For now, golden/fixture tests (which compare output against a saved, known-good file) enforce the contract instead.

## Still not built (genuine non-goals)

These are left out on purpose. Do not build them as part of the current model:

- setup wizard behavior
- import wizard implementation
- session revision worker
- local app
- AI providers
- lesson/insight generation
- corpus export changes
- a standing/resident daemon or review queue — refine is a one-shot you invoke; whether to add a daemon is still an open decision (see verification-and-trust-gate.md §5)

## Standing Constraints

These rules always hold, regardless of milestone:

- Go single binary.
- No Python runtime in Qratum.
- Local-first raw transcript safety.
- Do not send raw transcripts to external services.
- Do not render raw transcripts into shareable reports.
- Source hooks must stay fast and only do durable capture work.
- Fixture/golden tests remain the contract where practical.
- Supply-chain rules in `docs/supply-chain.md` still apply.

> **Honest boundary — deterministic redaction is best-effort alpha.** Redaction here means automatically stripping secrets, and it is not a guarantee. The shipped redactor only matches a fixed, enumerated set of secret classes, so it has known leak gaps. The known gaps are: a `=>` assignment edge case; the fields `git.branch` / `git.head_sha` / `started_at` / `ended_at` / `source_event_id` are not redacted; and SSH-style git remotes are not caught. A committed golden file currently ships some of these unredacted. Closing these gaps is the proposed verify/trust-gate work (P2-VERIFY-TRUST-GATE). Do not describe redaction as a guarantee.

## Compatibility

Old commands keep working for now. Milestone A commands can stay available as hidden or debug compatibility aliases while the new public model is designed and implemented.

Current compatibility behavior should keep working unless an accepted P0/P1+ contract intentionally replaces it.

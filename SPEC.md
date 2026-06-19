# Qratum SPEC

This is a short index. It points to the real specs and records what has shipped, what is proposed, and what is deliberately not built. The detailed design lives in the files this page links to.

## Plain-language glossary

- **vault**: the central workspace where captured transcripts and processed artifacts live (on disk at `~/.qratum/`).
- **blob store**: a file store that names each file by its content, so identical content is stored once (content-addressed).
- **idempotent**: an operation you can run repeatedly and it makes no further change after the first run.
- **redaction**: automatically removing secrets (tokens, keys, etc.) from text before it is shown or shared.
- **golden / fixture test**: a test that compares output against a saved, known-good file; the saved file is the contract.

## Source Of Truth

Start here. These are the specs that currently define Qratum, so read them
first. For onboarding and the public `qrt` command contract,
`specs/current/ui-first-onboarding.md` wins wherever it conflicts with older
docs.

```txt
specs/current/ui-first-onboarding.md         (accepted product direction;
                                             authoritative for onboarding and
                                             the public command contract;
                                             runtime implementation not shipped)
specs/current/operational-model-redesign.md  (base architecture background;
                                             superseded for onboarding and the
                                             public command contract wherever it
                                             conflicts with ui-first)
specs/current/qratum-vault-first.md          (accepted 2026-06-14 vault-first
                                             revision and sequencing contract)
specs/current/verification-and-trust-gate.md (accepted 2026-06-16;
                                             P2-VERIFY-TRUST-GATE)
specs/current/memory-curation-pipeline.md    (SUPERSEDED 2026-06-12;
                                             historical only)
```

The accepted review spine and dispatch context live under:

```txt
docs/reviews/2026-06-12-memory-architecture/
```

Milestone A is an earlier product model. It is finished and kept only for
history. Its fixtures and generated artifacts may stay as test/reference
material, but its public command surface is not the product model and should be
removed as UI-first onboarding replacements land.

Historical Milestone A notes live under:

```txt
specs/current/milestone-a/
```

## Current Milestone

Where things stand right now, in one line: the vault-first phase (P1-VAULT-FIRST) has shipped, and the verify/trust-gate phase (P2-VERIFY-TRUST-GATE) is now accepted and unlocked for implementation.

```txt
P1-VAULT-FIRST — SHIPPED (merged, test-backed)
P2-VERIFY-TRUST-GATE — ACCEPTED / UNLOCKED 2026-06-16
                       (specs/current/verification-and-trust-gate.md;
                       dispatchable via the ductum package)
```

The vault-first phase (P1-VAULT-FIRST) has shipped. It is merged, it runs from the `qrt` binary, and it is test-backed. The binary self-reports `milestone: vault-first` (Details: `specs/current/qratum-vault-first.md`).

The earlier spec-and-contracts phase (P0-SPEC-AND-CONTRACTS — schemas, ADRs, source-of-truth cleanup) is substantially complete. Its remaining gaps are tracked under "Known P0 gaps" below.

The verify/trust-gate phase (P2-VERIFY-TRUST-GATE) is now accepted and unlocked: the verification benchmark plus the confirmed-defect fixes (Details: `specs/current/verification-and-trust-gate.md`). It is delivered as the ductum spec package under `docs/reviews/2026-06-15-verification-benchmark/ductum-specs/qratum-verify-trust-gate/`.

## Shipped runtime (vault-first P1)

This is what already works today. These features are built, registered in the `qrt` binary, and test-backed. They are no longer "do not implement" items:

- the central workspace, the vault, at `~/.qratum/` (override with `QRATUM_HOME`)
- a hook that copies each transcript as it is captured (`qrt hook claude-code`), plus a content-addressed blob store (a file store that names each file by its content, so identical content is stored once)
- hook setup and status (`qrt hook install` / `qrt hook status`) — these edit global settings and can be run repeatedly without making further changes (idempotent)
- vault maintenance: `qrt vault backfill` / `archive` / `backup [--verify]` / `doctor`
- a status view (`qrt status`) showing vault counts, last backfill, and copy failures

The Milestone A refinery still ships too, but only as tooling you invoke by hand:
`normalize`/`redact`/`evidence`/`review`/`report`/`export`, plus `daemon
run-once` and `dogfood`. These are shipped reality, not the future public
surface; the UI-first contract removes them from `qrt` as replacements land.

## Known P0 gaps (still contracts/docs work)

Two pieces promised in the spec-and-contracts phase (P0-SPEC-AND-CONTRACTS) are not finished yet:

- the config schema (`config.schema.json`) is advertised but **not yet delivered**
- the JSON schemas (`schemas/*.json`) have two problems: they do not set `additionalProperties`, and no Go test checks them. For now, golden/fixture tests (which compare output against a saved, known-good file) enforce the contract instead.

## Accepted Next Direction

The onboarding direction is now UI-first:

- `qrt init` explains what Qratum found, then preserves existing local sessions
  and prepares the latest 10 for viewing after confirmation.
- `qrt open` opens the local Qratum app at `127.0.0.1:9473`.
- `qrt status`, `qrt doctor`, `qrt import <file-or-folder>`, `qrt sessions`,
  `qrt session <session_id>`, and `qrt export` are public operator commands.
- `qrt export` is explicit egress: it must show scope, destination, data class,
  and confirmation before data leaves Qratum.
- The old pipeline-shaped public commands are removed as the onboarding surface
  lands; do not keep hidden compatibility aliases by default.

Details: `specs/current/ui-first-onboarding.md`.

## Still not built (genuine non-goals)

These are not shipped yet. Do not build unrelated future behavior as part of
the onboarding work:

- import wizard implementation beyond the explicit onboarding contract
- session revision worker beyond the prepare-from-preserved-raw bridge
- SQLite projection behavior
- AI providers
- lesson/insight generation
- corpus export changes
- publisher behavior
- a standing/resident daemon or review queue; background preservation should use
  a source hook or OS schedule first unless a later contract accepts a daemon
- new source adapters beyond accepted schema fixtures

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

> **Honest boundary — deterministic redaction is best-effort alpha.** Redaction here means automatically stripping secrets, and it is not a guarantee. The shipped redactor only matches a fixed, enumerated set of secret classes, so it has known leak gaps. The known gaps are: a `=>` assignment edge case; the fields `git.branch` / `git.head_sha` / `started_at` / `ended_at` / `source_event_id` are not redacted; and SSH-style git remotes are not caught. Do not describe redaction as a guarantee.

## Compatibility

Old commands keep working only until their replacement lands. The UI-first
onboarding contract intentionally replaces the old pipeline-shaped public
surface. Remove those old public command paths; do not keep hidden aliases or
debug entrypoints just to preserve Milestone A behavior.

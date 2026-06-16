# Qratum agent instructions

This file tells an agent how to work in Qratum: where the rules live, what is
already built, what must not be built yet, and when a task counts as done.

## Plain-language glossary

A few terms recur below. Plain meaning first; the term is kept for accuracy.

- **Redaction**: removing secrets/sensitive strings from text before it leaves
  the system.
- **ADP**: the internal session data format (ADP JSONL) the runtime produces.
- **DTO**: a clean, UI-facing data shape, as opposed to a raw internal model.
- **Blob store**: a content-addressed file store (each file is keyed by a hash
  of its contents).
- **Hook**: a small program Claude Code calls on each event; it must stay fast
  and do almost nothing.
- **Fixture / golden test**: a saved input and its expected output, used to lock
  behavior so changes are caught.

## Source Of Truth

When you are unsure what's authoritative, start here. `SPEC.md` wins.

Follow `SPEC.md` first. It points to the canonical operational model:

```txt
specs/current/operational-model-redesign.md
```

The rest of the tree carries different weight:

- Files under `specs/current/milestone-a/` are historical Milestone A notes.
  They explain the completed vertical slice and compatibility behavior, but they
  do not define the current product model.
- Files under `docs/architecture/` are forward-design references only. They are
  not permission to implement future features.
- Files under `docs/decisions/` are accepted architectural decisions.
- Schemas under `schemas/` are contracts.
- Fixtures under `fixtures/` are part of the test contract.

## Current Milestone

This is where the project stands and what you may act on.

```txt
P1-VAULT-FIRST — SHIPPED (merged, test-backed)
P2-VERIFY-TRUST-GATE — ACCEPTED / UNLOCKED 2026-06-16
```

The vault-first runtime (P1) has shipped and is test-backed (the `qrt` binary
self-reports `milestone: vault-first`). The P0 contract work (schemas, ADRs,
source-of-truth cleanup) is substantially complete. P2-VERIFY-TRUST-GATE is now
accepted and unlocked — implement it per the ductum spec package under
`docs/reviews/2026-06-15-verification-benchmark/ductum-specs/qratum-verify-trust-gate/`
(the contract is `specs/current/verification-and-trust-gate.md`).

Do not implement P3-or-later runtime behavior unless the user explicitly unlocks
a later milestone.

## Shipped runtime (no longer "do not implement")

This is the live code. It is merged, registered in `qrt`, and test-backed, so
you may change it (carefully). It includes:

- central workspace at `~/.qratum/` (`QRATUM_HOME` override)
- copy-on-capture hook + content-addressed blob store
- `qrt hook install` / `hook status`
- `qrt vault backfill` / `archive` / `backup [--verify]` / `doctor`
- `qrt status`
- the Milestone A refinery, as on-demand tooling

When changing this shipped runtime, treat it as live code: fixture/golden tests
remain the contract; do not regress the demo.

## Still not built (genuine non-goals)

These do not exist yet, on purpose. Do not build any of them unless an accepted
milestone unlocks it:

- setup wizard behavior
- import wizard implementation
- session revision worker
- local app
- SQLite projection behavior
- AI providers
- lesson or insight generation
- corpus export changes
- publisher behavior
- a standing/resident daemon or review queue (refine is a one-shot; daemon vs
  no-daemon is an open decision per verification-and-trust-gate.md §5)
- new source adapters beyond accepted schema fixtures

## Runtime

Qratum is a single Go binary. There is no Python at runtime.

Use Go. No Python runtime in Qratum. The long-term runtime is still a Go single
binary named `qrt`.

## Compatibility Rules

Old Milestone A commands can stick around for compatibility/debug, but the hook
must stay tiny and fast.

Milestone A commands and artifacts may remain as compatibility/debug behavior
while the new operational model is implemented.

If you touch existing Milestone A runtime paths:

- `qrt hook claude-code` must stay fast.
- The hook only reads JSON from stdin, writes a capture event, and exits.
- No LLM calls from hooks.
- No full transcript parsing from hooks.
- No report generation from hooks.
- No network calls from hooks.

## Data Rule

Raw transcripts are sensitive. Get the path from the payload, and never let raw
transcripts escape.

Use `transcript_path` from source payloads where available. Do not hardcode
Claude local transcript paths as the primary capture mechanism.

Do not send raw transcripts to external services.

Do not render raw transcripts into shareable reports.

Redaction does not yet fully work — treat it as best-effort, not a guarantee.
Deterministic redaction is **best-effort alpha**: it matches an enumerated set
of secret classes and has known leak gaps. The known gaps are: a `=>` assignment
edge case; `git.branch`/`git.head_sha`/`started_at`/`ended_at`/`source_event_id`
are not redacted; and SSH-style git remotes are not caught. Do not treat
redaction as a guarantee, and do not weaken the "no raw in shareable reports"
rule on the assumption it already holds. Closing these gaps is proposed
P2-VERIFY-TRUST-GATE work.

## UI Contract Rule

The UI gets clean data shapes (DTOs). It must never have to parse raw internals.

Product surfaces consume DTOs, not raw internal models.

Backend code must not require UI code to parse:

- Claude transcript JSONL
- raw session internals
- ADP JSONL
- redaction internals
- provenance internals

## Testing

Behavior is locked by saved input/expected-output pairs (fixtures and golden
files). Update them only when an output contract intentionally changes.

Every behavior must be fixture-driven where practical.

Update fixtures and golden files when output contracts intentionally change.

For code changes, `make test` must run all tests. `make demo` should keep the
existing vertical slice working unless the accepted milestone intentionally
replaces that behavior. `make verify` mirrors the CI pipeline and includes
supply-chain checks.

For documentation-only changes, tests are not required.

## Supply-Chain Rule

Pin everything; pull nothing from the network at build/run time. Details live in
`docs/supply-chain.md`.

Follow `docs/supply-chain.md`.

- Pin GitHub Actions by commit SHA.
- Do not add pipe-to-shell installers.
- Do not use floating tool versions in CI or scripts.
- Do not add npm, npx, pip, curl installers, or shell-fetched binaries to the
  Qratum runtime pipeline.
- Keep Go modules readonly in verification commands.

## Ductum Factory Rules

When a task is dispatched to you via the Ductum factory:

- You are running inside an isolated git worktree. Make your changes on the
  current feature branch.
- Do not run `git push`. The factory's post-completion pipeline handles verify,
  review, and merge after you call `ductum_complete`.
- After you call `ductum_complete(result=...)`, stop making tool calls. Your
  session will end and the factory will take over.
- The workflow has three stages: `understand` (read context), `implement`
  (write code), `ship` (factory-owned). Work only in `implement` for code
  changes.
- Required verify command is in `.edictum/workflow-profile.yaml`. It will be
  run automatically; if it fails, a fix-loop task will be dispatched with the
  failure output.

## Definition Of Done

A task is done only when:

1. The requested source-of-truth or contract change is complete.
2. Code builds if code was changed.
3. Tests pass if code or executable contracts were changed.
4. Fixture/golden outputs are updated when output contracts intentionally
   change.
5. Existing demo behavior still works if affected.
6. No non-goal feature was implemented.

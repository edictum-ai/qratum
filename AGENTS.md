# Qratum agent instructions

## Source Of Truth

Follow `SPEC.md` first.

`SPEC.md` points to the canonical operational model:

```txt
specs/current/operational-model-redesign.md
```

Files under `specs/current/milestone-a/` are historical Milestone A notes. They
explain the completed vertical slice and compatibility behavior, but they do
not define the current product model.

Files under `docs/architecture/` are forward-design references only. They are
not permission to implement future features.

Files under `docs/decisions/` are accepted architectural decisions. Schemas
under `schemas/` are contracts. Fixtures under `fixtures/` are part of the test
contract.

## Current Milestone

Current milestone:

```txt
P0-SPEC-AND-CONTRACTS
```

P0 work is contract and source-of-truth work only:

- schema registry
- JSON Schemas for core objects
- config schema
- fixture examples
- schema validation tests
- migration notes from Milestone A
- documentation/source-of-truth cleanup

Do not implement P1+ runtime behavior unless the user explicitly changes the
milestone.

## P0 Non-Goals

Do not implement:

- setup wizard behavior
- central workspace creation behavior
- raw archive implementation
- import wizard implementation
- session revision worker
- local app
- SQLite projection behavior
- AI providers
- lesson or insight generation
- corpus export changes
- publisher behavior
- daemon behavior changes beyond compatibility fixes
- new source adapters beyond accepted schema fixtures

## Runtime

Use Go. No Python runtime in Qratum.

The long-term runtime is still a Go single binary named `qrt`.

## Compatibility Rules

Milestone A commands and artifacts may remain as compatibility/debug behavior
while the new operational model is implemented.

If touching existing Milestone A runtime paths:

- `qrt hook claude-code` must stay fast.
- The hook only reads JSON from stdin, writes a capture event, and exits.
- No LLM calls from hooks.
- No full transcript parsing from hooks.
- No report generation from hooks.
- No network calls from hooks.

## Data Rule

Use `transcript_path` from source payloads where available. Do not hardcode
Claude local transcript paths as the primary capture mechanism.

Do not send raw transcripts to external services.

Do not render raw transcripts into shareable reports.

## UI Contract Rule

Product surfaces consume DTOs, not raw internal models.

Backend code must not require UI code to parse:

- Claude transcript JSONL
- raw session internals
- ADP JSONL
- redaction internals
- provenance internals

## Testing

Every behavior must be fixture-driven where practical.

Update fixtures and golden files when output contracts intentionally change.

For code changes, `make test` must run all tests. `make demo` should keep the
existing vertical slice working unless the accepted milestone intentionally
replaces that behavior. `make verify` mirrors the CI pipeline and includes
supply-chain checks.

For documentation-only changes, tests are not required.

## Supply-Chain Rule

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

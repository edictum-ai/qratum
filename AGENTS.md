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
- **Hook**: a small program a supported source tool calls on an event; it must
  stay fast and do almost nothing.
- **Fixture / golden test**: a saved input and its expected output, used to lock
  behavior so changes are caught.

## Plain-language rule

Use simple English in new Qratum documentation, help text, status, errors, and
UI copy. Do not make the user decode implementation language.

For every feature or behavior, explain these three things in this order:

1. **What** it does or what happened.
2. **Why** the user needs it or what it enables.
3. **How** it works or what the user should do next.

Define a technical term in plain English before using it. Internal schema and
code names may stay precise, but the text around them must still follow this
rule.

## Source Of Truth

When you are unsure what's authoritative, start here. `SPEC.md` wins.

The accepted product direction is authoritative for product intent, user
journeys, public-surface direction, release order, and the minimum session
reference:

```txt
specs/current/product-direction.md
```

The accepted Wave 1 implementation contract is:

```txt
specs/current/wave-1-reliable-session-capture.md
```

It authorizes only reliable Claude Code and Codex capture. It does not prove
that the behavior exists or permit later-wave work.

Its complete reviewed user stories live at:

```txt
docs/reviews/2026-07-10-product-user-stories.md
docs/reviews/2026-07-12-project-intelligence-user-stories.md
docs/reviews/2026-07-12-project-intelligence-owner-decisions.md
```

The product-direction spec incorporates those reviewed documents by reference.
They are not implementation contracts for Waves 1 through 6, including
Wave 2.5. Each wave needs a separately accepted technical contract before
code begins.

The following older files are implementation evidence and historical design,
not product authority where they conflict with the accepted direction:

```txt
specs/current/ui-first-onboarding.md
specs/current/operational-model-redesign.md
specs/current/qratum-vault-first.md
specs/current/verification-and-trust-gate.md
docs/reviews/2026-07-10-runtime-rebootstrap.md
```

Read their supersession notices before reusing any decision. Preserve useful
engineering foundations, but do not carry forward old state, CLI, raw-viewing,
search, source, memory, or release-order rules by inertia.

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
v0.1.0 — PUBLISHED BASELINE (tag f7e21dc; vault-first runtime,
         old pipeline-shaped CLI, and P2 trust-gate implementation)
UI-FIRST RUNTIME — LOCAL CANDIDATE, NOT SHIPPED
PRODUCT DIRECTION — ACCEPTED 2026-07-11; PROJECT EXTENSION ACCEPTED 2026-07-12
WAVE 0 PRODUCT TRUTH — COMPLETE 2026-07-11 (docs/contract only)
WAVE 1 RELIABLE SESSION CAPTURE — CONTRACT ACCEPTED 2026-07-12;
                                  IMPLEMENTATION NOT STARTED
WAVES 2-6, INCLUDING 2.5 — BLOCKED ON ACCEPTED TECHNICAL CONTRACTS
```

The published `v0.1.0` tag is the evidence for the vault-first/P2 baseline. Do
not project that tag's proof onto later commits or the current candidate without
re-running the applicable proof against the exact artifact.

The current branch contains a UI-first CLI/API candidate, including `qrt init`
and `qrt open`, but it has not shipped and its previous product contract is
superseded. It is donor material for later waves, not proof that the accepted
product exists.

Wave 0 is complete and changed documentation/contracts only. Wave 1 has an
accepted technical contract and may now be implemented exactly to that
contract. No Wave 1 behavior is available yet. Do not implement Wave 2 or later
behavior until its own technical contract is accepted.

## Published Baseline And Local Candidate

Published `v0.1.0` includes the vault-first preservation baseline, Claude Code
capture, content-addressed raw storage, the old pipeline-shaped CLI, and the P2
trust-gate implementation. Treat the tag as historical release evidence.

The current worktree candidate additionally contains:

- a central `QRATUM_HOME` path and prepare-from-preserved-blob bridge;
- UI-first CLI candidates such as `qrt init`, `qrt open`, `qrt status`,
  `qrt doctor`, `qrt import`, `qrt sessions`, `qrt session`, and `qrt export`;
- a loopback DTO/API shell; and
- retained vault/refinery code and tests.

None of those candidate surfaces is accepted as shipped product behavior. Keep
fixture and golden evidence intact unless an accepted contract intentionally
changes it.

## Wave Boundaries

Wave 1 reliable session capture is governed by
`specs/current/wave-1-reliable-session-capture.md`. Implement only that accepted
scope and do not describe it as working until its installed proof passes.

Do not implement these later waves until their technical contracts are
accepted:

- Wave 2 UI library, exact reader, lexical search, repository awareness,
  continuation, deletion, exact export, cost, and health;
- Wave 2.5 deterministic Projects, project-scoped search, usage/cost, and
  organization export;
- Wave 3 semantic retrieval;
- Wave 4 source context, personal-memory handoff, and vendor imports;
- Wave 5 optional session enrichment and share-oriented export; and
- Wave 6 acceptance-gated Project intelligence.

The blocking decisions and minimum session-reference contract live in
`specs/current/product-direction.md`. Existing partial code is not permission
to skip them.

## Runtime

Qratum is a single Go binary. There is no Python at runtime.

Use Go. No Python runtime in Qratum. The long-term runtime is still a Go single
binary named `qrt`.

## Compatibility Rules

The accepted direction keeps the normal user workflow in the UI and aims for a
minimal public CLI. The exact Wave 2 command surface is not accepted yet.
Do not preserve, add, or delete public commands solely because an older spec or
candidate already contains them.

Underlying package code may remain only when the shipped use cases or tests call
it.

If you touch existing capture hook paths while they still exist:

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

The accepted product direction allows the owner to read exact local history in
the local app. That local owner-only reader is not an export or a shareable
report. Its masking, reveal, containment, and deletion behavior remains blocked
on the Wave 2 technical contract and proof.

Redaction is **best-effort alpha**: it is credentials-oriented and is not a PII
or third-party-content guarantee. The published v0.1.0 trust result is historical
evidence only; the current candidate and every future product claim begin
unverified under `specs/current/product-direction.md`. Do not weaken the "no raw
in shareable reports" rule on the assumption redaction is complete.

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

Pin all code, tools, build inputs, and bundled runtime data. Build and tests
pull nothing from the network. Normal runtime behavior stays local. The only
accepted Wave 1 runtime-data exception is an explicit user-requested pricing
catalog refresh from the allowlisted source in its accepted contract; it sends
no session or user data and never runs silently. Details live in
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
7. Any affected user-facing behavior is checked through its accepted fixture or
   installed flow, and missing reference data reports `unknown` rather than a
   fabricated value.

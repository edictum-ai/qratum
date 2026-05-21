# P9 — Demo Hardening

## Scope

Make `make demo` the honest Milestone A vertical-slice proof.

## Decision Trace

- SPEC.md acceptance.
- ADR 0002: daemon and hook model.
- ADR 0004: filesystem JSON for Milestone A.

## Behavior Contract

- CLI runtime must fail visibly when required input is missing or invalid.
- Output schema evidence must preserve session IDs, artifact paths, and deterministic fixture timestamps.
- Missing artifacts must reject the run or demo instead of being silently swallowed.
- Verification output must be operator-visible when behavior fails.
- Invalid config or input must refuse processing with an error.
- Runtime resolution logic must remain scoped to the current project and session.
- Evidence paths must round-trip through generated artifacts.
- Session state must preserve source IDs instead of silently inventing replacements.
- Runtime behavior must be deterministic under fixture inputs.
- Missing or invalid files must fail loudly with an operator-visible message.
- Output must preserve explicit evidence for every generated review or report.
- Schema output must reject unsupported values rather than silently accepting drift.

- `make demo` runs the full Milestone A local loop.
- It exits non-zero when any expected artifact is missing.
- It prints generated artifact paths for operator inspection.

## Deliverables

- `make demo` cleans or isolates `.qratum/`.
- Runs hook -> daemon -> sessions list.
- Verifies every expected artifact exists.
- Prints generated artifact paths.
- Exits non-zero if any artifact is missing.

## Non-goals

No installer, long-running daemon, real Claude Code plugin setup, GitHub
comment, server sync, or marketplace pack.

## Verification

```sh
make test
make build
make demo
```

## Drift Handling

If the demo needs external credentials, network, or a real Claude install, stop;
Milestone A demo must remain fixture-driven.

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack demos that pass without producing every expected artifact type.
- Test from a clean `.qratum/` directory.

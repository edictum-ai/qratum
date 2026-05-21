# P9 — Demo Hardening

## Scope

Make `make demo` the honest Milestone A vertical-slice proof.

## Decision Trace

- SPEC.md acceptance.
- ADR 0002: daemon and hook model.
- ADR 0004: filesystem JSON for Milestone A.

## Behavior Contract

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

- Attack demos that pass without producing every expected artifact type.
- Test from a clean `.qratum/` directory.

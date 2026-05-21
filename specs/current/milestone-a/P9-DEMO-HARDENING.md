# P9 — Demo Hardening

## Scope

Make `make demo` the honest Milestone A vertical-slice proof.

## Deliverables

- `make demo` cleans or isolates `.qratum/`.
- Runs hook -> daemon -> sessions list.
- Verifies every expected artifact exists.
- Prints generated artifact paths.
- Exits non-zero if any artifact is missing.

## Non-goals

No installer, long-running daemon, real Claude Code plugin setup, GitHub
comment, server sync, or marketplace pack.

## Acceptance

```sh
make test
make build
make demo
```

# P0 — Repo Skeleton Cleanup

## Scope

Create the minimal `qrt` Go binary and test scaffolding needed for later
stages. Keep this mechanical.

## Decision Trace

- ADR 0001: Go single binary.
- SPEC.md: Milestone A only.

## Behavior Contract

- The repo compiles without implementing future stages.
- `qrt --version` and `qrt status` work without `.qratum/`.
- No non-goal packages, services, databases, or network calls are added.

## Deliverables

- `cmd/qrt/main.go` compiles.
- `qrt --version` prints a fixed development version.
- `qrt status` prints local Milestone A status without requiring `.qratum/`.
- `make build` compiles `bin/qrt`.
- `go test ./...` passes.

## Non-goals

No hook parsing, daemon, redaction, evidence, reports, ADP export, server, web
UI, database, or LLM calls.

## Verification

```sh
go test ./...
make build
./bin/qrt --version
./bin/qrt status
```

## Drift Handling

If build setup requires a dependency beyond the Go standard library, document
why in a new ADR before adding it.

## Slop Review

- Attack any implementation that sneaks in hook, daemon, parser, server, DB, or
  web UI behavior.
- Test the commands above from a clean checkout.

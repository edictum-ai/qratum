# P0 — Repo Skeleton Cleanup

## Scope

Create the minimal `qrt` Go binary and test scaffolding needed for later
stages. Keep this mechanical.

## Deliverables

- `cmd/qrt/main.go` compiles.
- `qrt --version` prints a fixed development version.
- `qrt status` prints local Milestone A status without requiring `.qratum/`.
- `make build` compiles `bin/qrt`.
- `go test ./...` passes.

## Non-goals

No hook parsing, daemon, redaction, evidence, reports, ADP export, server, web
UI, database, or LLM calls.

## Acceptance

```sh
go test ./...
make build
./bin/qrt --version
./bin/qrt status
```

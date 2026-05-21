# P0 — Repo Skeleton Cleanup

## Scope

Create the minimal `qrt` Go binary and test scaffolding needed for later
stages. Keep this mechanical.

## Decision Trace

- ADR 0001: Go single binary.
- SPEC.md: Milestone A only.

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

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack any implementation that sneaks in hook, daemon, parser, server, DB, or
  web UI behavior.
- Test the commands above from a clean checkout.

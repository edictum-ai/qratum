# P2 — Daemon Run Once

## Scope

Implement `qrt daemon run-once` as the asynchronous pipeline shell.

## Decision Trace

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

- The daemon reads events from `.qratum/events/`.
- Relative transcript paths are resolved from the current repo.
- Re-running `run-once` does not duplicate completed artifacts for the same
  event.
- Heavy pipeline work belongs here, not in the hook.

## Deliverables

- Read pending `.qratum/events/*.json`.
- Resolve `session_ref.transcript_path` relative to the current repo when not
  absolute.
- Produce placeholder artifact files for the session pipeline where later
  stages will fill content.
- Avoid reprocessing already completed events.

## Non-goals

No long-running daemon, installer, launchd/systemd integration, database, or
network sync.

## Verification

```sh
rm -rf .qratum
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
./bin/qrt daemon run-once
find .qratum -type f | sort
go test ./...
```

## Drift Handling

If event indexing or retention becomes necessary, defer it; do not add a
database in Milestone A.

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack duplicate processing, missing transcript errors without ApiError
  shape, and daemon code that assumes one hardcoded fixture path.
- Test behavior after two consecutive `run-once` executions.

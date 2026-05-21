# P2 — Daemon Run Once

## Scope

Implement `qrt daemon run-once` as the asynchronous pipeline shell.

## Decision Trace

- ADR 0002: daemon and hook model.
- ADR 0004: filesystem JSON for Milestone A.

## Behavior Contract

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

- Attack duplicate processing, missing transcript errors without ApiError
  shape, and daemon code that assumes one hardcoded fixture path.
- Test behavior after two consecutive `run-once` executions.

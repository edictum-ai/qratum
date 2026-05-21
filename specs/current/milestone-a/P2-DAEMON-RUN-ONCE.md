# P2 — Daemon Run Once

## Scope

Implement `qrt daemon run-once` as the asynchronous pipeline shell.

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

## Acceptance

```sh
rm -rf .qratum
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
./bin/qrt daemon run-once
find .qratum -type f | sort
go test ./...
```

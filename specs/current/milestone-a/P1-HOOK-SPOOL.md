# P1 — Hook And Spool

## Scope

Implement `qrt hook claude-code`.

## Deliverables

- Read Claude Code hook JSON from stdin.
- Tolerate unknown fields.
- Use `transcript_path` from the payload.
- Emit a `qratum.event.v1` CaptureEvent JSON file under `.qratum/events/`.
- Return quickly and do no heavy processing.

## Non-goals

Do not parse full transcripts, redact, generate reviews, render reports, export
ADP, call network, or call LLMs.

## Acceptance

```sh
rm -rf .qratum
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
test -n "$(find .qratum/events -name '*.json' -print -quit)"
go test ./...
```

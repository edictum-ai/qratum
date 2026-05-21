# P1 — Hook And Spool

## Scope

Implement `qrt hook claude-code`.

## Decision Trace

- ADR 0002: daemon and hook model.
- ADR 0004: filesystem JSON for Milestone A.
- ADR 0007: local-first raw storage.

## Behavior Contract

- The hook reads JSON from stdin and writes exactly one CaptureEvent per call.
- `transcript_path` comes from the hook payload.
- Unknown hook fields are tolerated.
- The hook does not parse transcripts, redact, render reports, export ADP, call
  network, or call LLMs.

## Deliverables

- Read Claude Code hook JSON from stdin.
- Tolerate unknown fields.
- Use `transcript_path` from the payload.
- Emit a `qratum.event.v1` CaptureEvent JSON file under `.qratum/events/`.
- Return quickly and do no heavy processing.

## Non-goals

Do not parse full transcripts, redact, generate reviews, render reports, export
ADP, call network, or call LLMs.

## Verification

```sh
rm -rf .qratum
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
test -n "$(find .qratum/events -name '*.json' -print -quit)"
go test ./...
```

## Drift Handling

If real Claude Code hook payloads differ from the fixture, add a redacted
fixture and keep the parser tolerant rather than hardcoding local paths.

## Slop Review

- Attack any hook path that takes more than tiny enqueue work.
- Test invalid hook payloads and missing `transcript_path`.

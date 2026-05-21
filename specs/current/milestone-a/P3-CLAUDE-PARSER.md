# P3 — Claude Transcript Parser

## Scope

Parse tolerant Claude-like JSONL fixture records into `qratum.session.v1`.

## Decision Trace

- ADR 0003: ADP as boundary.
- ADR 0007: local-first raw storage.
- SPEC.md: QratumSession is internal source of truth.

## Behavior Contract

- Parser output is QratumSession, not ADP.
- Unknown JSONL fields do not fail parsing.
- Tool calls, command runs, file changes, timestamps, and model are captured
  from available fixture data.
- Parser never reads hardcoded Claude local transcript directories.

## Deliverables

- `qrt normalize <transcript>`.
- Session model with turns, tool calls, file changes, commands, timestamps,
  model, git/workspace data where available.
- Unknown transcript fields do not break parsing.
- Fixture/golden tests for basic and verification-gap transcripts.

## Non-goals

Do not depend on hardcoded `~/.claude/projects` paths. Do not add Codex,
OpenCode, or real Claude transcript scraping.

## Verification

```sh
./bin/qrt normalize fixtures/claude-code/transcript-verification-gap.jsonl
go test ./...
```

## Drift Handling

If real redacted Claude transcripts require extra record types, add fixtures and
tests before broadening parser logic.

## Slop Review

- Attack brittle parsing that only works for the exact synthetic line order.
- Test malformed lines, unknown fields, and missing optional timestamps.

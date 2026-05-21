# P3 — Claude Transcript Parser

## Scope

Parse tolerant Claude-like JSONL fixture records into `qratum.session.v1`.

## Deliverables

- `qrt normalize <transcript>`.
- Session model with turns, tool calls, file changes, commands, timestamps,
  model, git/workspace data where available.
- Unknown transcript fields do not break parsing.
- Fixture/golden tests for basic and verification-gap transcripts.

## Non-goals

Do not depend on hardcoded `~/.claude/projects` paths. Do not add Codex,
OpenCode, or real Claude transcript scraping.

## Acceptance

```sh
./bin/qrt normalize fixtures/claude-code/transcript-verification-gap.jsonl
go test ./...
```

# P3 — Claude Transcript Parser

## Scope

Parse tolerant Claude-like JSONL fixture records into `qratum.session.v1`.

## Decision Trace

- ADR 0003: ADP as boundary.
- ADR 0007: local-first raw storage.
- SPEC.md: QratumSession is internal source of truth.

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

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack brittle parsing that only works for the exact synthetic line order.
- Test malformed lines, unknown fields, and missing optional timestamps.

# P1 — Hook And Spool

## Scope

Implement `qrt hook claude-code`.

## Decision Trace

- ADR 0002: daemon and hook model.
- ADR 0004: filesystem JSON for Milestone A.
- ADR 0007: local-first raw storage.

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

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack any hook path that takes more than tiny enqueue work.
- Test invalid hook payloads and missing `transcript_path`.

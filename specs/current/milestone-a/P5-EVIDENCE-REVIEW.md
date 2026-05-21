# P5 — Evidence And Review

## Scope

Implement deterministic evidence extraction and ReviewCard generation.

## Decision Trace

- ADR 0005: compact evidence judging.
- ADR 0006: review not score.
- SPEC.md: Milestone A finding enum.

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

- Findings are deterministic from QratumSession structure.
- ReviewCard contains verdict, main finding, evidence, one next habit, and
  optional suggested skill.
- Review text does not rank, shame, or score developers.

## Deliverables

- `qrt evidence <redacted-session>`.
- `qrt review <evidence>`.
- Findings:
  - `verification.final_edit_after_last_test`
  - `verification.missing_final_verification`
  - `reliability.repeated_failing_command`
- ReviewCard leads with what happened, evidence, and one next habit.

## Non-goals

No score, ranking, shaming, LLM judge, or tool-risk findings.

## Verification

```sh
./bin/qrt evidence fixtures/evidence/verification-gap.input.json
./bin/qrt review .qratum/evidence/ses_0001.evidence.json
go test ./...
```

## Drift Handling

If a finding needs tool-risk semantics, document it only; do not implement
tool-risk findings in Milestone A.

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack score-like language and vague findings without concrete evidence.
- Test the repeated failing command detector with fixture data.

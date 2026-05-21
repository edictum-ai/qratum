# P5 — Evidence And Review

## Scope

Implement deterministic evidence extraction and ReviewCard generation.

## Decision Trace

- ADR 0005: compact evidence judging.
- ADR 0006: review not score.
- SPEC.md: Milestone A finding enum.

## Behavior Contract

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

- Attack score-like language and vague findings without concrete evidence.
- Test the repeated failing command detector with fixture data.

# P5 — Evidence And Review

## Scope

Implement deterministic evidence extraction and ReviewCard generation.

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

## Acceptance

```sh
./bin/qrt evidence fixtures/evidence/verification-gap.input.json
./bin/qrt review .qratum/evidence/ses_0001.evidence.json
go test ./...
```

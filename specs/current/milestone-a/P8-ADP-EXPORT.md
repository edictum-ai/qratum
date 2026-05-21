# P8 — ADP Strict Export

## Scope

Export fixture-constrained ADP strict JSONL.

## Decision Trace

- ADR 0003: ADP as boundary.
- SPEC.md: ADP strict is interchange only.

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

- Exporter is deterministic.
- Output contains no `x-qratum-*`, secret maps, provenance internals, or raw
  Qratum-only fields.
- Milestone A targets fixture-constrained ADP-like trajectory/action/
  observation records only.

## Deliverables

- `qrt export <session> --profile adp-strict`.
- Deterministic JSONL output.
- No `x-qratum-*` fields.
- No Qratum internals or secret maps.
- Map user messages, assistant messages, tool calls, shell commands, and tool
  results into ADP-like trajectory/action/observation records.

## Non-goals

No full ADP validator and no marketplace/enterprise export profiles.

## Verification

```sh
./bin/qrt export .qratum/sessions/ses_0001.normalized.json --profile adp-strict
cmp .qratum/exports/ses_0001.adp.jsonl fixtures/adp/session.adp-strict.golden.jsonl
go test ./...
```

## Drift Handling

If full ADP schema validation is needed, add it as a later stage; do not block
Milestone A on a heavy validator.

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack ADP output that includes Qratum internals or nondeterministic ordering.
- Test golden equality against the fixture.

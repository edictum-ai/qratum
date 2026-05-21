# P8 — ADP Strict Export

## Scope

Export fixture-constrained ADP strict JSONL.

## Decision Trace

- ADR 0003: ADP as boundary.
- SPEC.md: ADP strict is interchange only.

## Behavior Contract

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

- Attack ADP output that includes Qratum internals or nondeterministic ordering.
- Test golden equality against the fixture.

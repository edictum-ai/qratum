# P8 — ADP Strict Export

## Scope

Export fixture-constrained ADP strict JSONL.

## Deliverables

- `qrt export <session> --profile adp-strict`.
- Deterministic JSONL output.
- No `x-qratum-*` fields.
- No Qratum internals or secret maps.
- Map user messages, assistant messages, tool calls, shell commands, and tool
  results into ADP-like trajectory/action/observation records.

## Non-goals

No full ADP validator and no marketplace/enterprise export profiles.

## Acceptance

```sh
./bin/qrt export .qratum/sessions/ses_0001.normalized.json --profile adp-strict
cmp .qratum/exports/ses_0001.adp.jsonl fixtures/adp/session.adp-strict.golden.jsonl
go test ./...
```

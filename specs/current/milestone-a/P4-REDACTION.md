# P4 — Deterministic Redaction

## Scope

Implement deterministic redaction for Milestone A.

## Deliverables

- `qrt redact <session>`.
- Detect obvious API keys, tokens, private-key-like blocks, JWT-like strings,
  URLs with credentials, high-entropy strings, and absolute local paths.
- Per-session stable placeholders.
- Redaction findings for `redaction.secret_detected` and
  `redaction.path_redacted`.
- Golden fixture for `fixtures/redaction/secret-session.input.json`.

## Non-goals

No LLM redaction, encrypted vault, cloud/account ID coverage, emails, ticket
IDs, or enterprise sharing.

## Acceptance

```sh
./bin/qrt redact fixtures/redaction/secret-session.input.json
go test ./...
```

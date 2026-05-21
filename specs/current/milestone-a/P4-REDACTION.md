# P4 — Deterministic Redaction

## Scope

Implement deterministic redaction for Milestone A.

## Decision Trace

- ADR 0004: filesystem JSON for Milestone A.
- ADR 0007: local-first raw storage.
- SPEC.md: no LLM redaction and no encrypted vault in Milestone A.

## Behavior Contract

- Redaction is deterministic for golden tests.
- Raw secrets, credential URLs, JWT-like values, and local absolute paths are
  removed from redacted output.
- Secret placeholder maps are not exposed through UI DTOs.

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

## Verification

```sh
./bin/qrt redact fixtures/redaction/secret-session.input.json
go test ./...
```

## Drift Handling

If more detector classes are needed, add fixtures first and keep the scope to
deterministic local rules.

## Slop Review

- Attack redaction that leaves fixture secrets in output.
- Test by searching redacted artifacts for the raw API key, URL password, JWT,
  and local absolute path.

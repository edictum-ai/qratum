# P4 — Deterministic Redaction

## Scope

Implement deterministic redaction for Milestone A.

## Decision Trace

- ADR 0004: filesystem JSON for Milestone A.
- ADR 0007: local-first raw storage.
- SPEC.md: no LLM redaction and no encrypted vault in Milestone A.

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

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack redaction that leaves fixture secrets in output.
- Test by searching redacted artifacts for the raw API key, URL password, JWT,
  and local absolute path.

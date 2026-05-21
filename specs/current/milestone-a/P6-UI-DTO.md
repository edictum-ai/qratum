# P6 — UI DTO Outputs

## Scope

Map local artifacts into stable UI DTOs before any web UI exists.

## Decision Trace

- ADR 0009: UI contract first.
- docs/contracts/ui-api.md.

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

- CLI emits UI DTOs, not raw QratumSession internals.
- DTOs include stable `schema_version` fields.
- Raw transcripts, secret maps, redaction internals, ADP JSONL, and provenance
  internals are not required for UI rendering.

## Deliverables

- `qrt ui sessions --json`
- `qrt ui session <session_id> --json`
- `qrt ui review <session_id> --json`
- Fixtures under `fixtures/ui/`.
- DTOs for SessionListItem, SessionDetail, ReviewCard, EvidenceFinding,
  ArtifactLink, and ApiError.

## Non-goals

No HTTP server, React app, trend UI, skill candidates, or raw transcript
exposure.

## Verification

```sh
./bin/qrt ui sessions --json
./bin/qrt ui session ses_0001 --json
./bin/qrt ui review ses_0001 --json
go test ./...
```

## Drift Handling

If a UI need is not represented in the DTOs, amend the schema and fixture
together before changing CLI output.

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack DTOs that leak raw transcript content or require frontend parsing of
  internal artifacts.
- Test missing artifact and missing session error paths.

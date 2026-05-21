# P6 — UI DTO Outputs

## Scope

Map local artifacts into stable UI DTOs before any web UI exists.

## Decision Trace

- ADR 0009: UI contract first.
- docs/contracts/ui-api.md.

## Behavior Contract

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

- Attack DTOs that leak raw transcript content or require frontend parsing of
  internal artifacts.
- Test missing artifact and missing session error paths.

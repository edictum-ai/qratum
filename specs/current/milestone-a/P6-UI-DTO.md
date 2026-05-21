# P6 — UI DTO Outputs

## Scope

Map local artifacts into stable UI DTOs before any web UI exists.

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

## Acceptance

```sh
./bin/qrt ui sessions --json
./bin/qrt ui session ses_0001 --json
./bin/qrt ui review ses_0001 --json
go test ./...
```

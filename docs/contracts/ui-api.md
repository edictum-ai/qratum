# Qratum UI API Contract

The UI consumes Qratum UI DTOs, not raw internal models.

Milestone A has no HTTP server. The same contract is exposed through local CLI
commands:

- `qrt ui sessions --json`
- `qrt ui session <session_id> --json`
- `qrt ui review <session_id> --json`

Backend must provide:

- SessionListItem for lists
- SessionDetail for detail pages
- ReviewCard for review sections
- EvidenceFinding for findings
- ArtifactLink for artifacts
- ApiError for errors

Backend must not require UI to parse:

- Claude Code transcript JSONL
- raw QratumSession internals
- ADP JSONL
- redaction internals
- provenance internals

Backend must never expose by default:

- raw transcript
- vault content
- secret placeholder map
- unredacted file paths in share-safe mode
- unredacted prompts

Future HTTP shape:

- `GET /api/v1/health`
- `GET /api/v1/sessions`
- `GET /api/v1/sessions/{session_id}`
- `GET /api/v1/sessions/{session_id}/review`
- `GET /api/v1/sessions/{session_id}/evidence`
- `GET /api/v1/sessions/{session_id}/artifacts`

This document is a contract reference. SPEC.md remains the executable scope.

# Qratum Milestone A SPEC

## Goal

Build the first local vertical slice:

Claude Code hook fixture -> CaptureEvent -> filesystem spool -> daemon run-once
-> read transcript_path -> parse Claude Code transcript fixture -> emit
QratumSession -> deterministic redaction -> emit EvidenceBundle -> emit
ReviewCard -> emit UI DTOs -> render HTML report -> export ADP strict JSONL.

## Non-goals

Do not implement:

- enterprise server
- marketplace
- Codex adapter
- OpenCode adapter
- Copilot adapter
- MCP
- GitHub comments
- GitHub App
- GitLab
- LLM scoring
- LLM redaction
- web UI
- HTTP server
- database
- bbolt
- SQLite
- Postgres
- encrypted vault
- Edictum integration

## Runtime

Go single binary: `qrt`.

No Python runtime. No database. Filesystem JSON only.

## Required Commands

- `qrt --version`
- `qrt status`
- `qrt hook claude-code`
- `qrt daemon run-once`
- `qrt sessions list`
- `qrt normalize <transcript>`
- `qrt redact <session>`
- `qrt evidence <redacted-session>`
- `qrt review <evidence>`
- `qrt report <session>`
- `qrt export <session> --profile adp-strict`
- `qrt ui sessions --json`
- `qrt ui session <session_id> --json`
- `qrt ui review <session_id> --json`

## Acceptance

```sh
cat fixtures/claude-code/hook-session-end.json | qrt hook claude-code
qrt daemon run-once
make test
make demo
```

Expected artifacts:

```txt
.qratum/events/*.json
.qratum/sessions/*.normalized.json
.qratum/redacted/*.redacted.json
.qratum/evidence/*.evidence.json
.qratum/reviews/*.review.json
.qratum/reports/*.html
.qratum/exports/*.adp.jsonl
```

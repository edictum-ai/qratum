# Qratum Milestone A

Build Qratum's first local vertical slice only.

## Agent Routing

- Builder: `gpt-5-5`
- Reviewer: `opus`

## Goal

Claude Code hook fixture -> CaptureEvent -> filesystem spool -> daemon
run-once -> transcript_path parser -> QratumSession -> deterministic
redaction -> EvidenceBundle -> ReviewCard -> UI DTOs -> static HTML report ->
ADP strict JSONL export.

## Non-goals

Do not implement enterprise server, marketplace, Codex, OpenCode, Copilot,
MCP, GitHub comments, GitHub App, GitLab, LLM scoring, LLM redaction, web UI,
HTTP server, database, bbolt, SQLite, Postgres, encrypted vault, or Edictum
integration.

## Verification

Every stage must keep these green when affected:

```sh
go test ./...
make build
make demo
```

`make demo` must run:

```sh
rm -rf .qratum
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
./bin/qrt daemon run-once
./bin/qrt sessions list
```

and print all generated artifact paths.

## Stages

| # | Prompt | Deliverable |
|---|---|---|
| 0 | [P0-REPO-SKELETON-CLEANUP.md](P0-REPO-SKELETON-CLEANUP.md) | Empty `qrt` binary, build/test/demo shell discipline |
| 1 | [P1-HOOK-SPOOL.md](P1-HOOK-SPOOL.md) | `qrt hook claude-code` and CaptureEvent writer |
| 2 | [P2-DAEMON-RUN-ONCE.md](P2-DAEMON-RUN-ONCE.md) | event reader and pipeline shell |
| 3 | [P3-CLAUDE-PARSER.md](P3-CLAUDE-PARSER.md) | tolerant Claude JSONL parser to QratumSession |
| 4 | [P4-REDACTION.md](P4-REDACTION.md) | deterministic redaction and golden tests |
| 5 | [P5-EVIDENCE-REVIEW.md](P5-EVIDENCE-REVIEW.md) | findings and ReviewCard |
| 6 | [P6-UI-DTO.md](P6-UI-DTO.md) | UI DTO mappings and CLI JSON commands |
| 7 | [P7-HTML-REPORT.md](P7-HTML-REPORT.md) | escaped static HTML report |
| 8 | [P8-ADP-EXPORT.md](P8-ADP-EXPORT.md) | fixture-constrained ADP strict JSONL |
| 9 | [P9-DEMO-HARDENING.md](P9-DEMO-HARDENING.md) | full vertical slice demo hardened |

## Slop Review

- Attack any implementation that adds a database, server, web UI, LLM call, or
  non-Claude adapter in Milestone A.
- Attack any hook path that parses full transcripts, calls network, or does
  heavy work.
- Attack any report or UI DTO that exposes raw transcript content, secret maps,
  or unredacted local paths in share-safe output.
- Attack any ADP export that includes `x-qratum-*` fields or Qratum internals.
- Attack any output that depends on wall-clock time instead of fixture
  timestamps.

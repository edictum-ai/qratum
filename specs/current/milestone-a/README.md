# Qratum Milestone A

Status: historical. Milestone A proved the first local vertical slice and may
remain as compatibility/debug behavior. It is no longer the current product
model.

Current source of truth:

```txt
SPEC.md
specs/current/operational-model-redesign.md
```

Build Qratum's first local vertical slice only.

## Agent Routing

- Builder: `gpt-5-5`
- Reviewer: `opus`

## Goal

Claude Code hook fixture -> CaptureEvent -> filesystem spool -> daemon
run-once -> transcript_path parser -> QratumSession -> deterministic
redaction -> EvidenceBundle -> ReviewCard -> UI DTOs -> static HTML report ->
ADP strict JSONL export.

## Decision Trace

- ADR 0001: Go single binary.
- ADR 0002: daemon and hook model.
- ADR 0003: ADP as boundary.
- ADR 0004: filesystem JSON for Milestone A.
- ADR 0005: compact evidence judging.
- ADR 0006: review not score.
- ADR 0007: local-first raw storage.
- ADR 0008: GitHub only.
- ADR 0009: UI contract first.

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

- Milestone A stays local-only and filesystem-only.
- `qrt hook claude-code` reads hook JSON from stdin, writes one CaptureEvent,
  and exits without parsing transcripts, calling network, or invoking LLMs.
- `qrt daemon run-once` owns heavy work and produces the expected artifact set.
- UI outputs are DTOs, not raw transcripts, Qratum internals, ADP internals,
  redaction internals, or provenance internals.
- Review output leads with evidence and one next habit, never a score.
- ADP strict export is an interchange boundary and must not include Qratum-only
  fields.

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

## Drift Handling

- If a stage needs a database, server, web UI, marketplace, non-Claude adapter,
  LLM call, encrypted vault, or Edictum dependency, stop and add a new ADR
  before implementation.
- If fixture output changes intentionally, update the golden fixture and explain
  the contract change in the task completion.
- If the Claude Code fixture shape proves wrong against a real transcript,
  add a redacted fixture and document the parser tolerance added.

## Execution Order

| # | Prompt | Scope | Deliverable | Status | Depends On |
|---|---|---|---|---|---|
| 0 | [P0-REPO-SKELETON-CLEANUP.md](P0-REPO-SKELETON-CLEANUP.md) | cli/build | Empty `qrt` binary, build/test/demo shell discipline | [ ] | - |
| 1 | [P1-HOOK-SPOOL.md](P1-HOOK-SPOOL.md) | capture/spool | `qrt hook claude-code` and CaptureEvent writer | [ ] | P0 |
| 2 | [P2-DAEMON-RUN-ONCE.md](P2-DAEMON-RUN-ONCE.md) | daemon/spool | event reader and pipeline shell | [ ] | P1 |
| 3 | [P3-CLAUDE-PARSER.md](P3-CLAUDE-PARSER.md) | normalize | tolerant Claude JSONL parser to QratumSession | [ ] | P2 |
| 4 | [P4-REDACTION.md](P4-REDACTION.md) | redaction | deterministic redaction and golden tests | [ ] | P3 |
| 5 | [P5-EVIDENCE-REVIEW.md](P5-EVIDENCE-REVIEW.md) | evidence/review | findings and ReviewCard | [ ] | P4 |
| 6 | [P6-UI-DTO.md](P6-UI-DTO.md) | ui | UI DTO mappings and CLI JSON commands | [ ] | P5 |
| 7 | [P7-HTML-REPORT.md](P7-HTML-REPORT.md) | reports/provenance | escaped static HTML report | [ ] | P6 |
| 8 | [P8-ADP-EXPORT.md](P8-ADP-EXPORT.md) | adp | fixture-constrained ADP strict JSONL | [ ] | P7 |
| 9 | [P9-DEMO-HARDENING.md](P9-DEMO-HARDENING.md) | demo | full vertical slice demo hardened | [ ] | P8 |

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack any implementation that adds a database, server, web UI, LLM call, or
  non-Claude adapter in Milestone A.
- Attack any hook path that parses full transcripts, calls network, or does
  heavy work.
- Attack any report or UI DTO that exposes raw transcript content, secret maps,
  or unredacted local paths in share-safe output.
- Attack any ADP export that includes `x-qratum-*` fields or Qratum internals.
- Attack any output that depends on wall-clock time instead of fixture
  timestamps.
- Test the behavior with `make demo`; do not accept shape-only code that leaves
  the vertical slice broken.

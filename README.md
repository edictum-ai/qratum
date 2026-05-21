# Qratum

Qratum is a local-first trust pipeline for AI coding sessions.

Milestone A builds one local vertical slice: Claude Code hook capture,
filesystem JSON spool, daemon run-once, normalized session output,
deterministic redaction, evidence extraction, review cards, UI DTOs, static
HTML reports, and ADP strict JSONL export.

## Quick Start

```sh
make build
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
./bin/qrt daemon run-once
./bin/qrt sessions list
```

Or run the whole vertical slice:

```sh
make demo
```

## Verification

```sh
go test ./...
make build
make demo
```

## Scope

Follow `SPEC.md` for executable scope. Files under `docs/architecture/` are
forward design references and do not expand Milestone A.

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

## Dogfood on a local transcript

Process a local Claude Code JSONL transcript without uploading it:

```sh
qrt dogfood import /path/to/transcript.jsonl
qrt dogfood latest
```

Raw transcripts stay local and Qratum does not upload anything. The dogfood
import reads the transcript, writes redacted Qratum artifacts under `.qratum/`,
and does not copy the raw JSONL transcript there. Deterministic redaction is
best-effort alpha quality.

## Verification

```sh
make build
make test
make demo
make dogfood-demo
make supply-chain
make security
```

Use `make verify` for the full local CI mirror.

## Scope

Follow `SPEC.md` for executable scope. Files under `docs/architecture/` are
forward design references and do not expand Milestone A.

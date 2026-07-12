# Wave 1 Source Fixtures

## What

These are synthetic Claude Code `2.1.207` and Codex `0.144.1` JSONL examples.
They lock the source fields Qratum uses for session identity and token usage.

## Why

Source formats can change without notice. Qratum must show unsupported coverage
instead of silently guessing when the identity or usage shape changes.

## How

The fixtures reproduce only field names and value types observed locally on
2026-07-12. They contain invented IDs, paths, repositories, text, models, and
token counts. No real transcript text was copied into the repository.

Each source directory contains:

- a supported new-and-resumed session;
- explicit source-version drift;
- unknown record shapes; and
- wrong token-field types.

Codex also covers cumulative reconciliation, a counter reset, and a mismatch.
Parser tests must fail closed on version and type drift and must report unknown
records as incomplete coverage.

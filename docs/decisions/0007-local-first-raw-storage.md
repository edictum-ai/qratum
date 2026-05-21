# ADR 0007: Local-First Raw Storage

## Status

Accepted

## Decision

Raw transcripts stay local by default.

## Reason

Transcripts can contain secrets, prompts, local paths, and business context.

## Consequences

Only redacted artifacts are eligible for future sync or sharing.

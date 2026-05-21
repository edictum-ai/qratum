# ADR 0009: UI Contract First

## Status

Accepted

## Decision

Backend emits stable UI DTOs before a web UI exists.

## Reason

The future UI should consume explicit product DTOs, not parse transcripts,
internal sessions, ADP, redaction internals, or provenance internals.

## Consequences

Milestone A includes UI schemas, fixtures, docs, and CLI JSON outputs.

# ADR 0002: Daemon And Hook Model

## Status

Accepted

## Decision

Hooks are tiny and enqueue CaptureEvents. The daemon performs parsing,
redaction, evidence extraction, review, report rendering, and export.

## Reason

Agent hooks must return quickly and avoid heavy work in the agent process.

## Consequences

`qrt hook claude-code` reads stdin, writes one event, and exits.

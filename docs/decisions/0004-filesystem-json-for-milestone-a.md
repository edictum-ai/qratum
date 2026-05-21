# ADR 0004: Filesystem JSON For Milestone A

## Status

Accepted

## Decision

Milestone A stores artifacts as filesystem JSON and HTML under `.qratum/`.

## Reason

The first milestone needs a simple inspectable local loop, not indexing,
retention, or trend queries.

## Consequences

No bbolt, SQLite, or Postgres in Milestone A.

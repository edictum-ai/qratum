# ADR 0010: Vault First And Direct Gateway Integration

## Status

Accepted 2026-06-14

## Decision

- Qratum is vault-first: preservation is the first earned runtime, ahead of
  review, search, or corpus work.
- Qratum does not standardize a one-person publish ceremony just to move one
  person's curated data between their own tools.
- Personal-memory owns live-store curation and lifecycle; Qratum remains the
  librarian and provenance archive.
- Integration happens through direct gateway calls from small local scripts or
  commands that hold their own keychain or environment credential.

## Reason

The accepted review set in `docs/reviews/2026-06-12-memory-architecture/`
showed that preservation is the only urgent, irreversible gap; the
bundle/importer/receipt bridge added ceremony without a real trust split; the
live store, not Qratum, needs to own curation; and the one real boundary is
still explicit: never send raw transcripts.

## Consequences

- `specs/current/qratum-vault-first.md` is the accepted post-review revision
  alongside the operational model doc it edits in place.
- `specs/current/memory-curation-pipeline.md` remains historical and
  superseded, not active direction.
- Future integration work targets gateway verbs plus thin local scripts or
  commands, not a standing publish/import pipeline.
- This ADR changes specs and decisions only. It does not change current Go
  runtime, schemas, fixtures, or Makefiles.

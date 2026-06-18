# Qratum Schema Migration Contract

Readers must not silently default a missing or unknown `schema_version`.

A blank or unrecognized version is a loud error. `LoadState` silently
back-filling a missing vault state version is the anti-pattern this contract
forbids for new readers.

Content-addressed raw blobs are immutable. A v1-to-v2 derivation regenerates a
new object from the blob and writes a new ref or derived artifact; it never
edits an existing blob in place.

`qrt vault archive --kind memory_import_receipt` must pin the kind explicitly.
The historical default to `source_metadata` is a documented footgun, not a
receipt contract.

Forward design reference.
This document does not expand the current implementation scope.
Current executable scope is defined only in SPEC.md.

# Security Model

Hard rules:

- No raw transcript leaves the machine by default.
- No HTML report renders raw transcript.
- No UI DTO exposes secret mappings.
- No skill reads vault data.
- No marketplace pack includes raw local data.
- No PR comment includes file contents.

Milestone A uses deterministic redaction, HTML escaping, local-only raw storage,
strict export profiles, and provenance digests where practical.

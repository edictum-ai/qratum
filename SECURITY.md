# Security Policy

## Reporting A Vulnerability

Do not open a public GitHub issue for security vulnerabilities.

Email: security@edictum.ai

We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## Scope

Qratum handles AI coding transcripts, local paths, secrets, redacted artifacts,
review cards, and export files. Treat transcript handling, redaction, artifact
generation, HTML report rendering, and export boundaries as security-sensitive.

## Current Guarantees

- Raw transcripts stay local by default.
- Qratum does not upload transcript content in Milestone A.
- HTML reports must not render raw transcripts.
- UI DTOs must not expose secret placeholder maps.
- ADP strict export must not include Qratum-only fields or secret maps.
- CI uses pinned GitHub Actions and pinned Go security tools.

## Supply Chain

Follow `docs/supply-chain.md` for CI and dependency rules. The repository
rejects unpinned workflow actions, pipe-to-shell installers, floating tool
versions, and non-Go package manager installs in the runtime pipeline.

<p align="center">
  <img src="design/brand/qratum-logo.svg" width="96" height="96" alt="Qratum">
</p>

<h1 align="center">Qratum</h1>

<p align="center"><em>The local librarian for your AI coding sessions.</em></p>

<p align="center">
  <a href="https://qratum.dev">qratum.dev</a> ·
  <a href="#quick-start">Install</a> ·
  <a href="SPEC.md">Spec</a> ·
  <a href="LICENSE">MIT</a>
</p>

---

The accepted direction for **Qratum** is a private, searchable memory of Claude
and Codex work: a clean local UI for finding, reading, understanding, and when
useful continuing exact session history. That product direction was accepted on
2026-07-11; its implementation is tranche-gated and has not shipped.

The published `v0.1.0` artifact is the earlier vault-first baseline. It captures,
preserves, normalizes, redacts, and reviews local Claude Code sessions through a
pipeline-shaped CLI. Other sources are archive-only in that release. Single Go
binary (`qrt`). No cloud, no accounts, no telemetry.

Qratum is the system of record for **where session data came from**: raw
history, provenance, and deterministic derivations. The first user is the
developer running it on their own machine.

### Plain-language glossary

A few terms used below, in plain words:

- **Transcript** — the full text record of one AI coding session.
- **Blob** — a single stored file in the vault (e.g. one captured transcript).
- **Content-addressed / sha256-addressed** — each blob is named by a hash of its
  own contents, so identical files are stored once and any change produces a new
  name.
- **Redaction** — automatically removing secrets and sensitive data before
  anything is exported.
- **Deterministic** — given the same input, it always produces the exact same
  output (no randomness).
- **Provenance** — the recorded history of where each piece of data came from and
  how it was transformed.
- **Tombstone** — a marker that records "this was removed" instead of silently
  deleting it.
- **ADP** — the redacted export format the refinery produces for downstream use.

## Highlights

- **Local-first** — your data stays on your machine. Raw transcripts never leave it unless you explicitly approve. No cloud, no accounts, no telemetry.
- **Single Go binary** — one file to install. `qrt`, cross-platform.
- **Trust boundaries** — nothing leaks out by accident. No silent data-class upgrades; deterministic redaction gates export; no raw routes.
- **Content-addressed** — every blob is sha256-addressed so mutation can be
  detected and identical content can be deduplicated. This is an integrity
  foundation, not by itself a backup or no-loss guarantee.

## Why

The tools that make your AI coding sessions also throw them away. Claude Code
purges transcripts after ~30 days — months of debugging, decisions, and
trajectories vanish, and nothing records where the data came from (its
provenance). Qratum fixes both: it preserves your sessions, and it remembers
where the data came from.

## Accepted product pillars

The accepted product direction is built on three parts:

- **Exact local history** — owner-only session history remains readable even
  when a source tool no longer keeps its copy.
- **Search and continuity** — passage-level lexical and semantic retrieval,
  confirmed continuations, related work, and optional source-native continue
  actions.
- **Provenance and control** — exact, deterministic, and generated information
  stay distinct; export, deletion, external processing, and cost are explicit
  and truthful.

Local-first is the architecture, not a feature: raw never leaves the machine
unless explicitly approved; no boundary may silently upgrade to a more
sensitive data class; deterministic redaction gates export.

## Status

**v0.1.0 — published baseline.** It contains the vault-first runtime, the old
pipeline-shaped CLI, and the P2 trust-gate implementation. It does **not**
contain `qrt init` or `qrt open`.

The current development worktree contains a later UI-first CLI/API candidate,
but that candidate is not published and its old product contract has been
superseded. Existing candidate code is donor material for the accepted tranches,
not proof that the product direction already works.

Honest boundaries — three limits to know about today:

- Redaction is not yet airtight. The automatic secret-removal step
  (deterministic redaction) is **best-effort alpha**. Cheap credential classes
  are covered by the trust gate, but the known-miss ledger still names residual
  classes that can leak.
- Re-deriving from a deleted local Claude Code source is wired through the raw
  vault blob fallback. The proof is still local-only: cloud/web sessions are not
  captured, and `transcript_drift` remains a heuristic rather than a correctness
  gate.
- Capture and refine are Claude-Code-only. Cloud/web sessions are not captured,
  and non-Claude/vendor blobs are archive-only with no redaction path.

- Source of truth: [`SPEC.md`](SPEC.md) → [`specs/current/product-direction.md`](specs/current/product-direction.md)
- Accepted user stories: [`docs/reviews/2026-07-10-product-user-stories.md`](docs/reviews/2026-07-10-product-user-stories.md)
- Accepted Project Intelligence extension: [`docs/reviews/2026-07-12-project-intelligence-user-stories.md`](docs/reviews/2026-07-12-project-intelligence-user-stories.md) and [`docs/reviews/2026-07-12-project-intelligence-owner-decisions.md`](docs/reviews/2026-07-12-project-intelligence-owner-decisions.md)
- Vault-first (accepted 2026-06-14, [ADR 0010](docs/decisions/0010-vault-first-and-direct-gateway-integration.md)): [`specs/current/qratum-vault-first.md`](specs/current/qratum-vault-first.md)
- Verification & trust gate (published v0.1.0 baseline): [`specs/current/verification-and-trust-gate.md`](specs/current/verification-and-trust-gate.md)

## Install

```sh
brew tap edictum-ai/edictum
brew trust edictum-ai/edictum   # Homebrew gates third-party taps
brew install qratum             # installs the `qrt` binary
```

Prebuilt binaries (darwin/linux × amd64/arm64) are also on each
[release](https://github.com/edictum-ai/qratum/releases). Or build from source
with `make build`.

## Quick start

```sh
qrt status
qrt vault doctor
qrt vault backfill
qrt sessions list
qrt trust
```

These are `v0.1.0` commands, not the accepted future public surface.

Run the whole vertical slice from a source checkout:

```sh
make demo
```

## Process a local Claude Code transcript in v0.1.0

Run the published on-demand refinery without uploading the transcript:

```sh
qrt dogfood import /path/to/transcript.jsonl
qrt dogfood latest
```

Raw transcripts stay local; Qratum does not upload anything. Deterministic
redaction is best-effort alpha.

## Verification

```sh
make build
make test
make demo
make dogfood-demo
make supply-chain
make security
```

`make verify` mirrors the full CI pipeline locally.

## Scope

Follow [`SPEC.md`](SPEC.md) for executable scope. Files under `docs/architecture/`
are forward design references and do not expand the current milestone.

## Landing page & deploy

The public landing page lives in [`design/landing-drafts/`](design/landing-drafts)
(Vite + React + framer-motion; the "Light Ledger" neobrutalist direction). It
deploys to <https://qratum.dev> via Vercel, independently of the `qrt` Go
runtime. See its [`README`](design/landing-drafts/README.md) for the deploy
setup.

## Brand

The Qratum identity — the `#·` content-addressed mark (hash + dot), palette,
typography, and voice — lives in [`design/brand/`](design/brand/):

- [`brand-book.html`](design/brand/brand-book.html) — the full brand book.
- [`qratum-mark.svg`](design/brand/qratum-mark.svg) / [`qratum-mark-mono.svg`](design/brand/qratum-mark-mono.svg) — the mark (color / mono).
- [`qratum-logo.svg`](design/brand/qratum-logo.svg) — the bordered logo tile.

## Ecosystem

- **Qratum** — the librarian: vault + refinery for AI session data.
- **Edictum** — runtime process enforcement for AI agents.
- **Ductum** — _coming soon: the AI Software Factory._

## License

[MIT](LICENSE).

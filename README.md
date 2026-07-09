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

**Qratum** keeps a permanent, private copy of your AI coding sessions on your own
machine, so nothing is lost when the tool that made them deletes them. It is a
local-first library, vault, and review pipeline for AI coding sessions. It
captures, preserves, normalizes, redacts, and reviews local Claude Code sessions
**without ever uploading your raw transcripts.** Other sources are archive-only
today; they do not have a redaction/refinery path. Single Go binary (`qrt`). No
cloud, no accounts, no telemetry.

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
- **Content-addressed** — files can't be silently changed or lost. Every blob is sha256-addressed and immutable; tombstones (removal is recorded), never silent deletion.

## Why

The tools that make your AI coding sessions also throw them away. Claude Code
purges transcripts after ~30 days — months of debugging, decisions, and
trajectories vanish, and nothing records where the data came from (its
provenance). Qratum fixes both: it preserves your sessions, and it remembers
where the data came from.

## Three pillars

Qratum is built on three parts:

- **The Vault** — keeps a permanent, private copy of every session so nothing is
  lost. It copies each transcript on capture and archives it by content hash, so
  a transcript the tool deletes tomorrow is still recoverable. Identical files
  are stored once (dedup by sha256), and `backup --verify` confirms the copy is
  intact.
- **The Refinery** — turns a raw session into a safe, reviewed export, but only
  when you ask. On demand it runs each session through these steps: normalize →
  deterministic redaction (remove secrets) → evidence → review → report →
  corpus. It runs only when you ask (`qrt daemon run-once`): there is no standing
  daemon and no background queue.
- **Provenance** — records where every piece of data came from and how it
  changed. Every object carries its `schema_version`, the producer that made it,
  and the transform version that shaped it. Each one moves through a recorded
  data-class lineage (`raw → redacted → review → corpus → published`), and
  removal is always recorded as a tombstone — never a silent deletion.

Local-first is the architecture, not a feature: raw never leaves the machine
unless explicitly approved; no boundary may silently upgrade to a more
sensitive data class; deterministic redaction gates export.

## Status

**v0.1.0 — first release.** Here is what works today and what does not.

The vault has shipped. The vault is the part that keeps the permanent private
copy (P1, the vault-first runtime). These pieces are merged and test-backed: the
copy-on-capture hook, the content-addressed blob store, `hook install`, the
`vault backfill/archive/backup --verify/doctor/gc` commands, the tombstone-based
`vault erase`, the `vault install-schedule` backfill timer, and `status`. The
refinery (normalize → redact → evidence → review → report → ADP export) runs as
on-demand tooling.

The verification trust gate has shipped too. `qrt trust` (and `make trust`) runs
the real CLI against planted-secret and reflection-canary corpora and emits a
`qratum.trust_scorecard.v1` verdict — `TRUSTED`, `TRUSTED-WITH-NAMED-GAPS`, or
`NOT-TRUSTED` — with an honest-residual block stating exactly what it does not
cover. Every emitted object carries a `data_class` and is schema-validated.

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

- Source of truth: [`SPEC.md`](SPEC.md) → [`specs/current/operational-model-redesign.md`](specs/current/operational-model-redesign.md)
- Vault-first (accepted 2026-06-14, [ADR 0010](docs/decisions/0010-vault-first-and-direct-gateway-integration.md)): [`specs/current/qratum-vault-first.md`](specs/current/qratum-vault-first.md)
- Verification & trust gate (shipped in v0.1.0): [`specs/current/verification-and-trust-gate.md`](specs/current/verification-and-trust-gate.md)

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
qrt hook install        # capture every future Claude Code session into the vault
qrt daemon run-once     # refine on demand: normalize → redact → evidence → review → report → export
qrt sessions list
qrt trust               # the self-verification scorecard (what's proven, what's a named gap)
```

Run the whole vertical slice from a source checkout:

```sh
make demo
```

## Dogfood on a local transcript

Process a local Claude Code JSONL transcript without uploading it:

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

### Process-guard hook (optional)

This repo is governed by the [Engineering OS](https://github.com/acartag7/engineering-os).
CI runs a `process-guard` job on every PR. To mirror it locally, enable the
committed pre-commit hook once per clone:

```sh
git config core.hooksPath .githooks
```

The hook is best-effort: it prints a warning and exits 0 when `node` or an
`engineering-os` checkout is absent (this repo has no Node toolchain), so it never
blocks a commit on missing optional tooling. Set `ENGINEERING_OS_HOME=<path>` if
the checkout is not a sibling directory. CI enforces the guard regardless.

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

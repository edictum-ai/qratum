# Qratum

> The local librarian for your AI coding sessions.

**Qratum** is a local-first library, vault, and review pipeline for AI coding
sessions. It captures, preserves, normalizes, redacts, reviews, and searches
every session — **without ever uploading your raw transcripts.** Single Go
binary (`qrt`). No cloud, no accounts, no telemetry.

🌐 Live site: **<https://qratum.dev>**

Qratum is the system of record for **where session data came from**: raw
history, provenance, and deterministic derivations. The first user is the
developer running it on their own machine.

## Why

AI coding tools delete your sessions. Claude Code purges transcripts after
~30 days — months of debugging, decisions, and trajectories vanish, and nothing
records provenance. Qratum fixes both: it preserves, and it remembers where the
data came from.

## Three pillars

- **The Vault** — content-addressed capture & archive. A transcript the tool
  deletes tomorrow is recoverable. Copy-on-capture, dedup by sha256,
  `backup --verify`.
- **The Refinery** — on demand: normalize → deterministic redaction → evidence →
  review → report → corpus. Runs only when you ask. No daemon, no queue.
- **Provenance** — every object carries `schema_version`, producer, and
  transform version; data-class lineage `raw → redacted → review → corpus →
  published`; tombstones, never silent deletion.

Local-first is the architecture, not a feature: raw never leaves the machine
unless explicitly approved; no boundary may silently upgrade to a more
sensitive data class; deterministic redaction gates export.

## Status

**Pre-1.0, spec phase (P0).** Milestone A (one local vertical slice) is proven
and remains as compatibility/debug behavior while the new operational model is
specified. The vault-first proposal is under review. Deterministic redaction is
best-effort alpha. Nothing beyond Milestone A is promised as shipped.

- Source of truth: [`SPEC.md`](SPEC.md) → [`specs/current/operational-model-redesign.md`](specs/current/operational-model-redesign.md)
- Vault-first proposal: [`specs/current/qratum-vault-first.md`](specs/current/qratum-vault-first.md)

## Quick start

```sh
make build
cat fixtures/claude-code/hook-session-end.json | ./bin/qrt hook claude-code
./bin/qrt daemon run-once
./bin/qrt sessions list
```

Run the whole vertical slice:

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

## Scope

Follow [`SPEC.md`](SPEC.md) for executable scope. Files under `docs/architecture/`
are forward design references and do not expand the current milestone.

## Landing page & deploy

The public landing page lives in [`design/landing-drafts/`](design/landing-drafts)
(Vite + React + framer-motion; the "Light Ledger" neobrutalist direction). It
deploys to <https://qratum.dev> via Vercel, independently of the `qrt` Go
runtime. See its [`README`](design/landing-drafts/README.md) for the deploy
setup.

## Ecosystem

- **Qratum** — the librarian: vault + refinery for AI session data.
- **Edictum** — runtime process enforcement for AI agents.
- **Ductum** — _coming soon: the AI Software Factory._

## License

[MIT](LICENSE).

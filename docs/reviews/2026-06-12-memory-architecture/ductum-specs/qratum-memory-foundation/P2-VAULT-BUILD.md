# P2 — Qratum Vault Build (runtime)

Package: `qratum-memory-foundation` · Prompt 2 of 2 · Depends on: P1 ·
Scope: runtime/vault · Deliverable: local vault workflow + tests + runbook

## Objective

Build the Qratum vault: stop losing AI session transcripts by copying each one
into a content-addressed local archive at capture time, and give the vault
first-class install/inspect/backfill/archive/backup commands so preservation is
owned, not manual tribal knowledge. This is spine pillar 1 — the only component
with an irreversibility clock (Claude Code hard-deletes transcripts after ~30
days; one captured transcript is already gone).

Build only the vault-minimum. Do not build the full v2 workspace, search,
refinery expansion, lessons, or any "Dead" item.

## Read first

- `qratum/specs/current/qratum-vault-first.md` (the contract — sections
  "The Vault", "Operational ownership", "Backfill and archiving",
  "Multi-machine and cloud scope", "Acceptance")
- `qratum/AGENTS.md` (fast-hook rule, supply-chain rule, Definition of Done,
  Ductum Factory Rules)
- `qratum/docs/supply-chain.md`
- `qratum/Makefile` (build/test/verify/demo targets)
- `qratum/cmd/qrt/hook.go`, `qratum/cmd/qrt/daemon.go` (current pointer-only
  capture; storage is repo-local `./.qratum`, `internal/` is empty scaffolding)
- `qratum/cmd/qrt/sessions.go`, `status`/`report` commands for CLI patterns
- `qratum/schemas/` and `qratum/fixtures/` (test contract)
- `qratum/docs/reviews/2026-06-12-memory-architecture/BACKLOG.md` section B

## Allowed scope

- New vault code under `cmd/qrt` and (per AGENTS.md DDD direction) real
  packages under `internal/` as needed for vault, capture, and raw-ref.
- Migrate capture storage from repo-local `./.qratum` to a central
  `~/.qratum` workspace, with a `QRATUM_HOME` env override for tests/fixtures.
- Extend the capture hook with copy-on-capture.
- New commands: `qrt hook install`, `qrt hook status`, `qrt vault doctor`,
  `qrt vault backfill`, `qrt vault archive`, `qrt vault backup`; extend
  `qrt status`.
- New/extended schema `qratum.raw_ref.v1` + fixtures + golden tests.
- A second-machine runbook doc.

## Non-goals

- No SQLite, no new third-party Go dependency (vault is stdlib-only; adding a
  dependency is an explicit supply-chain decision and out of scope here).
- No normalizer/refinery changes, no lessons, no insights, no search.
- No `claude-ai-export` adapter, no curation queue, nothing on the "Dead" list.
- No network calls, no LLM calls, no full transcript parsing in the hook.
- Do NOT run install/backfill/archive against the real `~/.claude` or
  `~/.qratum` — those are Arnold-only manual steps.

## Implementation notes

### Workspace
- Central workspace `~/.qratum/`; `QRATUM_HOME` overrides the root (tests and
  `make demo` set it to a temp dir). Vault-minimum layout only:
  `raw/blobs/sha256/<ab>/<digest>...`, `raw/refs/raw_<digest12>.json`,
  `events/`, `state/vault.json`.

### Copy-on-capture (hook stays fast)
- Hook reads hook JSON from stdin, writes one capture event, and copies the
  file at `transcript_path` into the blob store: stream sha256, skip if the
  blob already exists, write via tmp+rename. No parsing, no network, no LLM.
- Degraded cases recorded, never swallowed: missing `transcript_path` → event
  with `raw_missing: true`; copy failure → event records the failure, surfaced
  by `qrt status` / `qrt vault doctor`.
- `qratum.raw_ref.v1` records the blob: digest, kind, original path, size,
  observed-at, `local_only: true`. Raw kinds include the existing set plus
  `source_export_bundle`, `source_memory`, `vendor_memory_dir`,
  `vendor_insight_report`, `memory_import_receipt`.

### Operational ownership
- `qrt hook install` — idempotently add the SessionEnd hook to the GLOBAL
  `~/.claude/settings.json`; print the exact diff and confirm before writing;
  detect and report an existing project-local hook to avoid double-capture.
- `qrt hook status` — report whether the global hook is installed.
- `qrt vault doctor` — answer "is preservation working now": global hook
  installed?, last capture time, last backfill time + staleness, copy-failure
  count, blob-vs-known-transcript drift, backup freshness; and explicitly
  STATE the cloud-session limitation (sessions that start+end on vendor infra
  are not captured in v1).
- `qrt vault backup [--verify] <dest>` — copy `~/.qratum`; `--verify` proves
  restorability (sample digest-check, or full round-trip for small vaults).

### Backfill and archiving
- `qrt vault backfill` — idempotent inventory of
  `~/.claude/projects/**/*.jsonl` (and subagent transcripts) into blobs; dedup
  by digest; re-runnable (second run no-op); intended to run periodically.
- `qrt vault archive <path> [--kind K]` — archive files/folders into blobs
  with a kind tag (the new kinds above + existing `source_metadata`).
- `qrt status` — add vault counts, last backfill, copy failures.

### Multi-machine / cloud
- Write a short second-machine runbook: each machine runs its own
  `qrt hook install` + `qrt vault backfill`; vaults merge blob-dedup-clean
  (content-addressed). State that cloud-only sessions are out of v1 scope.

### Tests
- Golden/fixture tests (fixture-driven per AGENTS.md): copy-on-capture
  (blob+ref+event written, idempotent skip), backfill idempotency (second run
  no-op), archive kinds, hook-install idempotency (diff shown, no double-write
  into a fixture settings file), degraded cases (raw_missing, copy failure).
- All tests must honor `QRATUM_HOME` and never touch the real home dir.

## Acceptance criteria

(from `qratum-vault-first.md` → "Acceptance")
- A captured session writes blob + ref + event within seconds of session end.
- `qrt hook install` is idempotent and shows its diff before writing.
- `qrt vault backfill` run twice → second run is a no-op (digest dedup).
- Deleting the source transcript does not lose data (blob survives).
- `qrt vault doctor` warns on: no global hook, stale backfill, copy failures,
  unverified/missing backup; and states the cloud-session limitation.
- `qrt vault backup --verify` proves restorability, not just copy success.
- `qrt status` shows vault counts, last backfill, copy failures.
- No raw content in logs or events (paths and digests only).
- `make verify` and `make demo` are green.

## Verification commands

```sh
# Full local CI mirror (build, vet, lint, test, race, demo, dogfood, security):
make -C /Users/acartagena/project/qratum verify

# Milestone-A vertical slice still works:
make -C /Users/acartagena/project/qratum demo

# End-to-end vault proof in an isolated workspace (no real home touched):
export QRATUM_HOME="$(mktemp -d)"
make -C /Users/acartagena/project/qratum build
cat /Users/acartagena/project/qratum/fixtures/claude-code/hook-session-end.json \
  | /Users/acartagena/project/qratum/bin/qrt hook claude-code
/Users/acartagena/project/qratum/bin/qrt vault doctor
# prove a captured blob survives deletion of its source transcript, then:
/Users/acartagena/project/qratum/bin/qrt status
unset QRATUM_HOME
```

VERIFY GAP: confirm the exact fixture path used for the end-to-end proof.
`fixtures/claude-code/hook-session-end.json` exists today, but its
`transcript_path` may point at a fixture that is not a standalone copy target.
Before dispatch, confirm which fixture provides a real transcript file the hook
can copy (candidate:
`fixtures/dogfood/real-shaped-transcript.jsonl` used by `make dogfood-demo`).

## Review prompt

> Review this vault implementation against `qratum/specs/current/qratum-vault-first.md`.
> Confirm: the hook still obeys the fast-hook rule (no parse/network/LLM, only
> stdin→event→file-copy); storage moved to `~/.qratum` with a working
> `QRATUM_HOME` override; copy-on-capture is content-addressed, idempotent, and
> records degraded cases; install/doctor/backfill/archive/backup --verify exist
> and behave per spec; doctor states the cloud-session limitation; no new
> third-party Go dependency was added; tests are fixture-driven and never touch
> the real home dir; `make verify` and `make demo` pass. Flag any "Dead"-list
> resurrection (LessonBackend, search, normalizer, curation queue) or any
> command that mutates the real `~/.claude`/`~/.qratum`.

## Stop conditions

- STOP if P1 has not landed (specs still contradictory) — this depends on P1.
- STOP if the Qratum milestone is still `P0-SPEC-AND-CONTRACTS` and Arnold has
  not explicitly unlocked the vault milestone — runtime build is gated.
- STOP if a vault feature appears to require a third-party Go dependency —
  report it as a supply-chain decision for Arnold rather than adding it.
- STOP before running any command against the real `~/.claude` or `~/.qratum`;
  those (global hook install, first backfill, export archive) are Arnold-only.
- STOP if `make verify` or `make demo` cannot be made green without weakening a
  check — report the failure, do not suppress it.

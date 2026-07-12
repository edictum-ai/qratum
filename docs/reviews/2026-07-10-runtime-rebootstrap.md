# Qratum Runtime Rebootstrap

Status: superseded restart plan

Date: 2026-07-10

Superseded: 2026-07-11 by `specs/current/product-direction.md`

This plan remains evidence about the audited runtime candidate and its defects.
Its find/preserve/prepare/browse product loop, no-raw reader, active R0-R4
slices, and release scope are no longer implementation authority. Do not
dispatch work from this plan without a new accepted tranche contract.

## Decision

Keep the accepted UI-first product direction and preserve the local runtime
candidate at `c9323a0`. Do not restart from the old public CLI and do not rebuild
the six runtime phases from scratch.

The next release is earned by making the candidate safe and usable end to end:

```txt
find -> preserve -> prepare latest 10 -> browse safely
```

The first usable release does not need SQLite, AI providers, lessons, insights,
publishing, or a polished analytics UI.

## Current Reality

| Layer | Commit | Actual state |
| --- | --- | --- |
| Published `v0.1.0` | `f7e21dc` | Vault-first runtime and the old pipeline-shaped CLI. `qrt init` and `qrt open` are not present. |
| Remote `main` | `e315cf3` | Published runtime plus the accepted UI-first contract; no UI-first runtime implementation. |
| Bootstrap branch | `c9323a0` | Local UI-first runtime candidate. It contains the prepare-from-vault bridge and the new CLI/API shell, but has release-blocking correctness and trust gaps. |
| Later product | not built | Polished browser UI, browser import/export execution, optional AI, SQLite/search, lessons/insights, publishing, and the live D10 gateway round-trip. |

The bootstrap branch is `feat/ui-first-runtime-bootstrap`. Local `main` is left
unchanged at `c9323a0`; it must not be rewritten until the runtime line is
preserved in a reviewed remote PR.

## Release Blockers

Real data must not be used for dogfood until these invariants are closed:

1. The hook must not trust payload `cwd` as a capture root.
2. An erasure tombstone must prevent later capture/import/backfill from restoring
   the erased blob.
3. Preparation must recover cleanly from partial writes.
4. Import must recognize accepted transcript structures, not arbitrary JSONL.
5. `qrt sessions` and `qrt session` must expose preserved-raw queue metadata and
   safe next actions.
6. Background preservation must invoke a supported command and must be genuinely
   installed/enabled.
7. Browser mutations must enforce server-side confirmation and same-origin/CSRF
   checks.
8. Status, capture, processing, job, and trust views must report live state.
9. Trust dimensions must execute their proof instead of being declared green.
10. Installed `qrt trust` must work without a source checkout.

## Ordered Development Slices

### R0 — Truthful baseline and gates

- Correct shipped-versus-candidate wording in `AGENTS.md`, `SPEC.md`, `README.md`,
  and the two Ductum package READMEs.
- Historical instruction, now superseded: keep
  `specs/current/ui-first-onboarding.md` authoritative for the product flow.
  Current authority is `specs/current/product-direction.md`.
- Make `.edictum/workflow-profile.yaml`, CI, and release run the authoritative
  verifier.
- Change unexecuted trust dimensions from green to blocking or execute them.
- Require an exact supported Go patch release before verification.

Exit: the repository cannot call the candidate shipped or report trust without
running the named proof.

### R1 — Data-safety invariants

- Fix hook-root confinement and add forged-`cwd` negative tests.
- Make tombstones terminal across hook, init, import, and backfill siblings.
- Make prepared-artifact publication atomic and retryable.
- Preserve owner-only permissions when editing Claude settings.

Exit: targeted security tests, full tests, race tests, and the trust dimensions
for capture, vault integrity, erasure, permissions, and recovery pass.

### R2 — Usable local CLI loop

- Validate supported import formats fail closed.
- List prepared sessions first and preserved-raw sessions second.
- Make `qrt session <id>` show safe metadata and a prepare action for raw entries.
- Repair source-hook/OS-schedule preservation and report its real state.
- Make status/doctor counts source-aware and failure-aware.

Exit: an isolated user can import or discover a session, preserve it, see it in
the queue, prepare it, and inspect safe output without internal commands.

### R3 — Minimal local app

- Replace the API-instruction shell with status, raw queue, prepared sessions,
  session detail, and prepare actions.
- Enforce confirmation plus same-origin/CSRF checks for every mutation.
- Persist real queued/running/succeeded/failed job state.
- Add working pagination and artifact routes.

Exit: `qrt open` completes the same safe loop as the CLI and never exposes raw
transcript text.

### R4 — Release and dogfood

- Make `make verify` green on the supported toolchain.
- Run the installed binary and trust command from outside the source checkout.
- Run the black-box flow against a dedicated isolated `QRATUM_HOME`.
- Open a reviewed PR, then cut `v0.2.0-rc1`.
- Use the release candidate daily before cutting `v0.2.0`.

Exit: the published artifact, remote `main`, documentation, and proof surfaces
all describe and execute the same product.

## Developer Bootstrap

```sh
git switch feat/ui-first-runtime-bootstrap
go version
GOTOOLCHAIN=local GOFLAGS=-mod=readonly go mod verify
make test
make demo
```

The current local Go 1.26.1 toolchain is below the required security patch
level, so `make verify` is expected to remain red until the toolchain is updated
and the trust gate is made truthful. A passing `make test` or `make trust` alone
is not release evidence.

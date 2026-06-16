# Qratum Verify & Trust Gate — Ductum Spec Package

Importable spec package for the Qratum side of the 2026-06-15
verification-benchmark workstream. It carves the `P2-VERIFY-TRUST-GATE`
milestone into gated, dispatchable phases. Source of truth for intent:

- `qratum/specs/current/verification-and-trust-gate.md` (the **contract** these
  tasks implement — the design source-of-truth, v2 2026-06-15)
- `../../GAPS.md` (the adversarial completeness gap review: M1–M11, Q1–Q4,
  out-of-scope items)
- `../../BENCHMARK.md` (the benchmark design: per-dimension acceptance detail and
  file:line evidence)

In one sentence: today "verified" only means the build is green, but a green
build was caught hiding real credential leaks — so this milestone fixes the
confirmed leaks and stands up a `make trust` gate that actually runs secrets
through the pipeline and proves nothing leaks.

Scope: this package touches **only the `qratum` repo**, with one stated
cross-repo dependency — P7 (the dream / curation tier, D10) consumes
`memory_import_receipt` blobs produced by the **personal-memory gateway** in
another repo. P7's contract-and-schema work lands here now; its behavioral
round-trip leak proof is gated on that gateway being deployed (it is the producer;
qratum is only the consumer). Do not build the gateway producer inside qratum, and
do not touch `edictum` or `edictum-harness`.

## Execution Order

| # | Prompt | Package | Scope | Deliverable | Status | Depends On |
|---|--------|---------|-------|-------------|--------|------------|
| 1 | [P1-SECURITY-FIXES.md](P1-SECURITY-FIXES.md) | qratum | runtime/security | Contained security fixes (FIX-1/2/5/6/8/11/12) + central-home placement (FIX-4, D11) + at-rest perms (D14) + capture import-isolation extraction + locking tests | [ ] | — |
| 2 | [P2-REDACTION-CROWN-JEWEL.md](P2-REDACTION-CROWN-JEWEL.md) | qratum | trust/redaction | D3 reflection-canary harness + encoding-aware no-leak checker + FIX-10 (D3/D4) | [ ] | P1 |
| 3 | [P3-SCHEMA-CONTRACTS.md](P3-SCHEMA-CONTRACTS.md) | qratum | schemas/contracts | Recursive `additionalProperties:false` + mini-validator + `data_class` (FIX-13) + config + `trust_scorecard.v1` (D9) | [ ] | P1 |
| 4 | [P4-VAULT-INTEGRITY-LIFECYCLE.md](P4-VAULT-INTEGRITY-LIFECYCLE.md) | qratum | runtime/vault | Golden-leak lint (D5), source-scope guard (D13), backup consent+streaming (D8), recoverability + integrity proofs + `gc`+erasure (FIX-3/14/15/16, D2/D6/D6a, FIX-3 KNOWN-RED) | [ ] | P1 |
| 5 | [P5-LIVENESS-SCHEDULER.md](P5-LIVENESS-SCHEDULER.md) | qratum | runtime/ops | Doctor truthfulness (D7) + `qrt vault install-schedule` + OS-timer test plan (D12) | [ ] | P1 |
| 6 | [P6-TRUST-GATE-AND-CI.md](P6-TRUST-GATE-AND-CI.md) | qratum | trust/CI | `make trust` skeleton + three-state gate + scorecard JSON + `qrt trust` + wire into `make verify`/CI + CLI reconciliation | [ ] | P2, P3, P4, P5 |
| 7 | [P7-CURATION-TIER.md](P7-CURATION-TIER.md) | qratum | contracts/import | `memory_import_receipt` schema + `--kind` archive round-trip (D10); behavioral round-trip gated | [ ] | P3 + personal-memory gateway deployed |

Dependency graph in plain terms: P1 stands alone (the cheapest, highest-value
contained fixes). P2/P3/P4/P5 each build on P1 and are otherwise independent of
each other (they can be dispatched in parallel). P6 is the integration phase — it
needs the dimensions from P2/P3/P4/P5 to exist before it can wire them into a
scorecard and CI. P7 needs P3's schema machinery (the `memory_import_receipt`
schema is wired into D9) and is additionally **gated on the personal-memory
gateway being deployed** before its round-trip leak proof means anything.

## Gate before dispatching

1. **Milestone unlock** — Qratum's current milestone is `P1-VAULT-FIRST` shipped;
   `P2-VERIFY-TRUST-GATE` is **PROPOSED, awaiting Arnold's acceptance**
   (`AGENTS.md` → Current Milestone). Every task in this package builds P2 runtime
   and therefore **STOPS at its first stop condition** unless Arnold has explicitly
   unlocked `P2-VERIFY-TRUST-GATE`. A dispatched agent must not implement
   P2-or-later runtime behavior on its own.
2. **Spec accepted** — `qratum/specs/current/verification-and-trust-gate.md` still
   reads `Status: Proposed — v2, 2026-06-15 (awaiting Arnold)`. The Status line
   must be flipped to "Accepted (date)" before any phase past the gate runs.
3. **Clean inputs** — the spec/review files each prompt reads must be committed
   (`git status --short` clean for those paths) so a task builds against a fixed
   contract, not a dirty working tree.
4. **P7 producer gate** — P7's behavioral round-trip additionally requires the
   personal-memory gateway (Phase 1+) to be deployed. Until then P7 ships only the
   committed schema + the pinned-`--kind` archive path; its synthetic no-leak check
   is labeled "not-yet-meaningful as a leak proof," not faked green.

## Manual, Arnold-only steps (never automated by a dispatched agent)

- Running `qrt vault install-schedule` against the **real** home (installs an OS
  timer into `~/Library/LaunchAgents` / the systemd user dir). Agents build, test
  (against a fake `t.TempDir()` schedule dir / dry-run print), and verify only.
- Running `qrt trust` (or `make trust`) against the **real** `~/.qratum`. Agents
  run it only inside an isolated `QRATUM_HOME` temp workspace.
- Installing or modifying the global `~/.claude/settings.json` hook.

These mutate the user's real home directory and live history. A dispatched agent
builds and tests the commands; it must **never** run them against the real
`~/.claude` or `~/.qratum`. Tests and CI set `QRATUM_HOME` to a temp dir.

## Decision Trace

Decisions taken by Arnold on 2026-06-15 (folded into the spec above; see its §5):

- **Spec everything in one place** — benchmark + confirmed defects together.
- **Scope = BOTH (insurance AND dream)** — reverses the earlier insurance-only
  call (GAPS Q3). The insurance lane (D1–D9, D11–D14) stays required; the dream /
  curation tier (D10) is **in scope as its own gated phase (P7)**, not flag-hidden.
- **Scheduler ships (Q1)** — `qrt vault install-schedule` (an OS launchd/systemd
  timer running `qrt vault backfill`), not a resident daemon; ships with an
  explicit OS-timer test plan (P5/D12).
- **Artifact placement (FIX-4, Q2) = central home** — all derived artifacts under
  `~/.qratum/sessions/<session_id>/` via `QRATUM_HOME`; no repo-local `./.qratum/`
  writes (P4/D11).
- **PII / third-party content (M7) = DEFER** — the in-binary redactor stays
  Go-native and **credentials-only**; third-party/PII content is preserved
  **verbatim**; Presidio was considered and **rejected** (Python vs. qratum's
  no-Python single-Go-binary constraint), and no Go PII pass ships in v1.
- **Two newly found leaks spec-now / fix-with-P2** — world-readable raw blobs
  (M4 → FIX-8/D14) and ungoverned raw backup egress (M5 → FIX-9/D8).
- **Preservation default (M10) = never auto-delete / "nothing lost"** — the only
  removal path is the explicit, recorded, tombstone-based erasure verb (FIX-16).
- **QRATUM_HOME isolation** — tests/CI never touch the real `~/.claude` or
  `~/.qratum`; everything runs against a temp `QRATUM_HOME`.
- **Scorecard surface (Q4) = `qrt trust` command + `qratum.trust_scorecard.v1`
  schema** wired into D9, with a provenance block and a self-leak-scan; the public
  qratum.dev badge is deferred.
- **Threat Model added** — the gap review's #1 finding (no stated attacker model).
- **Canary format fixed (M1)** — the reflection canary must provably evade all 8
  redaction classes by construction (lowercase-alpha, `<32` chars, single class,
  no separator); a UUID-v4 is forbidden (it self-redacts); the UUID stays as the
  precision tripwire.
- **CI paradox resolved (M11)** — three gate states: BLOCKING-RED, KNOWN-RED
  (CI-non-blocking, monotonic, owner+deadline), GREEN.

## Behavior Contract

- [ ] FAILS the task: any phase implements P2 runtime while
  `P2-VERIFY-TRUST-GATE` is still PROPOSED (not unlocked by Arnold).
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency
  (the trust harness is **stdlib-only**); evidence: `make verify` / `make trust`.
- [ ] FAILS if the hook parses/networks/calls an LLM or emits raw at runtime
  (fast-hook rule); evidence: D1/D3 golden tests.
- [ ] FAILS on real-home mutation in tests/CI; evidence: every test runs under a
  temp `QRATUM_HOME` and never touches `~/.claude`/`~/.qratum`.
- [ ] FAILS review: any non-goal feature smuggled in (SQLite, resident daemon,
  PII detection, new product surface, a producer/gateway built inside qratum,
  a tunable precision budget that absorbs known over-redaction).
- [ ] The scorecard never averages dimensions and never narrows itself to
  whatever currently passes; a KNOWN-RED count may only decrease (monotonic).

## Verification

- Per-phase: each task carries its own concrete `make`/`sh` commands plus the
  isolated-`QRATUM_HOME` end-to-end proof. See each task's Verification.
- Package-level: `make -C /Users/acartagena/project/qratum verify` (which after
  P6 includes `trust`) is green; CI fails on any BLOCKING-RED gate (a regression
  of a green gate, or a KNOWN-RED past its deadline) and uploads the scorecard JSON.
- `go test -race ./...` is clean (the FIX-6 TOCTOU and concurrency tests).
- Every emitted artifact validates against its schema with
  `additionalProperties:false`; the reflection-canary harness self-test proves the
  gate can both pass and fail.

## Drift Handling

- If `verification-and-trust-gate.md` changed since 2026-06-15 v2 and no longer
  maps to a phase, stop and report; re-confirm the file:line evidence in
  `BENCHMARK.md`/`GAPS.md` before building (the shipped code may have moved).
- If a confirmed defect no longer reproduces against current `main`, stop and
  report rather than writing a test that locks shut a bug that is already fixed.
- Update fixtures/golden only when an output contract intentionally changes, and
  say so (e.g. re-redacting the committed golden in FIX-2 is an intended change).

## Slop Review

- [ ] Behavior contract holds: no non-goal feature added; no producer/gateway
  built inside qratum; redactor stays credentials-only and Go-native.
- [ ] The canary is non-self-redacting (provably evades all 8 classes); the
  harness self-test (known-positive RED + canary-survives-`redactString` +
  panic-on-unreachable-field) is present and passing — the gate can fail.
- [ ] No-leak checker is **encoding-aware** (HTML-unescaped + JSON-unmarshaled),
  driven by the reflection token set, and a dropped field is not counted as a
  redacted field (R2).
- [ ] Loud failure on missing/invalid inputs (missing transcript, copy failure,
  disk-full, malformed `Input`/`Provenance`): recorded and surfaced, never
  swallowed; behavioral tests cover it.
- [ ] No real-home mutation in tests/CI (`QRATUM_HOME` set) and no new
  third-party Go dependency without supply-chain evidence.
- [ ] `make verify` (and, after P6, `make trust`) pass without weakening a check;
  KNOWN-RED entries each carry a tracking note, owner, and deadline.

# P6 — Trust Gate Harness, Scorecard, CI Wiring & Public-CLI Reconciliation

Package: `qratum-verify-trust-gate` · Prompt 6 of 7 · Depends on: P2, P3, P4, P5 ·
Scope: runtime/harness + CI + public-CLI · Deliverable: the `make trust`
harness, the `qratum.trust_scorecard.v1` scorecard + `qrt trust` command, the
three-state gate model wired into `make verify` and CI, and the two public-CLI
reconciliations.

## Objective

Stand up the thing the whole milestone exists to produce: a real `make trust`
gate that runs the actual shipped CLI over a known corpus and reports — honestly
— whether anything leaked or broke. It does not invent new checks; it harnesses
the dimensions built in P2–P5 (capture/hook safety, the redaction crown jewel,
boundary enforcement, vault integrity, schema conformance, the lifecycle verbs,
liveness) into one Go-native, stdlib-only runner. That runner emits a
machine-readable `qratum.trust_scorecard.v1` JSON plus a human summary, classifies
every gate into one of three states (GREEN / KNOWN-RED / BLOCKING-RED), and gets
wired into `make verify` and CI so a regression or an overdue known failure blocks
the merge — but a deliberately-red planned gate does not merge-lock the repo.

Then close the two shipped-vs-spec mismatches the milestone signed up to fix: the
`qrt` no-args behavior, and the README overstatement.

This is the assembly phase. Build only the harness, the scorecard, the gate
states, the CI wiring, and the two reconciliations. Do not add or change the
underlying dimension behavior — that is P2–P5. If a dimension is missing or
behaves differently than its phase spec says, STOP and report; do not paper over
it inside the harness.

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` — the contract. Read in
  full, but especially: §3 "The trust gate" (the dimension list, the scorecard &
  gate subsection, the three gate states M11, the anti-gaming monotonic recall
  split, the scorecard-as-governed-object rule, the honest-residual block, and
  the "what a green TRUSTED does NOT promise" list); §4 "Sequencing" step 5 and
  step 6; §5 the public-CLI reconciliation item; §7 "Acceptance".
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` — M11 (the CI
  paradox / three gate states), the anti-gaming framing, Q4 (scorecard surface =
  `qrt trust` + `qratum.trust_scorecard.v1` + provenance block), and the
  "Public-CLI vision vs. shipped-CLI reconciliation" item under "Explicitly out
  of scope" (the no-args exit-2 behavior and the README "across vendors / searches
  every session" overstatement).
- `qratum/AGENTS.md` — Current Milestone (P2 is PROPOSED, awaiting the maintainer),
  Supply-Chain Rule, Testing (fixture/golden), Data Rule, "Still not built".
- `qratum/docs/supply-chain.md` — what may and may not enter the runtime/CI.
- `qratum/Makefile` — `verify` is `supply-chain vet lint test test-race build demo
  dogfood-demo security`; `trust` must slot into this chain. Note the existing
  `QRATUM_HOME="$(mktemp -d ...)"` pattern in `dogfood-demo`.
- `qratum/.github/workflows/ci.yml` — the existing job runs each `make` target as
  a step; `trust` needs its own step plus a scorecard-artifact upload. Actions are
  pinned by SHA (supply-chain rule).
- `qratum/cmd/qrt/main.go:25-30` — `runWithIO` prints usage + `error: missing
  command` and returns exit 2 when `len(args) == 0`. This is the no-args behavior
  to reconcile.
- `qratum/README.md:21` — "captures, preserves, normalizes, redacts, reviews, and
  **searches every session**" — the overstatement to fix.
- The P2–P5 task specs in this package — for the exact entrypoints, scorecard
  field names, and KNOWN-RED entries each phase declares. The harness consumes
  their outputs; it must not duplicate or re-implement them.

## Allowed scope

- A new Go-native trust runner: `cmd/trustbench` (or a `//go:build trust`-tagged
  target) plus its supporting package under `internal/` if a shared
  scorecard/gate-state type is warranted. Stdlib-only.
- A new `make trust` target, and editing `make verify` to include it.
- A new `qrt trust` subcommand wired into `runWithIO` (registers in the `switch`,
  prints the scorecard's human summary, can emit JSON via a flag).
- The `qratum.trust_scorecard.v1` JSON Schema under `schemas/`, wired into the D9
  validator and self-test (the scorecard is a governed, leak-scanned object).
- The `.github/workflows/ci.yml` step for `make trust` + scorecard-artifact
  upload.
- The two public-CLI reconciliations: the `qrt` no-args behavior in `main.go`, and
  the README line.
- Fixtures/golden for the scorecard shape, the three gate states, the provenance
  block, and the self-leak-scan.

## Non-goals

- No new dimension behavior. The harness drives P2–P5; it does not add capture,
  redaction, vault, schema, or liveness logic. If a dimension is absent, STOP.
- No new third-party Go dependency — the runner is stdlib-only. A dependency is an
  explicit supply-chain decision = STOP-and-report.
- No averaging / no score-out-of-100. The headline is an enum; tiers are hard.
- No public qratum.dev badge, no signing/freshness service — Q4 deferred those.
- No SQLite, no resident daemon, no new product surface (the harness reports those
  RED/residual; it must not smuggle them in).
- No real secrets or real internal identifiers committed in any scorecard fixture.
- Do NOT run `make trust`, `qrt trust`, or any command against the real `~/.claude`
  or `~/.qratum`. All runs use `QRATUM_HOME` pointed at a temp dir.

## Implementation notes

### The `make trust` harness (R3 — drive the REAL CLI)
- Go-native, fixture-driven, stdlib-only. It exercises the **real shipped
  entrypoints**, not shortcuts: the daemon (`runDaemonOnce`, `runWithIO`) **and**
  the standalone `evidence` / `review` / `report` / `export` subcommands (R3).
  Build the binary, then drive it; or invoke the in-process entrypoints directly —
  but the artifacts under test must be the ones a user actually gets.
- It assembles results from the testable-now dimensions delivered by P2–P5
  (D1, D2, D3, D4, D5, D6, D6a, D7, D8, D9, D11, D12, D13, D14, and the D10
  contract-and-schema-now portion). It does **not** re-derive their pass/fail
  logic; each dimension exposes a result the harness collects.
- Every run sets `QRATUM_HOME` to an isolated temp dir. The harness never touches
  the real home. This is a hard rule, mirrored from `dogfood-demo`.

### Three-state gate model (M11 — resolves the CI paradox)
Each gate is exactly one of:
- **GREEN** — passing.
- **KNOWN-RED** — RED **by design** until its in-milestone fix lands (e.g. D6a /
  FIX-3 recoverability, the D9 schema-drift work, the not-yet-fixed leak classes).
  Tracked, **CI-non-blocking**, and **monotonic**: the KNOWN-RED set may only
  shrink, never grow. Each KNOWN-RED entry MUST carry a `note`, an `owner`, and a
  `deadline`. CI fails only if a KNOWN-RED entry is **past its deadline**, or if
  the KNOWN-RED count **increases** versus the committed baseline.
- **BLOCKING-RED** — a regression of a currently-GREEN gate. Fails CI hard.

The headline enum (no abbreviation tricks):
- `TRUSTED` — all security+integrity gates GREEN **and** no known-miss class in the
  corpus **and** recoverability wired.
- `TRUSTED-WITH-NAMED-GAPS` — gates GREEN but a known-miss class is present or
  recovery is unwired. **MUST NOT** be abbreviated to `TRUSTED`; carries a non-zero
  `gap_count`.
- `NOT-TRUSTED` — ≥1 BLOCKING-RED gate.

### Anti-gaming — the monotonic recall split
The D3 recall metric splits into (a) a **hard** "no leak in the covered corpus"
blocker (any covered-corpus leak → BLOCKING-RED), and (b) a tracked,
**CI-non-blocking, monotonic** extended-class recall. The harness fails CI if the
extended corpus has **fewer than the documented N classes** or if the extended
recall % **regresses**. Corpus shrinkage must surface as a visible, reviewable
regression — never as a silent omission. The baseline N and recall % are committed
(a golden or a checked-in baseline file the harness compares against).

### The scorecard — `qratum.trust_scorecard.v1`
- Define the JSON Schema under `schemas/`, with `additionalProperties:false`
  recursively, and wire it into the D9 validator and the D9 self-test (the
  scorecard is one of the "every emitted object maps to a schema" objects).
- Shape (names pinned against the schema): the `headline` enum; per-dimension
  results with `state` (GREEN / KNOWN-RED / BLOCKING-RED); `gap_count`; the
  KNOWN-RED list with `note` / `owner` / `deadline`; the extended-recall block
  (N classes + recall %); a **provenance block** (`build_commit`, `corpus_digest`,
  `schema_digest`, `timestamp`) so a green score is verifiable as "this score,
  from this code, over this corpus"; and the honest-residual block (below) as a
  carried, verbatim string array.
- **The scorecard is a governed object.** After emitting it, run the **shared
  no-leak checker** (the encoding-aware D4 checker, driven by the reflection token
  set) over the scorecard's **own bytes** and assert zero canary survival. A
  scorecard that itself leaked would be a perverse failure; prove it cannot.
- Determinism: the scorecard must be byte-stable for a fixed input/corpus except
  for the provenance `timestamp` (and `build_commit` in a dirty tree). The golden
  test pins everything but those, or injects a fixed clock/commit.

### The honest-residual block (printed verbatim under the headline)
The scorecard MUST carry — and `qrt trust` MUST print verbatim — the residual
block stating, in plain words, exactly what the gate does NOT cover. It MUST
include all of:
- **PII / third-party content is NOT redacted** — the redactor is **credentials
  only** and Go-native; third-party / PII content is preserved **verbatim**; PII
  detection is **explicitly deferred future work (M7)** (Presidio was considered
  and rejected for qratum's no-Python single-Go-binary constraint; no Go PII pass
  ships in v1).
- **At-rest disk encryption is out of scope** — the gate enforces file permissions
  (D14), not encryption.
- **The audit / event log is not tamper-evident** (no hash-chain / signatures).
- **Cloud / web sessions are uncaptured** by design (sessions that start and end on
  vendor infra are not captured in v1).
- **Cross-vault merge drops per-machine state / event cursors** — only blobs are
  dedup-clean; merge is otherwise UNVERIFIED.
- **`vault backup` of raw is the sanctioned, consent-gated exception** to "raw
  never leaves the machine."
- Plus the spec's other named residuals: extended-class recall % + the 8 enumerated
  regex classes; the unicode / `/`-exclusion / 32-char-floor limits; "redaction is a
  single upstream pass — the artifact checks are correlated, not independent
  layers"; recoverability and artifact-placement status; `transcript_drift` is a
  heuristic; config-schema status; D10 / the dream tier is an in-scope gated phase
  whose round-trip leak proof is meaningless until the personal-memory gateway is
  deployed; and the preservation default is "nothing lost" — nothing is ever
  auto-deleted, the only removal path is the tombstone-based erasure verb (FIX-16),
  alongside `qrt vault gc` (FIX-15) which refuses referenced blobs.

Reuse the residual text from §3 of the trust-gate spec verbatim; do not paraphrase
it loosely. If the residuals shipped by P2–P5 differ from this list, STOP and
report the divergence rather than silently editing the list.

### `make verify` + CI wiring
- Add `trust` to the `make verify` chain (it currently ends `... security`; the
  spec phrases it as `verify: … security trust`). Keep all existing targets.
- Add a `ci.yml` step that runs `make trust` and **uploads the scorecard JSON as a
  build artifact**. CI **fails** the job on any **BLOCKING-RED** gate, or on a
  **KNOWN-RED past its deadline**, or on a KNOWN-RED count increase, or on
  extended-corpus shrinkage / recall regression. CI does **not** fail merely
  because a KNOWN-RED gate is RED. Pin the upload action by commit SHA.
- Do not weaken any existing check to get green ("CI is sacred"). If `make trust`
  cannot be made to pass under the three-state model without weakening something,
  STOP and report.

### Public-CLI reconciliation — `qrt` no-args
- Today `runWithIO` with no args prints usage + `error: missing command` and
  returns exit 2 (`main.go:25-30`).
- **DECISION POINT (present to the maintainer; do not pick silently).** Two options:
  - **(A) default status view** — no-args runs the status/doctor view and exits 0
    (the spec's recommended direction). Implication: `qrt` becomes safe to run with
    no args; scripts that relied on exit 2 for "no command" change behavior.
  - **(B) keep the error** — no-args stays `error: missing command`, exit 2; the
    spec's "dashboard" framing is dropped as not-shipped.
  Per the standing rule that design/contract decisions go through the maintainer, surface
  both with this trade-off and let the maintainer pin it. Then implement the pinned choice,
  and make `qrt trust` / `qrt status` consistent with it (e.g. if (A), the default
  view and `qrt status` should not contradict each other). Update `main_test.go`
  (which currently asserts the exit-2 message) to match the pinned behavior.

### Public-CLI reconciliation — README overstatement
- `README.md:21` claims qratum "captures, preserves, normalizes, redacts, reviews,
  and **searches every session** — **without ever uploading your raw
  transcripts.**" This overstates shipped reality: **no search ships**, and
  **capture + refine are Claude-Code-only** (D13).
- Fix the line to describe shipped reality: drop the "searches every session"
  claim, and state plainly that **capture and refine are Claude-Code-only** (other
  sources are archive-only with no redaction path). Do not introduce new product
  claims; reconcile down to what ships. Keep the "without ever uploading your raw
  transcripts" promise (that one is true), but note backup-of-raw is the
  consent-gated exception if the surrounding copy implies an absolute.

### Tests
- Golden/fixture tests, honoring `QRATUM_HOME`, never touching the real home:
  - scorecard shape matches `qratum.trust_scorecard.v1` (validator accepts it,
    rejects an injected extra key);
  - the no-leak checker over the scorecard's own bytes finds zero canary survival;
  - the three gate states classify correctly: a GREEN baseline, a synthetic
    regression of a green gate → BLOCKING-RED → headline `NOT-TRUSTED`; a KNOWN-RED
    entry with a future deadline → CI-pass; the same entry past deadline → CI-fail;
    a KNOWN-RED count increase → CI-fail;
  - the anti-gaming split: a shrunk extended corpus → CI-fail; an extended-recall
    regression → CI-fail;
  - the honest-residual block is present and byte-equal to the committed text
    (a hard literal gate — like the doctor cloud-blind-spot line);
  - provenance block fields are populated and the scorecard is byte-stable modulo
    `timestamp` / `build_commit`;
  - the no-args reconciliation: assert the pinned behavior (exit code + output);
  - a README lint or test asserting the "searches every session" string is gone and
    the Claude-Code-only statement is present (a cheap standing tripwire).

## Acceptance criteria

(from `verification-and-trust-gate.md` §7 "Acceptance" + §3 scorecard subsection)
- `make trust` exists, runs the testable-now dimensions against **every**
  artifact-producing CLI entrypoint (daemon + standalone subcommands), and emits the
  `qratum.trust_scorecard.v1` JSON + a human summary using the three-state gate
  model.
- The headline is the enum; no averaging; `TRUSTED-WITH-NAMED-GAPS` is never
  abbreviated to `TRUSTED` and carries a non-zero `gap_count`.
- The three gate states behave per M11: BLOCKING-RED fails CI; KNOWN-RED is
  CI-non-blocking, monotonic, and each entry carries note + owner + deadline; CI
  fails on an over-deadline KNOWN-RED, a KNOWN-RED count increase, or a green-gate
  regression.
- The anti-gaming recall split is enforced: covered-corpus leak → BLOCKING-RED;
  extended-corpus shrinkage or recall regression → CI-fail.
- The scorecard carries the provenance block (build commit, corpus digest, schema
  digest, timestamp); the no-leak checker runs over the scorecard's own bytes and
  finds nothing; the scorecard schema is wired into D9.
- The honest-residual block prints **verbatim** under the headline and includes
  every required residual above (credentials-only / PII deferred; at-rest
  encryption out of scope; audit log not tamper-evident; cloud sessions uncaptured;
  cross-vault cursor loss; backup-of-raw consent-gated exception; plus the spec's
  other named residuals).
- `make verify` includes `trust`; CI runs `make trust` and uploads the scorecard
  artifact; no existing check was weakened.
- Public-CLI reconciliation done: `qrt` with no args has a **pinned** behavior
  (status view vs. error, chosen by the maintainer) and is implemented and tested; the
  README no longer overstates "searches every session" and states capture + refine
  are Claude-Code-only.
- No new third-party Go dependency; the harness is stdlib-only.
- No real secret or internal identifier in any committed scorecard fixture.
- `make verify` is green under the three-state model (KNOWN-RED gates allowed; no
  BLOCKING-RED, no over-deadline KNOWN-RED).

## Decision Trace

- Q4 resolved (2026-06-15): scorecard surface = `qrt trust` command +
  `qratum.trust_scorecard.v1` schema (wired into D9) + provenance block; public
  badge deferred.
- M11 resolved: three-state gate model (GREEN / KNOWN-RED / BLOCKING-RED) — the way
  `make trust` enters required CI without merge-locking and without weakening
  checks.
- Anti-gaming: D3 recall split into a hard covered-corpus blocker + a monotonic
  extended-class recall (corpus shrinkage is a visible regression).
- Public-CLI: the no-args behavior choice (status view vs. error) is a
  **contract decision that goes through the maintainer** — present both options, do not
  pick silently. The README overstatement fix is docs-reflect-reality (capture +
  refine are Claude-Code-only; no search ships).
- Runtime/CI/public-CLI build requires the P2-VERIFY-TRUST-GATE milestone to be
  explicitly unlocked by the maintainer (currently PROPOSED).

## Behavior Contract

- [ ] FAILS the task: the harness re-implements or alters any P2–P5 dimension's
  pass/fail logic instead of collecting its result.
- [ ] Runtime scorecard output must reject averages, score-out-of-100 summaries,
  and any abbreviation of `TRUSTED-WITH-NAMED-GAPS` to `TRUSTED`.
- [ ] FAILS: a KNOWN-RED entry without a note + owner + deadline, or a KNOWN-RED set
  that is allowed to grow, or a planned-RED gate wired as BLOCKING-RED (which would
  merge-lock the repo).
- [ ] CLI output must fail if the honest-residual block is missing any required
  residual, is paraphrased loosely, or is not printed verbatim.
- [ ] FAILS: the no-leak checker is not run over the scorecard's own bytes, or the
  scorecard schema is not wired into D9.
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency;
  evidence: `make verify` / `make supply-chain`.
- [ ] Verification must fail if any existing `make verify` / CI check is weakened
  to get green.
- [ ] FAILS on real-home mutation in tests/CI; evidence: `QRATUM_HOME` + temp dir.
- [ ] FAILS: picking the `qrt` no-args behavior without surfacing the options to
  the maintainer.

## Drift Handling

- If a P2–P5 dimension is missing, renamed, or behaves differently than its phase
  spec states, STOP and report — do not absorb the gap inside the harness.
- If the residual list shipped by P2–P5 diverges from the §3 honest-residual list,
  STOP and report; do not silently edit the printed block to match.
- If `cmd/qrt/main.go` or `README.md:21` has changed since this spec was written,
  re-read before editing and re-locate the exact lines (text matching is not
  semantic).
- Update golden/scorecard fixtures only when an output contract intentionally
  changes, and say so.

## Verification

```sh
# Full local CI mirror (now includes trust):
make -C . verify

# The trust gate alone, in an isolated workspace (no real home touched):
export QRATUM_HOME="$(mktemp -d)"
make -C . build
make -C . trust
# emit + inspect the scorecard JSON and confirm the headline enum + residual block:
./bin/qrt trust --json > "$QRATUM_HOME/scorecard.json"
./bin/qrt trust          # human summary + verbatim residual block
unset QRATUM_HOME

# Scorecard is a governed object: validate it against its schema and prove it
# carries no canary in its own bytes (driven by the trust runner's tests):
go -C . test ./cmd/trustbench/... ./cmd/qrt/... -run 'Trust|Scorecard|Residual|GateState|Recall'

# Race-clean (the harness must not introduce a data race):
make -C . test-race

# Public-CLI no-args reconciliation (assert the PINNED behavior — example for (A) status view):
export QRATUM_HOME="$(mktemp -d)"
./bin/qrt ; echo "exit=$?"   # (A): status view, exit 0  |  (B): error, exit 2
unset QRATUM_HOME

# README overstatement gone (no shipped search; capture/refine Claude-Code-only):
! grep -q "searches every session" ./README.md
grep -qi "claude code" ./README.md
```

VERIFY GAP: confirm the trust runner location before dispatch — the spec offers
both `cmd/trustbench` and a `//go:build trust`-tagged target. Pin one (the `make
trust` recipe and the `go test` path above must match the chosen layout). Also
confirm the exact `make trust` recipe shape mirrors the `dogfood-demo`
`QRATUM_HOME="$(mktemp -d ...)"` isolation pattern.

## Slop Review

- [ ] The harness drives the **real** entrypoints (daemon + standalone
  evidence/review/report/export per R3), not a shortcut, and collects P2–P5
  dimension results rather than re-implementing them.
- [ ] Three gate states behave correctly: BLOCKING-RED fails CI; KNOWN-RED is
  CI-non-blocking, monotonic, note+owner+deadline each; CI fails on over-deadline /
  count-increase / green-regression. Behavioral tests cover all four.
- [ ] Anti-gaming split present: hard covered-corpus blocker + monotonic extended
  recall; corpus shrinkage and recall regression both fail CI.
- [ ] The scorecard is a governed object: schema wired into D9, validator rejects an
  extra key, no-leak checker runs over its own bytes, provenance block populated,
  byte-stable modulo timestamp/commit.
- [ ] The honest-residual block prints verbatim and includes every required
  residual (credentials-only/PII deferred; at-rest encryption out; audit log not
  tamper-evident; cloud uncaptured; cross-vault cursor loss; backup-of-raw
  consent-gated exception; + the rest).
- [ ] `make verify` includes `trust`; CI uploads the scorecard artifact; no check
  weakened.
- [ ] `qrt` no-args behavior is the option the maintainer pinned (not chosen silently), and
  `main_test.go` matches it; README no longer claims "searches every session" and
  states capture + refine are Claude-Code-only.
- [ ] No new third-party Go dependency; no real secret/identifier in fixtures; tests
  never touch the real `~/.claude` or `~/.qratum`.

Reviewer guidance:

> Review this trust-gate harness against `verification-and-trust-gate.md` §3 and
> §7 and `GAPS.md` M11/Q4. Confirm: `make trust` drives the real shipped CLI
> entrypoints and assembles P2–P5 results without re-implementing them; the
> three-state gate model lets `make trust` into required CI without merge-locking
> (KNOWN-RED is tracked, monotonic, note+owner+deadline; CI fails only on
> BLOCKING-RED, over-deadline KNOWN-RED, KNOWN-RED count increase, green-gate
> regression, or extended-corpus shrinkage/recall regression); the headline is the
> enum with no averaging and no `TRUSTED-WITH-NAMED-GAPS`→`TRUSTED` abbreviation;
> the `qratum.trust_scorecard.v1` schema is committed and wired into D9, the
> no-leak checker runs over the scorecard's own bytes, and the provenance block is
> present; the honest-residual block prints verbatim and is complete. Confirm the
> two public-CLI reconciliations: the `qrt` no-args behavior was surfaced to the maintainer
> as a decision and the pinned choice is implemented and tested; the README no
> longer overstates "searches every session" and states capture + refine are
> Claude-Code-only. Flag: any new third-party Go dependency, any weakened existing
> check, any real-home mutation in tests/CI, any planned-RED gate wired as
> BLOCKING-RED, or any new product surface smuggled in (SQLite, resident daemon,
> search).

## Stop conditions

- STOP if the Qratum milestone is still `P1-VAULT-FIRST` / `P0` and the maintainer has not
  explicitly unlocked `P2-VERIFY-TRUST-GATE` — this is runtime + CI + public-CLI
  work and is gated.
- STOP if P2, P3, P4, or P5 has not landed — this prompt assembles their
  dimensions; without them the harness has nothing to collect.
- STOP and report if a dimension is missing or diverges from its phase spec, or if
  the P2–P5 residual list diverges from §3's honest-residual list — do not absorb
  the gap inside the harness or silently rewrite the residual block.
- STOP if the harness or scorecard appears to require a third-party Go dependency —
  report it as a supply-chain decision for the maintainer rather than adding it.
- STOP and present both options to the maintainer before implementing the `qrt` no-args
  behavior — this is a contract decision (status view vs. error), not a free call.
- STOP before running any command against the real `~/.claude` or `~/.qratum`;
  all runs use `QRATUM_HOME` pointed at a temp dir.
- STOP if `make verify` / `make trust` cannot be made green under the three-state
  model without weakening a check — report the failure, do not suppress it.

# P7 — Curation Tier / Cross-Repo Import (the dream tier, gated)

Package: `qratum-verify-trust-gate` · Prompt 7 of 7 · Depends on: P3 (the
`memory_import_receipt` schema) · Scope: runtime/cross-repo · Deliverable: the
`--kind memory_import_receipt` archive round-trip (now), plus the gated
cross-repo round-trip verification (after the gateway deploys)

## Cross-repo dependency — STATE THIS LOUDLY

This is the only task in the package that spans two repos. Read this before
anything else:

- The thing that **creates** `memory_import_receipt` files is a **producer in
  the `personal-memory` repo** (`scripts/import-claude-memories.ts`). It is
  **not built**, and it does **not** live in `qratum`. **Do NOT build it here.**
  qratum is the **consumer** — it archives and verifies receipts; it never
  produces them.
- The full behavioral verification in this task (round-trip across repos,
  idempotent re-run, `supersedes[]`, `namespace_forbidden`,
  unknown-`contentClass` reject, and rejecting any out-of-vocabulary
  outcome/errorClass) is **only meaningful once the personal-memory gateway
  Phase 1+ is deployed**. A no-leak proof against a hand-written synthetic
  fixture is **circular** — it only proves a clean fixture is clean. Until a
  real producer exists, that proof is **NOT-YET-MEANINGFUL** and must be
  labeled so on the scorecard, never scored green.
- Therefore this task has **two halves**:
  - **Half A (do now):** the `--kind memory_import_receipt` archive round-trip,
    with the kind **pinned** (the `--kind` default is `source_metadata`, a
    mislabel footgun). This is a contract check, not a security proof.
  - **Half B (GATED):** everything that needs the gateway. It **STOPS** until
    the personal-memory gateway Phase 1 is deployed.

This is the **dead-bridge lesson** made concrete. The earlier killed design
invented counterparty behavior (it minted a `duplicate` outcome the gateway
never emits). The rule here: **never invent the gateway's behavior.** qratum
must REJECT any receipt whose `outcome` or `errorClass` falls outside the
gateway's **real, documented vocabulary**. If that vocabulary is not yet
published by personal-memory, Half B does not run.

## Objective

Land qratum's **consumer** side of the cross-repo memory-import lane (the
"dream" / curation tier — in scope because Q3 resolved to BOTH). Make
`qrt vault archive <receipt> --kind memory_import_receipt` a real, pinned,
round-tripping deliverable now, and stand up the gated cross-repo verification
(behind the gateway) so that when personal-memory ships its producer, the leak
and idempotency proofs are wired and waiting — never faked green in the
meantime.

Build only the consumer half. No producer, no gateway, no grants/transport
logic. qratum imports receipts; it does not create or fetch them.

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` — the contract, in
  particular: §3 dimension **D10** ("the dream / curation tier"), the
  "Gated phase" block, §4 "Dream-tier gated phase", §5 Q3 (scope = BOTH),
  and the honest-residual block's D10 line.
- `qratum/docs/reviews/2026-06-15-verification-benchmark/BENCHMARK.md` §**D10**
  (the precise "testable now vs. gated" split, the default-kind footgun at
  `vault.go:312`, and the dead-bridge `duplicate`-outcome lesson).
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` "Should-address
  → schema / contract completeness" (`memory_import_receipt` is a shipped kind
  with no schema; the `source_metadata` default-kind footgun).
- P3 in this package — it delivers the `memory_import_receipt` JSON Schema and
  wires it into D9. **This task depends on that schema existing.**
- `qratum/internal/vault/vault.go` — `KindMemoryImport` (`:52`), `IsValidKind`,
  the archive path, content-addressed blob store, raw-ref records.
- `qratum/cmd/qrt/vault.go` — `parseArchiveArgs` (the `--kind` default is
  `vault.KindSourceMetadata`, the footgun); the archive command wiring.
- `qratum/cmd/qrt/vault_test.go` — the existing archive golden tests (the kind
  is asserted as `kind: source_metadata`).
- `qratum/schemas/qratum-raw-ref.v1.schema.json` — `memory_import_receipt` is
  already in the `kind` enum.
- `qratum/AGENTS.md` (fast-hook rule, supply-chain rule, Definition of Done,
  fixture/golden contract, the P2-VERIFY-TRUST-GATE milestone line).
- `qratum/docs/supply-chain.md`.

## Allowed scope

- **Half A (now):** make the `--kind memory_import_receipt` archive path a
  first-class, tested round-trip; pin the kind in tests; document and surface
  the default-kind footgun. Extend `cmd/qrt/vault.go` / `internal/vault` only
  as far as that needs.
- A synthetic `memory_import_receipt` fixture under `fixtures/` (a clean,
  hand-authored receipt — explicitly **NOT** a leak proof; only a shape/round-
  trip fixture). It must carry **no** real secret or internal identifier.
- **Half B (code authored now, gated tests):** the receipt-validation path that
  enforces the gateway's real vocabulary — idempotent re-run with
  `supersedes[]`, `namespace_forbidden`, unknown-`contentClass` hard-reject, and
  out-of-vocabulary `outcome`/`errorClass` hard-reject. These read the
  `memory_import_receipt` schema (P3). The **behavioral round-trip tests stay
  gated** until the gateway is deployed; mark them as such, do not score them.
- Scorecard/residual wiring for D10: the testable-now contract check is reported
  as a contract check (NOT a leak proof); the gated half is reported
  "not-yet-runnable (gateway Phase 1+)".

## Non-goals

- **Do NOT build the producer** (`scripts/import-claude-memories.ts`), the
  gateway, grants, transport, fetch, or any personal-memory-side code. qratum is
  the consumer only.
- Do NOT invent the gateway's `outcome`/`errorClass`/`contentClass` vocabulary.
  If personal-memory has not published it, the validator's allowed-set is a
  **STOP-and-report**, not a guess. (Dead-bridge lesson.)
- Do NOT score the synthetic-fixture no-leak check as a passing security gate —
  it is circular until a real producer exists.
- No SQLite, no resident daemon, no new product surface.
- No new third-party Go dependency. The receipt validator is stdlib-only,
  built on the same stdlib mini-validator the package uses for D9. Adding a
  dependency is an explicit supply-chain decision = **STOP and report**.
- No network calls, no LLM calls, no transcript parsing in the hook path.
- Do NOT run archive/import against the real `~/.claude` or `~/.qratum`.

## Implementation notes

### Half A — the archive round-trip (do now)
- A `memory_import_receipt` is archived through the existing content-addressed
  blob store, exactly like any other raw kind: stream sha256, dedup by digest,
  write via tmp+rename, write a `qratum.raw_ref.v1` record with
  `kind = memory_import_receipt` and `local_only: true`.
- **Pin the kind.** `parseArchiveArgs` defaults `--kind` to `source_metadata`.
  A receipt archived without `--kind memory_import_receipt` is **silently
  mislabeled**. The test must pass the flag explicitly and assert
  `kind: memory_import_receipt`. Document the footgun in the command help and in
  the runbook line; if cheap, detect a `*.receipt.json`-shaped input archived
  under the default kind and warn (loud, not silent) — but the mislabel itself
  is a usage footgun, do not auto-correct the kind.
- Round-trip = archive the synthetic receipt → read the blob back by digest →
  bytes are identical to the source. This is a **contract / integrity** check,
  not a leak proof. Say so in the test name and the scorecard line.

### Half B — the gated cross-repo verification (authored now, GATED tests)
The receipt-validation logic reads the `memory_import_receipt` schema (P3) and
enforces, against the gateway's **real** vocabulary:
- **Idempotent re-run with `supersedes[]`.** Re-importing a receipt that
  supersedes an earlier one is idempotent: the second run replaces by the
  `supersedes[]` lineage rather than double-importing or losing the prior. No
  duplicate blobs, no lost prior receipt.
- **`namespace_forbidden`.** A receipt naming a namespace qratum does not accept
  is **rejected** (recorded, loud), not silently imported.
- **Unknown `contentClass` → hard reject.** A receipt with a `contentClass`
  outside the schema's enum is **rejected**, never archived as `unknown`.
- **Out-of-vocabulary `outcome`/`errorClass` → hard reject.** This is the
  dead-bridge guard. The validator's allowed `outcome`/`errorClass` set is
  exactly the gateway's published vocabulary (from P3's schema, mirrored from
  personal-memory). Any value outside it (e.g. a fabricated `duplicate`) is
  **rejected**. The validator must **fail closed** on a malformed receipt shape
  (comma-ok, typed error — never panic).
- These behaviors are coded now against the schema, but their **round-trip
  tests are GATED**: they need a real receipt produced by the deployed gateway.
  Until then they are marked "not-yet-runnable (gateway Phase 1+)" on the
  scorecard, not run, not scored.

### Scorecard / residual (D10)
- Testable-now (Half A): report as a **contract round-trip check**, explicitly
  **"NOT-YET-MEANINGFUL as a leak proof"** until a real producer exists.
- Gated (Half B): report **"not-yet-runnable (gateway Phase 1+)"**.
- The honest-residual block must keep its line: D10 / the dream tier is an
  in-scope **gated phase** whose round-trip leak proof is meaningless until the
  personal-memory gateway is deployed.

## Acceptance criteria

(from `verification-and-trust-gate.md` → §3 D10, §4 dream-tier, §7 Acceptance)
- `qrt vault archive <receipt> --kind memory_import_receipt` round-trips: blob +
  ref written, `kind = memory_import_receipt` (NOT `source_metadata`), readable
  back byte-identical by digest.
- The default-kind footgun is documented and surfaced; the test pins the kind
  explicitly and would catch a regression to `source_metadata`.
- The `memory_import_receipt` schema (P3) is wired in: archiving a receipt that
  violates the schema is rejected loudly.
- Half-B validation logic exists and is unit-tested against the schema for:
  `namespace_forbidden` reject, unknown-`contentClass` reject, out-of-vocabulary
  `outcome`/`errorClass` reject, malformed-shape fail-closed.
- The cross-repo round-trip / idempotent-`supersedes[]` tests are present but
  **clearly marked gated** on the gateway, and the scorecard reports D10's
  testable-now part as a contract check (not a leak proof) and the gated part as
  not-yet-runnable.
- No real secret or internal identifier in the synthetic fixture (working tree
  and git history).
- `make verify` (incl. `make trust`) is green; gated D10 tests do not block CI
  and are not faked green.

## Decision Trace

- Scope = **BOTH** (Q3, the maintainer 2026-06-15) — reverses the earlier insurance-only
  call. The dream / curation tier is **in scope as its own gated phase**, not
  hidden behind a feature flag.
- The schema-and-archive work lands **now**; the full behavioral round-trip is
  **gated** on the personal-memory gateway Phase 1+ (the producer must exist
  before a round-trip leak proof means anything).
- Dead-bridge lesson (memory-architecture review, 2026-06-12): never invent
  counterparty behavior — reject any receipt outside the gateway's real
  outcome/errorClass vocabulary.
- Runtime build requires the P2-VERIFY-TRUST-GATE milestone unlock (the maintainer).

## Behavior Contract

- [ ] FAILS the task: building the producer, the gateway, grants, or any
  personal-memory-side code in qratum (consumer side only).
- [ ] FAILS the task: inventing any `outcome`/`errorClass`/`contentClass` value
  not in the gateway's published vocabulary (dead-bridge resurrection).
- [ ] FAILS review: scoring the synthetic-fixture no-leak check as a passing
  security gate (it is circular — contract check only).
- [ ] FAILS review: archiving a receipt under the default `source_metadata`
  kind without the footgun being surfaced; the test must pin
  `kind = memory_import_receipt`.
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency;
  evidence: `make verify`.
- [ ] FAILS on real-home mutation in tests/CI; evidence: `QRATUM_HOME` set.

## Drift Handling

- If the personal-memory gateway has **not** published its real
  outcome/errorClass/contentClass/namespace vocabulary, the Half-B allowed-set
  is unknowable — **STOP and report**. Do not guess it.
- If P3's `memory_import_receipt` schema is missing or its enums differ from
  what this task assumes, stop and reconcile against P3; do not duplicate or
  fork the schema.
- If the spec's scope decision (Q3 = BOTH) has changed since 2026-06-15, stop
  and report; this whole task is the dream tier and is moot under insurance-only.

## Verification

```sh
# Full local CI mirror (build, vet, lint, test, race, demo, dogfood, security, trust):
make -C . verify

# Half A — pinned-kind receipt archive round-trip in an isolated workspace
# (no real home touched):
export QRATUM_HOME="$(mktemp -d)"
make -C . build
./bin/qrt vault archive \
  ./fixtures/memory-import/synthetic-receipt.json \
  --kind memory_import_receipt
# assert the ref records kind=memory_import_receipt (NOT source_metadata):
./bin/qrt status
unset QRATUM_HOME

# Half A — the default-kind footgun is visible (archiving without --kind labels
# the receipt source_metadata): exercised by the vault_test.go golden tests.
go -C . test ./cmd/qrt/ -run TestArchive -count=1

# Half B — the receipt-vocabulary validator unit tests (schema-driven, no
# gateway): namespace_forbidden / unknown-contentClass / out-of-vocabulary
# outcome+errorClass / malformed-shape fail-closed:
go -C . test ./internal/... -run Receipt -count=1

# The trust gate reports D10 testable-now as a CONTRACT check (not a leak proof)
# and the gated half as not-yet-runnable; confirm the scorecard JSON says so:
make -C . trust
```

VERIFY GAP: confirm the exact fixture path for the synthetic receipt before
dispatch — this task creates `fixtures/memory-import/synthetic-receipt.json`
(no such fixture exists today). Confirm the P3 schema filename
(`schemas/qratum-memory-import-receipt.v1.schema.json` or similar) and reference
it, do not invent a second schema. Confirm `make trust` exists (delivered by an
earlier prompt in this package) before relying on it.

## Slop Review

- [ ] Did every Behavior Contract item get a behavioral test or explicit
  evidence path, and are gated cross-repo behaviors marked as gated instead of
  scored green?
- [ ] Are missing or invalid receipt inputs loud failures with operator-visible
  output, never swallowed by schema validation or archive import paths?
- [ ] The producer / gateway is NOT built in qratum (consumer side only).
- [ ] No invented counterparty vocabulary; out-of-vocabulary
  outcome/errorClass/contentClass/namespace is hard-rejected (dead-bridge guard).
- [ ] The `--kind` is pinned to `memory_import_receipt` in tests; the
  `source_metadata` default-kind footgun is documented and surfaced.
- [ ] The synthetic-fixture no-leak check is labeled NOT-YET-MEANINGFUL, never
  scored as a passing security gate.
- [ ] Gated round-trip/idempotency tests are clearly marked gateway-Phase-1+ and
  do not block CI; they are not faked green.
- [ ] The receipt validator fails closed (typed error, no panic) on a malformed
  shape; it reads the P3 schema, does not fork it.
- [ ] No real secret or internal identifier in the synthetic fixture (working
  tree and history).
- [ ] No new third-party Go dependency without supply-chain evidence; tests
  never touch the real `~/.claude` / `~/.qratum` (`QRATUM_HOME` set).
- [ ] `make verify` / `make trust` pass without weakening a check.

Reviewer guidance:

> Review this against `verification-and-trust-gate.md` §3 D10 + §4 dream-tier and
> BENCHMARK.md §D10. Confirm: qratum builds only the **consumer** side — no
> producer, no gateway, no grants. Confirm the `--kind memory_import_receipt`
> archive round-trips with the kind **pinned** (not the `source_metadata`
> default) and the footgun is surfaced. Confirm the receipt validator reads the
> P3 schema and **hard-rejects** out-of-vocabulary `outcome`/`errorClass`,
> unknown `contentClass`, and `namespace_forbidden` — never invents the
> gateway's behavior (dead-bridge lesson). Confirm the synthetic-fixture no-leak
> check is labeled a contract check / NOT-YET-MEANINGFUL, and the full cross-repo
> round-trip + idempotent `supersedes[]` tests are clearly **gated** on the
> personal-memory gateway Phase 1+ and not scored green. Flag any new
> third-party Go dependency, any real-home mutation, any invented counterparty
> value, or any attempt to score the gated half as passing.

## Stop conditions

- STOP if the **P2-VERIFY-TRUST-GATE** milestone is not unlocked by the maintainer —
  this is runtime/cross-repo work and is gated; the milestone is PROPOSED today.
- STOP if **P3** (the `memory_import_receipt` JSON Schema) has not landed —
  Half A's schema validation and all of Half B depend on it.
- STOP at Half B if the **personal-memory gateway Phase 1 is not deployed** AND
  its real `outcome`/`errorClass`/`contentClass`/namespace vocabulary is not
  published — do not guess it (dead-bridge lesson). Land Half A only; report the
  gated half as not-yet-runnable.
- STOP and report if any part requires building the producer or gateway inside
  qratum — that work lives in personal-memory.
- STOP if a feature appears to require a third-party Go dependency — report it as
  a supply-chain decision for the maintainer rather than adding it.
- STOP before running any command against the real `~/.claude` or `~/.qratum`.
- STOP if `make verify` or `make trust` cannot be made green without weakening a
  check — report the failure, do not suppress it.

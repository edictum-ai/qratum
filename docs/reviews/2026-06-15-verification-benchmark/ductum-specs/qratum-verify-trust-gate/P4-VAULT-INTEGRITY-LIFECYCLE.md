# P4 — Vault Integrity, Scale, Lifecycle & Recoverability (runtime)

Package: `qratum-verify-trust-gate` · Prompt 4 of 7 · Depends on: P1 ·
Scope: runtime/vault · Deliverable: integrity proofs (D2), idempotency + crash
recovery (D6), recoverability rewire (D6a/FIX-3), backup egress consent + streaming
(D8), the source-scope guard (D13), and the scale/lifecycle verbs (disk-full
preflight, `qrt vault gc`, tombstone-based erasure) — each with a locking
trust-gate dimension test

## Objective

Make the vault honest about the three things a forever-store has to get right:
the bytes are really what they claim to be, the store survives crashes and
concurrency without losing or corrupting a capture, and the product's headline
promise — "a transcript Claude deletes tomorrow is recoverable" — is actually
wired so a derived artifact can be rebuilt from the stored blob after the live
transcript is gone.

This task implements the integrity tier of the trust gate (D2, D5-adjacent, D6,
D6a, D8) plus the source-scope guard (D13), and the scale/lifecycle work the gap
review promoted from residual to in-scope (disk-full preflight FIX-14,
`qrt vault gc` FIX-15, tombstone-based per-object erasure FIX-16). It also fixes
the recoverability gap (FIX-3 / D6a). (The ref-id collision FIX-5 and the capture
TOCTOU FIX-6 land in P1, not here.)

This is integrity and lifecycle only. It does NOT touch the redactor, the
canary harness, schemas, the scorecard wrapper, or the install-schedule timer —
those are other phases. The preservation default never changes: **nothing is
ever auto-deleted**; the only path that removes anything is the explicit,
recorded, tombstone-based erasure verb (FIX-16).

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` (the contract — the
  Threat Model section, FIX-3, FIX-9, FIX-14, FIX-15, FIX-16, and §3 dimensions
  D2 / D6 / D6a / D8 / D13; §4 sequencing tier 4 + "Then"; §6 non-goals)
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` (the "scale and
  lifecycle" theme, the "concurrency beyond the one fixed TOCTOU" theme, M5
  (backup egress), and the "out of scope" notes on multi-machine merge and
  per-object erasure)
- `qratum/AGENTS.md` (fast-hook rule, supply-chain rule, Definition of Done,
  Ductum Factory Rules, fixture/golden discipline)
- `qratum/docs/supply-chain.md`
- `qratum/Makefile` (`build`, `test`, `test-race`, `verify`, `demo`,
  `dogfood-demo` targets; how `QRATUM_HOME` is set for `dogfood-demo`)
- `qratum/internal/vault/vault.go` (the load-bearing file for this task:
  `ArchiveFile` ~158, the `0o644` blob/ref/state writes, `UpdateState` ~321,
  `Backup`/`copyTree` ~400/483, `verifyTree` ~532, the `os.ReadFile` whole-file
  reads ~516/569)
- `qratum/internal/workspace/workspace.go` (`Resolve()` ~20-40 does NOT
  `MkdirAll` the root; the `raw/blobs/sha256`, `raw/blobs/.tmp`, `raw/refs`,
  `events`, `state` layout)
- `qratum/cmd/qrt/hook.go` (`UpdateState` call ~126; `copy_status` field ~50;
  the copy-on-capture path FIX-14 hooks into — note the `nextCaptureEventPath`
  TOCTOU at ~383 is fixed in P1, not here)
- `qratum/cmd/qrt/daemon.go` (`requireTranscriptFile` ~159 — the recoverability
  gap: hard-requires the live transcript, never falls back to the blob)
- `qratum/cmd/qrt/redact.go` (`validateQratumSession` ~121 — the source-scope
  check at ~128 rejects any source that is not `claude-code`; called from
  `redact`/`export`/`evidence`) and `qratum/internal/vault/vault.go` (the
  `SourceClaudeCode`/`SourceCodex`/`SourceManual` constants ~27-31 — `archive`
  ingests Codex/vendor/manual blobs that have no redaction path; the D13 gap)
- `qratum/cmd/qrt/vault.go` (the existing `vault doctor`/`backfill`/`archive`/
  `backup` command wiring; where new `gc` and erase verbs slot in)
- `qratum/fixtures/vault/`, `qratum/fixtures/dogfood/real-shaped-transcript.jsonl`

## Allowed scope

- `internal/vault/vault.go` and `internal/workspace/workspace.go` — integrity,
  atomicity, concurrency, GC, erasure, streaming backup.
- `cmd/qrt/daemon.go` — the FIX-3 blob-fallback rewire (D6a).
- `cmd/qrt/hook.go` — the FIX-14 disk-full preflight on copy-on-capture (the
  FIX-6 `O_EXCL` capture-path fix is P1's, not this task's).
- `cmd/qrt/redact.go` / `cmd/qrt/export.go` / `cmd/qrt/evidence.go` — the D13
  source-scope guard: assert that exporting or redacting a non-`claude-code`
  session is refused (the existing `validateQratumSession` source check is the
  enforcement point; do not loosen it).
- `cmd/qrt/vault.go` — new verbs `qrt vault gc` and the per-object erasure verb;
  the FIX-9 backup-consent gating; wiring `disk_free_min_gb` into capture.
- New/extended fixtures under `fixtures/vault/` (synthetic vaults, a synthetic
  orphan blob, two synthetic vaults for the merge-union property, a large
  synthetic blob for the streaming/RSS case, a synthetic non-`claude-code`
  (Codex/vendor) session for the D13 reject case).
- New tests in `internal/vault/` and `cmd/qrt/` driving the above, plus the D2 /
  D6 / D6a / D8 / D13 portions of the trust harness (`cmd/trustbench` or the
  `//go:build trust` tree — match whatever P5 / the gate-skeleton task lands;
  if the skeleton is not yet present, write the dimension tests as ordinary
  fixture/golden tests and leave a one-line note for the harness wire-up).

## Non-goals

- No redactor changes, no canary harness, no schema edits, no scorecard wrapper,
  no `install-schedule` timer — those are P2/P3/P5/P6 phases.
- No SQLite, no resident daemon, no new product surface.
- **No new third-party Go dependency.** Integrity, GC, erasure, streaming, and
  the merge-union check are all stdlib (`crypto/sha256`, `io`, `os`,
  `syscall`/`golang.org/x/sys` is NOT allowed without a supply-chain decision —
  prefer `syscall` from stdlib for `Statfs`). If any of these *appears* to need a
  dep, STOP and report it as a supply-chain decision.
- **No automatic deletion / retention / time- or size-based eviction.** The
  preservation default is "nothing lost." `gc` reclaims only genuinely orphaned
  blobs and refuses referenced ones; erasure is the only removal path and is
  always an explicit recorded operator action.
- No real multi-machine merge runtime — verify the dedup-union property with
  synthetic vaults only and state the state/event-cursor loss as a residual.
- Do NOT run `gc`, erase, backup, or capture against the real `~/.claude` or
  `~/.qratum`. Tests and demos set `QRATUM_HOME` to a temp dir.

## Implementation notes

### D2 — vault integrity (the proofs)
- **Independent-source byte-equality.** Hash the *original source file* with a
  separate reader and compare to `ref.Digest`. Re-hashing the committed blob is
  tautological and must NOT count as the proof — the whole point is that an
  independent read of the source agrees with what the vault recorded.
- **Dedup.** Two captures of identical content write one blob; the second is a
  recorded dedup, not a second blob.
- **Immutability.** A genuine digest collision (same full digest) is accepted as
  the same blob; a *different* digest must never overwrite an existing blob.
- **Atomicity.** No `.tmp` blob is left behind on a clean run. Inject a write
  failure mid-archive (e.g. a short/erroring `io.Reader`, or a read-only blob
  dir) and assert there is **no partial blob and no orphan ref** — the failed
  capture leaves the store exactly as it was.
- **Ref-id collision (FIX-5) lands in P1**, not here — do not build the
  `RawRefIDForDigest` fix or its collision-pair test in this task.
- **Multi-machine merge (property only).** Union two synthetic vaults; blobs are
  content-addressed so the union is collision-free and lossless for blobs. State
  plainly in the result that `state.json` cursors and event IDs are **NOT**
  merge-clean — cross-vault merge silently drops per-machine cursors/events; only
  blobs are dedup-clean. The scorecard line reads "cross-vault merge UNVERIFIED
  (blobs dedup-clean; state/events lossy)".
- **No auto-delete + GC + erasure** (see lifecycle below) are D2 assertions too.

### D6 — idempotency + crash recovery + concurrency
- **Idempotent re-run.** Re-running capture/backfill on the same content is a
  no-op (digest dedup).
- **Partial-artifact detect + crash-mid-write tmp recovery.** A `.tmp` blob left
  by a crash must be detected and not promoted; add a **startup sweep** of stale
  `.tmp` files at the start of `backfill` and `doctor` (the only cleanup today is
  a deferred remove at `vault.go:189` `CreateTemp`, which a crash skips — and
  `copyTree` even backs the stragglers up). The sweep must also be excluded from
  backup (see D8). Define "stale" concretely (e.g. `.tmp` older than a short
  grace window, or any `.tmp` with no in-flight writer — pin the rule and test
  both a stale orphan that gets swept and a fresh one that does not).
- **Missing-transcript loud-fail.** A missing live transcript fails loudly, not
  silently (this is the pre-FIX-3 behavior; D6a then makes it fall back to the
  blob).
- **Capture TOCTOU (FIX-6) lands in P1**, not here — do not build the
  `nextCaptureEventPath` `O_EXCL` fix or its goroutine-hammer race test in this
  task. (P4's concurrency sub-matrix below assumes that fix has landed.)
- **`UpdateState` non-atomic read-modify-write (concurrency sub-matrix).**
  `UpdateState` (`vault.go:321-328`) does LoadState → mutate → SaveState; the
  final rename is atomic but the read-modify-write around it is not, so
  concurrent hooks **lose counter increments** — and doctor's copy-failure
  counter is a D7 security-tier gate, so it silently undercounts and reads
  false-green about an irreversible store. Serialize state mutation with a
  `flock`/`O_EXCL` lockfile (stdlib `syscall.Flock` or an `O_EXCL` lock sentinel),
  **or** derive the counters from the append-only event store instead of mutable
  state. Test: a goroutine-hammering test asserts the final counter **equals**
  the injected failure count (today it would be less).
- **Concurrency sub-matrix.** Two `run-once` on the same session; backfill+hook
  on the same digest; backup-during-capture. Each run under `-race`. Per-file
  atomicity is not per-pipeline consistency — assert no lost capture and no
  corrupt state under each.
- **Refine-source consistency.** A mutated live file → refuse on digest mismatch,
  **or** document plainly that the blob is the only authoritative copy.

### D6a — recoverability rewire (FIX-3, the architectural fix)
- The daemon resolves and hard-requires the **live** `transcript_path`
  (`requireTranscriptFile`, `daemon.go:159`); it never falls back to the vault
  blob at `event.Raw.Digest`. So the headline promise is false for every derived
  artifact: a transcript Claude deleted but that exists as a blob fails the
  refinery.
- **Fix:** when the live path is missing or its content changed, read the raw
  bytes **from the blob by digest** and refine from those. This is the single
  most important correctness fix in the package.
- This dimension is **KNOWN-RED until the rewire lands** (per §3 / M11) — it is
  RED by design, CI-non-blocking, monotonic, with a tracking note + owner +
  deadline. Once it lands it goes GREEN and the scorecard headline can leave
  `NOT-TRUSTED` / `TRUSTED-WITH-NAMED-GAPS`.
- **Test:** capture → delete the source transcript → run the refinery → it
  succeeds **from the blob**, and (cross-ref to the redaction phase) the
  blob-sourced path must also leak zero secrets — leave a one-line note for that
  phase, do not implement the canary scan here.

### D8 — backup restorability + egress consent (FIX-9) + streaming
- **Backup of raw is a trust-boundary crossing.** `Backup` (`vault.go:400`)
  refuses only `dest==home`, then `copyTree`s the whole `~/.qratum` — including
  `raw/blobs/**` of un-redacted transcripts — to any path, and the vault docs
  advertise rclone/restic (cloud) as the use. That contradicts the model's "raw
  never leaves by default". **Fix:** before any backup whose source includes
  `raw/`, emit the one-line consent audit event; when `dest` is non-local (or
  undetectable) require an explicit `--allow-raw-egress` acknowledgement. Exporting
  raw means it is no longer private — say so loudly. Optionally offer a
  redacted-only backup mode.
- **Streaming, not `os.ReadFile`.** `Backup`/`verifyTree` read whole files into
  memory (`vault.go:516`/`569`, `verifyTree` reads both source and dest fully),
  so at the GB scale this product targets they OOM. Switch to a bounded-buffer
  `io.Copy`. Test a ~500MB synthetic blob and assert **bounded RSS** (not
  proportional to blob size).
- **Verify against the recorded `ref.Digest`, not a live re-read.** `verifyTree`
  re-reads the *source*, so post-capture drift and matched corruption both pass.
  Verify the dest against the recorded `ref.Digest` instead.
- **Corruption detection.** Flip one byte in a backed-up blob → verify fails.
- **Round-trip restore.** Point `QRATUM_HOME` at the dest; `status`/`doctor`/
  Summary match the source.
- **Refuses dest==home** (keep the existing guard).

### D13 — source-scope guard (no silent leak channel for vendor blobs)
- **The gap.** `validateQratumSession` (`redact.go:121`, source check at `:128`)
  already rejects any session whose `source` is not `claude-code`, and that
  function gates `redact`, `export`, and `evidence`. But the vault `archive` path
  accepts other sources — the `vault.go` constants `SourceCodex` (`:28`) and
  `SourceManual` (`:30`) exist, so Codex/vendor/manual blobs can land in the
  immutable store **with no redaction path behind them**. Capture and refine are
  Claude-Code-only; nothing else has a credential redactor.
- **Assert the boundary holds.** Lock it shut so a future loosening of the source
  check can't silently open a leak channel: feed a synthetic non-`claude-code`
  (Codex/vendor) session to **both** the refine/redact path **and** the export
  path and assert each is **rejected** (the existing `validateQratumSession`
  source check is the enforcement point). This is a guard, not a fix — do **not**
  loosen the check or add a redaction path for other sources here.
- **Scorecard line.** The scorecard states plainly: **capture and refine are
  Claude-Code-only** — vendor blobs are archive-only and have no redaction path.
- **No `archive`-side change.** Archiving a vendor blob into the vault is allowed
  (it is raw, never-deleted preservation); what is refused is *refining or
  exporting* it through a pipeline that assumes a Claude-Code redaction path.

### Scale / lifecycle verbs
- **Disk-full preflight on copy-on-capture (FIX-14).** `ArchiveFile`'s
  `io.Copy` (`vault.go:202`) is unbounded and there is a `disk_free_min_gb`
  config knob that nothing reads. Add a disk-free **preflight** (stdlib
  `syscall.Statfs` on the blob dir) that **degrades loudly** (a clear stderr
  warning + recorded failure, not a silent `copy_status=failed` with exit 0)
  when free space is under the threshold, plus a doctor escalation threshold.
  Wire `disk_free_min_gb` in — or remove the dead knob and say which.
- **`qrt vault gc` (FIX-15).** Reclaim only genuinely orphaned blobs (no live ref
  pointing at them). **Refuse to delete any referenced blob** — a blob named by a
  live ref, or by a tombstone that is not itself an erasure. Tombstone-respecting.
  The no-delete count invariant must hold for everything still referenced. Test:
  `gc` reclaims a synthetic orphan; attempting to gc a referenced blob is refused.
- **Per-object tombstone-based erasure verb (FIX-16).** The default is and stays
  **NEVER auto-delete / "nothing lost"** — nothing is removed by any automatic
  path. Erasure is the **one** explicit, operator-invoked, recorded action: it
  writes a tombstone and removes exactly the named blob's bytes, so the action is
  auditable and the reference graph stays consistent (`gc` and the no-delete
  invariants respect tombstones). This is how a third-party deletion request is
  honored. Test: an erasure writes a tombstone, removes exactly the targeted
  blob's bytes, records the action, and **no automatic path** ever removes a blob
  without a tombstone (a grep gate plus a count-never-decreases check, except
  through the erasure verb).

### Tests
- Fixture-driven (per AGENTS.md). Synthetic vaults / blobs only; no real secrets
  or real transcripts. Canary tokens that structurally can't match real formats;
  no real internal identifier in any committed fixture.
- Every test honors `QRATUM_HOME` and never touches the real `~/.claude` or
  `~/.qratum`. The injected-write-failure, race-hammer, and `Statfs`-low-space
  cases must be deterministic in CI.

## Acceptance criteria

(from `verification-and-trust-gate.md` → §3 D2/D6/D6a/D8/D13 and §7 Acceptance)
- **D2:** independent-source byte-equality (separate reader, not re-hash);
  dedup; immutability (wrong-digest overwrite rejected); atomicity (no `.tmp`
  leak; injected write-fail → no partial blob/ref); multi-machine merge union is
  blob-lossless and the result states state/event-cursor loss; no-auto-delete
  invariant holds. (Ref-id collision FIX-5 is P1's, not asserted here.)
- **D6:** idempotent re-run; partial-artifact detect; crash-mid-write `.tmp`
  recovery + **orphaned-`.tmp` sweep at backfill/doctor start** (and swept files
  excluded from backup); missing-transcript loud-fail; `UpdateState`
  serialized/locked so the goroutine-hammer counter equals the injected failure
  count; the concurrency sub-matrix (two run-once, backfill+hook,
  backup-during-capture) each `-race` clean; refine-source consistency decided
  and tested. (Capture TOCTOU FIX-6 is fixed and race-tested in P1, not here.)
- **D6a:** capture → delete source → refinery succeeds **from the blob**;
  KNOWN-RED until the rewire lands, then GREEN.
- **D13:** a non-`claude-code` (Codex/vendor) session is **rejected** by both the
  refine/redact path and the export path (no silent leak channel); the scorecard
  states capture + refine are Claude-Code-only.
- **D8:** backup of a raw-bearing vault without consent/ack is **refused**; with
  consent the audit event is emitted; non-local dest requires
  `--allow-raw-egress`; corruption (one flipped byte) is detected; round-trip
  restore matches; verify is against `ref.Digest`; large-blob backup/verify is
  **streamed** with bounded RSS; refuses `dest==home`.
- **FIX-14:** simulated low free space → capture refuses **loudly** and the
  failure is recorded (not a silent exit-0); doctor escalation fires;
  `disk_free_min_gb` wired (or removed, stated).
- **FIX-15:** `qrt vault gc` reclaims a synthetic orphan and **refuses any
  referenced blob**.
- **FIX-16:** the tombstone-based erasure verb removes exactly its target, writes
  a tombstone, records the action, and is the only removal path; **no automatic
  path** ever deletes.
- `make verify` (which includes `test-race`) is green; `make demo` and
  `make dogfood-demo` still pass.

## Decision Trace

- 2026-06-15 (the maintainer, in `verification-and-trust-gate.md` §5): promoted to
  in-scope required work — disk-full guard (FIX-14), tombstone-respecting
  `qrt vault gc` that refuses referenced blobs (FIX-15), per-object
  tombstone-based erasure verb (FIX-16, the only removal path), streaming
  backup-verify against `ref.Digest` (D8). Preservation default = **never
  auto-delete / "nothing lost"** (M10). Backup of raw is the sanctioned,
  consent-gated exception to "raw never leaves the machine" (M5/FIX-9).
- FIX-3/D6a recoverability is **KNOWN-RED** until the rewire lands (M11) — by
  design, CI-non-blocking, monotonic, with owner + deadline.
- 2026-06-16: the ref-id collision (FIX-5) and the capture TOCTOU (FIX-6) are
  **owned by P1**, not P4 — P4 no longer builds them or their tests, and only
  cross-references them. P4 adds the **D13 source-scope guard** (assert refining
  or exporting a non-`claude-code` session is rejected), since it is an
  integrity-boundary assertion over the same vault surface.
- Central-home artifact placement (FIX-4 / D11) is **owned by P1**; if P4 touches
  artifact location it notes the placement lands in P1.
- Multi-machine state/event merge and PII detection remain out of scope (named
  residuals), not this task.

## Behavior Contract

- [ ] FAILS review: any redactor/canary/schema/scorecard/install-schedule change
  (this task is integrity + lifecycle only). The D13 source-scope work is an
  **assert-only guard** over the existing `validateQratumSession` source check —
  it must NOT change redactor logic or add a redaction path for other sources.
- [ ] FAILS review: building the ref-id collision fix (FIX-5) or the capture
  TOCTOU fix (FIX-6) or their tests here — those are owned by P1.
- [ ] FAILS if refining or exporting a non-`claude-code` session is **not**
  rejected (D13), or if the source-scope check is loosened.
- [ ] FAILS review: any automatic deletion / retention / time- or size-based
  eviction. The only removal path is the explicit tombstoned erasure verb; `gc`
  reclaims only orphans and refuses referenced blobs.
- [ ] FAILS if the D2 byte-equality check re-hashes the committed blob instead of
  reading the original source with a separate reader (tautological proof).
- [ ] FAILS if `gc` can delete a referenced blob, or if any path other than the
  erasure verb removes a blob.
- [ ] FAILS if backup of a raw-bearing vault to a non-local dest proceeds without
  the consent audit event and the `--allow-raw-egress` ack.
- [ ] FAILS if backup/verify loads whole blobs into memory (RSS grows with blob
  size) instead of streaming, or verifies against a live source re-read instead
  of `ref.Digest`.
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency;
  evidence: `make verify`.
- [ ] FAILS on real-home mutation in tests/CI; evidence: `QRATUM_HOME` set + the
  capture→delete-source→refine-from-blob proof (D6a).

## Drift Handling

- The file:line refs above were confirmed against shipped `main` on 2026-06-15
  (`vault.go` `ArchiveFile`/`UpdateState`/`Backup`/`copyTree`/`verifyTree` and the
  `SourceClaudeCode`/`SourceCodex`/`SourceManual` constants; `workspace.go`
  `Resolve`; `redact.go` `validateQratumSession`; `daemon.go`
  `requireTranscriptFile`). If the code has since moved, re-locate by symbol name,
  not by line number, and note the drift.
- If a fix appears to need a third-party dep (e.g. for `Statfs` or flock), STOP
  and report it as a supply-chain decision rather than adding it — prefer stdlib
  `syscall`.
- Update fixtures/golden only when an output contract intentionally changes, and
  say so explicitly.

## Verification

```sh
# Full local CI mirror (build, vet, lint, test, RACE, demo, dogfood, security):
make -C . verify

# Race-only fast loop while iterating on the UpdateState concurrency work
# (the FIX-6 capture-path race is P1's):
make -C . test-race

# Integrity + lifecycle dimension tests in isolation:
go -C . test -race ./internal/vault/... ./cmd/qrt/...

# End-to-end recoverability proof (D6a / FIX-3) in an isolated workspace
# (no real home touched): capture, DELETE the source, prove refine succeeds
# from the blob.
export QRATUM_HOME="$(mktemp -d)"
make -C . build
cat ./fixtures/dogfood/real-shaped-transcript.jsonl \
  | ./bin/qrt hook claude-code   # capture → blob
# remove the live transcript the hook just captured, then run the refinery and
# confirm it rebuilds from the blob (not the deleted live path):
./bin/qrt vault doctor
./bin/qrt status
unset QRATUM_HOME
```

VERIFY GAP: confirm the exact fixture and hook entrypoint used for the D6a proof.
`fixtures/dogfood/real-shaped-transcript.jsonl` is the candidate standalone copy
target (it is what `make dogfood-demo` drives). Before dispatch, confirm which
fixture's `transcript_path` is a real, deletable file the hook copies, and what
the refinery entrypoint is called once FIX-3 lands (today `requireTranscriptFile`
in `daemon.go` hard-requires the live path — the proof is to delete that path
and assert the refine still succeeds from the blob).

VERIFY GAP: confirm whether the trust harness (`cmd/trustbench` or
`//go:build trust`) already exists from the gate-skeleton phase. If yes, wire the
D2/D6/D6a/D8/D13 assertions into it. If not, land them as ordinary fixture/golden
tests under `internal/vault/` and `cmd/qrt/` and leave a one-line note for the
harness wire-up — do not block on the skeleton.

## Slop Review

- [ ] D2 byte-equality reads the original source with a **separate reader**, not
  a re-hash of the committed blob (the tautology the gap review flagged).
- [ ] Atomicity test injects a real write failure and proves **no partial blob
  and no orphan ref** — not just "happy path leaves no `.tmp`".
- [ ] No FIX-5 (ref-id) or FIX-6 (capture TOCTOU) fix or test is built here —
  those are P1's; P4 only cross-references them.
- [ ] `UpdateState` is serialized/locked (or counters derived from events); the
  hammer test asserts the final counter **equals** the injected count, not "≈".
- [ ] D13 source-scope guard: refining or exporting a non-`claude-code` session
  is **rejected** (assert-only over the existing source check, no redactor change,
  no new redaction path); the scorecard states capture + refine are
  Claude-Code-only.
- [ ] D6a proves refine **from the blob** after the source is deleted; it is
  marked KNOWN-RED with owner + deadline until the rewire lands, never faked
  green.
- [ ] Backup streams (bounded RSS on a GB-scale blob), verifies against
  `ref.Digest`, detects a one-byte corruption, and refuses raw egress to a
  non-local dest without consent + `--allow-raw-egress`.
- [ ] `gc` refuses referenced blobs; erasure is the **only** removal path, writes
  a tombstone, records the action; **no automatic deletion** anywhere.
- [ ] Disk-full preflight degrades **loudly** (recorded + stderr), not a silent
  `copy_status=failed` exit 0; `disk_free_min_gb` is wired in or removed.
- [ ] No new third-party Go dependency; stdlib `syscall` for `Statfs`/flock.
- [ ] Tests never touch the real `~/.claude` or `~/.qratum` (`QRATUM_HOME` set);
  no real secret or internal identifier in any committed fixture.
- [ ] `make verify` (incl. `test-race`), `make demo`, `make dogfood-demo` pass
  without weakening a check.

Reviewer guidance:

> Review this integrity + lifecycle work against
> `qratum/specs/current/verification-and-trust-gate.md` (FIX-3/9/14/15/16,
> D2/D6/D6a/D8/D13) and `…/GAPS.md` (scale-and-lifecycle, concurrency, M5).
> Confirm: D2's byte-equality reads the original source independently (not a blob
> re-hash); atomicity is proven by an injected write failure leaving no partial
> state; the multi-machine union is blob-lossless and the result states
> state/event-cursor loss as a residual; `UpdateState` no longer loses concurrent
> counter increments (hammer test equals injected count); the `.tmp` sweep runs at
> backfill/doctor start and is excluded from backup; D6a refines from the blob
> after the source is deleted (KNOWN-RED until the rewire lands, with
> owner+deadline, never faked green); the D13 guard rejects refining or exporting
> a non-`claude-code` session without changing the redactor; backup is consent-
> gated for raw egress, streams without OOM, verifies against `ref.Digest`, and
> detects corruption; the disk-full preflight degrades loudly and wires
> `disk_free_min_gb`; `qrt vault gc` refuses referenced blobs and reclaims only
> orphans; the tombstone-based erasure verb is the only removal path and records
> the action; **nothing is ever auto-deleted**. Confirm the ref-id collision
> (FIX-5) and capture TOCTOU (FIX-6) fixes are **not** built here (they are P1's).
> Flag any redactor/canary/schema/scorecard change beyond the assert-only D13
> guard (out of scope here), any new third-party Go dependency, any automatic
> deletion/retention path, or any command that mutates the real
> `~/.claude`/`~/.qratum`.

## Stop conditions

- STOP if P1 (spec hygiene / acceptance) has not landed — this depends on P1.
- STOP if the Qratum milestone is not at `P2-VERIFY-TRUST-GATE` and the maintainer has
  not explicitly unlocked it — this is gated runtime work.
- STOP if any integrity/lifecycle fix appears to require a third-party Go
  dependency — report it as a supply-chain decision for the maintainer rather than adding
  it (prefer stdlib `syscall` for `Statfs`/flock).
- STOP before running `gc`, erase, backup, or capture against the real
  `~/.claude` or `~/.qratum`; tests/demos use `QRATUM_HOME`.
- STOP if FIX-3/D6a recoverability cannot be made GREEN within this milestone —
  do not fake it green; mark it KNOWN-RED (owner + deadline) and report.
- STOP if `make verify`, `make demo`, or `make dogfood-demo` cannot be made green
  without weakening a check — report the failure, do not suppress it.

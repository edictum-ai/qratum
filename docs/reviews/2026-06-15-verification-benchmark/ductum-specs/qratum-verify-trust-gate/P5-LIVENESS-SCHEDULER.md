# P5 — Passive-Capture Liveness & Doctor Truthfulness (runtime/ops)

Package: `qratum-verify-trust-gate` · Prompt 5 of 7 · Depends on: P1 ·
Scope: runtime/ops · Deliverable: `qrt vault install-schedule` (idempotent OS
timer running `qrt vault backfill`) + clean uninstall + a doctor that tells the
truth (D12 + D7), all with fixture/golden tests that never touch the real home.

## Objective

Two things, both about preservation staying alive without anyone babysitting it.

1. **Ship the backfill safety net (D12 / Q1).** The real-time SessionEnd hook is
   the primary capture path, but it only fires when a session ends locally. If
   the user installs the hook and walks away, anything Claude deletes before the
   next manual backfill is gone. So ship `qrt vault install-schedule` — an
   **idempotent OS timer** (a launchd plist on macOS, a systemd user timer on
   Linux) that runs **`qrt vault backfill`** on a schedule as the safety net
   behind the hook. This is **not** a resident daemon: the OS runs a one-shot
   `qrt vault backfill` on a cadence and exits. Refine stays on-demand. The
   command shows exactly what it writes before writing, re-installs idempotently,
   and uninstalls cleanly.

2. **Make doctor honest (D7).** `qrt vault doctor` must warn when something is
   wrong, stay quiet when all is well, and never claim a check it cannot actually
   make. Today's `transcript_drift` line is a live-tree heuristic that goes
   tautologically 0 right after a backfill and **false-warns on the
   source-deleted success case** — it must be relabeled a heuristic and compared
   against an **independent expectation**, not the live Claude tree. The
   cloud-blind-spot line must **always** be present.

Build only these two things. Do not build the trust gate harness, the redactor
fixes, the schema work, the vault lifecycle verbs, or any other dimension —
those are other prompts in this package.

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` — the contract. Sections
  "§3 The trust gate → D12 — Passive-capture liveness" (the Q1 test plan is
  written out there verbatim — follow it), "D7 — Operational truthfulness
  (doctor)", "§4 Sequencing step 6", "§7 Acceptance" (the `install-schedule` and
  doctor bullets).
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` — M10 (nothing
  makes backfill run → "survives abandonment" is structurally false), Q1 (the
  scheduler decision and recommendation), and the D7 reasoning behind the
  drift-is-a-heuristic finding.
- `qratum/AGENTS.md` (fast-hook rule, supply-chain rule, Definition of Done,
  Ductum Factory Rules), `qratum/docs/supply-chain.md`.
- `qratum/Makefile` (`build`, `test`, `test-race`, `vet`, `lint`, `demo`,
  `dogfood-demo`, `security`, `verify`; there is no `trust` target yet — another
  prompt adds it).
- `qratum/cmd/qrt/vault.go` — the current `vaultCommand` switch
  (`vault.go:24-44`, dispatch on `doctor|backfill|archive|backup`) and
  `vaultDoctor` (`vault.go:46-146`). Note the existing drift block at
  `vault.go:113-121` and the existing cloud line at `vault.go:136`.
- `qratum/cmd/qrt/hook.go` — the capture event shape (`event.Raw.CopyStatus`
  values `copied|deduped|failed|missing` at `hook.go:160-191`); the events the
  independent drift expectation must count come from here.
- `qratum/internal/vault/vault.go` — `State` (`vault.go:77-89`), `Summary`,
  `Store.New`, and the event/ref listing helpers; the central workspace root
  resolution lives in `internal/workspace`.
- `qratum/internal/claude/claude.go` — `SessionEndCommand = "qrt hook
  claude-code"` (`claude.go:14-15`), `HookStatusForProject`,
  `ListTranscriptFiles` (the live-tree call the drift heuristic uses today and
  must stop trusting for correctness).

## Allowed scope

- A new `qrt vault install-schedule` subcommand and a paired `qrt vault
  uninstall-schedule` (or `install-schedule --uninstall` — pick one and state
  it), wired into the `vaultCommand` switch in `cmd/qrt/vault.go`.
- A new internal package for the OS-timer generation/install/uninstall logic
  (e.g. `internal/schedule`), stdlib-only, with a platform split
  (`schedule_darwin.go` launchd, `schedule_linux.go` systemd user timer) behind a
  small platform-agnostic interface, so the generator and its assertions can be
  unit-tested on any host.
- Changes to `vaultDoctor` in `cmd/qrt/vault.go`: relabel `transcript_drift` as a
  heuristic, compute it against an independent expectation, add a
  schedule-installed/not-installed line, keep the always-on cloud line.
- An events-counting helper (count `session_end` events whose
  `copy_status ∈ {copied,deduped}`) — in `internal/vault` if the event store
  lives there, else a small reader in `cmd/qrt`.
- New fixtures/goldens under `qratum/fixtures/` for the generated plist/unit text
  and for doctor output under injected state.

## Non-goals

- **No resident daemon.** The OS runs a scheduled one-shot; `qrt` does not stay
  resident. Do not add a long-running process, a socket, or a watch loop.
- **No on-schedule refine.** The timer runs `qrt vault backfill` only. Refine
  stays on-demand. The installed command must be exactly `qrt vault backfill`
  (asserted) — it may not drift to `refine`, `archive`, or anything else.
- No new third-party Go dependency. launchd/systemd files are plain text the
  binary writes with stdlib (`encoding/xml` or `text/template` from stdlib for
  the plist, plain string assembly for the systemd unit). A third-party plist or
  unit library is an explicit supply-chain decision = STOP and report.
- **Never install into the real launchd/systemd, ever, in tests or CI.** Tests
  point the install target at a `t.TempDir()` standing in for
  `~/Library/LaunchAgents` / the systemd user unit dir, or use dry-run/print
  mode. No test may write to, `launchctl load`, or `systemctl --user enable`
  anything on the real machine.
- Do NOT run `install-schedule` / `uninstall-schedule` against the real home —
  that is an maintainer-only manual step (it mutates the user's real launchd/systemd
  and would start running real backfills).
- No trust harness, no redactor/schema/lifecycle work, no scorecard — other
  prompts.

## Implementation notes

### `qrt vault install-schedule` (D12)

- **Print / dry-run mode is mandatory and is what tests assert on.** Support a
  dry-run flag (e.g. `--print` or `--dry-run`) that prints the **exact**
  plist/unit text it would write **and** the **exact command it would install**,
  and writes nothing. Wire the same generator into both the dry-run path and the
  real install path so the printed bytes equal the written bytes.
- **The install target is overridable for tests.** The real target is
  `~/Library/LaunchAgents/<label>.plist` (macOS) or the systemd user unit dir
  (Linux). Tests must be able to redirect that target to a temp dir without
  touching the real one — via an env override (mirror the `QRATUM_HOME` pattern,
  e.g. `QRATUM_SCHEDULE_DIR`) or an explicit flag. State which you chose. The
  generated file lands there and **nowhere else**.
- **The installed command is `qrt vault backfill`.** Resolve the `qrt` binary
  path for the timer (absolute path of the running binary is fine), but the
  command the timer runs must be `<qrt> vault backfill` — assert the trailing
  `vault backfill` exactly.
- **macOS (launchd):** generate a `LaunchAgent` plist with a stable `Label`
  (e.g. `dev.qratum.backfill`), `ProgramArguments` = `[<qrt>, vault, backfill]`,
  and a `StartCalendarInterval` or `StartInterval` for the cadence. Use a stable
  cadence constant (state it — e.g. daily, or every N hours) so goldens are
  deterministic.
- **Linux (systemd user):** generate a `.service` unit (`Type=oneshot`,
  `ExecStart=<qrt> vault backfill`) plus a `.timer` unit (`OnCalendar=` /
  `OnUnitActiveSec=` matching the macOS cadence). Same stable cadence.
- **Idempotent install:** running install twice re-does no work and the written
  file is **byte-identical** the second time (no timestamps, no per-run nonce,
  no machine-specific value beyond the resolved binary path, which is stable for
  a given install). Print "already installed; no change" on the second run.
- **Show before writing:** print the path it will write and the content (or a
  diff against any existing file) before writing, consistent with how `qrt hook
  install` shows its diff. A real (non-dry-run) install still prints what it
  wrote.
- **Clean uninstall:** `uninstall` removes exactly the file(s) it wrote and
  leaves the schedule dir as it found it (empty if install created it); a second
  uninstall is a clean no-op (exit 0, "nothing to remove"). On Linux that means
  removing both the `.service` and `.timer` it wrote.
- **Doctor reflects schedule state:** until a schedule is installed, doctor and
  the eventual scorecard must state "preservation freshness depends on a schedule
  that is not installed." Add a `schedule_installed: yes|no` line to doctor that
  checks the (overridable) target dir for the file this command writes.

### Doctor truthfulness (D7)

- **`transcript_drift` is a heuristic, not a correctness gate — label it so.**
  Today (`cmd/qrt/vault.go:113-121`) drift = `len(ListTranscriptFiles()) -
  transcriptRefCount`, comparing archived refs against the **live** Claude tree.
  That is wrong twice: it reads 0 immediately after a backfill (tautological),
  and it **false-warns on the source-deleted success case** (a transcript Claude
  deleted is gone from the live tree but correctly preserved as a blob — the
  whole point of the vault). Replace the live-tree comparison with an
  **independent expectation**: the count of `session_end` capture events whose
  `copy_status ∈ {copied,deduped}`. Drift = that expected count minus the number
  of transcript-kind refs actually present. Print the line **labeled as a
  heuristic** (e.g. `transcript_drift (heuristic): +N (expected=… archived=…)`),
  and do not treat a non-zero value as a hard correctness failure — at most a
  soft warning, clearly named heuristic.
- **The cloud-blind-spot line is ALWAYS present** (a hard literal gate). Keep the
  existing line (`vault.go:136`): sessions that start and end on vendor infra are
  not captured in vault v1. It prints on every doctor run, healthy or not.
- **Warnings fire on injected state, against a stubbed clock.** The no-hook,
  stale-backfill, copy-failure, and unverified-backup warnings must each fire
  when the underlying state says so. Staleness comparisons must be testable
  without `time.Sleep` — inject the clock (the staleness checks at
  `vault.go:98/106` call `stale(...)` which uses `time.Since`; make the
  reference "now" injectable so a fixture timestamp + a fixed now produces a
  deterministic stale/ok result).
- **Healthy path stays quiet.** With a hook installed, a fresh backfill, zero
  copy failures, a verified recent backup, and an installed schedule → doctor
  prints `warnings: none`.

### Tests (fixture/golden, honor `QRATUM_HOME`, never touch the real machine)

- **install-schedule golden (per platform).** Run dry-run/print mode; assert the
  printed plist/unit **bytes** equal the committed golden, and that the golden's
  `ProgramArguments`/`ExecStart` end in exactly `vault backfill`. Parse the
  generated plist (stdlib XML) / unit and assert: a cadence field is present, and
  the installed command equals `qrt vault backfill`. This last assertion is the
  single most important one — guard it explicitly.
- **Real-but-fake install.** Point the schedule target at a `t.TempDir()`; run
  install; assert the file lands in that temp dir and that **nothing** appears in
  the real `~/Library/LaunchAgents` / systemd dir (the test must not even resolve
  the real path — use the override).
- **Idempotent re-install.** Run install twice into the fake dir; assert the file
  is byte-identical and the second run reports no change.
- **Clean uninstall.** Run uninstall; assert exactly the written file(s) are
  removed and the fake dir is empty; run uninstall again; assert clean no-op,
  exit 0.
- **Doctor drift heuristic.** Inject an event store with K `session_end` events
  `copy_status ∈ {copied,deduped}` and a vault with K transcript refs → drift
  reads 0 and the line is labeled a heuristic. Then the **source-deleted success
  case**: K events but the live Claude tree is empty (transcripts deleted) and K
  refs present → drift stays 0 and **does not** false-warn (the old live-tree
  computation would have warned).
- **Doctor warnings.** Inject: no global hook → no-hook warning; a stale
  `last_backfill_at` against a stubbed now → stale-backfill warning; a non-zero
  `copy_failure_count` → copy-failure warning; an empty
  `last_backup_verified_at` → unverified-backup warning. Assert each fires.
- **Doctor cloud line always present.** Assert the cloud-blind-spot line prints
  on both the healthy and the warning runs.
- **Doctor healthy path.** All-good injected state → `warnings: none`.
- All tests set `QRATUM_HOME` (and the schedule-dir override) to temp dirs.

## Acceptance criteria

(from `verification-and-trust-gate.md` → D12, D7, §7)
- `qrt vault install-schedule` installs an idempotent backfill timer (launchd on
  macOS, systemd user timer on Linux), shows what it writes before writing, and
  re-installs byte-identically.
- The installed command equals exactly `qrt vault backfill` — asserted by parsing
  the generated plist/unit. It is not a resident daemon and does not run refine.
- `uninstall` removes exactly the file(s) it wrote; a second uninstall is a clean
  no-op.
- Dry-run/print mode prints the exact plist/unit and command and writes nothing.
- No test or CI run ever writes to, loads, or enables the real launchd/systemd;
  the install target is redirected to a temp dir.
- Doctor's `transcript_drift` is **labeled a heuristic** and computed against the
  independent expectation (count of `session_end` events with `copy_status ∈
  {copied,deduped}`), so it does not false-warn on the source-deleted success
  case and is not tautologically 0 post-backfill.
- Doctor warnings (no-hook, stale-backfill, copy-failure, unverified-backup) all
  fire against injected state and a stubbed clock; the healthy path prints
  `warnings: none`.
- The cloud-blind-spot line is present on every doctor run.
- Doctor reports `schedule_installed: yes|no`; when no, the "preservation
  freshness depends on a schedule that is not installed" framing is stated.
- `make verify` (`vet lint test test-race build demo dogfood-demo security`,
  plus the supply-chain check) is green; `go test -race` is clean.

## Decision Trace

- Q1 resolved by the maintainer (2026-06-15): **ship `qrt vault install-schedule`** — an
  OS timer running `qrt vault backfill` as the safety net behind the real-time
  hook; **not** a resident daemon; refine stays on-demand; ships **with an
  explicit OS-timer test plan**. (`verification-and-trust-gate.md` §"Decisions
  already taken", D12; GAPS Q1 recommendation = Option 2.)
- D7 drift relabel: GAPS finding that `transcript_drift` is a live-tree heuristic
  that false-warns on the source-deleted success case; fix is to compare against
  an independent expectation (`session_end` events with `copy_status ∈
  {copied,deduped}`).
- Cloud-blind-spot line ALWAYS present: D7 hard literal gate.

## Behavior Contract

- [ ] FAILS review: any resident daemon, watch loop, socket, or on-schedule
  `refine` — the timer runs `qrt vault backfill` only.
- [ ] FAILS review: the installed timer command is anything other than `qrt
  vault backfill` (asserted by parsing the generated plist/unit).
- [ ] FAILS if any test or CI step writes to, loads, or enables the real
  launchd/systemd, or resolves the real LaunchAgents / systemd user dir; the
  install target must be redirected to a temp dir; evidence: the fake-dir tests.
- [ ] FAILS if `transcript_drift` is still computed against the live Claude tree,
  is unlabeled, or warns on the source-deleted success case; evidence: the
  source-deleted doctor test.
- [ ] FAILS if the cloud-blind-spot line is absent on any doctor run; evidence:
  the always-present cloud-line test.
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency
  (plist/unit/systemd library); evidence: `make verify` supply-chain check.
- [ ] FAILS on real-home mutation in tests/CI; evidence: `QRATUM_HOME` + the
  schedule-dir override are set in every test.

## Drift Handling

- If `vaultDoctor` or the event store has changed since this spec was written in
  a way that breaks the file:line references above (e.g. drift block moved, event
  shape changed), stop and report; re-read the current `cmd/qrt/vault.go` and
  `cmd/qrt/hook.go` before editing. Update goldens only when an output contract
  intentionally changes, and say so.
- If a stdlib-only plist/unit generation turns out to need a third-party library,
  STOP and report it as a supply-chain decision for the maintainer — do not add the dep.

## Verification

```sh
# Full local CI mirror (build, vet, lint, test, race, demo, dogfood, security):
make -C . verify

# Race-clean (the schedule generator and doctor counters):
make -C . test-race

# Dry-run / print mode in an isolated workspace (no real home, no real launchd):
export QRATUM_HOME="$(mktemp -d)"
export QRATUM_SCHEDULE_DIR="$(mktemp -d)"   # fake LaunchAgents / systemd user dir
make -C . build
# Print mode writes nothing and prints the exact plist/unit + the installed command:
./bin/qrt vault install-schedule --print
# Assert it printed `qrt vault backfill` and a cadence field, and that the fake
# schedule dir is still empty after --print:
ls -A "$QRATUM_SCHEDULE_DIR"            # expect: empty

# Real-but-fake install lands the file only in the fake dir, idempotent, then clean uninstall:
./bin/qrt vault install-schedule
ls -A "$QRATUM_SCHEDULE_DIR"            # expect: exactly the timer file(s)
./bin/qrt vault install-schedule   # second run: byte-identical, "no change"
./bin/qrt vault uninstall-schedule
ls -A "$QRATUM_SCHEDULE_DIR"            # expect: empty again
./bin/qrt vault uninstall-schedule # clean no-op, exit 0

# Doctor truthfulness in isolation (uses injected state via tests; manual smoke):
./bin/qrt vault doctor
# expect: a `transcript_drift (heuristic): …` line, an always-present cloud line,
# a `schedule_installed: …` line, and `warnings:` reflecting the injected state.
unset QRATUM_HOME QRATUM_SCHEDULE_DIR
```

VERIFY GAP: confirm the exact env var name for the schedule-dir override
(`QRATUM_SCHEDULE_DIR` is the suggested name following the `QRATUM_HOME`
pattern) and confirm whether uninstall is a separate `uninstall-schedule`
subcommand or `install-schedule --uninstall` before dispatch — pin one in the
implementation and make the goldens and the commands above match it.

## Slop Review

- [ ] The timer is a one-shot OS schedule, not a resident daemon; it runs exactly
  `qrt vault backfill` (parsed from the generated plist/unit), never `refine`.
- [ ] install-schedule has a real dry-run/print mode whose bytes equal the
  installed bytes; install is idempotent (byte-identical re-install); uninstall
  removes exactly what was written and a second uninstall is a clean no-op.
- [ ] No test, no CI step, and no manual verify writes to / loads / enables the
  real launchd or systemd; the target is always redirected to a temp dir.
- [ ] `transcript_drift` is labeled a heuristic and compared against the
  independent expectation (`session_end` events with `copy_status ∈
  {copied,deduped}`), and does NOT false-warn on the source-deleted success case.
- [ ] The cloud-blind-spot line prints on every doctor run; doctor warnings fire
  on injected state against a stubbed clock; the healthy path is quiet.
- [ ] No new third-party Go dependency (stdlib-only plist/unit generation).
- [ ] Tests are fixture/golden-driven and honor `QRATUM_HOME` + the schedule-dir
  override; `make verify` and `go test -race` pass without weakening a check.

Reviewer guidance:

> Review this against `verification-and-trust-gate.md` D12 + D7 and the Q1 test
> plan. Confirm: `qrt vault install-schedule` generates an idempotent OS timer
> (launchd plist / systemd user timer) whose installed command is exactly `qrt
> vault backfill` (parse it — do not trust a string match in prose); it is not a
> resident daemon and never schedules `refine`; it has a dry-run/print mode and a
> clean uninstall; every test redirects the install target to a temp dir and the
> real launchd/systemd is never touched. Confirm doctor relabels
> `transcript_drift` as a heuristic, computes it against the independent
> expectation (count of `session_end` events with `copy_status ∈
> {copied,deduped}`) rather than the live Claude tree, no longer false-warns on
> the source-deleted success case, always prints the cloud-blind-spot line, fires
> its four warnings on injected state against a stubbed clock, and is quiet on the
> healthy path. Flag any new third-party Go dependency, any real-home/real-timer
> mutation, or any drift back to a live-tree drift computation.

## Stop conditions

- STOP if the **P2-VERIFY-TRUST-GATE** milestone is not unlocked by the maintainer —
  this is gated runtime/ops work. If the milestone pointer in `SPEC.md` is still
  pre-P2-VERIFY-TRUST-GATE, stop and report.
- STOP if P1 (spec hygiene / contracts) has not landed — this prompt depends on
  P1.
- STOP if a stdlib-only plist/unit generator appears to require a third-party Go
  dependency — report it as a supply-chain decision for the maintainer, do not add it.
- STOP before running `install-schedule` / `uninstall-schedule` against the real
  home or the real launchd/systemd — that is an maintainer-only manual step.
- STOP if `make verify` or `go test -race` cannot be made green without weakening
  a check — report the failure, do not suppress it.

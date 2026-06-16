# P1 — Contained Security Fixes (runtime)

Package: `qratum-verify-trust-gate` · Prompt 1 of 7 · Depends on: none ·
Scope: runtime/security · Deliverable: the confirmed, contained security fixes
(FIX-1, FIX-2, FIX-4, FIX-5, FIX-6, FIX-8, FIX-11, FIX-12), each with a locking
test, plus the `internal/capture` import-isolation extraction (D1) and the
no-secret-in-committed-golden lint that also scans git history.

## Objective

Close the confirmed, contained credential-leak and integrity defects on `main`
that need no architectural change. These are the cheapest, highest-value fixes in
the whole milestone: each one is a small, local code change that already has an
adversarial finding behind it, and each ships **with a test that locks it shut**
so it cannot silently regress. This task does **not** build the trust-gate harness
itself (that is later prompts); it makes the confirmed leaks stop and proves they
stopped.

In plain terms: today a secret written with a `=>` arrow leaks in the clear, git
and timestamp fields ship raw, the daemon writes raw secret-bearing artifacts into
the working tree of whatever git repo you run from, two blobs can collide on a
short id, two hooks can clobber each other, every raw blob on disk is
world-readable, the external export lets unknown internal keys through, and the
hook will copy any file you point it at (including a symlink to `~/.ssh/id_rsa`)
into the never-deletable vault. Each of those is fixed here, with a test that goes
RED if the fix is undone, and the committed golden file is re-redacted so the
contract no longer encodes a leak. The capture path is also extracted into its own
`internal/capture` package whose import set is pinned by a golden, so a future
`net` import (the start of a passive-capture exfil channel) trips the gate RED.

Build only the fixes named below. Do **not** build the reflection-canary harness,
the scorecard, `data_class`, the standalone-re-redact fix (FIX-10), the disk-full
guard (FIX-14), `qrt vault gc` / erasure (FIX-15/16), or the recoverability rewire
(FIX-3) — those are other prompts in this package. This task **does** own the
central-home artifact placement (FIX-4 / D11) and the `internal/capture`
import-isolation extraction (the D1 carve-out), both named below.

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` — the contract. Read
  FIX-1, FIX-2, FIX-4, FIX-5, FIX-6, FIX-8, FIX-11, FIX-12 in §2; D11 (artifact
  placement / containment), D14, and the at-rest notes in §3; the import-isolation
  carve-out in D1 (§3 and §6 Non-goals); the Threat Model section; Tier 0 in §4
  (Sequencing). This file implements exactly that subset.
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` — M4 (world-
  readable blobs / D14), M9 (ADP fail-open denylist), the untrusted-input
  hardening item (hook `transcript_path`), and the "committed fixtures bake a real
  internal URL into permanent git history" item (drives the history-scanning lint).
- `qratum/AGENTS.md` — fast-hook rule, supply-chain rule (stdlib-only; a new
  third-party Go dependency is an explicit STOP-and-report decision), Definition
  of Done, Ductum Factory Rules, `QRATUM_HOME` test isolation.
- `qratum/docs/supply-chain.md`.
- `qratum/Makefile` (`build`, `test`, `test-race`, `vet`, `lint`, `verify`).
- The code each fix touches (confirmed at these locations on `main`, 2026-06-15):
  - `cmd/qrt/redact.go:22` — `secretAssignmentPattern` (FIX-1); `redactQratumSession`
    around `:200-250`, `git.Remote` routed but no SSH pattern; `started_at`,
    `ended_at`, `source_event_id`, `git.branch`, `git.head_sha` never routed (FIX-2).
  - `fixtures/redaction/secret-session.redacted.golden.json:32` — committed
    `git@github.com:edictum-ai/qratum.git` + branch + timestamps (FIX-2 re-redact).
  - `internal/workspace/workspace.go:92` — `RawRefIDForDigest` truncates to
    `"raw_" + digest[:12]` (FIX-5).
  - `cmd/qrt/hook.go:340/375/383-391` — `nextCaptureEventPath` does `os.Stat` then
    the caller does `os.Rename` (TOCTOU, FIX-6); `resolveHookTranscriptPath`
    (`:203`) + `ArchiveFile` (`internal/vault/vault.go:181` `os.Open`) follow
    symlinks with no confinement or cap (FIX-12).
  - `internal/vault/vault.go:207` (`Chmod(0o644)`), `:278`/`:315`
    (`writeFileAtomic(..., 0o644)`), `:177/:218/:255/:312/:491/:506/:578`
    (`MkdirAll(..., 0o750)`); `internal/workspace/workspace.go` `Resolve()` never
    `MkdirAll`s the root (FIX-8).
  - `cmd/qrt/export.go:373` — `isQratumOnlyExportKey` denylist (FIX-11).
  - `cmd/qrt/daemon.go` — the daemon writes derived artifacts to repo-local
    `./.qratum/sessions/*` relative to `os.Getwd()` (FIX-4); `cmd/qrt/sessions.go`
    (sessions `list`/`review` discovery) and `cmd/qrt/ui.go` (UI artifact paths)
    read those locations and must follow the relocation under `~/.qratum`.
  - the hook capture path in `cmd/qrt/hook.go` (and the vault/workspace helpers it
    calls) — the code extracted into `internal/capture` for the D1 import-isolation
    carve-out.

## Allowed scope

- `cmd/qrt/redact.go` (FIX-1 value-capture/separator fix; FIX-2 hybrid drop +
  redact + SSH-remote pattern).
- `internal/workspace/workspace.go` (FIX-5 ref-id; FIX-8 root `MkdirAll(0o700)`).
- `internal/vault/vault.go` (FIX-8 perms `0o600`/`0o700`; FIX-12 confinement +
  no-symlink + size cap in `ArchiveFile` or its caller).
- `cmd/qrt/hook.go` (FIX-6 `O_EXCL` create; FIX-12 path confinement on resolve).
- `cmd/qrt/export.go` (FIX-11 allowlist projection for the ADP).
- `cmd/qrt/daemon.go` (FIX-4: write **all** derived artifacts under
  `~/.qratum/sessions/<session_id>/`, `QRATUM_HOME` override, instead of repo-local
  `./.qratum/` relative to `os.Getwd()`; record the repo as session metadata).
- `cmd/qrt/sessions.go` and `cmd/qrt/ui.go` (FIX-4: read artifacts from the central
  home; let `sessions list`/`review` filter by the recorded repo metadata).
- A new `internal/capture` package (D1 import-isolation extraction): move the hook
  capture path here, importing **only** `crypto/sha256`, `os`, `io`, and the
  `vault`/`workspace` packages; `cmd/qrt/hook.go` calls into it.
- A new committed golden pinning the allowed import set of `internal/capture`
  (e.g. `internal/capture/imports.golden` or an in-test allowed-set constant) plus
  its negative-test fixture.
- New `*_test.go` files for each fix (fixture/golden + behavioral + the `-race`
  test for FIX-6).
- A new no-secret-in-golden lint: a Go test (or `//go:build` tool) that scans both
  the working tree **and** `git log -p` history for known internal identifiers.
- New/extended fixtures under `fixtures/redaction/` and `fixtures/claude-code/`
  for the hostile-payload cases (FIX-12) — synthetic/canary only.

## Non-goals

- No reflection-canary harness, no `make trust`, no scorecard, no `data_class`
  (those are later prompts). This task's tests are ordinary `go test` cases.
- No FIX-3 (recoverability rewire), FIX-7 (extra evasion classes), FIX-9 (backup
  consent), FIX-10 (standalone re-redact), FIX-13–16. (FIX-4 central-home placement
  **is** owned here — see Implementation notes / Acceptance below.)
- No new third-party Go dependency. The history-scanning lint must use `git` (via
  `os/exec`) plus stdlib only. If any fix appears to need a dependency, **STOP and
  report it as a supply-chain decision** — do not add it.
- No network, no LLM, no transcript parsing added to the hook (fast-hook rule).
- Do **not** run anything against the real `~/.claude` or `~/.qratum`.

## Implementation notes

### FIX-1 — `=>` partial-redaction bug (CRITICAL)
`secretAssignmentPattern` captures the value as `[^\s\"',;]+` after a `\s*[:=]\s*`
separator. For `PASSWORD => hunter2pass` it matches sep `=`, value `>`, redacts
only the `>`, and emits `hunter2pass` in the clear. Make the value capture robust
to arrow/extra separators — e.g. consume `[:=>]+` (or `\s*[:=]+>?\s*`) in the
separator group — **and/or** re-scan the post-placeholder text for any residual
secret token. The locking test asserts: after redaction of every `=>`/`==>`/`: =>`
variant, **no residual secret token survives** anywhere in the output.

### FIX-2 — git/time/event fields leak verbatim (CRITICAL) · decision: HYBRID
`redactQratumSession` never routes `started_at`, `ended_at`, `source_event_id`,
`git.branch`, `git.head_sha` through `redactString`; `git.remote` is routed but an
SSH remote (`git@host:org/repo.git`) matches no pattern. The committed golden ships
all of these raw. The locked hybrid fix:
- **Drop** `git.branch`, `git.head_sha`, `git.remote`, `started_at`, `ended_at`,
  `source_event_id` from every **shareable** artifact (HTML report, UI DTO, ADP
  export). They add little to those surfaces and a branch name can itself be a
  secret (e.g. `feature/customer-acme-prod-keys`).
- **Redact** the same fields in the redacted-session JSON (route through
  `redactString`) so the local redacted store carries no raw secret either.
- Add an **SSH-remote redaction pattern** (`git@host:org/repo.git`) as its own
  class so a routed SSH remote is actually caught.
- **Re-redact the committed golden** (`secret-session.redacted.golden.json`) so the
  contract no longer encodes the leak. The current golden has the real remote,
  branch, and timestamps at `:8`/`:32-34`.

Note (flag, do not fix here): downstream artifacts copy these fields straight from
the struct (`evidence.go` builds `Summary` and copies `started_at`/`ended_at`/
`source_event_id`), so the "redact in JSON" step alone does not stop a
re-introduction leak. That terminal byte-scan is a later prompt (D3/D4); call it
out in your handoff so it isn't assumed closed.

### FIX-4 — raw artifacts escape into the working tree (CRITICAL leak) · decision: CENTRAL HOME
The daemon writes every derived artifact into `./.qratum/sessions/*` relative to
`os.Getwd()` — so a run from inside a checkout drops `normalized.json` (carrying
raw `sk-ant-…` secrets), the redacted JSON, the evidence bundle, the review card,
the HTML report, and the ADP export into a possibly-git-tracked tree. That is a
raw-secret leak that depends only on a missing per-repo `.gitignore`, and it leaves
three locations in play (central event store, live transcript, repo-local
artifacts). Locked fix:
- Move **all** derived artifacts under the central workspace at
  `~/.qratum/sessions/<session_id>/` (`QRATUM_HOME` override), in the layout the
  operational model already specifies (`normalized.json`, `redacted.json`,
  `evidence.json`, `review.json`, `report.html`, `session.adp.jsonl`). No artifact
  is ever written relative to `os.Getwd()`.
- **Record the repo as metadata** on the session (so `qrt sessions list` / `review`
  can filter by repo) rather than as a write location. The repo identity is data,
  not a directory the pipeline writes into.
- Touches `daemon.go` (the artifact write paths), `sessions.go` (list/review
  discovery + repo filter), and `ui.go` (UI artifact path resolution). The vault
  root `MkdirAll(0o700)` in FIX-8 is the perms fix; this is the relocation — they
  are separate and both land here.
The locking test (D11) asserts: no artifact path resolves outside `QRATUM_HOME`,
and a run from **inside a git repo** writes **nothing** to the working tree (the
checkout is byte-identical before and after). Set `QRATUM_HOME` to a `t.TempDir()`;
never touch the real home.

### FIX-5 — short ref-id prefix can collide (HIGH, latent)
`RawRefIDForDigest` returns `"raw_" + digest[:12]`, while blobs use the full
digest. Two digests sharing a 12-hex prefix collide into one ref path and the
second is wrongly rejected as a duplicate (rejection at `vault.go:265`). Use enough
digest bytes that the ref identity matches blob identity — the full digest, or a
collision-safe prefix length with a stated rationale. The locking test stores two
**distinct** digests that share a 12-char prefix and asserts both refs persist
distinctly.

### FIX-6 — concurrent captures can clobber each other (HIGH, TOCTOU)
`nextCaptureEventPath` does `os.Stat`/`ErrNotExist` then the caller `os.Rename`s in
— two concurrent hooks can both see `ErrNotExist` and the second silently
overwrites the first. The ductum parallel-agent factory produces exactly this
concurrency. Fix: **create the event file with `O_EXCL`** (`os.OpenFile(path,
O_CREATE|O_EXCL|O_WRONLY, 0o600)`) and, on `EEXIST`, advance to the next candidate
id and retry — no stat-then-rename window. The locking test is a goroutine-hammering
race test (N concurrent captures → N distinct events, zero lost), run under
`go test -race` clean.

### FIX-8 — raw secret blobs are world-readable (CRITICAL, gap M4) + D14
Blobs/refs/state are written `0o644` (world-readable) and the workspace root is
never created with restrictive perms, so `~/.qratum` inherits the process umask
(often `0o755`). Any local user can read raw un-redacted transcripts and bypass the
entire redaction pipeline. Fix:
- `workspace.Resolve()` (or a setup step it calls) must `MkdirAll(root, 0o700)` and
  `Chmod(0o700)` an existing root.
- Blob/ref/event/state files: `0o644` → **`0o600`** (the `tmp.Chmod` at
  `vault.go:207` and the `writeFileAtomic(..., perm)` callers).
- All vault dirs (`MkdirAll(..., 0o750)` sites): `0o750` → **`0o700`**.
- **D14 at-rest locking test:** assert every vault file is `0o600` and every dir is
  `0o700`; then **flip one file to `0o644`** and assert the check goes RED (the test
  must prove it can fail, not just pass on a clean tree). State plainly in a comment
  that at-rest **encryption** is out of scope — this enforces permissions only.

### FIX-11 — ADP key strip is a fail-open denylist (HIGH, gap M9)
`isQratumOnlyExportKey` denylists six known keys plus an `x-qratum-` prefix; any
**other** internal key — a new field, a nested annotation inside a `tool_calls[].input`
map — passes straight into the external ADP. Convert the ADP (and any external
export builder) to an **allowlist projection**: build the output from named fields
only; never pass an arbitrary internal map through. The locking test injects a
**random unknown** internal key into a nested input map and asserts it is **absent**
from the ADP (proving allowlist, not denylist, behavior).

### FIX-12 — hook captures arbitrary files via an untrusted path (HIGH)
`resolveHookTranscriptPath` accepts any absolute/relative `transcript_path` from the
incoming hook JSON and `ArchiveFile` opens it with `os.Open` (follows symlinks),
unbounded, with no confinement. A symlinked or attacker-chosen path pulls arbitrary
files (`~/.ssh/id_rsa`, a device) into the immutable, never-deletable vault. Fix:
- **Confine** the resolved path to an allowlist of roots: `~/.claude/projects` and
  the resolved cwd subtree. After cleaning, reject any path outside those roots
  (resolve to absolute and check prefix containment; reject `..` escapes).
- **Reject symlinks:** `Lstat` the final path (and ideally open with `O_NOFOLLOW`)
  and refuse if it is a symlink.
- **Reject non-regular files** (devices, FIFOs, dirs): refuse unless
  `Mode().IsRegular()`.
- **Cap the read** with `io.LimitReader` so an oversized/again-grown file can't be
  copied unbounded.
- Each rejection is **recorded** as a degraded capture event (never swallowed) and
  the hook still **exits 0** (fast-hook rule: surface, don't crash). The locking
  test feeds hostile-payload fixtures (symlink-to-secret, `..` traversal,
  non-regular file, oversized) and asserts each is rejected, recorded, and exit 0,
  and that nothing was written into the blob store.

### D1 import-isolation extraction — `internal/capture` (the D1 carve-out)
The capture path (the hook's stdin → event → confined file-copy) currently lives in
package `main` (`cmd/qrt`), so its import surface is package-`main`-coarse and the
trust gate cannot prove capture never reaches the network. Extract that path into a
new `internal/capture` package whose **only** allowed imports are `crypto/sha256`,
`os`, `io`, and the `vault`/`workspace` packages (plus their transitive stdlib —
state the exact pinned set in the golden). Then:
- **Pin the allowed import set in a committed golden** (an `imports.golden` listing,
  or an in-test allowed-set constant) computed from the package's actual import
  graph via `go/parser`/`go/types` or `go list -deps` (stdlib / `os/exec`-`go`
  only; no new dependency). The test fails loudly if the real import set diverges
  from the golden in either direction.
- **Negative test:** add a fixture/scenario where introducing a `net` (or
  `net/http`) import into `internal/capture` flips the gate **RED** — proving the
  pin actually catches a new network import, not just that the current set matches.
This is the explicit P1 carve-out of D1's import-isolation; the rest of D1 (capture
fidelity, the speed gate, disk-full FIX-14) stays in later prompts. The hook stays
fast (no parse/network/LLM); this is a structural move plus a pin, not new behavior.

### No-secret-in-committed-golden lint (scans git HISTORY, not just the tree)
`git@github.com:edictum-ai/qratum.git` has been in committed fixtures since an early
commit; re-redacting the golden (FIX-2) removes it from the working tree but **not**
from history, so a fresh clone still exposes it. Add a lint (a Go test, or a
`//go:build` tool wired into `make verify` later) that:
- scans the **working tree** committed fixtures for known internal identifiers
  (`edictum-ai/qratum.git`, real-looking 40-hex head SHAs, `git@…:…` SSH remotes,
  any `sk-…`/`ghp_…`-shaped token), AND
- scans **git history** via `git log -p -- fixtures/` (through `os/exec`, stdlib
  only) for the same patterns.
- Fails loudly listing offending commit + path. If history already contains the
  identifier (it does), the lint should **report it as a known finding** with a
  clear message that history rewrite/relocation is required — do **not** silently
  pass, and do **not** attempt a history rewrite yourself (that is an maintainer-only
  decision; STOP and report it).

## Acceptance criteria

- FIX-1: no residual secret token survives any `=>`/`==>` assignment after
  redaction; locking test present and GREEN.
- FIX-2: the six git/time/event fields are **dropped** from report/DTO/ADP and
  **redacted** in the redacted-session JSON; the SSH-remote pattern catches
  `git@host:org/repo.git`; the committed golden is re-redacted and carries no real
  remote/branch/SHA/timestamp; locking test GREEN.
- FIX-4 / D11: all derived artifacts land under `~/.qratum/sessions/<session_id>/`
  (`QRATUM_HOME` override); the repo is recorded as session metadata and
  `sessions list`/`review` can filter by it; a run from inside a git repo writes
  **nothing** to the working tree and no artifact path resolves outside
  `QRATUM_HOME`; locking test GREEN.
- D1 import-isolation: the capture path lives in `internal/capture` importing only
  `crypto/sha256`/`os`/`io`/`vault`/`workspace`; the allowed import set is pinned in
  a committed golden; a negative test proves a `net` import flips the gate RED;
  locking test GREEN.
- FIX-5: two distinct digests sharing a 12-char prefix both store as distinct refs;
  locking test GREEN.
- FIX-6: `O_EXCL` create; N concurrent captures → N distinct events, none lost;
  `go test -race` clean.
- FIX-8: root + all dirs `0o700`, all blob/ref/event/state files `0o600`; D14
  locking test passes on a clean tree and goes RED when a file is flipped to `0o644`.
- FIX-11: ADP is an allowlist projection; an injected unknown nested internal key is
  absent from the ADP; locking test GREEN.
- FIX-12: hostile `transcript_path` payloads (symlink, traversal, non-regular,
  oversized) are confined/rejected, recorded, exit 0, nothing enters the blob store;
  locking test GREEN.
- The no-secret golden lint runs over the working tree **and** git history and fails
  loudly on a known internal identifier; the working-tree scan is GREEN after FIX-2.
- `make verify` is green (or, for any KNOWN-RED history finding, the lint reports it
  explicitly with a STOP-and-report note rather than silently passing).
- No new third-party Go dependency; tests never touch the real home (`QRATUM_HOME`).

## Decision Trace

- 2026-06-15 (the maintainer): redactor field-leak fix = **HYBRID** (drop git/time/event
  from shareable artifacts, redact them in the redacted JSON; fix the `=>` bug
  regardless; re-redact the golden). Per `verification-and-trust-gate.md` §"Decisions
  already taken" and FIX-2.
- 2026-06-15 (the maintainer): the two newly found leaks (world-readable raw blobs / M4,
  ungoverned raw egress / M5) are **spec-now / fix-with-P2** — M4 is FIX-8 / D14 in
  this task; M5 (FIX-9) is a later prompt.
- 2026-06-15 (the maintainer): at-rest **encryption is out of scope** — the gate enforces
  file **permissions**, not encryption (Threat Model).
- Tier 0 sequencing (§4) names FIX-1/2/5/6/8/11/12 as the cheapest, highest-value
  first batch, each with its locking test.

## Behavior Contract

- [ ] FAILS the task if it builds any later-prompt deliverable (canary harness,
  scorecard, `data_class`, FIX-3/7/9/10/13/14/15/16). FIX-4 / D11 and the
  `internal/capture` import-isolation carve-out are owned here, not later-prompt.
- [ ] FAILS if any fix lands **without** a locking test (each fix's test must go RED
  if the fix is reverted; FIX-6's must run clean under `-race`, FIX-8's must go RED
  when a file is flipped to `0o644`).
- [ ] FAILS the fast-hook rule if FIX-12 adds parse/network/LLM to the hook; the hook
  must still do only stdin → event → confined file-copy and exit 0 on rejection.
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency
  (the history lint uses `git` via `os/exec` + stdlib only).
- [ ] FAILS on real-home mutation in tests/CI (must use `QRATUM_HOME`).
- [ ] FAILS review if the re-redacted golden still contains a real internal
  identifier, or if the history lint silently passes on a known-leaking commit.

## Drift Handling

- If the code at the cited file:line has moved or diverged from the spec's
  description in a way the spec does not anticipate, **stop and report** with the
  new location and what changed; do not guess a different fix.
- If FIX-2's "drop from shareable artifacts" changes a committed report/DTO/ADP
  golden, regenerate that golden **and say so explicitly** (an intentional output
  contract change), and confirm the dropped fields are absent.
- If git history rewrite would be required to fully clear a leaked identifier, that
  is an maintainer-only decision — report it, do not rewrite history.

## Verification

```sh
# Build + unit/golden tests + race (the FIX-6 concurrency test):
make -C . build
make -C . test
make -C . test-race

# Targeted: each fix's locking test (names indicative — match what you add):
( cd . && \
  go test ./cmd/qrt/ -run 'Redact|ArrowSeparator|GitFieldHybrid|SSHRemote|ADPAllowlist|HookPathConfinement' -v )
( cd . && \
  go test ./internal/vault/... ./internal/workspace/... -run 'RefIDCollision|AtRestPerms|CaptureRace' -race -v )

# The no-secret golden lint over working tree AND git history:
( cd . && go test ./cmd/qrt/ -run 'NoSecretInGolden' -v )

# Confirm the committed golden no longer carries the real identifier:
grep -RnE 'edictum-ai/qratum\.git|git@github\.com' \
  ./fixtures/redaction/ ; echo "exit=$? (want non-zero / no match in tree)"

# Full local CI mirror (must stay green; do not weaken any check):
make -C . verify
```

VERIFY GAP: confirm the exact test package layout. `internal/vault` and
`internal/workspace` have their own `vault.go`/`workspace.go`; the FIX-5/FIX-8
tests likely live there, while FIX-1/2/6/11/12 tests live in `cmd/qrt`. Confirm
whether `secret-session.redacted.golden.json` is consumed by an existing
`cmd/qrt` golden test (it is the FIX-2 contract) before regenerating it, so the
regen path is wired, not hand-edited.

## Slop Review

- [ ] Did every Behavior Contract item get a behavioral test or explicit
  evidence path, and does the evidence go RED when the fix is reverted?
- [ ] Are missing or invalid inputs loud failures with operator-visible output,
  never swallowed or hidden behind a green `make verify`?
- [ ] FIX-1: a test proves **no residual secret token** survives `=>`/`==>`
  assignments — not just that the arrow itself is gone.
- [ ] FIX-2: shareable artifacts (report/DTO/ADP) **drop** the six fields; the
  redacted JSON **redacts** them; the SSH-remote pattern is real and tested; the
  committed golden is re-redacted and clean (tree).
- [ ] FIX-4 / D11: artifacts moved to `~/.qratum/sessions/<session_id>/`, nothing
  written relative to `os.Getwd()`; the repo is **metadata**, not a write location;
  the test runs from inside a git repo and asserts the working tree is untouched and
  no path escapes `QRATUM_HOME` — not just that the central path also exists.
- [ ] D1 import-isolation: capture is in `internal/capture` with the pinned import
  set in a committed golden; the negative test genuinely introduces a `net` import
  and the gate goes **RED** (proves the pin can fail, not just match today).
- [ ] FIX-5: the test uses two genuinely distinct digests sharing a 12-char prefix,
  not a re-hash of the same blob.
- [ ] FIX-6: `O_EXCL` (no stat-then-rename window); `-race` clean; the test actually
  hammers with goroutines and asserts zero lost captures.
- [ ] FIX-8: every vault file `0o600`, every dir + root `0o700`; the D14 test goes
  **RED** when a file is flipped to `0o644` (proves it can fail).
- [ ] FIX-11: allowlist projection; an **unknown** nested internal key is absent
  from the ADP (denylist would let it through).
- [ ] FIX-12: hostile payloads (symlink, `..`, non-regular, oversized) rejected,
  recorded, exit 0, nothing enters the vault; hook stayed fast (no parse/net/LLM).
- [ ] The golden lint scans **git history**, not just the working tree, and fails
  loudly / reports a known-leaking commit rather than silently passing.
- [ ] No new third-party Go dependency without supply-chain evidence; tests never
  touch the real `~/.claude` or `~/.qratum` (`QRATUM_HOME` set).
- [ ] No later-prompt deliverable was smuggled in; `make verify` passes without
  weakening a check.

Reviewer guidance:

> Review this change against `verification-and-trust-gate.md` §2 (FIX-1, FIX-2,
> FIX-5, FIX-6, FIX-8, FIX-11, FIX-12) and §3 D14. Confirm each fix has a test that
> goes RED if the fix is reverted, that FIX-6 is `-race` clean, that FIX-8's at-rest
> test fails on a flipped `0o644` file, that FIX-11 is an allowlist (an injected
> unknown nested key is absent from the ADP), and that FIX-12 rejects symlink/
> traversal/non-regular/oversized paths while keeping the hook fast and exit 0.
> Confirm the committed golden is re-redacted (no real remote/branch/SHA/timestamp)
> and that the no-secret lint scans git history. Flag any new third-party Go
> dependency, any real-home mutation in tests, any later-prompt deliverable pulled
> in early, or any weakened check.

## Stop conditions

- STOP if the **P2-VERIFY-TRUST-GATE** milestone is not explicitly unlocked by
  the maintainer. `AGENTS.md` shows it as PROPOSED (awaiting acceptance); this task builds
  runtime/security code and is gated on that unlock.
- STOP if any fix appears to require a new third-party Go dependency — report it as
  a supply-chain decision for the maintainer rather than adding it.
- STOP and report (do not rewrite history) if clearing a leaked identifier from git
  history is required — that is an maintainer-only decision.
- STOP before running anything against the real `~/.claude` or `~/.qratum`.
- STOP if `make verify` (including `-race`) cannot be made green without weakening a
  check — report the failure, do not suppress it.
- STOP if the cited code has moved/diverged such that a fix no longer maps cleanly —
  report the new location rather than guessing.

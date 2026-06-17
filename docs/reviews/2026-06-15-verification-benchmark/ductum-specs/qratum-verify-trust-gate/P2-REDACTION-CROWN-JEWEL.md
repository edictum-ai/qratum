# P2 — Redaction Crown Jewel + Trust Boundary (runtime/verification)

Package: `qratum-verify-trust-gate` · Prompt 2 of 7 · Depends on: P1 ·
Scope: runtime/verification · Deliverable: the D3 reflection-canary harness +
planted-secret corpus, the D4 encoding-aware no-leak boundary checker, FIX-10
standalone re-redaction, and the FIX-1 residual-token assertion — all GREEN over
the covered corpus.

## Objective

Build the load-bearing security gate of this milestone: prove that no secret
survives into any artifact qratum can show to anyone. Today "verified" only
means the build is green, and a green build was caught shipping real credential
leaks. This task stands up the proof that actually runs secrets through the
pipeline.

Two halves:

- **D3 — Redaction safety (the crown jewel).** Inject a unique marker into every
  text field of a fully-populated test session, run the real pipeline end to end,
  and prove not one marker survives into any output. The hard part is making the
  test honest: a marker that the redactor itself would eat, a harness that skips a
  field it can't reach, or a checker that is blind to HTML/JSON escaping would all
  read false-green. This task closes each of those holes by construction.
- **D4 — Trust-boundary enforcement.** One shared, encoding-aware checker that
  scans the actual bytes of every artifact, plus an allowlist-projection assertion
  for the ADP export and a no-network import gate on the refinery.

Plus the two coupled defect fixes the gates lock shut: **FIX-10** (standalone
`evidence`/`review`/`report` must re-redact or reject a raw session) and the
**FIX-1 residual-token assertion** (no secret survives after any placeholder).

Plus **FIX-7 — close the cheap evasion gaps and own the known-miss ledger.** P2
already owns the redactor and the corpus, so the redactor's coverage is P2's to
extend. Two parts: (a) widen the redactor to catch the cheap, high-confidence
classes it misses today — AWS keys that contain a `/`, relative paths like
`./config/prod.env`, home paths like `~/.aws/credentials`, and space-separated
assignments (`token foo`); (b) own a single version-controlled known-miss file
that lists the genuinely-hard classes the redactor still cannot catch
(unicode / zero-width tricks, base64-of-a-secret, bare `AKIA…`, and the
`sk_`/`pk_`/`AIza`/`glpat`/`SG.` prefixes), each marked as an expected miss
(`xfail`) with a short residual-risk note saying what survives and why it is
accepted for now. The point is honesty: catch what is cheap to catch, and write
down — in the repo, not in memory — exactly what still leaks.

Build only the D3/D4 harness and the FIX-10 fix. Do not build the schema work
(D9, that is a sibling prompt), the vault-integrity verbs, the scheduler, or any
new product surface. This prompt produces the corpus, the harness, the checker,
and one targeted code fix; the `make trust` skeleton that runs them is wired in a
later prompt.

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` — the contract. Sections
  §1 "Why" (R1/R2/R3), §"Threat Model" (the pipeline-leaking-to-itself principal),
  §2 FIX-1 / FIX-10 / FIX-11, §3 "D3 — Redaction safety (CROWN JEWEL)" and
  "D4 — Trust-boundary enforcement".
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` — the line-level
  evidence: M1 (self-redacting canary), M2 (no harness self-test), M3 (map keys /
  non-string scalars), M6 (standalone commands don't re-redact), M9 (ADP fail-open
  denylist), and the "scan correctness and re-introduction" theme (re-introduced
  fields, encoding-blind scan, non-independent terminal scan, `redactAny` panic).
- `qratum/docs/reviews/2026-06-15-verification-benchmark/BENCHMARK.md` §2 (full D3
  acceptance detail).
- `qratum/AGENTS.md` (fast-hook rule, supply-chain rule, fixture/golden tests,
  Definition of Done, Ductum Factory Rules).
- `qratum/cmd/qrt/redact.go` — `secretAssignmentPattern` (line 22),
  `highEntropyPattern` (line 23), `looksHighEntropy` (line 448), `redactString`,
  `redactQratumSession`, `redactAny` (line 264; map-value recursion at line 276,
  the non-string `default:` at 285, the unchecked `.(map[string]any)` assertions
  at lines 226/249).
- `qratum/cmd/qrt/export.go` (`isQratumOnlyExportKey`, line ~373 — the denylist),
  `qratum/cmd/qrt/evidence.go` (line 144 — `readQratumSessionFile`, no redact),
  `qratum/cmd/qrt/report.go` (line 57 — same), and `export.go:128` (the only
  command that does redact today).
- `qratum/cmd/qrt/daemon.go` (`runDaemonOnce`, the real end-to-end entrypoint the
  canary must drive).
- `qratum/fixtures/redaction/secret-session.input.json` and
  `…/secret-session.redacted.golden.json` (the existing redaction fixtures).
- `qratum/fixtures/claude-code/transcript-with-secret.jsonl` (wired in here for
  the first time).

## Allowed scope

- A new trust-harness package (Go-native, stdlib-only) for the D3 reflection
  canary and the D4 shared checker — under `cmd/trustbench` or behind a
  `//go:build trust` tag, matching the spec's §3 shape. It drives the **real**
  CLI entrypoints (`runDaemonOnce` and the standalone subcommands), never a
  shortcut.
- New hand-authored fixtures for the cases reflection cannot plant
  (map-key tokens, stringified-number tokens) and the precision tripwire corpus.
- The minimal code fixes the gates demand:
  - **FIX-10:** standalone `evidence`/`review`/`report` reject or internally
    redact a non-redacted session.
  - **FIX-1 residual-token** assertion support (the `=>` value-capture fix lands
    in the Tier-0 sibling prompt; here, lock it with the residual-token check —
    if the `=>` fix is not yet present, that case is KNOWN-RED, tracked, not a
    silent skip).
  - **FIX-11 allowlist** assertion for the ADP (the projection rewrite, if not
    already landed, is in scope here as the D4 boundary fix it demands).
  - `redactAny` map-key re-keying + non-string handling + **fail-closed**
    (comma-ok, not panic) on malformed `Input`/`Provenance` (M3 + the panic gap).
  - **FIX-7 (the cheap classes):** extend the redactor to catch AWS keys
    containing `/`, relative `./config/prod.env`-style and home
    `~/.aws/credentials`-style paths, and space-separated assignments
    (`token foo`). These are the high-confidence misses that fall to small,
    targeted pattern changes in `redact.go` — no new dependency, no PII work.
  - **FIX-7 (the known-miss ledger):** add and own a single version-controlled
    known-miss / `xfail` file under the corpus listing the genuinely-hard classes
    that still leak (unicode / zero-width, base64-of-a-secret, bare `AKIA…`,
    `sk_`/`pk_`/`AIza`/`glpat`/`SG.` prefixes), each with a one-line residual-risk
    note. These cases are tracked KNOWN-RED, never silently skipped.
- Wire `fixtures/claude-code/transcript-with-secret.jsonl` into the harness.

## Non-goals

- No D9 schema work (recursive `additionalProperties`, the mini-validator, the
  `data_class` field) — that is a sibling prompt. Do not add `data_class` here.
- No `make trust` target / scorecard JSON / three-state gate model — a later
  prompt wires the harness into CI. This prompt makes the harness pass.
- No vault-integrity verbs (`gc`, erasure, perms), no scheduler, no recoverability
  rewire (FIX-3 / D6a) — other prompts.
- No new third-party Go dependency. The harness, the canary planter, and the
  encoding-aware checker are stdlib-only (`reflect`, `encoding/json`,
  `html`, `crypto/sha256`, `regexp`). A dependency is an explicit supply-chain
  decision = STOP and report.
- No PII detection. The redactor stays credentials-only and Go-native.
- No real secrets or real transcripts committed — canary/synthetic only.
- Do NOT run anything against the real `~/.claude` or `~/.qratum`.

## Implementation notes

### The canary token (M1 — it must not redact itself)
- The marker must **provably evade all 8 redaction classes by construction**:
  lowercase-alpha only, `< 32` characters, single character class, **no
  separator** (e.g. `qratumcanaryNNNN`, where `NNNN` is a per-field zero-padded
  counter). A UUID-v4 is **forbidden** as the canary: its hyphens are matched by
  `highEntropyPattern` and `looksHighEntropy` excludes only `/` and `\`, so a
  UUID self-redacts and the gate passes for the wrong reason.
- Keep the hyphenated UUID strictly as a **precision tripwire** (it must
  *survive*), never as the canary. One token class may not be both "must vanish"
  and "must survive."

### The reflection canary harness (D3)
- Walk a fully-populated `qratumSession` struct with `reflect`, planting a unique
  token into **every** string field (recursing through `git`, `turns`,
  `tool_calls`, nested `input` maps, `provenance`). Drive `runDaemonOnce`
  end-to-end. Assert **zero** tokens survive into: the redacted session JSON,
  evidence, review, report HTML, ADP, and the capture event. Recall is
  **binary 100%** on the covered corpus — any survivor blocks.

### Harness self-test (M2 — prove it can both pass and fail)
- **(a) Known-positive:** deliberately route one field around `redactString` and
  assert the gate goes RED. Proves the gate *can* fail.
- **(b) Canary-alone:** feed the chosen canary through `redactString` and assert
  it returns **unchanged**. Proves the marker is not self-redacting.
- **(c) Panic-loudly:** the planter **panics** on any field kind it cannot reach
  (unexported, nil pointer, unhandled type) rather than silently skipping — a
  skipped field under-counts numerator and denominator identically and reads
  false-green.

### Map keys + non-string scalars (M3 — what reflection can't reach)
- `redactAny` copies map *keys* verbatim (line 276) and returns non-string
  scalars unchanged (line 285). `tool_calls[].input` and `provenance` are
  `map[string]any` — the highest-value attacker surface. Re-key map outputs
  through `redactString`; scan/coerce non-string scalars (or prove inert with a
  positive test).
- Add a **hand-authored** fixture that plants tokens **as map keys** and as
  stringified numbers inside `tool_calls[].input` (struct reflection cannot plant
  a dynamic map key). Drive it through the real daemon.
- `redactAny` must fail **closed** (comma-ok form, return a typed error — not
  panic) on a malformed `Input`/`Provenance` shape (lines 226/249). A panic leaves
  the raw `.normalized.json` on disk while the redacted artifact never gets
  written — a fail-open leak.

### Re-introduction coverage (downstream copies a struct field back in)
- Downstream artifacts copy fields straight from the struct (`evidence.go` builds
  `Summary` and copies `started_at`/`ended_at`/`source_event_id` from the session,
  not from the redacted JSON). So FIX-2's "redact in the redacted-session JSON"
  step alone is insufficient. Add a canary case whose **only** planted token is in
  a re-introduced field, and make the terminal per-artifact byte-scan a **hard**
  gate (see D4).

### Per-class recall + precision corpus + determinism
- Report per-class recall **including the leaking classes**; assert the FIX-1
  residual-token property (no secret token survives after any placeholder).
- Precision corpus, **hand-seeded** with a bare 40-hex SHA, a hyphenated UUID,
  and a `sha256:` digest — each must **survive**. Over-redaction here is a regex
  bug to fix, **not** a budget to inflate.
- Placeholder numbering must be deterministic and idempotent (same input → same
  placeholders; re-run changes nothing).

### FIX-7 — close the cheap evasions; ledger the hard ones
- **Extend the redactor** (`redact.go`) for the cheap, high-confidence classes it
  misses today, each with a focused corpus case that must now redact:
  - AWS-style keys whose value contains a `/` (today `looksHighEntropy` excludes
    `/`, so a key with a slash slips through).
  - Relative config paths (`./config/prod.env`) and home paths
    (`~/.aws/credentials`) that point at credential files.
  - Space-separated assignments (`token foo`, `password hunter2`) that the
    `=`/`:` assignment pattern does not catch.
  Keep these credentials-only and Go-native; do not widen into PII. Re-run the
  precision corpus afterward — these must not start eating the tripwires
  (a 40-hex SHA, a hyphenated UUID, a `sha256:` digest must still survive).
- **Own the known-miss ledger.** Add one version-controlled file under the
  corpus (e.g. `fixtures/redaction/known-misses.json` or a sibling `.md`) that
  lists the genuinely-hard classes the redactor still cannot catch — unicode /
  zero-width obfuscation, base64-of-a-secret, bare `AKIA…` access-key IDs, and
  the `sk_`/`pk_`/`AIza`/`glpat`/`SG.` provider prefixes. For each, record: an
  example shape, that it is an expected miss (`xfail`), and a one-line
  residual-risk note (what survives and why it is accepted for this milestone).
  The harness reads this file so each known miss is a tracked KNOWN-RED, not a
  silent skip; closing one is deleting its line, never re-redacting a golden to
  hide it.

### FIX-10 — standalone subcommands re-redact or reject
- `report.go:57` and `evidence.go:144` read a session file and build artifacts
  with **no** `redactQratumSession` call; only `export.go:128` redacts. So
  `qrt evidence <raw .normalized.json>` emits raw secrets.
- Fix: standalone `evidence`/`review`/`report` must either **reject** a
  non-redacted session (require `pipeline_status == redacted`) or redact
  internally before building. Pin which (rejecting is the simpler, louder
  default; state the choice in the Decision Trace).

### D4 — the shared encoding-aware no-leak checker
- One checker, driven by the reflection token set, over **all** artifact
  byte-streams. It is **encoding-aware**: HTML-unescape the report before
  scanning; JSON-unmarshal ADP/DTO and scan the decoded string *values*. A secret
  containing `< > & " + /` could otherwise survive escaped and read clean.
- Per R2: for a field an artifact **carries**, assert canary-absent **and**
  placeholder-present. For a field an artifact structurally **drops**, assert the
  drop and do **not** credit it as redaction.
- **Allowlist projection (FIX-11):** the ADP (and any external export) builds
  output from named fields only — never passes an arbitrary internal map through.
  Today `isQratumOnlyExportKey` (export.go:~373) is a fail-open denylist of six
  keys plus `x-qratum-`. Assert: inject a **random unknown** internal key into a
  nested input map → it is **absent** from the ADP (proves allowlist, not
  denylist).
- The **terminal per-artifact byte-scan is a HARD gate**, not optional
  defense-in-depth. Give it a genuinely independent generic high-recall detector
  (broad `AKIA`/`AIza`/`glpat`/`pk_`/`SG.` + entropy) distinct from the redactor's
  own patterns — **or** label it a regression tripwire in the residual (otherwise
  it inherits the redactor's recall exactly; state which you did).
- **No-network import gate on the refinery:** assert the refinery/redaction path
  does not import `net`. The `internal/capture` import-isolation extraction is
  **owned by P1** (P2 depends on it, P2 does not build it). Until P1's extraction
  lands, the import gate is package-`main`-coarse — state that in the harness
  output, don't fake a finer gate.

### Tests
- All tests honor `QRATUM_HOME` (point it at `t.TempDir()`); never touch the real
  `~/.claude` or `~/.qratum`. Fixture/golden-driven per AGENTS.md.
- Wire `fixtures/claude-code/transcript-with-secret.jsonl` into the harness as a
  real secret-bearing input fed end-to-end.

## Acceptance criteria

(from `verification-and-trust-gate.md` §3 D3/D4 + §7)
- The reflection canary runs through the **real daemon**, uses a
  **non-self-redacting** canary (the canary-alone self-test passes), and survives
  **zero** tokens into the redacted session, evidence, review, report HTML, ADP,
  and capture event.
- The harness self-test proves the gate can fail (known-positive → RED), proves
  the canary is inert (canary-alone → unchanged), and **panics** on any
  unreachable field.
- Map-key tokens and stringified-number tokens (hand-authored fixture) are caught
  or proven inert; `redactAny` re-keys map keys and fails **closed** (no panic) on
  malformed `Input`/`Provenance`.
- A canary planted **only** in a re-introduced field (e.g. `evidence.summary`'s
  copied `started_at`) is caught by the hard terminal byte-scan.
- Precision tripwires (bare 40-hex SHA, hyphenated UUID, `sha256:`) **survive**;
  the FIX-1 residual-token property holds (no secret after any placeholder).
- The D4 checker is encoding-aware (HTML-unescape report; JSON-unmarshal
  ADP/DTO); a random unknown internal key injected into a nested map is **absent**
  from the ADP (allowlist proven).
- FIX-10: feeding a raw secret-bearing session to each standalone command yields
  zero canary survival **or** a clean rejection.
- FIX-7: the redactor now catches the cheap classes (AWS key containing `/`,
  relative `./config/prod.env` and home `~/.aws/credentials` paths,
  space-separated assignments), each proven by a corpus case, **without** newly
  eating the precision tripwires. The version-controlled known-miss / `xfail`
  file lists every genuinely-hard class still leaking (unicode / zero-width,
  base64-of-secret, bare `AKIA…`, `sk_`/`pk_`/`AIza`/`glpat`/`SG.` prefixes) with
  a residual-risk note, and the harness reads it as tracked KNOWN-RED — no silent
  skip.
- `transcript-with-secret.jsonl` is wired in and exercised end-to-end.
- No new third-party Go dependency; `go test -race` clean; `make verify` green
  (or only the explicitly-tracked KNOWN-RED cases red, never a silent skip).

## Decision Trace

- M1 (2026-06-15): canary is a non-matching token (`qratumcanaryNNNN`), not a
  UUID-v4; UUID stays the precision tripwire.
- M2: the canary harness gets a mandatory self-test (the schema validator already
  had one; the harness did not).
- M3: `redactAny` re-keys map keys and handles non-string scalars; fails closed.
- FIX-10: standalone commands reject or internally redact — pin the choice and
  record it here.
- FIX-11: ADP becomes an allowlist projection.
- Runtime build requires an explicit milestone unlock to `P2-VERIFY-TRUST-GATE`
  (the maintainer).

## Behavior Contract

- [ ] FAILS the task: using a UUID-v4 (or any self-redacting token) as the canary.
- [ ] FAILS if the planter silently skips an unreachable field instead of
  panicking (M2c); evidence: the panic-loudly self-test.
- [ ] FAILS if the no-leak checker scans raw bytes only (encoding-blind); evidence:
  the escape-triggering fixture survives the HTML/JSON-decoded scan.
- [ ] FAILS if the ADP key strip stays a denylist; evidence: the unknown-key
  injection test finds the key in the ADP.
- [ ] FAILS if `redactAny` panics (rather than fails closed) on malformed
  `Input`/`Provenance`.
- [ ] FAILS if any standalone command emits raw secrets from a non-redacted
  session (FIX-10 unfixed).
- [ ] FAILS without a supply-chain decision: any new third-party Go dependency.
- [ ] FAILS on real-home mutation in tests/CI; evidence: `QRATUM_HOME` set to a
  temp dir.

## Drift Handling

- If `redact.go` line numbers / function shapes have moved since 2026-06-15,
  re-locate by name (`redactAny`, `highEntropyPattern`, `secretAssignmentPattern`,
  `isQratumOnlyExportKey`) and report the drift; do not assume the line numbers.
- If the `=>` value-capture fix (FIX-1) or the FIX-11 ADP rewrite already landed
  in a sibling prompt, do not re-do it — just assert it with the residual-token /
  unknown-key tests and note it in the Decision Trace.
- Update fixtures/golden only when an output contract intentionally changes, and
  say so. Re-redacting a golden to hide a leak is a FAIL, not a fix.

## Verification

```sh
# Full local CI mirror (build, vet, lint, test, race, demo, dogfood, security):
make -C . verify

# Race-clean is mandatory for the concurrency-sensitive harness:
go -C . test -race ./...

# Run the trust harness directly over the covered corpus (build-tagged or
# cmd/trustbench, per the package shape chosen in this prompt):
go -C . test -tags trust -run 'D3|D4|Canary|Boundary|Redact' ./...

# End-to-end secret proof in an isolated workspace (no real home touched):
export QRATUM_HOME="$(mktemp -d)"
make -C . build
# feed the secret-bearing transcript through the real daemon, then assert no
# canary/secret survives into any artifact under $QRATUM_HOME/sessions/<id>/:
./bin/qrt evidence \
  "$QRATUM_HOME"/sessions/*/normalized.json   # must reject or emit redacted
unset QRATUM_HOME
```

VERIFY GAP: confirm the exact daemon entrypoint name used to drive the canary
end-to-end (`runDaemonOnce` per `daemon.go`), and confirm the artifact filenames
the checker must scan under `~/.qratum/sessions/<session_id>/`
(`redacted.json`, `evidence.json`, `review.json`, `report.html`,
`session.adp.jsonl`) match what the daemon actually writes after the FIX-4
central-home move — if FIX-4 has not landed in a sibling prompt, the daemon may
still write `./.qratum/sessions/*.normalized.json` and the harness must read from
there and say so, not assume the post-FIX-4 layout.

## Slop Review

- [ ] Did every Behavior Contract item get a behavioral test or explicit
  evidence path, and does the evidence fail loudly on an unredacted fixture?
- [ ] Are missing or invalid inputs loud failures with operator-visible output,
  never silently skipped by the canary planter or no-leak checker?
- [ ] The canary provably evades all 8 redaction classes by construction
  (lowercase-alpha, `<32`, no separator); the canary-alone self-test passes.
- [ ] The harness self-test proves the gate can fail (known-positive → RED) and
  the planter panics — not skips — on any unreachable field.
- [ ] Map keys and non-string scalars are covered by a **hand-authored** fixture
  (reflection can't plant a dynamic map key); `redactAny` fails closed, not panics.
- [ ] The re-introduction case is covered and the terminal byte-scan is a HARD
  gate, not optional.
- [ ] The checker is encoding-aware (HTML-unescape report, JSON-unmarshal
  ADP/DTO), not a raw-byte grep.
- [ ] The ADP is an allowlist projection; an unknown internal key is absent.
- [ ] FIX-10: standalone commands reject or re-redact; no raw secret escapes.
- [ ] FIX-7: the cheap classes (AWS key with `/`, relative/home credential paths,
  space-separated assignments) now redact, each with a corpus case, and the
  precision tripwires still survive; the genuinely-hard classes are tracked in a
  version-controlled known-miss / `xfail` file with residual-risk notes, read by
  the harness as KNOWN-RED — not silently skipped.
- [ ] Precision tripwires survive; no precision budget was inflated to hide
  over-redaction.
- [ ] No new third-party Go dependency; `go test -race` clean; tests never touch
  the real `~/.claude`/`~/.qratum` (`QRATUM_HOME` set).
- [ ] `make verify` passes without weakening a check; KNOWN-RED cases are tracked,
  never silently skipped.

Reviewer guidance:

> Review the D3/D4 trust harness against `verification-and-trust-gate.md` §3 and
> `GAPS.md` M1/M2/M3/M6/M9 + the re-introduction/encoding/independence findings.
> Confirm: the canary cannot redact itself (try the UUID — it must NOT be the
> canary); the harness self-test forces a RED on a deliberately-unredacted field
> and panics on an unreachable one; map-key and stringified-number tokens are
> covered by hand-authored fixtures and `redactAny` fails closed on a malformed
> shape; a token planted only in a re-introduced field is still caught; the
> checker decodes HTML and JSON before scanning; the ADP is an allowlist
> projection that drops an unknown injected key; the standalone
> evidence/review/report commands reject or re-redact a raw session; precision
> tripwires survive; `transcript-with-secret.jsonl` is fed end-to-end through the
> real daemon. Flag any self-redacting canary, any silently-skipped field, any
> raw-byte (encoding-blind) scan, any denylist ADP strip, any `redactAny` panic
> path, or any new third-party Go dependency.

## Stop conditions

- STOP if P1 has not landed (specs/contracts still contradictory) — this depends
  on P1.
- STOP if the Qratum milestone is not unlocked to `P2-VERIFY-TRUST-GATE` by the maintainer
  — runtime/verification build is gated; do not build against a stale pointer.
- STOP if any part of the harness or checker appears to require a third-party Go
  dependency — report it as a supply-chain decision for the maintainer rather than adding
  it.
- STOP before running any command against the real `~/.claude` or `~/.qratum` —
  use `QRATUM_HOME` pointed at a temp dir.
- STOP if `make verify` or `go test -race` cannot be made green (excluding the
  explicitly-tracked KNOWN-RED cases) without weakening a check — report the
  failure, do not suppress it.

# P1 — Qratum Spec Hygiene (docs only)

Package: `qratum-memory-foundation` · Prompt 1 of 2 · Depends on: — ·
Scope: docs/spec · Deliverable: accepted spec docs + ADR 0010

## Objective

Make the on-disk Qratum specs match the accepted post-review direction
(vault-first), in place, with no runtime/code changes. The on-disk specs
currently disagree with each other: `SPEC.md` still calls the old operational
model canonical and lists raw archive as a P0 non-goal, while
`qratum-vault-first.md` is the accepted revision. This prompt resolves that.

This is a **documentation-only** task. No Go code, schemas, fixtures, or
Makefile changes.

## Read first

- `qratum/AGENTS.md` (source-of-truth rules; "documentation-only changes,
  tests are not required")
- `qratum/SPEC.md`
- `qratum/specs/current/qratum-vault-first.md` (the accepted revision — the
  contract for this work; its "Spec Hygiene" section is the exact edit list)
- `qratum/specs/current/operational-model-redesign.md` (the file being edited,
  in full)
- `qratum/specs/current/memory-curation-pipeline.md` (already superseded; leave
  its history sections intact)
- `qratum/docs/decisions/` (ADR format reference, 0001–0009)
- `qratum/docs/reviews/2026-06-12-memory-architecture/PROPOSAL.md` and
  `BACKLOG.md` section A

## Allowed scope

- Edit `specs/current/operational-model-redesign.md` in place.
- Edit `SPEC.md` pointers.
- Create `docs/decisions/0010-vault-first-and-direct-gateway-integration.md`.
- Replace stale point-in-time counts with dated measurements (re-measure;
  do not copy the old 18/263/1-missing numbers).

## Non-goals

- No Go code, schema, fixture, or Makefile changes.
- No edits to `qratum-vault-first.md` itself (it is the input contract).
- Do not delete `memory-curation-pipeline.md` or strip its superseded banner.
- Do not resurrect anything on the "Dead" list in `qratum-vault-first.md`.
- Do not touch `edictum`, `edictum-harness`, or the personal-memory repos.
- Do not change the current milestone marker to anything past
  `P0-SPEC-AND-CONTRACTS` (P2 owns the unlock, not this prompt).

## Implementation notes

Apply exactly the seven edits from `qratum-vault-first.md` → "Spec Hygiene":

1. **Locked Product Decisions** — unlock and rewrite with a dated note. Output
   priority becomes: preservation → lessons-to-memory → insights-harvest →
   search → review → corpus. "Primary surface: local app" → "Primary surface:
   CLI + vault; app earned later". "SQLite-backed search is the default local
   projection" → keep, but add the first-third-party-Go-dependency caveat
   (supply-chain decision under `docs/supply-chain.md`).
2. **Cut** every reference to: `LessonBackend` (ports list, adapters list,
   backend stack), `VectorBackend`/`sqlite-vec`/embedding policy, the
   `tidb_remote` backend mode and its config example, and `DuckDB`.
3. **Consent** — full record schema becomes a documented future shape; MVP
   behavior = config defaults + a one-line audit event. Note it deliberately
   mirrors Edictum semantics (documented resemblance, not a dependency).
4. **Approval-queue contradiction** — honor the existing non-goal "no
   persistent approval/pending item queues". Rewrite the lesson-candidate line
   ("higher-risk suggestions stored for user review") to "factory-curated,
   human-sampled, batch-approved".
5. **Milestones** — replace the P1–P5 ladder with vault-first sequencing
   (P1 = the vault spec; later milestones only on demonstrated pull).
6. **SPEC.md** — point to `qratum-vault-first.md` alongside the operational
   model; keep the current milestone at `P0-SPEC-AND-CONTRACTS`.
7. **ADR 0010** — `docs/decisions/0010-vault-first-and-direct-gateway-integration.md`
   in the 0001–0009 format: vault-first; no one-person publish ceremony; the
   store owns its own curation; direct gateway calls with a locally-held
   credential are the integration mechanism. Reference the review directory.

Dead terms to drive the grep check: `LessonBackend`, `sqlite-vec`,
`VectorBackend`, `tidb_remote`, `DuckDB`.

## Acceptance criteria

- `operational-model-redesign.md` contains none of the five dead terms, and
  its Locked Decisions / milestones / consent / approval-queue sections read as
  rewritten above.
- `SPEC.md` points at `qratum-vault-first.md` and still says milestone
  `P0-SPEC-AND-CONTRACTS`.
- `docs/decisions/0010-...md` exists and matches the existing ADR format.
- No stale point-in-time counts remain; any retained number is dated and
  re-measured.
- Build and tests are unaffected (docs-only).

## Decision Trace

- Accepted 2026-06-14: `qratum-vault-first.md` "Spec Hygiene" is the exact edit
  list; `../../PROPOSAL.md` is the spine.

## Behavior Contract

- [ ] FAILS the task: any Go/schema/fixture/Makefile change (docs-only).
- [ ] FAILS if a dead term remains in SPEC.md, operational-model-redesign.md,
  or ADR 0010; evidence: the dead-term grep returns zero hits.
- [ ] Evidence: SPEC.md shows the milestone marker still
  `P0-SPEC-AND-CONTRACTS`.

## Drift Handling

- If `operational-model-redesign.md` changed since 2026-06-14 and the seven
  edits no longer map cleanly, stop and report. Stop conditions below are hard.

## Verification

```sh
# Build/tests still pass (docs-only, but prove nothing broke):
make -C /Users/acartagena/project/qratum build
make -C /Users/acartagena/project/qratum test

# Dead terms gone from exactly the three edited targets (expect no output):
grep -nE 'LessonBackend|sqlite-vec|VectorBackend|tidb_remote|DuckDB' \
  /Users/acartagena/project/qratum/SPEC.md \
  /Users/acartagena/project/qratum/specs/current/operational-model-redesign.md \
  /Users/acartagena/project/qratum/docs/decisions/0010-vault-first-and-direct-gateway-integration.md

# ADR exists:
test -f /Users/acartagena/project/qratum/docs/decisions/0010-vault-first-and-direct-gateway-integration.md && echo "ADR 0010 present"
```

Note: `qratum-vault-first.md`, `memory-curation-pipeline.md`, and everything
under `docs/reviews/` intentionally keep the dead terms in their Dead/history
sections — do not grep or "fix" those.

## Slop Review

- [ ] Behavior contract holds: no Go/schema/fixture/Makefile file changed.
- [ ] Explicit evidence: dead-term grep over SPEC.md +
  operational-model-redesign.md + ADR 0010 returns zero hits.
- [ ] Milestone marker unchanged (still `P0-SPEC-AND-CONTRACTS`).
- [ ] `qratum-vault-first.md`, the superseded pipeline spec, and `docs/reviews/`
  still retain their Dead/history terms (not "fixed").

Reviewer guidance:

> Review this docs-only change against `qratum/specs/current/qratum-vault-first.md`
> section "Spec Hygiene". Confirm all seven edits were applied in place (not as
> an overlay), the five dead terms are gone from SPEC.md +
> operational-model-redesign.md + ADR 0010, the milestone marker is still
> `P0-SPEC-AND-CONTRACTS`, ADR 0010 matches the 0001–0009 format, and no stale
> counts remain. Flag any runtime/code/schema/fixture change as out of scope.

## Stop conditions

- STOP if `qratum-vault-first.md` or `PROPOSAL.md` Status line is not
  "Accepted" — report and do not edit.
- STOP if any input spec/review file is uncommitted/untracked
  (`git status --short`) — report.
- STOP and report instead of guessing if `operational-model-redesign.md` has
  changed materially since 2026-06-12 such that the seven edits no longer map
  cleanly.
- STOP if applying an edit would require touching Go code, schemas, or
  fixtures — that means the scope was misread.

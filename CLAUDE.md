# Claude Code session notes (qratum)

`AGENTS.md` is this repo's canonical agent-instruction file, and `SPEC.md` is the
authoritative index (`SPEC.md` wins; see AGENTS.md §Source Of Truth). Read both
before changing anything.

@AGENTS.md

## This repo is governed (Engineering OS)

tier: S
Reference: https://github.com/acartag7/engineering-os

This repo is onboarded to the Engineering OS. CI enforces the walls below; this
block just saves you a red build. It sits UNDER this repo's own source-of-truth
rules — `SPEC.md` wins, and `specs/current/ui-first-onboarding.md` is
authoritative for onboarding and the public `qrt` command contract (AGENTS.md
§Source Of Truth). Where a generic phrasing here conflicts with a repo rule, the
repo rule wins.

Non-negotiables — CI enforces these; this block just saves you a red build:

- Acceptance tests under `test/acceptance/` are FROZEN. Editing any of them turns
  CI red (process-guard freeze-hash check). Turn finished phases on via
  `test/acceptance/phases.json` only. If a test looks wrong: STOP and report —
  that's a contract change, not a patch. No suite exists yet, so the repo carries
  a `.process-guard-exempt` marker until the first frozen suite lands; that marker
  is a named gap, not a green light to skip the pipeline.
- Contract first: `SPEC.md` and the accepted spec it points at win over the code
  and over your inference. Never implement while a contract has open decisions or
  points at files outside this repo. Schemas under `schemas/` and fixtures under
  `fixtures/` are part of the contract.
- Trust-boundary decisions are allowlists, never blocklists. Empty config counts
  as missing config: fail closed. Type-check every externally-sourced value before
  using it. Malformed input fails closed, never best-effort.
- Build the least machinery the contract asks for. No unrequested parsers,
  validators, or abstractions (mirrors AGENTS.md §Still not built). If the simple
  approach feels insufficient, stop and ask — don't build.
- After fixing any defect, sweep sibling code paths BEFORE re-requesting review.
  Partial fixes are the top review-round multiplier.
- Never weaken a check to get green. Never push to protected branches (`main`) —
  the factory owns merge. PRs use conventional commit subjects (the `PR Title`
  workflow enforces this) and carry a `Spec: <path>` trailer.
- Review verifies; it never discovers. If review is teaching us what the spec
  should have said, say so — that's a process failure to record, not a grind to
  endure.

### Local setup: enable the process-guard pre-commit hook

This repo ships a guard hook under `.githooks/`. Enable it once per clone:

```sh
git config core.hooksPath .githooks
```

The hook is a local preview of the CI `process-guard` job. It no-ops with a
visible warning (exit 0) when `node` or an `engineering-os` checkout is absent —
this is a Go repo with no Node toolchain, so it never blocks a commit on missing
optional tooling. Point it at a checkout with `ENGINEERING_OS_HOME=<path>` if it
is not a sibling of this repo. CI is the real wall regardless.

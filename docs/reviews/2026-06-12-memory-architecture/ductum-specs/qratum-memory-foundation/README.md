# Qratum Memory Foundation — Ductum Spec Package

Importable spec package for the Qratum side of the 2026-06-12
memory-architecture workstream. Source of truth for intent:

- `../../PROPOSAL.md` (the spine: preserve → truthful store → import once → stop)
- `../../BACKLOG.md` (sections A and B)
- `../../DISPATCH-PROMPTS.md` (Prompts 1 and 2)
- `qratum/specs/current/qratum-vault-first.md` (the accepted vault spec)

Scope: this package touches **only the `qratum` repo**. It is personal
infrastructure and is kept deliberately separate from `edictum` (the product).
Do not touch `edictum` or `edictum-harness`.

## Execution Order

| # | Prompt | Package | Scope | Deliverable | Status | Depends On |
|---|--------|---------|-------|-------------|--------|------------|
| 1 | [P1-SPEC-HYGIENE.md](P1-SPEC-HYGIENE.md) | qratum | docs/spec | Accepted spec docs + ADR 0010 | [ ] | — |
| 2 | [P2-VAULT-BUILD.md](P2-VAULT-BUILD.md) | qratum | runtime/vault | Local vault workflow + tests + runbook | [ ] | P1 |

## Gate before importing or dispatching

1. **Acceptance gate** — `qratum/specs/current/qratum-vault-first.md` and
   `../../PROPOSAL.md` must have their Status lines flipped to "Accepted
   (date)". Until then, P1 stops at its first stop condition.
2. **Milestone unlock for P2** — Qratum is at `P0-SPEC-AND-CONTRACTS`
   (`AGENTS.md`). P2 builds runtime and therefore requires Arnold to explicitly
   unlock the vault milestone. P2 stops if the milestone is still P0.
3. **Clean inputs** — the spec/review files each prompt reads must be committed
   (`git status --short` clean for those paths).

## Manual, Arnold-only steps (never automated by a dispatched agent)

- Installing the GLOBAL `~/.claude/settings.json` SessionEnd hook on this
  machine (`qrt hook install`).
- Running the first `qrt vault backfill`.
- Archiving the Claude.ai export into the vault.

These mutate the user's real home directory and live history. A dispatched
agent builds and tests the commands; it must not run them against the real
`~/.claude` or `~/.qratum`.

## Decision Trace

- Accepted 2026-06-14: `../../PROPOSAL.md`, `qratum/specs/current/qratum-vault-first.md`.
  ADR 0010 records it (produced by P1).

## Behavior Contract

- P1 is docs-only; P2 is runtime and FAILS if built while the milestone is
  still `P0-SPEC-AND-CONTRACTS`.
- The vault never sends raw transcripts off the machine; a hook that parses,
  networks, calls an LLM, or emits raw FAILS. Evidence: `make verify` + the P2
  end-to-end vault proof.
- No `qratum-vault-first.md` "Dead"-list item is implemented.

## Verification

- P1: `make build && make test` + dead-term grep. P2: `make verify` + `make demo`
  + end-to-end vault proof. See each task's Verification.

## Drift Handling

- If the accepted spec changed since 2026-06-14 and no longer maps, stop and
  report; re-measure stale counts.

## Slop Review

- [ ] No "Dead"-list item resurrected (LessonBackend, search, normalizer,
  curation queue).
- [ ] Hook does only stdin → event → file-copy (no parse/network/LLM).
- [ ] No command mutates the real `~/.claude` or `~/.qratum` in tests/CI.
- [ ] No new third-party Go dependency without a supply-chain decision.

# Review 8: Independent Meta-Review (GPT-5.5)

- Lens: independent re-assessment + meta-review of reports 01-07 for correlated
  blind spots and overclaims; different model lineage
- Model: GPT-5.5 (via Codex)
- Date: 2026-06-12

This review was commissioned because reports 01-07, though using different
lenses, were all orchestrated by one Claude instance with one shared framing —
their convergence could be a correlated blind spot rather than truth. GPT-5.5
formed its own verdict from specs + code BEFORE reading 01-07, then
meta-reviewed them and re-measured every load-bearing number.

## Independent verdict

Vault-first is directionally right: current Qratum is pointer capture plus an
on-demand Milestone A refinery, not a vault, and pointer capture is already
losing value. Measured: 22 Qratum event files, one processed artifact chain,
267 local Claude JSONL transcripts, 2 event transcript paths already missing.
Copy-on-capture plus backfill is the only urgent, passive-value work.

Top concerns:
1. The replacement still does not fully own operations: global hook install,
   second machine/Hermes, cloud-only sessions, recurring backfill, and backup
   verification are mostly manual or out-of-band.
2. The personal-memory verb plan is correct in Phase 1, partly correct in
   Phase 2, and likely over-scoped in Phases 3-4 for the immediate 1+7
   memory-string import.
3. Qratum may still be too product-shaped for what remains.

The old bundle bridge deserved to die as ceremony, but it pointed at real
needs — receipt, memory IDs, subject pinning, source digest, auditability —
which must survive the direct script.

## Findings (condensed)

1. **Vault capture is operationally unowned** — VERIFIED / blocker. Global
   ~/.claude/settings.json has 0 SessionEnd hooks; ductum local has 1. Fix: P2
   ships `qrt hook install`, `qrt vault doctor`, `backup --verify`, explicit
   second-machine/cloud scope; status warns on no global hook, stale backfill,
   copy failures, missing backup.
2. **Source of truth still contradictory** — VERIFIED / serious. Vault-first
   is still "proposal", SPEC.md still calls the old model canonical and lists
   raw archive as a P0 non-goal; spec/review files untracked. Fix: spec
   hygiene first, commit it, replace volatile counts with dated measurements.
3. **Gateway plan larger than immediate need** — SUSPECTED / serious. Phase 1
   fixes real current gaps; Phases 3-4 (near-dup, retrieval report, re-embed,
   confirmed override, fixture export) should be evidence-gated by row count /
   duplicate incidents / actual blocked items.
4. **`personal` import target collides with planned reclassification** —
   VERIFIED / serious. Roadmap says existing `personal` rows need
   reclassification and must not be bulk-renamed. Fix: keep the default but
   record namespace decision version, subject, source digest, memory IDs in
   receipts; future migration preserves by memory ID.
5. **Bundle died, but receipt/provenance must not** — ATTACKED-AND-HELD /
   serious. Make receipt.jsonl a tested contract, archive it into Qratum with
   the source export digest, don't treat the script as disposable until
   receipt replay/status is boring.
6. **Qratum still has an existence problem** — SUSPECTED / serious. Constrain
   the next milestone to vault-minimum only; no app/SQLite/search/review
   loop/corpus/candidate lane until repeated real pull exists.

## Meta-review of reports 01-07

- Report 3 / README numeric claims: direction holds, counts drifted — now
  22 events and 2 missing paths (not 18/one), 267 transcripts, 71 recent,
  ~76 MB.
- Reports 1/3/7 "one-afternoon script": right on shape, underpriced
  auth/subject/provenance (the gateway plan since fixed this).
- Reports 2/4/5 near-dup consensus: technically plausible but over-promoted
  for a 1+7 import; Phase 1 safety matters more.
- Reports 1/3/6/7 killed the bundle correctly but over-killed the audit idea;
  the ceremony was wrong, the receipt was not.
- All reports under-owned cross-prompt seams: no prompt owned second-machine
  install, cloud-session inventory, recurring backfill, or restore drills.

## Refuted / corrected claims ledger

| Prior claim | Recheck |
|---|---|
| Qratum stores pointers only, no raw archive | Confirmed |
| Gateway cannot emit `duplicate` | Confirmed (success payload has no outcome; upsert result discarded) |
| Unknown contentClass coerces to `note` | Confirmed (`content-policy.ts:56`) |
| `private` allowed by scope + allowlist only | Confirmed (`access.ts:36`) |
| Export counts (556/11,962/447/1/7/13/5, zero headings) | Confirmed via jq |
| 18 events, one missing, 263 transcripts, 67 recent, 71 MB | REFUTED/stale: now 22 events, 2 missing, 267 transcripts, 71 recent, ~76 MB |
| internal/ has 16 empty directories | Stale count: 15 dirs, 0 files; essence confirmed |
| Global capture only wired in ductum | Confirmed (0 global SessionEnd hooks; 1 in ductum local) |

## The thing still off

The plan is still treating a founder-attention problem as an architecture
problem. The old design was too ceremonial; the new one is much better, but it
still spreads attention across vault work, gateway verbs, import tooling,
future git-native curation, and an Edictum wedge note. The real urgent object
is narrower: stop losing transcripts, make memory writes truthful and
reversible enough not to corrupt the store, import the curated strings once,
and stop. Anything beyond that should have to earn its next hour.

## What GPT would do differently

- Ship spec hygiene first, with dated measurements and no stale "verified"
  counts.
- Build vault as installable operations: hook install, doctor, backup
  --verify, second-machine runbook.
- Gateway: Phase 1 plus minimal correction verbs; defer near-dup/reporting
  until measured need.
- Import script: candidates file, exact gateway policy subject, subject
  pinning, receipt with memory IDs, receipt archived in vault.
- No git-native lane, SQLite search, app, or refinery expansion until there is
  repeated real usage.

## Disposition (folded into the specs 2026-06-12)

- Finding 1 -> vault spec gained an "Operational ownership" section
  (`qrt hook install`/`status`, `qrt vault doctor`, `backup --verify`) and a
  "Multi-machine and cloud scope" section; Prompt 2 updated.
- Finding 3 -> gateway Phases 3-4 reframed as explicitly evidence-gated with
  per-item triggers.
- Findings 4-5 -> receipt gains namespace/subject/source-digest/
  namespace_map_version, becomes a tested contract archived into the vault as
  raw kind `memory_import_receipt`; Prompt 4 updated.
- "Thing still off" -> PROPOSAL.md gained "The spine: four things, then stop".
- Stale counts (Finding 2) -> to be replaced with dated measurements during
  spec hygiene (Prompt 1); README carries the correction below.

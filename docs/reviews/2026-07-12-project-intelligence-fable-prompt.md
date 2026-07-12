# Fable Review Prompt: Qratum Project Intelligence

You are performing an adversarial product and systems-design review of a
proposed extension to Qratum.

This is review-only. Do not edit files, implement features, create branches, or
turn the proposal into a technical roadmap.

Repository:

```txt
/Users/acartagena/project/qratum
```

Read these files completely, in this order:

1. `AGENTS.md`
2. `SPEC.md`
3. `specs/current/product-direction.md`
4. `docs/reviews/2026-07-10-product-user-stories.md`
5. `docs/reviews/2026-07-12-project-intelligence-user-stories.md`

Authority and context:

- `specs/current/product-direction.md` and the incorporated 2026-07-10 user
  stories are the accepted core product direction.
- The 2026-07-12 Project Intelligence document is a proposal under review. It
  is not accepted and must not silently override the core direction.
- Older Qratum onboarding, operational-model, vault-first, trust-gate, and
  rebootstrap documents are historical evidence where the current authority
  marks them superseded. Do not restore their old product framing by inertia.
- Qratum is not backup software, a pipeline CLI, a project-management suite, a
  billing provider, or a developer-productivity scoring system.
- The accepted core is a private searchable memory of AI-assisted work with
  exact local reading, Claude and Codex sources, truthful cost, provenance,
  hybrid search, continuity, deletion, and explicit user control.
- The proposed extension adds Projects, repository/worktree grouping,
  deterministic analytics, Workstreams, code links, project summaries,
  decisions, open loops, project direction, and a user-owned AI-proposed
  roadmap.
- Exact evidence, deterministic derivations, semantic projections, generated
  interpretations, and user-confirmed items must remain distinguishable.
- Durable cross-agent memory belongs in personal-memory.
- Git/GitHub/Jira remain authoritative for their own objects.

Review the product extension, not the old runtime implementation. Use current
code only to challenge feasibility claims or identify reusable foundations.
When making current claims about Claude or Codex capabilities, verify them from
current authoritative sources instead of relying on older assumptions.

Evaluate:

1. Does Project become the correct organizing layer above sessions?
2. Is Workstream a necessary middle layer or a curation burden that will go
   unused?
3. Can repository, clone, worktree, fork, folder, and multi-machine identity be
   correct and understandable?
4. Can project/session/workstream assignment avoid silent misattribution?
5. Is token and cost accounting auditable, historically correct, and protected
   from cumulative/resume/import double counting?
6. Is the source/surface/harness/agent/model taxonomy supported by realistic
   source evidence?
7. Can session-to-code/commit/PR/issue links be useful without implying false
   causality?
8. Are project summaries evidence-linked and incrementally maintainable?
9. Can automatic decisions and open loops avoid becoming noisy generated task
   queues?
10. Can a user-owned AI-proposed roadmap remain useful without turning Qratum
    into a project manager or letting assistant speculation become direction?
11. Does the exact/deterministic/semantic/generated/confirmed information model
    remain clear in the proposed UI?
12. Is the no-LLM deterministic product still valuable?
13. Are local embeddings and local/external LLM boundaries coherent with the
    privacy promise?
14. Does deletion correctly propagate through projects, workstreams,
    aggregates, summaries, decisions, roadmap evidence, context packs, and
    personal-memory references?
15. Is the proposed tranche placement ordered by real dependency and value, or
    does it overload Tranches 1 and 2?
16. Which proposed features are indispensable, fast-follow, or speculative?
17. Which user journeys or correction mechanisms are missing?
18. Does this extension materially strengthen Qratum's product thesis, or does
    it make the product incoherent?

Return the review in this structure:

## Verdict

Choose one:

- Accept the extension
- Accept with specific reshaping
- Reject and keep the current product boundary

Explain in no more than five sentences.

## Product-Critical Findings

Only issues that would make the extension untrustworthy, unusable, or
incoherent. For each include severity, affected section, user consequence, and
smallest product-level correction.

## Project And Repository Identity Review

Review projects, repositories, remotes, clones, worktrees, forks, moved paths,
multiple machines, assignment confidence, and manual correction.

## Usage And Cost Review

Review usage-event identity, cumulative versus incremental counters, resumed
sessions, imports, historical pricing, actual versus estimated cost, coverage,
unknowns, workstream attribution, and drill-down.

## Workstream And Code-Link Review

Review whether workstreams have a real daily use, how suggestions should work,
correction burden, code/commit/PR/issue linkage, and false-causality risks.

## LLM And Information-Quality Review

Review the deterministic baseline, embeddings, generated summaries, decisions,
open loops, roadmap candidates, provenance, caching, incremental refresh,
staleness, privacy, and cost of Qratum's own analysis.

## Direction And Roadmap Review

Attack the distinction between description and prescription. Identify how
assistant speculation, abandoned plans, stale sessions, generated estimates,
or code evidence could corrupt the proposed direction. State the minimum safe
roadmap interaction.

## Missing Or Broken User Journeys

Describe each with starting condition, user action, expected result, and
failure/recovery behavior.

## Scope And Tranche Review

Classify every major feature as:

- indispensable foundation;
- daily product;
- immediate follow-up;
- speculative/defer.

Then propose the smallest credible dependency order. Do not optimize merely
for implementation convenience.

## Existing-System Fit

Separate reusable foundations, candidate code needing reshaping, concepts to
discard, and claims needing fresh verification. Ground code claims in current
file and line references.

## Blocking Product Decisions

Rewrite unresolved decisions as questions the product owner can answer without
choosing unnecessary technical architecture.

## Recommended Changes To The Draft

Give specific changes by section. Do not rewrite the document wholesale.

## Final Consistency Check

Explicitly answer:

- Is Qratum still a coherent product?
- Is Project the right organizing layer?
- Is Workstream worth its curation cost?
- Are project costs trustworthy?
- Can every aggregate and generated statement reach evidence?
- Is the roadmap truly user-owned?
- Does the app remain useful without a generative LLM?
- Is deletion complete across the new derived objects?
- Is the proposed release scope credible?

Be direct. Do not praise the proposal generally. Focus on defects,
contradictions, missing workflows, overreach, and decisions that must be made.

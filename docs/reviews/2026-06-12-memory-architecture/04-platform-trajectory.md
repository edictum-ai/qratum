# Review 4: Platform Trajectory

- Lens: will the 2026 platform memory wave make this design pointless before it pays off
- Model: Fable
- Date: 2026-06-12

---

Context read: `qratum/specs/current/operational-model-redesign.md` and `personal-memory-gateway/docs/roadmap.md`. Platform capabilities verified by web search against official docs where possible.

## Verified platform baseline (June 2026)

These are real, current, checked capabilities — not extrapolation:

1. **Claude Code auto memory is native, on by default** (v2.1.59+). Per-repo `~/.claude/projects/<project>/memory/MEMORY.md` + topic files, written by Claude itself, loaded every session (first 200 lines/25KB), subagents can have their own memory, `/memory` to audit. Plain local markdown. (Official docs.) **VERIFIED**
2. **Claude Code `/insights` is native**: 30-day local cross-session HTML report — friction patterns, strengths, CLAUDE.md suggestions, workflow recommendations. Community tooling (e.g., `claude-insights`) already converts the report into skills/rules/CLAUDE.md entries automatically. **VERIFIED** — and note: Qratum's spec "Insights" section is visibly reverse-engineered from this feature (the spec itself says "lessons from reviewed usage-insights implementations").
3. **Claude Code deletes transcripts by default**: 30-day `cleanupPeriodDays` silent hard-delete, no recovery, multiple open GitHub issues (#59248, #62959, #64999), plus a bug where `cleanupPeriodDays: 0` disables transcript persistence entirely, and reports of cleanup misfiring on updates/restarts. **VERIFIED**
4. **Codex memories is native and automatic**: background distillation of eligible prior threads into durable local memory files under `~/.codex/memories/` — summaries, durable entries, evidence — with secret redaction and rate-limit-aware scheduling. (Official OpenAI docs.) **VERIFIED**
5. **ChatGPT "Dreaming V3"**: background consolidation rewrote memory from fact-list into self-updating synthesis; notably the update *reduced* the user-visible audit trail. **VERIFIED**
6. **Memory portability is becoming table stakes — at the distilled tier only**: Claude.ai ships memory export/import (copy-paste text of the memory profile, imports from ChatGPT/Gemini, free tier included, explicitly experimental). Anthropic Managed Agents memory (public beta, April 2026): file-based, exportable, per-write audit logs, immutable versions, cross-agent sharing *within a vendor workspace*. **VERIFIED**
7. **Execution is moving off-machine**: Claude Code on the web runs sessions on Anthropic-managed infra; `--teleport` pulls cloud sessions down, but local hooks never see sessions that start and end in the cloud. **VERIFIED**

## Per-premise verdicts

### Premise (a): "Capture transcripts yourself before tools delete them" — **HOLDS, strongest premise**
- **VERIFIED**: the deletion threat is real, current, silent, and irreversible. Vendor incentive runs the wrong way for him: transcripts are resumption cache to Anthropic, not an archive product. No plausible roadmap makes the vendor your archivist of *cross-vendor* raw history.
- **SUSPECTED/TREND, severity medium**: the *capture mechanism* erodes even as the premise holds. Hook capture assumes local execution; cloud sessions, mobile-initiated runs, and Ductum agents on other machines are invisible to a hook watching one Mac. In 12 months, "the transcript is on my disk" stops being the default. The vault survives; hook-only capture becomes a partial sensor and needs a cloud-session-aware inventory/sync path.
- **BET-HOLDS**: content-addressed local raw archive + the redaction boundary survive *any* vendor roadmap, because no vendor will ever ship "unified permanent raw archive of your usage of our competitors."

### Premise (b): "Durable cross-agent memory must be self-hosted to be owned" — **HOLDS ONLY AT THE RAW/CROSS-VENDOR TIER**
- **VERIFIED counter-evidence**: vendors are racing to make distilled memory portable (to capture switchers), which makes it exportable as a side effect. Managed Agents memory is exportable, audited, shareable across agents — within one vendor's workspace.
- **BET-HOLDS**: the union across vendors, plus raw fidelity, plus deletion-proofness. Vendor memory sharing stops at the vendor boundary; his gateway is the only neutral point his Claude Code, Codex, Claude.ai, and Ductum agents all touch. Restate the premise honestly: *self-hosting is the moat for the raw tier and the cross-vendor union, not for "memory" generically.*

### Premise (c): "Sessions->lessons requires a dedicated local pipeline with human curation" — **MOSTLY INVALIDATED, severity high**
- **VERIFIED**: Codex already does background session->durable-lesson distillation natively, with redaction. Claude Code auto memory does continuous self-curation per repo. `/insights` does cross-session mining natively. This is the exact loop Qratum's AI facet/lesson mining proposes to build per-source, and vendors are shipping it as a free default, improving monthly.
- 12-month extrapolation (**SUSPECTED/TREND**): every serious coding agent ships background distillation + an insights surface; the per-vendor lesson-mining loop is fully commoditized.
- **BET-HOLDS**: three slivers survive any vendor roadmap — (i) *cross-source* distillation (lessons mined across Claude Code + Codex + Claude.ai jointly), (ii) distillation into *his* schema/namespace/provenance model, (iii) redaction before anything leaves the machine. The cheap dominant move: treat vendor distillation output as *input* — the spec already archives `source_insight_report` as a raw kind; extend that to the memory dirs (`~/.claude/projects/<project>/memory/`, `~/.codex/memories/`) and let vendors do the expensive mining for free.

### Premise (d): "Platform memory is a source, not a competitor" — **WRONG FRAMING, severity medium-high**
Half-right: as a *source*, verified and trivially ingestible (local markdown, exportable profiles). But platform memory competes for the same job — carrying lessons into the next session — and it wins on *distribution*, not quality: MEMORY.md is loaded automatically at session start; a Qratum lesson in personal-memory must be fetched via MCP and may never be queried. The platform doesn't obsolete the pipeline by being smarter; it obsoletes it by being *default*. Consequence: for a lesson to change agent behavior, Qratum's real export targets are the surfaces agents already read — CLAUDE.md rules, skills, hooks, AGENTS.md, Edictum gates — not only the gateway. The spec already lists "suggested rules, hooks, or skills" as lesson outputs; that line is more strategically important than the entire corpus-export apparatus around it.

## Interop angle
Portability is emerging but *shallow*: copy-paste profile strings and within-vendor exportable file stores. There is no standard for raw session/trajectory interchange (ADP is training-corpus-oriented, which the spec already treats as an export boundary). MCP under Linux Foundation governance standardizes the *pipe* (his gateway's interface is the right bet), not the *content*. Verdict: the import/normalize layer does **not** become trivial — formats are getting *more* heterogeneous (memory dirs, insight HTML, session DBs, cloud sessions, profile exports), and the neutral-hub value rises with heterogeneity. But it must stay thin and adapter-per-source with fixtures (the spec's SourceAdapter design is correct), because every individual format will churn.

## Self-curation trend
Agents curating their own memory is now the verified default everywhere — and notably getting *less* auditable on the vendor side (OpenAI's update reduced the audit trail). So: the curated/uncurated distinction does not collapse; a curated, provenance-bound tier becomes *more* valuable as uncurated volume explodes. But *per-candidate human review* is the wrong implementation — it breaks at exactly the moment it matters. The human's role shifts from reviewer-of-each to policy-author + sampler + approver-of-high-risk-writes. The spec contains a latent contradiction here: it requires "higher-risk suggestions stored as local candidates for user review" while listing "persistent approval/pending item queues" as an explicit non-goal. That contradiction is the trend knocking on the spec's door.

## His own stack at 10x
Scales: vault (append-only, content-addressed), deterministic normalize/redact/evidence, automated mining with the spec's AI job plans/budget caps. Breaks: the human staging lane, and any insight surface a human must read per-session. The Edictum-shaped opportunity is real and partially validated by Anthropic themselves: Managed Agents memory shipped per-write audit logs + immutable versions + rollback — i.e., a vendor just productized "governed memory writes" inside their walled garden. **"Approval boundaries for what agents may persist into shared memory" is a Workflow Gate** — evidence-gated `memory_store`, no write to `global`/`project:*` namespaces without provenance and policy pass, staged writes pending approval. The personal-memory roadmap *already specifies* `staging:<target>` namespaces and "agent-governed memory curation... require explicit approval or a governed workflow." Qratum's lesson lane should be the first customer of that pattern, not a parallel implementation of it.

## Timing table

| Component | Verdict | Why |
|---|---|---|
| Vault (hook capture + content-addressed raw archive) | **BUILD NOW** | Only component with an irreversibility clock (verified silent deletion). Everything downstream is rebuildable from raw; nothing is rebuildable without it. Add cloud-session inventory as a thin follow-on. |
| Claude.ai export adapter | **Archive the zip NOW (an hour); build the adapter THIN, LATER** | Once the blob is in the vault the data is static — no clock. Normalizing 556 conversations earns nothing until search/lessons need it. |
| Tier-0 memory split (~17 platform memory strings) | **DON'T BUILD** | It's 17 strings. Hand-curate into personal-memory in one afternoon. An adapter here is machinery worship. |
| AI session mining (facets/lessons) | **DON'T BUILD as designed; build thin harvest instead** | Vendors ship this natively (Codex memories, auto memory, Dreaming). Instead: archive vendor memory dirs + insight reports as raw kinds, mine only *cross-source* gaps, only after evidence vendor output misses things. |
| Human staging lane | **REDESIGN BEFORE BUILDING** | Per-candidate review breaks at Ductum scale. Build it as a policy-gated write boundary (auto-approve low-risk, queue high-risk, audit everything) and unify with the gateway's planned `staging:<target>` — one staging concept, not two. |
| Bundle+importer bridge | **BUILD THIN NOW** | Small (lesson JSONL + manifest -> `memory_store` with namespace + provenance). Keep the contract on the gateway side where `memory_export`/curation are already roadmapped. Keep it dumb. |
| Local SQLite FTS search | **BUILD NOW** | Cheap, no platform threat (no vendor will search your cross-vendor archive), and it's the day-one retention hook that makes the vault feel valuable weekly instead of annually. |
| Insights | **DON'T BUILD** | `/insights` is native, free, improving, and already convertible to rules/skills by community tools. Archive its HTML output as raw kinds and diff over time. Rebuilding it costs months to produce a worse report. |

**12-month-regret-minimizing order**: (1) raw capture + archive incl. vendor memory dirs/insight reports as raw kinds + `cleanupPeriodDays` mitigation; (2) Claude.ai export zip into blobs; (3) SQLite FTS over normalized Claude Code sessions; (4) thin governed-write bridge to personal-memory (staging namespace, provenance, hand-curated lessons first); (5) stop — everything else only on demonstrated vendor gap.

## The unnamed odd shape

**Qratum is a third governance system in an ecosystem that already has two — and the governance is the product-shaped part, sitting in the wrong repo.**

The spec is ~2,300 lines, and the majority of its mass — consent records, trust boundaries, data-class badges, policy evaluators, publish manifests, artifact provenance, approval flows, audit semantics — is enterprise-grade governance machinery wrapped around a single-user local tool. That's not random over-engineering; it's *Edictum's product DNA expressing itself in a personal side project*. He is building a personal, single-tenant "Edictum for memory" and calling it a session library. Meanwhile Edictum (the actual product) has no governed-memory-writes story, and personal-memory's roadmap is independently growing its own third copy of the same concepts (namespaces, grants, staging, reversible curation logs, approval-gated destructive ops).

Three implementations of approval-boundaries-for-persistence, zero shared contracts. The "odd shape" he senses is that redundancy: the design keeps re-deriving his company's core pattern in places where it can't become product.

The resolution that the platform trajectory itself recommends: Qratum shrinks to the librarian (capture, archive, normalize, search — the parts vendors verifiably destroy or won't unify); curation/staging lives in the gateway where it's already specified; and the governance pattern — evidence-gated, approval-bounded agent memory writes — gets extracted into the Edictum wedge, where Anthropic's Managed Agents memory launch just validated the demand inside a walled garden Edictum can serve *across* gardens.

## Sources

- Claude Code memory docs (official): code.claude.com/docs/en/memory
- Codex memories docs (official): developers.openai.com/codex/memories
- Claude memory import/export help article (official): support.claude.com/en/articles/12123587
- Claude Code data usage / retention docs: code.claude.com/docs/en/data-usage
- Silent retention cleanup issues: anthropics/claude-code #59248, #62959, #64999; cleanupPeriodDays:0 bug #23710
- claude-insights (insights -> skills/rules converter): github.com/yahav10/claude-insights
- Anthropic Managed Agents memory coverage: testingcatalog.com, edtechinnovationhub.com, venturebeat.com
- ChatGPT memory "Dreaming" update: TechTimes, Tom's Guide
- Claude Code on the web docs: code.claude.com/docs/en/claude-code-on-the-web
- Free Claude users memory + rival import: 9to5mac.com (2026-03-02)
- Agent interoperability 2026: zylos.ai research; mem0.ai "State of AI agent memory 2026"; newamerica.org OTI brief

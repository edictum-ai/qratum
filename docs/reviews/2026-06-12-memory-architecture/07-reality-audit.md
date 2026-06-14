# Review 7: Reality Audit

- Lens: verify every load-bearing claim against the actual repositories and data
- Model: Fable (relaunch; the killed Opus run's partial findings were independently re-derived)
- Date: 2026-06-12

---

Method: read both specs in full (`operational-model-redesign.md` 2,282 lines, `memory-curation-pipeline.md` 470 lines), read all gateway domain/use-case/repository/server code, read migrations + roadmap + contracts, inspected all Milestone A Go code, and verified the Claude.ai export with jq (structure and counts only — no content printed).

## VERIFIED FLAWS

**F1 — The receipt's "duplicate" outcome cannot be produced. (serious)**
The bridge design assumes the importer can distinguish `stored` from `duplicate`. The gateway cannot tell it. `memory_store` does `INSERT ... ON DUPLICATE KEY UPDATE` and discards `affectedRows` (`src/infrastructure/tidb/memory-repository.ts:8-42`), then returns an identical success payload either way — `"Stored memory <id>", persisted: true` (`src/application/use-cases/persistent-memory.ts:84-98`). And since the importer's contract grants only `memory:write` (`memory-curation-pipeline.md:374-376`), it cannot even call `memory_search` (requires `memory:read`, `src/domain/access.ts:32-34`) to check. The outcome enum is a contract with a counterparty that doesn't exist.
*Fix:* either drop `duplicate` from the receipt enum (upsert idempotency makes it cosmetic), or add `outcome: "inserted"|"updated"` to `memory_store` structuredContent — mysql2 returns `affectedRows` 1 vs 2; it's a few lines in the gateway, which owns the contract anyway.

**F2 — The bridge writes into an append-only store with no correction path. (serious)**
Gateway has no `memory_delete` or `memory_update` — both "Planned v1" (`docs/contracts.md`; `docs/roadmap.md:84-93`). The Qratum loop explicitly supports editing a lesson and re-exporting. An edited lesson has a new `content_hash`, so it inserts a NEW row while the stale one persists forever, uncorrectable over MCP. Worse, the documented dedupe semantic — upsert clears `deleted_at` (`memory-repository.ts:24`, acknowledged at `roadmap.md:87-88`) — means re-running an old bundle will resurrect rows the user deletes once delete exists. The pipeline spec never mentions staleness or supersession. This quietly breaks the proposal's own headline: "CURATED knowledge lives ONLY in personal-memory" is only meaningful if that home can be curated, and today it is write-only.
*Fix:* receipt records the memory id per lesson; lesson edits record `supersedes: <memory_id>`; spec states plainly that cleanup is blocked on gateway `memory_delete`/`memory_update`, and re-export of edited lessons is discouraged until then.

**F3 — "Deterministic markdown-structure split" describes structure the data doesn't have. (serious)**
Verified against the real export: **zero markdown headings in all 8 memory strings**. Bold pairs: 4-9 per string; bullet lines exist in only 2 of 8 (account: 10 lines, one project: 23 lines). So Tier 0's stated split keys — "headings, bold section markers, bullet groups" (`memory-curation-pipeline.md:141-143`) — are two-thirds absent. Measured outcomes of the two viable deterministic strategies:
- Bold-section split of the account memory (10,097 chars): sections of **494 / 715 / 973 / 5,629 / 2,282** chars. Everything clears the 8,000 hard cap (so "splitting is mandatory and sufficient" holds), but the 5,629-char chunk is 4.7x the recommended <=1,200 and is a multi-topic grab bag, not an "atomic" memory.
- Blank-block split: 24 blocks, **15 to 2,258** chars; the 15-17-char blocks recur across all strings — they are orphaned section labels that would be garbage as standalone memories, and bullets/blocks severed from their bold header are no longer "self-contained."
Also: the fixture plan (`memory-curation-pipeline.md:455-468`) mandates synthetic fixtures "with markdown structure (headings...)" — fixtures would exercise a heading-split code path that the only real-world input never triggers, i.e., parity-by-fixture against an invented reality.
*Fix:* respec Tier 0 as what it actually is — bold-line sectioning with header-text propagated as a prefix onto blank-block children, minimum-size merge for orphan labels — and state honestly that "atomic, self-contained" is produced by the human reviewer's edits (or a later AI pass), not the parser. Fixtures must mirror the real shape: bold + blank lines, no headings.

**F4 — The proposal contradicts the canonical specs it claims to extend, without editing them. (serious — process)**
- `memory-curation-pipeline.md:291-292`: "Lessons are filesystem objects... **projected into the SQLite LessonBackend**" — the proposal deletes LessonBackend but leaves this line standing.
- `operational-model-redesign.md:24-52` "Locked Product Decisions": output priority Review #1 ... Lessons #5, "Primary surface: local app." The proposal reorders to lessons #2 / review #5 and demotes the app — reversing *locked* decisions silently.
- The op model's P0 deliverables include the consent record contract (`:2118-2127`); the proposal demotes it to "future shape only."
The deletions themselves are real (not hallucinated): LessonBackend (`:220-230, :243, :1538-1543, :1560-1561`), VectorBackend/sqlite-vec/embedding policy (`:1544-1549, :1562-1564`), TiDB private perimeter (`:462-486, :1570-1575`), DuckDB (`:484-486, :1566-1568`) all exist in the spec to be deleted. But as written the ecosystem would have three disagreeing sources of truth.
*Fix:* the redesign must land as edits to both spec files (and unlock the "Locked" list explicitly), not as an overlay document.

**F5 — Receipt outcome `blocked_sensitive` doesn't match the gateway's actual error class. (annoyance)**
Gateway emits `confirmation_required` for high-sensitivity content (`src/domain/content-policy.ts:50`), not `blocked_sensitive`. Mappable, but the importer spec should pin the mapping. Note `metadata_too_large` and `empty_content` have no receipt outcome at all. Related doc bug worth flagging upstream: `contracts.md` says high-sensitivity content "requires explicit user confirmation," but `MemoryStoreInput` has no confirmation field (`src/application/ports/memory-tools.ts:9-14`) — it is a hard reject today; the pipeline draft is right and the gateway's own contract doc is the misleading one.

**F6 — Pre-flight mirror has a verified gap: the gateway scans metadata too. (annoyance)**
`evaluateContentPolicy` runs over `content + "\n" + JSON.stringify(metadata)` (`persistent-memory.ts:67, 101-103`). Qratum's spec'd pre-flight checks content only (`memory-curation-pipeline.md:260-287` mentions only content; metadata is checked for the 4,000-char cap, not policy). A lesson with clean content but a metadata value matching the k:v secret regex passes Qratum and bounces at the gateway.
*Fix:* mirror must evaluate the same concatenated policy subject.

## HALLUCINATED WORK

**H1 — The "shared contract fixture owned by personal-memory" doesn't exist.** The regexes are inline consts in `content-policy.ts:24-37`; the gateway repo has no fixture-export mechanism, no `fixtures/` dir for this. "Parity by fixture" is edictum-schemas culture being invoked as if personal-memory already had the machinery. It's net-new work in two repos; cost it as such.

**H2 — "Exists from Milestone A" oversells the foundation.** Verified reality of the Qratum repo: `internal/` is **16 empty directories, zero files** (scaffolding from May 21) — `internal/capture`, `internal/normalize`, `internal/redaction` etc. contain nothing. All code is 5,812 non-test LOC in a flat `cmd/qrt` main package. Storage is **repo-local `./.qratum/`** (`hook.go:72`, `daemon.go:108`), not `~/.qratum`. There is **no raw archive** (the daemon reads `transcript_path` live at run-once time and archives nothing), **no Session/Revision** (zero grep hits; `qratum.session.v1` schema has no revision concept), **no SQLite** (go.mod has zero dependencies), no lessons, no consent, no publishers, daemon is `run-once` only. What exists: the fast hook (pointer events, well-built) and a normalize->redact->evidence->review->report->ADP run-once pipeline. "Vault ships first" means building essentially the entire vault, plus migrating capture from repo-local spool to a central workspace.

**H3 — "Searchable via SQLite FTS" hides the repo's first third-party dependency.** go.mod is `module ... go 1.26` and nothing else. Adding SQLite means either CGO (`mattn`) or the enormous `modernc.org/sqlite` transitive tree — a major decision under this repo's strict supply-chain rules (`AGENTS.md:117-127`, SHA-pinned actions, readonly modules). The proposal treats it as ambient.

**H4 — The import receipt is an artifact whose only consumer is the other half of the same person.** It exists to patch the information asymmetry that the bundle-pull design itself created (Qratum can't see store outcomes because it was forbidden from talking to the gateway). Machinery manufactured by the architecture to solve a problem the architecture introduced.

## SUSPECTED FLAWS

**S1 — The importer's auth path is the unsolved "headless" problem, inherited. (serious)** Production gateway accepts only OAuth bearer (browser PKCE consent page, rotating refresh tokens, keychain-only storage per the bridge contract) or Cloudflare Access; `LOCAL_HEADER_MODE` is a fail-closed dev fixture (`contracts.md`). A headless service-credential path is exactly the roadmap's *pending* Hermes problem (`roadmap.md:178-181`). The "small importer script" must implement an OAuth client with token rotation and keychain storage, or piggyback on bridge tokens. Doable after one browser dance, but it is not the trivial script the spec implies.

**S2 — Namespace mapping targets a moving floor.** Exports default account-scope to `personal`, but the roadmap plans to reclassify `personal` into `global`/`coding`/`private`/`sensitive:*` and the whole namespace model "needs design" (`roadmap.md:35-82`). Low stakes at N=1, but exporter config will churn.

**S3 — Two curation layers are coming.** The gateway roadmap's own curation track (semantic dedupe, consolidation, contradiction handling, external daily optimizer) plus Qratum's staging/review means the same person will eventually run two curation queues over the same memories. "One home per data class" never assigns a home for curation *decisions*.

## ATTACKED-AND-HELD

- **Every export number is exact.** 556 conversations; 11,962 messages; 447 non-empty summaries; ~187 MB (195,653,921 bytes); account memory exactly 10,097 chars (field is actually `conversations_memory` inside a one-element array with `account_uuid` — implementer note); 7 project memories of exactly 2,274-5,158 chars keyed by 36-char uuids; 13 projects; 5 design chats with `uuid/title/project/messages`; messages carry `sender` (exactly `human|assistant`), `attachments`, `files`, `parent_message_uuid`; users.json is account identity. All 7 project-memory uuids join to project files; all 13 project names are ASCII-safe and slugify cleanly into `/^project:[a-zA-Z0-9._-]+$/`.
- **Gateway behavior claims all verified:** 8,000-char content cap (`config.ts:63` + zod `.max(8000)`); 4,000-char metadata cap; dedup unique key `(namespace, subject, content_hash)` with upsert (migration 002); silent contentClass coercion to `note` (`content-policy.ts:56-60`) — Qratum's refusal to rely on it is correct; namespace allowlist + project regex; no-bypass hard rejects; `clientMetadata` nesting; no bulk endpoint (exactly two tools); scopes; Titan v2 embeddings; TiDB vector store; ECS infra. The roadmap really does gate Qratum integration on contracts, forbid raw transcript ingestion, and leave push-vs-pull open (`roadmap.md:144-155`); `staging:<target>` really is unimplemented proposal-ware, so refusing to depend on it is right.
- **Mandatory split: TRUE** (10,097 > 8,000), and bold-split keeps every chunk under the hard cap.
- **Pre-flight earns its keep on this very dataset:** the account memory trips the legal-dispute pattern and one project memory trips the medical pattern (verified boolean regex tests, no content read). Sensitivity blocks are not hypothetical; surfacing them at review time instead of import time is a real win.
- **Vault-first sequencing survives attack:** capture currently stores only pointer events, processing is manual run-once, and source tools GC transcripts — preservation genuinely is the most urgent and least-built piece. The reorder is right even though it must be done as an explicit spec unlock (F4).
- **Kind enum 1:1 with gateway contentClass:** verified (gateway adds only `connector_test`).

## (a) Verdict: hallucinated vs grounded

This is one of the more grounded proposals I've audited: roughly **80% verified true**, including every single load-bearing number about the export and nearly every claim about gateway behavior. The hallucination is concentrated precisely at the integration seams: a receipt outcome the gateway cannot emit (F1), a "markdown structure" that isn't in the data (F3), a parity fixture that doesn't exist (H1), and a "Milestone A foundation" that is one good hook plus a flat main package with cosmetically convincing empty directories (H2). Pattern worth naming: the facts that were checked were checked perfectly; the things asserted about *counterparties' future behavior* were invented.

## (b) The odd shape, named

**It's a data diode between two processes owned by the same person on the same laptop.** The bridge — export manifest -> published folder -> separate-repo importer -> receipt file -> receipt re-ingestion — is an air-gapped clean-room handoff pattern. Its stated justification ("Qratum never holds gateway credentials") defends a trust boundary that does not exist: both binaries run as the same UNIX user; whoever owns `~/.qratum` owns the keychain the importer reads. The receipt exists only because the diode blinds Qratum to outcomes; the "duplicate" outcome is fictional because the diode also blinds the importer. And the spec's own tell is that the direct `personal_memory_http` publisher is named as the desired future — **the MVP is deliberately more complex than its own end state.**

The deeper version: this is Edictum's enterprise reflexes — evidence-gated progression, approval boundaries, auditable handoff, parity by fixture, publish manifests — transplanted onto a one-person chore of moving ~36KB of already-curated text into a database that person owns. The genuinely valuable governance moment is exactly one screen: "here is exactly what leaves the machine — approve?" Everything downstream of that approval (bundle format, folder publisher, foreign importer, receipt, receipt-ingestion state machine) is ceremony between the founder and himself.

*Better shape:* keep `qrt export memories` with pre-flight + full preview + per-bundle approval (that's the real boundary), then call the gateway directly with a keychain-held token and write the receipt locally from actual responses. One repo, one moving part, receipts grounded in outcomes the gateway actually returns. If credential isolation ever matters for real (multi-agent factory pushing memories), that's an Edictum-shaped problem — gate the call — not a file-format problem. And before any of the 19 schemas exist, a one-day slice (parse memories.json, bold-split, edit in `$EDITOR`, store through the already-connected MCP client) would validate the entire lesson loop end to end.

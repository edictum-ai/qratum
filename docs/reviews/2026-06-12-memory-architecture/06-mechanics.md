# Review 6: Pipeline Mechanics and Contracts

- Lens: data flow, state machines, idempotency, contracts, cross-language assumptions
- Model: Fable (relaunch; the killed Opus run's partial findings were independently re-derived)
- Date: 2026-06-12

---

All gateway claims were verified against source.

## 1. VERIFIED FLAW (serious): The "duplicate" receipt outcome is unimplementable against the current gateway

Verified: **the distinction does not exist.**

- `personal-memory-gateway/src/application/use-cases/persistent-memory.ts:84-98` — `storeMemory` returns `decision: "allowed"`, `persisted: true`, text `Stored memory ${record.id}.` unconditionally on success.
- `personal-memory-gateway/src/infrastructure/tidb/memory-repository.ts:8-42` — the `INSERT ... ON DUPLICATE KEY UPDATE` result is discarded. `affectedRows` (1 = inserted, 2 = updated under MySQL/TiDB semantics) is never read. The only `affectedRows` usage in the repo is in `oauth-client-repository.ts:41`.

So `import_receipt.jsonl` outcome `duplicate` cannot be produced. The importer has no exact-match read path either (search is vector-only, filtered by embedding provider/model).

**Worse — re-running a bundle is not observationally idempotent.** The upsert sets `updated_at = CURRENT_TIMESTAMP(3)` and re-embeds via Bedrock on every duplicate store (`persistent-memory.ts:72` runs embedding before store, unconditionally). `recent()` orders by `updated_at DESC` (`memory-repository.ts:90-96`). A bundle re-run therefore (a) pays Bedrock for every item again, and (b) stamps every exported memory as freshly updated, drowning genuinely recent memories in the no-query `memory_search` view. "Re-running a bundle is safe" (spec line 381-382) is true for data integrity and false for retrieval behavior.

**Fix:** one small gateway change, in the right repo order (gateway first): inspect `ResultSetHeader.affectedRows`, add `created: boolean` to `structuredContent`, and short-circuit the embedding call on duplicates by checking `(namespace, subject, content_hash)` existence first. Then the receipt outcome derives trivially.

## 2. VERIFIED FLAW (serious): Receipt outcome vocabulary is a treaty with a country that speaks a different language

Spec outcomes (`memory-curation-pipeline.md:385-386`): `stored | duplicate | blocked_secret | blocked_sensitive | content_too_large | failed`.

Actual gateway `errorClass` values (verified in `content-policy.ts`, `persistent-memory.ts`, `access.ts`):
`blocked_secret`, **`confirmation_required`** (not `blocked_sensitive`), `content_too_large`, **`metadata_too_large`** (missing from receipt taxonomy), **`empty_content`**, **`namespace_escape`**, **`missing_scope`**.

Concrete failure walk: a memory item whose serialized metadata exceeds 4000 chars (`persistent-memory.ts:11,63-65`) returns `metadata_too_large`. The importer has no bucket for it, so it files it under `failed`. The retry rule says "failed items may be retried; blocked items must not be retried unedited" — so this item is retried forever, failing identically each time. Same for a namespace that fails the `project:` regex (`access.ts:52`): exporter mapping bug becomes an infinite-retry `failed` instead of a terminal block.

The drift already happened **at design time** — the spec invented `blocked_sensitive` while the gateway says `confirmation_required`. That is direct evidence the "parity by memory" failure mode this ecosystem's own CLAUDE.md warns about is operating here.

**Fix:** receipts carry the gateway `errorClass` **verbatim** plus a single importer-derived field `terminal: bool`. No translation layer, no invented vocabulary. The fixture that defines outcome handling lives in personal-memory and Qratum's ingestion consumes it.

## 3. VERIFIED FLAW (blocker): Append-only memory store with no retire path — and the receipt drops the one field that could ever fix it

- The MCP surface exposes exactly two tools: `memory_search`, `memory_store` (`phase0-server.ts`). No delete, no update.
- `access.ts:1-7` defines `memory:delete` and `memory:export` scopes **that nothing uses** — the gateway's own hallucinated-work fossil, and proof the gap was seen and deferred.
- Dedup key is exact content hash (`uk_memory_records_dedupe (namespace, subject, content_hash)`, migration 002). Any rewording = new live row. The old row stays retrievable forever.

What breaks first, concretely: `config.memory.searchLimit = 5` (`config.ts:70`). The account "profile" memory — the highest-value content, 10,097 chars split into many fragments — is exactly the content that gets re-edited. After two or three edit->re-export cycles, each fact exists as 3-4 near-identical siblings competing in cosine space for 5 result slots. Retrieval quality of the entire vault degrades as a direct function of how diligently the founder curates. The system punishes its only user for using it.

And the design discards the escape hatch: `storeMemory` **returns the gateway memory id** (`memoryIds: [record.id]`, `persistent-memory.ts:93`), but the receipt schema is `lesson_id, content_sha256, outcome` (spec line 383-384). The memory id — the only handle a future retire/supersede operation could use — is returned by the gateway and dropped on the floor by the importer.

Also: lifecycle says `any non-exported status -> rejected` (spec line 218). An exported lesson can never be rejected, and the memory it created can never be retracted. Dead end with no door.

**Fix (shape):** (a) add `memory_id` to receipt lines now, store it on the lesson at ingestion — costs nothing, preserves the future; (b) before shipping the bridge, add a minimal `memory_delete` (soft-delete by id, the column and scope already exist) to the gateway; (c) define edit-after-export as: re-staging an exported lesson requires a supersede entry in the next bundle (`retire: [old_memory_id]`). Without (b)+(c), accept in writing that the vault is append-only and editing exported lessons is forbidden, and enforce that in Qratum.

## 4. VERIFIED FLAW (serious): Receipt ingestion — the step that closes the loop — is in no milestone

Milestone mapping, `memory-curation-pipeline.md:416-433`: P0 schemas, P2 adapter+Tier 0, P3 review surface, P4 AI mining, P5 "memory_bundle export + local_folder publish; importer lands in the personal-memory repo." **Qratum ingesting `import_receipt.jsonl` appears in zero milestones.** The lifecycle's terminal state `exported` has no scheduled writer; the phrase used is "Qratum can *later* ingest a receipt" (line 391).

Failure walk: approve 50 lessons -> export -> import succeeds -> receipt written -> lessons remain `approved` -> next `qrt export memories` lists the same 50 as approved-unexported -> operator either re-exports (triggering the updated_at pollution from finding 1) or learns to distrust the command's output. The pipeline's own status display lies after the first successful run of the happy path.

**Fix:** receipt ingestion is part of the bridge acceptance criteria, and `qrt export memories` pre-flight must scan the publish outbox for unconsumed receipts and refuse (or auto-ingest) before building a new bundle.

## 5. VERIFIED FLAW (serious): Nothing pins the importer's `subject` — a mismatch silently makes the whole corpus unretrievable

Dedup key and **every search filter** include `subject` (`memory-repository.ts:60-96`). Subject is the Cloudflare email (`cloudflare-identity.ts:79`) or, in local-header mode, **whatever string is in `x-memory-subject`** (`cloudflare-identity.ts:89-101`; `allowedSubject` waives the check entirely in that mode, line 133-134). The importer contract says "OAuth client or local header mode" with no subject constraint.

Failure walk: importer runs in local-header mode with `x-memory-subject: importer` (or any non-email value). All 200 memories store successfully, receipts say stored. Every daily client authenticates via Cloudflare as the user's email. `memory_search` filters `subject = ?` — result: the entire exported corpus is invisible, forever, with all-green receipts. Dedup also never fires across the two subjects, so fixing the subject later double-stores everything.

**Fix:** the bundle manifest declares the expected subject; the importer asserts its authenticated subject matches before the first store, and the receipt records the subject used. One line each.

## 6. VERIFIED FLAW (serious): The content policy will hard-block legitimate coding lessons, and the bypass it names was never built

Patterns, `content-policy.ts:24-37`:

- `/\b(?:api[_-]?key|secret|token|password)\s*[:=]\s*\S{8,}/i` — lesson text "set api_key: OPENROUTER_API_KEY env var" -> `blocked_secret`, terminal, no bypass. Lessons about auth configuration — a core genre of coding lessons — trip this.
- `/\b(?:medical|diagnosis|medication|health record)\b/i` — "root cause **diagnosis**" -> blocked. A debugging-methodology lesson cannot say "diagnosis."
- `/\b(?:bank account|credit card|iban|routing number)\b/i` — any Stripe/payments-project lesson mentioning "credit card" -> blocked.

The error class is literally named `confirmation_required` — a confirmation mechanism that **does not exist**; `denied()` is unconditional. The policy was designed to stop a chat agent from spontaneously storing PII; it is now sitting athwart a human-reviewed, explicitly-approved curation lane where the human's approval is the confirmation, and there is no way to express it.

**Fix:** in the gateway, split the taxonomy honestly: `blocked_secret` stays hard; `confirmation_required` gains an actual `confirmed: true` input parameter (audited), set only for items a human explicitly approved past a pre-flight warning. Gateway-first work, prerequisite for the lane being usable on real content.

## 7. ATTACKED-AND-HELD (with conditions): Cross-language regex parity

All nine patterns in `content-policy.ts:24-37` are RE2-clean: non-capturing groups, alternation, `\b`, `\s`/`\S`, bounded `{8,}`, `i` flag -> `(?i)`. No lookahead, no backreferences. They port verbatim to Go.

Conditions under which it still fails:

1. **There is no fixture substrate.** The gateway has no unit test harness at all — only smoke scripts under `src/dev/`. The patterns are hardcoded constants nothing external consumes. "Patterns become a shared contract fixture" is not a declaration; it is a workstream: build gateway test infra, refactor `content-policy.ts` to load-or-verify against the fixture, add behavioral input/expected-decision vectors. The architecture prices this at zero.
2. **The policy input is not the content.** `persistent-memory.ts:67,101-103`: policy runs over `content + "\n" + JSON.stringify(metadata)`. The Qratum spec's pre-flight mirrors checks over lesson content only. An item with clean content and a metadata value like `"project": "bank-account-service"` passes pre-flight and is blocked at import. The fixture must pin the exact policy-subject construction (including JS-vs-Go JSON serialization of metadata), or parity is parity over the wrong string.
3. A subtle real behavior worth a fixture case: JSON quoting shields key names — `"token": "xyz"` does *not* match (the `"` breaks `\s*[:=]`), but `token: xyz12345` in prose does. Anyone reimplementing "intuitively" in Go will get this wrong in one direction or the other.

So: parity by fixture, yes — but as behavioral test vectors over the full policy subject, with the pattern list as a secondary artifact.

## 8. VERIFIED FLAW (annoyance): Dead state `staged`, and the missing state that would actually close a race

`suggested -> staged -> approved -> exported` (spec line 216-218). Neither spec gives `staged` a transition trigger, an operational meaning, or a consumer. One state of pure ceremony.

Meanwhile the state machine is missing the state it needs: nothing distinguishes "approved" from "approved and currently inside a published, un-receipted bundle." That gap is what makes the double-export race in finding 4 possible, and it makes "approved lesson whose receipt says duplicate" unanswerable — the lifecycle has no slot for in-flight.

**Fix:** delete `staged`; add `export_pending` (set when a bundle is published, resolved to `exported`/`blocked` by receipt ingestion). This single change closes the re-export race and makes receipt ingestion structurally mandatory rather than aspirational. Also specify: receipt ingestion compares the receipt's `content_sha256` to the lesson's current hash and refuses to mark `exported` on mismatch — that field currently exists in the receipt with no specified consumer (half-designed artifact).

Related race, attacked and held: re-extraction while a bundle is open is safe — the bundle embeds content copies (snapshot), and the lesson dedup key prevents duplicate candidates. Held only as long as the splitter version is frozen — see finding 9.

## 9. VERIFIED FLAW (serious): Tier 0 — the determinism holds, the atomicity claim and the rejection-durability claim do not

Determinism: a pure function over a fixed input is deterministic. Held, trivially.

What breaks:

1. **Fixture-reality mismatch.** Split rules are "headings, bold section markers, bullet groups" (spec line 142-147), and the fixture plan mandates synthetic memories "with markdown structure" (lines 459-461). The real data has zero headings — only `**bold**` and blank lines. The fixtures will exercise a heading-driven code path the production input never takes, and the bold/blank-line path — the only one that matters — gets validated against synthetic structure invented to match the spec rather than the data. Parity-by-fixture produces false confidence when the fixture is shaped to the spec instead of the observed input.
2. **Atomicity vs self-containment is unresolved.** Splitting on bold markers and blank lines yields fragments whose meaning lives in the preceding marker. Is the section marker prepended to each fragment? Unspecified. Without it, candidates violate the spec's own "atomic, self-contained" constraint; with it, the dedup hash now depends on the marker text. Also unspecified: the fallback when a single bold section exceeds 8000 chars (splitting is "mandatory" but the second-level rule doesn't exist).
3. **Rejection durability is a frozen-splitter illusion.** Rejected lessons stay on file keyed by `scope + sha256(normalized content)` (lines 253-258). The first splitter bug-fix (`memory_parse.v2`) shifts fragment boundaries, every content hash changes, and **every previously rejected lesson resurfaces as a new suggestion** — the precise outcome the mechanism exists to prevent. The guarantee silently expires the first time the transform is improved.

**Fix:** fixtures must be shaped from the observed structure (bold + blank lines, zero headings), including one >8000-char heading-free section. Specify marker-prepending and the second-level split rule. Either key rejection memory by source span lineage (`source_digest` + offset range) so re-splits can inherit rejections, or document plainly: "upgrading the splitter resets rejection memory" and surface that at re-extraction time.

## 10. SUSPECTED FLAW (annoyance->serious): The local_folder hop is half real, half ceremony — and the half that is humanware is the half that will fail

What the hop genuinely buys: credential isolation (in intent), and a durable inspect-before-send artifact (real — consistent with the trust-boundary table).

What it does not buy: failure isolation between two processes on one laptop run by one person. The Publisher port, publish modes, outbox/history, and digest-verified folder-to-folder copy is machinery for moving a directory across the same disk.

The actual cost: the loop is **three unattached manual human steps** — `qrt export memories`, run the importer script, then (unscheduled, finding 4) ingest the receipt. No process supervises completion. For a solo founder, the predictable trajectory is: executed faithfully twice, then the receipt step gets skipped, then the state machine and reality diverge permanently.

**Fix:** keep the bundle as the artifact and the credential split; collapse the humanware. Either the personal-memory importer ends by invoking `qrt import-receipt <path>`, or `qrt export memories --deliver` shells out to the importer (which holds its own creds) and ingests the receipt in the same invocation. One command, same isolation, no discipline required.

## 11. HALLUCINATED WORK (annoyance): `provenance.json`, and the spec's undead deletions

- **`provenance.json` has zero consumers.** The importer contract reads `memories.jsonl` and writes the receipt — it never opens `provenance.json`. The gateway receives provenance per-item inside `metadata.clientMetadata`. Qratum keeps provenance inside lesson objects. This is a third copy for nobody. Fold a one-line provenance summary into `manifest.json` and delete the file.
- **The deletions didn't happen.** The architecture says LessonBackend/VectorBackend/TiDB-direct/DuckDB are DELETED, but the on-disk specs still contain all of them: `operational-model-redesign.md` lines 220-222 (ports), 243 (SQLiteLessonBackend), 1539-1568 (backend stack incl. sqlite-vec, DuckDB), 460-486 (tidb_remote config), and — directly contradicting "explicitly NOT a queryable knowledge store" — `memory-curation-pipeline.md:291-292`: "Lessons are filesystem objects under `~/.qratum/lessons/`, **projected into the SQLite LessonBackend**." Until the prose is actually cut, every future implementation prompt generated from these files will resurrect the deleted machinery.
- Gateway-side fossil, same genus: `memory:delete` and `memory:export` scopes defined and never used (`access.ts:1-7`).

## 12. ATTACKED-AND-HELD: SQLite FTS + filesystem source of truth

Crash-mid-write and index drift attacked; held at this scale. Existing code already does tmp+rename atomic writes (`cmd/qrt/hook.go:235`, `daemon.go:468`). The spec declares backends rebuildable projections with delete-by-object-id and a `qrt debug store verify` command. A crash between object write and index update leaves an unindexed object until rebuild — for one user over thousands of files, that is a shrug, not a flaw. Residual annoyance: nothing ever *schedules* verify/rebuild, so drift is discovered by missing search results. Note also that none of this exists yet — `internal/` is empty directories; all search/lesson machinery is pure spec.

## The odd shape

**This design treats a one-person, one-laptop handoff between two repos as a B2B integration between two sovereign companies — and puts all the formality on the boundary that doesn't exist while the boundary that does exist gets the weakest contract.**

The bundle contract, wire-shaped items, receipt files, outcome vocabularies, importer "owned by" the other repo, parity fixtures — that is the protocol stack two organizations build when they cannot read each other's code, share a process, or trust each other's uptime. Here, both sides are the same person, the same laptop, the same filesystem, the same authenticated email. The trust boundary being elaborately defended (Qratum must never know gateway outcomes except via a receipt file dropped in a folder) is organizational fiction; it is why the loop needs three manual steps and a receipt-ingestion phase that no milestone schedules.

Meanwhile the *real* boundary — laptop to AWS, the only place data actually leaves the machine — gets the flimsiest contract in the system: an append-only store with no delete tool, a "confirmation_required" class with no confirmation mechanism, a response that cannot say "duplicate", and a receipt that throws away the memory id.

The tell that the shape is forced: the parent spec lists "persistent approval/pending item queues" as an explicit non-goal (`operational-model-redesign.md:2280`), and this pipeline's lesson staging area *is* a persistent approval queue, renamed. The design is routing around its own principles instead of revising them.

The better shape: keep the artifacts (bundle, receipt — they are good audit objects), delete the organizational fiction. One command runs export->import->receipt-ingest as a single supervised operation with the credential split preserved at the subprocess boundary. Spend the saved formality budget on the real perimeter: `created` flag, `memory_delete`, confirmed-override, and memory ids in receipts — all gateway-side, all small, all verified missing.

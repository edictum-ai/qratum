# Qratum Verification Benchmark — Integrated Design

Status: Design proposal (P0 contract work). Scoped to `specs/current/qratum-vault-first.md` (Accepted 2026-06-14). Every claim below was verified against shipped code in `cmd/qrt/` and `internal/`, and several were confirmed by building `qrt` and running the pipeline. Where shipped code fails an invariant, that is stated as a **planned-RED** gate, not glossed over.

> **What this document is, in one paragraph.** Qratum captures Claude transcripts, scrubs secrets out of them, and turns them into shareable reports. This document defines what it means to call that pipeline "verified" — a concrete set of tests, each leading with the problem in plain words. The recurring theme: several safety promises the original design claimed are actually **false against the code that ships today**. The benchmark is written so those tests **fail loudly** until the code is fixed, rather than being quietly trimmed to whatever currently passes.

## Plain-language glossary

Read this once; the body then prefers the plain word and keeps the term in parentheses where useful.

- **Redaction** — scrubbing secrets (API keys, passwords, paths) out of text so they don't end up in a shareable file. A scrubbed value is replaced with a placeholder like `[REDACTED_SECRET_001]`.
- **Recall** — of all the secrets that exist, the fraction the redactor actually catches. A *recall failure* means a real secret slipped through.
- **Precision** — of everything the redactor scrubbed, the fraction that really was a secret. A *precision failure* means it over-redacted something harmless (e.g. a plain commit hash).
- **Planted-secret corpus** — a test data set where we deliberately insert known secrets, so any that survive are an automatic, provable leak.
- **Reflection canary** — a unique throwaway token (a UUID) that a test injects into *every* text field of a data structure by walking it programmatically ("reflection"). If any canary token survives into an output, that field leaked. Because the tokens are generated from the struct itself, no field can be quietly left out of the check.
- **Idempotent** — runs every time without changing the result; running it twice is the same as running it once.
- **Monotonic** — only moves in one direction; here, recall coverage can stay the same or improve but is never allowed to silently shrink.
- **TOCTOU** ("time-of-check to time-of-use") — a race condition: code checks a condition, then acts on it, but something changes in between. Here, two captures can both see "file doesn't exist" and then both write, so one clobbers the other.
- **Blob** — an immutable, content-addressed copy of a file stored in the vault (named by its hash). The vault's permanent record.
- **Ref** — a small pointer record that names a blob (by digest) and ties it to a transcript.
- **ADP** — the export format qratum emits (a strict JSONL artifact built by `buildADPStrictJSONL`). One of the seven shareable outputs.
- **Refinery** — the processing pipeline: `normalize → redact → evidence → review → report → export`.
- **Drift** — qratum's heuristic count of how far the live transcript tree has diverged from what the vault has archived.
- **Golden / golden test** — a committed expected-output file; the test asserts the pipeline reproduces it byte-for-byte. The goldens are qratum's real contract.

---

## 1. Qratum inventory

**What this section is.** A ground-truth map of what qratum actually ships today, versus what is only written down in specs. The headline correction: the vault runtime is already merged and running, even though `SPEC.md`/`AGENTS.md` still call P0 "docs only."

Concretely: qratum is a single Go binary, `qrt`. The vault runtime is **merged** and live (Details: `internal/vault/vault.go`, `internal/workspace/workspace.go`, `cmd/qrt/vault.go`, `cmd/qrt/hook.go`). The Milestone A refinery (`normalize → redact → evidence → review → report → export`) is shipped and runs via `qrt daemon run-once`.

| Bucket | Item | Evidence / status |
|---|---|---|
| **Shipped & tested** | Hook capture (copy-on-capture, blob+ref, degraded cases) | `hook.go:108-197`; `vault_test.go:14,43` cover raw_missing + copy-failure-surfaced-by-doctor |
| | Vault archive/backfill/backup/doctor commands | `vault.go`; `vault_test.go:165` (backfill idempotent), `:217` (archive kinds), `:258` (backup --verify copies), `:86/:131` (hook install idempotent/double-capture reject) |
| | Daemon run-once: idempotency, partial-artifact reject, missing-transcript loud-fail | `daemon.go`; `main_test.go:1470,1825,1708,1767,1790` — all present |
| | Normalize/redact/evidence/review/report/export golden pipeline | `main_test.go`, `report_test.go`, `ui_test.go`, `regen_test.go`; golden fixtures committed |
| | Deterministic redaction of the **enumerated** classes (sk-/ghp-/xox-, private-key block, credential URL, JWT, SECRET/TOKEN= assignment, absolute POSIX/Windows path, ≥32-char no-slash high-entropy) | `redact.go:18-26`; verified the api-key/JWT/db-URL/path classes redact end-to-end through the real daemon |
| **Shipped but thinly tested** | `internal/vault` package | NO standalone unit test; covered only transitively through `cmd/qrt` (red-team + architect agree) |
| | Redaction **recall** breadth (how many real secrets it catches) | Only ever validated on a pre-built `secret-session.input.json`; `transcript-with-secret.jsonl` has **zero** Go references (confirmed `grep`). Full-pipeline-through-normalize recall is untested |
| | Report/UI no-leak | `TestReportDoesNotLeakFixtureSecrets` (report_test.go:93) checks only the ~7 catchable literals from a hand-built fixture; `TestBuildUISessionDetailUsesRedactedSourceMetadata` (ui_test.go:79) proves the DTO *reads* `context.redacted` but that struct is never scrubbed for git/time/event fields |
| | Backup `--verify` | `vault_test.go:258` is happy-path only; corruption-detection and round-trip-restore untested. Also `verifyTree` re-reads source live, never the archive-time digest (vault.go:532-565) |
| | Doctor truthfulness | One copy-failure case tested; staleness/backup/drift permutations and always-present cloud line untested |
| **Spec'd, not built** | Refinery reads from **vault blobs** | NOT built — `daemon.go:159` resolves & requires the **live** `transcript_path`. A Claude-deleted-but-blob-present transcript fails the refinery (verified). This is the gap that makes the headline recoverability claim untrue for derived artifacts |
| | Central-workspace artifact placement | Events go to `~/.qratum/events` but all derived artifacts land **repo-local `./.qratum/`** (daemon writes relative to `os.Getwd()` — verified: secrets written to `./.qratum/sessions/*.normalized.json`). `vault-first.md:80` "migrating off repo-local" is unstarted |
| | Config schema (`config.toml` / `config.schema.json`) | `SPEC.md` P0 deliverable; **absent** from repo (no config schema in `schemas/`, no config handling in `main.go`) |
| | Session revision/resume worker, SQLite projection, retention/delete verbs, tombstones | All `P0 Non-Goals` / spec'd-only |
| | JSON Schema validation | `schemas/*.json` are **orphaned** — zero Go references (confirmed `grep`); no validator, no Makefile target; no `additionalProperties` anywhere |
| **Parked (with trigger)** | Local SQLite FTS search | Trigger: Arnold greps the vault twice (first 3rd-party Go dep — supply-chain decision) |
| | Thin claude-ai-export normalizer | Trigger: summary/conversation mining actually wanted (Tier-1 summaries preferred) |
| | Git-native curation lane (the "dream") | Trigger: real recurring candidates exist (`PROPOSAL.md` W5) |
| **Open decisions** | **Insurance vs dream** fork | `BACKLOG.md` pre-flight: preserve+import+stop, OR preserve-as-foundation+curation lane. Unresolved by Arnold |
| | **Daemon vs no-daemon / passive capture** | No scheduler ships. `daemon run-once` and `backfill` are manual; preservation freshness depends on out-of-band cron/launchd the product does not install |
| | Benchmark scope | Whether to fix the redactor (extend coverage) vs drop unredacted fields from artifacts; whether normalized-session-on-disk is in scope as raw retention store |
| **Cross-repo (gateway)** | `memory_import_receipt` archive | `KindMemoryImport` shipped (`vault.go:52`); qratum's **receiving** end is ready but **dormant**. Producer (`scripts/import-claude-memories.ts`) lives in personal-memory, gated on gateway Phase 1, **not built** |
| | Receipt **schema** | "to be defined" — no contract file exists yet. Gateway grants/supersedes[]/contentClass are `BACKLOG.md` D/E/F, undeployed |

---

## 2. The verification benchmark

**What "verified" means here.** Not "the unit tests are green." It means the trust pipeline **demonstrably holds its security and integrity promises end-to-end on a known data set, byte-for-byte deterministically**, against the code that actually ships.

**The core correction.** A red-team pass found that several promises the original design treated as guaranteed are **false against shipped code**. So the benchmark deliberately makes those tests **fail today** (planned-RED) until the code is fixed. It must never be quietly narrowed down to whatever currently passes.

Three structural rules apply to every test below:

- **R1 — Use adversarial data, not curated data.** Plant secrets independently of the redactor: inject unique throwaway tokens (reflection canaries) into every text field, rather than hand-writing a list of "secrets we already know we catch." Any token that survives is automatically a recall failure — not something we can wave away as "an un-listed literal."
- **R2 — Dropping a field is not the same as redacting it.** For any field an artifact actually carries, check two things: the canary token is gone **AND** the redaction placeholder is present. For a field an artifact structurally drops, check that it's dropped — but **do not** count that as evidence the redactor works.
- **R3 — Drive the real shipped CLI, not just library helpers.** Security and integrity tests must go through the real entrypoints (`runDaemonOnce`, `runWithIO`), so where files land on disk, whether the live transcript is read, and the events-vs-artifacts store split are all exercised — not assumed away.

### D3 — Redaction safety (the crown jewel)

**The problem in one line.** This is the most security-critical area and the biggest current gap: if the redactor misses a secret, raw credentials end up in a file someone shares. We have empirically confirmed leaks in the shipped redactor (Details: `redact.go`):

- **Whole fields are never even scanned.** `started_at`, `ended_at`, `source_event_id`, `git.branch`, and `git.head_sha` pass through **verbatim** — the redactor never looks at them. `git.branch` can carry names like `feature/customer-acme-prod-keys`. The committed golden ships an unredacted repo URL (Details: `secret-session.redacted.golden.json:32` contains `git@github.com:edictum-ai/qratum.git`).
- **Whole classes of secret leak.** AWS keys containing a `/` slip through, because the entropy detector returns "not a secret" for any string containing `/` (Details: `redact.go:452`). Relative paths like `./config/prod.env` and home paths like `~/.aws/credentials` slip through, because the POSIX path pattern only matches absolute paths (Details: `redact.go:24`). SSH remotes like `git@github.com:org/repo.git` and space-separated assignments also slip through.
- **An active partial-redaction bug.** Given `PASSWORD => hunter2pass`, the assignment regex matches `PASSWORD =` and redacts only the `>` character, emitting `PASSWORD =[REDACTED_SECRET_001] hunter2pass` — the real secret survives in cleartext. The cause: the regex requires `:` or `=`, and the `>` gets captured as the value (Details: `redact.go:22`).

**Acceptance tests:**

- **Run a planted-secret data set through the WHOLE pipeline** via the real daemon: `normalize(transcript-with-secret.jsonl) → redact → evidence → review → report → ADP export → capture event`. This wires in `transcript-with-secret.jsonl` for the first time.
- **Reflection-canary recall (R1):** inject a unique UUID-v4 token into **every** text field of a fully-populated `qratumSession`, recursing into `*qratumGitInfo`, `[]qratumTurn`, `[]qratumToolCall`, and nested tool inputs. Assert **none** of those tokens survive into the redacted session, the evidence, the review, the report HTML, the ADP export, or the capture event. This is the authoritative gate and it **fails today**. It is required to stay RED until the redactor either routes the leaking fields (`git.branch`/`git.head_sha`/`source_event_*`/`started_at`/`ended_at`/tool source ids) through `redactString`, **or** those fields are provably dropped from every artifact.
- **Field-coverage contract (no escape hatch for exemptions):** commit the exact set of fields the redactor scans, and assert it equals the full set of text fields discovered by reflection. Any field placed in a "non-sensitive, not redacted" set needs a reviewed written justification **and** a positive test proving that a planted secret in that field is non-shareable. Recurse into nested structs, slices, and pointers.
- **Per-class recall**, including the leaking classes above. Classes genuinely out of scope go into a version-controlled **known-miss file** (an "xfail" list) with a note on the residual risk. A **corpus-completeness meta-check** fails if any class is in neither the recall data set nor the known-miss file — this closes the "invisible omission" hole.
- **Fix and gate the partial-redaction bug:** after every placeholder, assert the remaining text contains no leftover secret token.
- **Precision, hardened (R3):** the benign data set **must include** the cases known to trip the redactor — a bare 40-hex commit hash, a hyphenated UUID, and a `sha256:` digest (all match `highEntropyPattern`). Over-redacting these is a **bug to fix in the regex**, not a budget to inflate. The over-redaction budget `K` is an externally-justified hard number (target 0 on real commit hashes/UUIDs sampled from repo history). Changing the budget requires separate sign-off, not just a code diff.
- **Determinism + runs-the-same-every-time (idempotent):** identical input produces identical `[REDACTED_*_NNN]` numbering, and redacting an already-redacted value changes nothing (`redact(redact(x)) == redact(x)`).

**Golden data set (planted-secret design):**

- `transcript-with-secret.jsonl` wired into a full-pipeline test, with an explicit list of the planted literals.
- An **every-field-canary** session built at test time by reflection (R1) — the token set is derived from the struct, never hand-curated.
- An **evasion** fixture: AWS key with a `/`, a relative path, a `~/` home path, an SSH remote, space- and arrow-separated assignments, and a mid-value-truncated secret.
- A **known-miss / xfail** file covering the classes we knowingly don't catch yet: unicode/zero-width-split keys (verified that `AKIA​IOSFODNN7EXAMPLE` survives), homoglyphs, base64/hex encodings of a secret, AKIA IDs, bare 40-hex tokens, Stripe `pk_`/`rk_`, `AIza`, `glpat-`, and `SG.` — each with a residual-risk note.
- A **benign / precision** fixture including the commit-hash / UUID / `sha256:` tripwires.

**Metric.**
- Recall on the reflection-canary plus covered data set = **binary 100%** (any leak blocks release).
- Extended-class recall = a published **percentage** with explicit known-miss entries (informational, but the data set must be non-empty and recall must not regress — see §3).
- Precision = a count gate against a hard, justified `K`.

**The "no raw leak on EVERY output format" property test.** One shared checker (literal-absent + placeholder-present, per R2), driven by the reflection token set, runs over all seven artifact byte-streams.
**Honest caveat the scorecard must print:** redaction happens as a **single upstream pass** — `writePipelineArtifacts` redacts once (Details: `daemon.go:404`), and evidence/review/report/ADP all derive from that one already-redacted session. So the seven checks are **correlated, not seven independent layers**. To make them truly independent, add a **terminal secret-scan gate** on each emitted artifact's actual bytes as defense-in-depth. Without that, every downstream artifact inherits exactly the redactor's recall — and the scorecard says so plainly.

### D4 — Trust-boundary enforcement

**The problem in one line.** Per the UI/Data rules in `AGENTS.md`, the visible surfaces must consume safe DTOs (not raw models), and raw transcripts must never reach a shareable artifact or an external service. This dimension proves that boundary actually holds.

**Acceptance tests (red-team-corrected):**

- **Export is gated by redaction (R2):** `buildADPStrictJSONL` redacts first (Details: `export.go:128` calls `redactQratumSession`) — verified clean for in-scope secrets. But the ADP **omits** git / started_at / source_event_* by its **shape** (Details: `export.go:179-277` emits only turns/tool_calls/commands). So a canary missing from the ADP because the whole field was dropped proves **nothing** about redaction. Split the test: for values the ADP actually carries (turn content, tool kwargs including `file_path`, command bodies) assert canary-absent + placeholder-present; for fields the ADP drops, assert the structural drop and **do not** credit it as redaction. Run this against the **evasion** data set (relative and home paths flow straight into `file_path` kwargs — Details: `export.go:327`).
- **Defense-in-depth re-scan of the ADP:** `sanitizeADPMap` only strips qratum-only **keys** (Details: `export.go:373-384`), never re-scans the values. Add an independent byte-scan of the final ADP for the planted token set.
- **ADP internal-key strip:** feed a session carrying `secret_map` / `provenance` / `redaction` / `artifact_paths` / `pipeline_status` / `x-qratum-*`, then assert none of those appear in the export.
- **Report no-leak (rewritten):** the report does **not** render the `turns[]` / `tool_calls[]` arrays. The real surface that can leak is the summary table (Details: `report.go:303-315` — started/ended/source-event/git remote/branch/head-sha). Scan the **full** HTML for every planted literal (the R1 token set), not a fixed 7-item list. **Fix required:** `buildUISessionDetail` reads `StartedAt`/`EndedAt` from the **raw** session (Details: `ui.go:675-676`) and reads git from `context.redacted` — but redaction never scrubs those fields, so the DTO leaks. Route the time/git/event fields through a source that is actually redacted.
- **UI DTO carries no raw internals:** feed the secret data set through the real `ui` command and assert there is no `transcript_path`, no raw turn content, and zero planted literals.
- **No network egress:** an import-graph gate proving the refinery (`normalize`/`redact`/`evidence`/`review`/`report`/`export`) imports no `net/http`, no `net`, and no external client.
- **Logging channel (new):** `hook.go:150` writes `copy_error` to stderr, and that message embeds a filesystem **path**; capture events store the full `transcript_path` and `cwd`. The operational model bans full paths in logs. **Decide and assert:** full paths inside local-only *events* are acceptable, but full paths or raw content in *stderr/logs* are not — assert the warning channel emits no raw transcript content.

### D1 — Capture fidelity + hook safety/speed

**The problem in one line.** The capture hook is the only sensor for an irreversible event (a transcript getting captured), and `AGENTS.md` forbids it from doing any network, parsing, or LLM work. This dimension proves the hook stays fast and safe and never loses or mangles a capture.

**Acceptance tests:** exactly one capture event with populated `raw.*`; degraded cases (empty or missing transcript → `raw_missing`/`failed`, counters incremented, a stderr warning, exit code 0 — Details: `hook.go:142-197`); payloads over 1 MiB and empty/invalid JSON rejected; no transcript-line content in the event; `deterministicEventID` stability; and a **speed gate** (a 50 MB synthetic transcript copied and hashed under the threshold).

**Red-team fix — scope the import gate precisely.** Everything in `cmd/qrt` is `package main`, so a package-level `go list -deps` starting from `spoolClaudeCodeHook` resolves to the **whole binary** (which also contains the refinery). That makes the gate coarse and brittle. **Plan:** extract the hook capture path into a new `internal/capture` package that imports only `crypto/sha256`, `os`, `io`, `vault`, and `workspace`; pin that allowed import set in a golden file; and add a **negative test** asserting that introducing a `net` import flips the gate RED. Until that refactor lands, the scorecard states honestly: "the package-main import gate proves the binary has no network imports, NOT that the hook itself is function-isolated."

### D2 — Vault integrity

**The problem in one line.** The vault (`internal/vault`) is the preservation engine — the permanent record — and it currently has no standalone test. This dimension proves blobs are stored correctly, never silently changed, and never lost.

**Acceptance tests (red-team-corrected):**

- **Byte-equality is the gate, not a re-hash.** Re-hashing the committed blob and comparing it to `ref.Digest` proves nothing — `ArchiveFile` hashed the same stream it wrote (Details: `vault.go:202`, `io.MultiWriter`), so it can only ever agree with itself. Instead, hash the **original source** with an independent reader and compare that to both the blob bytes and `ref.Digest`.
- **Dedup:** two files with identical content produce one blob (`BlobCreated` is true the first time, false the second).
- **Immutability:** no rewrite of an existing blob; a raw-ref collision is rejected (Details: `vault.go:265`).
- **Atomicity:** no `.tmp` file left behind on success; if a write failure is injected, no partial blob or ref is left behind.
- **No-delete:** a grep gate plus a test that backfill/archive never reduces `blob_count`. **Reframed per red-team:** state plainly that "deletion verbs are unbuilt; immutability holds **by absence, not enforcement**." Parameterize the test so the spec'd `qrt raw purge` / `workspace wipe` (from the operational model) don't auto-fail when added later — instead, those are **required** to write a tombstone and to refuse deleting a blob that is still referenced.
- **Backfill is idempotent** (runs again without changing anything); summary counts match the filesystem.
- **NEW — ref-id collision bug (red-team).** `RawRefIDForDigest` truncates the digest to a **12-hex prefix** for the ref filename (Details: `workspace.go:88-93`), while blobs use the full digest. Two distinct 64-char digests that happen to share a 12-char prefix collide into one ref path, and the second is **rejected as a false collision** (Details: `vault.go:265`). Add a test with two such digests asserting both get stored. **Flagging this as a latent correctness bug** — the birthday-bound collision risk grows across machines and hundreds of blobs.
- **NEW — multi-machine merge (red-team, testable now).** The spec claims cross-vault merge is "blob-dedup-clean today." Build two independent temp vaults, union their `raw/` trees into a third, and assert a consistent, collision-free union with no data loss. If this is left unverified, the scorecard says "cross-vault merge UNVERIFIED" rather than implying content-addressing guarantees it.

### D5 — Determinism / reproducibility

**The problem in one line.** The golden tests are the real contract, and a trustworthy attestation requires that the same input always produces byte-identical output, on any machine.

**Acceptance tests:** golden byte-equality; run-twice stability in two temp workspaces; machine-independence (`QRATUM_HOME` + `t.TempDir`, ToSlash paths); stable ADP ordering; and a regen guard.

**Red-team fix — determinism must run DOWNSTREAM of leak-freedom.** A regen guard that froze the current golden (Details: `secret-session.redacted.golden.json:32`, which holds the unredacted SSH remote) would **certify the leak as the intended, reproducible output** — exactly backwards. Add a standalone lint, independent of the redactor: **no committed golden contains a known secret or internal identifier** (for example `edictum-ai/qratum.git` or a real head SHA). Redact the SSH remote and head_sha in the golden *before* treating that golden as the contract.

### D6 — Idempotency + crash recovery

**The problem in one line.** `run-once` must be safe to run again and again, and a crash mid-run must not leave half-written artifacts behind.

**Acceptance tests:** idempotent re-run (the behavior exists — gate it); partial-artifact detection (exists — Details: `main_test.go:1825`); crash-mid-write tmp-leak recovery; missing-transcript loud-fail with `raw_missing`/`raw_copy_failed` skips.

**Red-team-corrected additions:**

- **Recoverability is its own first-class dimension (D6a), planned-RED.** Do NOT let "missing transcript → fail loud" stand in for actual recovery. The whole reason the vault exists (Details: `vault-first.md:75`, `BACKLOG.md` B) is the promise "capture → delete the source → the data still survives." That promise is verified **false**: the daemon reads the **live** path, never the blob (Details: `daemon.go:159`). Test: capture → delete the source → run the refinery → assert it succeeds **by reading the blob**. This **stays RED** until the daemon is rewired, and the scorecard headline must carry "recoverability NOT wired." It must not read TRUSTED while this is broken.
- **Refine-source consistency (red-team):** capture records `raw.digest`, but then the live transcript can be appended to or otherwise changed (a resumed session). Test: mutate the live file, run `run-once`, and assert the refinery **either** re-hashes and refuses on a mismatch **or** the scorecard documents "the refinery trusts the live bytes; the vault blob is the only authoritative copy, and artifacts may diverge from the recorded digest."
- **Concurrency race (TOCTOU) — RED, not just a "stated gap."** `nextCaptureEventPath` does an `os.Stat` and then the caller does an `os.Rename`, with no lock in between (Details: `hook.go:340-398/375`). Two concurrent hooks can both see "file does not exist" and the second one clobbers the first. The architect's sequential two-hook test **cannot trigger the race**, so it would wrongly mark this green. **Fix:** create with `O_EXCL` instead of stat-then-rename, with a real goroutine-hammering test. If left unfixed, mark this **RED known data-loss path** — never "informational."

### D7 — Operational truthfulness (doctor)

**The problem in one line.** `doctor` reports the health of an irreversible store; a doctor that reports false-green manufactures false confidence. This dimension proves doctor tells the truth.

**Acceptance tests:** no-hook / stale-backfill / copy-failure / unverified-backup warnings (using injected state + a stubbed clock); an **always-present cloud-blind-spot line** (Details: `vault.go:136` — a hard literal gate); and on a healthy path, "warnings: none."

**Red-team-corrected — drift is a heuristic, and it goes false-RED on the product's own success case.** `transcript_drift` is computed as `len(ListTranscriptFiles()) - transcriptRefCount` (Details: `vault.go:113-121`). `ListTranscriptFiles` reads the **same live `~/.claude/projects` tree** that backfill archives from, so post-backfill drift is **0 by construction** — a tautology, not a real check. Worse, a **source-deleted-but-blob-present** transcript (the exact GOOD case the product exists for) drives drift **negative** ("extra blob") and trips a false warning. Multi-machine refs inflate it permanently. **Tests must:** (a) capture → delete the source → assert how drift reports it (today it false-warns); (b) require the scorecard to **label `transcript_drift` a heuristic indicator, not a correctness gate**; and (c) ideally compare archived refs against an **independent expectation** (the count of `session_end` events whose `copy_status` is `copied` or `deduped`) rather than against the live tree.

### D8 — Backup restorability

**The problem in one line.** "A backup that has never been verified is not a backup." This dimension proves backups actually detect corruption and actually restore.

**Acceptance tests:** verify-success (exists — gate it); **corruption detection** (flip one byte in a copied blob → `verifyTree` FAILS with "backup verify mismatch"); round-trip restore (point `QRATUM_HOME` at the destination, then status/doctor/Summary match); and refuse when destination equals home (Details: `vault.go:409`).

**Red-team fix:** `verifyTree` re-reads the **live source** and re-hashes it (Details: `vault.go:532-565`) — it never compares against the **archive-time** digest stored in the raw-ref. So a source that was mutated after capture passes verification, and corruption that happens to match in both source and destination also passes. **Verify against the recorded `ref.Digest`** instead, so post-capture source drift and matched corruption both become detectable.

### D9 — Schema / contract conformance

**The problem in one line.** The JSON schemas in `schemas/*.json` are orphaned — zero Go references (confirmed) — and no test enforces them, so the emitted artifacts can quietly drift away from the published schemas. The real contract today is the goldens, not the schemas.

**Acceptance tests (red-team-corrected; this is a prerequisite-fix dimension):**

- **`additionalProperties:false` comes first.** No schema sets it (confirmed). A validator that only checks required keys + types + enums **cannot forbid a leaking extra key**, so "every artifact validates" is trivially green even for a session carrying a raw `transcript_path` or `secret_map`. **Require `additionalProperties:false` (or explicit forbidden-key lists) on every schema before this gate counts for anything.**
- **Drift direction is pinned to emitted-keys ⊆ schema-declared-keys** — the only direction that catches a leak. This **fails today on every artifact**, which is the point: it stays planned-RED until the schemas are completed. Make it a HARD gate with teeth.
- **Validator self-test:** assert the validator **REJECTS** an instance with an injected extra secret-bearing key — not merely that it accepts valid ones.
- **The denominator is emitted objects, not files.** Enumerate every emitted `schema_version` literal (`qratum.event.v1`, `qratum.session.v1`, `qratum.evidence.v1`, `qratum.review_card.v1`, `qratum.raw_ref.v1`, `qratum.vault_state.v1`, `qratum.ui.api_error.v1`, **plus the schemaless ADP export and the redaction summary**) and FAIL loudly when an emitted object has **no** schema file. Add a name-mapping assertion (file `qratum-X.v1` ↔ emitted `qratum.X.v1`) so the hyphen-vs-dot naming drift can't hide a missing schema. Note that the **config schema is a missing P0 deliverable** in the residual block.
- **Mini-validator stays stdlib-only** (supply-chain rule). A vetted, ≥7-day-old pure-Go validator only with Arnold's sign-off.

### D10 — Cross-repo import (GATED on gateway)

**The problem in one line.** `KindMemoryImport` is shipped and qratum's receiving end is ready, but it's dormant — the producer that would feed it lives in personal-memory and isn't built. So most of this can't be meaningfully tested yet.

**Red-team-corrected — the "testable now" half tests almost nothing real, so demote it:**

- **Testable now (a contract check only, NOT a security proof):** `qrt vault archive <receipt> --kind memory_import_receipt` round-trips. **But** the no-raw-leak check against a hand-written synthetic fixture is **circular** — it only proves that a clean fixture is clean. Mark it **NOT-YET-MEANINGFUL** as a leak proof until the real producer exists.
- **Prerequisite:** define the receipt JSON Schema as a committed contract and wire it into D9. Until that schema exists, the scorecard reports D10's testable-now part as **"contract-undefined, not testable,"** not as a passing gate.
- **Default-kind footgun:** `parseArchiveArgs` defaults `--kind` to `source_metadata` (Details: `vault.go:312`), so a receipt archived without the flag is **mislabeled**. Pin `kind=memory_import_receipt` in the test and document the footgun.
- **Gated (gateway Phase 1+):** receipt round-trip across repos; idempotent re-run with `supersedes[]`; `namespace_forbidden`; hard-reject of an unknown `contentClass`; and **reject a receipt whose outcome/errorClass is outside the gateway's real vocabulary** (the review's central lesson — the dead bridge invented a `duplicate` outcome). Reported as "not-yet-runnable (gateway Phase 1+)."

### NEW dimensions the red-team's completeness lens forces

- **D11 — Artifact placement / workspace containment (testable now, likely RED).** **The problem:** even a perfectly redacted artifact is unsafe if it lands somewhere it shouldn't. Verified: the daemon writes the **raw, un-redacted normalized session** to repo-local `./.qratum/sessions/*.normalized.json` (it contains `sk-ant-api03-…` and `supersecret`), and all derived artifacts land relative to `os.Getwd()` rather than under `QRATUM_HOME`. Name the three-location split clearly: events live in home, the transcript lives at the live path, and artifacts land repo-local. Then **either** assert artifacts land under `QRATUM_HOME`, **or** document that secret-bearing redacted/report/export files currently escape into a possibly-git-tracked repo dir, and assert they never reach the ADP / an external service / the network. **D4's "raw never in shareable artifacts" is moot if the artifacts themselves sit in a repo dir** — so this is in-scope, not assumed away.
- **D12 — Passive-capture liveness (testable now).** **The problem:** capture only helps if something actually runs the pipeline. Assert that the installed global hook command equals the shipped capture entrypoint, and that doctor's `last_capture_at`/`last_backfill_at` staleness gates fire. **State plainly: qratum ships no scheduler; `run-once` and `backfill` are manual.** This is the 0%-engagement failure the spec was written to prevent (18 events captured, 0 ever processed), and it is the daemon/no-daemon open decision — it must be named.
- **D13 — Source-scope guard (testable now).** **The problem:** the redactor only knows how to handle Claude Code sessions, but archive can ingest other vendors' blobs that have no redaction path. `validateQratumSession` rejects any non-`claude-code` source (Details: `redact.go:128`), yet `archive` can ingest Codex/vendor blobs (Details: `vault.go:29,428-438`) that have **no redaction or export path**. Assert that exporting or redacting a non-Claude session is rejected, so a future loosening can't silently open a leak channel for archived-but-unredactable vendor content. The scorecard states: capture and refinery are Claude-Code-only.

### Operational-command coverage (fold into D1/D2)

The archive directory-walk tags every file with the requested kind and dedups; omitting `--kind` yields `source_metadata` (the documented footgun); `hook install` is idempotent, aborts cleanly when the user declines the confirm, and refuses a project-local double-capture (Details: `hook_commands.go:32,72`).

---

## 3. Scorecard & gate

**Two hard tiers, no averaging.** **Why no averaging:** if you average dimensions, a row of greens elsewhere can mask a single redaction leak. Each gate stands on its own.

- **SECURITY GATES (any failure BLOCKS):** D1 hook safety; D3 reflection-canary recall = 100% on the covered data set; D4 trust-boundary (zero planted literals in any artifact + placeholder-present for carried fields + no network egress); D7 cloud-line present + doctor never false-green; **D11 no secret-bearing artifact escapes its intended store**; **D13 non-Claude export rejected**.
- **INTEGRITY GATES (any failure BLOCKS):** D2 (including ref-id-collision and multi-machine merge); D5 (runs downstream of leak-freedom + the no-secret-in-golden lint); D6 (idempotency / partial / crash) **and D6a recoverability**; D6 concurrency-TOCTOU (RED if unfixed); D8 verify-detects-corruption-against-the-recorded-digest; D9 schema conformance with `additionalProperties:false`.

**Score shape — a matrix, not a 0–100 number.** Each security and integrity dimension is PASS/FAIL. The headline is a small enum, **not a single bit**, because the red-team showed that a bare "TRUSTED" overstates the scope:

- `TRUSTED` — all security and integrity gates green **and** no known-miss credential class present **and** recoverability wired.
- `TRUSTED-WITH-NAMED-GAPS` — gates green, but a known-miss class is in the data set or recovery is unwired. **This string cannot be abbreviated to TRUSTED.** The machine-readable scorecard carries a non-zero `gap_count`; emitting a bare `TRUSTED` while any known-miss or unwired-recovery flag is set is forbidden.
- `NOT-TRUSTED` — one or more red gates.

**Honest-residual block (always printed verbatim beneath the headline):** the extended-class recall % plus the enumerated covered formats (the 8 regexes); the named limitations (unicode handling, the `/`-exclusion, the 32-char floor); "redaction is a single upstream pass — the 7 artifact checks are correlated, not independent layers"; "the refinery reads the live transcript, not the blob — a Claude-deleted transcript is NOT recoverable today"; "`transcript_drift` is a heuristic"; "config schema is a missing P0 deliverable"; "artifacts currently land repo-local"; "D10 gated."

**Anti-gaming structure (red-team).** Split the recall gate into **(a)** a hard "no leak in the COVERED data set" blocker and **(b)** a tracked, **non-blocking but never-shrinking (monotonic)** extended-class recall. CI checks that the extended data set still has at least the documented N classes and that recall % **does not regress**. This lets a team honestly record a known miss as a known-miss entry without turning CI red — while making **corpus shrinkage a visible, reviewable regression** rather than an invisible omission. (That invisible-omission pressure is exactly what the architect's "just append `make trust` as a hard gate" approach would have created.)

**Replacing the shallow gate.** Verified today, the existing target is `verify: supply-chain vet lint test test-race build demo dogfood-demo security` (Makefile) — it's green, but it never runs the secret data set through normalize, never validates schemas, never compares a blob to its source, and never corrupts a backup. Add a new `make trust` (Go-native, fixture-driven, stdlib-only; via a `//go:build trust` tag or a `cmd/trustbench`) that emits machine-readable JSON plus a human summary, then make the target `verify: … security trust`. CI fails the job on any red security or integrity gate and uploads the scorecard JSON as the artifact qratum "publishes about itself." `make trust` must drive the **real CLI entrypoints** (R3), not a parallel harness that bypasses the daemon's repo-local placement and its live-transcript reads.

**What a green score honestly does and does NOT promise.** A green `TRUSTED` asserts: every planted or known secret is absent from every shareable artifact; the vault is content-addressed, immutable, dedup-clean, and atomic; identical input → byte-identical output; crash / partial / missing paths fail loudly; doctor states its cloud blind spot; emitted objects validate (with extra keys forbidden); and the hook does only capture work. It **does NOT** promise: review quality (it's heuristic, with a 0% human-engagement baseline); cloud or web session coverage (uncaptured by design); **Codex/vendor coverage** (archive-only, no redaction path); that redaction catches **all** secrets (it's best-effort alpha over an enumerated allowlist — a novel format or a new struct field can still leak); multi-machine merge beyond dedup; regulatory compliance; **and — critically — that a Claude-deleted transcript is recoverable** (the refinery still reads the live path, Details: `daemon.go:159`).

---

## 4. Phasing

**P0 — now (design, golden data sets, contracts) — buildable against shipped code; closes the critical leaks:**

- Author the planted-secret data set (D3 redaction safety): wire `transcript-with-secret.jsonl` into a full daemon-driven pipeline test; add the **reflection every-field-canary** harness (R1); add the evasion fixture; add the known-miss/xfail file plus the completeness meta-check; add the benign/precision fixture with the commit-hash/UUID tripwires.
- Build the shared **no-raw-leak property checker** (literal-absence + placeholder-presence, R2) used by D3 (redaction), D4 (trust boundary), and D11 (placement).
- Wire up the orphaned `schemas/` (D9 schema conformance): add `additionalProperties:false`, the stdlib mini-validator with its reject self-test, the emitted-object enumeration, and the emitted ⊆ schema drift guard.
- Add the **no-secret-in-committed-golden** lint (D5 determinism) and **redact the golden's SSH remote / head_sha**.
- Define the scorecard JSON shape plus a `make trust` skeleton that runs the testable-now parts (D1, D3, D4, D5, D9, D11, D12, D13).
- Define the receipt JSON Schema plus a synthetic fixture (D10 cross-repo testable-now contract check).

**Attached-to-vault-impl — buildable now (the vault is already merged):**

- D2 (vault integrity) unit/property tests: independent-source byte-equality, dedup, atomic-failure, the no-delete grep, backfill idempotency, **ref-id-collision**, **multi-machine merge**.
- D6 (idempotency/crash): idempotency / partial / crash / missing, plus **refine-source consistency** and the **concurrency-TOCTOU fix**.
- D7 (doctor) matrix: injected state, stubbed clock, always-present cloud line, **drift-as-heuristic**.
- D8 (backup): corruption-detection against the recorded digest, plus round-trip restore.
- **D6a recoverability** stays RED until the refinery is rewired to read from blobs; once it is, extend D3 (redaction) and D4 (trust boundary) to assert the blob-sourced path also leaks zero secrets.

**Gated-on-gateway (D10 second half):** receipt round-trip, idempotent re-run with `supersedes[]`, `namespace_forbidden`, unknown-`contentClass` reject, and bad-outcome reject. Needs the personal-memory gateway Phase 1/2. **Do NOT build the producer in qratum.**

**Minimum next milestone for verification work:** **"D3 redaction crown-jewel + D11 placement + D9 `additionalProperties` — as planned-RED gates."** Specifically: (1) wire in `transcript-with-secret.jsonl` and the reflection-canary harness so the recall test runs through the real daemon and **fails on the confirmed git/time/event-field leaks**; (2) fix the `redact.go` partial-redaction `=>` bug and either route or provably-drop the leaking fields; (3) add the no-secret-in-golden lint and redact the committed golden; (4) add `additionalProperties:false` to the schemas and build the rejecting mini-validator. This closes the two confirmed CRITICAL leaks (secret-through-normalize, and the raw session written to disk) using only shipped code, and it makes the rest of the matrix meaningful.

---

## 5. Open decisions for Arnold

**A. Daemon / no-daemon (passive-capture liveness).** The hook captures, but nothing schedules `run-once`/`backfill`; the 0%-engagement history is the proof point.
- *Option 1 — stay manual, name it loudly:* D12 just asserts the staleness gates, and the scorecard states "no scheduler ships; freshness depends on out-of-band cron/launchd." Cheapest; honest; preservation freshness stays a user responsibility.
- *Option 2 — ship `qrt vault install-schedule` (launchd / systemd timer):* an idempotent timer install alongside `hook install`. Closes the engagement gap; adds an OS-integration surface to verify.
- *Option 3 — capture-time refine:* refine inline in or right after the hook. **Rejected** — it violates the fast-hook rule (`AGENTS.md`).
- **Recommendation: Option 1 now, Option 2 when the vault P1 sequence is unlocked.** Don't build a scheduler under a "docs-only" milestone; do make the absence a printed scorecard residual so green never implies live capture.

**B. Insurance vs the dream.** `BACKLOG.md` pre-flight, unresolved.
- *Option 1 — insurance (preserve + import + stop):* the scorecard scopes to D1–D9 + D11–D13; D10 stays gated and the dream lane is explicitly out of scope. The smallest, most defensible attestation.
- *Option 2 — the dream (curation lane later):* keep D10's gateway scaffolding in the headline tier and pre-commit verification to the W5/gateway phases.
- **Recommendation: Option 1.** The architect's design leaned toward the dream by building a full D10 tier around gateway round-trip / grants / supersedes, while giving **zero** standalone coverage to the insurance definition-of-done (delete-source-then-recover). Gate the dream-fork dimensions behind an explicit "curation fork accepted?" flag, and make **D6a recoverability** first-class now. The foundation is identical either way, so verify the foundation, not the speculative lane.

**C. Benchmark scope — fix the redactor vs drop the fields.** The confirmed git/time/event-field leaks must resolve one way:
- *Option 1 — extend `redact.go`:* route `git.branch` / `git.head_sha` / `git.remote` / `source_event_*` / `started_at` / `ended_at` / tool source ids through `redactString`. Risk: over-redacting timestamps and commit hashes (precision).
- *Option 2 — provably drop them from every shareable artifact:* keep them only in local-only stores, and assert their absence everywhere downstream.
- *Option 3 — hybrid:* drop them from the report / DTO / ADP (they add little there), and redact them in the redacted-session JSON.
- **Recommendation: Option 3**, plus fix the `=>` partial-redaction bug regardless. Also decide explicitly that **the `.normalized.json` raw store is in-scope as a documented local-only raw-retention store** (remove it from the "no raw in any artifact" headline; assert it never escapes to ADP / report / network) — rather than silently skipping it from the no-leak loop.

**D. Concurrency race (TOCTOU) — fix or accept.** *Option 1:* `O_EXCL` create + a goroutine race test (recommended — it's a real data-loss path under the parallel-agent factory). *Option 2:* mark it RED known-data-loss. **Do not** mark it informational "stated gap."

---

## 6. What NOT to build yet (non-goals, so the benchmark doesn't become its own scope monster)

- **No full JSON-Schema library.** Stdlib mini-validator only (supply-chain rule). A vetted, ≥7-day-old pure-Go validator only with explicit sign-off.
- **No new runtime features just to satisfy a gate**, beyond the minimal fixes the gates demand (redactor field coverage, the `=>` bug, golden redaction, `additionalProperties`, `O_EXCL`). The benchmark may NOT smuggle in the refinery-reads-from-blob rewrite, the central-workspace migration, a scheduler, config-schema implementation, retention/delete verbs, or SQLite — those stay spec'd-only; the benchmark **reports** them as RED/residual, it doesn't build them.
- **No producer script** (`import-claude-memories.ts`) in qratum — it lives in personal-memory and is parked.
- **No gateway-dependent D10 tests** until Phase 1 deploys. Build only the synthetic receipt-archive contract check (and don't score it as a leak proof).
- **No corpus of real secrets or real transcripts** committed — synthetic/canary only; never commit real `~/.claude` content (and the secret fixtures must contain only fabricated credentials).
- **No multi-machine merge runtime** — verify the dedup-union property with synthetic vaults only; the per-machine install runbook stays P2.
- **No review-quality / agent-judgment metric.** The benchmark attests process and safety invariants, not whether the agent did good work (Honest Boundary).
- **No tunable precision budget that absorbs known over-redaction** — fix the regex instead.

---

Verification status of this audit: I read the source-of-truth specs, ADR 0010, BACKLOG, and the shipped `redact.go`, `vault.go`, `workspace.go`, `hook.go`, `daemon.go`, `export.go`, `report.go`, `ui.go`, plus the schemas, fixtures, and Makefile. I **built `qrt` and empirically confirmed**: (1) the evasion-corpus leaks and the `=>` partial-redaction bug; (2) `started_at` / `ended_at` / `source_event_id` / `git.*` leak verbatim through `redact`; (3) the daemon writes a raw, un-redacted `.normalized.json` containing `sk-ant-api03-…` + `supersecret` to **repo-local** `./.qratum/`; (4) in the real pipeline the enumerated classes (api-key / JWT / db-URL / paths) DO redact into the redacted/report/ADP outputs. Confirmed by static inspection: the `schemas/` have zero Go references and no `additionalProperties`; `transcript-with-secret.jsonl` is unused by tests; `RawRefIDForDigest` truncates to 12 hex versus the full-digest blob path (collision risk); `parseArchiveArgs` defaults the kind to `source_metadata`; and no `make trust` target exists.

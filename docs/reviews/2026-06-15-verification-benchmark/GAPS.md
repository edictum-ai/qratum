
# Qratum Spec Set — Adversarial Completeness Gap Review

**Date:** 2026-06-15 · **Scope:** `verification-and-trust-gate.md` (primary), `qratum-vault-first.md`, `operational-model-redesign.md`, `BENCHMARK.md`, governance docs. Ground-truth verified against shipped `cmd/qrt` + `internal`.

**What this review did, in plain terms:** Six review lenses plus a completeness critic looked for holes in the spec set and turned up about 45 raw candidates. We then removed duplicates, dropped anything the spec already handles, and checked every load-bearing claim against the actual shipped code. The survivors are ranked below. Every "Must-address" item was confirmed by reading the binary's source, not guessed at.

---

## Plain-language glossary

Read this once; the body then uses the plain word where it can.

- **Trust gate / scorecard:** the automated check that decides whether qratum is "TRUSTED." It runs a set of numbered dimensions (D1, D3, …) and goes green or red.
- **Dimension (Dn):** one named check in the gate, e.g. the redaction-safety check (D3). The letter-number is just an ID for traceability.
- **Redaction:** automatically removing secrets (API keys, passwords) from a copy of the data before it leaves the safe zone.
- **Redactor / redaction pipeline:** the code that does that removal (`redact.go`).
- **Canary token:** a unique fake secret planted into the data on purpose. If it later shows up in an artifact, you know that path leaks. If it's gone, the path is (supposedly) safe.
- **Reflection canary / reflection walk:** the technique of using Go reflection to walk every field of a struct at runtime and plant a canary in each one.
- **Planted-secret corpus:** a test data set seeded with known fake secrets, used to measure how many the redactor catches.
- **Recall:** of the secrets that were present, the fraction the redactor actually caught. 100% recall = caught them all.
- **Precision:** of the things the redactor removed, the fraction that genuinely were secrets. Low precision = it also ate harmless text ("over-redaction").
- **Precision tripwire / precision corpus:** test data of look-alike-but-harmless strings that must *survive* redaction; if they get eaten, precision is too low and it's a regex bug to fix.
- **Entropy detector:** a heuristic in the redactor that redacts any long, random-looking string on the theory that it's probably a secret.
- **Blob:** one archived file in the vault. Here, each blob is a full raw transcript stored content-addressed (named by its hash) and never edited in place.
- **Raw vs redacted:** "raw" = the original transcript with secrets intact; "redacted" = a derived copy with secrets stripped.
- **ADP:** the external export object qratum produces for outside consumers. Crossing into the ADP is the main "leaves the safe zone" boundary.
- **Allowlist vs denylist:** an allowlist builds output from a named, approved set of fields (safe by default — unknown fields are excluded). A denylist removes a named set of bad fields and lets everything else through (unsafe by default — unknown fields leak). "Fail-open" = unknowns leak; "fail-closed" = unknowns are blocked.
- **`data_class`:** a label saying how sensitive an object is (e.g. raw vs redacted vs publishable). For the rule "never silently upgrade sensitivity" to mean anything, this label has to actually live on the object.
- **Monotonic:** only moves one direction. A monotonic lattice of data classes means you can downgrade sensitivity only through a named step, never silently raise it.
- **Idempotent:** safe to run repeatedly; running it again changes nothing new.
- **TOCTOU (time-of-check-to-time-of-use):** a race where something changes between when you check it and when you use it.
- **umask:** the OS setting that decides default permissions on newly created files/dirs. `0o644` = world-readable; `0o600` = owner-only; `0o700` = owner-only directory.
- **At-rest:** data sitting on disk (as opposed to in transit).
- **PII:** personally identifiable information — names, emails, customer content, etc.
- **GC:** garbage collection — reclaiming storage no longer referenced.
- **RSS:** resident memory a process is using.

---

## Must-address before building P2

These are real holes. Left unfixed, each one either ships a security or data risk, or makes the trust gate show green while a leak is still live.

### M1 — The canary token can be eaten by the redactor itself, so a real leak and a real pass look identical

**Plain problem:** The unique fake secret we plant to detect leaks (the canary) happens to look like a real secret to the redactor. So the redactor deletes it. That means the check passes whether the data was handled correctly *or* the redactor just swallowed the evidence. A true leak and a true pass produce the exact same observable result. On top of that, the same string is also used as a "must survive" tripwire elsewhere — so one token is told both "you must disappear" and "you must remain."

**Confirmed in code.** The spec calls for a UUID-v4 canary (Details: `BENCHMARK.md` §D3). But the entropy detector's pattern includes hyphens in its character class, and the "looks high entropy" guard excludes only `/` and `\`, never `-`. A 36-character hyphenated UUID-v4 therefore matches and gets redacted by the entropy detector (Details: `redact.go:23` `highEntropyPattern = \b[A-Za-z0-9+/=_-]{32,}\b`; `looksHighEntropy` at `redact.go:452`). So in the redaction-safety check (D3), "canary absent from artifact" is satisfied either because the field was correctly handled *or* because the redactor merely ate the token. Worse, that *same* hyphenated-UUID string is also seeded into D3's precision corpus, where it is required to **survive** (over-redaction there is treated as a regex bug to fix). One token class is simultaneously "must vanish" (canary) and "must survive" (tripwire).

**Why it matters:** The redaction-safety check (D3) is the single load-bearing security gate. A self-redacting canary makes the gate read "100% recall, always green" for any field whose path drops or redacts the token — which is exactly the false-green the spec opens by condemning.

**Spec addition:** Require a canary format that, by construction, cannot be caught by **any** of the eight redaction classes — for example lowercase letters only, under 32 characters, a single character class, no separators: `qratumcanaryNNNN`. Also add a harness self-test that feeds the chosen canary alone through `redactString` and asserts it comes back **unchanged**. Keep the hyphenated UUID strictly as the precision tripwire. (Dedup note: this is the sharpest framing of the "harness untested" and "UUID self-redaction" findings.)

### M2 — The leak-detection harness has no self-test, so a buggy planter would still report 100% recall

**Plain problem:** The same reflection walk that plants canaries is also what decides which fields got scanned. If that walk silently skips a field, it plants nothing there *and* counts nothing there — so the miss is invisible on both sides of the ratio and the gate still reads perfect.

**Detail.** The field-coverage contract is "the set of fields we scanned equals the set of reflected string fields." Because one walk does both jobs, a field it skips (unexported, a nil pointer, a `map[string]any` leaf, or a field added later) gets no canary and no scan, so the top and bottom of the recall fraction are under-counted identically. The spec already mandates a self-test for the schema mini-validator ("must REJECT an injected extra key") but mandates **none** for the canary harness.

**Spec addition:** Add a mandatory harness self-test to the redaction-safety check (D3) with three parts: (a) a known-positive — deliberately route one field around `redactString` and assert the gate goes RED (this proves the gate *can* fail); (b) a coverage cross-check — derive the field set from an independent source (AST / `go/types`, or a hand-maintained golden field list) and assert it equals the reflection-derived set; (c) make the planter **panic loudly** on any field kind it cannot handle, instead of silently skipping it.

### M3 — Map keys and non-string values can never be reached by the redactor or the canary

**Plain problem:** The redactor cleans the *values* inside maps but copies the *keys* through untouched, and it leaves non-string values (numbers, etc.) alone. So a secret used as a map key, or stored as a number, is never cleaned. And because the canary is planted via struct fields, it can't even be placed as a dynamic map key — so this whole leak class is invisible to the gate by design.

**Confirmed in code.** `redactAny` recurses into map values but copies keys verbatim, and its `default:` branch returns non-string scalars unchanged (Details: `redact.go:264-283`, `out[key] = redactAny(..., v[key])`). The fields `tool_calls[].Input` and `Provenance` are `map[string]any` — the highest-value, most attacker-controlled surface. A secret used as a JSON object key (for example an env-dump tool emitting `{"sk-ant-...": "value"}`), or surfaced as a non-string number, is never redacted. And a struct-reflection canary cannot be planted as a dynamic map key, so the redaction-safety check (D3) cannot see this class at all.

**Spec addition:** Extend `redactAny` to run map keys back through `redactString`, and to scan or coerce non-string values (or prove them inert with a positive test). Add a **hand-authored** D3 fixture that plants tokens as map keys and as stringified numbers inside `tool_calls[].input`, driven through the real daemon. Add a D3 sub-assertion: "map keys and non-string scalars in input/provenance are scanned or provably inert."

### M4 — Raw secret files are world-readable, and the vault root inherits a loose umask — there's no at-rest isolation check at all

**Plain problem:** Every archived blob is a full, un-redacted transcript — real `sk-ant` keys, passwords, the whole conversation — and it's written with permissions that let any other user on the machine read it. The vault's top directory is never created with tight permissions either, so it inherits whatever the OS default is (often world-traversable). No trust dimension checks file permissions, so CI can be fully green while every preserved secret is group- or world-readable — bypassing the entire redaction pipeline the gate fixates on.

**Confirmed in code.** Blobs are chmod'd `0o644` (Details: `vault.go:207`, `tmp.Chmod(0o644)`); refs and state are `0o644` (`vault.go:278`/`:315`); dirs are `0o750`. The workspace `Resolve()` does **not** `MkdirAll` the root at all — it only `Abs`/`Clean`s it (Details: `workspace.go` lines 36-40), and subdirs are created lazily at `0o750`, so the root inherits the process umask (commonly `0o755`, world-traversable). The spec set's only stated permission is `0600` for `app_auth.json`.

**Spec addition:** Add a new **data-at-rest / local-isolation dimension (D14)**. It must assert that the `~/.qratum` root and every `raw/` subtree are `0o700`, and that every blob/ref/event/state file is `0o600`. Have `Resolve()` (or a setup step) explicitly `MkdirAll(root, 0o700)` and chmod existing roots. Change blob/ref/state writes from `0o644` to `0o600`, and dirs from `0o750` to `0o700`. Add a negative test: flip a file to `0o644` and confirm the gate goes RED. Name "multi-user shared machine" as an explicit threat, and state plainly whether at-rest disk encryption is in or out of scope.

### M5 — `vault backup` ships the entire raw vault anywhere, with no consent step and no redaction

**Plain problem:** The backup command copies the *whole* vault — including the raw, un-redacted transcripts — to any destination, and the docs recommend pointing it at cloud buckets. There's no consent prompt and no redaction. This is the single biggest way raw secrets can leave the machine, and it directly contradicts the project's own rule that raw data never leaves the machine without recorded consent.

**Confirmed in code.** `Backup` trims the destination, refuses only when `dest==home`, then `copyTree`s the entire `~/.qratum` — including `raw/blobs/**` of un-redacted transcripts — to any filesystem path (Details: `vault.go:400-413`, refusal at `:410`). The vault docs advertise this as an rclone/restic wrapper, meaning cloud/network buckets are the *intended* use (Details: `vault-first.md:155`). Yet the operational model's own Trust Boundary rule says "Raw to external… requires recorded consent" and "Raw archive: Local-only. Never exported by default." The existing D8 dimension tests only restorability and corruption on a *local* directory; nothing checks whether shipping raw blobs off-machine is policy-gated.

**Spec addition:** Treat backup-of-raw as a trust-boundary crossing inside D8. Require an explicit consent/audit event before any backup whose source includes `raw/`. Warn loudly — or require an `--i-understand-raw-leaves-machine` acknowledgement — when the destination is non-local. Add a dimension asserting that backup emits the consent audit and refuses raw egress without it. State this in the residual block: "backup egress of raw blobs is the sanctioned exception to 'raw never leaves the machine'; until it is consent-gated it is local-copy-verified only." Consider adding a redacted-only backup mode.

### M6 — The `evidence`/`report`/`review` commands don't re-redact their input and are never benchmarked

**Plain problem:** If you run `evidence`, `report`, or `review` directly on a raw session file, they build their output without removing secrets first. Only `export` redacts. So a single command can emit a bundle with full `sk-ant` keys in it — and the gate never exercises these commands, so the "TRUSTED" headline gets earned while this one-command leak path stays untested.

**Confirmed in code.** The R3 enumeration covers only `runDaemonOnce`/`runWithIO`. But `report.go:57` and `evidence.go:144` call `readQratumSessionFile` and then build artifacts with **no** `redactQratumSession` call — only `export.go:128` redacts. So `qrt evidence ./.qratum/sessions/foo.normalized.json` (the raw, repo-local file from FIX-4, full of `sk-ant` keys) emits an evidence bundle with raw secrets in its `Summary` and `OutputExcerpt`. These are shipped, documented CLI verbs that a user or a factory agent can invoke directly; the gate drives only the daemon.

**Spec addition:** Extend R3 to enumerate **all** artifact-producing subcommands and run the no-leak checker over each one driven directly. Decide and assert: the standalone `evidence`/`review`/`report` commands must either reject a non-redacted session (require `pipeline_status==redacted`) or redact internally. Add a fixture that feeds a raw, secret-bearing session to each standalone command and asserts zero canary survival.

### M7 — There's no privacy model: "sensitive" means "credential," but transcripts are verbatim human conversations full of other people's content

**Plain problem:** The redactor only recognizes credential formats. But a transcript is a faithful record of real work — prose the operator typed, content pasted from customers or teammates, code others wrote, names, emails, PII in test data. None of that matches a credential pattern, so it all flows through verbatim into the outputs. The canary can't catch it either (no PII canary is planted), and the consent model only ever names the local user — it has no concept that some captured content belongs to a third party who never agreed to have it preserved forever, stored world-readable, or backed up to the cloud.

**Confirmed by absence.** The redactor matches eight credential formats only. Pasted third-party content flows verbatim into evidence/review/report/ADP. The reflection canary cannot catch it because no PII canary is planted. The consent model only names `local_user` as approver (Details: `operational-model` §Consent) — there is **no concept** of third-party-owned content. The "does NOT promise" list omits PII entirely.

**Why it matters:** A green TRUSTED scorecard could sit on top of a vault full of other people's PII, in an immutable, world-readable, cloud-backupable store, with no way to erase it. For any future team or shared use, or any privacy regime such as GDPR's right to erasure, *this* is the real liability — not the credential case.

**Spec addition:** Add a "Data subjects and scope of captured content" section stating plainly: (a) qratum captures verbatim conversation that may contain third-party PII and others' authored content; (b) the redactor targets **credentials only** and makes **no PII/ownership guarantee** — add this to the "does NOT promise" list; (c) the v1 stance is single-developer / own-machine / own-data, with a printed warning that pasted third-party content is preserved un-redacted, and team/shared use documented as out-of-scope. Note the interaction with backup egress (M5) and with immutability (there is no erasure path) as a named residual.

### M8 — The "no silent sensitivity upgrade" rule can't be enforced, because `data_class` is never stored on anything

**Plain problem:** The headline rule says no boundary may silently move data to a more sensitive class. But the sensitivity label (`data_class`) is never actually a field on any saved object — it lives only in a prose table and a UI-badge idea. With no label on the object, there's nothing to compare at any boundary, so the rule has nothing to check.

**Confirmed in code.** Grepping for `data_class`/`DataClass` across `cmd/`, `internal/`, and `schemas/` returns **zero hits**. The invariant is asserted in a prose table and a UI-badge concept, not as a persisted field. The existing D4 dimension tests literal secret-absence; D9 validates keys; neither requires data-class lineage as a field. Future code that routes a "redacted" artifact to a "publishable" path would pass every test, because the class was never on the object.

**Spec addition:** Make `data_class` a required field on every emitted object schema (raw_ref, session, review_card, evidence, ADP wrapper, UI DTO), with an enum and a committed monotonic lattice (sensitivity can only be lowered through a named step, never raised silently). Extend D4 to assert: every emitted object declares a class; no transform raises the class except through a named downgrade; every export boundary refuses a class above its allowlist. Without the carried field, D4 can be no more than a literal text scan.

### M9 — The ADP key-stripping is a fail-open denylist, not the allowlist the gate assumes

**Plain problem:** When building the external export (ADP), the code removes a fixed list of known-internal keys and lets everything else through. So any internal key *not* on that list — including ones added later, or buried in a nested map — sails straight into the external output. A "remove the bad ones" approach on the one external boundary is exactly the insecure default this product exists to prevent; it should be "keep only the approved ones."

**Confirmed in code.** `isQratumOnlyExportKey` denylists exactly `secret_map, secret_maps, provenance, redaction, artifact_paths, pipeline_status` plus the `x-qratum-` prefix (Details: `export.go:373-379`). Any internal key **not** in that list (for example `source_transcript_session_id`, `repo_id`, a field added later, or an annotation inside a nested tool-input map) passes straight into the external ADP. The D4 test covers only the six already-known keys.

**Spec addition:** Specify that the ADP builder (and any external export builder) is an **allowlist projection** — it constructs output from named fields only and never passes an arbitrary internal map through. Add a D4 test that injects a **random unknown** internal key into a nested input map and asserts it is absent from the ADP (which proves allowlist behavior, not denylist).

### M10 — Nothing actually makes refine/backfill run, so "survives abandonment" is structurally false

**Plain problem:** The original failure was zero engagement — 18 sessions captured, 0 processed. The "preserve first, refine on demand, fine to stay unused" answer doesn't change that; it just makes the unprocessed pile durable. Even capture isn't automatic: backfill is manual, so a user who installs the hook and walks away only gets hook-captured sessions. Anything in the cloud, on another machine, or that Claude deletes before a manual backfill is simply lost. So the accepted acceptance criterion "survives abandonment gracefully" is false under actual abandonment.

**Detail.** The D12 dimension honestly *names* this and then asserts a staleness warning fires — which is yet another thing nobody runs. Critically, **backfill itself is manual** (Details: `vault-first.md:80` lists "Survives abandonment gracefully" as an accepted acceptance criterion).

**Spec addition:** Force the daemon / no-daemon decision to closure for this milestone (see Q1). At minimum, spec `qrt vault install-schedule` — an idempotent (runs repeatedly without re-doing work) launchd/systemd timer — so **preservation freshness** is not a manual responsibility even if refine stays on-demand. If it stays fully manual, the scorecard must print a **CRITICAL residual**: "preservation freshness depends on an out-of-band scheduler this product does not install; on abandonment the vault decays." Do not let "survives abandonment" stand as a green acceptance bullet while it is false.

### M11 — CI paradox: this milestone adds gates that are RED by design, so it can't pass its own CI

**Plain problem:** The spec wires the trust gate into required CI (`make trust` into `make verify`; CI fails on any red security/integrity gate) — but at the same time it declares several gates "planned-RED until the code is fixed," and one of those fixes is sequenced to land *after* this milestone. Both can't be true at once: a gate that's red on purpose, wired into required CI, locks the whole repo (including unrelated work) until a big rewrite lands. The only escapes are "disable the gate" or "merge-lock everything," and both are forbidden.

**Detail.** The planned-RED gates are D6a, D9-drift, and D3 field-coverage; FIX-3/D6a is explicitly sequenced *after* this milestone ("Then"). The §3 anti-gaming split (hard-blocker vs monotonic) covers only extended-class recall, not D6a/D9/D3-field-coverage. "Disable the gate" defeats the purpose; "merge-lock" is forbidden by "CI is sacred."

**Spec addition:** Define a third gate state explicitly: **KNOWN-RED** — tracked, CI-non-blocking, and monotonic (the count can only go down, never up; each entry needs a tracking note, an owner, and a deadline). This is distinct from **BLOCKING-RED** (which fails CI). Map D6a, D9-drift, and the not-yet-fixed leak classes to KNOWN-RED until their fix lands in this milestone's sequence. CI then fails only on a regression of a currently-green gate, or if a KNOWN-RED item outlives its deadline. State this so that "wire trust into CI" does not silently merge-lock the repo or invite weakening the checks.

---

## Should-address (strengthens the spec)

### Theme: scale and lifecycle the gate never exercises

- **Backup/verify load whole files into memory.** They read the entire file at once, so at the GB scale this product targets they can run out of memory; the existing D8 case only flips one byte in a small blob (Details: `vault.go:516`/`:569` `os.ReadFile`; `verifyTree` reads both source and dest fully). *Add:* mandate streaming `io.Copy` with a bounded buffer; add a D8 case with a 500MB synthetic blob that asserts bounded RSS; verify against `ref.Digest` (already noted) **and** stream.
- **No disk-full guard when copying on capture.** A forever-growing store on a finite disk runs out eventually; the unbounded copy then fails silently every session (`copy_status=failed`, exit 0). There's even a config knob nothing reads (Details: `ArchiveFile`'s unbounded `io.Copy` at `vault.go:202`, no `LimitReader`; config has `disk_free_min_gb` that nothing reads). *Add:* a disk-free preflight that degrades loudly plus a doctor escalation threshold; wire `disk_free_min_gb` in, or remove it.
- **No garbage collection, retention, or safe-shrink path.** It's append-only forever with no reaper for orphaned blobs, so the only way to shrink is the unsafe manual `rm -rf ~/.qratum`, which destroys the irreplaceable data the tool exists to keep. *Add:* at minimum print a CRITICAL residual "unbounded growth, no GC, no safe shrink"; spec a tombstone-respecting `qrt vault gc` that refuses to delete referenced blobs.
- **Orphaned `.tmp` blobs pile up after a crash.** Cleanup only happens via a deferred remove, so a crash leaves stragglers; there's no startup sweep, and `copyTree` even backs them up (Details: `vault.go:189` `CreateTemp`). *Add:* sweep stale `.tmp` files at backfill/doctor start and exclude them from backup.
- **Doctor/backfill cost grows with the number of refs, over a flat directory.** `ListRawRefs` globs and unmarshals every ref; blobs are sharded but refs are not, so doctor — the very lever meant to drive engagement — degrades exactly where the vault is most valuable (Details: `ListRawRefs`). *Add:* shard `refs/` or maintain a derived index; short-circuit backfill on unchanged mtime/size; add a steady-state perf case (10k refs → doctor under threshold).

### Theme: concurrency beyond the one fixed TOCTOU

- **`UpdateState` is a non-atomic read-modify-write.** It loads state, mutates it, then saves; the rename at the end is atomic but the read-modify-write around it is not, so concurrent hooks lose counter increments (Details: `vault.go:322-328`, LoadState→mutate→SaveState). Under the factory, doctor's failure counters — a D7 security-tier gate — silently undercount, producing a false-green about an irreversible store through a path the existing FIX-6 doesn't cover. *Add:* serialize state mutation (a flock / `O_EXCL` lockfile) or derive counters from append-only events; add a goroutine-hammering test asserting the final counter equals the injected failure count.
- **Parallel pipelines aren't addressed at all.** Per-file atomicity is not the same as per-pipeline consistency, and parallel refine, backfill-while-hook, and backup-while-archive are unhandled. *Add:* a D6 concurrency sub-matrix (two run-once on the same session; backfill+hook on the same digest; backup during capture), each run with `-race`.

### Theme: schema / contract completeness

- **The session schema describes a different object than the code emits.** About 12 emitted fields are undeclared (`transcript_path`, `source_event_id`, `git`, `pipeline_status`, `artifact_paths`, `business_metrics`, `provenance`, …), so D9's `additionalProperties:false` is **unsatisfiable** until the schema is brought to full field parity — and the spec never states that completing the schema is the actual deliverable (Details: `qratum-session.v1.schema.json`; a `qratum-provenance.v1.schema.json` exists but the session schema doesn't reference it). *Add:* make D9's deliverable explicit — enumerate every emitted field with its type, give provenance a typed sub-schema, and add a self-test asserting the struct-tag set equals the schema-property set.
- **`additionalProperties:false` has no teeth where the leaks actually live.** Nested containers are declared as bare `{"type":"array"}` / `{"type":"object"}` with no `items`/`properties`, so a leaking key inside `turns[]`, `tool_calls[].input`, `raw`, or `workspace` passes — and that nesting is exactly where the audited leaks live. *Add:* require recursive property/items schemas (each with its own `additionalProperties:false`) before D9 counts; re-evaluate whether the hand-rolled stdlib mini-validator (now needing `const`/`enum`/`required`/recursive `properties`/`items`) is still the honest call versus a vetted, at-least-7-days-old pure-Go validator.
- **No schema-migration story for a forever store.** The mandated `migration_version` / `producer_version` / `qrt migrate` are entirely absent from the code. Content-addressed blobs are immutable, so a v1 blob can never be re-normalized in place. And `LoadState` silently back-fills `SchemaVersion` when it's blank, which masks drift. *Add:* a reader that does **not** silently default a missing or unknown version; a D9 case feeding a v1 fixture and asserting defined behavior; define the v1→v2 derivation shape (regenerate from the blob) before any v2 field exists.
- **`memory_import_receipt` is a shipped, accepted raw_ref kind with no schema.** `qrt vault archive <file> --kind memory_import_receipt` writes a permanent immutable blob against an undefined contract *today*, and `--kind` defaults to `source_metadata` (a mislabel footgun). *Add:* gate the kind behind "a committed receipt schema exists," or require the receipt JSON Schema (a closed outcome/errorClass enum mirroring the gateway's real vocabulary) as P0, wired into D9, before the kind can be archived.
- **The ADP boundary version isn't frozen.** It pins `adpStrictSchemaVersion = "1.1.0"` with no committed ADP schema, diverging from the `qratum.X.v1` convention; D5 only pins byte-determinism, which guarantees "the same wrong shape every time." *Add:* commit an ADP JSON Schema with `additionalProperties:false`, wire it into D9, and state the targeted ADP spec version plus a compatibility policy.

### Theme: scan correctness and re-introduction

- **Downstream artifacts re-introduce content the redactor never re-scanned.** `evidence.go` builds `Summary` via `fmt.Sprintf("%q", …)` and copies `started_at`/`ended_at`/`source_event_id` straight from the session struct, not from the redacted JSON text (Details: `evidence.go:445-447`). So FIX-2's "redact in the redacted-session JSON" is not enough — a canary planted in `started_at` can reappear in `evidence.summary.started_at`. *Add:* make the terminal per-artifact byte-scan a **hard** gate (not optional defense-in-depth) on every artifact; enumerate every field downstream artifacts copy from the struct and route each one through `redactString` at the read point, or drop it; add a canary case where the only planted token is in a re-introduced field.
- **The scan is blind to encoding.** The shared literal-absence checker scans raw bytes, but report output is HTML (entity-escaped) and ADP/DTO are JSON (escaped), and the escape-safe UUID canary conveniently never triggers this. A secret containing `< > & " + / =` could survive HTML-escaped while the scan still reports clean. *Add:* make the checker encoding-aware per artifact (scan HTML-unescaped text; scan JSON-unmarshaled string values); add fixtures with escape-triggering characters and a deliberate escaped-only negative case.
- **The "independent terminal byte-scan" (D4) is not actually independent for unknown secrets.** It re-scans the *same canary set*, so it inherits the redactor's recall exactly — it can't catch anything the redactor would also miss. *Add:* give it a genuinely independent generic high-recall detector (broad `AKIA`/`AIza`/`glpat`/`pk_`/`SG.` prefixes plus entropy) distinct from the redactor's patterns — **or** drop the "independent layer" framing and label it a regression tripwire in the residual block.
- **`redactAny` can panic on a bad shape, which is a fail-open risk.** It uses unchecked type assertions, so a malformed or non-map `Input` or `Provenance` (a future schema change, a hand-supplied file) panics the daemon — and the raw `.normalized.json` is already on disk while the redacted artifact never gets written (Details: `redact.go:226`/`:249`, `.(map[string]any)`). *Add:* use the comma-ok form and return a typed error (fail-closed and loud); add a D3/D6 malformed-shape fixture asserting graceful rejection; validate the `Provenance` shape.

### Theme: untrusted-input hardening

- **The hook archives any `transcript_path` verbatim, and follows symlinks.** It returns absolute paths as-is, opens them in a way that follows symlinks, and copies unbounded (Details: `hook.go:200-206`; `ArchiveFile` `os.Open` follows symlinks; unbounded `io.Copy`, no `LimitReader`). So a symlinked or attacker-chosen path pulls an arbitrary file (`~/.ssh/id_rsa`, a device) permanently into the immutable, never-deletable vault. *Add to D1:* confine `transcript_path` to an allowlist of roots (`~/.claude/projects`, the resolved cwd subtree); use `Lstat`/`O_NOFOLLOW` to reject symlinks; reject non-regular files; cap archived size with `LimitReader`; add hostile-payload fixtures (symlink-to-secret, traversal, device).
- **Committed fixtures bake a real internal URL into permanent git history.** `git@github.com:edictum-ai/qratum.git` has been in history since commit 8639cda. The D5 lint scans only the working tree, so re-redacting the golden file does **not** remove the leak from history — the gate reads green while a fresh clone still exposes it. *Add:* make the lint scan history (or require a relocation/rewrite, documenting the residual if declined); specify a corpus-secrecy contract (canary tokens that structurally can't match real formats; no real internal identifier in any committed fixture) as a standing pre-commit lint.

### Theme: missing thresholds and specifics

- **Several acceptance criteria are untestable because the numbers are unnamed.** D1's speed-gate "under threshold," the precision budget K, and the D7/D12 staleness windows are all unspecified. *Add:* pin each, or specify the methodology (speed gate as throughput or relative; K=0 flatly; staleness windows that assert both sides of the boundary).
- **Three uncoordinated size limits could slice a secret in half.** The >10MiB normalize line cap, the 1MiB hook cap, and the 50MB speed gate don't agree, so a secret on a >10MiB line could be truncated mid-value (a partial-secret leak, like FIX-1), and neither corpus exercises it. *Add:* a boundary case asserting the blob stores the full bytes, the normalizer never emits a partial secret, and any blob/artifact divergence at the cap is a named residual.
- **The capture import-isolation work conflicts with the "no new runtime features" non-goal.** Extracting `internal/capture` (D1) sits in tension with the §6 non-goal, and the coarse package-main gate may be impossible to pass if any report/UI surface imports `net`. *Add:* designate the extraction an explicit in-scope required deliverable, carved out of the non-goal; confirm via `go list -deps` whether the binary already imports `net`.

---

## Open questions for the maintainer (product/design decisions)

### Q1 — Passive-capture liveness: does this milestone ship a scheduler?

**Plain framing:** This is the original zero-engagement failure, and it decides whether anyone gets value at all. The spec parks it as "carried, not blocking," but it determines whether the whole verification edifice is built around a tool nobody runs.

- **Option 1 — stay manual, name it loudly** (the spec's current recommendation). Cheapest and honest, but "survives abandonment" is then *false* and must print as a CRITICAL residual.
- **Option 2 — ship `qrt vault install-schedule`** (an idempotent launchd/systemd timer for *backfill* freshness; refine stays on-demand). Closes the preservation-decay gap, but adds an OS-integration surface to verify.
- **Recommendation:** **Option 2, for backfill freshness specifically.** Preservation freshness being a manual responsibility is the gap that makes "a transcript Claude deletes tomorrow is recoverable" untrue on abandonment — and that recoverability is the product's core promise, not a nice-to-have. Keep refine on-demand. Pull this into the milestone, not after.

### Q2 — Artifact placement (FIX-4): move under `QRATUM_HOME` or keep repo-local?

**Plain framing:** This is **not** a deferrable fork. D11's acceptance ("never lands in a git-tracked path **unless** explicitly designated local-only") is **untestable until this is decided**, because the two branches demand different tests, and `.gitignore` can only be asserted for a test workspace — never for every downstream user's repo.

- **Option A — move all derived artifacts under `QRATUM_HOME`.** Eliminates both the world-readable-raw-secrets-in-a-project-tree surface and the per-repo `.gitignore` dependency entirely.
- **Option B — keep them repo-local, designated local-only.** Requires `0o600` perms, a `.gitignore` write into the *target* repo (not qratum's), and a cleanup/retention policy — none of which the spec currently mandates, and the per-user-repo ignore can't be guaranteed.
- **Recommendation:** **Option A.** It is the only branch that closes the filesystem-secrecy surface, not just the git-commit surface. This decision **blocks** D11 — make it before the milestone starts, not "carried."

### Q3 — Insurance vs. the dream fork

- **Option 1 — insurance** (preserve + import + stop): the scorecard scopes to D1-D9 / D11-D13; the D10 / receipt / gateway tier stays gated; this is the smallest defensible attestation.
- **Option 2 — dream** (a curation lane): keep the D10 scaffolding in the headline tier.
- **Recommendation:** **Option 1, treated as blocking for this milestone, not carried.** The asymmetry the spec under-weights: building dream scaffolding (the receipt schema "now," D10) under an insurance reality is wasted complexity in a tool whose whole virtue is being small; whereas declaring insurance now and later wanting the dream is cheap to reopen. If insurance: **drop** "define the receipt schema now" from §4 and move `memory_import_receipt` / D10 fully behind the dream flag. Also resolve the §5-vs-header contradiction on scope-decision C (it reads HYBRID in the header but still "open" in §5).

### Q4 — What is the scorecard's product surface?

**Plain framing:** The spec says "qratum publishes about itself" but never defines the consumer, the surface, or versioning — and the scorecard is itself an un-schema'd, un-leak-scanned emitted object (it isn't in D9's enumeration).

- **Options:** (a) CI gate only (internal); (b) a `qrt trust` user command; (c) a public badge on qratum.dev (which would need signing/freshness); (d) embedded in report/ADP provenance.
- **Recommendation:** At minimum, a `qrt trust` command, a `qratum.trust_scorecard.v1` schema wired into D9, and a provenance block (build commit, corpus digest, schema digest, timestamp) so a green score is verifiable as "this score, from this code, over this corpus." Run the no-leak checker over the scorecard's own bytes. Defer the public badge.

---

## Explicitly out of scope (named, not forgotten)

- **Audit-record tamper-evidence (hash chain / signatures).** Real — capture events and refs are plain mutable `0o644` JSON, and immutability today is "by absence of a delete verb," not enforced — but correctly deferred for an alpha against a benign-local-user model. *Must be named in the residual block as a deliberate OUT-OF-SCOPE*, with the threat model stating "a green scorecard assumes a non-compromised `qrt` binary and a benign local environment."
- **Multi-machine state/event merge.** Divergent `state.json` cursors and event-ID collisions across machines; "blob-dedup-clean" is true for blobs but false for state/events. Defer the runtime, but note that D2's "UNVERIFIED" label *understates* this as merely unproven rather than actively lossy — upgrade the residual to "cross-vault merge silently drops per-machine cursors/events; only blobs are dedup-clean."
- **`qrt hook uninstall` / `qrt vault export` (data egress-out).** A real lock-in / "roach motel" gap, but a v1.x lifecycle item, not a trust-gate blocker. Name it as a known residual so "recoverable" isn't quietly hollow (recovery currently requires spelunking the content-addressed store by hand).
- **Per-object erasure verb.** Coupled to the privacy gap (M7) — without it the vault cannot honor a third-party deletion request — but the tombstone / reference-counting contract is correctly P0 *contract* work, not implementation, for this milestone. State in the residual: "no per-object erasure ships; the vault cannot honor a third-party deletion request in v1."
- **`raw_ref` `original_path`/`archived_path` carry verbatim username-bearing absolute paths.** A real leak channel *only if* cross-vault merge or ref-export is in scope; for local-only single-machine v1 it is fine. Name the data class of ref records explicitly so the path channel isn't silently treated as safe.
- **Public-CLI vision vs. shipped-CLI reconciliation, plus README overstatement.** Genuine docs-reflect-reality drift: `qrt` with no args prints "error: missing command" (exit 2), the *opposite* of the spec'd dashboard; capture/refine are Claude-only per D13; no search ships, despite the README's "across vendors / searches every session." Real, but a messaging/product-shape fix, not a trust-gate blocker — pair it with the v1 definition-of-done.
- **Corpus format-drift sentinel.** The synthetic fixture may diverge from the real Claude Code transcript schema. Worth a sanitized-real-shaped fixture plus a "parser is total over the corpus, warns on unknown shapes" test, but a hardening item, not a P2 blocker.

---

## The single most important thing the spec set is missing

**A written threat model — specifically, a stated definition of "sensitive" and "who the adversary is."**

In plain terms: the whole spec quietly assumes "sensitive" means "a credential string" and "the adversary" is "an honest user who accidentally pastes a secret into a shareable artifact." That one unstated assumption is what lets all eleven must-fix holes hide in plain sight:

- Raw blobs world-readable to a local user (M4) and shipped wholesale to a cloud bucket (M5) are invisible because the threat model only watches the report/ADP egress, never the disk.
- Third-party PII and pasted customer/teammate content (M7) are invisible because "sensitive" was defined as a credential format.
- The `data_class` lineage the product's headline promises (M8) was never materialized because nobody decided what classes exist or who may read them.

So the spec is a rigorous verifier of *one narrow property* — a known credential token surviving into a shareable artifact — presented as if it attests *trust*.

Before building P2, the spec must add a short Threat Model section naming the principals it defends against — other local users, a malicious hook payload, a cloud backup destination, third-party data subjects, a future loosened source — and what it explicitly does **not** defend against — a root-level attacker, a compromised `qrt` binary. Without that anchor, a green TRUSTED scorecard is an honest measurement of the wrong thing, and the gate's whole purpose — to keep the product from giving false confidence — is quietly defeated at the definitional layer.

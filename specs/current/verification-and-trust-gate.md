# Qratum Verification & Trust Gate

Status: published v0.1.0 trust-gate design; current-candidate proof unverified

Accepted 2026-06-16 and implemented in the published v0.1.0 baseline. The
historical proof surfaces are `make trust` / `qrt trust`. Their result must not
be projected onto later commits or candidate code without executing the exact
artifact and every claimed dimension.

Product supersession: `specs/current/product-direction.md` became authoritative
on 2026-07-11. This file remains security evidence and a source of reusable
invariants, fixtures, and adversarial findings. Its product scope, public CLI,
Claude-only source guard, D10 sequencing, raw-index defaults, and claim that D10
is the only current gap do not govern future tranches. A tranche may reuse a
dimension from this file only when it checks behavior required by that tranche
against the exact artifact.
v2 folds in the adversarial gap review
(`docs/reviews/2026-06-15-verification-benchmark/GAPS.md`).

Proposed milestone: **P2-VERIFY-TRUST-GATE** (sits after vault-first P1 in
`specs/current/qratum-vault-first.md`, or may be pulled forward — see
"Sequencing"). Spec-first: this milestone produces contracts, golden corpora,
the trust-gate harness, and the minimal code fixes the gates demand. It does
**not** build new product surfaces.

**Historical delivery shape.** This spec was the **design source-of-truth for
published P2**. It is not the current product authority. Delivery used a
**Ductum-shaped phased package** — the per-phase task specs live in
`docs/reviews/2026-06-15-verification-benchmark/ductum-specs/qratum-verify-trust-gate/`.
This file states *what* must be true and *why*; the ductum specs carve it into
gated phases that can be dispatched and verified independently.

Source analysis (rationale, line-level evidence, red-team findings):
`docs/reviews/2026-06-15-verification-benchmark/BENCHMARK.md` (benchmark design)
and `…/GAPS.md` (completeness gap review).

In one sentence: today "verified" only means the build is green, but a green
build was caught hiding real credential leaks — so this milestone fixes the
confirmed leaks and stands up a test gate that actually runs secrets through the
pipeline and proves nothing leaks.

Decisions already taken by the maintainer (2026-06-15):
- Spec everything in one place; tackle benchmark + confirmed defects together
  (this file).
- Redactor field-leak fix = **hybrid**: drop git/time/event fields from
  shareable artifacts (report/DTO/ADP), redact them in the redacted-session
  JSON. Fix the `=>` partial-redaction bug regardless.
- **Add a Threat Model section** (§Threat Model below) — the gap review's #1
  finding was that the gate had no stated attacker model.
- **The two newly found leaks (world-readable raw blobs, M4; ungoverned raw
  backup egress, M5) are spec-now / fix-with-P2** — folded as FIX-8/FIX-9 +
  D14/D8 below.
- **Artifact placement (FIX-4) = move derived artifacts under `~/.qratum`**
  (the central workspace, `QRATUM_HOME` override) in `sessions/<session_id>/`.
  No more repo-local `./.qratum/` writes.
- **Scope = BOTH (insurance AND dream)** — this **reverses** the earlier
  insurance-only call (Q3). The insurance lane (preserve + import + stop) stays
  required: verify D1–D9, D11–D14. The **dream / curation tier** (cross-repo
  gateway import, the `memory_import_receipt` schema, D10) is now **in scope as
  its own gated phase** — *not* hidden behind a feature flag. It is gated on the
  personal-memory gateway being deployed (the producer that creates those
  receipts must exist before the round-trip leak proof means anything), but it is
  an included deliverable, not parked.
- **Preservation liveness = ship `qrt vault install-schedule`** — an OS timer
  (launchd/systemd) running `qrt vault backfill` as the safety net behind the
  real-time hook. Not a resident daemon; refine stays on-demand. Ships **with an
  explicit test plan** for the timer (Q1).
- **PII / third-party content (M7) = DEFER.** The in-binary redactor stays
  **Go-native and credentials-only**. qratum is a no-Python single Go binary, so
  Presidio (Python) was considered and **rejected**; no Go PII pass is added
  either. The spec states plainly that qratum redacts **credentials only** and
  preserves third-party/PII content **verbatim**; PII detection is explicit
  deferred future work (Threat Model + honest-residual block).
- **Preservation default (M10 / "nothing lost") = never auto-delete.** Nothing is
  ever removed automatically. The only thing that removes anything is an explicit,
  recorded, tombstone-based erasure verb — now an in-scope deliverable (see below).
- **Promoted from residual to in-scope required work:** disk-full guard on
  copy-on-capture; a tombstone-respecting `qrt vault gc` (refuses to delete
  referenced blobs); a per-object **tombstone-based erasure verb** (so a
  third-party deletion request can be honored as an explicit recorded action);
  streaming backup-verify (no whole-file load, no OOM at GB scale); schema
  completeness as an explicit deliverable (full field parity); scan
  re-introduction as a **hard** gate; untrusted-input hardening (transcript_path
  confinement + no-symlink + size cap).
- **Canary format fixed (M1):** the reflection canary must be a token that
  provably evades all redaction classes by construction (NOT a UUID-v4, which
  the entropy detector self-redacts).

No remaining blocking open decisions for scope; see §5 for the scorecard-surface
question (Q4 — agreed: `qrt trust` command + `qratum.trust_scorecard.v1` schema)
and the public-CLI reconciliation item.

---

## Plain-language glossary

Terms used throughout. The body prefers the plain word; the jargon is kept once
in parentheses for precision.

- **Artifact** — any file the pipeline produces from a captured session
  (normalized JSON, redacted JSON, evidence bundle, review card, HTML report,
  ADP export, UI DTO).
- **Shareable artifact** — an artifact meant to leave the machine or be shown
  to others (HTML report, UI DTO, ADP export). These must carry no raw secrets.
- **Redaction** — removing secrets from text and replacing them with a
  placeholder.
- **Blob** — a raw, un-redacted copy of a transcript stored in the vault.
- **Vault** — the immutable, never-deletable local store of raw blobs.
- **ADP** — the external export format qratum emits for other tools.
- **Canary / reflection-canary** — a unique marker token automatically injected
  into every text field of a test session, so any copy of it that survives into
  an output is a proven leak. "Reflection" = the injector walks the struct's
  fields programmatically (Go reflection) to plant the token everywhere.
- **Planted-secret corpus** — a test data set where secrets are generated
  independently of the redactor, never a hand-picked "secrets we know we catch"
  list.
- **Recall** — the share of real secrets the redactor actually catches (a
  missed secret is a recall failure).
- **Precision** — the share of redactions that were warranted (redacting a
  harmless value is a precision failure / over-redaction).
- **Idempotent** — running the operation again changes nothing.
- **Monotonic** — a count that is only ever allowed to move in one direction
  (here: a known-failure count may never grow).
- **TOCTOU** — time-of-check-to-time-of-use: a race where the world changes
  between "look" and "act," so two concurrent runs collide.
- **Data class / `data_class`** — a sensitivity label attached to every object
  (raw is most sensitive, published is least), with rules about which
  transitions are allowed.
- **Egress** — data leaving the machine (e.g. a backup to a cloud or network
  mount).
- **Fail closed** — on bad/unexpected input, refuse safely rather than pass
  data through.
- **Tombstone** — a recorded marker that an object was intentionally removed, kept
  so the reference graph stays consistent and the removal is auditable. The vault
  never silently deletes; a removal always leaves a tombstone.
- **Erasure verb** — the single explicit, operator-invoked command that removes a
  named object's bytes (writing a tombstone). It is the only path that removes
  anything, and the way a third-party deletion request is honored.
- **GC (garbage collection)** — `qrt vault gc`: reclaims genuinely orphaned blobs
  only, and **refuses** to delete any blob still referenced by a live ref.
- **Insurance vs. dream** — two tiers of qratum's scope. *Insurance* = preserve +
  import + stop (the safety net). *Dream* = the cross-repo curation lane (the
  memory-import gateway, D10). Both are in scope (Q3); the dream tier is a gated
  phase, not flag-hidden.

---

## 1. Why

**The problem in plain terms:** Qratum is a local-first **trust pipeline**, but
"verified" today only means `make verify` is green. That target never runs a
secret through `normalize`, never validates a schema, never compares a stored
blob to its source, and never corrupts a backup to test recovery. An adversarial
audit (2026-06-15) built `qrt`, ran the pipeline, and found **confirmed secret
leaks and an unverified recoverability claim in shipped `main`**. For a security
product, a green build that hides a credential leak is worse than no build.

This milestone does two coupled things:

1. **Fix the confirmed defects** (Part 2), spec-first with locked decisions.
2. **Stand up a real trust gate** (Part 3) — a `make trust` target that actually
   exercises the pipeline. It is Go-native, fixture-driven, and stdlib-only
   (supply-chain rule). It proves the pipeline's security and integrity
   invariants end-to-end on a known corpus, and emits a machine-readable
   scorecard qratum publishes about itself. The gate is designed to **fail
   loudly today** on every confirmed defect (planned-RED), and to never narrow
   itself to whatever currently passes.

Three structural rules apply to every gate. In plain terms: test against secrets
the redactor has never seen, never confuse "the field was deleted" with "the
field was cleaned," and drive the real shipped commands, not a shortcut.

- **R1 — Use adversarial test data, not a curated list.** Planted secrets are
  generated independently of the redactor (unique canary tokens injected by
  reflection into every string field), never a hand-authored "secrets we know we
  catch" list. A surviving token is automatically a recall failure.
- **R2 — A dropped field is not a redacted field.** For any field an artifact
  *carries*, assert the canary is absent **and** the placeholder is present. For
  a field an artifact structurally *drops*, assert the drop — and do **not**
  count it as redaction evidence.
- **R3 — Drive every shipped command that produces an artifact.** Security and
  integrity gates run the real entrypoints — `runDaemonOnce`, `runWithIO`, **and
  the standalone `evidence`/`review`/`report`/`export` subcommands**. Those
  standalone commands read a session file directly and do **not** re-redact today
  (see the standalone-re-redact fix, FIX-10). Artifact placement, live-vs-blob
  reads, and the event/artifact store split are all in scope.

---

## Threat Model

**Why this section exists:** the gap review found the gate had quietly assumed
"sensitive" meant "a credential string" and "attacker" meant "a user who pastes a
secret into a shareable artifact." That single unstated assumption let a whole
class of leaks hide. This section states exactly what the trust gate defends
against, so a green scorecard is an honest measurement of the right thing.

**Who the gate defends against:**

- **Another local user on a shared/multi-user machine.** They must not be able to
  read raw secret blobs at rest. (Drives the file-permission gates, D14. The
  ecosystem already runs more than one host — e.g. a Mac mini running Hermes.)
- **A malicious or buggy hook payload.** An untrusted `transcript_path` coming in
  from stdin JSON must not let `qrt` ingest arbitrary files (via symlink or path
  traversal) into the immutable vault. (Drives the hook-confinement hardening,
  FIX-12 / D1.)
- **A backup destination.** `vault backup` to a cloud or network mount is raw
  data leaving the machine. It must be a consent-gated trust-boundary crossing,
  not a silent default. (Drives the backup-consent fix, FIX-9 / D8.)
- **Third-party data subjects.** Transcripts are verbatim human conversation that
  may contain other people's PII, pasted customer/teammate content, and others'
  authored code. The redactor targets **credentials only** and makes **no
  PII/ownership guarantee** — third-party/PII content is preserved **verbatim**.
  PII detection is **explicitly deferred future work** (M7): the in-binary
  redactor stays Go-native and credentials-only. Microsoft Presidio was
  considered for PII and **rejected** — it is Python, and qratum is a no-Python
  single Go binary; no Go PII pass is added in v1 either. Two named mitigations
  for the third-party case ship in this milestone even though PII *detection*
  does not: the tombstone-based per-object erasure verb (so a deletion request
  can be honored as an explicit recorded action) and consent-gated raw egress
  (M5/FIX-9). (Drives the residual statement and scope note, M7.)
- **A future loosened source.** Non-Claude/vendor content that is archived but has
  no redaction path must not become a silent leak channel. (Drives the
  source-scope guard, D13.)
- **The pipeline leaking to itself.** A downstream artifact could re-introduce
  content the redactor never re-scanned, or a sensitivity class could be silently
  upgraded. (Drives the boundary-enforcement work, D4, plus the data-class field,
  FIX-13.)

**Explicitly NOT defended against (stated, not assumed away):**

- A **root-level attacker**, or anyone who can already read the user's whole home
  directory by other means.
- A **compromised `qrt` binary** or a compromised build/supply chain of the trust
  harness itself. (A green scorecard assumes a non-compromised binary and a
  benign local environment.)
- **At-rest disk encryption** — out of scope; qratum relies on the OS/disk. The
  gate enforces file *permissions*, not encryption.
- **Tamper-evidence of the audit/event log** (hash-chain/signatures) — a real
  gap, deferred for alpha; named in the residual block.

A green `TRUSTED` means exactly this: "this code, over this corpus, leaks no
known or planted credential into any shareable artifact and keeps the vault
integrity invariants, under the benign-local threat model above" — nothing
broader.

---

## 2. Confirmed defects to fix

All confirmed against shipped code on `main` (2026-06-15); evidence in
`BENCHMARK.md` §1 and §"Verification status". Each fix ships **with a test that
locks it shut** (the corresponding trust dimension stays RED until then).

### FIX-1 — the `=>` partial-redaction bug (CRITICAL)
Problem: a secret written with a `=>` arrow leaks in cleartext. The
secret-assignment matcher captures the value as `[^\s\"',;]+`. For the input
`PASSWORD => hunter2pass` it matches key=`PASSWORD `, sep=`=`, value=`>` — so it
redacts only the `>` and emits `hunter2pass` in the clear. (Details:
`secretAssignmentPattern`, `redact.go:22`.)
**Fix:** make the value capture robust to arrow/extra separators — e.g. consume
`[:=>]+` in the separator, or re-scan the text after the placeholder for any
residual secret tokens.
**Test:** assert no residual secret token survives after any placeholder.

### FIX-2 — git/time/event fields leak verbatim (CRITICAL) · decision: HYBRID
Problem: a set of metadata fields is never sent through the redactor, so they
ship raw — and a branch name can itself be a secret. `redactQratumSession` never
routes `started_at`, `ended_at`, `source_event_id`, `git.branch`, or
`git.head_sha` through `redactString`; `git.remote` is routed but SSH remotes
match no pattern. The committed golden file even ships
`git@github.com:edictum-ai/qratum.git` plus branch and timestamps unredacted. A
branch like `feature/customer-acme-prod-keys` would leak straight into the report
and ADP. (Details: `redact.go:206-246`;
`fixtures/redaction/secret-session.redacted.golden.json:32`.)

**Locked fix (hybrid):**
- **Drop** `git.branch`, `git.head_sha`, `git.remote`, `started_at`,
  `ended_at`, `source_event_id` from every *shareable* artifact (HTML report,
  UI DTO, ADP export). They add little to those surfaces.
- **Redact** them in the redacted-session JSON (route through `redactString`)
  so the local redacted store carries no raw secret either.
- Extend the SSH-remote class to a redaction pattern (`git@host:org/repo.git`).
- **Re-redact the committed golden** so the contract no longer encodes the leak.

### FIX-3 — recoverability is not actually wired (CRITICAL, architectural)
Problem: the headline promise "a transcript Claude deletes tomorrow is
recoverable" is true for the raw blob but **false for every derived artifact** —
because the refinery only reads the live transcript, never the stored blob. The
daemon resolves and hard-requires the **live** `transcript_path`; it never falls
back to the vault blob (`event.Raw.Digest`). So a transcript that Claude deleted
but that exists as a blob fails the refinery. (Details: `daemon.go:159`,
`requireTranscriptFile`.)
**Fix:** when the live path is missing or changed, read from the blob by digest.
**Test (recoverability, D6a):** capture → delete source → run refinery →
succeeds from the blob.
This is the single most important correctness fix. The scorecard headline must
read `NOT-TRUSTED` (or carry an explicit "recoverability not wired" gap) until it
lands.

### FIX-4 — raw un-redacted artifacts escape into repo directories (HIGH) · decision: CENTRAL HOME
Problem: the daemon writes raw secret-bearing artifacts into the current working
directory, possibly inside a git-tracked repo. It writes
`./.qratum/sessions/*.normalized.json` (containing `sk-ant-…`, `supersecret`)
relative to `os.Getwd()`. That leaves three different locations in play: the
event store is central (`~/.qratum`), the transcript is the live path, and the
artifacts are repo-local.
**Locked fix:** land **all** derived artifacts under the central workspace at
`~/.qratum/sessions/<session_id>/` (`QRATUM_HOME` override), in the layout the
operational model already specifies (`normalized.json`, `redacted.json`,
`evidence.json`, `review.json`, `report.html`, `session.adp.jsonl`). Record the
repo as metadata so `qrt sessions list` / `review` can filter by repo. No more
`./.qratum/` writes. This closes both the "raw secrets in a git-tracked project
tree" surface and the per-repo `.gitignore` dependency.
**Test (artifact placement, D11):** no artifact is written outside `QRATUM_HOME`;
a run from inside a git repo leaves the working tree untouched.

### FIX-5 — short ref-id prefix can collide (HIGH, latent)
Problem: two different blobs whose digests share the same first 12 hex characters
collide into one ref path, and the second is wrongly rejected as a duplicate. The
ref filename truncates to a 12-hex prefix while the blobs themselves use the full
digest. The birthday-bound risk grows across machines and hundreds of blobs.
(Details: `RawRefIDForDigest`, `workspace.go:88`; rejection at `vault.go:265`.)
**Fix:** use enough digest bytes (or the full digest) for ref identity.
**Test (vault integrity, D2):** two distinct digests sharing a 12-char prefix
both store.

### FIX-6 — concurrent captures can clobber each other (HIGH, real under the factory)
Problem: two hooks running at the same time can both pick the same capture path,
and the second silently overwrites the first — a lost capture. The code does
`os.Stat` and then the caller does `os.Rename` with no lock, so two concurrent
hooks can both see `ErrNotExist`. The ductum parallel-agent factory produces
exactly this concurrency. This is a time-of-check-to-time-of-use race (TOCTOU).
(Details: `nextCaptureEventPath`, `hook.go:340/375`.)
**Fix:** create the file with `O_EXCL` instead of stat-then-rename.
**Test (idempotency + recovery, D6):** a goroutine-hammering race test; `go test
-race` clean.

### FIX-7 — known evasion classes the redactor misses (best-effort; xfail allowed)
Problem: several secret shapes slip past the current patterns. AWS keys
containing a `/` evade the high-entropy detector (it returns false on any
`/`-bearing string). Relative paths like `./config/prod.env` and home paths like
`~/.aws/credentials` evade the absolute-path patterns. Space-separated
assignments evade the `[:=]` separator. Out of current scope:
unicode/zero-width-split keys, base64-of-secret, AKIA IDs, and
Stripe/`AIza`/`glpat-`/`SG.` prefixes. (Details: `highEntropyPattern`,
`redact.go:452`.)
**Fix:** extend the cheap, high-confidence classes (AWS `/`-keys, relative/home
paths, space-separated assignments). The **genuinely-hard classes go in a
version-controlled xfail/known-miss file** with a residual-risk note — they are
*named*, not silently absent. Redaction stays honestly labeled **best-effort
alpha over an enumerated allowlist**.

### FIX-8 — raw secret blobs are world-readable (CRITICAL, from gap M4)
Problem: every archived raw blob is a world-readable un-redacted transcript, so
any local user can read it and bypass the entire redaction pipeline. **Confirmed:**
blobs, refs, and state are written `0o644` (world-readable); the workspace root is
never created with restrictive permissions, so `~/.qratum` inherits the process
umask (often `0o755`). (Details: `vault.go:207` `Chmod(0o644)`, refs at `:278`,
state at `:315`; `workspace.Resolve()` never `MkdirAll`s the root.)
**Fix:** have `Resolve()` (or a setup step) `MkdirAll(root, 0o700)` and chmod an
existing root; change blob/ref/event/state files from `0o644` to `0o600`; change
dirs from `0o750` to `0o700`.
**Test (data-at-rest, D14):** every vault file is `0o600`, every dir `0o700`;
flip one to `0o644` → gate RED.

### FIX-9 — `vault backup <dest>` ships raw off-machine with no consent (CRITICAL, gap M5)
Problem: backup copies raw secret blobs to any destination — including a cloud or
network mount — with no consent, violating the model's own "raw never leaves by
default" boundary. **Confirmed:** `Backup` refuses only `dest==home`, then
`copyTree`s the whole `~/.qratum` (including `raw/blobs/**`) to any path.
`vault-first` even advertises rclone/restic (cloud) as the use. (Details:
`vault.go:400`.)
**Fix:** treat a backup that includes raw as a trust-boundary crossing. Emit the
one-line consent audit event before any backup whose source includes `raw/`. When
`dest` is non-local (or undetectable), require an explicit `--allow-raw-egress`
acknowledgement. Optionally offer a redacted-only backup mode.
**Test (backup restorability + egress consent, D8):** a backup of a raw-bearing
vault without consent/ack is refused; with it, the consent audit event is emitted.

### FIX-10 — standalone subcommands do not re-redact their input (HIGH, gap M6)
Problem: the standalone commands emit artifacts straight from whatever session
file they are handed, with no redaction step — so feeding them a raw session
produces raw secrets in the output. **Confirmed:** `report.go:57` and
`evidence.go:144` read a session file and emit artifacts with **no**
`redactQratumSession` call; only `export.go:128` redacts. `qrt evidence <raw
.normalized.json>` emits an evidence bundle with raw secrets.
**Fix:** standalone `evidence`/`review`/`report` must either **reject** a
non-redacted session (require `pipeline_status == redacted`) or redact internally
before building.
**Test (boundary enforcement, D4 / R3):** feed a raw secret-bearing session to
each standalone command; assert zero canary survival or a clean rejection.

### FIX-11 — the ADP key strip is a fail-open denylist (HIGH, gap M9)
Problem: the export only blocks a fixed list of known internal keys, so any *new*
internal key passes straight into the external ADP. **Confirmed:**
`isQratumOnlyExportKey` denylists six known keys plus `x-qratum-`; any other
internal key (a new field, a nested annotation) passes through. (Details:
`export.go:373`.)
**Fix:** make the ADP (and any external export) an **allowlist projection** —
build the output from named fields only; never pass arbitrary internal maps
through.
**Test (boundary enforcement, D4):** inject a random unknown internal key into a
nested input map; assert it is absent from the ADP.

### FIX-12 — the hook will capture arbitrary files via an untrusted path (HIGH, gap)
Problem: the hook archives whatever path the incoming JSON names, with no checks,
so an attacker-chosen or symlinked path can pull arbitrary files into the
permanent vault. **Confirmed:** `ArchiveFile` uses `os.Open` (which follows
symlinks), with no confinement and no size cap. A symlinked or attacker-chosen
path pulls arbitrary files (`~/.ssh/id_rsa`, a device) into the immutable,
never-deletable vault.
**Fix:** confine `transcript_path` to an allowlist of roots
(`~/.claude/projects`, the resolved cwd subtree); use `Lstat`/`O_NOFOLLOW` to
reject symlinks; reject non-regular files; cap the read with `LimitReader`.
**Test (capture fidelity + hook safety, D1):** hostile-payload fixtures
(symlink-to-secret, `..` traversal, non-regular file) are all rejected, recorded,
and exit 0.

### FIX-13 — there is no carried `data_class` field to check (HIGH, gap M8)
Problem: the model's headline "no boundary may silently upgrade to a more
sensitive data class" has nothing to enforce, because the field does not exist.
**Confirmed:** zero `data_class`/`DataClass` hits across `cmd/`, `internal/`, and
`schemas/`.
**Fix:** add a required `data_class` field to every emitted object schema
(raw_ref, session, review_card, evidence, ADP wrapper, UI DTO). It is an enum
ordered by a committed, one-direction-only sensitivity lattice (`raw > redacted >
review > corpus > published`).
**Test (boundary enforcement, D4):** every emitted object declares a class; no
transform raises the class except a named downgrade; every export boundary
refuses a class above its allowlist.

### FIX-14 — copy-on-capture has no disk-full guard (HIGH, promoted from residual)
Problem: the store grows forever on a finite disk, and the unbounded copy then
fails silently every session (`copy_status=failed`, exit 0). There is even a
config knob (`disk_free_min_gb`) that nothing reads. (Details: `ArchiveFile`'s
unbounded `io.Copy` at `vault.go:202`, no `LimitReader`; `disk_free_min_gb`
unused.)
**Fix:** add a disk-free **preflight** that degrades **loudly** (not a silent
`copy_status=failed`) plus a doctor escalation threshold; wire `disk_free_min_gb`
in (or remove the dead knob). This is now **in-scope required work**, not a
residual.
**Test (capture fidelity, D1):** with simulated low free space, capture refuses
loudly and the failure is recorded, not swallowed.

### FIX-15 — no garbage collection / retention / safe-shrink path (HIGH, promoted from residual)
Problem: the store is append-only forever with no reaper for orphaned blobs, so
the only way to shrink is the unsafe manual `rm -rf ~/.qratum`, which destroys
the irreplaceable data the tool exists to keep.
**Fix:** ship a **tombstone-respecting `qrt vault gc`** that reclaims only
genuinely orphaned blobs and **refuses to delete any referenced blob** (a blob
named by a live ref, or by a tombstone that is not itself an erasure). The
preservation default is unchanged — see FIX-16, "nothing lost." This is now
**in-scope required work**, not a residual.
**Test (vault integrity, D2):** `qrt vault gc` reclaims a synthetic orphan;
attempting to gc a referenced blob is refused; the no-delete count invariant
holds for everything still referenced.

### FIX-16 — no per-object erasure verb; the vault can't honor a deletion request (HIGH, promoted from residual; M7/M10)
Problem: the model's preservation promise is correct ("a transcript Claude
deletes tomorrow is recoverable"), but it left the vault with **no** way to ever
remove anything — so a legitimate third-party deletion request (e.g. someone's
PII captured in a transcript) cannot be honored at all.
**Fix:** add a per-object, **tombstone-based** erasure verb. The default is and
remains **NEVER auto-delete / "nothing lost"** (M10): nothing is removed by any
automatic path. Erasure is an **explicit, operator-invoked, recorded action**
that writes a tombstone and removes the named blob's bytes, so the action is
auditable and the reference graph stays consistent (`qrt vault gc` and the
no-delete invariants respect tombstones). This is the one path that can honor a
third-party deletion request — and it is the **only** thing that removes anything.
This is now **in-scope required work**, not a residual.
**Test (vault integrity, D2):** an erasure writes a tombstone and removes exactly
the targeted blob's bytes, the action is recorded, and no automatic path ever
removes a blob without one.

---

## 3. The trust gate

`make trust` is the new gate. In plain terms: a Go program that runs the real CLI
against known test data and reports whether anything leaked or broke. It is
Go-native (`//go:build trust` tag or `cmd/trustbench`), fixture-driven, and
**stdlib-only** (supply-chain rule), and it drives the **real CLI entrypoints**.
It emits machine-readable JSON plus a human summary, then runs as `verify: …
security trust`. CI fails the job on any red security/integrity gate and uploads
the scorecard JSON.

### Dimensions (acceptance criteria condensed; full detail in BENCHMARK.md §2)

Each "dimension" (D-number) is one thing the gate checks. They split into a
security tier and an integrity tier; any failure in either tier blocks release.

**Security tier — any failure BLOCKS release:**

- **D1 — Capture fidelity + hook safety.** In plain terms: every capture records
  exactly one faithful event, broken inputs are recorded instead of swallowed,
  and hostile inputs are refused. Assert: exactly one event with populated
  `raw.*`; degraded cases recorded not swallowed (exit 0, counts incremented,
  stderr warning); oversized/invalid payload rejected; no transcript content in
  the event; a speed gate (50MB transcript copy+hash under threshold); the
  warning channel (stderr) emits no raw transcript content. Import-isolation:
  extract capture into `internal/capture` importing only
  `crypto/sha256,os,io,vault,workspace`; pin the allowed import set; a `net`
  import flips it RED. (Until that refactor, the scorecard states the import gate
  is package-`main`-coarse.) **Disk-full guard (FIX-14):** with simulated low free
  space, capture refuses **loudly** (not a silent `copy_status=failed`) and the
  failure is recorded; the doctor escalation threshold fires.
- **D3 — Redaction safety (CROWN JEWEL).** In plain terms: inject a unique marker
  into every text field of a fully-populated test session, run the real
  pipeline, and prove not one marker survives into any output. The
  reflection-canary harness injects a unique token into every string field of a
  fully-populated `qratumSession` (recursing through git/turns/tool-calls/nested
  inputs), drives the **real daemon** end-to-end, and asserts **zero tokens**
  survive into the redacted session, evidence, review, report HTML, ADP, or
  capture event. Recall = **binary 100%** on the covered corpus (any leak
  blocks). Hardened by the gap review:
  - **Canary format (M1) — the marker must not redact itself.** The token must
    **provably evade all 8 redaction classes by construction**:
    lowercase-alpha-only, `<32` chars, single character class, no separator
    (e.g. `qratumcanaryNNNN`). A UUID-v4 is **forbidden** as the canary, because
    its hyphens are matched by `highEntropyPattern` and `looksHighEntropy` only
    excludes `/`/`\` — so a UUID would self-redact and the gate would pass for
    the wrong reason. Keep the hyphenated UUID strictly as the **precision
    tripwire** (it must survive), never as the canary.
  - **Harness self-test (M2) — prove the test can both pass and fail.** (a) A
    known-positive: route one field deliberately around `redactString` and assert
    the gate goes RED (proves it *can* fail). (b) Feed the chosen canary alone
    through `redactString` and assert it comes back **unchanged**. (c) The
    injector **panics loudly** on any field kind it cannot reach (unexported, nil
    pointer, unhandled type) rather than silently skipping — a skipped field
    would under-count the numerator and denominator identically and read
    false-green.
  - **Map keys + non-string scalars (M3) — cover what reflection can't reach.**
    `redactAny` copies map *keys* verbatim and returns non-string scalars
    unchanged; `tool_calls[].input` and `provenance` are `map[string]any`.
    Re-key map outputs through `redactString` and scan/coerce non-string scalars
    (or prove them inert). Add a **hand-authored** fixture that plants tokens **as
    map keys** and as stringified numbers, since struct-reflection cannot plant a
    dynamic map key.
  - **Re-introduction (gap) — catch fields copied back in downstream.** Downstream
    artifacts copy fields straight from the struct (`evidence.go` builds `Summary`
    and copies `started_at`/`ended_at`/`source_event_id`), so the FIX-2 "redact in
    JSON" step alone is insufficient. Add a canary case whose only planted token is
    in a re-introduced field, and make the terminal per-artifact byte-scan a
    **hard** gate (see D4).

  Plus: a field-coverage contract (the scanned set must equal the
  reflected-string-field set, via an independent AST / `go/types`-derived list,
  with justified exemptions only and a positive non-shareability test); per-class
  recall including the leaking classes; the `=>` residual-token assertion (FIX-1);
  a precision corpus seeded with a bare 40-hex SHA, a hyphenated UUID, and a
  `sha256:` digest (over-redaction is a regex bug to fix, not a budget to
  inflate); determinism and idempotency of placeholder numbering; and `redactAny`
  must fail **closed** (comma-ok, not panic) on a malformed `Input`/`Provenance`
  shape. **Wire `transcript-with-secret.jsonl` in for the first time.**
- **D4 — Trust-boundary enforcement.** In plain terms: one shared checker that
  understands text encodings scans every output's bytes and confirms each carried
  secret is gone and its placeholder is present. The checker is
  **encoding-aware** (literal-absence + placeholder-presence, per R2), driven by
  the reflection token set, over all artifact byte-streams — it scans
  HTML-unescaped text for the report and JSON-unmarshaled string values for
  ADP/DTO (a secret containing `< > & " + /` could otherwise survive escaped and
  read clean). For ADP/report/DTO: assert canary-absent + placeholder-present for
  *carried* fields; assert structural *drop* for dropped fields (don't credit a
  drop as redaction). The ADP and every external export must be an **allowlist
  projection**, not a denylist strip (FIX-11) — test that an injected unknown
  internal key is absent. The terminal per-artifact byte-scan is a **hard** gate
  (not optional defense-in-depth); give it a genuinely independent generic
  high-recall detector (broad `AKIA`/`AIza`/`glpat`/`pk_`/`SG.` + entropy)
  distinct from the redactor's own patterns, **or** label it a regression
  tripwire in the residual (otherwise it inherits the redactor's recall exactly).
  `data_class` lineage is enforced per FIX-13. No-network import gate on the
  refinery.
- **D7 — Operational truthfulness (doctor).** In plain terms: the health command
  must tell the truth — warn when something is wrong, stay quiet when all is well,
  and never claim a check it can't actually make. Assert: the no-hook,
  stale-backfill, copy-failure, and unverified-backup warnings all fire (injected
  state, stubbed clock); the cloud-blind-spot line is **always present** (a hard
  literal gate); the healthy path → "warnings: none". `transcript_drift` is
  **labeled a heuristic, not a correctness gate** — it goes tautologically 0
  post-backfill and false-warns on the source-deleted success case. (Fix is to
  compare archived refs against an independent expectation, e.g. the count of
  `session_end` events with `copy_status ∈ {copied,deduped}`.)
- **D11 — Artifact placement / containment.** In plain terms: everything the
  pipeline writes lands under the central home, never in a repo. Per FIX-4: **all**
  derived artifacts land under `QRATUM_HOME` (`~/.qratum/sessions/<session_id>/`),
  never in a repo or working tree. Assert that a run from inside a git repo writes
  nothing to the working tree, and that no artifact path resolves outside
  `QRATUM_HOME`.
- **D13 — Source-scope guard.** In plain terms: only Claude-Code sessions have a
  redaction path, so exporting any other source must be refused — closing a future
  silent leak channel. `validateQratumSession` rejects non-`claude-code` sources,
  but `archive` ingests Codex/vendor blobs with no redaction path. Assert that
  exporting or redacting a non-Claude session is **rejected**, so a future
  loosening can't silently open a leak channel. The scorecard states: capture +
  refinery are Claude-Code-only.
- **D14 — Data-at-rest / local isolation (FIX-8).** In plain terms: lock down file
  permissions so another local user can't read raw blobs directly. Assert that the
  `~/.qratum` root and every `raw/` subtree dir is `0o700` and every
  blob/ref/event/state/artifact file is `0o600`; flip one file to `0o644` → gate
  RED. The threat is another local user on a shared machine reading raw secret
  blobs directly and bypassing redaction. The scorecard states plainly: **at-rest
  disk encryption is out of scope** — the gate enforces permissions, not
  encryption.

**Integrity tier — any failure BLOCKS release:**

- **D2 — Vault integrity.** In plain terms: prove the stored blob really matches
  its source, deduplicates, can't be silently mutated or auto-deleted, survives a
  write failure cleanly, and that the only removal path is an explicit recorded
  erasure. Assert: **independent-source** byte-equality (hash the original source
  with a *separate* reader — re-hashing the committed blob is tautological);
  dedup; immutability (a collision is rejected); atomicity (no `.tmp` leak; inject
  a write failure → no partial blob or ref); **no auto-delete** (a grep gate plus
  a count-never-decreases check: nothing is removed except through the explicit
  tombstoned erasure verb, FIX-16 — preservation default is "nothing lost");
  **`qrt vault gc` (FIX-15)** reclaims a synthetic orphan but **refuses any
  referenced blob**; the **tombstone-based erasure verb (FIX-16)** removes exactly
  its target, writes a tombstone, and records the action; the FIX-5 ref-id
  collision case; and **multi-machine merge** (union two synthetic vaults →
  collision-free, no loss; else the scorecard says "cross-vault merge
  UNVERIFIED").
- **D5 — Determinism / reproducibility.** In plain terms: the same input always
  produces byte-identical output, on any machine — and a golden file can never
  quietly bless a leak. Assert: golden byte-equality; run-twice stability in two
  temp workspaces; machine-independence (`QRATUM_HOME` + `t.TempDir`, ToSlash);
  stable ADP ordering. This **runs downstream of leak-freedom:** a standalone lint
  asserts **no committed golden contains a known secret or internal identifier**
  (`edictum-ai/qratum.git`, real head SHAs) — so a regen guard can't certify a
  leak as the intended output.
- **D6 — Idempotency + crash recovery.** In plain terms: re-running changes
  nothing, a crash mid-write recovers cleanly, and a missing transcript fails
  loudly rather than silently. Assert: idempotent re-run; partial-artifact detect;
  crash-mid-write tmp recovery; missing-transcript loud-fail; the FIX-6 TOCTOU race
  test; refine-source consistency (a mutated live file → refuse on digest
  mismatch, or document "the blob is the only authoritative copy").
- **D6a — Recoverability (FIX-3).** In plain terms: prove a deleted source still
  refines from the blob. Capture → delete source → refinery succeeds from the blob.
  **Planned-RED until the daemon is rewired.**
- **D8 — Backup restorability + egress consent (FIX-9).** In plain terms: a backup
  must restore correctly, detect corruption, stream large blobs without blowing up
  memory, and require consent before raw data leaves the machine. Assert: backup
  success; **corruption detection** (flip one byte → fail); round-trip restore
  (point `QRATUM_HOME` at dest; status/doctor/Summary match); refuses dest==home.
  **Verify against the recorded `ref.Digest`**, not a live re-read (`verifyTree`
  currently re-reads the source, so post-capture drift and matched corruption pass).
  **Stream, don't `os.ReadFile`** whole blobs (OOM at GB scale; test bounded RSS on
  a large synthetic blob). **Egress consent:** a backup whose source includes
  `raw/` requires the consent audit event (and a non-local-dest ack) — assert it is
  refused without and emitted with.
- **D9 — Schema / contract conformance.** In plain terms: every output is validated
  against a strict schema that forbids extra keys, so a leaking extra field gets
  caught. **Schema completeness is an explicit deliverable, not an assumption.**
  The session schema today describes a different object than the code emits (about
  12 emitted fields are undeclared: `transcript_path`, `source_event_id`, `git`,
  `pipeline_status`, `artifact_paths`, `business_metrics`, `provenance`, …), which
  makes `additionalProperties:false` unsatisfiable. So the deliverable is **full
  field parity**: enumerate every emitted field with its type, give `provenance` a
  typed sub-schema, and add a self-test asserting the struct-tag set **equals** the
  schema-property set. Add `additionalProperties:false` (or explicit forbidden-key
  lists) to every `schemas/*.json` **first**, **recursively** (nested `turns[]`,
  `tool_calls[].input`, `raw`, `workspace` each get their own `items`/`properties`
  + `additionalProperties:false`) — that nesting is exactly where the audited leaks
  live. Pin drift direction to **emitted-keys ⊆ schema-declared-keys**. The
  validator self-test must **reject** an instance with an injected extra key. The
  denominator = **emitted objects** (every `schema_version` literal, including the
  schemaless ADP and the redaction summary), with a hyphen/dot name-mapping
  assertion; a missing schema → loud fail. Use a stdlib-only mini-validator. **The
  config schema is a missing P0 deliverable** — add it. **The
  `memory_import_receipt` schema (D10) is a committed contract deliverable in this
  milestone** (the dream tier is in scope) — wire it in here.

**Gated phase — the dream / curation tier (IN SCOPE, gated on the gateway):**

D10 is **in scope** (Q3 resolved to BOTH — insurance and dream). It is delivered
as its own **gated phase**, not hidden behind a feature flag. The gate is real:
the round-trip leak proof is only meaningful once the personal-memory gateway —
the producer that actually creates these receipts — is deployed. So the
contract-and-schema work lands now; the full behavioral round-trip is sequenced
behind gateway Phase 1+.

- **D10 — Cross-repo import.** In plain terms: round-trip a memory-import receipt.
  **Now (committed this milestone):** define the `memory_import_receipt` JSON
  Schema as a committed contract and wire it into D9; make `qrt vault archive
  <receipt> --kind memory_import_receipt` round-trip with the kind pinned (the
  default is `source_metadata`, a mislabel footgun). Because the no-leak check
  against a synthetic fixture is **circular** until a real producer exists, that
  specific check is labeled **"not-yet-meaningful as a leak proof"** until the
  gateway is deployed — but the schema and the archive path are real deliverables,
  not parked behind a flag. **Gated on personal-memory gateway Phase 1+:** the
  full round-trip / idempotent re-run with `supersedes[]` / `namespace_forbidden`
  / unknown-`contentClass` reject / **reject outcomes outside the gateway's real
  vocabulary**. Do **not** build the gateway producer **inside** qratum — qratum
  imports its receipts; it does not create them.

**Liveness:**

- **D12 — Passive-capture liveness.** In plain terms: prove the global hook is
  actually installed and pointed at the right command, and ship a scheduled
  backfill as the safety net so preservation survives even if the user walks away.
  Assert that the installed global hook command equals the shipped capture
  entrypoint; assert that doctor's `last_capture_at`/`last_backfill_at` staleness
  gates fire. **Decision (Q1): ship `qrt vault install-schedule`** — an idempotent
  OS timer (launchd plist / systemd timer) running `qrt vault backfill` as the
  safety net behind the real-time hook. It is **not** a resident daemon (the OS
  runs a one-shot on a schedule); refine stays on-demand. Until installed,
  doctor/scorecard state "preservation freshness depends on a schedule that is not
  installed." This closes the 0%-engagement / "survives abandonment" gap (M10).

  **Test plan (Q1 — how to test an OS timer without touching the real home):**
  - **Print / dry-run mode.** `install-schedule` supports a dry-run that prints the
    exact plist/unit it would write and the exact command it would install, writing
    nothing — the test asserts on that output.
  - **Fake schedule dir.** Point the install target at a temp directory (a
    `t.TempDir()` standing in for `~/Library/LaunchAgents` / the systemd user
    dir), never the real one; the test asserts the file lands there and nowhere
    else.
  - **Assert generated content.** Parse the generated plist/unit and assert its
    fields — schedule cadence present, and the **installed command equals `qrt
    vault backfill`** (the single most important assertion: it must not drift to
    `refine` or anything else).
  - **Idempotency.** Run install twice; the second run re-does no work and the
    file is byte-identical.
  - **Clean uninstall.** `uninstall` removes exactly the file it wrote and leaves
    the fake schedule dir empty; a second uninstall is a clean no-op.

### Scorecard & gate

**How the result is reported, in plain terms:** two hard tiers, no averaging, and
a plain-word headline instead of a score out of 100. Averaging is banned because a
green-elsewhere lull could hide a single leak. Each security and integrity
dimension is PASS/FAIL. The headline is an enum:

- `TRUSTED` — all security+integrity gates green **and** no known-miss class in
  the corpus **and** recoverability wired.
- `TRUSTED-WITH-NAMED-GAPS` — gates green but a known-miss class is present or
  recovery is unwired. **Cannot be abbreviated to `TRUSTED`**; carries a non-zero
  `gap_count`.
- `NOT-TRUSTED` — ≥1 red gate.

**Three gate states (M11 — resolves the CI paradox).** The problem: wiring
"planned-RED" gates straight into required CI would merge-lock the whole repo. So
every gate is one of:
- **BLOCKING-RED** — a regression of a currently-green gate; fails CI hard.
- **KNOWN-RED** — a gate that is RED by design until its in-milestone fix lands
  (D6a/FIX-3, the D9 drift work, the not-yet-fixed leak classes). Tracked,
  **CI-non-blocking**, **monotonic** (the KNOWN-RED count may not increase), each
  entry carrying a tracking note + owner + deadline. CI fails only if a KNOWN-RED
  outlives its deadline or a green gate regresses.
- **GREEN.**

This lets `make trust` go into required CI without merge-locking, without
weakening checks ("CI is sacred"), and without letting a real new leak hide.

**Anti-gaming.** Split the D3 recall metric into (a) a hard "no leak in the
covered corpus" blocker and (b) a tracked, non-blocking, **monotonic**
extended-class recall — CI fails if the extended corpus has fewer than the
documented N classes or recall % regresses. Corpus shrinkage becomes a visible,
reviewable regression rather than an invisible omission.

**The scorecard is itself a governed object.** Define `qratum.trust_scorecard.v1`
(wired into D9), run the no-leak checker over its own bytes, and carry a
provenance block (build commit, corpus digest, schema digest, timestamp) so a
green score is verifiable as "this score, from this code, over this corpus."

**Honest-residual block (always printed verbatim under the headline).** It states,
in plain words, exactly what the gate does NOT cover: extended-class recall % plus
the 8 enumerated regex classes; the unicode / `/`-exclusion / 32-char-floor limits;
"redaction is a single upstream pass — the artifact checks are correlated, not
independent layers"; the recoverability and artifact-placement status;
`transcript_drift` is a heuristic; config schema status; **D10 / the dream tier is
an in-scope gated phase whose round-trip leak proof is meaningless until the
personal-memory gateway is deployed**; **the redactor is credentials-only and
Go-native — PII / third-party content is preserved VERBATIM and is NOT redacted;
PII detection is explicitly deferred future work (Presidio was considered and
rejected for qratum's no-Python single-Go-binary constraint, and no Go PII pass
ships in v1)**; **at-rest encryption out of scope (permissions only)**; **the
audit/event log is not tamper-evident**; **the preservation default is "nothing
lost" — nothing is ever auto-deleted; the only removal path is the explicit,
recorded, tombstone-based erasure verb (FIX-16), which is how a third-party
deletion request is honored, alongside `qrt vault gc` (FIX-15) which refuses
referenced blobs**; **cross-vault merge drops per-machine state/event cursors
(blobs only are dedup-clean)**; **`vault backup` of raw is the sanctioned,
consent-gated exception to "raw never leaves the machine."**

**What a green `TRUSTED` does NOT promise:** review quality (heuristic, 0% human
baseline); cloud/web session coverage (uncaptured by design); Codex/vendor
coverage (archive-only, no redaction path); that redaction catches *all* secrets
(best-effort alpha — a novel format or new struct field can leak); **any PII or
third-party-content guarantee** (transcripts are verbatim human conversation; the
redactor targets credentials only); multi-machine merge beyond dedup;
tamper-evidence of the audit log; protection against a root-level attacker or a
compromised `qrt` binary; at-rest encryption; regulatory compliance.

---

## 4. Sequencing

The order to build in, cheapest-and-highest-value first.

**Tier 0 — the contained security fixes (cheapest, highest value):**
1. FIX-1 (`=>` arrow bug), FIX-2 (hybrid git/time/event + re-redact golden),
   FIX-5 (ref-id collision), FIX-6 (TOCTOU race), **FIX-8 (file perms
   `0o600`/`0o700` — one-line, zero-risk), FIX-11 (ADP allowlist), FIX-12 (hook
   path confinement), FIX-14 (disk-full preflight guard on copy-on-capture)** —
   each with its locking test.

**Tier 1 — the crown jewel + boundary gates:**
2. The D3 corpus + reflection-canary harness (correct canary format M1 +
   self-test M2 + map-key/non-string fixtures M3) + the shared **encoding-aware**
   no-leak checker (R1/R2); wire `transcript-with-secret.jsonl`; the evasion +
   xfail + precision fixtures. **FIX-10** (standalone subcommands re-redact or
   reject).
3. D9 schema hardening (recursive `additionalProperties:false`, the mini-validator
   with a reject self-test, emitted-object enumeration, **the `data_class` field /
   FIX-13**) + the **config schema** + `qratum.trust_scorecard.v1`.
4. The D5 no-secret-in-golden lint (+ scan git history, gap). The D2/D6/D7/D8
   property tests (**FIX-9 backup consent**, **streaming backup-verify against
   `ref.Digest` — no whole-file load, bounded RSS on a GB-scale synthetic blob**).
   The D11 (central-home placement / FIX-4) / D13 / **D14 (at-rest perms)** guards.
   **The vault lifecycle verbs: `qrt vault gc` (FIX-15, refuses referenced blobs)
   and the tombstone-based per-object erasure verb (FIX-16) — preservation default
   stays "nothing lost"; both wire into D2.**
5. The scorecard JSON shape + three-state gate model + `make trust` skeleton; wire
   into `make verify` / CI.
6. **`qrt vault install-schedule`** (D12) — the backfill timer, **with the Q1 test
   plan** (dry-run/print mode, fake schedule dir, assert generated plist/unit +
   installed command equals `qrt vault backfill`, idempotent install, clean
   uninstall).

**Then (the architectural fix):** FIX-3 / D6a — rewire the refinery to read from
the blob; **KNOWN-RED until done**; extend D3/D4 to assert the blob-sourced path
also leaks zero secrets.

**Dream-tier gated phase (Q3 = BOTH — in scope, not flag-hidden):** the
`memory_import_receipt` JSON Schema is a committed contract delivered **now**
(wired into D9), and the `--kind memory_import_receipt` archive round-trip lands
now with the kind pinned. The full behavioral round-trip (idempotent re-run,
`supersedes[]`, `namespace_forbidden`, unknown-`contentClass` reject,
reject-outside-vocabulary) is **gated on the personal-memory gateway being
deployed** — qratum imports receipts, it does not build the gateway producer.

**Minimum next milestone if pulled forward:** **FIX-1/2/8/11/12/14 + the D3
crown-jewel (correct canary) + D11 central-home + D9 `additionalProperties` +
`data_class`** — this closes the confirmed CRITICAL leaks (secret-through-
normalize, raw-session-on-disk, world-readable blobs, ADP fail-open) plus the
silent disk-full failure, using only shipped code, and makes the rest of the
matrix meaningful.

---

## 5. Decisions resolved (2026-06-15)

All scope decisions are now resolved; nothing below blocks building.

**Resolved by the maintainer (folded into the spec above):**
- **Passive-capture liveness (Q1)** → ship `qrt vault install-schedule` (timer +
  backfill); not a resident daemon — **with an explicit OS-timer test plan** (D12).
- **Artifact placement (FIX-4) (Q2)** → central `~/.qratum/sessions/<session_id>/`
  via `QRATUM_HOME` (already locked); no repo-local writes (D11).
- **Scope (Q3)** → **BOTH** — this **reverses** the earlier insurance-only call.
  The insurance lane stays required (D1–D9, D11–D14). The **dream / curation tier
  is now in scope** as its own **gated phase**, not flag-hidden: the
  `memory_import_receipt` schema and the `--kind` archive round-trip land now; the
  full behavioral round-trip is gated on the personal-memory gateway being
  deployed (D10).
- **PII / third-party content (M7)** → **DEFER**. Redactor stays Go-native and
  **credentials-only**; PII content preserved verbatim; PII detection is explicit
  future work. **Presidio considered and rejected** (Python vs. qratum's no-Python
  single Go binary); no Go PII pass in v1. Stated in the Threat Model and the
  honest-residual block.
- **Promoted from residual to in-scope required work** → disk-full guard
  (FIX-14); tombstone-respecting `qrt vault gc` that refuses referenced blobs
  (FIX-15); per-object **tombstone-based erasure verb** (FIX-16, the only removal
  path, honors third-party deletion requests); **streaming** backup-verify
  (no whole-file load, no OOM at GB scale, verify against `ref.Digest`) (D8);
  schema completeness as an explicit full-field-parity deliverable (D9); scan
  re-introduction as a **hard** gate (D3/D4); untrusted-input hardening —
  transcript_path confinement + no-symlink + size cap (FIX-12 / D1).
- **Preservation default (M10 / "nothing lost")** → **never auto-delete**; only
  the explicit tombstoned erasure verb (FIX-16) removes anything.
- **The two newly found leaks (M4/M5)** → spec now (FIX-8/FIX-9, D14/D8), fix
  with P2.
- **Threat Model** → added (§Threat Model).
- **Canary format (M1)** → a non-matching token + the harness self-test (D3).
- **Scorecard surface (Q4)** → **agreed**: a `qrt trust` user command emitting the
  **`qratum.trust_scorecard.v1`** schema (wired into D9), run the no-leak checker
  over its own bytes, carry a provenance block. The public qratum.dev badge is
  deferred.

**Public-CLI reconciliation (scope item).** Two shipped-vs-spec mismatches must be
reconciled in this milestone (docs-reflect-reality):
- `qrt` with **no args** currently prints `error: missing command` (exit 2), the
  opposite of the spec'd dashboard. **Decide its behavior** — a default **status
  view** (recommended: print vault/doctor status, exit 0) vs. keeping the error.
  Pin the choice and make `qrt trust` / status consistent with it.
- The README's **"across vendors / searches every session"** is an
  **overstatement**: no search ships, and **capture + refine are Claude-Code-only**
  (D13). Reconcile the README to shipped reality and state plainly that capture and
  refine are Claude-Code-only.

---

## 6. Non-goals (so the benchmark doesn't become its own scope monster)

What this milestone deliberately does NOT build:

- No full JSON-Schema library — a stdlib mini-validator only (a vetted,
  ≥7-day-old pure-Go validator only with explicit sign-off; revisit if recursive
  `properties`/`items`/`enum` make the hand-rolled one a false economy).
- No new runtime features beyond the minimal fixes the gates demand and the
  explicitly-unlocked items: the **central-home artifact placement (FIX-4)**,
  **`qrt vault install-schedule` (D12)**, the **disk-full guard (FIX-14)**, the
  **tombstone-respecting `qrt vault gc` (FIX-15)**, and the **per-object
  tombstone-based erasure verb (FIX-16)**. The benchmark may **not** smuggle in
  SQLite, a resident daemon, or new product surfaces — it **reports** them
  RED/residual. (The `internal/capture` import-isolation extraction in D1 is an
  explicit in-scope deliverable, carved out of this.)
- No producer script (`import-claude-memories.ts`) and no personal-memory gateway
  in qratum — qratum imports `memory_import_receipt` blobs, it does not create
  them. (The dream tier is in scope, but only qratum's consumer side.)
- No real secrets or real transcripts committed — synthetic/canary only; the
  committed corpus carries no real internal identifier (and the no-secret lint
  scans git history, not just the working tree).
- No multi-machine merge runtime — verify the dedup-union property with synthetic
  vaults only; state the state/event-cursor loss as a residual.
- No automatic deletion / retention runtime — the preservation default is "nothing
  lost." `qrt vault gc` (FIX-15) reclaims only orphans and refuses referenced
  blobs; the tombstone-based erasure verb (FIX-16) is the only removal path and is
  always an explicit, recorded operator action. No time-based or size-based
  auto-eviction.
- No PII detection of any kind — credentials-only, Go-native; Presidio rejected
  for the no-Python constraint; PII is explicit deferred future work.
- No review-quality / agent-judgment metric.
- No tunable precision budget that absorbs known over-redaction — fix the regex.
- No at-rest encryption or audit-log tamper-evidence — named out of scope in the
  threat model.

---

## 7. Acceptance (of this milestone, when unlocked)

What "done" looks like:

- `make trust` exists, runs the testable-now dimensions against **every**
  artifact-producing CLI entrypoint (daemon + standalone subcommands), and emits
  the `qratum.trust_scorecard.v1` JSON + human summary with the three-state gate
  model.
- The reflection-canary recall test runs through the real daemon, uses a
  **non-self-redacting canary** with the harness self-test passing, covers map
  keys / non-string scalars / re-introduced fields, and is GREEN (FIX-1, FIX-2,
  FIX-10, FIX-11 closed; remaining classes xfail with a residual note).
- FIX-5, FIX-6, **FIX-8 (perms), FIX-12 (hook confinement)** land with locking
  tests; `go test -race` clean. **D14** asserts `0o600`/`0o700` at rest.
- **FIX-9:** `vault backup` of raw is consent-gated and **streams** (no whole-file
  load, verified against `ref.Digest`, bounded RSS on a GB-scale blob); **D11:** no
  artifact is written outside `QRATUM_HOME`.
- **FIX-14:** copy-on-capture has a disk-full preflight that degrades **loudly**;
  `disk_free_min_gb` is wired in (or the dead knob removed).
- **FIX-15 / FIX-16:** `qrt vault gc` reclaims orphans and **refuses referenced
  blobs**; the **tombstone-based erasure verb** removes exactly its target, records
  the action, and is the **only** removal path — the preservation default is
  "nothing lost" and **D2** proves nothing is ever auto-deleted.
- D6a recoverability is GREEN (FIX-3 landed) **or** the scorecard honestly reads
  `NOT-TRUSTED` / `TRUSTED-WITH-NAMED-GAPS` with a KNOWN-RED "recoverability not
  wired" entry (owner + deadline).
- Schemas carry recursive `additionalProperties:false` at **full field parity**
  (the struct-tag set equals the schema-property set); the mini-validator rejects
  an injected extra key; every emitted object maps to a schema; the `data_class`
  field (FIX-13) is required and lineage is enforced; the config schema exists.
- **Dream tier (D10):** the `memory_import_receipt` JSON Schema is committed and
  wired into D9, and the `--kind memory_import_receipt` archive round-trips with
  the kind pinned; the full behavioral round-trip is correctly marked gated on the
  personal-memory gateway, not faked green.
- The committed redaction golden contains no real secret or internal identifier
  (working tree **and** git history).
- `qrt vault install-schedule` (D12) installs an idempotent backfill timer, shows
  what it writes, and ships its **test plan** GREEN: dry-run/print output asserted,
  fake schedule dir, generated plist/unit parsed, **installed command equals `qrt
  vault backfill`**, install idempotent, uninstall clean — none touching the real
  home.
- **Public-CLI reconciliation done:** `qrt` with no args has a pinned behavior
  (status view vs. error) and the README no longer overstates "across vendors /
  searches every session" — it states capture + refine are Claude-Code-only.
- The Threat Model section is present (credentials-only redactor, Presidio
  rejected, PII deferred) and the scorecard prints the full honest-residual block
  verbatim.
- `make verify` includes `trust`; CI fails on any BLOCKING-RED gate (a regression
  of a green gate, or a KNOWN-RED past deadline) and uploads the scorecard.
- The dispatchable per-phase task specs exist under
  `docs/reviews/2026-06-15-verification-benchmark/ductum-specs/qratum-verify-trust-gate/`
  and trace back to this design source-of-truth.
- No non-goal feature was implemented.

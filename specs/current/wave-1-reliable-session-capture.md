# Qratum Wave 1: Reliable Session Capture

Status: accepted technical contract

Date: 2026-07-12

Accepted by product owner: 2026-07-12

Accepted review digest: `5ff1cdb7a5c1cc07fdf1d43994317740a2267c5ab1eba2b1b1af976e85b2b11d`

Target branch: `docs/wave-1-reliable-session-capture`

## Authority And Scope

This contract implements only Wave 1 from:

```txt
specs/current/product-direction.md
```

A **wave** is one bounded stage of work. Wave 1 is the first implementation
stage after the completed product-truth documentation work.

It does not override that product direction. The product owner accepted the
reviewed behavior on 2026-07-12. This file is now the implementation authority
for Wave 1 only.

Wave 1 makes Claude Code and Codex history trustworthy enough for later
library, reader, search, Project, and cost work. It is foundation work, not the
usable application.

## What, Why, And How

### What this wave builds

Qratum reliably gathers exact local Claude Code and Codex session history. It
keeps one identity for a session when that session resumes, moves, or gains
child-agent work. It also records the usage and repository facts needed later.

### Why we need it

The reader, search, Projects, and cost pages will all be wrong if capture loses
sessions, duplicates resumed work, counts tokens twice, or guesses missing
facts. This wave gives those later features trustworthy input.

### How it works

Small source hooks record that something changed. A short macOS background job
then copies a stable snapshot into Qratum's private local storage. Source IDs,
not file paths, identify sessions. Saved fixtures prove how each supported
source version represents sessions and token usage.

## Language Rule

All new Qratum documentation, help, status, errors, and UI copy must use simple
English. Every feature or failure explains:

1. what it does or what happened;
2. why that matters or what it enables; and
3. how it works or what the user should do next.

Technical schema and code names may stay precise, but user-facing text defines
them before using them. For example, a “reconciler” is described first as the
short capture-refresh job that saves changed session files.

### Short glossary

- **Capture refresh / reconciler:** the short job that checks changed source
  files and saves safe copies.
- **Stable snapshot:** a complete file copy whose source did not change while
  Qratum copied it.
- **Content-addressed storage:** files stored by a digest of their exact bytes,
  so identical content is stored once.
- **Atomic publication:** an all-or-nothing update; a crash cannot expose a
  half-written result.
- **Tombstone:** a small deletion marker that prevents an erased session from
  being gathered again.
- **Fixture:** a synthetic saved example and its expected result, used to catch
  source-format changes.

## Plain-English Design Map

| Part | What | Why | How |
| --- | --- | --- | --- |
| Source hooks | Record that a session changed | New work must not be missed | Write one small local event and exit |
| Capture refresh | Saves changed session files | Hooks must remain fast | A macOS job runs every five minutes and on demand |
| Stable snapshots | Saves only a complete file version | A half-written transcript is not exact history | Check the file before and after a streaming copy |
| Safe source roots | Limits which files Qratum may read | A payload path is untrusted | Accept only known Claude Code and Codex data folders |
| Session identity | Keeps one session through resumes and moves | Paths and file contents change | Use the source name and source session ID |
| Child agents | Keeps child-agent history linked to its session | Agent work is part of the exact record | Store each child as a linked stream |
| Git observation | Records a worktree before it disappears | Project identity may be lost later | Run short local Git commands during the hook |
| Harness identity | Records the harness only when proven | Guessing would make analytics wrong | Require an explicit harness marker |
| Usage | Counts tokens once | Cost pages need correct inputs | Apply fixture-tested rules for each source |
| Pricing | Gives usage an API-equivalent value | Subscription sessions do not report actual spend | Use bundled prices or an explicit user refresh |
| Deletion | Erases a whole source session | Deleted history must not return | Remove every linked record and keep a minimal blocker |
| Status | Says whether capture is healthy | Silent gaps destroy trust | Report hook, refresh, format, and usage coverage honestly |

## Outcome

After Wave 1:

- one installed `qrt` binary can enable local Claude Code and Codex gathering;
- supported source hooks record small, local capture events without parsing or
  copying transcripts inside the source process;
- a non-resident one-shot reconciler preserves exact main and subagent
  transcript snapshots into owner-only content-addressed storage;
- new, resumed, moved, archived, and child-agent histories keep stable source
  identity without duplicate logical sessions or usage;
- capture-time repository/worktree facts survive deletion of an ephemeral
  working copy;
- Claude Code and Codex usage records are fixture-locked and reconcilable;
- source format drift, missing hooks, untrusted hooks, lag, failed copies, and
  unknown usage are visible failures rather than silent gaps;
- session-addressed erasure removes the accepted Wave 1 representations and
  prevents hooks, scans, or imports from resurrecting the session; and
- installed-artifact proof exercises both adapters in an isolated
  `QRATUM_HOME`.

## Explicit Non-Goals

Wave 1 does not build:

- the polished library or exact reader;
- lexical or semantic search;
- Project pages or cost dashboards;
- user-visible Workstreams, decisions, open loops, or roadmaps;
- source-context or personal-memory gathering;
- Claude.ai or ChatGPT import;
- external AI, transcript egress, or automatic background network calls;
- share-safe export;
- backup/restore product UI;
- a resident daemon;
- a new source adapter beyond Claude Code and Codex;
- Linux or Windows capture installation and scheduling; or
- any platform claim beyond macOS;
- a Qratum-operated shared pricing service.

## Evidence Snapshot

These facts informed the contract. They are evidence, not permanent vendor
guarantees.

### Installed sources checked on 2026-07-12

- Claude Code `2.1.207`.
- Codex CLI `0.144.1`.
- Codex reports `hooks` as a stable enabled feature.

### Current official hook surfaces

Claude Code documents `SessionStart`, turn-level `Stop`, `StopFailure`,
`SessionEnd`, `CwdChanged`, and `SubagentStop`. Hook input carries
`session_id`, `transcript_path`, and `cwd`; `SubagentStop` also carries
`agent_id`, `agent_type`, and `agent_transcript_path`.

Reference:

```txt
https://code.claude.com/docs/en/hooks
```

Codex documents `SessionStart`, turn-level `Stop`, and `SubagentStop`. Common
input carries `session_id`, optional `transcript_path`, `cwd`, and `model`;
`SubagentStop` also carries the child transcript path. Codex explicitly warns
that its transcript format is not a stable hook interface.

Reference:

```txt
https://learn.chatgpt.com/docs/hooks
https://learn.chatgpt.com/docs/config-file/config-reference
```

### Sanitized local shape probe

The review inspected keys, counts, and file sizes only, never transcript text:

```txt
Claude main JSONL files:       961
Claude subagent JSONL files:  2722
largest Claude JSONL:      9,816,802 bytes

Codex session JSONL files:    1245
Codex archived JSONL files:     24
largest Codex JSONL:      406,093,181 bytes
```

Claude assistant records expose stable message/session identifiers, per-message
model, and usage classes including input, output, cache creation, and cache
read. Codex rollouts expose `session_meta`, per-turn `turn_context`, and
`token_count` events containing both cumulative and last-turn usage.

The file sizes rule out copying the full source transcript synchronously on
every turn hook.

## Locked Architecture

Acceptance of this contract locks the following choices.

### 1. Hooks mark changes; the capture refresh reads files later

Every supported hook:

1. reads a bounded JSON payload from stdin;
2. validates required scalar fields and event type;
3. records a small owner-only event with an exclusive create;
4. attempts the bounded capture-time Git observation described below; and
5. exits.

The hook does not:

- read or copy transcript bytes;
- parse JSONL history;
- calculate a digest over the transcript;
- calculate usage or price;
- generate a report;
- start an LLM;
- access the network; or
- wait for the reconciler.

Hook stdout is empty. Hook code never emits decision-control JSON and never
intentionally blocks the source agent. Failure to spool is reported through the
source's normal non-blocking hook error path and later by `qrt doctor`.

### 2. Capture refresh is short, scheduled, and available on demand

Qratum does not install a resident daemon.

The same one-shot reconciler runs:

- every five minutes through a user-level OS schedule;
- before user-facing `qrt` commands that need current history or status;
- after `qrt init` installs capture; and
- when the later UI invokes “Check for new history.”

Hook and other hidden machine entrypoints never trigger reconciliation merely
because they started the `qrt` process.

Wave 1 supports only a macOS user LaunchAgent. Linux and Windows report
`unsupported`; they do not claim continuous capture. The schedule runs the
installed absolute `qrt` path and performs no network access.

The scheduler is a safety net and event consumer. Source hooks remain the
low-latency dirty-session signal.

### 3. Hook events are source-specific

Claude Code installs user-level command hooks for:

```txt
SessionStart
Stop
StopFailure
SessionEnd
CwdChanged
SubagentStop
```

Codex installs user-level command hooks for:

```txt
SessionStart
Stop
SubagentStop
```

Codex non-managed hook trust is not bypassed. `qrt init` may install the exact
configuration after confirmation, but source status remains
`installed_needs_trust` until the user accepts that exact hook definition in
Codex. A successful event from that definition is the evidence that upgrades
capture from configured to connected; Qratum does not invent a programmatic
Codex trust signal.

Claude `SessionEnd` records the terminal reason. Codex has no accepted
session-end event in this contract; a root session remains `available` with a
last-observed time rather than receiving an invented end time.

When a source resumes a session, its hook sends the same source session ID.
Qratum records another small dirty event; it does not create another logical
session. The capture refresh sees the longer stable transcript, saves a new
content revision, and adds only usage records whose source identities were not
already counted. `SessionStart` may say that the source resumed, but the stable
source session ID is the identity proof.

### 4. Qratum saves only stable snapshots

For each dirty or discovered transcript, the reconciler:

1. resolves the path under an allowlisted source root;
2. rejects traversal, symlinks, directories, devices, sockets, and other
   non-regular files;
3. checks the configured size limit and disk-free floor;
4. records pre-copy file identity, size, and modification time;
5. streams bytes once into an owner-only temp file while hashing;
6. records post-copy file identity, size, and modification time;
7. discards the temp file when the source changed during the copy;
8. atomically publishes the content-addressed blob only when stable; and
9. atomically records the source revision, observation, and usage projection.

No partial or truncated transcript is published as exact history. A changed
source is retried on a later one-shot run.

The largest transcript found in the sanitized 2026-07-12 local probe was
406,093,181 bytes, about 387 MiB. The default per-transcript hard limit is
1 GiB, about 2.6 times that observed maximum. This leaves room for longer
sessions while still stopping an unexpected file from consuming unbounded disk
and copy time. The user may explicitly raise the limit up to 16 GiB after
seeing the required free space and expected copy size. The 16 GiB ceiling is a
safety bound, not a claim that such a transcript is practical. Exceeding the
configured limit creates a visible failure; Qratum never silently truncates.

### 5. Source roots, not `cwd`, authorize transcript reads

`cwd` is untrusted metadata and is never an allowlisted raw-ingest root.

Accepted transcript roots are resolved from the source configuration and the
known local source homes:

```txt
Claude Code: source-configured projects root, normally ~/.claude/projects
Codex:       source-configured sessions and archived_sessions roots,
             normally ~/.codex/sessions and ~/.codex/archived_sessions
```

A hook payload path outside its source's accepted roots is rejected and
recorded. This intentionally supersedes the donor capture behavior that allows
the hook `cwd` as a transcript root.

### 6. Source identity is independent of path and digest

A logical session key is:

```txt
source + source_session_id
```

A child-agent stream key is:

```txt
source + source_session_id + source_agent_id
```

Moving a file from Codex `sessions` to `archived_sessions`, moving a working
copy, or observing a new digest never creates a second logical session.

A source revision key is:

```txt
logical_stream_key + exact_blob_digest
```

Re-observing the same digest is idempotent. A new digest appends one immutable
revision. Qratum does not derive session identity from a file path, timestamp,
repository, or digest.

### 7. Main and child-agent transcripts are first-class

Claude and Codex subagent transcripts are preserved as child streams linked to
their root source session and source agent identifier.

The root session owns display order and aggregate usage later. Wave 1 does
not flatten child-agent content into the main transcript or count the same usage
twice.

`SubagentStop` is the primary discovery signal for child transcripts. The
reconciliation scan is the fallback for children missed by hooks, nested
workflows, abrupt exits, older source versions, or hook disablement.
Fallback discovery must recover the root and child identities from a
fixture-locked source mapping or bounded identity probe; it never assigns a
parent from filename similarity or timing alone.

### 8. Capture-time Git observation is bounded and fail-soft

The hook records the source-reported `cwd` and attempts a local Git observation
before an ephemeral worktree disappears.

Qratum invokes the installed `git` binary with fixed arguments, no shell, no
network, and a hard timeout. It may record:

- repository root;
- Git common directory;
- branch or detached state;
- HEAD commit;
- normalized sanitized remote identity; and
- observation status and failure reason.

Remote credentials and user-info are removed before event storage. Failure,
timeout, non-Git directories, and missing Git record honest status without
preventing the hook event.

The hook never infers Project membership. Repository and Project projection
belongs to later waves.

### 9. Harness attribution requires an explicit marker

The capture event accepts harness name, version, run identity, and observation
time only when a source or harness explicitly supplies all required fields.

Repository path, prompt content, process ancestry observed later, or the
presence of Ductum files does not prove Ductum attribution. Until Ductum emits
the accepted marker, harness is `unknown`.

### 10. Each source has fixture-tested token-counting rules

#### Claude Code

- Usage is read from assistant-message usage fields.
- Model identity is per message, not per session.
- Stable usage identity uses source session, stream identity, message UUID, and
  source message/request identifier.
- Input, output, cache creation, cache read, service tier, and source-reported
  usage subfields remain separate.
- Duplicate message identity across main and child streams is counted once and
  reported as duplicate evidence.

#### Codex

- `token_count.info.last_token_usage` is the incremental usage record.
- `token_count.info.total_token_usage` is cumulative reconciliation evidence,
  never an additional usage record.
- Null `info` events contribute no usage and remain valid rate-limit/status
  events.
- Usage identity is derived from source session, stable event ordinal, and raw
  event digest because the source event has no usage ID.
- Usage attaches to the nearest preceding applicable `turn_context` and its
  model/turn identity.
- The sum of accepted incremental records must reconcile to the latest
  cumulative total for the same counter epoch.
- A counter reset begins a new epoch. A mismatch marks affected usage coverage
  incomplete; Qratum never guesses the missing delta.

Supported records default to `exact/source-reported`. Unknown source versions
or changed field semantics fail closed into unsupported coverage.

### 11. Pricing works offline and refreshes only when asked

**What:** Qratum ships with a known price catalog and lets the user explicitly
refresh it online.

**Why:** Bundled prices keep cost lookup working offline, while an explicit
refresh gets newer model prices without waiting for a Qratum release.

**How:** Wave 1 vendors a pinned snapshot of LiteLLM's
`model_prices_and_context_window.json`. The user may run:

```txt
qrt pricing refresh
```

That command makes only these allowlisted HTTPS reads:

```txt
https://api.github.com/repos/BerriAI/litellm/commits/main
https://raw.githubusercontent.com/BerriAI/litellm/<resolved-commit>/model_prices_and_context_window.json
```

Qratum validates the resolved commit as a full Git SHA, fetches the catalog by
that immutable commit, caps the response at 8 MiB, validates the expected JSON
shape and required price fields, shows the old and new catalog identities, and
publishes the new snapshot atomically. The catalog checked during this review
was 1,631,511 bytes, so the cap leaves about five times current headroom.
Redirects to another host, malformed data, invalid price values, or a failed
request leave the previous catalog active.

The request sends no transcript, session, usage, model, Project, repository,
machine, or credential data. Refresh is never automatic or started by capture,
reconciliation, status, or cost calculation. A file import remains available
for fully offline updates.

Every active catalog has a manifest containing:

- upstream repository and commit or release identity;
- snapshot digest;
- retrieved-at or imported-at date;
- retrieval method and allowlisted source URL when fetched;
- supported currency;
- per-model, per-token-class effective prices; and
- the Qratum transform version when normalization is required.

Normal runtime lookup never fetches the network. Unknown or missing model
pricing remains `unknown`, not zero.

Each bundled, fetched, or imported catalog remains an immutable version. A
later calculation records the exact catalog digest it used. Upstream LiteLLM
data does not by itself prove what a provider charged on an older session date,
so Qratum labels the result with the catalog date and never calls it historical
billed cost unless the source provides that evidence.

Wave 1 proves catalog integrity and lookup semantics. User-facing session
cost appears in Wave 2; Project accounting appears in Wave 2.5.

#### Future shared pricing service — not Wave 1

**What:** A later Qratum-operated endpoint can publish one versioned model-price
catalog that Qratum and the user's other projects can reuse.

**Why:** One maintained source avoids every project independently scraping and
interpreting provider pricing. It can also keep older catalog versions for
honest date-based calculations.

**How:** A separate contract will define an agent that checks allowlisted
official provider pricing pages or APIs, produces a proposed change with source
links and observation dates, and runs deterministic schema, range, and change
checks. A changed price or new model does not publish merely because an LLM
said it was correct. Until the future service proves safe automated approval,
those changes require explicit review. Published catalogs are immutable,
versioned, integrity-protected, and contain no user data. Clients still fetch
only after an explicit refresh request.

### 12. Deleting a source session prevents it from returning

Deletion identity is `source + source_session_id`, not a raw-ref identifier.

A session tombstone retains only the minimum terminal audit identity:

- schema version;
- source;
- a non-reversible keyed digest of the source session identity;
- erased time;
- removal status/counts.

The deletion operation removes every Wave 1 representation for the session:

- all main and child raw blobs that have no other live reference;
- all raw refs and source revision mappings;
- source observations and exact paths;
- normalized source/session metadata;
- usage records and calculated values;
- pending and retained diagnostic capture events;
- dirty-state and scan-checkpoint entries; and
- temporary files attributable to the session.

Hooks, scheduled reconciliation, manual reconciliation, discovery scans, and
future imports check the tombstone before copying, parsing full content, or
publishing. When source identity is not available from the event or
fixture-locked path mapping, the adapter may perform only its bounded identity
probe under an accepted source root before the tombstone check. A later source
file with the erased identity is counted as suppressed, not recaptured.

Deletion of one session never deletes a shared blob still referenced by a
different live session. The current donor `EraseRawRef` behavior is insufficient
because it is raw-ref addressed and leaves representations outside the blob.

## Capture Cadence And User Truth

The accepted mechanism is hooks plus a five-minute one-shot schedule, not a
resident service.

The status model reports per source:

```txt
unsupported
not_configured
installed_needs_trust
connected
degraded
failed
```

It also reports:

- installed source version;
- configured hook events;
- hook trust state when the source exposes it;
- last hook event time;
- pending event count;
- last reconciliation start/end/result;
- oldest pending event age;
- last successful snapshot time;
- discovered root and child stream counts;
- supported/unsupported source-record counts;
- usage-record coverage and reconciliation state;
- schedule installed/active state; and
- explicit warnings for gather lag, rejected paths, changed-during-copy,
  oversize files, disk pressure, format drift, and suppressed erased sessions.

An all-day active session may be behind by the hook-to-reconcile lag. The status
surface says so; it never labels the corpus live-complete without evidence.

## Hook Performance Contract

The hook path is measured independently from reconciliation.

- stdin is capped at 1 MiB;
- one hook writes at most one small event plus bounded Git observation;
- no transcript file is opened by hook code;
- no network-capable package may be imported by the capture package;
- event files and directories are owner-only;
- concurrent hooks cannot select the same event path;
- a hook timeout/failure cannot corrupt an existing event; and
- installed-artifact fixtures measure p95 hook wall time at or below 250 ms on
  the supported CI runner, with a hard one-second Git-observation timeout.

Performance failure blocks the source-correctness claim for that platform.

## Reconciliation State Machine

The internal per-stream state is:

```txt
observed
dirty
copying
preserved
retryable_failure
unsupported
erased
```

This is operational state, not primary user vocabulary.

Rules:

- one workspace lock serializes publication and deletion;
- events are processed idempotently in stable event order;
- multiple dirty events for the same stream coalesce to the newest observation;
- failure leaves the prior preserved revision readable;
- retryable failures retain bounded diagnostic evidence and retry metadata;
- unsupported format does not fall back to best-effort parsing;
- successful publication checkpoints the event atomically; and
- crash recovery removes stale temp files without deleting live blobs.

Successfully checkpointed hook events are deleted. Only bounded diagnostic
evidence for failures may be retained, and it remains session-addressable for
terminal deletion.

## Stored Contracts

Implementation must add fixture-backed schemas with these exact responsibilities.
Field names may change during review, but no schema may absorb another concern
without updating this contract.

### `qratum.capture_event.v2`

Owner-only hook observation:

- event identity and source event type;
- source and source version;
- root session identity;
- optional child agent identity/type;
- transcript kind and source path;
- cwd, model, event time, and observed time;
- bounded Git observation;
- optional explicit harness marker; and
- schema/data-class/provenance fields.

No transcript content or prompt text is stored in the event.
Source version comes from Qratum's last cached source inventory; the hook never
launches the source CLI to discover a version.

### `qratum.session_revision.v1`

Mapping from one logical root/child stream and digest to:

- raw ref;
- revision number;
- source path observation;
- stable pre/post-copy file facts;
- observed/preserved times;
- source adapter and parser version; and
- superseded/latest state.

### `qratum.usage_record.v1`

One accepted incremental usage record:

- source/session/stream identity;
- stable or derived source event identity;
- turn/message identity;
- model/provider;
- separate token classes;
- timestamp and time basis;
- source semantics and reliability class;
- raw evidence revision/digest; and
- duplicate/reconciliation status.

### `qratum.session_tombstone.v1`

Terminal session deletion record containing only the minimal fields named in
the deletion section.

### `qratum.capture_state.v1`

Per-source and per-stream liveness, dirty, retry, scan checkpoint, schedule,
coverage, and failure state used by doctor and reconciliation.

### `qratum.price_catalog_manifest.v1`

Pinned catalog provenance, resolved upstream commit, retrieval method/source,
digest, currency, effective-date, and transform metadata. The bundled snapshot
and each explicitly refreshed snapshot are immutable inputs, never silently
rewritten runtime state.

Existing v1 event, raw-ref, and raw-tombstone schemas remain historical
compatibility contracts for v0.1.0. Wave 1 does not silently reinterpret
their fields.

## On-Disk Layout

The accepted logical layout under `QRATUM_HOME` is:

```txt
events/pending/
raw/blobs/sha256/
raw/blobs/.tmp/
raw/refs/
sessions/revisions/
sessions/tombstones/
usage/records/
state/capture.json
catalog/model-prices/
```

Exact filenames and sharding belong to implementation, but all directories are
owner-only and recoverable without a database. SQLite remains out of scope for
Wave 1.

## Initialization And Internal Surfaces

### Public/headless bootstrap

`qrt init` is the accepted headless bootstrap path. For Wave 1 it:

1. resolves and secures `QRATUM_HOME`;
2. inventories Claude Code and Codex source roots and installed versions;
3. shows the exact user-level hook and OS-schedule plan;
4. obtains explicit confirmation before modifying source settings or schedules;
5. writes atomically without discarding unrelated source configuration;
6. reports Codex hook trust as a separate required user action;
7. runs initial reconciliation; and
8. prints truthful source and coverage status.

`qrt init --yes` is the explicit automation equivalent. It prints the same plan
and changes only the paths/configuration named by that plan.

### Public recovery

`qrt doctor` reports the status contract above and gives one concrete recovery
action per failure. `qrt doctor --json` exposes the same facts through a strict
versioned DTO.

### Public pricing update

`qrt pricing refresh` performs the explicit allowlisted online refresh defined
above. `qrt pricing refresh --file <path>` performs the same validation and
atomic update from a local file without network access. Neither command changes
session history or usage records; it changes only the catalog used for later
API-equivalent calculations.

### Hidden machine entrypoints

The following are allowed but omitted from public help:

```txt
qrt hook claude-code
qrt hook codex
qrt internal reconcile --once
```

They are source/scheduler integration points, not user workflows. No other
legacy pipeline commands are retained merely for debugging.

### Self-contained proof

`qrt trust --json` may expose the Wave 1 proof only when it executes its
isolated fixtures and installed binary. It cannot mark a source live-connected
from configuration inspection alone.

## Failure And Recovery Contract

| Failure | Required behavior | Recovery |
| --- | --- | --- |
| Hook not installed | Source is `not_configured`; no connected claim | Re-run confirmed source setup |
| Codex hook untrusted | `installed_needs_trust`; no connected claim | Review exact hook in Codex `/hooks` |
| Hook event write fails | Source continues; failure is visible | Repair `QRATUM_HOME`, then reconcile |
| Transcript path rejected | No read outside source root | Correct source config or adapter |
| Source changes during copy | Publish nothing; retain prior revision | Retry next one-shot run |
| Transcript exceeds limit | No truncation; explicit failure | Raise limit explicitly or reduce source file |
| Disk floor reached | No partial publication | Free space, then retry |
| Unknown source record | Preserve exact bytes; mark affected parsing/usage coverage unsupported without guessing | Add reviewed fixture/parser version |
| Usage reconciliation mismatch | Exact transcript remains; usage coverage incomplete | Inspect source-version adapter |
| Schedule missing/inactive | Degraded continuous-capture status | Reinstall/enable schedule |
| Source file moved to archive | Same session identity; update path observation | Reconciler discovers accepted archive root |
| Session tombstoned | Suppress every recapture path | No automatic recovery; explicit future policy required |
| Crash during publication | Prior revision remains; temp removed later | Next run recovers and retries |

## Security And Privacy Invariants

- Raw transcripts, paths, repository identity, usage, and hook events are
  owner-only local data.
- Hook and scanner inputs are untrusted and fail closed.
- Transcript authorization is an allowlist of source roots.
- All path checks occur after clean/absolute resolution and without following
  symlinks.
- All transcript reads require a regular file and a size limit.
- Copies stream; no large transcript is loaded wholly into memory.
- Atomic publication uses owner-only temp files in the destination filesystem.
- A failed copy never replaces a good revision.
- Remote URLs are sanitized before event storage.
- Hooks and reconciliation make no network calls.
- Pricing refresh is the only Wave 1 runtime network path. It runs only from
  an explicit user command, uses the two allowlisted HTTPS endpoints above, and
  is isolated from transcript, usage, Project, repository, and capture code.
- Pricing refresh sends no authorization header or user data, rejects
  cross-host redirects, caps response bytes, and keeps the prior catalog on
  every failure.
- Catalog update is bundled with a release, explicitly refreshed online, or
  explicitly imported from a file.
- Logs and diagnostics never include transcript text, prompts, credentials, or
  unsanitized remotes.
- Test fixtures contain synthetic values only.
- Tombstones are checked before full source parsing, copying, or publication;
  only the bounded identity probe defined above may precede the check.

## Required Fixture Matrix

### Claude Code adapter

- SessionStart new and resume with stable session identity;
- Stop and SessionEnd for one main transcript;
- StopFailure without invented success/end state;
- CwdChanged across repositories;
- SubagentStop with child identity and child transcript;
- multiple subagents with distinct streams;
- resumed main transcript with one old and one new message;
- model switch within one session;
- input/output/cache creation/cache read usage;
- duplicate message evidence across streams;
- malformed JSONL line;
- unknown record type;
- transcript deleted before reconciliation; and
- source version drift.

### Codex adapter

- SessionStart new and resume with stable source session identity;
- Stop with main transcript path;
- SubagentStop with child identity/path;
- `session_meta` Git/provider/source fields;
- per-turn model and cwd changes;
- token event with incremental and cumulative usage;
- null token info;
- cumulative counter reset and new epoch;
- incremental/cumulative mismatch;
- active-to-archived path move;
- parent thread and child-agent relationship;
- duplicate path observation;
- unknown event and payload type;
- missing transcript path; and
- source version drift.

### Shared hostile-input and storage fixtures

- empty, malformed, oversized, and wrong-typed hook payloads;
- absolute escape, `..` traversal, intermediate symlink, final symlink,
  directory, FIFO/device, and unsupported root;
- concurrent identical hook events;
- concurrent publication and deletion;
- file append/replace during copy;
- generated 1 GiB-boundary streaming input with an asserted peak-memory ceiling
  and no committed giant fixture;
- disk-floor failure;
- blob/ref collision;
- crash after temp write and before rename;
- owner-only permissions;
- session erasure with main and multiple children;
- erased session reappearing through hook, scan, archive move, and import; and
- shared blob referenced by one erased and one live session.

### Pricing fixtures

- bundled catalog digest and manifest match;
- the commit lookup returns one full Git SHA and the catalog request uses that
  immutable SHA;
- non-SHA commit values, non-allowlisted URLs, cross-host redirects, oversized
  responses, malformed JSON, wrong field types, and invalid negative prices
  fail closed;
- missing model entries remain `unknown`, not zero;
- no request contains transcript, session, usage, model, Project, repository,
  machine, cookie, or authorization data;
- a failed refresh leaves the prior catalog byte-for-byte active; and
- local-file refresh uses the same validation and atomic publication.

## Executable Acceptance

### Required repository checks

```txt
make test
make verify
make trust
```

The commands must use readonly Go module mode and the pinned supply-chain
configuration. No check may download runtime data or a pricing catalog.

### Installed-artifact flow

Against a built `qrt` and isolated homes:

1. create isolated `QRATUM_HOME`, Claude home, Codex home, schedule directory,
   and synthetic Git repositories/worktrees;
2. run `qrt init`, inspect the exact plan, confirm it, and verify unrelated
   source configuration remains byte-for-byte preserved;
3. invoke the installed Claude and Codex hook entrypoints with fixture payloads
   and assert only bounded owner-only events were written;
4. run the installed one-shot reconciler and verify exact blobs, revisions,
   root/child identity, Git observations, and usage records;
5. repeat all events and reconciliation and prove no duplicate sessions,
   revisions, or usage;
6. append/resume, move a Codex rollout to the archive root, and prove stable
   logical identity with a new revision only when bytes changed;
7. inject hostile paths, changed-during-copy, format drift, disk pressure, and
   usage mismatch and verify truthful degraded status;
8. erase one source session and prove every Wave 1 representation is removed
   while a shared blob remains for a live session;
9. rerun hook, scan, archive, and import paths and prove the erased session is
   suppressed; and
10. run `qrt doctor --json` and `qrt trust --json` and validate their strict
    schemas and executed evidence.

### Live owner proof before a source-connected release claim

For each supported source on macOS, an owner-run proof must record:

- installed source and `qrt` versions;
- hook configuration and trust state without secret values;
- one new root session;
- one resumed turn;
- one child-agent session;
- observed hook and reconciliation times;
- resulting exact digests and usage reconciliation; and
- schedule execution after the source tool exits.

Fixture and unit-test success alone cannot upgrade a source from implemented to
live-connected.

Before online pricing refresh is described as available, one owner-run proof
must also record the resolved LiteLLM commit, fetched catalog digest, prior and
new manifest identities, and successful offline lookup after the refresh. It
must confirm that a failed refresh leaves the prior catalog active.

## Donor-Code Disposition

The current published baseline and local candidate are donor material only.

### Reuse after contract-aligned changes

- owner-only `QRATUM_HOME` creation;
- content-addressed blob storage;
- streaming hash/copy and atomic temp-rename patterns;
- no-follow, regular-file, disk-floor, and free-space guards;
- exclusive-create event spool;
- flock-serialized state mutation;
- Claude settings merge/diff discipline;
- launchd one-shot schedule generation;
- schema registry and fixture/golden discipline; and
- raw backup consent and integrity helpers for later lifecycle work.

### Reshape

- Claude-only event schema becomes source-neutral v2;
- SessionEnd-only capture becomes multi-event dirty signaling;
- hook-time transcript copy moves to reconciliation;
- six-hour `vault backfill` schedule becomes five-minute source reconciliation;
- path-or-digest session identity becomes source identity plus revisions;
- `cwd` is removed from transcript-root authorization;
- 50 MiB archive cap becomes the explicit 1 GiB transcript limit with streaming
  and visible oversize failure;
- Claude-only hook install/status becomes separate Claude and Codex adapters;
- raw-ref erasure becomes terminal session-addressed erasure; and
- hard-coded trust states become executed source-specific status.

### Do not carry forward

- public `vault`, `daemon run-once`, or `backfill` vocabulary;
- full transcript parsing in hooks;
- direct transcript copies on every turn;
- path-based logical session identity;
- implicit Ductum attribution;
- best-effort parsing of unknown source versions; or
- claims that hook configuration inspection proves live capture.

## Implementation Slices After Acceptance

Implementation remains sequential and reviewable:

### T1.1 — Schemas, fixtures, and source probes

Add the stored schemas, source-version fixture corpora, strict parsers, and
negative format-drift tests. No hook installation yet.

### T1.2 — Source-neutral hook spool and Git observation

Add bounded Claude/Codex hook payload adapters, event v2, exclusive writes,
source-root metadata, Git timeout/sanitization, and hook performance tests.

### T1.3 — Reconciliation and exact revisions

Add dirty-event coalescing, initial scan, stable streaming copy, revisions,
root/child mapping, crash recovery, and five-minute macOS LaunchAgent schedule
generation.

### T1.4 — Usage and price catalog

Add Claude per-message usage, Codex delta/cumulative reconciliation, duplicate
handling, reliability/coverage, bundled catalog integrity, explicit allowlisted
online refresh, and local-file refresh.

### T1.5 — Terminal session deletion

Add session tombstones, representation enumeration, shared-blob safety,
recapture suppression, and concurrency tests.

### T1.6 — Init, doctor, trust, and installed proof

Wire confirmed source setup, Codex trust status, source diagnostics,
self-contained installed proof, and the owner-run live verification record.

No slice starts until the prior slice passes its accepted checks. Wave 2 does
not start until the complete Wave 1 installed proof is accepted.

## Review Decisions

Accepting this contract accepts these externally visible choices:

1. hooks plus a five-minute macOS one-shot schedule, with no resident daemon;
2. event-only hooks rather than hook-time transcript copies;
3. Claude and Codex hook events listed above;
4. source-root-only transcript authorization;
5. source session identity plus immutable content revisions;
6. first-class child-agent transcripts and usage;
7. bounded capture-time Git observation;
8. default 1 GiB transcript limit with explicit failure and optional increase;
9. Claude per-message and Codex delta/cumulative usage semantics;
10. bundled LiteLLM-style catalog input plus explicit allowlisted online or
    local-file refresh, with no silent update;
11. session-addressed terminal erasure; and
12. `qrt init`, `qrt doctor`, `qrt pricing refresh`, hidden hook/reconcile
    entrypoints, and executed `qrt trust` as the Wave 1 operator surfaces.

Rejecting or changing any item changes the contract and requires another review
before implementation.

## Acceptance Boundary

Acceptance authorizes implementation of Wave 1 only. It does not claim that
the behavior exists, that either source is live-connected, or that the product
is usable. Those claims require the installed and live proofs above.

# Qratum UI-First Onboarding

Status: superseded product contract; retained as candidate-runtime evidence

Date: 2026-06-17

Superseded: 2026-07-11 by `specs/current/product-direction.md`

Do not implement this document as the current product contract. Its UI-first
intent, DTO boundary principle, loopback-auth design, source-hook minimalism,
and explicit egress confirmation remain useful donor material. The following
sections are specifically superseded:

- `User Model` and `First-Run State Machine`;
- `Public Commands` and every command-specific product workflow;
- `Prepare` and the raw-queue interaction model;
- `Background Model` as a single assumed source mechanism;
- `AI Boundary` where it prohibits any local raw processing by accepted design;
- `UI DTO Contract` and its state/action enums;
- the prohibition on owner-only raw transcript viewing; and
- `Acceptance Criteria` as release authority.

The local shell described here exists only in a local candidate and was not
published as `v0.1.0`. No `SHIPPED` claim in this document applies to the
accepted product direction or to the current worktree without fresh proof.

## Decision

Qratum onboarding is UI-first. The CLI should get the user into the local app,
explain what exists, and provide recovery/status commands. The CLI should not
ask a new user to understand `vault backfill`, `daemon run-once`, or the old
Milestone A pipeline.

The product name is always Qratum. `9473` is only the default local app port.

Default local app address:

```txt
http://127.0.0.1:9473
```

## User Model

The first user question is:

```txt
where are my past AI coding sessions, and can I safely view them?
```

Qratum answers with four plain states:

- **Found**: Qratum detected local source files. This is read-only inventory.
- **Preserved**: Qratum copied raw, unredacted source bytes into the vault.
- **Prepared**: Qratum produced browseable redacted artifacts for selected
  sessions.
- **Viewable**: the local app can show safe DTOs, review cards, reports, and
  artifact links without exposing raw transcript text.

Do not expose "backfill" or "dogfood" as first-run product nouns.

## First-Run State Machine

The onboarding state machine is:

```txt
uninitialized -> found -> preserved -> prepared -> open
```

State meanings:

- **uninitialized**: no usable Qratum workspace exists yet, or the workspace has
  not completed first-run discovery.
- **found**: Qratum has completed read-only inventory of supported local
  sources. Nothing has been copied or prepared yet.
- **preserved**: Qratum has copied supported local raw session bytes into the
  content-addressed vault and recorded raw refs. The data is durable but still
  raw and unredacted.
- **prepared**: Qratum has prepared a bounded selected set, starting with latest
  10, into browseable artifacts under `QRATUM_HOME/sessions`.
- **open**: the local Qratum app is running on `127.0.0.1:9473` and can show
  DTO-backed session views, raw queue metadata, status, doctor output, and safe
  next actions. The app can open in any state; if nothing is prepared yet, it
  shows found/preserved counts, the raw queue, and preparation actions instead
  of pretending sessions are viewable.

Transient sub-states are allowed for honest UI/status reporting:

- **finding**: read-only inventory is running.
- **preserving**: raw bytes are being copied, hashed, deduped, and recorded.
- **preparing**: a selected deterministic preparation pass is running.
- **failed**: the latest action failed and needs doctor/recovery detail.

These sub-states do not add new product steps. They let `qrt open` show the real
queue instead of blocking until work finishes.

Allowed transitions:

```txt
qrt init
  uninitialized -> found -> preserved -> prepared

qrt open
  any state -> open

qrt import <file-or-folder>
  uninitialized/found/preserved/prepared -> found/preserved/prepared,
  depending on the confirmed import plan

qrt status
  read only; no transition

qrt doctor
  read only; no transition
```

Invalid transitions:

- `qrt open` must not preserve, prepare, call AI, export, publish, or delete.
- `qrt status` and `qrt doctor` must not start work.
- Background preservation must not move sessions to `prepared`.
- No command may jump from `found` to `open` while pretending sessions are
  viewable.

## Public Commands

The first-run public command set is deliberately small:

```txt
qrt init
qrt open
qrt status
qrt doctor
qrt import <file-or-folder>
```

Library and egress commands are public too, because users need to list, inspect,
and export their data explicitly:

```txt
qrt sessions
qrt session <session_id>
qrt export
```

Command name decision:

- Use `qrt open`, not `qrt ui`, for the app launch command. `open` says the
  action in plain language. `ui` is an implementation noun and conflicts with
  the old `qrt ui ... --json` DTO command that this redesign removes.
- Use `qrt sessions` to list sessions and `qrt session <session_id>` to open or
  show one session. These replace the old `sessions list` shape.
- Use `qrt export` for the explicit export planner. Export is allowed, but it is
  a named egress boundary and must show what could leave Qratum before writing
  or sending anything.

Delete the old public surface as this onboarding work lands. Do not keep hidden
legacy command paths just to preserve Milestone A behavior.

Commands or command shapes to remove from the public runtime:

- `hook`
- `vault`
- `daemon`
- `dogfood`
- `normalize`
- `redact`
- `evidence`
- `review`
- `report`
- old `export <session> --profile ...`
- old `sessions list [--repo ...]`
- old `ui ... --json`

The underlying implementation pieces may move behind package-level use cases
where needed, but the `qrt` user surface should not retain hidden aliases.

## `qrt init`

`qrt init` bootstraps Qratum and tells the user what it found. It is allowed to
perform the first preservation and first preparation after showing an explicit
plan.

Required sequence:

1. Create or verify `QRATUM_HOME` with owner-only permissions.
2. Discover supported local sources without rendering transcript contents.
3. Show source counts, byte totals, oldest/newest timestamps when available,
   and unsupported/gap notes.
4. Show exactly what will happen before writes:
   - preserve all discovered supported local sessions into the raw vault
   - prepare the latest 10 supported sessions for viewing by default
   - offer "prepare all" only after showing time/disk/token estimates and
     receiving explicit confirmation
   - leave background preservation off unless the user enables it
   - make no upload, publish, external AI, or raw transcript preview
5. Ask for confirmation unless an explicit automation flag is supplied.
6. Run preservation idempotently.
7. Run bounded preparation for the latest 10 supported sessions.
8. Print the local app command and status command.

Example shape:

```txt
qrt init
home: /Users/example/.qratum

found:
  claude-code local transcripts: 184 sessions, 92 MB
  cloud/web sessions: not visible to local Qratum

will do:
  preserve 184 local sessions as raw, unredacted vault copies
  prepare latest 10 sessions for viewing as a fast first sample

will not do:
  upload data
  call external AI
  show raw transcript text
  prepare all history without a separate estimate and confirmation
  enable background capture without approval

continue? [y/N]
```

Completion output:

```txt
preserved: 184 raw local copies, 0 deduped, 0 failed
prepared: 10 viewable, 0 failed
remaining: 174 preserved raw sessions not prepared yet

why latest 10:
  Qratum preserved everything. It prepares a small first set so you can inspect
  the result quickly before spending time, disk, or optional AI budget on the
  whole archive.

next:
  qrt open
  qrt status
```

## `qrt open`

`qrt open` starts the local Qratum app and opens the browser:

```txt
qratum app
url: http://127.0.0.1:9473
```

The command name is intentionally `qrt open`, not `qrt ui`. A new user is trying
to open the product, not choose an implementation surface.

It may create lightweight app auth state, but it must not import, preserve,
prepare, review, export, publish, or call AI automatically.

Local app auth contract:

- bind only to `127.0.0.1` in the first implementation
- create a one-time bootstrap nonce when `qrt open` starts the app
- exchange the nonce for an HttpOnly SameSite cookie
- redirect to a clean URL without the nonce
- store only a token hash under `QRATUM_HOME/state/app_auth.json` with
  owner-only permissions
- reject unauthenticated app/API requests, including requests from other local
  browser contexts
- add no CORS wildcard and no LAN bind in the first implementation

First screen requirements:

- show found, preserved, and prepared counts
- show whether capture/background preservation is enabled
- show the latest processing job state
- show the trust/redaction boundary in plain language
- show supported sources and current source gaps
- show "Prepare latest 10" or "Prepare selected" when raw preserved sessions
  exist but nothing is viewable yet
- show no raw transcript preview in the first implementation

## `qrt status`

`qrt status` is the small current-state view. It should be safe to run often and
must not start work.

Example shape:

```txt
qratum status
home: /Users/example/.qratum
library: 184 found, 184 preserved, 10 prepared
processing: idle
capture: off
background_preserve: off
last_preserve: 2026-06-17T22:10:14Z
last_prepare: 2026-06-17T22:11:02Z
warnings: redaction is best-effort alpha; cloud/web sessions require qrt import of a vendor export

open: qrt open
details: qrt doctor
```

## `qrt doctor`

`qrt doctor` is the deeper diagnostic view. It replaces `qrt vault doctor` as
the public command. Remove the old `qrt vault doctor` entrypoint when this lands.

It should include:

- workspace permissions
- hook/capture state
- background preservation mechanism, such as source hook, OS schedule, or later
  local service
- raw vault counts and drift warnings
- prepared-session counts
- backup verification state
- trust-gate verdict when available
- recent failures and recovery commands

## `qrt import <file-or-folder>`

`qrt import` is for data outside the normal source paths. It accepts a single
file or a folder and produces an explicit import plan before heavy work.

Rules:

- folder imports inventory first
- unsupported files are counted and shown
- raw archive and preparation are explicit plan actions
- import does not call AI or publish data by default
- re-running the same import is idempotent

Cloud/web sessions enter Qratum through import, not through local capture. If a
source tool only has cloud/web history, the user must export it from that tool
and run `qrt import <file-or-folder>`.

## `qrt sessions` / `qrt session <session_id>`

`qrt sessions` lists safe session metadata:

- prepared sessions first
- preserved-but-not-prepared sessions as raw queue metadata
- source, date, repo/workspace when known, status, and next action
- no raw transcript text

`qrt session <session_id>` opens the local app on that session if possible. In
non-browser contexts it may print a compact safe summary and the `qrt open`
URL. If the session is only preserved raw, it shows metadata and the prepare
action; it must not dump raw transcript text.

## `qrt export`

`qrt export` is explicit egress. It is allowed because users need the big export
path, but it must be treated as a boundary where data can leave Qratum.

Export requirements:

- start with an export plan, not a blind write
- show selected scope, destination, data classes, item counts, and file size
  estimate
- say whether raw, redacted, review, corpus, or AI-derived data is included
- default to no raw transcript export
- require confirmation before writing outside `QRATUM_HOME` or sending anywhere
- record export provenance and the exact data classes exported
- repeat the redaction warning at the export boundary: deterministic redaction is
  best-effort alpha, not a privacy or PII guarantee

## Prepare

"Prepare" is the user-facing word for the deterministic viewing pipeline:

```txt
normalize -> redact credentials -> evidence -> review card -> report/export DTOs
```

The report/export outputs here are safe DTOs for the app and explicit export
planner. They are not the removed Milestone A public `report` or old
`export <session> --profile ...` commands.

Preparation is selected and bounded. First run prepares the latest 10 sessions
so the user sees value quickly without preparing every raw transcript.

Preparation must be able to read preserved raw refs. This is the main bridge
missing from the shipped v0.1.0 runtime: today raw preservation and prepared
session artifacts are separate paths.

Implementation requirement:

- preparation reads from raw refs and vault blobs, not only from live source
  transcript paths
- prepared artifacts land under `QRATUM_HOME/sessions/<session_id>/...`
- deleting or losing the original source transcript must not prevent preparation
  from a preserved raw ref
- re-running preparation for the same raw digest is idempotent

## Background Model

Background work is preserve-only until explicitly expanded. It is off by
default unless `qrt init` asks and the user enables it.

MVP implementation decision:

- Prefer an OS schedule or source hook for preservation freshness.
- Do not require a resident always-running daemon for the first implementation.
- If a small local service is introduced later, it must expose the same status,
  enable, disable, and doctor behavior.

Enablement surfaces:

- `qrt init` asks whether to enable background preservation.
- the local app Settings screen can enable or disable it later.
- `qrt status` shows whether it is enabled and what it last did.
- `qrt doctor` explains the exact mechanism, such as launchd schedule, source
  hook, or local service.

The config file records the decision, but editing config by hand is not the
primary enablement path. A user should be able to enable or disable background
preservation through `qrt init` or the local app.

Allowed background behavior:

- detect new local supported transcripts
- copy raw bytes into the vault
- hash, dedupe, and record refs
- report failures

Not allowed by default:

- prepare all history
- render raw transcript content
- call external AI
- export or publish
- delete/purge

Background work must never be silent in the product sense. If the user enables
it, Qratum records that decision, shows it in status, and limits the work to raw
preservation. It does not silently prepare, export, upload, call AI, or delete.

## AI Boundary

AI is not required for first-run onboarding. The first `prepare` pass is
deterministic.

What deterministic prepare gives without AI:

- normalized session metadata
- credential redaction pass
- evidence facts from tools, commands, files, and verification signals
- review card and findings from deterministic rules
- safe reports/DTOs for the local app

What optional AI can add later:

- human-readable title and summary
- goal, outcome, friction, and usefulness labels
- reusable lesson candidates
- search/curation tags
- cross-session insight drafts

Token and cost rule:

- before any AI run, Qratum must show an AI plan with estimated input tokens,
  estimated output tokens, provider, model, local/external status, and estimated
  external cost when the provider price is known
- if provider pricing is unknown, Qratum must say cost is unknown instead of
  guessing
- one session's token use is roughly the selected transcript/review content plus
  summarization instructions; Qratum should count or estimate from the actual
  selected bytes before asking for approval
- external AI is opt-in; local AI still needs a visible action because it can be
  slow and resource-intensive
- external AI may receive only prepared/redacted artifacts, never raw transcript
  bytes

Future AI summaries, facets, lessons, or insights must be separate actions with
explicit local/external provider disclosure and consent.

Example AI plan wording:

```txt
selected: 1 prepared session
input: about 24k tokens from redacted review/evidence artifacts
output: about 2k tokens
model: configured by the user
cost: computed from the configured provider's current input/output token prices;
      if Qratum does not know the provider price, show "cost unknown"
will produce: title, short summary, goal/outcome/friction labels, and lesson
              candidates
will not send: raw transcript bytes
```

## UI DTO Contract

The local app should receive clean DTOs. It must not parse raw transcripts,
ADP JSONL, redaction internals, or provenance internals.

User-facing DTOs must use closed enums where possible:

- `source.status`: `ready`, `gap`, `needs_setup`, `error`, `unsupported`
- `session.status`: `prepared`, `preserved_raw`, `preparing`, `failed`
- `processing.status`: `idle`, `finding`, `preserving`, `preparing`,
  `importing`, `exporting`, `failed`
- `job.status`: `queued`, `running`, `succeeded`, `failed`, `cancelled`
- `trust.verdict`: `TRUSTED`, `TRUSTED-WITH-NAMED-GAPS`, `UNTRUSTED`

In UI/status copy, "processing" means a currently running job. The user-facing
verb for making sessions browseable is still "prepare."

Initial onboarding status shape:

```json
{
  "schema_version": "qratum.ui.onboarding_status.v1",
  "home": "/Users/example/.qratum",
  "app_url": "http://127.0.0.1:9473",
  "sources": [
    {
      "source": "claude-code",
      "status": "ready",
      "found": 184,
      "preserved": 184,
      "prepared": 10,
      "bytes": 92000000,
      "oldest_at": "2026-05-19T09:30:00Z",
      "newest_at": "2026-06-17T22:05:00Z"
    }
  ],
  "processing": {
    "status": "idle",
    "latest_job": null
  },
  "capture": {
    "enabled": false,
    "background_preserve_enabled": false,
    "background_mechanism": null,
    "last_preserve_at": "2026-06-17T22:10:14Z",
    "last_prepare_at": "2026-06-17T22:11:02Z"
  },
  "trust": {
    "verdict": "TRUSTED-WITH-NAMED-GAPS",
    "warnings": [
      "redaction is best-effort alpha",
      "known redaction gaps: => assignment edge, selected git/time/source fields, SSH-style git remotes",
      "cloud/web sessions require qrt import of a vendor export"
    ]
  },
  "actions": [
    "prepare_latest_10",
    "prepare_all_after_estimate",
    "enable_background_preserve",
    "sessions",
    "session_detail",
    "import_path",
    "export",
    "doctor"
  ]
}
```

Job DTO:

```json
{
  "schema_version": "qratum.ui.job.v1",
  "job_id": "job_20260617_221102_prepare_latest",
  "kind": "prepare",
  "status": "running",
  "started_at": "2026-06-17T22:11:02Z",
  "total": 10,
  "done": 4,
  "failed": 0,
  "current_session_id": "sess_abc123",
  "failure_ref": null
}
```

Session list DTO:

```json
{
  "schema_version": "qratum.ui.session_list.v1",
  "items": [
    {
      "session_id": "sess_abc123",
      "source": "claude-code",
      "occurred_at": "2026-06-17T21:44:00Z",
      "repo_or_workspace": "/repo/qratum",
      "status": "prepared",
      "summary": "Fix redaction bug",
      "next_action": "open"
    },
    {
      "session_id": "sess_def456",
      "source": "claude-code",
      "occurred_at": "2026-06-16T18:12:00Z",
      "repo_or_workspace": "/repo/qratum",
      "status": "preserved_raw",
      "summary": null,
      "next_action": "prepare_selected"
    }
  ],
  "pagination": {
    "limit": 50,
    "next_cursor": null
  }
}
```

Session detail DTO:

```json
{
  "schema_version": "qratum.ui.session_detail.v1",
  "session_id": "sess_abc123",
  "source": "claude-code",
  "status": "prepared",
  "occurred_at": "2026-06-17T21:44:00Z",
  "repo_or_workspace": "/repo/qratum",
  "title": "Fix redaction bug",
  "safe_summary": "Prepared deterministic review card and evidence.",
  "artifacts": [
    {
      "kind": "review_card",
      "href": "/api/sessions/sess_abc123/review-card"
    }
  ],
  "raw_preview_available": false,
  "next_actions": ["open_export_plan"]
}
```

Import plan DTO:

```json
{
  "schema_version": "qratum.ui.import_plan.v1",
  "source_path": "/Users/example/Downloads/vendor-export",
  "supported_files": 42,
  "unsupported_files": 3,
  "estimated_bytes": 12000000,
  "will_preserve_raw": true,
  "will_prepare": false,
  "requires_confirmation": true
}
```

Export plan DTO:

```json
{
  "schema_version": "qratum.ui.export_plan.v1",
  "scope": "prepared_sessions",
  "destination": "/Users/example/Desktop/qratum-export",
  "data_classes": ["redacted", "review"],
  "item_count": 10,
  "estimated_bytes": 2500000,
  "includes_raw": false,
  "redaction_warning": "best-effort alpha; not a PII guarantee",
  "requires_confirmation": true
}
```

AI plan DTO:

```json
{
  "schema_version": "qratum.ui.ai_plan.v1",
  "scope": "selected_sessions",
  "provider": "local",
  "model": "example-local-model",
  "external": false,
  "estimated_input_tokens": 24000,
  "estimated_output_tokens": 2000,
  "estimated_external_cost": null,
  "cost_known": true,
  "input_data_classes": ["redacted", "review"],
  "includes_raw": false,
  "requires_confirmation": true
}
```

Action contract:

| Action | Required params | Response DTO | Confirmation |
| --- | --- | --- | --- |
| `prepare_latest_10` | none | `qratum.ui.job.v1` | no, if already confirmed by `qrt init`; otherwise yes |
| `prepare_selected` | `session_ids[]` | `qratum.ui.job.v1` | yes |
| `prepare_all_after_estimate` | none | prepare estimate, then `qratum.ui.job.v1` | yes, after estimate |
| `session_detail` | `session_id` | `qratum.ui.session_detail.v1` | no |
| `import_path` | `path` | `qratum.ui.import_plan.v1` | yes, before writes |
| `export` | `scope`, `destination` | `qratum.ui.export_plan.v1` | yes, before egress |
| `ai_plan` | `scope`, `provider`, `model` | `qratum.ui.ai_plan.v1` | yes, before AI |
| `doctor` | none | doctor DTO or text report | no; read-only |

## Acceptance Criteria

- `qrt init` is understandable without knowing the words hook, vault, daemon,
  backfill, or dogfood.
- `qrt open` opens Qratum at `127.0.0.1:9473`.
- unauthenticated requests to the local app/API are rejected.
- Qratum branding is consistent everywhere; no alternate product name appears.
- `qrt status` gives a short summary; `qrt doctor` gives deep diagnostics.
- The public `qrt` help shows only the new first-run command set plus any
  explicitly accepted future commands.
- Old public commands listed in this spec are removed, not hidden.
- `qrt sessions` and `qrt session <session_id>` provide safe list/detail access
  without raw transcript dumps.
- `qrt export` exists as an explicit egress planner and requires confirmation
  before writing or sending data outside Qratum.
- UI list/detail/job/import/export/AI actions use DTOs; the app never parses raw
  transcripts, ADP JSONL, or redaction internals.
- Preserved raw sessions can be prepared from the vault without the original
  source file still existing.
- Preparation and import are idempotent for the same raw digest/input path.
- If there are zero prepared sessions and some preserved raw sessions, the app
  shows a raw queue and the "Prepare latest 10" action.
- The app can open before preparation finishes and must show the real state.
- Background preservation is only enabled after explicit user approval and is
  visible in status/doctor output.
- Background preservation never prepares all history, renders raw transcript
  content, calls AI, exports, publishes, uploads, deletes, or purges by default.
- Optional AI shows token/cost estimates, model/provider, local/external status,
  and cost unknown when pricing is unavailable before running.
- Optional external AI receives only prepared/redacted artifacts, never raw
  transcript bytes.
- Raw transcript text is not shown in the first implementation.
- No external AI, upload, publish, or full-history preparation happens during
  onboarding unless the user explicitly approves it.

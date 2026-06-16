# P3 — Schemas & Contracts (D9 + data_class + config + scorecard + receipt)

Package: `qratum-verify-trust-gate` · Prompt 3 of 7 · Depends on: P1 ·
Scope: schemas/contracts · Deliverable: complete, strict, self-tested schemas
for every emitted object + a stdlib mini-validator + the `data_class` field +
the new config, trust-scorecard, and memory-import-receipt schemas.

## Objective

Make every object qratum emits describable by a strict schema, validate it with
a tiny stdlib-only validator, and add the one field (`data_class`) the boundary
rule needs to mean anything. Today the session schema describes a different
object than the code actually writes (about 12 emitted fields are undeclared),
so `additionalProperties: false` is unsatisfiable and nothing is really checked.
This task fixes that — full field parity, recursive strictness on every nested
container, a validator that proves it can reject a planted extra key, and an
enumeration that proves every emitted object maps to a real schema.

This is the D9 dimension of the trust gate plus FIX-13 (the `data_class`
field). It is the contract foundation that later tasks (the crown-jewel leak
gate, the boundary checker, the scorecard emitter, the dream-tier receipt
round-trip) all build on. Build only the schema/contract layer here — no
runtime rewiring, no new commands, no leak-harness.

## Read first

- `qratum/specs/current/verification-and-trust-gate.md` — the contract this
  implements. Especially: §3 "D9 — Schema / contract conformance"; FIX-13
  (`data_class`); the "Schema completeness is an explicit deliverable" language;
  the scorecard-as-governed-object block (`qratum.trust_scorecard.v1`); the D10
  dream-tier `memory_import_receipt` paragraph; §5 (config schema is a missing
  P0 deliverable); §6 non-goals (stdlib mini-validator only; a third-party
  validator is an explicit supply-chain decision).
- `qratum/docs/reviews/2026-06-15-verification-benchmark/GAPS.md` — the
  line-level evidence. Especially: the "schema / contract completeness" theme
  (the ~12 undeclared session fields; bare `{"type":"array"}` containers with no
  teeth; the no-migration story; `LoadState` silently back-filling
  `SchemaVersion`); M8 (`data_class` has zero hits in the codebase); the
  `memory_import_receipt` "shipped kind with no schema, `--kind` defaults to
  `source_metadata`" footgun.
- `qratum/AGENTS.md` — supply-chain rule, no-Python single-Go-binary rule,
  fixture/golden contract, current milestone (P2-VERIFY-TRUST-GATE is PROPOSED,
  awaiting the maintainer), Definition of Done.
- `qratum/docs/supply-chain.md`.
- `qratum/schemas/*.json` — the current schemas. Note exactly what is wrong:
  `qratum-session.v1.schema.json` declares 8 fields and no
  `additionalProperties`; `qratum-raw-ref.v1.schema.json` and
  `qratum-provenance.v1.schema.json` likewise have no `additionalProperties`
  and use bare `{"type":"object"}` for `digests`/`local_only`.
- The emitting code, to read the real field set off the structs (do NOT
  enumerate fields from memory — read the tags): `cmd/qrt/normalize.go`
  (`qratumSession` and its nested `turns`/`tool_calls`/`git`/`workspace`),
  `cmd/qrt/redact.go`, `cmd/qrt/evidence.go`, `cmd/qrt/review` types,
  `cmd/qrt/export.go` (the ADP wrapper + `adpStrictSchemaVersion`),
  `cmd/qrt/ui.go` (the UI DTO), `internal/vault/vault.go` (raw_ref, state,
  capture event), `schemas/qratum-event.v1.schema.json`.

## Allowed scope

- Edit every file under `qratum/schemas/` (and add new ones). Bring each emitted
  object's schema to **full field parity** with the struct that produces it.
- Add `additionalProperties: false` (or an explicit forbidden-key list where a
  closed enum is wrong) to **every** schema, **recursively** — every nested
  container (`turns[]`, `tool_calls[]` and its `input`, `git`, `workspace`,
  `raw`, `digests`, `provenance`, …) gets its own `properties`/`items` +
  `additionalProperties: false`.
- Add the required `data_class` field (FIX-13) to every emitted object schema,
  as an enum over a committed monotonic lattice.
- Add three new schemas: `qratum-config.v1.schema.json` (the missing config
  schema), `qratum-trust-scorecard.v1.schema.json`
  (`qratum.trust_scorecard.v1`), and `qratum-memory-import-receipt.v1.schema.json`
  (the closed-vocabulary receipt schema D7/D10 needs).
- A new `internal/schema` package: a stdlib-only mini-validator
  (`const`/`enum`/`required`/`type`/recursive `properties`/`items`/
  `additionalProperties:false`) plus an emitted-object registry mapping each
  `schema_version` literal to its schema file.
- Fixtures + golden tests for all of the above under `fixtures/schema/` (or the
  existing per-area fixture dirs), honoring `QRATUM_HOME`.
- A short schema-migration note (a reader contract: no silent default of a
  missing/unknown version).

## Non-goals

- No new third-party Go dependency. The validator is **stdlib-only**
  (`encoding/json` + hand-rolled checks). Reaching for a JSON-Schema library is
  an explicit supply-chain decision — STOP and report it to the maintainer instead of
  adding it (§6 names a vetted ≥7-day-old pure-Go validator as a *possible*
  later call, sign-off only).
- No `$ref` resolver, no remote-schema fetch, no full JSON-Schema draft
  coverage — only the keyword subset above, enough to make
  `additionalProperties:false` real and recursive.
- No runtime rewiring: do not touch the daemon's blob-read path (FIX-3), the
  redactor's logic, the leak harness, or any command's behavior. This task
  changes schemas, adds one carried field, and adds a validator. If wiring the
  validator into a command surfaces a real emitted field the schema can't yet
  describe, fix the schema — do not change the emitted object's shape to dodge
  it (that is a behavior change owned by another task).
- No `data_class` *enforcement* logic (the "no transform raises the class"
  runtime check lives in the boundary task). Here `data_class` is a declared,
  required, defaulted-on-emit field with a committed lattice + a lineage *rule
  written down* and a self-test that the lattice is monotonic — not a runtime
  transform guard.
- Do NOT build the personal-memory gateway or any receipt *producer*. qratum
  imports `memory_import_receipt` objects; it does not create them. Only the
  consumer-side schema lands here.
- No SQLite, no migration *runtime* (`qrt migrate`) — only the reader contract
  that a missing/unknown version fails loud.

## Implementation notes

### Schema completeness (full field parity — the actual deliverable)

For each emitted object, read the producing struct and declare **every** field
it can emit, with its type:

- `qratum.session.v1` — add the ~12 currently-undeclared fields
  (`transcript_path`, `source_event_id`, `git`, `pipeline_status`,
  `artifact_paths`, `business_metrics`, `provenance`, `agent_model`, …; confirm
  the full set from the struct tags, do not trust this list). Give `git`,
  `workspace`, each `turns[]` item, each `tool_calls[]` item (including its
  `input` object), and `provenance` their own nested `properties` +
  `additionalProperties:false`. Reference the existing
  `qratum-provenance.v1.schema.json` shape but make it a typed sub-schema, not
  a bare `{"type":"object"}`.
- `qratum.raw_ref.v1` — already close; add `additionalProperties:false`,
  `data_class`, and confirm the `kind` enum still includes
  `memory_import_receipt`.
- `qratum.evidence.v1`, `qratum.review_card.v1`, the ADP wrapper, the UI DTO —
  bring each to parity and strict.
- `qratum.event.v1` — the capture event; declare `raw.*`, `copy_status`, etc.;
  strict.

Add a **self-test** per emitted struct: derive the struct-tag set by reflection
(`reflect` over the Go struct's json tags) and assert it **equals** the
schema's declared property-set. This is the parity guard — it fails loudly when
a future field is added to the struct but not the schema (and vice-versa).
Recurse into nested structs/maps so the parity check covers `turns[]` items,
`tool_calls[].input`, `git`, `workspace`, `provenance`, not just the top level.

### Recursive strictness (where the leaks actually live)

The audited leaks live inside nested containers. A bare `{"type":"array"}` lets
a leaking key inside `turns[]` or `tool_calls[].input` pass. So:

- Every array gets typed `items` with their own
  `properties`+`additionalProperties:false`.
- Every object/map gets explicit `properties`+`additionalProperties:false`,
  **except** genuinely open user-content maps (`tool_calls[].input`,
  `provenance.digests`) — for those, document why they are open and pin the
  *drift direction* (`emitted-keys ⊆ schema-declared-keys`) so an unexpected
  *new* key is still caught at the level where it can be.
- Pin drift direction explicitly: the validator's parity self-test asserts
  emitted-keys is a subset of schema-declared-keys for closed objects.

### The `data_class` field (FIX-13)

- Add a **required** `data_class` string field to every *emitted object*
  schema: `raw_ref`, `session`, `review_card`, `evidence`, the ADP wrapper, the
  UI DTO, the capture event, and the trust scorecard.
- It is an `enum` over the committed, one-direction-only lattice:
  `raw > redacted > review > corpus > published`.
- Commit the lattice once, in a single place (a `const`/ordered slice in
  `internal/schema` plus a short table in the migration note), and write the
  **lineage rule** in prose: a transform may keep or lower the class, never
  raise it, except via a single named downgrade step. Add a self-test asserting
  the committed lattice is strictly ordered (monotonic) and that the schema
  enums match the lattice members exactly.
- This task only declares + defaults the field and proves the lattice is
  well-formed. The runtime "no transform raises the class" enforcement is the
  boundary task's job — say so plainly so the next agent doesn't assume it's
  done.

### The stdlib mini-validator (`internal/schema`)

- Pure stdlib: parse the schema JSON and the instance JSON with
  `encoding/json`, walk them together. Support exactly: `type`, `required`,
  `const`, `enum`, `properties`, `items`, `additionalProperties:false`, and
  recursion into nested `properties`/`items`.
- **Reject-self-test (mandatory):** take a known-good instance of every schema,
  inject one extra key (top-level **and** inside a nested container like
  `turns[0]` and `tool_calls[0].input` where the object is closed), and assert
  the validator **REJECTS** it. A validator that accepts an injected extra key
  is a silent leak channel — this test is the proof the strictness has teeth.
- Validate on the comma-ok / typed-error path, never `panic`, so a malformed
  instance fails **closed** and loud (matches the redactor's fail-closed
  intent).

### Emitted-object enumeration + name-mapping assertion

- Build a registry: every `schema_version` literal the code can emit →
  its schema file. Include the schemaless surfaces too — the ADP export
  (`adpStrictSchemaVersion`) and the **redaction summary** object — by giving
  each a schema and a registry entry (the D9 denominator = *emitted objects*,
  not just the ones that already had a `schema_version`).
- Assert the **hyphen/dot name mapping**: the dotted `schema_version` literal
  (`qratum.session.v1`) maps to the hyphenated filename
  (`qratum-session.v1.schema.json`). A mismatch must fail the test, not
  silently skip a schema.
- A **missing schema → loud fail**: if the registry has a `schema_version` with
  no file (or a file with no registry entry), the test fails. No silent gaps.

### New schemas

- `qratum-config.v1.schema.json` — the missing P0 config schema. Declare every
  config key the binary reads (read them off the config struct;
  `disk_free_min_gb` is one — declare it whether or not the runtime wires it,
  that wiring is FIX-14's job in another task). Strict.
- `qratum-trust-scorecard.v1.schema.json` (`qratum.trust_scorecard.v1`) — the
  scorecard is itself a governed object. Declare: the headline enum
  (`TRUSTED` / `TRUSTED-WITH-NAMED-GAPS` / `NOT-TRUSTED`), the per-dimension
  PASS/FAIL/KNOWN-RED states, `gap_count`, the honest-residual block, and a
  `provenance` sub-block (build commit, corpus digest, schema digest,
  timestamp). Required `data_class: published`. The scorecard emitter (another
  task) will validate its own output against this and run the leak checker over
  its bytes — so the schema must be real now.
- `qratum-memory-import-receipt.v1.schema.json` — the consumer-side receipt
  schema D7/D10 needs (and which P7's round-trip depends on). It must mirror the
  **real gateway vocabulary** as **closed enums**: a closed `outcome` enum and a
  closed `errorClass` enum (pull the exact members from the gateway's vocabulary
  — do NOT invent values; if the canonical list isn't recorded anywhere
  readable, STOP and report rather than guessing). Include `supersedes[]`,
  `namespace`, `contentClass`. Strict, with `data_class`. Wire it into D9's
  registry. Note in the migration note that `qrt vault archive --kind
  memory_import_receipt` defaulting to `source_metadata` is the documented
  footgun this schema closes; pinning the kind is P7's job.

### Schema-migration story (reader contract)

- Write a short note (in the schema dir or the migration note doc): readers
  must **not** silently default a missing or unknown `schema_version`. A blank
  or unrecognized version is a loud error, not a back-fill. (GAPS flags
  `LoadState` silently back-filling `SchemaVersion` — call that out as the
  anti-pattern this contract forbids; the actual fix to `LoadState` is a
  runtime change owned elsewhere, but the contract is stated here.)
- State the immutability consequence plainly: content-addressed blobs are never
  re-normalized in place; a v1→v2 derivation regenerates from the blob. No v2
  field is added here.

## Acceptance criteria

(from `verification-and-trust-gate.md` §3 "D9" + FIX-13 + §5 + §7)

- Every `schemas/*.json` carries `additionalProperties:false` (or an explicit
  forbidden-key list), **recursively** — nested `turns[]`, `tool_calls[].input`,
  `raw`, `git`, `workspace`, `provenance` each have their own
  `properties`/`items` + `additionalProperties:false`.
- **Full field parity:** every field the code emits is declared, with its type;
  `provenance` has a typed sub-schema; a self-test asserts the struct-tag set
  **equals** the schema property-set (recursively), so a drift in either
  direction fails loudly.
- The stdlib mini-validator validates a good instance of every schema and
  **REJECTS** an instance with an injected extra key (top-level and nested).
- Every emitted object maps to a schema: the enumeration covers every
  `schema_version` literal **plus** the schemaless ADP export and the redaction
  summary; the hyphen/dot name-mapping is asserted; a missing schema fails loud.
- `data_class` is a required enum on every emitted object schema, over the
  committed monotonic lattice (`raw > redacted > review > corpus > published`),
  with a self-test that the lattice is strictly ordered and the enums match.
- The three new schemas exist and validate sample instances:
  `qratum-config.v1.schema.json`, `qratum.trust_scorecard.v1`, and
  `qratum-memory-import-receipt.v1.schema.json` (closed `outcome`/`errorClass`
  enums mirroring the real gateway vocabulary), the receipt wired into the D9
  registry.
- The migration note states the reader contract: no silent default of a
  missing/unknown `schema_version`.
- `make verify` is green and no new third-party Go dependency was added.

## Decision Trace

- Schema completeness is an **explicit deliverable**, not an assumption
  (`verification-and-trust-gate.md` §3 D9; GAPS "the session schema describes a
  different object than the code emits").
- `data_class` (FIX-13) is required on every emitted object over a committed
  monotonic lattice (M8: zero `data_class` hits in the shipped code today).
- Config schema is a named **missing P0 deliverable** (`§5`,
  `AGENTS.md` Current Milestone: "Two P0 gaps remain: the missing config schema
  and the unvalidated `schemas/`").
- `memory_import_receipt` is an **in-scope committed contract** this milestone
  (Q3 = BOTH; dream tier in scope, gated). The schema lands now; the behavioral
  round-trip is P7, gated on the gateway.
- `qratum.trust_scorecard.v1` is agreed (Q4) and wired into D9 as a governed
  object.
- Stdlib mini-validator only (§6 non-goal); a third-party validator is a
  supply-chain decision requiring the maintainer's sign-off.

## Behavior Contract

- [ ] FAILS the task: adding any new third-party Go dependency without an
  explicit supply-chain decision (stdlib-only validator); evidence:
  `make supply-chain` / `make verify`.
- [ ] FAILS: any schema left with a bare `{"type":"array"}` /
  `{"type":"object"}` for a *closed* container (no `items`/`properties`,
  no `additionalProperties:false`); evidence: the recursive-strictness test.
- [ ] FAILS: the parity self-test passing while the struct-tag set and the
  schema property-set differ in either direction.
- [ ] FAILS: the validator accepting an instance with an injected extra key
  (top-level or nested) — the reject-self-test must go RED on that injection.
- [ ] FAILS: any emitted `schema_version` literal (or the ADP / redaction
  summary) with no schema in the registry, or a hyphen/dot name mismatch that
  is silently skipped.
- [ ] FAILS: a missing/unknown `schema_version` being silently defaulted rather
  than failing loud (reader contract).
- [ ] FAILS review: changing an emitted object's runtime shape to make a schema
  pass (that is a behavior change owned by another task — fix the schema, not
  the object), or inventing `outcome`/`errorClass` receipt values not in the
  real gateway vocabulary.

## Drift Handling

- If reading the structs reveals an emitted field the spec/GAPS did not name,
  declare it in the schema and note it — do not drop it from the object. If the
  field looks like it should not be emitted at all, STOP and report (that is a
  redaction/boundary decision, not a schema decision).
- If the real gateway `outcome`/`errorClass` vocabulary is not recorded anywhere
  readable in the repo, STOP and report rather than guessing the enum members.
- If an existing golden file would change because a schema now validates it
  differently, update the golden only when the output contract intentionally
  changed, and say so explicitly.

## Verification

```sh
# Full local CI mirror (build, vet, lint, test, race, demo, dogfood, security):
make -C . verify

# The schema/validator tests specifically (in an isolated workspace):
export QRATUM_HOME="$(mktemp -d)"
go -C . test ./internal/schema/... -run 'Schema|Validator|Parity|DataClass|Registry|Receipt|Scorecard|Config' -v
unset QRATUM_HOME

# Prove the validator REJECTS an injected extra key (the strictness has teeth):
go -C . test ./internal/schema/... -run 'RejectExtraKey' -v

# Prove the struct-tag set == schema-property-set for every emitted object:
go -C . test ./internal/schema/... -run 'Parity' -v

# Prove every emitted schema_version (incl. ADP + redaction summary) maps to a
# schema, with the hyphen/dot name assertion and missing-schema loud-fail:
go -C . test ./internal/schema/... -run 'Registry|NameMapping|MissingSchema' -v
```

VERIFY GAP: confirm the package name/path the harness expects for the validator
(`internal/schema` is the proposed location; the existing `internal/` dirs are
`claude`, `textdiff`, `vault`, `workspace`). Confirm the exact `schema_version`
literals the code emits today by grepping the structs before writing the
registry — do not enumerate them from this spec. Confirm where (if anywhere) the
real `memory_import_receipt` gateway `outcome`/`errorClass` vocabulary is
recorded before authoring those enums.

## Slop Review

- [ ] Recursive strictness is real: no closed container left as a bare
  `{"type":"array"}` / `{"type":"object"}`; the leaks' nesting (`turns[]`,
  `tool_calls[].input`, `raw`, `git`, `workspace`, `provenance`) each have
  their own `properties`/`items` + `additionalProperties:false`.
- [ ] The parity self-test actually compares the **reflected struct-tag set** to
  the **schema property-set** (not a hand-maintained list that can rot), and
  recurses into nested objects.
- [ ] The validator reject-self-test demonstrably goes RED on an injected extra
  key — not a test that only ever exercises the happy path.
- [ ] `data_class` is required on every emitted object and the lattice
  self-test proves strict monotonic ordering; the lattice is committed in one
  place, not duplicated.
- [ ] The emitted-object enumeration includes the ADP + redaction summary, the
  hyphen/dot mapping is asserted, and a missing schema fails loud.
- [ ] The receipt `outcome`/`errorClass` enums are closed and mirror the real
  gateway vocabulary — not invented placeholders.
- [ ] Stdlib-only; no new third-party Go dependency; tests honor `QRATUM_HOME`
  and never touch the real `~/.claude` or `~/.qratum`.
- [ ] No emitted object's runtime shape was changed to dodge a schema; no
  runtime behavior (daemon, redactor, commands) was altered.

Reviewer guidance:

> Review this schema/contract work against `verification-and-trust-gate.md` §3
> (D9) + FIX-13 + §5 + §7. Confirm: every emitted object has a strict,
> recursively-`additionalProperties:false` schema at full field parity with its
> producing struct (with a reflection-based parity self-test); the stdlib
> mini-validator rejects an injected extra key (top-level and nested); every
> emitted `schema_version` (plus the ADP export and the redaction summary) maps
> to a schema with a hyphen/dot name assertion and a missing-schema loud-fail;
> `data_class` is required on every emitted object over a committed strictly-
> monotonic lattice (`raw > redacted > review > corpus > published`); the config,
> `qratum.trust_scorecard.v1`, and `memory_import_receipt` schemas exist, the
> receipt has closed `outcome`/`errorClass` enums mirroring the real gateway
> vocabulary, and all three are wired into the registry; the migration note
> forbids silently defaulting a missing/unknown version. Flag: any bare-container
> schema, any invented receipt vocabulary value, any new third-party dependency,
> any runtime/object-shape change made to pass a schema, or any test that mutates
> the real home dir.

## Stop conditions

- STOP if the P2-VERIFY-TRUST-GATE milestone is still PROPOSED and the maintainer has
  not explicitly unlocked it — this task is gated. (`AGENTS.md` Current
  Milestone: "Do not implement P2-or-later runtime behavior unless the user
  explicitly accepts the proposed milestone.")
- STOP if P1 has not landed (the contract/spec hygiene this depends on).
- STOP and report as a supply-chain decision if the stdlib mini-validator
  becomes a false economy and a third-party validator looks warranted — do not
  add the dependency yourself.
- STOP if the real `memory_import_receipt` gateway `outcome`/`errorClass`
  vocabulary cannot be found in the repo — report rather than invent enum values.
- STOP if bringing a schema to parity would require changing the emitted
  object's runtime shape — that is a behavior change owned by another task;
  report it.
- STOP if `make verify` cannot be made green without weakening a check — report
  the failure, do not suppress it.

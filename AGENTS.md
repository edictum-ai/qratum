# Qratum agent instructions

## Source of truth

Follow SPEC.md first. SPEC.md defines the executable scope for the current
milestone.

Files under docs/architecture/ are forward-design references only. They are not
permission to implement future features.

Files under docs/decisions/ are accepted architectural decisions. Schemas under
schemas/ are contracts. Fixtures under fixtures/ are part of the test contract.

## Current milestone

Milestone A only.

Build the first local vertical slice:

Claude Code hook fixture -> CaptureEvent -> filesystem spool -> daemon run-once
-> QratumSession -> deterministic redaction -> EvidenceBundle -> ReviewCard ->
UI DTOs -> HTML report -> ADP strict export.

## Non-goals

Do not implement:

- enterprise server
- marketplace
- Codex adapter
- OpenCode adapter
- Copilot adapter
- MCP server
- GitHub App
- GitHub comments
- GitLab
- LLM scoring
- LLM redaction
- web UI
- HTTP server
- bbolt
- SQLite
- Postgres
- encrypted vault
- Edictum integration

## Runtime

Use Go. No Python runtime. No database for Milestone A. Use filesystem JSON
storage.

## Hook rule

`qrt hook claude-code` must be fast. It only reads JSON from stdin, writes a
CaptureEvent, and exits.

No LLM calls. No full transcript parsing. No report generation. No network
calls.

## Data rule

Use transcript_path from the hook payload. Do not hardcode Claude local
transcript paths.

Do not send raw transcript to any external service. Do not render raw transcript
into HTML.

## UI contract rule

The UI consumes UI DTOs, not raw internal models.

Backend must not require the UI to parse:

- Claude transcript JSONL
- raw QratumSession internals
- ADP JSONL
- redaction internals
- provenance internals

## Testing

Every behavior must be fixture-driven where practical.

Update fixtures and golden files when output contracts intentionally change.

`make test` must run all tests. `make demo` must run the first vertical slice.

## Ductum factory rules

When a task is dispatched to you via the Ductum factory:

- You are running inside an isolated git worktree. Make your changes on the current feature branch.
- **Do not run `git push`.** The factory's post-completion pipeline handles verify, review, and merge after you call `ductum_complete`.
- After you call `ductum_complete(result=...)`, stop making tool calls. Your session will end and the factory will take over.
- The workflow has three stages: `understand` (read context), `implement` (write code), `ship` (factory-owned). Work only in `implement` for code changes.
- Required verify command is in `.edictum/workflow-profile.yaml`. It will be run automatically; if it fails, a fix-loop task will be dispatched with the failure output.

## Definition of done

A task is done only when:

1. code builds
2. tests pass
3. fixture/golden outputs are updated
4. `make demo` still works if affected
5. no non-goal feature was implemented

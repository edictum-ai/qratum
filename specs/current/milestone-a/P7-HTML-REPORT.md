# P7 — HTML Report

## Scope

Render a safe static HTML report from session artifacts.

## Decision Trace

- ADR 0006: review not score.
- ADR 0007: local-first raw storage.
- docs/architecture/security-model.md.

## Behavior Contract

- CLI runtime must fail visibly when required input is missing or invalid.
- Output schema evidence must preserve session IDs, artifact paths, and deterministic fixture timestamps.
- Missing artifacts must reject the run or demo instead of being silently swallowed.
- Verification output must be operator-visible when behavior fails.
- Invalid config or input must refuse processing with an error.
- Runtime resolution logic must remain scoped to the current project and session.
- Evidence paths must round-trip through generated artifacts.
- Session state must preserve source IDs instead of silently inventing replacements.
- Runtime behavior must be deterministic under fixture inputs.
- Missing or invalid files must fail loudly with an operator-visible message.
- Output must preserve explicit evidence for every generated review or report.
- Schema output must reject unsupported values rather than silently accepting drift.

- Report is static escaped HTML.
- No JavaScript, no external assets, no raw transcript rendering, and no secret
  maps.
- Report includes review, evidence, missing evidence, redaction summary,
  artifact links, and provenance digests.

## Deliverables

- `qrt report <session>`.
- Escaped HTML.
- Sections: session summary, review card, evidence findings, missing evidence,
  redaction summary, artifacts, provenance digests.
- No JavaScript and no external assets.

## Non-goals

No raw transcript rendering, secret maps, web app, or external CDN.

## Verification

```sh
./bin/qrt report .qratum/sessions/ses_0001.normalized.json
test -f .qratum/reports/ses_0001.html
go test ./...
```

## Drift Handling

If richer report interactivity is requested, defer it to web UI work outside
Milestone A.

## Slop Review

- Require behavioral tests for missing or invalid inputs.
- Attack swallowed failures, missing explicit evidence, duplicate resolution logic, dead config, and future features.
- Attack behavior contract drift where runtime output no longer matches fixture evidence.

- Attack XSS risk from transcript-derived content.
- Test that report output contains escaped content and no fixture secrets.

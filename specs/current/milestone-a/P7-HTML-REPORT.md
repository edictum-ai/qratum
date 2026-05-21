# P7 — HTML Report

## Scope

Render a safe static HTML report from session artifacts.

## Decision Trace

- ADR 0006: review not score.
- ADR 0007: local-first raw storage.
- docs/architecture/security-model.md.

## Behavior Contract

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

- Attack XSS risk from transcript-derived content.
- Test that report output contains escaped content and no fixture secrets.

# P7 — HTML Report

## Scope

Render a safe static HTML report from session artifacts.

## Deliverables

- `qrt report <session>`.
- Escaped HTML.
- Sections: session summary, review card, evidence findings, missing evidence,
  redaction summary, artifacts, provenance digests.
- No JavaScript and no external assets.

## Non-goals

No raw transcript rendering, secret maps, web app, or external CDN.

## Acceptance

```sh
./bin/qrt report .qratum/sessions/ses_0001.normalized.json
test -f .qratum/reports/ses_0001.html
go test ./...
```

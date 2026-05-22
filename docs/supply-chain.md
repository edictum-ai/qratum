# Supply-Chain Policy

Qratum is a local-first tool that handles AI coding transcripts and redacted
artifacts, so the build pipeline must avoid ambient trust.

## Rules

- Pin GitHub Actions by full commit SHA, not tags.
- Use explicit workflow permissions.
- Set `persist-credentials: false` on checkout steps.
- Use `GOTOOLCHAIN=local` in CI so Go does not fetch a different toolchain at
  runtime.
- Use `GOFLAGS=-mod=readonly` in CI so commands cannot silently edit module
  metadata.
- Run `go mod verify` before tests.
- Pin Go command-line tools to exact versions in the Makefile.
- Do not use pipe-to-shell installers in CI or scripts.
- Do not use floating tool versions in CI or scripts.
- Do not add npm, npx, pip, curl installers, or shell-fetched binaries to the
  Qratum runtime pipeline.
- New runtime dependencies need a short rationale in the PR description.

## Local Checks

Run the same checks locally:

```sh
make verify
```

The `make supply-chain` target rejects common unsafe patterns before they land
in the repository.

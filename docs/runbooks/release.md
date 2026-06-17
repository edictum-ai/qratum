# Release runbook — `qrt` distribution

Qratum ships as a single Go binary via **GitHub Releases** (cross-platform
tarballs + checksums) and a **Homebrew formula** in the edictum org tap.

Install (end users):

```sh
brew tap edictum-ai/edictum
brew install qratum          # installs the `qrt` binary
qrt --version
```

Raw binaries are also attached to each GitHub Release for non-brew users.

## One-time setup

1. **Create the tap repo** `edictum-ai/homebrew-edictum` (public). Homebrew maps
   `edictum-ai/edictum` → the repo `edictum-ai/homebrew-edictum`. It needs a
   `Formula/` directory (goreleaser commits `Formula/qratum.rb` automatically).
2. **Create a tap-write token.** A fine-grained PAT (or deploy key) with
   `contents:write` on `edictum-ai/homebrew-edictum` only. Add it to the
   **qratum** repo as the secret `HOMEBREW_TAP_TOKEN`. (The release's own
   GitHub release upload uses the built-in `GITHUB_TOKEN`; the separate token is
   only because the formula lands in a *different* repo.)

## Cut a release

```sh
# from a clean main that already passed CI
git tag v0.1.0
git push origin v0.1.0
```

The tag triggers `.github/workflows/release.yml`:

- **gate** — `go build`, `go test -race`, supply-chain policy.
- **release** — goreleaser builds darwin/linux/windows × amd64/arm64, publishes
  the GitHub Release with `checksums.txt`, and pushes `Formula/qratum.rb` to the
  tap. The binary's `--version` is stamped from the tag via
  `-ldflags -X main.version`.

## Dry run (no publish, no token)

Validate the whole pipeline locally before tagging:

```sh
goreleaser check                          # lint .goreleaser.yaml
goreleaser release --snapshot --clean     # build all targets + formula, no push
sh scripts/homebrew-smoke.sh              # run the host binary: --version + status
```

PRs that touch the release plumbing (`.goreleaser.yaml`, `release.yml`,
`homebrew-smoke.sh`, `cmd/qrt/**`) run the same snapshot + smoke in CI via the
`dry-run` job, so config breakage is caught before a tag is cut.

## Notes

- `qrt` is pure-stdlib Go, built `CGO_ENABLED=0` → fully static binaries.
- The tap can also host other edictum CLIs later (e.g. `edictum-go`) under the
  same `brew tap edictum-ai/edictum`.

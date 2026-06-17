#!/usr/bin/env sh
# Smoke-test a goreleaser-built qrt binary for the host platform: confirm the
# released artifact actually runs (--version, status) without needing a
# published Homebrew tap. Used as a release dry-run gate.
#
# Usage:
#   goreleaser release --snapshot --clean   # produces dist/
#   sh scripts/homebrew-smoke.sh            # discovers + smokes the host binary
#   sh scripts/homebrew-smoke.sh path/to/qrt
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${1:-}"

if [ -z "$BIN" ]; then
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	esac
	BIN="$(find "$ROOT/dist" -type f -name qrt 2>/dev/null | grep -E "${os}_${arch}" | head -1 || true)"
fi

if [ -z "$BIN" ] || [ ! -x "$BIN" ]; then
	echo "homebrew-smoke: no runnable qrt binary found (build first with: goreleaser release --snapshot --clean)" >&2
	exit 1
fi

echo "homebrew-smoke: using $BIN"

# 1. The binary reports a non-empty version (set via -ldflags at release time).
ver="$("$BIN" --version)"
echo "homebrew-smoke: $ver"
case "$ver" in
*dev*) echo "homebrew-smoke: warning: version is still 'dev' (ldflags not applied)" >&2 ;;
esac

# 2. A core command runs against an isolated, throwaway home (never the real one).
smoke_home="$(mktemp -d)"
trap 'rm -rf "$smoke_home"' EXIT
QRATUM_HOME="$smoke_home" "$BIN" status >/dev/null

echo "homebrew-smoke: ok"

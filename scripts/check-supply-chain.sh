#!/bin/sh
set -eu

fail=0

report() {
	echo "supply-chain error: $*" >&2
	fail=1
}

if [ ! -d .github/workflows ]; then
	report "missing .github/workflows"
else
	workflow_files="$(find .github/workflows -type f \( -name '*.yml' -o -name '*.yaml' \) | sort)"
	if [ -z "$workflow_files" ]; then
		report "missing workflow files"
	fi

	for file in $workflow_files; do
		if ! grep -Eq '^[[:space:]]*permissions:' "$file"; then
			report "$file is missing explicit permissions"
		fi
	done

	unpinned_actions="$(grep -RInE 'uses:[[:space:]]*[^[:space:]#]+@' .github/workflows 2>/dev/null | grep -vE '@[0-9a-f]{40}([[:space:]]|$|#)' || true)"
	if [ -n "$unpinned_actions" ]; then
		report "GitHub Actions must be pinned by 40-character commit SHA:"
		printf '%s\n' "$unpinned_actions" >&2
	fi

	checkout_without_persist_false="$(grep -RIn 'actions/checkout@' .github/workflows 2>/dev/null | while IFS=: read -r file _line _rest; do
		if ! sed -n "/actions\\/checkout@/,/^[[:space:]]*-[[:space:]]*uses:/p" "$file" | grep -q 'persist-credentials:[[:space:]]*false'; then
			printf '%s\n' "$file"
		fi
	done | sort -u)"
	if [ -n "$checkout_without_persist_false" ]; then
		report "actions/checkout steps must set persist-credentials: false:"
		printf '%s\n' "$checkout_without_persist_false" >&2
	fi
fi

scan_targets=""
if [ -d .github ]; then
	scan_targets="$scan_targets .github"
fi
if [ -d scripts ]; then
	for script in $(find scripts -type f ! -name check-supply-chain.sh | sort); do
		scan_targets="$scan_targets $script"
	done
fi
if [ -f Makefile ]; then
	scan_targets="$scan_targets Makefile"
fi
if [ -f go.mod ]; then
	scan_targets="$scan_targets go.mod"
fi
if [ -f go.sum ]; then
	scan_targets="$scan_targets go.sum"
fi

check_for_pattern() {
	label="$1"
	pattern="$2"

	if [ -z "$scan_targets" ]; then
		return
	fi

	matches="$(grep -RInE "$pattern" $scan_targets 2>/dev/null || true)"
	if [ -n "$matches" ]; then
		report "$label"
		printf '%s\n' "$matches" >&2
	fi
}

check_for_pattern "pipe-to-shell installs are not allowed" 'curl[^|]*\|[[:space:]]*(sh|bash)|wget[^|]*\|[[:space:]]*(sh|bash)'
check_for_pattern "floating tool versions are not allowed" '@latest'
check_for_pattern "npx is not allowed in CI or scripts" '(^|[^[:alnum:]_-])npx([[:space:]]|$)'
check_for_pattern "global npm installs are not allowed in CI or scripts" 'npm[[:space:]]+install[[:space:]]+-g'
check_for_pattern "pip installs are not part of the Qratum runtime pipeline" 'pip3?[[:space:]]+install'

if [ "$fail" -ne 0 ]; then
	exit 1
fi

echo "supply-chain checks passed"

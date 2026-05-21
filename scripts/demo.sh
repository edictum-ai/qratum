#!/bin/sh
set -eu

usage() {
	echo "usage: $0 [--verify-only] [./bin/qrt]" >&2
	exit 2
}

mode="run"
if [ "${1:-}" = "--verify-only" ]; then
	mode="verify"
	shift
fi

if [ "$mode" = "run" ]; then
	if [ "$#" -ne 1 ]; then
		usage
	fi
	bin="$1"
	if [ ! -x "$bin" ]; then
		echo "demo error: qrt binary is not executable: $bin" >&2
		exit 1
	fi
else
	if [ "$#" -ne 0 ]; then
		usage
	fi
fi

count_matching_files() {
	find "$1" -type f -name "$2" | wc -l | tr -d '[:space:]'
}

first_matching_file() {
	find "$1" -type f -name "$2" | sort | sed -n '1p'
}

require_demo_artifact() {
	label="$1"
	dir="$2"
	pattern="$3"

	if [ ! -d "$dir" ]; then
		echo "demo error: missing $label directory $dir" >&2
		exit 1
	fi

	count="$(count_matching_files "$dir" "$pattern")"
	if [ "$count" -eq 0 ]; then
		echo "demo error: missing $label artifact; expected one file matching $dir/$pattern" >&2
		exit 1
	fi
	if [ "$count" -ne 1 ]; then
		echo "demo error: expected exactly one $label artifact matching $dir/$pattern, found $count" >&2
		find "$dir" -type f -name "$pattern" | sort >&2
		exit 1
	fi

	path="$(first_matching_file "$dir" "$pattern")"
	if [ ! -s "$path" ]; then
		echo "demo error: empty $label artifact $path" >&2
		exit 1
	fi
}

verify_demo_artifacts() {
	require_demo_artifact "event" ".qratum/events" "*.json"
	require_demo_artifact "normalized session" ".qratum/sessions" "*.normalized.json"
	require_demo_artifact "redacted session" ".qratum/redacted" "*.redacted.json"
	require_demo_artifact "evidence" ".qratum/evidence" "*.evidence.json"
	require_demo_artifact "review" ".qratum/reviews" "*.review.json"
	require_demo_artifact "HTML report" ".qratum/reports" "*.html"
	require_demo_artifact "ADP strict export" ".qratum/exports" "*.adp.jsonl"
}

if [ "$mode" = "verify" ]; then
	verify_demo_artifacts
	exit 0
fi

echo "Running Milestone A demo with fixture input..."
rm -rf .qratum

cat fixtures/claude-code/hook-session-end.json | "$bin" hook claude-code
"$bin" daemon run-once

sessions_output="$("$bin" sessions list)"
printf '%s\n' "$sessions_output"
session_count="$(printf '%s\n' "$sessions_output" | sed '/^[[:space:]]*$/d' | wc -l | tr -d '[:space:]')"
if [ "$session_count" -ne 1 ]; then
	echo "demo error: expected exactly one generated session, found $session_count" >&2
	exit 1
fi

session_id="$(printf '%s\n' "$sessions_output" | awk 'NF {print $1; exit}')"
if [ -z "$session_id" ]; then
	echo "demo error: sessions list did not expose a session id" >&2
	exit 1
fi

verify_demo_artifacts

"$bin" ui sessions --json >/dev/null
"$bin" ui session "$session_id" --json >/dev/null
"$bin" ui review "$session_id" --json >/dev/null
echo "Verified UI DTOs for session $session_id"

echo "Generated artifacts:"
find .qratum -type f | sort

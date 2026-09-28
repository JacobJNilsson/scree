#!/bin/sh
# This script tests the coverage gate.
#
# No other check fails when the gate passes every profile. Each case feeds
# a synthetic profile to the gate and checks its exit code.
set -eu

gate="$(dirname "$0")/coverage-gate.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
fails=0

# expect DESC WANT MIN PROFILE EXCLUDES [ALLOW_EMPTY]
expect() {
	if COVERAGE_GATE_ALLOW_EMPTY="${6:-}" COVERAGE_MIN="$3" \
		sh "$gate" "$4" "$5" >/dev/null 2>&1; then
		got=0
	else
		got=1
	fi
	if [ "$got" -ne "$2" ]; then
		echo "coverage-gate-test: FAIL: $1 (exit $got, want $2)" >&2
		fails=$((fails + 1))
	fi
}

: >"$tmp/none.txt"

cat >"$tmp/covered.out" <<'EOF'
mode: atomic
example.com/m/a/a.go:1.1,2.2 3 1
example.com/m/b/b.go:1.1,2.2 2 5
EOF

# Three of five statements are covered, so coverage is 60%.
cat >"$tmp/partial.out" <<'EOF'
mode: atomic
example.com/m/a/a.go:1.1,2.2 3 1
example.com/m/b/b.go:1.1,2.2 2 0
EOF

printf 'mode: atomic\n' >"$tmp/header.out"

printf '# comment\n  b/b.go  # reason\n' >"$tmp/exclude-b.txt"
printf 'example.com/m/\n' >"$tmp/exclude-all.txt"

expect "full coverage passes at 100" 0 100 "$tmp/covered.out" "$tmp/none.txt"
expect "60% fails at 90" 1 90 "$tmp/partial.out" "$tmp/none.txt"
expect "60% passes at 50" 0 50 "$tmp/partial.out" "$tmp/none.txt"
expect "60% passes at 60" 0 60 "$tmp/partial.out" "$tmp/none.txt"
expect "60% fails at 61" 1 61 "$tmp/partial.out" "$tmp/none.txt"
expect "excluding the uncovered file passes" 0 90 "$tmp/partial.out" "$tmp/exclude-b.txt"
expect "a header-only profile fails" 1 90 "$tmp/header.out" "$tmp/none.txt"
expect "a fully excluded profile fails" 1 90 "$tmp/covered.out" "$tmp/exclude-all.txt"
expect "a missing profile fails" 1 90 "$tmp/missing.out" "$tmp/none.txt"
expect "a missing exclude list fails" 1 90 "$tmp/covered.out" "$tmp/missing.txt"
expect "a non-numeric bar fails" 1 abc "$tmp/covered.out" "$tmp/none.txt"
expect "an empty bar fails" 1 "" "$tmp/covered.out" "$tmp/none.txt"
expect "scoped mode passes an empty profile" 0 90 "$tmp/header.out" "$tmp/none.txt" 1
expect "scoped mode still fails below the bar" 1 90 "$tmp/partial.out" "$tmp/none.txt" 1

if [ "$fails" -gt 0 ]; then
	echo "coverage-gate-test: $fails case(s) failed" >&2
	exit 1
fi
echo "coverage-gate-test: all cases pass"

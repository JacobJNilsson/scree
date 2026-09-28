#!/bin/sh
# This script is the statement coverage gate.
#
# The script reads a Go coverage profile and drops every file whose path
# contains an entry of the exclude list. It exits 1 when the remaining
# statement coverage is below COVERAGE_MIN percent. The Makefile sets
# COVERAGE_MIN, so an empty or unset bar fails. A profile with nothing left
# to measure fails, so a broken test run or a broad exclude cannot pass.
# The pre-commit tier sets COVERAGE_GATE_ALLOW_EMPTY=1, because a commit
# can touch only packages that have no statements.
#
# Usage: scripts/coverage-gate.sh [profile] [exclude-file]
set -eu

profile="${1:-coverage.out}"
excludes="${2:-$(dirname "$0")/coverage-exclude.txt}"
min="${COVERAGE_MIN-}"

case "$min" in
'' | *[!0-9]*)
	echo "coverage-gate: COVERAGE_MIN '$min' is not a whole number" >&2
	exit 1
	;;
esac
if [ ! -f "$profile" ]; then
	echo "coverage-gate: profile '$profile' not found" >&2
	exit 1
fi
if [ ! -f "$excludes" ]; then
	echo "coverage-gate: exclude list '$excludes' not found" >&2
	exit 1
fi

awk -v min="$min" -v exfile="$excludes" -v allow_empty="${COVERAGE_GATE_ALLOW_EMPTY:-}" '
BEGIN {
	n = 0
	while ((getline line < exfile) > 0) {
		sub(/#.*$/, "", line)
		gsub(/^[ \t]+|[ \t]+$/, "", line)
		if (line != "") ex[++n] = line
	}
	close(exfile)
}
FNR == 1 && /^mode:/ { next }
{
	file = $1
	sub(/:[^:]*$/, "", file)
	for (i = 1; i <= n; i++) if (index(file, ex[i]) > 0) next
	total += $2
	if ($3 > 0) covered += $2
}
END {
	if (total == 0) {
		if (allow_empty == "1") {
			print "coverage-gate: no statements to measure, pass in scoped mode"
			exit 0
		}
		print "coverage-gate: FAIL, no statements to measure" > "/dev/stderr"
		exit 1
	}
	pct = covered * 100 / total
	printf "coverage-gate: %.1f%% of %d statements, bar %s%%\n", pct, total, min
	if (pct < min + 0) {
		printf "coverage-gate: FAIL, coverage below %s%%\n", min > "/dev/stderr"
		exit 1
	}
}' "$profile"

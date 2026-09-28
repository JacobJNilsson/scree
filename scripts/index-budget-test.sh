#!/bin/sh
# This script tests scripts/index-budget.sh against small reports and budgets.
set -eu

gate="$(dirname "$0")/index-budget.sh"
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

# report writes a report with the given index and partial flag in the layout of scree audit --json.
report() {
	printf '{\n  "score": {\n    "index": %s,\n    "direction": "lower-is-better",\n    "partial": %s,\n    "contributions": []\n  }\n}\n' "$1" "$2" >"$dir/r.json"
}

# expect runs the gate and fails the test when its exit status is not the wanted one.
expect() {
	want=$1
	name=$2
	status=0
	sh "$gate" "$dir/r.json" "$dir/b.json" >/dev/null 2>&1 || status=$?
	if [ "$status" -ne "$want" ]; then
		echo "index-budget-test: $name: exit $status, want $want" >&2
		exit 1
	fi
}

echo '{"maxIndex": 20}' >"$dir/b.json"
report 20 false
expect 0 "index at the budget"
report 21 false
expect 1 "index above the budget"
report 5 true
expect 1 "partial index"
report 20 false
printf '{"a": {\n  "index": 99,\n  "partial": true\n}}\n' >>"$dir/r.json"
expect 0 "second index after the score"
echo '{"maxIndex": 20}{"maxIndex": 99}' >"$dir/b.json"
report 21 false
expect 1 "two budgets on one line"
echo '{"maxIndex": 20}' >"$dir/b.json"
echo '{}' >"$dir/b.json"
expect 1 "budget without maxIndex"
echo '{"maxIndex": 20}' >"$dir/b.json"
echo '{}' >"$dir/r.json"
expect 1 "report without a score"
rm "$dir/r.json"
expect 1 "missing report"
echo "index-budget-test: all cases pass"

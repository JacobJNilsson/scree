#!/bin/sh
# This script is the index budget gate of the self-audit.
#
# The script reads the score of a JSON report that scree audit --json wrote
# and the maxIndex of the budget file. It prints both numbers. It exits 1
# when the index is above the budget, when the score is partial, or when a
# file lacks its number. The report is indented JSON in which "index" and
# "partial" appear only in the score block.
#
# Usage: scripts/index-budget.sh [report] [budget]
set -eu

report="${1:-.scree-self.json}"
budget="${2:-scree-budget.json}"

for file in "$report" "$budget"; do
	if [ ! -f "$file" ]; then
		echo "index-budget: '$file' not found" >&2
		exit 1
	fi
done

# The first match is the score block, because the report writes no other "index" or "partial" key before it.
index=$(sed -En 's/^ *"index": *([0-9]+),?$/\1/p' "$report" | head -n 1)
partial=$(sed -En 's/^ *"partial": *(true|false),?$/\1/p' "$report" | head -n 1)
max=$(sed -En 's/^[^"]*"maxIndex": *([0-9]+).*/\1/p' "$budget" | head -n 1)

# whole succeeds when its argument is one unsigned integer.
whole() {
	case "$1" in
	'' | *[!0-9]*) return 1 ;;
	esac
}

if ! whole "$index" || [ -z "$partial" ]; then
	echo "index-budget: '$report' has no score index or partial flag" >&2
	exit 1
fi
if ! whole "$max"; then
	echo "index-budget: '$budget' has no maxIndex" >&2
	exit 1
fi

echo "index-budget: index $index, budget $max"
if [ "$partial" = true ]; then
	echo "index-budget: the index is partial, so the self-audit did not measure all code" >&2
	exit 1
fi
if [ "$index" -gt "$max" ]; then
	echo "index-budget: index $index is above the budget $max" >&2
	exit 1
fi

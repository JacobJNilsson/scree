#!/bin/sh
# This script is the pre-commit tier of the gate.
#
# It checks only the packages that own a staged .go file, with the same
# vet, lint, race test, and coverage bar as `make check`. Pre-push and CI
# still run the full `make check` over the whole module.
set -eu

cd "$(git rev-parse --show-toplevel)"

staged=$(git diff --cached --name-only --no-renames)

full_gate() {
	echo "pre-commit gate: $1, running make check"
	make check
	exit 0
}

gatefiles=$(printf '%s\n' "$staged" |
	grep -E '^(go\.mod|go\.sum|Makefile|\.golangci\.yml)$|^(scripts|\.githooks)/' || true)
if [ -n "$gatefiles" ]; then
	full_gate "gate files are staged"
fi

# The go command skips testdata directories in ./..., so this script skips them too.
gofiles=$(printf '%s\n' "$staged" | grep '\.go$' | grep -Ev '(^|/)testdata/' || true)
if [ -z "$gofiles" ]; then
	echo "pre-commit gate: no .go file outside testdata is staged, running tidy-check and lint-fast"
	make tidy-check lint-fast
	exit 0
fi

dirs=$(printf '%s\n' "$gofiles" | while IFS= read -r f; do dirname "$f"; done | sort -u)
pkgs=""
lintdirs=""
for d in $dirs; do
	if ! p=$(go list "./$d" 2>/dev/null); then
		full_gate "no package resolves for staged directory $d"
	fi
	pkgs="$pkgs $p"
	lintdirs="$lintdirs ./$d"
done

echo "pre-commit gate: checking$pkgs"
make tidy-check
# shellcheck disable=SC2086
go vet $pkgs
# shellcheck disable=SC2086
golangci-lint run $lintdirs
# shellcheck disable=SC2086
go test -race -coverprofile=coverage.scoped.out $pkgs
COVERAGE_GATE_ALLOW_EMPTY=1 sh scripts/coverage-gate.sh coverage.scoped.out

# 002: Metrics

Every number in a report has one definition here. An implementation that
drifts from this file changes this file in the same pull request.

## Scope of an audit

An audit takes one root directory. The root is a Go module when it holds a
`go.mod`; the module path is recorded in the report. A root without `go.mod`
is audited the same way and recorded as unnamed.

The audit walks the root in lexical order and visits every regular file.
A directory that holds its own `go.mod` below the root is a nested module.
The walk skips it and counts it once under coverage as `nested-module`. A
directory that the source-set rules below place in `vendored`, `testdata`,
or `excluded` is never a nested module, whatever it holds.

The walk skips files and directories that `.gitignore` files under the root
exclude, with the same pattern semantics as Git. Only `.gitignore` files
are read. Git's `.git/info/exclude` and the global excludes file are state
of one machine and never apply. A `.git` entry, file or directory, is
skipped. Nothing else is skipped by name.

Build constraints do not matter. A file named `foo_windows.go` or one that
starts with `//go:build linux` is parsed like any other file. The code exists
and someone maintains it, and an offline audit must not depend on the
machine it runs on.

## Source sets

Each file belongs to exactly one source set. The first matching rule wins.

1. `testdata`: any path with a `testdata` directory segment. The Go tool
   ignores these directories, and their contents are fixtures, not code.
2. `vendored`: any path with a `vendor` directory segment.
3. `excluded`: a path matched by an `exclude` pattern in `scree.yaml`, or
   with a directory segment or file name that starts with `.` or `_`. The
   Go tool ignores such files and directories.
4. `unsupported`: a file whose name does not end in `.go`.
5. `generated`: a `.go` file that `ast.IsGenerated` recognises.
6. `test`: a file whose name ends in `_test.go`, or a `.go` file matched by
   a `classify.test` pattern in `scree.yaml`.
7. `production`: every other `.go` file.

Only `production` and `test` are parsed and measured. They are measured
separately and never mixed: a production clone never matches test code, and
test metrics never enter the index. The other sets are counted by files and
reported as coverage. Coverage is not cleanliness. A module with 40% of its
files in `unsupported` shows 40% unsupported, not a clean bill.

Patterns use glob syntax with `**` for any number of segments, matched
against the repo-relative path with forward slashes.

## Functions

The shared inventory holds one entry per function in a measured file.

- Every `FuncDecl` with a body is a function. A method is a function whose
  identity includes its receiver type.
- Every `FuncLit` is a function of its own. Its measurements never fold into
  the enclosing function.
- A `FuncDecl` without a body (an assembly or linkname declaration) is not a
  function.

Each function has a scoped identity, used by baseline comparison:

- Named: `<package dir>:<receiver>.<name>` or `<package dir>:<name>`. The
  receiver is the type name with any pointer star removed.
- Anonymous: `<enclosing identity>#<ordinal>`, where the ordinal counts
  `FuncLit`s inside the enclosing declaration in source order, starting at
  1. For a closure outside any function, the enclosing identity is
  `<package dir>:<name>` where the name is the first name of the `var` or
  `const` spec that holds it. An anonymous identity is ambiguous by
  definition.
- Identities are unique within one source set. When two named functions,
  or two package-level `var` specs that hold closures, in one package and
  one set share an identity, which happens with build-tag variants in
  different files, each identity gets the file base name appended after
  `@`. Only then. Closures inside them inherit the suffix. When identities still
  collide, which several `init` functions or several `var _` specs in one
  file cause, each gets its source-order ordinal appended after a second
  `@`.

## Complexity

Cyclomatic complexity (CC) per function is 1 plus the number of decision
points in its body, excluding nested `FuncLit` bodies:

- `if` (an `else if` is an `if`)
- `for`, including range form
- each `case` clause in an expression switch, type switch, or `select`,
  excluding `default`
- each `&&` and `||`

Nothing else counts. Go has no ternary or null-coalescing operators.

Maximum nesting depth per function is the deepest chain of nested `if`,
`for`, `switch`, and `select` statements, starting at 0 for the function
body. A `case` clause adds no level of its own. Nested `FuncLit`s start
their own count.

SLOC per function is the number of lines within the function's source range
that hold at least one token that is not a comment. A multi-line string
literal counts every line it spans. SLOC per file uses the same rule over
the whole file.

Distributions per source set: `complexity.cc.p50`, `complexity.cc.p90`,
`complexity.cc.max`, `complexity.functions` (count). Percentiles use the
nearest-rank method over the sorted CC values. A source set with no
functions reports the percentiles and the maximum as `not-applicable` and
`complexity.functions` as `0`, `complete`.

## Erosion

Erosion weights complexity by size so that a large tangled function outranks
a small tangled one.

- Function mass is `CC × sqrt(SLOC)`.
- A function is eroded when its CC exceeds `ErosionCCThreshold` (10).
- `erosion.mass`: sum of mass over all functions in the set. `0`,
  `complete` on an empty set.
- `erosion.eroded-count`: number of eroded functions.
- `erosion.eroded-share`: sum of mass over eroded functions divided by
  `erosion.mass`. A set with zero mass reports `not-applicable` for the share
  and `0` for the count.

Every floating-point sum in a report adds its terms in ascending numeric
order, with ties in any order. The result then depends only on the values,
so a file rename never moves a metric.

Aggregation from files to the set sums masses. It never averages ratios, so
one large package cannot be diluted by many small clean ones.

## Duplication

Duplication runs per source set over a normalized token stream.

Token stream. Each measured file is scanned with `go/scanner`. Comments are
dropped. The package clause and every import declaration are dropped, so
files never match on their import blocks. Every other token becomes one
symbol:

- an identifier becomes the symbol `IDENT`
- an integer, float, imaginary, rune, or string literal becomes one symbol
  per literal kind
- every keyword, operator, and delimiter is its own symbol, including the
  semicolons the scanner inserts at line ends

Two token runs are a clone when their symbol sequences are identical. This
detects exact copies and copies with renamed identifiers or changed
literals. Runs that differ by an inserted or deleted token split into
separate maximal clones. Near clones with edits are out of scope.

Thresholds. A clone group needs at least `DuplicationMinTokens` (100)
symbols and `DuplicationMinLines` (3) lines in every member.

Grouping. A clone group is the set of every maximal run that shares one
identical symbol sequence. A group needs at least two members. Repeats
within one file count, and members of one group may overlap each other.
Groups never merge through pairwise overlap.

A group is subsumed when every one of its members lies inside a member of
a longer group. Subsumed groups are dropped. A block that repeats only as
part of a larger repeated block is not independent evidence. Periodic code,
such as a switch table with many similar rows, otherwise yields one group
per multiple of the row length, and this rule keeps only the longest.

A run may begin on a semicolon that the scanner inserted at the end of the
previous line. The reported line range starts at the first symbol that is
not such a semicolon. The token count and the group id include every symbol
of the run.

Metrics per set:

- `duplication.groups`: number of clone groups.
- `duplication.duplicated-lines`: number of distinct code lines covered by at
  least one group member. A line covered by several members counts once.
- `duplication.density`: `duplicated-lines / code lines in the set`, where
  code lines are the SLOC of the set as the inventory counts them. The
  numerator uses the same per-line rule, from one shared definition.

Budgets. Before running, the detector checks the set against fixed caps:
`DuplicationMaxTokens` (2,000,000) symbols per set and
`DuplicationMaxWork` (100,000,000) work units for indexing and extraction.
A set that exceeds a cap reports every duplication metric as `incomplete`
with the cap named in the detail. No partial value is reported as a
measurement.

Engine. A suffix array built with SA-IS over the symbol sequence, an LCP
array by Kasai's algorithm, and maximal repeat extraction over LCP
intervals. Positions are token offsets, mapped back to file and line
through the inventory.

## Findings

A finding is a located piece of evidence. Findings are sorted by kind, then
path, then start line, then identity, so their order never depends on
filesystem enumeration.

- `complexity.hotspot`: one per eroded function. Carries the scoped identity,
  CC, nesting depth, SLOC, mass, and the source set.
- `duplication.clone-group`: one per clone group. Carries the group id (a
  hash of the symbol sequence), the token count, and every member with path,
  start line, and end line.

Renderers show a bounded number of findings and print the total.

## Analysis states

Every metric carries one state:

- `complete`: measured over its whole intended scope.
- `incomplete`: part of the scope was not measured. The detail says what and
  where. A parse error in a file makes every metric of that file's set
  `incomplete` and lists the file. A file or directory that cannot be read
  is listed the same way. Its contents are unknown, so it makes both
  measured sets `incomplete`, unless the path rules already place it in
  `testdata`, `vendored`, or `excluded`. An unreadable `.gitignore`, or one
  that is not a regular file, is a read error too, because the files it
  names are then measured. A read
  error never stops the audit.
- `not-applicable`: the scope is genuinely empty, for example a set with no
  functions.
- `unsupported` is a coverage category, not a metric state.

## Score

The index is a pure function of production-set metrics. Test metrics,
coverage, and safeguards never enter it.

| dimension          | weight | terms                                                                                  |
| ------------------ | ------ | -------------------------------------------------------------------------------------- |
| complexity-erosion | 0.6    | `erosion.eroded-share` saturates at 0.25 (share 0.5); `erosion.eroded-count` count scale 20 (share 0.5) |
| duplication        | 0.4    | `duplication.density` saturates at 0.15 (share 0.5); `duplication.groups` count scale 15 (share 0.5) |

- A saturating term scores `100 × min(1, value / saturatesAt)`.
- A count term scores `100 × b / (1 + b)` where `b = ln(1 + count / scale)`.
  It has no finite cap and no size denominator, so a large clean addition
  cannot erase a hotspot count.
- A dimension scores the share-weighted sum of its terms, 0–100.
- `index = clamp(floor(Σ weight × dimension + 0.5), 0, 100)`. Round half up
  over IEEE-754 doubles.
- Reported contributions are integers by largest-remainder apportionment
  (ties by dimension id) and always sum to the index.
- A `not-applicable` ratio next to a zero count scores 0. Empty scope is
  zero debt.
- A dimension whose required metric is `incomplete` scores its full weight
  and the report marks the index `partial`. Missing analysis is never zero
  debt.

Every renderer shows the index as `N/100`, the words `lower is better`, and
`scoring <version>` on one line.
The index is not a percentage of bad code and is never shown as one.

Every constant in this file, the weights, saturation points, count scales,
thresholds, and budgets, is defined once in `internal/formula/formula.go`.
Changing any of them bumps the scoring version and the golden tests in the
same commit.

## Safeguards

Safeguards are evidence about the project's process. They inspect
configuration files only. They never run anything, never import an
executable configuration, and never infer that a check passes. They
contribute nothing to the index in either direction.

Surfaces read:

- `Makefile` and `GNUmakefile` at the root: target names and recipe lines.
- `.golangci.yml`, `.golangci.yaml`, `.golangci.toml`, `.golangci.json`:
  presence and, for YAML, the enabled linter list.
- `.github/workflows/*.yml`: `run:` scalar and block values and `uses:`
  values. Expressions and conditionals are never evaluated.
- Git hooks: a `git config core.hooksPath <dir>` line in a Makefile recipe,
  `lefthook.yml` and its spelling variants, `.pre-commit-config.yaml`, and
  `.husky/` hook files.
- Agent hooks: the `hooks` map in `.claude/settings.json`.
- Coverage budget: a `COVERAGE_MIN` style variable in a Makefile, or a
  `-coverprofile` run followed by a threshold check in a Makefile or
  workflow.

Each safeguard reports one evidence level:

| level                | meaning                                                              |
| -------------------- | -------------------------------------------------------------------- |
| `absent`             | no configuration surface found                                       |
| `configured`         | configuration exists                                                 |
| `structurally-wired` | configuration is verifiably reached from an enforcement point, such as a CI step or hook that invokes it |
| `unknown`            | the surface uses constructs outside the documented subset; explicitly unverified |

Safeguard ids: `pre-commit-hook`, `pre-push-hook`, `lint-config`,
`vet-check`, `test-check`, `coverage-budget`, `ci-workflow`, `agent-hooks`.
A broken reference, such as a hook that names a missing script, is a
`safeguard.broken-reference` finding with path and line.

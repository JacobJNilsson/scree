# 004: Implementation

## Layout

```
scree/
├─ scree.go               # public package: Audit, Compare, Evaluate, LoadReport, LoadConfig
├─ cmd/scree/             # flag parsing and dispatch only
├─ internal/
│  ├─ discover/           # walk, .gitignore, source-set classification
│  ├─ inventory/          # one parse per file; functions with identity, CC, nesting, SLOC
│  ├─ complexity/         # distributions, erosion, hotspot findings
│  ├─ duplication/        # token stream, suffix array, LCP, groups, line accounting
│  ├─ formula/            # every constant and the scoring function
│  ├─ report/             # Report type, JSON round-trip, terminal and Markdown renderers
│  ├─ config/             # scree.yaml load and strict validation
│  ├─ compare/            # report comparison and policy evaluation
│  └─ safeguards/         # configuration inspection
└─ docs/spec/
```

The pipeline runs one way: discover → inventory → measure → score → report.
No package imports a package to its right.

## Stack

- Go 1.23 as the floor in `go.mod`. `go mod tidy -diff` needs 1.23.
- Standard library for parsing (`go/parser`, `go/ast`, `go/scanner`,
  `go/token`) and for the CLI (`flag`).
- `gopkg.in/yaml.v3` with `KnownFields(true)` for configuration.
- One small `.gitignore` matcher library, chosen in step 1 after a comparison
  of pattern conformance. No other runtime dependency.
- `golangci-lint` v2 with the same linter set as `anti-slop-go`, plus the
  `anti-slop-go` standalone binary run by `make antislop` at a pinned
  version.

## Conventions

- Tests sit beside the unit as `<name>_test.go`. Golden files live under
  `testdata/golden/`. Fixture modules live under `testdata/fixtures/<name>/`
  and are real Go source that the fixture's own tests parse.
- Every sorted output has a test that shuffles the input and asserts the
  same order.
- Every metric has a test that produces `incomplete` and asserts that no
  value is reported.
- The coverage gate is the same two-tier `make check` and `make check-scoped`
  as `anti-slop-go`, with `COVERAGE_MIN` at 90.
- `make selfcheck` runs `scree audit .` with the repository's own
  `scree.yaml`. A rise in the repository's own index fails the gate.
- No private repository names or identifiers in this repository. Corpus
  entries are public modules at pinned revisions.

## Steps

One pull request per step, each following the way-of-working: spec first,
tests from the spec, implementation, simplicity check, review loop, unslop
pass, draft PR.

0. Repository. `go.mod`, `Makefile`, tracked hooks, `.golangci.yml`, CI
   workflow, `AGENTS.md`, this spec, README with the trellis credit,
   `NOTICE`.
1. Discover and inventory. Walk, `.gitignore`, source sets, coverage counts,
   parse, functions with identity, CC, nesting, SLOC. Golden tests over
   fixture modules including build-tag variants and nested `go.mod`.
2. Complexity and erosion. Distributions, mass, eroded share and count,
   hotspot findings, `not-applicable` on empty sets.
3. Duplication. Token stream, SA-IS, Kasai, maximal repeats, groups,
   overlap-union lines, density, budgets that yield `incomplete`. Paired
   fixtures: exact copy, renamed copy, four-way copy, below-threshold idiom,
   near clone with two changed lines.
4. Contract, formula, renderers. `Report` type, strict JSON load, formula
   with apportionment, terminal and Markdown renderers, golden reports over
   five fixture repositories: clean, sloppy, incomplete, function-free,
   mixed-language.
5. Config, compare, policy, CLI. `scree.yaml`, `Compare`, `Evaluate`,
   `cmd/scree` with exit codes, CLI-to-package parity test.
6. Safeguards. Every surface in 002, evidence levels, broken references.
7. Corpus and calibration. A fixed list of public Go modules at pinned
   revisions, paired refactor fixtures, recorded indexes. Adjust constants
   with a scoring version bump when the index moves the wrong way on a
   paired refactor. Tag `v0.1.0` after this step.

Deferred to a later version: SQLite history, fleet over `targets.yaml`, a
scored dimension over the package graph.

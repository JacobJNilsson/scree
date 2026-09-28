# scree

Measure structural debt in a Go module.

`scree` reads a Go module from disk, parses every file once, and reports two
things: functions that have grown too many branches, and blocks of code that
repeat with renamed identifiers. It folds them into a 0–100 index, lower is
better, where every point traces back to a raw measurement and a list of
findings with file and line.

A saved report is a baseline. A later audit compares against it and a policy
in `scree.yaml` decides whether the run passes. Exit code `2` on a failed
policy, `1` when the audit could not run, `0` otherwise.

Audits run offline. No network, no model, no database, no execution of the
target's code or tools. A default run writes nothing.

## Status

Pre-release. The spec is written; see [docs/spec](docs/spec). Implementation
follows the steps in [004-implementation.md](docs/spec/004-implementation.md).

## Influenced by

`scree` is a Go re-imagining of [trellis](https://github.com/jayminwest/trellis)
by Jaymin West, a deterministic sloppiness audit for TypeScript. The metric
definitions, the traceable index, the rule that test code and safeguards
never offset production debt, and the report contract come from that work.
Code ported directly from trellis carries its MIT notice in [NOTICE](NOTICE).

## License

[MIT](LICENSE).

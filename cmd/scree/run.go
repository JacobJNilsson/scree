package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/JacobJNilsson/scree"
	"github.com/JacobJNilsson/scree/internal/report"
)

const usage = "usage: scree audit <path> [--json|--md] [--out <file>] [--quiet]\n       scree version"

const help = `usage: scree <command>

commands:
  audit <path>  measure the Go module at path and print the report
  version       print the scree version

Run scree audit --help for the audit flags and terms.
`

const auditHelp = `usage: scree audit <path> [--json|--md] [--out <file>] [--quiet]

flags:
  --json        print the report as JSON
  --md          print the report as Markdown
  --out <file>  write the report to the file, in the format of --json, --md, the .json or .md extension, else text
  --quiet       do not print the line that names the written file
  --help, -h    print this help

terms:
  index            0 to 100, the structural debt of the production code, lower is better
  production/test  the measured sets, test holds _test.go files and classify.test matches, production the other measured Go files
  cc               the cyclomatic complexity of one function
  p50/p90/max      the median, the 90th percentile, and the highest cc of a set
  eroded           the functions with cc above 10, with their share of the set mass
  mass             the cc of a function times the square root of its source lines
  clones           the groups of repeated code of at least 100 tokens and 3 lines
  dup lines        the source lines in clones, with their share of the set lines
  partial          a production input failed or hit a cap, so the index counts that part as full debt
`

// errExclusive reports two output formats in one run.
var errExclusive = errors.New("--json and --md are exclusive")

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scree", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// The program has no other place to report a failed write to stderr.
	fs.Usage = func() { _, _ = fmt.Fprintln(stderr, usage) }
	wantHelp := helpFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 1
	}
	switch {
	case *wantHelp:
		return writeText(stdout, help)
	case fs.NArg() == 1 && fs.Arg(0) == "version":
		return writeText(stdout, "scree "+scree.Version+"\n")
	case fs.NArg() > 0 && fs.Arg(0) == "audit":
		return audit(fs.Args()[1:], stdout, stderr)
	}
	fs.Usage()
	return 1
}

// helpFlags defines -h and --help, so that a help request prints to stdout and exits 0.
func helpFlags(fs *flag.FlagSet) *bool {
	wantHelp := fs.Bool("help", false, "print the help")
	fs.BoolVar(wantHelp, "h", false, "print the help")
	return wantHelp
}

func writeText(w io.Writer, text string) int {
	if _, err := io.WriteString(w, text); err != nil {
		return 1
	}
	return 0
}

// renderer writes a report in one format.
type renderer func(io.Writer, *scree.Report) error

func audit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scree audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = fmt.Fprintln(stderr, usage) }
	wantHelp := helpFlags(fs)
	asJSON := fs.Bool("json", false, "print the report as JSON")
	asMarkdown := fs.Bool("md", false, "print the report as Markdown")
	out := fs.String("out", "", "write the report to the file")
	quiet := fs.Bool("quiet", false, "do not name the written file")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	// The flag package stops at the first argument, so a flag after the path needs a second parse.
	var paths []string
	for fs.NArg() > 0 {
		paths = append(paths, fs.Arg(0))
		if err := fs.Parse(fs.Args()[1:]); err != nil {
			return 1
		}
	}
	if *wantHelp {
		return writeText(stdout, auditHelp)
	}
	if len(paths) != 1 {
		fs.Usage()
		return 1
	}
	render, err := format(*asJSON, *asMarkdown, *out)
	if err == nil {
		err = auditTo(paths[0], render, *out, stdout)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scree: %v\n", err)
		return 1
	}
	if *out != "" && !*quiet {
		_, _ = fmt.Fprintf(stderr, "wrote %s\n", *out)
	}
	return 0
}

// format picks the renderer from the flags, and from the extension of the output file when no flag names one.
func format(asJSON, asMarkdown bool, out string) (renderer, error) {
	switch {
	case asJSON && asMarkdown:
		return nil, errExclusive
	case asJSON:
		return writeJSON, nil
	case asMarkdown:
		return report.RenderMarkdown, nil
	}
	switch filepath.Ext(out) {
	case ".json":
		return writeJSON, nil
	case ".md":
		return report.RenderMarkdown, nil
	}
	return report.Render, nil
}

// auditTo audits root and writes the report to the file out, or to stdout when out is empty.
func auditTo(root string, render renderer, out string, stdout io.Writer) error {
	r, err := scree.Audit(context.Background(), root, scree.Options{})
	if err != nil {
		return err
	}
	if out == "" {
		return render(stdout, r)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := render(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// writeJSON prints the same bytes as json.MarshalIndent with two spaces, plus a newline.
func writeJSON(w io.Writer, r *scree.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

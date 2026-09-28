package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/JacobJNilsson/scree"
	"github.com/JacobJNilsson/scree/internal/report"
)

const usage = "usage: scree audit <path> [--json]\n       scree version"

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scree", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// The program has no other place to report a failed write to stderr.
	fs.Usage = func() { _, _ = fmt.Fprintln(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 1
	}
	switch {
	case fs.NArg() == 1 && fs.Arg(0) == "version":
		if _, err := fmt.Fprintf(stdout, "scree %s\n", scree.Version); err != nil {
			return 1
		}
		return 0
	case fs.NArg() > 0 && fs.Arg(0) == "audit":
		return audit(fs.Args()[1:], stdout, stderr)
	}
	fs.Usage()
	return 1
}

func audit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scree audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = fmt.Fprintln(stderr, usage) }
	asJSON := fs.Bool("json", false, "print the report as JSON")
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
	if len(paths) != 1 {
		fs.Usage()
		return 1
	}
	r, err := scree.Audit(context.Background(), paths[0], scree.Options{})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scree: %v\n", err)
		return 1
	}
	if *asJSON {
		err = writeJSON(stdout, r)
	} else {
		err = report.Render(stdout, r)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scree: %v\n", err)
		return 1
	}
	return 0
}

// writeJSON prints the same bytes as json.MarshalIndent with two spaces, plus a newline.
func writeJSON(w io.Writer, r *scree.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

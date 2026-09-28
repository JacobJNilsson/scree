package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/JacobJNilsson/scree"
)

const usage = "usage: scree version"

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scree", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// The program has no other place to report a failed write to stderr.
	fs.Usage = func() { _, _ = fmt.Fprintln(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() == 1 && fs.Arg(0) == "version" {
		if _, err := fmt.Fprintf(stdout, "scree %s\n", scree.Version); err != nil {
			return 1
		}
		return 0
	}
	fs.Usage()
	return 1
}

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/JacobJNilsson/scree"
)

// baselineFile is the default output of scree baseline, at the audited root.
const baselineFile = "scree-baseline.json"

// baselineFlags holds the parsed command line of scree baseline.
type baselineFlags struct {
	root, out, config string
	quiet, help       bool
}

func baselineCommand(args []string, stdout, stderr io.Writer) int {
	f, ok := parseBaseline(args, stderr)
	if !ok {
		return 1
	}
	if f.help {
		return writeText(stdout, baselineHelp)
	}
	if err := writeBaseline(f); err != nil {
		_, _ = fmt.Fprintf(stderr, "scree: %v\n", err)
		return 1
	}
	if !f.quiet {
		_, _ = fmt.Fprintf(stderr, "wrote %s\n", f.out)
	}
	return 0
}

func parseBaseline(args []string, stderr io.Writer) (baselineFlags, bool) {
	f := baselineFlags{root: "."}
	fs := flag.NewFlagSet("scree baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = fmt.Fprintln(stderr, usage) }
	wantHelp := helpFlags(fs)
	fs.StringVar(&f.out, "out", "", "write the baseline to the file")
	fs.BoolVar(&f.quiet, "quiet", false, "do not name the written file")
	fs.StringVar(&f.config, "config", "", "read the configuration from the file")
	paths, ok := parseInterleaved(fs, args)
	if !ok {
		return f, false
	}
	if len(paths) > 1 {
		fs.Usage()
		return f, false
	}
	if len(paths) == 1 {
		f.root = paths[0]
	}
	if f.out == "" {
		f.out = filepath.Join(f.root, baselineFile)
	}
	f.help = *wantHelp
	return f, true
}

// writeBaseline audits with the output file left out of the walk, so the file never counts as an unsupported file of its own audit.
func writeBaseline(f baselineFlags) error {
	cfg, err := loadConfig(f.config, f.root)
	if err != nil {
		return err
	}
	data, err := scree.Baseline(context.Background(), f.root, scree.Options{Config: cfg, Omit: f.out})
	if err != nil {
		return err
	}
	return os.WriteFile(f.out, data, 0o644)
}

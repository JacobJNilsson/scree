package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// main reads the flags and hands the work to run, which the tests cover.
func main() {
	list := flag.String("list", "corpus/modules.txt", "the file that lists the modules, one path@version per line")
	out := flag.String("out", "corpus/results.md", "the file that receives the results table")
	flag.Parse()
	if err := run(context.Background(), *list, *out, download, audit); err != nil {
		fmt.Fprintln(os.Stderr, "corpus:", err)
		os.Exit(1)
	}
}

// Package report holds the audit result and prints it.
package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

// Report is the result of one audit.
// The inventory block is provisional until step 4, and the metrics block follows spec 003.
type Report struct {
	Repo      Repo                       `json:"repo"`
	Coverage  Coverage                   `json:"coverage"`
	Metrics   map[string]contract.Metric `json:"metrics"`
	Inventory Inventory                  `json:"inventory"`
}

// Repo names the audited root.
type Repo struct {
	Root   string `json:"root"`
	Module string `json:"module"`
}

// Measured counts a set that the audit parses.
type Measured struct {
	Files int `json:"files"`
	SLOC  int `json:"sloc"`
}

// Counted counts a set that the audit does not parse.
type Counted struct {
	Files int `json:"files"`
}

// Coverage has the shape of the coverage block in spec 003.
type Coverage struct {
	Production    Measured `json:"production"`
	Test          Measured `json:"test"`
	Generated     Counted  `json:"generated"`
	Vendored      Counted  `json:"vendored"`
	Testdata      Counted  `json:"testdata"`
	Excluded      Counted  `json:"excluded"`
	Unsupported   Counted  `json:"unsupported"`
	NestedModules []string `json:"nestedModules"`
}

// Inventory summarises the function inventory.
type Inventory struct {
	Functions map[discover.SourceSet]int `json:"functions"`
	Errors    []inventory.Error          `json:"errors"`
}

// New builds the report of an inventory and its metrics.
func New(inv *inventory.Inventory, metrics map[string]contract.Metric) *Report {
	tree := inv.Tree
	measured := func(set discover.SourceSet) Measured {
		return Measured{Files: tree.Coverage[set].Files, SLOC: inv.SLOC[set]}
	}
	counted := func(set discover.SourceSet) Counted { return Counted{Files: tree.Coverage[set].Files} }
	functions := map[discover.SourceSet]int{discover.Production: 0, discover.Test: 0}
	for _, f := range inv.Functions {
		functions[f.Set]++
	}
	return &Report{
		Repo: Repo{Root: tree.Root, Module: tree.Module},
		Coverage: Coverage{
			Production:    measured(discover.Production),
			Test:          measured(discover.Test),
			Generated:     counted(discover.Generated),
			Vendored:      counted(discover.Vendored),
			Testdata:      counted(discover.Testdata),
			Excluded:      counted(discover.Excluded),
			Unsupported:   counted(discover.Unsupported),
			NestedModules: tree.NestedModules,
		},
		Metrics:   metrics,
		Inventory: Inventory{Functions: functions, Errors: inv.Errors},
	}
}

// Render prints a short terminal summary of a report.
func Render(w io.Writer, r *Report) error {
	var b strings.Builder
	module := r.Repo.Module
	if module == "" {
		module = "(no go.mod)"
	}
	fmt.Fprintf(&b, "root    %s\nmodule  %s\n\n", r.Repo.Root, module)
	c := r.Coverage
	fmt.Fprintf(&b, "%-12s %6d files %8d sloc %6d functions\n", "production", c.Production.Files, c.Production.SLOC, r.Inventory.Functions[discover.Production])
	fmt.Fprintf(&b, "%-12s %6d files %8d sloc %6d functions\n", "test", c.Test.Files, c.Test.SLOC, r.Inventory.Functions[discover.Test])
	for _, row := range []struct {
		name  string
		count Counted
	}{
		{"generated", c.Generated}, {"vendored", c.Vendored}, {"testdata", c.Testdata},
		{"excluded", c.Excluded}, {"unsupported", c.Unsupported},
	} {
		fmt.Fprintf(&b, "%-12s %6d files\n", row.name, row.count.Files)
	}
	fmt.Fprintf(&b, "\nnested modules: %d\n", len(c.NestedModules))
	for _, m := range c.NestedModules {
		fmt.Fprintf(&b, "  %s\n", m)
	}
	fmt.Fprintf(&b, "errors: %d\n", len(r.Inventory.Errors))
	for _, e := range r.Inventory.Errors {
		where := e.Path
		if e.Set != "" {
			where += " (" + string(e.Set) + ")"
		}
		fmt.Fprintf(&b, "  %s error in %s: %s\n", e.Kind, where, e.Message)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

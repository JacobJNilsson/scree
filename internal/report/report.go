// Package report holds the audit result and prints it.
package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/JacobJNilsson/scree/internal/complexity"
	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

// Report is the result of one audit.
// The inventory block is provisional until step 4, and the metrics and findings blocks follow spec 003.
type Report struct {
	Repo      Repo                       `json:"repo"`
	Coverage  Coverage                   `json:"coverage"`
	Metrics   map[string]contract.Metric `json:"metrics"`
	Findings  []contract.Finding         `json:"findings"`
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

// New builds the report of an inventory, its metrics, and its findings.
func New(inv *inventory.Inventory, metrics map[string]contract.Metric, findings []contract.Finding) *Report {
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
		Findings:  findings,
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
	renderComplexity(&b, r.Metrics, discover.Production)
	fmt.Fprintf(&b, "%-12s %6d files %8d sloc %6d functions\n", "test", c.Test.Files, c.Test.SLOC, r.Inventory.Functions[discover.Test])
	renderComplexity(&b, r.Metrics, discover.Test)
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
	b.WriteString("\n")
	renderHotspots(&b, r.Findings, discover.Production, true)
	renderHotspots(&b, r.Findings, discover.Test, false)
	_, err := io.WriteString(w, b.String())
	return err
}

// maxHotspots bounds the hotspot list of each set in the terminal summary.
const maxHotspots = 10

// renderComplexity prints the CC distribution and the erosion of one set on one line.
func renderComplexity(b *strings.Builder, metrics map[string]contract.Metric, set discover.SourceSet) {
	metric := func(name string) contract.Metric { return metrics[name+"."+string(set)] }
	functions := metric("complexity.functions")
	switch {
	case functions.State == contract.Incomplete:
		fmt.Fprintf(b, "  incomplete, %d read or parse errors\n", len(functions.Detail.Errors))
	case metric("complexity.cc.max").State != contract.Complete:
		b.WriteString("  no functions\n")
	default:
		cc := fmt.Sprintf("cc p50 %d  p90 %d  max %d", int(metric("complexity.cc.p50").Value), int(metric("complexity.cc.p90").Value), int(metric("complexity.cc.max").Value))
		fmt.Fprintf(b, "  %-29seroded %d of %d (share %.2f)\n", cc, int(metric("erosion.eroded-count").Value), int(functions.Value), metric("erosion.eroded-share").Value)
	}
}

// renderHotspots prints the hotspots of one set with the largest mass first, and it prints nothing for a set without hotspots unless always is set.
func renderHotspots(b *strings.Builder, findings []contract.Finding, set discover.SourceSet, always bool) {
	var hotspots []contract.Finding
	for _, f := range findings {
		if f.Kind == complexity.KindHotspot && f.SourceSet == set {
			hotspots = append(hotspots, f)
		}
	}
	if len(hotspots) == 0 {
		if always {
			fmt.Fprintf(b, "hotspots (%s): none\n", set)
		}
		return
	}
	// The findings arrive in their report order, so a stable sort keeps that order between equal masses.
	sort.SliceStable(hotspots, func(i, j int) bool { return hotspots[i].Facts.Hotspot.Mass > hotspots[j].Facts.Hotspot.Mass })
	shown := hotspots[:min(len(hotspots), maxHotspots)]
	fmt.Fprintf(b, "hotspots (%s): showing %d of %d, sorted by mass\n", set, len(shown), len(hotspots))
	for _, f := range shown {
		fmt.Fprintf(b, "  %s:%d-%d  %s  cc %d  nesting %d  sloc %d  mass %.1f\n",
			f.Path, f.StartLine, f.EndLine, f.Identity, f.Facts.Hotspot.CC, f.Facts.Hotspot.Nesting, f.Facts.Hotspot.SLOC, f.Facts.Hotspot.Mass)
	}
}

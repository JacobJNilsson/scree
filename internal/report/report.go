// Package report holds the audit result and prints it.
package report

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/JacobJNilsson/scree/internal/complexity"
	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/duplication"
	"github.com/JacobJNilsson/scree/internal/formula"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

// SchemaVersion is the version of the report shape of spec 003.
const SchemaVersion = "1.0.0"

// Completeness says whether every production metric was measured.
type Completeness string

// The completeness values of spec 003.
const (
	Complete   Completeness = "complete"
	Incomplete Completeness = "incomplete"
)

// Report is the result of one audit, in the shape of spec 003.
type Report struct {
	SchemaVersion   string                     `json:"schemaVersion"`
	AnalyzerVersion string                     `json:"analyzerVersion"`
	ScoringVersion  string                     `json:"scoringVersion"`
	Repo            Repo                       `json:"repo"`
	ConfigDigest    string                     `json:"configDigest"`
	Coverage        Coverage                   `json:"coverage"`
	Completeness    Completeness               `json:"completeness"`
	Metrics         map[string]contract.Metric `json:"metrics"`
	Score           contract.Score             `json:"score"`
	Findings        []contract.Finding         `json:"findings"`
	Limits          []contract.Limit           `json:"limits"`
	Meta            Meta                       `json:"meta"`
}

// Meta holds the facts of one run that two equal audits may not share.
type Meta struct {
	DurationMs int64 `json:"durationMs"`
}

// Run names what produced a report besides the inventory.
type Run struct {
	AnalyzerVersion string
	Config          contract.Config
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

// New builds the report of an inventory, its metrics, its findings, and the limits sorted by metric id, and it leaves Meta zero.
func New(inv *inventory.Inventory, metrics map[string]contract.Metric, findings []contract.Finding, limits []contract.Limit, run Run) *Report {
	tree := inv.Tree
	measured := func(set discover.SourceSet) Measured {
		return Measured{Files: tree.Coverage[set].Files, SLOC: inv.SLOC[set]}
	}
	counted := func(set discover.SourceSet) Counted { return Counted{Files: tree.Coverage[set].Files} }
	sorted := append([]contract.Limit{}, limits...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].MetricID < sorted[j].MetricID })
	return &Report{
		SchemaVersion:   SchemaVersion,
		AnalyzerVersion: run.AnalyzerVersion,
		ScoringVersion:  formula.ScoringVersion,
		Repo:            Repo{Root: tree.Root, Module: tree.Module},
		ConfigDigest:    contract.Digest(run.Config),
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
		Completeness: completeness(metrics),
		Metrics:      metrics,
		Score:        formula.Score(metrics),
		Findings:     findings,
		Limits:       sorted,
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
	fmt.Fprintf(&b, "%-12s %6d files %8d sloc %6d functions\n", "production", c.Production.Files, c.Production.SLOC, functionCount(r, discover.Production))
	renderComplexity(&b, r.Metrics, discover.Production)
	renderDuplication(&b, r, discover.Production)
	fmt.Fprintf(&b, "%-12s %6d files %8d sloc %6d functions\n", "test", c.Test.Files, c.Test.SLOC, functionCount(r, discover.Test))
	renderComplexity(&b, r.Metrics, discover.Test)
	renderDuplication(&b, r, discover.Test)
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
	paths := errorPaths(r)
	fmt.Fprintf(&b, "errors: %d\n", len(paths))
	for _, p := range paths {
		fmt.Fprintf(&b, "  %s\n", p)
	}
	b.WriteString("\n")
	for _, l := range []list{hotspots, clones} {
		renderList(&b, l, r, discover.Production, true)
		renderList(&b, l, r, discover.Test, false)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// completeness is incomplete when any production metric is incomplete.
func completeness(metrics map[string]contract.Metric) Completeness {
	for id, m := range metrics {
		if strings.HasSuffix(id, "."+string(contract.Production)) && m.State == contract.Incomplete {
			return Incomplete
		}
	}
	return Complete
}

// functionCount is the function count of a set, and 0 when the set was not measured.
func functionCount(r *Report, set discover.SourceSet) int {
	return int(r.Metrics["complexity.functions."+string(set)].Value)
}

// errorPaths lists the paths whose read or parse errors made a metric incomplete, sorted and without repeats.
func errorPaths(r *Report) []string {
	var paths []string
	for _, m := range r.Metrics {
		paths = append(paths, m.Detail.Errors...)
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

// maxListed bounds each finding list of the terminal summary.
const maxListed = 10

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

// renderDuplication prints the clone groups, the duplicated lines, and the density of one set on one line.
func renderDuplication(b *strings.Builder, r *Report, set discover.SourceSet) {
	groupsID := "duplication.groups." + string(set)
	groups := r.Metrics[groupsID]
	if groups.State == contract.Incomplete {
		var reasons []string
		if n := len(groups.Detail.Errors); n > 0 {
			reasons = append(reasons, fmt.Sprintf("%d read or parse errors", n))
		}
		for _, l := range r.Limits {
			if l.MetricID == groupsID && groups.Detail.Limit != nil {
				reasons = append(reasons, fmt.Sprintf("%s (%d)", l.Reason, groups.Detail.Limit.Observed))
			}
		}
		fmt.Fprintf(b, "  incomplete, %s\n", strings.Join(reasons, ", "))
		return
	}
	if r.Metrics["duplication.density."+string(set)].State != contract.Complete {
		b.WriteString("  no code lines\n")
		return
	}
	fmt.Fprintf(b, "  clones %d groups  %d dup lines  density %.3f\n",
		int(groups.Value), int(r.Metrics["duplication.duplicated-lines."+string(set)].Value), r.Metrics["duplication.density."+string(set)].Value)
}

// list describes how the terminal summary prints the findings of one kind.
type list struct {
	kind, title, sortedBy string
	// metric names the metric that is incomplete when the audit did not measure the set.
	metric string
	// size orders the list with the largest value first.
	size  func(f contract.Finding) float64
	write func(b *strings.Builder, f contract.Finding)
}

var hotspots = list{
	kind: complexity.KindHotspot, title: "hotspots", sortedBy: "mass", metric: "complexity.functions",
	size: func(f contract.Finding) float64 { return f.Facts.Hotspot.Mass },
	write: func(b *strings.Builder, f contract.Finding) {
		h := f.Facts.Hotspot
		fmt.Fprintf(b, "  %s:%d-%d  %s  cc %d  nesting %d  sloc %d  mass %.1f\n", f.Path, f.StartLine, f.EndLine, f.Identity, h.CC, h.Nesting, h.SLOC, h.Mass)
	},
}

var clones = list{
	kind: duplication.KindCloneGroup, title: "clones", sortedBy: "tokens", metric: "duplication.groups",
	size: func(f contract.Finding) float64 { return float64(f.Facts.Clone.Tokens) },
	write: func(b *strings.Builder, f contract.Finding) {
		c := f.Facts.Clone
		fmt.Fprintf(b, "  %s  %d tokens  %d members\n", c.GroupID, c.Tokens, len(c.Members))
		for _, m := range c.Members {
			fmt.Fprintf(b, "    %s:%d-%d\n", m.Path, m.StartLine, m.EndLine)
		}
	},
}

// renderList prints the findings of one kind and set with the largest first.
// An empty list prints "not measured" for an incomplete set, and "none" only when always is set.
func renderList(b *strings.Builder, l list, r *Report, set discover.SourceSet, always bool) {
	var matched []contract.Finding
	for _, f := range r.Findings {
		if f.Kind == l.kind && f.SourceSet == set {
			matched = append(matched, f)
		}
	}
	if len(matched) == 0 {
		switch {
		case r.Metrics[l.metric+"."+string(set)].State == contract.Incomplete:
			fmt.Fprintf(b, "%s (%s): not measured\n", l.title, set)
		case always:
			fmt.Fprintf(b, "%s (%s): none\n", l.title, set)
		}
		return
	}
	// The findings arrive in their report order, so a stable sort keeps that order between equal sizes.
	sort.SliceStable(matched, func(i, j int) bool { return l.size(matched[i]) > l.size(matched[j]) })
	shown := matched[:min(len(matched), maxListed)]
	fmt.Fprintf(b, "%s (%s): showing %d of %d, sorted by %s\n", l.title, set, len(shown), len(matched), l.sortedBy)
	for _, f := range shown {
		l.write(b, f)
	}
}

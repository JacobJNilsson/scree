package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/JacobJNilsson/scree/internal/complexity"
	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/duplication"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

func newReport(t *testing.T, fixture string) *Report {
	t.Helper()
	tree, err := discover.Walk(context.Background(), "../../testdata/fixtures/"+fixture, discover.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return measure(inventory.Build(tree))
}

// measure runs both measures over an inventory and merges their results the way scree.Audit does.
func measure(inv *inventory.Inventory) *Report {
	metrics, findings := complexity.Measure(inv)
	dupMetrics, clones, limits := duplication.Measure(inv)
	for id, m := range dupMetrics {
		metrics[id] = m
	}
	findings = append(findings, clones...)
	return New(inv, metrics, findings, limits)
}

var update = flag.Bool("update", false, "rewrite the golden files")

// TestDeterministicJSON shuffles the functions behind the findings and asserts byte-equal JSON.
func TestDeterministicJSON(t *testing.T) {
	tree, err := discover.Walk(context.Background(), "../../testdata/fixtures/functions", discover.Options{})
	if err != nil {
		t.Fatal(err)
	}
	inv := inventory.Build(tree)
	marshal := func() []byte {
		data, err := json.Marshal(measure(inv))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	want := marshal()
	rng := rand.New(rand.NewSource(1))
	for range 10 {
		rng.Shuffle(len(inv.Functions), func(i, j int) { inv.Functions[i], inv.Functions[j] = inv.Functions[j], inv.Functions[i] })
		if got := marshal(); !bytes.Equal(got, want) {
			t.Fatalf("JSON after a shuffle differs:\n%s\nwant:\n%s", got, want)
		}
	}
}

func TestGolden(t *testing.T) {
	for _, name := range []string{
		"sets", "functions", "broken", "empty",
		"clones/exact", "clones/renamed", "clones/fourway", "clones/idiom", "clones/near", "clones/within", "clones/nested", "clones/ladder",
	} {
		t.Run(name, func(t *testing.T) {
			r := newReport(t, name)
			got, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			rootJSON, err := json.Marshal(r.Repo.Root)
			if err != nil {
				t.Fatal(err)
			}
			got = bytes.ReplaceAll(got, rootJSON, []byte(`"<root>"`))
			got = append(got, '\n')
			path := filepath.Join("..", "..", "testdata", "golden", "report-"+strings.ReplaceAll(name, "/", "-")+".json")
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from the golden file, run make golden and review the diff:\n%s", path, got)
			}
		})
	}
}

func TestNewSetsFixture(t *testing.T) {
	r := newReport(t, "sets")
	want := Coverage{
		Production:    Measured{Files: 2, SLOC: 4},
		Test:          Measured{Files: 1, SLOC: 7},
		Generated:     Counted{Files: 1},
		Vendored:      Counted{Files: 1},
		Testdata:      Counted{Files: 1},
		Excluded:      Counted{Files: 6},
		Unsupported:   Counted{Files: 2},
		NestedModules: []string{"tools/gen"},
	}
	if !reflect.DeepEqual(r.Coverage, want) {
		t.Errorf("coverage:\n got %+v\nwant %+v", r.Coverage, want)
	}
	wantFunctions := map[discover.SourceSet]int{discover.Production: 2, discover.Test: 1}
	if !reflect.DeepEqual(r.Inventory.Functions, wantFunctions) {
		t.Errorf("functions = %v, want %v", r.Inventory.Functions, wantFunctions)
	}
}

func TestRender(t *testing.T) {
	r := newReport(t, "broken")
	r.Repo.Root = "/r"
	r.Coverage.NestedModules = []string{"tools/gen"}
	r.Inventory.Errors = append(r.Inventory.Errors, inventory.Error{Kind: inventory.KindRead, Path: "locked", Message: "open: permission denied"})
	var b bytes.Buffer
	if err := Render(&b, r); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"root    /r",
		"module  example.com/broken",
		"",
		"production        2 files        2 sloc      1 functions",
		"  incomplete, 1 read or parse errors",
		"  incomplete, 1 read or parse errors",
		"test              2 files        5 sloc      1 functions",
		"  incomplete, 1 read or parse errors",
		"  incomplete, 1 read or parse errors",
		"generated         0 files",
		"vendored          0 files",
		"testdata          0 files",
		"excluded          0 files",
		"unsupported       1 files",
		"",
		"nested modules: 1",
		"  tools/gen",
		"errors: 3",
		"  parse error in bad.go (production): bad.go:3:11: expected ')', found '{'",
		"  parse error in bad_test.go (test): bad_test.go:4:5: missing condition in if statement",
		"  read error in locked: open: permission denied",
		"",
		"hotspots (production): not measured",
		"hotspots (test): not measured",
		"clones (production): not measured",
		"clones (test): not measured",
		"",
	}, "\n")
	if b.String() != want {
		t.Errorf("render:\n%s\nwant:\n%s", b.String(), want)
	}
}

func render(t *testing.T, r *Report) string {
	t.Helper()
	var b bytes.Buffer
	if err := Render(&b, r); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestRenderFunctionsFixture(t *testing.T) {
	got := render(t, newReport(t, "functions"))
	want := strings.Join([]string{
		"production        9 files      146 sloc     23 functions",
		"  cc p50 1  p90 11  max 15     eroded 3 of 23 (share 0.52)",
		"  clones 0 groups  0 dup lines  density 0.000",
		"test              1 files       13 sloc      3 functions",
		"  cc p50 2  p90 12  max 12     eroded 1 of 3 (share 0.77)",
		"  clones 0 groups  0 dup lines  density 0.000",
	}, "\n")
	if !strings.Contains(got, want) {
		t.Errorf("render lacks the set summary:\n%s\nwant:\n%s", got, want)
	}
	wantHotspots := strings.Join([]string{
		"",
		"hotspots (production): showing 3 of 3, sorted by mass",
		"  cc.go:35-70  .:Long  cc 11  nesting 1  sloc 36  mass 66.0",
		"  cc.go:73-76  .:Short  cc 15  nesting 0  sloc 4  mass 30.0",
		"  cc.go:79-81  .:wrap#1  cc 11  nesting 0  sloc 3  mass 19.1",
		"hotspots (test): showing 1 of 1, sorted by mass",
		"  functions_test.go:14-17  .:allSet  cc 12  nesting 0  sloc 4  mass 24.0",
		"clones (production): none",
		"",
	}, "\n")
	if !strings.HasSuffix(got, wantHotspots) {
		t.Errorf("render lacks the hotspots:\n%s\nwant suffix:\n%s", got, wantHotspots)
	}
}

func TestRenderEmptyFixture(t *testing.T) {
	got := render(t, newReport(t, "empty"))
	for _, line := range []string{
		"production        0 files        0 sloc      0 functions\n  no functions\n  no code lines\n",
		"test              0 files        0 sloc      0 functions\n  no functions\n  no code lines\n",
	} {
		if !strings.Contains(got, line) {
			t.Errorf("render lacks %q:\n%s", line, got)
		}
	}
	for _, list := range []string{"hotspots (test)", "clones (test)"} {
		if strings.Contains(got, list) {
			t.Errorf("render prints %s without any:\n%s", list, got)
		}
	}
}

// TestRenderHotspotTies shuffles hotspots of equal mass into a report and asserts that the list keeps the report order.
func TestRenderHotspotTies(t *testing.T) {
	// Thirty entries with two masses, because sort.Slice keeps a short or already sorted list stable by chance.
	var findings []contract.Finding
	for i := range 30 {
		findings = append(findings, contract.Finding{
			Kind: complexity.KindHotspot, Path: fmt.Sprintf("f%02d.go", i/2), StartLine: 1 + i%2, EndLine: 2,
			Identity: fmt.Sprintf(".:F%02d", i), SourceSet: contract.Production,
			Facts: contract.Facts{Hotspot: &contract.HotspotFacts{CC: 11, SLOC: 1, Mass: float64(11 - 6*(i%2))}},
		})
	}
	var want []string
	for _, f := range findings {
		if len(want) == 10 || f.Facts.Hotspot.Mass != 11 {
			continue
		}
		want = append(want, fmt.Sprintf("  %s:%d-2  %s  ", f.Path, f.StartLine, f.Identity))
	}
	rng := rand.New(rand.NewSource(1))
	for range 20 {
		shuffled := append([]contract.Finding(nil), findings...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		// A report holds its findings sorted by path, start line, and identity, which here is the index order.
		sort.Slice(shuffled, func(i, j int) bool { return shuffled[i].Identity < shuffled[j].Identity })
		lines := strings.Split(render(t, &Report{Findings: shuffled}), "\n")
		start := slices.Index(lines, "hotspots (production): showing 10 of 30, sorted by mass") + 1
		if start == 0 {
			t.Fatalf("render lacks the hotspot header:\n%s", strings.Join(lines, "\n"))
		}
		for i, prefix := range want {
			if !strings.HasPrefix(lines[start+i], prefix) {
				t.Fatalf("hotspot %d = %q, want prefix %q", i, lines[start+i], prefix)
			}
		}
	}
}

// TestRenderHotspotBound prints the ten largest hotspots of a set and the total.
func TestRenderHotspotBound(t *testing.T) {
	r := &Report{}
	for i := range 12 {
		r.Findings = append(r.Findings, contract.Finding{
			Kind: complexity.KindHotspot, Path: fmt.Sprintf("f%02d.go", i), StartLine: 1, EndLine: 2,
			Identity: fmt.Sprintf(".:F%02d", i), SourceSet: contract.Production,
			Facts: contract.Facts{Hotspot: &contract.HotspotFacts{CC: 11, SLOC: i + 1, Mass: float64(i)}},
		})
	}
	got := render(t, r)
	if !strings.Contains(got, "hotspots (production): showing 10 of 12, sorted by mass\n  f11.go:1-2  .:F11  cc 11  nesting 0  sloc 12  mass 11.0\n") {
		t.Errorf("render lacks the bounded list with the largest mass first:\n%s", got)
	}
	if strings.Contains(got, ".:F01") || !strings.Contains(got, ".:F02") {
		t.Errorf("render shows the wrong ten hotspots:\n%s", got)
	}
}

func TestRenderWithoutModule(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, &Report{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "module  (no go.mod)\n") {
		t.Errorf("render without a module:\n%s", b.String())
	}
}

var errWrite = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestRenderWriteFailure(t *testing.T) {
	if err := Render(failingWriter{}, &Report{}); !errors.Is(err, errWrite) {
		t.Errorf("error = %v, want %v", err, errWrite)
	}
}

// TestNewSortsLimits shuffles the limits of a report and asserts that New sorts them by metric id.
func TestNewSortsLimits(t *testing.T) {
	inv := inventory.Build(&discover.Tree{})
	want := []contract.Limit{
		{MetricID: "duplication.density.production", Reason: "work cap 1 exceeded"},
		{MetricID: "duplication.groups.production", Reason: "work cap 1 exceeded"},
		{MetricID: "duplication.groups.test", Reason: "tokens cap 1 exceeded"},
	}
	rng := rand.New(rand.NewSource(1))
	for range 20 {
		shuffled := append([]contract.Limit(nil), want...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if got := New(inv, nil, nil, shuffled).Limits; !reflect.DeepEqual(got, want) {
			t.Fatalf("limits:\n got %v\nwant %v", got, want)
		}
	}
}

func TestRenderClonesFixture(t *testing.T) {
	got := render(t, newReport(t, "clones/nested"))
	for _, want := range []string{
		"  cc p50 17  p90 17  max 17    eroded 2 of 3 (share 0.88)\n  clones 2 groups  141 dup lines  density 0.946\n",
		"test              0 files        0 sloc      0 functions\n  no functions\n  no code lines\n",
		strings.Join([]string{
			"clones (production): showing 2 of 2, sorted by tokens",
			"  4801221e89fa382b  321 tokens  2 members",
			"    a.go:4-62",
			"    b.go:4-62",
			"  a9131f502c4085e8  122 tokens  3 members",
			"    a.go:6-28",
			"    b.go:6-28",
			"    light.go:7-29",
			"",
		}, "\n"),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("render lacks:\n%s\ngot:\n%s", want, got)
		}
	}
	if !strings.HasSuffix(render(t, newReport(t, "clones/exact")), strings.Join([]string{
		"clones (test): showing 1 of 1, sorted by tokens",
		"  e9223f9221094ed5  120 tokens  2 members",
		"    a_test.go:6-31",
		"    b_test.go:4-29",
		"",
	}, "\n")) {
		t.Errorf("render lacks the test clones")
	}
}

// TestRenderCloneLimit prints the cap and the observed count of a set that a budget stopped.
func TestRenderCloneLimit(t *testing.T) {
	stopped := contract.Metric{State: contract.Incomplete, Unit: "count", Detail: contract.Detail{Limit: &contract.LimitDetail{Cap: "tokens", Observed: 2417311}}}
	r := &Report{
		Metrics: map[string]contract.Metric{"duplication.groups.production": stopped},
		Limits:  []contract.Limit{{MetricID: "duplication.groups.production", Reason: "tokens cap 2000000 exceeded"}},
	}
	got := render(t, r)
	if !strings.Contains(got, "functions\n  no functions\n  incomplete, tokens cap 2000000 exceeded (2417311)\n") {
		t.Errorf("render lacks the cap:\n%s", got)
	}
	if !strings.HasSuffix(got, "\nhotspots (production): none\nclones (production): not measured\n") {
		t.Errorf("render lacks the unmeasured clone list:\n%s", got)
	}
}

// TestRenderCloneBound prints the ten largest clone groups of a set, keeps the report order between equal sizes, and prints the total.
func TestRenderCloneBound(t *testing.T) {
	r := &Report{}
	for i := range 14 {
		id := fmt.Sprintf("%016x", i)
		r.Findings = append(r.Findings, contract.Finding{
			Kind: duplication.KindCloneGroup, Path: fmt.Sprintf("f%02d.go", i), StartLine: 1, EndLine: 5, Identity: id,
			SourceSet: contract.Production,
			Facts: contract.Facts{Clone: &contract.CloneFacts{GroupID: id, Tokens: 100 + 10*(i/2), Members: []contract.CloneMember{
				{Path: fmt.Sprintf("f%02d.go", i), StartLine: 1, EndLine: 5}, {Path: "z.go", StartLine: i + 1, EndLine: i + 5},
			}}},
		})
	}
	got := render(t, r)
	want := "clones (production): showing 10 of 14, sorted by tokens\n" +
		"  000000000000000c  160 tokens  2 members\n    f12.go:1-5\n    z.go:13-17\n" +
		"  000000000000000d  160 tokens  2 members\n    f13.go:1-5\n    z.go:14-18\n" +
		"  000000000000000a  150 tokens  2 members\n"
	if !strings.Contains(got, want) {
		t.Errorf("render lacks the bounded list:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "0000000000000003 ") || !strings.Contains(got, "0000000000000004 ") {
		t.Errorf("render shows the wrong ten groups:\n%s", got)
	}
}

// TestRenderCapWithErrors prints both reasons of a set that has parse errors and passes a cap.
func TestRenderCapWithErrors(t *testing.T) {
	r := &Report{
		Metrics: map[string]contract.Metric{"duplication.groups.production": {
			State: contract.Incomplete, Unit: "count",
			Detail: contract.Detail{Errors: []string{"a.go", "b.go"}, Limit: &contract.LimitDetail{Cap: "tokens", Observed: 2417311}},
		}},
		Limits: []contract.Limit{{MetricID: "duplication.groups.production", Reason: "tokens cap 2000000 exceeded"}},
	}
	if got := render(t, r); !strings.Contains(got, "\n  incomplete, 2 read or parse errors, tokens cap 2000000 exceeded (2417311)\n") {
		t.Errorf("render lacks both reasons:\n%s", got)
	}
}

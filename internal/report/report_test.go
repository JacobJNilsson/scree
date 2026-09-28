package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JacobJNilsson/scree/internal/complexity"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

func newReport(t *testing.T, fixture string) *Report {
	t.Helper()
	tree, err := discover.Walk(context.Background(), "../../testdata/fixtures/"+fixture, discover.Options{})
	if err != nil {
		t.Fatal(err)
	}
	inv := inventory.Build(tree)
	return New(inv, complexity.Measure(inv))
}

var update = flag.Bool("update", false, "rewrite the golden files")

func TestGolden(t *testing.T) {
	for _, name := range []string{"sets", "functions", "broken", "empty"} {
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
			path := filepath.Join("..", "..", "testdata", "golden", "report-"+name+".json")
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
		"test              2 files        5 sloc      1 functions",
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
	}, "\n")
	if b.String() != want {
		t.Errorf("render:\n%s\nwant:\n%s", b.String(), want)
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

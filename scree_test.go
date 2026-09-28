package scree

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// TestAuditMergesMeasures asserts that a report holds the metrics and findings of both measures in the order of spec 002.
func TestAuditMergesMeasures(t *testing.T) {
	r, err := Audit(context.Background(), "testdata/fixtures/clones/nested", Options{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range r.Findings {
		got = append(got, f.Kind+" "+f.Path+":"+strconv.Itoa(f.StartLine))
	}
	want := []string{
		"complexity.hotspot a.go:4", "complexity.hotspot b.go:4",
		"duplication.clone-group a.go:4", "duplication.clone-group a.go:6",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findings = %v, want %v", got, want)
	}
	for _, id := range []string{"complexity.functions.production", "duplication.groups.production"} {
		if _, ok := r.Metrics[id]; !ok {
			t.Errorf("metrics lack %s", id)
		}
	}
	if r.Limits == nil || len(r.Limits) != 0 {
		t.Errorf("limits = %#v, want an empty list", r.Limits)
	}
}

func TestAuditOptions(t *testing.T) {
	r, err := Audit(context.Background(), "testdata/fixtures/functions", Options{
		Exclude:      []string{"sub/**"},
		TestPatterns: []string{"cc.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := []int{r.Coverage.Production.Files, r.Coverage.Test.Files, r.Coverage.Excluded.Files}
	if want := []int{7, 2, 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("production, test, and excluded files = %v, want %v", got, want)
	}
}

func TestAuditErrors(t *testing.T) {
	ctx := context.Background()
	if _, err := Audit(ctx, filepath.Join(t.TempDir(), "missing"), Options{}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing root: error %v, want fs.ErrNotExist", err)
	}
	if _, err := Audit(ctx, "scree.go", Options{}); err == nil {
		t.Error("a file as root gave no error")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Audit(cancelled, ".", Options{}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: error %v, want context.Canceled", err)
	}
}

func TestAuditParseErrorIsNoError(t *testing.T) {
	r, err := Audit(context.Background(), "testdata/fixtures/broken", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Inventory.Errors) != 2 {
		t.Errorf("errors = %v, want two", r.Inventory.Errors)
	}
}

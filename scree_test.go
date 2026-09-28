package scree

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/JacobJNilsson/scree/internal/config"
	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/formula"
	"github.com/JacobJNilsson/scree/internal/report"
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
		Config: &Config{Exclude: []string{"sub/**"}, Classify: config.Classify{Test: []string{"cc.go"}}},
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
	if r.Completeness != report.Incomplete || !r.Score.Partial {
		t.Errorf("completeness %q, partial %v, want incomplete and partial", r.Completeness, r.Score.Partial)
	}
}

func TestAuditVersionsAndDigest(t *testing.T) {
	cfg := &Config{Exclude: []string{"sub/**"}, Classify: config.Classify{Test: []string{"cc.go"}}}
	opts := Options{Config: cfg}
	r, err := Audit(context.Background(), "testdata/fixtures/functions", opts)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{r.SchemaVersion, r.AnalyzerVersion, r.ScoringVersion, r.ConfigDigest}
	want := []string{"1.0.0", Version, formula.ScoringVersion, contract.Digest(contract.Config{Exclude: cfg.Exclude, TestPatterns: cfg.Classify.Test})}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("versions and digest = %v, want %v", got, want)
	}
	if r.Score.Index != formula.Score(r.Metrics).Index {
		t.Errorf("index %d, want the formula index", r.Score.Index)
	}
}

// TestAuditDeterministic audits one fixture twice and asserts byte-equal JSON once meta is zeroed.
func TestAuditDeterministic(t *testing.T) {
	var outputs [2][]byte
	for i := range outputs {
		r, err := Audit(context.Background(), "testdata/fixtures/clones/nested", Options{})
		if err != nil {
			t.Fatal(err)
		}
		if r.Meta.DurationMs < 0 {
			t.Errorf("duration %d ms is negative", r.Meta.DurationMs)
		}
		r.Meta = report.Meta{}
		if outputs[i], err = json.Marshal(r); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(outputs[0], outputs[1]) {
		t.Errorf("two audits differ:\n%s\n%s", outputs[0], outputs[1])
	}
}

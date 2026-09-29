package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestMetricJSONRoundTrip(t *testing.T) {
	for _, in := range []string{
		`{"state":"complete","value":3,"unit":"count","detail":{}}`,
		`{"state":"complete","value":0.25,"unit":"ratio","numerator":1,"denominator":4,"detail":{}}`,
		`{"state":"not-applicable","unit":"ratio","detail":{}}`,
		`{"state":"incomplete","unit":"count","detail":{"errors":["a.go"]}}`,
		`{"state":"incomplete","unit":"count","detail":{"limit":{"cap":"tokens","observed":7}}}`,
	} {
		var m Metric
		if err := json.Unmarshal([]byte(in), &m); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		out, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != in {
			t.Errorf("round trip = %s, want %s", out, in)
		}
	}
}

func TestMetricUnmarshalRejects(t *testing.T) {
	for _, tc := range []struct{ in, field string }{
		{`{"state":"done","unit":"count","detail":{}}`, "state"},
		{`{"unit":"count","detail":{}}`, "state"},
		{`{"state":"incomplete","value":1,"unit":"count","detail":{}}`, "value"},
		{`{"state":"complete","unit":"count","detail":{}}`, "value"},
		{`{"state":"not-applicable","unit":"ratio","numerator":1,"denominator":2,"detail":{}}`, "numerator"},
		{`{"state":"complete","value":1,"unit":"ratio","numerator":1,"detail":{}}`, "denominator"},
		{`{"state":"complete","value":1,"unit":"ratio","numerator":1,"denominator":0,"detail":{}}`, "denominator"},
		{`{"state":"complete","value":1,"unit":"count","detail":{},"extra":1}`, "extra"},
		{`{"state":"complete","value":1,"unit":"count","detail":{"cause":"x"}}`, "cause"},
		{`{"state":1}`, "state"},
		{`{"state":"complete","value":1,"detail":{}}`, "unit"},
		{`{"state":"complete","value":1,"unit":"count"}`, "detail"},
		{`{"state":"complete","value":1,"unit":"count","detail":null}`, "detail"},
	} {
		var m Metric
		var fieldErr *FieldError
		if err := json.Unmarshal([]byte(tc.in), &m); !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
			t.Errorf("%s: error %v, want a FieldError naming %q", tc.in, err, tc.field)
		}
	}
}

func TestFactsUnmarshal(t *testing.T) {
	var hotspot Facts
	if err := json.Unmarshal([]byte(`{"cc":11,"nesting":2,"sloc":9,"mass":33}`), &hotspot); err != nil {
		t.Fatal(err)
	}
	if want := (HotspotFacts{CC: 11, Nesting: 2, SLOC: 9, Mass: 33}); hotspot.Clone != nil || hotspot.Hotspot == nil || *hotspot.Hotspot != want {
		t.Errorf("hotspot facts = %+v", hotspot)
	}
	var clone Facts
	if err := json.Unmarshal([]byte(`{"groupId":"00ff","tokens":120,"members":[{"path":"a.go","startLine":3,"endLine":9}]}`), &clone); err != nil {
		t.Fatal(err)
	}
	if clone.Hotspot != nil || clone.Clone == nil || clone.Clone.Members[0].Path != "a.go" {
		t.Errorf("clone facts = %+v", clone)
	}
	var broken Facts
	if err := json.Unmarshal([]byte(`{"command":"make verify","ref":"verify"}`), &broken); err != nil {
		t.Fatal(err)
	}
	if want := (BrokenReferenceFacts{Command: "make verify", Ref: "verify"}); broken.Hotspot != nil || broken.BrokenReference == nil || *broken.BrokenReference != want {
		t.Errorf("broken reference facts = %+v", broken)
	}
	for _, in := range []string{`{"cc":1,"tokens":2}`, `{"cc":"x"}`, `[]`, `{"command":"x","ref":"y","cc":1}`} {
		var bad Facts
		if err := json.Unmarshal([]byte(in), &bad); err == nil {
			t.Errorf("%s: no error", in)
		}
	}
}

func TestUnsortedFinding(t *testing.T) {
	sorted := []Finding{{Kind: "a", Path: "b.go"}, {Kind: "b", Path: "a.go"}}
	if i := UnsortedFinding(sorted); i != -1 {
		t.Errorf("sorted findings: index %d, want -1", i)
	}
	if i := UnsortedFinding([]Finding{sorted[1], sorted[0]}); i != 1 {
		t.Errorf("reversed findings: index %d, want 1", i)
	}
}

func TestDigest(t *testing.T) {
	empty := Digest(Config{})
	if want := "sha256:" + sha256Hex(`{"exclude":[],"testPatterns":[]}`); empty != want {
		t.Errorf("digest = %s, want %s", empty, want)
	}
	if got := Digest(Config{Exclude: []string{}, TestPatterns: []string{}}); got != empty {
		t.Errorf("empty lists digest %q, want the nil digest %q", got, empty)
	}
	a := Digest(Config{Exclude: []string{"b/**", "a/**"}, TestPatterns: []string{"x.go"}})
	if b := Digest(Config{Exclude: []string{"a/**", "b/**"}, TestPatterns: []string{"x.go"}}); a != b {
		t.Errorf("digest depends on pattern order: %s, %s", a, b)
	}
	if b := Digest(Config{TestPatterns: []string{"b/**", "a/**", "x.go"}}); a == b {
		t.Error("digest ignores which list holds a pattern")
	}
	patterns := []string{"b", "a"}
	Digest(Config{Exclude: patterns})
	if patterns[0] != "b" {
		t.Error("digest sorted the caller's slice")
	}
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestStrictRejectsTrailingData(t *testing.T) {
	var v struct{}
	var fieldErr *FieldError
	for _, in := range []string{`{}{}`, `{}}`, `{}]`, `{} x`} {
		if err := Strict(strings.NewReader(in), &v); !errors.As(err, &fieldErr) || fieldErr.Field != "(end)" {
			t.Errorf("%s: error %v, want a FieldError naming (end)", in, err)
		}
	}
	if err := Strict(strings.NewReader("{}\n"), &v); err != nil {
		t.Errorf("a trailing newline gave %v", err)
	}
	if err := Strict(strings.NewReader(`{`), &v); err == nil {
		t.Error("truncated JSON gave no error")
	}
}

func TestTermJSONRoundTrip(t *testing.T) {
	for _, in := range []string{
		`{"metricId":"a.production","state":"complete","value":0.18,"saturatesAt":0.25,"score":72}`,
		`{"metricId":"b.production","state":"incomplete","countScale":15,"score":100}`,
		`{"metricId":"c.production","state":"not-applicable","saturatesAt":0.15,"score":0}`,
	} {
		var term Term
		if err := json.Unmarshal([]byte(in), &term); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		out, err := json.Marshal(term)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != in {
			t.Errorf("round trip = %s, want %s", out, in)
		}
	}
}

func TestTermUnmarshalRejects(t *testing.T) {
	for _, tc := range []struct{ in, field string }{
		{`{"metricId":"a","state":"incomplete","value":0,"score":100}`, "value"},
		{`{"metricId":"a","state":"complete","score":0}`, "value"},
		{`{"metricId":"a","state":"done","score":0}`, "state"},
		{`{"metricId":"a","state":"complete","value":1,"score":0,"weight":1}`, "weight"},
	} {
		var term Term
		var fieldErr *FieldError
		if err := json.Unmarshal([]byte(tc.in), &term); !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
			t.Errorf("%s: error %v, want a FieldError naming %q", tc.in, err, tc.field)
		}
	}
}

func TestSafeguardUnmarshal(t *testing.T) {
	var ok Safeguard
	if err := json.Unmarshal([]byte(`{"id":"a","evidence":"absent","locations":[],"notes":""}`), &ok); err != nil || ok.Locations == nil {
		t.Errorf("valid safeguard: %+v, %v", ok, err)
	}
	for _, tc := range []struct{ in, field string }{
		{`{"id":"a","evidence":"absent","notes":""}`, "locations"},
		{`{"id":"a","evidence":"absent","locations":null,"notes":""}`, "locations"},
		{`{"id":"a","evidence":"absent","locations":[]}`, "notes"},
		{`{"id":"a","evidence":"absent","locations":[],"notes":"","x":1}`, "x"},
	} {
		var s Safeguard
		var fieldErr *FieldError
		if err := json.Unmarshal([]byte(tc.in), &s); !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
			t.Errorf("%s: error %v, want a FieldError naming %q", tc.in, err, tc.field)
		}
	}
}

func TestFactsUnmarshalRequiresEveryField(t *testing.T) {
	for _, tc := range []struct{ in, field string }{
		{`null`, "facts"},
		{`{}`, "facts.cc"},
		{`{"cc":1,"nesting":0,"sloc":1}`, "facts.mass"},
		{`{"groupId":"00"}`, "facts.tokens"},
		{`{"groupId":"00","tokens":100}`, "facts.members"},
		{`{"command":"make"}`, "facts.ref"},
	} {
		var f Facts
		var fieldErr *FieldError
		if err := json.Unmarshal([]byte(tc.in), &f); !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
			t.Errorf("%s: error %v, want a FieldError naming %q", tc.in, err, tc.field)
		}
	}
}

package contract

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestMetricJSON(t *testing.T) {
	for _, tc := range []struct {
		name   string
		metric Metric
		want   string
	}{
		{
			name:   "count",
			metric: Metric{State: Complete, Value: 3, Unit: "count"},
			want:   `{"state":"complete","value":3,"unit":"count","detail":{}}`,
		},
		{
			name:   "ratio",
			metric: Metric{State: Complete, Value: 0.25, Unit: "ratio", Numerator: 1, Denominator: 4},
			want:   `{"state":"complete","value":0.25,"unit":"ratio","numerator":1,"denominator":4,"detail":{}}`,
		},
		{
			name:   "zero value",
			metric: Metric{State: Complete, Unit: "count"},
			want:   `{"state":"complete","value":0,"unit":"count","detail":{}}`,
		},
		{
			name:   "not applicable",
			metric: Metric{State: NotApplicable, Value: 1, Unit: "ratio", Numerator: 1, Denominator: 1},
			want:   `{"state":"not-applicable","unit":"ratio","detail":{}}`,
		},
		{
			name:   "incomplete",
			metric: Metric{State: Incomplete, Value: 1, Unit: "count", Detail: Detail{Errors: []string{"a.go"}}},
			want:   `{"state":"incomplete","unit":"count","detail":{"errors":["a.go"]}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.metric)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("JSON = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestFactsJSON(t *testing.T) {
	hotspot := &HotspotFacts{CC: 11, Nesting: 2, SLOC: 9, Mass: 33}
	clone := &CloneFacts{GroupID: "00ff", Tokens: 120, Members: []CloneMember{{Path: "a.go", StartLine: 3, EndLine: 9}}}
	for _, tc := range []struct {
		name  string
		facts Facts
		want  string
	}{
		{name: "hotspot", facts: Facts{Hotspot: hotspot}, want: `{"cc":11,"nesting":2,"sloc":9,"mass":33}`},
		{
			name:  "clone",
			facts: Facts{Clone: clone},
			want:  `{"groupId":"00ff","tokens":120,"members":[{"path":"a.go","startLine":3,"endLine":9}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.facts)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("JSON = %s, want %s", got, tc.want)
			}
		})
	}
	for _, facts := range []Facts{{}, {Hotspot: hotspot, Clone: clone}} {
		if _, err := json.Marshal(facts); !errors.Is(err, errFacts) {
			t.Errorf("Marshal(%+v) error = %v, want %v", facts, err, errFacts)
		}
	}
}

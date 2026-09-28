package contract

import (
	"encoding/json"
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

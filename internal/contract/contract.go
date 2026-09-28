// Package contract holds the types that several pipeline stages share, and it imports no other internal package.
package contract

import "encoding/json"

// MetricState says how much of its scope a metric measured.
type MetricState string

// The analysis states of spec 002.
const (
	Complete      MetricState = "complete"
	Incomplete    MetricState = "incomplete"
	NotApplicable MetricState = "not-applicable"
)

// Metric is one measured number with its state.
type Metric struct {
	State MetricState
	Value float64
	Unit  string
	// Numerator and Denominator are set for a ratio only, and a complete ratio always has a nonzero denominator.
	Numerator   float64
	Denominator float64
	Detail      Detail
}

// Detail says what a metric left out.
type Detail struct {
	// Errors lists the paths whose errors make the metric incomplete.
	Errors []string `json:"errors,omitempty"`
}

// MarshalJSON omits every number of a metric that is not complete, so that a report never shows an unmeasured value.
func (m Metric) MarshalJSON() ([]byte, error) {
	out := struct {
		State       MetricState `json:"state"`
		Value       *float64    `json:"value,omitempty"`
		Unit        string      `json:"unit"`
		Numerator   *float64    `json:"numerator,omitempty"`
		Denominator *float64    `json:"denominator,omitempty"`
		Detail      Detail      `json:"detail"`
	}{State: m.State, Unit: m.Unit, Detail: m.Detail}
	if m.State == Complete {
		out.Value = &m.Value
		if m.Denominator != 0 {
			out.Numerator, out.Denominator = &m.Numerator, &m.Denominator
		}
	}
	return json.Marshal(out)
}

// SourceSet names the group a file belongs to.
type SourceSet string

// The source sets of spec 002, in the order a report lists them.
const (
	Production  SourceSet = "production"
	Test        SourceSet = "test"
	Generated   SourceSet = "generated"
	Vendored    SourceSet = "vendored"
	Testdata    SourceSet = "testdata"
	Excluded    SourceSet = "excluded"
	Unsupported SourceSet = "unsupported"
)

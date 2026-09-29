// Package contract holds the types that several pipeline stages share, and it imports no other internal package.
package contract

import (
	"encoding/json"
	"errors"
	"sort"
)

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
	// Limit names the budget cap that stopped the measurement.
	Limit *LimitDetail `json:"limit,omitempty"`
}

// LimitDetail names a budget cap and the count that exceeded it.
type LimitDetail struct {
	Cap      string `json:"cap"`
	Observed int    `json:"observed"`
}

// Limit is one entry of the limits list of spec 003, a metric that a budget cap stopped.
type Limit struct {
	MetricID string `json:"metricId"`
	Reason   string `json:"reason"`
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

// Score is the score block of spec 003.
type Score struct {
	Index     int    `json:"index"`
	Direction string `json:"direction"`
	// Partial is true when a dimension scored its full weight because a required metric was not measured.
	Partial       bool           `json:"partial"`
	Contributions []Contribution `json:"contributions"`
}

// Contribution is the share of the index that one dimension adds.
type Contribution struct {
	Dimension string  `json:"dimension"`
	Points    int     `json:"points"`
	Weight    float64 `json:"weight"`
	Terms     []Term  `json:"terms"`
}

// Term is one metric of a dimension with the score it earned.
type Term struct {
	MetricID string      `json:"metricId"`
	State    MetricState `json:"state"`
	// Value is 0 for a metric that is not complete.
	Value float64
	// Exactly one of SaturatesAt and CountScale is set.
	SaturatesAt float64
	CountScale  float64
	Score       float64
}

// termJSON is the wire shape of a term, and Value is nil unless the term is complete.
type termJSON struct {
	MetricID    string      `json:"metricId"`
	State       MetricState `json:"state"`
	Value       *float64    `json:"value,omitempty"`
	SaturatesAt float64     `json:"saturatesAt,omitempty"`
	CountScale  float64     `json:"countScale,omitempty"`
	Score       float64     `json:"score"`
}

// MarshalJSON omits the value of a term that is not complete, so that an unmeasured metric never shows as zero.
func (t Term) MarshalJSON() ([]byte, error) {
	out := termJSON{MetricID: t.MetricID, State: t.State, SaturatesAt: t.SaturatesAt, CountScale: t.CountScale, Score: t.Score}
	if t.State == Complete {
		out.Value = &t.Value
	}
	return json.Marshal(out)
}

// Finding is one located piece of evidence.
type Finding struct {
	Kind      string    `json:"kind"`
	Path      string    `json:"path"`
	StartLine int       `json:"startLine"`
	EndLine   int       `json:"endLine"`
	Identity  string    `json:"identity"`
	Ambiguous bool      `json:"ambiguous"`
	SourceSet SourceSet `json:"sourceSet"`
	Facts     Facts     `json:"facts"`
}

// Facts holds the measurements of one finding, and only the field of the finding kind is set.
type Facts struct {
	Hotspot *HotspotFacts
	Clone   *CloneFacts
}

// errFacts reports facts that do not hold exactly one kind.
var errFacts = errors.New("contract: finding facts must hold exactly one kind")

// MarshalJSON emits the facts of the one kind that is set, so that the JSON shape depends on the finding kind alone.
func (f Facts) MarshalJSON() ([]byte, error) {
	switch {
	case f.Hotspot != nil && f.Clone == nil:
		return json.Marshal(f.Hotspot)
	case f.Clone != nil && f.Hotspot == nil:
		return json.Marshal(f.Clone)
	}
	return nil, errFacts
}

// SortFindings orders findings by kind, path, start line, and identity, as spec 002 demands.
func SortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		return a.Identity < b.Identity
	})
}

// HotspotFacts are the measurements of the function behind a complexity hotspot.
type HotspotFacts struct {
	CC      int     `json:"cc"`
	Nesting int     `json:"nesting"`
	SLOC    int     `json:"sloc"`
	Mass    float64 `json:"mass"`
}

// CloneFacts are the measurements of a clone group.
type CloneFacts struct {
	GroupID string        `json:"groupId"`
	Tokens  int           `json:"tokens"`
	Members []CloneMember `json:"members"`
}

// CloneMember is one located copy of a clone group.
type CloneMember struct {
	Path      string `json:"path"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

// Location is a repo-relative path with an optional 1-based line.
type Location struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
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

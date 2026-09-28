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

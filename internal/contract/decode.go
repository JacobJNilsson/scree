package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// UnmarshalJSON reads a metric strictly and rejects the numbers that MarshalJSON would never write.
func (m *Metric) UnmarshalJSON(data []byte) error {
	var in metricJSON
	if err := strict(data, &in); err != nil {
		return err
	}
	if err := in.check(); err != nil {
		return err
	}
	*m = Metric{State: in.State, Unit: *in.Unit, Detail: *in.Detail}
	if in.Value != nil {
		m.Value = *in.Value
	}
	if in.Denominator != nil {
		m.Numerator, m.Denominator = *in.Numerator, *in.Denominator
	}
	return nil
}

// metricJSON is the wire shape of a metric, with pointers that tell an absent number from a zero.
type metricJSON struct {
	State       MetricState `json:"state"`
	Value       *float64    `json:"value"`
	Unit        *string     `json:"unit"`
	Numerator   *float64    `json:"numerator"`
	Denominator *float64    `json:"denominator"`
	Detail      *Detail     `json:"detail"`
}

func (in metricJSON) check() error {
	if err := in.checkPresent(); err != nil {
		return err
	}
	if (in.Value != nil) != (in.State == Complete) {
		return &FieldError{Field: "value", Reason: "must appear exactly when the state is complete"}
	}
	if in.State != Complete && (in.Numerator != nil || in.Denominator != nil) {
		return &FieldError{Field: "numerator", Reason: "must not appear unless the state is complete"}
	}
	if (in.Numerator != nil) != (in.Denominator != nil) || (in.Denominator != nil && *in.Denominator == 0) {
		return &FieldError{Field: "denominator", Reason: "must be nonzero and appear with the numerator"}
	}
	return nil
}

// checkPresent rejects an unknown state and a missing unit or detail.
func (in metricJSON) checkPresent() error {
	if err := checkState(in.State); err != nil {
		return err
	}
	if in.Unit == nil {
		return &FieldError{Field: "unit", Reason: "is missing"}
	}
	if in.Detail == nil {
		return &FieldError{Field: "detail", Reason: "is missing"}
	}
	return nil
}

// UnmarshalJSON reads a safeguard strictly and rejects a missing or null locations list and a missing note, which a plain decode reads as empty.
func (s *Safeguard) UnmarshalJSON(data []byte) error {
	var in struct {
		ID        string      `json:"id"`
		Evidence  Evidence    `json:"evidence"`
		Locations *[]Location `json:"locations"`
		Notes     *string     `json:"notes"`
	}
	if err := strict(data, &in); err != nil {
		return err
	}
	if in.Locations == nil || *in.Locations == nil {
		return &FieldError{Field: "locations", Reason: "must be a list"}
	}
	if in.Notes == nil {
		return &FieldError{Field: "notes", Reason: "is missing"}
	}
	*s = Safeguard{ID: in.ID, Evidence: in.Evidence, Locations: *in.Locations, Notes: *in.Notes}
	return nil
}

// UnmarshalJSON reads the facts of the kind whose key it finds, since the JSON carries no kind of its own.
func (f *Facts) UnmarshalJSON(data []byte) error {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	if keys == nil {
		return &FieldError{Field: "facts", Reason: "must be an object"}
	}
	if _, ok := keys["command"]; ok {
		*f = Facts{BrokenReference: &BrokenReferenceFacts{}}
		if err := requireKeys(keys, "command", "ref"); err != nil {
			return err
		}
		return strict(data, f.BrokenReference)
	}
	if _, ok := keys["groupId"]; ok {
		*f = Facts{Clone: &CloneFacts{}}
		if err := requireKeys(keys, "groupId", "tokens", "members"); err != nil {
			return err
		}
		return strict(data, f.Clone)
	}
	*f = Facts{Hotspot: &HotspotFacts{}}
	if err := requireKeys(keys, "cc", "nesting", "sloc", "mass"); err != nil {
		return err
	}
	return strict(data, f.Hotspot)
}

// requireKeys names the first missing key, because a missing number would decode as a zero.
func requireKeys(keys map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		if _, ok := keys[name]; !ok {
			return &FieldError{Field: "facts." + name, Reason: "is missing"}
		}
	}
	return nil
}

// UnmarshalJSON reads a term strictly and rejects a value on a term that is not complete, as MarshalJSON never writes one.
func (t *Term) UnmarshalJSON(data []byte) error {
	var in termJSON
	if err := strict(data, &in); err != nil {
		return err
	}
	if err := checkState(in.State); err != nil {
		return err
	}
	if (in.Value != nil) != (in.State == Complete) {
		return &FieldError{Field: "value", Reason: "must appear exactly when the state is complete"}
	}
	*t = Term{MetricID: in.MetricID, State: in.State, SaturatesAt: in.SaturatesAt, CountScale: in.CountScale, Score: in.Score}
	if in.Value != nil {
		t.Value = *in.Value
	}
	return nil
}

func checkState(state MetricState) error {
	switch state {
	case Complete, Incomplete, NotApplicable:
		return nil
	}
	return &FieldError{Field: "state", Reason: fmt.Sprintf("%q is unknown", state)}
}

// FieldError names the report field that breaks the contract of spec 003.
type FieldError struct {
	Field  string
	Reason string
}

func (e *FieldError) Error() string {
	return e.Field + ": " + e.Reason
}

// Strict decodes one JSON value and rejects unknown fields, because a custom unmarshaler does not inherit that setting from its caller.
func Strict[T any](r io.Reader, v *T) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fieldError(err)
	}
	// More reports false before a stray closing bracket, so only io.EOF proves that the value ends the input.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return &FieldError{Field: "(end)", Reason: "trailing data after the value"}
	}
	return nil
}

func strict[T any](data []byte, v *T) error {
	return Strict(bytes.NewReader(data), v)
}

// fieldError names the field of a decode error, since encoding/json names an unknown field only in the error text.
func fieldError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return &FieldError{Field: typeErr.Field, Reason: err.Error()}
	}
	if name, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		return &FieldError{Field: strings.Trim(name, `"`), Reason: "is unknown"}
	}
	return err
}

// UnsortedFinding returns the index of the first finding out of the order of SortFindings, or -1.
func UnsortedFinding(findings []Finding) int {
	for i := 1; i < len(findings); i++ {
		pair := []Finding{findings[i], findings[i-1]}
		SortFindings(pair)
		if pair[0] != findings[i-1] {
			return i
		}
	}
	return -1
}

// Config is the effective configuration that the digest covers.
type Config struct {
	Exclude      []string
	TestPatterns []string
}

// Digest returns "sha256:" and the hex SHA-256 of the canonical JSON of the configuration, with each list sorted.
func Digest(c Config) string {
	canonical := struct {
		Exclude      []string `json:"exclude"`
		TestPatterns []string `json:"testPatterns"`
	}{sortedCopy(c.Exclude), sortedCopy(c.TestPatterns)}
	// Marshal cannot fail on a struct of string slices.
	data, _ := json.Marshal(canonical)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	slices.Sort(out)
	return out
}

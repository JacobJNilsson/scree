package report

import (
	"fmt"
	"io"

	"github.com/JacobJNilsson/scree/internal/complexity"
	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/duplication"
	"github.com/JacobJNilsson/scree/internal/formula"
)

// maxIndex is the highest index that the score of spec 002 can take.
const maxIndex = 100

// Load decodes a report strictly and checks the rules of spec 003 that decoding cannot check.
func Load(r io.Reader) (*Report, error) {
	var out Report
	if err := contract.Strict(r, &out); err != nil {
		return nil, fmt.Errorf("report: %w", err)
	}
	if err := validate(&out); err != nil {
		return nil, fmt.Errorf("report: %w", err)
	}
	return &out, nil
}

func validate(r *Report) error {
	if r.SchemaVersion != SchemaVersion {
		return &contract.FieldError{Field: "schemaVersion", Reason: fmt.Sprintf("%q, want %q", r.SchemaVersion, SchemaVersion)}
	}
	if r.Completeness != Complete && r.Completeness != Incomplete {
		return &contract.FieldError{Field: "completeness", Reason: fmt.Sprintf("%q is unknown", r.Completeness)}
	}
	s := r.Score
	if s.Direction != formula.DirectionLowerIsBetter {
		return &contract.FieldError{Field: "score.direction", Reason: fmt.Sprintf("%q, want %q", s.Direction, formula.DirectionLowerIsBetter)}
	}
	if s.Index < 0 || s.Index > maxIndex {
		return &contract.FieldError{Field: "score.index", Reason: fmt.Sprintf("%d is outside 0..%d", s.Index, maxIndex)}
	}
	points := 0
	for _, c := range s.Contributions {
		points += c.Points
	}
	if points != s.Index {
		return &contract.FieldError{Field: "score.contributions", Reason: fmt.Sprintf("sum to %d, want the index %d", points, s.Index)}
	}
	return checkFindings(r.Findings)
}

func checkFindings(findings []contract.Finding) error {
	for i, f := range findings {
		if err := checkFinding(f); err != nil {
			err.Field = fmt.Sprintf("findings[%d].%s", i, err.Field)
			return err
		}
	}
	if i := contract.UnsortedFinding(findings); i >= 0 {
		return &contract.FieldError{Field: fmt.Sprintf("findings[%d]", i), Reason: "is out of order"}
	}
	return nil
}

// checkFinding rejects an unknown kind, a set that the audit does not measure, and facts of another kind.
func checkFinding(f contract.Finding) *contract.FieldError {
	if f.SourceSet != contract.Production && f.SourceSet != contract.Test {
		return &contract.FieldError{Field: "sourceSet", Reason: fmt.Sprintf("%q is not a measured set", f.SourceSet)}
	}
	var shapeFits bool
	switch f.Kind {
	case complexity.KindHotspot:
		shapeFits = f.Facts.Hotspot != nil
	case duplication.KindCloneGroup:
		shapeFits = f.Facts.Clone != nil
	default:
		return &contract.FieldError{Field: "kind", Reason: fmt.Sprintf("%q is unknown", f.Kind)}
	}
	if !shapeFits {
		return &contract.FieldError{Field: "facts", Reason: fmt.Sprintf("do not have the shape of %s", f.Kind)}
	}
	return nil
}

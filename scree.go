// Package scree measures the structural debt of a Go module.
package scree

import (
	"context"

	"github.com/JacobJNilsson/scree/internal/complexity"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/duplication"
	"github.com/JacobJNilsson/scree/internal/inventory"
	"github.com/JacobJNilsson/scree/internal/report"
)

// Version is the scree release that produced a report.
const Version = "0.1.0-dev"

// Report is the result of one audit.
type Report = report.Report

// Options holds the classification patterns of an audit.
type Options struct {
	// Exclude puts matching files into the excluded set.
	Exclude []string
	// TestPatterns puts matching .go files into the test set.
	TestPatterns []string
}

// Audit measures the Go module at root.
// A parse error is part of the report and not an error.
func Audit(ctx context.Context, root string, opts Options) (*Report, error) {
	tree, err := discover.Walk(ctx, root, discover.Options{Exclude: opts.Exclude, TestPatterns: opts.TestPatterns})
	if err != nil {
		return nil, err
	}
	inv := inventory.Build(tree)
	metrics, findings := complexity.Measure(inv)
	dupMetrics, clones, limits := duplication.Measure(inv)
	for id, m := range dupMetrics {
		metrics[id] = m
	}
	// Each measure sorts its own findings, and every complexity kind sorts before the clone kind, so the joined list keeps the order of spec 002.
	findings = append(findings, clones...)
	return report.New(inv, metrics, findings, limits), nil
}

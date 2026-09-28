// Package formula holds the constants that spec 002 names.
// Later steps add the scoring constants here.
package formula

// ErosionCCThreshold is the CC that a function must exceed to count as eroded.
const ErosionCCThreshold = 10

// The duplication thresholds and budgets of spec 002.
const (
	// DuplicationMinTokens is the fewest symbols of a clone group member.
	DuplicationMinTokens = 100
	// DuplicationMinLines is the fewest lines that a clone group member spans.
	DuplicationMinLines = 3
	// DuplicationMaxTokens is the most symbols of one source set that the detector indexes.
	DuplicationMaxTokens = 2_000_000
	// DuplicationMaxWork is the most work units that indexing and extraction of one source set may spend.
	DuplicationMaxWork = 100_000_000
)

package after

import "example.com/pairs/extract-clone/after/shared"

// FoldLines folds the lines of the second file.
func FoldLines(lines []string, seed int) int {
	return shared.Fold(lines, seed)
}

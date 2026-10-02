package after

import "example.com/pairs/extract-clone/after/shared"

// FoldWords folds the words of the first file.
func FoldWords(words []string, seed int) int {
	return shared.Fold(words, seed)
}

package after

// Fold folds a list of words into one value.
func Fold(words []string, seed int) int {
	total := seed

	// The loop folds each word into the total.
	for i, w := range words {
		if len(w) > 3 {
			total += i * len(w)
		} else {
			total -= i + 7
		}
		switch {
		case total > 500:
			total /= 2
		case total < 0:
			total = -total
		}
		total = total%1000 + 1
	}
	for len(words) > 0 && total > seed {
		words = words[1:]
		total -= len(words)
	}
	return total
}

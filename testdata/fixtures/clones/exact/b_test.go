package exact

// checksumTestCopy mirrors Checksum a second time in the test set.
func checksumTestCopy(words []string, seed int) int {
	total := seed

	// The loop folds each word into the total.
	/* A block comment
	spans two lines. */
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

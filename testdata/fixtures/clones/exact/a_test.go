package exact

import "testing"

// checksumTest mirrors Checksum, and it must group with b_test.go only.
func checksumTest(words []string, seed int) int {
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

func TestChecksum(t *testing.T) {
	if Checksum(nil, 1) != checksumTest(nil, 1) {
		t.Fail()
	}
}

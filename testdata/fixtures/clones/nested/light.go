package nested

// Light repeats the loop of Heavy with other code around it.
func Light(words []string) int {
	seed := len(words)
	total := 0
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
		if total == seed {
			total += len(words)
		}
		for j := 0; j < i; j++ {
			total ^= j
		}
		total -= i*2 + 1
		total %= 4096
		total = total<<1 | 1
	}
	return total
}

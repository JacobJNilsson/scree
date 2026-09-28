package within

// First, Second, and Third share one body, and the token before each body differs.
func First(words []string, seed int) int {
	total := seed
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

func Second[T any](words []string, seed int) int {
	total := seed
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

var Third = func(words []string, seed int) int {
	total := seed
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

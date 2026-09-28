package nested

// HeavyCopy holds the loop that light.go repeats.
func HeavyCopy(words []string, seed int) int {
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
	for len(words) > 0 && total > seed {
		words = words[1:]
		total -= len(words)
	}
	limit := len(words) * 3
	if total > limit {
		total = limit
	}
	for j := 0; j < 3; j++ {
		total = total*seed + j
	}
	switch total % 4 {
	case 0:
		total++
	case 1:
		total--
	default:
		total += 2
	}
	names := make(map[string]int, len(words))
	for k, w := range words {
		if _, ok := names[w]; !ok {
			names[w] = k
		}
	}
	for w, k := range names {
		if len(w) > k {
			total += k
		} else {
			total -= len(w)
		}
	}
	return total
}

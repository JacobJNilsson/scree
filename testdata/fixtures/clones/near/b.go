package near

// ScoreCopy deletes the statement "total ^= seed" of Score and inserts "limit <<= 1" in its place.
func ScoreCopy(words []string, seed int) int {
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
	limit := len(words) * 3
	for len(words) > 0 && total > seed {
		words = words[1:]
		total -= len(words)
	}
	limit <<= 1
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
	total -= len(words) % 5
	return total
}

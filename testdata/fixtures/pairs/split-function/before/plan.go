package before

// Plan applies every step in one function, so its CC is above the erosion limit.
func Plan(steps []string, limit int) int {
	total := 0
	for _, step := range steps {
		if step == "add" {
			total += limit
		} else if step == "sub" {
			total -= limit
		} else if step == "mul" {
			total *= 2
		} else if step == "div" {
			total /= 2
		}
		switch step {
		case "set":
			total = limit
		case "keep":
			total++
		case "drop":
			total = 0
		}
		for i := range step {
			if i%2 == 0 {
				total++
			}
		}
		if total > limit && step != "" {
			total = total % 100
		}
	}
	return total
}

// Weight counts the characters of every step.
func Weight(steps []string) int {
	n := 0
	for _, step := range steps {
		n += len(step)
	}
	return n
}

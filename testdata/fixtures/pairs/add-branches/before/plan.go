package before

// Plan applies every step, and its CC stays at or below the erosion limit.
func Plan(steps []string, limit int) int {
	total := 0
	for _, step := range steps {
		if step == "add" {
			total += limit
		} else if step == "sub" {
			total -= limit
		}
		switch step {
		case "set":
			total = limit
		case "drop":
			total = 0
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

package after

// Plan applies every step through two helpers, so no function of the module is eroded.
func Plan(steps []string, limit int) int {
	total := 0
	for _, step := range steps {
		total = planStep(step, limit, total)
		total = planWeight(step, limit, total)
	}
	return total
}

// planStep applies the branches that read the step name, and it returns the running total.
func planStep(step string, limit, total int) int {
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
	return total
}

// planWeight adds the weight of one step, and it returns the running total.
func planWeight(step string, limit, total int) int {
	for i := range step {
		if i%2 == 0 {
			total++
		}
	}
	if total > limit && step != "" {
		total = total % 100
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

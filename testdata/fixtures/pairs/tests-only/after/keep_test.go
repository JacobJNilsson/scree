package after

import "testing"

// checkTangleCopy is a second copy of the helper, so the test source set holds a clone group.
func checkTangleCopy(t *testing.T, steps []string, limit int) int {
	t.Helper()
	total := 0
	for _, step := range steps {
		if len(step) == 0 {
			total -= 1
			continue
		}
		switch {
		case step == "add":
			total += limit
		case step == "sub":
			total -= limit
		}
		total += len(step) * 2
	}
	for i := range steps {
		if i%3 == 0 {
			total += i
		}
	}
	if total > limit && len(steps) > 1 {
		total = total % 50
	}
	return total
}

// auditSteps is an eroded function of the test source set.
func auditSteps(t *testing.T, steps []string, limit int) int {
	t.Helper()
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

func TestKeep(t *testing.T) {
	if got := Keep([]string{"a", "bb"}); got != 3 {
		t.Errorf("Keep = %d, want 3", got)
	}
	if got := checkTangleCopy(t, []string{"add", ""}, 10); got >= 0 {
		t.Errorf("checkTangleCopy = %d, want a negative value", got)
	}
	if got := auditSteps(t, []string{"add", "keep", "drop"}, 10); got != 11 {
		t.Errorf("auditSteps = %d, want 11", got)
	}
}

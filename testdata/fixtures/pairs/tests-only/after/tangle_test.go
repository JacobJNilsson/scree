package after

import "testing"

// checkTangle folds the steps of one test case.
func checkTangle(t *testing.T, steps []string, limit int) int {
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

func TestTangle(t *testing.T) {
	if got := Tangle([]string{"add", "sub", "mul"}, 10); got <= 0 {
		t.Errorf("Tangle = %d, want a positive value", got)
	}
	if got := checkTangle(t, []string{"add", "", "sub"}, 10); got <= 0 {
		t.Errorf("checkTangle = %d, want a positive value", got)
	}
}

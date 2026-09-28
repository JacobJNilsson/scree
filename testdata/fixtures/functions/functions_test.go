package functions

import "testing"

func TestDecide(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		if Decide(0, nil, nil) != 0 {
			t.Fatal("want 0")
		}
	})
}

// allSet is an eroded test helper.
func allSet(a, b, c, d, e, f, g, h, i, j, k, l bool) bool {
	ok := a && b && c && d && e && f && g && h && i && j && k || l
	return ok
}

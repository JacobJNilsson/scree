package functions

import "testing"

func TestDecide(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		if Decide(0, nil, nil) != 0 {
			t.Fatal("want 0")
		}
	})
}

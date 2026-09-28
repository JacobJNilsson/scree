package functions

// Deep nests four levels.
func Deep(xs []int) int {
	n := 0
	for _, x := range xs {
		switch {
		case x > 0:
			if x%2 == 0 {
				for i := 0; i < x; i++ {
					n += i
				}
			}
		}
	}
	return n
}

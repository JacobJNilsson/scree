package functions

// Nest builds closures three deep.
func Nest() int {
	a := func() int {
		b := func() int {
			c := func() int {
				if true {
					return 1
				}
				return 0
			}
			return c()
		}
		return b()
	}
	return a()
}

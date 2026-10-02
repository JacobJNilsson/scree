package after

// Keep is a clean function, so the eroded share of the module is below 1.
func Keep(steps []string) int {
	n := 0
	for _, step := range steps {
		n += len(step)
	}
	return n
}

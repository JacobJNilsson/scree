package functions

// Decide uses every decision point once.
func Decide(x int, v any, ch chan int) int {
	if x > 0 && x < 10 {
		x++
	} else if x < 0 || x > 100 {
		x--
	}
	for i := 0; i < 3; i++ {
		x += i
	}
	for range ch {
		x++
	}
	switch x {
	case 1:
		x = 2
	default:
		x = 3
	}
	switch v.(type) {
	case int:
		x++
	}
	select {
	case <-ch:
		x++
	default:
	}
	return x
}

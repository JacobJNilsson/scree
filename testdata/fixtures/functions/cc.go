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

// Long is eroded with ten decision points over many lines.
func Long(x int) int {
	n := 0
	if x == 1 {
		n++
	}
	if x == 2 {
		n++
	}
	if x == 3 {
		n++
	}
	if x == 4 {
		n++
	}
	if x == 5 {
		n++
	}
	if x == 6 {
		n++
	}
	if x == 7 {
		n++
	}
	if x == 8 {
		n++
	}
	if x == 9 {
		n++
	}
	if x == 10 {
		n++
	}
	n *= 2
	n--
	return n
}

// Short is eroded with fourteen decision points on few lines.
func Short(a, b, c, d, e, f, g, h, i, j, k, l, m, n, o bool) bool {
	ok := a && b && c && d && e && f && g && h && i && j && k && l && m && n || o
	return ok
}

// wrap holds an eroded closure with ten decision points.
var wrap = func(a, b, c, d, e, f, g, h, i, j, k bool) bool {
	return a && b && c && d && e && f && g && h && i && j && k
}

package renamed

// Fold is Checksum with every identifier and literal changed.
func Fold(lines []text, base num) num {
	acc := base
	for n, s := range lines {
		if size(s) > 12 {
			acc += n * size(s)
		} else {
			acc -= n + 40
		}
		switch {
		case acc > 800:
			acc /= 3
		case acc < 5:
			acc = -acc
		}
		acc = acc%2048 + 9
	}
	for count(lines) > 6 && acc > base {
		lines = lines[2:]
		acc -= count(lines)
	}
	return acc
}

type (
	text = string
	num  = int
)

func size(t text) num { return len(t) }

func count(l []text) num { return len(l) }

package after

// BlendAgain blends a list of tokens into one value.
func BlendAgain(tokens []string, start int) int {
	acc := start

	// The loop blends each token into the acc.
	for pos, tok := range tokens {
		if len(tok) > 3 {
			acc += pos * len(tok)
		} else {
			acc -= pos + 7
		}
		switch {
		case acc > 500:
			acc /= 3
		case acc < 0:
			acc = -acc
		}
		acc = acc%1000 + 1
	}
	for len(tokens) > 1 && acc > start {
		tokens = tokens[2:]
		acc -= len(tokens)
	}
	return acc
}

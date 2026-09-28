package ladder

// Code maps a spelling word to its entry, one similar row per word.
func Code(word string) entry {
	switch word {
	case "alpha":
		return lookup("alpha", 0, 0.5, true)
	case "bravo":
		return lookup("bravo", 1, 1.5, true)
	case "charlie":
		return lookup("charlie", 2, 2.5, true)
	case "delta":
		return lookup("delta", 3, 3.5, true)
	case "echo":
		return lookup("echo", 4, 4.5, true)
	case "foxtrot":
		return lookup("foxtrot", 5, 5.5, true)
	case "golf":
		return lookup("golf", 6, 6.5, true)
	case "hotel":
		return lookup("hotel", 7, 7.5, true)
	case "india":
		return lookup("india", 8, 8.5, true)
	case "juliett":
		return lookup("juliett", 9, 9.5, true)
	case "kilo":
		return lookup("kilo", 10, 10.5, true)
	case "lima":
		return lookup("lima", 11, 11.5, true)
	}
	return entry{}
}

type entry struct{}

func lookup(name string, n int, weight float64, ok bool) entry { return entry{} }

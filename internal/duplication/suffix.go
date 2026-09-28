// Package duplication finds clone groups in the normalized token streams of the measured source sets.
package duplication

// This file builds a suffix array with SA-IS, from Nong, Zhang, and Chan, "Linear Suffix Array Construction by Almost Pure Induced-Sorting", 2009.
// A suffix is S-type when it sorts before the next suffix, and L-type otherwise.
// An LMS position is an S-type position whose left neighbour is L-type.
// Phase 1 sorts the LMS substrings by induced sorting from their unsorted positions.
// Phase 2 names the sorted LMS substrings and sorts the reduced string of names, by recursion when two names are equal.
// Phase 3 places the sorted LMS suffixes and induces the order of every other suffix from them.
// A virtual sentinel follows the sequence and sorts before every symbol, so the input needs no terminator.

// suffixArray returns the start positions of the suffixes of s in ascending order, for symbols in [0, alphabet).
// It adds one unit of work per symbol at each level of the recursion.
func suffixArray(s []int32, alphabet int, work *int) []int32 {
	n := len(s)
	*work += n
	sa := make([]int32, n)
	if n <= 1 {
		return sa
	}
	sType := classify(s)
	isLMS := func(i int) bool { return i > 0 && i < n && sType[i] && !sType[i-1] }
	counts := make([]int32, alphabet)
	for _, c := range s {
		counts[c]++
	}

	// This block is phase 1.
	var lms []int32
	for i := 1; i < n; i++ {
		if isLMS(i) {
			lms = append(lms, int32(i))
		}
	}
	induce(s, sa, sType, counts, lms)

	// This block is phase 2.
	sortedLMS := make([]int32, 0, len(lms))
	for _, p := range sa {
		if isLMS(int(p)) {
			sortedLMS = append(sortedLMS, p)
		}
	}
	// LMS positions lie at least two apart, so p/2 gives each one its own slot.
	names := make([]int32, n/2+1)
	name := int32(-1)
	for i, p := range sortedLMS {
		if i == 0 || !equalLMS(s, sType, isLMS, int(sortedLMS[i-1]), int(p)) {
			name++
		}
		names[p/2] = name
	}
	reduced := make([]int32, len(lms))
	for i, p := range lms {
		reduced[i] = names[p/2]
	}
	var reducedSA []int32
	if int(name)+1 < len(lms) {
		reducedSA = suffixArray(reduced, int(name)+1, work)
	} else {
		reducedSA = make([]int32, len(lms))
		for i, r := range reduced {
			reducedSA[r] = int32(i)
		}
	}

	// This block is phase 3.
	for i, r := range reducedSA {
		sortedLMS[i] = lms[r]
	}
	induce(s, sa, sType, counts, sortedLMS)
	return sa
}

// classify marks the S-type positions of s, where the last position is L-type because the virtual sentinel follows it.
func classify(s []int32) []bool {
	n := len(s)
	sType := make([]bool, n)
	for i := n - 2; i >= 0; i-- {
		sType[i] = s[i] < s[i+1] || (s[i] == s[i+1] && sType[i+1])
	}
	return sType
}

// equalLMS reports whether the LMS substrings at a and b hold the same symbols and types.
// A substring that reaches the sentinel equals no other substring.
func equalLMS(s []int32, sType []bool, isLMS func(int) bool, a, b int) bool {
	n := len(s)
	for d := 0; ; d++ {
		if a+d == n || b+d == n || s[a+d] != s[b+d] || sType[a+d] != sType[b+d] {
			return false
		}
		// Equal types at d-1 and d make both positions LMS or neither.
		if d > 0 && isLMS(a+d) {
			return true
		}
	}
}

// induce places the LMS positions at the ends of their buckets in the given order, then induces the L-type and the S-type suffixes.
func induce(s, sa []int32, sType []bool, counts []int32, lms []int32) {
	n := len(s)
	for i := range sa {
		sa[i] = -1
	}
	tails := bucketTails(counts)
	for i := len(lms) - 1; i >= 0; i-- {
		c := s[lms[i]]
		tails[c]--
		sa[tails[c]] = lms[i]
	}
	heads := bucketHeads(counts)
	// The virtual sentinel sorts first, and its left neighbour is always L-type.
	c := s[n-1]
	sa[heads[c]] = int32(n - 1)
	heads[c]++
	for i := range n {
		j := sa[i] - 1
		if sa[i] > 0 && !sType[j] {
			sa[heads[s[j]]] = j
			heads[s[j]]++
		}
	}
	tails = bucketTails(counts)
	for i := n - 1; i >= 0; i-- {
		j := sa[i] - 1
		if sa[i] > 0 && sType[j] {
			tails[s[j]]--
			sa[tails[s[j]]] = j
		}
	}
}

func bucketHeads(counts []int32) []int32 {
	heads := make([]int32, len(counts))
	sum := int32(0)
	for c, k := range counts {
		heads[c] = sum
		sum += k
	}
	return heads
}

func bucketTails(counts []int32) []int32 {
	tails := make([]int32, len(counts))
	sum := int32(0)
	for c, k := range counts {
		sum += k
		tails[c] = sum
	}
	return tails
}

// lcpArray returns, by Kasai's algorithm, the longest common prefix of each suffix in sa with the one before it, and 0 at index 0.
// It adds one unit of work per symbol.
func lcpArray(s, sa []int32, work *int) []int32 {
	n := len(s)
	*work += n
	rank := make([]int32, n)
	for i, p := range sa {
		rank[p] = int32(i)
	}
	lcp := make([]int32, n)
	h := 0
	for i := range n {
		if rank[i] == 0 {
			h = 0
			continue
		}
		j := int(sa[rank[i]-1])
		for i+h < n && j+h < n && s[i+h] == s[j+h] {
			h++
		}
		lcp[rank[i]] = int32(h)
		// The suffix at i+1 shares at least h-1 symbols with its predecessor.
		if h > 0 {
			h--
		}
	}
	return lcp
}

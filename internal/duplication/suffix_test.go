package duplication

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// naiveSuffixArray sorts every suffix by direct comparison, where a proper prefix sorts first.
func naiveSuffixArray(s []int32) []int32 {
	sa := make([]int32, len(s))
	for i := range sa {
		sa[i] = int32(i)
	}
	sort.Slice(sa, func(i, j int) bool {
		a, b := s[sa[i]:], s[sa[j]:]
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	return sa
}

// naiveLCP compares each pair of neighbours in a suffix array symbol by symbol.
func naiveLCP(s, sa []int32) []int32 {
	lcp := make([]int32, len(sa))
	for i := 1; i < len(sa); i++ {
		a, b := s[sa[i-1]:], s[sa[i]:]
		for int(lcp[i]) < len(a) && int(lcp[i]) < len(b) && a[lcp[i]] == b[lcp[i]] {
			lcp[i]++
		}
	}
	return lcp
}

// randomInputs returns 500 sequences of length 0 to 300 for each alphabet size, with the alphabet size of each.
func randomInputs() (seqs [][]int32, alphabets []int) {
	rng := rand.New(rand.NewSource(1))
	for _, alphabet := range []int{2, 5, 40} {
		for range 500 {
			s := make([]int32, rng.Intn(301))
			for i := range s {
				s[i] = int32(rng.Intn(alphabet))
			}
			seqs = append(seqs, s)
			alphabets = append(alphabets, alphabet)
		}
	}
	return seqs, alphabets
}

func TestSuffixArrayRandom(t *testing.T) {
	seqs, alphabets := randomInputs()
	for i, s := range seqs {
		if got, want := suffixArray(s, alphabets[i], new(int)), naiveSuffixArray(s); !reflect.DeepEqual(got, want) {
			t.Fatalf("suffixArray(%v):\n got %v\nwant %v", s, got, want)
		}
	}
}

func TestSuffixArrayEdgeCases(t *testing.T) {
	increasing := make([]int32, 50)
	for i := range increasing {
		increasing[i] = int32(i)
	}
	for _, tc := range []struct {
		name     string
		s        []int32
		alphabet int
	}{
		{"empty", []int32{}, 1},
		{"one symbol", []int32{0}, 1},
		{"two symbols", []int32{1, 0}, 2},
		{"two equal symbols", []int32{0, 0}, 1},
		{"all equal", []int32{3, 3, 3, 3, 3, 3, 3, 3, 3}, 4},
		{"strictly increasing", increasing, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := suffixArray(tc.s, tc.alphabet, new(int)), naiveSuffixArray(tc.s); !reflect.DeepEqual(got, want) {
				t.Errorf("suffixArray(%v):\n got %v\nwant %v", tc.s, got, want)
			}
		})
	}
}

// TestSuffixArrayWork counts one unit per symbol at every level of the recursion, so the count lies between n and 2n.
func TestSuffixArrayWork(t *testing.T) {
	s := []int32{1, 0, 1, 0, 1, 0, 1, 0, 2, 1, 0, 1, 0}
	work := 0
	suffixArray(s, 3, &work)
	if work < len(s) || work > 2*len(s) {
		t.Errorf("work = %d, want between %d and %d", work, len(s), 2*len(s))
	}
}

func TestLCPArrayRandom(t *testing.T) {
	seqs, _ := randomInputs()
	for _, s := range seqs {
		sa := naiveSuffixArray(s)
		work := 0
		if got, want := lcpArray(s, sa, &work), naiveLCP(s, sa); !reflect.DeepEqual(got, want) {
			t.Fatalf("lcpArray(%v):\n got %v\nwant %v", s, got, want)
		}
		if work != len(s) {
			t.Fatalf("work = %d, want %d", work, len(s))
		}
	}
}

package duplication

import (
	"fmt"
	"go/token"
	"math/rand"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/formula"
)

// naiveGroups finds the maximal repeats by a scan of every suffix array range and drops subsumed groups by a comparison of every member pair.
func naiveGroups(s *stream, sa, lcp []int32) []cloneGroup {
	var candidates []cloneGroup
	for lb := 0; lb < len(sa); lb++ {
		length := int32(1 << 30)
		for rb := lb + 1; rb < len(sa); rb++ {
			length = min(length, lcp[rb])
			if length < formula.DuplicationMinTokens {
				break
			}
			// A range is an interval when both neighbours share fewer symbols with it.
			if (lb > 0 && lcp[lb] >= length) || (rb+1 < len(sa) && lcp[rb+1] >= length) {
				continue
			}
			preds := map[int32]bool{}
			for _, p := range sa[lb : rb+1] {
				if p == 0 {
					preds[-1] = true
				} else {
					preds[s.symbols[p-1]] = true
				}
			}
			// A run that every occurrence extends to the left is not a maximal repeat.
			if len(preds) < 2 {
				continue
			}
			if members := s.members(sa[lb:rb+1], int(length)); len(members) >= 2 {
				candidates = append(candidates, cloneGroup{int(length), members})
			}
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].length > candidates[j].length })
	var kept []cloneGroup
	for _, g := range candidates {
		subsumed := true
		for _, m := range g.members {
			inside := false
			for _, k := range kept {
				for _, o := range k.members {
					inside = inside || (o.start <= m.start && m.start+g.length <= o.start+k.length)
				}
			}
			subsumed = subsumed && inside
		}
		if !subsumed {
			kept = append(kept, g)
		}
	}
	return kept
}

// randomStream builds files from a few blocks with random edits, so that long repeats nest, overlap, and repeat periodically.
func randomStream(rng *rand.Rand) *stream {
	var blocks [][]int32
	for range 3 {
		b := make([]int32, 40+rng.Intn(120))
		for i := range b {
			b[i] = int32(token.IDENT + token.Token(rng.Intn(4)))
			if rng.Intn(5) == 0 {
				b[i] = int32(token.SEMICOLON)
			}
		}
		blocks = append(blocks, b)
	}
	s := &stream{}
	for f := range 1 + rng.Intn(4) {
		s.files = append(s.files, streamFile{path: fmt.Sprintf("f%d.go", f), start: len(s.symbols)})
		line := int32(1)
		for range rng.Intn(8) {
			b := blocks[rng.Intn(len(blocks))]
			if rng.Intn(3) == 0 {
				b = append([]int32{int32(token.IDENT + token.Token(rng.Intn(4)))}, b[rng.Intn(20):]...)
			}
			for _, sym := range b {
				s.symbols = append(s.symbols, sym)
				s.lines = append(s.lines, line)
				s.endLines = append(s.endLines, line)
				s.inserted = append(s.inserted, sym == int32(token.SEMICOLON))
				if sym == int32(token.SEMICOLON) {
					line++
				}
			}
		}
		s.symbols = append(s.symbols, int32(sentinelBase+f))
		s.lines = append(s.lines, 0)
		s.endLines = append(s.endLines, 0)
		s.inserted = append(s.inserted, false)
	}
	return s
}

func findGroups(s *stream, maxWork int) ([]cloneGroup, *budget, []int32, []int32) {
	work := &budget{max: maxWork}
	sa := suffixArray(s.symbols, s.alphabet(), &work.used)
	lcp := lcpArray(s.symbols, sa, &work.used)
	return s.groups(intervalTree(sa, lcp, work), sa, work), work, sa, lcp
}

// TestGroupsMatchNaive compares the linear walk over the interval tree with a direct search for the kept groups.
func TestGroupsMatchNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	found := 0
	byLength := func(gs []cloneGroup) {
		sort.Slice(gs, func(i, j int) bool {
			return gs[i].length > gs[j].length || (gs[i].length == gs[j].length && gs[i].members[0].start < gs[j].members[0].start)
		})
	}
	for range 300 {
		s := randomStream(rng)
		got, _, sa, lcp := findGroups(s, 1<<40)
		want := naiveGroups(s, sa, lcp)
		byLength(got)
		byLength(want)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("symbols %v:\n got %+v\nwant %+v", s.symbols, got, want)
		}
		found += len(got)
	}
	if found < 100 {
		t.Errorf("the random streams hold %d groups, too few to test the walk", found)
	}
}

// TestPeriodicWorkIsLinear runs a table of 5000 similar rows, which costs quadratic work when every interval checks every occurrence.
func TestPeriodicWorkIsLinear(t *testing.T) {
	var src strings.Builder
	src.WriteString("package p\n\nvar rows = []row{\n")
	for i := range 5000 {
		fmt.Fprintf(&src, "\t{name: \"w%d\", code: %d},\n", i, i)
	}
	src.WriteString("}\n\ntype row struct {\n\tname string\n\tcode int\n}\n")
	inv := buildSource(t, map[string]string{"go.mod": "module example.com/m\n", "rows.go": src.String()})
	metrics, findings, limits := Measure(inv)
	if len(limits) != 0 || len(findings) != 1 || metrics["duplication.groups.production"].State != contract.Complete {
		t.Fatalf("limits %v, %d findings, want one complete group", limits, len(findings))
	}
	s := newStream(inv, contract.Production, 1<<30)
	if _, work, _, _ := findGroups(s, 1<<40); work.used > 5*len(s.symbols) {
		t.Errorf("work %d for %d symbols, want at most 5 per symbol", work.used, len(s.symbols))
	}
}

// TestWorkCapInWalk stops the walk when the budget runs out between the interval tree and the containment queries.
func TestWorkCapInWalk(t *testing.T) {
	s := newStream(build(t, filepath.Join(fixtures, "clones", "nested")), contract.Production, 1<<30)
	_, full, _, _ := findGroups(s, 1<<40)
	index := 0
	suffixArray(s.symbols, s.alphabet(), &index)
	index += len(s.symbols)
	for limit := index; limit < full.used; limit++ {
		if got, work, _, _ := findGroups(s, limit); got != nil || !work.exceeded() {
			t.Fatalf("cap %d: %d groups, work %d, want none past the cap", limit, len(got), work.used)
		}
	}
}

// symbolAt is one symbol of a synthetic file with its line.
type symbolAt struct{ sym, line int32 }

// synthetic builds a stream from files of symbols with lines and no inserted semicolons.
func synthetic(files ...[]symbolAt) *stream {
	s := &stream{}
	for f, file := range files {
		s.files = append(s.files, streamFile{path: fmt.Sprintf("f%d.go", f), start: len(s.symbols)})
		for _, t := range file {
			s.symbols = append(s.symbols, t.sym)
			s.lines = append(s.lines, t.line)
			s.endLines = append(s.endLines, t.line)
			s.inserted = append(s.inserted, false)
		}
		s.symbols = append(s.symbols, int32(sentinelBase+f))
		s.lines = append(s.lines, 0)
		s.endLines = append(s.endLines, 0)
		s.inserted = append(s.inserted, false)
	}
	return s
}

// randomRun returns n random identifier-class symbols that start on line first and move to the next line at each break.
func randomRun(rng *rand.Rand, n int, first int32, breaks ...int) []symbolAt {
	out := make([]symbolAt, n)
	line := first
	for i := range out {
		if slices.Contains(breaks, i) {
			line++
		}
		out[i] = symbolAt{int32(token.IDENT + token.Token(rng.Intn(4))), line}
	}
	return out
}

// onLine moves every symbol to one line.
func onLine(syms []symbolAt, line int32) []symbolAt {
	out := make([]symbolAt, len(syms))
	for i, t := range syms {
		out[i] = symbolAt{t.sym, line}
	}
	return out
}

// shift moves the lines of symbols by delta.
func shift(syms []symbolAt, delta int32) []symbolAt {
	out := make([]symbolAt, len(syms))
	for i, t := range syms {
		out[i] = symbolAt{t.sym, t.line + delta}
	}
	return out
}

func describe(gs []cloneGroup) []string {
	var out []string
	for _, g := range gs {
		d := fmt.Sprint(g.length)
		for _, m := range g.members {
			d += fmt.Sprintf(" %s:%d-%d", m.path, m.startLine, m.endLine)
		}
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func checkBoth(t *testing.T, s *stream, want []string) {
	t.Helper()
	got, _, sa, lcp := findGroups(s, 1<<40)
	if d := describe(got); !reflect.DeepEqual(d, want) {
		t.Errorf("groups = %v, want %v", d, want)
	}
	if d := describe(naiveGroups(s, sa, lcp)); !reflect.DeepEqual(d, want) {
		t.Errorf("naive groups = %v, want %v", d, want)
	}
}

// TestCoveredBoundary keeps a run with one uncovered occurrence, because a covered occurrence spans exactly DuplicationMinLines lines at the run length.
func TestCoveredBoundary(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	// x spans lines 1 to 3 in exactly 100 symbols.
	x := randomRun(rng, 100, 1, 50, 99)
	outer := append(append(randomRun(rng, 30, 1), shift(x, 1)...), randomRun(rng, 30, 5)...)
	alone := append(append([]symbolAt{{int32(token.STRING), 1}}, x...), symbolAt{int32(token.CHAR), 3})
	checkBoth(t, synthetic(outer, outer, alone), []string{"100 f0.go:2-4 f1.go:2-4 f2.go:1-3", "160 f0.go:1-5 f1.go:1-5"})
	// With x on two lines inside the copies, the covered occurrences are too short, and the lone occurrence forms no group.
	flatX := append(append(randomRun(rng, 30, 1), onLine(x, 2)...), randomRun(rng, 30, 3)...)
	checkBoth(t, synthetic(flatX, flatX, alone), []string{"160 f0.go:1-3 f1.go:1-3"})
}

// TestUncoveredChild passes the one valid occurrence of a dropped longer run to the shorter run that holds it.
func TestUncoveredChild(t *testing.T) {
	rng := rand.New(rand.NewSource(8))
	b := randomRun(rng, 110, 1, 40, 80)
	// The copy on one line spans too few lines at any length, so the 110-symbol run keeps one valid occurrence.
	c := append(append([]symbolAt{}, b[:105]...), symbolAt{int32(token.STRING), 3})
	checkBoth(t, synthetic(b, onLine(b, 1), c), []string{"105 f0.go:1-3 f2.go:1-3"})
}

// TestParentLinkSavesWork runs a stream whose nested intervals close at one suffix array index.
// Without the link from each closed interval to its parent, a parent queries the covered occurrences of its child again.
// The work then passes 4 units per symbol.
func TestParentLinkSavesWork(t *testing.T) {
	s := randomStream(rand.New(rand.NewSource(124)))
	if _, work, _, _ := findGroups(s, 1<<40); work.used > 4*len(s.symbols) {
		t.Errorf("work %d for %d symbols, want at most 4 per symbol", work.used, len(s.symbols))
	}
}

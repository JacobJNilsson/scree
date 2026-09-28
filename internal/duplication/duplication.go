// Package duplication finds clone groups in the normalized token streams of the measured source sets.
package duplication

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/scanner"
	"go/token"
	"hash/fnv"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/JacobJNilsson/scree/internal/contract"
	"github.com/JacobJNilsson/scree/internal/discover"
	"github.com/JacobJNilsson/scree/internal/formula"
	"github.com/JacobJNilsson/scree/internal/inventory"
)

// KindCloneGroup is the finding kind of a clone group.
const KindCloneGroup = "duplication.clone-group"

// sentinelBase lies above every token kind, and file i ends with the symbol sentinelBase + i.
const sentinelBase = 1000

const (
	unitCount = "count"
	unitRatio = "ratio"
)

// The metric names of one set, without the set suffix.
const (
	metricGroups = "duplication.groups"
	metricLines  = "duplication.duplicated-lines"
	metricRatio  = "duplication.density"
)

// Measure returns the duplication metrics of both measured sets, one finding per clone group, and one limit per metric that a budget cap stopped.
func Measure(inv *inventory.Inventory) (map[string]contract.Metric, []contract.Finding, []contract.Limit) {
	return measureWithCaps(inv, formula.DuplicationMaxTokens, formula.DuplicationMaxWork)
}

func measureWithCaps(inv *inventory.Inventory, maxTokens, maxWork int) (map[string]contract.Metric, []contract.Finding, []contract.Limit) {
	metrics := map[string]contract.Metric{}
	findings := []contract.Finding{}
	limits := []contract.Limit{}
	for _, set := range []contract.SourceSet{contract.Production, contract.Test} {
		r := measureSet(inv, set, maxTokens, maxWork)
		detail := contract.Detail{Errors: inv.ErrorPaths(set), Limit: r.limit}
		for name, m := range r.metrics {
			id := name + "." + string(set)
			if detail.Errors != nil || detail.Limit != nil {
				m = contract.Metric{State: contract.Incomplete, Unit: m.Unit, Detail: detail}
			}
			if r.limit != nil {
				limits = append(limits, contract.Limit{MetricID: id, Reason: r.reason})
			}
			metrics[id] = m
		}
		findings = append(findings, r.findings...)
	}
	contract.SortFindings(findings)
	sortLimits(limits)
	return metrics, findings, limits
}

// setResult is the measurement of one set, where a set past a cap has a limit, no findings, and metrics without values.
type setResult struct {
	metrics  map[string]contract.Metric
	findings []contract.Finding
	limit    *contract.LimitDetail
	reason   string
}

func measureSet(inv *inventory.Inventory, set contract.SourceSet, maxTokens, maxWork int) setResult {
	stopped := func(capName string, limit, observed int) setResult {
		return setResult{
			metrics: map[string]contract.Metric{
				metricGroups: {Unit: unitCount}, metricLines: {Unit: unitCount}, metricRatio: {Unit: unitRatio},
			},
			limit:  &contract.LimitDetail{Cap: capName, Observed: observed},
			reason: fmt.Sprintf("%s cap %d exceeded", capName, limit),
		}
	}
	s := newStream(inv, set, maxTokens)
	if s.tokens > maxTokens {
		return stopped("tokens", maxTokens, s.tokens)
	}
	// The suffix array costs one work unit per symbol at each level, and the LCP array costs one unit per symbol.
	// Each LCP interval of at least DuplicationMinTokens symbols costs one unit, and each containment query costs one unit.
	work := &budget{max: maxWork}
	sa := suffixArray(s.symbols, s.alphabet(), &work.used)
	lcp := lcpArray(s.symbols, sa, &work.used)
	nodes := intervalTree(sa, lcp, work)
	groups := s.groups(nodes, sa, work)
	if work.exceeded() {
		return stopped("work", maxWork, work.used)
	}
	var findings []contract.Finding
	var covered []member
	for _, g := range groups {
		findings = append(findings, s.finding(set, g))
		covered = append(covered, g.members...)
	}
	lines := s.duplicatedLines(covered)
	density := contract.Metric{State: contract.NotApplicable, Unit: unitRatio}
	if sloc := inv.SLOC[set]; sloc > 0 {
		density = contract.Metric{
			State: contract.Complete, Value: float64(lines) / float64(sloc), Unit: unitRatio,
			Numerator: float64(lines), Denominator: float64(sloc),
		}
	}
	return setResult{
		metrics: map[string]contract.Metric{
			metricGroups: {State: contract.Complete, Value: float64(len(findings)), Unit: unitCount},
			metricLines:  {State: contract.Complete, Value: float64(lines), Unit: unitCount},
			metricRatio:  density,
		},
		findings: findings,
	}
}

// stream is the symbol sequence of one set, with the lines of each symbol.
type stream struct {
	symbols []int32
	// lines and endLines hold the first and the last line of the symbol at each position, and 0 for a sentinel.
	lines    []int32
	endLines []int32
	// inserted marks the semicolons that the scanner inserts at a line end.
	inserted []bool
	files    []streamFile
	// tokens counts the symbols of every file, also the ones past the cap that the stream does not keep.
	// The count and the token cap leave out the file sentinels.
	tokens int
}

// streamFile is one file of a stream.
type streamFile struct {
	path string
	// start is the position of the first symbol of the file.
	start int
	// codeLines[l] counts the code lines before line l, by the rule of the inventory.
	codeLines []int32
}

// newStream scans the parsed files of a set in path order and stops keeping symbols once their count passes maxTokens.
func newStream(inv *inventory.Inventory, set contract.SourceSet, maxTokens int) *stream {
	var files []*discover.File
	for i := range inv.Tree.Files {
		if f := &inv.Tree.Files[i]; f.Set == set && f.ParseErr == nil {
			files = append(files, f)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	s := &stream{}
	for _, f := range files {
		s.addFile(f, dropRanges(f.Syntax, inv.Tree.Fset.File(f.Syntax.Pos())), inv.CodeLines[f.Path], maxTokens)
	}
	return s
}

// span is a half-open range of byte offsets.
type span struct{ start, end int }

// dropRanges returns the byte ranges of the package clause and of every import declaration, in source order.
func dropRanges(syntax *ast.File, file *token.File) []span {
	ranges := []span{{file.Offset(syntax.Package), file.Offset(syntax.Name.End())}}
	for _, decl := range syntax.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			ranges = append(ranges, span{file.Offset(gen.Pos()), file.Offset(gen.End())})
		}
	}
	return ranges
}

func (s *stream) addFile(f *discover.File, drops []span, code []bool, maxTokens int) {
	// A scratch file set keeps the scan from adding files to the tree's file set.
	file := token.NewFileSet().AddFile(f.Path, -1, len(f.Src))
	var sc scanner.Scanner
	sc.Init(file, f.Src, nil, 0)
	sf := streamFile{path: f.Path, start: len(s.symbols)}
	next, afterDrop := 0, false
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		offset := file.Offset(pos)
		for next < len(drops) && offset >= drops[next].end {
			next++
		}
		if next < len(drops) && offset >= drops[next].start {
			afterDrop = true
			continue
		}
		if afterDrop && tok == token.SEMICOLON {
			afterDrop = false
			continue
		}
		afterDrop = false
		s.tokens++
		if s.tokens > maxTokens {
			continue
		}
		// PositionFor without adjustment keeps a //line comment from moving the real lines.
		line := file.PositionFor(pos, false).Line
		end := line
		if tok == token.STRING {
			end += strings.Count(lit, "\n")
		}
		s.symbols = append(s.symbols, int32(tok))
		s.lines = append(s.lines, int32(line))
		s.endLines = append(s.endLines, int32(end))
		s.inserted = append(s.inserted, tok == token.SEMICOLON && lit == "\n")
	}
	if s.tokens > maxTokens {
		return
	}
	sf.codeLines = make([]int32, len(code)+1)
	for l, isCode := range code {
		sf.codeLines[l+1] = sf.codeLines[l]
		if isCode {
			sf.codeLines[l+1]++
		}
	}
	s.symbols = append(s.symbols, int32(sentinelBase+len(s.files)))
	s.lines = append(s.lines, 0)
	s.endLines = append(s.endLines, 0)
	s.inserted = append(s.inserted, false)
	s.files = append(s.files, sf)
}

func (s *stream) alphabet() int {
	return sentinelBase + len(s.files)
}

// fileAt returns the index of the file that holds a position.
func (s *stream) fileAt(pos int) int {
	return sort.Search(len(s.files), func(i int) bool { return s.files[i].start > pos }) - 1
}

// budget counts work units and stops counting once they pass the maximum.
type budget struct {
	used, max int
}

// spend adds units and reports whether the work stays within the maximum.
func (b *budget) spend(units int) bool {
	b.used += units
	return !b.exceeded()
}

func (b *budget) exceeded() bool {
	return b.used > b.max
}

// node is an LCP interval of at least DuplicationMinTokens symbols.
// The suffixes in the suffix array range [lb, rb] share exactly length symbols.
type node struct {
	lb, rb, length int32
	// parent is the index of the smallest enclosing node, or -1.
	parent int32
}

// intervalTree walks the LCP-interval tree bottom-up and returns every interval of at least DuplicationMinTokens symbols, each with its parent.
// It returns nil once the budget runs out.
func intervalTree(sa, lcp []int32, work *budget) []node {
	if work.exceeded() {
		return nil
	}
	type frame struct {
		lcp, lb int32
		// node is the index of the node of the frame, or -1 below the threshold.
		node int32
	}
	var nodes []node
	push := func(stack []frame, lcp, lb int32) []frame {
		id := int32(-1)
		if lcp >= formula.DuplicationMinTokens {
			id = int32(len(nodes))
			nodes = append(nodes, node{lb: lb, length: lcp, parent: -1})
		}
		return append(stack, frame{lcp, lb, id})
	}
	stack := []frame{{0, 0, -1}}
	for i := 1; i <= len(sa); i++ {
		// The final value 0 closes every open interval.
		cur := int32(0)
		if i < len(sa) {
			cur = lcp[i]
		}
		lb := int32(i - 1)
		last := int32(-1)
		for cur < stack[len(stack)-1].lcp {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			lb = top.lb
			if last >= 0 {
				nodes[last].parent = top.node
			}
			last = top.node
			if top.node >= 0 {
				nodes[top.node].rb = int32(i - 1)
				if !work.spend(1) {
					return nil
				}
			}
		}
		if cur > stack[len(stack)-1].lcp {
			stack = push(stack, cur, lb)
		}
		if last >= 0 {
			nodes[last].parent = stack[len(stack)-1].node
		}
	}
	return nodes
}

// member is one occurrence of a clone group.
type member struct {
	path               string
	file               int
	startLine, endLine int
	// start is the stream position of the first symbol.
	start int
}

// cloneGroup is a run of length symbols with the members that pass the thresholds.
type cloneGroup struct {
	length  int
	members []member
}

// groups returns the intervals with at least two members, without a group whose every member lies inside a member of a longer group.
// An interval that is not left-maximal needs no check of its own, because the longer run that extends it to the left subsumes it.
// It visits the nodes longest first, so every child comes before its parent.
// An occurrence inside a kept or subsumed child needs no query, so every position takes one query, plus one per dropped node.
// It returns nil once the budget runs out.
func (s *stream) groups(nodes []node, sa []int32, work *budget) []cloneGroup {
	if work.exceeded() {
		return nil
	}
	minLen := s.minLengths()
	children := childLists(nodes)
	order := make([]int32, len(nodes))
	for i := range order {
		order[i] = int32(i)
	}
	// Two groups of equal length have no member inside each other, so their order does not matter.
	slices.SortStableFunc(order, func(a, b int32) int { return cmp.Compare(nodes[b].length, nodes[a].length) })
	// covered is the smallest minimum length of an occurrence below a node that lies inside a kept member.
	covered := make([]int32, len(nodes))
	// open holds the occurrences below a node that are not known to lie inside a kept member.
	open := make([][]int32, len(nodes))
	for i := range nodes {
		covered[i] = math.MaxInt32
	}
	ends := make(maxEnds, len(s.symbols)+1)
	var kept []cloneGroup
	for _, k := range order {
		n := nodes[k]
		next := n.lb
		for _, c := range append(children[k], -1) {
			end := n.rb + 1
			if c >= 0 {
				end = nodes[c].lb
			}
			open[k] = append(open[k], sa[next:end]...)
			if c >= 0 {
				next = nodes[c].rb + 1
			}
		}
		length := n.length
		var uncovered []int32
		for _, p := range open[k] {
			if minLen[p] > length {
				continue
			}
			if !work.spend(1) {
				return nil
			}
			if ends.upTo(int(p)) >= int(p+length) {
				covered[k] = min(covered[k], minLen[p])
				continue
			}
			uncovered = append(uncovered, p)
		}
		open[k] = uncovered
		if len(uncovered) >= 2 || (len(uncovered) == 1 && covered[k] <= length) {
			members := s.members(sa[n.lb:n.rb+1], int(length))
			for _, m := range members {
				ends.add(m.start, m.start+int(length))
				covered[k] = min(covered[k], minLen[m.start])
			}
			kept = append(kept, cloneGroup{length: int(length), members: members})
			open[k] = nil
		}
		if parent := n.parent; parent >= 0 {
			covered[parent] = min(covered[parent], covered[k])
			open[parent] = append(open[parent], open[k]...)
		}
		open[k] = nil
	}
	return kept
}

// childLists returns the children of each node in suffix array order, which is the order in which the walk creates them.
func childLists(nodes []node) [][]int32 {
	children := make([][]int32, len(nodes))
	for i, n := range nodes {
		if n.parent >= 0 {
			children[n.parent] = append(children[n.parent], int32(i))
		}
	}
	return children
}

// minLengths returns, for each position, the fewest symbols that a run from it needs to span DuplicationMinLines lines, or MaxInt32.
// A run spans lines from its first symbol that is not an inserted semicolon to its last symbol, as in members.
func (s *stream) minLengths() []int32 {
	out := make([]int32, len(s.symbols))
	for i := range out {
		out[i] = math.MaxInt32
	}
	for f, file := range s.files {
		// The sentinel at end is never part of a run.
		end := len(s.symbols) - 1
		if f+1 < len(s.files) {
			end = s.files[f+1].start - 1
		}
		first := end
		for p := end - 1; p >= file.start; p-- {
			if !s.inserted[p] {
				first = p
			}
			if first == end {
				continue
			}
			target := s.lines[first] + formula.DuplicationMinLines - 1
			// The end lines rise within a file, and the first symbol that reaches the target line is never an inserted semicolon.
			q := first + sort.Search(end-first, func(i int) bool { return s.endLines[first+i] >= target })
			if q < end {
				out[p] = int32(q - p + 1)
			}
		}
	}
	return out
}

// maxEnds is a Fenwick tree that returns the largest end of the kept member ranges that start at or before a position.
type maxEnds []int

func (t maxEnds) add(start, end int) {
	for i := start + 1; i < len(t); i += i & -i {
		t[i] = max(t[i], end)
	}
}

func (t maxEnds) upTo(pos int) int {
	end := 0
	for i := pos + 1; i > 0; i -= i & -i {
		end = max(end, t[i])
	}
	return end
}

// members returns the occurrences of a run of length symbols that span at least DuplicationMinLines lines, sorted.
// The line range skips inserted semicolons at the start of the run, which lie on the line before it.
func (s *stream) members(positions []int32, length int) []member {
	var out []member
	for _, p := range positions {
		first, last := int(p), int(p)+length-1
		for first < last && s.inserted[first] {
			first++
		}
		m := member{file: s.fileAt(int(p)), startLine: int(s.lines[first]), endLine: int(s.endLines[last]), start: int(p)}
		m.path = s.files[m.file].path
		if m.endLine-m.startLine+1 >= formula.DuplicationMinLines {
			out = append(out, m)
		}
	}
	sortMembers(out)
	return out
}

// sortMembers orders members by path, start line, and end line, then by stream position, which keeps its order within one file.
func sortMembers(members []member) {
	slices.SortFunc(members, func(a, b member) int {
		return cmp.Or(strings.Compare(a.path, b.path), cmp.Compare(a.startLine, b.startLine), cmp.Compare(a.endLine, b.endLine), cmp.Compare(a.start, b.start))
	})
}

func (s *stream) finding(set contract.SourceSet, g cloneGroup) contract.Finding {
	members := g.members
	id := groupID(s.symbols[members[0].start : members[0].start+g.length])
	facts := &contract.CloneFacts{GroupID: id, Tokens: g.length}
	for _, m := range members {
		facts.Members = append(facts.Members, contract.CloneMember{Path: m.path, StartLine: m.startLine, EndLine: m.endLine})
	}
	return contract.Finding{
		Kind: KindCloneGroup, Path: members[0].path, StartLine: members[0].startLine, EndLine: members[0].endLine,
		Identity: id, SourceSet: set, Facts: contract.Facts{Clone: facts},
	}
}

// groupID hashes the token names of a run with FNV-1a, so that the id depends on the names and not on the numbering of go/token.
func groupID(symbols []int32) string {
	h := fnv.New64a()
	for _, sym := range symbols {
		// Writes to a hash never fail.
		_, _ = h.Write([]byte(token.Token(sym).String()))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%016x", h.Sum64())
}

// duplicatedLines counts the code lines that at least one member covers, where each line counts once.
func (s *stream) duplicatedLines(members []member) int {
	sort.Slice(members, func(i, j int) bool {
		if members[i].file != members[j].file {
			return members[i].file < members[j].file
		}
		return members[i].startLine < members[j].startLine
	})
	count := 0
	// The loop merges the overlapping line ranges of each file and counts the code lines of each merged range.
	for i := 0; i < len(members); {
		file, start, end := members[i].file, members[i].startLine, members[i].endLine
		i++
		for i < len(members) && members[i].file == file && members[i].startLine <= end+1 {
			end = max(end, members[i].endLine)
			i++
		}
		code := s.files[file].codeLines
		count += int(code[end+1] - code[start])
	}
	return count
}

func sortLimits(limits []contract.Limit) {
	sort.Slice(limits, func(i, j int) bool { return limits[i].MetricID < limits[j].MetricID })
}

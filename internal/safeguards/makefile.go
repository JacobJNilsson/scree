package safeguards

import (
	"regexp"
	"slices"
	"strings"
)

// Makefile is the target and variable model of the root Makefile.
type Makefile struct {
	Path    string   `json:"path"`
	Targets []Target `json:"targets"`
	Vars    []Var    `json:"vars"`
	// Includes is true when a top-level include directive reads another file, which may define targets the inspector cannot see.
	Includes bool `json:"includes"`
}

// Target is one rule of a Makefile.
type Target struct {
	Name    string    `json:"name"`
	Line    int       `json:"line"`
	Prereqs []string  `json:"prereqs"`
	Recipe  []Command `json:"recipe"`
	// Unknown is true when a recipe line holds a construct that a text read cannot follow.
	Unknown bool `json:"unknown"`
}

// Var is one variable assignment of a Makefile.
type Var struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Line  int    `json:"line"`
}

var assignment = regexp.MustCompile(`^(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*(?:\?=|:=|\+=|=)\s*(.*)$`)

// readMakefile reads GNUmakefile before Makefile, because GNU make reads that name first.
func (r *reader) readMakefile() *Makefile {
	name := r.firstFile("GNUmakefile", "Makefile")
	text, ok := r.read(name)
	if !ok {
		return nil
	}
	m := &Makefile{Path: name, Targets: []Target{}, Vars: []Var{}}
	// current holds the indexes of the targets that the next recipe line belongs to.
	var current []int
	for _, l := range joinLines(text) {
		current = m.addLine(current, l)
	}
	return m
}

// addLine records one logical line and returns the targets that the next recipe line belongs to.
func (m *Makefile) addLine(current []int, l sourceLine) []int {
	word := firstWord(l.Text)
	switch {
	// An include inside a conditional is indented, so every line counts.
	case word == "include" || word == "-include" || word == "sinclude":
		m.Includes = true
	// The inspector reads every branch of a conditional as taken, so a directive leaves the current target open.
	case conditionals[word]:
	case strings.HasPrefix(l.Text, "\t"):
		m.addRecipe(current, l)
	case l.Text == "" || strings.HasPrefix(l.Text, " ") || strings.HasPrefix(l.Text, "#"):
	default:
		return m.addRule(l)
	}
	return current
}

func (m *Makefile) addRecipe(current []int, l sourceLine) {
	text := strings.TrimSpace(l.Text)
	if text == "" || strings.HasPrefix(text, "#") {
		return
	}
	unknown := strings.Contains(text, "$(shell") || strings.Contains(text, "`")
	commands := parseLine(l.Line, text)
	for _, i := range current {
		m.Targets[i].Recipe = append(m.Targets[i].Recipe, commands...)
		m.Targets[i].Unknown = m.Targets[i].Unknown || unknown
	}
}

// addRule records a variable or a rule line, and it returns the indexes of the targets that the rule opens.
func (m *Makefile) addRule(l sourceLine) []int {
	if match := assignment.FindStringSubmatch(l.Text); match != nil {
		m.Vars = append(m.Vars, Var{Name: match[1], Value: strings.TrimSpace(match[2]), Line: l.Line})
		return nil
	}
	names, prereqs, ok := strings.Cut(l.Text, ":")
	if !ok {
		return nil
	}
	prereqs = strings.TrimPrefix(prereqs, ":")
	prereqs, _, _ = strings.Cut(prereqs, ";")
	var opened []int
	for _, name := range strings.Fields(names) {
		// A dot-target such as .PHONY names special rules, so its recipe belongs to no target.
		if strings.HasPrefix(name, ".") {
			continue
		}
		i := slices.IndexFunc(m.Targets, func(t Target) bool { return t.Name == name })
		if i < 0 {
			i = len(m.Targets)
			m.Targets = append(m.Targets, Target{Name: name, Line: l.Line, Prereqs: []string{}, Recipe: []Command{}})
		}
		// A second rule adds its prerequisites and recipe lines, which over-approximates GNU make, because make keeps only the last recipe.
		m.Targets[i].Prereqs = append(m.Targets[i].Prereqs, strings.Fields(prereqs)...)
		opened = append(opened, i)
	}
	return opened
}

// conditionals are the directive words of make conditionals.
var conditionals = map[string]bool{"ifeq": true, "ifneq": true, "ifdef": true, "ifndef": true, "else": true, "endif": true}

func firstWord(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

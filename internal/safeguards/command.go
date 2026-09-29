package safeguards

import (
	"path"
	"regexp"
	"strconv"
	"strings"
)

// CommandKind names the form of one command, as spec 002 lists the understood forms.
type CommandKind string

// The command forms of spec 002, and KindOther for every line outside them.
const (
	KindMake             CommandKind = "make"
	KindScript           CommandKind = "script"
	KindGoVet            CommandKind = "go-vet"
	KindGoTest           CommandKind = "go-test"
	KindGoBuild          CommandKind = "go-build"
	KindGoRun            CommandKind = "go-run"
	KindGolangci         CommandKind = "golangci"
	KindGofmt            CommandKind = "gofmt"
	KindHooksPath        CommandKind = "hooks-path"
	KindLefthookInstall  CommandKind = "lefthook-install"
	KindPreCommitInstall CommandKind = "pre-commit-install"
	KindOther            CommandKind = "other"
)

// Command is one command of a recipe, hook, or workflow line.
type Command struct {
	Line int         `json:"line"`
	Text string      `json:"text"`
	Kind CommandKind `json:"kind"`
	// Ref is the make target, the script path, or the hooks directory, and it is empty for the other kinds.
	Ref string `json:"ref,omitempty"`
}

// sourceLine is one logical line, with the number of its first physical line.
type sourceLine struct {
	Line int
	Text string
}

// joinLines splits text into logical lines, and a line that ends in a backslash continues on the next one.
func joinLines(text string) []sourceLine {
	var out []sourceLine
	var cur *sourceLine
	for i, raw := range strings.Split(text, "\n") {
		if cur != nil {
			if next := strings.TrimLeft(raw, " \t"); next != "" {
				cur.Text += " " + next
			}
		} else {
			out = append(out, sourceLine{Line: i + 1, Text: raw})
			cur = &out[len(out)-1]
		}
		if !strings.HasSuffix(cur.Text, "\\") {
			cur = nil
			continue
		}
		cur.Text = strings.TrimRight(strings.TrimSuffix(cur.Text, "\\"), " \t")
	}
	if n := len(out); n > 0 && out[n-1].Text == "" && strings.HasSuffix(text, "\n") {
		out = out[:n-1]
	}
	return out
}

// parseScript reads the commands of a shell script whose first line has the number first.
func parseScript(text string, first int) []Command {
	commands := []Command{}
	for _, l := range joinLines(text) {
		trimmed := strings.TrimSpace(l.Text)
		// A shebang, a comment, and a set of shell options run no check, so they are no commands.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "set" || strings.HasPrefix(trimmed, "set ") {
			continue
		}
		commands = append(commands, parseLine(l.Line+first-1, trimmed)...)
	}
	return commands
}

// shellOperators are the operators that make a line one other command, because the inspector cannot tell which part runs.
var shellOperators = []string{"||", "|", ";", "&", ">", "<", "`"}

// parseLine splits one line at "&&" and classifies each part, and a line with any other shell operator is one other command.
func parseLine(line int, text string) []Command {
	// The scan removes each "&&" first, because the lone "&" operator would match it.
	rest := strings.ReplaceAll(text, "&&", "")
	for _, op := range shellOperators {
		if strings.Contains(rest, op) {
			return []Command{{Line: line, Text: strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(text), "@-")), Kind: KindOther}}
		}
	}
	var out []Command
	for _, part := range strings.Split(text, "&&") {
		part = strings.TrimLeft(strings.TrimSpace(part), "@-")
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kind, refs := classify(strings.Fields(part))
		// A make line with several targets gives one command per target, so reach follows each of them.
		for _, ref := range refs {
			out = append(out, Command{Line: line, Text: part, Kind: kind, Ref: ref})
		}
	}
	return out
}

// toolKinds are the commands whose first word alone decides the kind.
var toolKinds = map[string]CommandKind{"golangci-lint": KindGolangci, "gofmt": KindGofmt}

// classify names the form of one command and its references, and a kind without a reference gets one empty reference.
func classify(tokens []string) (CommandKind, []string) {
	none := []string{""}
	if kind, ok := toolKinds[tokens[0]]; ok {
		return kind, none
	}
	switch tokens[0] {
	case "make", "$(MAKE)", "${MAKE}":
		return makeTargets(tokens[1:])
	case "sh", "bash":
		return scriptKind(tokens[1:])
	case "go":
		return goKind(tokens), none
	case "git":
		kind, dir := hooksPath(tokens)
		return kind, []string{dir}
	case "lefthook", "pre-commit":
		return installKind(tokens), none
	}
	if strings.HasPrefix(tokens[0], "./") {
		return KindScript, []string{path.Clean(tokens[0])}
	}
	return KindOther, none
}

// scriptKind reads the arguments of sh or bash, and a flag such as -c makes the line other.
func scriptKind(args []string) (CommandKind, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return KindOther, []string{""}
	}
	return KindScript, []string{path.Clean(args[0])}
}

// makeFlagsWithArgument are the make flags whose next token is their argument and not a target.
var makeFlagsWithArgument = map[string]bool{
	"-f": true, "--file": true, "--makefile": true,
	"-I": true, "--include-dir": true,
	"-o": true, "--old-file": true, "--assume-old": true,
	"-W": true, "--what-if": true, "--new-file": true, "--assume-new": true,
	"--jobs": true,
}

// makeTargets returns every argument that is neither a flag, a flag argument, nor a variable assignment.
func makeTargets(args []string) (CommandKind, []string) {
	var targets []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			if !strings.Contains(a, "=") {
				targets = append(targets, a)
			}
			continue
		}
		skip, ok := makeFlag(a, args[i+1:])
		if !ok {
			return KindOther, []string{""}
		}
		i += skip
	}
	if len(targets) == 0 {
		return KindMake, []string{""}
	}
	return KindMake, targets
}

// numericFlags are the make flags whose argument is an optional number, so a word after them may be a target.
var numericFlags = map[string]bool{"-j": true, "-l": true}

var joinedNumber = regexp.MustCompile(`^-[jl][0-9]+$`)

// makeFlag returns how many following tokens a flag consumes, and ok is false when the flag makes the line ambiguous.
func makeFlag(flag string, next []string) (skip int, ok bool) {
	switch {
	// The flag runs the Makefile of another directory, which the inspector does not read.
	case strings.HasPrefix(flag, "-C") || strings.HasPrefix(flag, "--directory"):
		return 0, false
	case numericFlags[flag]:
		if len(next) == 0 {
			return 0, true
		}
		if _, err := strconv.Atoi(next[0]); err != nil {
			return 0, false
		}
		return 1, true
	case makeFlagsWithArgument[flag]:
		return 1, true
	// A combined short flag such as -kj hides which letter takes the next token.
	case !strings.HasPrefix(flag, "--") && len(flag) > 2 && !joinedNumber.MatchString(flag):
		return 0, false
	}
	return 0, true
}

func goKind(tokens []string) CommandKind {
	if len(tokens) < 2 {
		return KindOther
	}
	switch tokens[1] {
	case "vet":
		return KindGoVet
	case "test":
		return KindGoTest
	case "build":
		return KindGoBuild
	case "run":
		return KindGoRun
	}
	return KindOther
}

// hooksPath recognises "git config [flags] core.hooksPath <dir>".
func hooksPath(tokens []string) (CommandKind, string) {
	var args []string
	for _, t := range tokens[1:] {
		if !strings.HasPrefix(t, "-") {
			args = append(args, t)
		}
	}
	if len(args) == 3 && args[0] == "config" && args[1] == "core.hooksPath" {
		return KindHooksPath, path.Clean(args[2])
	}
	return KindOther, ""
}

func installKind(tokens []string) CommandKind {
	if len(tokens) < 2 || tokens[1] != "install" {
		return KindOther
	}
	if tokens[0] == "lefthook" {
		return KindLefthookInstall
	}
	return KindPreCommitInstall
}

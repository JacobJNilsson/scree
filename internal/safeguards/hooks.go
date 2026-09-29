package safeguards

import (
	"encoding/json"
	"path"
	"strings"
)

// HookFile is one Git hook script that a tracked directory holds.
type HookFile struct {
	Path  string    `json:"path"`
	Hook  string    `json:"hook"`
	Lines []Command `json:"lines"`
}

// HookTool is a hook manager configuration and the hooks it declares.
type HookTool struct {
	Path  string   `json:"path"`
	Tool  string   `json:"tool"`
	Hooks []string `json:"hooks"`
}

// AgentHooks counts the hook entries of a Claude Code settings file.
type AgentHooks struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Count int    `json:"count"`
}

// hookNames are the Git hooks that a safeguard inspects.
var hookNames = []string{"pre-commit", "pre-push"}

// readHookFiles reads the hooks of every core.hooksPath directory that the Makefile sets, and of .husky, and never .git/hooks.
func (r *reader) readHookFiles() {
	dirs := map[string]bool{".husky": true}
	if m := r.s.Makefile; m != nil {
		for _, t := range m.Targets {
			for _, c := range t.Recipe {
				if c.Kind == KindHooksPath {
					dirs[c.Ref] = true
				}
			}
		}
	}
	for dir := range dirs {
		for _, hook := range hookNames {
			rel := path.Join(dir, hook)
			if text, ok := r.read(rel); ok {
				r.s.HookFiles = append(r.s.HookFiles, HookFile{Path: rel, Hook: hook, Lines: parseScript(text, 1)})
			}
		}
	}
}

func (r *reader) readHookTools() {
	for _, name := range []string{"lefthook.yml", "lefthook.yaml", ".lefthook.yml", ".lefthook.yaml"} {
		if tool, ok := r.readLefthook(name); ok {
			r.s.HookTools = append(r.s.HookTools, tool)
		}
	}
	const preCommitConfig = ".pre-commit-config.yaml"
	if text, ok := r.read(preCommitConfig); ok {
		hooks := []string{"pre-commit"}
		// The pre-commit tool installs a pre-push hook only when a stage names it, so the text alone decides.
		if strings.Contains(text, "pre-push") {
			hooks = append(hooks, "pre-push")
		}
		r.s.HookTools = append(r.s.HookTools, HookTool{Path: preCommitConfig, Tool: "pre-commit", Hooks: hooks})
	}
}

func (r *reader) readLefthook(rel string) (HookTool, bool) {
	root, ok := r.readYAML(rel)
	if !ok {
		return HookTool{}, false
	}
	tool := HookTool{Path: rel, Tool: "lefthook", Hooks: []string{}}
	for _, hook := range hookNames {
		if mappingValue(root, hook) != nil {
			tool.Hooks = append(tool.Hooks, hook)
		}
	}
	return tool, true
}

func (r *reader) readAgentHooks() *AgentHooks {
	const rel = ".claude/settings.json"
	text, ok := r.read(rel)
	if !ok {
		return nil
	}
	// Each event holds matcher groups, and each group holds the hook entries that run.
	var settings struct {
		Hooks map[string][]struct {
			Hooks []json.RawMessage `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(text), &settings); err != nil {
		r.fail(rel, err)
		return nil
	}
	if settings.Hooks == nil {
		return nil
	}
	h := &AgentHooks{Path: rel, Line: 1}
	// A key written with escapes does not match the plain text, so the line falls back to 1.
	if i := strings.Index(text, `"hooks"`); i >= 0 {
		h.Line += strings.Count(text[:i], "\n")
	}
	for _, groups := range settings.Hooks {
		for _, g := range groups {
			h.Count += len(g.Hooks)
		}
	}
	return h
}

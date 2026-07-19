package cli

// Explainable changed-scope validation (spec pose-changed-scope-validation):
// select the minimum safe set of modules from explicit Git revisions,
// declared dependency edges and policy. Selection is deterministic for the
// same base/head/config; every decision carries a reason and every
// unselected check is recorded as skipped with a machine-readable reason.
// Uncertainty always widens toward safe execution.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// safeGitRevRE rejects revisions that could be parsed as options or escape
// expected forms (confinement: never pass untrusted flags to git).
var safeGitRevRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/~^-]*$`)

type scopeModuleMeta struct {
	Criticality string   `json:"criticality"`
	DependsOn   []string `json:"dependsOn"`
}

type scopeMetadata struct {
	Modules map[string]scopeModuleMeta `json:"modules"`
}

func loadScopeMetadata(root string) scopeMetadata {
	var meta scopeMetadata
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json"))
	if err == nil {
		_ = json.Unmarshal(raw, &meta)
	}
	return meta
}

// scopeSelection maps selected module → deterministic selection reason.
type scopeSelection struct {
	Base, Head string
	Changed    []string
	Selected   map[string]string
}

func (s scopeSelection) rangeLabel() string {
	if s.Head == "" {
		return s.Base + "..worktree"
	}
	return s.Base + ".." + s.Head
}

// computeChangedScope resolves the selection. Modules are selected when they
// contain changed files, depend (transitively) on a changed module, or policy
// widens them (criticality high). A change outside every module selects
// everything — uncertainty prefers safe execution.
func computeChangedScope(root, base, head string, modules []validationModule) (scopeSelection, error) {
	sel := scopeSelection{Base: base, Head: head, Selected: map[string]string{}}
	for _, rev := range []string{base, head} {
		if rev != "" && !safeGitRevRE.MatchString(rev) {
			return sel, fmt.Errorf("unsafe git revision: %q", rev)
		}
	}
	gitArgs := []string{"-C", root, "diff", "--name-only", base}
	if head != "" {
		gitArgs = append(gitArgs, head)
	}
	gitArgs = append(gitArgs, "--")
	out, err := exec.Command("git", gitArgs...).Output()
	if err != nil {
		return sel, fmt.Errorf("git diff %s failed (revision unknown?)", sel.rangeLabel())
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			sel.Changed = append(sel.Changed, filepath.ToSlash(line))
		}
	}
	if head == "" {
		// Worktree comparisons must also see untracked files — invisible new
		// code must never silently narrow validation (safe execution wins).
		if untracked, err := exec.Command("git", "-C", root, "ls-files", "--others", "--exclude-standard").Output(); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(untracked)), "\n") {
				if line != "" {
					sel.Changed = append(sel.Changed, filepath.ToSlash(line))
				}
			}
		}
	}
	sort.Strings(sel.Changed)

	meta := loadScopeMetadata(root)
	moduleOf := func(file string) string {
		match := ""
		for _, m := range modules {
			if file == m.Rel || strings.HasPrefix(file, m.Rel+"/") {
				if len(m.Rel) > len(match) {
					match = m.Rel
				}
			}
		}
		return match
	}
	for _, file := range sel.Changed {
		if m := moduleOf(file); m != "" {
			if _, ok := sel.Selected[m]; !ok {
				sel.Selected[m] = "contains changed file: " + file
			}
		} else {
			// Root-level or unmapped change: safe execution wins.
			for _, m := range modules {
				if _, ok := sel.Selected[m.Rel]; !ok {
					sel.Selected[m.Rel] = "root-level change outside any module: " + file
				}
			}
			return sel, nil
		}
	}
	// Policy widening: high-criticality modules always run.
	for _, m := range modules {
		if meta.Modules[m.Rel].Criticality == "high" {
			if _, ok := sel.Selected[m.Rel]; !ok {
				sel.Selected[m.Rel] = "policy: criticality high always runs"
			}
		}
	}
	// Transitive dependents of selected modules (declared edges only).
	for changed := true; changed; {
		changed = false
		for _, m := range modules {
			if _, ok := sel.Selected[m.Rel]; ok {
				continue
			}
			for _, dep := range meta.Modules[m.Rel].DependsOn {
				if reason, ok := sel.Selected[filepath.ToSlash(filepath.Clean(dep))]; ok {
					sel.Selected[m.Rel] = "depends on selected module " + dep + " (" + firstScopeReason(reason) + ")"
					changed = true
					break
				}
			}
		}
	}
	return sel, nil
}

// firstScopeReason keeps chained dependency reasons short and readable.
func firstScopeReason(reason string) string {
	if i := strings.Index(reason, " ("); i > 0 {
		return reason[:i]
	}
	return reason
}

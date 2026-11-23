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
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
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
	// SelfContained declares that the module's checks read only its own
	// directory and its declared dependencies, so a change outside every
	// module does not select it (spec pose-validate-carries-unchanged-checks R6).
	SelfContained bool `json:"selfContained"`
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
// every module that does not declare itself self-contained — uncertainty
// prefers safe execution. Excluded paths (the run's own canonical result) are
// not changes.
func computeChangedScope(root, base, head string, modules []validationModule, exclude ...string) (scopeSelection, error) {
	sel := scopeSelection{Base: base, Head: head, Selected: map[string]string{}}
	for _, rev := range []string{base, head} {
		if rev != "" && !safeGitRevRE.MatchString(rev) {
			return sel, fmt.Errorf("unsafe git revision: %q", rev)
		}
	}
	// NUL-delimited and without rename pairing: a quoted or multi-line name
	// keeps its owner, and a rename names both its source and its destination
	// (review round 2, N1 and N3).
	gitArgs := []string{"-C", root, "diff", "--name-only", "-z", "--no-renames", base}
	if head != "" {
		gitArgs = append(gitArgs, head)
	}
	gitArgs = append(gitArgs, "--")
	out, err := posemodel.RawGit(gitArgs...).Output()
	if err != nil {
		return sel, fmt.Errorf("git diff %s failed (revision unknown?)", sel.rangeLabel())
	}
	for _, name := range strings.Split(string(out), "\x00") {
		if name != "" {
			sel.Changed = append(sel.Changed, filepath.ToSlash(name))
		}
	}
	if head == "" {
		// Worktree comparisons must also see untracked files — invisible new
		// code must never silently narrow validation (safe execution wins).
		if untracked, err := posemodel.RawHistoryGit(root, "ls-files", "-z", "--others", "--exclude-standard").Output(); err == nil {
			for _, name := range strings.Split(string(untracked), "\x00") {
				if name != "" {
					sel.Changed = append(sel.Changed, filepath.ToSlash(name))
				}
			}
		}
	}
	if len(exclude) > 0 {
		excluded := map[string]bool{}
		for _, path := range exclude {
			excluded[filepath.ToSlash(filepath.Clean(path))] = true
		}
		kept := sel.Changed[:0]
		for _, file := range sel.Changed {
			if !excluded[file] {
				kept = append(kept, file)
			}
		}
		sel.Changed = kept
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
			// Root-level or unmapped change: safe execution wins, except for
			// modules that declare their checks read nothing outside them. A
			// root module owns the files no other module owns, so its
			// declaration never exempts it from them (review round 2, N2).
			for _, m := range modules {
				if meta.Modules[m.Rel].SelfContained && m.Rel != "." {
					continue
				}
				if _, ok := sel.Selected[m.Rel]; !ok {
					sel.Selected[m.Rel] = "root-level change outside any module: " + file
				}
			}
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

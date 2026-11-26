package cli

// Explainable changed-scope validation (spec pose-changed-scope-validation):
// select the minimum safe set of modules from explicit Git revisions,
// declared dependency edges and policy. Selection is deterministic for the
// same base/head/config; every decision carries a reason and every
// unselected check is recorded as skipped with a machine-readable reason.
// Uncertainty always widens toward safe execution.

import (
	"encoding/json"
	"errors"
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

// loadScopeMetadata reads the module metadata. An absent file is empty
// metadata; a file that does not decode is an error, never a partial
// reading that could keep an exemption and drop a dependency (review round
// 3, N6).
func loadScopeMetadata(root string) (scopeMetadata, error) {
	var meta scopeMetadata
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json"))
	if errors.Is(err, os.ErrNotExist) {
		return meta, nil
	}
	if err != nil {
		return meta, fmt.Errorf("module metadata is unreadable: %v", err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return scopeMetadata{}, fmt.Errorf("module metadata does not decode: %v", err)
	}
	return meta, nil
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
	meta, err := loadScopeMetadata(root)
	if err != nil {
		return scopeSelection{Base: base, Head: head, Selected: map[string]string{}}, err
	}
	return computeChangedScopeWith(root, base, head, modules, meta, exclude...)
}

// computeChangedScopeWith selects with the given metadata snapshot, so a
// caller that verified one snapshot selects with exactly it (review round 17,
// N50).
func computeChangedScopeWith(root, base, head string, modules []validationModule, meta scopeMetadata, exclude ...string) (scopeSelection, error) {
	sel := scopeSelection{Base: base, Head: head, Selected: map[string]string{}}
	for _, rev := range []string{base, head} {
		if rev != "" && !safeGitRevRE.MatchString(rev) {
			return sel, fmt.Errorf("unsafe git revision: %q", rev)
		}
	}
	// NUL-delimited and without rename pairing: a quoted or multi-line name
	// keeps its owner, and a rename names both its source and its destination
	// (review round 2, N1 and N3).
	// Configuration never hides a changed gitlink (review round 8, N23).
	gitArgs := []string{"-C", root, "-c", "diff.ignoreSubmodules=none", "diff", "--ignore-submodules=none", "--name-only", "-z", "--no-renames", base}
	if head != "" {
		gitArgs = append(gitArgs, head)
	}
	gitArgs = append(gitArgs, "--")
	// Against the work tree (no head) the diff runs over a private index
	// (spec pose-validate-reads-leave-the-index-alone R1).
	out, err := posemodel.RawGitOutputWithPrivateIndex(root, gitArgs...)
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
	// Every changed file selects, independently of one another (review rounds
	// 2–4, N2, N5, N7, N8):
	//   - its most specific owning module;
	//   - every module under it (a gitlink or a replaced directory changes what
	//     lies beneath);
	//   - every module whose declared dependsOn contains it, lies under it, or
	//     is the project root ".", whether or not the dependency is discovered;
	//   - when no module owns it: the root module and every module that does
	//     not declare itself self-contained (safe execution wins).
	selectModule := func(rel, reason string) {
		if _, ok := sel.Selected[rel]; !ok {
			sel.Selected[rel] = reason
		}
	}
	for _, file := range sel.Changed {
		owner := moduleOf(file)
		if owner != "" {
			selectModule(owner, "contains changed file: "+file)
			// An enclosing module's checks may read inside the modules nested
			// in its directory, and the root module encloses them all (review
			// round 9, N25).
			for _, m := range modules {
				if m.Rel != owner && (m.Rel == "." || strings.HasPrefix(owner, m.Rel+"/")) {
					selectModule(m.Rel, "changed file in nested module "+owner+": "+file)
				}
			}
		}
		for _, m := range modules {
			if m.Rel != "." && strings.HasPrefix(m.Rel, file+"/") {
				selectModule(m.Rel, "changed path above the module: "+file)
			}
			// Every eligible dependency counts, whatever its order (review
			// round 5, N11).
			// A dependency that is the file's own owner reaches its dependents
			// through the transitive widening; any other one, discovered or not,
			// selects here (review round 6, N15).
			for _, dep := range declaredDependencyHits(file, meta.Modules[m.Rel].DependsOn) {
				if dep != owner {
					selectModule(m.Rel, "declared dependency "+dep+" changed: "+file)
					break
				}
			}
			if owner == "" && (m.Rel == "." || !meta.Modules[m.Rel].SelfContained) {
				selectModule(m.Rel, "root-level change outside any module: "+file)
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

// declaredDependencyHits returns every declared dependency file changes: one
// it lies in, one it lies above, or the project root ".", which every file
// changes.
func declaredDependencyHits(file string, deps []string) []string {
	var hits []string
	for _, dep := range deps {
		dep = filepath.ToSlash(filepath.Clean(dep))
		if dep == "." || file == dep || strings.HasPrefix(file, dep+"/") || strings.HasPrefix(dep, file+"/") {
			hits = append(hits, dep)
		}
	}
	return hits
}

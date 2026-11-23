package cli

// Carried checks (spec pose-validate-carries-unchanged-checks): with a
// comparable canonical result as base, validate --carry runs the modules the
// changed scope selects between the base head and HEAD and carries the base
// results of every other module, each stating the head it ran at. Any doubt
// about the base runs everything and records why (R2).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// carryRecord is the run's account of its base (R5).
type carryRecord struct {
	Used     bool              `json:"used"`
	Reason   string            `json:"reason,omitempty"`
	BaseHead string            `json:"base_head,omitempty"`
	Changed  int               `json:"changed"`
	Selected map[string]string `json:"selected,omitempty"`
}

// carryPlan is what a usable base lets the run reuse.
type carryPlan struct {
	record    *carryRecord
	selection *scopeSelection
	base      map[string]checkResult
	baseHead  string
	baseTime  string
}

// carried reports whether module's checks come from the base.
func (p carryPlan) carried(module string) bool {
	if p.selection == nil {
		return false
	}
	_, selected := p.selection.Selected[module]
	return !selected
}

// carriedResult is the base result of a check as this run records it (R4):
// unchanged, plus where and when it really ran, kept through chained carries.
func (p carryPlan) carriedResult(id string) checkResult {
	result := p.base[id]
	if result.CarriedFrom == "" {
		result.CarriedFrom, result.CarriedGeneratedAt = p.baseHead, p.baseTime
	}
	return result
}

// planCarry judges the canonical result at resultsPath as a base. A nil
// selection means every check runs; the record says why.
func planCarry(root, resultsPath, mode, matrixSHA, head string, matrix validationMatrix, modules []validationModule) carryPlan {
	plan := carryPlan{record: &carryRecord{}}
	engine, engineVerifiable := engineIdentityFrom(debug.ReadBuildInfo())
	unused := func(format string, args ...any) carryPlan {
		plan.record.Reason = fmt.Sprintf(format, args...)
		return plan
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resultsPath)))
	if err != nil {
		return unused("no canonical result at %s", resultsPath)
	}
	var base validationRunResult
	if err := json.Unmarshal(raw, &base); err != nil || base.SchemaVersion != validationResultSchema {
		return unused("the canonical result is unreadable or has another schema")
	}
	switch {
	case base.ModuleFilter != "" || base.StackFilter != "":
		return unused("the canonical result was filtered by module or stack, so it is not comparable")
	case base.Mode != mode:
		return unused("the canonical result ran in mode %q, this run is %q", base.Mode, mode)
	case base.MatrixSHA256 != matrixSHA:
		return unused("the validation matrix changed since the canonical result")
	case !engineVerifiable:
		return unused("this engine build was modified from its recorded revision, so its identity cannot be compared")
	case base.EngineVersion == "" || base.EngineVersion != engine || strings.HasSuffix(base.EngineVersion, ".dirty"):
		return unused("the engine version differs from the canonical result's or was not recorded")
	case base.GitHead == "" || !safeGitRevRE.MatchString(base.GitHead):
		return unused("the canonical result records no usable head")
	case head == "" || !safeGitRevRE.MatchString(head):
		return unused("the current head cannot be read")
	}
	for _, check := range base.Checks {
		if check.LimitState == "cancelled" {
			return unused("the canonical result was cancelled")
		}
	}
	// The base must have validated the commit it records: a run over a working
	// copy that differed from its head proves nothing about that head.
	if !base.MaterialClean {
		return unused("the canonical result did not validate a clean working copy of its head")
	}
	for _, binding := range base.MaterialProvenance {
		if binding == posemodel.MaterialBindingUnavailable {
			return unused("the canonical result records an unavailable material binding")
		}
	}
	if !materialWorktreeClean(root) {
		return unused("the working copy differs from HEAD in material paths")
	}
	// Selection is lexical: a module reached through a symlink aliases another
	// path, so its inputs cannot be told apart (review round 4, N9).
	// Two modules sharing one physical directory (a case alias on a
	// case-insensitive filesystem) alias each other too (review round 5, N13).
	if reason := aliasedModule(modules, filepath.EvalSymlinks, os.Stat); reason != "" {
		return unused("%s", reason)
	}
	// A declared dependency reached through a symlink aliases another path
	// the selection cannot see (review round 6, N16).
	meta, err := loadScopeMetadata(root)
	if err != nil {
		return unused("changed-scope selection failed: %v", err)
	}
	for rel, module := range meta.Modules {
		for _, dep := range module.DependsOn {
			abs := filepath.Join(root, filepath.FromSlash(filepath.ToSlash(filepath.Clean(dep))))
			if _, err := os.Lstat(abs); err != nil {
				continue
			}
			if resolved, err := filepath.EvalSymlinks(abs); err != nil || filepath.Clean(resolved) != filepath.Clean(abs) {
				return unused("dependency %s of module %s is reached through a symlink, so its changes cannot be attributed", dep, rel)
			}
		}
	}
	if err := posemodel.RawGit("-C", root, "merge-base", "--is-ancestor", base.GitHead, head).Run(); err != nil {
		return unused("the canonical result's head %s is not an ancestor of HEAD", base.GitHead)
	}
	selection, err := computeChangedScope(root, base.GitHead, head, modules, carryExclusions(root, base.GitHead, head, resultsPath)...)
	if err != nil {
		return unused("changed-scope selection failed: %v", err)
	}
	plan.base = map[string]checkResult{}
	for _, check := range base.Checks {
		plan.base[check.ID] = check
	}
	// A module whose checks are not all in the base runs (R3).
	for _, module := range modules {
		if _, selected := selection.Selected[module.Rel]; selected {
			continue
		}
		stack, checks, _ := moduleValidationChecks(module, matrix, "", "")
		for _, check := range checks {
			if check.Program == "" {
				continue
			}
			id := module.Rel + "/" + stack + "/" + check.Name
			if prior, ok := plan.base[id]; !ok || prior.Outcome == "skipped" || prior.Outcome == "" {
				selection.Selected[module.Rel] = "carry: no prior result for " + id
				break
			}
		}
	}
	plan.selection, plan.baseHead, plan.baseTime = &selection, base.GitHead, base.GeneratedAt
	plan.record.Used, plan.record.BaseHead, plan.record.Changed, plan.record.Selected = true, base.GitHead, len(selection.Changed), selection.Selected
	return plan
}

// carryExclusions are the changed paths that are not changes for the carry
// selection (R3): the canonical result itself and the derived administrative
// records of the engine's closed list (the same classes material bindings
// ignore) that are still regular files, or gone, at HEAD. A symlink,
// submodule or directory in a class stays a change.
func carryExclusions(root, base, head, resultsPath string) []string {
	exclude := []string{resultsPath}
	out, err := posemodel.RawGit("-C", root, "diff", "--name-only", "-z", "--no-renames", base, head, "--").Output()
	if err != nil {
		return exclude
	}
	for _, file := range strings.Split(string(out), "\x00") {
		file = filepath.ToSlash(file)
		if file == "" || !posemodel.IsAdministrativePath(file) {
			continue
		}
		entry, err := posemodel.RawGit("-C", root, "ls-tree", "-z", head, "--", file).Output()
		if err != nil {
			continue
		}
		fields := strings.Fields(string(entry))
		if len(fields) == 0 || fields[0] == "100644" || fields[0] == "100755" {
			exclude = append(exclude, file)
		}
	}
	return exclude
}

// validationEngineIdentity names the engine that ran a validation: its version
// plus, when the binary records it, the VCS revision it was built from, marked
// when the build tree was modified. Development builds share a version string,
// so the revision is what tells two of them apart (R2).
func validationEngineIdentity() string {
	identity, _ := engineIdentityFrom(debug.ReadBuildInfo())
	return identity
}

// engineIdentityFrom builds the identity from build information. A build of a
// modified tree is not verifiable: two such builds at one revision share an
// identity while their code differs (review round 2, N4).
func engineIdentityFrom(info *debug.BuildInfo, ok bool) (string, bool) {
	identity := version.Version
	if !ok || info == nil {
		return identity, true
	}
	revision, modified := "", false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision != "" {
		identity += "+" + revision
		if modified {
			identity += ".dirty"
		}
	}
	return identity, !modified
}

// aliasedModule names a module whose changes cannot be attributed lexically:
// one reached through a symlink, or one sharing its physical directory with
// another module, as a case alias does on a case-insensitive filesystem
// (review rounds 4–5, N9 and N13).
func aliasedModule(modules []validationModule, resolve func(string) (string, error), stat func(string) (os.FileInfo, error)) string {
	seen := make([]os.FileInfo, 0, len(modules))
	for _, module := range modules {
		resolved, err := resolve(module.Abs)
		if err != nil || filepath.Clean(resolved) != filepath.Clean(module.Abs) {
			return fmt.Sprintf("module %s is reached through a symlink, so its changes cannot be attributed", module.Rel)
		}
		info, err := stat(module.Abs)
		if err != nil {
			return fmt.Sprintf("module %s cannot be read", module.Rel)
		}
		for _, other := range seen {
			if os.SameFile(info, other) {
				return fmt.Sprintf("module %s shares its directory with another module, so its changes cannot be attributed", module.Rel)
			}
		}
		seen = append(seen, info)
	}
	return ""
}

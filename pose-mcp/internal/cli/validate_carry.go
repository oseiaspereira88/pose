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
func planCarry(root, resultsPath, mode, matrixSHA string, matrix validationMatrix, modules []validationModule) carryPlan {
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
	if err := posemodel.RawGit("-C", root, "merge-base", "--is-ancestor", base.GitHead, "HEAD").Run(); err != nil {
		return unused("the canonical result's head %s is not an ancestor of HEAD", base.GitHead)
	}
	selection, err := computeChangedScope(root, base.GitHead, "HEAD", modules, carryExclusions(root, base.GitHead, resultsPath)...)
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
func carryExclusions(root, base, resultsPath string) []string {
	exclude := []string{resultsPath}
	out, err := posemodel.RawGit("-C", root, "diff", "--name-only", "-z", "--no-renames", base, "HEAD", "--").Output()
	if err != nil {
		return exclude
	}
	for _, file := range strings.Split(string(out), "\x00") {
		file = filepath.ToSlash(file)
		if file == "" || !posemodel.IsAdministrativePath(file) {
			continue
		}
		entry, err := posemodel.RawGit("-C", root, "ls-tree", "-z", "HEAD", "--", file).Output()
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

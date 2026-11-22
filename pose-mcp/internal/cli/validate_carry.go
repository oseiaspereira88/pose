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
	case base.EngineVersion == "" || base.EngineVersion != version.Version:
		return unused("the engine version differs from the canonical result's or was not recorded")
	case base.GitHead == "" || !safeGitRevRE.MatchString(base.GitHead):
		return unused("the canonical result records no usable head")
	}
	for _, check := range base.Checks {
		if check.LimitState == "cancelled" {
			return unused("the canonical result was cancelled")
		}
	}
	if !materialWorktreeClean(root) {
		return unused("the working copy differs from HEAD in material paths")
	}
	if err := posemodel.RawGit("-C", root, "merge-base", "--is-ancestor", base.GitHead, "HEAD").Run(); err != nil {
		return unused("the canonical result's head %s is not an ancestor of HEAD", base.GitHead)
	}
	selection, err := computeChangedScope(root, base.GitHead, "HEAD", modules, resultsPath)
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

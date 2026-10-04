package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harne8/pose-mcp/internal/version"
)

// AssessmentBinding is what a component assessment was computed from (spec
// pose-adaptive-assessment-freshness). An assessment is reused only while
// the component's committed content, the engine and the validation matrix
// are the same; age alone never decides.
type AssessmentBinding struct {
	ContentTree   string `json:"content_tree"`
	EngineVersion string `json:"engine_version"`
	MatrixDigest  string `json:"matrix_digest"`
}

// AssessmentFreshness is the reuse decision for one component.
type AssessmentFreshness struct {
	Component string            `json:"component"`
	Path      string            `json:"path"`
	State     string            `json:"state"`
	Reasons   []string          `json:"reasons,omitempty"`
	Current   AssessmentBinding `json:"current"`
}

// Assessment freshness states.
const (
	AssessmentFresh   = "fresh"
	AssessmentStale   = "stale"
	AssessmentMissing = "missing"
)

// CurrentAssessmentBinding computes the binding of a component path now. A
// path with uncommitted changes has no committed content to bind to, so it
// is never fresh.
func (s Store) CurrentAssessmentBinding(relPath string) (AssessmentBinding, bool) {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	binding := AssessmentBinding{EngineVersion: version.Version, MatrixDigest: "absent"}
	if raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "validation-matrix.json")); err == nil {
		binding.MatrixDigest = digestBytes(raw)
	}
	spec := "HEAD:" + clean
	if clean == "." {
		spec = "HEAD^{tree}"
	}
	tree, err := exec.Command("git", "-C", s.Root, "rev-parse", spec).Output()
	if err != nil {
		return binding, false
	}
	binding.ContentTree = strings.TrimSpace(string(tree))
	status, err := exec.Command("git", "-C", s.Root, "status", "--porcelain", "--untracked-files=normal", "--", clean).Output()
	if err != nil || len(strings.TrimSpace(string(status))) > 0 {
		return binding, false
	}
	return binding, true
}

// ComponentAssessmentFreshness decides reuse for one component path.
func (s Store) ComponentAssessmentFreshness(relPath string) AssessmentFreshness {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	out := AssessmentFreshness{Component: slugifyAssessmentPath(clean), Path: clean}
	current, committed := s.CurrentAssessmentBinding(clean)
	out.Current = current
	stored, err := s.LoadComponentState(out.Component)
	switch {
	case err != nil || stored == nil:
		out.State, out.Reasons = AssessmentMissing, []string{"no assessment recorded for this component"}
		return out
	case stored.Binding == nil:
		out.State, out.Reasons = AssessmentStale, []string{"the recorded assessment predates binding; its inputs are unknown"}
		return out
	}
	if !committed {
		out.Reasons = append(out.Reasons, "the component has uncommitted changes")
	}
	if stored.Binding.ContentTree != current.ContentTree {
		out.Reasons = append(out.Reasons, "component content changed ("+short(stored.Binding.ContentTree)+" → "+short(current.ContentTree)+")")
	}
	if stored.Binding.EngineVersion != current.EngineVersion {
		out.Reasons = append(out.Reasons, "engine changed ("+stored.Binding.EngineVersion+" → "+current.EngineVersion+")")
	}
	if stored.Binding.MatrixDigest != current.MatrixDigest {
		out.Reasons = append(out.Reasons, "validation matrix changed")
	}
	out.State = AssessmentFresh
	if len(out.Reasons) > 0 {
		out.State = AssessmentStale
	}
	return out
}

func short(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

// assessmentObligations projects missing or stale assessments of the
// components that in-progress specs declare. They are advisory: an
// assessment informs work, it does not gate it.
func (s Store) assessmentObligations(project string, specs []Spec) []Obligation {
	seen := map[string]bool{}
	var out []Obligation
	for _, sp := range specs {
		if sp.Status != "in-progress" {
			continue
		}
		for _, component := range sp.Components {
			component = strings.TrimSpace(component)
			if component == "" || seen[component] {
				continue
			}
			seen[component] = true
			if info, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(component))); err != nil || !info.IsDir() {
				continue
			}
			fresh := s.ComponentAssessmentFreshness(component)
			if fresh.State == AssessmentFresh {
				continue
			}
			o := newObligation(project, "assessments", specNode(project, sp.Slug), "assessment-freshness", component)
			o.Category = ObligationEvidence
			o.ReasonCode = "assessment-" + fresh.State
			o.Condition = "`pose assess discover --if-stale --component " + component + "` records an assessment bound to the current content, engine and matrix"
			o.Effects = []ObligationEffect{{Phase: PhaseExecution, Mode: EffectAdvisory}}
			o.Waiting = WaitingExecution
			o.Message = component + " assessment is " + fresh.State + ": " + strings.Join(fresh.Reasons, "; ")
			out = append(out, o)
		}
	}
	return out
}

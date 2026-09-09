// `roadmap-check` reaches its gate
// (spec pose-roadmap-check-reaches-its-gate).
//
// The graph builder returned before it loaded a single roadmap when the
// delivery profile index was absent, so the command reported zero cut criteria
// and exited 0 on a repository whose roadmap declared several. A gate that
// cannot evaluate its criteria was reading as a gate that passed them.
//
// It was found while writing a coverage fixture: the first version of that
// fixture had no profile index, and both --strict and --tolerant returned 0 on
// a criterion naming a delivery target that does not exist.

package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// roadmapGateFixture is a repository with a roadmap that declares one criterion
// and nothing else — no profile index, no delivery targets. That is the shape
// the early return used to answer for.
func roadmapGateFixture(t *testing.T, criterion string) string {
	t.Helper()
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "roadmaps", "alpha.md"),
		"---\nslug: alpha\nstatus: in-progress\n---\n\n## Cut criteria\n- "+criterion+"\n")
	return root
}

func TestRoadmapCheckEvaluatesCriteriaWithoutDeliveryProfiles(t *testing.T) {
	root := roadmapGateFixture(t, "C1: surface:dashboard check:web-reachability evidence:e2e")

	out, errOut, code := runCLI(t, root, "roadmap-check", "alpha", "--json")
	if !strings.Contains(out, "C1") {
		t.Fatalf("the criterion never reached the gate: out=%q err=%q", out, errOut)
	}
	var result struct {
		Blockers []string `json:"blockers"`
		Criteria []struct {
			ID      string   `json:"id"`
			Passed  bool     `json:"passed"`
			Reasons []string `json:"reasons"`
		} `json:"criteria"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(result.Criteria) != 1 {
		t.Fatalf("criteria = %+v, want the one the roadmap declares", result.Criteria)
	}
	if result.Criteria[0].Passed {
		t.Error("a criterion naming a delivery target that does not exist was reported passed")
	}
	if len(result.Blockers) == 0 {
		t.Error("an unmet criterion produced no blocker")
	}
	if code == 0 {
		t.Errorf("strict passed a roadmap with an unmet criterion: %s%s", out, errOut)
	}
}

// A criterion that needs no delivery profile at all is the case the early
// return was least defensible for: `manual-review:` resolves against a path,
// and nothing about it depends on the profile index.
func TestRoadmapCheckEvaluatesAManualReviewCriterion(t *testing.T) {
	root := roadmapGateFixture(t, "C1: manual-review:../outside.md")

	out, errOut, code := runCLI(t, root, "roadmap-check", "alpha", "--json")
	if !strings.Contains(out, "C1") {
		t.Fatalf("the criterion never reached the gate: out=%q err=%q", out, errOut)
	}
	if code == 0 {
		t.Errorf("a manual-review ref pointing outside the project passed: %s", out)
	}
}

// And the gate still passes what it should: a criterion whose manual-review ref
// is a confined path has nothing against it, so a repository with no delivery
// profiles is not newly blocked by this change.
func TestRoadmapCheckStillPassesACriterionItCanSatisfy(t *testing.T) {
	root := roadmapGateFixture(t, "C1: manual-review:docs/review.md")
	mustWrite(t, filepath.Join(root, "docs", "review.md"), "# Review\n\nSigned off.\n")

	out, errOut, code := runCLI(t, root, "roadmap-check", "alpha", "--json")
	if code != 0 {
		t.Errorf("a satisfiable criterion was blocked (%d): %s%s", code, out, errOut)
	}
}

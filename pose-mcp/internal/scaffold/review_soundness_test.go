package scaffold_test

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/scaffold"
)

// Verify the shipped instructions and profile using the runtime classifier,
// so an installed agent cannot be instructed to auto-approve judged criteria.
func TestDistributedReviewSoundnessContract(t *testing.T) {
	dist := scaffold.Dist()
	for _, prefix := range []string{"", "locales/pt-BR/"} {
		read := func(path string) string {
			t.Helper()
			raw, err := fs.ReadFile(dist, prefix+path)
			if err != nil {
				t.Fatal(err)
			}
			return string(raw)
		}
		manual := read("POSE.md")
		if !strings.Contains(manual, "judgment") || !strings.Contains(manual, "pending") && !strings.Contains(manual, "pendência") || !strings.Contains(manual, "wont-fix") {
			t.Fatalf("%smanual omits the explicit judgment or risk contract", prefix)
		}
		skill := read(".agents/skills/pose-review/SKILL.md")
		if strings.Contains(skill, "auto-attest <bundle-id> --reviewer agent:reviewer-subagent --apply") {
			t.Fatalf("%sskill still instructs automatic approval", prefix)
		}
		if !strings.Contains(skill, "review_attestation.pending") || !strings.Contains(skill, "wont-fix") {
			t.Fatalf("%sskill omits pending judgments or accepted-risk obligations", prefix)
		}
	}
	raw, err := fs.ReadFile(dist, ".pose/review-profiles/spec-closeout.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile struct {
		Criteria []pose.ReviewPlanCriterion `json:"criteria"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatal(err)
	}
	judged := map[string]bool{}
	for _, criterion := range profile.Criteria {
		if pose.ReviewCriterionKind(criterion) == pose.ReviewCriterionKindJudgment {
			judged[criterion.ID] = true
		}
	}
	for _, id := range []string{"scope", "compatibility", "security", "documentation", "operability"} {
		if !judged[id] {
			t.Fatalf("installed profile no longer requires explicit %s judgment", id)
		}
	}
}

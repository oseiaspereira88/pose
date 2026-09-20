package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestABMProgressiveReviewInstallAndCLIPlan(t *testing.T) {
	root := newGitRepo(t)
	var out, errOut bytes.Buffer
	if code := Main([]string{"install", root, "--skip-mcp"}, &out, &errOut); code != 0 {
		t.Fatalf("install code=%d err=%s", code, errOut.String())
	}
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"engineering-judgment", "high-criticality-review"} {
		if _, err := os.Stat(filepath.Join(root, ".pose/review-profiles", name+".json")); err != nil {
			t.Fatalf("profile not installed: %v", err)
		}
		if bytes.Contains(raw, []byte(name+"@1")) {
			t.Fatalf("install activated %s", name)
		}
	}
	// Adopt explicitly only in the isolated installed fixture.
	policy["overlay_profiles"] = []string{"engineering-judgment@1", "high-criticality-review@1"}
	policy["component_aware"] = true
	policy["schema_version"] = 2
	policy["component_aware_adopted_at"] = "2026-09-19"
	raw, err = json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", string(raw))
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"packages":[{"name":"api","path":"api","language":"go","criticality":"critical","metadataStatus":{"source":"declared"}}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/progressive.md", "---\nslug: progressive\nstatus: in-progress\ncreated_at: 2026-09-19\ncomponents: api\ndelivers: governance:authority\n---\n# Spec\n\n### Delivery targets\n- governance:authority module:api profile:backend-go entrypoint:api/AGENTS.md\n")
	out.Reset()
	errOut.Reset()
	if code := cmdReviewPlan(root, []string{"spec:progressive", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("review-plan code=%d err=%s", code, errOut.String())
	}
	var plan posemodel.ReviewPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	expected, err := (posemodel.Store{Root: root}).ReviewPlan("spec:progressive")
	if err != nil {
		t.Fatal(err)
	}
	// Compare the wire contract: omitempty intentionally removes empty slices.
	wantJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, err := json.Marshal(plan)
	if err != nil || !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("CLI diverged from Store JSON: err=%v", err)
	}
	if plan.Independence != "different-actor" {
		t.Fatalf("critical governance scope did not escalate: %+v", plan)
	}
	count := 0
	for _, criterion := range plan.Criteria {
		if criterion.ID == "solution-proportionality" {
			count++
			if criterion.Kind != "judgment" || len(criterion.Profiles) != 2 {
				t.Fatalf("lost explicit judgment or origins: %+v", criterion)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one proportionality judgment, got %d", count)
	}
}

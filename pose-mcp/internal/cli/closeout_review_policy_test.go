package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// CLI tests of spec pose-closeout-review-policy-by-cost-benefit; the permanent
// integration check runs every TestCloseoutReviewPolicy test.

func adoptCloseoutPolicyFixture(t *testing.T, root, milestone, roadmap string) {
	t.Helper()
	path := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["schema_version"] = posemodel.CloseoutReviewPolicySchemaVersion
	policy[posemodel.MilestoneReviewRequiredKey], policy[posemodel.RoadmapReviewRequiredKey] = milestone, roadmap
	profiles := policy["profiles"].(map[string]any)
	profiles["roadmap"] = "roadmap-outcome@1"
	out, _ := json.Marshal(policy)
	mustWrite(t, path, string(out))
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/roadmap-outcome.json", `{"schema_version":2,"id":"roadmap-outcome","version":1,"scope":"roadmap","criteria":[{"id":"outcome","description":"reviewed","evidence_classes":["unit"]}]}`)
}

// closeoutPolicyCLIFixture is the one-milestone program whose only member,
// bundle, has an approved review, under the adopted pair.
func closeoutPolicyCLIFixture(t *testing.T, milestone, roadmap string) string {
	t.Helper()
	root := reachedMilestoneFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: first\n- specs: bundle\n")
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"deliveryProfiles":{"api-contract":{"kind":"contract","requiredEvidenceClasses":["integration","unit"]}}}`)
	adoptCloseoutPolicyFixture(t, root, milestone, roadmap)
	commitCloseoutFixture(t, root, "adopt closeout review policy")
	if code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 0 {
		t.Fatalf("close spec %d %s", code, out)
	}
	commitCloseoutFixture(t, root, "closeout")
	return root
}

func commitCloseoutFixture(t *testing.T, root, message string) {
	t.Helper()
	artifactGit(t, root, "add", "-A")
	artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", message)
	publishReviewCLIIndexFixture(t, root)
}

func TestCloseoutReviewPolicyReceipt(t *testing.T) {
	root := closeoutPolicyCLIFixture(t, "never", "never")
	receiptPath := filepath.Join(root, ".pose/review-closeouts/milestone/program/first.json")
	if _, err := os.Stat(receiptPath); !os.IsNotExist(err) {
		t.Fatalf("the member's close closed the milestone implicitly: %v", err)
	}
	var state posemodel.CloseoutState
	if code, out := runPoseJSON(t, root, &state, "closeout-check", "milestone:program/first", "--json"); state.Terminal || !state.CloseRequired {
		t.Fatalf("eligible milestone read as terminal: %d %s", code, out)
	}
	code, out := runPose(t, root, "close", "milestone:program/first")
	if code != 0 || !strings.Contains(out, "close.receipt=review: not-required; milestone_review_required=never; members=1, never required") {
		t.Fatalf("close milestone %d %s", code, out)
	}
	if !strings.Contains(out, "close.next=pose close roadmap:program") {
		t.Fatalf("the close did not name the roadmap: %s", out)
	}
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"review": "not-required"`, `"policy_key": "milestone_review_required"`, `"policy_value": "never"`, `"evaluator_version": "closeout-review@1"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("receipt lacks %s: %s", want, raw)
		}
	}
	if code, again := runPose(t, root, "close", "milestone:program/first"); code != 0 || !strings.Contains(again, "close.already-satisfied") {
		t.Fatalf("a repeated close was not a verification: %d %s", code, again)
	}
	if after, _ := os.ReadFile(receiptPath); string(after) != string(raw) {
		t.Fatal("a repeated close rewrote the receipt")
	}
	commitCloseoutFixture(t, root, "milestone closed")
	code, out = runPose(t, root, "close", "roadmap:program")
	if code != 0 || !strings.Contains(out, "close.receipt=review: not-required; roadmap_review_required=never") {
		t.Fatalf("close roadmap %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/review-closeouts/roadmap/program.json")); err != nil {
		t.Fatalf("no roadmap receipt: %v", err)
	}
	if rm, _ := os.ReadFile(filepath.Join(root, ".pose/roadmaps/program.md")); !strings.Contains(string(rm), "status: done") {
		t.Fatalf("roadmap status not published: %s", rm)
	}
}

func TestCloseoutReviewPolicyAlwaysRefusesWithoutReview(t *testing.T) {
	root := closeoutPolicyCLIFixture(t, "always", "always")
	code, out := runPose(t, root, "close", "milestone:program/first")
	if code == 0 || !strings.Contains(out, "not eligible") {
		t.Fatalf("always closed a one-member milestone without its own review: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/review-closeouts/milestone/program/first.json")); !os.IsNotExist(err) {
		t.Fatal("a refused close published a receipt")
	}
	var plan posemodel.ReviewPlan
	if code, out := runPoseJSON(t, root, &plan, "review-plan", "milestone:program/first", "--json"); code != 0 || plan.CloseoutReview == nil || !plan.CloseoutReview.Required {
		t.Fatalf("plan does not carry the decision: %d %s", code, out)
	}
}

func TestCloseoutReviewPolicyPublication(t *testing.T) {
	root := closeoutPolicyCLIFixture(t, "never", "never")
	path := filepath.Join(root, ".pose/review-closeouts/milestone/program/first.json")
	// A different receipt already there is never overwritten to close again.
	writeCloseoutCLIFile(t, root, ".pose/review-closeouts/milestone/program/first.json", `{"schema_version":1,"scope":"milestone:program/first"}`)
	code, out := runPose(t, root, "close", "milestone:program/first")
	if code == 0 {
		t.Fatalf("a corrupt receipt was closed over: %s", out)
	}
	if raw, _ := os.ReadFile(path); !strings.Contains(string(raw), `"scope":"milestone:program/first"}`) {
		t.Fatalf("the corrupt receipt was rewritten: %s", raw)
	}
	os.Remove(path)
	// A roadmap left done without its receipt is incomplete; close verifies and publishes.
	if code, out := runPose(t, root, "close", "milestone:program/first"); code != 0 {
		t.Fatalf("close milestone %d %s", code, out)
	}
	commitCloseoutFixture(t, root, "milestone closed")
	rm := filepath.Join(root, ".pose/roadmaps/program.md")
	raw, _ := os.ReadFile(rm)
	mustWrite(t, rm, strings.Replace(string(raw), "status: active", "status: done", 1))
	commitCloseoutFixture(t, root, "status only")
	if code, out := runPose(t, root, "close", "roadmap:program"); code != 0 || !strings.Contains(out, "close.receipt=") {
		t.Fatalf("the incomplete roadmap close was not completed: %d %s", code, out)
	}
	if code, out := runPose(t, root, "close", "roadmap:program"); code != 0 || !strings.Contains(out, "close.already-satisfied") {
		t.Fatalf("%d %s", code, out)
	}
}

func TestCloseoutReviewPolicyAdoption(t *testing.T) {
	root := reachedMilestoneFixture(t)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	before, _ := os.ReadFile(policyPath)
	code, out := runPose(t, root, "adopt", "closeout-review-policy")
	if code != 0 || !strings.Contains(out, "adopt.current=milestone_review_required=legacy") || !strings.Contains(out, "adopt.change=set milestone_review_required=never") || !strings.Contains(out, "adopt.apply=false") {
		t.Fatalf("preview %d %s", code, out)
	}
	if after, _ := os.ReadFile(policyPath); string(after) != string(before) {
		t.Fatal("a preview wrote the policy")
	}
	for _, args := range [][]string{
		{"adopt", "closeout-review-policy", "--milestone-review-required", "always"},
		{"adopt", "closeout-review-policy", "--off", "--milestone-review-required", "always", "--roadmap-review-required", "never"},
		{"adopt", "criterion-reuse", "--milestone-review-required", "always", "--roadmap-review-required", "never"},
		{"adopt", "closeout-review-policy", "--milestone-review-required", "sometimes", "--roadmap-review-required", "never", "--apply"},
	} {
		if code, out := runPose(t, root, args...); code == 0 {
			t.Fatalf("%v was accepted: %s", args, out)
		}
	}
	if after, _ := os.ReadFile(policyPath); string(after) != string(before) {
		t.Fatal("a refused adoption wrote the policy")
	}
	if code, out := runPose(t, root, "adopt", "closeout-review-policy", "--apply"); code != 0 {
		t.Fatalf("apply %d %s", code, out)
	}
	policy, err := posemodel.Store{Root: root}.GetReviewPolicy()
	if err != nil || policy.SchemaVersion != 5 || policy.MilestoneReviewRequired != "never" || policy.RoadmapReviewRequired != "never" {
		t.Fatalf("%+v %v", policy, err)
	}
	if code, out := runPose(t, root, "adopt", "closeout-review-policy", "--milestone-review-required", "when-justified", "--roadmap-review-required", "always", "--apply"); code != 0 {
		t.Fatalf("change %d %s", code, out)
	}
	policy, _ = posemodel.Store{Root: root}.GetReviewPolicy()
	if policy.MilestoneReviewRequired != "when-justified" || policy.RoadmapReviewRequired != "always" {
		t.Fatalf("%+v", policy)
	}
	if code, out := runPose(t, root, "adopt", "closeout-review-policy", "--off", "--apply"); code == 0 || !strings.Contains(out, "cannot be turned off") {
		t.Fatalf("the pair was removed: %d %s", code, out)
	}
	// Adopting references afterwards keeps schema 5 and the pair.
	if code, out := runPose(t, root, "adopt", "qualified-artifact-refs", "--apply"); code != 0 {
		t.Fatalf("refs %d %s", code, out)
	}
	if code, out := runPose(t, root, "adopt", "qualified-artifact-refs", "--off", "--apply"); code != 0 {
		t.Fatalf("refs off %d %s", code, out)
	}
	policy, err = posemodel.Store{Root: root}.GetReviewPolicy()
	if err != nil || policy.SchemaVersion != 5 || policy.RoadmapReviewRequired != "always" {
		t.Fatalf("another capability lowered schema 5: %+v %v", policy, err)
	}
	var plan setupPlan
	if code, out := runPoseJSON(t, root, &plan, "setup", "--json"); code != 0 || !plan.CloseoutReview.Adopted || plan.CloseoutReview.Milestone != "when-justified" {
		t.Fatalf("setup %d %s", code, out)
	}
}

func TestCloseoutReviewPolicySetupShowsTheLegacyRule(t *testing.T) {
	root := reachedMilestoneFixture(t)
	var plan setupPlan
	if code, out := runPoseJSON(t, root, &plan, "setup", "--json"); code != 0 || plan.CloseoutReview.Adopted || !strings.HasPrefix(plan.CloseoutReview.Milestone, "legacy") || plan.CloseoutReview.ProposedMilestone != "never" {
		t.Fatalf("setup %d %s", code, out)
	}
	code, out := runPose(t, root, "setup")
	if code != 0 || !strings.Contains(out, "setup.closeout_review=milestone_review_required=legacy") || !strings.Contains(out, "adopting proposes milestone_review_required=never") {
		t.Fatalf("setup text %d %s", code, out)
	}
}

func TestCloseoutReviewPolicyDoctor(t *testing.T) {
	root := closeoutPolicyCLIFixture(t, "always", "never")
	code, out := runPose(t, root, "doctor")
	_ = code
	if !strings.Contains(out, "milestone:program/first owes its own review") || !strings.Contains(out, "previous rule unknown") {
		t.Fatalf("doctor did not report the owed review: %s", out)
	}
	adoptCloseoutPolicyFixture(t, root, "never", "never")
	if _, out := runPose(t, root, "doctor"); strings.Contains(out, "owes its own review") {
		t.Fatalf("doctor reported a review never owes: %s", out)
	}
}

// TestCloseoutReviewPolicySmoke drives the built binary from the production
// entrypoint: adopt the pair, close the member, then the milestone and the
// roadmap, each publishing its receipt.
func TestCloseoutReviewPolicySmoke(t *testing.T) {
	root := reachedMilestoneFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: first\n- specs: bundle\n")
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"deliveryProfiles":{"api-contract":{"kind":"contract","requiredEvidenceClasses":["integration","unit"]}}}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/roadmap-outcome.json", `{"schema_version":2,"id":"roadmap-outcome","version":1,"scope":"roadmap","criteria":[{"id":"outcome","description":"reviewed","evidence_classes":["unit"]}]}`)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, _ := os.ReadFile(policyPath)
	mustWrite(t, policyPath, strings.Replace(string(raw), `"milestone":"milestone-integration@1"`, `"milestone":"milestone-integration@1","roadmap":"roadmap-outcome@1"`, 1))
	commitCloseoutFixture(t, root, "one milestone")
	if code, out := runBuilt(t, root, "adopt", "closeout-review-policy", "--milestone-review-required", "when-justified", "--roadmap-review-required", "never", "--apply"); code != 0 {
		t.Fatalf("adopt %d %s", code, out)
	}
	commitCloseoutFixture(t, root, "adopt")
	if code, out := runBuilt(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 0 || !strings.Contains(out, "close.next=pose close milestone:program/first") {
		t.Fatalf("close spec %d %s", code, out)
	}
	commitCloseoutFixture(t, root, "closeout")
	if code, out := runBuilt(t, root, "close", "milestone:program/first"); code != 0 || !strings.Contains(out, "close.receipt=review: not-required; milestone_review_required=when-justified; members=1, threshold>2") {
		t.Fatalf("close milestone %d %s", code, out)
	}
	commitCloseoutFixture(t, root, "milestone")
	if code, out := runBuilt(t, root, "close", "roadmap:program"); code != 0 || !strings.Contains(out, "close.receipt=review: not-required; roadmap_review_required=never") {
		t.Fatalf("close roadmap %d %s", code, out)
	}
	var state posemodel.CloseoutState
	code, out := runBuilt(t, root, "closeout-check", "roadmap:program", "--json")
	if err := json.Unmarshal([]byte(out), &state); err != nil || state.CloseoutReceipt == nil || state.CloseoutReceipt.Review != "not-required" {
		t.Fatalf("closeout-check %d %s %v", code, out, err)
	}
}

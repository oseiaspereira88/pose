package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeReviewFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func reviewFixture(t *testing.T) (string, Store) {
	t.Helper()
	root := t.TempDir()
	policy := `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1","milestone":"milestone-integration@1","roadmap":"roadmap-outcome@1"},"accepted_risk_severities":["low"],"continuous_closeout":true}`
	writeReviewFixture(t, root, ".pose/policy/review.json", policy)
	for _, item := range []struct{ name, scope, criterion string }{
		{"spec-closeout", "spec", "requirements"},
		{"milestone-integration", "milestone", "integration"},
		{"roadmap-outcome", "roadmap", "outcome"},
	} {
		profile := fmt.Sprintf(`{"schema_version":1,"id":%q,"version":1,"scope":%q,"criteria":[{"id":%q,"description":"reviewed"}]}`, item.name, item.scope, item.criterion)
		writeReviewFixture(t, root, ".pose/review-profiles/"+item.name+".json", profile)
	}
	writeReviewFixture(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-02\ncompleted_at: 2026-08-02\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n")
	writeReviewFixture(t, root, ".pose/roadmaps/delivery.md", "---\nslug: delivery\nstatus: done\n---\n\n# Roadmap\n\n**Outcome:** shipped\n\n## Milestone: core\n- after:\n- specs: alpha\n\n**Exit gate:** integrated\n\n## Cut criteria\n- C1: verified\n")
	return root, Store{Root: root}
}

func recordApprovedReview(t *testing.T, store Store, id, scope, profile, criterion, supersedes string) {
	t.Helper()
	digest, err := store.ScopeDigest(scope)
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("---\nschema_version: 1\nreview_id: %s\nscope: %s\nscope_digest: %s\nprofile: %s\nreviewer: agent:test-review\ndecision: approved\nreviewed_at: 2026-08-02T12:00:00Z\nsupersedes: %s\nevidence_refs: [check:test]\n---\n\n## Criteria\n- %s [passed] evidence:check:test\n\n## Findings\n", id, scope, digest, profile, supersedes, criterion)
	writeReviewFixture(t, store.Root, ".pose/reviews/"+id+".md", body)
}

func TestHierarchicalCloseoutRequiresFreshReviewsAtEveryLevel(t *testing.T) {
	_, store := reviewFixture(t)
	state, err := store.GetCloseoutState("roadmap:delivery")
	if err != nil {
		t.Fatal(err)
	}
	if state.Terminal || !strings.Contains(state.NextAction, "spec:alpha") {
		t.Fatalf("unexpected initial state: %+v", state)
	}
	recordApprovedReview(t, store, "rvw-spec", "spec:alpha", "spec-closeout@1", "requirements", "")
	recordApprovedReview(t, store, "rvw-milestone", "milestone:delivery/core", "milestone-integration@1", "integration", "")
	recordApprovedReview(t, store, "rvw-roadmap", "roadmap:delivery", "roadmap-outcome@1", "outcome", "")
	state, err = store.GetCloseoutState("roadmap:delivery")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Terminal || state.NextAction != "none" {
		t.Fatalf("expected terminal closeout: %+v", state)
	}
}

func TestReviewBecomesStaleAfterScopeChangeButNotLifecycleEdit(t *testing.T) {
	root, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-spec", "spec:alpha", "spec-closeout@1", "requirements", "")
	path := filepath.Join(root, ".pose/specs/alpha/spec.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "status: done", "status: in-progress", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil || !eval.Fresh {
		t.Fatalf("lifecycle-only edit invalidated review: eval=%+v err=%v", eval, err)
	}
	raw, _ = os.ReadFile(path)
	if err := os.WriteFile(path, append(raw, []byte("\nmaterial scope change\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	eval, err = store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Fresh || eval.Approved || !strings.Contains(strings.Join(eval.Blockers, " "), "stale") {
		t.Fatalf("scope change did not invalidate review: %+v", eval)
	}
}

func TestReviewRejectsTraversalOpenFindingsAndIncompleteCriteria(t *testing.T) {
	if _, err := ParseScopeRef("spec:../../etc"); err == nil {
		t.Fatal("expected traversal scope rejection")
	}
	_, store := reviewFixture(t)
	digest, _ := store.ScopeDigest("spec:alpha")
	body := fmt.Sprintf("---\nreview_id: rvw-bad\nscope: spec:alpha\nscope_digest: %s\nprofile: spec-closeout@1\nreviewer: agent:test\ndecision: approved\nreviewed_at: 2026-08-02T12:00:00Z\n---\n\n## Criteria\n- requirements [finding]\n\n## Findings\n- F1 [open] severity:high action:fix evidence:test:x\n", digest)
	writeReviewFixture(t, store.Root, ".pose/reviews/rvw-bad.md", body)
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Approved || !strings.Contains(strings.Join(eval.Blockers, " "), "finding F1 is open") {
		t.Fatalf("open finding did not block: %+v", eval)
	}
}

func TestReviewPolicyExemptsLegacyDoneScopesUnlessOptedIn(t *testing.T) {
	root, store := reviewFixture(t)
	writeReviewFixture(t, root, ".pose/specs/legacy/spec.md", "---\nslug: legacy\nstatus: done\ncreated_at: 2026-08-01\ncompleted_at: 2026-08-01\n---\n\n# Spec: legacy\n\n## 2. Requirements\n- R1: shipped before review adoption\n")
	writeReviewFixture(t, root, ".pose/roadmaps/legacy-delivery.md", "---\nslug: legacy-delivery\nstatus: done\ncreated_at: 2026-08-01\n---\n\n# Roadmap\n\n**Outcome:** shipped before review adoption\n\n## Milestone: core\n- after:\n- specs: legacy\n\n**Exit gate:** integrated\n\n## Cut criteria\n- C1: verified\n")

	for _, scope := range []string{"spec:legacy", "milestone:legacy-delivery/core", "roadmap:legacy-delivery"} {
		eval, err := store.ReviewCheck(scope)
		if err != nil {
			t.Fatalf("ReviewCheck(%s): %v", scope, err)
		}
		if eval.Required || !eval.PolicyEnabled || len(eval.Blockers) != 0 || !strings.Contains(strings.Join(eval.Warnings, " "), "legacy done scope") {
			t.Fatalf("legacy scope %s was not exempt: %+v", scope, eval)
		}
	}

	state, err := store.GetCloseoutState("roadmap:legacy-delivery")
	if err != nil || !state.Terminal {
		t.Fatalf("legacy roadmap closeout remained blocked: state=%+v err=%v", state, err)
	}

	current, err := store.ReviewCheck("spec:alpha")
	if err != nil || !current.Required {
		t.Fatalf("scope created at adoption must require review: eval=%+v err=%v", current, err)
	}

	writeReviewFixture(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1","milestone":"milestone-integration@1","roadmap":"roadmap-outcome@1"},"require_review_for_legacy_done_scopes":true}`)
	eval, err := store.ReviewCheck("spec:legacy")
	if err != nil || !eval.Required || !strings.Contains(strings.Join(eval.Blockers, " "), "no review attempt") {
		t.Fatalf("explicit legacy enforcement was ignored: eval=%+v err=%v", eval, err)
	}
}

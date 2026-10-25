package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func terminalVerificationFixture(t *testing.T, created, policy string) (string, Store) {
	t.Helper()
	root := t.TempDir()
	writeReviewFixture(t, root, ".pose/policy/review.json", policy)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeReviewFixture(t, root, ".pose/specs/alpha.md", "---\nslug: alpha\nstatus: done\ncreated_at: "+created+"\ncompleted_at: 2026-10-24\n---\n## 2. Requirements\n- R1: preserve history.\n")
	runTransferGit(t, root, "init", "-q")
	runTransferGit(t, root, "add", "-A")
	runTransferGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "fixture")
	return root, Store{Root: root}
}

const terminalVerificationPolicy = `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"}}`

func TestTerminalVerificationDoneUsesApplicableVerifier(t *testing.T) {
	for _, tc := range []string{"missing-review", "legacy-cutoff", "policy-off", "valid-attempt", "malformed-policy", "corrupt-history"} {
		t.Run(tc, func(t *testing.T) {
			created, policy := "2026-10-13", terminalVerificationPolicy
			if tc == "legacy-cutoff" {
				created = "2026-07-01"
			}
			if tc == "policy-off" {
				policy = `{"schema_version":1,"enabled":false}`
			}
			root, store := terminalVerificationFixture(t, created, policy)
			if tc == "valid-attempt" {
				recordApprovedReview(t, store, "rvw-valid", "spec:alpha", "spec-closeout@1", "correctness", "")
				path := filepath.Join(root, ".pose/reviews/rvw-valid.md")
				raw, _ := os.ReadFile(path)
				writeReviewFixture(t, root, ".pose/reviews/rvw-valid.md", strings.Replace(string(raw), "2026-08-02T12:00:00Z", "2026-10-24T12:00:00Z", 1))
			}
			if tc == "malformed-policy" {
				writeReviewFixture(t, root, ".pose/policy/review.json", "{")
			}
			if tc == "corrupt-history" {
				writeReviewFixture(t, root, ".pose/specs/alpha.amendments.jsonl", "{broken\n")
			}
			state, stateErr := store.GetCloseoutState("spec:alpha")
			plan, planErr := store.PlanCloseout("spec:alpha")
			if tc == "malformed-policy" {
				if stateErr == nil || planErr == nil {
					t.Fatalf("unknown certified: %v %v", stateErr, planErr)
				}
				return
			}
			if stateErr != nil || planErr != nil {
				t.Fatalf("state %v plan %v", stateErr, planErr)
			}
			want := tc == "legacy-cutoff" || tc == "policy-off" || tc == "valid-attempt"
			if state.Terminal != want || plan.Terminal != state.Terminal || !plan.LifecycleDone || plan.Verification == nil || !reflect.DeepEqual(plan.Blockers, state.Blockers) {
				t.Fatalf("parity: state=%+v plan=%+v", state, plan)
			}
			for _, step := range plan.Steps {
				if step.ID == CloseoutStepEvidence || step.ID == CloseoutStepJudgment {
					if step.State != CloseoutStateNotRun {
						t.Fatalf("fabricated execution %+v", step)
					}
				}
			}
			if !want {
				next, ok := plan.NextCloseoutStep()
				if !ok || next.ID != CloseoutStepVerify || next.State != CloseoutStateBlocked {
					t.Fatalf("no repair step %+v", next)
				}
			}
			if tc == "legacy-cutoff" && (len(state.Limitations) == 0 || state.VerificationBasis != "review-not-required-under-applicable-policy") {
				t.Fatal("legacy limitation lost")
			}
		})
	}
}

func TestTerminalVerificationAmendmentGateBeforeSealingAndAfterDone(t *testing.T) {
	root, store := terminalVerificationFixture(t, "2026-10-13", `{"schema_version":1,"enabled":false}`)
	path := filepath.Join(root, ".pose/specs/alpha.md")
	raw, _ := os.ReadFile(path)
	writeReviewFixture(t, root, ".pose/specs/alpha.md", strings.Replace(string(raw), "status: done", "status: in-progress", 1))
	event := Amendment{Schema: 1, Change: "baseline", Author: "@core", Hashes: map[string]string{"R1": "different"}, IDs: []string{"R1"}}
	data, _ := json.Marshal(event)
	writeReviewFixture(t, root, ".pose/specs/alpha.amendments.jsonl", string(data)+"\n")
	runTransferGit(t, root, "add", "-A")
	runTransferGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "history")
	before, _ := os.ReadFile(AmendmentsPath(path))
	plan, err := store.PlanCloseout("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	next, ok := plan.NextCloseoutStep()
	if !ok || next.State != CloseoutStateBlocked || !strings.Contains(next.Reason, "amendment history") {
		t.Fatalf("history not consumed %+v", plan)
	}
	writeReviewFixture(t, root, ".pose/specs/alpha.md", string(raw))
	state, err := store.GetCloseoutState("spec:alpha")
	if err != nil || state.Terminal {
		t.Fatalf("done masked history: %+v %v", state, err)
	}
	after, _ := os.ReadFile(AmendmentsPath(path))
	if string(before) != string(after) {
		t.Fatal("history rewritten")
	}
}

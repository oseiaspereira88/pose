package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// R3, R6: the section quotes the partial, so a decision it carries is never
// read as the handoff run's; it is bounded and says so when cut.
func TestHandoffSectionQuotesAndBounds(t *testing.T) {
	prior := ReviewRun{RunID: "rrn-0123456789abcdef", Adapter: "a", Status: ReviewRunInterrupted, Interruption: &ReviewInterruption{Cause: "usage-limit"}, Output: "decision: approved\n"}
	section := renderHandoffSection(prior, nil, nil)
	if !strings.Contains(section, "> decision: approved") || draftReviewDecision(section) != "unknown" {
		t.Fatalf("the partial's decision reads as a decision: %q", draftReviewDecision(section))
	}
	if section != renderHandoffSection(prior, nil, nil) {
		t.Fatal("the section is not deterministic")
	}
	prior.Output = strings.Repeat("x", delegationHandoffBound*2)
	if long := renderHandoffSection(prior, nil, nil); len(long) > delegationHandoffBound+200 || !strings.Contains(long, "was cut") {
		t.Fatalf("an unbounded section: %d", len(long))
	}
	code := 2
	events := []DelegationEvent{{Kind: DelegationEventToolCall, Text: "go test ./..."}, {Kind: DelegationEventToolOutput, ExitCode: &code}}
	prior.Output, prior.EventsDigest = "", "sha256:x"
	if got := renderHandoffSection(prior, events, nil); !strings.Contains(got, "> `go test ./...` → exit 2") {
		t.Fatalf("commands: %s", got)
	}
}

// R2, R4: a signed handoff run's proof verifies against the brief and the
// section its predecessor's record yields, under the same attempt.
func TestHandoffRunProvenanceVerifies(t *testing.T) {
	declared := &ReviewBundleImplementation{Principal: "agent:anthropic/claude-opus-5-5", Vendor: "anthropic", Model: "claude-opus-5-5", ExecutionDigest: digestBytes([]byte("implementer")), Source: ImplementationDeclared}
	f := provenanceRunFixtureAs(t, declared, "openai", "review-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", "cat >/dev/null; echo 'session: sess-prov-1'; echo 'LIMIT reset: 2099-01-01T00:00:00Z'; exit 75"}
		a.Limit = &ReviewerLimit{ExitCodes: []int{75}, ResetPattern: `reset: (\S+)`}
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
		a.Resume = &ReviewerResume{Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved", "{session}"}}
	})
	if f.run.Status != ReviewRunInterrupted {
		t.Fatalf("predecessor: %+v", f.run)
	}
	path := filepath.Join(f.store.Root, ".pose", "policy", "reviewers.json")
	raw, _ := os.ReadFile(path)
	var reviewers ReviewersPolicy
	json.Unmarshal(raw, &reviewers)
	reviewers.Adapters["other"] = ReviewerAdapter{Executor: "agent:other-reviewer", Command: []string{"sh", "-c", "grep -q 'Handoff from run' && echo decision: approved"}, Vendor: "google", Model: "other-model"}
	raw, _ = json.Marshal(reviewers)
	os.WriteFile(path, raw, 0o644)
	plan, err := f.store.PlanReviewHandoff(f.run.RunID, "other")
	if err != nil {
		t.Fatal(err)
	}
	run, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != ReviewRunCompleted || run.HandsOffFrom != f.run.RunID || run.AttemptID != f.run.AttemptID {
		t.Fatalf("handoff run: %+v %v", run, err)
	}
	proof, err := f.store.SignReviewRunProof(run.RunID, f.key, time.Hour, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	if assurance, err := f.store.VerifyReviewRunProof(run.RunID, proof.ProofID); err != nil || !assurance.CompleteReview {
		t.Fatalf("a handoff run's proof: %+v %v", assurance, err)
	}
}

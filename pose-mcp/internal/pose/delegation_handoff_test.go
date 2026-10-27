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

// R3: a command is given the exit status of its own output, paired by the
// call identifier; without one, an output is paired only when one call waits,
// and any other is listed with its command unknown, never guessed.
func TestHandoffCommandsPairOnlyWhatIsKnown(t *testing.T) {
	two, zero := 2, 0
	interleaved := []DelegationEvent{
		{Kind: DelegationEventToolCall, Text: "test-A", Call: "a"}, {Kind: DelegationEventToolCall, Text: "test-B", Call: "b"},
		{Kind: DelegationEventToolOutput, ExitCode: &two, Call: "a"}, {Kind: DelegationEventToolOutput, ExitCode: &zero, Call: "b"},
	}
	got := strings.Join(handoffCommands(interleaved), "\n")
	if got != "> `test-A` → exit 2\n> `test-B` → exit 0" {
		t.Fatalf("interleaved by identifier: %s", got)
	}
	anonymous := []DelegationEvent{
		{Kind: DelegationEventToolCall, Text: "test-A"}, {Kind: DelegationEventToolCall, Text: "test-B"},
		{Kind: DelegationEventToolOutput, ExitCode: &two}, {Kind: DelegationEventToolOutput, ExitCode: &zero},
		{Kind: DelegationEventToolCall, Text: "test-C"}, {Kind: DelegationEventToolOutput, ExitCode: &zero},
	}
	got = strings.Join(handoffCommands(anonymous), "\n")
	for _, want := range []string{"> `test-A` → exit unknown", "> `test-B` → exit unknown", "> `(an output whose command is not known)` → exit 2", "> `test-C` → exit 0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "test-B` → exit 2") || strings.Contains(got, "test-A` → exit 0") {
		t.Fatalf("a status was guessed:\n%s", got)
	}
}

// R2, D1: the attempt's budget is the opening agent's, whatever agent
// continues it; its default applies when it declares none.
func TestAttemptContinuationBudgetIsTheOpeningAgents(t *testing.T) {
	at := func(start string, d time.Duration) (string, string) {
		s, _ := time.Parse(time.RFC3339Nano, start)
		return s.Format(time.RFC3339Nano), s.Add(d).Format(time.RFC3339Nano)
	}
	s1, e1 := at("2026-10-26T10:00:00Z", 2*time.Minute)
	s2, e2 := at("2026-10-26T11:00:00Z", 3*time.Minute)
	runs := []ReviewRun{
		{RunID: "r1", AttemptID: "rat-1", Adapter: "small", StartedAt: s1, EndedAt: e1},
		{RunID: "r2", AttemptID: "rat-1", Adapter: "large", HandsOffFrom: "r1", StartedAt: s2, EndedAt: e2},
	}
	policy := ReviewersPolicy{Adapters: map[string]ReviewerAdapter{
		"small": {Resume: &ReviewerResume{MaxResumes: 1, MaxWallTime: "4m"}},
		"large": {Resume: &ReviewerResume{MaxResumes: 10, MaxWallTime: "10h"}},
	}}
	count, spent, limit, total, err := attemptContinuationBudget(runs, runs[1], policy)
	if err != nil || count != 1 || spent != 5*time.Minute || limit != 1 || total != 4*time.Minute {
		t.Fatalf("budget: %d %s %d %s %v", count, spent, limit, total, err)
	}
	policy.Adapters["small"] = ReviewerAdapter{}
	if _, _, limit, total, _ := attemptContinuationBudget(runs, runs[1], policy); limit != 3 || total != time.Hour {
		t.Fatalf("defaults: %d %s", limit, total)
	}
	delete(policy.Adapters, "small")
	if _, _, _, _, err := attemptContinuationBudget(runs, runs[1], policy); err == nil {
		t.Fatal("an attempt whose opening agent is gone was budgeted")
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

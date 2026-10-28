package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-delegated-review-action-fulfillment over the CLI (R1, R2, R9,
// R11): pose review request opens the request, a run concluded by the
// engine is submitted with pose action fulfil — by flags, and as the
// envelope a CI job writes — and the request is fulfilled once.
func TestReviewRequestIsFulfilledOverTheCLI(t *testing.T) {
	root, draftPath, bundleID := cliProvenanceInstance(t)
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var opened posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &opened) != nil || opened.Request.Kind != posemodel.ActionDelegatedReview || opened.State != posemodel.ActionStateOpen {
		t.Fatalf("request: %d %s", code, out)
	}
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, bundleID, draftPath)
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--apply"); code != 0 {
		t.Fatal(code, out)
	}
	runs, _, _ := s.ReadReviewRuns("")
	run := runs[0].RunID
	if code, out := runPose(t, root, "action", "fulfil", opened.Request.ID, "--run", run, "--runner", "agent:ci-runner", "--idempotency-key", "k", "--expected-revision", "1", "--apply"); code == 0 || !strings.Contains(out, "no conclusion") {
		t.Fatalf("an unconcluded run fulfilled: %d %s", code, out)
	}
	if code, out := runPose(t, root, "review", "conclude", run, "--issuer", "fixture-executor", "--apply"); code != 0 {
		t.Fatal(code, out)
	}
	envelope := posemodel.ActionFulfillmentEnvelope{RequestID: opened.Request.ID, RequestDigest: opened.Request.RequestDigest, ExpectedRevision: 1,
		IdempotencyKey: "ci-" + run, RunID: run, Runner: "agent:ci-runner", Channel: "ci"}
	raw, _ := json.Marshal(envelope)
	path := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "action", "fulfil", "--envelope", path); code != 0 || !strings.Contains(out, "action.fulfil.decision=approved") {
		t.Fatalf("preview: %d %s", code, out)
	}
	code, out = runPose(t, root, "action", "fulfil", "--envelope", path, "--apply", "--json")
	var done posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &done) != nil || done.State != posemodel.ActionStateFulfilled || done.Events[len(done.Events)-1].Channel != "ci" {
		t.Fatalf("fulfil: %d %s", code, out)
	}
	if code, out := runPose(t, root, "action", "fulfil", "--envelope", path, "--apply"); code != 0 {
		t.Fatalf("a retry was refused: %d %s", code, out)
	}
}

// R7 over the CLI: a scope whose plan reserves its review for a person is
// asked of a person, not of an agent.
func TestReviewRequestAsksAPersonWhenThePlanRequiresOne(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t)
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(code, out)
	}
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply")
	if code != 0 || !strings.Contains(out, "review_request.person_required=true") || !strings.Contains(out, "review_request.escalation=act-") {
		t.Fatalf("a person-required review was asked of an agent: %d %s", code, out)
	}
}

// R3 and R7 over pose delegate: a run asked for by a request runs only as
// the request asks — its notes, an agent it accepts — and is submitted once
// the engine records its conclusion; a spent budget asks a person.
func TestDelegateRunsWhatTheRequestAsks(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(approvingAgent))
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--agents", "ok", "--apply", "--json")
	var opened posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &opened) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	notes := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(notes, []byte("other notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--request", opened.Request.ID, "--note-file", notes); code == 0 || !strings.Contains(out+errOut, "other notes") {
		t.Fatalf("a run with other notes ran: %d %s %s", code, out, errOut)
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--request", opened.Request.ID, "--agent", "someone"); code == 0 || !strings.Contains(out+errOut, "does not accept agent someone") {
		t.Fatalf("an unaccepted agent ran: %d %s %s", code, out, errOut)
	}
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--request", opened.Request.ID)
	if code != 0 || !strings.Contains(out+errOut, "is not fulfilled: the engine recorded no conclusion") {
		t.Fatalf("the request's run: %d %s %s", code, out, errOut)
	}
	store := posemodel.Store{Root: root}
	view, _ := store.LoadActionRequest(opened.Request.ID)
	var stdout, stderr strings.Builder
	delegateEscalateRefusal(store, &view, &posemodel.DelegationRefusal{Code: posemodel.DelegationReasonBudgetExhausted, Err: os.ErrPermission}, false, &stdout, &stderr)
	if !strings.Contains(stdout.String()+stderr.String(), "asked of a person: act-") {
		t.Fatalf("a spent budget asked nobody: %s %s", stdout.String(), stderr.String())
	}
}

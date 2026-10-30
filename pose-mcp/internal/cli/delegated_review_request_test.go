package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code != 0 {
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
	// Asked again a second later, as JSON: the same escalation (review round
	// 1, F04, F09).
	ask := func() string {
		t.Helper()
		code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
		var doc struct {
			PersonRequired bool   `json:"person_required"`
			Escalation     string `json:"escalation"`
		}
		if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || !doc.PersonRequired || doc.Escalation == "" {
			t.Fatalf("as JSON: %d %s", code, out)
		}
		return doc.Escalation
	}
	first := ask()
	time.Sleep(1100 * time.Millisecond)
	if again := ask(); again != first {
		t.Fatalf("asked twice: %s then %s", first, again)
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

// R3, R11 (review round 1, F05): the CI journey the documentation shows,
// run as written — in its own checkout the job runs the requested review
// with an accepted agent, the engine concludes it and fulfils the request,
// and the job commits .pose — and the repository that receives the commit
// reads the request as fulfilled and the scope as approved.
func TestDelegatedReviewCIJourneyAsDocumented(t *testing.T) {
	root, draftPath, bundleID := cliProvenanceInstance(t, func(root, draftPath string) {
		if err := os.Remove(filepath.Join(root, ".pose", "policy", "reviewers.json")); err != nil {
			t.Fatal(err)
		}
		// Every checkout records the same project only when it is declared.
		writeCloseoutCLIFile(t, root, ".pose/project.json", `{"schema_version":1,"project_id":"proj.fixture","name":"fixture"}`)
		writeCloseoutCLIFile(t, root, ".pose/policy/delegation.json", `{"schema_version": 2,
  "provenance": {"trust": "executor-attested", "implementation": {"principal": "agent:implementer", "execution": "implementation-fixture", "vendor": "openai", "model": "implementation-model"}},
  "conclusions": {"issuer": "fixture-executor"},
  "agents": {"ci-agent": {"vendor": "openai", "model": "review-model", "executor": "agent:reviewer", "timeout": "30s",
    "commands": {"read": ["sh", "-c", "cat >/dev/null; cat '`+draftPath+`'"]}}},
  "purposes": {"review": {"max_runs_per_bundle": 5}}}`)
	})
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--agents", "ci-agent", "--apply", "--json")
	var opened posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &opened) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	writeRunConclusion(t, posemodel.Store{Root: root}, bundleID, draftPath)
	ci := filepath.Join(t.TempDir(), "ci-checkout")
	if out, err := exec.Command("cp", "-a", root, ci).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	// The job's steps.
	if code, out, errOut := runPoseSplit(t, ci, "delegate", "review", "spec:alpha", "--request", opened.Request.ID, "--agent", "ci-agent"); code != 0 || !strings.Contains(out+errOut, "request "+opened.Request.ID+" fulfilled") {
		t.Fatalf("delegate --request in CI: %d\nOUT:%s\nERR:%s", code, out, errOut)
	}
	if code, out := runPose(t, ci, "action", "show", opened.Request.ID); code != 0 || !strings.Contains(out, "action.state=fulfilled") {
		t.Fatalf("action show in CI: %d %s", code, out)
	}
	// The commit lands in the repository.
	if out, err := exec.Command("cp", "-a", filepath.Join(ci, ".pose")+"/.", filepath.Join(root, ".pose")).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if code, out := runPose(t, root, "action", "show", opened.Request.ID); code != 0 || !strings.Contains(out, "action.state=fulfilled") {
		t.Fatalf("action show in the repository: %d %s", code, out)
	}
	if code, out := runPose(t, root, "review", "verify", "spec:alpha"); code != 0 {
		t.Fatalf("review verify in the repository: %d %s", code, out)
	}
}

// Review round 3, R3-F01 over the CLI: review request --supersedes opens a
// successor linked to its predecessor and retires it in the same step.
func TestReviewRequestSupersedes(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t)
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var first posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &first) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	code, out = runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--question", "Review spec:alpha again", "--supersedes", first.Request.ID, "--apply", "--json")
	var second posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &second) != nil || second.Request.Supersedes != first.Request.ID {
		t.Fatalf("successor: %d %s", code, out)
	}
	if old, _ := (posemodel.Store{Root: root}).LoadActionRequest(first.Request.ID); old.State != posemodel.ActionStateSuperseded {
		t.Fatalf("the predecessor was not retired: %s", old.State)
	}
}

// Review round 4, R4-F01: the replacement carries the authority that
// retires its predecessor — here a maintainer's role for a predecessor
// another principal asked — and its preview shows the preflight.
func TestReviewRequestSupersedesWithARole(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t)
	writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var first posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &first) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	replace := []string{"review", "request", "spec:alpha", "--kind", "delegated-review", "--question", "Review spec:alpha again", "--requested-by", "human:ada", "--supersedes", first.Request.ID}
	if code, out := runPose(t, root, append(replace, "--apply")...); code == 0 {
		t.Fatalf("another principal retired it without a role: %s", out)
	}
	code, out = runPose(t, root, append(replace, "--role", "maintainer", "--json")...)
	if code != 0 || !strings.Contains(out, `"preflight"`) {
		t.Fatalf("preview: %d %s", code, out)
	}
	if code, out := runPose(t, root, append(replace, "--role", "maintainer", "--apply")...); code != 0 {
		t.Fatalf("a maintainer could not retire it: %d %s", code, out)
	}
	if old, _ := (posemodel.Store{Root: root}).LoadActionRequest(first.Request.ID); old.State != posemodel.ActionStateSuperseded {
		t.Fatalf("the predecessor was not retired: %s", old.State)
	}
}

// Review round 4, R4-F02: when the plan now reserves the review for a
// person, the replacement is the person's escalation, which retires the
// predecessor; nothing is left restricting beside it.
func TestReviewRequestReplacementByAPerson(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t)
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var first posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &first) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(code, out)
	}
	code, out = runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--supersedes", first.Request.ID, "--apply", "--json")
	var doc struct {
		Escalation string `json:"escalation"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Escalation == "" {
		t.Fatalf("replacement: %d %s", code, out)
	}
	s := posemodel.Store{Root: root}
	if old, _ := s.LoadActionRequest(first.Request.ID); old.State != posemodel.ActionStateSuperseded {
		t.Fatalf("the predecessor was not retired: %s", old.State)
	}
	if e, _ := s.LoadActionRequest(doc.Escalation); e.Request.Supersedes != first.Request.ID || e.Request.Kind != posemodel.ActionDecision {
		t.Fatalf("the escalation is not the successor: %+v", e.Request)
	}
}

// Review round 4: a maintainer replaces, by role, a predecessor another
// principal asked, when the plan now reserves the review for a person.
func TestReviewRequestReplacementByAPersonWithARole(t *testing.T) {
	root, _, _ := cliProvenanceInstance(t)
	writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var first posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &first) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--requested-by", "human:ada", "--role", "maintainer", "--supersedes", first.Request.ID, "--apply"); code != 0 {
		t.Fatalf("replacement by role: %d %s", code, out)
	}
	if old, _ := (posemodel.Store{Root: root}).LoadActionRequest(first.Request.ID); old.State != posemodel.ActionStateSuperseded {
		t.Fatalf("the predecessor was not retired: %s", old.State)
	}
}

// Review round 5, R5-F01 and R5-F02: a replacement by role, previewed and
// applied with the preview's instant, replays as a no-op — to a delegated
// successor and to a person's escalation — and the person's preview shows
// the successor and its preflight.
func TestReviewRequestReplacementReplays(t *testing.T) {
	for _, person := range []bool{false, true} {
		t.Run(map[bool]string{false: "delegated", true: "person"}[person], func(t *testing.T) {
			root, _, _ := cliProvenanceInstance(t)
			writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
			code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
			var first posemodel.ActionRequestView
			if code != 0 || json.Unmarshal([]byte(out), &first) != nil {
				t.Fatalf("request: %d %s", code, out)
			}
			if person {
				path := filepath.Join(root, ".pose", "policy", "review.json")
				raw, _ := os.ReadFile(path)
				if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1)), 0o644); err != nil {
					t.Fatal(err)
				}
				if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
					t.Fatal(code, out)
				}
			}
			replace := []string{"review", "request", "spec:alpha", "--kind", "delegated-review", "--question", "Review spec:alpha again", "--requested-by", "human:ada", "--role", "maintainer", "--supersedes", first.Request.ID}
			code, out = runPose(t, root, append(replace, "--json")...)
			var preview struct {
				Request   posemodel.ActionRequest       `json:"request"`
				Preflight posemodel.ActionOpenPreflight `json:"preflight"`
			}
			if code != 0 || json.Unmarshal([]byte(out), &preview) != nil || preview.Request.ID == "" && preview.Preflight.Request.ID == "" {
				t.Fatalf("preview: %d %s", code, out)
			}
			at := preview.Preflight.Request.RequestedAt
			if at == "" {
				at = preview.Request.RequestedAt
			}
			apply := append(replace, "--requested-at", at, "--apply", "--json")
			code, first2 := runPose(t, root, apply...)
			if code != 0 {
				t.Fatalf("apply: %d %s", code, first2)
			}
			if code, again := runPose(t, root, apply...); code != 0 || again != first2 {
				t.Fatalf("a replay differed: %d %s\n%s", code, again, first2)
			}
		})
	}
}

// Review round 6, R6-F02: a fulfilled request whose bundle aged into a
// person's review, when the person is already asked, is retired by the
// recovery the refusal names: the person answers, the predecessor is
// waived by its authority, and nothing restricts closeout any more.
func TestFulfilledPredecessorWithAPersonAlreadyAsked(t *testing.T) {
	root, draftPath, bundleID := cliProvenanceInstance(t)
	writeCloseoutCLIFile(t, root, ".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:ada"]}}`)
	code, out := runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var p posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &p) != nil {
		t.Fatalf("request: %d %s", code, out)
	}
	s := posemodel.Store{Root: root}
	writeRunConclusion(t, s, bundleID, draftPath)
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code != 0 {
		t.Fatal(code, out)
	}
	runs, _, _ := s.ReadReviewRuns("")
	if code, out := runPose(t, root, "review", "conclude", runs[0].RunID, "--issuer", "fixture-executor", "--apply"); code != 0 {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "action", "fulfil", p.Request.ID, "--run", runs[0].RunID, "--runner", "agent:ci-runner", "--idempotency-key", "k", "--expected-revision", "1", "--apply"); code != 0 {
		t.Fatal(code, out)
	}
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"spec":"different-actor"`, `"spec":"mandatory-human"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(code, out)
	}
	code, out = runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--apply", "--json")
	var doc struct {
		Escalation string `json:"escalation"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Escalation == "" {
		t.Fatalf("escalation: %d %s", code, out)
	}
	code, out = runPose(t, root, "review", "request", "spec:alpha", "--kind", "delegated-review", "--supersedes", p.Request.ID, "--apply")
	if code == 0 || !strings.Contains(out, doc.Escalation) || !strings.Contains(out, "pose action waive "+p.Request.ID) {
		t.Fatalf("the collision did not name its recovery: %d %s", code, out)
	}
	a, _ := s.LoadActionRequest(doc.Escalation)
	if code, out := runPose(t, root, "action", "resolve", a.Request.ID, "--actor", "human:ada", "--answer", "person-reviews", "--request-digest", a.Request.RequestDigest,
		"--expected-revision", "1", "--idempotency-key", "a1", "--apply"); code != 0 {
		t.Fatalf("answer: %d %s", code, out)
	}
	old, _ := s.LoadActionRequest(p.Request.ID)
	if code, out := runPose(t, root, "action", "waive", p.Request.ID, "--actor", "human:ada", "--role", "maintainer", "--request-digest", p.Request.RequestDigest,
		"--expected-revision", strconv.Itoa(old.Revision), "--idempotency-key", "w1", "--reason", "a person reviews the new bundle", "--apply"); code != 0 {
		t.Fatalf("the named recovery was refused: %d %s", code, out)
	}
	if left, err := s.GovernedEffectRestrictions(posemodel.PhaseCloseout, []string{"alpha"}); err != nil || len(left) != 0 {
		t.Fatalf("restrictions remain: %+v %v", left, err)
	}
}

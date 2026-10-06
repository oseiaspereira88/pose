package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-action-requests.

func actionFixture(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/2026-10-04-storage.md", "---\nslug: storage\nstatus: in-progress\n---\n\n# Spec: storage\n\n## 2. Requirements\n\n- R4: Keep reading schema v1 records.\n")
	write(".pose/policy/actions.json", `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]}}`)
	return Store{Root: root}
}

func decisionRequest() ActionRequest {
	return ActionRequest{
		Origin:      "spec:storage",
		RequestedBy: ActionPrincipal{Principal: "agent:impl", Execution: "run-17"},
		Recipient:   ObligationActor{Role: "maintainer"},
		Kind:        ActionDecision,
		Question:    "Will the next delivery keep reading schema v1?",
		Options:     []ActionOption{{ID: "preserve-v1", Consequence: "Keep the v1 reader."}, {ID: "break-v1", Consequence: "Consumers must migrate."}},
		Recommend:   "preserve-v1",
		Targets:     []NodeRef{{Artifact: "self", Kind: "requirement", ID: "R4"}},
		Effects:     []ObligationEffect{{Phase: PhaseExecution, Mode: EffectBlock, Scope: []NodeRef{{Artifact: "self", Kind: "requirement", ID: "R4"}}}, {Phase: PhaseCloseout, Mode: EffectBlock}},
	}
}

func TestActionRequestSurvivesTheSessionWithIDAndDigest(t *testing.T) {
	s := actionFixture(t)
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	opened, err := s.OpenActionRequest(decisionRequest(), now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(opened.Request.ID, "act-") || opened.State != ActionStateOpen || opened.Satisfaction != SatisfactionPending || opened.Revision != 1 {
		t.Fatalf("opened: %+v", opened)
	}
	// A new process: nothing shared but the files.
	reloaded, err := Store{Root: s.Root}.LoadActionRequest(opened.Request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Request.RequestDigest != opened.Request.RequestDigest || reloaded.Request.Targets[0].Artifact != "xref:"+opened.Request.Project+"/spec:storage" {
		t.Fatalf("reloaded request differs: %+v", reloaded.Request)
	}
	again, err := s.OpenActionRequest(decisionRequest(), now)
	if err != nil || again.Request.ID != opened.Request.ID || again.Revision != 1 {
		t.Fatalf("opening the same request again is not idempotent: %+v %v", again, err)
	}
}

func TestActionRequestOpeningRefusesIncompleteRequests(t *testing.T) {
	s := actionFixture(t)
	now := time.Now()
	cases := map[string]func(*ActionRequest){
		"no targets":           func(r *ActionRequest) { r.Targets = nil },
		"no effects":           func(r *ActionRequest) { r.Effects = nil },
		"unknown phase":        func(r *ActionRequest) { r.Effects = []ObligationEffect{{Phase: "deploy", Mode: EffectBlock}} },
		"unqualified artifact": func(r *ActionRequest) { r.Targets = []NodeRef{{Artifact: "spec:other", Kind: "requirement", ID: "R1"}} },
		"single option":        func(r *ActionRequest) { r.Options = r.Options[:1] },
		"no question":          func(r *ActionRequest) { r.Question = " " },
		"consequence missing":  func(r *ActionRequest) { r.Options[1].Consequence = "" },
		"unknown origin":       func(r *ActionRequest) { r.Origin = "spec:nope" },
		"foreign origin":       func(r *ActionRequest) { r.Origin = "xref:proj.elsewhere/spec:storage" },
		"bad requester":        func(r *ActionRequest) { r.RequestedBy.Principal = "claude" },
		"approval with option": func(r *ActionRequest) { r.Kind = ActionApproval },
	}
	for name, mutate := range cases {
		r := decisionRequest()
		mutate(&r)
		if _, err := s.PrepareActionRequest(r, now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	unassigned := decisionRequest()
	unassigned.Recipient = ObligationActor{}
	prepared, err := s.PrepareActionRequest(unassigned, now)
	if err != nil || !prepared.Recipient.Unassigned {
		t.Fatalf("a request without owner stays visible as unassigned: %+v %v", prepared.Recipient, err)
	}
}

func TestOpenActionRequestIsProjectedAsAnActorObligation(t *testing.T) {
	s := actionFixture(t)
	opened, err := s.OpenActionRequest(decisionRequest(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.ProjectObligations(ObligationQuery{Category: ObligationActorAction})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Obligations) != 1 {
		t.Fatalf("obligations: %+v coverage %+v", r.Obligations, r.Coverage)
	}
	o := r.Obligations[0]
	if o.Recipient.Role != "maintainer" || o.Waiting != WaitingActor || !o.Restricts(PhaseExecution) || !o.Restricts(PhaseCloseout) || o.Restricts(PhaseStart) {
		t.Fatalf("projected action: %+v", o)
	}
	if !strings.Contains(o.Message, opened.Request.ID) || o.Source.Detail != opened.Request.ID {
		t.Fatalf("not navigable to the request: %+v", o)
	}
	for _, c := range r.Coverage {
		if c.Producer == "action-requests" && c.State != CoverageStateCurrent {
			t.Fatalf("action requests not covered: %+v", c)
		}
	}
}

func TestSupersededRequestLeavesTheProjection(t *testing.T) {
	s := actionFixture(t)
	first, _ := s.OpenActionRequest(decisionRequest(), time.Now())
	revised := decisionRequest()
	revised.Question = "Will the next delivery keep reading schema v1 and v2?"
	revised.Supersedes = first.Request.ID
	second, err := s.OpenActionRequest(revised, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	old, _ := s.LoadActionRequest(first.Request.ID)
	if old.State != ActionStateSuperseded || second.Request.RequestDigest == first.Request.RequestDigest {
		t.Fatalf("supersession: old %s, digests equal %v", old.State, second.Request.RequestDigest == first.Request.RequestDigest)
	}
	r, _ := s.ProjectObligations(ObligationQuery{Category: ObligationActorAction})
	if len(r.Obligations) != 1 || r.Obligations[0].Source.Detail != second.Request.ID {
		t.Fatalf("the superseded request is still projected: %+v", r.Obligations)
	}
}

// Spec pose-project-identity-file: a write never falls back to the directory
// name when .pose/project.json is malformed.
func TestActionWritesFailClosedOnAMalformedProjectFile(t *testing.T) {
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	s := actionFixture(t)
	view, err := s.OpenActionRequest(decisionRequest(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, ".pose", "project.json"), []byte(`{"schema_version":1,"project_id":"Not A Slug"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveDefaultProjectID(s.Root); err == nil || !strings.Contains(err.Error(), "invalid-project-id") {
		t.Fatalf("a malformed project file resolved: %v", err)
	}
	if _, err := s.OpenActionRequest(decisionRequest(), time.Now()); err == nil || !strings.Contains(err.Error(), "invalid-project-id") {
		t.Fatalf("an action request was opened under a derived project: %v", err)
	}
	res := ActionResolution{RequestID: view.Request.ID, Type: ActionEventAnswered, Actor: "human:maintainer", Answer: "preserve-v1", RequestDigest: view.Request.RequestDigest, ExpectedRevision: view.Revision, IdempotencyKey: "k1"}
	if _, err := s.ResolveActionRequest(res, time.Now()); err == nil || !strings.Contains(err.Error(), "invalid-project-id") {
		t.Fatalf("an answer was recorded under a derived project: %v", err)
	}
}

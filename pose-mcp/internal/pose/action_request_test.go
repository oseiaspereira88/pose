package pose

import (
	"errors"
	"fmt"
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

func TestGovernanceWriteGuardActionRequestFencesSubjectRootBeforeJournalWrite(t *testing.T) {
	s := actionFixture(t)
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	r := decisionRequest()
	prepared, err := s.PrepareActionRequest(r, now)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(actionJournalPath(s.Root, prepared.ID)) == filepath.Join(s.Root, ".pose", "specs") {
		t.Fatal("fixture must separate the subject input from the journal destination")
	}
	err = WithGovernanceWriteGuard("test.hold-subject-root", []Store{s}, func([]Store) error {
		if _, err := s.OpenActionRequest(r, now); !errors.Is(err, ErrGovernanceWriteGuardBusy) {
			return fmt.Errorf("action request under root contention = %v, want guard refusal", err)
		}
		if _, err := os.Lstat(actionJournalPath(s.Root, prepared.ID)); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("action journal exists after refused write: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceWriteGuardActionResolutionFencesForeignSubjectRoot(t *testing.T) {
	s := actionFixture(t)
	sourceRoot, _ := federatedTestComposedRoadmapSource(t, "source")
	resolver := federatedTestResolver("coordinator", "source", s.Root, sourceRoot)
	s.FederatedProjectID, s.FederatedResolver = "coordinator", &resolver
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	foreign := NodeRef{Artifact: "xref:source/spec:backend"}
	request := ActionRequest{
		Project:     "coordinator",
		Origin:      "spec:storage",
		RequestedBy: ActionPrincipal{Principal: "agent:impl", Execution: "foreign-subject-run"},
		Recipient:   ObligationActor{Unassigned: true},
		Kind:        ActionDecision,
		Question:    "Should this foreign material be adopted?",
		Options:     []ActionOption{{ID: "adopt", Consequence: "adopt the current material"}, {ID: "defer", Consequence: "defer the material"}},
		Targets:     []NodeRef{foreign},
		Effects:     []ObligationEffect{{Phase: PhaseExecution, Mode: EffectBlock, Scope: []NodeRef{foreign}}},
		Subject:     &ActionSubject{Node: foreign},
	}
	opened, err := s.OpenActionRequest(request, now)
	if err != nil {
		t.Fatalf("open federated request: %v", err)
	}
	resolve := ActionResolution{Type: ActionEventAnswered, RequestID: opened.Request.ID, RequestDigest: opened.Request.RequestDigest, ExpectedRevision: 1, IdempotencyKey: "foreign-subject-answer-1", Actor: "human:maintainer", Role: "maintainer", Answer: "adopt"}

	foreignStore := Store{Root: sourceRoot, FederatedProjectID: "source"}
	err = WithGovernanceWriteGuard("test.hold-foreign-subject", []Store{foreignStore}, func([]Store) error {
		if _, err := s.ResolveActionRequest(resolve, now); !errors.Is(err, ErrGovernanceWriteGuardBusy) {
			return fmt.Errorf("resolution while subject root is held = %v, want root busy", err)
		}
		events, err := s.readActionRequestEventsForWrite(opened.Request.ID)
		if err != nil {
			return err
		}
		if len(events) != 1 {
			return fmt.Errorf("local journal grew while foreign subject guard was busy: %d events", len(events))
		}
		writeErr := foreignStore.withGovernanceWriteGuard("test.foreign-spec-write", func(guarded Store) error {
			path := filepath.Join(guarded.Root, ".pose", "specs", "backend", "spec.md")
			return guarded.fsys().Publish(path, []byte("---\nslug: backend\nstatus: done\n---\n\n# Changed while fenced\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAny})
		})
		if !errors.Is(writeErr, ErrGovernanceWriteGuardBusy) {
			return fmt.Errorf("foreign writer while root is held = %v, want root busy", writeErr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	subjectPath := filepath.Join(sourceRoot, ".pose", "specs", "backend", "spec.md")
	subjectBytes, err := os.ReadFile(subjectPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subjectPath, append(subjectBytes, []byte("\nUncoordinated material change.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := s.ResolveActionRequest(resolve, now); err == nil {
		t.Fatal("resolution accepted a foreign subject changed after the request was opened")
	}
	events, err := s.readActionRequestEventsForWrite(opened.Request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("stale foreign subject appended a local answer: %d events", len(events))
	}
}

func TestGovernanceWriteGuardActionRefusesDiscoveredForeignRootWithoutExplicitBinding(t *testing.T) {
	s := actionFixture(t)
	projectsDir := t.TempDir()
	sourceRoot := filepath.Join(projectsDir, "source")
	writeReviewFixture(t, sourceRoot, ".pose/specs/backend.md", "---\nslug: backend\nstatus: done\n---\n\n# Backend\n")
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: s.Root, DefaultProjectID: "coordinator", ProjectsDir: projectsDir})}
	resolver.Authorize = func(string) bool { return true }
	s.FederatedProjectID, s.FederatedResolver = "coordinator", &resolver
	foreign := NodeRef{Artifact: "xref:source/spec:backend"}
	request := ActionRequest{
		Project: "coordinator", Origin: "spec:storage", RequestedBy: ActionPrincipal{Principal: "agent:impl"},
		Recipient: ObligationActor{Unassigned: true}, Kind: ActionDecision, Question: "Adopt foreign material?",
		Options: []ActionOption{{ID: "adopt", Consequence: "adopt"}, {ID: "defer", Consequence: "defer"}},
		Targets: []NodeRef{foreign}, Effects: []ObligationEffect{{Phase: PhaseExecution, Mode: EffectBlock, Scope: []NodeRef{foreign}}}, Subject: &ActionSubject{Node: foreign},
	}
	prepared, err := s.PrepareActionRequest(request, time.Now())
	if err != nil {
		t.Fatalf("read-discovered request should be readable: %v", err)
	}
	if _, err := s.OpenActionRequest(request, time.Now()); !errors.Is(err, ErrGovernanceWriteGuardRootBinding) {
		t.Fatalf("write through a discovered-only foreign root = %v, want explicit binding refusal", err)
	}
	if _, err := os.Lstat(actionJournalPath(s.Root, prepared.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("action journal exists after missing write authority: %v", err)
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

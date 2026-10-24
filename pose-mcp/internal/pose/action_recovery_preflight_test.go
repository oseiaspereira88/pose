package pose

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type recoveryFaultFiles struct {
	Files
	target, mode string
	fired        bool
}

func (f *recoveryFaultFiles) CreateExclusive(name string, data []byte, perm os.FileMode, durable bool) error {
	if name != f.target || f.fired {
		return f.Files.CreateExclusive(name, data, perm, durable)
	}
	f.fired = true
	switch f.mode {
	case "before-child":
		return errors.New("synthetic crash before successor creation")
	case "intent-tail":
		if err := f.Files.CreateExclusive(name, data[:len(data)/2], perm, true); err != nil {
			return err
		}
		return errors.New("synthetic interrupted intent")
	case "after-child":
		if err := f.Files.CreateExclusive(name, data, perm, true); err != nil {
			return err
		}
		return errors.New("synthetic successor Sync error after materialization")
	}
	return f.Files.CreateExclusive(name, data, perm, durable)
}
func (f *recoveryFaultFiles) syncActionJournal(file *os.File) error {
	if err := file.Sync(); err != nil {
		return err
	}
	if f.mode == "after-retirement" && !f.fired && file.Name() == f.target {
		f.fired = true
		return errors.New("synthetic Sync failure after retirement bytes")
	}
	return nil
}
func recoveryPrepared(t *testing.T, s Store, v ActionRequestView) ActionRequest {
	t.Helper()
	r := decisionRequest()
	r.Question = "Materially revised delivery question"
	r.Supersedes = v.Request.ID
	p, err := s.PrepareActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestActionRecoveryPartialOpenResumesExactlyOnce(t *testing.T) {
	for _, mode := range []string{"before-child", "after-child", "after-retirement"} {
		t.Run(mode, func(t *testing.T) {
			s := actionFixture(t)
			v := openedDecision(t, s)
			p := recoveryPrepared(t, s, v)
			target := actionJournalPath(s.Root, p.ID)
			if mode == "after-retirement" {
				target = actionJournalPath(s.Root, v.Request.ID)
			}
			fault := &recoveryFaultFiles{Files: NativeFiles(), target: target, mode: mode}
			failing := s
			failing.FS = fault
			if _, err := failing.OpenActionRequest(p, lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
				t.Fatalf("no explicit partial failure: %v", err)
			}
			prior, err := s.LoadActionRequest(v.Request.ID)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "after-retirement" {
				want := "reserved"
				if mode == "after-child" {
					want = "successor-opened"
				}
				if prior.Recovery == nil || prior.Recovery.Phase != want || prior.Satisfaction != SatisfactionPending || prior.NextStep.Kind != "resume-supersession" {
					t.Fatalf("hidden partial phase: %+v", prior)
				}
				if mode == "after-child" {
					child, err := s.LoadActionRequest(p.ID)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := s.ResolveActionRequest(answer(child, "human:maintainer", "preserve-v1", "early"), lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
						t.Fatalf("unretired child answered: %v", err)
					}
				}
			}
			for n := 0; n < 2; n++ {
				got, err := s.OpenActionRequest(p, lifecycleNow.Add(time.Duration(n)*time.Second))
				if err != nil || got.Request.ID != p.ID || got.Recovery == nil || got.Recovery.Phase != "complete" {
					t.Fatalf("resume %d: %+v %v", n, got, err)
				}
			}
			prior, err = s.LoadActionRequest(v.Request.ID)
			if err != nil || prior.Revision != 2 || prior.State != ActionStateSuperseded {
				t.Fatalf("duplicate/lost retirement: %+v %v", prior, err)
			}
		})
	}
}
func TestActionRecoveryCompetingIntentsAndCorruptionAreExplicit(t *testing.T) {
	s := actionFixture(t)
	v := openedDecision(t, s)
	p := recoveryPrepared(t, s, v)
	f := s
	f.FS = &recoveryFaultFiles{Files: NativeFiles(), target: actionJournalPath(s.Root, p.ID), mode: "before-child"}
	if _, err := f.OpenActionRequest(p, lifecycleNow); err == nil {
		t.Fatal("fault did not interrupt")
	}
	other := p
	other.Question = "A different material successor"
	if _, err := s.OpenActionRequest(other, lifecycleNow); !errors.Is(err, ErrActionIntentConflict) {
		t.Fatalf("competing intent won: %v", err)
	}
	// Preserve a corrupt reservation rather than manufacturing completion.
	path := actionIntentPath(s.Root, v.Request.ID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, before[:len(before)/2], 0644); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadActionRequest(v.Request.ID)
	if err != nil || got.Recovery == nil || got.Recovery.Phase != "repair-required" || got.Satisfaction != SatisfactionPending {
		t.Fatalf("corrupt phase hidden: %+v %v", got, err)
	}
	if _, err := s.OpenActionRequest(p, lifecycleNow); !errors.Is(err, ErrActionRecoveryUnknown) {
		t.Fatalf("corrupt reservation repaired silently: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, before[:len(before)/2]) {
		t.Fatal("corrupt reservation changed")
	}
}
func TestActionRecoveryConcurrentSuccessorsCannotBothRetire(t *testing.T) {
	s := actionFixture(t)
	v := openedDecision(t, s)
	a := recoveryPrepared(t, s, v)
	b := a
	b.Question = "Another material successor"
	var err error
	b, err = s.PrepareActionRequest(b, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, p := range []ActionRequest{a, b} {
		wg.Add(1)
		go func(p ActionRequest) {
			defer wg.Done()
			<-start
			_, err := s.OpenActionRequest(p, lifecycleNow)
			results <- err
		}(p)
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrActionIntentConflict) && !errors.Is(err, ErrActionRevisionConflict) {
			t.Fatalf("untyped contention: %v", err)
		}
	}
	got, err := s.LoadActionRequest(v.Request.ID)
	if err != nil || success != 1 || got.Revision != 2 || got.Recovery.Phase != "complete" {
		t.Fatalf("nonexclusive retirement: successes=%d view=%+v err=%v", success, got, err)
	}
}
func TestActionRecoveryPreflightRejectsStructuralFailuresWithoutWrites(t *testing.T) {
	for _, tc := range []string{"actor", "option", "digest", "revision", "project", "valid"} {
		t.Run(tc, func(t *testing.T) {
			s := actionFixture(t)
			v := openedDecision(t, s)
			res := answer(v, "human:maintainer", "preserve-v1", "preflight")
			switch tc {
			case "actor":
				res.Actor = "agent:intruder"
			case "option":
				res.Answer = "non-admitted"
			case "digest":
				res.RequestDigest = "stale"
			case "revision":
				res.ExpectedRevision = 7
			case "project":
				s2 := actionFixture(t)
				raw, err := os.ReadFile(actionJournalPath(s.Root, v.Request.ID))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(s2.Root, actionDir), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(actionJournalPath(s2.Root, v.Request.ID), raw, 0644); err != nil {
					t.Fatal(err)
				}
				s = s2
			}
			before, err := os.ReadFile(actionJournalPath(s.Root, v.Request.ID))
			if err != nil {
				t.Fatal(err)
			}
			preview, err := s.PreflightActionResolution(res, lifecycleNow)
			if tc == "valid" {
				if err != nil || !preview.Admissible || preview.Acceptance != "not-certified" {
					t.Fatalf("valid preview: %+v %v", preview, err)
				}
			} else if err == nil || preview.Admissible || preview.Refusal == nil {
				t.Fatalf("permissive preview: %+v %v", preview, err)
			}
			if preview.AuditPersisted {
				t.Fatal("fabricated refusal audit")
			}
			assertLifecycleUnchanged(t, s, v.Request.ID, before)
		})
	}
}
func TestActionRecoveryProofPendingAndStaleApply(t *testing.T) {
	s, v, _ := lifecycleFixture(t)
	res := answer(v, "human:maintainer", "preserve-v1", "preview")
	got, err := s.PreflightActionResolution(res, lifecycleNow)
	if err != nil || !got.Admissible || !got.ProofPending || got.Acceptance != "not-certified" {
		t.Fatalf("proof falsely certified: %+v %v", got, err)
	}
	if _, err := s.ResolveActionRequest(res, lifecycleNow); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("preview authorized unsigned apply: %v", err)
	}
	// The shared protected append rechecks even answers after policy changes.
	plain := actionFixture(t)
	view := openedDecision(t, plain)
	proposal := answer(view, "human:maintainer", "preserve-v1", "a")
	_, event, _, err := plain.prepareActionResolutionAgainst(view, proposal, lifecycleNow, false)
	if err != nil {
		t.Fatal(err)
	}
	writeLifecycleFile(t, plain, ".pose/policy/actions.json", `{"schema_version":2,"roles":{"maintainer":["human:other"]}}`)
	if _, err := plain.appendActionEvent(view.Request.ID, event, 1, lifecycleNow); !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("policy race authorized answer: %v", err)
	}
}
func TestActionRecoveryNextStepsAndSessionRespectUnmetConditions(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Kind, r.Options, r.Recommend = ActionApproval, nil, ""
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "self", Kind: "requirement", ID: "R4"}}
	v, err := s.OpenActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	declined, err := s.ResolveActionRequest(answer(v, "human:maintainer", "decline", "decline"), lifecycleNow)
	if err != nil || declined.NextStep == nil || declined.NextStep.Kind != "remediate-or-waive" {
		t.Fatalf("decline lost condition: %+v %v", declined, err)
	}
	presented := PresentActionRequests([]ActionRequestView{declined}, "")
	if len(presented.Groups) != 1 || len(presented.Groups[0].Requests[0].Options) != 0 {
		t.Fatalf("decline was re-asked or hidden: %+v", presented)
	}
	waiver := lifecycleRequest(declined, ActionEventWaived, "human:maintainer")
	waived, err := s.ResolveActionRequest(waiver, lifecycleNow)
	if err != nil || waived.Satisfaction != SatisfactionWaived {
		t.Fatalf("authorized waiver unsupported: %+v %v", waived, err)
	}
	other := r
	other.Question = "A distinct approval"
	view, err := s.OpenActionRequest(other, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveActionRequest(answer(view, "human:maintainer", "approve", "approved"), lifecycleNow); err != nil {
		t.Fatal(err)
	}
	current, err := s.LoadActionRequest(view.Request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(PresentActionRequests([]ActionRequestView{current}, "").Groups) != 0 {
		t.Fatal("valid satisfaction repeated")
	}
	writeLifecycleFile(t, s, ".pose/specs/2026-10-04-storage.md", "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R4: Changed material.\n")
	invalid, err := s.LoadActionRequest(view.Request.ID)
	if err != nil || invalid.NextStep.Kind != "replace-request" {
		t.Fatalf("invalidated next step: %+v %v", invalid, err)
	}
	if _, err := s.PreflightActionResolution(answer(invalid, "human:maintainer", "approve", "stale"), lifecycleNow); !errors.Is(err, ErrActionNotOpen) {
		t.Fatalf("old digest answer offered: %v", err)
	}
	writeLifecycleFile(t, s, ".pose/specs/2026-10-04-storage.md", "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R9: Different node.\n")
	unknown, err := s.LoadActionRequest(view.Request.ID)
	if err != nil || unknown.NextStep.Kind != "repair-subject" {
		t.Fatalf("missing subject hidden: %+v %v", unknown, err)
	}
	report, err := s.ProjectObligations(ObligationQuery{Category: ObligationActorAction})
	if err != nil {
		t.Fatal(err)
	}
	attention := BuildAttention(report, "")
	if len(attention.NextSteps) == 0 {
		t.Fatal("Attention lost typed next step")
	}
}
func TestActionRecoveryIdempotentResolutionAndConflictingKey(t *testing.T) {
	s := actionFixture(t)
	v := openedDecision(t, s)
	res := answer(v, "human:maintainer", "preserve-v1", "one")
	if _, err := s.ResolveActionRequest(res, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreflightActionResolution(res, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	again, err := s.ResolveActionRequest(res, lifecycleNow.Add(time.Second))
	if err != nil || again.Revision != 2 {
		t.Fatalf("retry duplicated: %+v %v", again, err)
	}
	res.Answer = "break-v1"
	if _, err := s.PreflightActionResolution(res, lifecycleNow); !errors.Is(err, ErrActionIdempotencyConflict) {
		t.Fatalf("conflicting retry accepted: %v", err)
	}
}
func TestActionRecoveryIncompleteRetryNeedsCurrentAuthority(t *testing.T) {
	s, v, key := lifecycleFixture(t)
	p := recoveryPrepared(t, s, v)
	auth := lifecycleRequest(v, ActionEventSuperseded, "human:maintainer")
	auth.SuccessorID, auth.SuccessorDigest = p.ID, p.RequestDigest
	auth = signLifecycle(t, v, auth, key)
	f := s
	f.FS = &recoveryFaultFiles{Files: NativeFiles(), target: actionJournalPath(s.Root, p.ID), mode: "after-child"}
	if _, err := f.OpenActionRequestAuthorized(p, &auth, lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
		t.Fatal(err)
	}
	writeLifecycleFile(t, s, ".pose/policy/actions.json", `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:other"]}}`)
	if _, err := s.OpenActionRequestAuthorized(p, &auth, lifecycleNow); !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("stale authority resumed retirement: %v", err)
	}
	prior, err := s.LoadActionRequest(v.Request.ID)
	if err != nil || prior.Revision != 1 || prior.Recovery.Phase != "successor-opened" {
		t.Fatalf("refusal changed predecessor: %+v %v", prior, err)
	}
}
func TestActionRecoveryTornIntentAndJournalRequireRepair(t *testing.T) {
	for _, mode := range []string{"intent-tail", "journal-tail"} {
		t.Run(mode, func(t *testing.T) {
			s := actionFixture(t)
			v := openedDecision(t, s)
			p := recoveryPrepared(t, s, v)
			target := actionIntentPath(s.Root, v.Request.ID)
			faultMode := mode
			if mode == "journal-tail" {
				target = actionJournalPath(s.Root, p.ID)
				faultMode = "after-child"
			}
			f := s
			f.FS = &recoveryFaultFiles{Files: NativeFiles(), target: target, mode: faultMode}
			if _, err := f.OpenActionRequest(p, lifecycleNow); err == nil {
				t.Fatal("fault failed")
			}
			if mode == "journal-tail" {
				file, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := fmt.Fprint(file, `{"schema_version":1`); err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.OpenActionRequest(p, lifecycleNow); err == nil {
				t.Fatal("torn data completed")
			}
			after, err := os.ReadFile(target)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("torn data silently rewritten")
			}
		})
	}
}

func TestActionRecoveryOpenSubjectChangeRefusesBeforeAnswer(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "self", Kind: "requirement", ID: "R4"}}
	v, err := s.OpenActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	res := answer(v, "human:maintainer", "preserve-v1", "new-decision")
	if _, err := s.PreflightActionResolution(res, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	writeLifecycleFile(t, s, ".pose/specs/2026-10-04-storage.md", "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R4: New material.\n")
	if _, err := s.PreflightActionResolution(res, lifecycleNow); !errors.Is(err, ErrActionNotOpen) {
		t.Fatalf("changed subject preview: %v", err)
	}
	if _, err := s.ResolveActionRequest(res, lifecycleNow); !errors.Is(err, ErrActionNotOpen) {
		t.Fatalf("changed subject apply: %v", err)
	}
	got, err := s.LoadActionRequest(v.Request.ID)
	if err != nil || got.Revision != 1 || got.NextStep.Kind != "replace-request" {
		t.Fatalf("old digest was answered: %+v %v", got, err)
	}
}

func TestActionRecoveryChainedOutgoingIntentRemainsRestrictive(t *testing.T) {
	s := actionFixture(t)
	a := openedDecision(t, s)
	b := recoveryPrepared(t, s, a)
	complete, err := s.OpenActionRequest(b, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	c := b
	c.ID = ""
	c.Question = "Third materially revised question"
	c.Supersedes = b.ID
	c.RequestedAt = lifecycleNow.Add(time.Second).Format(time.RFC3339)
	c, err = s.PrepareActionRequest(c, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	f := s
	f.FS = &recoveryFaultFiles{Files: NativeFiles(), target: actionJournalPath(s.Root, c.ID), mode: "before-child"}
	if _, err := f.OpenActionRequest(c, lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
		t.Fatal(err)
	}
	pending, err := s.LoadActionRequest(b.ID)
	if err != nil || pending.Recovery.Phase != "reserved" || pending.Recovery.Predecessor != b.ID {
		t.Fatalf("outgoing intent hidden by incoming completion: %+v %v", pending, err)
	}
	if _, err := s.ResolveActionRequest(answer(complete, "human:maintainer", "preserve-v1", "must-not-answer"), lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
		t.Fatalf("reserved predecessor answered: %v", err)
	}
	if _, err := s.OpenActionRequest(c, lifecycleNow); err != nil {
		t.Fatal(err)
	}
}
func TestActionRecoveryIncompleteIncomingCannotStartOutgoing(t *testing.T) {
	s := actionFixture(t)
	a := openedDecision(t, s)
	b := recoveryPrepared(t, s, a)
	f := s
	f.FS = &recoveryFaultFiles{Files: NativeFiles(), target: actionJournalPath(s.Root, b.ID), mode: "after-child"}
	if _, err := f.OpenActionRequest(b, lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
		t.Fatal(err)
	}
	c := b
	c.Question = "Third question"
	c.Supersedes = b.ID
	c.ID = ""
	if _, err := s.OpenActionRequest(c, lifecycleNow); !errors.Is(err, ErrActionRecoveryPending) {
		t.Fatalf("incomplete child used as predecessor: %v", err)
	}
}
func TestActionRecoveryDeclinedMaterialCannotBeReaskedIdentically(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Kind, r.Options, r.Recommend = ActionApproval, nil, ""
	v, err := s.OpenActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResolveActionRequest(answer(v, "human:maintainer", "decline", "decline"), lifecycleNow); err != nil {
		t.Fatal(err)
	}
	r.Supersedes = v.Request.ID
	r.RequestedAt = lifecycleNow.Add(time.Second).Format(time.RFC3339)
	if _, err := s.PreflightActionOpen(r, nil, lifecycleNow); !errors.Is(err, ErrActionMaterialUnchanged) {
		t.Fatalf("identical declined material preview: %v", err)
	}
	if _, err := s.OpenActionRequest(r, lifecycleNow); !errors.Is(err, ErrActionMaterialUnchanged) {
		t.Fatalf("identical declined material re-asked: %v", err)
	}
	if _, err := os.Stat(actionIntentPath(s.Root, v.Request.ID)); !os.IsNotExist(err) {
		t.Fatalf("refusal wrote reservation: %v", err)
	}
}

func TestActionRecoveryCorruptIncomingIntentCannotBeHidden(t *testing.T) {
	s := actionFixture(t)
	a := openedDecision(t, s)
	b := recoveryPrepared(t, s, a)
	bv, err := s.OpenActionRequest(b, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	c := b
	c.Question = "Third material question"
	c.Supersedes = b.ID
	c.ID = ""
	path := actionIntentPath(s.Root, a.Request.ID)
	if err := os.WriteFile(path, []byte(`{"schema_version":`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenActionRequest(c, lifecycleNow); !errors.Is(err, ErrActionRecoveryUnknown) {
		t.Fatalf("corrupt incoming intent bypassed: %v", err)
	}
	current, err := s.LoadActionRequest(bv.Request.ID)
	if err != nil || current.Recovery.Phase != "repair-required" {
		t.Fatalf("corruption hidden: %+v %v", current, err)
	}
}
func TestActionRecoveryAncestorCorruptionRemainsVisible(t *testing.T) {
	s := actionFixture(t)
	a := openedDecision(t, s)
	b := recoveryPrepared(t, s, a)
	if _, err := s.OpenActionRequest(b, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	c := b
	c.Question = "Third material question"
	c.Supersedes = b.ID
	c.ID = ""
	c, err := s.PrepareActionRequest(c, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenActionRequest(c, lifecycleNow); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(actionIntentPath(s.Root, a.Request.ID), []byte("corrupt"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadActionRequest(c.ID)
	if err != nil || got.Recovery.Phase != "repair-required" || got.NextStep.Kind != "repair" {
		t.Fatalf("ancestor corruption hidden: %+v %v", got, err)
	}
}

func TestActionRecoveryIntentBoundIsSharedWithPreview(t *testing.T) {
	s := actionFixture(t)
	v := openedDecision(t, s)
	r := decisionRequest()
	r.Supersedes = v.Request.ID
	r.Question = string(bytes.Repeat([]byte("x"), maxActionIntentBytes))
	if _, err := s.PreflightActionOpen(r, nil, lifecycleNow); !errors.Is(err, ErrActionRecoveryUnknown) {
		t.Fatalf("oversized intent preview accepted: %v", err)
	}
	if _, err := s.OpenActionRequest(r, lifecycleNow); !errors.Is(err, ErrActionRecoveryUnknown) {
		t.Fatalf("oversized intent written: %v", err)
	}
	if _, err := os.Stat(actionIntentPath(s.Root, v.Request.ID)); !os.IsNotExist(err) {
		t.Fatalf("refusal persisted intent: %v", err)
	}
}

func TestActionRecoveryEmptySliceRetryPreservesLegacyIdentity(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Kind = ActionInput
	r.Options = []ActionOption{}
	r.Recommend = ""
	r.Effects[0].Scope = []NodeRef{}
	legacy, err := s.PrepareActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Options == nil || legacy.Effects[0].Scope == nil {
		t.Fatal("preparation changed nil-vs-empty shape")
	}
	first, err := s.OpenActionRequest(legacy, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.OpenActionRequest(legacy, lifecycleNow.Add(time.Hour))
	if err != nil || again.Request.ID != first.Request.ID || again.Revision != 1 {
		t.Fatalf("legacy retry identity changed: %+v %v", again, err)
	}
}
func TestActionRecoveryTypedNextStepReachesWriteDiagnostics(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Kind, r.Options, r.Recommend = ActionApproval, nil, ""
	v, err := s.OpenActionRequest(r, lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.ResolveActionRequest(answer(v, "human:maintainer", "decline", "decline"), lifecycleNow)
	if err != nil {
		t.Fatal(err)
	}
	o := actionObligation(v.Request.Project, v)
	d := governedEffectDiagnostic(o, PhaseCloseout, "spec:storage")
	if d.NextStep == nil || d.NextStep.Kind != "remediate-or-waive" || d.Condition != v.NextStep.Reason {
		t.Fatalf("closeout lost next step: %+v", d)
	}
	report, err := s.ProjectObligations(ObligationQuery{Category: ObligationActorAction})
	if err != nil {
		t.Fatal(err)
	}
	spec, err := s.GetSpec("storage")
	if err != nil {
		t.Fatal(err)
	}
	phases := PhasesFromReport(spec, report)
	found := false
	for _, p := range phases.Phases {
		for _, restriction := range p.Restricting {
			if restriction.NextStep != nil && restriction.NextStep.Kind == "remediate-or-waive" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("readiness lost next step: %+v", phases)
	}
}

func TestActionRecoveryAuthorityTargetAndOperationMustAgree(t *testing.T) {
	for _, field := range []string{"target", "operation"} {
		t.Run(field, func(t *testing.T) {
			s := actionFixture(t)
			v := openedDecision(t, s)
			p := recoveryPrepared(t, s, v)
			auth := lifecycleRequest(v, ActionEventSuperseded, "agent:impl")
			auth.SuccessorID, auth.SuccessorDigest = p.ID, p.RequestDigest
			if field == "target" {
				auth.RequestID = "act-0000000000000000"
			} else {
				auth.Type = ActionEventWaived
			}
			if _, err := s.PreflightActionOpen(p, &auth, lifecycleNow); !errors.Is(err, ErrActionIntentConflict) {
				t.Fatalf("inconsistent authority preview: %v", err)
			}
			if _, err := s.OpenActionRequestAuthorized(p, &auth, lifecycleNow); !errors.Is(err, ErrActionIntentConflict) {
				t.Fatalf("inconsistent authority apply: %v", err)
			}
		})
	}
}

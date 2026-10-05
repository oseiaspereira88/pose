package pose

import (
	"strings"
	"testing"
	"time"
)

// Spec pose-transfer-preserves-obligations.

func openSourceRequest(t *testing.T, sourceRoot string) ActionRequestView {
	t.Helper()
	view, err := Store{Root: sourceRoot}.OpenActionRequest(ActionRequest{
		Origin: "spec:source-task", Kind: ActionApproval, Question: "Ship R1 as is?",
		RequestedBy: ActionPrincipal{Principal: "agent:impl"}, Recipient: ObligationActor{Role: "maintainer"},
		Targets: []NodeRef{{Artifact: "self", Kind: "requirement", ID: "R1"}}, Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}},
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	runTransferGit(t, sourceRoot, "add", ".pose")
	runTransferGit(t, sourceRoot, "commit", "-q", "-m", "open request")
	return view
}

func TestTransferInvalidatesSourceRequestsOnceEvenWhenResumed(t *testing.T) {
	// Each fixture root is its own project, identified by its directory; a
	// declared identity in the environment would make them all one project.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	resolver, sourceRoot, destinationRoot, _, _ := setupSiblingTransferFixture(t)
	view := openSourceRequest(t, sourceRoot)
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.ActionRequests) != 1 || plan.ActionRequests[0].ID != view.Request.ID || plan.ActionRequests[0].Disposition != "invalidate-in-source" {
		t.Fatalf("the preview does not list the source request: %+v", plan.ActionRequests)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, func(phase string) error {
		if phase == "prepared" {
			return specTransferError("injected-interruption")
		}
		return nil
	}); err == nil {
		t.Fatal("expected the injected interruption")
	}
	if mid, _ := (Store{Root: sourceRoot}).LoadActionRequest(view.Request.ID); mid.State != ActionStateOpen {
		t.Fatalf("interrupted before retirement, the source still owns the request: %+v", mid.State)
	}
	for i := 0; i < 2; i++ {
		if _, err := ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil); err != nil {
			t.Fatalf("resume %d: %v", i, err)
		}
	}
	after, err := Store{Root: sourceRoot}.LoadActionRequest(view.Request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.State != ActionStateInvalidated || after.Revision != 2 || !strings.Contains(after.Events[1].Reason, "grant nothing in the destination") {
		t.Fatalf("source request after transfer: state=%s revision=%d", after.State, after.Revision)
	}
	report, err := Store{Root: destinationRoot}.ProjectObligations(ObligationQuery{Category: ObligationActorAction})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Obligations) != 0 {
		t.Fatalf("a source request authorizes or restricts the destination: %+v", report.Obligations)
	}
}

func TestTransferRefusesARequestOpenedAfterThePreview(t *testing.T) {
	// Each fixture root is its own project, identified by its directory; a
	// declared identity in the environment would make them all one project.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	resolver, sourceRoot, _, _, _ := setupSiblingTransferFixture(t)
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	// Opened without a commit, so the source revision the plan pinned is
	// unchanged and only the request inventory differs.
	if _, err := (Store{Root: sourceRoot}).OpenActionRequest(ActionRequest{
		Origin: "spec:source-task", Kind: ActionApproval, Question: "Ship R1 as is?",
		RequestedBy: ActionPrincipal{Principal: "agent:impl"}, Recipient: ObligationActor{Role: "maintainer"},
		Targets: []NodeRef{{Artifact: "self"}}, Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	// The request inventory is part of the plan, so apply's re-preview no
	// longer matches it; resume additionally re-checks the inventory before
	// retiring the source.
	_, err = ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil)
	if err == nil || !(strings.Contains(err.Error(), "stale-plan") || strings.Contains(err.Error(), "action-requests-changed-since-preview")) {
		t.Fatalf("a request unknown to the plan was moved silently: %v", err)
	}
}

package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-delegated-approval-survives-routine-drift.

// approvedByARunThenDone approves the fixture's scope by a delegated run, concludes the
// run, and marks the spec done: the state of every spec closed by a delegated review.
func approvedByARunThenDone(t *testing.T) (runProvenanceFixture, ReviewRun) {
	t.Helper()
	f, draftPath := conclusionFixture(t)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "a run-backed approval")
	if conclusion, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || conclusion.State != ReviewConclusionRecorded || conclusion.Attestation == "" {
		t.Fatalf("conclusion: %+v %v", conclusion, err)
	}
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || !verification.Approved {
		t.Fatalf("the run-backed approval does not verify before the scope is done: %+v %v", verification.Blockers, err)
	}
	for _, kv := range [][]string{{"user.name", "fixture"}, {"user.email", "fixture@example.invalid"}, {"commit.gpgsign", "false"}} {
		federatedTestGit(t, f.store.Root, "config", kv[0], kv[1])
	}
	spec := filepath.Join(f.store.Root, ".pose/specs/backend/spec.md")
	raw, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec, []byte(strings.Replace(string(raw), "status: in-progress", "status: done", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, f.store.Root, "add", "--all")
	federatedTestGit(t, f.store.Root, "commit", "-q", "-m", "done")
	return f, run
}

// R1: a done scope keeps the approval a delegated run backed, when a file of its subject
// changed afterwards: the same retention a person's attestation already had.
func TestADelegatedApprovalSurvivesRoutineDriftOfADoneScope(t *testing.T) {
	f, _ := approvedByARunThenDone(t)
	commitReviewFixture(t, f.store.Root, "false") // a later change to a file the review sealed
	verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || verification.State != "closed" {
		t.Fatalf("a done scope lost its delegated approval to routine drift: state=%s blockers=%v", verification.State, verification.Blockers)
	}
	if !strings.Contains(strings.Join(verification.Warnings, " "), retainedCompletedReviewWarning) {
		t.Fatalf("the retained approval is not disclosed as retained: %v", verification.Warnings)
	}
}

// R3: an open scope keeps the live rule: the approval of a delegated run does not survive a
// change to what it sealed until the scope is done.
func TestADelegatedApprovalOfAnOpenScopeDoesNotSurviveDrift(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "an open scope")
	if conclusion, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || conclusion.Attestation == "" {
		t.Fatalf("conclusion: %+v %v", conclusion, err)
	}
	for _, kv := range [][]string{{"user.name", "fixture"}, {"user.email", "fixture@example.invalid"}, {"commit.gpgsign", "false"}} {
		federatedTestGit(t, f.store.Root, "config", kv[0], kv[1])
	}
	commitReviewFixture(t, f.store.Root, "false")
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || verification.Approved {
		t.Fatalf("an open scope kept a delegated approval over a change to its subject: %+v %v", verification, err)
	}
}

// R4: what the record says still binds. A run whose transcript was altered, or whose issuer
// no longer holds its pin, does not keep its approval, drift or none.
func TestARetainedDelegatedApprovalRefusesATamperedTranscript(t *testing.T) {
	f, run := approvedByARunThenDone(t)
	commitReviewFixture(t, f.store.Root, "false")
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || !verification.Approved {
		t.Fatalf("the approval must be retained before it is tampered with, or this test proves nothing: %+v %v", verification.Blockers, err)
	}
	path := filepath.Join(f.store.Root, filepath.FromSlash(run.TranscriptPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, []byte("\n{\"tampered\":true}\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || verification.Approved {
		t.Fatalf("an approval was retained over an altered transcript: %+v %v", verification.Blockers, err)
	}
}

func TestARetainedDelegatedApprovalRefusesARevokedIssuer(t *testing.T) {
	f, _ := approvedByARunThenDone(t)
	commitReviewFixture(t, f.store.Root, "false")
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || !verification.Approved {
		t.Fatalf("the approval must be retained before it is tampered with, or this test proves nothing: %+v %v", verification.Blockers, err)
	}
	policyPath := filepath.Join(f.store.Root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	revoked := strings.Replace(string(raw), f.key.Pin(), "ed25519:revoked", 1)
	if revoked == string(raw) {
		t.Fatal("the fixture policy does not pin the issuer")
	}
	if err := os.WriteFile(policyPath, []byte(revoked), 0o644); err != nil {
		t.Fatal(err)
	}
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || verification.Approved {
		t.Fatalf("an approval was retained after its issuer lost its pin: %+v %v", verification.Blockers, err)
	}
}

// R6: the assurance read for a retained approval is the one of the sealed run, verified.
func TestTheAssuranceOfARetainedDelegatedApprovalIsVerified(t *testing.T) {
	f, _ := approvedByARunThenDone(t)
	commitReviewFixture(t, f.store.Root, "false")
	verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref)
	if err != nil || !verification.Approved || verification.Bundle == nil || verification.Attestation == nil {
		t.Fatalf("the approval is not retained: %+v %v", verification.Blockers, err)
	}
	assurance := f.store.DescribeReviewAssurance(*verification.Bundle, verification.Attestation)
	if assurance.ReviewerIdentity != ReviewRunMachineAttested || assurance.Run == nil {
		t.Fatalf("the assurance of a retained delegated approval is not the sealed run's: %+v", assurance)
	}
}

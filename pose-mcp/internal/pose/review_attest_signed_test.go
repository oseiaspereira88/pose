package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-mcp-review-attest-signed-only: a confirmation of review
// conclusions is recorded from a trusted issuer's envelope over the exact
// attestation a preview completed.

func trustHumanAuthority(t *testing.T, f authorityFixture) {
	t.Helper()
	path := filepath.Join(f.root, ".pose/policy/review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["human_authority_issuers"] = []string{f.issuer + "#" + digestBytes(f.public)}
	encoded, _ := json.MarshalIndent(policy, "", "  ")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// confirmationDraft is an agent-prepared approval a person confirms.
func confirmationDraft(f authorityFixture, confirmer, role string) ReviewAttestation {
	att := approvedBundleAttestation(f.bundle, confirmer)
	att.Authority = &ReviewAuthorityClaim{
		SchemaVersion: ReviewBundleSchemaVersion, Project: "fixture-project", Audience: "fixture-project", BundleDigest: f.bundle.BundleDigest,
		Principal: confirmer, Role: role, ReviewExecution: "review-run-2", ImplementationPrincipal: "agent:implementer", ImplementationExecution: "implementation-run-1",
		Issuer: f.issuer, IssuedAt: f.now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: f.now.Add(time.Hour).Format(time.RFC3339),
	}
	att.Attribution = &ReviewAttribution{SchemaVersion: ReviewAttributionSchemaVersion, PreparedBy: "agent:writer", ConcludedBy: "agent:writer",
		ConfirmedBy: confirmer, ConfirmationMode: ReviewConfirmationAdoptedConclusions, AppliedBy: "agent:writer"}
	return att
}

func signEnvelope(f authorityFixture, att ReviewAttestation, canonical []byte) ReviewAttestationEnvelope {
	return ReviewAttestationEnvelope{SchemaVersion: ReviewBundleSchemaVersion, Issuer: f.issuer, Subject: att.BundleID, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(f.public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(f.private, canonical)), Attestation: att}
}

// R1/R2/R4: the preview completes the draft as recording would, the issuer
// signs those bytes, the envelope records that very attestation, and the
// human confirmation is disclosed as verified.
func TestSignedPreviewIsWhatTheEnvelopeRecordsAndTheConfirmationIsVerified(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	trustHumanAuthority(t, f)
	completed, canonical, err := f.store.SignableReviewAttestation(confirmationDraft(f, "human:maintainer", "human"), f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if completed.AttestationID == "" || completed.BundleDigest != f.bundle.BundleDigest || completed.AttestedAt == "" {
		t.Fatalf("the draft was not completed: %+v", completed)
	}
	if completed.Attribution.ConfirmationDigest != ReviewConfirmationDigest(completed) {
		t.Fatal("the confirmation is not bound to the conclusions it confirms")
	}
	again, _ := json.Marshal(completed)
	if string(again) != string(canonical) {
		t.Fatal("the signing bytes are not the completed attestation")
	}
	if _, err := os.Stat(filepath.Join(f.root, ".pose/review-attestations", completed.AttestationID+".json")); err == nil {
		t.Fatal("the preview recorded the attestation")
	}
	recorded, err := f.store.RecordReviewAttestationEnvelope(signEnvelope(f, completed, canonical), true)
	if err != nil {
		t.Fatal(err)
	}
	if recorded.AttestationID != completed.AttestationID {
		t.Fatalf("recorded %s, signed %s", recorded.AttestationID, completed.AttestationID)
	}
	assurance := f.store.DescribeReviewAssurance(f.bundle, &recorded)
	if assurance.Attribution == nil || assurance.Attribution.ConfirmationAssurance != ReviewIdentityAssuranceVerified {
		t.Fatalf("a signed human confirmation was not verified: %+v", assurance.Attribution)
	}
}

// R3: content changed after signing, or a confirmation rebound to other
// conclusions, is refused and nothing is recorded.
func TestAnEnvelopeOverOtherContentIsRefused(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	trustHumanAuthority(t, f)
	completed, canonical, err := f.store.SignableReviewAttestation(confirmationDraft(f, "human:maintainer", "human"), f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	tampered := completed
	tampered.Decision = "approved-with-reservations"
	if _, err := f.store.RecordReviewAttestationEnvelope(signEnvelope(f, tampered, canonical), true); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("content changed after signing was recorded: %v", err)
	}
	stranger := f
	seed := make([]byte, ed25519.SeedSize)
	stranger.private = ed25519.NewKeyFromSeed(seed)
	stranger.public = stranger.private.Public().(ed25519.PublicKey)
	if _, err := f.store.RecordReviewAttestationEnvelope(signEnvelope(stranger, completed, canonical), true); err == nil || !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("an untrusted key was accepted: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(f.root, ".pose/review-attestations"))
	if len(entries) != 0 {
		t.Fatalf("a refused envelope left %d records", len(entries))
	}
}

// R4: the same confirmation by an agent principal stays declared.
func TestAnAgentConfirmationStaysDeclared(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	trustHumanAuthority(t, f)
	completed, canonical, err := f.store.SignableReviewAttestation(confirmationDraft(f, "agent:reviewer", "agent"), f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := f.store.RecordReviewAttestationEnvelope(signEnvelope(f, completed, canonical), true)
	if err != nil {
		t.Fatal(err)
	}
	assurance := f.store.DescribeReviewAssurance(f.bundle, &recorded)
	if assurance.Attribution == nil || assurance.Attribution.ConfirmationAssurance != ReviewIdentityAssuranceDeclared {
		t.Fatalf("an agent confirmation was disclosed as %+v", assurance.Attribution)
	}
}

// The preview never fills a confirmation digest for a draft without a
// confirming principal, and refuses a draft for a bundle that does not exist.
func TestThePreviewBindsOnlyWhatItIsGiven(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	draft := approvedBundleAttestation(f.bundle, "agent:reviewer")
	completed, _, err := f.store.SignableReviewAttestation(draft, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Attribution != nil {
		t.Fatal("an attribution was invented for a draft that had none")
	}
	draft.BundleID = "rvb-0000000000000000"
	if _, _, err := f.store.SignableReviewAttestation(draft, f.now); err == nil {
		t.Fatal("a draft for an unknown bundle was completed")
	}
}

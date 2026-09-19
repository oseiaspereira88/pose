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

type authorityFixture struct {
	root    string
	store   Store
	bundle  ReviewBundle
	private ed25519.PrivateKey
	public  ed25519.PublicKey
	issuer  string
	now     time.Time
}

func verifiedAuthorityFixture(t *testing.T, independence string) authorityFixture {
	t.Helper()
	root, store := reviewBundleFixture(t)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 41)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	issuer := "conductor:authority-fixture"
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["reviewer_independence"] = map[string]any{"spec": independence}
	policy["identity_assurance"] = map[string]any{"spec": ReviewIdentityAssuranceVerified}
	policy["authority_audience"] = "fixture-project"
	policy["trusted_attestation_issuers"] = []string{issuer + "#" + digestBytes(public)}
	encoded, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	return authorityFixture{root: root, store: store, bundle: bundle, private: private, public: public, issuer: issuer, now: now}
}

func (f authorityFixture) attestation(t *testing.T, reviewer, implementationPrincipal, reviewExecution, implementationExecution string) ReviewAttestation {
	t.Helper()
	att := approvedBundleAttestation(f.bundle, reviewer)
	att.SchemaVersion = ReviewBundleSchemaVersion
	att.BundleDigest = f.bundle.BundleDigest
	att.AttestedAt = f.now.Add(time.Minute).Format(time.RFC3339)
	att.Authority = &ReviewAuthorityClaim{
		SchemaVersion:           ReviewSchemaVersion,
		Project:                 "fixture-project",
		BundleDigest:            f.bundle.BundleDigest,
		Principal:               reviewer,
		Role:                    "agent",
		ReviewExecution:         reviewExecution,
		ImplementationPrincipal: implementationPrincipal,
		ImplementationExecution: implementationExecution,
		Issuer:                  f.issuer,
		IssuedAt:                f.now.Add(-time.Minute).Format(time.RFC3339),
		ExpiresAt:               f.now.Add(time.Hour).Format(time.RFC3339),
		Audience:                "fixture-project",
	}
	att.AttestationID = reviewAttestationID(att)
	unsigned := att
	unsigned.Path = ""
	unsigned.Envelope = nil
	raw, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	att.Envelope = &ReviewAttestationSignature{
		Issuer:    f.issuer,
		Subject:   f.bundle.BundleID,
		Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(f.public),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(f.private, raw)),
	}
	return att
}

func verifyAuthorityAttestation(t *testing.T, f authorityFixture, att ReviewAttestation) ReviewBundleVerification {
	t.Helper()
	if _, err := f.store.RecordReviewAttestation(att, f.now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	verification, err := f.store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	return verification
}

func TestABMReviewAuthorityValid(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	verification := verifyAuthorityAttestation(t, f, att)
	if !verification.Approved {
		t.Fatalf("valid trusted authority claim was rejected: %v", verification.Blockers)
	}
}

func TestABMReviewAuthorityMissingClaim(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.SchemaVersion = ReviewBundleSchemaVersion
	att.BundleDigest = f.bundle.BundleDigest
	att.AttestedAt = f.now.Add(time.Minute).Format(time.RFC3339)
	att.AttestationID = reviewAttestationID(att)
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "carries no authority claim") {
		t.Fatalf("missing claim was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityPrincipalMustMatchReviewer(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	att.Authority.Principal = "agent:someone-else"
	att = resignAuthorityAttestation(t, f, att)
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "principal does not match") {
		t.Fatalf("principal mismatch was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityRejectsSameActor(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:reviewer", "review-run-2", "implementation-run-1")
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "same principal implemented and reviewed") {
		t.Fatalf("same actor was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityRejectsSameExecution(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "same-run", "same-run")
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "separate review execution") {
		t.Fatalf("same execution was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityRejectsReplayAndExpiry(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	att.Authority.Audience = "another-project"
	att = resignAuthorityAttestation(t, f, att)
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "addressed to another-project") {
		t.Fatalf("audience replay was accepted or diagnosed incorrectly: %+v", verification)
	}

	f = verifiedAuthorityFixture(t, "different-actor")
	att = f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	att.Authority.ExpiresAt = f.now.Add(-time.Minute).Format(time.RFC3339)
	att = resignAuthorityAttestation(t, f, att)
	verification = verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "authority claim expired") {
		t.Fatalf("expired claim was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityRejectsKeyRotationWithoutNewPin(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 99)
	}
	rotated := ed25519.NewKeyFromSeed(seed)
	unsigned := att
	unsigned.Path = ""
	unsigned.Envelope = nil
	raw, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	att.Envelope = &ReviewAttestationSignature{
		Issuer:    f.issuer,
		Subject:   f.bundle.BundleID,
		Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(rotated.Public().(ed25519.PublicKey)),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(rotated, raw)),
	}
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "untrusted review attestation issuer") {
		t.Fatalf("rotated unpinned key was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityHumanRoleNeedsGrant(t *testing.T) {
	f := verifiedAuthorityFixture(t, "mandatory-human")
	att := f.attestation(t, "human:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	att.Authority.Role = "human"
	att = resignAuthorityAttestation(t, f, att)
	verification := verifyAuthorityAttestation(t, f, att)
	if verification.Approved || !containsSubstring(verification.Blockers, "not authorised to assert a human principal") {
		t.Fatalf("human claim without grant was accepted or diagnosed incorrectly: %+v", verification)
	}
}

func TestABMReviewAuthorityPolicyRejectsUnknownAndIncompleteVerifiedConfig(t *testing.T) {
	root, store := reviewBundleFixture(t)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["identity_assurance"] = map[string]any{"spec": "maybe"}
	encoded, _ := json.Marshal(policy)
	if err := os.WriteFile(policyPath, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetReviewPolicy(); err == nil || !strings.Contains(err.Error(), "invalid identity assurance") {
		t.Fatalf("unknown assurance was accepted: %v", err)
	}

	policy["identity_assurance"] = map[string]any{"spec": ReviewIdentityAssuranceVerified}
	policy["authority_audience"] = ""
	encoded, _ = json.Marshal(policy)
	if err := os.WriteFile(policyPath, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetReviewPolicy(); err == nil || !strings.Contains(err.Error(), "authority_audience") {
		t.Fatalf("verified policy without audience was accepted: %v", err)
	}
}

func resignAuthorityAttestation(t *testing.T, f authorityFixture, att ReviewAttestation) ReviewAttestation {
	t.Helper()
	att.AttestationID = ""
	att.Path = ""
	att.SchemaVersion = ReviewBundleSchemaVersion
	att.AttestationID = reviewAttestationID(att)
	unsigned := att
	unsigned.Path = ""
	unsigned.Envelope = nil
	raw, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	att.Envelope = &ReviewAttestationSignature{
		Issuer:    f.issuer,
		Subject:   f.bundle.BundleID,
		Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(f.public),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(f.private, raw)),
	}
	return att
}

// grantHumanAuthority adds the issuer to the human grant list, so a claim can
// assert a person without that assertion being the thing under test.
func grantHumanAuthority(t *testing.T, f authorityFixture) {
	t.Helper()
	policyPath := filepath.Join(f.root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["human_authority_issuers"] = []string{f.issuer + "#" + digestBytes(f.public)}
	encoded, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// resealAfterPolicyChange re-seals the subject, because changing the policy
// changes the effective plan and the previously sealed bundle stops
// representing the current inputs — which is the staleness rule working, not a
// fixture problem.
func resealAfterPolicyChange(t *testing.T, f *authorityFixture) {
	t.Helper()
	bundle, err := f.store.SealReviewBundle("spec:backend", f.now)
	if err != nil {
		t.Fatal(err)
	}
	f.bundle = bundle
}

// The enum is ordered — same-actor-separate-execution, different-actor,
// mandatory-human — and the checks have to be ordered with it. This asserted
// only the role, so the strongest value verified less separation than the one
// below it: one human, implementing and reviewing in a single run, satisfied
// `mandatory-human` while `different-actor` refused exactly that
// (spec pose-abm-review-authority R5).
func TestABMReviewAuthorityMandatoryHumanKeepsDifferentActorSeparation(t *testing.T) {
	f := verifiedAuthorityFixture(t, "mandatory-human")
	grantHumanAuthority(t, f)
	resealAfterPolicyChange(t, &f)

	same := f.attestation(t, "human:reviewer", "human:reviewer", "one-run", "one-run")
	same.Authority.Role = "human"
	same = resignAuthorityAttestation(t, f, same)
	verification := verifyAuthorityAttestation(t, f, same)
	if verification.Approved {
		t.Fatal("one human implementing and reviewing in one run must not satisfy mandatory-human")
	}
	if !containsSubstring(verification.Blockers, "same principal implemented and reviewed") {
		t.Fatalf("the refusal must name the separation that failed: %v", verification.Blockers)
	}

	// A separate execution alone is still the same actor, which is what the
	// value below this one already refuses.
	separateRun := f.attestation(t, "human:reviewer", "human:reviewer", "review-run-2", "implementation-run-1")
	separateRun.Authority.Role = "human"
	separateRun = resignAuthorityAttestation(t, f, separateRun)
	if blockers := f.store.validateBundleAttestation(f.bundle, separateRun); !containsSubstring(blockers, "same principal implemented and reviewed") {
		t.Fatalf("a separate run by the same human must still be refused: %v", blockers)
	}

	// And a human reviewing another principal's work, in another run, passes.
	valid := f.attestation(t, "human:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	valid.Authority.Role = "human"
	valid = resignAuthorityAttestation(t, f, valid)
	if blockers := f.store.validateBundleAttestation(f.bundle, valid); len(blockers) != 0 {
		t.Fatalf("a granted human reviewing another actor's run must pass: %v", blockers)
	}
}

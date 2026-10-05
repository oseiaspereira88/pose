package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-authority-claim-project-is-not-the-audience: the project a claim
// governs and the verifier installation it is addressed to are two bindings.

func setAuthorityBindings(t *testing.T, root, project, audience string) {
	t.Helper()
	path := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	if project == "" {
		delete(policy, "authority_project")
	} else {
		policy["authority_project"] = project
	}
	policy["authority_audience"] = audience
	encoded, _ := json.MarshalIndent(policy, "", "  ")
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

// boundAttestation signs a review claim naming project and audience.
func (f authorityFixture) boundAttestation(t *testing.T, project, audience string) ReviewAttestation {
	t.Helper()
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	att.Authority.Project, att.Authority.Audience = project, audience
	att.Envelope = nil
	att.AttestationID = reviewAttestationID(att)
	unsigned := att
	unsigned.Path = ""
	raw, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatal(err)
	}
	att.Envelope = &ReviewAttestationSignature{
		Issuer: f.issuer, Subject: f.bundle.BundleID, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(f.public),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(f.private, raw)),
	}
	return att
}

func TestAuthorityProjectBindsAReviewClaimApartFromItsAudience(t *testing.T) {
	cases := []struct {
		name, project, audience, blocker string
	}{
		{"right project and audience", "proj.fixture", "harne8:tenant-a", ""},
		{"another project under the same verifier", "proj.other", "harne8:tenant-a", "the authority claim names project proj.other and this project is proj.fixture"},
		{"another verifier", "proj.fixture", "harne8:tenant-b", "the authority claim is addressed to harne8:tenant-b and this project answers to harne8:tenant-a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := verifiedAuthorityFixture(t, "different-actor")
			setAuthorityBindings(t, f.root, "proj.fixture", "harne8:tenant-a")
			// The policy is a sealed input: reseal against the bindings.
			bundle, err := f.store.SealReviewBundle("spec:backend", f.now)
			if err != nil {
				t.Fatal(err)
			}
			f.bundle = bundle
			verification := verifyAuthorityAttestation(t, f, f.boundAttestation(t, tc.project, tc.audience))
			if tc.blocker == "" {
				if !verification.Approved {
					t.Fatalf("a claim bound to this project and verifier was rejected: %v", verification.Blockers)
				}
				return
			}
			if verification.Approved || !containsSubstring(verification.Blockers, tc.blocker) {
				t.Fatalf("want blocker %q, got approved=%v %v", tc.blocker, verification.Approved, verification.Blockers)
			}
		})
	}
}

func TestAuthorityProjectIsRequiredByVerifiedAssurance(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	setAuthorityBindings(t, f.root, "", "harne8:tenant-a")
	if _, err := f.store.GetReviewPolicy(); err == nil || !strings.Contains(err.Error(), "authority_project") {
		t.Fatalf("verified assurance without authority_project was accepted: %v", err)
	}
	setAuthorityBindings(t, f.root, "not a project id", "harne8:tenant-a")
	if _, err := f.store.GetReviewPolicy(); err == nil || !strings.Contains(err.Error(), "authority_project") {
		t.Fatalf("a malformed authority_project was accepted: %v", err)
	}
}

func TestAuthorityProjectBindsAnActionClaimApartFromItsAudience(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 7)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	pin := "harne8:confirm#" + digestBytes(public)
	resolve := func(t *testing.T, policyProject, claimProject string) error {
		t.Helper()
		s := actionFixture(t)
		project := ""
		if policyProject != "" {
			project = `"authority_project":"` + policyProject + `",`
		}
		_ = os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), []byte(`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},`+project+`"authority_audience":"harne8:tenant-a","trusted_attestation_issuers":["`+pin+`"],"human_authority_issuers":["`+pin+`"]}`), 0o644)
		_ = os.WriteFile(filepath.Join(s.Root, ".pose/policy/actions.json"), []byte(`{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"identity_assurance":"verified"}`), 0o644)
		view := openedDecision(t, s)
		claim := ActionAuthorityClaim{SchemaVersion: 1, Project: claimProject, Audience: "harne8:tenant-a", RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest,
			Principal: "human:maintainer", Role: "human", Answer: "preserve-v1", Issuer: "harne8:confirm", IssuedAt: time.Now().UTC().Format(time.RFC3339)}
		canonical, _ := json.Marshal(claim)
		envelope := ActionClaimEnvelope{Issuer: "harne8:confirm", Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}
		res := answer(view, "human:maintainer", "preserve-v1", "k1")
		res.Claim, res.Envelope = &claim, &envelope
		_, err := s.ResolveActionRequest(res, time.Now())
		return err
	}
	if err := resolve(t, "proj.fixture", "proj.fixture"); err != nil {
		t.Fatalf("a claim bound to this project and verifier was refused: %v", err)
	}
	if err := resolve(t, "proj.fixture", "proj.other"); !errors.Is(err, ErrActionVerificationFailed) || !strings.Contains(err.Error(), "proj.other") {
		t.Fatalf("a claim for another project under the same verifier was accepted: %v", err)
	}
	if err := resolve(t, "", "proj.fixture"); !errors.Is(err, ErrActionVerificationFailed) || !strings.Contains(err.Error(), "authority_project") {
		t.Fatalf("a verified answer was accepted without authority_project: %v", err)
	}
}

func TestAuthorityProjectIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "`authority_project`") || !strings.Contains(string(raw), "`authority_audience`") {
			t.Fatalf("%s does not document both claim bindings", rel)
		}
	}
}

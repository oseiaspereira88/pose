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

// Spec pose-native-attestation-issuer.

func useIssuerHome(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "issuers")
	t.Setenv("POSE_ISSUER_HOME", dir)
	return dir
}

// pinNative writes the plan that trusts key into the fixture's policy.
func pinNative(t *testing.T, root string, key IssuerKey, attestations, human bool) IssuerPinPlan {
	t.Helper()
	docs, err := LoadPolicyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanIssuerPin(docs, key.Pin(), attestations, human, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := docs.Write(root, Store{Root: root}); err != nil {
		t.Fatal(err)
	}
	return plan
}

// reseal seals the fixture's scope again: a policy change is a semantic input,
// so a bundle sealed before it no longer represents the scope.
func reseal(t *testing.T, f *authorityFixture) {
	t.Helper()
	bundle, err := f.store.SealReviewBundle("spec:backend", f.now)
	if err != nil {
		t.Fatal(err)
	}
	f.bundle = bundle
}

// R1 and the security constraints: the key is created outside the project,
// owner-only, never overwritten, and refused once others can read it.
func TestNativeIssuerKeyIsPrivateAndNeverOverwritten(t *testing.T) {
	dir := useIssuerHome(t)
	key, err := CreateIssuerKey("", "maintainer", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key.Path, dir) {
		t.Fatalf("key written outside POSE_ISSUER_HOME: %s", key.Path)
	}
	for path, want := range map[string]os.FileMode{dir: 0o700, key.Path: 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s has mode %04o, want %04o", path, info.Mode().Perm(), want)
		}
	}
	if !strings.HasPrefix(key.Pin(), "maintainer#sha256:") || !validHumanAuthorityIssuerPin(key.Pin()) {
		t.Fatalf("malformed pin %q", key.Pin())
	}
	if _, err := CreateIssuerKey("", "maintainer", time.Now()); err == nil || !strings.Contains(err.Error(), "already has a key") {
		t.Fatalf("an existing key was overwritten: %v", err)
	}
	loaded, err := LoadIssuerKey("", "maintainer")
	if err != nil || loaded.Pin() != key.Pin() {
		t.Fatalf("reload changed the key: %v", err)
	}
	if err := os.Chmod(key.Path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadIssuerKey("", "maintainer"); err == nil || !strings.Contains(err.Error(), "accessible to other users") {
		t.Fatalf("a world-readable key was used: %v", err)
	}
	for _, bad := range []string{"Upper", "a#b", "../x", "", "a/b"} {
		if _, err := CreateIssuerKey("", bad, time.Now()); err == nil {
			t.Fatalf("issuer name %q was accepted", bad)
		}
	}
}

// R2/R5: pinning a native issuer keeps the external pin, and both issuers'
// envelopes count while an unpinned key is refused.
func TestNativeAndExternalIssuersCoexist(t *testing.T) {
	useIssuerHome(t)
	f := verifiedAuthorityFixture(t, "different-actor")
	external := f.issuer + "#" + digestBytes(f.public)
	native, err := CreateIssuerKey("", "maintainer", f.now)
	if err != nil {
		t.Fatal(err)
	}
	plan := pinNative(t, f.root, native, true, true)
	if len(plan.Changes) != 2 {
		t.Fatalf("pin changes: %v", plan.Changes)
	}
	docs, _ := LoadPolicyDocs(f.root)
	for _, key := range []string{"trusted_attestation_issuers"} {
		got := governedStringList(docs.Review[key])
		if len(got) != 2 || got[0] != external || got[1] != native.Pin() {
			t.Fatalf("%s = %v, want the external pin kept and the native pin added", key, got)
		}
	}
	if again, _ := PlanIssuerPin(docs, native.Pin(), true, true, "", ""); len(again.Changes) != 0 {
		t.Fatalf("pinning twice changed policy: %v", again.Changes)
	}

	// The native issuer signs and its envelope records.
	authority := &NativeAuthority{ReviewExecution: "review-run-2", ImplementationPrincipal: "agent:implementer", ImplementationExecution: "implementation-run-1", TTL: time.Hour}
	envelope, err := f.store.SignReviewAttestation(approvedBundleAttestation(f.bundle, "agent:reviewer"), native, authority, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.RecordReviewAttestationEnvelope(envelope, true); err != nil {
		t.Fatalf("a pinned native issuer was refused: %v", err)
	}
	// The external issuer still signs and records next to it.
	if _, err := f.store.RecordReviewAttestation(f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-3", "implementation-run-1"), f.now.Add(2*time.Minute)); err != nil {
		t.Fatalf("the external issuer stopped counting once a native one was pinned: %v", err)
	}
	// A native key that is not pinned is refused.
	stranger, err := CreateIssuerKey("", "stranger", f.now)
	if err != nil {
		t.Fatal(err)
	}
	unpinned, err := f.store.SignReviewAttestation(approvedBundleAttestation(f.bundle, "agent:reviewer"), stranger, nil, f.now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.RecordReviewAttestationEnvelope(unpinned, false); err == nil || !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("an unpinned native issuer was accepted: %v", err)
	}
}

// R3/R4: a natively signed attestation with a native authority claim
// satisfies verified identity, including a human reviewer when the issuer
// holds the human-authority grant, and is refused without that grant.
func TestNativeAuthorityClaimSatisfiesVerifiedIdentity(t *testing.T) {
	useIssuerHome(t)
	f := verifiedAuthorityFixture(t, "different-actor")
	native, err := CreateIssuerKey("", "maintainer", f.now)
	if err != nil {
		t.Fatal(err)
	}
	pinNative(t, f.root, native, true, false)
	reseal(t, &f)
	authority := &NativeAuthority{ReviewExecution: "review-run-2", ImplementationPrincipal: "agent:implementer", ImplementationExecution: "implementation-run-1"}
	envelope, err := f.store.SignReviewAttestation(approvedBundleAttestation(f.bundle, "human:ada"), native, authority, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Attestation.Authority == nil || envelope.Attestation.Authority.Issuer != "maintainer" || envelope.Attestation.Authority.Audience != "fixture-project" || envelope.Attestation.Authority.Role != "human" {
		t.Fatalf("the native claim is not bound to the issuer and the project: %+v", envelope.Attestation.Authority)
	}
	if _, err := f.store.RecordReviewAttestationEnvelope(envelope, true); err != nil {
		t.Fatal(err)
	}
	verification, err := f.store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved || !containsSubstring(verification.Blockers, "not authorised to assert a human principal") {
		t.Fatalf("a human claim from an issuer without the grant was approved: %v", verification.Blockers)
	}

	pinNative(t, f.root, native, false, true)
	reseal(t, &f)
	envelope, err = f.store.SignReviewAttestation(approvedBundleAttestation(f.bundle, "human:ada"), native, authority, f.now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.RecordReviewAttestationEnvelope(envelope, true); err != nil {
		t.Fatal(err)
	}
	verification, err = f.store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved {
		t.Fatalf("a native human claim with the grant was rejected: %v", verification.Blockers)
	}
}

// R7: rotation keeps the old key beside the new one, and an attestation the
// old key signed keeps verifying while its pin stays.
func TestRotatedIssuerKeepsEarlierSignaturesValid(t *testing.T) {
	useIssuerHome(t)
	f := verifiedAuthorityFixture(t, "different-actor")
	old, err := CreateIssuerKey("", "maintainer", f.now)
	if err != nil {
		t.Fatal(err)
	}
	pinNative(t, f.root, old, true, false)
	signed, err := f.store.SignReviewAttestation(approvedBundleAttestation(f.bundle, "agent:reviewer"), old, nil, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	retired, current, err := RotateIssuerKey("", "maintainer", f.now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if retired.Pin() != old.Pin() || current.Pin() == old.Pin() {
		t.Fatal("rotation did not produce a new key")
	}
	pinNative(t, f.root, current, true, false)
	if _, err := f.store.RecordReviewAttestationEnvelope(signed, false); err != nil {
		t.Fatalf("an attestation signed before the rotation stopped verifying: %v", err)
	}
	keys, err := ListIssuerKeys("")
	if err != nil || len(keys) != 2 || keys[0].Retired == keys[1].Retired {
		t.Fatalf("list after rotation: %+v %v", keys, err)
	}
}

// No serialized form a caller can reach carries the private key.
func TestNativeIssuerNeverExposesThePrivateKey(t *testing.T) {
	useIssuerHome(t)
	key, err := CreateIssuerKey("", "maintainer", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	seed := base64.StdEncoding.EncodeToString(key.private.Seed())
	keys, _ := ListIssuerKeys("")
	exposed, _ := json.Marshal(map[string]any{"key": key, "list": keys})
	if strings.Contains(string(exposed), seed) {
		t.Fatal("the private key is reachable through a serialized value")
	}
	if len(key.public) != ed25519.PublicKeySize {
		t.Fatal("public key not loaded")
	}
}

// R1: a key directory inside the project is refused, directly or through a
// symlink, before any key is written (found in review by agent:gpt-6.1-sol).
func TestNativeIssuerKeyIsRefusedInsideTheProject(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, ".pose", "issuers")
	t.Setenv("POSE_ISSUER_HOME", inside)
	if _, err := CreateIssuerKey(root, "maintainer", time.Now()); err == nil || !strings.Contains(err.Error(), "outside the project") {
		t.Fatalf("a key directory inside the project was accepted: %v", err)
	}
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("the refused directory was created: %v", err)
	}
	outside := t.TempDir()
	link := filepath.Join(outside, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_ISSUER_HOME", filepath.Join(link, "keys"))
	if _, err := CreateIssuerKey(root, "maintainer", time.Now()); err == nil || !strings.Contains(err.Error(), "outside the project") {
		t.Fatalf("a symlink into the project was accepted: %v", err)
	}
	t.Setenv("POSE_ISSUER_HOME", filepath.Join(outside, "keys"))
	if _, err := CreateIssuerKey(root, "maintainer", time.Now()); err != nil {
		t.Fatalf("a directory outside the project was refused: %v", err)
	}
	if _, err := LoadIssuerKey(root, "maintainer"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_ISSUER_HOME", filepath.Join(link, "keys"))
	if _, err := LoadIssuerKey(root, "maintainer"); err == nil {
		t.Fatal("signing from a key directory inside the project was allowed")
	}
}

// R1: the key file itself must be a regular, singly linked file: a symlink or
// a hard link can place the key inside the project even when the key
// directory is outside it (found in review by agent:gpt-6.1-sol).
func TestNativeIssuerKeyFileMustNotBeALink(t *testing.T) {
	root := t.TempDir()
	dir := useIssuerHome(t)
	key, err := CreateIssuerKey(root, "maintainer", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	inProject := filepath.Join(root, "maintainer.key")
	if err := os.Rename(key.Path, inProject); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inProject, key.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadIssuerKey(root, "maintainer"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("a key file symlinked into the project was loaded: %v", err)
	}
	if err := os.Remove(key.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(inProject, key.Path); err != nil {
		t.Skipf("hard links unsupported here: %v", err)
	}
	if _, err := LoadIssuerKey(root, "maintainer"); err == nil || !strings.Contains(err.Error(), "hard link") {
		t.Fatalf("a key file hard-linked into the project was loaded: %v", err)
	}
	_ = dir
}

// R7: rotations in the same second keep every retired key.
func TestRotationsInTheSameSecondKeepEveryRetiredKey(t *testing.T) {
	useIssuerHome(t)
	now := time.Now()
	first, err := CreateIssuerKey("", "maintainer", now)
	if err != nil {
		t.Fatal(err)
	}
	retired1, _, err := RotateIssuerKey("", "maintainer", now)
	if err != nil {
		t.Fatal(err)
	}
	retired2, current, err := RotateIssuerKey("", "maintainer", now)
	if err != nil {
		t.Fatal(err)
	}
	if retired1.Path == retired2.Path {
		t.Fatalf("two rotations archived to the same file %s", retired1.Path)
	}
	keys, err := ListIssuerKeys("")
	if err != nil || len(keys) != 3 {
		t.Fatalf("keys after two rotations: %+v %v", keys, err)
	}
	pins := map[string]bool{}
	for _, k := range keys {
		pins[k.Pin] = true
	}
	if !pins[first.Pin()] || !pins[current.Pin()] || len(pins) != 3 {
		t.Fatalf("a retired key was lost: %+v", keys)
	}
}

package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Spec pose-review-approval-domain-preflight: the shared writer refuses an
// approval the verifier would reject, before any artifact of the attempt is
// written, with the same typed refusal on every channel.

type preflightFixture struct {
	root    string
	store   Store
	bundle  ReviewBundle
	private ed25519.PrivateKey
	public  ed25519.PublicKey
	now     time.Time
}

const preflightIssuer = "test:preflight"

func newPreflightFixture(t *testing.T, policyEdits map[string]any) preflightFixture {
	t.Helper()
	root, store := reviewBundleFixture(t)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 7)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	edits := map[string]any{"trusted_attestation_issuers": []string{preflightIssuer + "#" + digestBytes(public)}}
	for key, value := range policyEdits {
		edits[key] = value
	}
	editReviewPolicy(t, root, edits)
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	return preflightFixture{root: root, store: store, bundle: bundle, private: private, public: public, now: now}
}

func editReviewPolicy(t *testing.T, root string, edits map[string]any) {
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
	for key, value := range edits {
		policy[key] = value
	}
	encoded, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// envelope signs att as recording completes it.
func (f preflightFixture) envelope(t *testing.T, att ReviewAttestation) ReviewAttestationEnvelope {
	t.Helper()
	completed, canonical, err := f.store.SignableReviewAttestation(att, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return ReviewAttestationEnvelope{SchemaVersion: ReviewBundleSchemaVersion, Issuer: preflightIssuer, Subject: completed.BundleID, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(f.public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(f.private, canonical)), Attestation: completed}
}

func (f preflightFixture) orphanFindingApproval() ReviewAttestation {
	att := approvedBundleAttestation(f.bundle, "agent:preflight")
	att.Criteria[0].Disposition = "finding"
	att.Criteria[0].Evidence = "finding-does-not-exist"
	return att
}

// V1/R2/R7: the same invalid approval through the Store, a signed envelope and
// a signed import gets the same typed refusal, and nothing is written.
func TestReviewDomainPreflightRefusesTheSameApprovalOnEveryChannel(t *testing.T) {
	f := newPreflightFixture(t, nil)
	att := f.orphanFindingApproval()
	att.AttestedAt = f.now.Add(time.Minute).Format(time.RFC3339)
	envelope := f.envelope(t, att)
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, f.root, "preflight-envelope.json", string(raw))
	before := poseTreeSnapshot(t, f.root)

	preview := f.store.PreflightReviewAttestation(att, f.now.Add(time.Minute))
	_, err = f.store.RecordReviewAttestation(att, f.now.Add(time.Minute))
	store := requireApprovalRefusal(t, f.root, before, err)
	_, err = f.store.RecordReviewAttestationEnvelope(envelope, true)
	signed := requireApprovalRefusal(t, f.root, before, err)
	_, err = f.store.ImportReviewAttestationEnvelope("preflight-envelope.json", true)
	imported := requireApprovalRefusal(t, f.root, before, err)

	if !containsSubstring(store, "which this attestation does not record") {
		t.Fatalf("the semantic blocker was not reported: %v", store)
	}
	for name, got := range map[string][]string{"preview": preview, "signed": signed, "import": imported} {
		if !reflect.DeepEqual(got, store) {
			t.Fatalf("%s diverged from the Store:\n%v\n%v", name, got, store)
		}
	}
	if !strings.Contains(err.Error(), "nothing was written") {
		t.Fatalf("the refusal does not say that nothing was written: %v", err)
	}
}

// V2: approved-with-reservations is an approval, held to the findings and
// reservation gates the bundle sealed.
func TestReviewDomainPreflightRefusesReservationsWithInvalidRisk(t *testing.T) {
	f := newPreflightFixture(t, nil)
	att := approvedBundleAttestation(f.bundle, "agent:preflight")
	att.Decision = "approved-with-reservations"
	att.Findings = []ReviewFinding{{ID: "f1", Severity: "critical", Disposition: "accepted-risk", Action: "ship anyway"}}
	before := poseTreeSnapshot(t, f.root)
	_, err := f.store.RecordReviewAttestation(att, f.now.Add(time.Minute))
	blockers := requireApprovalRefusal(t, f.root, before, err)
	for _, want := range []string{"finding f1 has unapproved or incomplete accepted risk"} {
		if !containsSubstring(blockers, want) {
			t.Fatalf("want %q in %v", want, blockers)
		}
	}
	if !f.bundle.Payload.SealedGates().AllowApprovedWithReservations && !containsSubstring(blockers, "review decision does not permit closeout: approved-with-reservations") {
		t.Fatalf("reservations the bundle does not allow were not refused: %v", blockers)
	}
}

// V3: each semantic blocker the verifier keeps is kept by the writer.
func TestReviewDomainPreflightKeepsEverySemanticBlocker(t *testing.T) {
	cases := []struct {
		name string
		edit func(f preflightFixture, att *ReviewAttestation)
		want string
	}{
		{"missing criterion", func(_ preflightFixture, att *ReviewAttestation) { att.Criteria = att.Criteria[1:] }, "missing criterion "},
		{"unknown finding", func(_ preflightFixture, att *ReviewAttestation) {
			att.Criteria[0].Disposition, att.Criteria[0].Evidence = "finding", "nobody-filed-this"
		}, "does not record"},
		{"missing tool", func(_ preflightFixture, att *ReviewAttestation) { att.Tools = nil }, "missing required review tool disposition"},
		{"evidence outside the bundle", func(_ preflightFixture, att *ReviewAttestation) {
			for i := range att.Criteria {
				if att.Criteria[i].Disposition == "passed" {
					att.Criteria[i].Evidence = "unit:absent/go/test"
					return
				}
			}
		}, "cites evidence absent from the sealed bundle"},
		{"reuse of an unknown source", func(f preflightFixture, att *ReviewAttestation) {
			criterion := f.bundle.Payload.Plan.Criteria[0]
			att.ReusedFrom = []ReviewAttestationReuse{{Criterion: criterion.ID, FromAttestation: "rva-0000000000000000", InputDigest: reviewCriterionInputDigest(f.bundle, criterion)}}
		}, "references an unavailable attestation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newPreflightFixture(t, map[string]any{"allow_criterion_reuse": true})
			att := approvedBundleAttestation(f.bundle, "agent:preflight")
			tc.edit(f, &att)
			before := poseTreeSnapshot(t, f.root)
			_, err := f.store.RecordReviewAttestation(att, f.now.Add(time.Minute))
			blockers := requireApprovalRefusal(t, f.root, before, err)
			if !containsSubstring(blockers, tc.want) {
				t.Fatalf("want %q in %v", tc.want, blockers)
			}
		})
	}
}

// V4/R4: a legitimate negative decision is recorded without meeting the
// approval bar, and is not exempted from the channel's authentication.
func TestReviewDomainPreflightRecordsNegativesButKeepsChannelProof(t *testing.T) {
	f := newPreflightFixture(t, nil)
	for i, decision := range []string{"changes-requested", "rejected"} {
		att := approvedBundleAttestation(f.bundle, "agent:preflight")
		att.Decision = decision
		att.Criteria = nil
		att.Findings = []ReviewFinding{{ID: "f1", Severity: "high", Disposition: "open", Action: "fix the contract"}}
		recorded, err := f.store.RecordReviewAttestation(att, f.now.Add(time.Duration(i+1)*time.Minute))
		if err != nil {
			t.Fatalf("%s was refused: %v", decision, err)
		}
		if _, err := f.store.LoadReviewAttestation(recorded.AttestationID); err != nil {
			t.Fatal(err)
		}
	}

	g := newPreflightFixture(t, map[string]any{"require_signed_attestations": true})
	negative := approvedBundleAttestation(g.bundle, "agent:preflight")
	negative.Decision = "rejected"
	before := poseTreeSnapshot(t, g.root)
	if _, err := g.store.RecordReviewAttestation(negative, g.now.Add(time.Minute)); err == nil || !strings.Contains(err.Error(), "requires a trusted signed") || errors.Is(err, ErrReviewApprovalRefused) {
		t.Fatalf("an unsigned negative bypassed the signed channel: %v", err)
	}
	assertPoseTreeUnchanged(t, g.root, before)
	if _, err := g.store.RecordReviewAttestationEnvelope(g.envelope(t, negative), true); err != nil {
		t.Fatalf("a signed negative was refused: %v", err)
	}
}

// V5/R5: a valid approval keeps its canonical identity, and a retry, signed or
// not, returns it without a second record.
func TestReviewDomainPreflightValidApprovalAndRetryKeepIdentity(t *testing.T) {
	f := newPreflightFixture(t, nil)
	att := approvedBundleAttestation(f.bundle, "agent:preflight")
	att.AttestedAt = f.now.Add(time.Minute).Format(time.RFC3339)
	if blockers := f.store.PreflightReviewAttestation(att, f.now.Add(time.Minute)); len(blockers) != 0 {
		t.Fatalf("a valid approval was refused by the preview: %v", blockers)
	}
	first, err := f.store.RecordReviewAttestation(att, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.store.RecordReviewAttestation(att, f.now.Add(2*time.Minute))
	if err != nil || again.AttestationID != first.AttestationID {
		t.Fatalf("retry changed identity: %s %s %v", first.AttestationID, again.AttestationID, err)
	}
	verification, err := f.store.VerifyReviewBundle("spec:backend")
	if err != nil || !verification.Approved || verification.Attestation.AttestationID != first.AttestationID {
		t.Fatalf("a valid approval does not verify: %+v %v", verification, err)
	}

	signedAtt := approvedBundleAttestation(f.bundle, "agent:preflight-signed")
	envelope := f.envelope(t, signedAtt)
	signed, err := f.store.RecordReviewAttestationEnvelope(envelope, true)
	if err != nil {
		t.Fatal(err)
	}
	retried, err := f.store.RecordReviewAttestationEnvelope(envelope, true)
	if err != nil || retried.AttestationID != signed.AttestationID || signed.AttestationID != envelope.Attestation.AttestationID {
		t.Fatalf("signed retry changed identity: %s %s %v", signed.AttestationID, retried.AttestationID, err)
	}
	entries, err := os.ReadDir(filepath.Join(f.root, ".pose/review-attestations"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("retries duplicated records: %d %v", len(entries), err)
	}
}

// V6/R5: apply revalidates the subject, so the approval of a superseded or
// stale bundle is not written and the artifacts already there stay.
func TestReviewDomainPreflightRefusesStaleOrSupersededBundles(t *testing.T) {
	f := newPreflightFixture(t, nil)
	stale := approvedBundleAttestation(f.bundle, "agent:preflight")
	envelope := f.envelope(t, stale)

	// The inputs change: the sealed bundle no longer represents them.
	writeReviewFixture(t, f.root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	before := poseTreeSnapshot(t, f.root)
	_, err := f.store.RecordReviewAttestation(stale, f.now.Add(time.Minute))
	if blockers := requireApprovalRefusal(t, f.root, before, err); !containsSubstring(blockers, "no longer represents the current semantic inputs") {
		t.Fatalf("a stale bundle was approved: %v", blockers)
	}

	// A newer bundle is sealed: the previewed and signed approval of the old one
	// is refused, while both bundles stay on disk.
	current, err := f.store.SealReviewBundle("spec:backend", f.now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if current.BundleID == f.bundle.BundleID {
		t.Fatal("fixture: the inputs did not change the bundle")
	}
	before = poseTreeSnapshot(t, f.root)
	_, err = f.store.RecordReviewAttestationEnvelope(envelope, true)
	if blockers := requireApprovalRefusal(t, f.root, before, err); !containsSubstring(blockers, "is superseded by "+current.BundleID) {
		t.Fatalf("the approval of a superseded bundle was written: %v", blockers)
	}
	if _, err := f.store.LoadReviewBundle(f.bundle.BundleID); err != nil {
		t.Fatalf("the superseded bundle was not preserved: %v", err)
	}
	if _, err := f.store.RecordReviewAttestation(approvedBundleAttestation(current, "agent:preflight"), f.now.Add(3*time.Minute)); err != nil {
		t.Fatalf("the current bundle cannot be approved: %v", err)
	}
}

// V7/R6: a valid proof over invalid semantics still discloses a verified
// identity, but is never written as an approval.
func TestReviewDomainPreflightSignatureIsNotApproval(t *testing.T) {
	f := newPreflightFixture(t, nil)
	envelope := f.envelope(t, f.orphanFindingApproval())
	verified, err := f.store.VerifyReviewAttestationEnvelope(envelope)
	if err != nil || verified.Envelope == nil {
		t.Fatalf("a trusted signature did not verify: %v", err)
	}
	if err := f.store.verifyStoredReviewAttestationSignature(verified); err != nil {
		t.Fatalf("the verified identity is not disclosable: %v", err)
	}
	if blockers := f.store.PreflightReviewAttestation(verified, f.now.Add(time.Minute)); !containsSubstring(blockers, "does not record") {
		t.Fatalf("the preview admitted a signed invalid approval: %v", blockers)
	}
	before := poseTreeSnapshot(t, f.root)
	_, err = f.store.RecordReviewAttestationEnvelope(envelope, true)
	requireApprovalRefusal(t, f.root, before, err)
	if attestations, err := f.store.ListReviewAttestations(f.bundle.BundleID); err != nil || len(attestations) != 0 {
		t.Fatalf("a signed invalid approval was recorded: %v %v", attestations, err)
	}
}

// R3: the preflight judges new writes only; an invalid approval persisted
// before it keeps being read and judged, not rewritten.
func TestReviewDomainPreflightLeavesHistoryAlone(t *testing.T) {
	f := newPreflightFixture(t, nil)
	historical := persistHistoricalAttestation(t, f.store, f.orphanFindingApproval(), f.now.Add(time.Minute))
	before := poseTreeSnapshot(t, f.root)
	verification, err := f.store.VerifyReviewBundle("spec:backend")
	if err != nil || verification.Approved || verification.Attestation == nil || verification.Attestation.AttestationID != historical.AttestationID {
		t.Fatalf("persisted history was not judged as it is: %+v %v", verification, err)
	}
	assertPoseTreeUnchanged(t, f.root, before)
}

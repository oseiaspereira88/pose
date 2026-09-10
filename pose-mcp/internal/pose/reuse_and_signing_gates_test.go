// Reuse is sealed, signing is not
// (spec pose-reuse-is-sealed-signing-stays-live).
//
// Criterion reuse carries a prior disposition into this attestation, so whether
// it was permitted is a property of the review that happened — read live, a
// project enabling reuse today would retroactively legitimise an attestation
// that reused a criterion when its own policy forbade it.
//
// Signing is the opposite. A project that starts requiring signed attestations
// is raising a bar, and a bundle sealed before that must not be permanently
// exempt: the exemption is exactly the gap a hurry would reach for.

package pose

import (
	"strings"
	"testing"
)

func TestCriterionReuseTakesTheSealedGate(t *testing.T) {
	store := Store{Root: t.TempDir()}
	att := ReviewAttestation{
		AttestationID: "rva-x", BundleID: "rvb-x", BundleDigest: "sha256:x",
		Decision: "approved", AttestedAt: "2026-09-10T12:00:00Z", Reviewer: "agent:x",
		ReusedFrom: []ReviewAttestationReuse{{Criterion: "c1", InputDigest: "sha256:d"}},
	}

	forbidden := ReviewBundle{
		BundleID: "rvb-x", BundleDigest: "sha256:x", State: "sealed",
		Payload: ReviewBundlePayload{Scope: ReviewBundleScope{Ref: "spec:alpha"}},
	}
	blockers := strings.Join(store.validateBundleAttestationWith(forbidden, att, true), "; ")
	if !strings.Contains(blockers, "reuse is not permitted by the bundle's sealed gates") {
		t.Errorf("a bundle whose gates forbid reuse accepted it: %s", blockers)
	}

	permitted := forbidden
	permitted.Payload.Gates = &ReviewBundleGates{AllowCriterionReuse: true}
	blockers = strings.Join(store.validateBundleAttestationWith(permitted, att, true), "; ")
	if strings.Contains(blockers, "reuse is not permitted") {
		t.Errorf("a bundle whose gates permit reuse refused it: %s", blockers)
	}
}

// The gate the bundle carries decides, not the policy on disk. This is the case
// the sealing exists for: an instance that enables reuse afterwards must not
// legitimise an attestation recorded when it did not.
func TestReuseIgnoresThePolicyOnDisk(t *testing.T) {
	root := t.TempDir()
	writeReviewFixture(t, root, ".pose/policy/review.json",
		`{"schema_version":2,"enabled":true,"allow_criterion_reuse":true,"profiles":{"spec":"spec-closeout@1"}}`)
	store := Store{Root: root}

	bundle := ReviewBundle{
		BundleID: "rvb-x", BundleDigest: "sha256:x", State: "sealed",
		Payload: ReviewBundlePayload{Scope: ReviewBundleScope{Ref: "spec:alpha"}},
	}
	att := ReviewAttestation{
		AttestationID: "rva-x", BundleID: "rvb-x", BundleDigest: "sha256:x",
		Decision: "approved", AttestedAt: "2026-09-10T12:00:00Z", Reviewer: "agent:x",
		ReusedFrom: []ReviewAttestationReuse{{Criterion: "c1", InputDigest: "sha256:d"}},
	}
	blockers := strings.Join(store.validateBundleAttestationWith(bundle, att, true), "; ")
	if !strings.Contains(blockers, "reuse is not permitted by the bundle's sealed gates") {
		t.Errorf("a policy enabling reuse today legitimised an attestation sealed without it: %s", blockers)
	}
}

// And signing stays live: the requirement is read from the policy, so raising
// the bar reaches a bundle sealed before it.
func TestSigningRequirementStaysLive(t *testing.T) {
	root := t.TempDir()
	writeReviewFixture(t, root, ".pose/policy/review.json",
		`{"schema_version":2,"enabled":true,"require_signed_attestations":true,"profiles":{"spec":"spec-closeout@1"}}`)
	store := Store{Root: root}

	// A bundle sealed before the requirement carries no gate for it, and is
	// held to it anyway.
	bundle := ReviewBundle{
		BundleID: "rvb-x", BundleDigest: "sha256:x", State: "sealed",
		Payload: ReviewBundlePayload{Scope: ReviewBundleScope{Ref: "spec:alpha"}},
	}
	att := ReviewAttestation{
		AttestationID: "rva-x", BundleID: "rvb-x", BundleDigest: "sha256:x",
		Decision: "approved", AttestedAt: "2026-09-10T12:00:00Z", Reviewer: "agent:x",
	}
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) == 0 {
		t.Error("an unsigned attestation passed under a policy requiring signatures")
	}
}

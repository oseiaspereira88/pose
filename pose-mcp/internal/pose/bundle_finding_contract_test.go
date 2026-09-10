// The bundle path enforces the finding contract
// (spec pose-bundle-findings-take-the-contract-the-legacy-path-had).
//
// `validateBundleAttestationWith` blocked a finding only when its disposition
// was `open` or `changes-requested`. Everything else passed: a `critical`
// accepted risk with no owner, no rationale and no review date; a disposition
// the engine does not know; a finding with neither severity nor action. The
// legacy attempt path refused all three, so a project that adopted review
// bundles silently lost the gate — and `allow_approved_with_reservations` and
// `accepted_risk_severities` stopped being consulted at all.
//
// Found by measuring the two paths against each other, not from a report.

package pose

import (
	"strings"
	"testing"
)

func gateFixture(gates ReviewBundleGates) (Store, ReviewBundle, ReviewAttestation) {
	store := Store{Root: "."}
	bundle := ReviewBundle{
		BundleID: "rvb-x", BundleDigest: "sha256:x", State: "sealed",
		Payload: ReviewBundlePayload{Scope: ReviewBundleScope{Ref: "spec:alpha"}, Gates: gates},
	}
	att := ReviewAttestation{
		AttestationID: "rva-x", BundleID: "rvb-x", BundleDigest: "sha256:x",
		Decision: "approved", AttestedAt: "2026-09-10T12:00:00Z", Reviewer: "agent:x",
	}
	return store, bundle, att
}

func TestBundlePathRefusesTheFindingsTheLegacyPathAlwaysDid(t *testing.T) {
	for _, tc := range []struct {
		name    string
		finding ReviewFinding
		want    string
	}{
		{
			"an accepted risk with no owner, rationale or review date",
			ReviewFinding{ID: "f1", Severity: "critical", Action: "accept", Disposition: "accepted-risk"},
			"unapproved or incomplete accepted risk",
		},
		{
			"a disposition the engine does not know",
			ReviewFinding{ID: "f2", Severity: "low", Action: "accept", Disposition: "banana"},
			"invalid disposition",
		},
		{
			"a finding with neither severity nor action",
			ReviewFinding{ID: "f3", Disposition: "resolved"},
			"lacks severity or action",
		},
		{
			"an open finding, which was the only one it ever caught",
			ReviewFinding{ID: "f4", Severity: "low", Action: "fix", Disposition: "open"},
			"is open",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, bundle, att := gateFixture(ReviewBundleGates{})
			att.Findings = []ReviewFinding{tc.finding}
			blockers := store.validateBundleAttestationWith(bundle, att, true)
			if !strings.Contains(strings.Join(blockers, "; "), tc.want) {
				t.Errorf("blockers = %v, want one containing %q", blockers, tc.want)
			}
		})
	}
}

// A complete accepted risk, of a severity the sealed gates accept, passes —
// otherwise the refusals above would be a gate that blocks everything.
func TestBundlePathAcceptsACompleteAcceptedRisk(t *testing.T) {
	store, bundle, att := gateFixture(ReviewBundleGates{AcceptedRiskSeverities: []string{"low"}})
	att.Findings = []ReviewFinding{{
		ID: "f1", Severity: "low", Action: "accept", Disposition: "accepted-risk",
		Owner: "@team", Rationale: "cost outweighs the risk", ReviewBy: "2027-01-01",
	}}
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) != 0 {
		t.Errorf("a complete accepted risk was refused: %v", blockers)
	}

	// And the severity list is what decides: the same finding at a severity the
	// gates do not accept is refused.
	att.Findings[0].Severity = "critical"
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) == 0 {
		t.Error("a severity outside the sealed accepted set was accepted")
	}
}

// The reservations flag comes from the bundle, so a policy edited after the
// fact cannot approve a closeout the review did not.
func TestApprovedWithReservationsTakesTheSealedGate(t *testing.T) {
	store, bundle, att := gateFixture(ReviewBundleGates{})
	att.Decision = "approved-with-reservations"
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) == 0 {
		t.Error("reservations were accepted by a bundle whose gates do not allow them")
	}

	store, bundle, att = gateFixture(ReviewBundleGates{AllowApprovedWithReservations: true})
	att.Decision = "approved-with-reservations"
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) != 0 {
		t.Errorf("a bundle sealed under a policy allowing reservations refused them: %v", blockers)
	}
}

// A bundle sealed before the gates existed carries none, which reads as the
// conservative answer and is exactly what that path did before: reservations
// refused, no severity accepted. The 469 attestations already recorded carry no
// findings at all, so nothing is re-judged.
func TestAnUnsealedGateIsTheConservativeReading(t *testing.T) {
	store, bundle, att := gateFixture(ReviewBundleGates{})
	att.Decision = "approved-with-reservations"
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) == 0 {
		t.Error("a bundle with no sealed gates allowed reservations")
	}
	store, bundle, att = gateFixture(ReviewBundleGates{})
	att.Findings = []ReviewFinding{{
		ID: "f1", Severity: "low", Action: "accept", Disposition: "accepted-risk",
		Owner: "@team", Rationale: "r", ReviewBy: "2027-01-01",
	}}
	if blockers := store.validateBundleAttestationWith(bundle, att, true); len(blockers) == 0 {
		t.Error("a bundle with no sealed severities accepted a risk")
	}
}

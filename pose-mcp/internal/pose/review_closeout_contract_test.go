package pose

import (
	"testing"
)

// The stamp is a date, not an instant, and it names the day the contract
// reached the instance. A review recorded at 09:00 on that day happened before
// the 15:00 update that delivered the contract, so it must be exempt — the
// promise is "everything reviewed before the update keeps its approval", and a
// midnight cutoff breaks it for exactly the work most likely to be affected.
func TestContractExemptionCoversReviewsEarlierOnTheAdoptionDay(t *testing.T) {
	for _, tc := range []struct {
		name       string
		reviewedAt string
		want       bool
	}{
		{"the day before", "2026-09-07T23:59:00Z", true},
		{"earlier the same day", "2026-09-08T09:00:00Z", true},
		{"late the same day", "2026-09-08T23:59:00Z", true},
		{"the next day", "2026-09-09T00:00:01Z", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := reviewPredatesAdoption("2026-09-08", tc.reviewedAt)
			if got != tc.want {
				t.Errorf("reviewedAt=%s exempt=%v, want %v", tc.reviewedAt, got, tc.want)
			}
		})
	}
}

// An unparseable or absent date exempts nothing: the waiver has to be a
// statement the instance made, never a parsing accident.
func TestContractExemptionRequiresAUsableDate(t *testing.T) {
	for _, stamped := range []string{"", "not-a-date", "2026-09-08T00:00:00Z"} {
		if reviewPredatesAdoption(stamped, "2020-01-01T00:00:00Z") {
			t.Errorf("stamped %q exempted a review", stamped)
		}
	}
	if reviewPredatesAdoption("2026-09-08", "not-a-timestamp") {
		t.Error("an unparseable review date exempted a review")
	}
	if !reviewPredatesAdoption("2026-09-08", "2020-01-01T00:00:00Z") {
		t.Error("a usable date exempted nothing; the test would pass by always rejecting")
	}
}

// The map-first contract has to hold for every registered contract, not only
// the one it was introduced for. A policy that records these dates only in
// `contract_adoptions` must load, and must be honoured by the exemptions —
// otherwise the registry advertises a shape that works in one place out of
// three.
func TestPolicyRecordingDatesOnlyInTheMapIsValidAndHonoured(t *testing.T) {
	root := t.TempDir()
	writeReviewFixture(t, root, ".pose/policy/review.json", `{
  "schema_version": 2, "enabled": true, "adopted_at": "2026-08-02",
  "profiles": {"spec": "spec-closeout@2"},
  "reviewer_independence": {"spec": "same-actor-separate-execution"},
  "component_aware": true, "unmapped_component_behavior": "warning",
  "review_bundles": true,
  "contract_adoptions": {
    "component-aware": "2026-08-13",
    "review-bundles": "2026-08-14",
    "evidence-vocabulary": "2026-09-08"
  }
}`)
	store := Store{Root: root}
	policy, _, err := store.loadReviewPolicy()
	if err != nil {
		t.Fatalf("a policy recording its dates only in the map failed to load: %v", err)
	}
	for id, want := range map[string]string{
		"component-aware":     "2026-08-13",
		"review-bundles":      "2026-08-14",
		"evidence-vocabulary": "2026-09-08",
	} {
		if got := policy.ContractAdoptedAt(id); got != want {
			t.Errorf("contract %s resolved to %q, want %q", id, got, want)
		}
	}

	// And the legacy field still wins nothing it should not: with the map
	// silent, the old key is read.
	legacy := ReviewPolicy{ReviewBundlesAdoptedAt: "2026-01-01"}
	if got := legacy.ContractAdoptedAt("review-bundles"); got != "2026-01-01" {
		t.Errorf("a policy written before the registry resolved to %q", got)
	}
	// The map wins when both are present, which is the documented order.
	both := ReviewPolicy{ReviewBundlesAdoptedAt: "2026-01-01", ContractAdoptions: map[string]string{"review-bundles": "2026-06-06"}}
	if got := both.ContractAdoptedAt("review-bundles"); got != "2026-06-06" {
		t.Errorf("the map did not win over the legacy field: %q", got)
	}
}

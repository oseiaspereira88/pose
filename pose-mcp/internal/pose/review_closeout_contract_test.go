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

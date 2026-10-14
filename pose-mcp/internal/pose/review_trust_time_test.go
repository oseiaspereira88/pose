package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewTrustTimeBoundaries(t *testing.T) {
	at := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, issued, expires string
		precision             time.Duration
		valid                 bool
	}{
		{"equality", at.Format(time.RFC3339), at.Format(time.RFC3339), 0, true},
		{"future", at.Add(time.Nanosecond).Format(time.RFC3339Nano), "", 0, false},
		{"reverse", at.Add(-time.Second).Format(time.RFC3339), at.Add(-2 * time.Second).Format(time.RFC3339), 0, false},
		{"expired", at.Add(-time.Hour).Format(time.RFC3339), at.Add(-time.Nanosecond).Format(time.RFC3339Nano), 0, false},
		{"no-expiry", at.Add(-time.Hour).Format(time.RFC3339), "", 0, true},
		{"historical-second", at.Add(500 * time.Millisecond).Format(time.RFC3339Nano), "", reviewTimestampPrecision(at.Format(time.RFC3339)), true},
		{"second-upper-bound", at.Add(time.Second).Format(time.RFC3339Nano), "", reviewTimestampPrecision(at.Format(time.RFC3339)), false},
		{"invalid-issued", "not-a-time", "", 0, false},
		{"invalid-expiry", at.Format(time.RFC3339), "not-a-time", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := reviewClaimTimeBlockers(&ReviewAuthorityClaim{IssuedAt: tc.issued, ExpiresAt: tc.expires}, at, tc.precision)
			if (len(b) == 0) != tc.valid {
				t.Fatalf("valid=%t blockers=%v", tc.valid, b)
			}
		})
	}
}

func TestReviewTrustTimeHistorySurvivesExpiryWithoutHistoricalAuthorization(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review", "implement")
	att.Authority.ExpiresAt = f.now.Add(time.Minute).Format(time.RFC3339)
	att = resignAuthorityAttestation(t, f, att)
	s := f.store.atReviewTime(f.now.Add(time.Hour))
	got := s.reviewAuthorityTime(att)
	if got.HistoricalValidity != "valid-claim" || got.CurrentValidity != "invalid-claim" || got.HistoricalAuthorization != "not-established" || got.TimeBasis != "issuer-asserted-signed-attested-at" {
		t.Fatalf("%+v", got)
	}
	if b := s.verifiedAuthorityBlockers(f.bundle, att); len(b) != 0 {
		t.Fatal(b)
	}
	a := s.DescribeReviewAssurance(f.bundle, &att)
	if a.ReviewerIdentity != ReviewIdentityAssuranceVerified || a.Temporal.CurrentValidity != "invalid-claim" {
		t.Fatalf("%+v", a)
	}
	before, err := os.ReadDir(filepath.Join(f.root, ".pose/review-attestations"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if _, err := s.recordReviewAttestation(att, f.now.Add(time.Hour), true); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired new operation: %v", err)
	}
	after, _ := os.ReadDir(filepath.Join(f.root, ".pose/review-attestations"))
	if len(after) != len(before) {
		t.Fatal("temporal rejection wrote an artifact")
	}
}

func TestReviewTrustTimeRevocationAndTamperingDoNotInventHistory(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review", "implement")
	att.AttestedAt = f.now.Add(-time.Hour).Format(time.RFC3339)
	if got := f.store.reviewAuthorityTime(att); got.HistoricalValidity == "valid-claim" || got.TimeBasis != "legacy-editable-attested-at" {
		t.Fatalf("tampered time promoted: %+v", got)
	}
	att = f.attestation(t, "agent:reviewer", "agent:implementer", "review", "implement")
	changeAuthorityPolicy(t, f, func(p map[string]any) { p["trusted_attestation_issuers"] = []string{} })
	got := f.store.reviewAuthorityTime(att)
	if got.CurrentTrust == "trusted-under-current-pins" || got.HistoricalValidity == "valid-claim" || got.HistoricalAuthorization != "not-established" {
		t.Fatalf("revoked pin promoted: %+v", got)
	}
}

func TestReviewTrustTimeFutureAndReverseClaimsAreRejected(t *testing.T) {
	for _, tc := range []string{"issued-future", "expiry-before-issue", "attested-future"} {
		t.Run(tc, func(t *testing.T) {
			f := verifiedAuthorityFixture(t, "different-actor")
			att := f.attestation(t, "agent:reviewer", "agent:implementer", "review", "implement")
			switch tc {
			case "issued-future":
				att.Authority.IssuedAt = f.now.Add(time.Hour).Format(time.RFC3339)
			case "expiry-before-issue":
				att.Authority.ExpiresAt = f.now.Add(-time.Hour).Format(time.RFC3339)
			case "attested-future":
				att.AttestedAt = f.now.Add(time.Hour).Format(time.RFC3339)
			}
			att = resignAuthorityAttestation(t, f, att)
			s := f.store.atReviewTime(f.now)
			if len(s.verifiedAuthorityBlockers(f.bundle, att)) == 0 || s.DescribeReviewAssurance(f.bundle, &att).ReviewerIdentity == ReviewIdentityAssuranceVerified {
				t.Fatal("invalid time context verified")
			}
		})
	}
}

func changeAuthorityPolicy(t *testing.T, f authorityFixture, change func(map[string]any)) {
	t.Helper()
	p := filepath.Join(f.root, ".pose/policy/review.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	change(policy)
	raw, err = json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

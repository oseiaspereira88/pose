package pose

import (
	"strings"
	"time"
)

// ReviewTrustTime separates an authenticated statement about historical time
// from present trust and from the time admissibility of a new operation.
// No local or issuer timestamp proves an observed historical authorization.
type ReviewTrustTime struct {
	TimeBasis               string   `json:"time_basis"`
	HistoricalValidity      string   `json:"historical_validity"`
	CurrentValidity         string   `json:"current_validity"`
	CurrentTrust            string   `json:"current_trust"`
	HistoricalAuthorization string   `json:"historical_authorization"`
	EvaluatedAt             string   `json:"evaluated_at"`
	HistoricalBlockers      []string `json:"historical_blockers,omitempty"`
	CurrentBlockers         []string `json:"current_blockers,omitempty"`
}

func (s Store) reviewClock() time.Time {
	if s.reviewTime != nil {
		return *s.reviewTime
	}
	return time.Now().UTC()
}

func (s Store) atReviewTime(now time.Time) Store {
	now = now.UTC()
	s.reviewTime = &now
	s.reviewReuse = nil
	return s
}

// VerifyReviewBundleAt supplies one explicit clock for all consumers of this
// verification. It changes no stored record or authorization policy.
func (s Store) VerifyReviewBundleAt(scope string, now time.Time) (ReviewBundleVerification, error) {
	return s.atReviewTime(now).VerifyReviewBundle(scope)
}

func (s Store) reviewOperation() Store {
	if s.reviewTime == nil {
		now := s.reviewClock()
		s.reviewTime = &now
	}
	if s.reviewReuse == nil {
		s.reviewReuse = newReviewReuseTraversal()
	}
	return s
}

// A second-resolution historical timestamp represents [second, next-second).
// Precision is not clock skew: fractional timestamps receive no tolerance.
func reviewTimestampPrecision(raw string) time.Duration {
	if !strings.Contains(raw, ".") && !strings.Contains(raw, ",") {
		return time.Second - time.Nanosecond
	}
	return 0
}

func reviewClaimTimeBlockers(claim *ReviewAuthorityClaim, at time.Time, precision time.Duration) []string {
	if claim == nil {
		return nil
	}
	blockers := []string{}
	issued, err := time.Parse(time.RFC3339Nano, claim.IssuedAt)
	if err != nil {
		return []string{"the authority claim has no valid issued_at"}
	}
	if at.IsZero() {
		return []string{"authority time context is missing"}
	}
	if issued.After(at.Add(precision)) {
		blockers = append(blockers, "the authority claim issued_at is in the future of the applicable time context")
	}
	if claim.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339Nano, claim.ExpiresAt)
		if err != nil {
			blockers = append(blockers, "the authority claim has an unparseable expires_at")
		} else {
			if expiry.Before(issued) {
				blockers = append(blockers, "the authority claim expires before it was issued")
			}
			if expiry.Before(at) {
				blockers = append(blockers, "the authority claim expired at "+claim.ExpiresAt)
			}
		}
	}
	return blockers
}

func reviewNewOperationTimeBlockers(att ReviewAttestation, now time.Time) []string {
	blockers := reviewClaimTimeBlockers(att.Authority, now, 0)
	if att.Authority != nil {
		at, err := time.Parse(time.RFC3339Nano, att.AttestedAt)
		if err != nil {
			blockers = append(blockers, "attested_at must be RFC3339")
		} else if at.After(now) {
			blockers = append(blockers, "attested_at is in the future of the operation")
		}
	}
	return blockers
}

func (s Store) reviewAuthorityTime(att ReviewAttestation) ReviewTrustTime {
	now := s.reviewClock()
	out := ReviewTrustTime{TimeBasis: "legacy-editable-attested-at", HistoricalValidity: "not-established", CurrentValidity: "not-evaluated", CurrentTrust: "not-verified", HistoricalAuthorization: "not-established", EvaluatedAt: now.Format(time.RFC3339Nano)}
	if att.Authority == nil {
		return out
	}
	out.CurrentBlockers = reviewNewOperationTimeBlockers(att, now)
	if err := s.verifyStoredReviewAttestationSignature(att); err != nil || att.Envelope == nil {
		out.HistoricalBlockers = []string{"historical claim lacks authenticated time under current issuer trust"}
		out.CurrentBlockers = append(out.CurrentBlockers, out.HistoricalBlockers...)
	} else {
		out.CurrentTrust = "trusted-under-current-pins"
		out.TimeBasis = "issuer-asserted-signed-attested-at"
		at, err := time.Parse(time.RFC3339Nano, att.AttestedAt)
		if err != nil {
			out.HistoricalBlockers = []string{"attested_at must be RFC3339"}
		} else {
			if at.After(now) {
				out.HistoricalBlockers = append(out.HistoricalBlockers, "signed attested_at is in the future of the verification context")
			}
			out.HistoricalBlockers = append(out.HistoricalBlockers, reviewClaimTimeBlockers(att.Authority, at, reviewTimestampPrecision(att.AttestedAt))...)
		}
		if len(out.HistoricalBlockers) == 0 {
			out.HistoricalValidity = "valid-claim"
		} else {
			out.HistoricalValidity = "invalid-claim"
		}
	}
	if len(out.CurrentBlockers) == 0 {
		out.CurrentValidity = "valid-claim"
	} else {
		out.CurrentValidity = "invalid-claim"
	}
	return out
}

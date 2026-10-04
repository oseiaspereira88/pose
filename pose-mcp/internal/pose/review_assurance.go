package pose

import "strings"

// ReviewAssurance discloses what a review record proves about who reviewed
// and how separate that reviewer was (spec pose-review-assurance-disclosure).
//
// Three separation readings are kept apart because they answer different
// questions. Required is what the plan demands. Declared is what the record's
// own identity strings say, which under `declared` assurance anyone can type.
// Verified is what an authorized issuer's signed claim, bound to this bundle,
// established. None of them is cognitive independence: the engine can observe
// principals, executions and credentials, never whether a second reviewer
// thought independently, so that field is a constant and exists to be read.
type ReviewAssurance struct {
	IdentityAssurance     string                `json:"identity_assurance"`
	Reviewer              string                `json:"reviewer,omitempty"`
	ReviewerIdentity      string                `json:"reviewer_identity"`
	ReviewerRole          string                `json:"reviewer_role"`
	SeparationRequired    string                `json:"separation_required"`
	SeparationDeclared    string                `json:"separation_declared"`
	SeparationVerified    string                `json:"separation_verified"`
	VerifiedClaim         *ReviewAssuranceClaim `json:"verified_claim,omitempty"`
	CognitiveIndependence string                `json:"cognitive_independence"`
	Limitations           []string              `json:"limitations"`
}

// ReviewAssuranceClaim is the part of a verified authority claim the
// disclosure names: who, in which role, which two executions, which issuer,
// for which project.
type ReviewAssuranceClaim struct {
	Principal               string `json:"principal"`
	Role                    string `json:"role"`
	ReviewExecution         string `json:"review_execution"`
	ImplementationPrincipal string `json:"implementation_principal,omitempty"`
	ImplementationExecution string `json:"implementation_execution,omitempty"`
	Issuer                  string `json:"issuer"`
	Audience                string `json:"audience"`
	BundleDigest            string `json:"bundle_digest"`
}

const (
	reviewAssuranceNotVerified   = "not-verified"
	reviewAssuranceNotRequired   = "not-required"
	reviewCognitiveNotObservable = "not-observable"

	// Limitation codes. They are stable identifiers; renders explain them.
	ReviewLimitationIdentityDeclared   = "identity-is-the-string-the-reviewer-wrote"
	ReviewLimitationSeparationDeclared = "separation-is-declared-not-authenticated"
	ReviewLimitationClaimNotVerified   = "verified-assurance-required-but-claim-not-verified"
	ReviewLimitationIssuerTruthfulness = "issuer-truthfulness-is-the-issuers-authority"
	ReviewLimitationCognitive          = "cognitive-independence-is-not-observable"
	ReviewLimitationNoRecord           = "no-review-record"
)

func reviewDeclaredRole(reviewer string) string {
	switch {
	case strings.HasPrefix(reviewer, "human:"):
		return "human"
	case strings.HasPrefix(reviewer, "agent:"):
		return "agent"
	}
	return "unknown"
}

// reviewDeclaredSeparation is the prefix reading the declared path performs,
// named as a declaration.
func reviewDeclaredSeparation(reviewer string) string {
	switch {
	case reviewer == "":
		return "none"
	case strings.HasPrefix(reviewer, "human:"):
		return "declared-human"
	case strings.HasPrefix(reviewer, "agent:independent-"):
		return "declared-independent-agent"
	case strings.HasPrefix(reviewer, "agent:"):
		return "declared-agent"
	}
	return "malformed"
}

func reviewVerifiedSeparation(independence string) string {
	switch independence {
	case "same-actor-separate-execution":
		return "execution"
	case "different-actor":
		return "actor-and-execution"
	case "mandatory-human":
		return "human-actor-and-execution"
	case "":
		return reviewAssuranceNotRequired
	}
	return independence
}

// DescribeReviewAssurance discloses a bundle attestation. att may be nil when
// the bundle has no attestation yet; the disclosure then states what will be
// required and that nothing was verified.
func (s Store) DescribeReviewAssurance(bundle ReviewBundle, att *ReviewAttestation) ReviewAssurance {
	out := ReviewAssurance{
		IdentityAssurance:     bundle.Payload.SealedGates().IdentityAssurance,
		SeparationRequired:    firstNonempty(bundle.Payload.Plan.Independence, "none"),
		SeparationVerified:    reviewAssuranceNotVerified,
		ReviewerIdentity:      reviewAssuranceNotVerified,
		ReviewerRole:          "unknown",
		SeparationDeclared:    "none",
		CognitiveIndependence: reviewCognitiveNotObservable,
	}
	if att == nil {
		out.Limitations = []string{ReviewLimitationNoRecord, ReviewLimitationCognitive}
		return out
	}
	out.Reviewer = att.Reviewer
	out.SeparationDeclared = reviewDeclaredSeparation(att.Reviewer)
	out.ReviewerRole = reviewDeclaredRole(att.Reviewer)
	if out.IdentityAssurance != ReviewIdentityAssuranceVerified {
		out.ReviewerIdentity = ReviewIdentityAssuranceDeclared
		out.Limitations = []string{ReviewLimitationIdentityDeclared, ReviewLimitationSeparationDeclared, ReviewLimitationCognitive}
		return out
	}
	if att.Authority == nil || len(s.verifiedAuthorityBlockers(bundle, *att)) > 0 {
		out.Limitations = []string{ReviewLimitationClaimNotVerified, ReviewLimitationCognitive}
		return out
	}
	claim := att.Authority
	out.ReviewerIdentity = ReviewIdentityAssuranceVerified
	out.ReviewerRole = claim.Role
	out.SeparationVerified = reviewVerifiedSeparation(bundle.Payload.Plan.Independence)
	out.VerifiedClaim = &ReviewAssuranceClaim{
		Principal: claim.Principal, Role: claim.Role, ReviewExecution: claim.ReviewExecution,
		ImplementationPrincipal: claim.ImplementationPrincipal, ImplementationExecution: claim.ImplementationExecution,
		Issuer: claim.Issuer, Audience: claim.Audience, BundleDigest: claim.BundleDigest,
	}
	out.Limitations = []string{ReviewLimitationIssuerTruthfulness, ReviewLimitationCognitive}
	return out
}

// describeAttemptAssurance discloses the legacy attempt path, which has no
// sealed bundle and therefore no verified assurance at all.
func describeAttemptAssurance(assurance, independence string, attempt *ReviewAttempt) ReviewAssurance {
	out := ReviewAssurance{
		IdentityAssurance: firstNonempty(assurance, ReviewIdentityAssuranceDeclared), SeparationRequired: firstNonempty(independence, "none"),
		SeparationVerified: reviewAssuranceNotVerified, ReviewerIdentity: ReviewIdentityAssuranceDeclared,
		ReviewerRole: "unknown", SeparationDeclared: "none", CognitiveIndependence: reviewCognitiveNotObservable,
		Limitations: []string{ReviewLimitationIdentityDeclared, ReviewLimitationSeparationDeclared, ReviewLimitationCognitive},
	}
	if attempt == nil {
		out.ReviewerIdentity = reviewAssuranceNotVerified
		out.Limitations = []string{ReviewLimitationNoRecord, ReviewLimitationCognitive}
		return out
	}
	out.Reviewer = attempt.Reviewer
	out.ReviewerRole = reviewDeclaredRole(attempt.Reviewer)
	out.SeparationDeclared = reviewDeclaredSeparation(attempt.Reviewer)
	if out.IdentityAssurance == ReviewIdentityAssuranceVerified {
		// A legacy attempt carries no signed claim, so a policy that requires
		// verified assurance has verified nothing about it.
		out.ReviewerIdentity = reviewAssuranceNotVerified
		out.Limitations = []string{ReviewLimitationClaimNotVerified, ReviewLimitationCognitive}
	}
	return out
}

// RenderReviewAssurance is the one-line human summary every review surface
// prints, so no surface can describe the same record more strongly than
// another.
func RenderReviewAssurance(a ReviewAssurance) string {
	reviewer := a.Reviewer
	if reviewer == "" {
		reviewer = "(no record)"
	}
	identity := "declared identity"
	switch a.ReviewerIdentity {
	case ReviewIdentityAssuranceVerified:
		identity = "verified " + a.ReviewerRole + " identity"
	case reviewAssuranceNotVerified:
		identity = "identity not verified"
	}
	return reviewer + " — " + identity + "; assurance " + a.IdentityAssurance +
		"; separation required " + a.SeparationRequired + ", declared " + a.SeparationDeclared + ", verified " + a.SeparationVerified +
		"; cognitive independence " + a.CognitiveIndependence
}

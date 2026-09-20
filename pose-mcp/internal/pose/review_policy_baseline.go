package pose

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// A diff that changes the review contract cannot be the authority that approves
// itself. When the reviewed scope touches the review policy or a review profile,
// the obligations are resolved a second time from the contract as it stood at the
// change set's resolved base, and anything the diff weakened is restored for the
// duration of this review. Adopting a weaker contract stays possible; approving
// it under the weaker contract does not.
//
// Those two directories are the contract: the policy and the profiles carry their
// own schema versions, so there is no separate schema path to guard. Rule bodies
// under .pose/rules are deliberately outside this gate — a criterion names a rule,
// and it is the criterion contract that is compared here.
//
// The restoration is not a second policy engine: it loads the same files through
// the same parsers and composes them through the same selector and composer, at
// a different revision. What it adds is direction — obligations may be restored
// upward, never downward.
const (
	reviewGovernancePolicyPrefix  = ".pose/policy/"
	reviewGovernanceProfilePrefix = ".pose/review-profiles/"

	// A policy or profile is a small JSON document. The bound exists so a
	// crafted blob at the base revision cannot be read without limit.
	reviewBaselineMaxBytes = 1 << 20
)

// ReviewPlanPolicyBaseline reports whether this plan was resolved under a
// protected contract and what that protection changed.
type ReviewPlanPolicyBaseline struct {
	Protected bool `json:"protected"`
	// Reason is filled when the scope changes the contract but protection could
	// not be established. It is stated rather than silently skipped: an
	// unprotected governance review is the failure this field exists to expose.
	Reason   string   `json:"reason,omitempty"`
	Revision string   `json:"revision,omitempty"`
	Paths    []string `json:"paths,omitempty"`
	// Weakened lists contract changes observed in the diff. Restored lists the
	// obligations this plan put back for the review of that diff.
	Weakened []string `json:"weakened,omitempty"`
	Restored []string `json:"restored,omitempty"`
}

func reviewGovernanceContractPath(path string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
	return strings.HasPrefix(clean, reviewGovernancePolicyPrefix) ||
		strings.HasPrefix(clean, reviewGovernanceProfilePrefix)
}

// reviewGovernanceGuard is what the plan knows before it trusts the working
// tree's policy: which contract paths this scope changes, and the revision whose
// contract governed before it did.
type reviewGovernanceGuard struct {
	Paths    []string
	Revision string
	Reason   string
	// Unreadable records that the working tree's own contract could not be read
	// for this scope, so the protected one governed in its place.
	Unreadable string
}

func (s Store) resolveReviewGovernanceGuard(scope ScopeRef) reviewGovernanceGuard {
	guard := reviewGovernanceGuard{}
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return guard
	}
	wanted := map[string]bool{}
	for _, spec := range specs {
		wanted[spec.Slug] = true
		claims, found, claimErr := ParseArtifactClaims(spec, ArtifactPolicy{})
		if claimErr != nil || !found {
			continue
		}
		for _, claim := range claims {
			for _, path := range []string{claim.Path, claim.OldPath, claim.NewPath} {
				if path != "" && reviewGovernanceContractPath(path) {
					guard.Paths = append(guard.Paths, filepath.ToSlash(filepath.Clean(path)))
				}
			}
		}
	}
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		if len(guard.Paths) > 0 {
			guard.Reason = "delivery-integrity index is unavailable, so no base revision is attributable; run `pose index`"
		}
		return guard
	}
	sets := []ChangeSet{}
	for _, set := range graph.ChangeSets {
		if wanted[set.Spec] {
			sets = append(sets, set)
		}
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].ID < sets[j].ID })
	for _, set := range sets {
		for _, observed := range set.Paths {
			for _, path := range []string{observed.Path, observed.OldPath, observed.NewPath} {
				if path != "" && reviewGovernanceContractPath(path) {
					guard.Paths = append(guard.Paths, filepath.ToSlash(filepath.Clean(path)))
				}
			}
		}
		// Mirror the review subject: the first attributed set, ordered by id,
		// supplies the base. A scope with several sets is reviewed against the
		// contract that governed before any of them landed.
		if guard.Revision == "" {
			guard.Revision = set.ResolvedBase
		}
	}
	guard.Paths = uniqueSorted(guard.Paths)
	if len(guard.Paths) > 0 && guard.Revision == "" {
		guard.Reason = "no attributed change set resolves a base revision for the contract change"
	}
	return guard
}

// reviewPolicyBaseline is the contract as it stood at a revision.
type reviewPolicyBaseline struct {
	Revision string
	Policy   ReviewPolicy
	Base     ReviewProfile
	Overlays []ReviewProfile
	Resolved bool
}

func (s Store) parseReviewPolicyAt(revision string) (ReviewPolicy, error) {
	raw, err := gitShowBounded(s.Root, revision, ".pose/policy/review.json", reviewBaselineMaxBytes)
	if err != nil {
		return ReviewPolicy{}, fmt.Errorf("pose: reading review policy at %s: %w", revision, err)
	}
	return s.parseReviewPolicy(raw)
}

func (s Store) parseReviewProfileAt(revision, ref string) (ReviewProfile, error) {
	parts := strings.Split(ref, "@")
	if len(parts) != 2 || ValidateSlug(parts[0]) != nil {
		return ReviewProfile{}, fmt.Errorf("pose: invalid review profile ref %q", ref)
	}
	raw, err := gitShowBounded(s.Root, revision, ".pose/review-profiles/"+parts[0]+".json", reviewBaselineMaxBytes)
	if err != nil {
		return ReviewProfile{}, fmt.Errorf("pose: reading review profile %q at %s: %w", ref, revision, err)
	}
	return s.parseReviewProfile(ref, raw)
}

// resolveReviewPolicyBaseline reads the whole contract at one revision. A
// baseline that cannot be read is reported, never assumed permissive.
func (s Store) resolveReviewPolicyBaseline(revision string, scopeKind string) (reviewPolicyBaseline, error) {
	baseline := reviewPolicyBaseline{Revision: revision}
	policy, err := s.parseReviewPolicyAt(revision)
	if err != nil {
		return baseline, err
	}
	baseline.Policy = policy
	if !policy.Enabled {
		// Nothing to protect: review was not governed at the base revision, so
		// the diff is not weakening a contract that existed.
		baseline.Resolved = true
		return baseline, nil
	}
	baseRef := policy.Profiles[scopeKind]
	if baseRef == "" {
		return baseline, fmt.Errorf("pose: baseline policy at %s configures no review profile for %s", revision, scopeKind)
	}
	base, err := s.parseReviewProfileAt(revision, baseRef)
	if err != nil {
		return baseline, err
	}
	if base.Scope != scopeKind {
		return baseline, fmt.Errorf("pose: baseline profile %s at %s cannot review %s scopes", base.Ref(), revision, scopeKind)
	}
	baseline.Base = base
	if policy.SchemaVersion >= ReviewPolicySchemaVersion && policy.ComponentAware {
		for _, ref := range uniqueSorted(policy.OverlayProfiles) {
			overlay, overlayErr := s.parseReviewProfileAt(revision, ref)
			if overlayErr != nil {
				return baseline, overlayErr
			}
			if overlay.SchemaVersion < ReviewPolicySchemaVersion || !hasReviewSelectors(overlay.Selectors) || overlay.Scope != scopeKind {
				continue
			}
			baseline.Overlays = append(baseline.Overlays, overlay)
		}
	}
	baseline.Resolved = true
	return baseline, nil
}

// applyProtectedPolicyBaseline restores what the reviewed diff weakened.
//
// Restoration is one-directional by construction: independence takes the
// stricter of the two, a baseline criterion the diff dropped is added back, and
// a criterion the diff kept but softened — required to optional, judged to
// collected — is returned to the baseline reading. Nothing here can make an
// obligation lighter than the working tree already made it, so a baseline that
// happens to be weaker than the diff changes nothing.
func applyProtectedPolicyBaseline(plan *ReviewPlan, baseline reviewPolicyBaseline, policy ReviewPolicy, scope ScopeRef, context reviewPlanContext) {
	report := &plan.PolicyBaseline
	report.Revision = baseline.Revision
	// Anything the caller already recorded — an unreadable tree contract, for
	// instance — is a weakening too, and survives this pass.
	prior := append([]string{}, report.Weakened...)
	defer func() { report.Weakened = uniqueSorted(append(report.Weakened, prior...)) }()
	if !baseline.Policy.Enabled {
		report.Protected = true
		report.Reason = "review was not governed at the base revision"
		return
	}
	report.Protected = true

	profiles := []ReviewProfile{baseline.Base}
	for _, item := range selectReviewOverlays(baseline.Overlays, context) {
		profiles = append(profiles, item.profile)
	}
	baselineCriteria, _ := composeReviewCriteria(profiles, nil)

	floor := normalizeReviewIndependence(baseline.Policy.ReviewerIndependence[scope.Kind])
	for _, profile := range profiles {
		floor = stricterReviewIndependence(floor, profile.Independence)
	}
	if restored := stricterReviewIndependence(plan.Independence, floor); restored != plan.Independence {
		report.Weakened = append(report.Weakened, "reviewer_independence:"+plan.Independence)
		report.Restored = append(report.Restored, "independence:"+restored)
		plan.Independence = restored
	}

	effective := map[string]int{}
	for i, criterion := range plan.Criteria {
		effective[criterion.ID] = i
	}
	for _, criterion := range baselineCriteria {
		index, present := effective[criterion.ID]
		if !present {
			restored := criterion
			restored.Profiles = []string{"protected-baseline:" + criterion.Profiles[0]}
			plan.Criteria = append(plan.Criteria, restored)
			report.Weakened = append(report.Weakened, "removed_criterion:"+criterion.ID)
			report.Restored = append(report.Restored, "criterion:"+criterion.ID)
			continue
		}
		if criterion.Required && !plan.Criteria[index].Required {
			plan.Criteria[index].Required = true
			report.Weakened = append(report.Weakened, "optional_criterion:"+criterion.ID)
			report.Restored = append(report.Restored, "criterion:"+criterion.ID+":required")
		}
		if ReviewCriterionKind(criterion) == ReviewCriterionKindJudgment && ReviewCriterionKind(plan.Criteria[index]) != ReviewCriterionKindJudgment {
			plan.Criteria[index].Kind = ReviewCriterionKindJudgment
			report.Weakened = append(report.Weakened, "collected_criterion:"+criterion.ID)
			report.Restored = append(report.Restored, "criterion:"+criterion.ID+":judgment")
		}
	}
	sort.Slice(plan.Criteria, func(i, j int) bool { return plan.Criteria[i].ID < plan.Criteria[j].ID })

	// Policy flags that change what an approval means are reported even when
	// this plan has no obligation to restore for them. The dispositions they
	// govern are enforced by the closeout and attestation gates, which read the
	// policy directly; stating the change here is what makes it reviewable.
	if !baseline.Policy.AllowApprovedWithReservations && policy.AllowApprovedWithReservations {
		report.Weakened = append(report.Weakened, "allow_approved_with_reservations:true")
	}
	if baseline.Policy.RequireSignedAttestations && !policy.RequireSignedAttestations {
		report.Weakened = append(report.Weakened, "require_signed_attestations:false")
	}
	for kind, assurance := range baseline.Policy.IdentityAssurance {
		if assurance == ReviewIdentityAssuranceVerified && policy.IdentityAssurance[kind] != ReviewIdentityAssuranceVerified {
			report.Weakened = append(report.Weakened, "identity_assurance:"+kind+":"+firstNonempty(policy.IdentityAssurance[kind], "absent"))
		}
	}
	if baseline.Policy.ComponentAware && !policy.ComponentAware {
		report.Weakened = append(report.Weakened, "component_aware:false")
	}
	report.Weakened = uniqueSorted(report.Weakened)
	report.Restored = uniqueSorted(report.Restored)
}

// reviewPolicyBaselineExplain renders the protection into the plan's explain
// trail, which is digested. That is deliberate: restoring an obligation changes
// what the review owes, so it must change the plan's identity, while the
// structured report above stays a projection of these lines.
func reviewPolicyBaselineExplain(report ReviewPlanPolicyBaseline) []string {
	if len(report.Paths) == 0 {
		return nil
	}
	explain := []string{}
	if !report.Protected {
		return append(explain, "protected policy baseline unavailable for contract change "+strings.Join(report.Paths, ",")+": "+report.Reason)
	}
	line := "protected policy baseline " + report.Revision + " governs contract change " + strings.Join(report.Paths, ",")
	if report.Reason != "" {
		line += " (" + report.Reason + ")"
	}
	explain = append(explain, line)
	for _, item := range report.Weakened {
		explain = append(explain, "protected policy baseline observed weakening "+item)
	}
	for _, item := range report.Restored {
		explain = append(explain, "protected policy baseline restored "+item)
	}
	return explain
}

// reviewPolicyBaselineBand summarizes the protection as one band entry. A
// restoration means the diff actually lowered the contract governing its own
// review, which is the critical case; a contract change that lowered nothing is
// still elevated, because the trust root moved.
func reviewPolicyBaselineBand(report ReviewPlanPolicyBaseline) (ReviewPlanBand, bool) {
	if len(report.Paths) == 0 {
		return ReviewPlanBand{}, false
	}
	band := ReviewPlanBand{
		Band: ReviewBandElevated, Trigger: "governance_contract=" + strings.Join(report.Paths, ","),
		Basis: reviewBasisObserved, Source: "change-set base:" + report.Revision,
		Policy: "protected-baseline", Obligations: report.Restored,
	}
	if len(report.Restored) > 0 {
		band.Band = ReviewBandCritical
	}
	if !report.Protected {
		band.Band = ReviewBandUnknown
		band.Basis = reviewBasisUnknown
		band.Source = "unavailable: " + report.Reason
		band.Obligations = []string{}
	}
	return band, true
}

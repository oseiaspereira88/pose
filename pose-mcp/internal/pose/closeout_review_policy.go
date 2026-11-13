package pose

// Closeout review policy by cost and benefit (spec
// pose-closeout-review-policy-by-cost-benefit, ADR
// closeout-review-policy-by-cost-and-benefit). An instance that adopts review
// policy schema 5 decides, per aggregate scope kind, whether the close of a
// milestone or a roadmap owes its own independent review. Closing stays
// explicit in every mode; what changes is only the review of the aggregate.
// One evaluator answers for every reader, so plan, verification, state,
// doctor and close cannot disagree about the threshold.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	CloseoutReviewNever         = "never"
	CloseoutReviewWhenJustified = "when-justified"
	CloseoutReviewAlways        = "always"

	CloseoutReviewEvaluatorVersion = "closeout-review@1"

	MilestoneReviewRequiredKey = "milestone_review_required"
	RoadmapReviewRequiredKey   = "roadmap_review_required"

	// The when-justified thresholds the maintainer chose: a milestone owes a
	// review with more than two distinct specs; a roadmap with more than one
	// distinct spec and at least one ADR association.
	closeoutMilestoneMemberThreshold = 2
	closeoutRoadmapMemberThreshold   = 1
)

func validCloseoutReviewValue(value string) bool {
	switch value {
	case CloseoutReviewNever, CloseoutReviewWhenJustified, CloseoutReviewAlways:
		return true
	}
	return false
}

// validateCloseoutReviewPolicyKeys refuses the pair outside schema 5, and
// under schema 5 refuses a missing, null, mistyped, duplicated or unknown value
// before anything plans or writes against it.
func validateCloseoutReviewPolicyKeys(raw []byte, fields map[string]json.RawMessage, schema int) error {
	_, hasMilestone := fields[MilestoneReviewRequiredKey]
	_, hasRoadmap := fields[RoadmapReviewRequiredKey]
	if schema != CloseoutReviewPolicySchemaVersion {
		if hasMilestone || hasRoadmap {
			return fmt.Errorf("pose: %s and %s require review policy schema %d", MilestoneReviewRequiredKey, RoadmapReviewRequiredKey, CloseoutReviewPolicySchemaVersion)
		}
		return nil
	}
	if err := rejectDuplicateJSONKeys(raw, false, true); err != nil {
		return fmt.Errorf("pose: invalid schema-%d review policy: %w", CloseoutReviewPolicySchemaVersion, err)
	}
	if !hasMilestone || !hasRoadmap {
		return fmt.Errorf("pose: review policy schema %d requires both %s and %s", CloseoutReviewPolicySchemaVersion, MilestoneReviewRequiredKey, RoadmapReviewRequiredKey)
	}
	for _, key := range []string{MilestoneReviewRequiredKey, RoadmapReviewRequiredKey} {
		var value string
		if string(fields[key]) == "null" || json.Unmarshal(fields[key], &value) != nil || !validCloseoutReviewValue(value) {
			return fmt.Errorf("pose: %s must be one of never, when-justified, always (got %s)", key, string(fields[key]))
		}
	}
	return nil
}

// CloseoutReviewAdopted reports a policy that decides aggregate reviews by the
// closeout pair. Every other policy keeps the legacy rule unchanged.
func (p ReviewPolicy) CloseoutReviewAdopted() bool {
	return p.SchemaVersion == CloseoutReviewPolicySchemaVersion
}

// CloseoutReviewSetting names the policy key and value that govern a scope kind.
func (p ReviewPolicy) CloseoutReviewSetting(kind string) (string, string) {
	switch kind {
	case "milestone":
		return MilestoneReviewRequiredKey, p.MilestoneReviewRequired
	case "roadmap":
		return RoadmapReviewRequiredKey, p.RoadmapReviewRequired
	}
	return "", ""
}

// CloseoutReviewDecision is the evaluator's answer for one aggregate scope.
// Projections render it; none of them recomputes a threshold.
type CloseoutReviewDecision struct {
	Scope            string `json:"scope"`
	Adopted          bool   `json:"adopted"`
	EvaluatorVersion string `json:"evaluator_version,omitempty"`
	PolicyKey        string `json:"policy_key,omitempty"`
	PolicyValue      string `json:"policy_value,omitempty"`
	// Required is true when the scope owes its own independent review. An
	// unknown input never yields false.
	Required bool `json:"required"`
	// Known is false when a member or an association could not be resolved:
	// the decision then requires the review and says what is missing.
	Known       bool     `json:"known"`
	Reason      string   `json:"reason,omitempty"`
	Members     []string `json:"members,omitempty"`
	ADRs        []string `json:"adr_associations,omitempty"`
	Diagnostics []string `json:"diagnostics,omitempty"`
}

// Review is the literal the closeout receipt records: required or not-required.
func (d CloseoutReviewDecision) Review() string {
	if d.Required {
		return "required"
	}
	return "not-required"
}

// EvaluateCloseoutReview answers whether an aggregate scope owes its own
// review under the adopted policy. A legacy policy returns Adopted=false and
// the caller keeps the legacy rule.
func (s Store) EvaluateCloseoutReview(ref string) (CloseoutReviewDecision, error) {
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return CloseoutReviewDecision{}, err
	}
	return s.evaluateCloseoutReview(ref, policy)
}

func (s Store) evaluateCloseoutReview(ref string, policy ReviewPolicy) (CloseoutReviewDecision, error) {
	decision := CloseoutReviewDecision{Scope: ref, Known: true}
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return decision, err
	}
	if !policy.CloseoutReviewAdopted() || (scope.Kind != "milestone" && scope.Kind != "roadmap") {
		return decision, nil
	}
	decision.Adopted = true
	decision.EvaluatorVersion = CloseoutReviewEvaluatorVersion
	decision.PolicyKey, decision.PolicyValue = policy.CloseoutReviewSetting(scope.Kind)
	slug := scope.Roadmap
	if scope.Kind == "roadmap" {
		slug = scope.Slug
	}
	rm, err := s.GetRoadmap(slug)
	if err != nil {
		return decision, err
	}
	project, projectErr := s.WriteProjectID()
	var raws []string
	if scope.Kind == "milestone" {
		found := false
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				raws, found = milestone.Specs, true
			}
		}
		if !found {
			return decision, fmt.Errorf("pose: milestone %s not found in roadmap %s", scope.Milestone, scope.Roadmap)
		}
	} else {
		for _, milestone := range rm.Milestones {
			raws = append(raws, milestone.Specs...)
		}
	}
	if projectErr != nil {
		decision.Diagnostics = append(decision.Diagnostics, "the project identity that qualifies local members is unknown: "+projectErr.Error())
	} else {
		decision.Members, decision.Diagnostics = s.closeoutCanonicalMembers(raws, project)
	}
	if len(decision.Diagnostics) > 0 {
		decision.Known = false
	}
	members := len(decision.Members)
	switch decision.PolicyValue {
	case CloseoutReviewNever:
		decision.Required = false
		decision.Reason = fmt.Sprintf("members=%d, never required", members)
		// never decides without counting, but a member it cannot resolve is
		// still a scope it cannot close.
		return decision, nil
	case CloseoutReviewAlways:
		decision.Required = true
		decision.Reason = fmt.Sprintf("members=%d, always required", members)
		return decision, nil
	}
	if !decision.Known {
		decision.Required = true
		decision.Reason = "members unknown; an exemption needs every member resolved"
		return decision, nil
	}
	if scope.Kind == "milestone" {
		decision.Required = members > closeoutMilestoneMemberThreshold
		decision.Reason = fmt.Sprintf("members=%d, threshold>%d", members, closeoutMilestoneMemberThreshold)
		return decision, nil
	}
	// The associations are resolved even below the member threshold: an
	// association input that cannot be read never proves an exemption.
	adrs, diagnostics := s.closeoutADRAssociations(rm, project)
	decision.ADRs = adrs
	if len(diagnostics) > 0 {
		decision.Diagnostics = append(decision.Diagnostics, diagnostics...)
		decision.Known = false
		decision.Required = true
		decision.Reason = fmt.Sprintf("members=%d, threshold>%d, adr associations unknown", members, closeoutRoadmapMemberThreshold)
		return decision, nil
	}
	decision.Required = members > closeoutRoadmapMemberThreshold && len(adrs) > 0
	decision.Reason = fmt.Sprintf("members=%d, threshold>%d, adrs=%d", members, closeoutRoadmapMemberThreshold, len(adrs))
	return decision, nil
}

// closeoutCanonicalMembers resolves milestone members to their final
// identities, xref:<project>/spec:<slug>, sorted and deduplicated. A local
// member follows only a verified transfer redirect; an external one is
// resolved by the federated resolver or stays unknown. Unknown members are
// diagnostics, never a count of zero.
func (s Store) closeoutCanonicalMembers(raws []string, project string) ([]string, []string) {
	seen := map[string]bool{}
	members, diagnostics := []string{}, []string{}
	for _, raw := range raws {
		identity, diagnostic := s.closeoutCanonicalMember(strings.TrimSpace(raw), project)
		if diagnostic != "" {
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		if !seen[identity] {
			seen[identity] = true
			members = append(members, identity)
		}
	}
	sort.Strings(members)
	return members, diagnostics
}

func (s Store) closeoutCanonicalMember(raw, project string) (string, string) {
	ref, err := ParseArtifactRef(raw)
	if err != nil || ref.Kind != "spec" || ref.Milestone != "" {
		return "", fmt.Sprintf("member %q is not a spec reference", raw)
	}
	if ref.Project == "" {
		ref.Project = project
	}
	resolve := func(why string) (string, string) {
		if s.FederatedResolver == nil {
			return "", fmt.Sprintf("member %s %s and no federated resolver is bound to follow it", ref.String(), why)
		}
		resolved := s.FederatedResolver.Resolve(project, ref.String())
		if !resolved.Resolved {
			return "", fmt.Sprintf("member %s did not resolve: %s", ref.String(), resolved.State)
		}
		if resolved.CanonicalIdentity != nil {
			return resolved.CanonicalIdentity.String(), ""
		}
		return resolved.Identity.String(), ""
	}
	if ref.Project != project {
		return resolve("belongs to another project")
	}
	_, found, err := readSpecTransferRedirect(s.fsys(), s, ref)
	if err != nil {
		return "", fmt.Sprintf("member %s has an unverifiable redirect: %v", ref.String(), err)
	}
	if found {
		// The resolver follows the whole chain, with its cycle and depth
		// limits and the destination's authorization: the immediate
		// destination may itself have moved.
		return resolve("was transferred")
	}
	if _, err := s.GetSpec(ref.Slug); err != nil {
		return "", fmt.Sprintf("member %s names no spec of this project", ref.String())
	}
	return ref.String(), ""
}

// CurrentCloseoutDecision is the decision for an open aggregate scope now:
// the evaluator, then every protection that can only make it stronger. It is
// what close judges before publishing and again inside the protected region.
func (s Store) CurrentCloseoutDecision(ref string) (CloseoutReviewDecision, error) {
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return CloseoutReviewDecision{}, err
	}
	return s.currentCloseoutDecision(ref, policy)
}

func (s Store) currentCloseoutDecision(ref string, policy ReviewPolicy) (CloseoutReviewDecision, error) {
	decision, err := s.evaluateCloseoutReview(ref, policy)
	if err != nil || !decision.Adopted || decision.Required {
		return decision, err
	}
	scope, _ := ParseScopeRef(ref)
	s.protectCloseoutDecision(scope, &decision)
	if !decision.Required && s.mandatoryHumanReview(scope.Kind, policy) {
		decision.Required = true
		decision.Reason += "; a mandatory-human review is never dispensed"
	}
	return decision, nil
}

// mandatoryHumanReview reports a scope kind whose policy or base profile asks
// for a human reviewer. No setting dispenses that review. A profile that
// cannot be read is treated as asking for it.
func (s Store) mandatoryHumanReview(kind string, policy ReviewPolicy) bool {
	if policy.ReviewerIndependence[kind] == "mandatory-human" {
		return true
	}
	refs := append([]string{}, policy.OverlayProfiles...)
	if ref := policy.Profiles[kind]; ref != "" {
		refs = append(refs, ref)
	}
	// An adopted overlay for this kind that can ask for a human counts
	// whether or not its selector matches now: an exemption must not depend
	// on a selection the review plan makes later.
	for _, ref := range refs {
		profile, _, err := s.loadReviewProfile(ref)
		if err != nil {
			return true
		}
		if (profile.Scope == kind || profile.Scope == "" || ref == policy.Profiles[kind]) && profile.Independence == "mandatory-human" {
			return true
		}
	}
	return false
}

// closeoutReviewGate is the one place the readers of an approval ask whether
// an aggregate scope owes its own review under an adopted policy. It returns
// the decision that governs the scope and whether its own review is dispensed.
//
// Order (spec pose-closeout-review-policy-by-cost-benefit, "One evaluator and
// precedence"): a receipt already published decides by what it recorded, as
// long as the scope it closed is still the scope; a roadmap closed before
// adoption keeps its sealed contract (nil decision); an open scope is decided
// now. A review recorded on the scope, required or not, is never bypassed: it
// is verified as before and its veto stands. An unknown input never exempts.
func (s Store) closeoutReviewGate(ref string, policy ReviewPolicy) (*CloseoutReviewDecision, bool, error) {
	if !policy.CloseoutReviewAdopted() {
		return nil, false, nil
	}
	scope, err := ParseScopeRef(ref)
	if err != nil || (scope.Kind != "milestone" && scope.Kind != "roadmap") {
		return nil, false, err
	}
	receipt, err := s.ReadCloseoutReceipt(ref)
	if err != nil {
		return nil, false, err
	}
	if receipt != nil {
		if err := s.closeoutReceiptStands(*receipt, policy); err != nil {
			return nil, false, err
		}
		decision := receipt.Decision
		return &decision, !decision.Required && !s.scopeReviewRecorded(ref), nil
	}
	if s.closeoutScopeHistorical(scope) {
		return nil, false, nil
	}
	decision, err := s.currentCloseoutDecision(ref, policy)
	if err != nil {
		return nil, false, err
	}
	return &decision, !decision.Required && decision.Known && !s.scopeReviewRecorded(ref), nil
}

// closeoutReceiptStands checks a receipt against the scope it closed: the
// same canonical members. A receipt is never re-decided under a later policy,
// but it does not carry over to a scope whose composition changed.
func (s Store) closeoutReceiptStands(receipt CloseoutReceipt, policy ReviewPolicy) error {
	// The closed contract, read without the current policy: a later policy
	// never re-decides a receipt, but a scope whose composition or exit gate
	// changed is no longer the scope it closed.
	members, gate, diagnostics, err := s.closeoutSubject(receipt.Scope)
	if err != nil {
		return fmt.Errorf("pose: closeout receipt of %s cannot be checked against its scope: %w", receipt.Scope, err)
	}
	if len(diagnostics) > 0 {
		return fmt.Errorf("pose: closeout receipt of %s cannot be checked against its scope: %s", receipt.Scope, strings.Join(diagnostics, "; "))
	}
	drift := func(what string) error {
		return fmt.Errorf("pose: closeout receipt of %s no longer matches its scope: its %s changed since the close; a receipt is never rewritten, so restore it or record the change in a new scope", receipt.Scope, what)
	}
	if strings.Join(members, ",") != strings.Join(receipt.Decision.Members, ",") {
		return drift(fmt.Sprintf("members (closed with %v, now %v)", receipt.Decision.Members, members))
	}
	if receipt.ExitGateDigest != gate {
		return drift("exit gate")
	}
	return nil
}

// protectCloseoutDecision restores the review when the scope itself changes
// the review contract: an exemption the diff introduced cannot exempt the diff.
// The decision is made a second time under the policy at the change set's base
// revision, and a stronger answer there wins. Protection that cannot be
// established keeps the review required.
func (s Store) protectCloseoutDecision(scope ScopeRef, decision *CloseoutReviewDecision) {
	guard := s.resolveReviewGovernanceGuard(scope)
	if len(guard.Paths) == 0 {
		return
	}
	restore := func(reason string) {
		decision.Required = true
		decision.Reason += "; " + reason
	}
	if guard.Revision == "" {
		restore("the scope changes the review contract and no protected baseline resolves: " + guard.Reason)
		return
	}
	base, err := s.parseReviewPolicyAt(guard.Revision)
	if err != nil {
		restore("the protected baseline policy cannot be read: " + err.Error())
		return
	}
	if !base.Enabled {
		return
	}
	if !base.CloseoutReviewAdopted() {
		restore("the protected baseline at " + guard.Revision + " keeps the legacy rule, under which this scope owes its review")
		return
	}
	protected, err := s.evaluateCloseoutReview(decision.Scope, base)
	if err != nil || protected.Required {
		restore("the protected baseline at " + guard.Revision + " requires the review (" + protected.PolicyKey + "=" + protected.PolicyValue + ")")
	}
}

// closeoutSubject is what a closeout receipt binds: the canonical members of
// the scope and, for a milestone, the digest of its exit gate (a roadmap's is
// the gates of its milestones, which their own receipts bind).
func (s Store) closeoutSubject(ref string) ([]string, string, []string, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return nil, "", nil, err
	}
	slug := scope.Roadmap
	if scope.Kind == "roadmap" {
		slug = scope.Slug
	}
	rm, err := s.GetRoadmap(slug)
	if err != nil {
		return nil, "", nil, err
	}
	project, err := s.WriteProjectID()
	if err != nil {
		return nil, "", []string{"the project identity is unknown: " + err.Error()}, nil
	}
	var raws []string
	gate := ""
	for _, milestone := range rm.Milestones {
		if scope.Kind == "milestone" && milestone.ID != scope.Milestone {
			continue
		}
		raws = append(raws, milestone.Specs...)
		if scope.Kind == "milestone" {
			gate = digestText(milestone.ExitGate)
		}
	}
	members, diagnostics := s.closeoutCanonicalMembers(raws, project)
	return members, gate, diagnostics, nil
}

// closeoutScopeHistorical reports a scope whose roadmap was closed before
// adoption: done, with no receipt and no close of this engine in progress,
// and with positive evidence of its legacy closure: an approval recorded on
// the roadmap, or the legacy rule exempting it as done before review
// adoption. Status alone proves nothing.
// Its milestones and itself are verified by the contract they were sealed
// under, never re-evaluated, and no receipt is backfilled for them. A close
// interrupted between its status and its receipt left an intent, so it stays
// incomplete instead of reading as history.
func (s Store) closeoutScopeHistorical(scope ScopeRef) bool {
	slug := scope.Slug
	if scope.Kind == "milestone" {
		slug = scope.Roadmap
	}
	rm, err := s.GetRoadmap(slug)
	if err != nil || rm.Status != "done" {
		return false
	}
	ref := "roadmap:" + slug
	if receipt, err := s.ReadCloseoutReceipt(ref); err != nil || receipt != nil {
		return false
	}
	pending, err := s.closeoutIntentPending(ref)
	if err != nil || pending {
		return false
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return false
	}
	if required, err := s.reviewRequiredForScope(ScopeRef{Kind: "roadmap", Slug: slug}, policy); err == nil && !required {
		return true
	}
	recorded, refused, err := s.milestonesWithRecordedApproval([]string{ref})
	return err == nil && recorded[ref] && !refused[ref]
}

// closedByReceipt reports an aggregate scope its receipt still closes: the
// receipt stands for the scope as it is, and no refusal was recorded on the
// scope after it. It is what reached, attention and doctor ask, instead of
// the mere presence of a receipt.
func (s Store) closedByReceipt(ref string) (bool, error) {
	receipt, err := s.ReadCloseoutReceipt(ref)
	if err != nil || receipt == nil {
		return false, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return false, err
	}
	if s.closeoutReceiptStands(*receipt, policy) != nil {
		return false, nil
	}
	return !s.milestoneReviewRefused(ref), nil
}

// scopeReviewRecorded reports a review of the scope on file: a bundle or a
// legacy attempt. A record that cannot be read counts as recorded, so an
// unreadable rejection never turns into an exemption.
func (s Store) scopeReviewRecorded(ref string) bool {
	bundles, err := s.ReadReviewBundles(ref)
	if err != nil || len(bundles.Bundles) > 0 || !bundles.Coverage.CompleteFor(ref) {
		return true
	}
	attempts, err := s.ReadReviewAttempts(ref)
	return err != nil || len(attempts.Attempts) > 0 || !attempts.Coverage.CompleteFor(ref)
}

// closeoutApprovalOrigin is the origin an exempt scope's approval reports.
func closeoutApprovalOrigin(decision CloseoutReviewDecision) string {
	return "not-required:" + decision.PolicyKey + "=" + decision.PolicyValue
}

func closeoutExemptionWarning(ref string, decision CloseoutReviewDecision) string {
	return "review: not-required for " + ref + " (" + decision.PolicyKey + "=" + decision.PolicyValue + ", " + decision.Reason + "); `pose close " + ref + "` is still required"
}

// CloseoutReviewOwing is an open aggregate scope that owes its own review under
// the adopted policy and holds no recorded approval yet, for `pose doctor`.
type CloseoutReviewOwing struct {
	Scope    string                 `json:"scope"`
	Decision CloseoutReviewDecision `json:"decision"`
	// Previous is the rule that applied before, when a recorded input shows
	// it; with no snapshot it says the comparison is unknown.
	Previous string `json:"previous"`
	Action   string `json:"action"`
}

// CloseoutReviewsOwed lists the open milestones and roadmaps that, under the
// adopted closeout review policy, owe a review they do not have. A legacy
// policy returns nothing. Close remains the authoritative gate.
func (s Store) CloseoutReviewsOwed() ([]CloseoutReviewOwing, error) {
	policy, _, err := s.loadReviewPolicy()
	if err != nil || !policy.CloseoutReviewAdopted() {
		return nil, err
	}
	roadmaps, err := s.ListRoadmaps()
	if err != nil {
		return nil, err
	}
	refs := []string{}
	for _, rm := range roadmaps {
		if s.closeoutScopeHistorical(ScopeRef{Kind: "roadmap", Slug: rm.Slug}) {
			continue
		}
		for _, milestone := range rm.Milestones {
			refs = append(refs, "milestone:"+rm.Slug+"/"+milestone.ID)
		}
		refs = append(refs, "roadmap:"+rm.Slug)
	}
	owing := []CloseoutReviewOwing{}
	for _, ref := range refs {
		closed, err := s.closedByReceipt(ref)
		if err != nil {
			return nil, err
		}
		if closed {
			continue
		}
		decision, err := s.currentCloseoutDecision(ref, policy)
		if err != nil {
			return nil, err
		}
		if !decision.Required {
			continue
		}
		// A recorded approval counts only while it satisfies the current
		// obligation: fresh, for the scope as it is now.
		if eval, err := s.ReviewCheck(ref); err == nil && eval.Approved && eval.Fresh {
			continue
		}
		action := "seal and review " + ref + " (`pose review bundle " + ref + " --seal`), then `pose close " + ref + "`"
		if !decision.Known {
			action = "resolve " + strings.Join(decision.Diagnostics, "; ")
		}
		owing = append(owing, CloseoutReviewOwing{Scope: ref, Decision: decision, Previous: "unknown: no snapshot of the earlier policy or composition is recorded", Action: action})
	}
	return owing, nil
}

func mustScope(ref string) ScopeRef {
	scope, _ := ParseScopeRef(ref)
	return scope
}

// CloseoutRecoveryDecision decides a roadmap left done without its receipt,
// under the same rules as an open scope: the protected decision, and an
// exemption only when every input is known and no review is on file.
func (s Store) CloseoutRecoveryDecision(ref string) (CloseoutReviewDecision, bool, error) {
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return CloseoutReviewDecision{}, false, err
	}
	decision, err := s.currentCloseoutDecision(ref, policy)
	if err != nil {
		return decision, false, err
	}
	return decision, !decision.Required && decision.Known && !s.scopeReviewRecorded(ref), nil
}

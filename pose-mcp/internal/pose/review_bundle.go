package pose

// Review bundles provide the fixed review subject described by
// ADR 2026-08-13-sealed-review-bundles-and-attestations. A bundle hashes only
// governed semantic inputs. Its attestation is a separate immutable artifact,
// so recording approval and applying closeout cannot invalidate that approval.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ReviewBundleSchemaVersion = 1
const maxReviewBundleBytes = 4 << 20

type ReviewBundleInput struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Digest string `json:"digest,omitempty"`
	Reason string `json:"reason"`
}

type ReviewBundleScope struct {
	Ref        string              `json:"ref"`
	Kind       string              `json:"kind"`
	Slug       string              `json:"slug,omitempty"`
	Roadmap    string              `json:"roadmap,omitempty"`
	Milestone  string              `json:"milestone,omitempty"`
	DependsOn  []string            `json:"depends_on,omitempty"`
	Remediates []string            `json:"remediates,omitempty"`
	Supersedes string              `json:"supersedes,omitempty"`
	Components []string            `json:"components,omitempty"`
	Deliveries []string            `json:"deliveries,omitempty"`
	Sections   []ReviewBundleInput `json:"sections"`
}

type ReviewBundleSubjectEntry struct {
	Action  string `json:"action"`
	Path    string `json:"path,omitempty"`
	OldPath string `json:"old_path,omitempty"`
	NewPath string `json:"new_path,omitempty"`
	Class   string `json:"class"`
	Digest  string `json:"digest,omitempty"`
	Reason  string `json:"reason"`
}

// ReviewBundleRangeObservation records what a change set's commit range spans,
// as distinct from what it attributes.
//
// A change set resolved from trailers takes its paths from the commits that
// carry the trailer, which is exact. Its Base and Head, however, describe a
// range, and that range spans whatever else was committed in between. Reading
// Base..Head as "the changes of this spec" is therefore wrong whenever work on
// two specs interleaved, and nothing said so.
type ReviewBundleRangeObservation struct {
	ChangeSet string `json:"change_set"`
	// State is `clean`, `contaminated` or `unknown`. Unknown is not clean: it
	// means the range could not be counted here, and saying so is the point.
	State               string `json:"state"`
	AttributedCommits   int    `json:"attributed_commits"`
	RangeCommits        int    `json:"range_commits,omitempty"`
	UnattributedCommits int    `json:"unattributed_commits,omitempty"`
	Reason              string `json:"reason,omitempty"`
}

type ReviewBundleSubject struct {
	ChangeSets []string `json:"change_sets"`
	Base       string   `json:"base,omitempty"`
	Head       string   `json:"head,omitempty"`
	// ImplementationDigest identifies the implementation content alone: the
	// attributed change sets, the range they resolve to and the ordered
	// manifest of entries. It is computed from the subject and nothing else.
	//
	// The bundle digest covers this, the plan and the evidence. Anything that
	// observes the implementation — a structural assessment, a receipt, a cache
	// key — anchors here instead, so the graph stays a DAG:
	// implementation -> assessment -> plan -> bundle -> attestation. An
	// assessment that keyed on the bundle containing it would be defining
	// itself in terms of its own output.
	ImplementationDigest string                     `json:"implementation_digest,omitempty"`
	PatchDigest          string                     `json:"patch_digest"`
	TreeDigest           string                     `json:"tree_digest"`
	Entries              []ReviewBundleSubjectEntry `json:"entries"`
}

type ReviewBundlePlan struct {
	PlanDigest   string                `json:"plan_digest"`
	Independence string                `json:"independence"`
	Components   []ReviewPlanComponent `json:"components"`
	Criteria     []ReviewPlanCriterion `json:"criteria"`
	Tools        []ReviewPlanTool      `json:"tools"`
	// SelectedProfiles records which components each profile was selected for.
	// A criterion names the profiles it came from, and without this mapping the
	// sealed subject cannot say which components that criterion answers for —
	// so evidence could only be matched by reference and class, never by where
	// it came from. Re-reading it from the current policy at verification time
	// would judge an immutable bundle by today's selection, which is the whole
	// thing sealing exists to prevent.
	SelectedProfiles []ReviewPlanProfile `json:"selected_profiles,omitempty"`
	// Structure seals the material structural facts the plan observed, so the
	// mapping obligation is fixed at seal time. Recomputing it at verification
	// would let a later detector improvement retroactively block an approved
	// review; sealing it means a changed observation changes the plan digest and
	// the review goes stale, which is the mechanism this engine already has for
	// "the thing you approved is not the thing in front of me". Everything in it
	// is content-derived, so it carries no provider ref into bundle identity.
	Structure *ReviewPlanStructure `json:"structure,omitempty"`
}

type ReviewBundleEvidence struct {
	ID            string `json:"id"`
	Module        string `json:"module,omitempty"`
	Check         string `json:"check"`
	EvidenceClass string `json:"evidence_class"`
	Outcome       string `json:"outcome"`
	// SubjectDigest is the semantic implementation identity observed by a
	// derived producer that is not itself a commit-based validation. It keeps
	// the observation current across provider-ref/derived-only movement while
	// still binding the evidence to the sealed subject content.
	SubjectDigest string `json:"subject_digest,omitempty"`
	// SubjectObservation is `observed`, `carried-forward` or `unknown`, and is
	// sealed rather than inferred later. A result that names no commit is
	// unknown, never current: an absent fingerprint is a gap in what we know,
	// and reading it as agreement is how a check that never saw this change
	// comes to stand for it.
	SubjectObservation string `json:"subject_observation,omitempty"`
	GitHead            string `json:"git_head,omitempty"`
	ProvenanceDigest   string `json:"provenance_digest,omitempty"`
	Report             string `json:"report,omitempty"`
}

// ReviewEvidenceObservation classifies one sealed result against the subject
// head it is sealed beside.
func ReviewEvidenceObservation(subjectHead string, evidence ReviewBundleEvidence) string {
	if evidence.GitHead == "" || subjectHead == "" {
		return "unknown"
	}
	if evidence.GitHead == subjectHead {
		return "observed"
	}
	return "carried-forward"
}

// ReviewEvidenceObservationForSubject prefers the semantic subject identity
// for derived evidence and retains the commit-based compatibility path for
// ordinary validation results.
func ReviewEvidenceObservationForSubject(subject ReviewBundleSubject, evidence ReviewBundleEvidence) string {
	if evidence.SubjectDigest != "" {
		if subject.ImplementationDigest == "" {
			return "unknown"
		}
		if evidence.SubjectDigest == subject.ImplementationDigest {
			return "observed"
		}
		return "carried-forward"
	}
	return ReviewEvidenceObservation(subject.Head, evidence)
}

type ReviewBundleChild struct {
	Scope        string `json:"scope"`
	BundleID     string `json:"bundle_id"`
	BundleDigest string `json:"bundle_digest"`
}

type ReviewBundlePayload struct {
	Scope    ReviewBundleScope      `json:"scope"`
	Subject  ReviewBundleSubject    `json:"subject"`
	Plan     ReviewBundlePlan       `json:"plan"`
	Evidence []ReviewBundleEvidence `json:"evidence"`
	// GoverningContracts are the governance contracts in force when this bundle
	// was sealed, so verification judges it by the rules that existed then
	// rather than by an editable date in today's policy
	// (spec pose-bundles-seal-the-contracts-that-govern-them).
	//
	// A bundle sealed before this field existed carries none, and is read by the
	// dated rule it was always read by. That is the only thing the dates are
	// for now.
	GoverningContracts []string `json:"governing_contracts,omitempty"`
	// Gates are the two closeout settings that depend on policy, frozen at seal
	// time for the same reason the contracts are: read live, a flag flipped
	// today would approve a closeout recorded years ago
	// (spec pose-bundle-findings-take-the-contract-the-legacy-path-had).
	// A pointer, not a value: `omitempty` does nothing for a struct, so a value
	// here serialised as `"gates":{}` in every payload and changed the digest of
	// every bundle already sealed. Nil means the bundle predates the field; a
	// pointer to an empty struct means it was sealed with the defaults, and the
	// two are different facts.
	Gates          *ReviewBundleGates  `json:"gates,omitempty"`
	Children       []ReviewBundleChild `json:"children,omitempty"`
	ConsumedInputs []ReviewBundleInput `json:"consumed_inputs"`
}

// ReviewBundleGates are the policy-dependent parts of the closeout contract.
// The rest of it — a finding needing a severity, an action, a disposition the
// engine knows, and an accepted risk needing an owner, a rationale and a review
// date — is not configuration and is not sealed.
type ReviewBundleGates struct {
	AllowApprovedWithReservations bool     `json:"allow_approved_with_reservations,omitempty"`
	AcceptedRiskSeverities        []string `json:"accepted_risk_severities,omitempty"`
	// AllowCriterionReuse is sealed for the same reason: reuse carries a prior
	// disposition into this attestation, and whether that was permitted is a
	// property of the review, not of the policy as it stands today
	// (spec pose-reuse-is-sealed-signing-stays-live).
	AllowCriterionReuse bool `json:"allow_criterion_reuse,omitempty"`
	// IdentityAssurance is sealed for the same reason as the rest: whether this
	// review had to prove the reviewer's identity, or could declare it, is a
	// property of the review and not of the policy as it stands today.
	IdentityAssurance string `json:"identity_assurance,omitempty"`
}

// SealedGates returns the gates sealed into the bundle, or the conservative
// defaults when it predates the field: reservations refused, no risk severity
// accepted, no criterion reuse. That is exactly what this path enforced before
// the gates existed, so no bundle already sealed changes verdict.
//
// Identity assurance is the one gate whose conservative default is the weaker
// value. A bundle sealed before the field existed was reviewed under an engine
// where a prefix was the only identity there was; reading its absence as
// `verified` would retroactively claim a proof that was never asked for, and
// fail every review already recorded.
func (p ReviewBundlePayload) SealedGates() ReviewBundleGates {
	if p.Gates == nil {
		return ReviewBundleGates{IdentityAssurance: ReviewIdentityAssuranceDeclared}
	}
	gates := *p.Gates
	if gates.IdentityAssurance == "" {
		gates.IdentityAssurance = ReviewIdentityAssuranceDeclared
	}
	return gates
}

// BundleGovernedBy reports whether the contract governed this bundle, and
// whether the bundle says so at all. An unstamped bundle answers `false, false`
// and its caller falls back to the dated reading.
func BundleGovernedBy(bundle ReviewBundle, contractID string) (governed, stamped bool) {
	if len(bundle.Payload.GoverningContracts) == 0 {
		return false, false
	}
	for _, id := range bundle.Payload.GoverningContracts {
		if id == contractID {
			return true, true
		}
	}
	return false, true
}

// governingContractsAtSeal is every contract this engine knows. A bundle sealed
// now is held to all of them; one sealed by an engine that did not know a
// contract never lists it, and is never retroactively held to it. That is the
// property the adoption date could not have, because a date can be edited after
// the fact and this cannot.
func governingContractsAtSeal() []string {
	ids := []string{}
	for _, contract := range ReviewContracts() {
		ids = append(ids, contract.ID)
	}
	sort.Strings(ids)
	return ids
}

type ReviewBundle struct {
	SchemaVersion  int                 `json:"schema_version"`
	BundleID       string              `json:"bundle_id"`
	BundleDigest   string              `json:"bundle_digest"`
	State          string              `json:"state"`
	SealedAt       string              `json:"sealed_at,omitempty"`
	Payload        ReviewBundlePayload `json:"payload"`
	ExcludedInputs []ReviewBundleInput `json:"excluded_inputs,omitempty"`
	// RangeObservations sit outside the payload, so outside the digest.
	//
	// They describe the repository around the change — how many commits the
	// attributed range spans — and that moves whenever anything else is
	// committed or reindexed. Sealing it would make an unrelated commit stale a
	// review of content that did not move, which is the invalidation rule this
	// contract exists to keep narrow. They are diagnostics, recomputed on every
	// preparation, like the warnings beside them.
	RangeObservations []ReviewBundleRangeObservation `json:"range_observations,omitempty"`
	Warnings          []string                       `json:"warnings,omitempty"`
	Blockers          []string                       `json:"blockers,omitempty"`
	Path              string                         `json:"path,omitempty"`
}

type ReviewAttestationReuse struct {
	Criterion       string `json:"criterion"`
	FromAttestation string `json:"from_attestation"`
	InputDigest     string `json:"input_digest"`
}

type ReviewAttestationSignature struct {
	Issuer    string `json:"issuer"`
	Subject   string `json:"subject"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

// ReviewAuthorityClaim is what an authorized issuer asserts about who reviewed,
// bound to the exact bundle they reviewed.
//
// The engine does not learn who anyone is from this. It verifies that an issuer
// it trusts, holding the grant for the role being claimed, signed a statement
// binding these principals and executions to this bundle — and that the
// statement is current and addressed to this project. Whether the issuer told
// the truth is the issuer's authority, not POSE's inference, and that boundary
// is the honest one: the alternative is a local engine claiming to know
// something no local evidence can show.
type ReviewAuthorityClaim struct {
	SchemaVersion int    `json:"schema_version"`
	Project       string `json:"project"`
	BundleDigest  string `json:"bundle_digest"`
	// Principal and Role describe the reviewer. Role is `agent` or `human`.
	Principal string `json:"principal"`
	Role      string `json:"role"`
	// ReviewExecution and ImplementationExecution are the two runs the
	// separation is about; ImplementationPrincipal is who produced the change.
	ReviewExecution         string `json:"review_execution"`
	ImplementationPrincipal string `json:"implementation_principal"`
	ImplementationExecution string `json:"implementation_execution"`
	Issuer                  string `json:"issuer"`
	IssuedAt                string `json:"issued_at"`
	ExpiresAt               string `json:"expires_at,omitempty"`
	// Audience is the project this claim was issued for. A claim replayed from
	// another project names that project here and stops being satisfied.
	Audience string `json:"audience"`
}

type ReviewAttestation struct {
	SchemaVersion int                         `json:"schema_version"`
	AttestationID string                      `json:"attestation_id"`
	BundleID      string                      `json:"bundle_id"`
	BundleDigest  string                      `json:"bundle_digest"`
	Reviewer      string                      `json:"reviewer"`
	Decision      string                      `json:"decision"`
	Criteria      []ReviewCriterion           `json:"criteria"`
	Tools         []ReviewToolDisposition     `json:"tools,omitempty"`
	EvidenceRefs  []string                    `json:"evidence_refs,omitempty"`
	Findings      []ReviewFinding             `json:"findings"`
	Authority     *ReviewAuthorityClaim       `json:"authority,omitempty"`
	ReusedFrom    []ReviewAttestationReuse    `json:"reused_from,omitempty"`
	Supersedes    string                      `json:"supersedes,omitempty"`
	Envelope      *ReviewAttestationSignature `json:"envelope,omitempty"`
	AttestedAt    string                      `json:"attested_at"`
	Path          string                      `json:"path,omitempty"`
}

type ReviewAttestationEnvelope struct {
	SchemaVersion int               `json:"schema_version"`
	Issuer        string            `json:"issuer"`
	Subject       string            `json:"subject"`
	Algorithm     string            `json:"algorithm"`
	PublicKey     string            `json:"public_key"`
	Signature     string            `json:"signature"`
	Attestation   ReviewAttestation `json:"attestation"`
}

// ReviewAttestationPendency is one answer the attestation still owes. It is
// not a disposition: nothing is written for a pending criterion, because the
// whole point is that no value stands in for the answer.
type ReviewAttestationPendency struct {
	Criterion string `json:"criterion"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
}

// ReviewAttestationPreparation is what automation can honestly produce on its
// own: every mechanical criterion answered from sealed evidence, and an
// explicit list of what a reviewer still has to answer.
type ReviewAttestationPreparation struct {
	Attestation ReviewAttestation           `json:"attestation"`
	Pending     []ReviewAttestationPendency `json:"pending,omitempty"`
	Complete    bool                        `json:"complete"`
}

type ReviewBundleDelta struct {
	FromBundle             string   `json:"from_bundle,omitempty"`
	ToBundle               string   `json:"to_bundle"`
	ChangedComponents      []string `json:"changed_components,omitempty"`
	ChangedSections        []string `json:"changed_sections,omitempty"`
	ChangedPaths           []string `json:"changed_paths,omitempty"`
	ChangedCriteria        []string `json:"changed_criteria,omitempty"`
	ChangedEvidence        []string `json:"changed_evidence,omitempty"`
	ChangedEvidenceClasses []string `json:"changed_evidence_classes,omitempty"`
	ChangedFindings        []string `json:"changed_findings,omitempty"`
	ReusableCriteria       []string `json:"reusable_criteria,omitempty"`
}

type ReviewBundleVerification struct {
	Scope       string             `json:"scope"`
	State       string             `json:"state"`
	Bundle      *ReviewBundle      `json:"bundle,omitempty"`
	Attestation *ReviewAttestation `json:"attestation,omitempty"`
	Delta       *ReviewBundleDelta `json:"delta,omitempty"`
	Fresh       bool               `json:"fresh"`
	Approved    bool               `json:"approved"`
	NextAction  string             `json:"next_action"`
	Warnings    []string           `json:"warnings,omitempty"`
	Blockers    []string           `json:"blockers,omitempty"`
}

// PrepareReviewBundle resolves a deterministic bundle without writing it.
func (s Store) PrepareReviewBundle(ref string) (ReviewBundle, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return ReviewBundle{}, err
	}
	plan, err := s.ReviewPlan(ref)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle := ReviewBundle{SchemaVersion: ReviewBundleSchemaVersion, State: "prepared", Warnings: uniqueSorted(append([]string{}, plan.Warnings...)), Blockers: uniqueSorted(append([]string{}, plan.Blockers...))}
	bundlePlanDigest, err := reviewBundlePlanDigest(plan)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle.Payload.Plan = ReviewBundlePlan{PlanDigest: bundlePlanDigest, Independence: plan.Independence, Components: append([]ReviewPlanComponent{}, plan.Components...), Criteria: append([]ReviewPlanCriterion{}, plan.Criteria...), Tools: append([]ReviewPlanTool{}, plan.Tools...), SelectedProfiles: append([]ReviewPlanProfile{}, plan.SelectedProfiles...), Structure: plan.Structure}

	scopeProjection, excluded, err := s.reviewBundleScopeProjection(scope)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle.Payload.Scope = scopeProjection
	bundle.ExcludedInputs = excluded

	graph, graphErr := s.GetDeliveryIntegrity("")
	if graphErr != nil {
		bundle.Blockers = append(bundle.Blockers, "delivery integrity is unavailable: run pose index")
	} else {
		var subjectExcluded []ReviewBundleInput
		bundle.Payload.Subject, subjectExcluded, bundle.Blockers, err = s.reviewBundleSubject(scope, bundle.Payload.Plan.Components, graph, bundle.Blockers)
		if err != nil {
			return ReviewBundle{}, err
		}
		bundle.ExcludedInputs = append(bundle.ExcludedInputs, subjectExcluded...)
		bundle.Payload.Evidence = s.reviewBundleEvidence(scope, graph, bundle.Payload.Subject)
		for i := range bundle.Payload.Evidence {
			bundle.Payload.Evidence[i].SubjectObservation = ReviewEvidenceObservationForSubject(bundle.Payload.Subject, bundle.Payload.Evidence[i])
		}
		if len(bundle.Payload.Evidence) == 0 && s.reviewScopeRequiresValidationEvidence(scope, bundle.Payload.Plan, graph) {
			bundle.Blockers = append(bundle.Blockers, "no passed structured validation evidence is attributed to the review scope")
		}
		bundle.Warnings = append(bundle.Warnings, staleEvidenceWarnings(bundle.Payload.Subject, bundle.Payload.Evidence)...)
		bundle.RangeObservations = s.reviewBundleRangeObservations(bundle.Payload.Subject.ChangeSets, graph)
		bundle.Warnings = append(bundle.Warnings, rangeObservationWarnings(bundle.RangeObservations)...)
	}

	if scope.Kind != "spec" {
		children, childBlockers, childErr := s.reviewBundleChildren(scope)
		if childErr != nil {
			return ReviewBundle{}, childErr
		}
		bundle.Payload.Children = children
		bundle.Blockers = append(bundle.Blockers, childBlockers...)
	}

	bundle.Payload.ConsumedInputs = s.reviewBundleConsumedInputs(plan)
	bundle.Payload.GoverningContracts = governingContractsAtSeal()
	if policy, _, policyErr := s.loadReviewPolicy(); policyErr == nil {
		bundle.Payload.Gates = &ReviewBundleGates{
			AllowApprovedWithReservations: policy.AllowApprovedWithReservations,
			AcceptedRiskSeverities:        append([]string{}, policy.AcceptedRiskSeverities...),
			AllowCriterionReuse:           policy.AllowCriterionReuse,
			IdentityAssurance:             policy.ReviewIdentityAssurance(scope.Kind),
		}
	}
	bundle.Blockers = uniqueSorted(bundle.Blockers)
	bundle.Warnings = uniqueSorted(bundle.Warnings)
	bundle.ExcludedInputs = sortedBundleInputs(bundle.ExcludedInputs)
	normalizeReviewBundlePayload(&bundle.Payload)
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil {
		return ReviewBundle{}, err
	}
	bundle.BundleDigest = digest
	bundle.BundleID = "rvb-" + strings.TrimPrefix(digest, "sha256:")[:16]
	return bundle, nil
}

func (s Store) reviewBundleScopeProjection(scope ScopeRef) (ReviewBundleScope, []ReviewBundleInput, error) {
	projection := ReviewBundleScope{Ref: scope.String(), Kind: scope.Kind, Slug: scope.Slug, Roadmap: scope.Roadmap, Milestone: scope.Milestone, Sections: []ReviewBundleInput{}}
	excluded := []ReviewBundleInput{}
	switch scope.Kind {
	case "spec":
		sp, err := s.GetSpec(scope.Slug)
		if err != nil {
			return projection, nil, err
		}
		projection.DependsOn = append([]string{}, sp.DependsOn...)
		if err := s.ValidateRemediationLineage(*sp); err != nil {
			return projection, nil, err
		}
		projection.Remediates = append([]string{}, sp.Remediates...)
		sort.Strings(projection.Remediates)
		projection.Supersedes = sp.Supersedes
		projection.Components = append([]string{}, sp.Components...)
		projection.Deliveries = append([]string{}, sp.Delivers...)
		sort.Strings(projection.DependsOn)
		sort.Strings(projection.Components)
		sort.Strings(projection.Deliveries)
		sections := markdownLevelTwoSections(sp.Body)
		for _, name := range []string{"intent", "requirements", "technical plan", "decisions"} {
			body := sections[name]
			if body == "" {
				continue
			}
			projection.Sections = append(projection.Sections, ReviewBundleInput{Kind: "semantic-section", Path: name, Digest: digestText(body), Reason: "governed semantic review input"})
		}
		for _, name := range []string{"tasks", "validation", "final report"} {
			if sections[name] != "" {
				excluded = append(excluded, ReviewBundleInput{Kind: "derived-section", Path: name, Digest: digestText(sections[name]), Reason: "operational closeout content is verified by its own gate"})
			}
		}
		excluded = append(excluded, ReviewBundleInput{Kind: "lifecycle", Path: filepath.ToSlash(sp.Path), Reason: "status and completed_at do not define the reviewed semantic subject"})
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return projection, nil, err
		}
		content := normalizeBundleText(roadmapMilestoneSection(rm.Body, scope.Milestone))
		if content == "" {
			return projection, nil, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
		}
		projection.Sections = append(projection.Sections, ReviewBundleInput{Kind: "milestone", Path: scope.Milestone, Digest: digestText(content), Reason: "governed milestone exit and cut criteria"})
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return projection, nil, err
		}
		// Repo-relative by construction, not rm.Path (absolute — Store.Root
		// joined with the roadmap file): an absolute path bakes the sealing
		// machine's checkout location into the sealed bundle's semantic
		// identity, so the exact same content re-sealed from a different
		// checkout path (any other clone, including CI's) computes a
		// different ChangedSections key and reads as "changed" even though
		// nothing changed. Every other scope kind already avoids this
		// (spec uses section names, milestone uses the milestone ID).
		projection.Sections = append(projection.Sections, ReviewBundleInput{Kind: "roadmap", Path: filepath.ToSlash(filepath.Join(".pose", "roadmaps", scope.Slug+".md")), Digest: digestText(normalizeBundleText(rm.Body)), Reason: "governed roadmap outcomes and ordered membership"})
	}
	return projection, excluded, nil
}

func markdownLevelTwoSections(body string) map[string]string {
	sections := map[string]string{}
	current := ""
	var lines []string
	flush := func() {
		if current != "" {
			sections[current] = normalizeBundleText(strings.Join(lines, "\n"))
		}
		lines = nil
	}
	for _, raw := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			flush()
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if dot := strings.Index(title, "."); dot >= 0 && dot < 3 {
				title = strings.TrimSpace(title[dot+1:])
			}
			current = strings.ToLower(title)
			continue
		}
		if current != "" {
			lines = append(lines, strings.TrimRight(raw, " \t"))
		}
	}
	flush()
	return sections
}

func (s Store) reviewBundleSubject(scope ScopeRef, components []ReviewPlanComponent, graph DeliveryIntegrityGraph, blockers []string) (ReviewBundleSubject, []ReviewBundleInput, []string, error) {
	if len(components) == 0 {
		if ctx, err := s.resolveReviewPlanContext(scope); err == nil {
			components = ctx.Components
		}
	}
	subject := ReviewBundleSubject{ChangeSets: []string{}, Entries: []ReviewBundleSubjectEntry{}}
	excluded := []ReviewBundleInput{}
	allowedSpecs, err := s.reviewBundleScopeSpecs(scope)
	if err != nil {
		return subject, excluded, blockers, err
	}
	sets := []ChangeSet{}
	for _, set := range graph.ChangeSets {
		if allowedSpecs[set.Spec] {
			sets = append(sets, set)
		}
	}
	sets = reduceReviewBundleChangeSets(sets)
	if len(sets) == 0 {
		blockers = append(blockers, "no immutable attributed change set exists for "+scope.String())
		return subject, excluded, blockers, nil
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].ID < sets[j].ID })
	seen := map[string]bool{}
	supersededPaths := map[string]bool{}
	for _, set := range sets {
		for _, observed := range set.Paths {
			if observed.Action == "renamed" && observed.OldPath != "" {
				supersededPaths[observed.OldPath] = true
			}
		}
	}
	// Resolve every gitlink up front, in one call. A submodule is a gitlink in
	// the index and a directory in the working tree, so it has no blob to read —
	// and it is not recognisable by path shape, which means asking after the
	// shape rules have already classified it is too late: a submodule under a
	// mapped component or a known prefix is classified as implementation, skips
	// the lookup, and then fails when the digest step tries to read a directory.
	candidatePaths := []string{}
	for _, set := range sets {
		for _, observed := range set.Paths {
			candidatePaths = append(candidatePaths, observed.Path)
			if observed.NewPath != "" {
				candidatePaths = append(candidatePaths, observed.NewPath)
			}
		}
	}
	gitlinks := reviewBundleGitlinks(s.Root, uniqueSorted(candidatePaths))
	for _, set := range sets {
		subject.ChangeSets = append(subject.ChangeSets, set.ID)
		if subject.Base == "" {
			subject.Base = set.ResolvedBase
		}
		subject.Head = set.ResolvedHead
		for _, observed := range set.Paths {
			if observed.Action != "renamed" && supersededPaths[observed.Path] {
				excluded = append(excluded, ReviewBundleInput{Kind: "superseded-path", Path: observed.Path, Reason: "a later attributed rename supplies the current review subject identity"})
				continue
			}
			key := observedKey(observed)
			if seen[key] {
				continue
			}
			seen[key] = true
			entry := ReviewBundleSubjectEntry{Action: observed.Action, Path: observed.Path, OldPath: observed.OldPath, NewPath: observed.NewPath}
			path := observed.Path
			if observed.Action == "renamed" {
				path = observed.NewPath
			}
			class, include := reviewBundlePathClass(path, scope, components)
			// Git's answer wins over the path's shape. reviewBundlePathClass stays
			// a pure function of the path; being a gitlink is a fact about the
			// index that no path shape can express.
			gitlinkSHA := gitlinks[path]
			if gitlinkSHA != "" {
				class, include = "submodule", true
			}
			switch {
			case class == "" && observed.Action == "removed":
				// A removal has no content to classify and none to read. The
				// reviewable fact is the deletion itself, and it is in the
				// subject either way — so refusing the whole bundle because the
				// shape rules do not recognise a path that no longer exists
				// costs the reviewer the entire review to tell them nothing.
				entry.Class, include = "removed", true
				entry.Reason = "attributed removal of a path with no governed classification"
			case class == "":
				blockers = append(blockers, "unclassified review subject path "+path)
				entry.Class = "unclassified"
				entry.Reason = "attributed path has no governed review classification"
			case gitlinkSHA != "":
				entry.Class = class
				entry.Reason = "attributed submodule pinned to " + gitlinkSHA
			default:
				entry.Class = class
				entry.Reason = "attributed " + class + " path in the immutable change set"
			}
			if !include {
				excluded = append(excluded, ReviewBundleInput{Kind: class, Path: path, Reason: "attributed path is outside the semantic review subject"})
				continue
			}
			if dirty, detail := reviewBundleWorkingTreeChange(s.Root, path); dirty {
				blockers = append(blockers, "review subject path "+path+" has working-tree-only content"+detail)
			}
			if include && observed.Action != "removed" {
				if gitlinkSHA != "" {
					// The reviewable identity of a submodule is the commit it is
					// pinned to. There is no file to read: the path is a directory
					// in the working tree and a gitlink in the index.
					entry.Digest = digestBytes([]byte(gitlinkSHA))
				} else if digest, err := s.reviewBundleFileDigest(path); err != nil {
					blockers = append(blockers, err.Error())
				} else {
					entry.Digest = digest
				}
			}
			subject.Entries = append(subject.Entries, entry)
		}
	}
	sort.Slice(subject.Entries, func(i, j int) bool {
		a, b := subject.Entries[i], subject.Entries[j]
		return a.Action+"\x00"+a.Path+"\x00"+a.OldPath+"\x00"+a.NewPath < b.Action+"\x00"+b.Path+"\x00"+b.OldPath+"\x00"+b.NewPath
	})
	patchRaw, _ := json.Marshal(subject.Entries)
	subject.PatchDigest = digestBytes(patchRaw)
	treeEntries := make([]struct {
		Path, Digest string
	}, 0, len(subject.Entries))
	for _, entry := range subject.Entries {
		path := entry.Path
		if entry.NewPath != "" {
			path = entry.NewPath
		}
		treeEntries = append(treeEntries, struct{ Path, Digest string }{path, entry.Digest})
	}
	treeRaw, _ := json.Marshal(treeEntries)
	subject.TreeDigest = digestBytes(treeRaw)
	subject.ImplementationDigest = reviewImplementationDigest(subject)
	return subject, sortedBundleInputs(excluded), blockers, nil
}

// reviewImplementationDigest identifies the implementation content of a subject
// and nothing else.
func reviewImplementationDigest(subject ReviewBundleSubject) string {
	// Content only. Not the change-set ids, not base and head: a squash merge
	// or a rebase gives the same content a different SHA, and an implementation
	// identity that moved with the SHA would call that a different subject. It
	// is the same rule the bundle digest already follows, named once so an
	// observer outside the bundle can anchor on it.
	raw, err := json.Marshal(struct {
		Entries     []ReviewBundleSubjectEntry `json:"entries"`
		PatchDigest string                     `json:"patch_digest"`
		TreeDigest  string                     `json:"tree_digest"`
	}{Entries: subject.Entries, PatchDigest: subject.PatchDigest, TreeDigest: subject.TreeDigest})
	if err != nil {
		return ""
	}
	return digestBytes(raw)
}

// reviewBundleRangeObservations counts, for each attributed change set, how
// many commits its range spans against how many it attributes.
//
// The count needs Git, and Git is not always there — an exported tree, a unit
// fixture. When it cannot be taken the observation is `unknown`, never `clean`,
// because the whole purpose is to stop a range being read as an attribution.
func (s Store) reviewBundleRangeObservations(ids []string, graph DeliveryIntegrityGraph) []ReviewBundleRangeObservation {
	// Resolved from the ids the subject sealed, so the observation always
	// describes the same change sets the review is about.
	byID := map[string]ChangeSet{}
	for _, set := range graph.ChangeSets {
		byID[set.ID] = set
	}
	observations := []ReviewBundleRangeObservation{}
	for _, id := range ids {
		set, ok := byID[id]
		if !ok {
			observations = append(observations, ReviewBundleRangeObservation{ChangeSet: id, State: "unknown", Reason: "the change set is not present in the current integrity graph"})
			continue
		}
		observation := ReviewBundleRangeObservation{
			ChangeSet:         set.ID,
			AttributedCommits: len(set.Commits),
		}
		switch {
		case set.ResolvedBase == "" || set.ResolvedHead == "":
			observation.State = "unknown"
			observation.Reason = "the change set resolves no immutable range to count against"
		default:
			count, err := gitRevListCount(s.Root, set.ResolvedBase, set.ResolvedHead)
			switch {
			case err != nil:
				observation.State = "unknown"
				observation.Reason = "the range could not be counted in this working tree"
			case len(set.Commits) == 0:
				observation.State = "unknown"
				observation.Reason = "the change set attributes no commit to compare the range with"
			case count > len(set.Commits):
				observation.State = "contaminated"
				observation.RangeCommits = count
				observation.UnattributedCommits = count - len(set.Commits)
				observation.Reason = "the range spans commits this change set does not attribute; its paths are attributed, its base..head is not"
			default:
				observation.State = "clean"
				observation.RangeCommits = count
			}
		}
		observations = append(observations, observation)
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].ChangeSet < observations[j].ChangeSet })
	return observations
}

func gitRevListCount(root, base, head string) (int, error) {
	out, err := exec.Command("git", "-C", root, "rev-list", "--count", base+".."+head, "--").Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

func reduceReviewBundleChangeSets(sets []ChangeSet) []ChangeSet {
	result := make([]ChangeSet, 0, len(sets))
	for i, candidate := range sets {
		candidateCommits := map[string]bool{}
		for _, commit := range candidate.Commits {
			candidateCommits[commit] = true
		}
		subsumed := false
		if len(candidateCommits) > 0 {
			for j, other := range sets {
				if i == j || len(other.Commits) <= len(candidate.Commits) {
					continue
				}
				containsAll := true
				otherCommits := map[string]bool{}
				for _, commit := range other.Commits {
					otherCommits[commit] = true
				}
				for commit := range candidateCommits {
					if !otherCommits[commit] {
						containsAll = false
						break
					}
				}
				if containsAll {
					subsumed = true
					break
				}
			}
		}
		if !subsumed {
			result = append(result, candidate)
		}
	}
	return result
}

func (s Store) reviewBundleScopeSpecs(scope ScopeRef) (map[string]bool, error) {
	result := map[string]bool{}
	switch scope.Kind {
	case "spec":
		result[scope.Slug] = true
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				for _, slug := range milestone.Specs {
					result[slug] = true
				}
				return result, nil
			}
		}
		return nil, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			for _, slug := range milestone.Specs {
				result[slug] = true
			}
		}
	}
	return result, nil
}

// reviewBundleGitlinks maps each of the given paths that is a gitlink to the
// commit it is pinned to. Git records a submodule in the index with mode 160000
// and the commit id in place of a blob id, so that pointer is the only content a
// submodule has to review. Resolved in one call over the candidate set rather
// than one call per path, and bounded by the subject's own size.
func reviewBundleGitlinks(root string, paths []string) map[string]string {
	links := map[string]string{}
	if len(paths) == 0 {
		return links
	}
	args := append([]string{"-C", root, "ls-files", "--stage", "--"}, paths...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// Unit fixtures and exported source trees may not have Git metadata. A
		// path that cannot be resolved keeps its shape-based class and fails
		// closed downstream if it has no readable content.
		return links
	}
	for _, line := range strings.Split(string(out), "\n") {
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		fields := strings.Fields(line[:tab])
		if len(fields) < 2 || fields[0] != "160000" {
			continue
		}
		links[filepath.ToSlash(line[tab+1:])] = fields[1]
	}
	return links
}

func reviewBundleWorkingTreeChange(root, path string) (bool, string) {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain=v1", "--untracked-files=all", "--", path)
	out, err := cmd.Output()
	if err != nil {
		// Unit fixtures and exported source trees may not have Git metadata. The
		// immutable change-set gate remains authoritative there.
		return false, ""
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return false, ""
	}
	if newline := strings.IndexByte(line, '\n'); newline >= 0 {
		line = line[:newline]
	}
	status := strings.TrimSpace(strings.TrimSuffix(line, path))
	if status == "" {
		return true, ""
	}
	return true, " (git status " + status + ")"
}

func reviewBundlePathClass(path string, scope ScopeRef, components []ReviewPlanComponent) (string, bool) {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || path == "" {
		return "", false
	}
	if scope.Kind == "spec" && (path == ".pose/specs/"+scope.Slug+"/spec.md" || path == ".pose/specs/"+scope.Slug+".md" || strings.HasPrefix(path, ".pose/specs/"+scope.Slug+"/")) {
		return "semantic-scope", false
	}
	if strings.HasPrefix(path, ".pose/specs/") {
		return "semantic-scope", false
	}
	for _, prefix := range []string{".pose/state/", ".pose/assessments/", ".pose/reports/", ".pose/results/", ".pose/reviews/", ".pose/review-bundles/", ".pose/review-attestations/", ".pose/contributions/", ".pose/feedback/"} {
		if strings.HasPrefix(path, prefix) {
			return "derived-evidence", false
		}
	}
	for _, exact := range []string{".pose/indexes/delivery-integrity.json", ".pose/indexes/releases.json", ".pose/indexes/spec-graph.json"} {
		if path == exact {
			return "derived-index", false
		}
	}
	for _, exact := range []string{".pose/indexes/validation-matrix.json", ".pose/indexes/module-metadata.json", ".pose/indexes/task-map.json", ".pose/indexes/repo-map.json"} {
		if path == exact {
			return "governance", true
		}
	}
	if strings.HasPrefix(path, ".pose/indexes/") {
		return "derived-index", false
	}
	// .pose/public/ holds the public claims contract (spec
	// pose-public-claims-contract): what the project asserts about itself in
	// public. It is authored policy, not derived output, so a change to it
	// belongs in the review subject exactly like a rule or a workflow.
	for _, prefix := range []string{".pose/policy/", ".pose/public/", ".pose/releases/", ".pose/review-profiles/", ".pose/rules/", ".pose/workflows/", ".pose/roadmaps/", ".agents/skills/", "extensions/", ".pose/changelogs/", ".pose/adr/", ".pose/knowledge/", ".pose/templates/"} {
		if strings.HasPrefix(path, prefix) {
			return "governance", true
		}
	}
	// composition-contract.json joins the list for the same reason compatibility.json
	// is on it: both are published root manifests describing what this repository
	// exposes, and a change to either is a governance change. Without it, a spec that
	// adds an environment variable — which the composition contract enumerates —
	// could not seal a review bundle at all, because the classifier refuses an
	// unclassified subject path rather than guessing at one.
	for _, exact := range []string{".pose/docs.json", ".pose/docs-review.jsonl", ".pose/release-policy.json", ".pose/project.json", "compatibility.json", "composition-contract.json", "pose-mcp/server.json"} {
		if path == exact {
			return "governance", true
		}
	}
	baseName := filepath.Base(path)
	manifestNames := map[string]bool{
		"go.mod": true, "go.sum": true, "go.work": true, "go.work.sum": true,
		"package.json": true, "package-lock.json": true, "pnpm-lock.yaml": true, "yarn.lock": true, "bun.lockb": true,
		"Cargo.toml": true, "Cargo.lock": true,
		"pyproject.toml": true, "poetry.lock": true, "requirements.txt": true, "Pipfile": true, "Pipfile.lock": true, "setup.py": true, "setup.cfg": true,
		"pom.xml": true, "build.gradle": true, "build.gradle.kts": true, "settings.gradle": true, "settings.gradle.kts": true,
		"CMakeLists.txt": true, "Makefile": true, "Dockerfile": true, "docker-compose.yml": true, "docker-compose.yaml": true, "compose.yaml": true, "compose.yml": true,
		"tsconfig.json": true, "jsconfig.json": true, "turbo.json": true, "biome.json": true,
		".gitignore": true, ".gitattributes": true, ".editorconfig": true,
		".golangci.yml": true, ".golangci.yaml": true, "buf.yaml": true, "buf.gen.yaml": true,
		".pre-commit-config.yaml": true, ".goreleaser.yaml": true, ".goreleaser.yml": true,
	}
	if manifestNames[baseName] || strings.HasPrefix(baseName, ".eslintrc") || strings.HasPrefix(baseName, ".prettierrc") || strings.HasPrefix(baseName, "vite.config.") || strings.HasPrefix(baseName, "webpack.config.") || strings.HasPrefix(baseName, "rollup.config.") || strings.HasPrefix(baseName, "next.config.") {
		return "governance", true
	}
	if !strings.Contains(path, "/") {
		baseLower := strings.ToLower(path)
		if strings.HasSuffix(baseLower, ".md") || strings.HasSuffix(baseLower, ".txt") || strings.HasPrefix(baseLower, "license") || baseLower == "notice" {
			return "documentation", true
		}
	}
	for _, component := range components {
		root := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(component.Path)), "/")
		if root == "." || root == "" {
			if !strings.HasPrefix(path, ".pose/") && !strings.HasPrefix(path, ".git/") {
				return "implementation", true
			}
		} else if path == root || strings.HasPrefix(path, root+"/") {
			return "implementation", true
		}
	}
	if strings.HasPrefix(path, "docs/decisions/") {
		return "governance", true
	}
	if strings.HasPrefix(path, "docs-site/docs/") || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, "locales/") {
		return "documentation", true
	}
	for _, prefix := range []string{"pose-mcp/", "mcp-enforce/", "docs-site/", "locales/", "scripts/", "tests/", ".github/", "cmd/", "internal/", "pkg/", "src/", "lib/", "app/", "api/"} {
		if strings.HasPrefix(path, prefix) {
			return "implementation", true
		}
	}
	return "", false
}

func (s Store) reviewBundleFileDigest(rel string) (string, error) {
	if err := ValidateArtifactPath(s.Root, rel, false); err != nil {
		return "", fmt.Errorf("review subject path %s is invalid: %w", rel, err)
	}
	clean, _ := validateArtifactPathSyntax(rel)
	raw, err := os.ReadFile(filepath.Join(s.Root, clean))
	if err != nil {
		return "", fmt.Errorf("review subject path %s cannot be read: %w", rel, err)
	}
	return digestBytes(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))), nil
}

// staleEvidenceWarnings names evidence that ran against a commit other than the
// head this bundle approves.
//
// A warning, not a blocker. `deliveryEvidenceCurrent` already decides what
// counts as current, and it accepts a result whose provenance digest matches the
// graph or whose scope is closed — legitimately, since re-running every check to
// re-seal a finished spec would be theatre. What it does not do is say so, and a
// reviewer reading a sealed bundle cannot tell a result produced against this
// subject from one carried forward.
//
// That distinction is the whole question an attestation answers, so it belongs
// where the reviewer sees it rather than inferred from two commit hashes nobody
// compares.
func staleEvidenceWarnings(subject ReviewBundleSubject, evidence []ReviewBundleEvidence) []string {
	carried := []string{}
	unknown := []string{}
	for _, ev := range evidence {
		switch ReviewEvidenceObservationForSubject(subject, ev) {
		case "carried-forward":
			carried = append(carried, ev.EvidenceClass+":"+ev.ID)
		case "unknown":
			unknown = append(unknown, ev.EvidenceClass+":"+ev.ID)
		}
	}
	warnings := []string{}
	if len(carried) > 0 {
		sort.Strings(carried)
		warnings = append(warnings, "sealed evidence ran against a commit other than the subject head "+shortCommit(subject.Head)+
			"; it is current by provenance, not by having observed this change: "+strings.Join(carried, ", "))
	}
	// The third state, which used to be silently folded into the first two. A
	// result carrying no commit says nothing about which content it observed,
	// and a reviewer reading a clean bundle had no way to tell that apart from
	// a result that did observe this change.
	if len(unknown) > 0 {
		sort.Strings(unknown)
		warnings = append(warnings, "sealed evidence names no commit, so which content it observed is unknown rather than current: "+strings.Join(unknown, ", "))
	}
	return warnings
}

// rangeObservationWarnings reports a change set whose range spans commits it
// does not attribute, so nobody reads base..head as this scope's work.
func rangeObservationWarnings(observations []ReviewBundleRangeObservation) []string {
	warnings := []string{}
	for _, observation := range observations {
		switch observation.State {
		case "contaminated":
			warnings = append(warnings, fmt.Sprintf("change set %s attributes %d commit(s) but its range spans %d; %d commit(s) in base..head belong to other work, so the range is provenance and the attributed paths are the subject",
				observation.ChangeSet, observation.AttributedCommits, observation.RangeCommits, observation.UnattributedCommits))
		case "unknown":
			warnings = append(warnings, fmt.Sprintf("change set %s could not have its range counted: %s", observation.ChangeSet, observation.Reason))
		}
	}
	return warnings
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func (s Store) reviewBundleEvidence(scope ScopeRef, graph DeliveryIntegrityGraph, subject ReviewBundleSubject) []ReviewBundleEvidence {
	modules := map[string]bool{}
	for _, target := range graph.Deliveries {
		if scope.Kind == "spec" && target.Spec == scope.Slug {
			modules[target.Module] = true
		}
	}
	status := ""
	if scope.Kind == "spec" {
		if spec, err := s.GetSpec(scope.Slug); err == nil {
			status = spec.Status
		}
	}
	sets := []ChangeSet{}
	for _, set := range graph.ChangeSets {
		if scope.Kind == "spec" && set.Spec == scope.Slug {
			sets = append(sets, set)
		}
	}
	result := []ReviewBundleEvidence{}
	for _, evidence := range graph.ValidationResults {
		if evidence.Outcome != "pass" || evidence.Severity != "required" {
			continue
		}
		if len(modules) > 0 {
			matched := false
			for targetMod := range modules {
				if moduleMatchesTarget(evidence.Module, targetMod) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if scope.Kind == "spec" && !deliveryEvidenceCurrent(evidence, scope.Slug, status, graph, sets) {
			continue
		}
		result = append(result, ReviewBundleEvidence{ID: evidence.ID, Module: evidence.Module, Check: evidence.Check, EvidenceClass: evidence.EvidenceClass, Outcome: evidence.Outcome, GitHead: evidence.GitHead, ProvenanceDigest: evidence.ProvenanceDigest, Report: evidence.Report})
	}
	if scope.Kind == "spec" && subject.ImplementationDigest != "" {
		if design, err := AssessDesignDelta(s.Root, subject, scope.String(), DesignDeltaOptions{}); err == nil && len(design.InputDigest) > len("sha256:")+16 {
			id := "design-" + strings.TrimPrefix(design.InputDigest, "sha256:")[:16]
			result = append(result, ReviewBundleEvidence{
				ID: id, Check: "assess-design", EvidenceClass: "structure", Outcome: "pass",
				SubjectDigest: subject.ImplementationDigest, ProvenanceDigest: design.InputDigest,
				Report: "derived:design-delta/" + strings.TrimPrefix(design.InputDigest, "sha256:"),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s Store) reviewScopeRequiresValidationEvidence(scope ScopeRef, plan ReviewBundlePlan, graph DeliveryIntegrityGraph) bool {
	switch scope.Kind {
	case "spec":
		for _, target := range graph.Deliveries {
			if target.Spec == scope.Slug {
				return true
			}
		}
		if len(plan.Components) > 0 {
			return true
		}
		for _, tool := range plan.Tools {
			if tool.ID == "validate" && tool.Requiredness == "required" && !containsFold(tool.Preconditions, "delivery-target-declared") {
				return true
			}
		}
		return false
	case "milestone", "roadmap":
		return len(graph.Deliveries) > 0
	default:
		return true
	}
}

func (s Store) reviewBundleChildRefs(scope ScopeRef) ([]string, error) {
	refs := []string{}
	if scope.Kind == "milestone" {
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				for _, slug := range milestone.Specs {
					refs = append(refs, "spec:"+slug)
				}
			}
		}
	} else if scope.Kind == "roadmap" {
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			refs = append(refs, "milestone:"+rm.Slug+"/"+milestone.ID)
		}
	}
	return refs, nil
}

func (s Store) reviewBundleChildren(scope ScopeRef) ([]ReviewBundleChild, []string, error) {
	refs, err := s.reviewBundleChildRefs(scope)
	if err != nil {
		return nil, nil, err
	}
	children, blockers := []ReviewBundleChild{}, []string{}
	for _, ref := range refs {
		sealed, err := s.CurrentReviewBundle(ref)
		if err != nil {
			return nil, nil, err
		}
		if sealed == nil {
			blockers = append(blockers, "child "+ref+" has no sealed review bundle")
			continue
		}
		children = append(children, ReviewBundleChild{Scope: ref, BundleID: sealed.BundleID, BundleDigest: sealed.BundleDigest})
	}
	return children, blockers, nil
}

func (s Store) reviewBundleConsumedInputs(plan ReviewPlan) []ReviewBundleInput {
	planDigest, _ := reviewBundlePlanDigest(plan)
	inputs := []ReviewBundleInput{
		{Kind: "review-plan", Path: plan.BaseProfile, Digest: planDigest, Reason: "effective component-aware plan contract"},
		{Kind: "tool-catalog", Path: "native-review-tools@1", Digest: planDigest, Reason: "closed resolved native tool contract"},
		{Kind: "schema", Path: "review-bundle@1", Digest: digestText("review-bundle-schema-v1\nreview-attestation-schema-v1\nreview-attestation-envelope-schema-v1"), Reason: "canonical bundle and attestation schema identity"},
	}
	for _, rel := range []string{".pose/policy/review.json", ".pose/indexes/validation-matrix.json"} {
		if raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(rel))); err == nil {
			inputs = append(inputs, ReviewBundleInput{Kind: "governed-config", Path: rel, Digest: digestBytes(raw), Reason: "consumed review policy or validation contract"})
		}
	}
	for _, profile := range plan.SelectedProfiles {
		raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(profile.Source)))
		if err == nil {
			inputs = append(inputs, ReviewBundleInput{Kind: "review-profile", Path: profile.Source, Digest: digestBytes(raw), Reason: "selected by effective review plan"})
		}
	}
	rules := map[string]bool{}
	for _, criterion := range plan.Criteria {
		for _, rule := range criterion.Rules {
			rules[rule] = true
		}
	}
	for rule := range rules {
		rel := ".pose/rules/" + rule + ".md"
		if raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(rel))); err == nil {
			inputs = append(inputs, ReviewBundleInput{Kind: "rule", Path: rel, Digest: digestBytes(raw), Reason: "selected by effective review criterion"})
		}
	}
	return sortedBundleInputs(inputs)
}

func reviewBundlePlanDigest(plan ReviewPlan) (string, error) {
	contract := struct {
		SchemaVersion       int                   `json:"schema_version"`
		PolicySchemaVersion int                   `json:"policy_schema_version"`
		BaseProfile         string                `json:"base_profile"`
		Independence        string                `json:"independence"`
		Components          []ReviewPlanComponent `json:"components"`
		SelectedProfiles    []ReviewPlanProfile   `json:"selected_profiles"`
		Criteria            []ReviewPlanCriterion `json:"criteria"`
		Tools               []ReviewPlanTool      `json:"tools"`
	}{plan.SchemaVersion, plan.PolicySchemaVersion, plan.BaseProfile, plan.Independence, append([]ReviewPlanComponent{}, plan.Components...), append([]ReviewPlanProfile{}, plan.SelectedProfiles...), append([]ReviewPlanCriterion{}, plan.Criteria...), append([]ReviewPlanTool{}, plan.Tools...)}
	return digestJSON(contract)
}

func normalizeReviewBundlePayload(payload *ReviewBundlePayload) {
	sort.Slice(payload.Scope.Sections, func(i, j int) bool { return payload.Scope.Sections[i].Path < payload.Scope.Sections[j].Path })
	sort.Slice(payload.Plan.Components, func(i, j int) bool { return payload.Plan.Components[i].ID < payload.Plan.Components[j].ID })
	sort.Slice(payload.Plan.Criteria, func(i, j int) bool { return payload.Plan.Criteria[i].ID < payload.Plan.Criteria[j].ID })
	sort.Slice(payload.Plan.Tools, func(i, j int) bool {
		return reviewToolKey(payload.Plan.Tools[i].ID, payload.Plan.Tools[i].Component) < reviewToolKey(payload.Plan.Tools[j].ID, payload.Plan.Tools[j].Component)
	})
	payload.ConsumedInputs = sortedBundleInputs(payload.ConsumedInputs)
}

func reviewBundlePayloadDigest(payload ReviewBundlePayload) (string, error) {
	canonical := payload
	// Change-set IDs and Git refs remain exported for audit/debugging but are
	// advisory provenance. Stable subject identity is the classified patch and
	// tree manifest, so provider ref movement or a derived-only follow-up commit
	// cannot change the bundle identity.
	canonical.Subject.ChangeSets = nil
	canonical.Subject.Base = ""
	canonical.Subject.Head = ""
	// The observation is derived from the head just cleared above, so sealing
	// it would smuggle that ref back into the identity: a provider ref moving,
	// or a derived-only follow-up commit, would flip a result from observed to
	// carried-forward and stale a review of content that did not move. It stays
	// in the written bundle for the reader, as advisory provenance, exactly
	// like the refs it is computed from.
	//
	// Copied first: `canonical := payload` shares the slice backing array, so
	// clearing in place would erase the state on the bundle the caller is
	// holding, not on the copy being hashed.
	canonical.Evidence = append([]ReviewBundleEvidence{}, canonical.Evidence...)
	for i := range canonical.Evidence {
		canonical.Evidence[i].SubjectObservation = ""
	}
	return digestJSON(canonical)
}

func sortedBundleInputs(values []ReviewBundleInput) []ReviewBundleInput {
	result := append([]ReviewBundleInput{}, values...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Kind+"\x00"+result[i].Path+"\x00"+result[i].Digest < result[j].Kind+"\x00"+result[j].Path+"\x00"+result[j].Digest
	})
	return result
}

func normalizeBundleText(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func digestText(value string) string { return digestBytes([]byte(normalizeBundleText(value))) }
func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s Store) reviewBundlesDir() string { return filepath.Join(s.Root, ".pose", "review-bundles") }
func (s Store) reviewAttestationsDir() string {
	return filepath.Join(s.Root, ".pose", "review-attestations")
}

func ensureReviewArtifactDir(root, rel string, create bool) (string, error) {
	clean, err := validateArtifactPathSyntax(rel)
	if err != nil {
		return "", err
	}
	current := root
	for _, part := range strings.Split(filepath.Clean(clean), string(os.PathSeparator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) && create {
			if mkdirErr := os.Mkdir(current, 0o755); mkdirErr != nil && !os.IsExist(mkdirErr) {
				return "", mkdirErr
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			relPath, _ := filepath.Rel(root, current)
			return "", fmt.Errorf("pose: refusing to follow review artifact symlink at %s", filepath.ToSlash(relPath))
		}
		if !info.IsDir() {
			return "", fmt.Errorf("pose: review artifact path is not a directory: %s", filepath.ToSlash(rel))
		}
	}
	return current, nil
}

// SealReviewBundle writes a prepared, unblocked bundle atomically. Replaying
// the same payload is idempotent.
func (s Store) SealReviewBundle(ref string, now time.Time) (ReviewBundle, error) {
	bundle, err := s.PrepareReviewBundle(ref)
	if err != nil {
		return ReviewBundle{}, err
	}
	if len(bundle.Blockers) > 0 {
		return bundle, fmt.Errorf("pose: review bundle is not sealable: %s", strings.Join(bundle.Blockers, "; "))
	}
	bundle.State = "sealed"
	bundle.SealedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	bundle.Path = filepath.ToSlash(filepath.Join(".pose", "review-bundles", bundle.BundleID+".json"))
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-bundles")), true)
	if err != nil {
		return ReviewBundle{}, err
	}
	path := filepath.Join(dir, bundle.BundleID+".json")
	if existing, readErr := s.LoadReviewBundle(bundle.BundleID); readErr == nil {
		if existing.BundleDigest != bundle.BundleDigest {
			return ReviewBundle{}, fmt.Errorf("pose: review bundle identity collision for %s", bundle.BundleID)
		}
		return existing, nil
	}
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return ReviewBundle{}, err
	}
	raw = append(raw, '\n')
	if err := writeImmutableJSON(path, raw); err != nil {
		return ReviewBundle{}, err
	}
	return bundle, nil
}

func writeImmutableJSON(path string, raw []byte) error {
	if len(raw) > maxReviewBundleBytes {
		return fmt.Errorf("pose: review artifact exceeds %d bytes", maxReviewBundleBytes)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pose-review-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmpPath, path); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("pose: immutable review artifact already exists: %s", filepath.Base(path))
		}
		return err
	}
	return os.Remove(tmpPath)
}

func strictJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(raw) > maxReviewBundleBytes {
		return fmt.Errorf("review artifact exceeds %d bytes", maxReviewBundleBytes)
	}
	if err := rejectDuplicateJSONKeysAndControls(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON content")
		}
		return fmt.Errorf("trailing JSON content: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeysAndControls(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var walk func() error
	checkText := func(value string) error {
		for _, r := range value {
			if r < 0x20 || r == 0x7f {
				return fmt.Errorf("JSON string contains a control character")
			}
		}
		return nil
	}
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{':
				seen := map[string]bool{}
				for decoder.More() {
					keyToken, err := decoder.Token()
					if err != nil {
						return err
					}
					key, ok := keyToken.(string)
					if !ok {
						return fmt.Errorf("JSON object key must be a string")
					}
					if err := checkText(key); err != nil {
						return err
					}
					if seen[key] {
						return fmt.Errorf("duplicate JSON field %q", key)
					}
					seen[key] = true
					if err := walk(); err != nil {
						return err
					}
				}
				end, err := decoder.Token()
				if err != nil || end != json.Delim('}') {
					return fmt.Errorf("malformed JSON object")
				}
			case '[':
				for decoder.More() {
					if err := walk(); err != nil {
						return err
					}
				}
				end, err := decoder.Token()
				if err != nil || end != json.Delim(']') {
					return fmt.Errorf("malformed JSON array")
				}
			default:
				return fmt.Errorf("unexpected JSON delimiter %q", value)
			}
		case string:
			return checkText(value)
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON content")
		}
		return fmt.Errorf("trailing JSON content: %w", err)
	}
	return nil
}

func (s Store) LoadReviewBundle(id string) (ReviewBundle, error) {
	if !strings.HasPrefix(id, "rvb-") || len(id) != 20 {
		return ReviewBundle{}, fmt.Errorf("pose: invalid review bundle id %q", id)
	}
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-bundles")), false)
	if err != nil {
		return ReviewBundle{}, err
	}
	path := filepath.Join(dir, id+".json")
	var bundle ReviewBundle
	if err := strictJSONFile(path, &bundle); err != nil {
		return ReviewBundle{}, fmt.Errorf("pose: reading review bundle %s: %w", id, err)
	}
	if bundle.SchemaVersion != ReviewBundleSchemaVersion || bundle.BundleID != id || bundle.State != "sealed" {
		return ReviewBundle{}, fmt.Errorf("pose: malformed review bundle %s", id)
	}
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil || digest != bundle.BundleDigest || id != "rvb-"+strings.TrimPrefix(digest, "sha256:")[:16] {
		return ReviewBundle{}, fmt.Errorf("pose: review bundle %s digest mismatch", id)
	}
	bundle.Path = filepath.ToSlash(filepath.Join(".pose", "review-bundles", id+".json"))
	return bundle, nil
}

func (s Store) ListReviewBundles(scope string) ([]ReviewBundle, error) {
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-bundles")), false)
	if os.IsNotExist(err) {
		return []ReviewBundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ReviewBundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	bundles := []ReviewBundle{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if scope != "" {
			raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			var header struct {
				Payload struct {
					Scope struct {
						Ref string `json:"ref"`
					} `json:"scope"`
				} `json:"payload"`
			}
			if err := json.Unmarshal(raw, &header); err != nil || header.Payload.Scope.Ref != scope {
				continue
			}
		}
		bundle, loadErr := s.LoadReviewBundle(id)
		if loadErr != nil {
			if scope != "" {
				return nil, loadErr
			}
			continue
		}
		if scope == "" || bundle.Payload.Scope.Ref == scope {
			bundles = append(bundles, bundle)
		}
	}
	sort.Slice(bundles, func(i, j int) bool {
		if bundles[i].SealedAt == bundles[j].SealedAt {
			return bundles[i].BundleID < bundles[j].BundleID
		}
		return bundles[i].SealedAt < bundles[j].SealedAt
	})
	return bundles, nil
}

func (s Store) CurrentReviewBundle(scope string) (*ReviewBundle, error) {
	prepared, err := s.PrepareReviewBundle(scope)
	if err != nil {
		return nil, err
	}
	bundles, err := s.ListReviewBundles(scope)
	if err != nil {
		return nil, err
	}
	for i := len(bundles) - 1; i >= 0; i-- {
		if bundles[i].BundleDigest == prepared.BundleDigest {
			bundle := bundles[i]
			return &bundle, nil
		}
	}
	return nil, nil
}

// AutoAttestReviewBundle prepares an attestation from the bundle's validated
// evidence and plan dispositions, and records it only when nothing is left for
// a reviewer to answer.
//
// It used to record an `approved` decision for every required criterion,
// including the ones no check reports on. In POSE's own history that produced
// 458 of 470 attestations in which every criterion cites the same single
// evidence reference and no criterion carries a conclusion — `roadmap-outcome`
// has six criteria and not one evidence class, so a roadmap closeout was
// approved in full by whichever result happened to be first in the bundle. The
// collection half of that was always useful and is kept; the approval half is
// what this stops doing (spec pose-abm-review-soundness).
func (s Store) AutoAttestReviewBundle(bundleID, reviewer string, apply bool, now time.Time) (ReviewAttestation, error) {
	prepared, err := s.PrepareReviewAttestation(bundleID, reviewer, now)
	if err != nil {
		return ReviewAttestation{}, err
	}
	if !prepared.Complete {
		if !apply {
			return prepared.Attestation, nil
		}
		return ReviewAttestation{}, fmt.Errorf("pose: %s", ReviewPendencySummary(bundleID, prepared.Pending))
	}
	if !apply {
		return prepared.Attestation, nil
	}
	return s.RecordReviewAttestation(prepared.Attestation, now)
}

// ReviewPendencySummary renders the pendencies as one actionable line: what is
// unanswered, why, and the command that answers it.
func ReviewPendencySummary(bundleID string, pending []ReviewAttestationPendency) string {
	ids := make([]string, 0, len(pending))
	for _, item := range pending {
		ids = append(ids, item.Criterion)
	}
	sort.Strings(ids)
	return fmt.Sprintf("bundle %s has %d criteria awaiting a reviewer's judgment (%s); record them with `pose review attest --criterion ID|passed|<evidence>|<conclusion>` (or `not-applicable`, or `finding`), because no collected evidence answers them", bundleID, len(pending), strings.Join(ids, ", "))
}

// PrepareReviewAttestation answers every criterion automation may answer and
// reports the rest as pendencies. It never writes.
func (s Store) PrepareReviewAttestation(bundleID, reviewer string, now time.Time) (ReviewAttestationPreparation, error) {
	att, pending, err := s.prepareReviewAttestation(bundleID, reviewer, now)
	if err != nil {
		return ReviewAttestationPreparation{}, err
	}
	return ReviewAttestationPreparation{Attestation: att, Pending: pending, Complete: len(pending) == 0}, nil
}

func (s Store) prepareReviewAttestation(bundleID, reviewer string, now time.Time) (ReviewAttestation, []ReviewAttestationPendency, error) {
	bundle, err := s.LoadReviewBundle(bundleID)
	if err != nil {
		return ReviewAttestation{}, nil, err
	}
	if reviewer == "" {
		reviewer = "agent:auto-attest"
	}
	scope, _ := ParseScopeRef(bundle.Payload.Scope.Ref)
	graph, _ := s.GetDeliveryIntegrity("")
	requiresEvidence := s.reviewScopeRequiresValidationEvidence(scope, bundle.Payload.Plan, graph)
	if len(bundle.Payload.Evidence) == 0 && requiresEvidence {
		return ReviewAttestation{}, nil, fmt.Errorf("pose: bundle %s has no passed structured validation evidence", bundleID)
	}
	evidenceRefs := make([]string, 0, len(bundle.Payload.Evidence))
	byClass := map[string][]string{}
	evidenceModule := map[string]string{}
	for _, ev := range bundle.Payload.Evidence {
		ref := ev.EvidenceClass + ":" + ev.ID
		evidenceRefs = append(evidenceRefs, ref)
		byClass[ev.EvidenceClass] = append(byClass[ev.EvidenceClass], ref)
		evidenceModule[ref] = ev.Module
	}
	// Pick evidence that answers for the component being asked about. Taking the
	// first of a class was fine while nothing compared modules; now that the
	// validator does, it would make this command record an immutable
	// attestation the engine itself rejects — and `--apply` reports success
	// before verification ever runs.
	pickScoped := func(classes []string, components []string) string {
		for _, class := range classes {
			for _, ref := range byClass[class] {
				if len(components) == 0 {
					return ref
				}
				for _, component := range components {
					if moduleMatchesTarget(evidenceModule[ref], component) {
						return ref
					}
				}
			}
		}
		return ""
	}
	sort.Strings(evidenceRefs)

	judgmentGoverned, _ := BundleGovernedBy(bundle, "explicit-judgment")
	pending := []ReviewAttestationPendency{}
	criteria := make([]ReviewCriterion, 0, len(bundle.Payload.Plan.Criteria))
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if !criterion.Required {
			continue
		}
		if judgmentGoverned && ReviewCriterionKind(criterion) == ReviewCriterionKindJudgment {
			// The criterion no check reports on. Collecting evidence for it is
			// still useful and still happens — it is sealed in the bundle and
			// the reviewer cites it. What stops here is answering on the
			// reviewer's behalf.
			reason := "no registered check reports on this criterion"
			if len(criterion.EvidenceClasses) > 0 {
				reason = "the profile asks a reviewer to conclude, even though evidence of class " + strings.Join(criterion.EvidenceClasses, "|") + " exists"
			}
			pending = append(pending, ReviewAttestationPendency{Criterion: criterion.ID, Kind: ReviewCriterionKindJudgment, Reason: reason})
			continue
		}
		critEvidence := pickScoped(criterion.EvidenceClasses, reviewCriterionComponents(bundle.Payload.Plan, criterion))
		if critEvidence == "" && len(criterion.EvidenceClasses) == 0 && len(bundle.Payload.Evidence) > 0 {
			// A criterion that asks for no particular class is satisfied by any
			// sealed evidence. Citing the first is weak, but it is real and in
			// the bundle, which is the property that was missing.
			first := bundle.Payload.Evidence[0]
			critEvidence = first.EvidenceClass + ":" + first.ID
		}
		if critEvidence == "" {
			// This used to invent a reference — `<class>:auto-attest`, or the
			// first unrelated ref in the bundle, or `docs:auto-attest`. The
			// invented ref pointed at nothing, and nothing downstream checked
			// it, so the command recorded a passed criterion that no evidence
			// supported.
			//
			// Where the scope is expected to carry validation evidence, the
			// honest answer is to refuse: the check exists and was not run, and
			// a reviewer must run it or disposition the criterion themselves
			// with a reason. Where it is not — a documentation-only spec with no
			// delivery target, the case the same distinction already exempts
			// above — the criterion is recorded not-applicable, naming what is
			// missing. Either way the absence is visible instead of dressed as
			// a pass.
			if requiresEvidence {
				if len(criterion.EvidenceClasses) > 0 {
					return ReviewAttestation{}, nil, fmt.Errorf("pose: criterion %s requires evidence class %s and bundle %s seals none; run the check, or record the criterion as not-applicable with a rationale using `pose review attest`", criterion.ID, strings.Join(criterion.EvidenceClasses, "|"), bundleID)
				}
				return ReviewAttestation{}, nil, fmt.Errorf("pose: criterion %s has no evidence in bundle %s; run a check, or record the criterion as not-applicable with a rationale using `pose review attest`", criterion.ID, bundleID)
			}
			rationale := "the scope carries no delivery target, so no validation evidence is collected for it"
			if len(criterion.EvidenceClasses) > 0 {
				rationale = "the scope carries no delivery target, so no evidence of class " + strings.Join(criterion.EvidenceClasses, "|") + " is collected for it"
			}
			criteria = append(criteria, ReviewCriterion{
				ID:          criterion.ID,
				Disposition: "not-applicable",
				Rationale:   rationale,
			})
			continue
		}
		evidenceRefs = append(evidenceRefs, critEvidence)
		criteria = append(criteria, ReviewCriterion{
			ID:          criterion.ID,
			Disposition: "passed",
			Evidence:    critEvidence,
		})
	}
	evidenceRefs = uniqueSorted(evidenceRefs)

	tools := make([]ReviewToolDisposition, 0, len(bundle.Payload.Plan.Tools))
	for _, tool := range bundle.Payload.Plan.Tools {
		disposition := ReviewToolDisposition{
			ID:        tool.ID,
			Component: tool.Component,
		}
		if containsFold(tool.Preconditions, "review-complete") {
			disposition.Disposition = "deferred"
			disposition.Rationale = "post-review gate"
		} else if tool.ProducerCoverage == "none" {
			// A fact the engine computed from the matrix, not a judgment: this
			// component declares that it runs no check, so nothing can produce
			// what the tool asks for. Preparing it here is what keeps the
			// reviewer from having to cite another component's result.
			disposition.Disposition = "not-used"
			disposition.Rationale = "the validation matrix declares this component runs no check, so no producer can emit evidence of class " + strings.Join(tool.EvidenceClasses, "|")
		} else if tool.Requiredness == "recommended" {
			disposition.Disposition = "not-used"
			disposition.Rationale = "not used during automated attestation"
		} else {
			toolEv := ""
			toolComponents := []string{}
			if tool.Component != "" {
				toolComponents = append(toolComponents, tool.Component)
			}
			toolEv = pickScoped(tool.EvidenceClasses, toolComponents)
			if toolEv == "" && len(tool.EvidenceClasses) == 0 {
				// A tool that declares no evidence class does not report a
				// validation result at all — `artifact-check` and `review-check`
				// are POSE commands, and what supports their disposition is that
				// the command ran. Naming the tool is a real reference to that;
				// reaching for an unrelated sealed result, as this used to, said
				// nothing about whether the tool ran.
				toolEv = "check:" + tool.ID
			}
			if toolEv == "" {
				// The same invention the criteria path stopped doing one spec
				// ago, on the other half of the attestation: `validation:auto-
				// attest`, `<class>:auto-attest`, `docs:auto-attest` — none of
				// them pointing at anything, and nothing downstream checking.
				// The distinction between a judged review and a stamped one
				// does not survive one half of it being fabricated.
				if requiresEvidence {
					if len(tool.EvidenceClasses) > 0 {
						return ReviewAttestation{}, nil, fmt.Errorf("pose: review tool %s requires evidence class %s and bundle %s seals none; run the tool, or record its disposition with `pose review attest --tool`", tool.ID, strings.Join(tool.EvidenceClasses, "|"), bundleID)
					}
					return ReviewAttestation{}, nil, fmt.Errorf("pose: review tool %s has no evidence in bundle %s; run the tool, or record its disposition with `pose review attest --tool`", tool.ID, bundleID)
				}
				// `deferred`, not `not-used`: a required tool recorded not-used
				// is a blocker whatever the reason, while a deferral is the
				// disposition the engine already accepts for a tool whose
				// precondition this scope cannot meet.
				disposition.Disposition = "deferred"
				disposition.Rationale = "the scope carries no delivery target, so no evidence this tool reports is collected for it"
				tools = append(tools, disposition)
				continue
			}
			disposition.Disposition = "passed"
			disposition.Evidence = toolEv
		}
		tools = append(tools, disposition)
	}

	decision := "approved"
	if len(pending) > 0 {
		// A preparation that still owes answers is not an approval waiting to
		// be written. Naming the decision `changes-requested` keeps the object
		// honest if anything persists it anyway: the verifier refuses it, which
		// is the correct outcome for a review nobody finished.
		decision = "changes-requested"
	}
	att := ReviewAttestation{
		BundleID:     bundle.BundleID,
		BundleDigest: bundle.BundleDigest,
		Reviewer:     reviewer,
		Decision:     decision,
		Criteria:     criteria,
		Tools:        tools,
		EvidenceRefs: evidenceRefs,
		Findings:     []ReviewFinding{},
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].Criterion < pending[j].Criterion })
	return att, pending, nil
}

// RecordReviewAttestation appends an immutable decision for one sealed bundle.
func (s Store) RecordReviewAttestation(att ReviewAttestation, now time.Time) (ReviewAttestation, error) {
	return s.recordReviewAttestation(att, now, false)
}

func (s Store) recordReviewAttestation(att ReviewAttestation, now time.Time, signed bool) (ReviewAttestation, error) {
	bundle, err := s.LoadReviewBundle(att.BundleID)
	if err != nil {
		return ReviewAttestation{}, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewAttestation{}, err
	}
	if policy.RequireSignedAttestations && !signed {
		return ReviewAttestation{}, fmt.Errorf("pose: review policy requires a trusted signed attestation envelope")
	}
	if att.BundleDigest == "" {
		att.BundleDigest = bundle.BundleDigest
	}
	if att.BundleDigest != bundle.BundleDigest {
		return ReviewAttestation{}, fmt.Errorf("pose: attestation bundle digest mismatch")
	}
	if att.Reviewer == "" || strings.ContainsAny(att.Reviewer, "\r\n") || (!strings.HasPrefix(att.Reviewer, "agent:") && !strings.HasPrefix(att.Reviewer, "human:")) {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid reviewer execution identity")
	}
	if att.Decision != "approved" && att.Decision != "approved-with-reservations" && att.Decision != "changes-requested" && att.Decision != "rejected" {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid review decision %q", att.Decision)
	}
	att.SchemaVersion = ReviewBundleSchemaVersion
	if att.AttestedAt == "" {
		att.AttestedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	} else if _, err := time.Parse(time.RFC3339, att.AttestedAt); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: attested_at must be RFC3339")
	}
	att.BundleID = bundle.BundleID
	att.BundleDigest = bundle.BundleDigest
	expectedAttestationID := reviewAttestationID(att)
	if att.AttestationID == "" {
		att.AttestationID = expectedAttestationID
	} else if att.AttestationID != expectedAttestationID {
		return ReviewAttestation{}, fmt.Errorf("pose: attestation id does not match its content digest")
	}
	if !strings.HasPrefix(att.AttestationID, "rva-") || len(att.AttestationID) != 20 {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid attestation id")
	}
	att.Path = filepath.ToSlash(filepath.Join(".pose", "review-attestations", att.AttestationID+".json"))
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-attestations")), true)
	if err != nil {
		return ReviewAttestation{}, err
	}
	path := filepath.Join(dir, att.AttestationID+".json")
	if existing, loadErr := s.LoadReviewAttestation(att.AttestationID); loadErr == nil {
		existingRaw, _ := json.Marshal(existing)
		candidateRaw, _ := json.Marshal(att)
		if bytes.Equal(existingRaw, candidateRaw) {
			return existing, nil
		}
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation identity collision for %s", att.AttestationID)
	}
	raw, err := json.MarshalIndent(att, "", "  ")
	if err != nil {
		return ReviewAttestation{}, err
	}
	if err := writeImmutableJSON(path, append(raw, '\n')); err != nil {
		return ReviewAttestation{}, err
	}
	return att, nil
}

// VerifyReviewAttestationEnvelope verifies an optional provider-neutral
// Ed25519 envelope. Trust is pinned as <issuer>#sha256:<public-key-digest> in
// review policy; a self-declared issuer or public key is never trusted alone.
func (s Store) VerifyReviewAttestationEnvelope(envelope ReviewAttestationEnvelope) (ReviewAttestation, error) {
	if envelope.SchemaVersion != ReviewBundleSchemaVersion || envelope.Algorithm != "ed25519" {
		return ReviewAttestation{}, fmt.Errorf("pose: unsupported review attestation envelope")
	}
	if envelope.Subject != envelope.Attestation.BundleID || envelope.Issuer == "" || strings.ContainsAny(envelope.Issuer, "\r\n") {
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation envelope subject or issuer mismatch")
	}
	publicKey, err := base64.StdEncoding.DecodeString(envelope.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid Ed25519 public key")
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid Ed25519 signature")
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewAttestation{}, err
	}
	pin := envelope.Issuer + "#" + digestBytes(publicKey)
	trusted := false
	for _, candidate := range policy.TrustedAttestationIssuers {
		if candidate == pin {
			trusted = true
			break
		}
	}
	if !trusted {
		return ReviewAttestation{}, fmt.Errorf("pose: untrusted review attestation issuer or public key")
	}
	attestation := envelope.Attestation
	attestation.Path = ""
	attestation.Envelope = nil
	raw, err := json.Marshal(attestation)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), raw, signature) {
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation signature verification failed")
	}
	bundle, err := s.LoadReviewBundle(attestation.BundleID)
	if err != nil {
		return ReviewAttestation{}, err
	}
	if attestation.BundleDigest != bundle.BundleDigest {
		return ReviewAttestation{}, fmt.Errorf("pose: signed attestation does not bind the exact bundle")
	}
	attestation.Envelope = &ReviewAttestationSignature{Issuer: envelope.Issuer, Subject: envelope.Subject, Algorithm: envelope.Algorithm, PublicKey: envelope.PublicKey, Signature: envelope.Signature}
	return attestation, nil
}

func (s Store) ImportReviewAttestationEnvelope(rel string, apply bool) (ReviewAttestation, error) {
	if err := ValidateArtifactPath(s.Root, rel, false); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid attestation envelope path: %w", err)
	}
	clean, _ := validateArtifactPathSyntax(rel)
	var envelope ReviewAttestationEnvelope
	if err := strictJSONFile(filepath.Join(s.Root, clean), &envelope); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: reading review attestation envelope: %w", err)
	}
	attestation, err := s.VerifyReviewAttestationEnvelope(envelope)
	if err != nil || !apply {
		return attestation, err
	}
	return s.recordReviewAttestation(attestation, time.Now(), true)
}

func (s Store) LoadReviewAttestation(id string) (ReviewAttestation, error) {
	if !strings.HasPrefix(id, "rva-") || len(id) != 20 {
		return ReviewAttestation{}, fmt.Errorf("pose: invalid review attestation id %q", id)
	}
	var att ReviewAttestation
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-attestations")), false)
	if err != nil {
		return ReviewAttestation{}, err
	}
	path := filepath.Join(dir, id+".json")
	if err := strictJSONFile(path, &att); err != nil {
		return ReviewAttestation{}, fmt.Errorf("pose: reading review attestation %s: %w", id, err)
	}
	if att.SchemaVersion != ReviewBundleSchemaVersion || att.AttestationID != id {
		return ReviewAttestation{}, fmt.Errorf("pose: malformed review attestation %s", id)
	}
	if reviewAttestationID(att) != id {
		return ReviewAttestation{}, fmt.Errorf("pose: review attestation %s content digest mismatch", id)
	}
	att.Path = filepath.ToSlash(filepath.Join(".pose", "review-attestations", id+".json"))
	return att, nil
}

func (s Store) ListReviewAttestations(bundleID string) ([]ReviewAttestation, error) {
	dir, err := ensureReviewArtifactDir(s.Root, filepath.ToSlash(filepath.Join(".pose", "review-attestations")), false)
	if os.IsNotExist(err) {
		return []ReviewAttestation{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ReviewAttestation{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []ReviewAttestation{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if bundleID != "" {
			raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			var header struct {
				BundleID string `json:"bundle_id"`
			}
			if err := json.Unmarshal(raw, &header); err != nil || header.BundleID != bundleID {
				continue
			}
		}
		att, loadErr := s.LoadReviewAttestation(id)
		if loadErr != nil {
			if bundleID != "" {
				return nil, loadErr
			}
			continue
		}
		if bundleID == "" || att.BundleID == bundleID {
			result = append(result, att)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].AttestedAt == result[j].AttestedAt {
			return result[i].AttestationID < result[j].AttestationID
		}
		return result[i].AttestedAt < result[j].AttestedAt
	})
	return result, nil
}

func (s Store) VerifyReviewBundle(scope string) (ReviewBundleVerification, error) {
	policy, err := s.GetReviewPolicy()
	if err != nil {
		return ReviewBundleVerification{}, err
	}
	scopeRef, scopeErr := ParseScopeRef(scope)
	if scopeErr != nil {
		return ReviewBundleVerification{}, scopeErr
	}

	bundles, err := s.ListReviewBundles(scope)
	if err != nil {
		return ReviewBundleVerification{}, err
	}

	if (policy.SchemaVersion < ReviewPolicySchemaVersion || !policy.ReviewBundles) && len(bundles) == 0 {
		eval, evalErr := s.ReviewCheck(scope)
		if evalErr != nil {
			return ReviewBundleVerification{}, evalErr
		}
		verification := ReviewBundleVerification{
			Scope:    scope,
			State:    "needs-review",
			Fresh:    eval.Fresh,
			Approved: eval.Approved,
			Warnings: append([]string{}, eval.Warnings...),
			Blockers: append([]string{}, eval.Blockers...),
		}
		if eval.Current != nil {
			verification.NextAction = "record fresh review for " + scope
			if eval.Current.Decision == "changes-requested" || eval.Current.Decision == "rejected" {
				verification.State = eval.Current.Decision
			} else if !eval.Fresh {
				verification.State = "superseded"
			}
		} else {
			verification.NextAction = "record review for " + scope
		}
		if eval.Approved && eval.Fresh {
			done, doneErr := s.scopeLifecycleDone(scopeRef)
			if doneErr == nil && done {
				verification.State = "closed"
				verification.NextAction = "scope is closed with an approved review"
			} else {
				verification.State = "ready-to-close"
				verification.NextAction = "apply the guarded lifecycle transition for " + scope
			}
		}
		verification.Blockers = uniqueSorted(verification.Blockers)
		verification.Warnings = uniqueSorted(verification.Warnings)
		return verification, nil
	}

	prepared, err := s.PrepareReviewBundle(scope)
	if err != nil {
		return ReviewBundleVerification{}, err
	}
	verification := ReviewBundleVerification{Scope: scope, State: "needs-validation", Fresh: false, Approved: false, Warnings: append([]string{}, prepared.Warnings...), Blockers: append([]string{}, prepared.Blockers...)}
	if len(prepared.Blockers) == 0 {
		verification.State = "ready-to-seal"
	}
	var current *ReviewBundle
	for i := len(bundles) - 1; i >= 0; i-- {
		if bundles[i].BundleDigest == prepared.BundleDigest {
			bundle := bundles[i]
			current = &bundle
			break
		}
	}
	if current == nil {
		if len(bundles) > 0 {
			previous := bundles[len(bundles)-1]
			delta := ReviewBundleDiff(previous, prepared)
			if attempts, listErr := s.ListReviewAttestations(previous.BundleID); listErr == nil && len(attempts) > 0 {
				for _, finding := range attempts[len(attempts)-1].Findings {
					delta.ChangedFindings = append(delta.ChangedFindings, finding.ID)
				}
				delta.ChangedFindings = uniqueSorted(delta.ChangedFindings)
			}
			verification.Delta = &delta
			verification.State = "superseded"
			verification.Blockers = append(verification.Blockers, "current semantic inputs are not represented by a sealed bundle")
		}
		verification.NextAction = "seal the current review bundle for " + scope
		verification.Blockers = uniqueSorted(verification.Blockers)
		return verification, nil
	}
	verification.Bundle = current
	verification.Fresh = true
	verification.State = "ready-for-review"
	attestations, err := s.ListReviewAttestations(current.BundleID)
	if err != nil {
		return verification, err
	}
	if len(attestations) == 0 {
		verification.Blockers = append(verification.Blockers, "sealed review bundle has no attestation")
		verification.NextAction = "review and attest bundle " + current.BundleID
		verification.Blockers = uniqueSorted(verification.Blockers)
		return verification, nil
	}
	att := attestations[len(attestations)-1]
	verification.Attestation = &att
	verification.Blockers = append(verification.Blockers, s.validateBundleAttestation(*current, att)...)
	if len(verification.Blockers) == 0 {
		verification.Approved = true
		scopeRef, scopeErr := ParseScopeRef(scope)
		done, doneErr := false, scopeErr
		if scopeErr == nil {
			done, doneErr = s.scopeLifecycleDone(scopeRef)
		}
		if doneErr == nil && done {
			verification.State = "closed"
			verification.NextAction = "scope is closed with a fresh bundle attestation"
		} else {
			verification.State = "ready-to-close"
			verification.NextAction = "apply the guarded lifecycle transition for " + scope
		}
	} else if att.Decision == "changes-requested" || att.Decision == "rejected" {
		verification.State = "changes-requested"
		verification.NextAction = "remediate findings and seal a superseding bundle"
	} else {
		verification.NextAction = "replace the invalid attestation for " + current.BundleID
	}
	verification.Blockers = uniqueSorted(verification.Blockers)
	return verification, nil
}

// validateBundleAttestation reports why an attestation does not stand for its
// bundle. skipEvidenceSupport waives only the checks introduced by
// pose-attestation-evidence-must-be-in-the-bundle, for a completed scope whose
// approval predates them; everything else still applies. See
// evidenceVocabularyLegacyExempt for why that exemption exists and its bounds.
func (s Store) validateBundleAttestation(bundle ReviewBundle, att ReviewAttestation) []string {
	return s.validateBundleAttestationWith(bundle, att, false)
}

func (s Store) validateBundleAttestationWith(bundle ReviewBundle, att ReviewAttestation, skipEvidenceSupport bool) []string {
	blockers := []string{}
	policy, _, policyErr := s.loadReviewPolicy()
	if policyErr != nil {
		blockers = append(blockers, policyErr.Error())
	}
	if att.BundleDigest != bundle.BundleDigest || att.BundleID != bundle.BundleID {
		blockers = append(blockers, "attestation does not reference the exact sealed bundle")
	}
	// The one gate that stays live, deliberately. Everything else a bundle is
	// judged by is sealed with it, because a setting flipped today must not
	// re-judge a review recorded years ago. Signing is the opposite case: a
	// project that starts requiring signed attestations is raising a bar, and a
	// bundle sealed before that must not be permanently exempt from it — the
	// exemption would be exactly the work an attacker or a hurry would want
	// (spec pose-reuse-is-sealed-signing-stays-live).
	if policy.RequireSignedAttestations || att.Envelope != nil {
		if err := s.verifyStoredReviewAttestationSignature(att); err != nil {
			blockers = append(blockers, err.Error())
		}
	}
	if _, err := time.Parse(time.RFC3339, att.AttestedAt); err != nil {
		blockers = append(blockers, "attested_at must be RFC3339")
	}
	if !strings.HasPrefix(att.Reviewer, "agent:") && !strings.HasPrefix(att.Reviewer, "human:") {
		blockers = append(blockers, "reviewer execution identity is malformed")
	}
	// Two axes, kept apart. Independence says what separation the review
	// requires; assurance says whether the reviewer's identity is read from the
	// string they wrote or from an authority an authorized issuer signed.
	//
	// Under `declared` this is the prefix check it always was, and the warning
	// beside it stops the enum being read as proof: `agent:independent-anything`
	// satisfies `different-actor`, and `human:` satisfies `mandatory-human`,
	// because nothing here compares a principal, a session or a grant
	// (spec pose-abm-review-authority).
	switch bundle.Payload.SealedGates().IdentityAssurance {
	case ReviewIdentityAssuranceVerified:
		blockers = append(blockers, s.verifiedAuthorityBlockers(bundle, att)...)
	default:
		switch bundle.Payload.Plan.Independence {
		case "different-actor":
			if !strings.HasPrefix(att.Reviewer, "agent:independent-") && !strings.HasPrefix(att.Reviewer, "human:") {
				blockers = append(blockers, "review policy requires an independent reviewer identity")
			}
		case "mandatory-human":
			if !strings.HasPrefix(att.Reviewer, "human:") {
				blockers = append(blockers, "review policy requires human approval")
			}
		}
	}
	required := map[string]ReviewPlanCriterion{}
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if criterion.Required {
			required[criterion.ID] = criterion
		}
	}
	// The findings this attestation records, so a criterion disposed as
	// `finding` can be checked against them. The CLI parser already refused a
	// criterion naming a finding it does not record; the Store did not, so any
	// other caller — a Store client, a signed import, a reuse — could approve a
	// criterion that explicitly did not pass while filing nothing. A signature
	// authenticates bytes; it does not repair a reference to something that
	// does not exist (spec pose-abm-review-soundness).
	recordedFindings := map[string]bool{}
	for _, finding := range att.Findings {
		recordedFindings[finding.ID] = true
	}
	sealedClasses := map[string]bool{}
	for _, ev := range bundle.Payload.Evidence {
		sealedClasses[ev.EvidenceClass] = true
	}
	judgmentGoverned, _ := BundleGovernedBy(bundle, "explicit-judgment")
	if structuralGoverned, _ := BundleGovernedBy(bundle, "structural-causality"); structuralGoverned {
		blockers = append(blockers, s.reviewStructuralCausalityBlockers(bundle.Payload.Scope.Ref, bundle.Payload.Plan.Structure, required, att.Criteria)...)
	}
	toolScope, _ := ParseScopeRef(bundle.Payload.Scope.Ref)
	toolGraph, _ := s.GetDeliveryIntegrity("")
	hasDeliveryTarget := s.reviewScopeRequiresValidationEvidence(toolScope, bundle.Payload.Plan, toolGraph)
	seen := map[string]bool{}
	notApplicable := 0
	for _, criterion := range att.Criteria {
		if seen[criterion.ID] {
			blockers = append(blockers, "duplicate criterion "+criterion.ID)
		}
		seen[criterion.ID] = true
		planned, known := required[criterion.ID]
		if !known {
			blockers = append(blockers, "unknown criterion "+criterion.ID)
		}
		if criterion.Disposition != "passed" && criterion.Disposition != "not-applicable" && criterion.Disposition != "finding" {
			blockers = append(blockers, "criterion "+criterion.ID+" has invalid disposition")
		}
		if criterion.Disposition == "not-applicable" {
			notApplicable++
			if criterion.Rationale == "" {
				blockers = append(blockers, "criterion "+criterion.ID+" lacks not-applicable rationale")
			}
			// Inapplicability and absence are different states, and only one of
			// them is an answer. A criterion that asks for a class the bundle
			// actually seals is applicable by construction: dispensing with it
			// would be reading missing judgment as missing relevance.
			//
			// Gated, unlike the finding-reference rule beside it. That one is
			// referential integrity — a criterion naming a finding nobody filed
			// is incomplete under any policy, and no stored record does it. This
			// one is a stricter judgment default, and 12 of the 499 attestations
			// in the two repositories would fail it. Holding a review to a rule
			// that did not exist when it was given is what the sealed-contract
			// mechanism exists to prevent.
			if known && judgmentGoverned {
				for _, class := range planned.EvidenceClasses {
					if sealedClasses[class] {
						blockers = append(blockers, "criterion "+criterion.ID+" is not-applicable while the bundle seals evidence of class "+class+"; missing judgment is a pendency, not inapplicability")
						break
					}
				}
			}
		}
		if criterion.Disposition == "finding" {
			if criterion.Evidence == "" {
				blockers = append(blockers, "criterion "+criterion.ID+" is disposed as a finding and names none")
			} else if !recordedFindings[criterion.Evidence] {
				blockers = append(blockers, "criterion "+criterion.ID+" names finding "+criterion.Evidence+", which this attestation does not record")
			}
		}
		if criterion.Disposition == "passed" {
			if !skipEvidenceSupport {
				blockers = append(blockers, reviewCriterionEvidenceBlockers(bundle, planned, criterion)...)
			}
			// A judged criterion passes on a conclusion, not on a reference.
			// Gated on the contract the bundle sealed, so the 470 attestations
			// recorded before it keep their verdict and stay readable: this
			// raises the bar for reviews taken under the contract, and never
			// re-judges one taken before it existed.
			if judgmentGoverned && known && ReviewCriterionKind(planned) == ReviewCriterionKindJudgment && strings.TrimSpace(criterion.Rationale) == "" {
				blockers = append(blockers, "criterion "+criterion.ID+" is a judgment criterion passed with no conclusion; record what was examined and what it concluded")
			}
		}
	}
	for id := range required {
		if !seen[id] {
			blockers = append(blockers, "missing criterion "+id)
		}
	}
	// Every criterion dispensed with is not a review of anything. The engine
	// cannot judge whether one dispensation is honest, but it can refuse the
	// degenerate case, where the attestation states that nothing the plan asked
	// about applies to the change it approves.
	//
	// Bounded to a scope that carries a delivery target, because the other case
	// is already modelled and already honest: a documentation-only spec collects
	// no validation evidence by design, and recording every criterion
	// not-applicable — naming what is missing — is the answer the engine itself
	// prepares for it. Refusing that would break the distinction this contract
	// depends on rather than reinforce it.
	if judgmentGoverned && hasDeliveryTarget && len(required) > 0 && notApplicable == len(required) && len(att.Criteria) == len(required) {
		blockers = append(blockers, "every required criterion is not-applicable; a scope where the whole plan is inapplicable is not reviewed by it")
	}
	reused := map[string]bool{}
	for _, reuse := range att.ReusedFrom {
		if reused[reuse.Criterion] {
			blockers = append(blockers, "duplicate reused criterion "+reuse.Criterion)
			continue
		}
		reused[reuse.Criterion] = true
		if !bundle.Payload.SealedGates().AllowCriterionReuse {
			blockers = append(blockers, "criterion reuse is not permitted by the bundle's sealed gates")
			continue
		}
		current, ok := required[reuse.Criterion]
		if !ok {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" is not required by the current bundle")
			continue
		}
		currentDigest := reviewCriterionInputDigest(bundle, current)
		if currentDigest != reuse.InputDigest {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" input digest changed")
			continue
		}
		prior, err := s.LoadReviewAttestation(reuse.FromAttestation)
		if err != nil {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" references an unavailable attestation")
			continue
		}
		priorBundle, err := s.LoadReviewBundle(prior.BundleID)
		if err != nil {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" references an unavailable bundle")
			continue
		}
		priorContract := ""
		for _, criterion := range priorBundle.Payload.Plan.Criteria {
			if criterion.ID == reuse.Criterion {
				priorContract = reviewCriterionInputDigest(priorBundle, criterion)
				break
			}
		}
		priorPassed := false
		for _, criterion := range prior.Criteria {
			if criterion.ID == reuse.Criterion && criterion.Disposition == "passed" {
				priorPassed = true
				break
			}
		}
		if priorContract != reuse.InputDigest || !priorPassed {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+" is not unchanged and passed in the referenced attestation")
		}
	}
	sealedEvidence := map[string]ReviewBundleEvidence{}
	for _, ev := range bundle.Payload.Evidence {
		sealedEvidence[ev.EvidenceClass+":"+ev.ID] = ev
	}
	if skipEvidenceSupport {
		sealedEvidence = nil
	}
	toolWarnings, toolBlockers := evaluateReviewToolCoverage(s.Root, bundle.Payload.Plan.Tools, att.Tools, sealedEvidence, hasDeliveryTarget)
	_ = toolWarnings
	blockers = append(blockers, toolBlockers...)
	// The finding contract, which this path did not have. It accepted a
	// `critical` risk with no owner, no rationale and no review date; a
	// disposition the engine does not know; and a finding with neither severity
	// nor action — all three with no blocker at all, while the legacy attempt
	// path refused every one of them. A project that adopted review bundles
	// silently lost the gate (spec
	// pose-bundle-findings-take-the-contract-the-legacy-path-had).
	//
	// Only the accepted-risk severities and the reservations flag come from the
	// sealed gates. The rest is not configuration: a finding without a severity
	// is incomplete under any policy.
	allowedRisk := map[string]bool{}
	for _, severity := range bundle.Payload.SealedGates().AcceptedRiskSeverities {
		allowedRisk[severity] = true
	}
	seenFindings := map[string]bool{}
	for _, finding := range att.Findings {
		if seenFindings[finding.ID] {
			blockers = append(blockers, "duplicate finding "+finding.ID)
		}
		seenFindings[finding.ID] = true
		if finding.Severity == "" || finding.Action == "" {
			blockers = append(blockers, "finding "+finding.ID+" lacks severity or action")
		}
		switch finding.Disposition {
		case "resolved", "wont-fix":
		case "accepted-risk":
			if !allowedRisk[finding.Severity] || finding.Owner == "" || finding.Rationale == "" || finding.ReviewBy == "" {
				blockers = append(blockers, "finding "+finding.ID+" has unapproved or incomplete accepted risk")
			}
		case "open", "changes-requested":
			blockers = append(blockers, "finding "+finding.ID+" is "+finding.Disposition)
		default:
			blockers = append(blockers, "finding "+finding.ID+" has invalid disposition")
		}
	}
	if att.Decision != "approved" && !(att.Decision == "approved-with-reservations" && bundle.Payload.SealedGates().AllowApprovedWithReservations) {
		blockers = append(blockers, "review decision does not permit closeout: "+att.Decision)
	}
	return blockers
}

func (s Store) verifyStoredReviewAttestationSignature(att ReviewAttestation) error {
	if att.Envelope == nil {
		return fmt.Errorf("signed attestation proof is required by review policy")
	}
	proof := *att.Envelope
	unsigned := att
	unsigned.Path = ""
	unsigned.Envelope = nil
	_, err := s.VerifyReviewAttestationEnvelope(ReviewAttestationEnvelope{SchemaVersion: ReviewBundleSchemaVersion, Issuer: proof.Issuer, Subject: proof.Subject, Algorithm: proof.Algorithm, PublicKey: proof.PublicKey, Signature: proof.Signature, Attestation: unsigned})
	return err
}

// reviewCriterionEvidenceBlockers checks that a criterion recorded as passed is
// supported by evidence the sealed bundle actually contains, of a class the
// criterion asks for.
//
// Nothing checked this before. A `passed` disposition could cite evidence
// absent from the bundle, evidence of the wrong class, or nothing at all, and
// the attestation verified — which is how three closeouts in an adopting
// repository were approved against a bundle carrying zero evidence, and how a
// criterion requiring `integration` passed on a `go vet` result. An attestation
// that cannot be checked against its own subject is a signature on an empty
// page.
//
// Only `passed` is constrained. `not-applicable` already requires a rationale,
// which is the reviewer's judgement standing in for evidence, and `finding`
// records a problem rather than a clearance.
func reviewCriterionEvidenceBlockers(bundle ReviewBundle, planned ReviewPlanCriterion, attested ReviewCriterion) []string {
	if attested.Evidence == "" {
		return []string{"criterion " + attested.ID + " is passed with no evidence"}
	}
	sealed := map[string]ReviewBundleEvidence{}
	for _, ev := range bundle.Payload.Evidence {
		sealed[ev.EvidenceClass+":"+ev.ID] = ev
	}
	evidence, ok := sealed[attested.Evidence]
	if !ok {
		return []string{"criterion " + attested.ID + " cites evidence absent from the sealed bundle: " + attested.Evidence}
	}
	if len(planned.EvidenceClasses) > 0 {
		matched := false
		for _, want := range planned.EvidenceClasses {
			if want == evidence.EvidenceClass {
				matched = true
				break
			}
		}
		if !matched {
			return []string{"criterion " + attested.ID + " requires evidence class " + strings.Join(planned.EvidenceClasses, "|") + " but cites " + evidence.EvidenceClass + ": " + attested.Evidence}
		}
	}
	// Only a criterion that demands a class is scoped. Without one the plan made
	// no claim about what the evidence shows, so narrowing it by module would
	// invent a constraint the plan never stated — and auto-attest deliberately
	// takes any sealed evidence for such a criterion, so scoping it here would
	// make the engine reject its own output. The spec said this in its
	// non-goals and the first implementation did it anyway.
	if len(planned.EvidenceClasses) == 0 {
		return nil
	}
	if components := reviewCriterionComponents(bundle.Payload.Plan, planned); len(components) > 0 {
		for _, component := range components {
			if moduleMatchesTarget(evidence.Module, component) {
				return nil
			}
		}
		return []string{"criterion " + attested.ID + " is scoped to " + strings.Join(components, "|") + " but cites evidence from " + evidenceModuleLabel(evidence) + ": " + attested.Evidence}
	}
	return nil
}

// reviewCriterionComponents resolves the components a criterion answers for, or
// nil when it answers for all of them.
//
// A criterion comes from one or more profiles, and the plan records which
// components each profile was selected for. An overlay matched specific
// components; a base profile carries none and governs every one. So a criterion
// governed by any base profile has no component constraint, and one governed
// only by overlays is constrained to what they matched.
//
// Without this, evidence was matched by reference and class alone: a backend
// criterion could be satisfied by a frontend sibling's integration result, in a
// bundle that seals both. The reference was real, the class was demanded, and
// the result said nothing about the component the criterion is about.
func reviewCriterionComponents(plan ReviewBundlePlan, criterion ReviewPlanCriterion) []string {
	selected := map[string][]string{}
	for _, profile := range plan.SelectedProfiles {
		selected[profile.Ref] = profile.Components
	}
	components := []string{}
	for _, ref := range criterion.Profiles {
		scoped, known := selected[ref]
		if !known {
			// A profile the plan does not record cannot be scoped, and guessing
			// would narrow a criterion on no evidence.
			return nil
		}
		if len(scoped) == 0 {
			return nil
		}
		components = append(components, scoped...)
	}
	return uniqueSorted(components)
}

func evidenceModuleLabel(evidence ReviewBundleEvidence) string {
	if evidence.Module == "" {
		return "the repository root"
	}
	return evidence.Module
}

func reviewCriterionInputDigest(bundle ReviewBundle, criterion ReviewPlanCriterion) string {
	evidence := []ReviewBundleEvidence{}
	classes := map[string]bool{}
	for _, class := range criterion.EvidenceClasses {
		classes[class] = true
	}
	for _, item := range bundle.Payload.Evidence {
		if classes[item.EvidenceClass] {
			evidence = append(evidence, item)
		}
	}
	inputs := []ReviewBundleInput{}
	for _, input := range bundle.Payload.ConsumedInputs {
		if input.Kind == "schema" || input.Kind == "governed-config" {
			inputs = append(inputs, input)
			continue
		}
		if (input.Kind == "review-plan" || input.Kind == "tool-catalog") && reviewCriterionSubjectSensitive(criterion) {
			inputs = append(inputs, input)
			continue
		}
		if input.Kind == "rule" {
			for _, rule := range criterion.Rules {
				if input.Path == ".pose/rules/"+rule+".md" {
					inputs = append(inputs, input)
				}
			}
		}
		if input.Kind == "review-profile" {
			for _, profile := range criterion.Profiles {
				if input.Path == ".pose/review-profiles/"+strings.Split(profile, "@")[0]+".json" {
					inputs = append(inputs, input)
				}
			}
		}
	}
	subjectSensitive := reviewCriterionSubjectSensitive(criterion)
	relevantTools := []ReviewPlanTool{}
	for _, tool := range bundle.Payload.Plan.Tools {
		for _, criterionID := range tool.Criteria {
			if criterionID == criterion.ID {
				relevantTools = append(relevantTools, tool)
				break
			}
		}
	}
	contract := struct {
		Criterion    ReviewPlanCriterion        `json:"criterion"`
		Independence string                     `json:"independence"`
		Scope        []ReviewBundleInput        `json:"scope,omitempty"`
		Subject      []ReviewBundleSubjectEntry `json:"subject,omitempty"`
		PatchDigest  string                     `json:"patch_digest,omitempty"`
		TreeDigest   string                     `json:"tree_digest,omitempty"`
		Evidence     []ReviewBundleEvidence     `json:"evidence,omitempty"`
		Inputs       []ReviewBundleInput        `json:"inputs"`
		Tools        []ReviewPlanTool           `json:"tools,omitempty"`
	}{Criterion: criterion, Independence: bundle.Payload.Plan.Independence, Evidence: evidence, Inputs: sortedBundleInputs(inputs), Tools: relevantTools}
	if subjectSensitive {
		contract.Scope = append([]ReviewBundleInput{}, bundle.Payload.Scope.Sections...)
		contract.Subject = append([]ReviewBundleSubjectEntry{}, bundle.Payload.Subject.Entries...)
		contract.PatchDigest = bundle.Payload.Subject.PatchDigest
		contract.TreeDigest = bundle.Payload.Subject.TreeDigest
	} else {
		contract.Scope = append([]ReviewBundleInput{}, bundle.Payload.Scope.Sections...)
		for _, entry := range bundle.Payload.Subject.Entries {
			// `removed` is the one class whose category is unknown: the path
			// carries no governed classification and there is no content left to
			// read, so it cannot be shown to be irrelevant to this criterion the
			// way an unrelated implementation file can. Omitting it would leave
			// the digest unchanged when a superseding bundle adds the deletion,
			// and a passed criterion would be reused over a subject it never saw.
			if entry.Class == "documentation" || entry.Class == "governance" || entry.Class == "removed" {
				contract.Subject = append(contract.Subject, entry)
			}
		}
	}
	digest, _ := digestJSON(contract)
	return digest
}

func reviewAttestationID(att ReviewAttestation) string {
	identity := att
	identity.AttestationID = ""
	identity.Path = ""
	identity.Envelope = nil
	digest, _ := digestJSON(identity)
	return "rva-" + strings.TrimPrefix(digest, "sha256:")[:16]
}

func reviewCriterionSubjectSensitive(criterion ReviewPlanCriterion) bool {
	if len(criterion.EvidenceClasses) > 0 || len(criterion.Rules) == 0 {
		return true
	}
	for _, rule := range criterion.Rules {
		if rule != "documentation-style" && rule != "knowledge-governance" {
			return true
		}
	}
	return false
}

func ReviewBundleDiff(from, to ReviewBundle) ReviewBundleDelta {
	delta := ReviewBundleDelta{FromBundle: from.BundleID, ToBundle: to.BundleID}
	fromComponents := map[string]string{}
	toComponents := map[string]string{}
	for _, component := range from.Payload.Plan.Components {
		raw, _ := json.Marshal(component)
		fromComponents[component.ID] = digestBytes(raw)
	}
	for _, component := range to.Payload.Plan.Components {
		raw, _ := json.Marshal(component)
		toComponents[component.ID] = digestBytes(raw)
	}
	delta.ChangedComponents = changedReviewBundleKeys(fromComponents, toComponents)
	fromSections := map[string]string{}
	toSections := map[string]string{}
	for _, input := range from.Payload.Scope.Sections {
		fromSections[input.Path] = input.Digest
	}
	for _, input := range to.Payload.Scope.Sections {
		toSections[input.Path] = input.Digest
	}
	delta.ChangedSections = changedReviewBundleKeys(fromSections, toSections)
	fromPaths := map[string]string{}
	toPaths := map[string]string{}
	pathNames := map[string]string{}
	for _, entry := range from.Payload.Subject.Entries {
		key := entry.Action + "\x00" + entry.Path + "\x00" + entry.NewPath
		fromPaths[key] = entry.Digest
		pathNames[key] = reviewBundleEntryPath(entry)
	}
	for _, entry := range to.Payload.Subject.Entries {
		key := entry.Action + "\x00" + entry.Path + "\x00" + entry.NewPath
		toPaths[key] = entry.Digest
		pathNames[key] = reviewBundleEntryPath(entry)
	}
	for _, key := range changedReviewBundleKeys(fromPaths, toPaths) {
		delta.ChangedPaths = append(delta.ChangedPaths, pathNames[key])
	}
	fromCriteria := map[string]string{}
	toCriteria := map[string]string{}
	for _, criterion := range from.Payload.Plan.Criteria {
		fromCriteria[criterion.ID] = reviewCriterionInputDigest(from, criterion)
	}
	for _, criterion := range to.Payload.Plan.Criteria {
		toCriteria[criterion.ID] = reviewCriterionInputDigest(to, criterion)
		if fromCriteria[criterion.ID] == toCriteria[criterion.ID] {
			delta.ReusableCriteria = append(delta.ReusableCriteria, criterion.ID)
		}
	}
	delta.ChangedCriteria = changedReviewBundleKeys(fromCriteria, toCriteria)
	fromEvidence := map[string]string{}
	toEvidence := map[string]string{}
	evidenceClasses := map[string][]string{}
	for _, evidence := range from.Payload.Evidence {
		raw, _ := json.Marshal(evidence)
		fromEvidence[evidence.ID] = digestBytes(raw)
		evidenceClasses[evidence.ID] = append(evidenceClasses[evidence.ID], evidence.EvidenceClass)
	}
	for _, evidence := range to.Payload.Evidence {
		raw, _ := json.Marshal(evidence)
		toEvidence[evidence.ID] = digestBytes(raw)
		evidenceClasses[evidence.ID] = append(evidenceClasses[evidence.ID], evidence.EvidenceClass)
	}
	delta.ChangedEvidence = changedReviewBundleKeys(fromEvidence, toEvidence)
	for _, id := range delta.ChangedEvidence {
		delta.ChangedEvidenceClasses = append(delta.ChangedEvidenceClasses, evidenceClasses[id]...)
	}
	delta.ChangedComponents = uniqueSorted(delta.ChangedComponents)
	delta.ChangedSections = uniqueSorted(delta.ChangedSections)
	delta.ChangedPaths = uniqueSorted(delta.ChangedPaths)
	delta.ChangedCriteria = uniqueSorted(delta.ChangedCriteria)
	delta.ChangedEvidence = uniqueSorted(delta.ChangedEvidence)
	delta.ChangedEvidenceClasses = uniqueSorted(delta.ChangedEvidenceClasses)
	delta.ReusableCriteria = uniqueSorted(delta.ReusableCriteria)
	return delta
}

func changedReviewBundleKeys(from, to map[string]string) []string {
	keys := map[string]bool{}
	for key, value := range from {
		if to[key] != value {
			keys[key] = true
		}
	}
	for key, value := range to {
		if from[key] != value {
			keys[key] = true
		}
	}
	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	return uniqueSorted(result)
}

func reviewBundleEntryPath(entry ReviewBundleSubjectEntry) string {
	if entry.NewPath != "" {
		return entry.NewPath
	}
	return entry.Path
}

// verifiedAuthorityBlockers holds an attestation to a signed authority claim
// instead of to the string the reviewer wrote.
//
// What it proves is bounded and worth stating exactly: that an issuer this
// repository pinned, holding the grant for the role being claimed, signed a
// statement binding these principals and executions to this bundle, addressed
// to this project, still current. It does not prove the issuer told the truth,
// and it does not prove two principals think differently. Independence of
// execution and identity is what the contract can carry; cognitive
// independence is not, and the engine must not be read as claiming it.
func (s Store) verifiedAuthorityBlockers(bundle ReviewBundle, att ReviewAttestation) []string {
	blockers := []string{}
	claim := att.Authority
	if claim == nil {
		return append(blockers, "review policy requires verified identity assurance and the attestation carries no authority claim; a reviewer prefix is a declaration, not a proof")
	}
	if att.Envelope == nil {
		return append(blockers, "an authority claim is only as good as its signature and this attestation carries none")
	}
	policy, _, policyErr := s.loadReviewPolicy()
	if policyErr != nil {
		return append(blockers, policyErr.Error())
	}
	if claim.SchemaVersion != ReviewSchemaVersion {
		blockers = append(blockers, fmt.Sprintf("the authority claim has unsupported schema version %d", claim.SchemaVersion))
	}
	if claim.BundleDigest != bundle.BundleDigest {
		blockers = append(blockers, "the authority claim binds a different bundle than the one attested")
	}
	if claim.Issuer != att.Envelope.Issuer {
		blockers = append(blockers, "the authority claim names an issuer other than the one that signed the attestation")
	}
	if claim.Principal == "" || claim.Principal != att.Reviewer {
		blockers = append(blockers, "the authority claim principal does not match the attestation reviewer")
	}
	// Replay from another project names that project here. An instance that
	// requires verified authority and does not say who it is cannot check that,
	// so the configuration is the blocker rather than the claim.
	switch {
	case policy.AuthorityAudience == "":
		blockers = append(blockers, "review policy requires verified identity assurance but declares no authority_audience, so a claim issued for any project would satisfy it")
	default:
		if claim.Audience == "" {
			blockers = append(blockers, "the authority claim names no audience, so it is satisfied by any project that replays it")
		} else if claim.Audience != policy.AuthorityAudience {
			blockers = append(blockers, "the authority claim is addressed to "+claim.Audience+" and this project answers to "+policy.AuthorityAudience)
		}
		if claim.Project == "" {
			blockers = append(blockers, "the authority claim names no project")
		} else if claim.Project != policy.AuthorityAudience {
			blockers = append(blockers, "the authority claim names project "+claim.Project+" and this project answers to "+policy.AuthorityAudience)
		}
	}
	if _, err := time.Parse(time.RFC3339, claim.IssuedAt); err != nil {
		blockers = append(blockers, "the authority claim has no valid issued_at")
	}
	if claim.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, claim.ExpiresAt)
		switch {
		case err != nil:
			blockers = append(blockers, "the authority claim has an unparseable expires_at")
		case expiry.Before(time.Now().UTC()):
			blockers = append(blockers, "the authority claim expired at "+claim.ExpiresAt)
		}
	}
	if claim.ReviewExecution == "" {
		blockers = append(blockers, "the authority claim must name the reviewing execution")
	}
	switch claim.Role {
	case "agent":
		if !strings.HasPrefix(claim.Principal, "agent:") {
			blockers = append(blockers, "an agent authority claim must name an agent principal")
		}
	case "human":
		if !strings.HasPrefix(claim.Principal, "human:") {
			blockers = append(blockers, "a human authority claim must name a human principal")
		}
		// Signing an attestation and vouching for a person being a person are
		// different authorities, so asserting a human role needs its own grant.
		if !reviewIssuerHoldsGrant(policy.HumanAuthorityIssuers, att.Envelope.Issuer, att.Envelope.PublicKey) {
			blockers = append(blockers, "issuer "+att.Envelope.Issuer+" is trusted to sign attestations but is not authorised to assert a human principal")
		}
	default:
		blockers = append(blockers, "the authority claim names an unknown role "+claim.Role)
	}
	// The three values are ordered, and the checks have to be ordered with them.
	//
	// `mandatory-human` used to assert only the role, so the strongest value in
	// the enum verified *less* separation than the one below it: the same human
	// principal could implement and review in one run and satisfy it, while
	// `different-actor` refused exactly that. Under `declared` the inconsistency
	// was moot, since nothing was verified either way. Under `verified` it was a
	// hole, because there the engine does claim to have checked separation
	// (spec pose-abm-review-authority).
	switch bundle.Payload.Plan.Independence {
	case "same-actor-separate-execution":
		if claim.ImplementationPrincipal == "" || claim.ImplementationExecution == "" {
			blockers = append(blockers, "review policy requires the implementation principal and execution in the authority claim")
		} else if claim.Principal != claim.ImplementationPrincipal {
			blockers = append(blockers, "review policy requires the same actor and the authority claim names different principals")
		} else if claim.ReviewExecution == claim.ImplementationExecution {
			blockers = append(blockers, "review policy requires a separate execution and the claim names the implementation's own run")
		}
	case "different-actor", "mandatory-human":
		if claim.ImplementationPrincipal == "" || claim.ImplementationExecution == "" {
			blockers = append(blockers, "review policy requires a different actor and the claim does not say who implemented")
		} else if claim.Principal == claim.ImplementationPrincipal {
			blockers = append(blockers, "review policy requires a different actor and the same principal implemented and reviewed")
		} else if claim.ReviewExecution == claim.ImplementationExecution {
			blockers = append(blockers, "review policy requires a separate review execution and the claim names the implementation's own run")
		}
		if bundle.Payload.Plan.Independence == "mandatory-human" && claim.Role != "human" {
			blockers = append(blockers, "review policy requires human approval and the claim asserts role "+claim.Role)
		}
	}
	return blockers
}

// reviewIssuerHoldsGrant compares against the same `<issuer>#sha256:<digest>`
// pin form the attestation trust list uses, so a grant follows the key rather
// than the name: rotating a key revokes the grant until the new pin is added.
func reviewIssuerHoldsGrant(grants []string, issuer, publicKey string) bool {
	raw, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return false
	}
	pin := issuer + "#" + digestBytes(raw)
	for _, grant := range grants {
		if grant == pin {
			return true
		}
	}
	return false
}

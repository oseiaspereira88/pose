package pose

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const ReviewSchemaVersion = 1
const ReviewPolicySchemaVersion = 2

// ScopeRef is the canonical address of a reviewable POSE scope.
type ScopeRef struct {
	Kind      string `json:"kind"`
	Slug      string `json:"slug,omitempty"`
	Roadmap   string `json:"roadmap,omitempty"`
	Milestone string `json:"milestone,omitempty"`
}

func (r ScopeRef) String() string {
	if r.Kind == "milestone" {
		return "milestone:" + r.Roadmap + "/" + r.Milestone
	}
	return r.Kind + ":" + r.Slug
}

func ParseScopeRef(value string) (ScopeRef, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return ScopeRef{}, fmt.Errorf("pose: invalid scope ref %q", value)
	}
	switch parts[0] {
	case "spec", "roadmap":
		if err := ValidateSlug(parts[1]); err != nil {
			return ScopeRef{}, fmt.Errorf("pose: invalid %s scope: %w", parts[0], err)
		}
		return ScopeRef{Kind: parts[0], Slug: parts[1]}, nil
	case "milestone":
		pair := strings.Split(parts[1], "/")
		if len(pair) != 2 || ValidateSlug(pair[0]) != nil || ValidateSlug(pair[1]) != nil {
			return ScopeRef{}, fmt.Errorf("pose: invalid milestone scope %q", value)
		}
		return ScopeRef{Kind: "milestone", Roadmap: pair[0], Milestone: pair[1]}, nil
	default:
		return ScopeRef{}, fmt.Errorf("pose: unsupported scope kind %q", parts[0])
	}
}

type ReviewCriterionProfile struct {
	ID              string   `json:"id"`
	Description     string   `json:"description"`
	Rules           []string `json:"rules,omitempty"`
	EvidenceClasses []string `json:"evidence_classes,omitempty"`
	Required        *bool    `json:"required,omitempty"`
}

type ReviewProfileSelectors struct {
	Languages     []string `json:"languages,omitempty"`
	Domains       []string `json:"domains,omitempty"`
	ComponentIDs  []string `json:"component_ids,omitempty"`
	DeliveryKinds []string `json:"delivery_kinds,omitempty"`
	Criticalities []string `json:"criticalities,omitempty"`
}

type ReviewProfileTool struct {
	ID              string   `json:"id"`
	Requiredness    string   `json:"requiredness,omitempty"`
	EvidenceClasses []string `json:"evidence_classes,omitempty"`
	Criteria        []string `json:"criteria,omitempty"`
	Preconditions   []string `json:"preconditions,omitempty"`
}

type ReviewProfile struct {
	SchemaVersion int                      `json:"schema_version"`
	ID            string                   `json:"id"`
	Version       int                      `json:"version"`
	Scope         string                   `json:"scope"`
	Criteria      []ReviewCriterionProfile `json:"criteria"`
	Selectors     ReviewProfileSelectors   `json:"selectors,omitempty"`
	Tools         []ReviewProfileTool      `json:"tools,omitempty"`
	Independence  string                   `json:"independence,omitempty"`
}

func (p ReviewProfile) Ref() string { return fmt.Sprintf("%s@%d", p.ID, p.Version) }

type ReviewPolicy struct {
	SchemaVersion                    int               `json:"schema_version"`
	Enabled                          bool              `json:"enabled"`
	AdoptedAt                        string            `json:"adopted_at,omitempty"`
	Profiles                         map[string]string `json:"profiles"`
	AllowApprovedWithReservations    bool              `json:"allow_approved_with_reservations,omitempty"`
	AcceptedRiskSeverities           []string          `json:"accepted_risk_severities,omitempty"`
	ReviewerIndependence             map[string]string `json:"reviewer_independence,omitempty"`
	ContinuousCloseout               bool              `json:"continuous_closeout,omitempty"`
	AllowInScopeRemediationSpec      bool              `json:"allow_in_scope_remediation_spec,omitempty"`
	RequireReviewForLegacyDoneScopes bool              `json:"require_review_for_legacy_done_scopes,omitempty"`
	ComponentAware                   bool              `json:"component_aware,omitempty"`
	ComponentAwareAdoptedAt          string            `json:"component_aware_adopted_at,omitempty"`
	UnmappedComponentBehavior        string            `json:"unmapped_component_behavior,omitempty"`
	OverlayProfiles                  []string          `json:"overlay_profiles,omitempty"`
	ReviewBundles                    bool              `json:"review_bundles,omitempty"`
	ReviewBundlesAdoptedAt           string            `json:"review_bundles_adopted_at,omitempty"`
	AllowCriterionReuse              bool              `json:"allow_criterion_reuse,omitempty"`
	RequireSignedAttestations        bool              `json:"require_signed_attestations,omitempty"`
	TrustedAttestationIssuers        []string          `json:"trusted_attestation_issuers,omitempty"`
}

type ReviewCriterion struct {
	ID          string `json:"id"`
	Disposition string `json:"disposition"`
	Evidence    string `json:"evidence,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

type ReviewFinding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Evidence    string `json:"evidence,omitempty"`
	Action      string `json:"action,omitempty"`
	Disposition string `json:"disposition"`
	Owner       string `json:"owner,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
	ReviewBy    string `json:"review_by,omitempty"`
}

type ReviewToolDisposition struct {
	ID          string `json:"id"`
	Component   string `json:"component,omitempty"`
	Disposition string `json:"disposition"`
	Evidence    string `json:"evidence,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

type ReviewAttempt struct {
	SchemaVersion int                     `json:"schema_version"`
	ReviewID      string                  `json:"review_id"`
	Scope         string                  `json:"scope"`
	ScopeDigest   string                  `json:"scope_digest"`
	PlanDigest    string                  `json:"plan_digest,omitempty"`
	Profile       string                  `json:"profile"`
	Reviewer      string                  `json:"reviewer"`
	Decision      string                  `json:"decision"`
	ReviewedAt    string                  `json:"reviewed_at"`
	Supersedes    string                  `json:"supersedes,omitempty"`
	EvidenceRefs  []string                `json:"evidence_refs,omitempty"`
	Criteria      []ReviewCriterion       `json:"criteria"`
	Tools         []ReviewToolDisposition `json:"tools,omitempty"`
	Findings      []ReviewFinding         `json:"findings"`
	Path          string                  `json:"path,omitempty"`
}

type ReviewEvaluation struct {
	Required      bool           `json:"required"`
	Scope         string         `json:"scope"`
	ScopeDigest   string         `json:"scope_digest"`
	PlanDigest    string         `json:"plan_digest,omitempty"`
	Profile       string         `json:"profile,omitempty"`
	Current       *ReviewAttempt `json:"current,omitempty"`
	Fresh         bool           `json:"fresh"`
	Approved      bool           `json:"approved"`
	Blockers      []string       `json:"blockers"`
	Warnings      []string       `json:"warnings,omitempty"`
	PolicyEnabled bool           `json:"policy_enabled"`
	BundleID      string         `json:"bundle_id,omitempty"`
	BundleDigest  string         `json:"bundle_digest,omitempty"`
	AttestationID string         `json:"attestation_id,omitempty"`
	BundleState   string         `json:"bundle_state,omitempty"`
}

type CloseoutState struct {
	SchemaVersion int              `json:"schema_version"`
	Scope         string           `json:"scope"`
	ScopeDigest   string           `json:"scope_digest"`
	Review        ReviewEvaluation `json:"review"`
	Children      []CloseoutState  `json:"children,omitempty"`
	LifecycleDone bool             `json:"lifecycle_done"`
	Terminal      bool             `json:"terminal"`
	NextAction    string           `json:"next_action"`
	Blockers      []string         `json:"blockers"`
}

var reviewLineRE = regexp.MustCompile(`^-\s+([A-Za-z0-9._-]+)\s+\[([^]]+)\](?:\s+(.*))?$`)

func (s Store) loadReviewPolicy() (ReviewPolicy, []byte, error) {
	path := filepath.Join(s.Root, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ReviewPolicy{}, nil, nil
	}
	if err != nil {
		return ReviewPolicy{}, nil, fmt.Errorf("pose: reading review policy: %w", err)
	}
	var p ReviewPolicy
	if err := json.Unmarshal(raw, &p); err != nil {
		return ReviewPolicy{}, nil, fmt.Errorf("pose: invalid review policy: %w", err)
	}
	if p.SchemaVersion == ReviewPolicySchemaVersion {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&p); err != nil {
			return ReviewPolicy{}, nil, fmt.Errorf("pose: invalid schema-v2 review policy: %w", err)
		}
	}
	if p.SchemaVersion != ReviewSchemaVersion && p.SchemaVersion != ReviewPolicySchemaVersion {
		return ReviewPolicy{}, nil, fmt.Errorf("pose: unsupported review policy schema %d", p.SchemaVersion)
	}
	if p.SchemaVersion == ReviewPolicySchemaVersion {
		if p.UnmappedComponentBehavior == "" {
			p.UnmappedComponentBehavior = "warning"
		}
		if p.UnmappedComponentBehavior != "warning" && p.UnmappedComponentBehavior != "blocker" {
			return ReviewPolicy{}, nil, fmt.Errorf("pose: invalid unmapped component behavior %q", p.UnmappedComponentBehavior)
		}
		if p.ComponentAware {
			if _, err := time.Parse(time.DateOnly, p.ComponentAwareAdoptedAt); err != nil {
				return ReviewPolicy{}, nil, fmt.Errorf("pose: component_aware_adopted_at must be YYYY-MM-DD when component-aware review is enabled")
			}
		}
		if p.ReviewBundles {
			if _, err := time.Parse(time.DateOnly, p.ReviewBundlesAdoptedAt); err != nil {
				return ReviewPolicy{}, nil, fmt.Errorf("pose: review_bundles_adopted_at must be YYYY-MM-DD when review bundles are enabled")
			}
		}
		for _, issuer := range p.TrustedAttestationIssuers {
			if strings.TrimSpace(issuer) == "" || strings.ContainsAny(issuer, "\r\n") {
				return ReviewPolicy{}, nil, fmt.Errorf("pose: invalid trusted attestation issuer")
			}
		}
	}
	for scope, independence := range p.ReviewerIndependence {
		if !validReviewIndependence(independence) {
			return ReviewPolicy{}, nil, fmt.Errorf("pose: invalid reviewer independence %q for %s", independence, scope)
		}
	}
	return p, raw, nil
}

// GetReviewPolicy exposes the validated provider-neutral policy to command
// frontends without allowing them to bypass its gates.
func (s Store) GetReviewPolicy() (ReviewPolicy, error) {
	policy, _, err := s.loadReviewPolicy()
	return policy, err
}

func (s Store) loadReviewProfile(ref string) (ReviewProfile, []byte, error) {
	parts := strings.Split(ref, "@")
	if len(parts) != 2 || ValidateSlug(parts[0]) != nil {
		return ReviewProfile{}, nil, fmt.Errorf("pose: invalid review profile ref %q", ref)
	}
	path := filepath.Join(s.Root, ".pose", "review-profiles", parts[0]+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ReviewProfile{}, nil, fmt.Errorf("pose: reading review profile %q: %w", ref, err)
	}
	var p ReviewProfile
	if err := json.Unmarshal(raw, &p); err != nil {
		return ReviewProfile{}, nil, fmt.Errorf("pose: invalid review profile %q: %w", ref, err)
	}
	if p.SchemaVersion == ReviewPolicySchemaVersion {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&p); err != nil {
			return ReviewProfile{}, nil, fmt.Errorf("pose: invalid schema-v2 review profile %q: %w", ref, err)
		}
	}
	if (p.SchemaVersion != ReviewSchemaVersion && p.SchemaVersion != ReviewPolicySchemaVersion) || p.Ref() != ref || p.Scope == "" || len(p.Criteria) == 0 {
		return ReviewProfile{}, nil, fmt.Errorf("pose: malformed review profile %q", ref)
	}
	seen := map[string]bool{}
	for _, c := range p.Criteria {
		if !slugPattern.MatchString(c.ID) || seen[c.ID] {
			return ReviewProfile{}, nil, fmt.Errorf("pose: invalid or duplicate criterion %q in %s", c.ID, ref)
		}
		// Closed rule/evidence catalogs are a schema-v2 contract. Schema-v1
		// profiles keep their historical namespaces so repositories that never
		// opted into component-aware planning remain readable (spec R29).
		if p.SchemaVersion == ReviewPolicySchemaVersion {
			if err := s.validateReviewContractRefs(ref, c.Rules, c.EvidenceClasses); err != nil {
				return ReviewProfile{}, nil, err
			}
		}
		seen[c.ID] = true
	}
	if p.SchemaVersion == ReviewSchemaVersion && (hasReviewSelectors(p.Selectors) || len(p.Tools) > 0 || p.Independence != "") {
		return ReviewProfile{}, nil, fmt.Errorf("pose: schema-v1 review profile %q cannot declare selectors, tools or independence", ref)
	}
	if p.Independence != "" && !validReviewIndependence(p.Independence) {
		return ReviewProfile{}, nil, fmt.Errorf("pose: invalid reviewer independence %q in %s", p.Independence, ref)
	}
	for _, selector := range append(append(append(append([]string{}, p.Selectors.Languages...), p.Selectors.Domains...), p.Selectors.DeliveryKinds...), p.Selectors.Criticalities...) {
		if !slugPattern.MatchString(selector) {
			return ReviewProfile{}, nil, fmt.Errorf("pose: invalid review selector %q in %s", selector, ref)
		}
	}
	for _, component := range p.Selectors.ComponentIDs {
		if !slugPattern.MatchString(component) {
			if _, err := validateArtifactPathSyntax(component); err != nil {
				return ReviewProfile{}, nil, fmt.Errorf("pose: invalid component selector %q in %s", component, ref)
			}
		}
	}
	for _, tool := range p.Tools {
		if !slugPattern.MatchString(tool.ID) || (tool.Requiredness != "" && tool.Requiredness != "recommended" && tool.Requiredness != "required") {
			return ReviewProfile{}, nil, fmt.Errorf("pose: invalid review tool %q in %s", tool.ID, ref)
		}
		if err := s.validateReviewContractRefs(ref, nil, tool.EvidenceClasses); err != nil {
			return ReviewProfile{}, nil, err
		}
		for _, precondition := range tool.Preconditions {
			if !reviewPreconditionCatalog[precondition] {
				return ReviewProfile{}, nil, fmt.Errorf("pose: unknown review tool precondition %q in %s", precondition, ref)
			}
		}
	}
	return p, raw, nil
}

func (s Store) validateReviewContractRefs(ref string, rules, evidenceClasses []string) error {
	for _, rule := range rules {
		if !slugPattern.MatchString(rule) {
			return fmt.Errorf("pose: invalid review rule %q in %s", rule, ref)
		}
		path := filepath.Join(s.Root, ".pose", "rules", rule+".md")
		if err := ValidateArtifactPath(s.Root, filepath.ToSlash(filepath.Join(".pose", "rules", rule+".md")), false); err != nil {
			return fmt.Errorf("pose: unsafe review rule %q in %s: %w", rule, ref, err)
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("pose: unknown review rule %q in %s", rule, ref)
		}
	}
	for _, class := range evidenceClasses {
		if !reviewEvidenceClassCatalog[class] {
			return fmt.Errorf("pose: unknown review evidence class %q in %s", class, ref)
		}
	}
	return nil
}

func hasReviewSelectors(selectors ReviewProfileSelectors) bool {
	return len(selectors.Languages)+len(selectors.Domains)+len(selectors.ComponentIDs)+len(selectors.DeliveryKinds)+len(selectors.Criticalities) > 0
}

func validReviewIndependence(value string) bool {
	return value == "same-actor-separate-execution" || value == "different-actor" || value == "mandatory-human"
}

// ReviewProfileForScope returns the profile selected by policy for a typed
// scope. It is used by mutation-capable frontends to scaffold a complete
// immutable attempt without duplicating policy parsing.
func (s Store) ReviewProfileForScope(ref string) (ReviewProfile, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return ReviewProfile{}, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewProfile{}, err
	}
	if !policy.Enabled {
		return ReviewProfile{}, fmt.Errorf("pose: review policy is absent or disabled")
	}
	profileRef := policy.Profiles[scope.Kind]
	if profileRef == "" {
		return ReviewProfile{}, fmt.Errorf("pose: no review profile configured for %s", scope.Kind)
	}
	profile, _, err := s.loadReviewProfile(profileRef)
	if err == nil && profile.Scope != scope.Kind {
		return ReviewProfile{}, fmt.Errorf("pose: profile %s cannot review %s scopes", profile.Ref(), scope.Kind)
	}
	return profile, err
}

func parseReviewAttempt(path string) (ReviewAttempt, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ReviewAttempt{}, err
	}
	fm, body := SplitFrontmatter(string(raw))
	a := ReviewAttempt{SchemaVersion: ReviewSchemaVersion, ReviewID: fm["review_id"], Scope: fm["scope"], ScopeDigest: fm["scope_digest"], PlanDigest: fm["plan_digest"], Profile: fm["profile"], Reviewer: fm["reviewer"], Decision: fm["decision"], ReviewedAt: fm["reviewed_at"], Supersedes: fm["supersedes"], Path: path}
	if v := fm["schema_version"]; v != "" && v != "1" {
		return a, fmt.Errorf("pose: review %s has unsupported schema_version %s", a.ReviewID, v)
	}
	if refs := strings.TrimSpace(fm["evidence_refs"]); refs != "" {
		a.EvidenceRefs = splitListValue(refs)
	}
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch line {
		case "## Criteria":
			section = "criteria"
			continue
		case "## Tools":
			section = "tools"
			continue
		case "## Findings":
			section = "findings"
			continue
		}
		match := reviewLineRE.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		id, disposition, rest := match[1], strings.ToLower(match[2]), strings.TrimSpace(match[3])
		if section == "criteria" {
			a.Criteria = append(a.Criteria, ReviewCriterion{ID: id, Disposition: disposition, Evidence: fieldValue(rest, "evidence"), Rationale: fieldValue(rest, "rationale")})
		} else if section == "tools" {
			component := fieldValue(rest, "component")
			if decoded, decodeErr := url.QueryUnescape(component); decodeErr == nil {
				component = decoded
			}
			a.Tools = append(a.Tools, ReviewToolDisposition{ID: id, Component: component, Disposition: disposition, Evidence: fieldValue(rest, "evidence"), Rationale: fieldValue(rest, "rationale")})
		} else if section == "findings" {
			a.Findings = append(a.Findings, ReviewFinding{ID: id, Disposition: disposition, Severity: fieldValue(rest, "severity"), Evidence: fieldValue(rest, "evidence"), Action: fieldValue(rest, "action"), Owner: fieldValue(rest, "owner"), Rationale: fieldValue(rest, "rationale"), ReviewBy: fieldValue(rest, "review_by")})
		}
	}
	if err := scanner.Err(); err != nil {
		return a, err
	}
	return a, nil
}

func splitListValue(value string) []string {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(strings.Trim(p, `"'`)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fieldValue(rest, key string) string {
	prefix := key + ":"
	for _, part := range strings.Fields(rest) {
		if strings.HasPrefix(part, prefix) {
			return strings.TrimPrefix(part, prefix)
		}
	}
	return ""
}

func (s Store) ListReviewAttempts(scope string) ([]ReviewAttempt, error) {
	dir := filepath.Join(s.Root, ".pose", "reviews")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []ReviewAttempt{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pose: reading reviews: %w", err)
	}
	attempts := []ReviewAttempt{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		a, err := parseReviewAttempt(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if scope == "" || a.Scope == scope {
			attempts = append(attempts, a)
		}
	}
	sort.Slice(attempts, func(i, j int) bool {
		if attempts[i].ReviewedAt == attempts[j].ReviewedAt {
			return attempts[i].ReviewID < attempts[j].ReviewID
		}
		return attempts[i].ReviewedAt < attempts[j].ReviewedAt
	})
	return attempts, nil
}

func digestJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s Store) ScopeDigest(ref string) (string, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return "", err
	}
	policy, policyRaw, err := s.loadReviewPolicy()
	if err != nil {
		return "", err
	}
	profileRef := policy.Profiles[scope.Kind]
	var profileRaw []byte
	if profileRef != "" {
		_, profileRaw, err = s.loadReviewProfile(profileRef)
		if err != nil {
			return "", err
		}
	}
	base := map[string]any{"schema_version": ReviewSchemaVersion, "scope": ref, "policy": string(policyRaw), "profile": string(profileRaw)}
	switch scope.Kind {
	case "spec":
		sp, err := s.GetSpec(scope.Slug)
		if err != nil {
			return "", err
		}
		base["content"] = strings.ReplaceAll(strings.TrimSpace(sp.Body), "\r\n", "\n")
		base["depends_on"] = sp.DependsOn
		base["components"] = sp.Components
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return "", err
		}
		var target *Milestone
		for i := range rm.Milestones {
			if rm.Milestones[i].ID == scope.Milestone {
				target = &rm.Milestones[i]
				break
			}
		}
		if target == nil {
			return "", fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
		}
		children := map[string]string{}
		for _, slug := range target.Specs {
			d, err := s.ScopeDigest("spec:" + slug)
			if err != nil {
				return "", err
			}
			children[slug] = d
		}
		base["milestone"] = target
		base["section"] = roadmapMilestoneSection(rm.Body, target.ID)
		base["children"] = children
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return "", err
		}
		children := map[string]string{}
		for _, milestone := range rm.Milestones {
			d, err := s.ScopeDigest("milestone:" + rm.Slug + "/" + milestone.ID)
			if err != nil {
				return "", err
			}
			children[milestone.ID] = d
		}
		base["content"] = strings.ReplaceAll(strings.TrimSpace(rm.Body), "\r\n", "\n")
		base["children"] = children
	}
	return digestJSON(base)
}

func roadmapMilestoneSection(body, id string) string {
	start := "## Milestone: " + id
	idx := strings.Index(body, start)
	if idx < 0 {
		return ""
	}
	rest := body[idx:]
	if next := strings.Index(rest[len(start):], "\n## Milestone: "); next >= 0 {
		return strings.TrimSpace(rest[:len(start)+next])
	}
	return strings.TrimSpace(rest)
}

func (s Store) ReviewCheck(ref string) (ReviewEvaluation, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return ReviewEvaluation{}, err
	}
	digest, err := s.ScopeDigest(ref)
	if err != nil {
		return ReviewEvaluation{}, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewEvaluation{}, err
	}
	eval := ReviewEvaluation{Scope: ref, ScopeDigest: digest, PolicyEnabled: policy.Enabled, Required: policy.Enabled, Fresh: false, Approved: false, Blockers: []string{}}
	if !policy.Enabled {
		eval.Warnings = append(eval.Warnings, "review policy is absent or disabled; closeout review is not enforced")
		return eval, nil
	}
	profileRef := policy.Profiles[scope.Kind]
	eval.Profile = profileRef
	required, err := s.reviewRequiredForScope(scope, policy)
	if err != nil {
		return eval, err
	}
	eval.Required = required
	if !required {
		eval.Warnings = append(eval.Warnings, "review is not required for a legacy done scope created before policy adoption")
		return eval, nil
	}
	if profileRef == "" {
		eval.Blockers = append(eval.Blockers, "review policy has no profile for "+scope.Kind)
		return eval, nil
	}
	if policy.SchemaVersion >= ReviewPolicySchemaVersion && policy.ReviewBundles {
		done, _ := s.scopeLifecycleDone(scope)
		verification, verifyErr := s.VerifyReviewBundle(ref)
		if verifyErr == nil && verification.Approved {
			eval.Fresh = verification.Fresh
			eval.Approved = verification.Approved
			eval.Blockers = append(eval.Blockers, verification.Blockers...)
			eval.Warnings = append(eval.Warnings, verification.Warnings...)
			eval.BundleState = verification.State
			if verification.Bundle != nil {
				eval.ScopeDigest = verification.Bundle.BundleDigest
				eval.BundleID = verification.Bundle.BundleID
				eval.BundleDigest = verification.Bundle.BundleDigest
				eval.PlanDigest = verification.Bundle.Payload.Plan.PlanDigest
			}
			if verification.Attestation != nil {
				eval.AttestationID = verification.Attestation.AttestationID
				att := verification.Attestation
				eval.Current = &ReviewAttempt{SchemaVersion: ReviewBundleSchemaVersion, ReviewID: att.AttestationID, Scope: ref, ScopeDigest: att.BundleDigest, PlanDigest: eval.PlanDigest, Profile: profileRef, Reviewer: att.Reviewer, Decision: att.Decision, ReviewedAt: att.AttestedAt, Supersedes: att.Supersedes, EvidenceRefs: att.EvidenceRefs, Criteria: att.Criteria, Tools: att.Tools, Findings: att.Findings, Path: att.Path}
			}
			eval.Blockers = uniqueSorted(eval.Blockers)
			eval.Warnings = uniqueSorted(eval.Warnings)
			return eval, nil
		}
		if done && s.reviewBundlesLegacyAttemptExempt(scope, policy) {
			// Check if this done scope has an earlier approved bundle attestation
			bundles, listErr := s.ListReviewBundles(ref)
			if listErr == nil && len(bundles) > 0 {
				for i := len(bundles) - 1; i >= 0; i-- {
					attestations, attErr := s.ListReviewAttestations(bundles[i].BundleID)
					if attErr == nil && len(attestations) > 0 {
						att := attestations[len(attestations)-1]
						if len(s.validateBundleAttestation(bundles[i], att)) == 0 {
							eval.Fresh = true
							eval.Approved = true
							eval.BundleState = "closed"
							eval.ScopeDigest = bundles[i].BundleDigest
							eval.BundleID = bundles[i].BundleID
							eval.BundleDigest = bundles[i].BundleDigest
							eval.PlanDigest = bundles[i].Payload.Plan.PlanDigest
							eval.AttestationID = att.AttestationID
							eval.Current = &ReviewAttempt{SchemaVersion: ReviewBundleSchemaVersion, ReviewID: att.AttestationID, Scope: ref, ScopeDigest: att.BundleDigest, PlanDigest: eval.PlanDigest, Profile: profileRef, Reviewer: att.Reviewer, Decision: att.Decision, ReviewedAt: att.AttestedAt, Supersedes: att.Supersedes, EvidenceRefs: att.EvidenceRefs, Criteria: att.Criteria, Tools: att.Tools, Findings: att.Findings, Path: att.Path}
							eval.Warnings = append(eval.Warnings, "completed scope retains its approved sealed review bundle attestation")
							return eval, nil
						}
					}
				}
			}
			// Fall through to legacy check
		} else if verifyErr != nil {
			return eval, verifyErr
		} else {
			eval.Fresh = verification.Fresh
			eval.Approved = verification.Approved
			eval.Blockers = append(eval.Blockers, verification.Blockers...)
			eval.Warnings = append(eval.Warnings, verification.Warnings...)
			eval.BundleState = verification.State
			if verification.Bundle != nil {
				eval.ScopeDigest = verification.Bundle.BundleDigest
				eval.BundleID = verification.Bundle.BundleID
				eval.BundleDigest = verification.Bundle.BundleDigest
				eval.PlanDigest = verification.Bundle.Payload.Plan.PlanDigest
			}
			if verification.Attestation != nil {
				eval.AttestationID = verification.Attestation.AttestationID
				att := verification.Attestation
				eval.Current = &ReviewAttempt{SchemaVersion: ReviewBundleSchemaVersion, ReviewID: att.AttestationID, Scope: ref, ScopeDigest: att.BundleDigest, PlanDigest: eval.PlanDigest, Profile: profileRef, Reviewer: att.Reviewer, Decision: att.Decision, ReviewedAt: att.AttestedAt, Supersedes: att.Supersedes, EvidenceRefs: att.EvidenceRefs, Criteria: att.Criteria, Tools: att.Tools, Findings: att.Findings, Path: att.Path}
			}
			eval.Blockers = uniqueSorted(eval.Blockers)
			eval.Warnings = uniqueSorted(eval.Warnings)
			return eval, nil
		}
	}
	profile, _, err := s.loadReviewProfile(profileRef)
	if err != nil {
		return eval, err
	}
	baseRequiredCriteria := append([]ReviewCriterionProfile{}, profile.Criteria...)
	requiredCriteria := append([]ReviewCriterionProfile{}, baseRequiredCriteria...)
	effectiveTools := []ReviewPlanTool{}
	baseIndependence := policy.ReviewerIndependence[scope.Kind]
	independence := baseIndependence
	if policy.SchemaVersion >= ReviewPolicySchemaVersion && policy.ComponentAware {
		plan, planErr := s.ReviewPlan(ref)
		if planErr != nil {
			return eval, planErr
		}
		eval.PlanDigest = plan.PlanDigest
		eval.Warnings = append(eval.Warnings, plan.Warnings...)
		eval.Blockers = append(eval.Blockers, plan.Blockers...)
		requiredCriteria = requiredCriteria[:0]
		for _, criterion := range plan.Criteria {
			if !criterion.Required {
				continue
			}
			required := true
			requiredCriteria = append(requiredCriteria, ReviewCriterionProfile{ID: criterion.ID, Description: criterion.Description, Rules: criterion.Rules, EvidenceClasses: criterion.EvidenceClasses, Required: &required})
		}
		effectiveTools = append(effectiveTools, plan.Tools...)
		independence = plan.Independence
	}
	attempts, err := s.ListReviewAttempts(ref)
	if err != nil {
		return eval, err
	}
	if len(attempts) == 0 {
		eval.Blockers = append(eval.Blockers, "no review attempt exists for "+ref)
		return eval, nil
	}
	current := attempts[len(attempts)-1]
	eval.Current = &current
	legacyPlanExempt := s.componentAwareLegacyAttemptExempt(scope, policy, current)
	if legacyPlanExempt {
		requiredCriteria = append([]ReviewCriterionProfile{}, baseRequiredCriteria...)
		effectiveTools = nil
		independence = baseIndependence
		eval.Warnings = append(eval.Warnings, "completed scope retains its approved pre-component-aware review attempt")
	}
	if len(attempts) > 1 && current.Supersedes != attempts[len(attempts)-2].ReviewID {
		eval.Blockers = append(eval.Blockers, "latest review does not supersede the previous attempt")
	}
	if current.ScopeDigest != digest && !legacyPlanExempt {
		eval.Blockers = append(eval.Blockers, "review is stale: scope digest changed")
	} else {
		eval.Fresh = true
	}
	if eval.PlanDigest != "" && current.PlanDigest != eval.PlanDigest && !legacyPlanExempt {
		eval.Fresh = false
		if current.PlanDigest == "" {
			eval.Blockers = append(eval.Blockers, "review is stale: effective plan digest is missing")
		} else {
			eval.Blockers = append(eval.Blockers, "review is stale: effective plan digest changed")
		}
	}
	if current.Profile != profile.Ref() {
		eval.Blockers = append(eval.Blockers, "review profile does not match current policy")
	}
	for _, ref := range current.EvidenceRefs {
		if err := validateReviewEvidenceRef(s.Root, ref); err != nil {
			eval.Blockers = append(eval.Blockers, err.Error())
		}
	}
	if len(effectiveTools) > 0 {
		toolWarnings, toolBlockers := evaluateReviewToolCoverage(s.Root, effectiveTools, current.Tools)
		eval.Warnings = append(eval.Warnings, toolWarnings...)
		eval.Blockers = append(eval.Blockers, toolBlockers...)
	}
	if current.Reviewer == "" || strings.ContainsAny(current.Reviewer, "\r\n") {
		eval.Blockers = append(eval.Blockers, "reviewer execution identity is missing or malformed")
	}
	switch independence {
	case "", "same-actor-separate-execution":
		if !strings.HasPrefix(current.Reviewer, "agent:") && !strings.HasPrefix(current.Reviewer, "human:") {
			eval.Blockers = append(eval.Blockers, "reviewer must use an agent: or human: execution identity")
		}
	case "different-actor":
		if !strings.HasPrefix(current.Reviewer, "agent:independent-") && !strings.HasPrefix(current.Reviewer, "human:") {
			eval.Blockers = append(eval.Blockers, "review policy requires an independent reviewer identity")
		}
	case "mandatory-human":
		if !strings.HasPrefix(current.Reviewer, "human:") {
			eval.Blockers = append(eval.Blockers, "review policy requires human approval")
		}
	default:
		eval.Blockers = append(eval.Blockers, "review policy has invalid independence mode "+independence)
	}
	if _, err := time.Parse(time.RFC3339, current.ReviewedAt); err != nil {
		eval.Blockers = append(eval.Blockers, "reviewed_at must be RFC3339")
	}
	criteria := map[string]ReviewCriterion{}
	profileCriteria := map[string]bool{}
	for _, required := range requiredCriteria {
		profileCriteria[required.ID] = true
	}
	for _, criterion := range current.Criteria {
		if _, duplicate := criteria[criterion.ID]; duplicate {
			eval.Blockers = append(eval.Blockers, "duplicate criterion "+criterion.ID)
		}
		criteria[criterion.ID] = criterion
		if !profileCriteria[criterion.ID] {
			eval.Blockers = append(eval.Blockers, "unknown criterion "+criterion.ID)
		}
		if criterion.Evidence != "" {
			if err := validateReviewEvidenceRef(s.Root, criterion.Evidence); err != nil {
				eval.Blockers = append(eval.Blockers, err.Error())
			}
		}
	}
	for _, required := range requiredCriteria {
		criterion, ok := criteria[required.ID]
		if !ok {
			eval.Blockers = append(eval.Blockers, "missing criterion "+required.ID)
			continue
		}
		switch criterion.Disposition {
		case "passed":
			if criterion.Evidence == "" && len(current.EvidenceRefs) == 0 {
				eval.Blockers = append(eval.Blockers, "criterion "+required.ID+" has no evidence")
			}
		case "finding":
			if len(current.Findings) == 0 {
				eval.Blockers = append(eval.Blockers, "criterion "+required.ID+" references a missing finding")
			}
		case "not-applicable":
			if criterion.Rationale == "" {
				eval.Blockers = append(eval.Blockers, "criterion "+required.ID+" lacks not-applicable rationale")
			}
		default:
			eval.Blockers = append(eval.Blockers, "criterion "+required.ID+" has invalid disposition")
		}
		if len(required.EvidenceClasses) > 0 && criterion.Disposition == "passed" {
			matched := false
			refs := append([]string{criterion.Evidence}, current.EvidenceRefs...)
			for _, ref := range refs {
				class := strings.SplitN(ref, ":", 2)[0]
				for _, requiredClass := range required.EvidenceClasses {
					if class == requiredClass {
						matched = true
					}
				}
			}
			if !matched {
				eval.Blockers = append(eval.Blockers, "criterion "+required.ID+" lacks a required evidence class")
			}
		}
	}
	allowedRisk := map[string]bool{}
	for _, severity := range policy.AcceptedRiskSeverities {
		allowedRisk[severity] = true
	}
	seenFindings := map[string]bool{}
	for _, finding := range current.Findings {
		if seenFindings[finding.ID] {
			eval.Blockers = append(eval.Blockers, "duplicate finding "+finding.ID)
		}
		seenFindings[finding.ID] = true
		if finding.Severity == "" || finding.Action == "" {
			eval.Blockers = append(eval.Blockers, "finding "+finding.ID+" lacks severity or action")
		}
		switch finding.Disposition {
		case "resolved", "wont-fix":
		case "accepted-risk":
			if !allowedRisk[finding.Severity] || finding.Owner == "" || finding.Rationale == "" || finding.ReviewBy == "" {
				eval.Blockers = append(eval.Blockers, "finding "+finding.ID+" has unapproved or incomplete accepted risk")
			}
		case "open", "changes-requested":
			eval.Blockers = append(eval.Blockers, "finding "+finding.ID+" is "+finding.Disposition)
		default:
			eval.Blockers = append(eval.Blockers, "finding "+finding.ID+" has invalid disposition")
		}
	}
	decisionAllowed := current.Decision == "approved" || (current.Decision == "approved-with-reservations" && policy.AllowApprovedWithReservations)
	if !decisionAllowed {
		eval.Blockers = append(eval.Blockers, "review decision does not permit closeout: "+current.Decision)
	}
	sort.Strings(eval.Blockers)
	eval.Approved = len(eval.Blockers) == 0
	return eval, nil
}

func evaluateReviewToolCoverage(root string, planTools []ReviewPlanTool, dispositions []ReviewToolDisposition) ([]string, []string) {
	warnings, blockers := []string{}, []string{}
	planned := map[string]ReviewPlanTool{}
	for _, tool := range planTools {
		planned[reviewToolKey(tool.ID, tool.Component)] = tool
	}
	observed := map[string]ReviewToolDisposition{}
	for _, disposition := range dispositions {
		key := reviewToolKey(disposition.ID, disposition.Component)
		label := reviewToolLabel(disposition.ID, disposition.Component)
		tool, ok := planned[key]
		if !ok {
			blockers = append(blockers, "unknown review tool disposition "+label)
			continue
		}
		if _, duplicate := observed[key]; duplicate {
			blockers = append(blockers, "duplicate review tool disposition "+label)
			continue
		}
		observed[key] = disposition
		if disposition.Evidence != "" {
			if err := validateReviewEvidenceRef(root, disposition.Evidence); err != nil {
				blockers = append(blockers, err.Error())
			}
		}
		completion := containsFold(tool.Preconditions, "review-complete")
		switch disposition.Disposition {
		case "passed", "failed":
			message := reviewToolEvidenceBlocker(tool, disposition)
			if message != "" {
				if tool.Requiredness == "required" {
					blockers = append(blockers, message)
				} else {
					warnings = append(warnings, message)
				}
			}
			if disposition.Disposition == "failed" {
				message = "review tool " + label + " failed"
				if tool.Requiredness == "required" {
					blockers = append(blockers, message)
				} else {
					warnings = append(warnings, message)
				}
			}
		case "not-used":
			if tool.Requiredness == "required" {
				blockers = append(blockers, "required review tool "+label+" was not used")
			} else if disposition.Rationale == "" {
				warnings = append(warnings, "recommended review tool "+label+" lacks not-used rationale")
			}
		case "deferred":
			if !completion || disposition.Rationale == "" {
				message := "review tool " + label + " has invalid deferred disposition"
				if tool.Requiredness == "required" {
					blockers = append(blockers, message)
				} else {
					warnings = append(warnings, message)
				}
			}
		default:
			blockers = append(blockers, "review tool "+label+" has invalid disposition")
		}
		if tool.Requiredness == "required" && !completion && disposition.Disposition != "passed" {
			blockers = append(blockers, "required review tool "+label+" did not pass")
		}
	}
	for key, tool := range planned {
		if _, ok := observed[key]; ok {
			continue
		}
		label := reviewToolLabel(tool.ID, tool.Component)
		if tool.Requiredness == "required" {
			blockers = append(blockers, "missing required review tool disposition "+label)
		} else {
			warnings = append(warnings, "recommended review tool "+label+" has no disposition")
		}
	}
	return uniqueSorted(warnings), uniqueSorted(blockers)
}

func reviewToolEvidenceBlocker(tool ReviewPlanTool, disposition ReviewToolDisposition) string {
	label := reviewToolLabel(tool.ID, tool.Component)
	if disposition.Evidence == "" {
		return "review tool " + label + " has no evidence"
	}
	if len(tool.EvidenceClasses) == 0 {
		return ""
	}
	class := strings.SplitN(disposition.Evidence, ":", 2)[0]
	if containsFold(tool.EvidenceClasses, class) {
		return ""
	}
	return "review tool " + label + " lacks a required evidence class"
}

func reviewToolKey(id, component string) string {
	return id + "\x00" + component
}

func reviewToolLabel(id, component string) string {
	if component == "" {
		return id
	}
	return id + " (component " + component + ")"
}

func (s Store) componentAwareLegacyAttemptExempt(scope ScopeRef, policy ReviewPolicy, attempt ReviewAttempt) bool {
	if !policy.ComponentAware || policy.ComponentAwareAdoptedAt == "" || attempt.PlanDigest != "" {
		return false
	}
	adopted, err := time.Parse(time.DateOnly, policy.ComponentAwareAdoptedAt)
	if err != nil {
		return false
	}
	reviewed, err := time.Parse(time.RFC3339, attempt.ReviewedAt)
	if err != nil || !reviewed.Before(adopted) {
		return false
	}
	done, err := s.scopeLifecycleDone(scope)
	return err == nil && done
}

func (s Store) reviewBundlesLegacyAttemptExempt(scope ScopeRef, policy ReviewPolicy) bool {
	if !policy.ReviewBundles || policy.ReviewBundlesAdoptedAt == "" {
		return false
	}
	done, err := s.scopeLifecycleDone(scope)
	if err != nil || !done {
		return false
	}
	adopted, err := time.Parse(time.DateOnly, policy.ReviewBundlesAdoptedAt)
	if err != nil {
		return false
	}
	var createdAt string
	switch scope.Kind {
	case "spec":
		if sp, err := s.GetSpec(scope.Slug); err == nil {
			createdAt = sp.CreatedAt
		}
	case "milestone":
		if rm, err := s.GetRoadmap(scope.Roadmap); err == nil {
			createdAt = rm.CreatedAt
		}
	case "roadmap":
		if rm, err := s.GetRoadmap(scope.Slug); err == nil {
			createdAt = rm.CreatedAt
		}
	}
	if createdAt == "" {
		return false
	}
	created, err := time.Parse(time.DateOnly, createdAt)
	if err != nil {
		return false
	}
	return created.Before(adopted)
}

func (s Store) scopeLifecycleDone(scope ScopeRef) (bool, error) {
	switch scope.Kind {
	case "spec":
		sp, err := s.GetSpec(scope.Slug)
		if err != nil {
			return false, err
		}
		return sp.Status == "done", nil
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return false, err
		}
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				for _, slug := range milestone.Specs {
					sp, specErr := s.GetSpec(slug)
					if specErr != nil {
						return false, specErr
					}
					if sp.Status != "done" {
						return false, nil
					}
				}
				return true, nil
			}
		}
		return false, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return false, err
		}
		return rm.Status == "done", nil
	default:
		return false, nil
	}
}

func (s Store) reviewRequiredForScope(scope ScopeRef, policy ReviewPolicy) (bool, error) {
	if !policy.Enabled {
		return false, nil
	}
	if policy.RequireReviewForLegacyDoneScopes || policy.AdoptedAt == "" {
		return true, nil
	}
	adoptedAt, err := time.Parse(time.DateOnly, policy.AdoptedAt)
	if err != nil {
		return true, nil
	}

	var createdAt, status string
	switch scope.Kind {
	case "spec":
		sp, err := s.GetSpec(scope.Slug)
		if err != nil {
			return false, err
		}
		createdAt, status = sp.CreatedAt, sp.Status
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return false, err
		}
		createdAt, status = rm.CreatedAt, rm.Status
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return false, err
		}
		createdAt, status = rm.CreatedAt, rm.Status
	}
	if status != "done" {
		return true, nil
	}
	created, err := time.Parse(time.DateOnly, createdAt)
	if err != nil {
		return true, nil
	}
	return !created.Before(adoptedAt), nil
}

func validateReviewEvidenceRef(root, ref string) error {
	parts := strings.SplitN(strings.TrimSpace(ref), ":", 2)
	if len(parts) != 2 || !slugPattern.MatchString(parts[0]) || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("invalid review evidence ref %q", ref)
	}
	if parts[0] != "report" {
		return nil
	}
	rel := filepath.Clean(filepath.FromSlash(parts[1]))
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("review evidence path escapes project root: %q", ref)
	}
	if !strings.HasPrefix(filepath.ToSlash(rel), ".pose/reports/") {
		rel = filepath.Join(".pose", "reports", rel)
	}
	path := filepath.Join(root, rel)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("review evidence does not resolve: %q", ref)
	}
	return nil
}

func (s Store) GetCloseoutState(ref string) (CloseoutState, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return CloseoutState{}, err
	}
	review, err := s.ReviewCheck(ref)
	if err != nil {
		return CloseoutState{}, err
	}
	state := CloseoutState{SchemaVersion: ReviewSchemaVersion, Scope: ref, ScopeDigest: review.ScopeDigest, Review: review, LifecycleDone: true, Blockers: append([]string{}, review.Blockers...)}
	switch scope.Kind {
	case "spec":
		sp, err := s.GetSpec(scope.Slug)
		if err != nil {
			return state, err
		}
		state.LifecycleDone = sp.Status == "done"
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return state, err
		}
		var found bool
		for _, milestone := range rm.Milestones {
			if milestone.ID != scope.Milestone {
				continue
			}
			found = true
			for _, slug := range milestone.Specs {
				child, err := s.GetCloseoutState("spec:" + slug)
				if err != nil {
					return state, err
				}
				state.Children = append(state.Children, child)
				if !child.Terminal {
					state.Blockers = append(state.Blockers, "child "+child.Scope+" is not closed")
				}
			}
			break
		}
		if !found {
			return state, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
		}
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return state, err
		}
		state.LifecycleDone = rm.Status == "done"
		for _, milestone := range rm.Milestones {
			child, err := s.GetCloseoutState("milestone:" + rm.Slug + "/" + milestone.ID)
			if err != nil {
				return state, err
			}
			state.Children = append(state.Children, child)
			if !child.Terminal {
				state.Blockers = append(state.Blockers, "child "+child.Scope+" is not closed")
			}
		}
	}
	if !review.Required {
		state.Blockers = removeReviewOnlyBlockers(state.Blockers)
	}
	if !state.LifecycleDone && scope.Kind != "milestone" {
		state.Blockers = append(state.Blockers, "lifecycle status is not done")
	}
	sort.Strings(state.Blockers)
	state.Terminal = len(state.Blockers) == 0 && (review.Approved || !review.Required) && state.LifecycleDone
	if state.Terminal {
		state.NextAction = "none"
	} else if len(state.Children) > 0 {
		for _, child := range state.Children {
			if !child.Terminal {
				state.NextAction = "continue " + child.Scope + ": " + child.NextAction
				break
			}
		}
	} else if !review.Approved && review.Required {
		state.NextAction = "record or remediate a fresh review for " + ref
	} else if !state.LifecycleDone {
		state.NextAction = "apply the guarded lifecycle transition for " + ref
	}
	if state.NextAction == "" {
		state.NextAction = "resolve closeout blockers for " + ref
	}
	return state, nil
}

func removeReviewOnlyBlockers(blockers []string) []string {
	out := blockers[:0]
	for _, blocker := range blockers {
		if strings.Contains(blocker, "review") || strings.Contains(blocker, "criterion") || strings.Contains(blocker, "finding") {
			continue
		}
		out = append(out, blocker)
	}
	return out
}

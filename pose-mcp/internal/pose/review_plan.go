package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ReviewPlanSchemaVersion = 1

type ReviewPlanComponent struct {
	ID                string   `json:"id"`
	Path              string   `json:"path"`
	Kind              string   `json:"kind,omitempty"`
	Language          string   `json:"language,omitempty"`
	Domain            string   `json:"domain,omitempty"`
	Owner             string   `json:"owner,omitempty"`
	Criticality       string   `json:"criticality,omitempty"`
	ValidationProfile string   `json:"validation_profile,omitempty"`
	MetadataStatus    string   `json:"metadata_status,omitempty"`
	MetadataMissing   []string `json:"metadata_missing,omitempty"`
	// Origin says whether the component was declared by the scope, observed
	// from delivery provenance, or both. It is derived from Sources, so it does
	// not enter the plan digest, and it is what keeps a forecast distinguishable
	// from a final observation for every consumer of the plan.
	Origin  string   `json:"origin,omitempty"`
	Sources []string `json:"sources"`

	metadataIncompleteFlag bool `json:"-"`
}

type ReviewPlanProfile struct {
	Ref        string   `json:"ref"`
	Category   string   `json:"category"`
	Order      int      `json:"order"`
	Source     string   `json:"source"`
	Components []string `json:"components,omitempty"`
	Rationale  string   `json:"rationale"`
}

type ReviewPlanCriterion struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	// Kind is resolved at plan time, so the bundle seals what each criterion
	// was when it was reviewed. A profile edited afterwards cannot retroactively
	// turn a judged criterion into a collected one, or the reverse.
	Kind            string   `json:"kind,omitempty"`
	Rules           []string `json:"rules,omitempty"`
	EvidenceClasses []string `json:"evidence_classes,omitempty"`
	// RequiresStructuralMapping is resolved from the profiles at plan time and
	// composes monotonically, the way Kind and independence do: one profile can
	// raise a criterion to answer for observed structure, and none can lower it.
	RequiresStructuralMapping bool     `json:"requires_structural_mapping,omitempty"`
	Profiles                  []string `json:"profiles"`
}

// ReviewCriterionKind resolves a planned criterion's kind, falling back to the
// derivation for a plan sealed before the field existed. Reading it through one
// accessor is what keeps auto-attest, the CLI and the verifier from disagreeing
// about which criteria a reviewer still owes an answer for.
func ReviewCriterionKind(criterion ReviewPlanCriterion) string {
	if criterion.Kind == ReviewCriterionKindMechanical || criterion.Kind == ReviewCriterionKindJudgment {
		return criterion.Kind
	}
	return DeriveReviewCriterionKind(criterion.EvidenceClasses)
}

type ReviewPlanTool struct {
	ID           string `json:"id"`
	Requiredness string `json:"requiredness"`
	// ProducerCoverage is `none` when the component this tool is scoped to
	// declares, in the validation matrix, that it runs no check at all. The
	// plan then asks for evidence nothing in the repository can emit, and a
	// required tool may only be disposed `passed` or `failed` citing a result
	// of a class it accepts — so the reviewer's options are to cite another
	// component's result, which is false, or to stay blocked forever.
	//
	// This is the same failure the profile loader already refuses one level up:
	// a profile demanding a class no registered check may emit plans a gate
	// only a fabricated disposition can pass. Per component, the matrix is what
	// says so, and the plan had not been reading it.
	//
	// Empty means the ordinary case: producers exist, or the matrix does not
	// say otherwise and the tool stays required. The engine claims the gap only
	// where the repository declared it.
	ProducerCoverage string   `json:"producer_coverage,omitempty"`
	Args             []string `json:"args"`
	Rationale        string   `json:"rationale"`
	EvidenceClasses  []string `json:"evidence_classes,omitempty"`
	Criteria         []string `json:"criteria,omitempty"`
	Component        string   `json:"component,omitempty"`
	Preconditions    []string `json:"preconditions,omitempty"`
}

type ReviewPlan struct {
	SchemaVersion       int                   `json:"schema_version"`
	Scope               string                `json:"scope"`
	ScopeDigest         string                `json:"scope_digest"`
	PlanDigest          string                `json:"plan_digest"`
	PolicySchemaVersion int                   `json:"policy_schema_version"`
	BaseProfile         string                `json:"base_profile"`
	Components          []ReviewPlanComponent `json:"components"`
	SelectedProfiles    []ReviewPlanProfile   `json:"selected_profiles"`
	Criteria            []ReviewPlanCriterion `json:"criteria"`
	Tools               []ReviewPlanTool      `json:"tools"`
	Independence        string                `json:"independence"`
	// Band and Bands explain the resolved plan; Projection reports the same
	// plan over declared scope alone. All three are derived from the fields
	// above, add no obligation, and stay out of the plan digest. See
	// review_bands.go for why that separation is the point.
	Band       string               `json:"band,omitempty"`
	Bands      []ReviewPlanBand     `json:"bands,omitempty"`
	Projection ReviewPlanProjection `json:"projection"`
	// PolicyBaseline reports the protected contract this plan was resolved
	// under when the scope changes policy, profiles or their schemas. Its
	// consequences reach the digest through Independence, Criteria and Explain;
	// the struct itself is the readable projection of those explain lines.
	PolicyBaseline ReviewPlanPolicyBaseline `json:"policy_baseline"`
	// Structure is the observed structural context, resolved only when an
	// adopted profile selects on it. Unlike the summaries above it is an
	// obligation input, so it enters the plan digest: see review_structure.go.
	Structure *ReviewPlanStructure `json:"structure,omitempty"`
	Warnings  []string             `json:"warnings,omitempty"`
	Blockers  []string             `json:"blockers,omitempty"`
	Explain   []string             `json:"explain"`
}

type reviewRepoEntry struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	Language          string `json:"language"`
	Owner             string `json:"owner"`
	Domain            string `json:"domain"`
	Criticality       string `json:"criticality"`
	ValidationProfile string `json:"validationProfile"`
	Metadata          struct {
		Owner             string `json:"owner"`
		Domain            string `json:"domain"`
		Criticality       string `json:"criticality"`
		ValidationProfile string `json:"validationProfile"`
	} `json:"metadata"`
	MetadataStatus struct {
		Source        string   `json:"source"`
		IsComplete    *bool    `json:"isComplete"`
		MissingFields []string `json:"missingFields"`
	} `json:"metadataStatus"`
	Kind string `json:"-"`
}

type reviewPlanContext struct {
	Components    []ReviewPlanComponent
	DeliveryKinds []string
	ArtifactPaths []string
	SpecSlugs     []string
	// StructuralKinds is what the subject was observed to do. It is empty unless
	// an adopted profile selects on it, and it is deliberately absent from the
	// declared-scope projection: an observation is not a forecast.
	StructuralKinds []string
	Warnings        []string
	Blockers        []string
}

type reviewToolDefinition struct {
	Rationale string
	Phase     int
}

var reviewToolCatalog = map[string]reviewToolDefinition{
	"suggest-review":   {Rationale: "resolve the component-specific workflow, skill, rules and validation trail", Phase: 10},
	"assess-discover":  {Rationale: "inspect component structure, language, size and visible debt", Phase: 20},
	"assess-design":    {Rationale: "observe bounded structural deltas on the canonical review subject", Phase: 20},
	"assess-tech-debt": {Rationale: "inspect unresolved TODO, FIXME, panic and stub findings", Phase: 20},
	"assess-integrate": {Rationale: "inspect providers, consumers and inter-component contract gaps", Phase: 20},
	"artifact-check":   {Rationale: "reconcile declared artifacts with Git-observed provenance", Phase: 30},
	"validate":         {Rationale: "run the registered deterministic checks for the affected module", Phase: 30},
	"surface-check":    {Rationale: "verify delivery composition, reachability and fresh evidence", Phase: 30},
	"roadmap-check":    {Rationale: "verify milestone and roadmap outcome roll-up", Phase: 30},
	"history-check":    {Rationale: "verify append-only governance history", Phase: 30},
	"knowledge-check":  {Rationale: "verify knowledge schema, sensitivity and lifecycle", Phase: 30},
	"skills-check":     {Rationale: "verify distributed skill and workflow conformance", Phase: 30},
	"review-check":     {Rationale: "verify exact review-plan coverage and freshness", Phase: 40},
	"closeout-check":   {Rationale: "verify remaining hierarchical closeout blockers", Phase: 50},
}

var reviewPreconditionCatalog = map[string]bool{
	"artifacts-attributed": true, "component-mapped": true,
	"delivery-target-declared": true, "multi-component": true,
	"review-complete": true, "scope-authorized": true,
}

// ReviewPlan resolves the immutable, read-only plan that governs a review.
// It never executes a recommendation or writes project state.
func (s Store) ReviewPlan(ref string) (ReviewPlan, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return ReviewPlan{}, err
	}
	digest, err := s.ScopeDigest(ref)
	if err != nil {
		return ReviewPlan{}, err
	}
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return ReviewPlan{}, err
	}
	// Resolved before the policy below is trusted. A diff that disables review
	// while changing the contract would otherwise be unreviewable: the plan
	// would refuse to exist, and the change would land unreviewed.
	guard := s.resolveReviewGovernanceGuard(scope)
	baseline := reviewPolicyBaseline{}
	if len(guard.Paths) > 0 && guard.Revision != "" {
		baseline, err = s.resolveReviewPolicyBaseline(guard.Revision, scope.Kind)
		if err != nil {
			guard.Reason, baseline = err.Error(), reviewPolicyBaseline{}
		}
	}
	// A profile source: the working tree normally, the protected revision when
	// the tree no longer governs this scope at all.
	profileRevision := ""
	if !policy.Enabled {
		if !baseline.Resolved || !baseline.Policy.Enabled {
			return ReviewPlan{}, fmt.Errorf("pose: review policy is absent or disabled")
		}
		policy, profileRevision = baseline.Policy, baseline.Revision
	}
	baseRef := policy.Profiles[scope.Kind]
	if baseRef == "" {
		return ReviewPlan{}, fmt.Errorf("pose: no review profile configured for %s", scope.Kind)
	}
	base, err := s.loadReviewProfileFrom(profileRevision, baseRef)
	if err != nil {
		// The reviewed diff removed or broke the profile that governs its own
		// review. The protected contract governs instead; without this the plan
		// would refuse to exist and the change would land unreviewed.
		if !baseline.Resolved || !baseline.Policy.Enabled {
			return ReviewPlan{}, err
		}
		unreadable := err.Error()
		policy, profileRevision = baseline.Policy, baseline.Revision
		baseRef = policy.Profiles[scope.Kind]
		if base, err = s.loadReviewProfileFrom(profileRevision, baseRef); err != nil {
			return ReviewPlan{}, err
		}
		guard.Unreadable = unreadable
	}
	if base.Scope != scope.Kind {
		return ReviewPlan{}, fmt.Errorf("pose: profile %s cannot review %s scopes", base.Ref(), scope.Kind)
	}

	plan := ReviewPlan{
		SchemaVersion: ReviewPlanSchemaVersion, Scope: ref, ScopeDigest: digest,
		PolicySchemaVersion: policy.SchemaVersion, BaseProfile: baseRef,
		Components: []ReviewPlanComponent{}, SelectedProfiles: []ReviewPlanProfile{},
		Criteria: []ReviewPlanCriterion{}, Tools: []ReviewPlanTool{}, Explain: []string{},
		Independence: normalizeReviewIndependence(policy.ReviewerIndependence[scope.Kind]),
	}
	profiles := []ReviewProfile{base}
	plan.SelectedProfiles = append(plan.SelectedProfiles, ReviewPlanProfile{Ref: base.Ref(), Category: "base", Order: 0, Source: ".pose/review-profiles/" + base.ID + ".json", Rationale: "selected by terminal scope policy"})
	plan.Explain = append(plan.Explain, "base profile "+base.Ref()+" selected by "+scope.Kind+" scope policy")

	context := reviewPlanContext{}
	overlays, selected := []ReviewProfile{}, []reviewOverlaySelection{}
	if policy.SchemaVersion >= ReviewPolicySchemaVersion && policy.ComponentAware {
		context, err = s.resolveReviewPlanContext(scope)
		if err != nil {
			return ReviewPlan{}, err
		}
		plan.Components = context.Components
		plan.Warnings = append(plan.Warnings, context.Warnings...)
		plan.Blockers = append(plan.Blockers, context.Blockers...)
		for _, message := range append([]string{}, context.Warnings...) {
			if policy.UnmappedComponentBehavior == "blocker" && (strings.HasPrefix(message, "unmapped review component") || strings.HasPrefix(message, "metadata-incomplete review component")) {
				plan.Blockers = append(plan.Blockers, message)
			}
		}

		for _, overlayRef := range uniqueSorted(policy.OverlayProfiles) {
			overlay, loadErr := s.loadReviewProfileFrom(profileRevision, overlayRef)
			if loadErr != nil {
				plan.Blockers = append(plan.Blockers, loadErr.Error())
				continue
			}
			if overlay.SchemaVersion < ReviewPolicySchemaVersion || !hasReviewSelectors(overlay.Selectors) {
				plan.Blockers = append(plan.Blockers, "review overlay "+overlayRef+" must use schema v2 with typed selectors")
				continue
			}
			if overlay.Scope != scope.Kind {
				continue
			}
			overlays = append(overlays, overlay)
		}
		// Resolved only when something selects on it. A repository that adopted
		// no structural profile pays nothing here, not even the Git reads, which
		// is what keeps an opt-in contract opt-in in cost as well as in effect.
		for _, overlay := range overlays {
			if len(overlay.Selectors.StructuralKinds) == 0 {
				continue
			}
			plan.Structure = s.resolveReviewStructure(scope, context.Components)
			context.StructuralKinds = plan.Structure.Kinds
			if !plan.Structure.Observed {
				plan.Warnings = append(plan.Warnings, "structural selectors are adopted but the subject was not observed: "+firstNonempty(plan.Structure.Reason, "no reason reported"))
			}
			for _, unknown := range plan.Structure.Unknown {
				plan.Warnings = append(plan.Warnings, "unresolved structural coverage "+unknown)
			}
			break
		}
		selected = selectReviewOverlays(overlays, context)
		for _, item := range selected {
			profiles = append(profiles, item.profile)
			plan.SelectedProfiles = append(plan.SelectedProfiles, item.selection)
			plan.Explain = append(plan.Explain, item.profile.Ref()+" selected: "+item.selection.Rationale)
			plan.Independence = stricterReviewIndependence(plan.Independence, item.profile.Independence)
		}
	}

	plan.Criteria, plan.Blockers = composeReviewCriteria(profiles, plan.Blockers)
	var added bool
	plan.Criteria, plan.Blockers, added = addCrossComponentReviewCriterion(plan.Criteria, plan.Blockers, len(plan.Components))
	if added {
		plan.Explain = append(plan.Explain, "cross-component-integration added because multiple mapped component roots are affected")
	}
	plan.PolicyBaseline.Paths = guard.Paths
	plan.PolicyBaseline.Reason = guard.Reason
	if guard.Unreadable != "" {
		plan.PolicyBaseline.Weakened = append(plan.PolicyBaseline.Weakened, "unreadable_tree_contract:"+guard.Unreadable)
	}
	if len(guard.Paths) > 0 && baseline.Resolved {
		applyProtectedPolicyBaseline(&plan, baseline, policy, scope, context)
	}
	plan.Explain = append(plan.Explain, reviewPolicyBaselineExplain(plan.PolicyBaseline)...)
	if !plan.PolicyBaseline.Protected && len(guard.Paths) > 0 {
		plan.Warnings = append(plan.Warnings, "unprotected review contract change "+strings.Join(guard.Paths, ",")+": "+firstNonempty(guard.Reason, "the base contract could not be resolved"))
	}
	plan.Tools, plan.Blockers, plan.Warnings = buildReviewTools(scope, context, profiles, plan.SelectedProfiles, plan.Criteria, plan.Blockers, plan.Warnings)
	plan.Warnings = s.annotateReviewToolProducerCoverage(plan.Tools, plan.Warnings)
	for _, c := range plan.Criteria {
		for _, rule := range c.Rules {
			path := filepath.Join(s.Root, ".pose", "rules", rule+".md")
			if _, statErr := os.Stat(path); statErr != nil {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("uninstalled review rule %q for criterion %s (install via extension)", rule, c.ID))
			}
		}
	}
	plan.Warnings = uniqueSorted(plan.Warnings)
	plan.Blockers = uniqueSorted(plan.Blockers)
	plan.Explain = uniqueStable(plan.Explain)
	floor := normalizeReviewIndependence(policy.ReviewerIndependence[scope.Kind])
	plan.Bands, plan.Band = deriveReviewBands(plan, scope.Kind, floor, selected, undecidableReviewOverlays(overlays, selected), context)
	if band, ok := reviewPolicyBaselineBand(plan.PolicyBaseline); ok {
		plan.Bands = append(plan.Bands, band)
		if reviewBandRank[band.Band] > reviewBandRank[plan.Band] {
			plan.Band = band.Band
		}
	}
	plan.Projection = projectDeclaredReviewObligations(plan, scope, base, floor, overlays, context)
	plan.PlanDigest, err = digestReviewPlan(plan)
	if err != nil {
		return ReviewPlan{}, err
	}
	return plan, nil
}

func (s Store) resolveReviewPlanContext(scope ScopeRef) (reviewPlanContext, error) {
	context := reviewPlanContext{Components: []ReviewPlanComponent{}, DeliveryKinds: []string{}, ArtifactPaths: []string{}, SpecSlugs: []string{}}
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return context, err
	}
	entries, err := s.loadReviewRepoEntries()
	if err != nil {
		context.Blockers = append(context.Blockers, err.Error())
	}
	byPath := map[string]ReviewPlanComponent{}
	unmapped := map[string]bool{}
	addEntry := func(entry reviewRepoEntry, source string) {
		path := filepath.ToSlash(filepath.Clean(entry.Path))
		component := byPath[path]
		component.ID, component.Path, component.Kind, component.Language = entry.Name, path, entry.Kind, entry.Language
		component.Owner, component.Domain, component.Criticality = firstNonempty(entry.Owner, entry.Metadata.Owner), firstNonempty(entry.Domain, entry.Metadata.Domain), firstNonempty(entry.Criticality, entry.Metadata.Criticality)
		component.ValidationProfile = firstNonempty(entry.ValidationProfile, entry.Metadata.ValidationProfile)
		component.MetadataStatus = entry.MetadataStatus.Source
		component.MetadataMissing = uniqueSorted(append(append([]string{}, component.MetadataMissing...), entry.MetadataStatus.MissingFields...))
		component.metadataIncompleteFlag = component.metadataIncompleteFlag || (entry.MetadataStatus.IsComplete != nil && !*entry.MetadataStatus.IsComplete) || entry.MetadataStatus.Source == "defaulted"
		clearIncompleteReviewMetadata(&component)
		component.Sources = append(component.Sources, source)
		component.Sources = uniqueSorted(component.Sources)
		byPath[path] = component
	}
	for _, spec := range specs {
		context.SpecSlugs = append(context.SpecSlugs, spec.Slug)
		for _, explicit := range spec.Components {
			matches := matchExplicitReviewComponent(entries, explicit)
			switch len(matches) {
			case 0:
				unmapped["explicit:"+explicit] = true
			case 1:
				addEntry(matches[0], "spec:"+spec.Slug+":component:"+explicit)
			default:
				context.Blockers = append(context.Blockers, "ambiguous review component "+explicit)
			}
		}
		claims, found, parseErr := ParseArtifactClaims(spec, ArtifactPolicy{})
		if parseErr != nil {
			context.Blockers = append(context.Blockers, parseErr.Error())
		} else if found {
			for _, claim := range claims {
				for _, path := range []string{claim.Path, claim.OldPath, claim.NewPath} {
					if path != "" {
						context.ArtifactPaths = append(context.ArtifactPaths, path)
						s.mapReviewPath(entries, path, "artifact:"+spec.Slug+":"+path, addEntry, unmapped, &context)
					}
				}
			}
		}
		targets, found, parseErr := ParseDeliveryTargets(spec)
		if parseErr != nil {
			context.Blockers = append(context.Blockers, parseErr.Error())
		} else if found {
			for _, target := range targets {
				context.DeliveryKinds = append(context.DeliveryKinds, target.Kind)
				for _, path := range []string{target.Module, target.Entrypoint} {
					context.ArtifactPaths = append(context.ArtifactPaths, path)
					s.mapReviewPath(entries, path, "delivery:"+target.Ref+":"+path, addEntry, unmapped, &context)
				}
			}
		}
	}
	s.loadObservedReviewPaths(context.SpecSlugs, entries, addEntry, unmapped, &context)
	for _, key := range reviewSortedKeys(unmapped) {
		context.Warnings = append(context.Warnings, "unmapped review component "+key)
	}
	for _, component := range byPath {
		component.Sources = uniqueSorted(component.Sources)
		component.Origin = reviewComponentOrigin(component.Sources)
		if metadataIncomplete(component) {
			reason := "missing:" + strings.Join(component.MetadataMissing, ",")
			if len(component.MetadataMissing) == 0 {
				reason = "source:" + component.MetadataStatus
			}
			context.Warnings = append(context.Warnings, "metadata-incomplete review component "+component.Path+" "+reason)
		}
		context.Components = append(context.Components, component)
	}
	sort.Slice(context.Components, func(i, j int) bool { return context.Components[i].Path < context.Components[j].Path })
	context.DeliveryKinds = uniqueSorted(context.DeliveryKinds)
	context.ArtifactPaths = uniqueSorted(context.ArtifactPaths)
	context.SpecSlugs = uniqueSorted(context.SpecSlugs)
	return context, nil
}

func metadataIncomplete(component ReviewPlanComponent) bool {
	return component.metadataIncompleteFlag || len(component.MetadataMissing) > 0
}

func clearIncompleteReviewMetadata(component *ReviewPlanComponent) {
	if !metadataIncomplete(*component) {
		return
	}
	if len(component.MetadataMissing) == 0 {
		component.Language = ""
		component.Owner = ""
		component.Domain = ""
		component.Criticality = ""
		component.ValidationProfile = ""
		return
	}
	for _, field := range component.MetadataMissing {
		switch strings.ToLower(field) {
		case "language":
			component.Language = ""
		case "owner":
			component.Owner = ""
		case "domain":
			component.Domain = ""
		case "criticality":
			component.Criticality = ""
		case "validationprofile", "validation_profile":
			component.ValidationProfile = ""
		}
	}
}

func (s Store) reviewScopeSpecs(scope ScopeRef) ([]Spec, error) {
	slugs := []string{}
	switch scope.Kind {
	case "spec":
		slugs = append(slugs, scope.Slug)
	case "milestone":
		rm, err := s.GetRoadmap(scope.Roadmap)
		if err != nil {
			return nil, err
		}
		found := false
		for _, milestone := range rm.Milestones {
			if milestone.ID == scope.Milestone {
				slugs, found = append(slugs, milestone.Specs...), true
			}
		}
		if !found {
			return nil, fmt.Errorf("pose: milestone %s/%s not found", scope.Roadmap, scope.Milestone)
		}
	case "roadmap":
		rm, err := s.GetRoadmap(scope.Slug)
		if err != nil {
			return nil, err
		}
		for _, milestone := range rm.Milestones {
			slugs = append(slugs, milestone.Specs...)
		}
	}
	specs := []Spec{}
	for _, slug := range uniqueSorted(slugs) {
		sp, err := s.GetSpec(slug)
		if err != nil {
			return nil, err
		}
		specs = append(specs, *sp)
	}
	return specs, nil
}

func (s Store) loadReviewRepoEntries() ([]reviewRepoEntry, error) {
	entries := []reviewRepoEntry{}
	seen := map[string]bool{}

	// 1. Try .pose/indexes/repo-map.json if present
	if raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "repo-map.json")); err == nil {
		var repo struct {
			Apps     []reviewRepoEntry `json:"apps"`
			Services []reviewRepoEntry `json:"services"`
			Packages []reviewRepoEntry `json:"packages"`
		}
		if err := json.Unmarshal(raw, &repo); err != nil {
			return nil, fmt.Errorf("review component map is invalid: %w", err)
		}
		for kind, values := range map[string][]reviewRepoEntry{"app": repo.Apps, "service": repo.Services, "package": repo.Packages} {
			for _, entry := range values {
				entry.Kind = kind
				if entry.Name == "" || entry.Path == "" {
					continue
				}
				entry.Path = filepath.ToSlash(filepath.Clean(entry.Path))
				if err := validateReviewPath(s.Root, entry.Path); err != nil {
					return nil, fmt.Errorf("review component map path %q is unsafe: %w", entry.Path, err)
				}
				if !seen[entry.Path] {
					seen[entry.Path] = true
					entries = append(entries, entry)
				}
			}
		}
	}

	// 2. Load from .pose/indexes/module-metadata.json if present
	if raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "module-metadata.json")); err == nil {
		var meta struct {
			Modules map[string]struct {
				Domain            string `json:"domain"`
				Owner             string `json:"owner"`
				Criticality       string `json:"criticality"`
				ValidationProfile string `json:"validationProfile"`
			} `json:"modules"`
		}
		if json.Unmarshal(raw, &meta) == nil {
			for modPath, modMeta := range meta.Modules {
				cleanPath := filepath.ToSlash(filepath.Clean(modPath))
				if cleanPath == "." || cleanPath == "" || strings.HasPrefix(cleanPath, ".tmp/") {
					continue
				}
				if err := validateReviewPath(s.Root, cleanPath); err == nil && !seen[cleanPath] {
					seen[cleanPath] = true
					entries = append(entries, reviewRepoEntry{
						Name:              filepath.Base(cleanPath),
						Path:              cleanPath,
						Kind:              "module",
						Domain:            modMeta.Domain,
						Owner:             modMeta.Owner,
						Criticality:       modMeta.Criticality,
						ValidationProfile: modMeta.ValidationProfile,
					})
				}
			}
		}
	}

	// 3. Load from .pose/state/components/*.json if present
	if compFiles, err := filepath.Glob(filepath.Join(s.Root, ".pose", "state", "components", "*.json")); err == nil {
		for _, compFile := range compFiles {
			if raw, err := os.ReadFile(compFile); err == nil {
				var comp struct {
					ComponentSlug string `json:"component_slug"`
					RootPath      string `json:"root_path"`
					Language      string `json:"primary_language"`
				}
				if json.Unmarshal(raw, &comp) == nil {
					slug := comp.ComponentSlug
					if slug == "" {
						slug = strings.TrimSuffix(filepath.Base(compFile), ".json")
					}
					path := comp.RootPath
					if path == "" {
						path = slug
					}
					cleanPath := filepath.ToSlash(filepath.Clean(path))
					if cleanPath != "." && cleanPath != "" && !seen[cleanPath] {
						if err := validateReviewPath(s.Root, cleanPath); err == nil {
							seen[cleanPath] = true
							entries = append(entries, reviewRepoEntry{
								Name:     slug,
								Path:     cleanPath,
								Kind:     "component",
								Language: comp.Language,
							})
						}
					}
				}
			}
		}
	}

	// 4. Auto-discover top-level component directories in project root if they exist
	if dirEntries, err := os.ReadDir(s.Root); err == nil {
		for _, de := range dirEntries {
			if !de.IsDir() {
				continue
			}
			name := de.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == "target" || name == "docs" || name == "doc" || name == "locales" {
				continue
			}
			cleanPath := filepath.ToSlash(filepath.Clean(name))
			if !seen[cleanPath] {
				if err := validateReviewPath(s.Root, cleanPath); err == nil {
					seen[cleanPath] = true
					entries = append(entries, reviewRepoEntry{
						Name: name,
						Path: cleanPath,
						Kind: "directory",
					})
				}
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (s Store) loadObservedReviewPaths(specs []string, entries []reviewRepoEntry, add func(reviewRepoEntry, string), unmapped map[string]bool, context *reviewPlanContext) {
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "delivery-integrity.json"))
	if err != nil {
		return
	}
	var graph struct {
		Reverse map[string][]string `json:"reverse"`
	}
	if json.Unmarshal(raw, &graph) != nil {
		return
	}
	wanted := map[string]bool{}
	for _, slug := range specs {
		wanted[slug] = true
	}
	for path, slugs := range graph.Reverse {
		for _, slug := range slugs {
			if wanted[slug] {
				context.ArtifactPaths = append(context.ArtifactPaths, path)
				s.mapReviewPath(entries, path, "observed:"+slug+":"+path, add, unmapped, context)
				break
			}
		}
	}
}

func matchExplicitReviewComponent(entries []reviewRepoEntry, value string) []reviewRepoEntry {
	matches := []reviewRepoEntry{}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name, value) || strings.EqualFold(entry.Path, filepath.ToSlash(filepath.Clean(value))) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func (s Store) mapReviewPath(entries []reviewRepoEntry, path, source string, add func(reviewRepoEntry, string), unmapped map[string]bool, context *reviewPlanContext) {
	clean := filepath.ToSlash(filepath.Clean(path))
	if err := validateReviewPath(s.Root, clean); err != nil {
		context.Blockers = append(context.Blockers, "invalid review component path "+path+": "+err.Error())
		return
	}
	longest := -1
	matches := []reviewRepoEntry{}
	for _, entry := range entries {
		root := strings.TrimSuffix(entry.Path, "/")
		if clean != root && !strings.HasPrefix(clean, root+"/") {
			continue
		}
		if len(root) > longest {
			longest, matches = len(root), []reviewRepoEntry{entry}
		} else if len(root) == longest {
			matches = append(matches, entry)
		}
	}
	switch len(matches) {
	case 0:
		unmapped["path:"+clean] = true
	case 1:
		add(matches[0], source)
	default:
		context.Blockers = append(context.Blockers, "ambiguous review component path "+clean)
	}
}

func matchReviewOverlay(selectors ReviewProfileSelectors, context reviewPlanContext) ([]string, string, int) {
	category, order := "language", 1
	if len(selectors.Domains) > 0 {
		category, order = "domain", 2
	}
	if len(selectors.ComponentIDs) > 0 {
		category, order = "component", 3
	}
	if len(selectors.DeliveryKinds)+len(selectors.Criticalities) > 0 {
		category, order = "delivery", 4
	}
	if len(selectors.StructuralKinds) > 0 {
		category, order = "structure", 5
	}
	if len(selectors.DeliveryKinds) > 0 && !intersectsFold(selectors.DeliveryKinds, context.DeliveryKinds) {
		return nil, category, order
	}
	if len(selectors.StructuralKinds) > 0 && !intersectsFold(selectors.StructuralKinds, context.StructuralKinds) {
		return nil, category, order
	}
	matched := []string{}
	for _, component := range context.Components {
		if len(selectors.Languages) > 0 && !containsFold(selectors.Languages, component.Language) {
			continue
		}
		if len(selectors.Domains) > 0 && !containsFold(selectors.Domains, component.Domain) {
			continue
		}
		if len(selectors.ComponentIDs) > 0 && !containsFold(selectors.ComponentIDs, component.ID) && !containsFold(selectors.ComponentIDs, component.Path) {
			continue
		}
		if len(selectors.Criticalities) > 0 && !containsFold(selectors.Criticalities, component.Criticality) {
			continue
		}
		matched = append(matched, component.Path)
	}
	if len(context.Components) == 0 && len(selectors.Languages)+len(selectors.Domains)+len(selectors.ComponentIDs)+len(selectors.Criticalities) > 0 {
		return nil, category, order
	}
	if len(matched) == 0 && len(selectors.Languages)+len(selectors.Domains)+len(selectors.ComponentIDs)+len(selectors.Criticalities) > 0 {
		return nil, category, order
	}
	if len(selectors.DeliveryKinds)+len(selectors.StructuralKinds) > 0 && len(matched) == 0 && len(selectors.Languages)+len(selectors.Domains)+len(selectors.ComponentIDs)+len(selectors.Criticalities) == 0 {
		return []string{}, category, order
	}
	return uniqueSorted(matched), category, order
}

// loadReviewProfileFrom reads a profile from the working tree, or from a
// protected revision when the tree's policy no longer governs this scope.
func (s Store) loadReviewProfileFrom(revision, ref string) (ReviewProfile, error) {
	if revision == "" {
		profile, _, err := s.loadReviewProfile(ref)
		return profile, err
	}
	return s.parseReviewProfileAt(revision, ref)
}

// reviewOverlaySelection is one overlay that matched, with the selection record
// the plan publishes for it. Selection is kept in a function of its own so the
// declared-scope projection resolves through exactly this code and not a second
// reading of the same selectors.
type reviewOverlaySelection struct {
	profile   ReviewProfile
	selection ReviewPlanProfile
	order     int
	component string
}

func selectReviewOverlays(overlays []ReviewProfile, context reviewPlanContext) []reviewOverlaySelection {
	selected := []reviewOverlaySelection{}
	for _, overlay := range overlays {
		matched, category, order := matchReviewOverlay(overlay.Selectors, context)
		if matched == nil {
			continue
		}
		selection := ReviewPlanProfile{Ref: overlay.Ref(), Category: category, Order: order, Source: ".pose/review-profiles/" + overlay.ID + ".json", Components: matched, Rationale: "matched typed " + category + " selector"}
		component := ""
		if len(matched) > 0 {
			component = matched[0]
		}
		selected = append(selected, reviewOverlaySelection{overlay, selection, order, component})
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].order != selected[j].order {
			return selected[i].order < selected[j].order
		}
		if selected[i].order == 3 && selected[i].component != selected[j].component {
			return selected[i].component < selected[j].component
		}
		return selected[i].profile.Ref() < selected[j].profile.Ref()
	})
	return selected
}

// undecidableReviewOverlays returns the adopted, scope-compatible overlays that
// did not match, so the band summary can state which ones stayed undecided.
func undecidableReviewOverlays(overlays []ReviewProfile, selected []reviewOverlaySelection) []ReviewProfile {
	matched := map[string]bool{}
	for _, item := range selected {
		matched[item.profile.Ref()] = true
	}
	rest := []ReviewProfile{}
	for _, overlay := range overlays {
		if !matched[overlay.Ref()] {
			rest = append(rest, overlay)
		}
	}
	return rest
}

func addCrossComponentReviewCriterion(criteria []ReviewPlanCriterion, blockers []string, components int) ([]ReviewPlanCriterion, []string, bool) {
	if components <= 1 {
		return criteria, blockers, false
	}
	criteria, blockers = addReviewCriterion(criteria, ReviewPlanCriterion{
		ID: "cross-component-integration", Description: "Observed component boundaries and contracts are integrated and covered by current evidence.",
		Required: true, Kind: ReviewCriterionKindMechanical, EvidenceClasses: []string{"integration"}, Profiles: []string{"synthetic:cross-component"},
	}, blockers)
	return criteria, blockers, true
}

// declaredReviewContext drops the components the plan learned about only from
// the delivery integrity graph. What remains is the scope the author declared,
// which is the honest input for a pre-implementation forecast.
func declaredReviewContext(context reviewPlanContext) reviewPlanContext {
	projected := context
	// An observation is not a forecast. Dropping the observed structural kinds
	// is what makes a structural trigger show up as an expansion rather than as
	// something the author could have declared up front.
	projected.StructuralKinds = nil
	projected.Components = []ReviewPlanComponent{}
	for _, component := range context.Components {
		if reviewComponentOrigin(component.Sources) == reviewBasisObserved {
			continue
		}
		projected.Components = append(projected.Components, component)
	}
	return projected
}

// projectDeclaredReviewObligations resolves the plan again over declared scope
// alone, then reports what the observed scope added. It reuses the selection,
// composition and tool builders above verbatim: the difference is the input, not
// the engine, so a delta here is always attributable to a fact and never to a
// second set of rules.
func projectDeclaredReviewObligations(plan ReviewPlan, scope ScopeRef, base ReviewProfile, floor string, overlays []ReviewProfile, context reviewPlanContext) ReviewPlanProjection {
	declared := declaredReviewContext(context)
	profiles := []ReviewProfile{base}
	selections := []ReviewPlanProfile{{Ref: base.Ref(), Category: "base", Order: 0, Source: ".pose/review-profiles/" + base.ID + ".json", Rationale: "selected by terminal scope policy"}}
	independence := floor
	selected := selectReviewOverlays(overlays, declared)
	for _, item := range selected {
		profiles = append(profiles, item.profile)
		selections = append(selections, item.selection)
		independence = stricterReviewIndependence(independence, item.profile.Independence)
	}
	criteria, _ := composeReviewCriteria(profiles, nil)
	criteria, _, _ = addCrossComponentReviewCriterion(criteria, nil, len(declared.Components))
	tools, _, _ := buildReviewTools(scope, declared, profiles, selections, criteria, nil, nil)

	forecast := ReviewPlan{
		BaseProfile: plan.BaseProfile, Components: declared.Components, SelectedProfiles: selections,
		Criteria: criteria, Independence: independence,
	}
	_, band := deriveReviewBands(forecast, scope.Kind, floor, selected, nil, declared)

	projection := ReviewPlanProjection{
		Basis: ReviewProjectionBasis, Band: band, Independence: independence,
		Profiles: reviewProfileRefs(selections), Criteria: reviewCriterionIDs(criteria), Tools: reviewToolKeys(tools),
	}
	for _, component := range plan.Components {
		if reviewComponentOrigin(component.Sources) == reviewBasisObserved {
			projection.ObservedComponents = append(projection.ObservedComponents, component.Path)
		}
	}
	projection.ObservedStructure = append([]string{}, context.StructuralKinds...)
	projection.ScopeExpanded = len(projection.ObservedComponents)+len(projection.ObservedStructure) > 0
	projection.AddedProfiles = missingFrom(reviewProfileRefs(plan.SelectedProfiles), projection.Profiles)
	projection.AddedCriteria = missingFrom(reviewCriterionIDs(plan.Criteria), projection.Criteria)
	projection.AddedTools = missingFrom(reviewToolKeys(plan.Tools), projection.Tools)
	if plan.Independence != independence {
		projection.RaisedIndependence = independence + " -> " + plan.Independence
	}
	if plan.Band != band {
		projection.RaisedBand = band + " -> " + plan.Band
	}
	return projection
}

func reviewProfileRefs(profiles []ReviewPlanProfile) []string {
	refs := []string{}
	for _, profile := range profiles {
		refs = append(refs, profile.Ref)
	}
	return uniqueSorted(refs)
}

func reviewCriterionIDs(criteria []ReviewPlanCriterion) []string {
	ids := []string{}
	for _, criterion := range criteria {
		ids = append(ids, criterion.ID)
	}
	return uniqueSorted(ids)
}

// reviewToolKeys names a tool by id and component, because the same tool scoped
// to a component the observed scope introduced is an additional obligation.
func reviewToolKeys(tools []ReviewPlanTool) []string {
	keys := []string{}
	for _, tool := range tools {
		key := tool.ID
		if tool.Component != "" {
			key += "@" + tool.Component
		}
		keys = append(keys, key)
	}
	return uniqueSorted(keys)
}

func missingFrom(values, known []string) []string {
	present := map[string]bool{}
	for _, value := range known {
		present[value] = true
	}
	missing := []string{}
	for _, value := range values {
		if !present[value] {
			missing = append(missing, value)
		}
	}
	return uniqueSorted(missing)
}

func composeReviewCriteria(profiles []ReviewProfile, blockers []string) ([]ReviewPlanCriterion, []string) {
	criteria := []ReviewPlanCriterion{}
	for _, profile := range profiles {
		for _, item := range profile.Criteria {
			required := item.Required == nil || *item.Required
			// No filter here any more. A profile that declares a class outside
			// the vocabulary now fails to load, so nothing unproducible reaches
			// this point — the blocker this used to raise, and the class-drop
			// tools used to receive, were both guarding against input the
			// contract no longer admits.
			classes := uniqueSorted(item.EvidenceClasses)
			kind := item.Kind
			if kind == "" {
				kind = DeriveReviewCriterionKind(classes)
			}
			criterion := ReviewPlanCriterion{ID: item.ID, Description: item.Description, Required: required, Kind: kind, Rules: uniqueSorted(item.Rules), EvidenceClasses: classes, RequiresStructuralMapping: item.RequiresStructuralMapping, Profiles: []string{profile.Ref()}}
			criteria, blockers = addReviewCriterion(criteria, criterion, blockers)
		}
	}
	sort.Slice(criteria, func(i, j int) bool { return criteria[i].ID < criteria[j].ID })
	return criteria, blockers
}

func addReviewCriterion(criteria []ReviewPlanCriterion, candidate ReviewPlanCriterion, blockers []string) ([]ReviewPlanCriterion, []string) {
	for i := range criteria {
		if criteria[i].ID != candidate.ID {
			continue
		}
		if criteria[i].Description != candidate.Description || criteria[i].Required != candidate.Required || strings.Join(criteria[i].Rules, "\x00") != strings.Join(candidate.Rules, "\x00") || strings.Join(criteria[i].EvidenceClasses, "\x00") != strings.Join(candidate.EvidenceClasses, "\x00") {
			return criteria, append(blockers, "conflicting review criterion "+candidate.ID+" from "+strings.Join(append(criteria[i].Profiles, candidate.Profiles...), ","))
		}
		// Kind composes monotonically, the way independence already does: an
		// overlay may raise a collected criterion to a judged one, and may never
		// lower a judged one back. Two profiles disagreeing is therefore not a
		// conflict — the stricter reading wins, and neither profile can weaken
		// the obligation the other stated.
		if ReviewCriterionKind(criteria[i]) == ReviewCriterionKindJudgment || ReviewCriterionKind(candidate) == ReviewCriterionKindJudgment {
			criteria[i].Kind = ReviewCriterionKindJudgment
		} else {
			criteria[i].Kind = ReviewCriterionKindMechanical
		}
		criteria[i].RequiresStructuralMapping = criteria[i].RequiresStructuralMapping || candidate.RequiresStructuralMapping
		criteria[i].Profiles = uniqueSorted(append(criteria[i].Profiles, candidate.Profiles...))
		return criteria, blockers
	}
	criteria = append(criteria, candidate)
	return criteria, blockers
}

// annotateReviewToolProducerCoverage marks the component-scoped tools whose
// component the validation matrix declares runs no check.
//
// It reads one declaration and infers nothing else. A component with checks, a
// component the matrix does not mention, an unreadable matrix: all of them keep
// the tool exactly as required as it was. The gap is claimed only where the
// repository itself wrote `replaceDefaultChecks` with an empty list, which is
// an explicit statement that this component emits no evidence.
func (s Store) annotateReviewToolProducerCoverage(tools []ReviewPlanTool, warnings []string) []string {
	silent := map[string]bool{}
	for _, tool := range tools {
		if tool.Component == "" || (len(tool.EvidenceClasses) == 0 && tool.ID != "validate") {
			continue
		}
		if _, known := silent[tool.Component]; !known {
			silent[tool.Component] = s.componentDeclaresNoChecks(tool.Component)
		}
		if !silent[tool.Component] {
			continue
		}
		for i := range tools {
			if tools[i].ID == tool.ID && tools[i].Component == tool.Component {
				tools[i].ProducerCoverage = "none"
			}
		}
		requested := "validation evidence"
		if len(tool.EvidenceClasses) > 0 {
			requested = "evidence of class " + strings.Join(tool.EvidenceClasses, "|")
		}
		warnings = append(warnings, "review tool "+reviewToolLabel(tool.ID, tool.Component)+" asks for "+
			requested+" from a component the validation matrix declares runs no check; record it not-used with the reason, or register a check for that component")
	}
	return warnings
}

// componentDeclaresNoChecks answers only for the explicit declaration. Anything
// it cannot read is false, because the conservative answer is to keep asking.
func (s Store) componentDeclaresNoChecks(component string) bool {
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "validation-matrix.json"))
	if err != nil {
		return false
	}
	var matrix struct {
		ModuleOverrides map[string]struct {
			ReplaceDefaultChecks bool `json:"replaceDefaultChecks"`
			Checks               []struct {
				Name string `json:"name"`
			} `json:"checks"`
		} `json:"moduleOverrides"`
	}
	if err := json.Unmarshal(raw, &matrix); err != nil {
		return false
	}
	override, ok := matrix.ModuleOverrides[component]
	return ok && override.ReplaceDefaultChecks && len(override.Checks) == 0
}

func buildReviewTools(scope ScopeRef, context reviewPlanContext, profiles []ReviewProfile, selected []ReviewPlanProfile, criteria []ReviewPlanCriterion, blockers, warnings []string) ([]ReviewPlanTool, []string, []string) {
	tools := []ReviewPlanTool{}
	// Which components each profile was selected for. A base profile carries no
	// component list and governs all of them; an overlay carries the components
	// its selector matched.
	profileComponents := map[string][]string{}
	for _, selection := range selected {
		profileComponents[selection.Ref] = selection.Components
	}
	// Evidence classes for a synthesised tool come from the profile that governs
	// it, never from a literal in this function: the profile is where a project
	// can reconcile them, and a literal here is unreachable by definition.
	//
	// Scoped to the component, so an overlay's classes do not leak onto a
	// component it never matched. Unioning across every selected profile would
	// let a Go API's validate disposition be satisfied by a web app's e2e
	// evidence, which is the opposite of component-level provenance. The
	// repository-wide tool (component "") keeps the full union, since it is the
	// one that answers for everything.
	profileEvidence := func(id, component string) []string {
		classes := []string{}
		for _, profile := range profiles {
			if component != "" {
				if matched, known := profileComponents[profile.Ref()]; known && len(matched) > 0 && !containsFold(matched, component) {
					continue
				}
			}
			for _, tool := range profile.Tools {
				if tool.ID == id {
					classes = append(classes, tool.EvidenceClasses...)
				}
			}
		}
		return uniqueSorted(classes)
	}
	add := func(id, requiredness, component string, evidence, criterionIDs []string) {
		definition, ok := reviewToolCatalog[id]
		if !ok {
			blockers = append(blockers, "unknown review tool "+id)
			return
		}
		args := reviewToolArgs(id, scope, component)
		candidate := ReviewPlanTool{ID: id, Requiredness: requiredness, Args: args, Rationale: definition.Rationale, EvidenceClasses: uniqueSorted(evidence), Criteria: uniqueSorted(criterionIDs), Component: component, Preconditions: reviewToolPreconditions(id)}
		key := id + "\x00" + component
		for i := range tools {
			if tools[i].ID+"\x00"+tools[i].Component != key {
				continue
			}
			if requiredness == "required" {
				tools[i].Requiredness = "required"
			}
			tools[i].EvidenceClasses = uniqueSorted(append(tools[i].EvidenceClasses, evidence...))
			tools[i].Criteria = uniqueSorted(append(tools[i].Criteria, criterionIDs...))
			tools[i].Preconditions = uniqueSorted(append(tools[i].Preconditions, reviewToolPreconditions(id)...))
			return
		}
		tools = append(tools, candidate)
	}
	for _, component := range context.Components {
		add("suggest-review", "recommended", component.Path, nil, nil)
		add("assess-discover", "recommended", component.Path, nil, nil)
		add("validate", "required", component.Path, profileEvidence("validate", component.Path), nil)
	}
	if len(context.Components) == 0 && len(context.DeliveryKinds) > 0 {
		add("validate", "required", "", profileEvidence("validate", ""), nil)
	}
	if scope.Kind == "spec" {
		// Recommended rather than required: a scope with no immutable subject
		// must remain able to prepare a truthful bundle with an explicit unknown
		// state, without a new mandatory gate.
		add("assess-design", "recommended", "", []string{"structure"}, nil)
	}
	add("assess-tech-debt", "recommended", "", nil, nil)
	if len(context.Components) > 1 {
		add("assess-integrate", "required", "", []string{"integration"}, []string{"cross-component-integration"})
	}
	if scope.Kind == "spec" && len(context.ArtifactPaths) > 0 {
		add("artifact-check", "required", "", nil, nil)
	}
	if intersectsFold(context.DeliveryKinds, []string{"surface", "capability"}) && scope.Kind == "spec" {
		add("surface-check", "required", "", []string{"reachability", "integration"}, nil)
	}
	if scope.Kind == "roadmap" {
		add("roadmap-check", "required", "", []string{"integration"}, nil)
	}
	for _, path := range context.ArtifactPaths {
		switch {
		case strings.HasPrefix(path, ".pose/knowledge/"):
			add("knowledge-check", "required", "", nil, nil)
		case strings.HasPrefix(path, ".agents/skills/") || strings.Contains(path, "/.agents/skills/") || strings.HasPrefix(path, ".pose/workflows/"):
			add("skills-check", "required", "", nil, nil)
		case strings.HasPrefix(path, ".pose/reports/history/"):
			add("history-check", "required", "", nil, nil)
		}
	}
	criterionSet := map[string]bool{}
	for _, criterion := range criteria {
		criterionSet[criterion.ID] = true
	}
	for _, profile := range profiles {
		for _, tool := range profile.Tools {
			for _, criterion := range tool.Criteria {
				if !criterionSet[criterion] {
					blockers = append(blockers, "review tool "+tool.ID+" references unknown criterion "+criterion)
				}
			}
			requiredness := tool.Requiredness
			if requiredness == "" {
				requiredness = "recommended"
			}
			add(tool.ID, requiredness, "", tool.EvidenceClasses, tool.Criteria)
			for i := range tools {
				if tools[i].ID == tool.ID && tools[i].Component == "" {
					tools[i].Preconditions = uniqueSorted(append(tools[i].Preconditions, tool.Preconditions...))
				}
			}
		}
	}
	add("review-check", "required", "", nil, nil)
	add("closeout-check", "required", "", nil, nil)
	sort.Slice(tools, func(i, j int) bool {
		left, right := reviewToolCatalog[tools[i].ID], reviewToolCatalog[tools[j].ID]
		if left.Phase != right.Phase {
			return left.Phase < right.Phase
		}
		if tools[i].ID != tools[j].ID {
			return tools[i].ID < tools[j].ID
		}
		return tools[i].Component < tools[j].Component
	})
	return tools, blockers, warnings
}

func reviewToolPreconditions(id string) []string {
	switch id {
	case "suggest-review", "assess-discover":
		return []string{"scope-authorized"}
	case "validate", "surface-check", "roadmap-check":
		return []string{"delivery-target-declared"}
	case "assess-integrate":
		return []string{"scope-authorized"}
	case "artifact-check":
		return []string{"artifacts-attributed"}
	case "review-check", "closeout-check":
		return []string{"review-complete"}
	default:
		return []string{"scope-authorized"}
	}
}

func reviewToolArgs(id string, scope ScopeRef, component string) []string {
	ref := scope.String()
	switch id {
	case "suggest-review":
		path := component
		if path == "" {
			path = "."
		}
		return []string{"pose", "suggest", "review", "--path", path}
	case "assess-discover":
		args := []string{"pose", "assess", "discover"}
		if component != "" {
			args = append(args, "--component", component)
		}
		return args
	case "assess-design":
		return []string{"pose", "assess", "design", "--spec", scope.Slug, "--json"}
	case "assess-integrate":
		return []string{"pose", "assess", "integrate"}
	case "assess-tech-debt":
		return []string{"pose", "assess", "tech-debt"}
	case "artifact-check":
		return []string{"pose", "artifact-check", "--spec", scope.Slug, "--strict"}
	case "validate":
		args := []string{"pose", "validate", "--strict"}
		if component != "" {
			args = append(args, "--module", component)
		}
		return args
	case "surface-check":
		return []string{"pose", "surface-check", "--spec", scope.Slug, "--strict"}
	case "roadmap-check":
		return []string{"pose", "roadmap-check", scope.Slug, "--strict"}
	case "history-check", "knowledge-check", "skills-check":
		return []string{"pose", id, "--strict"}
	case "review-check", "closeout-check":
		return []string{"pose", id, ref}
	default:
		return nil
	}
}

func digestReviewPlan(plan ReviewPlan) (string, error) {
	type digestComponent struct {
		ID, Path, Kind, Language, Domain, Criticality, ValidationProfile, MetadataStatus string
		Sources                                                                          []string
	}
	components := []digestComponent{}
	for _, component := range plan.Components {
		components = append(components, digestComponent{component.ID, component.Path, component.Kind, component.Language, component.Domain, component.Criticality, component.ValidationProfile, component.MetadataStatus, component.Sources})
	}
	// The material structural set is hashed, the coverage report is not. A
	// criterion answering for observed structure owes one answer per material
	// fact, so gaining a fact must stale a sealed review; a detector that learns
	// to report one more unknown must not. Omitted entirely when nothing selected
	// on structure, so a plan from before this contract keeps its digest.
	var structure *[]ReviewStructuralFact
	if plan.Structure != nil && len(plan.Structure.Material) > 0 {
		material := append([]ReviewStructuralFact{}, plan.Structure.Material...)
		structure = &material
	}
	return digestJSON(struct {
		SchemaVersion                                 int
		Scope, ScopeDigest, BaseProfile, Independence string
		PolicySchemaVersion                           int
		Components                                    []digestComponent
		Profiles                                      []ReviewPlanProfile
		Criteria                                      []ReviewPlanCriterion
		Tools                                         []ReviewPlanTool
		Warnings, Blockers                            []string
		Explain                                       []string
		Structure                                     *[]ReviewStructuralFact `json:",omitempty"`
	}{plan.SchemaVersion, plan.Scope, plan.ScopeDigest, plan.BaseProfile, plan.Independence, plan.PolicySchemaVersion, components, plan.SelectedProfiles, plan.Criteria, plan.Tools, plan.Warnings, plan.Blockers, plan.Explain, structure})
}

func validateReviewPath(root, value string) error {
	clean, err := validateArtifactPathSyntax(value)
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return err
	}
	candidate := filepath.Join(rootAbs, clean)
	for current := candidate; current != rootAbs; current = filepath.Dir(current) {
		if _, statErr := os.Lstat(current); statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return statErr
		}
		real, evalErr := filepath.EvalSymlinks(current)
		if evalErr != nil {
			return evalErr
		}
		rel, relErr := filepath.Rel(rootReal, real)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("symlink escapes project root")
		}
	}
	return nil
}

func uniqueStable(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func normalizeReviewIndependence(value string) string {
	if value == "" {
		return "same-actor-separate-execution"
	}
	return value
}

func stricterReviewIndependence(left, right string) string {
	rank := map[string]int{"same-actor-separate-execution": 1, "different-actor": 2, "mandatory-human": 3}
	left, right = normalizeReviewIndependence(left), normalizeReviewIndependence(right)
	if rank[right] > rank[left] {
		return right
	}
	return left
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func reviewSortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}

func intersectsFold(left, right []string) bool {
	for _, value := range left {
		if containsFold(right, value) {
			return true
		}
	}
	return false
}

func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

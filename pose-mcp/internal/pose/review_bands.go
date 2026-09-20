package pose

import (
	"strings"
)

// Bands and the declared forecast are explanations, never a second policy
// engine. Every value below is a projection of what the plan already resolved:
// the selected profiles, the composed criteria, the effective independence and
// the mapped components. Nothing here selects a profile, adds a criterion or
// raises a floor.
//
// That is also why none of it reaches digestReviewPlan. Given the same digested
// inputs the summary is a pure function of them, so shipping it cannot change a
// plan's identity and cannot supersede a review sealed before it existed. A
// summary that did change the digest would be indistinguishable, to a verifier,
// from a real change in obligations.
const (
	ReviewBandBaseline = "baseline"
	ReviewBandElevated = "elevated"
	ReviewBandCritical = "critical"
	// ReviewBandUnknown is a band entry that states an undecided selector, not
	// a level. It never raises the plan's band and never lowers it: an
	// unreadable criticality is not evidence of a low one.
	ReviewBandUnknown = "unknown"

	// ReviewProjectionBasis labels the pre-implementation reading as the
	// forecast it is. Declared scope is what the author asserts; observed scope
	// is what provenance attributes. The two are reported separately so a
	// forecast is never read back as a final observation.
	ReviewProjectionBasis = "declared-forecast"

	reviewBasisDeclared = "declared"
	reviewBasisObserved = "observed"
	reviewBasisBoth     = "declared+observed"
	reviewBasisPolicy   = "policy"
	reviewBasisUnknown  = "unknown"
)

var reviewBandRank = map[string]int{ReviewBandBaseline: 1, ReviewBandElevated: 2, ReviewBandCritical: 3}

// ReviewPlanBand explains one reason the effective plan is where it is.
type ReviewPlanBand struct {
	Band string `json:"band"`
	// Trigger is the fact, as selector=value pairs resolved against the scope.
	Trigger string `json:"trigger"`
	// Basis says whether the fact was declared by the author, observed from
	// provenance, read from policy, or left undecided.
	Basis string `json:"basis"`
	// Source is where the fact is readable back.
	Source string `json:"source"`
	// Policy is the ref that made the fact consequential.
	Policy string `json:"policy"`
	// Obligations are what that ref contributed: `criterion:<id>` entries and,
	// when the ref raised the floor, `independence:<value>`. An undecided entry
	// carries none, because uncertainty is shown and never charged.
	Obligations []string `json:"obligations"`
}

// ReviewPlanProjection is the same plan resolved over declared scope alone.
// It exists so a preflight forecast and a final observation cannot be confused,
// and so the obligations that only the observed scope produced are visible as
// such instead of appearing to have been foreseeable.
type ReviewPlanProjection struct {
	Basis        string   `json:"basis"`
	Band         string   `json:"band"`
	Independence string   `json:"independence"`
	Profiles     []string `json:"profiles"`
	Criteria     []string `json:"criteria"`
	Tools        []string `json:"tools"`

	ScopeExpanded      bool     `json:"scope_expanded"`
	ObservedComponents []string `json:"observed_components,omitempty"`
	AddedProfiles      []string `json:"added_profiles,omitempty"`
	AddedCriteria      []string `json:"added_criteria,omitempty"`
	AddedTools         []string `json:"added_tools,omitempty"`
	RaisedIndependence string   `json:"raised_independence,omitempty"`
	RaisedBand         string   `json:"raised_band,omitempty"`
}

// reviewComponentOrigin classifies a mapped component by where the plan learned
// about it. A spec, artifact or delivery-target source is declared; the delivery
// integrity graph is observed; a component reached both ways is both.
func reviewComponentOrigin(sources []string) string {
	declared, observed := false, false
	for _, source := range sources {
		if strings.HasPrefix(source, "observed:") {
			observed = true
			continue
		}
		declared = true
	}
	switch {
	case declared && observed:
		return reviewBasisBoth
	case observed:
		return reviewBasisObserved
	default:
		return reviewBasisDeclared
	}
}

// reviewComponentUnknownFields lists the metadata fields the plan refused to
// report for a component. clearIncompleteReviewMetadata already blanked them;
// this recovers which ones so an undecided selector can name its own gap.
func reviewComponentUnknownFields(component ReviewPlanComponent) []string {
	if !metadataIncomplete(component) {
		return nil
	}
	if len(component.MetadataMissing) == 0 {
		return []string{"criticality", "domain", "language", "owner", "validation_profile"}
	}
	fields := []string{}
	for _, field := range component.MetadataMissing {
		switch strings.ToLower(field) {
		case "validationprofile", "validation_profile":
			fields = append(fields, "validation_profile")
		default:
			fields = append(fields, strings.ToLower(field))
		}
	}
	return uniqueSorted(fields)
}

// reviewOverlayUndecided returns the selector fields an overlay could not
// decide for a component: every field it does know matches, and at least one it
// selects on is unreadable. An overlay excluded by a field it can read is not
// undecided, it simply does not apply.
func reviewOverlayUndecided(selectors ReviewProfileSelectors, component ReviewPlanComponent) []string {
	unknown := map[string]bool{}
	for _, field := range reviewComponentUnknownFields(component) {
		unknown[field] = true
	}
	undecided := []string{}
	decided := true
	// An empty value counts as undecided even when the metadata status says
	// `declared`: a component that declares no criticality has not declared a
	// low one. A value present but outside the selector's vocabulary is a
	// different problem — metadata validity — and is left to the checks that
	// own it rather than reinterpreted here as uncertainty.
	check := func(values []string, field, actual string) {
		if len(values) == 0 {
			return
		}
		if unknown[field] || actual == "" {
			undecided = append(undecided, field)
			return
		}
		if !containsFold(values, actual) {
			decided = false
		}
	}
	check(selectors.Languages, "language", component.Language)
	check(selectors.Domains, "domain", component.Domain)
	check(selectors.Criticalities, "criticality", component.Criticality)
	if len(selectors.ComponentIDs) > 0 && !containsFold(selectors.ComponentIDs, component.ID) && !containsFold(selectors.ComponentIDs, component.Path) {
		decided = false
	}
	if !decided {
		return nil
	}
	return uniqueSorted(undecided)
}

func reviewCriteriaForProfile(criteria []ReviewPlanCriterion, ref string) []string {
	obligations := []string{}
	for _, criterion := range criteria {
		if containsFold(criterion.Profiles, ref) {
			obligations = append(obligations, "criterion:"+criterion.ID)
		}
	}
	return uniqueSorted(obligations)
}

// reviewOverlayFacts resolves the selector facts that actually matched, so a
// trigger names the observation and not the whole selector block.
func reviewOverlayFacts(selectors ReviewProfileSelectors, matched []string, components map[string]ReviewPlanComponent, deliveryKinds []string) (string, string, bool) {
	facts, criticalities := []string{}, []string{}
	values := func(pick func(ReviewPlanComponent) string, accepted []string) []string {
		found := []string{}
		for _, path := range matched {
			component, ok := components[path]
			if !ok {
				continue
			}
			if value := pick(component); value != "" && containsFold(accepted, value) {
				found = append(found, value)
			}
		}
		return uniqueSorted(found)
	}
	if len(selectors.DeliveryKinds) > 0 {
		kinds := []string{}
		for _, kind := range deliveryKinds {
			if containsFold(selectors.DeliveryKinds, kind) {
				kinds = append(kinds, kind)
			}
		}
		facts = append(facts, "delivery_kind="+strings.Join(uniqueSorted(kinds), ","))
	}
	if len(selectors.Languages) > 0 {
		facts = append(facts, "language="+strings.Join(values(func(c ReviewPlanComponent) string { return c.Language }, selectors.Languages), ","))
	}
	if len(selectors.Domains) > 0 {
		facts = append(facts, "domain="+strings.Join(values(func(c ReviewPlanComponent) string { return c.Domain }, selectors.Domains), ","))
	}
	if len(selectors.ComponentIDs) > 0 {
		facts = append(facts, "component="+strings.Join(uniqueSorted(matched), ","))
	}
	if len(selectors.Criticalities) > 0 {
		criticalities = values(func(c ReviewPlanComponent) string { return c.Criticality }, selectors.Criticalities)
		facts = append(facts, "criticality="+strings.Join(criticalities, ","))
	}
	escalating := containsFold(criticalities, "high") || containsFold(criticalities, "critical")
	basis := reviewBasisDeclared
	if len(matched) > 0 {
		origins := []string{}
		for _, path := range matched {
			if component, ok := components[path]; ok {
				origins = append(origins, reviewComponentOrigin(component.Sources))
			}
		}
		origins = uniqueSorted(origins)
		switch {
		case len(origins) == 1:
			basis = origins[0]
		case len(origins) > 1:
			basis = reviewBasisBoth
		}
	}
	return strings.Join(facts, " "), basis, escalating
}

// deriveReviewBands summarizes a resolved plan. floor is the policy's own
// independence for the scope, before any overlay raised it.
func deriveReviewBands(plan ReviewPlan, scopeKind, floor string, selected []reviewOverlaySelection, undecidable []ReviewProfile, context reviewPlanContext) ([]ReviewPlanBand, string) {
	components := map[string]ReviewPlanComponent{}
	for _, component := range plan.Components {
		components[component.Path] = component
	}
	bands := []ReviewPlanBand{}
	baseRef := plan.BaseProfile
	if len(plan.SelectedProfiles) > 0 {
		baseRef = plan.SelectedProfiles[0].Ref
	}
	bands = append(bands, ReviewPlanBand{
		Band: ReviewBandBaseline, Trigger: "scope_kind=" + scopeKind, Basis: reviewBasisPolicy,
		Source: ".pose/policy/review.json", Policy: baseRef,
		Obligations: append(reviewCriteriaForProfile(plan.Criteria, baseRef), "independence:"+floor),
	})
	effective := floor
	for _, item := range selected {
		ref := item.profile.Ref()
		raised := stricterReviewIndependence(effective, item.profile.Independence) != effective
		trigger, basis, escalating := reviewOverlayFacts(item.profile.Selectors, item.selection.Components, components, context.DeliveryKinds)
		band := ReviewBandElevated
		if escalating || raised {
			band = ReviewBandCritical
		}
		sources := []string{}
		if len(item.profile.Selectors.DeliveryKinds) > 0 {
			sources = append(sources, "delivery-target:"+strings.Join(context.SpecSlugs, ","))
		}
		if len(item.selection.Components) > 0 {
			sources = append(sources, "component:"+strings.Join(item.selection.Components, ","))
		}
		obligations := reviewCriteriaForProfile(plan.Criteria, ref)
		if raised {
			obligations = append(obligations, "independence:"+normalizeReviewIndependence(item.profile.Independence))
		}
		bands = append(bands, ReviewPlanBand{
			Band: band, Trigger: trigger, Basis: basis,
			Source: strings.Join(sources, " "), Policy: ref, Obligations: obligations,
		})
		effective = stricterReviewIndependence(effective, item.profile.Independence)
	}
	for _, overlay := range undecidable {
		if len(overlay.Selectors.DeliveryKinds) > 0 && !intersectsFold(overlay.Selectors.DeliveryKinds, context.DeliveryKinds) {
			continue
		}
		for _, component := range plan.Components {
			fields := reviewOverlayUndecided(overlay.Selectors, component)
			if len(fields) == 0 {
				continue
			}
			facts := []string{}
			for _, field := range fields {
				facts = append(facts, field+"=unknown")
			}
			bands = append(bands, ReviewPlanBand{
				Band: ReviewBandUnknown, Trigger: strings.Join(facts, " "), Basis: reviewBasisUnknown,
				Source:      "component:" + component.Path + " metadata:" + firstNonempty(component.MetadataStatus, "absent"),
				Policy:      overlay.Ref(),
				Obligations: []string{},
			})
		}
	}
	highest := ReviewBandBaseline
	for _, band := range bands {
		if reviewBandRank[band.Band] > reviewBandRank[highest] {
			highest = band.Band
		}
	}
	return bands, highest
}

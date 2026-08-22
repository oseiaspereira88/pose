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
	Sources           []string `json:"sources"`

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
	ID              string   `json:"id"`
	Description     string   `json:"description"`
	Required        bool     `json:"required"`
	Rules           []string `json:"rules,omitempty"`
	EvidenceClasses []string `json:"evidence_classes,omitempty"`
	Profiles        []string `json:"profiles"`
}

type ReviewPlanTool struct {
	ID              string   `json:"id"`
	Requiredness    string   `json:"requiredness"`
	Args            []string `json:"args"`
	Rationale       string   `json:"rationale"`
	EvidenceClasses []string `json:"evidence_classes,omitempty"`
	Criteria        []string `json:"criteria,omitempty"`
	Component       string   `json:"component,omitempty"`
	Preconditions   []string `json:"preconditions,omitempty"`
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
	Warnings            []string              `json:"warnings,omitempty"`
	Blockers            []string              `json:"blockers,omitempty"`
	Explain             []string              `json:"explain"`
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
	Warnings      []string
	Blockers      []string
}

type reviewToolDefinition struct {
	Rationale string
	Phase     int
}

var reviewToolCatalog = map[string]reviewToolDefinition{
	"suggest-review":   {Rationale: "resolve the component-specific workflow, skill, rules and validation trail", Phase: 10},
	"assess-discover":  {Rationale: "inspect component structure, language, size and visible debt", Phase: 20},
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

var reviewEvidenceClassCatalog = map[string]bool{
	"a11y": true, "build": true, "contract": true, "e2e": true,
	"integration": true, "integration-test": true, "lint": true,
	"manual-review": true, "observability": true, "reachability": true,
	"requirement-trace": true, "security-scan": true, "test": true,
	"typecheck": true, "unit": true, "validation": true,
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
	if !policy.Enabled {
		return ReviewPlan{}, fmt.Errorf("pose: review policy is absent or disabled")
	}
	baseRef := policy.Profiles[scope.Kind]
	if baseRef == "" {
		return ReviewPlan{}, fmt.Errorf("pose: no review profile configured for %s", scope.Kind)
	}
	base, _, err := s.loadReviewProfile(baseRef)
	if err != nil {
		return ReviewPlan{}, err
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

		selected := []struct {
			profile   ReviewProfile
			selection ReviewPlanProfile
			order     int
			component string
		}{}
		for _, overlayRef := range uniqueSorted(policy.OverlayProfiles) {
			overlay, _, loadErr := s.loadReviewProfile(overlayRef)
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
			matched, category, order := matchReviewOverlay(overlay.Selectors, context)
			if matched == nil {
				continue
			}
			selection := ReviewPlanProfile{Ref: overlay.Ref(), Category: category, Order: order, Source: ".pose/review-profiles/" + overlay.ID + ".json", Components: matched, Rationale: "matched typed " + category + " selector"}
			component := ""
			if len(matched) > 0 {
				component = matched[0]
			}
			selected = append(selected, struct {
				profile   ReviewProfile
				selection ReviewPlanProfile
				order     int
				component string
			}{overlay, selection, order, component})
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
		for _, item := range selected {
			profiles = append(profiles, item.profile)
			plan.SelectedProfiles = append(plan.SelectedProfiles, item.selection)
			plan.Explain = append(plan.Explain, item.profile.Ref()+" selected: "+item.selection.Rationale)
			plan.Independence = stricterReviewIndependence(plan.Independence, item.profile.Independence)
		}
	}

	plan.Criteria, plan.Blockers = composeReviewCriteria(profiles, plan.Blockers)
	if len(plan.Components) > 1 {
		plan.Criteria, plan.Blockers = addReviewCriterion(plan.Criteria, ReviewPlanCriterion{
			ID: "cross-component-integration", Description: "Observed component boundaries and contracts are integrated and covered by current evidence.",
			Required: true, EvidenceClasses: []string{"integration"}, Profiles: []string{"synthetic:cross-component"},
		}, plan.Blockers)
		plan.Explain = append(plan.Explain, "cross-component-integration added because multiple mapped component roots are affected")
	}
	plan.Tools, plan.Blockers = buildReviewTools(scope, context, profiles, plan.Criteria, plan.Blockers)
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
	if len(selectors.DeliveryKinds) > 0 && !intersectsFold(selectors.DeliveryKinds, context.DeliveryKinds) {
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
	if len(selectors.DeliveryKinds) > 0 && len(matched) == 0 && len(selectors.Languages)+len(selectors.Domains)+len(selectors.ComponentIDs)+len(selectors.Criticalities) == 0 {
		return []string{}, category, order
	}
	return uniqueSorted(matched), category, order
}

func composeReviewCriteria(profiles []ReviewProfile, blockers []string) ([]ReviewPlanCriterion, []string) {
	criteria := []ReviewPlanCriterion{}
	for _, profile := range profiles {
		for _, item := range profile.Criteria {
			required := item.Required == nil || *item.Required
			criterion := ReviewPlanCriterion{ID: item.ID, Description: item.Description, Required: required, Rules: uniqueSorted(item.Rules), EvidenceClasses: uniqueSorted(item.EvidenceClasses), Profiles: []string{profile.Ref()}}
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
		criteria[i].Profiles = uniqueSorted(append(criteria[i].Profiles, candidate.Profiles...))
		return criteria, blockers
	}
	criteria = append(criteria, candidate)
	return criteria, blockers
}

func buildReviewTools(scope ScopeRef, context reviewPlanContext, profiles []ReviewProfile, criteria []ReviewPlanCriterion, blockers []string) ([]ReviewPlanTool, []string) {
	tools := []ReviewPlanTool{}
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
		add("validate", "required", component.Path, []string{"validation"}, nil)
	}
	if len(context.Components) == 0 && len(context.DeliveryKinds) > 0 {
		add("validate", "required", "", []string{"validation"}, nil)
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
	return tools, blockers
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
	}{plan.SchemaVersion, plan.Scope, plan.ScopeDigest, plan.BaseProfile, plan.Independence, plan.PolicySchemaVersion, components, plan.SelectedProfiles, plan.Criteria, plan.Tools, plan.Warnings, plan.Blockers, plan.Explain})
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

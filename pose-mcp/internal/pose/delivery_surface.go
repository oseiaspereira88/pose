package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var deliveryRefRE = regexp.MustCompile(`^(surface|contract|capability|infrastructure|governance):([a-z0-9][a-z0-9._-]*)$`)
var roadmapCriterionRE = regexp.MustCompile(`^\s*-\s*(C[0-9]+):\s*(.+?)\s*$`)
var criterionRefRE = regexp.MustCompile(`\b(surface|contract|capability|infrastructure|governance|check|evidence|manual-review):[^\s,;)\]]+`)

var ValidEvidenceClasses = map[string]bool{
	"build": true, "unit": true, "integration": true, "e2e": true,
	"reachability": true, "a11y": true, "design-system": true,
	"contrast": true, "visual-regression": true,
}

type DeliveryPolicy struct {
	SchemaVersion int               `json:"schema_version"`
	Enabled       bool              `json:"enabled"`
	AdoptedAt     string            `json:"adopted_at"`
	ResultsPath   string            `json:"results_path"`
	Roots         []DeliveryRoot    `json:"roots"`
	Severities    map[string]string `json:"severities"`
}

type DeliveryRoot struct {
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Profile    string `json:"profile"`
	Entrypoint string `json:"entrypoint"`
}

type DeliveryProfile struct {
	Kind                    string   `json:"kind"`
	RequiredEvidenceClasses []string `json:"requiredEvidenceClasses"`
	AnyEvidenceClasses      []string `json:"anyEvidenceClasses,omitempty"`
}

type DeliveryTarget struct {
	Spec       string `json:"spec"`
	Ref        string `json:"ref"`
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Module     string `json:"module"`
	Profile    string `json:"profile"`
	Entrypoint string `json:"entrypoint"`
}

type DeliveryValidationResult struct {
	ID               string `json:"id"`
	Module           string `json:"module"`
	Check            string `json:"check"`
	EvidenceClass    string `json:"evidence_class"`
	Severity         string `json:"severity,omitempty"`
	Outcome          string `json:"outcome"`
	GitHead          string `json:"git_head,omitempty"`
	GeneratedAt      string `json:"generated_at,omitempty"`
	ProvenanceDigest string `json:"provenance_digest,omitempty"`
	Report           string `json:"report,omitempty"`
}

type RoadmapCriterion struct {
	Roadmap string   `json:"roadmap"`
	ID      string   `json:"id"`
	Refs    []string `json:"refs"`
	Raw     string   `json:"raw"`
	Passed  bool     `json:"passed"`
	Reasons []string `json:"reasons,omitempty"`
}

func LoadDeliveryPolicy(root string) (DeliveryPolicy, error) {
	path := filepath.Join(root, ".pose", "policy", "delivery.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DeliveryPolicy{}, nil
	}
	if err != nil {
		return DeliveryPolicy{}, err
	}
	var policy DeliveryPolicy
	if err := json.Unmarshal(raw, &policy); err != nil || policy.SchemaVersion != DeliveryIntegritySchemaVersion {
		return DeliveryPolicy{}, fmt.Errorf("pose: invalid delivery policy")
	}
	var metadata struct {
		Modules map[string]map[string]string `json:"modules"`
	}
	if metadataRaw, readErr := os.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json")); readErr == nil && json.Unmarshal(metadataRaw, &metadata) == nil {
		for module, values := range metadata.Modules {
			if values["deliveryRole"] == "" || values["entrypoints"] == "" {
				continue
			}
			entrypoint := strings.TrimSpace(strings.Split(values["entrypoints"], ",")[0])
			derived := DeliveryRoot{Path: module, Kind: "capability", Profile: "composed-capability", Entrypoint: entrypoint}
			if values["deliveryRole"] == "product-surface" {
				derived.Kind, derived.Profile = "surface", "cli-surface"
			}
			duplicate := false
			for _, existing := range policy.Roots {
				if existing.Path == derived.Path {
					duplicate = true
				}
			}
			if !duplicate {
				policy.Roots = append(policy.Roots, derived)
			}
		}
	}
	sort.Slice(policy.Roots, func(i, j int) bool { return policy.Roots[i].Path < policy.Roots[j].Path })
	if policy.ResultsPath != "" {
		if err := ValidateArtifactPath(root, policy.ResultsPath, true); err != nil {
			return DeliveryPolicy{}, fmt.Errorf("pose: invalid delivery results path: %w", err)
		}
	}
	for _, item := range policy.Roots {
		if err := ValidateArtifactPath(root, item.Path, true); err != nil {
			return DeliveryPolicy{}, fmt.Errorf("pose: invalid delivery root %q: %w", item.Path, err)
		}
		if _, _, err := ParseDeliveryRef(item.Kind + ":root"); err != nil || item.Profile == "" {
			return DeliveryPolicy{}, fmt.Errorf("pose: invalid delivery root kind/profile for %q", item.Path)
		}
		if err := ValidateArtifactPath(root, item.Entrypoint, false); err != nil {
			return DeliveryPolicy{}, fmt.Errorf("pose: invalid delivery entrypoint %q: %w", item.Entrypoint, err)
		}
	}
	return policy, nil
}

func ParseDeliveryRef(value string) (kind, id string, err error) {
	m := deliveryRefRE.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return "", "", fmt.Errorf("invalid delivery ref %q", value)
	}
	return m[1], m[2], nil
}

func ParseDeliveryTargets(spec Spec) ([]DeliveryTarget, bool, error) {
	lines := strings.Split(strings.ReplaceAll(spec.Body, "\r\n", "\n"), "\n")
	inSection, found := false, false
	inFence := false
	targets := []DeliveryTarget{}
	seen := map[string]bool{}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if line == "### Delivery targets" {
			inSection, found = true, true
			continue
		}
		if inSection && strings.HasPrefix(line, "### ") {
			break
		}
		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "- "))
		if len(fields) != 4 {
			return nil, found, fmt.Errorf("pose: spec %s has malformed delivery target %q", spec.Slug, line)
		}
		kind, id, err := ParseDeliveryRef(fields[0])
		if err != nil || seen[fields[0]] {
			return nil, found, fmt.Errorf("pose: spec %s has invalid or duplicate delivery ref %q", spec.Slug, fields[0])
		}
		values := map[string]string{}
		for _, field := range fields[1:] {
			key, value, ok := strings.Cut(field, ":")
			if !ok || value == "" || values[key] != "" {
				return nil, found, fmt.Errorf("pose: spec %s has malformed delivery field %q", spec.Slug, field)
			}
			values[key] = value
		}
		if values["module"] == "" || values["profile"] == "" || values["entrypoint"] == "" || len(values) != 3 {
			return nil, found, fmt.Errorf("pose: spec %s delivery target requires module, profile and entrypoint", spec.Slug)
		}
		seen[fields[0]] = true
		targets = append(targets, DeliveryTarget{Spec: spec.Slug, Ref: fields[0], Kind: kind, ID: id, Module: filepath.ToSlash(filepath.Clean(values["module"])), Profile: values["profile"], Entrypoint: filepath.ToSlash(filepath.Clean(values["entrypoint"]))})
	}
	declared := append([]string{}, spec.Delivers...)
	sort.Strings(declared)
	bodyRefs := make([]string, 0, len(targets))
	for _, target := range targets {
		bodyRefs = append(bodyRefs, target.Ref)
	}
	sort.Strings(bodyRefs)
	if strings.Join(declared, "\x00") != strings.Join(bodyRefs, "\x00") {
		return nil, found, fmt.Errorf("pose: spec %s delivers frontmatter and Delivery targets must contain the exact same refs", spec.Slug)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Ref < targets[j].Ref })
	return targets, found, nil
}

func ValidateDeliveryTargets(root string, targets []DeliveryTarget, profiles map[string]DeliveryProfile) error {
	for _, target := range targets {
		if err := ValidateArtifactPath(root, target.Module, true); err != nil {
			return fmt.Errorf("%s module: %w", target.Ref, err)
		}
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(target.Module))); err != nil || !info.IsDir() {
			return fmt.Errorf("%s module must resolve to an existing directory", target.Ref)
		}
		if err := ValidateArtifactPath(root, target.Entrypoint, false); err != nil {
			return fmt.Errorf("%s entrypoint: %w", target.Ref, err)
		}
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(target.Entrypoint))); err != nil || info.IsDir() {
			return fmt.Errorf("%s entrypoint must resolve to an existing file", target.Ref)
		}
		profile, ok := profiles[target.Profile]
		if !ok || profile.Kind != target.Kind {
			return fmt.Errorf("%s uses unknown or incompatible profile %q", target.Ref, target.Profile)
		}
	}
	return nil
}

func ValidateDeliveryTrace(spec Spec, targets []DeliveryTarget, specStatuses map[string]string) []string {
	trace := ParseRequirementTrace(spec.Body)
	errors := []string{}
	for _, requirement := range trace.Requirements {
		if requirement.Entry == nil {
			continue
		}
		for _, ref := range requirement.Entry.Refs {
			if strings.HasPrefix(ref, "evidence:") {
				level := strings.TrimPrefix(ref, "evidence:")
				if !map[string]bool{"unit": true, "integration": true, "e2e": true, "manual": true}[level] {
					errors = append(errors, requirement.ID+": unknown evidence level "+level)
				}
			}
		}
	}
	for _, target := range targets {
		if target.Kind != "surface" {
			continue
		}
		delivered, deferred := false, false
		for _, requirement := range trace.Requirements {
			if requirement.Entry == nil || !containsDeliveryRef(requirement.Entry.Refs, target.Ref) {
				continue
			}
			if requirement.Entry.Disposition == "satisfied" && (containsDeliveryRef(requirement.Entry.Refs, "evidence:integration") || containsDeliveryRef(requirement.Entry.Refs, "evidence:e2e")) {
				delivered = true
			}
			if requirement.Entry.Disposition == "deferred-integration" {
				ref := strings.TrimSpace(requirement.Entry.Reason)
				if strings.HasPrefix(ref, "spec:") && specStatuses[strings.TrimPrefix(ref, "spec:")] != "" && specStatuses[strings.TrimPrefix(ref, "spec:")] != "done" {
					deferred = true
				} else {
					errors = append(errors, requirement.ID+": deferred-integration must reference an existing non-terminal spec")
				}
			}
		}
		if !delivered && !deferred {
			errors = append(errors, target.Ref+": surface requires a satisfied requirement with evidence:integration or evidence:e2e")
		}
	}
	sort.Strings(errors)
	return errors
}

func containsDeliveryRef(refs []string, wanted string) bool {
	for _, ref := range refs {
		if ref == wanted {
			return true
		}
	}
	return false
}

func deferredDeliveryTargets(specs []Spec) map[string]bool {
	statuses := map[string]string{}
	for _, spec := range specs {
		statuses[spec.Slug] = spec.Status
	}
	deferred := map[string]bool{}
	for _, spec := range specs {
		for _, requirement := range ParseRequirementTrace(spec.Body).Requirements {
			if requirement.Entry == nil || requirement.Entry.Disposition != "deferred-integration" {
				continue
			}
			owner := strings.TrimPrefix(strings.TrimSpace(requirement.Entry.Reason), "spec:")
			status, exists := statuses[owner]
			if !strings.HasPrefix(strings.TrimSpace(requirement.Entry.Reason), "spec:") || !exists || status == "done" {
				continue
			}
			for _, ref := range requirement.Entry.Refs {
				if deliveryRefRE.MatchString(ref) {
					deferred[spec.Slug+"\x00"+ref] = true
				}
			}
		}
	}
	return deferred
}

func ParseRoadmapCriteria(roadmap Roadmap) ([]RoadmapCriterion, []string) {
	criteria := []RoadmapCriterion{}
	errors := []string{}
	inSection := false
	for _, raw := range strings.Split(roadmap.Body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "## Cut criteria" {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "## ") {
			break
		}
		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}
		m := roadmapCriterionRE.FindStringSubmatch(line)
		if m == nil || strings.ContainsAny(m[2], "`|;&$<>") {
			errors = append(errors, "malformed or executable cut criterion: "+line)
			continue
		}
		refs := criterionRefRE.FindAllString(m[2], -1)
		if len(refs) == 0 {
			errors = append(errors, m[1]+": cut criterion requires a registered delivery, check or manual-review ref")
			continue
		}
		criteria = append(criteria, RoadmapCriterion{Roadmap: roadmap.Slug, ID: m[1], Refs: refs, Raw: m[2]})
	}
	return criteria, errors
}

func BuildDeliverySurface(graph DeliveryIntegrityGraph, specs []Spec, targets []DeliveryTarget, results []DeliveryValidationResult, roadmaps []Roadmap, profiles map[string]DeliveryProfile, policy DeliveryPolicy) DeliveryIntegrityGraph {
	graph.Deliveries = append([]DeliveryTarget{}, targets...)
	graph.ValidationResults = append([]DeliveryValidationResult{}, results...)
	sort.Slice(graph.Deliveries, func(i, j int) bool { return graph.Deliveries[i].Ref < graph.Deliveries[j].Ref })
	sort.Slice(graph.ValidationResults, func(i, j int) bool { return graph.ValidationResults[i].ID < graph.ValidationResults[j].ID })
	if graph.Paths == nil {
		graph.Paths = map[string][]string{}
	}
	claimsBySpec := map[string][]ArtifactClaim{}
	specCreatedAt := map[string]string{}
	setsBySpec := map[string][]ChangeSet{}
	observedBySpecPath := map[string]bool{}
	targetsBySpec := map[string][]DeliveryTarget{}
	targetByRef := map[string]DeliveryTarget{}
	deferredTargets := deferredDeliveryTargets(specs)
	for _, claim := range graph.Claims {
		claimsBySpec[claim.Spec] = append(claimsBySpec[claim.Spec], claim)
	}
	for _, spec := range specs {
		specCreatedAt[spec.Slug] = spec.CreatedAt
	}
	for _, set := range graph.ChangeSets {
		setsBySpec[set.Spec] = append(setsBySpec[set.Spec], set)
		for _, observed := range set.Paths {
			for _, path := range observedPaths(observed) {
				observedBySpecPath[set.Spec+"\x00"+path] = true
			}
		}
	}
	for _, result := range graph.ValidationResults {
		graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: "validation-result:" + result.ID, Type: "validation-result", Attributes: map[string]string{"check": result.Check, "evidence_class": result.EvidenceClass, "outcome": result.Outcome}})
	}
	for _, target := range graph.Deliveries {
		targetsBySpec[target.Spec] = append(targetsBySpec[target.Spec], target)
		targetByRef[target.Ref] = target
		targetNode := "delivery:" + target.Ref
		entryNode := "entrypoint:" + target.Entrypoint
		graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: targetNode, Type: target.Kind, Attributes: map[string]string{"module": target.Module, "profile": target.Profile}})
		if deferredTargets[target.Spec+"\x00"+target.Ref] {
			graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "spec:" + target.Spec, To: targetNode, Type: "defers"})
			graph.Paths[target.Ref] = []string{"spec:" + target.Spec, targetNode}
			continue
		}
		graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: entryNode, Type: "entrypoint", Attributes: map[string]string{"path": target.Entrypoint}})
		graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "spec:" + target.Spec, To: targetNode, Type: "delivers"})
		graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: targetNode, To: entryNode, Type: map[bool]string{true: "reaches", false: "composes"}[target.Kind == "surface"]})
		path := []string{"spec:" + target.Spec}
		provenanceLinks := 0
		for _, claim := range claimsBySpec[target.Spec] {
			if claim.Action == "none" {
				continue
			}
			artifact := claim.Path
			if claim.Action == "renamed" {
				artifact = claim.NewPath
			}
			if !observedBySpecPath[target.Spec+"\x00"+artifact] {
				continue
			}
			graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "artifact:" + artifact, To: targetNode, Type: "implemented-by"})
			path = append(path, "artifact:"+artifact)
			provenanceLinks++
		}
		if provenanceLinks == 0 {
			code := "unconnected-artifact"
			if target.Kind == "surface" {
				code = "surface-without-provenance"
			}
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding(code, deliverySeverity(policy, code), target.Spec, target.Ref, "", "delivery target has no structured artifact provenance", "declare and reconcile exact artifacts before asserting delivery"))
		}
		path = append(path, targetNode, entryNode)
		profile := profiles[target.Profile]
		classes := append([]string{}, profile.RequiredEvidenceClasses...)
		missing := []string{}
		for _, class := range classes {
			passed := false
			for _, result := range graph.ValidationResults {
				if result.EvidenceClass != class || !moduleMatchesTarget(result.Module, target.Module) {
					continue
				}
				if result.Outcome == "pass" && result.Severity == "required" && result.ProvenanceDigest == graph.ProvenanceDigest {
					passed = true
					graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: targetNode, To: "validation-result:" + result.ID, Type: "validated-by"})
					graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: entryNode, To: "validation-result:" + result.ID, Type: "validated-by"})
					path = append(path, "validation-result:"+result.ID)
				} else if result.Outcome == "pass" && result.ProvenanceDigest != graph.ProvenanceDigest {
					graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("stale-evidence", deliverySeverity(policy, "stale-evidence"), target.Spec, target.Ref, result.ID, "passed "+class+" evidence is bound to a different provenance digest", "rerun the registered validation check for the current source provenance"))
				}
			}
			if !passed {
				missing = append(missing, class)
			}
		}
		if len(profile.AnyEvidenceClasses) > 0 {
			any := false
			for _, class := range profile.AnyEvidenceClasses {
				for _, result := range graph.ValidationResults {
					if result.EvidenceClass != class || !moduleMatchesTarget(result.Module, target.Module) {
						continue
					}
					if result.Outcome == "pass" && result.Severity == "required" && result.ProvenanceDigest == graph.ProvenanceDigest {
						any = true
					}
				}
			}
			if !any {
				missing = append(missing, "one-of:"+strings.Join(profile.AnyEvidenceClasses, "|"))
			}
		}
		if len(missing) > 0 {
			code := "unconsumed-capability"
			if target.Kind == "surface" {
				code = "surface-without-reachability"
			}
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding(code, deliverySeverity(policy, code), target.Spec, target.Ref, "", "required current evidence is missing: "+strings.Join(missing, ", "), "run registered checks with the required evidenceClass against the current provenance digest"))
		}
		graph.Paths[target.Ref] = uniqueDeliveryPath(path)
	}
	for _, setList := range setsBySpec {
		for _, set := range setList {
			for _, observed := range set.Paths {
				path := firstObservedDeliveryPath(observed)
				if root, ok := matchingDeliveryRoot(path, policy.Roots); ok && len(targetsBySpec[set.Spec]) == 0 {
					level := deliverySeverity(policy, "undeclared-delivery")
					if policy.AdoptedAt != "" && specCreatedAt[set.Spec] < policy.AdoptedAt {
						level = "warning"
					}
					graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("undeclared-delivery", level, set.Spec, path, set.ID, "changed delivery root "+root.Path+" has no delivery declaration", "declare a typed delivery target or narrow the attributed change set"))
				}
			}
		}
	}
	for _, roadmap := range roadmaps {
		criteria, parseErrors := ParseRoadmapCriteria(roadmap)
		for _, message := range parseErrors {
			level := deliverySeverity(policy, "roadmap-criterion")
			if policy.AdoptedAt != "" && roadmap.CreatedAt < policy.AdoptedAt {
				level = "warning"
			}
			graph.Findings = append(graph.Findings, NewDeliveryIntegrityFinding("roadmap-criterion", level, "", roadmap.Slug, "", message, "reference registered delivery refs, checks or confined manual-review reports"))
		}
		for i := range criteria {
			criterion := &criteria[i]
			criterion.Passed = true
			for _, ref := range criterion.Refs {
				switch {
				case deliveryRefRE.MatchString(ref):
					target, ok := targetByRef[ref]
					if !ok {
						criterion.Passed = false
						criterion.Reasons = append(criterion.Reasons, "unknown delivery ref "+ref)
					} else if deferredTargets[target.Spec+"\x00"+ref] {
						criterion.Passed = false
						criterion.Reasons = append(criterion.Reasons, "deferred delivery ref "+ref)
					}
				case strings.HasPrefix(ref, "check:"):
					name := strings.TrimPrefix(ref, "check:")
					found := false
					for _, result := range results {
						if result.Check == name && result.Outcome == "pass" && result.ProvenanceDigest == graph.ProvenanceDigest {
							found = true
						}
					}
					if !found {
						criterion.Passed = false
						criterion.Reasons = append(criterion.Reasons, "check is missing, failing or stale: "+name)
					}
				case strings.HasPrefix(ref, "manual-review:"):
					path := strings.TrimPrefix(ref, "manual-review:")
					if _, err := validateArtifactPathSyntax(path); err != nil {
						criterion.Passed = false
						criterion.Reasons = append(criterion.Reasons, "invalid manual review ref")
					}
				}
			}
			graph.Nodes = appendNode(graph.Nodes, DeliveryIntegrityNode{ID: "roadmap-criterion:" + roadmap.Slug + "/" + criterion.ID, Type: "roadmap-criterion", Attributes: map[string]string{"passed": fmt.Sprint(criterion.Passed)}})
			for _, ref := range criterion.Refs {
				graph.Edges = append(graph.Edges, DeliveryIntegrityEdge{From: "roadmap-criterion:" + roadmap.Slug + "/" + criterion.ID, To: criterionNodeRef(ref), Type: "gates"})
			}
		}
		graph.RoadmapCriteria = append(graph.RoadmapCriteria, criteria...)
	}
	finalizeExtendedDeliveryGraph(&graph)
	return graph
}

func finalizeExtendedDeliveryGraph(graph *DeliveryIntegrityGraph) {
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].ID < graph.Nodes[j].ID })
	sort.Slice(graph.Edges, func(i, j int) bool {
		return graph.Edges[i].From+"\x00"+graph.Edges[i].Type+"\x00"+graph.Edges[i].To < graph.Edges[j].From+"\x00"+graph.Edges[j].Type+"\x00"+graph.Edges[j].To
	})
	sort.Slice(graph.Findings, func(i, j int) bool { return graph.Findings[i].ID < graph.Findings[j].ID })
	sort.Slice(graph.Deliveries, func(i, j int) bool { return graph.Deliveries[i].Ref < graph.Deliveries[j].Ref })
	sort.Slice(graph.ValidationResults, func(i, j int) bool { return graph.ValidationResults[i].ID < graph.ValidationResults[j].ID })
	sort.Slice(graph.RoadmapCriteria, func(i, j int) bool {
		return graph.RoadmapCriteria[i].Roadmap+"\x00"+graph.RoadmapCriteria[i].ID < graph.RoadmapCriteria[j].Roadmap+"\x00"+graph.RoadmapCriteria[j].ID
	})
	copy := *graph
	copy.InputDigest = ""
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	graph.InputDigest = "sha256:" + hex.EncodeToString(sum[:])
}

func deliverySeverity(policy DeliveryPolicy, code string) string {
	if value := policy.Severities[code]; value != "" {
		return value
	}
	return "warning"
}
func moduleMatchesTarget(module, target string) bool {
	return module == target || strings.HasPrefix(target, strings.TrimSuffix(module, "/")+"/") || strings.HasPrefix(module, strings.TrimSuffix(target, "/")+"/")
}
func firstObservedDeliveryPath(path ObservedPath) string {
	if path.Action == "renamed" {
		return path.NewPath
	}
	return path.Path
}
func matchingDeliveryRoot(path string, roots []DeliveryRoot) (DeliveryRoot, bool) {
	for _, root := range roots {
		clean := strings.TrimSuffix(root.Path, "/")
		if path == clean || strings.HasPrefix(path, clean+"/") {
			return root, true
		}
	}
	return DeliveryRoot{}, false
}
func uniqueDeliveryPath(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func criterionNodeRef(ref string) string {
	if strings.HasPrefix(ref, "check:") {
		return "validation-check:" + strings.TrimPrefix(ref, "check:")
	}
	if strings.HasPrefix(ref, "manual-review:") {
		return "manual-review:" + strings.TrimPrefix(ref, "manual-review:")
	}
	if deliveryRefRE.MatchString(ref) {
		return "delivery:" + ref
	}
	return "evidence:" + ref
}

func (s Store) GetSurfaceAssurance(ref, roadmap string) (DeliveryIntegrityGraph, error) {
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		return graph, err
	}
	if ref != "" {
		if _, _, err := ParseDeliveryRef(ref); err != nil {
			return graph, err
		}
		filtered := graph.Deliveries[:0]
		for _, target := range graph.Deliveries {
			if target.Ref == ref {
				filtered = append(filtered, target)
			}
		}
		graph.Deliveries = filtered
		paths := map[string][]string{ref: graph.Paths[ref]}
		graph.Paths = paths
	}
	if roadmap != "" {
		if ValidateSlug(roadmap) != nil {
			return graph, fmt.Errorf("pose: invalid roadmap slug %q", roadmap)
		}
		filtered := graph.RoadmapCriteria[:0]
		for _, criterion := range graph.RoadmapCriteria {
			if criterion.Roadmap == roadmap {
				filtered = append(filtered, criterion)
			}
		}
		graph.RoadmapCriteria = filtered
	}
	return graph, nil
}

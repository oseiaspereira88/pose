package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func loadDeliveryProfiles(root string) (map[string]posemodel.DeliveryProfile, error) {
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"))
	if err != nil {
		return nil, err
	}
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		return nil, err
	}
	if err := validateDeliveryMatrixContract(matrix); err != nil {
		return nil, err
	}
	return matrix.DeliveryProfiles, nil
}

func collectDeliveryTargets(root string, specs []posemodel.Spec, profiles map[string]posemodel.DeliveryProfile) ([]posemodel.Spec, []posemodel.DeliveryTarget, error) {
	store := posemodel.Store{Root: root}
	fullSpecs := make([]posemodel.Spec, 0, len(specs))
	targets := []posemodel.DeliveryTarget{}
	for _, summary := range specs {
		full, err := store.GetSpec(summary.Slug)
		if err != nil {
			return nil, nil, err
		}
		parsed, _, err := posemodel.ParseDeliveryTargets(*full)
		if err != nil {
			return nil, nil, err
		}
		if err := posemodel.ValidateDeliveryTargets(root, parsed, profiles); err != nil {
			return nil, nil, err
		}
		fullSpecs = append(fullSpecs, *full)
		targets = append(targets, parsed...)
	}
	return fullSpecs, targets, nil
}

func loadDeliveryValidationResults(root, relative string) ([]posemodel.DeliveryValidationResult, error) {
	if relative == "" {
		return []posemodel.DeliveryValidationResult{}, nil
	}
	if !confinedRelativePath(relative) {
		return nil, fmt.Errorf("validation result path must remain inside project")
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if os.IsNotExist(err) {
		return []posemodel.DeliveryValidationResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	var run validationRunResult
	if err := json.Unmarshal(raw, &run); err != nil || run.SchemaVersion != validationResultSchema {
		return nil, fmt.Errorf("invalid structured validation result")
	}
	results := make([]posemodel.DeliveryValidationResult, 0, len(run.Checks))
	for _, check := range run.Checks {
		if check.EvidenceClass == "" {
			continue
		}
		results = append(results, posemodel.DeliveryValidationResult{ID: check.ID, Module: check.Module, Check: check.Name, EvidenceClass: check.EvidenceClass, Severity: check.Severity, Outcome: check.Outcome, GitHead: run.GitHead, GeneratedAt: run.GeneratedAt, ProvenanceDigest: run.ProvenanceDigest, ScopeProvenance: run.ScopeProvenance, Report: relative})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results, nil
}

func extendCurrentDeliveryGraph(root string, base posemodel.DeliveryIntegrityGraph) (posemodel.DeliveryIntegrityGraph, error) {
	policy, err := posemodel.LoadDeliveryPolicy(root)
	if err != nil {
		return base, err
	}
	profiles, err := loadDeliveryProfiles(root)
	if err != nil {
		if os.IsNotExist(err) {
			return base, nil
		}
		return base, err
	}
	store := posemodel.Store{Root: root}
	specs, err := store.ListSpecs("", "")
	if err != nil {
		if os.IsNotExist(err) {
			return base, nil
		}
		return base, err
	}
	fullSpecs, targets, err := collectDeliveryTargets(root, specs, profiles)
	if err != nil {
		return base, err
	}
	results, err := loadDeliveryValidationResults(root, policy.ResultsPath)
	if err != nil {
		return base, err
	}
	roadmapSummaries, err := store.ListRoadmaps()
	if err != nil {
		return base, err
	}
	roadmaps := []posemodel.Roadmap{}
	for _, summary := range roadmapSummaries {
		full, err := store.GetRoadmap(summary.Slug)
		if err != nil {
			return base, err
		}
		roadmaps = append(roadmaps, *full)
	}
	return posemodel.BuildDeliverySurface(base, fullSpecs, targets, results, roadmaps, profiles, policy), nil
}

func cmdSurfaceCheck(root string, args []string, stdout, stderr io.Writer) int {
	var spec, resultsPath string
	strict, jsonOutput := true, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--spec", "--results":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose surface-check [--spec slug] [--results path] [--strict|--tolerant] [--json]")
			}
			i++
			if args[i-1] == "--spec" {
				spec = args[i]
			} else {
				resultsPath = args[i]
			}
		case "--strict":
			strict = true
		case "--tolerant":
			strict = false
		case "--json":
			jsonOutput = true
		default:
			return usageError(stderr, "Usage: pose surface-check [--spec slug] [--results path] [--strict|--tolerant] [--json]")
		}
	}
	if spec != "" && posemodel.ValidateSlug(spec) != nil {
		fmt.Fprintln(stderr, "pose surface-check: invalid spec slug")
		return 2
	}
	policy, err := posemodel.LoadDeliveryPolicy(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose surface-check: %v\n", err)
		return 1
	}
	if resultsPath != "" {
		if !confinedRelativePath(resultsPath) {
			fmt.Fprintln(stderr, "pose surface-check: results path escapes project")
			return 2
		}
		policy.ResultsPath = filepath.ToSlash(filepath.Clean(resultsPath))
	}
	baseSpecs, claims, sets, tracked, artifactPolicy, err := collectArtifactGraphInputs(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose surface-check: %v\n", err)
		return 1
	}
	base := posemodel.BuildDeliveryIntegrity(baseSpecs, claims, sets, tracked, artifactPolicy)
	profiles, err := loadDeliveryProfiles(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose surface-check: %v\n", err)
		return 1
	}
	fullSpecs, targets, err := collectDeliveryTargets(root, baseSpecs, profiles)
	if err != nil {
		fmt.Fprintf(stderr, "pose surface-check: %v\n", err)
		return 1
	}
	results, err := loadDeliveryValidationResults(root, policy.ResultsPath)
	if err != nil {
		fmt.Fprintf(stderr, "pose surface-check: %v\n", err)
		return 1
	}
	graph := posemodel.BuildDeliverySurface(base, fullSpecs, targets, results, nil, profiles, policy)
	if spec != "" {
		graph = focusSurfaceGraph(graph, spec)
	}
	if jsonOutput {
		_ = writeJSON(stdout, graph)
	} else {
		fmt.Fprintf(stdout, "surface.spec=%s\nsurface.targets=%d\nsurface.results=%d\nsurface.findings=%d\nsurface.provenance_digest=%s\n", spec, len(graph.Deliveries), len(graph.ValidationResults), len(graph.Findings), graph.ProvenanceDigest)
		for _, finding := range graph.Findings {
			fmt.Fprintf(stdout, "[%s] %s %s: %s; remediation: %s\n", strings.ToUpper(finding.Severity), finding.Code, finding.Path, finding.Message, finding.Remediation)
		}
	}
	return deliveryFindingExit(graph.Findings, strict)
}

func cmdRoadmapCheck(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose roadmap-check <slug> [--strict|--tolerant] [--json]")
	}
	slug := args[0]
	strict, jsonOutput := true, false
	for _, arg := range args[1:] {
		switch arg {
		case "--strict":
			strict = true
		case "--tolerant":
			strict = false
		case "--json":
			jsonOutput = true
		default:
			return usageError(stderr, "Usage: pose roadmap-check <slug> [--strict|--tolerant] [--json]")
		}
	}
	if posemodel.ValidateSlug(slug) != nil {
		fmt.Fprintln(stderr, "pose roadmap-check: invalid roadmap slug")
		return 2
	}
	graph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose roadmap-check: %v\n", err)
		return 1
	}
	store := posemodel.Store{Root: root}
	roadmap, err := store.GetRoadmap(slug)
	if err != nil {
		fmt.Fprintf(stderr, "pose roadmap-check: %v\n", err)
		return 1
	}
	criteria := []posemodel.RoadmapCriterion{}
	for _, item := range graph.RoadmapCriteria {
		if item.Roadmap == slug {
			criteria = append(criteria, item)
		}
	}
	blockers := []string{}
	for _, criterion := range criteria {
		if !criterion.Passed {
			blockers = append(blockers, criterion.ID+": "+strings.Join(criterion.Reasons, "; "))
		}
		for _, ref := range criterion.Refs {
			if !strings.HasPrefix(ref, "manual-review:") {
				continue
			}
			relative := strings.TrimPrefix(ref, "manual-review:")
			if !confinedRelativePath(relative) {
				blockers = append(blockers, criterion.ID+": manual review escapes project")
				continue
			}
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil || info.IsDir() {
				blockers = append(blockers, criterion.ID+": manual review does not resolve: "+relative)
			}
		}
	}
	for _, finding := range graph.Findings {
		if finding.Path == slug && (finding.Severity == "error" || finding.Severity == "critical") {
			blockers = append(blockers, finding.Code+": "+finding.Message)
		}
	}
	for _, milestone := range roadmap.Milestones {
		for _, member := range milestone.Specs {
			state, err := store.GetCloseoutState("spec:" + member)
			if err != nil || !state.Terminal {
				blockers = append(blockers, "member spec is not terminal: "+member)
			}
		}
	}
	result := map[string]any{"schema_version": 1, "roadmap": slug, "status": roadmap.Status, "criteria": criteria, "terminal": len(blockers) == 0, "blockers": uniqueCLIStrings(blockers), "graph_digest": graph.InputDigest}
	if jsonOutput {
		_ = writeJSON(stdout, result)
	} else {
		fmt.Fprintf(stdout, "roadmap.slug=%s\nroadmap.criteria=%d\nroadmap.terminal=%t\n", slug, len(criteria), len(blockers) == 0)
		for _, blocker := range uniqueCLIStrings(blockers) {
			fmt.Fprintln(stdout, "[BLOCKER] "+blocker)
		}
	}
	if strict && len(blockers) > 0 {
		return 1
	}
	return 0
}

func focusSurfaceGraph(graph posemodel.DeliveryIntegrityGraph, spec string) posemodel.DeliveryIntegrityGraph {
	deliveries := graph.Deliveries[:0]
	refs := map[string]bool{}
	for _, target := range graph.Deliveries {
		if target.Spec == spec {
			deliveries = append(deliveries, target)
			refs[target.Ref] = true
		}
	}
	graph.Deliveries = deliveries
	findings := graph.Findings[:0]
	for _, finding := range graph.Findings {
		if finding.Spec == spec || refs[finding.Path] {
			findings = append(findings, finding)
		}
	}
	graph.Findings = findings
	paths := map[string][]string{}
	for ref := range refs {
		paths[ref] = graph.Paths[ref]
	}
	graph.Paths = paths
	return graph
}

func deliveryFindingExit(findings []posemodel.DeliveryIntegrityFinding, strict bool) int {
	if strict {
		for _, finding := range findings {
			if finding.Severity == "error" || finding.Severity == "critical" {
				return 1
			}
		}
	}
	return 0
}

func deliverySpecBlockers(root, slug string) []string {
	graph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		return []string{err.Error()}
	}
	graph = focusSurfaceGraph(graph, slug)
	blockers := []string{}
	for _, finding := range graph.Findings {
		if finding.Severity == "error" || finding.Severity == "critical" {
			blockers = append(blockers, finding.Code+": "+finding.Message)
		}
	}
	return uniqueCLIStrings(blockers)
}
func uniqueCLIStrings(values []string) []string {
	sort.Strings(values)
	out := []string{}
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

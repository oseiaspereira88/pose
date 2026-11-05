package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

type indexGenerationBuild struct {
	Encoded                 map[string][]byte
	Inputs                  map[string]string
	Projects                []string
	ReferencedProjects      []string
	RequiresPrimaryRevision bool
	Specs                   int
	Roadmaps                int
	Modules                 int
	DeliveryFindings        int
}

type indexMaterialPreflight struct {
	Projects           []string
	ReferencedProjects []string
}

type indexGenerationBuildFailure struct {
	Code    string
	Message string
}

func (e *indexGenerationBuildFailure) Error() string { return e.Message }

func indexBuildFailure(err error) (string, string) {
	var buildErr *indexGenerationBuildFailure
	if errors.As(err, &buildErr) {
		return buildErr.Code, buildErr.Message
	}
	switch {
	case errors.Is(err, posepkg.ErrGovernanceReadViewChanged):
		return "index-generation-read-view-changed", "index inputs changed during generation; nothing was written, run `pose index` again"
	case errors.Is(err, posepkg.ErrGovernanceReadViewUnknown):
		return "index-generation-read-view-unknown", "index input closure could not be proved: " + err.Error() + "; nothing was written"
	default:
		return "index-generation-read-view-unknown", "index generation could not be built safely: " + err.Error() + "; nothing was written"
	}
}

func generateIndexGeneration(files posepkg.Files, root string, gate *gateOutput, out *cliout.Renderer, fail func(code, message string) int) int {
	resolver, project, err := posepkg.EnvironmentArtifactResolverIn(files, root, "")
	if err != nil {
		return fail("invalid-project-configuration", "invalid-project-configuration: "+err.Error())
	}
	store := posepkg.Store{Root: root, FS: files, FederatedProjectID: project, FederatedResolver: &resolver}
	preflight, err := captureIndexMaterialPreflight(store)
	if err != nil {
		code, message := indexBuildFailure(err)
		return fail(code, message)
	}

	status := 1
	guardErr := store.WithGovernanceWriteGuardForProjects("index.generate", preflight.Projects, func(guarded posepkg.Store) error {
		built, buildErr := captureIndexGeneration(guarded)
		if buildErr != nil {
			code, message := indexBuildFailure(buildErr)
			status = fail(code, message)
			return nil
		}
		expanded := expandedIndexMaterialProjects(preflight, built)
		if len(expanded) > 0 {
			message := "federated index material expanded after root preflight (" + strings.Join(expanded, ", ") + "); nothing was written, rerun `pose index`"
			status = fail("index-material-expanded", message)
			return nil
		}

		// Publish only after the captured read view has passed its final audit.
		// Keep using the guarded source Files; captured Files are read-only.
		receipt, publishErr := guarded.PublishIndexGeneration(built.Encoded, built.Inputs)
		if publishErr != nil {
			switch {
			case receipt.Committed:
				out.RecordField("index_generation", receipt.GenerationID)
				status = fail(indexPublishCode(publishErr), fmt.Sprintf("generation %s was committed but not confirmed durable: %v; run `pose index --verify`", receipt.GenerationID, publishErr))
			case errors.Is(publishErr, posepkg.ErrIndexGenerationBusy):
				status = fail(indexPublishCode(publishErr), fmt.Sprintf("%v; nothing was written, run `pose index` again when it finishes", publishErr))
			default:
				status = fail(indexPublishCode(publishErr), fmt.Sprintf("publishing the index generation: %v (the previous generation stays in force; run `pose index` again)", publishErr))
			}
			return nil
		}
		out.RecordField("index_generation", receipt.GenerationID)
		if receipt.Previous != "" {
			out.RecordField("index_generation_previous", receipt.Previous)
		}
		if receipt.CleanupError != nil {
			out.Finding(cliout.Finding{State: cliout.StateWarning, Code: "index-generation-cleanup", Message: "retention of old index generations failed: " + receipt.CleanupError.Error()})
		}
		indexesDir := filepath.Join(root, ".pose", "indexes")
		out.RecordField("indexes_dir", indexesDir)
		out.RecordCount("specs", built.Specs)
		out.RecordCount("roadmaps", built.Roadmaps)
		out.RecordCount("modules", built.Modules)
		out.RecordCount("delivery_findings", built.DeliveryFindings)
		gate.Verdict(cliout.StatePass, "POSE indexes updated at "+indexesDir, "indexes updated")
		status = 0
		return nil
	})
	if guardErr != nil {
		message := fmt.Sprintf("%v; nothing was written, run `pose index` again when the roots are available", guardErr)
		return fail(indexPublishCode(guardErr), message)
	}
	return status
}

func captureIndexMaterialPreflight(store posepkg.Store) (indexMaterialPreflight, error) {
	var preflight indexMaterialPreflight
	err := store.WithGovernanceReadViewForIndexBuild("index.generate.preflight", func(captured posepkg.Store) (bool, error) {
		material, err := posepkg.BuildIndexFederatedMaterialIn(captured)
		if err != nil {
			return false, err
		}
		preflight = indexMaterialPreflight{Projects: material.Projects, ReferencedProjects: material.ReferencedProjects}
		return material.RequiresPrimaryRevision, nil
	})
	if err != nil {
		return indexMaterialPreflight{}, err
	}
	return preflight, nil
}

func captureIndexGeneration(store posepkg.Store) (indexGenerationBuild, error) {
	var built indexGenerationBuild
	err := store.WithGovernanceReadViewForIndexBuild("index.generate", func(captured posepkg.Store) (bool, error) {
		attempt, err := buildIndexGenerationIn(captured)
		if err != nil {
			return false, err
		}
		built = attempt
		return attempt.RequiresPrimaryRevision, nil
	})
	if err != nil {
		return indexGenerationBuild{}, err
	}
	return built, nil
}

// buildIndexGenerationIn is a pure in-memory projection over the captured
// Store. It performs no publication and binds the exact federated projections
// it used to build the output files.
func buildIndexGenerationIn(store posepkg.Store) (indexGenerationBuild, error) {
	files, root := store.FS, store.Root
	modules, manifests, dockers, charts, readmes, scanErr := scanModulesBoundedIn(files, root)
	if scanErr != nil {
		return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "module-discovery-unavailable", Message: fmt.Sprintf("module discovery: %v", scanErr)}
	}
	metadataDefaults, metadata := loadModuleMetadataIn(files, root)
	apps, services, packages := []indexedModule{}, []indexedModule{}, []indexedModule{}
	for _, module := range modules {
		decl := metadata[module.Path]
		defaults := map[string]string{"owner": "unknown", "criticality": "medium", "domain": "unknown", "validationProfile": "baseline"}
		for key, value := range metadataDefaults {
			if value != "" {
				defaults[key] = value
			}
		}
		missing := []string{}
		for key := range defaults {
			if value := decl[key]; value != "" {
				defaults[key] = value
			} else {
				missing = append(missing, key)
			}
		}
		sort.Strings(missing)
		module.Owner = defaults["owner"]
		module.Criticality = defaults["criticality"]
		module.Domain = defaults["domain"]
		module.ValidationProfile = defaults["validationProfile"]
		module.Metadata = defaults
		source := "declared"
		if len(decl) == 0 {
			source = "defaulted"
		} else if len(missing) > 0 {
			source = "partial"
		}
		module.MetadataStatus = map[string]any{"isComplete": len(missing) == 0 && len(decl) > 0, "source": source, "missingFields": missing}
		lower := strings.ToLower(module.Path + "/" + module.Name)
		if strings.Contains(lower, "service") || strings.Contains(lower, "/services/") {
			services = append(services, module)
		} else if strings.Contains(lower, "/app") || strings.Contains(lower, "web") || strings.Contains(lower, "portal") || strings.HasSuffix(lower, "-ui") {
			apps = append(apps, module)
		} else {
			packages = append(packages, module)
		}
	}
	repo := map[string]any{"root": ".", "apps": apps, "services": services, "packages": packages, "manifests": manifests, "dockerfiles": dockers, "helmCharts": charts, "readmes": readmes, "moduleMetadata": map[string]any{"schemaVersion": 1, "source": ".pose/indexes/module-metadata.json"}}

	material, err := posepkg.BuildIndexFederatedMaterialIn(store)
	if err != nil {
		code := "index-material-unavailable"
		if strings.Contains(err.Error(), "federated roadmap ") {
			code = "federated-roadmap"
		}
		return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: code, Message: err.Error()}
	}
	specMap := map[string]any{}
	edges := []map[string]string{}
	for _, spec := range material.Specs {
		if _, exists := specMap[spec.Slug]; exists {
			return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "conflicting-artifact-identity", Message: "conflicting-artifact-identity"}
		}
		identity, _ := posepkg.ParseArtifactRef("spec:" + spec.Slug)
		identity.Project = store.FederatedProjectID
		specMap[spec.Slug] = map[string]any{"identity": identity, "dependency_resolutions": material.SpecDependencyResolutions[spec.Slug], "status": spec.Status, "depends_on": spec.DependsOn, "priority": spec.Priority, "path": relativePath(root, spec.Path)}
		for _, raw := range spec.DependsOn {
			if ref, parseErr := posepkg.ParseArtifactRef(raw); parseErr == nil && ref.Kind == "spec" {
				edges = append(edges, map[string]string{"from": spec.Slug, "to": raw})
			}
		}
	}
	roadmapMap := map[string]any{}
	for _, roadmap := range material.Roadmaps {
		roadmapMap[roadmap.Slug] = map[string]any{"status": roadmap.Status, "created_at": roadmap.CreatedAt, "depends_on": roadmap.DependsOn, "consumes": roadmap.Consumes, "milestones": roadmap.Milestones, "federated_acceptance": material.RoadmapAcceptances[roadmap.Slug], "path": relativePath(root, roadmap.Path)}
	}
	deliveryGraph, err := buildCurrentDeliveryGraphIn(files, root)
	if err != nil {
		return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "delivery-integrity", Message: fmt.Sprintf("delivery integrity: %v", err)}
	}
	releaseStatus, err := store.GetReleaseStatus("")
	if err != nil {
		return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "release-lifecycle", Message: fmt.Sprintf("release lifecycle: %v", err)}
	}
	outputs := map[string]any{"repo-map.json": repo, "services.json": services, "packages.json": packages, "spec-graph.json": map[string]any{"schemaVersion": 1, "specs": specMap, "edges": edges}, "roadmaps.json": map[string]any{"schemaVersion": 1, "roadmaps": roadmapMap}, "delivery-integrity.json": deliveryGraph, "releases.json": releaseStatus}
	encoded := make(map[string][]byte, len(outputs))
	for name, value := range outputs {
		var raw []byte
		if graph, ok := value.(posepkg.DeliveryIntegrityGraph); ok {
			raw, err = graph.IndexJSON()
		} else {
			raw, err = json.MarshalIndent(value, "", "  ")
		}
		if err != nil {
			return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "encode", Message: fmt.Sprintf("encoding %s: %v", name, err)}
		}
		encoded[name] = append(raw, '\n')
	}
	inputs, unknown := posepkg.IndexInputBindingsForFederatedMaterialIn(files, root, material.Projection)
	if len(unknown) > 0 {
		return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "index-input-unknown", Message: "index inputs could not be bound: " + strings.Join(unknown, ", ")}
	}
	for _, name := range posepkg.IndexInputNames {
		if strings.TrimSpace(inputs[name]) == "" {
			return indexGenerationBuild{}, &indexGenerationBuildFailure{Code: "index-input-unknown", Message: "required index input has no binding: " + name}
		}
	}
	return indexGenerationBuild{
		Encoded: encoded, Inputs: inputs, Projects: material.Projects, ReferencedProjects: material.ReferencedProjects,
		RequiresPrimaryRevision: material.RequiresPrimaryRevision, Specs: len(specMap), Roadmaps: len(roadmapMap),
		Modules: len(modules), DeliveryFindings: len(deliveryGraph.Findings),
	}, nil
}

func expandedIndexMaterialProjects(preflight indexMaterialPreflight, guarded indexGenerationBuild) []string {
	initialProjects := make(map[string]bool, len(preflight.Projects))
	for _, project := range preflight.Projects {
		initialProjects[project] = true
	}
	initialReferences := make(map[string]bool, len(preflight.ReferencedProjects))
	for _, project := range preflight.ReferencedProjects {
		initialReferences[project] = true
	}
	expanded := []string{}
	for _, project := range guarded.Projects {
		if !initialProjects[project] {
			expanded = append(expanded, project)
		}
	}
	for _, project := range guarded.ReferencedProjects {
		if !initialReferences[project] {
			expanded = append(expanded, project)
		}
	}
	return uniqueSortedStrings(expanded)
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	sort.Strings(values)
	unique := values[:1]
	for _, value := range values[1:] {
		if value != unique[len(unique)-1] {
			unique = append(unique, value)
		}
	}
	return unique
}

package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// IndexFederatedSpecMaterial is the dependency-resolution projection emitted
// for one spec by `pose index`.
type IndexFederatedSpecMaterial struct {
	Slug                  string               `json:"slug"`
	DependencyResolutions []ArtifactResolution `json:"dependency_resolutions"`
}

// IndexFederatedRoadmapMaterial is the acceptance projection emitted for one
// roadmap by `pose index`.
type IndexFederatedRoadmapMaterial struct {
	Slug       string                           `json:"slug"`
	Acceptance FederatedRoadmapAcceptanceReport `json:"acceptance"`
}

// IndexFederatedMaterialProjection contains only the federated semantic
// inputs represented in the generated indexes. It deliberately excludes
// unrelated foreign files and roots.
type IndexFederatedMaterialProjection struct {
	SchemaVersion int                             `json:"schema_version"`
	Specs         []IndexFederatedSpecMaterial    `json:"specs"`
	Roadmaps      []IndexFederatedRoadmapMaterial `json:"roadmaps"`
}

// IndexFederatedMaterial carries both the semantic input projection and the
// source records used to build index outputs. The projection is shared by the
// builder and the generation binding so output bytes and their binding cannot
// be computed from separate, interleaved reads.
type IndexFederatedMaterial struct {
	Projection                IndexFederatedMaterialProjection
	Specs                     []Spec
	Roadmaps                  []Roadmap
	SpecDependencyResolutions map[string][]ArtifactResolution
	RoadmapAcceptances        map[string]FederatedRoadmapAcceptanceReport
	Projects                  []string
	ReferencedProjects        []string
	RequiresPrimaryRevision   bool
}

// BuildIndexFederatedMaterialIn returns the specs, roadmaps, semantic
// projections and exact foreign projects whose contents were resolved. The
// Store must carry the operation's authorized resolver when one has already
// been established; it is reused unchanged except for the captured Files the
// operation view supplies.
func BuildIndexFederatedMaterialIn(store Store) (IndexFederatedMaterial, error) {
	store.indexMaterial = true
	resolver := ArtifactResolver{}
	project := store.FederatedProjectID
	if store.FederatedResolver != nil {
		resolver = *store.FederatedResolver
		if resolver.Roots == nil {
			return IndexFederatedMaterial{}, errors.New("pose: authorized project roots are unavailable for index material")
		}
	} else {
		var err error
		resolver, project, err = EnvironmentArtifactResolverIn(store.fsys(), store.Root, "")
		if err != nil {
			return IndexFederatedMaterial{}, fmt.Errorf("pose: index material resolver: %w", err)
		}
	}
	if project == "" {
		var found, ambiguous, overflow bool
		project, found, ambiguous, overflow = uniqueProjectIDForRootIn(store.fsys(), resolver.Roots, store.Root, resolver.Authorize)
		if overflow {
			return IndexFederatedMaterial{}, errors.New("pose: index material root identity exceeds the authorized-root snapshot budget")
		}
		if ambiguous {
			return IndexFederatedMaterial{}, errors.New("pose: index material root has multiple authorized project identities")
		}
		if !found {
			project = DefaultProjectIDIn(store.fsys(), store.Root)
			if ValidateSlug(project) != nil || resolver.Authorize != nil && !resolver.Authorize(project) {
				return IndexFederatedMaterial{}, errors.New("pose: index material project identity is not authorized")
			}
			bound, err := resolver.Roots.StoreFor(project)
			if err != nil || !sameProjectRoot(store.fsys(), bound.Root, store.Root) {
				return IndexFederatedMaterial{}, errors.New("pose: index material root is outside the authorized project registry")
			}
		}
	}
	if ValidateSlug(project) != nil {
		return IndexFederatedMaterial{}, errors.New("pose: index material project identity is unavailable")
	}

	// Fresh for this build, never inherited from an earlier advisory operation.
	resolver = resolver.WithSpecSnapshots()
	specs, err := store.ListSpecs("", "")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return IndexFederatedMaterial{}, fmt.Errorf("pose: listing specs for index material: %w", err)
	}
	roadmaps, err := store.ListRoadmaps()
	if err != nil {
		return IndexFederatedMaterial{}, fmt.Errorf("pose: listing roadmaps for index material: %w", err)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Slug < specs[j].Slug })
	sort.Slice(roadmaps, func(i, j int) bool { return roadmaps[i].Slug < roadmaps[j].Slug })

	material := IndexFederatedMaterial{
		Specs:                     specs,
		Roadmaps:                  roadmaps,
		SpecDependencyResolutions: make(map[string][]ArtifactResolution, len(specs)),
		RoadmapAcceptances:        make(map[string]FederatedRoadmapAcceptanceReport, len(roadmaps)),
		Projection: IndexFederatedMaterialProjection{
			SchemaVersion: 1,
			Specs:         make([]IndexFederatedSpecMaterial, 0, len(specs)),
			Roadmaps:      make([]IndexFederatedRoadmapMaterial, 0, len(roadmaps)),
		},
	}
	projects := map[string]bool{}
	referencedProjects := map[string]bool{}
	addProject := func(id string) {
		if id == "" || id == project {
			return
		}
		referencedProjects[id] = true
		if resolver.Authorize != nil && !resolver.Authorize(id) {
			return
		}
		if _, err := resolver.Roots.StoreFor(id); err == nil {
			// Include only foreign roots named by this semantic projection. A
			// project discovered for reading still has to pass the explicit-root
			// write guard before this generation can be published.
			projects[id] = true
		}
	}
	for _, spec := range specs {
		resolutions := make([]ArtifactResolution, 0, len(spec.DependsOn))
		semanticResolutions := make([]ArtifactResolution, 0, len(spec.DependsOn))
		for _, raw := range spec.DependsOn {
			resolution := resolver.Resolve(project, raw)
			resolutions = append(resolutions, resolution)
			semanticResolution := resolution
			// Same-project resolution revisions describe repository HEAD, while
			// the identity, state, status, digest and dependency edges below carry
			// the semantic result. Keep local unrelated commits out of the index
			// binding; the generated output still preserves the observed revision.
			if semanticResolution.Identity.Project == project {
				semanticResolution.Revision = ""
			}
			semanticResolutions = append(semanticResolutions, semanticResolution)
			addProject(resolution.Identity.Project)
		}
		material.SpecDependencyResolutions[spec.Slug] = resolutions
		material.Projection.Specs = append(material.Projection.Specs, IndexFederatedSpecMaterial{
			Slug: spec.Slug, DependencyResolutions: semanticResolutions,
		})
	}
	for _, roadmap := range roadmaps {
		acceptance, err := store.FederatedRoadmapAcceptance(project, roadmap.Slug, resolver)
		if err != nil {
			return IndexFederatedMaterial{}, fmt.Errorf("pose: federated roadmap %s: %w", roadmap.Slug, err)
		}
		material.RoadmapAcceptances[roadmap.Slug] = acceptance
		material.Projection.Roadmaps = append(material.Projection.Roadmaps, IndexFederatedRoadmapMaterial{
			Slug: roadmap.Slug, Acceptance: indexSemanticAcceptanceReport(project, acceptance),
		})
		for _, dependency := range acceptance.Manifest.Dependencies {
			addProject(dependency.Identity.Project)
		}
	}
	material.Projects = make([]string, 0, len(projects))
	for id := range projects {
		material.Projects = append(material.Projects, id)
	}
	sort.Strings(material.Projects)
	material.ReferencedProjects = make([]string, 0, len(referencedProjects))
	for id := range referencedProjects {
		material.ReferencedProjects = append(material.ReferencedProjects, id)
	}
	sort.Strings(material.ReferencedProjects)
	material.RequiresPrimaryRevision = indexMaterialRequiresPrimaryRevision(store, resolver, project, material)
	return material, nil
}

func indexMaterialRequiresPrimaryRevision(store Store, resolver ArtifactResolver, localProject string, material IndexFederatedMaterial) bool {
	files := store.fsys()
	for _, report := range material.RoadmapAcceptances {
		for _, dependency := range report.Manifest.Dependencies {
			if dependency.Identity.Project == localProject && dependency.SourceDigest != "" {
				// Roadmap acceptance proves the local artifact against its
				// committed revision; a worktree-only answer is not enough.
				return true
			}
			if dependency.Identity.Project == "" || dependency.Identity.Project == localProject || dependency.SourceDigest == "" || dependency.SourceRevision == "" || !hasGitMetadataIn(files, store.Root) {
				continue
			}
			foreign, err := resolver.Roots.StoreFor(dependency.Identity.Project)
			if err != nil {
				continue
			}
			localReal, localErr := files.EvalSymlinks(store.Root)
			foreignReal, foreignErr := files.EvalSymlinks(foreign.Root)
			if localErr != nil || foreignErr != nil {
				return true
			}
			relative, err := filepath.Rel(localReal, foreignReal)
			if err != nil {
				return true
			}
			inside := relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
			if inside {
				return true
			}
		}
	}
	return false
}

// indexSemanticAcceptanceReport drops only local repository revisions whose
// changes do not alter the accepted artifact projection. This normalized copy
// is used only for the input binding; generated output keeps the exact report
// produced by the federated acceptance operation. The acceptance code already
// uses artifact-level resolution and trust digests for meaningful local and
// foreign changes, so recompute the binding copy's digest after normalization.
func indexSemanticAcceptanceReport(localProject string, report FederatedRoadmapAcceptanceReport) FederatedRoadmapAcceptanceReport {
	report.Manifest.Dependencies = append([]FederatedDependency(nil), report.Manifest.Dependencies...)
	report.Manifest.CoordinatorRevision = ""
	for i := range report.Manifest.Dependencies {
		if report.Manifest.Dependencies[i].Identity.Project == localProject {
			report.Manifest.Dependencies[i].SourceRevision = ""
		}
	}
	report.Manifest.Digest = ""
	report.Manifest.Digest = canonicalFederatedDigest(report.Manifest)
	return report
}

// IndexFederatedMaterialDigest returns the stable semantic digest bound to an
// index generation. The digest does not include root paths or unrelated files.
func IndexFederatedMaterialDigest(material IndexFederatedMaterialProjection) (string, bool) {
	raw, err := json.Marshal(material)
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), true
}

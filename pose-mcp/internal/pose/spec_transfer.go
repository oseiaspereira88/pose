package pose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

const SpecTransferSchemaVersion = 1

const (
	maxSpecTransferProjects       = 256
	maxSpecTransferImpacts        = 10000
	maxSpecTransferArtifactBytes  = 8 << 20
	maxSpecTransferOperationsRead = 1024
	maxSpecTransferJournalEntries = 16
)

var specTransferIDRE = regexp.MustCompile(`^stf-[0-9a-f]{24}$`)
var specRequirementIDRE = regexp.MustCompile(`(?m)^\s*[-*]?\s*(R[0-9]+)\s*:`)

type SpecTransferRequirementMapping struct {
	SourceRequirement      string `json:"source_requirement,omitempty"`
	DestinationRequirement string `json:"destination_requirement,omitempty"`
	Disposition            string `json:"disposition"`
}

type SpecTransferRequest struct {
	Source      ArtifactRef                      `json:"source"`
	Destination ArtifactRef                      `json:"destination"`
	Mappings    []SpecTransferRequirementMapping `json:"requirement_map"`
}

type SpecTransferProjectRevision struct {
	ProjectID string `json:"project_id"`
	Revision  string `json:"expected_revision"`
}

type SpecTransferImpact struct {
	ProjectID    string      `json:"project_id"`
	Owner        ArtifactRef `json:"owner"`
	Path         string      `json:"path"`
	BeforeDigest string      `json:"before_digest"`
	AfterDigest  string      `json:"after_digest"`
	From         []string    `json:"references"`
	To           string      `json:"replacement"`
}

// SpecTransferPlan is a deterministic, path-root-free snapshot. Paths are
// project-relative artifact paths; project roots and absolute paths are never
// serialized.
type SpecTransferPlan struct {
	SchemaVersion          int                              `json:"schema_version"`
	OperationID            string                           `json:"operation_id"`
	Digest                 string                           `json:"digest"`
	Request                SpecTransferRequest              `json:"request"`
	Source                 ArtifactRef                      `json:"source"`
	Destination            ArtifactRef                      `json:"destination"`
	EffectiveDate          string                           `json:"effective_date"`
	SourceRevision         string                           `json:"source_revision"`
	DestinationRevision    string                           `json:"destination_revision"`
	SourceDigest           string                           `json:"source_digest"`
	DestinationDigest      string                           `json:"destination_digest,omitempty"`
	DestinationExists      bool                             `json:"destination_exists"`
	SourcePath             string                           `json:"source_path"`
	DestinationPath        string                           `json:"destination_path"`
	DestinationStageDigest string                           `json:"destination_stage_digest"`
	DestinationFinalDigest string                           `json:"destination_final_digest"`
	SourceRedirectDigest   string                           `json:"source_redirect_digest"`
	SourceStubDigest       string                           `json:"source_stub_digest"`
	SourceStatus           string                           `json:"source_status"`
	DestinationStatus      string                           `json:"destination_status"`
	FinalDestinationStatus string                           `json:"final_destination_status"`
	RequiresFreshEvidence  bool                             `json:"requires_fresh_evidence"`
	Mappings               []SpecTransferRequirementMapping `json:"requirement_map"`
	ImpactedReferences     []SpecTransferImpact             `json:"impacted_references,omitempty"`
	ProjectRevisions       []SpecTransferProjectRevision    `json:"project_revisions"`
	AffectedProjects       []string                         `json:"affected_projects"`
	Blockers               []string                         `json:"blockers,omitempty"`
}

type SpecTransferStatus struct {
	SchemaVersion int         `json:"schema_version"`
	OperationID   string      `json:"operation_id"`
	PlanDigest    string      `json:"plan_digest"`
	ProjectID     string      `json:"project_id"`
	Phase         string      `json:"phase"`
	Source        ArtifactRef `json:"source"`
	Destination   ArtifactRef `json:"destination"`
	UpdatedAt     string      `json:"updated_at,omitempty"`
}

type SpecTransferFailureHook func(phase string) error

type specTransferReceipt struct {
	SchemaVersion int      `json:"schema_version"`
	OperationID   string   `json:"operation_id"`
	PlanDigest    string   `json:"plan_digest"`
	ProjectID     string   `json:"project_id"`
	Phase         string   `json:"phase"`
	Files         []string `json:"files,omitempty"`
	RecordedAt    string   `json:"recorded_at"`
}

type specTransferRedirect struct {
	SchemaVersion int         `json:"schema_version"`
	OperationID   string      `json:"operation_id"`
	Source        ArtifactRef `json:"source"`
	Destination   ArtifactRef `json:"destination"`
}

func specTransferError(code string) error { return fmt.Errorf("spec-transfer: %s", code) }

func validateTransferIdentity(ref ArtifactRef) error {
	if ref.Project == "" || ref.Kind != "spec" || ref.Slug == "" || ValidateSlug(ref.Project) != nil || ValidateSlug(ref.Slug) != nil || ref.Milestone != "" {
		return specTransferError("qualified-spec-identity-required")
	}
	return nil
}

func PreviewSpecTransfer(resolver ArtifactResolver, request SpecTransferRequest, effectiveDate string) (SpecTransferPlan, error) {
	plan := SpecTransferPlan{SchemaVersion: SpecTransferSchemaVersion, Request: request}
	if resolver.Roots == nil {
		return plan, specTransferError("project-registry-unavailable")
	}
	if err := validateTransferIdentity(request.Source); err != nil {
		return plan, err
	}
	if err := validateTransferIdentity(request.Destination); err != nil {
		return plan, err
	}
	if request.Source == request.Destination || request.Source.Project == request.Destination.Project {
		return plan, specTransferError("source-and-destination-must-be-distinct-projects")
	}
	if effectiveDate == "" {
		effectiveDate = time.Now().UTC().Format(time.DateOnly)
	}
	if _, err := time.Parse(time.DateOnly, effectiveDate); err != nil {
		return plan, specTransferError("invalid-effective-date")
	}
	plan.EffectiveDate = effectiveDate
	plan.Request.Mappings = canonicalRequirementMappings(request.Mappings)
	plan.Mappings = append([]SpecTransferRequirementMapping{}, plan.Request.Mappings...)

	projectRoots := map[string]Store{}
	registryBlockers := []string{}
	projectIDs := resolver.Roots.Projects()
	if len(projectIDs) > maxSpecTransferProjects {
		registryBlockers = append(registryBlockers, "project-registry-limit:256")
		projectIDs = projectIDs[:maxSpecTransferProjects]
	}
	for _, id := range projectIDs {
		if resolver.Authorize != nil && !resolver.Authorize(id) {
			registryBlockers = append(registryBlockers, "unresolved-project:"+id)
			continue
		}
		store, err := resolver.Roots.StoreFor(id)
		if err != nil {
			registryBlockers = append(registryBlockers, "unavailable-project:"+id)
			continue
		}
		projectRoots[id] = store
	}
	for _, id := range []string{request.Source.Project, request.Destination.Project} {
		if resolver.Authorize != nil && !resolver.Authorize(id) {
			return plan, specTransferError("unauthorized-project")
		}
		store, err := resolver.Roots.StoreFor(id)
		if err != nil {
			return plan, specTransferError("unavailable-project")
		}
		projectRoots[id] = store
	}

	sourceStore := projectRoots[request.Source.Project]
	destinationStore := projectRoots[request.Destination.Project]
	if sameProjectRoot(sourceStore.Root, destinationStore.Root) {
		return plan, specTransferError("source-and-destination-must-be-distinct-projects")
	}
	if err := requireSpecTransferCapability(sourceStore); err != nil {
		return plan, err
	}
	if err := requireSpecTransferCapability(destinationStore); err != nil {
		return plan, err
	}
	sourceSpec, err := sourceStore.GetSpec(request.Source.Slug)
	if err != nil {
		return plan, specTransferError("source-spec-unavailable")
	}
	if sourceSpec.Status != "draft" && sourceSpec.Status != "in-progress" && sourceSpec.Status != "blocked" {
		return plan, specTransferError("source-spec-not-transferable")
	}
	sourcePath, sourceRaw, err := transferSpecFile(sourceStore, sourceSpec)
	if err != nil {
		return plan, err
	}
	if err := validateTransferSpecIdentityPath(sourceStore, request.Source.Slug, sourcePath); err != nil {
		return plan, err
	}
	if err := transferPathClean(sourceStore.Root, sourcePath); err != nil {
		return plan, err
	}
	if sourceRedirectPathExists(sourceStore.Root, request.Source.Slug) {
		return plan, specTransferError("source-already-redirected")
	}
	if _, active := incompleteTransferFor(sourceStore.Root, request.Source); active {
		return plan, specTransferError("transfer-in-progress")
	}

	destinationSpec, destErr := destinationStore.GetSpec(request.Destination.Slug)
	destinationExists := destErr == nil && destinationSpec != nil
	if destErr != nil {
		var conflict SpecIdentityConflictError
		if errors.As(destErr, &conflict) {
			return plan, specTransferError("destination-identity-conflict")
		}
		if !strings.HasSuffix(destErr.Error(), " not found") {
			return plan, specTransferError("destination-spec-unavailable")
		}
	}
	var destinationPath, destinationRaw string
	if destinationExists {
		if !validTransferLifecycle(destinationSpec.Status) {
			return plan, specTransferError("destination-spec-not-reconcilable")
		}
		destinationPath, destinationRaw, err = transferSpecFile(destinationStore, destinationSpec)
		if err != nil {
			return plan, err
		}
		if err := validateTransferSpecIdentityPath(destinationStore, request.Destination.Slug, destinationPath); err != nil {
			return plan, err
		}
		if err := transferPathClean(destinationStore.Root, destinationPath); err != nil {
			return plan, err
		}
		if _, active := incompleteTransferFor(destinationStore.Root, request.Destination); active {
			return plan, specTransferError("transfer-in-progress")
		}
	} else {
		if err := validateTransferSpecIdentityPathAbsent(destinationStore, request.Destination.Slug); err != nil {
			return plan, err
		}
		destinationPath = filepath.ToSlash(filepath.Join(".pose", "specs", effectiveDate+"-"+request.Destination.Slug+".md"))
		destinationRaw, err = renderTransferredSpec(sourceRaw, *sourceSpec, request.Source.Project, request.Destination.Slug, "blocked")
		if err != nil {
			return plan, err
		}
	}

	sourceReqs := requirementIDs(sourceSpec.Body)
	var destinationReqs []string
	if destinationExists {
		destinationReqs = requirementIDs(destinationSpec.Body)
	} else {
		destinationReqs = append([]string{}, sourceReqs...)
	}
	if err := validateRequirementMap(sourceReqs, destinationReqs, plan.Mappings); err != nil {
		return plan, err
	}

	finalStatus := sourceSpec.Status
	if destinationExists {
		finalStatus = destinationSpec.Status
	}
	for _, mapping := range plan.Mappings {
		if mapping.Disposition == "reformulated" || mapping.Disposition == "pending" {
			plan.RequiresFreshEvidence = true
		}
	}
	if plan.RequiresFreshEvidence {
		finalStatus = "blocked"
	}

	sourceRevision, err := transferGitRevision(sourceStore.Root)
	if err != nil {
		return plan, specTransferError("source-revision-unavailable")
	}
	destinationRevision, err := transferGitRevision(destinationStore.Root)
	if err != nil {
		return plan, specTransferError("destination-revision-unavailable")
	}
	plan.Source, plan.Destination = request.Source, request.Destination
	plan.SourceRevision, plan.DestinationRevision = sourceRevision, destinationRevision
	plan.SourceDigest, plan.DestinationExists = digestHex([]byte(sourceRaw)), destinationExists
	plan.DestinationDigest = digestHex([]byte(destinationRaw))
	plan.SourcePath, plan.DestinationPath = filepath.ToSlash(sourcePath), filepath.ToSlash(destinationPath)
	plan.SourceStatus = sourceSpec.Status
	plan.DestinationStatus = ""
	if destinationExists {
		plan.DestinationStatus = destinationSpec.Status
	}
	plan.FinalDestinationStatus = finalStatus
	plan.DestinationStageDigest = digestHex([]byte(setSpecLifecycle(destinationRaw, request.Destination.Slug, "blocked")))
	plan.DestinationFinalDigest = digestHex([]byte(setSpecLifecycle(destinationRaw, request.Destination.Slug, finalStatus)))
	plan.SourceStubDigest = digestHex([]byte(renderTransferRedirectStub(request.Source, request.Destination)))
	plan.SourceRedirectDigest = digestHex(redirectBytes(request.Source, request.Destination, ""))

	impacts, blockers, err := discoverTransferImpacts(projectRoots, request.Source, request.Destination)
	if err != nil {
		return plan, err
	}
	plan.ImpactedReferences, plan.Blockers = impacts, uniqueStrings(append(blockers, registryBlockers...))
	projects := map[string]bool{request.Source.Project: true, request.Destination.Project: true}
	for _, impact := range impacts {
		projects[impact.ProjectID] = true
	}
	for projectID := range projects {
		store := projectRoots[projectID]
		if store.Root == "" {
			plan.Blockers = append(plan.Blockers, "unresolved-project:"+projectID)
			continue
		}
		revision, err := transferGitRevision(store.Root)
		if err != nil {
			plan.Blockers = append(plan.Blockers, "revision-unavailable:"+projectID)
			continue
		}
		plan.ProjectRevisions = append(plan.ProjectRevisions, SpecTransferProjectRevision{ProjectID: projectID, Revision: revision})
		if err := requireSpecTransferCapability(store); err != nil {
			plan.Blockers = append(plan.Blockers, "capability-not-adopted:"+projectID)
		}
	}
	if len(projects) > maxSpecTransferProjects {
		return plan, specTransferError("affected-project-limit")
	}
	plan.AffectedProjects = make([]string, 0, len(projects))
	for projectID := range projects {
		plan.AffectedProjects = append(plan.AffectedProjects, projectID)
	}
	sort.Strings(plan.AffectedProjects)
	sort.Slice(plan.ProjectRevisions, func(i, j int) bool { return plan.ProjectRevisions[i].ProjectID < plan.ProjectRevisions[j].ProjectID })
	sort.Slice(plan.ImpactedReferences, func(i, j int) bool {
		a, b := plan.ImpactedReferences[i], plan.ImpactedReferences[j]
		if a.ProjectID != b.ProjectID {
			return a.ProjectID < b.ProjectID
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Owner.String() < b.Owner.String()
	})
	sort.Strings(plan.Blockers)
	plan.OperationID = transferOperationID(plan)
	plan.SourceRedirectDigest = digestHex(redirectBytes(request.Source, request.Destination, plan.OperationID))
	plan.Digest = specTransferPlanDigest(plan)
	return plan, nil
}

func canonicalRequirementMappings(in []SpecTransferRequirementMapping) []SpecTransferRequirementMapping {
	out := append([]SpecTransferRequirementMapping{}, in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].SourceRequirement != out[j].SourceRequirement {
			return out[i].SourceRequirement < out[j].SourceRequirement
		}
		if out[i].Disposition != out[j].Disposition {
			return out[i].Disposition < out[j].Disposition
		}
		return out[i].DestinationRequirement < out[j].DestinationRequirement
	})
	return out
}

func requirementIDs(body string) []string {
	section := body
	if start := strings.Index(body, "## 2."); start >= 0 {
		section = body[start:]
		if end := strings.Index(section, "## 3."); end >= 0 {
			section = section[:end]
		}
	}
	seen := map[string]bool{}
	var ids []string
	for _, match := range specRequirementIDRE.FindAllStringSubmatch(section, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			ids = append(ids, match[1])
		}
	}
	sort.Strings(ids)
	return ids
}

func validateRequirementMap(source, destination []string, mappings []SpecTransferRequirementMapping) error {
	sourceSet, destinationSet := map[string]bool{}, map[string]bool{}
	for _, id := range source {
		sourceSet[id] = true
	}
	for _, id := range destination {
		destinationSet[id] = true
	}
	coveredSource, coveredDestination := map[string]bool{}, map[string]bool{}
	for _, mapping := range mappings {
		if mapping.Disposition != "equivalent" && mapping.Disposition != "reformulated" && mapping.Disposition != "withdrawn" && mapping.Disposition != "pending" {
			return specTransferError("invalid-requirement-disposition")
		}
		if mapping.SourceRequirement != "" {
			if !sourceSet[mapping.SourceRequirement] || coveredSource[mapping.SourceRequirement] {
				return specTransferError("requirement-map-source-mismatch")
			}
			coveredSource[mapping.SourceRequirement] = true
		}
		if mapping.DestinationRequirement != "" {
			if !destinationSet[mapping.DestinationRequirement] {
				return specTransferError("requirement-map-destination-mismatch")
			}
			coveredDestination[mapping.DestinationRequirement] = true
		}
		switch mapping.Disposition {
		case "equivalent", "reformulated":
			if mapping.SourceRequirement == "" || mapping.DestinationRequirement == "" {
				return specTransferError("requirement-map-incomplete")
			}
		case "withdrawn":
			if mapping.SourceRequirement == "" || mapping.DestinationRequirement != "" {
				return specTransferError("requirement-map-incomplete")
			}
		case "pending":
			if mapping.SourceRequirement == "" && mapping.DestinationRequirement == "" {
				return specTransferError("requirement-map-incomplete")
			}
		}
	}
	for _, id := range source {
		if !coveredSource[id] {
			return specTransferError("requirement-map-incomplete")
		}
	}
	for _, id := range destination {
		if !coveredDestination[id] {
			return specTransferError("requirement-map-incomplete")
		}
	}
	return nil
}

func requireSpecTransferCapability(store Store) error {
	policy, err := store.GetReviewPolicy()
	if err != nil {
		return specTransferError("unsupported-policy")
	}
	if policy.SchemaVersion != SpecAuthorityTransferPolicySchemaVersion || policy.QualifiedArtifactRefsVersion != 1 || policy.SpecAuthorityTransferVersion != 1 {
		return specTransferError("spec-authority-transfer-capability-not-adopted")
	}
	return nil
}

func transferSpecFile(store Store, spec *Spec) (string, string, error) {
	if spec == nil || spec.Path == "" {
		return "", "", specTransferError("spec-path-unavailable")
	}
	if !artifactPathWithin(store.Root, spec.Path) {
		return "", "", specTransferError("path-escape")
	}
	info, err := os.Lstat(spec.Path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() || info.Size() > maxSpecTransferArtifactBytes {
		return "", "", specTransferError("unsupported-spec-layout")
	}
	raw, err := os.ReadFile(spec.Path)
	if err != nil {
		return "", "", specTransferError("spec-content-unavailable")
	}
	rel, err := filepath.Rel(store.Root, spec.Path)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", specTransferError("path-escape")
	}
	return filepath.ToSlash(rel), string(raw), nil
}

func validateTransferSpecIdentityPath(store Store, slug, selectedPath string) error {
	candidates, err := transferSpecIdentityCandidates(store, slug)
	if err != nil {
		return err
	}
	if len(candidates) != 1 || candidates[0] != selectedPath {
		return specTransferError("spec-identity-conflict")
	}
	return nil
}

func validateTransferSpecIdentityPathAbsent(store Store, slug string) error {
	candidates, err := transferSpecIdentityCandidates(store, slug)
	if err != nil {
		return err
	}
	if len(candidates) != 0 {
		return specTransferError("destination-identity-conflict")
	}
	return nil
}

func transferSpecIdentityCandidates(store Store, slug string) ([]string, error) {
	entries, err := os.ReadDir(store.specsDir())
	if err != nil {
		return nil, specTransferError("spec-index-unavailable")
	}
	candidates := []string{}
	for _, entry := range entries {
		name := entry.Name()
		candidateSlug := name
		if strings.HasSuffix(strings.ToLower(name), ".md") {
			candidateSlug = strings.TrimSuffix(name, filepath.Ext(name))
		}
		if specFilenameSlug(candidateSlug) != slug {
			continue
		}
		path := filepath.Join(store.specsDir(), name)
		if entry.IsDir() {
			if _, statErr := os.Lstat(filepath.Join(path, "spec.md")); statErr == nil {
				path = filepath.Join(path, "spec.md")
			}
		}
		rel, relErr := filepath.Rel(store.Root, path)
		if relErr != nil {
			return nil, specTransferError("path-escape")
		}
		candidates = append(candidates, filepath.ToSlash(rel))
	}
	return candidates, nil
}

func transferPathClean(root, relative string) error {
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
		return specTransferError("path-escape")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return specTransferError("path-escape")
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return specTransferError("path-escape")
	}
	current := base
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return specTransferError("path-escape")
		}
		rel, err := filepath.Rel(base, current)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return specTransferError("path-escape")
		}
	}
	return nil
}

func transferGitRevision(root string) (string, error) {
	top, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").Output()
	if err != nil || !sameProjectRoot(root, strings.TrimSpace(string(top))) {
		return "", errors.New("not-a-project-root")
	}
	head, err := exec.Command("git", "-C", root, "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(head)), nil
}

func discoverTransferImpacts(projects map[string]Store, source, destination ArtifactRef) ([]SpecTransferImpact, []string, error) {
	byFile := map[string]*SpecTransferImpact{}
	blockers := []string{}
	add := func(projectID string, owner ArtifactRef, path string, raw string, refs []string) error {
		if len(refs) == 0 {
			return nil
		}
		key := projectID + "\x00" + path
		impact := byFile[key]
		if impact == nil {
			rel, err := filepath.Rel(projects[projectID].Root, path)
			if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return specTransferError("path-escape")
			}
			info, err := os.Lstat(path)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxSpecTransferArtifactBytes {
				blockers = append(blockers, "reference-unavailable:"+owner.String())
				return nil
			}
			before, err := os.ReadFile(path)
			if err != nil {
				blockers = append(blockers, "reference-unavailable:"+owner.String())
				return nil
			}
			if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(projects[projectID].Root, path) {
				return specTransferError("path-escape")
			}
			impact = &SpecTransferImpact{ProjectID: projectID, Owner: owner, Path: filepath.ToSlash(rel), BeforeDigest: digestHex(before), To: destination.String()}
			byFile[key] = impact
			if len(byFile) > maxSpecTransferImpacts {
				blockers = append(blockers, "reference-inventory-limit:10000")
				delete(byFile, key)
				return nil
			}
		}
		impact.From = append(impact.From, refs...)
		return nil
	}
	projectIDs := make([]string, 0, len(projects))
	for projectID := range projects {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	for _, projectID := range projectIDs {
		store := projects[projectID]
		if _, err := os.Stat(filepath.Join(store.Root, ".pose")); err != nil {
			blockers = append(blockers, "unavailable-project:"+projectID)
			continue
		}
		specs, err := store.ListSpecs("", "")
		if err != nil {
			blockers = append(blockers, "unreadable-spec-index:"+projectID)
			continue
		}
		for _, spec := range specs {
			refs := matchingTransferRefs(projectID, spec.DependsOn, source)
			if len(refs) == 0 {
				continue
			}
			if len(byFile) >= maxSpecTransferImpacts {
				blockers = append(blockers, "reference-inventory-limit:10000")
				break
			}
			if info, err := os.Stat(spec.Path); err != nil || info.IsDir() {
				blockers = append(blockers, "unreadable-reference:"+(ArtifactRef{Project: projectID, Kind: "spec", Slug: spec.Slug}).String())
				continue
			}
			if err := add(projectID, ArtifactRef{Project: projectID, Kind: "spec", Slug: spec.Slug}, spec.Path, "", refs); err != nil {
				return nil, nil, err
			}
		}
		roadmaps, err := store.ListRoadmaps()
		if err != nil {
			blockers = append(blockers, "unreadable-roadmap-index:"+projectID)
			continue
		}
		for _, roadmap := range roadmaps {
			var refs []string
			refs = append(refs, matchingTransferRefs(projectID, roadmap.DependsOn, source)...)
			refs = append(refs, matchingTransferRefs(projectID, roadmap.Consumes, source)...)
			for _, milestone := range roadmap.Milestones {
				refs = append(refs, matchingTransferRefs(projectID, milestone.Specs, source)...)
				refs = append(refs, matchingTransferRefs(projectID, milestone.Consumes, source)...)
			}
			if len(refs) == 0 {
				continue
			}
			if len(byFile) >= maxSpecTransferImpacts {
				blockers = append(blockers, "reference-inventory-limit:10000")
				break
			}
			path := filepath.Join(store.roadmapsDir(), roadmap.Slug+".md")
			if err := add(projectID, ArtifactRef{Project: projectID, Kind: "roadmap", Slug: roadmap.Slug}, path, "", refs); err != nil {
				return nil, nil, err
			}
		}
	}
	impacts := make([]SpecTransferImpact, 0, len(byFile))
	for _, impact := range byFile {
		sort.Strings(impact.From)
		impact.From = uniqueStrings(impact.From)
		root := projects[impact.ProjectID].Root
		before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(impact.Path)))
		if err != nil {
			blockers = append(blockers, "reference-unavailable:"+impact.Owner.String())
			continue
		}
		after := rewriteTransferReferences(string(before), impact.From, impact.To, impact.Owner.Kind == "roadmap")
		impact.AfterDigest = digestHex([]byte(after))
		if impact.BeforeDigest == impact.AfterDigest {
			continue
		}
		impacts = append(impacts, *impact)
	}
	sort.Strings(blockers)
	return impacts, uniqueStrings(blockers), nil
}

func matchingTransferRefs(projectID string, refs []string, source ArtifactRef) []string {
	var out []string
	for _, raw := range refs {
		parsed, err := ParseArtifactRef(raw)
		if err != nil {
			continue
		}
		if parsed.Project == "" {
			parsed.Project = projectID
		}
		if parsed == source {
			out = append(out, raw)
		}
	}
	return out
}

func rewriteTransferReferences(content string, refs []string, replacement string, roadmap bool) string {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return content
	}
	front := strings.Split(parts[1], "\n")
	allowed := map[string]bool{"depends_on": true}
	if roadmap {
		allowed["consumes"] = true
	}
	activeKey, activeIndent := "", -1
	for i, line := range front {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if key, value, ok := strings.Cut(trimmed, ":"); ok && allowed[key] {
			activeKey, activeIndent = key, indent
			front[i] = replaceTransferRefsInValue(line, value, refs, replacement)
			continue
		}
		if activeKey != "" && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if indent <= activeIndent {
				activeKey = ""
			} else {
				front[i] = replaceTransferRefsInValue(line, trimmed, refs, replacement)
			}
		}
	}
	body := strings.Split(parts[2], "\n")
	inMilestone := false
	for i, line := range body {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Milestone:") {
			inMilestone = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			inMilestone = false
			continue
		}
		if !roadmap || !inMilestone || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), ":")
		if ok && (strings.TrimSpace(key) == "specs" || strings.TrimSpace(key) == "consumes") {
			body[i] = replaceTransferRefsInValue(line, value, refs, replacement)
		}
	}
	return parts[0] + "---" + strings.Join(front, "\n") + "---" + strings.Join(body, "\n")
}

func replaceTransferRefsInValue(line, value string, refs []string, replacement string) string {
	for _, ref := range refs {
		if ref != "" {
			line = strings.ReplaceAll(line, ref, replacement)
		}
	}
	return line
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func renderTransferredSpec(sourceRaw string, spec Spec, sourceProject, destinationSlug, status string) (string, error) {
	if _, body := SplitFrontmatter(sourceRaw); body == sourceRaw {
		return "", specTransferError("unsupported-spec-layout")
	}
	lines := strings.Split(sourceRaw, "\n")
	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontmatterEnd = i
			break
		}
	}
	if frontmatterEnd < 0 {
		return "", specTransferError("unsupported-spec-layout")
	}
	transferred := setSpecLifecycle(sourceRaw, destinationSlug, status)
	fm, _ := SplitFrontmatter(transferred)
	if fm["slug"] != destinationSlug || fm["status"] != status {
		return "", specTransferError("unsupported-spec-layout")
	}
	lines = strings.Split(transferred, "\n")
	for i := 1; i < frontmatterEnd; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok || strings.TrimSpace(key) != "depends_on" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			for j := i + 1; j < frontmatterEnd; j++ {
				child := strings.TrimSpace(lines[j])
				if child == "" || strings.HasPrefix(child, "#") {
					continue
				}
				childIndent := len(lines[j]) - len(strings.TrimLeft(lines[j], " \t"))
				if childIndent <= indent {
					break
				}
				return "", specTransferError("unsupported-spec-layout")
			}
			continue
		}
		if len(spec.DependsOn) == 0 {
			continue
		}
		qualified := strings.Join(qualifyTransferredDependencies(spec.DependsOn, sourceProject), ", ")
		if strings.HasPrefix(strings.TrimSpace(value), "[") && strings.HasSuffix(strings.TrimSpace(value), "]") {
			qualified = "[" + qualified + "]"
		}
		colon := strings.Index(line, ":")
		lines[i] = line[:colon+1] + " " + qualified
	}
	return strings.Join(lines, "\n"), nil
}

func qualifyTransferredDependencies(dependencies []string, sourceProject string) []string {
	out := make([]string, 0, len(dependencies))
	for _, raw := range dependencies {
		ref, err := ParseArtifactRef(raw)
		if err != nil {
			out = append(out, raw)
			continue
		}
		if ref.Project == "" {
			ref.Project = sourceProject
		}
		out = append(out, ref.String())
	}
	return out
}

func setSpecLifecycle(content, slug, status string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	statusUpdated := false
	slugUpdated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && trimmed == "---" {
			break
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(trimmed, "status:") {
			lines[i] = "status: " + status
			statusUpdated = true
		}
		if strings.HasPrefix(trimmed, "slug:") {
			lines[i] = "slug: " + slug
			slugUpdated = true
		}
	}
	if !statusUpdated || !slugUpdated {
		return content
	}
	return strings.Join(lines, "\n")
}

func renderTransferRedirectStub(source, destination ArtifactRef) string {
	return fmt.Sprintf("---\nslug: %s\nstatus: superseded\n---\n# Authority transferred\n\nCanonical task: `%s`.\n", source.Slug, destination.String())
}

func redirectBytes(source, destination ArtifactRef, operationID string) []byte {
	return CanonicalJSON(specTransferRedirect{SchemaVersion: 1, OperationID: operationID, Source: source, Destination: destination})
}

func digestHex(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func transferOperationID(plan SpecTransferPlan) string {
	plan.OperationID, plan.Digest = "", ""
	plan.SourceRedirectDigest = ""
	raw, _ := json.Marshal(plan)
	return "stf-" + digestHex(raw)[:24]
}

func specTransferPlanDigest(plan SpecTransferPlan) string {
	plan.Digest = ""
	raw, _ := json.Marshal(plan)
	return digestHex(raw)
}

func sourceRedirectPath(root, slug string) string {
	return filepath.Join(root, ".pose", "transfers", "redirects", slug+".json")
}

func sourceRedirectPathExists(root, slug string) bool {
	_, err := os.Lstat(sourceRedirectPath(root, slug))
	return err == nil
}

func transferOperationDir(root, operationID string) string {
	return filepath.Join(root, ".pose", "transfers", operationID)
}

func transferPlanFile(root, operationID string) string {
	return filepath.Join(transferOperationDir(root, operationID), "plan.json")
}

func incompleteTransferFor(root string, identity ArtifactRef) (string, bool) {
	transfersDir := filepath.Join(root, ".pose", "transfers")
	info, err := os.Lstat(transfersDir)
	if os.IsNotExist(err) {
		return "", false
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(root, transfersDir) {
		return "transfer-state-invalid", true
	}
	entries, err := os.ReadDir(transfersDir)
	if err != nil {
		return "transfer-state-unavailable", true
	}
	if len(entries) > maxSpecTransferOperationsRead {
		return "transfer-state-invalid", true
	}
	for _, entry := range entries {
		if !entry.IsDir() || !specTransferIDRE.MatchString(entry.Name()) {
			continue
		}
		plan, err := readSpecTransferPlan(transferPlanFile(root, entry.Name()))
		if err != nil || plan.OperationID != entry.Name() {
			return "transfer-state-invalid", true
		}
		impacted := false
		for _, impact := range plan.ImpactedReferences {
			if impact.ProjectID == identity.Project && impact.Owner == identity {
				impacted = true
				break
			}
		}
		if plan.Source != identity && plan.Destination != identity && !impacted {
			continue
		}
		status, err := ReadSpecTransferStatus(Store{Root: root}, entry.Name(), identity.Project)
		if err == nil && status.Phase != "activated" {
			return entry.Name(), true
		}
		if err != nil {
			return "transfer-state-invalid", true
		}
	}
	return "", false
}

func readSpecTransferRedirect(store Store, identity ArtifactRef) (specTransferRedirect, bool, error) {
	path := sourceRedirectPath(store.Root, identity.Slug)
	redirectRel, _ := filepath.Rel(store.Root, path)
	if err := transferPathClean(store.Root, filepath.ToSlash(redirectRel)); err != nil {
		return specTransferRedirect{}, false, specTransferError("invalid-spec-redirect")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return specTransferRedirect{}, false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(store.Root, path) {
		return specTransferRedirect{}, false, specTransferError("invalid-spec-redirect")
	}
	if info.Size() > 64<<10 {
		return specTransferRedirect{}, false, specTransferError("redirect-too-large")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return specTransferRedirect{}, false, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var redirect specTransferRedirect
	if err := decoder.Decode(&redirect); err != nil {
		return redirect, false, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return redirect, false, specTransferError("invalid-spec-redirect")
	}
	if redirect.SchemaVersion != SpecTransferSchemaVersion || !specTransferIDRE.MatchString(redirect.OperationID) || redirect.Source != identity || validateTransferIdentity(redirect.Destination) != nil || redirect.Destination.Project == identity.Project {
		return redirect, false, specTransferError("invalid-spec-redirect")
	}
	if err := requireSpecTransferCapability(store); err != nil {
		return redirect, false, err
	}
	planRel := filepath.ToSlash(filepath.Join(".pose", "transfers", redirect.OperationID, "plan.json"))
	if err := transferPathClean(store.Root, planRel); err != nil {
		return redirect, false, specTransferError("invalid-spec-redirect")
	}
	plan, err := readSpecTransferPlan(transferPlanFile(store.Root, redirect.OperationID))
	if err != nil || validateTransferPlan(plan) != nil || plan.Source != redirect.Source || plan.Destination != redirect.Destination {
		return redirect, false, specTransferError("invalid-spec-redirect")
	}
	status, err := ReadSpecTransferStatus(store, redirect.OperationID, identity.Project)
	if err != nil || status.Phase != "activated" {
		return redirect, false, specTransferError("invalid-spec-redirect")
	}
	return redirect, true, nil
}

func readSpecTransferPlan(path string) (SpecTransferPlan, error) {
	var plan SpecTransferPlan
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > 2<<20 {
		return plan, specTransferError("invalid-plan")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return plan, err
	}
	if len(raw) > 2<<20 {
		return plan, specTransferError("plan-too-large")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return plan, specTransferError("invalid-plan")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return plan, specTransferError("invalid-plan")
	}
	if plan.SchemaVersion != SpecTransferSchemaVersion || !specTransferIDRE.MatchString(plan.OperationID) || plan.OperationID != transferOperationID(plan) || plan.Digest == "" || plan.Digest != specTransferPlanDigest(plan) {
		return plan, specTransferError("plan-digest-mismatch")
	}
	return plan, nil
}

func writeTransferJSONExclusive(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return specTransferError("path-escape")
	}
	raw := CanonicalJSON(value)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			previous, readErr := os.ReadFile(path)
			if readErr == nil && bytes.Equal(previous, raw) {
				return nil
			}
			return specTransferError("append-only-record-conflict")
		}
		return err
	}
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeTransferFile(path string, raw []byte, expectedCurrentDigest string) error {
	expectedAfterDigest := digestHex(raw)
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
			return specTransferError("path-escape")
		}
		current, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		currentDigest := digestHex(current)
		if currentDigest == expectedAfterDigest {
			return nil
		}
		if expectedCurrentDigest == "" || currentDigest != expectedCurrentDigest {
			return specTransferError("compare-and-swap-conflict")
		}
	} else if os.IsNotExist(err) {
		if expectedCurrentDigest != "" {
			return specTransferError("compare-and-swap-conflict")
		}
	} else {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pose-transfer-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
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
	return os.Rename(name, path)
}

func ReadSpecTransferStatus(store Store, operationID, projectID string) (SpecTransferStatus, error) {
	status := SpecTransferStatus{SchemaVersion: SpecTransferSchemaVersion, OperationID: operationID, ProjectID: projectID}
	if !specTransferIDRE.MatchString(operationID) {
		return status, specTransferError("invalid-operation-id")
	}
	planRel := filepath.ToSlash(filepath.Join(".pose", "transfers", operationID, "plan.json"))
	if err := transferPathClean(store.Root, planRel); err != nil {
		return status, specTransferError("operation-not-found")
	}
	plan, err := readSpecTransferPlan(transferPlanFile(store.Root, operationID))
	if err != nil {
		return status, specTransferError("operation-not-found")
	}
	if projectID == "" {
		projectID = plan.Source.Project
	}
	if !transferContains(plan.AffectedProjects, projectID) {
		return status, specTransferError("operation-not-found")
	}
	status.ProjectID, status.PlanDigest, status.Source, status.Destination = projectID, plan.Digest, plan.Source, plan.Destination
	status.Phase = "planned"
	journal := filepath.Join(transferOperationDir(store.Root, operationID), "journal")
	journalRel, _ := filepath.Rel(store.Root, journal)
	if err := transferPathClean(store.Root, filepath.ToSlash(journalRel)); err != nil {
		return status, specTransferError("journal-unavailable")
	}
	entries, err := os.ReadDir(journal)
	if err != nil {
		if os.IsNotExist(err) {
			return status, nil
		}
		return status, specTransferError("journal-unavailable")
	}
	if len(entries) > maxSpecTransferJournalEntries {
		return status, specTransferError("journal-invalid")
	}
	phaseRank := map[string]int{"planned": 0, "prepared": 1, "source-retired": 2, "activated": 3}
	seenPhases := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		receiptPath := filepath.Join(journal, entry.Name())
		info, err := os.Lstat(receiptPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > 256<<10 {
			return status, specTransferError("journal-invalid")
		}
		raw, err := os.ReadFile(receiptPath)
		if err != nil {
			return status, specTransferError("journal-unavailable")
		}
		var receipt specTransferReceipt
		if json.Unmarshal(raw, &receipt) != nil || receipt.SchemaVersion != SpecTransferSchemaVersion || receipt.OperationID != operationID || receipt.PlanDigest != plan.Digest {
			return status, specTransferError("journal-invalid")
		}
		if entry.Name() != receipt.Phase+".json" || phaseRank[receipt.Phase] == 0 && receipt.Phase != "planned" || seenPhases[receipt.Phase] {
			return status, specTransferError("journal-invalid")
		}
		if !transferContains(plan.AffectedProjects, receipt.ProjectID) || receipt.Phase == "source-retired" && receipt.ProjectID != plan.Source.Project {
			return status, specTransferError("journal-invalid")
		}
		seenPhases[receipt.Phase] = true
		if receipt.ProjectID == projectID && phaseRank[receipt.Phase] > phaseRank[status.Phase] {
			status.Phase, status.UpdatedAt = receipt.Phase, receipt.RecordedAt
		}
	}
	return status, nil
}

func specTransferSourceRetired(plan SpecTransferPlan, store Store) bool {
	sourcePath := filepath.Join(store.Root, filepath.FromSlash(plan.SourcePath))
	archivePath := filepath.Join(transferOperationDir(store.Root, plan.OperationID), "source-spec.md")
	redirectPath := sourceRedirectPath(store.Root, plan.Source.Slug)
	for path, expected := range map[string]string{
		sourcePath:   plan.SourceStubDigest,
		archivePath:  plan.SourceDigest,
		redirectPath: plan.SourceRedirectDigest,
	} {
		raw, err := os.ReadFile(path)
		if err != nil || digestHex(raw) != expected {
			return false
		}
	}
	status, err := ReadSpecTransferStatus(store, plan.OperationID, plan.Source.Project)
	return err == nil && (status.Phase == "source-retired" || status.Phase == "activated")
}

func ApplySpecTransfer(resolver ArtifactResolver, plan SpecTransferPlan, expectedDigest string, authorizedProjects map[string]bool, failAfter SpecTransferFailureHook) (SpecTransferStatus, error) {
	if err := validateTransferPlan(plan); err != nil {
		return SpecTransferStatus{}, err
	}
	if expectedDigest == "" || expectedDigest != plan.Digest {
		return SpecTransferStatus{}, specTransferError("plan-digest-mismatch")
	}
	current, err := PreviewSpecTransfer(resolver, plan.Request, plan.EffectiveDate)
	if err != nil {
		return SpecTransferStatus{}, err
	}
	if current.Digest != plan.Digest {
		return SpecTransferStatus{}, specTransferError("stale-plan")
	}
	return advanceSpecTransfer(resolver, plan, authorizedProjects, failAfter)
}

func ResumeSpecTransfer(resolver ArtifactResolver, projectID, operationID string, authorizedProjects map[string]bool, failAfter SpecTransferFailureHook) (SpecTransferStatus, error) {
	if resolver.Roots == nil || !specTransferIDRE.MatchString(operationID) {
		return SpecTransferStatus{}, specTransferError("invalid-operation-id")
	}
	if resolver.Authorize != nil && !resolver.Authorize(projectID) {
		return SpecTransferStatus{}, specTransferError("unauthorized-project")
	}
	selected, err := resolver.Roots.StoreFor(projectID)
	if err != nil {
		return SpecTransferStatus{}, specTransferError("unavailable-project")
	}
	planRel := filepath.ToSlash(filepath.Join(".pose", "transfers", operationID, "plan.json"))
	if err := transferPathClean(selected.Root, planRel); err != nil {
		return SpecTransferStatus{}, specTransferError("operation-not-found")
	}
	plan, err := readSpecTransferPlan(transferPlanFile(selected.Root, operationID))
	if err != nil {
		return SpecTransferStatus{}, specTransferError("operation-not-found")
	}
	if !transferContains(plan.AffectedProjects, projectID) {
		return SpecTransferStatus{}, specTransferError("operation-not-found")
	}
	for _, id := range plan.AffectedProjects {
		if resolver.Authorize != nil && !resolver.Authorize(id) {
			return SpecTransferStatus{}, specTransferError("unauthorized-project")
		}
		store, err := resolver.Roots.StoreFor(id)
		if err != nil {
			return SpecTransferStatus{}, specTransferError("unavailable-project")
		}
		otherRel := filepath.ToSlash(filepath.Join(".pose", "transfers", operationID, "plan.json"))
		if err := transferPathClean(store.Root, otherRel); err != nil {
			return SpecTransferStatus{}, specTransferError("operation-record-conflict")
		}
		otherPath := transferPlanFile(store.Root, operationID)
		if _, err := os.Lstat(otherPath); os.IsNotExist(err) {
			continue
		}
		other, err := readSpecTransferPlan(otherPath)
		if err != nil || other.Digest != plan.Digest {
			return SpecTransferStatus{}, specTransferError("operation-record-conflict")
		}
	}
	return advanceSpecTransfer(resolver, plan, authorizedProjects, failAfter)
}

func validateTransferPlan(plan SpecTransferPlan) error {
	if plan.SchemaVersion != SpecTransferSchemaVersion || !specTransferIDRE.MatchString(plan.OperationID) || plan.OperationID != transferOperationID(plan) || plan.Digest == "" || plan.Digest != specTransferPlanDigest(plan) {
		return specTransferError("plan-digest-mismatch")
	}
	if err := validateTransferIdentity(plan.Source); err != nil {
		return err
	}
	if err := validateTransferIdentity(plan.Destination); err != nil {
		return err
	}
	if plan.Source != plan.Request.Source || plan.Destination != plan.Request.Destination || plan.Source.Project == plan.Destination.Project || !transferContains(plan.AffectedProjects, plan.Source.Project) || !transferContains(plan.AffectedProjects, plan.Destination.Project) {
		return specTransferError("invalid-plan")
	}
	if !validTransferLifecycle(plan.SourceStatus) || plan.DestinationExists && !validTransferLifecycle(plan.DestinationStatus) || !plan.DestinationExists && plan.DestinationStatus != "" || !validTransferLifecycle(plan.FinalDestinationStatus) {
		return specTransferError("invalid-plan")
	}
	if !reflect.DeepEqual(plan.Mappings, plan.Request.Mappings) || !reflect.DeepEqual(plan.Mappings, canonicalRequirementMappings(plan.Mappings)) {
		return specTransferError("invalid-plan")
	}
	if !isGitRevision(plan.SourceRevision) || !isGitRevision(plan.DestinationRevision) || !isDigest(plan.SourceDigest) || !isDigest(plan.DestinationStageDigest) || !isDigest(plan.DestinationFinalDigest) || !isDigest(plan.SourceStubDigest) || !isDigest(plan.SourceRedirectDigest) {
		return specTransferError("invalid-plan")
	}
	if _, err := time.Parse(time.DateOnly, plan.EffectiveDate); err != nil {
		return specTransferError("invalid-plan")
	}
	if len(plan.AffectedProjects) == 0 || len(plan.AffectedProjects) > maxSpecTransferProjects || len(plan.ProjectRevisions) != len(plan.AffectedProjects) || len(plan.ImpactedReferences) > maxSpecTransferImpacts || len(plan.Mappings) > 10000 || len(plan.Blockers) > 10000 {
		return specTransferError("invalid-plan")
	}
	seenProjects := map[string]bool{}
	revisionsByProject := map[string]string{}
	for _, snapshot := range plan.ProjectRevisions {
		if ValidateSlug(snapshot.ProjectID) != nil || !transferContains(plan.AffectedProjects, snapshot.ProjectID) || seenProjects[snapshot.ProjectID] || !isGitRevision(snapshot.Revision) {
			return specTransferError("invalid-plan")
		}
		seenProjects[snapshot.ProjectID] = true
		revisionsByProject[snapshot.ProjectID] = snapshot.Revision
	}
	for _, projectID := range plan.AffectedProjects {
		if !seenProjects[projectID] {
			return specTransferError("invalid-plan")
		}
	}
	if revisionsByProject[plan.Source.Project] != plan.SourceRevision || revisionsByProject[plan.Destination.Project] != plan.DestinationRevision {
		return specTransferError("invalid-plan")
	}
	expectedProjects := map[string]bool{plan.Source.Project: true, plan.Destination.Project: true}
	seenImpactPaths := map[string]bool{}
	for _, impact := range plan.ImpactedReferences {
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(impact.Path)))
		key := impact.ProjectID + "\x00" + path
		if ValidateSlug(impact.ProjectID) != nil || !transferContains(plan.AffectedProjects, impact.ProjectID) || impact.Owner.Project != impact.ProjectID || ValidateSlug(impact.Owner.Project) != nil || ValidateSlug(impact.Owner.Slug) != nil || path != impact.Path || impact.To != plan.Destination.String() || len(impact.From) == 0 || !isDigest(impact.BeforeDigest) || !isDigest(impact.AfterDigest) || !transferImpactPathMatchesOwner(path, impact.Owner) || seenImpactPaths[key] {
			return specTransferError("invalid-plan")
		}
		seenImpactPaths[key] = true
		expectedProjects[impact.ProjectID] = true
		seenRefs := map[string]bool{}
		for _, rawRef := range impact.From {
			ref, err := ParseArtifactRef(rawRef)
			if err != nil || seenRefs[rawRef] {
				return specTransferError("invalid-plan")
			}
			seenRefs[rawRef] = true
			if ref.Project == "" {
				ref.Project = impact.ProjectID
			}
			if ref != plan.Source {
				return specTransferError("invalid-plan")
			}
		}
	}
	if len(expectedProjects) != len(plan.AffectedProjects) {
		return specTransferError("invalid-plan")
	}
	for _, projectID := range plan.AffectedProjects {
		if !expectedProjects[projectID] {
			return specTransferError("invalid-plan")
		}
	}
	if !transferSpecPathMatchesSlug(plan.SourcePath, plan.Source.Slug) || !transferSpecPathMatchesSlug(plan.DestinationPath, plan.Destination.Slug) {
		return specTransferError("path-escape")
	}
	if plan.DestinationExists && !isDigest(plan.DestinationDigest) {
		return specTransferError("invalid-plan")
	}
	requiresFreshEvidence := false
	for _, mapping := range plan.Mappings {
		if mapping.Disposition == "reformulated" || mapping.Disposition == "pending" {
			requiresFreshEvidence = true
		}
	}
	if plan.RequiresFreshEvidence != requiresFreshEvidence || requiresFreshEvidence && plan.FinalDestinationStatus != "blocked" {
		return specTransferError("invalid-plan")
	}
	expectedFinalStatus := plan.SourceStatus
	if plan.DestinationExists {
		expectedFinalStatus = plan.DestinationStatus
	}
	if requiresFreshEvidence {
		expectedFinalStatus = "blocked"
	}
	if plan.FinalDestinationStatus != expectedFinalStatus {
		return specTransferError("invalid-plan")
	}
	return nil
}

func validTransferLifecycle(status string) bool {
	return status == "draft" || status == "in-progress" || status == "blocked"
}

func isDigest(value string) bool { return len(value) == 64 && isGitRevision(value) }

func transferSpecPathMatchesSlug(path, slug string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if !strings.HasPrefix(clean, ".pose/specs/") || !strings.HasSuffix(clean, ".md") {
		return false
	}
	if specFilenameSlug(strings.TrimSuffix(filepath.Base(clean), ".md")) == slug {
		return true
	}
	return strings.Count(clean, "/") == 3 && filepath.Base(clean) == "spec.md" && specFilenameSlug(filepath.Base(filepath.Dir(clean))) == slug
}

func transferImpactPathMatchesOwner(path string, owner ArtifactRef) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if ValidateSlug(owner.Slug) != nil || owner.Project == "" || ValidateSlug(owner.Project) != nil {
		return false
	}
	switch owner.Kind {
	case "spec":
		return transferSpecPathMatchesSlug(clean, owner.Slug)
	case "roadmap":
		return clean == filepath.ToSlash(filepath.Join(".pose", "roadmaps", owner.Slug+".md"))
	default:
		return false
	}
}

func validateSpecTransferPlanRoots(plan SpecTransferPlan, stores map[string]Store) error {
	sourceStore, destinationStore := stores[plan.Source.Project], stores[plan.Destination.Project]
	if sourceStore.Root == "" || destinationStore.Root == "" || sameProjectRoot(sourceStore.Root, destinationStore.Root) {
		return specTransferError("invalid-plan")
	}
	source, err := sourceStore.GetSpec(plan.Source.Slug)
	if err != nil {
		return specTransferError("source-spec-unavailable")
	}
	sourcePath, _, err := transferSpecFile(sourceStore, source)
	if err != nil || sourcePath != plan.SourcePath {
		return specTransferError("invalid-plan")
	}
	if err := validateTransferSpecIdentityPath(sourceStore, plan.Source.Slug, sourcePath); err != nil {
		return err
	}
	if plan.DestinationExists {
		destination, err := destinationStore.GetSpec(plan.Destination.Slug)
		if err != nil {
			return specTransferError("destination-spec-unavailable")
		}
		destinationPath, _, err := transferSpecFile(destinationStore, destination)
		if err != nil || destinationPath != plan.DestinationPath {
			return specTransferError("invalid-plan")
		}
		if err := validateTransferSpecIdentityPath(destinationStore, plan.Destination.Slug, destinationPath); err != nil {
			return err
		}
	} else {
		expected := filepath.ToSlash(filepath.Join(".pose", "specs", plan.EffectiveDate+"-"+plan.Destination.Slug+".md"))
		if plan.DestinationPath != expected {
			return specTransferError("invalid-plan")
		}
		if err := validateTransferSpecIdentityPathAbsent(destinationStore, plan.Destination.Slug); err != nil {
			return err
		}
	}
	for _, impact := range plan.ImpactedReferences {
		store := stores[impact.ProjectID]
		if err := transferPathClean(store.Root, impact.Path); err != nil {
			return err
		}
		if impact.Owner.Kind == "spec" {
			owner, err := store.GetSpec(impact.Owner.Slug)
			if err != nil {
				return specTransferError("reference-unavailable")
			}
			path, _, err := transferSpecFile(store, owner)
			if err != nil || path != impact.Path {
				return specTransferError("invalid-plan")
			}
		} else if impact.Owner.Kind != "roadmap" || impact.Path != filepath.ToSlash(filepath.Join(".pose", "roadmaps", impact.Owner.Slug+".md")) {
			return specTransferError("invalid-plan")
		}
	}
	return nil
}

func validateTransferReferenceInventory(projects map[string]Store, plan SpecTransferPlan) error {
	current, blockers, err := discoverTransferImpacts(projects, plan.Source, plan.Destination)
	if err != nil {
		return err
	}
	if len(blockers) > 0 {
		return specTransferError("reference-inventory-unresolved")
	}
	planned := make(map[string]SpecTransferImpact, len(plan.ImpactedReferences))
	for _, impact := range plan.ImpactedReferences {
		planned[impact.ProjectID+"\x00"+impact.Path] = impact
	}
	for _, impact := range current {
		prior, ok := planned[impact.ProjectID+"\x00"+impact.Path]
		if !ok {
			return specTransferError("source-reference-inventory-changed")
		}
		for _, ref := range impact.From {
			if !transferContains(prior.From, ref) {
				return specTransferError("source-reference-inventory-changed")
			}
		}
	}
	return nil
}

func ensureTransferOperationDir(root, operationID string) error {
	base := filepath.Join(root, ".pose", "transfers")
	info, err := os.Lstat(base)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(root, base) {
		return specTransferError("path-escape")
	}
	path := transferOperationDir(root, operationID)
	if info, err := os.Lstat(path); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(root, path) {
			return specTransferError("path-escape")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(path, 0o755)
}

func ensureTransferJournalDir(root, operationID string) error {
	if err := ensureTransferOperationDir(root, operationID); err != nil {
		return err
	}
	path := filepath.Join(transferOperationDir(root, operationID), "journal")
	if info, err := os.Lstat(path); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(root, path) {
			return specTransferError("path-escape")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Mkdir(path, 0o755)
}

func advanceSpecTransfer(resolver ArtifactResolver, plan SpecTransferPlan, authorizedProjects map[string]bool, failAfter SpecTransferFailureHook) (SpecTransferStatus, error) {
	status := SpecTransferStatus{SchemaVersion: SpecTransferSchemaVersion, OperationID: plan.OperationID, PlanDigest: plan.Digest}
	if err := validateTransferPlan(plan); err != nil {
		return status, err
	}
	if len(plan.Blockers) != 0 {
		return status, specTransferError("unresolved-impact-blockers")
	}
	stores := map[string]Store{}
	for _, id := range plan.AffectedProjects {
		if !authorizedProjects[id] {
			return status, specTransferError("write-authorization-required")
		}
		if resolver.Authorize != nil && !resolver.Authorize(id) {
			return status, specTransferError("unauthorized-project")
		}
		store, err := resolver.Roots.StoreFor(id)
		if err != nil {
			return status, specTransferError("unavailable-project")
		}
		if err := requireSpecTransferCapability(store); err != nil {
			return status, err
		}
		poseDir := filepath.Join(store.Root, ".pose")
		info, err := os.Lstat(poseDir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(store.Root, poseDir) {
			return status, specTransferError("path-escape")
		}
		stores[id] = store
	}
	inventoryStores := map[string]Store{}
	projectIDs := resolver.Roots.Projects()
	if len(projectIDs) > maxSpecTransferProjects {
		return status, specTransferError("project-registry-limit")
	}
	for _, id := range projectIDs {
		if resolver.Authorize != nil && !resolver.Authorize(id) {
			return status, specTransferError("unresolved-impact-blockers")
		}
		store, err := resolver.Roots.StoreFor(id)
		if err != nil {
			return status, specTransferError("unresolved-impact-blockers")
		}
		poseDir := filepath.Join(store.Root, ".pose")
		info, err := os.Lstat(poseDir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !artifactPathWithin(store.Root, poseDir) {
			return status, specTransferError("unresolved-impact-blockers")
		}
		inventoryStores[id] = store
	}
	locks, err := lockSpecTransferRoots(stores)
	if err != nil {
		return status, err
	}
	defer locks()
	if err := validateSpecTransferPlanRoots(plan, stores); err != nil {
		return status, err
	}
	if err := validateTransferReferenceInventory(inventoryStores, plan); err != nil {
		return status, err
	}
	for _, snapshot := range plan.ProjectRevisions {
		store := stores[snapshot.ProjectID]
		revision, err := transferGitRevision(store.Root)
		if err != nil || revision != snapshot.Revision {
			return status, specTransferError("compare-and-swap-conflict")
		}
	}
	for _, id := range plan.AffectedProjects {
		if err := ensureTransferOperationDir(stores[id].Root, plan.OperationID); err != nil {
			return status, specTransferError("operation-store-unavailable")
		}
		if err := writeTransferJSONExclusive(transferPlanFile(stores[id].Root, plan.OperationID), plan); err != nil {
			return status, err
		}
	}
	if failAfter != nil {
		if err := failAfter("planned"); err != nil {
			return status, err
		}
	}

	filesByProject := map[string][]string{}
	for _, impact := range plan.ImpactedReferences {
		store := stores[impact.ProjectID]
		path := filepath.Join(store.Root, filepath.FromSlash(impact.Path))
		if err := transferPathClean(store.Root, impact.Path); err != nil {
			return status, err
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return status, specTransferError("reference-unavailable")
		}
		currentDigest := digestHex(current)
		if currentDigest == impact.AfterDigest {
			filesByProject[impact.ProjectID] = append(filesByProject[impact.ProjectID], impact.Path)
			continue
		}
		if currentDigest != impact.BeforeDigest {
			return status, specTransferError("compare-and-swap-conflict")
		}
		after := []byte(rewriteTransferReferences(string(current), impact.From, impact.To, impact.Owner.Kind == "roadmap"))
		if digestHex(after) != impact.AfterDigest {
			return status, specTransferError("plan-digest-mismatch")
		}
		if err := writeTransferFile(path, after, impact.BeforeDigest); err != nil {
			return status, err
		}
		filesByProject[impact.ProjectID] = append(filesByProject[impact.ProjectID], impact.Path)
	}
	destinationStore := stores[plan.Destination.Project]
	destinationPath := filepath.Join(destinationStore.Root, filepath.FromSlash(plan.DestinationPath))
	if err := transferPathClean(destinationStore.Root, plan.DestinationPath); err != nil {
		return status, err
	}
	destinationCurrent, err := os.ReadFile(destinationPath)
	if err == nil {
		currentDigest := digestHex(destinationCurrent)
		sourceRetired := specTransferSourceRetired(plan, stores[plan.Source.Project])
		if currentDigest == plan.DestinationStageDigest {
			// A previous run has already made the destination non-executable.
		} else if currentDigest == plan.DestinationFinalDigest && sourceRetired {
			// The source retirement receipt proves this is an idempotent replay after activation.
		} else if currentDigest == plan.DestinationDigest {
			if currentDigest == plan.DestinationFinalDigest && sourceRetired {
				// The intended final status already matched the original executor status.
			} else {
				staged := []byte(setSpecLifecycle(string(destinationCurrent), plan.Destination.Slug, "blocked"))
				if digestHex(staged) != plan.DestinationStageDigest {
					return status, specTransferError("plan-digest-mismatch")
				}
				if err := writeTransferFile(destinationPath, staged, plan.DestinationDigest); err != nil {
					return status, err
				}
			}
		} else {
			return status, specTransferError("compare-and-swap-conflict")
		}
	} else if os.IsNotExist(err) && !plan.DestinationExists {
		sourceStore := stores[plan.Source.Project]
		sourcePath := filepath.Join(sourceStore.Root, filepath.FromSlash(plan.SourcePath))
		sourceRaw, err := os.ReadFile(sourcePath)
		if err != nil || digestHex(sourceRaw) != plan.SourceDigest {
			return status, specTransferError("source-changed-before-stage")
		}
		sourceSpec, err := sourceStore.GetSpec(plan.Source.Slug)
		if err != nil {
			return status, specTransferError("source-spec-unavailable")
		}
		stagedBody, err := renderTransferredSpec(string(sourceRaw), *sourceSpec, plan.Source.Project, plan.Destination.Slug, "blocked")
		if err != nil {
			return status, err
		}
		staged := []byte(stagedBody)
		if digestHex(staged) != plan.DestinationStageDigest {
			return status, specTransferError("plan-digest-mismatch")
		}
		if err := writeTransferFile(destinationPath, staged, ""); err != nil {
			return status, err
		}
	} else if err != nil {
		return status, specTransferError("destination-spec-unavailable")
	}
	filesByProject[plan.Destination.Project] = append(filesByProject[plan.Destination.Project], plan.DestinationPath)
	for _, id := range plan.AffectedProjects {
		files := uniqueStrings(filesByProject[id])
		if err := recordTransferReceipt(stores[id].Root, plan, id, "prepared", files); err != nil {
			return status, err
		}
	}
	if failAfter != nil {
		if err := failAfter("prepared"); err != nil {
			return status, err
		}
	}

	sourceStore := stores[plan.Source.Project]
	sourcePath := filepath.Join(sourceStore.Root, filepath.FromSlash(plan.SourcePath))
	archivePath := filepath.Join(transferOperationDir(sourceStore.Root, plan.OperationID), "source-spec.md")
	if err := transferPathClean(sourceStore.Root, filepath.ToSlash(strings.TrimPrefix(archivePath, sourceStore.Root+string(filepath.Separator)))); err != nil {
		return status, err
	}
	sourceCurrent, err := os.ReadFile(sourcePath)
	if err != nil {
		return status, specTransferError("source-spec-unavailable")
	}
	if digestHex(sourceCurrent) == plan.SourceDigest {
		if err := writeTransferFile(archivePath, sourceCurrent, ""); err != nil {
			return status, err
		}
		stub := []byte(renderTransferRedirectStub(plan.Source, plan.Destination))
		if digestHex(stub) != plan.SourceStubDigest {
			return status, specTransferError("plan-digest-mismatch")
		}
		if err := writeTransferFile(sourcePath, stub, plan.SourceDigest); err != nil {
			return status, err
		}
	} else if digestHex(sourceCurrent) != plan.SourceStubDigest {
		return status, specTransferError("compare-and-swap-conflict")
	} else {
		archive, err := os.ReadFile(archivePath)
		if err != nil || digestHex(archive) != plan.SourceDigest {
			return status, specTransferError("source-history-unavailable")
		}
	}
	redirectPath := sourceRedirectPath(sourceStore.Root, plan.Source.Slug)
	redirectRel, _ := filepath.Rel(sourceStore.Root, redirectPath)
	if err := transferPathClean(sourceStore.Root, filepath.ToSlash(redirectRel)); err != nil {
		return status, err
	}
	redirect := redirectBytes(plan.Source, plan.Destination, plan.OperationID)
	if digestHex(redirect) != plan.SourceRedirectDigest {
		return status, specTransferError("plan-digest-mismatch")
	}
	if err := writeTransferFile(redirectPath, redirect, ""); err != nil {
		return status, err
	}
	if err := recordTransferReceipt(sourceStore.Root, plan, plan.Source.Project, "source-retired", []string{plan.SourcePath, filepath.ToSlash(filepath.Join(".pose", "transfers", plan.OperationID, "source-spec.md")), filepath.ToSlash(filepath.Join(".pose", "transfers", "redirects", plan.Source.Slug+".json"))}); err != nil {
		return status, err
	}
	if failAfter != nil {
		if err := failAfter("source-retired"); err != nil {
			return status, err
		}
	}

	staged, err := os.ReadFile(destinationPath)
	if err != nil {
		return status, specTransferError("destination-spec-unavailable")
	}
	stagedDigest := digestHex(staged)
	if stagedDigest == plan.DestinationStageDigest {
		final := []byte(setSpecLifecycle(string(staged), plan.Destination.Slug, plan.FinalDestinationStatus))
		if digestHex(final) != plan.DestinationFinalDigest {
			return status, specTransferError("plan-digest-mismatch")
		}
		if err := writeTransferFile(destinationPath, final, plan.DestinationStageDigest); err != nil {
			return status, err
		}
	} else if stagedDigest != plan.DestinationFinalDigest {
		return status, specTransferError("compare-and-swap-conflict")
	}
	for _, id := range plan.AffectedProjects {
		if err := recordTransferReceipt(stores[id].Root, plan, id, "activated", filesByProject[id]); err != nil {
			return status, err
		}
	}
	if failAfter != nil {
		if err := failAfter("activated"); err != nil {
			return status, err
		}
	}
	return ReadSpecTransferStatus(destinationStore, plan.OperationID, plan.Destination.Project)
}

func recordTransferReceipt(root string, plan SpecTransferPlan, projectID, phase string, files []string) error {
	if err := ensureTransferJournalDir(root, plan.OperationID); err != nil {
		return err
	}
	path := filepath.Join(transferOperationDir(root, plan.OperationID), "journal", phase+".json")
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return specTransferError("path-escape")
	}
	if raw, err := os.ReadFile(path); err == nil {
		var existing specTransferReceipt
		if json.Unmarshal(raw, &existing) != nil || existing.SchemaVersion != SpecTransferSchemaVersion || existing.OperationID != plan.OperationID || existing.PlanDigest != plan.Digest || existing.ProjectID != projectID || existing.Phase != phase {
			return specTransferError("append-only-record-conflict")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	receipt := specTransferReceipt{SchemaVersion: SpecTransferSchemaVersion, OperationID: plan.OperationID, PlanDigest: plan.Digest, ProjectID: projectID, Phase: phase, Files: uniqueStrings(files), RecordedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	return writeTransferJSONExclusive(path, receipt)
}

func transferContains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

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
	"sort"
	"strings"
	"time"
)

const FederatedRoadmapPolicySchemaVersion = 1

const (
	maxFederatedPolicyBytes          = 256 << 10
	maxFederatedGovernanceInputBytes = 2 << 20
)

type FederatedProjectTrust struct {
	ContractDigest string `json:"contract_digest"`
	PolicyDigest   string `json:"policy_digest"`
	Revision       string `json:"revision"`
}

type FederatedRoadmapPolicy struct {
	SchemaVersion   int                              `json:"schema_version"`
	Enabled         bool                             `json:"enabled"`
	AdoptedAt       string                           `json:"adopted_at,omitempty"`
	TrustedProjects map[string]FederatedProjectTrust `json:"trusted_projects,omitempty"`
}

type FederatedDependencyEvidence struct {
	ID                 string `json:"id"`
	Check              string `json:"check"`
	EvidenceClass      string `json:"evidence_class"`
	SubjectObservation string `json:"subject_observation"`
}

type FederatedDependency struct {
	Ref                  string                        `json:"ref"`
	Relationship         string                        `json:"relationship"`
	Relationships        []string                      `json:"relationships,omitempty"`
	Identity             ArtifactRef                   `json:"identity"`
	ResolutionState      string                        `json:"resolution_state"`
	Status               string                        `json:"status,omitempty"`
	SourceRevision       string                        `json:"source_revision,omitempty"`
	SourceDigest         string                        `json:"source_digest,omitempty"`
	SourceContractDigest string                        `json:"source_contract_digest,omitempty"`
	SourcePolicyDigest   string                        `json:"source_policy_digest,omitempty"`
	ReviewBundleID       string                        `json:"review_bundle_id,omitempty"`
	ReviewBundleDigest   string                        `json:"review_bundle_digest,omitempty"`
	ImplementationDigest string                        `json:"implementation_digest,omitempty"`
	PlanDigest           string                        `json:"plan_digest,omitempty"`
	GoverningContracts   []string                      `json:"governing_contracts,omitempty"`
	Evidence             []FederatedDependencyEvidence `json:"evidence,omitempty"`
}

type FederatedRoadmapManifest struct {
	SchemaVersion       int                   `json:"schema_version"`
	Coordinator         ArtifactRef           `json:"coordinator"`
	CoordinatorRevision string                `json:"coordinator_revision,omitempty"`
	TrustPolicyDigest   string                `json:"trust_policy_digest,omitempty"`
	Dependencies        []FederatedDependency `json:"dependencies"`
	Ready               bool                  `json:"ready"`
	Blockers            []string              `json:"blockers"`
	Digest              string                `json:"digest"`
}

type FederatedRoadmapAcceptanceReport struct {
	Manifest FederatedRoadmapManifest `json:"manifest"`
	Ready    bool                     `json:"ready"`
	Blockers []string                 `json:"blockers"`
}

type federatedRoadmapEdge struct {
	project      string
	raw          string
	relationship string
}

type federatedQueuedRef struct {
	project      string
	parent       string
	raw          string
	relationship string
}

// FederatedProjectTrustFor computes the exact governance inputs a consumer
// must accept for one source project. It is read-only: callers still have to
// record the returned digests in their own policy.
func FederatedProjectTrustFor(root, projectID string) (FederatedProjectTrust, error) {
	if ValidateSlug(projectID) != nil {
		return FederatedProjectTrust{}, fmt.Errorf("pose: invalid federated project id")
	}
	revision := gitHeadAtRoot(root)
	if !isGitRevision(revision) {
		return FederatedProjectTrust{}, fmt.Errorf("pose: federated-project-revision-unavailable")
	}
	contractFiles := []string{".pose/schema-version", ".pose/indexes/validation-matrix.json"}
	policyFiles := []string{".pose/policy/artifacts.json", ".pose/policy/delivery.json", ".pose/policy/review.json"}
	contracts, err := committedFederatedInputs(root, contractFiles)
	if err != nil {
		return FederatedProjectTrust{}, err
	}
	policies, err := committedFederatedInputs(root, policyFiles)
	if err != nil {
		return FederatedProjectTrust{}, err
	}
	return FederatedProjectTrust{ContractDigest: canonicalFederatedDigest(contracts), PolicyDigest: canonicalFederatedDigest(policies), Revision: revision}, nil
}

func LoadFederatedRoadmapPolicy(root string) (FederatedRoadmapPolicy, string, error) {
	rel := ".pose/policy/federation.json"
	if err := ValidateArtifactPath(root, rel, false); err != nil {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: invalid-federation-policy-path")
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return FederatedRoadmapPolicy{}, "", nil
	}
	if err != nil {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: federation-policy-unavailable")
	}
	if !info.Mode().IsRegular() {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: federation-policy-unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: federation-policy-unavailable")
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maxFederatedPolicyBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: federation-policy-unavailable")
	}
	if len(raw) > maxFederatedPolicyBytes {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: federation-policy-too-large")
	}
	var policy FederatedRoadmapPolicy
	if rejectDuplicateJSONKeysAndControls(raw) != nil || strictJSONBytes(raw, &policy) != nil || policy.SchemaVersion != FederatedRoadmapPolicySchemaVersion {
		return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: unsupported-federation-contract")
	}
	if policy.Enabled {
		if _, err := time.Parse(time.DateOnly, policy.AdoptedAt); err != nil {
			return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: invalid-federation-adoption-date")
		}
	}
	for projectID, pin := range policy.TrustedProjects {
		if ValidateSlug(projectID) != nil || !isSHA256(pin.ContractDigest) || !isSHA256(pin.PolicyDigest) || !isGitRevision(pin.Revision) {
			return FederatedRoadmapPolicy{}, "", fmt.Errorf("pose: invalid-federated-project-trust")
		}
	}
	return policy, digestBytes(raw), nil
}

func strictJSONBytes(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON content")
		}
		return err
	}
	return nil
}

func isSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func isGitRevision(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func committedFederatedInputs(root string, paths []string) (map[string]string, error) {
	inputs := make(map[string]string, len(paths))
	for _, rel := range paths {
		if err := ValidateArtifactPath(root, rel, false); err != nil {
			return nil, fmt.Errorf("pose: invalid-federated-governance-input")
		}
		absolute := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(absolute)
		if err != nil {
			return nil, fmt.Errorf("pose: missing-federated-governance-input")
		}
		if !info.Mode().IsRegular() || info.Size() > maxFederatedGovernanceInputBytes {
			return nil, fmt.Errorf("pose: invalid-federated-governance-input")
		}
		if !artifactPathWithin(root, absolute) {
			return nil, fmt.Errorf("pose: invalid-federated-governance-input")
		}
		file, err := os.Open(absolute)
		if err != nil {
			return nil, fmt.Errorf("pose: missing-federated-governance-input")
		}
		working, readErr := io.ReadAll(io.LimitReader(file, maxFederatedGovernanceInputBytes+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || len(working) > maxFederatedGovernanceInputBytes {
			return nil, fmt.Errorf("pose: invalid-federated-governance-input")
		}
		top, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").Output()
		if err != nil || !sameProjectRoot(root, strings.TrimSpace(string(top))) {
			return nil, fmt.Errorf("pose: federated-project-revision-unavailable")
		}
		revision, err := exec.Command("git", "-C", root, "rev-parse", "--verify", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("pose: federated-project-revision-unavailable")
		}
		object := strings.TrimSpace(string(revision)) + ":" + filepath.ToSlash(rel)
		blobSize, err := exec.Command("git", "-C", root, "cat-file", "-s", object).Output()
		if err != nil {
			return nil, fmt.Errorf("pose: federated-project-revision-unavailable")
		}
		var committedSize int64
		if _, err := fmt.Sscan(strings.TrimSpace(string(blobSize)), &committedSize); err != nil || committedSize < 0 || committedSize > maxFederatedGovernanceInputBytes {
			return nil, fmt.Errorf("pose: invalid-federated-governance-input")
		}
		head, err := exec.Command("git", "-C", root, "show", object).Output()
		if err != nil || !bytes.Equal(working, head) {
			return nil, fmt.Errorf("pose: federated-governance-input-not-committed")
		}
		inputs[rel] = digestBytes(working)
	}
	return inputs, nil
}

func canonicalFederatedDigest(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s Store) FederatedRoadmapAcceptance(projectID, slug string, resolver ArtifactResolver) (FederatedRoadmapAcceptanceReport, error) {
	rm, err := s.GetRoadmap(slug)
	if err != nil {
		return FederatedRoadmapAcceptanceReport{}, err
	}
	if resolver.Roots == nil {
		return FederatedRoadmapAcceptanceReport{}, fmt.Errorf("pose: project roots are unavailable")
	}
	policy, policyDigest, err := LoadFederatedRoadmapPolicy(s.Root)
	if err != nil {
		return FederatedRoadmapAcceptanceReport{}, err
	}
	coordinator := ArtifactRef{Project: projectID, Kind: "roadmap", Slug: slug}
	revision := gitHeadAtRoot(s.Root)
	report := FederatedRoadmapAcceptanceReport{Manifest: FederatedRoadmapManifest{
		SchemaVersion: 1, Coordinator: coordinator, CoordinatorRevision: revision,
		TrustPolicyDigest: policyDigest, Dependencies: []FederatedDependency{},
	}, Blockers: []string{}}
	queue := []federatedQueuedRef{}
	for _, edge := range collectFederatedRoadmapEdges(projectID, rm) {
		queue = append(queue, federatedQueuedRef{project: edge.project, parent: edge.project, raw: edge.raw, relationship: edge.relationship})
	}
	visited := map[string]bool{}
	dependencyIndexes := map[string]int{}
	projectRoots := map[string]Store{projectID: s}
	snapshotRevisions := map[string]string{}
	if revision != "" {
		snapshotRevisions[projectID] = revision
	}
	for len(queue) > 0 {
		if len(visited) >= 256 {
			report.Blockers = append(report.Blockers, "resolution-limit")
			break
		}
		item := queue[0]
		queue = queue[1:]
		resolved := resolver.Resolve(item.project, item.raw)
		identity := resolved.Identity
		if identity.Project == "" {
			identity.Project = item.project
		}
		key := identity.String()
		if visited[key] {
			if index, ok := dependencyIndexes[key]; ok {
				entry := &report.Manifest.Dependencies[index]
				found := item.relationship == entry.Relationship
				for _, relationship := range entry.Relationships {
					found = found || relationship == item.relationship
				}
				if !found {
					entry.Relationships = append(entry.Relationships, item.relationship)
					sort.Strings(entry.Relationships)
					entry.Relationship = entry.Relationships[0]
				}
			}
			continue
		}
		visited[key] = true
		entry := FederatedDependency{Ref: key, Relationship: item.relationship, Relationships: []string{item.relationship}, Identity: identity, ResolutionState: resolved.State, Status: resolved.Status, SourceRevision: resolved.Revision, SourceDigest: resolved.Digest}
		dependencyIndexes[key] = len(report.Manifest.Dependencies)
		if !resolved.Resolved {
			report.Blockers = append(report.Blockers, key+":"+resolved.State)
			report.Manifest.Dependencies = append(report.Manifest.Dependencies, entry)
			continue
		}
		if selected, exists := snapshotRevisions[identity.Project]; exists && selected != "" && resolved.Revision != selected {
			report.Blockers = append(report.Blockers, key+":source-revision-changed-during-resolution")
		} else if _, exists := snapshotRevisions[identity.Project]; !exists && resolved.Revision != "" {
			snapshotRevisions[identity.Project] = resolved.Revision
		}
		graphValid := true
		if reason := resolver.ValidateGraph(item.project, item.raw); reason != "" {
			graphValid = false
			entry.ResolutionState = reason
			report.Blockers = append(report.Blockers, key+":"+reason)
		}
		statusDone := resolved.Status == "done"
		if !statusDone {
			report.Blockers = append(report.Blockers, key+":not-done")
		}
		source, sourceErr := resolver.Roots.StoreFor(identity.Project)
		if sourceErr != nil {
			entry.ResolutionState = projectResolutionState(sourceErr)
			report.Blockers = append(report.Blockers, key+":"+entry.ResolutionState)
			report.Manifest.Dependencies = append(report.Manifest.Dependencies, entry)
			continue
		}
		projectRoots[identity.Project] = source
		source.FederatedProjectID = identity.Project
		source.FederatedResolver = &resolver
		artifactValid := true
		if err := verifyFederatedArtifactAtRevision(source, identity, resolved); err != nil {
			artifactValid = false
			entry.ResolutionState = err.Error()
			report.Blockers = append(report.Blockers, key+":"+err.Error())
		}
		if identity.Project != projectID {
			// Do not recurse into source closeout when graph traversal has already
			// found a cycle or stale artifact. The blocker is conclusive, and
			// recursively asking each side of a cycle for its review would never
			// converge.
			gitlinkValid := true
			pin, pinErr := FederatedProjectTrustFor(source.Root, identity.Project)
			if pinErr != nil {
				entry.ResolutionState = pinErr.Error()
				report.Blockers = append(report.Blockers, key+":"+entry.ResolutionState)
			} else {
				entry.SourceContractDigest, entry.SourcePolicyDigest = pin.ContractDigest, pin.PolicyDigest
			}
			if !policy.Enabled {
				entry.ResolutionState = "consumer-trust-not-adopted"
				report.Blockers = append(report.Blockers, key+":"+entry.ResolutionState)
			} else {
				accepted, exists := policy.TrustedProjects[identity.Project]
				if pinErr != nil {
					// The pin error was recorded above; retain the stable state.
				} else if !exists {
					entry.ResolutionState = "consumer-trust-not-adopted"
					report.Blockers = append(report.Blockers, key+":"+entry.ResolutionState)
				} else if accepted != pin {
					entry.ResolutionState = "consumer-trust-stale"
					report.Blockers = append(report.Blockers, key+":"+entry.ResolutionState)
				} else {
					if identity.Project == item.parent {
						// This edge remains inside the selected source repository;
						// its local artifacts inherit that repository's pin.
					} else if parentStore, parentErr := resolver.Roots.StoreFor(item.parent); parentErr != nil || !federatedGitlinkMatches(parentStore, source, resolved.Revision) {
						gitlinkValid = false
						if parentErr != nil {
							entry.ResolutionState = projectResolutionState(parentErr)
						} else {
							entry.ResolutionState = "gitlink-revision-mismatch"
						}
						report.Blockers = append(report.Blockers, key+":"+entry.ResolutionState)
					}
				}
			}
			accepted, trusted := policy.TrustedProjects[identity.Project]
			proofInputsReady := graphValid && statusDone && artifactValid && pinErr == nil && policy.Enabled && trusted && accepted == pin && gitlinkValid
			if proofInputsReady {
				bundleID, bundleDigest, subjectDigest, planDigest, contracts, evidence, proofBlockers := federatedSourceProof(source, identity)
				entry.ReviewBundleID, entry.ReviewBundleDigest = bundleID, bundleDigest
				entry.ImplementationDigest, entry.PlanDigest = subjectDigest, planDigest
				entry.GoverningContracts, entry.Evidence = contracts, evidence
				report.Blockers = append(report.Blockers, proofBlockers...)
				if len(proofBlockers) > 0 {
					entry.ResolutionState = "source-evidence-incomplete"
				}
			}
		}
		report.Manifest.Dependencies = append(report.Manifest.Dependencies, entry)
		for _, dep := range resolved.Dependencies {
			queue = append(queue, federatedQueuedRef{project: dep.Project, parent: identity.Project, raw: dep.String(), relationship: "prerequisite"})
		}
	}
	for project, selected := range snapshotRevisions {
		if selected == "" {
			continue
		}
		root, ok := projectRoots[project]
		if ok && gitHeadAtRoot(root.Root) != selected {
			report.Blockers = append(report.Blockers, "source-snapshot-changed-during-resolution:"+project)
		}
	}
	report.Blockers = append(report.Blockers, federatedOwnershipConflicts(projectRoots, resolver)...)
	sort.Slice(report.Manifest.Dependencies, func(i, j int) bool {
		left, right := report.Manifest.Dependencies[i], report.Manifest.Dependencies[j]
		if left.Identity.String() != right.Identity.String() {
			return left.Identity.String() < right.Identity.String()
		}
		return left.Relationship < right.Relationship
	})
	report.Blockers = uniqueSorted(report.Blockers)
	report.Manifest.Ready = len(report.Blockers) == 0
	report.Manifest.Blockers = append([]string{}, report.Blockers...)
	report.Manifest.Digest = ""
	report.Manifest.Digest = canonicalFederatedDigest(report.Manifest)
	report.Ready = report.Manifest.Ready
	return report, nil
}

func collectFederatedRoadmapEdges(project string, rm *Roadmap) []federatedRoadmapEdge {
	edges := []federatedRoadmapEdge{}
	add := func(raw, kind, relationship string) {
		if raw == "" {
			return
		}
		if !strings.Contains(raw, ":") {
			if kind == "milestone" {
				raw = "milestone:" + rm.Slug + "/" + raw
			} else {
				raw = kind + ":" + raw
			}
		}
		edges = append(edges, federatedRoadmapEdge{project: project, raw: raw, relationship: relationship})
	}
	for _, ref := range rm.DependsOn {
		add(ref, "roadmap", "prerequisite")
	}
	for _, ref := range rm.Consumes {
		add(ref, "roadmap", "outcome-consumption")
	}
	for _, milestone := range rm.Milestones {
		for _, ref := range milestone.After {
			add(ref, "milestone", "prerequisite")
		}
		for _, ref := range milestone.Specs {
			add(ref, "spec", "ownership")
		}
		for _, ref := range milestone.Consumes {
			add(ref, "roadmap", "outcome-consumption")
		}
	}
	return edges
}

func verifyFederatedArtifactAtRevision(source Store, identity ArtifactRef, resolved ArtifactResolution) error {
	if resolved.Revision == "" {
		return fmt.Errorf("source-revision-unavailable")
	}
	var path string
	if identity.Kind == "spec" {
		sp, err := source.GetSpec(identity.Slug)
		if err != nil {
			return fmt.Errorf("unavailable-artifact")
		}
		path = sp.Path
	} else {
		rm, err := source.GetRoadmap(identity.Slug)
		if err != nil {
			return fmt.Errorf("unavailable-artifact")
		}
		path = rm.Path
	}
	rel, err := filepath.Rel(source.Root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("unavailable-artifact")
	}
	rel = filepath.ToSlash(rel)
	if err := ValidateArtifactPath(source.Root, rel, false); err != nil {
		return fmt.Errorf("unavailable-artifact")
	}
	committed, err := exec.Command("git", "-C", source.Root, "show", resolved.Revision+":"+rel).Output()
	if err != nil {
		return fmt.Errorf("source-revision-unavailable")
	}
	sum := sha256.Sum256(committed)
	if hex.EncodeToString(sum[:]) != resolved.Digest {
		return fmt.Errorf("source-artifact-revision-mismatch")
	}
	return nil
}

func federatedGitlinkMatches(parent, child Store, revision string) bool {
	top, err := exec.Command("git", "-C", child.Root, "rev-parse", "--show-toplevel").Output()
	if err != nil || !sameProjectRoot(child.Root, strings.TrimSpace(string(top))) {
		return true // sibling checkouts have no consumer gitlink to compare.
	}
	parentTop, err := exec.Command("git", "-C", parent.Root, "rev-parse", "--show-toplevel").Output()
	if err != nil || !sameProjectRoot(parent.Root, strings.TrimSpace(string(parentTop))) {
		return true
	}
	parentReal, err1 := filepath.EvalSymlinks(parent.Root)
	childReal, err2 := filepath.EvalSymlinks(child.Root)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(parentReal, childReal)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true // sibling project, explicitly selected in Roots.
	}
	listing, err := exec.Command("git", "-C", parent.Root, "ls-tree", "HEAD", "--", filepath.ToSlash(rel)).Output()
	if err != nil {
		return false
	}
	fields := strings.Fields(string(listing))
	if len(fields) < 3 || fields[0] != "160000" || fields[1] != "commit" {
		return false
	}
	return fields[2] == revision
}

func federatedSourceProof(source Store, identity ArtifactRef) (string, string, string, string, []string, []FederatedDependencyEvidence, []string) {
	scope := localFederatedScope(identity)
	state, err := source.GetCloseoutState(scope)
	if err != nil || !state.Terminal {
		return "", "", "", "", nil, nil, []string{scope + ":source-closeout-not-terminal"}
	}
	verification, err := source.VerifyReviewBundle(scope)
	if err != nil || !verification.Fresh || !verification.Approved || verification.Bundle == nil || verification.Attestation == nil {
		return "", "", "", "", nil, nil, []string{scope + ":source-review-not-approved"}
	}
	bundle := verification.Bundle
	if identity.Kind == "spec" {
		spec, err := source.GetSpec(identity.Slug)
		if err != nil {
			return "", "", "", "", nil, nil, []string{scope + ":unavailable-artifact"}
		}
		targets, _, err := ParseDeliveryTargets(*spec)
		if err != nil || len(targets) == 0 {
			return bundle.BundleID, bundle.BundleDigest, bundle.Payload.Subject.ImplementationDigest, bundle.Payload.Plan.PlanDigest, bundle.Payload.GoverningContracts, nil, []string{scope + ":source-delivery-target-missing"}
		}
		profiles, err := loadFederatedDeliveryProfiles(source.Root)
		if err != nil {
			return bundle.BundleID, bundle.BundleDigest, bundle.Payload.Subject.ImplementationDigest, bundle.Payload.Plan.PlanDigest, bundle.Payload.GoverningContracts, nil, []string{scope + ":source-delivery-profile-unavailable"}
		}
		blockers := []string{}
		for _, target := range targets {
			profile, ok := profiles[target.Profile]
			if !ok || profile.Kind != target.Kind {
				blockers = append(blockers, scope+":source-delivery-profile-unsupported")
				continue
			}
			seen := map[string]bool{}
			for _, evidence := range bundle.Payload.Evidence {
				if evidence.Outcome == "pass" && evidence.SubjectObservation == "observed" && moduleMatchesTarget(evidence.Module, target.Module) {
					seen[evidence.EvidenceClass] = true
				}
			}
			for _, required := range profile.RequiredEvidenceClasses {
				if !seen[required] {
					blockers = append(blockers, scope+":source-delivery-evidence-missing:"+target.Ref+":"+required)
				}
			}
			if len(profile.AnyEvidenceClasses) > 0 {
				found := false
				for _, class := range profile.AnyEvidenceClasses {
					found = found || seen[class]
				}
				if !found {
					blockers = append(blockers, scope+":source-delivery-evidence-missing:"+target.Ref+":one-of")
				}
			}
		}
		proof := federatedBundleEvidence(bundle)
		return bundle.BundleID, bundle.BundleDigest, bundle.Payload.Subject.ImplementationDigest, bundle.Payload.Plan.PlanDigest, bundle.Payload.GoverningContracts, proof, uniqueSorted(blockers)
	}
	return bundle.BundleID, bundle.BundleDigest, bundle.Payload.Subject.ImplementationDigest, bundle.Payload.Plan.PlanDigest, bundle.Payload.GoverningContracts, federatedBundleEvidence(bundle), nil
}

func localFederatedScope(identity ArtifactRef) string {
	if identity.Kind == "milestone" {
		return "milestone:" + identity.Slug + "/" + identity.Milestone
	}
	return identity.Kind + ":" + identity.Slug
}

func federatedBundleEvidence(bundle *ReviewBundle) []FederatedDependencyEvidence {
	evidence := []FederatedDependencyEvidence{}
	for _, item := range bundle.Payload.Evidence {
		if item.Outcome == "pass" && item.SubjectObservation == "observed" {
			evidence = append(evidence, FederatedDependencyEvidence{ID: item.ID, Check: item.Check, EvidenceClass: item.EvidenceClass, SubjectObservation: item.SubjectObservation})
		}
	}
	sort.Slice(evidence, func(i, j int) bool {
		if evidence[i].EvidenceClass != evidence[j].EvidenceClass {
			return evidence[i].EvidenceClass < evidence[j].EvidenceClass
		}
		return evidence[i].ID < evidence[j].ID
	})
	return evidence
}

func loadFederatedDeliveryProfiles(root string) (map[string]DeliveryProfile, error) {
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"))
	if err != nil {
		return nil, err
	}
	var matrix struct {
		DeliveryProfiles map[string]DeliveryProfile `json:"deliveryProfiles"`
	}
	if err := json.Unmarshal(raw, &matrix); err != nil || len(matrix.DeliveryProfiles) == 0 {
		return nil, fmt.Errorf("unsupported delivery profile contract")
	}
	return matrix.DeliveryProfiles, nil
}

func federatedOwnershipConflicts(projectRoots map[string]Store, resolver ArtifactResolver) []string {
	owners := map[string]string{}
	blockers := []string{}
	for project, store := range projectRoots {
		roadmaps, err := store.ListRoadmaps()
		if err != nil {
			blockers = append(blockers, "federated-ownership-unavailable:"+project)
			continue
		}
		for _, roadmap := range roadmaps {
			if roadmap.Status != "active" {
				continue
			}
			for _, milestone := range roadmap.Milestones {
				for _, raw := range milestone.Specs {
					if !strings.Contains(raw, ":") {
						raw = "spec:" + raw
					}
					resolved := resolver.Resolve(project, raw)
					if !resolved.Resolved || resolved.Identity.Kind != "spec" {
						continue
					}
					key := resolved.Identity.String()
					owner := (ArtifactRef{Project: project, Kind: "roadmap", Slug: roadmap.Slug}).String()
					if previous := owners[key]; previous != "" && previous != owner {
						blockers = append(blockers, "conflicting-roadmap-ownership:"+key)
					} else {
						owners[key] = owner
					}
				}
			}
		}
	}
	return uniqueSorted(blockers)
}

func gitHeadAtRoot(root string) string {
	if top, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").Output(); err != nil || !sameProjectRoot(root, strings.TrimSpace(string(top))) {
		return ""
	}
	if head, err := exec.Command("git", "-C", root, "rev-parse", "--verify", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(head))
	}
	return ""
}

func projectResolutionState(err error) string {
	var conflict ProjectBindingConflictError
	var ambiguous ProjectAmbiguousError
	if strings.Contains(err.Error(), "unknown project") {
		return "unknown-project"
	}
	if strings.Contains(err.Error(), "unavailable") {
		return "unavailable-project"
	}
	if errors.As(err, &conflict) || errors.As(err, &ambiguous) {
		return "conflicting-project-binding"
	}
	return "unavailable-project"
}

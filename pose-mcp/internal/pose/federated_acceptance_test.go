package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func federatedTestGit(t *testing.T, root string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func federatedTestWrite(t *testing.T, root, path, body string) {
	t.Helper()
	if strings.HasPrefix(path, ".pose/") {
		path = strings.TrimPrefix(path, ".pose/")
	}
	qualifiedFile(t, root, path, body)
}

func federatedTestProject(t *testing.T, root, projectID string) FederatedProjectTrust {
	t.Helper()
	federatedTestWrite(t, root, ".pose/schema-version", "1\n")
	federatedTestWrite(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"deliveryProfiles":{"release-governance":{"kind":"governance","requiredEvidenceClasses":["integration"]}},"stacks":{}}`)
	federatedTestWrite(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":false,"governed_roots":[]}`)
	federatedTestWrite(t, root, ".pose/policy/delivery.json", `{"schema_version":1,"enabled":false,"roots":[]}`)
	federatedTestWrite(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":false}`)
	federatedTestGit(t, root, "init", "-q")
	federatedTestGit(t, root, "config", "user.name", "POSE fixture")
	federatedTestGit(t, root, "config", "user.email", "pose@example.invalid")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "federated fixture")
	trust, err := FederatedProjectTrustFor(root, projectID)
	if err != nil {
		t.Fatal(err)
	}
	return trust
}

func federatedTestResolver(parentID, sourceID, parent, source string) ArtifactResolver {
	return ArtifactResolver{Roots: NewRoots(RootsConfig{
		DefaultRoot: parent, DefaultProjectID: parentID,
		Explicit: map[string]string{sourceID: source},
	})}
}

func federatedTestComposedRoadmapSource(t *testing.T, projectID string) (string, FederatedProjectTrust) {
	t.Helper()
	root, store := reviewBundleFixture(t)
	federatedTestWrite(t, root, ".pose/schema-version", "1\n")
	federatedTestWrite(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":false,"governed_roots":[]}`)
	federatedTestWrite(t, root, ".pose/policy/delivery.json", `{"schema_version":1,"enabled":false,"roots":[]}`)
	federatedTestWrite(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"deliveryProfiles":{"api-contract":{"kind":"contract","requiredEvidenceClasses":["integration","unit"]}},"stacks":{}}`)
	federatedTestWrite(t, root, ".pose/review-profiles/milestone-integration.json", `{"schema_version":1,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"integration","description":"The milestone is integrated.","kind":"judgment","evidence_classes":["integration"]}]}`)
	federatedTestWrite(t, root, ".pose/review-profiles/roadmap-outcome.json", `{"schema_version":1,"id":"roadmap-outcome","version":1,"scope":"roadmap","criteria":[{"id":"outcome","description":"The roadmap outcome is verified.","kind":"judgment","evidence_classes":["integration"]}]}`)
	policyPath := filepath.Join(root, ".pose", "policy", "review.json")
	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(policyBytes, &policy); err != nil {
		t.Fatal(err)
	}
	profiles := policy["profiles"].(map[string]any)
	profiles["milestone"], profiles["roadmap"] = "milestone-integration@1", "roadmap-outcome@1"
	policy["review_bundles"], policy["review_bundles_adopted_at"] = true, "2026-09-24"
	policyBytes, _ = json.Marshal(policy)
	federatedTestWrite(t, root, ".pose/policy/review.json", string(policyBytes))
	writeReviewFixture(t, root, ".pose/roadmaps/component.md", "---\nslug: component\nstatus: done\n---\n\n## Milestone: core\n- specs: backend\n\n## Outcome\nVerified source roadmap.\n")
	specPath := filepath.Join(root, ".pose", "specs", "backend", "spec.md")
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte(strings.Replace(string(specBytes), "status: in-progress", "status: done", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, root, "init", "-q")
	federatedTestGit(t, root, "config", "user.name", "POSE fixture")
	federatedTestGit(t, root, "config", "user.email", "pose@example.invalid")
	federatedTestGit(t, root, "config", "gc.auto", "0")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "source roadmap base")
	baseRaw, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSpace(string(baseRaw))
	if err := os.WriteFile(filepath.Join(root, "api", "server.go"), []byte("package api\n\nfunc Ready() bool { return false }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, root, "add", "api/server.go")
	federatedTestGit(t, root, "commit", "-q", "-m", "source implementation change")
	headRaw, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(string(headRaw))
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.ChangeSets) != 1 {
		t.Fatalf("fixture change sets = %d, want 1", len(graph.ChangeSets))
	}
	graph.ChangeSets[0].Base, graph.ChangeSets[0].ResolvedBase = base, base
	graph.ChangeSets[0].Head, graph.ChangeSets[0].ResolvedHead = head, head
	graph.ChangeSets[0].Commits = []string{head}
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].GitHead = head
		graph.ValidationResults[i].ProvenanceDigest = graph.ProvenanceDigest
	}
	graphBytes, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "delivery-integrity.json"), graphBytes, 0644); err != nil {
		t.Fatal(err)
	}
	sourceResolver := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: root, DefaultProjectID: projectID})}
	store.FederatedProjectID, store.FederatedResolver = projectID, &sourceResolver
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	for _, scope := range []string{"spec:backend", "milestone:component/core", "roadmap:component"} {
		bundle, err := store.SealReviewBundle(scope, now)
		if err != nil {
			t.Fatalf("seal %s bundle: %v; blockers=%v", scope, err, bundle.Blockers)
		}
		attestation, pending, err := store.prepareReviewAttestation(bundle.BundleID, "agent:fixture-review", now.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range pending {
			attestation.Criteria = append(attestation.Criteria, ReviewCriterion{ID: item.Criterion, Disposition: "passed", Evidence: "integration:validate-backend", Rationale: "the fixture reviewer inspected the sealed outcome"})
		}
		attestation.Decision = "approved"
		if _, err := store.RecordReviewAttestation(attestation, now.Add(time.Minute)); err != nil {
			t.Fatalf("record %s attestation: %v", scope, err)
		}
	}
	trust, err := FederatedProjectTrustFor(root, projectID)
	if err != nil {
		t.Fatal(err)
	}
	return root, trust
}

func TestFederatedAcceptanceNegativeStatusOnlySpecCannotClose(t *testing.T) {
	parent, source := t.TempDir(), t.TempDir()
	federatedTestWrite(t, source, ".pose/specs/engine.md", `---
slug: engine
status: done
delivers: governance:engine-delivery
---

# Spec: engine

### Delivery targets
- governance:engine-delivery module:engine profile:release-governance entrypoint:engine/main.go
`)
	federatedTestWrite(t, source, "engine/main.go", "package engine\n")
	trust := federatedTestProject(t, source, "source")
	federatedTestWrite(t, parent, ".pose/roadmaps/program.md", `---
slug: program
status: active
---

## Milestone: implementation
- specs: xref:source/spec:engine
`)
	policy := FederatedRoadmapPolicy{SchemaVersion: FederatedRoadmapPolicySchemaVersion, Enabled: true, AdoptedAt: "2026-09-24", TrustedProjects: map[string]FederatedProjectTrust{"source": trust}}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	federatedTestWrite(t, parent, ".pose/policy/federation.json", string(raw))
	resolver := federatedTestResolver("coordinator", "source", parent, source)
	report, err := (Store{Root: parent}).FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || !strings.Contains(strings.Join(report.Blockers, " "), "source-review-not-approved") {
		t.Fatalf("done status without a fresh approved source bundle was accepted: %+v", report)
	}
}

func TestFederatedLocalArtifactRevisionsSurviveEvidenceCommit(t *testing.T) {
	root := t.TempDir()
	federatedTestWrite(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: done\n---\n\n## Milestone: core\n- specs: engine\n")
	federatedTestWrite(t, root, ".pose/specs/engine.md", "---\nslug: engine\nstatus: done\n---\n")
	federatedTestProject(t, root, "coordinator")
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: root, DefaultProjectID: "coordinator"})}
	store := Store{Root: root}
	before, err := store.FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Manifest.Dependencies) == 0 {
		t.Fatal("fixture needs a local dependency")
	}
	federatedTestWrite(t, root, ".pose/results/evidence.json", "{}\n")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "record unrelated evidence")
	after, err := store.FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if before.Manifest.CoordinatorRevision != after.Manifest.CoordinatorRevision ||
		before.Manifest.Dependencies[0].SourceRevision != after.Manifest.Dependencies[0].SourceRevision ||
		before.Manifest.Digest != after.Manifest.Digest {
		t.Fatalf("unrelated evidence commit changed local federation identity: before=%+v after=%+v", before.Manifest, after.Manifest)
	}
}

func TestFederatedAcceptanceComposesReviewedSiblingRoadmapAndStalesOnSourceRevision(t *testing.T) {
	parent := t.TempDir()
	source, trust := federatedTestComposedRoadmapSource(t, "source")
	federatedTestWrite(t, parent, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:source/roadmap:component\n---\n")
	policy := FederatedRoadmapPolicy{SchemaVersion: FederatedRoadmapPolicySchemaVersion, Enabled: true, AdoptedAt: "2026-09-24", TrustedProjects: map[string]FederatedProjectTrust{"source": trust}}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	federatedTestWrite(t, parent, ".pose/policy/federation.json", string(raw))
	resolver := federatedTestResolver("coordinator", "source", parent, source)
	store := Store{Root: parent}
	report, err := store.FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready || len(report.Manifest.Dependencies) != 3 {
		t.Fatalf("valid reviewed source roadmap did not compose: %+v", report)
	}
	closeout, err := store.GetCloseoutStateWithFederatedAcceptance("roadmap:program", "coordinator", resolver)
	if err != nil || closeout.FederatedAcceptance == nil || !closeout.FederatedAcceptance.Ready {
		t.Fatalf("closeout did not expose the same current federated composition: state=%+v err=%v", closeout, err)
	}
	roadmapFound := false
	for _, dependency := range report.Manifest.Dependencies {
		if dependency.SourceRevision != trust.Revision || dependency.ReviewBundleID == "" || dependency.ReviewBundleDigest == "" {
			t.Fatalf("manifest omitted a pinned reviewed dependency: %+v", dependency)
		}
		roadmapFound = roadmapFound || dependency.Identity.Kind == "roadmap" && dependency.Identity.Slug == "component"
	}
	if !roadmapFound {
		t.Fatalf("manifest omitted the consumed source roadmap: %+v", report.Manifest.Dependencies)
	}
	federatedTestWrite(t, source, ".pose/roadmaps/component.md", "---\nslug: component\nstatus: done\n---\n\n## Outcome\nChanged after consumer adoption.\n")
	federatedTestGit(t, source, "add", "--all")
	federatedTestGit(t, source, "commit", "-q", "-m", "change selected source revision")
	report, err = store.FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || !strings.Contains(strings.Join(report.Blockers, " "), "consumer-trust-stale") {
		t.Fatalf("changed source revision did not stale consumer trust: %+v", report)
	}
}

func TestFederatedManifestIsSealedAndInvalidatesCoordinatorReview(t *testing.T) {
	source, trust := federatedTestComposedRoadmapSource(t, "source")
	root, store := reviewBundleFixture(t)
	policyPath := filepath.Join(root, ".pose", "policy", "review.json")
	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var reviewPolicy map[string]any
	if err := json.Unmarshal(policyBytes, &reviewPolicy); err != nil {
		t.Fatal(err)
	}
	profiles := reviewPolicy["profiles"].(map[string]any)
	profiles["milestone"], profiles["roadmap"] = "milestone-integration@1", "roadmap-outcome@1"
	reviewPolicy["review_bundles"], reviewPolicy["review_bundles_adopted_at"] = true, "2026-09-24"
	policyBytes, _ = json.Marshal(reviewPolicy)
	federatedTestWrite(t, root, ".pose/policy/review.json", string(policyBytes))
	federatedTestWrite(t, root, ".pose/review-profiles/milestone-integration.json", `{"schema_version":1,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"integration","description":"The milestone is integrated.","kind":"judgment","evidence_classes":["integration"]}]}`)
	federatedTestWrite(t, root, ".pose/review-profiles/roadmap-outcome.json", `{"schema_version":1,"id":"roadmap-outcome","version":1,"scope":"roadmap","criteria":[{"id":"outcome","description":"The roadmap outcome is verified.","kind":"judgment","evidence_classes":["integration"]}]}`)
	federatedTestWrite(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:source/roadmap:component\n---\n\n## Milestone: core\n- specs: backend\n")
	federationPolicy := FederatedRoadmapPolicy{SchemaVersion: FederatedRoadmapPolicySchemaVersion, Enabled: true, AdoptedAt: "2026-09-24", TrustedProjects: map[string]FederatedProjectTrust{"source": trust}}
	federationRaw, err := json.Marshal(federationPolicy)
	if err != nil {
		t.Fatal(err)
	}
	federatedTestWrite(t, root, ".pose/policy/federation.json", string(federationRaw))
	resolver := federatedTestResolver("coordinator", "source", root, source)
	sourceAuthorized := true
	resolver.Authorize = func(projectID string) bool {
		return projectID == "coordinator" || projectID == "source" && sourceAuthorized
	}
	store.FederatedProjectID, store.FederatedResolver = "coordinator", &resolver
	now := time.Date(2026, 9, 24, 13, 0, 0, 0, time.UTC)
	for _, scope := range []string{"spec:backend", "milestone:program/core", "roadmap:program"} {
		bundle, err := store.SealReviewBundle(scope, now)
		if err != nil {
			t.Fatalf("seal coordinator %s bundle: %v; blockers=%v", scope, err, bundle.Blockers)
		}
		attestation := approvedBundleAttestation(bundle, "agent:coordinator-review")
		attestation.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
		if _, err := store.RecordReviewAttestation(attestation, now.Add(time.Minute)); err != nil {
			t.Fatalf("attest coordinator %s bundle: %v", scope, err)
		}
	}
	first, err := store.VerifyReviewBundle("roadmap:program")
	if err != nil || !first.Fresh || !first.Approved || first.Bundle == nil || first.Bundle.Payload.FederatedManifest == nil {
		t.Fatalf("coordinator bundle did not seal current federated snapshot: verification=%+v err=%v", first, err)
	}
	firstManifestDigest := first.Bundle.Payload.FederatedManifest.Digest
	sourceAuthorized = false
	revoked, err := store.VerifyReviewBundle("roadmap:program")
	if err != nil || revoked.Fresh || revoked.Approved {
		t.Fatalf("authorization revocation did not invalidate coordinator review: verification=%+v err=%v", revoked, err)
	}
	sourceAuthorized = true
	restored, err := store.VerifyReviewBundle("roadmap:program")
	if err != nil || !restored.Fresh || !restored.Approved {
		t.Fatalf("restored authorization did not restore the unchanged sealed snapshot: verification=%+v err=%v", restored, err)
	}
	federatedTestWrite(t, source, ".pose/roadmaps/component.md", "---\nslug: component\nstatus: done\n---\n\n## Milestone: core\n- specs: backend\n\n## Outcome\nThe selected source outcome changed.\n")
	federatedTestGit(t, source, "add", "--all")
	federatedTestGit(t, source, "commit", "-q", "-m", "change reviewed source outcome")
	second, err := store.PrepareReviewBundle("roadmap:program")
	if err != nil {
		t.Fatal(err)
	}
	stale, err := store.VerifyReviewBundle("roadmap:program")
	if err != nil {
		t.Fatal(err)
	}
	if stale.Fresh || stale.Approved || second.Payload.FederatedManifest == nil || second.Payload.FederatedManifest.Digest == firstManifestDigest {
		t.Fatalf("source change did not stale coordinator review: %+v", stale)
	}
}

func TestLoadFederatedPolicyRejectsTrailingContent(t *testing.T) {
	root := t.TempDir()
	federatedTestWrite(t, root, ".pose/policy/federation.json", `{"schema_version":1} {"enabled":true}`)
	if _, _, err := LoadFederatedRoadmapPolicy(root); err == nil || !strings.Contains(err.Error(), "unsupported-federation-contract") {
		t.Fatalf("trailing JSON content was accepted: %v", err)
	}
}

func TestLoadFederatedPolicyBoundsInputSize(t *testing.T) {
	root := t.TempDir()
	federatedTestWrite(t, root, ".pose/policy/federation.json", strings.Repeat(" ", maxFederatedPolicyBytes+1))
	if _, _, err := LoadFederatedRoadmapPolicy(root); err == nil || !strings.Contains(err.Error(), "federation-policy-too-large") {
		t.Fatalf("oversized federation policy was accepted: %v", err)
	}
}

func TestFederatedRoadmapRejectsMixedProjectCycles(t *testing.T) {
	parent, source := t.TempDir(), t.TempDir()
	federatedTestWrite(t, parent, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:source/roadmap:component\n---\n")
	federatedTestWrite(t, source, ".pose/roadmaps/component.md", "---\nslug: component\nstatus: active\nconsumes: xref:coordinator/roadmap:program\n---\n")
	resolver := federatedTestResolver("coordinator", "source", parent, source)
	report, err := (Store{Root: parent}).FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(report.Blockers, " "), "dependency-cycle") {
		t.Fatalf("mixed project roadmap cycle was not rejected: %+v", report)
	}
}

func TestFederatedRoadmapRejectsSelfReferenceThroughProjectAlias(t *testing.T) {
	root := t.TempDir()
	federatedTestWrite(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:alias/roadmap:program\n---\n")
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{
		DefaultRoot: root, DefaultProjectID: "coordinator",
		Explicit: map[string]string{"alias": root},
	})}
	report, err := (Store{Root: root}).FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(report.Blockers, " "), "dependency-cycle") {
		t.Fatalf("self-reference through a project-root alias was not rejected: %+v", report)
	}
}

func TestFederatedRoadmapRejectsDuplicateActiveOwnership(t *testing.T) {
	parent, source := t.TempDir(), t.TempDir()
	federatedTestWrite(t, parent, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n## Milestone: current\n- specs: xref:source/spec:engine\n")
	federatedTestWrite(t, source, ".pose/specs/engine.md", "---\nslug: engine\nstatus: done\n---\n")
	federatedTestWrite(t, source, ".pose/roadmaps/component.md", "---\nslug: component\nstatus: active\n---\n## Milestone: current\n- specs: engine\n")
	resolver := federatedTestResolver("coordinator", "source", parent, source)
	report, err := (Store{Root: parent}).FederatedRoadmapAcceptance("coordinator", "program", resolver)
	if err != nil {
		t.Fatal(err)
	}
	wanted := "conflicting-roadmap-ownership:xref:source/spec:engine"
	if !strings.Contains(strings.Join(report.Blockers, " "), wanted) {
		t.Fatalf("duplicate active owners were not rejected: %+v", report)
	}
}

func TestFederatedGitlinkPinChecksNestedRevisionAndAllowsSiblingProjects(t *testing.T) {
	parent, child := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, "vendor"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(parent, "vendor", "dep")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, child, "init", "-q")
	federatedTestGit(t, child, "config", "user.name", "POSE fixture")
	federatedTestGit(t, child, "config", "user.email", "pose@example.invalid")
	if err := os.WriteFile(filepath.Join(child, "README.md"), []byte("source\n"), 0644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, child, "add", "--all")
	federatedTestGit(t, child, "commit", "-q", "-m", "child revision")
	childHeadRaw, err := exec.Command("git", "-C", child, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	childHead := strings.TrimSpace(string(childHeadRaw))
	federatedTestGit(t, nested, "init", "-q")
	federatedTestGit(t, nested, "config", "user.name", "POSE fixture")
	federatedTestGit(t, nested, "config", "user.email", "pose@example.invalid")
	if err := os.WriteFile(filepath.Join(nested, "README.md"), []byte("source\n"), 0644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, nested, "add", "--all")
	federatedTestGit(t, nested, "commit", "-q", "-m", "nested child revision")
	nestedHeadRaw, err := exec.Command("git", "-C", nested, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	nestedHead := strings.TrimSpace(string(nestedHeadRaw))
	federatedTestGit(t, parent, "init", "-q")
	federatedTestGit(t, parent, "config", "user.name", "POSE fixture")
	federatedTestGit(t, parent, "config", "user.email", "pose@example.invalid")
	if err := os.WriteFile(filepath.Join(parent, "README.md"), []byte("parent\n"), 0644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, parent, "add", "README.md")
	federatedTestGit(t, parent, "commit", "-q", "-m", "parent base")
	federatedTestGit(t, parent, "update-index", "--add", "--cacheinfo", "160000,"+nestedHead+",vendor/dep")
	federatedTestGit(t, parent, "commit", "-q", "-m", "pin child gitlink")
	if !federatedGitlinkMatches(Store{Root: parent}, Store{Root: nested}, nestedHead) {
		t.Fatal("nested submodule matching its selected revision was rejected")
	}
	wrongHead := strings.Repeat("0", len(nestedHead))
	if federatedGitlinkMatches(Store{Root: parent}, Store{Root: nested}, wrongHead) {
		t.Fatal("nested submodule accepted a revision different from its gitlink")
	}
	if !federatedGitlinkMatches(Store{Root: parent}, Store{Root: child}, childHead) {
		t.Fatal("explicitly selected sibling project was incorrectly required to have a gitlink")
	}
}

func TestFederatedProjectTrustRejectsMissingContractInputs(t *testing.T) {
	root := t.TempDir()
	federatedTestWrite(t, root, ".pose/schema-version", "1\n")
	federatedTestGit(t, root, "init", "-q")
	federatedTestGit(t, root, "config", "user.name", "POSE fixture")
	federatedTestGit(t, root, "config", "user.email", "pose@example.invalid")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "incomplete trust fixture")
	if _, err := FederatedProjectTrustFor(root, "source"); err == nil || !strings.Contains(err.Error(), "missing-federated-governance-input") {
		t.Fatalf("incomplete contract inputs accepted: %v", err)
	}
}

func TestFederatedProjectTrustRejectsUncommittedContractInputs(t *testing.T) {
	root := t.TempDir()
	federatedTestProject(t, root, "source")
	federatedTestWrite(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true}`)
	if _, err := FederatedProjectTrustFor(root, "source"); err == nil || !strings.Contains(err.Error(), "federated-governance-input-not-committed") {
		t.Fatalf("uncommitted governance input was accepted: %v", err)
	}
}

func TestLegacyReviewBundlePayloadOmitsFederatedManifest(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Payload.FederatedManifest != nil {
		t.Fatalf("legacy non-roadmap bundle gained a federated manifest: %+v", bundle.Payload.FederatedManifest)
	}
	raw, err := json.Marshal(bundle.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"federated_manifest"`) {
		t.Fatalf("nil federated manifest changed the historic payload shape: %s", raw)
	}
}

func TestFederatedProjectTrustPathsStayWithinRepository(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "validation-matrix.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	federatedTestWrite(t, root, ".pose/schema-version", "1\n")
	if err := os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".pose", "indexes", "validation-matrix.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := FederatedProjectTrustFor(root, "source"); err == nil {
		t.Fatal("external governance input was accepted")
	}
}

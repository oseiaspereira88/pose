package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func initSpecTransferProject(t *testing.T, root, projectID string, specs map[string]string) {
	t.Helper()
	for _, dir := range []string{".pose/policy", ".pose/specs", ".pose/roadmaps", ".pose/review-bundles"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"enabled":true,"adopted_at":"2026-09-24","profiles":{"spec":"spec-closeout@1"}}` + "\n"
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	for slug, body := range specs {
		path := filepath.Join(root, ".pose/specs", "2026-09-24-"+slug+".md")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runTransferGit(t, root, "init", "-q")
	runTransferGit(t, root, "config", "user.email", "pose-transfer@example.invalid")
	runTransferGit(t, root, "config", "user.name", "POSE transfer test")
	runTransferGit(t, root, "add", ".pose")
	runTransferGit(t, root, "commit", "-q", "-m", "fixture "+projectID)
}

func runTransferGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	} else {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func specTransferFixtureBody(slug, status, depends string, requirementIDs ...string) string {
	var b strings.Builder
	b.WriteString("---\nslug: " + slug + "\nstatus: " + status + "\ncreated_at: 2026-09-24\n")
	if depends != "" {
		b.WriteString("depends_on: " + depends + "\n")
	}
	b.WriteString("---\n# Spec: " + slug + "\n\n## 2 Requirements\n\n")
	for _, id := range requirementIDs {
		b.WriteString("- " + id + ": Requirement " + id + "\n")
	}
	b.WriteString("\n## 3 Technical Plan\n\nKeep the transfer fixture small.\n")
	return b.String()
}

func newSpecTransferResolver(sourceRoot, sourceID string, projects map[string]string) ArtifactResolver {
	roots := NewRoots(RootsConfig{DefaultRoot: sourceRoot, DefaultProjectID: sourceID, Explicit: projects})
	allowed := map[string]bool{sourceID: true}
	for id := range projects {
		allowed[id] = true
	}
	return ArtifactResolver{Roots: roots, Authorize: func(id string) bool { return allowed[id] }}
}

func equivalentTransferMap(ids ...string) []SpecTransferRequirementMapping {
	out := make([]SpecTransferRequirementMapping, 0, len(ids))
	for _, id := range ids {
		out = append(out, SpecTransferRequirementMapping{SourceRequirement: id, DestinationRequirement: id, Disposition: "equivalent"})
	}
	return out
}

func setupSiblingTransferFixture(t *testing.T) (ArtifactResolver, string, string, string, string) {
	t.Helper()
	base := t.TempDir()
	sourceRoot, destinationRoot, consumerRoot := filepath.Join(base, "source"), filepath.Join(base, "destination"), filepath.Join(base, "consumer")
	initSpecTransferProject(t, sourceRoot, "proj.source", map[string]string{"source-task": specTransferFixtureBody("source-task", "in-progress", "", "R1", "R2")})
	initSpecTransferProject(t, destinationRoot, "proj.destination", map[string]string{"executor-task": specTransferFixtureBody("executor-task", "in-progress", "", "R1", "R2")})
	consumerRef := "xref:proj.source/spec:source-task"
	initSpecTransferProject(t, consumerRoot, "proj.consumer", map[string]string{"consumer-task": specTransferFixtureBody("consumer-task", "in-progress", consumerRef, "R1")})
	resolver := newSpecTransferResolver(sourceRoot, "proj.source", map[string]string{"proj.destination": destinationRoot, "proj.consumer": consumerRoot})
	return resolver, sourceRoot, destinationRoot, consumerRoot, base
}

func transferRequest(existingDestination bool) SpecTransferRequest {
	destinationSlug := "new-task"
	if existingDestination {
		destinationSlug = "executor-task"
	}
	return SpecTransferRequest{
		Source:      ArtifactRef{Project: "proj.source", Kind: "spec", Slug: "source-task"},
		Destination: ArtifactRef{Project: "proj.destination", Kind: "spec", Slug: destinationSlug},
		Mappings:    equivalentTransferMap("R1", "R2"),
	}
}

func TestSpecTransferPreviewIsDeterministicAndReadOnly(t *testing.T) {
	resolver, sourceRoot, destinationRoot, consumerRoot, _ := setupSiblingTransferFixture(t)
	request := transferRequest(true)
	beforeSource, err := os.ReadFile(filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md"))
	if err != nil {
		t.Fatal(err)
	}
	beforeDestination, err := os.ReadFile(filepath.Join(destinationRoot, ".pose/specs/2026-09-24-executor-task.md"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	second, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	if first.OperationID != second.OperationID || first.Digest != second.Digest {
		t.Fatalf("preview is not deterministic: %+v / %+v", first, second)
	}
	if len(first.ImpactedReferences) != 1 || first.ImpactedReferences[0].ProjectID != "proj.consumer" {
		t.Fatalf("impacts = %+v", first.ImpactedReferences)
	}
	if len(first.AffectedProjects) != 3 {
		t.Fatalf("affected projects = %v", first.AffectedProjects)
	}
	afterSource, _ := os.ReadFile(filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md"))
	afterDestination, _ := os.ReadFile(filepath.Join(destinationRoot, ".pose/specs/2026-09-24-executor-task.md"))
	if string(beforeSource) != string(afterSource) || string(beforeDestination) != string(afterDestination) {
		t.Fatal("preview changed an artifact")
	}
	for _, root := range []string{sourceRoot, destinationRoot, consumerRoot} {
		if _, err := os.Stat(filepath.Join(root, ".pose/transfers")); !os.IsNotExist(err) {
			t.Fatalf("preview created transfer state in %s", root)
		}
	}
}

func TestSpecTransferPreviewSupportsSiblingAndNestedSubmoduleRoots(t *testing.T) {
	resolver, _, _, _, base := setupSiblingTransferFixture(t)
	if _, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24"); err != nil {
		t.Fatalf("sibling layout: %v", err)
	}

	parentRoot := filepath.Join(base, "nested-parent")
	childSource := filepath.Join(base, "nested-child-source")
	initSpecTransferProject(t, parentRoot, "proj.parent", map[string]string{"parent-task": specTransferFixtureBody("parent-task", "in-progress", "", "R1")})
	initSpecTransferProject(t, childSource, "proj.nested", map[string]string{"nested-task": specTransferFixtureBody("nested-task", "in-progress", "", "R1")})
	cmd := exec.Command("git", "-c", "protocol.file.allow=always", "-C", parentRoot, "submodule", "add", "-q", childSource, "nested-child")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("add nested fixture submodule: %v: %s", err, out)
	}
	runTransferGit(t, parentRoot, "add", ".gitmodules", "nested-child")
	runTransferGit(t, parentRoot, "commit", "-q", "-m", "add nested project")
	nestedRoot := filepath.Join(parentRoot, "nested-child")
	nestedResolver := newSpecTransferResolver(parentRoot, "proj.parent", map[string]string{"proj.nested": nestedRoot})
	request := SpecTransferRequest{
		Source:      ArtifactRef{Project: "proj.parent", Kind: "spec", Slug: "parent-task"},
		Destination: ArtifactRef{Project: "proj.nested", Kind: "spec", Slug: "new-task"},
		Mappings:    equivalentTransferMap("R1"),
	}
	if _, err := PreviewSpecTransfer(nestedResolver, request, "2026-09-24"); err != nil {
		t.Fatalf("nested Git submodule layout: %v", err)
	}
}

func TestSpecTransferPreservesFolderSpecAmendments(t *testing.T) {
	base := t.TempDir()
	sourceRoot, destinationRoot := filepath.Join(base, "source"), filepath.Join(base, "destination")
	initSpecTransferProject(t, sourceRoot, "proj.source", map[string]string{"source-task": specTransferFixtureBody("source-task", "in-progress", "", "R1")})
	initSpecTransferProject(t, destinationRoot, "proj.destination", map[string]string{"executor-task": specTransferFixtureBody("executor-task", "in-progress", "", "R1")})
	oldPath := filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md")
	if err := os.Remove(oldPath); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	body := specTransferFixtureBody("source-task", "in-progress", "", "R1")
	if err := os.WriteFile(filepath.Join(folder, "spec.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	amendment := []byte("{\"id\":\"amend-1\",\"note\":\"preserve me\"}\n")
	if err := os.WriteFile(filepath.Join(folder, "amendments.jsonl"), amendment, 0o644); err != nil {
		t.Fatal(err)
	}
	runTransferGit(t, sourceRoot, "add", "-A")
	runTransferGit(t, sourceRoot, "commit", "-q", "-m", "move source fixture to directory layout")
	resolver := newSpecTransferResolver(sourceRoot, "proj.source", map[string]string{"proj.destination": destinationRoot})
	request := SpecTransferRequest{
		Source:      ArtifactRef{Project: "proj.source", Kind: "spec", Slug: "source-task"},
		Destination: ArtifactRef{Project: "proj.destination", Kind: "spec", Slug: "executor-task"},
		Mappings:    equivalentTransferMap("R1"),
	}
	plan, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourcePath != ".pose/specs/2026-09-24-source-task/spec.md" {
		t.Fatalf("folder source path = %q", plan.SourcePath)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err != nil {
		t.Fatal(err)
	}
	gotAmendment, err := os.ReadFile(filepath.Join(folder, "amendments.jsonl"))
	if err != nil || string(gotAmendment) != string(amendment) {
		t.Fatalf("source amendment changed: %s, err=%v", gotAmendment, err)
	}
	redirected, err := (Store{Root: sourceRoot}).GetSpec("source-task")
	if err != nil || redirected.Status != "superseded" {
		t.Fatalf("source folder redirect = %+v, err=%v", redirected, err)
	}
}

func TestSpecTransferRewritesRoadmapMembershipAndKeepsOtherObligations(t *testing.T) {
	resolver, _, _, consumerRoot, _ := setupSiblingTransferFixture(t)
	path := filepath.Join(consumerRoot, ".pose/roadmaps/consumer-roadmap.md")
	roadmap := "---\nslug: consumer-roadmap\nstatus: active\nconsumes: xref:proj.source/spec:source-task\n---\n## Milestone: build\n- specs: xref:proj.source/spec:source-task\n- consumes: xref:proj.source/spec:source-task, xref:proj.consumer/spec:consumer-task\n"
	if err := os.WriteFile(path, []byte(roadmap), 0o644); err != nil {
		t.Fatal(err)
	}
	runTransferGit(t, consumerRoot, "add", ".pose/roadmaps/consumer-roadmap.md")
	runTransferGit(t, consumerRoot, "commit", "-q", "-m", "add roadmap membership fixture")
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	foundRoadmap := false
	for _, impact := range plan.ImpactedReferences {
		if impact.Owner.Kind == "roadmap" && impact.Owner.Slug == "consumer-roadmap" {
			foundRoadmap = true
		}
	}
	if !foundRoadmap {
		t.Fatalf("roadmap membership was missing from inventory: %+v", plan.ImpactedReferences)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(after)
	if strings.Contains(text, "xref:proj.source/spec:source-task") || strings.Count(text, "xref:proj.destination/spec:executor-task") != 3 || !strings.Contains(text, "xref:proj.consumer/spec:consumer-task") {
		t.Fatalf("roadmap references were not selectively rewritten: %s", text)
	}
}

func TestSpecTransferRequirementMapRequiresExplicitCoverage(t *testing.T) {
	resolver, _, _, _, _ := setupSiblingTransferFixture(t)
	request := transferRequest(true)
	request.Mappings = nil
	if _, err := PreviewSpecTransfer(resolver, request, "2026-09-24"); err == nil || !strings.Contains(err.Error(), "requirement-map-incomplete") {
		t.Fatalf("missing map error = %v", err)
	}
	request.Mappings = []SpecTransferRequirementMapping{{SourceRequirement: "R99", DestinationRequirement: "R1", Disposition: "equivalent"}}
	if _, err := PreviewSpecTransfer(resolver, request, "2026-09-24"); err == nil || !strings.Contains(err.Error(), "requirement-map-source-mismatch") {
		t.Fatalf("unknown requirement error = %v", err)
	}
}

func TestSpecTransferApplyPreservesHistoryAndRewritesApprovedReferences(t *testing.T) {
	resolver, sourceRoot, destinationRoot, consumerRoot, _ := setupSiblingTransferFixture(t)
	request := transferRequest(true)
	reviewBundlePath := filepath.Join(destinationRoot, ".pose/review-bundles/rvb-original.json")
	reviewBundle := []byte(`{"bundle_id":"rvb-original","scope":"spec:executor-task","subject":"original-executor-subject","subject_digest":"sha256:original"}` + "\n")
	if err := os.WriteFile(reviewBundlePath, reviewBundle, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	status, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != "activated" || status.ProjectID != "proj.destination" {
		t.Fatalf("status = %+v", status)
	}
	sourceRaw, err := os.ReadFile(filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sourceRaw), "- R1:") || !strings.Contains(string(sourceRaw), "Canonical task: `xref:proj.destination/spec:executor-task`") {
		t.Fatalf("source is not a requirement-free redirect: %s", sourceRaw)
	}
	archive, err := os.ReadFile(filepath.Join(sourceRoot, ".pose/transfers", plan.OperationID, "source-spec.md"))
	if err != nil || !strings.Contains(string(archive), "- R1:") {
		t.Fatalf("source history archive missing: %v", err)
	}
	destination, err := (Store{Root: destinationRoot}).GetSpec("executor-task")
	if err != nil || destination.Status != "in-progress" {
		t.Fatalf("destination status = %+v, err=%v", destination, err)
	}
	consumer, err := os.ReadFile(filepath.Join(consumerRoot, ".pose/specs/2026-09-24-consumer-task.md"))
	if err != nil || !strings.Contains(string(consumer), "xref:proj.destination/spec:executor-task") || strings.Contains(string(consumer), "xref:proj.source/spec:source-task") {
		t.Fatalf("approved dependency rewrite = %s, err=%v", consumer, err)
	}
	resolver.Authorize = func(id string) bool { return true }
	resolved := resolver.Resolve("proj.source", "xref:proj.source/spec:source-task")
	if !resolved.Resolved || !resolved.Redirected || resolved.CanonicalIdentity == nil || *resolved.CanonicalIdentity != plan.Destination {
		t.Fatalf("source redirect resolution = %+v", resolved)
	}
	bundleAfter, err := os.ReadFile(reviewBundlePath)
	if err != nil || string(bundleAfter) != string(reviewBundle) {
		t.Fatalf("existing executor review subject changed: %s, err=%v", bundleAfter, err)
	}
	if _, err := ReadSpecTransferStatus(Store{Root: sourceRoot}, plan.OperationID, "proj.unrelated"); err == nil || !strings.Contains(err.Error(), "operation-not-found") {
		t.Fatalf("unrelated project status error = %v", err)
	}
	if err := os.Remove(filepath.Join(destinationRoot, plan.DestinationPath)); err != nil {
		t.Fatal(err)
	}
	missingTarget := resolver.Resolve("proj.source", plan.Source.String())
	if missingTarget.Resolved || missingTarget.State != "redirect-target-unknown-spec" {
		t.Fatalf("redirect with missing target was not rejected: %+v", missingTarget)
	}
}

func TestSpecTransferApplyPreservesSourceFrontmatterForEmptyDestination(t *testing.T) {
	resolver, sourceRoot, destinationRoot, _, _ := setupSiblingTransferFixture(t)
	sourcePath := filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md")
	source := `---
slug: source-task
status: in-progress
created_at: 2026-09-24
priority: 2
custom_policy: preserve-this-field
depends_on: spec:prerequisite-task
delivers: governance:source-delivery
components: pose-mcp
---
# Spec: source-task

## 2 Requirements

- R1: Preserve the original task contract.
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	runTransferGit(t, sourceRoot, "add", ".pose/specs/2026-09-24-source-task.md")
	runTransferGit(t, sourceRoot, "commit", "-q", "-m", "add transfer metadata fixture")
	request := transferRequest(false)
	request.Mappings = equivalentTransferMap("R1")
	plan, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err != nil {
		t.Fatal(err)
	}
	destinationPath := filepath.Join(destinationRoot, filepath.FromSlash(plan.DestinationPath))
	transferred, err := os.ReadFile(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"slug: new-task", "status: in-progress", "priority: 2", "custom_policy: preserve-this-field",
		"depends_on: xref:proj.source/spec:prerequisite-task", "delivers: governance:source-delivery", "components: pose-mcp",
		"- R1: Preserve the original task contract.",
	} {
		if !strings.Contains(string(transferred), expected) {
			t.Errorf("transferred spec lost %q:\n%s", expected, transferred)
		}
	}
}

func TestSpecTransferResolverRejectsRedirectCycles(t *testing.T) {
	resolver, _, destinationRoot, _, _ := setupSiblingTransferFixture(t)
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err != nil {
		t.Fatal(err)
	}
	cycle := plan
	cycle.Source, cycle.Destination = plan.Destination, plan.Source
	cycle.Request.Source, cycle.Request.Destination = cycle.Source, cycle.Destination
	cycle.SourcePath, cycle.DestinationPath = plan.DestinationPath, plan.SourcePath
	cycle.SourceDigest, cycle.DestinationDigest = plan.DestinationDigest, plan.SourceDigest
	cycle.SourceRevision, cycle.DestinationRevision = plan.DestinationRevision, plan.SourceRevision
	cycle.ImpactedReferences = nil
	cycle.AffectedProjects = []string{"proj.destination", "proj.source"}
	cycle.ProjectRevisions = []SpecTransferProjectRevision{{ProjectID: "proj.destination", Revision: plan.DestinationRevision}, {ProjectID: "proj.source", Revision: plan.SourceRevision}}
	cycle.SourceStatus, cycle.DestinationStatus = "in-progress", "in-progress"
	cycle.FinalDestinationStatus = "in-progress"
	cycle.SourceStubDigest = digestHex([]byte(renderTransferRedirectStub(cycle.Source, cycle.Destination)))
	cycle.SourceRedirectDigest = ""
	cycle.OperationID = transferOperationID(cycle)
	cycle.SourceRedirectDigest = digestHex(redirectBytes(cycle.Source, cycle.Destination, cycle.OperationID))
	cycle.Digest = specTransferPlanDigest(cycle)
	if err := validateTransferPlan(cycle); err != nil {
		t.Fatalf("test cycle plan invalid: %v", err)
	}
	if err := ensureTransferOperationDir(destinationRoot, cycle.OperationID); err != nil {
		t.Fatal(err)
	}
	if err := writeTransferJSONExclusive(transferPlanFile(destinationRoot, cycle.OperationID), cycle); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"prepared", "activated"} {
		if err := recordTransferReceipt(destinationRoot, cycle, cycle.Source.Project, phase, nil); err != nil {
			t.Fatal(err)
		}
	}
	redirectPath := sourceRedirectPath(destinationRoot, cycle.Source.Slug)
	if err := writeTransferFile(redirectPath, redirectBytes(cycle.Source, cycle.Destination, cycle.OperationID), ""); err != nil {
		t.Fatal(err)
	}
	got := resolver.Resolve("proj.source", plan.Source.String())
	if got.Resolved || !strings.HasSuffix(got.State, "redirect-cycle") {
		t.Fatalf("redirect cycle resolution = %+v", got)
	}
}

func TestSpecTransferPreviewBlocksUnscannedProjects(t *testing.T) {
	resolver, _, _, _, _ := setupSiblingTransferFixture(t)
	resolver.Authorize = func(id string) bool { return id == "proj.source" || id == "proj.destination" }
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	if !transferContains(plan.Blockers, "unresolved-project:proj.consumer") {
		t.Fatalf("inaccessible project was not reported as unresolved: %v", plan.Blockers)
	}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, map[string]bool{"proj.source": true, "proj.destination": true}, nil); err == nil || !strings.Contains(err.Error(), "unresolved-impact-blockers") {
		t.Fatalf("apply with an unscanned project error = %v", err)
	}
}

func TestSpecTransferInterruptedOperationsResumeWithoutDuplicateAuthority(t *testing.T) {
	for _, interruptedAt := range []string{"planned", "prepared", "source-retired"} {
		t.Run(interruptedAt, func(t *testing.T) {
			resolver, sourceRoot, destinationRoot, _, _ := setupSiblingTransferFixture(t)
			request := transferRequest(true)
			plan, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
			if err != nil {
				t.Fatal(err)
			}
			permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
			_, err = ApplySpecTransfer(resolver, plan, plan.Digest, permissions, func(phase string) error {
				if phase == interruptedAt {
					return specTransferError("injected-interruption")
				}
				return nil
			})
			if err == nil || !strings.Contains(err.Error(), "injected-interruption") {
				t.Fatalf("interrupt error = %v", err)
			}
			if interruptedAt == "source-retired" {
				resolved := resolver.Resolve("proj.source", "xref:proj.source/spec:source-task")
				if resolved.Resolved || resolved.State != "transfer-in-progress" {
					t.Fatalf("partial transfer was not blocked: %+v", resolved)
				}
			}
			status, err := ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil)
			if err != nil {
				t.Fatal(err)
			}
			if status.Phase != "activated" {
				t.Fatalf("resumed status = %+v", status)
			}
			source, err := (Store{Root: sourceRoot}).GetSpec("source-task")
			if err != nil || source.Status != "superseded" {
				t.Fatalf("source status = %+v, err=%v", source, err)
			}
			destination, err := (Store{Root: destinationRoot}).GetSpec("executor-task")
			if err != nil || destination.Status != "in-progress" {
				t.Fatalf("destination status = %+v, err=%v", destination, err)
			}
			if _, err := ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil); err != nil {
				t.Fatalf("idempotent second resume: %v", err)
			}
		})
	}
}

func TestSpecTransferResumeBlocksChangedReferenceInventory(t *testing.T) {
	resolver, _, destinationRoot, consumerRoot, _ := setupSiblingTransferFixture(t)
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	_, err = ApplySpecTransfer(resolver, plan, plan.Digest, permissions, func(phase string) error {
		if phase == "source-retired" {
			return specTransferError("injected-interruption")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "injected-interruption") {
		t.Fatalf("interrupted apply error = %v", err)
	}
	interruptedDestination, err := (Store{Root: destinationRoot}).GetSpec("executor-task")
	if err != nil || interruptedDestination.Status != "blocked" {
		t.Fatalf("destination before resume = %+v, err=%v", interruptedDestination, err)
	}
	lateReference := specTransferFixtureBody("late-reference", "in-progress", "xref:proj.source/spec:source-task", "R1")
	latePath := filepath.Join(consumerRoot, ".pose/specs/2026-09-24-late-reference.md")
	if err := os.WriteFile(latePath, []byte(lateReference), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil)
	if err == nil || !strings.Contains(err.Error(), "source-reference-inventory-changed") {
		t.Fatalf("resume with a newly added source reference = %v", err)
	}
	resolved := resolver.Resolve("proj.source", plan.Source.String())
	if resolved.Resolved || resolved.State != "transfer-in-progress" {
		t.Fatalf("partial transfer was not kept blocked: %+v", resolved)
	}
	destination, err := (Store{Root: destinationRoot}).GetSpec("executor-task")
	if err != nil || destination.Status != "blocked" {
		t.Fatalf("destination status = %+v, err=%v", destination, err)
	}
}

func TestSpecTransferConcurrentApplySerializesWriters(t *testing.T) {
	resolver, sourceRoot, destinationRoot, _, _ := setupSiblingTransferFixture(t)
	request := transferRequest(true)
	plan, err := PreviewSpecTransfer(resolver, request, "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		} else if !strings.Contains(err.Error(), "transfer-in-progress") && !strings.Contains(err.Error(), "concurrent-transfer") {
			t.Errorf("concurrent apply error = %v", err)
		}
	}
	if successes == 0 {
		t.Fatal("neither concurrent apply completed")
	}
	source, err := (Store{Root: sourceRoot}).GetSpec("source-task")
	if err != nil || source.Status != "superseded" {
		t.Fatalf("source = %+v, err=%v", source, err)
	}
	destination, err := (Store{Root: destinationRoot}).GetSpec("executor-task")
	if err != nil || destination.Status != "in-progress" {
		t.Fatalf("destination = %+v, err=%v", destination, err)
	}
}

func TestSpecTransferNegativeGatesRejectStaleAndUnauthorizedApply(t *testing.T) {
	resolver, sourceRoot, _, _, _ := setupSiblingTransferFixture(t)
	plan, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24")
	if err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	if _, err := ApplySpecTransfer(resolver, plan, strings.Repeat("0", 64), permissions, nil); err == nil || !strings.Contains(err.Error(), "plan-digest-mismatch") {
		t.Fatalf("bad digest error = %v", err)
	}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, map[string]bool{"proj.source": true}, nil); err == nil || !strings.Contains(err.Error(), "write-authorization-required") {
		t.Fatalf("unauthorized apply error = %v", err)
	}
	path := filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(original, []byte("\n<!-- changed after preview -->\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}, nil); err == nil || !strings.Contains(err.Error(), "stale-plan") {
		t.Fatalf("stale plan error = %v", err)
	}
}

func TestSpecTransferNegativeGatesRejectUnsupportedCapabilityAndUnsafePath(t *testing.T) {
	resolver, sourceRoot, destinationRoot, _, _ := setupSiblingTransferFixture(t)
	policyPath := filepath.Join(destinationRoot, ".pose/policy/review.json")
	if err := os.WriteFile(policyPath, []byte(`{"schema_version":3,"qualified_artifact_refs_version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24"); err == nil || !strings.Contains(err.Error(), "capability-not-adopted") {
		t.Fatalf("unsupported capability error = %v", err)
	}
	if err := os.Remove(policyPath); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte(specTransferFixtureBody("source-task", "in-progress", "", "R1", "R2")), 0o644); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(sourceRoot, ".pose/specs/2026-09-24-source-task.md")
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, original); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewSpecTransfer(resolver, transferRequest(true), "2026-09-24"); err == nil {
		t.Fatal("symlink escape was accepted")
	}
}

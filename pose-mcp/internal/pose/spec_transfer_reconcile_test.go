package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTerminalReconcileFixture holds a draft coordinator, a done executor
// whose project does not require review (so its closeout is terminal), a
// consumer that depends on the coordinator, and an open follow-up spec.
func setupTerminalReconcileFixture(t *testing.T, executorStatus string) (ArtifactResolver, string, string, string) {
	t.Helper()
	base := t.TempDir()
	sourceRoot, destinationRoot, consumerRoot := filepath.Join(base, "source"), filepath.Join(base, "destination"), filepath.Join(base, "consumer")
	initSpecTransferProject(t, sourceRoot, "proj.source", map[string]string{"shared-task": specTransferFixtureBody("shared-task", "draft", "", "R1", "R2", "R3", "R4")})
	initSpecTransferProject(t, destinationRoot, "proj.destination", map[string]string{"shared-task": specTransferFixtureBody("shared-task", executorStatus, "", "R1", "R2", "R9")})
	disableTransferFixtureReview(t, destinationRoot)
	initSpecTransferProject(t, consumerRoot, "proj.consumer", map[string]string{
		"consumer-task": specTransferFixtureBody("consumer-task", "in-progress", "xref:proj.source/spec:shared-task", "R1"),
		"followup-task": specTransferFixtureBody("followup-task", "draft", "", "R1"),
		"closed-task":   specTransferFixtureBody("closed-task", "done", "", "R1"),
	})
	resolver := newSpecTransferResolver(sourceRoot, "proj.source", map[string]string{"proj.destination": destinationRoot, "proj.consumer": consumerRoot})
	return resolver, sourceRoot, destinationRoot, consumerRoot
}

func disableTransferFixtureReview(t *testing.T, root string) {
	t.Helper()
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"enabled":false,"adopted_at":"2026-09-24","profiles":{"spec":"spec-closeout@1"}}` + "\n"
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	runTransferGit(t, root, "commit", "-qam", "executor project closes without review")
}

func terminalReconcileRequest() SpecTransferRequest {
	return SpecTransferRequest{
		Source:      ArtifactRef{Project: "proj.source", Kind: "spec", Slug: "shared-task"},
		Destination: ArtifactRef{Project: "proj.destination", Kind: "spec", Slug: "shared-task"},
		Mode:        SpecTransferModeReconcileTerminal,
		Mappings: []SpecTransferRequirementMapping{
			{SourceRequirement: "R1", DestinationRequirement: "R1", Disposition: "equivalent"},
			{SourceRequirement: "R2", DestinationRequirement: "R2", Disposition: "reformulated"},
			{SourceRequirement: "R2", DestinationRequirement: "R9", Disposition: "reformulated"},
			{SourceRequirement: "R3", DestinationRef: "xref:proj.consumer/spec:followup-task#R1", Disposition: "pending", Rationale: "The executor never delivered it."},
			{SourceRequirement: "R4", Disposition: "withdrawn", Rationale: "Superseded by the executor's design."},
		},
	}
}

func TestSpecTransferReconcileTerminalRetiresCoordinatorWithoutTouchingExecutor(t *testing.T) {
	resolver, sourceRoot, destinationRoot, consumerRoot := setupTerminalReconcileFixture(t, "done")
	executorPath := filepath.Join(destinationRoot, ".pose/specs/2026-09-24-shared-task.md")
	executorBefore, err := os.ReadFile(executorPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PreviewSpecTransfer(resolver, terminalReconcileRequest(), "2026-09-26")
	if err != nil {
		t.Fatal(err)
	}
	if plan.SchemaVersion != SpecReconcileSchemaVersion || plan.FinalDestinationStatus != "done" || plan.RequiresFreshEvidence || plan.DestinationStageDigest != plan.DestinationDigest || plan.DestinationFinalDigest != plan.DestinationDigest {
		t.Fatalf("terminal plan would reopen or rewrite the executor: %+v", plan)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	status, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil)
	if err != nil || status.Phase != "activated" {
		t.Fatalf("apply = %+v, err=%v", status, err)
	}
	executorAfter, err := os.ReadFile(executorPath)
	if err != nil || string(executorAfter) != string(executorBefore) {
		t.Fatalf("executor changed: %s err=%v", executorAfter, err)
	}
	sourceRaw, err := os.ReadFile(filepath.Join(sourceRoot, ".pose/specs/2026-09-24-shared-task.md"))
	if err != nil || strings.Contains(string(sourceRaw), "- R1:") || !strings.Contains(string(sourceRaw), "xref:proj.destination/spec:shared-task") {
		t.Fatalf("coordinator was not retired to a redirect stub: %s err=%v", sourceRaw, err)
	}
	var redirect specTransferRedirect
	raw, err := os.ReadFile(sourceRedirectPath(sourceRoot, "shared-task"))
	if err != nil || json.Unmarshal(raw, &redirect) != nil {
		t.Fatalf("redirect unreadable: %v", err)
	}
	if redirect.SchemaVersion != SpecReconcileSchemaVersion || redirect.Mode != SpecTransferModeReconcileTerminal || len(redirect.OpenObligations) != 1 || redirect.OpenObligations[0].DestinationRef != "xref:proj.consumer/spec:followup-task#R1" {
		t.Fatalf("redirect lost the owed requirement: %+v", redirect)
	}
	consumer, err := os.ReadFile(filepath.Join(consumerRoot, ".pose/specs/2026-09-24-consumer-task.md"))
	if err != nil || !strings.Contains(string(consumer), "xref:proj.destination/spec:shared-task") {
		t.Fatalf("consumer dependency not rewritten: %s err=%v", consumer, err)
	}
	resolved := resolver.Resolve("proj.source", "xref:proj.source/spec:shared-task")
	if !resolved.Resolved || !resolved.Redirected || resolved.CanonicalIdentity == nil || *resolved.CanonicalIdentity != plan.Destination {
		t.Fatalf("coordinator does not resolve to the executor: %+v", resolved)
	}
	again, err := ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil)
	if err != nil || again.Phase != "activated" {
		t.Fatalf("resume after activation is not idempotent: %+v err=%v", again, err)
	}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err == nil {
		t.Fatal("a second apply was accepted after the coordinator was retired")
	}
	if after, _ := os.ReadFile(executorPath); string(after) != string(executorBefore) {
		t.Fatal("resume rewrote the executor")
	}
}

func TestSpecTransferNegativeReconcileTerminalGates(t *testing.T) {
	expect := func(name, code string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), code) {
			t.Fatalf("%s: expected %s, got %v", name, code, err)
		}
	}
	resolver, _, _, _ := setupTerminalReconcileFixture(t, "in-progress")
	_, err := PreviewSpecTransfer(resolver, terminalReconcileRequest(), "2026-09-26")
	expect("open executor", "destination-spec-not-terminal", err)

	resolver, _, destinationRoot, _ := setupTerminalReconcileFixture(t, "done")
	policy := `{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1,"enabled":true,"review_bundles":true,"review_bundles_adopted_at":"2026-09-01","adopted_at":"2026-09-01","profiles":{"spec":"spec-closeout@1"}}` + "\n"
	if err := os.WriteFile(filepath.Join(destinationRoot, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	runTransferGit(t, destinationRoot, "commit", "-qam", "executor project requires review")
	_, err = PreviewSpecTransfer(resolver, terminalReconcileRequest(), "2026-09-26")
	expect("executor done without approved review", "destination-closeout-not-terminal", err)

	resolver, _, _, _ = setupTerminalReconcileFixture(t, "done")
	mutate := func(change func(*SpecTransferRequest)) error {
		request := terminalReconcileRequest()
		change(&request)
		_, err := PreviewSpecTransfer(resolver, request, "2026-09-26")
		return err
	}
	expect("uncovered requirement", "requirement-map-incomplete", mutate(func(r *SpecTransferRequest) { r.Mappings = r.Mappings[:4] }))
	expect("owed requirement in a closed spec", "requirement-map-obligation-target-closed", mutate(func(r *SpecTransferRequest) {
		r.Mappings[3].DestinationRef = "xref:proj.consumer/spec:closed-task"
	}))
	expect("owed requirement without a target", "requirement-map-incomplete", mutate(func(r *SpecTransferRequest) { r.Mappings[3].DestinationRef = "" }))
	expect("withdrawn without rationale", "requirement-map-incomplete", mutate(func(r *SpecTransferRequest) { r.Mappings[4].Rationale = "" }))
	expect("unknown executor requirement", "requirement-map-destination-mismatch", mutate(func(r *SpecTransferRequest) { r.Mappings[0].DestinationRequirement = "R7" }))
	expect("missing target requirement", "requirement-map-target-unavailable", mutate(func(r *SpecTransferRequest) {
		r.Mappings[3].DestinationRef = "xref:proj.consumer/spec:followup-task#R5"
	}))
	expect("reconcile fields on a transfer", "requirement-map-fields-need-reconcile-terminal", mutate(func(r *SpecTransferRequest) {
		r.Mode = ""
	}))
	expect("unknown mode", "unsupported-transfer-mode", mutate(func(r *SpecTransferRequest) { r.Mode = "reconcile-anything" }))
}

func TestSpecTransferNegativeReconcileTerminalChangedExecutorAndOldSchema(t *testing.T) {
	resolver, sourceRoot, destinationRoot, _ := setupTerminalReconcileFixture(t, "done")
	plan, err := PreviewSpecTransfer(resolver, terminalReconcileRequest(), "2026-09-26")
	if err != nil {
		t.Fatal(err)
	}
	downgraded := plan
	downgraded.SchemaVersion = SpecTransferSchemaVersion
	if err := validateTransferPlan(downgraded); err == nil {
		t.Fatal("a terminal plan validated under the transfer schema")
	}
	executorPath := filepath.Join(destinationRoot, ".pose/specs/2026-09-24-shared-task.md")
	raw, err := os.ReadFile(executorPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executorPath, append(raw, []byte("\nEdited after review.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
	if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, nil); err == nil {
		t.Fatal("apply retired the coordinator onto an executor that changed after preview")
	}
	sourceRaw, err := os.ReadFile(filepath.Join(sourceRoot, ".pose/specs/2026-09-24-shared-task.md"))
	if err != nil || !strings.Contains(string(sourceRaw), "- R1:") {
		t.Fatalf("coordinator was retired despite the refused apply: %s", sourceRaw)
	}
}

func TestSpecTransferReconcileTerminalResumesAfterInterruption(t *testing.T) {
	for _, phase := range []string{"planned", "prepared", "source-retired"} {
		t.Run(phase, func(t *testing.T) {
			resolver, sourceRoot, destinationRoot, _ := setupTerminalReconcileFixture(t, "done")
			executorPath := filepath.Join(destinationRoot, ".pose/specs/2026-09-24-shared-task.md")
			executorBefore, _ := os.ReadFile(executorPath)
			plan, err := PreviewSpecTransfer(resolver, terminalReconcileRequest(), "2026-09-26")
			if err != nil {
				t.Fatal(err)
			}
			permissions := map[string]bool{"proj.source": true, "proj.destination": true, "proj.consumer": true}
			interrupt := func(current string) error {
				if current == phase {
					return os.ErrDeadlineExceeded
				}
				return nil
			}
			if _, err := ApplySpecTransfer(resolver, plan, plan.Digest, permissions, interrupt); err == nil {
				t.Fatal("interrupted apply reported success")
			}
			status, err := ResumeSpecTransfer(resolver, "proj.destination", plan.OperationID, permissions, nil)
			if err != nil || status.Phase != "activated" {
				t.Fatalf("resume = %+v err=%v", status, err)
			}
			if after, _ := os.ReadFile(executorPath); string(after) != string(executorBefore) {
				t.Fatal("interrupted reconciliation rewrote the executor")
			}
			resolved := resolver.Resolve("proj.source", "xref:proj.source/spec:shared-task")
			if !resolved.Redirected || resolved.CanonicalIdentity == nil || *resolved.CanonicalIdentity != plan.Destination {
				t.Fatalf("two authorities after resume: %+v", resolved)
			}
			if raw, _ := os.ReadFile(filepath.Join(sourceRoot, ".pose/specs/2026-09-24-shared-task.md")); strings.Contains(string(raw), "- R1:") {
				t.Fatal("coordinator stayed executable after resume")
			}
		})
	}
}

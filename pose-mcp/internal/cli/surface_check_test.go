package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func surfaceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeArtifactTestFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["web"]}`)
	writeArtifactTestFile(t, root, ".pose/policy/delivery.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","results_path":".pose/results/current.json","roots":[{"path":"web","kind":"surface","profile":"web-ui","entrypoint":"cmd/app/main.go"}],"severities":{"surface-without-reachability":"error","undeclared-delivery":"error","roadmap-criterion":"error"}}`)
	writeArtifactTestFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"deliveryProfiles":{"web-ui":{"kind":"surface","requiredEvidenceClasses":["reachability"],"anyEvidenceClasses":["integration","e2e"]}},"stacks":{},"moduleOverrides":{}}`)
	writeArtifactTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-03\ndelivers: surface:dashboard\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: dashboard\n\n## 3. Technical Plan\n\n### Artifacts\n- modified: web/view.go\n\n### Delivery targets\n- surface:dashboard module:web profile:web-ui entrypoint:cmd/app/main.go\n\n## 4. Tasks\nwork\n")
	writeArtifactTestFile(t, root, "web/view.go", "package web\n")
	writeArtifactTestFile(t, root, "cmd/app/main.go", "package main\n")
	writeArtifactTestFile(t, root, ".pose/reports/history/alpha.jsonl", `{"change_set":{"id":"cs-alpha","spec":"alpha","selector":"fixture","resolved_base":"a","resolved_head":"b","paths":[{"action":"modified","path":"web/view.go"}],"diff_digest":"sha256:fixture"}}`+"\n")
	return root
}

func writeSurfaceResult(t *testing.T, root, provenance string, checks []checkResult) {
	t.Helper()
	run := validationRunResult{SchemaVersion: validationResultSchema, GeneratedAt: "2026-08-03T00:00:00Z", Outcome: "pass", ProvenanceDigest: provenance, Checks: checks}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".pose/results/current.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func surfaceProvenance(t *testing.T, root string) string {
	t.Helper()
	specs, claims, sets, tracked, policy, err := collectArtifactGraphInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	return posemodel.BuildDeliveryIntegrity(specs, claims, sets, tracked, policy).ProvenanceDigest
}

func TestSurfaceCheckRejectsGreenUnitEvidenceForUnreachableSurface(t *testing.T) {
	root := surfaceFixture(t)
	digest := surfaceProvenance(t, root)
	writeSurfaceResult(t, root, digest, []checkResult{{ID: "web/go/unit", Module: "web", Name: "unit", EvidenceClass: "unit", Severity: "required", Outcome: "pass"}})
	var out, errOut bytes.Buffer
	if code := cmdSurfaceCheck(root, []string{"--spec", "alpha", "--strict"}, &out, &errOut); code != 1 {
		t.Fatalf("unreachable surface passed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "surface-without-reachability") {
		t.Fatalf("missing composed finding: %s", out.String())
	}
}

func TestSurfaceCheckAcceptsCurrentReachabilityAndIntegration(t *testing.T) {
	root := surfaceFixture(t)
	digest := surfaceProvenance(t, root)
	writeSurfaceResult(t, root, digest, []checkResult{
		{ID: "web/go/reach", Module: "web", Name: "web-reachability", EvidenceClass: "reachability", Severity: "required", Outcome: "pass"},
		{ID: "web/go/integration", Module: "web", Name: "web-integration", EvidenceClass: "integration", Severity: "required", Outcome: "pass"},
	})
	var out, errOut bytes.Buffer
	if code := cmdSurfaceCheck(root, []string{"--spec", "alpha", "--strict", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("current surface failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	var graph posemodel.DeliveryIntegrityGraph
	if err := json.Unmarshal(out.Bytes(), &graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.Paths["surface:dashboard"]) < 4 {
		t.Fatalf("missing explainable path: %+v", graph.Paths)
	}
}

func TestSurfaceCheckRejectsStaleEvidenceAndEscapingResultPath(t *testing.T) {
	root := surfaceFixture(t)
	writeSurfaceResult(t, root, "sha256:stale", []checkResult{{ID: "web/go/reach", Module: "web", Name: "reach", EvidenceClass: "reachability", Severity: "required", Outcome: "pass"}})
	var out, errOut bytes.Buffer
	if code := cmdSurfaceCheck(root, []string{"--spec", "alpha", "--strict"}, &out, &errOut); code != 1 {
		t.Fatalf("stale evidence passed: %d", code)
	}
	out.Reset()
	errOut.Reset()
	if code := cmdSurfaceCheck(root, []string{"--results", "../../secret", "--strict"}, &out, &errOut); code != 2 {
		t.Fatalf("escaping result path accepted: %d", code)
	}
}

func TestValidationMatrixRejectsUnknownEvidenceClass(t *testing.T) {
	_, err := parseValidationMatrix([]byte(`{"stacks":{"go":{"checks":[{"name":"x","program":"go","evidenceClass":"invented"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	matrix, _ := parseValidationMatrix([]byte(`{"stacks":{"go":{"checks":[{"name":"x","program":"go","evidenceClass":"invented"}]}}}`))
	if err := validateDeliveryMatrixContract(matrix); err == nil {
		t.Fatal("unknown evidence class accepted")
	}
}

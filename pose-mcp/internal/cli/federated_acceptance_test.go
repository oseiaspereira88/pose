package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

const federatedAcceptanceTarget = "governance:federated-roadmap-acceptance"

func federatedAcceptanceNegativeFixture(t *testing.T, criteria, results string) string {
	t.Helper()
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"), `{
  "defaults":{"mode":"strict"},
  "deliveryProfiles":{"release-governance":{"kind":"governance","requiredEvidenceClasses":["integration"]}},
  "stacks":{}
}`)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "delivery.json"), `{
  "schema_version":1,
  "enabled":true,
  "results_path":".pose/results/delivery-validation.json",
  "roots":[],
  "severities":{"roadmap-criterion":"error"}
}`)
	mustWrite(t, filepath.Join(root, ".pose", "specs", "federated-implementation", "spec.md"), `---
slug: federated-implementation
status: in-progress
created_at: 2026-09-24
delivers: governance:federated-roadmap-acceptance
---

# Spec: federated implementation

### Delivery targets
- governance:federated-roadmap-acceptance module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go
`)
	mustWrite(t, filepath.Join(root, "pose-mcp", "cmd", "pose", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, ".pose", "roadmaps", "pose-multirepo-foundation.md"), `---
slug: pose-multirepo-foundation
status: active
---

## Cut criteria
`+criteria+`
`)
	if results != "" {
		mustWrite(t, filepath.Join(root, ".pose", "results", "delivery-validation.json"), results)
	}
	return root
}

func TestFederatedAcceptanceNegativeTargetWithoutCurrentEvidence(t *testing.T) {
	root := federatedAcceptanceNegativeFixture(t,
		"- C1: "+federatedAcceptanceTarget+" check:federated-roadmap-acceptance-integration", "")
	out, errOut, code := runCLI(t, root, "roadmap-check", "pose-multirepo-foundation", "--json")
	if code == 0 {
		t.Fatalf("target criterion passed without current integration evidence: out=%s err=%s", out, errOut)
	}
	var report struct {
		Criteria []struct {
			ID      string   `json:"id"`
			Passed  bool     `json:"passed"`
			Reasons []string `json:"reasons"`
		} `json:"criteria"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("roadmap-check returned invalid JSON: %v\n%s", err, out)
	}
	if len(report.Criteria) != 1 || report.Criteria[0].ID != "C1" || report.Criteria[0].Passed {
		t.Fatalf("C1 did not fail closed: %+v", report.Criteria)
	}
	if !strings.Contains(strings.Join(report.Criteria[0].Reasons, " "), "federated-roadmap-acceptance-integration") {
		t.Fatalf("C1 blocker does not name its missing producer: %+v", report.Criteria[0].Reasons)
	}
}

func TestFederatedAcceptanceNegativeFailingCheckDoesNotSatisfyCut(t *testing.T) {
	results := `{"schema_version":1,"generated_at":"2026-09-24T00:00:00Z","outcome":"fail","checks":[{"id":"pose-mcp/federated-roadmap-negative-gates","module":"pose-mcp","name":"federated-roadmap-negative-gates","severity":"required","evidence_class":"integration","outcome":"fail"}]}`
	root := federatedAcceptanceNegativeFixture(t, "- C2: check:federated-roadmap-negative-gates", results)
	out, errOut, code := runCLI(t, root, "roadmap-check", "pose-multirepo-foundation", "--json")
	if code == 0 {
		t.Fatalf("failed negative-gates producer satisfied C2: out=%s err=%s", out, errOut)
	}
	var report struct {
		Criteria []struct {
			ID      string   `json:"id"`
			Passed  bool     `json:"passed"`
			Reasons []string `json:"reasons"`
		} `json:"criteria"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("roadmap-check returned invalid JSON: %v\n%s", err, out)
	}
	if len(report.Criteria) != 1 || report.Criteria[0].ID != "C2" || report.Criteria[0].Passed {
		t.Fatalf("C2 did not fail closed: %+v", report.Criteria)
	}
	if !strings.Contains(strings.Join(report.Criteria[0].Reasons, " "), "federated-roadmap-negative-gates") {
		t.Fatalf("C2 blocker does not name its failing producer: %+v", report.Criteria[0].Reasons)
	}
}

func TestFederatedRoadmapCompositionIsWrittenToIndex(t *testing.T) {
	root := portfolioFixtureProject(t, t.TempDir(), "index-project")
	if err := os.MkdirAll(filepath.Join(root, ".pose", "roadmaps"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "roadmaps", "program.md"), []byte("---\nslug: program\nstatus: active\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := cmdIndex(root, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("pose index failed: %s%s", stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "roadmaps.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Roadmaps map[string]struct {
			Federated posepkg.FederatedRoadmapAcceptanceReport `json:"federated_acceptance"`
		} `json:"roadmaps"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	if report := index.Roadmaps["program"].Federated; !report.Ready || report.Manifest.Coordinator.Kind != "roadmap" {
		t.Fatalf("roadmap index omitted federated readiness: %+v", report)
	}
}

func TestFederatedRoadmapCheckBlocksSelfReference(t *testing.T) {
	root := t.TempDir()
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "coordinator")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	mustWrite(t, filepath.Join(root, ".pose", "roadmaps", "program.md"), `---
slug: program
status: active
consumes: xref:coordinator/roadmap:program
---
`)
	out, errOut, code := runCLI(t, root, "roadmap-check", "program", "--json")
	if code == 0 {
		t.Fatalf("roadmap-check accepted a self-referential consumed outcome: out=%s err=%s", out, errOut)
	}
	var report struct {
		Terminal bool     `json:"terminal"`
		Blockers []string `json:"blockers"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("roadmap-check returned invalid JSON: %v\n%s", err, out)
	}
	if report.Terminal || !strings.Contains(strings.Join(report.Blockers, " "), "dependency-cycle") {
		t.Fatalf("self-reference blocker was not projected: %+v", report)
	}
}

package pose_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func writeAssessmentFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func neutralAssessmentRepo(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "acme-neutral")
	modules := `{
  "schemaVersion": 1,
  "modules": {
    "service-a": {"owner":"team-a","criticality":"high","domain":"backend-go","validationProfile":"go"},
    "service-b": {"owner":"team-b","criticality":"medium","domain":"backend-go","validationProfile":"go"},
    "contracts": {"owner":"api-team","criticality":"high","domain":"contracts","validationProfile":"contract"},
    "mcp-provider": {"owner":"tools-team","criticality":"medium","domain":"tools","validationProfile":"node"}
  }
}`
	writeAssessmentFixture(t, root, ".pose/indexes/module-metadata.json", modules)
	writeAssessmentFixture(t, root, ".pose/specs/widget-maintenance/spec.md", `---
slug: widget-maintenance
status: in-progress
created_at: 2026-08-03
priority: 0
---
# Widget maintenance
Track the debt at service-a/debt.go before closeout.
`)
	writeAssessmentFixture(t, root, ".pose/specs/closed-work/spec.md", `---
slug: closed-work
status: done
created_at: 2026-08-01
completed_at: 2026-08-02
priority: 1
---
# Closed work
The old delivery referenced service-b/untracked.go.
`)
	writeAssessmentFixture(t, root, "service-a/go.mod", "module example.test/service-a\n\ngo 1.24\n")
	writeAssessmentFixture(t, root, "service-a/routes.go", `package servicea
func routes(mux *Mux) {
  mux.HandleFunc("/v1/widgets", widgets)
  mux.HandleFunc("/v1/orphan", orphan)
}
func events(bus *Bus) { bus.Publish("widgets.changed", "payload.json") }
`)
	writeAssessmentFixture(t, root, "service-a/debt.go", `package servicea
// TODO: remove the compatibility branch tracked by the active spec
func legacy() { panic("compatibility branch") }
`)
	writeAssessmentFixture(t, root, "service-a/lifetime.rs", "fn pending<'a>(value: &'a str) { todo!() }\n")
	writeAssessmentFixture(t, root, "service-b/go.mod", "module example.test/service-b\n\ngo 1.24\n")
	writeAssessmentFixture(t, root, "service-b/client.go", `package serviceb
func load(base string) { http.Get(base + "/v1/widgets"); log("/tmp/not-an-endpoint") }
func documented(base string) { http.Get(base + "/v1/documented") }
func events(bus *Bus) {
  bus.Subscribe("widgets.changed")
  bus.Subscribe("payments.missing")
}
var audit = struct{ Consumer string }{Consumer: "not.a.topic"}
func tool(client *Client) { client.CallTool("widget_lookup") /* tool_name */ }
var _ WidgetService
`)
	writeAssessmentFixture(t, root, "service-b/untracked.go", "package serviceb\n// FIXME: no active backlog reference\n")
	writeAssessmentFixture(t, root, "service-b/labels.go", "package serviceb\nvar markerLabels = `TODO FIXME HACK panic( unimplemented!(`\n")
	writeAssessmentFixture(t, root, "contracts/widget.proto", "syntax = \"proto3\";\nservice WidgetService {}\nmessage Widget {}\n")
	// This directory is deliberately absent from module-metadata: repository
	// assessment must still observe it and derive a fallback component identity.
	writeAssessmentFixture(t, root, "external-contracts/openapi.yaml", "openapi: 3.1.0\npaths:\n  /v1/documented:\n    get:\n      responses: {}\n")
	writeAssessmentFixture(t, root, "mcp-provider/package.json", "{\"name\":\"neutral-mcp\"}\n")
	writeAssessmentFixture(t, root, "mcp-provider/catalog.json", "{\"name\":\"POSE_PROJECT_ROOT\",\"tools\":[{\"inputSchema\":{\"type\":\"object\"},\"description\":\"lookup\",\"name\":\"widget_lookup\"}]}\n")
	writeAssessmentFixture(t, root, ".generated-site/bundle.js", "bus.subscribe(\"generated.noise\"); // TODO generated\n")
	return root
}

func findContract(matrix *pose.IntegrationMatrix, protocol, contains string) *pose.IntegrationContract {
	for index := range matrix.Contracts {
		contract := &matrix.Contracts[index]
		if contract.Protocol == protocol && strings.Contains(contract.Name, contains) {
			return contract
		}
	}
	return nil
}

func TestProjectAgnosticAssessmentEngines(t *testing.T) {
	root := neutralAssessmentRepo(t)
	store := pose.Store{Root: root}

	component, err := store.DiscoverComponent("service-a")
	if err != nil {
		t.Fatal(err)
	}
	if component.ComponentSlug != "service-a" || component.Metadata["owner"] != "team-a" {
		t.Fatalf("discovery identity/metadata = %#v", component)
	}
	if err := store.SaveComponentState(component); err != nil {
		t.Fatal(err)
	}
	if err := store.GenerateConsolidatedAssessment([]*pose.ComponentDiscoveryState{component}); err != nil {
		t.Fatal(err)
	}

	matrix, err := store.AnalyzeIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	if contract := findContract(matrix, "rest", "/v1/widgets"); contract == nil || contract.Status != "active" {
		t.Fatalf("active REST contract not derived: %#v", contract)
	}
	if contract := findContract(matrix, "rest", "/v1/orphan"); contract == nil || contract.Status != "gap" {
		t.Fatalf("provider-only REST gap not derived: %#v", contract)
	}
	if contract := findContract(matrix, "rest", "/tmp/not-an-endpoint"); contract != nil {
		t.Fatalf("non-client path misclassified as REST consumer: %#v", contract)
	}
	if contract := findContract(matrix, "rest", "/v1/documented"); contract == nil || contract.Status != "active" || contract.Provider != "external-contracts" {
		t.Fatalf("OpenAPI REST contract not derived: %#v", contract)
	}
	if contract := findContract(matrix, "message", "payments.missing"); contract == nil || contract.Provider != "unobserved" {
		t.Fatalf("consumer-only message gap not derived: %#v", contract)
	}
	if contract := findContract(matrix, "message", "payload.json"); contract != nil {
		t.Fatalf("message payload misclassified as topic: %#v", contract)
	}
	if contract := findContract(matrix, "message", "not.a.topic"); contract != nil {
		t.Fatalf("ordinary consumer field misclassified as message topic: %#v", contract)
	}
	if contract := findContract(matrix, "message", "generated.noise"); contract != nil {
		t.Fatalf("hidden generated directory was assessed: %#v", contract)
	}
	if contract := findContract(matrix, "protobuf", "widget.proto"); contract == nil || contract.Status != "active" {
		t.Fatalf("protobuf contract not derived: %#v", contract)
	}
	if contract := findContract(matrix, "mcp", "widget_lookup"); contract == nil || contract.Status != "active" {
		t.Fatalf("MCP contract not derived: %#v", contract)
	}
	if contract := findContract(matrix, "mcp", "POSE_PROJECT_ROOT"); contract != nil {
		t.Fatalf("ordinary JSON name misclassified as MCP tool: %#v", contract)
	}
	if err := store.SaveIntegrationMatrix(matrix); err != nil {
		t.Fatal(err)
	}

	report, err := store.AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TotalMarkers != 4 || report.Summary.CoveredCount != 2 || report.Summary.UncoveredCount != 2 {
		t.Fatalf("debt reconciliation summary = %#v", report.Summary)
	}
	for _, item := range report.Items {
		switch item.File {
		case "service-a/debt.go":
			if item.Coverage != "covered_by_spec" || item.CoverageRef != "spec:widget-maintenance" || item.Recommendation != "none" {
				t.Fatalf("active spec coverage = %#v", item)
			}
		case "service-b/untracked.go":
			if item.Coverage != "uncovered" || item.CoverageRef != "" {
				t.Fatalf("done spec must not cover debt = %#v", item)
			}
		case "service-a/lifetime.rs":
			if item.Marker != "STUB" || item.Coverage != "uncovered" {
				t.Fatalf("Rust lifetime/stub classification = %#v", item)
			}
		}
	}
	if err := store.SaveTechDebtReport(report); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		".pose/assessments/service-a.md", ".pose/assessments/consolidated.md",
		".pose/assessments/integrations.md", ".pose/assessments/technical-debt.md",
	} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(raw))
		for _, forbidden := range []string{"harne8", "graphforge", "conductor", "harness", "portal", "pose-dist"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s leaked producer identity %q", rel, forbidden)
			}
		}
	}

	second, err := store.AnalyzeIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(matrix.Contracts, second.Contracts) || !reflect.DeepEqual(matrix.Gaps, second.Gaps) {
		t.Fatal("integration observations are not deterministic")
	}
}

func TestComponentPathConfinement(t *testing.T) {
	root := neutralAssessmentRepo(t)
	store := pose.Store{Root: root}
	for _, candidate := range []string{"../outside", filepath.Join(root, "service-a")} {
		if _, err := store.DiscoverComponent(candidate); err == nil {
			t.Fatalf("DiscoverComponent(%q) accepted an escaping path", candidate)
		}
	}
	outside := t.TempDir()
	link := filepath.Join(root, "outside-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := store.DiscoverComponent("outside-link"); err == nil {
		t.Fatal("DiscoverComponent accepted a symlink outside the project")
	}
}

func TestComponentSlugFallbackIsStableAndNonEmpty(t *testing.T) {
	root := neutralAssessmentRepo(t)
	writeAssessmentFixture(t, root, "---/main.go", "package punctuation\n")
	store := pose.Store{Root: root}
	first, err := store.DiscoverComponent("---")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.DiscoverComponent("---")
	if err != nil {
		t.Fatal(err)
	}
	if first.ComponentSlug == "" || first.ComponentSlug != second.ComponentSlug || !strings.HasPrefix(first.ComponentSlug, "component-") {
		t.Fatalf("invalid fallback slug: %q / %q", first.ComponentSlug, second.ComponentSlug)
	}
}

func TestProjectAgnosticSourceTemplates(t *testing.T) {
	for _, filename := range []string{"discovery.go", "integration.go", "techdebt.go"} {
		raw, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(raw))
		for _, forbidden := range []string{"harne8", "graphforge", "conductor", "harness", "portal", "pose-dist"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains project identity %q", filename, forbidden)
			}
		}
	}
}

func TestProjectAgnosticEmptyIntegrationMatrix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "empty-neutral")
	writeAssessmentFixture(t, root, "README.md", "empty fixture\n")
	matrix, err := (pose.Store{Root: root}).AnalyzeIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(matrix)
	if len(matrix.Contracts) != 0 || len(matrix.Gaps) != 0 || strings.Contains(strings.ToLower(string(encoded)), "graph") {
		t.Fatalf("empty repository produced foreign integrations: %s", encoded)
	}
}

package cli

// Structured result contract behavior (spec pose-structured-validation-results):
// canonical JSON with stable IDs and distinguishable outcomes, JUnit/SARIF
// projections, deterministic skip reasons, bounded capture and redaction.

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resultFixture builds a project whose matrix exercises pass, fail, infra
// error, skip and secret redaction in one deterministic run.
func resultFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mod/go.mod", "module fixture\n")
	matrix := map[string]any{
		"defaults": map[string]any{"mode": "strict"},
		"stacks": map[string]any{
			"go": map[string]any{"checks": []map[string]any{
				{"name": "ok", "program": "true", "severity": "required"},
				{"name": "leaky", "program": "env", "severity": "optional",
					"env": map[string]string{"MY_API_TOKEN": "hush123", "PLAIN": "visible"}},
				{"name": "broken", "program": "definitely-missing-binary-xyz", "severity": "optional"},
				{"name": "conditional", "program": "true", "severity": "required",
					"when": map[string]string{"fileExists": "nope.txt"}},
			}},
		},
	}
	raw, _ := json.Marshal(matrix)
	write(".pose/indexes/validation-matrix.json", string(raw))
	return root
}

func runValidate(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := cmdValidate(root, args, &out, &errB)
	return code, out.String() + errB.String()
}

func loadRun(t *testing.T, root string) validationRunResult {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var run validationRunResult
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	return run
}

func TestValidateStructuredJSONContract(t *testing.T) {
	root := resultFixture(t)
	code, out := runValidate(t, root, "--json", "result.json")
	if code != 0 {
		t.Fatalf("only optional failures must not block: exit=%d output=%s", code, out)
	}
	run := loadRun(t, root)
	if run.SchemaVersion != validationResultSchema || run.Mode != "strict" {
		t.Errorf("schema/mode = %d/%s", run.SchemaVersion, run.Mode)
	}
	if run.Outcome != "partial" {
		t.Errorf("outcome = %q, want partial (tolerated failures stay distinguishable)", run.Outcome)
	}
	byName := map[string]checkResult{}
	for _, c := range run.Checks {
		byName[c.Name] = c
	}
	if c := byName["ok"]; c.Outcome != "pass" || c.ID != "mod/go/ok" {
		t.Errorf("ok = %+v", c)
	}
	if c := byName["broken"]; c.Outcome != "error" || c.ExitCode != nil {
		t.Errorf("infra failure must be error without exit code: %+v", c)
	}
	if c := byName["conditional"]; c.Outcome != "skipped" || !strings.Contains(c.SkipReason, "when.fileExists not met: nope.txt") {
		t.Errorf("skip must carry its deterministic reason: %+v", c)
	}
	if run.Counts.Skipped != 1 || run.Counts.Passed < 1 || run.Counts.Errored != 1 {
		t.Errorf("counts = %+v", run.Counts)
	}
}

func TestValidateRedactionAndBoundedCapture(t *testing.T) {
	root := resultFixture(t)
	if code, _ := runValidate(t, root, "--json", "result.json"); code != 0 {
		t.Fatal("run failed")
	}
	run := loadRun(t, root)
	var leaky checkResult
	for _, c := range run.Checks {
		if c.Name == "leaky" {
			leaky = c
		}
	}
	if leaky.Env["MY_API_TOKEN"] != redactedValue || leaky.Env["PLAIN"] != "visible" {
		t.Errorf("env metadata redaction: %+v", leaky.Env)
	}
	if strings.Contains(leaky.Output, "hush123") {
		t.Errorf("captured output leaked a configured secret: %s", leaky.Output)
	}
	if len(leaky.Output) > 5000 {
		t.Errorf("capture is unbounded: %d bytes", len(leaky.Output))
	}
	raw, _ := os.ReadFile(filepath.Join(root, "result.json"))
	if strings.Contains(string(raw), "hush123") {
		t.Error("secret value present anywhere in the JSON result")
	}
}

func TestValidateRequiredFailureOutcome(t *testing.T) {
	root := resultFixture(t)
	matrix := `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"boom","program":"false","severity":"required"}]}}}`
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _ := runValidate(t, root, "--json", "result.json")
	if code != 1 {
		t.Fatalf("required failure must exit 1, got %d", code)
	}
	run := loadRun(t, root)
	if run.Outcome != "fail" || run.Counts.Failed != 1 {
		t.Errorf("outcome/counts = %s/%+v", run.Outcome, run.Counts)
	}
	for _, c := range run.Checks {
		if c.Name == "boom" && (c.Outcome != "fail" || c.ExitCode == nil || *c.ExitCode != 1) {
			t.Errorf("boom = %+v", c)
		}
	}
}

func TestValidateJUnitProjection(t *testing.T) {
	root := resultFixture(t)
	if code, _ := runValidate(t, root, "--junit", "result.xml"); code != 0 {
		t.Fatal("run failed")
	}
	raw, err := os.ReadFile(filepath.Join(root, "result.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc junitSuites
	if err := xml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("JUnit output is not well-formed XML: %v", err)
	}
	if len(doc.Suites) != 1 || doc.Suites[0].Name != "mod" {
		t.Fatalf("suites = %+v", doc.Suites)
	}
	s := doc.Suites[0]
	if s.Tests != 4 || s.Errors != 1 || s.Skipped != 1 {
		t.Errorf("suite counters = tests:%d errors:%d skipped:%d", s.Tests, s.Errors, s.Skipped)
	}
	if strings.Contains(string(raw), "hush123") {
		t.Error("secret leaked into JUnit projection")
	}
}

func TestValidateSARIFProjection(t *testing.T) {
	root := resultFixture(t)
	if code, _ := runValidate(t, root, "--sarif", "result.sarif"); code != 0 {
		t.Fatal("run failed")
	}
	raw, err := os.ReadFile(filepath.Join(root, "result.sarif"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string           `json:"name"`
					Rules []map[string]any `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID     string         `json:"ruleId"`
				Level      string         `json:"level"`
				Properties map[string]any `json:"properties"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("SARIF is not valid JSON: %v", err)
	}
	if doc.Version != "2.1.0" || len(doc.Runs) != 1 || doc.Runs[0].Tool.Driver.Name != "pose-validate" {
		t.Fatalf("SARIF envelope: %+v", doc)
	}
	if len(doc.Runs[0].Tool.Driver.Rules) != 4 {
		t.Errorf("rules = %d, want one per check", len(doc.Runs[0].Tool.Driver.Rules))
	}
	if len(doc.Runs[0].Results) != 1 {
		t.Fatalf("results = %d, want only the infra error", len(doc.Runs[0].Results))
	}
	r := doc.Runs[0].Results[0]
	if r.RuleID != "mod/go/broken" || r.Level != "warning" || r.Properties["pose/outcome"] != "error" {
		t.Errorf("result = %+v (POSE outcome must survive in properties)", r)
	}
}

func TestValidateOutputPathConfined(t *testing.T) {
	root := resultFixture(t)
	code, out := runValidate(t, root, "--json", "../escape.json")
	if code != 2 || !strings.Contains(out, fmt.Sprintf("inside the project")) {
		t.Fatalf("escaping output path must be rejected: code=%d out=%s", code, out)
	}
}

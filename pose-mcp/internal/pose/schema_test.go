package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestSchemaDrift guards the contract between the Go structs and the
// versioned response schemas (ADR-014, additive-only): every field the Go
// side serializes must be declared in the schema, and every schema-required
// field must actually serialize.
func TestSchemaDrift(t *testing.T) {
	cases := []struct {
		schema string
		sample any
	}{
		{"spec.schema.json", Spec{Slug: "s", Status: "done", CreatedAt: "2026-01-01",
			CompletedAt: "2026-01-02", Supersedes: "old", Remediates: []string{"spec:old@defect-fix"}, Title: "T", Path: "p", Body: "b"}},
		{"artifact.schema.json", Markdown{Name: "n", Title: "T", Path: "p", Body: "b"}},
		{"gate-result.schema.json", GateResult{Command: "c", ExitCode: 1, Passed: false, Output: "o"}},
		{"insights.schema.json", InsightsResult{GroupBy: "workflow", SinceDays: 30,
			RecordsScanned: 2, RecordsSkippedByWindow: 1, RecordsSkippedInvalid: 0,
			Rows: []InsightRow{{Key: "feature", Pass: 1, Total: 1}}}},
		{"obligation.schema.json", sampleObligation(ObligationDependency)},
		// Spec pose-review-declared-limits-followups R5.
		{"review-plan.schema.json", ReviewPlan{AcceptedLimits: []ReviewAcceptedLimit{{ID: "L1", Text: "t", Spec: "s"}}, Limitations: []string{"l"}}},
		{"contract-nodes.schema.json", ProjectContractNodes("nodes", "# Spec\n\n## 2. Requirements\n\n- R1: One.\n\n## 5. Decisions\n\n### Decision D1\n- Basis: R7\n")},
	}
	for _, tc := range cases {
		t.Run(tc.schema, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "v1", tc.schema))
			if err != nil {
				t.Fatalf("reading schema: %v", err)
			}
			var schema struct {
				Properties map[string]any `json:"properties"`
				Required   []string       `json:"required"`
			}
			if err := json.Unmarshal(raw, &schema); err != nil {
				t.Fatalf("parsing schema: %v", err)
			}

			data, err := json.Marshal(tc.sample)
			if err != nil {
				t.Fatal(err)
			}
			var serialized map[string]any
			if err := json.Unmarshal(data, &serialized); err != nil {
				t.Fatal(err)
			}

			for field := range serialized {
				if _, ok := schema.Properties[field]; !ok {
					t.Errorf("field %q serialized by Go but absent from %s (additive-only: declare it)", field, tc.schema)
				}
			}
			for _, req := range schema.Required {
				if _, ok := serialized[req]; !ok {
					t.Errorf("schema-required field %q not serialized by the Go struct", req)
				}
			}
		})
	}
}

// Spec pose-review-declared-limits-followups R5: the bundle schema's plan
// declares the fields the sealed bundle plan carries.
func TestTheBundleSchemaDeclaresTheAcceptedLimits(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "v1", "review-bundle.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	var walk func(any) bool
	walk = func(node any) bool {
		switch v := node.(type) {
		case map[string]any:
			if p, ok := v["properties"].(map[string]any); ok {
				if _, digest := p["plan_digest"]; digest {
					_, limits := p["accepted_limits"]
					_, limitations := p["limitations"]
					return limits && limitations
				}
			}
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	if !walk(schema) {
		t.Fatal("the bundle schema's plan does not declare accepted_limits and limitations")
	}
}

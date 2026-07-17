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
			CompletedAt: "2026-01-02", Supersedes: "old", Title: "T", Path: "p", Body: "b"}},
		{"artifact.schema.json", Markdown{Name: "n", Title: "T", Path: "p", Body: "b"}},
		{"gate-result.schema.json", GateResult{Command: "c", ExitCode: 1, Passed: false, Output: "o"}},
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

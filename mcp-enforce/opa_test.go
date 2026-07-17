package mcpenforce

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestQueryOPA_WireDoc_Golden captures the exact JSON document the gate sends to
// OPA and pins it against a golden — the contract specs B and C must not regress.
func TestQueryOPA_WireDoc_Golden(t *testing.T) {
	cases := []struct {
		name   string
		input  PolicyInput
		golden string
	}{
		{
			name: "minimal",
			input: PolicyInput{
				Principal: "alice", ProjectID: "proj.a",
				Method: "tools/call", ToolName: "pose_get_spec",
			},
			golden: "testdata/opa_input_minimal.json",
		},
		{
			name: "aggregate_projects",
			input: PolicyInput{
				Principal: "svc.portal", ProjectIDs: []string{"proj.a", "proj.b"},
				Method: "tools/call", ToolName: "analytics_query",
			},
			golden: "testdata/opa_input_aggregate_projects.json",
		},
		{
			name: "with_scope",
			input: PolicyInput{
				Principal: "svc.worker", ProjectID: "proj.a",
				Method: "tools/call", ToolName: "graph_query",
				Scopes:    []string{"repo:read", "graph:read"},
				RunID:     "run-123",
				ExpiresAt: time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
			},
			golden: "testdata/opa_input_with_scope.json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured []byte
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result": map[string]any{"allow": true, "violations": []string{}},
				})
			}))
			defer ts.Close()

			// Freeze the clock before the with_scope ExpiresAt so the time-box
			// does not short-circuit the OPA call this test is capturing.
			frozen := time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC)
			g := NewPolicyGate(PolicyConfig{OPAURL: ts.URL, HTTPClient: ts.Client(), Clock: func() time.Time { return frozen }})
			if _, err := g.Evaluate(context.Background(), tc.input); err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, captured, "", "  "); err != nil {
				t.Fatalf("indent captured body: %v (body=%s)", err, captured)
			}
			compareGolden(t, tc.golden, pretty.Bytes())
		})
	}
}

// TestQueryOPA_HitsConfiguredPath confirms the request targets /v1/data/<path>.
func TestQueryOPA_HitsConfiguredPath(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"allow": true}})
	}))
	defer ts.Close()

	g := NewPolicyGate(PolicyConfig{OPAURL: ts.URL, OPAPath: "pose/mcp/allow", HTTPClient: ts.Client()})
	if _, err := g.Evaluate(context.Background(), PolicyInput{ToolName: "pose_get_spec"}); err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if want := "/v1/data/pose/mcp/allow"; gotPath != want {
		t.Errorf("OPA path = %q, want %q", gotPath, want)
	}
}

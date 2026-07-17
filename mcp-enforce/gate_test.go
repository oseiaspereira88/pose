package mcpenforce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// opaFixture starts a minimal OPA-compatible HTTP stub that returns body
// (JSON-encoded) with statusCode on any POST /v1/data/... request.
func opaFixture(t *testing.T, statusCode int, body any) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestPolicyGate_DevMode_AllowAll(t *testing.T) {
	g := NewPolicyGate(PolicyConfig{})
	for _, tool := range []string{"pose_get_spec", "pose_list_specs", "pose_check"} {
		d, err := g.Evaluate(context.Background(), PolicyInput{
			Principal: "alice", ProjectID: "proj.a", ToolName: tool,
		})
		if err != nil {
			t.Fatalf("tool %s: unexpected error: %v", tool, err)
		}
		if !d.Allow {
			t.Errorf("tool %s: expected allow in dev mode, got deny (violations=%v)", tool, d.Violations)
		}
	}
}

func TestPolicyGate_InvalidProjectScopeDeniedInDevMode(t *testing.T) {
	g := NewPolicyGate(PolicyConfig{})
	d, err := g.Evaluate(context.Background(), PolicyInput{
		Principal: "svc.portal", ToolName: "analytics_query", InvalidProjectScope: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Allow || len(d.Violations) != 1 || d.Violations[0] != "invalid_project_scope" {
		t.Fatalf("decision = %+v, want invalid_project_scope denial", d)
	}
}

func TestPolicyGate_RequirePrincipal_DeniesAnonymous(t *testing.T) {
	g := NewPolicyGate(PolicyConfig{RequirePrincipal: true})
	d, err := g.Evaluate(context.Background(), PolicyInput{ProjectID: "proj.a", ToolName: "pose_get_spec"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Allow {
		t.Fatal("anonymous call allowed with RequirePrincipal, want deny")
	}
	if len(d.Violations) != 1 || d.Violations[0] != "missing_principal" {
		t.Fatalf("violations = %v, want [missing_principal]", d.Violations)
	}
	// A named principal passes the principal check (allowed in dev mode).
	d, err = g.Evaluate(context.Background(), PolicyInput{Principal: "svc.worker", ProjectID: "proj.a", ToolName: "pose_get_spec"})
	if err != nil || !d.Allow {
		t.Fatalf("named principal = %+v err=%v, want allow", d, err)
	}
}

func TestPolicyGate_OPA_Allow(t *testing.T) {
	stub := opaFixture(t, http.StatusOK, map[string]any{
		"result": map[string]any{"allow": true, "violations": []string{}},
	})
	g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub.Client()})
	d, err := g.Evaluate(context.Background(), PolicyInput{Principal: "alice", ProjectID: "proj.a", ToolName: "pose_get_spec"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Allow {
		t.Errorf("expected allow from OPA, got deny (violations=%v)", d.Violations)
	}
}

func TestPolicyGate_OPA_Deny(t *testing.T) {
	stub := opaFixture(t, http.StatusOK, map[string]any{
		"result": map[string]any{"allow": false, "violations": []string{"principal_not_authorized"}},
	})
	g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub.Client()})
	d, err := g.Evaluate(context.Background(), PolicyInput{Principal: "eve", ProjectID: "proj.a", ToolName: "pose_get_spec"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Allow {
		t.Error("expected deny from OPA, got allow")
	}
	if len(d.Violations) == 0 || d.Violations[0] != "principal_not_authorized" {
		t.Errorf("violations = %v", d.Violations)
	}
}

func TestPolicyGate_OPA_UndefinedPath(t *testing.T) {
	stub := opaFixture(t, http.StatusOK, map[string]any{"result": nil})
	g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub.Client()})
	d, err := g.Evaluate(context.Background(), PolicyInput{ToolName: "pose_get_spec"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Allow {
		t.Error("expected deny for undefined OPA path, got allow")
	}
	if len(d.Violations) == 0 || d.Violations[0] != "policy_path_undefined" {
		t.Errorf("violations = %v, want [policy_path_undefined]", d.Violations)
	}
}

func TestPolicyGate_OPA_ServerError_ReturnsError(t *testing.T) {
	stub := opaFixture(t, http.StatusInternalServerError, nil)
	g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub.Client()})
	if _, err := g.Evaluate(context.Background(), PolicyInput{ToolName: "pose_get_spec"}); err == nil {
		t.Error("expected error from OPA 500, got nil")
	}
}

func TestPolicyGate_OPA_Unreachable_ReturnsError(t *testing.T) {
	g := NewPolicyGate(PolicyConfig{OPAURL: "http://127.0.0.1:0", Timeout: 100 * time.Millisecond})
	if _, err := g.Evaluate(context.Background(), PolicyInput{ToolName: "pose_get_spec"}); err == nil {
		t.Error("expected error for unreachable OPA, got nil")
	}
}

func TestPolicyGate_RequireIdentity_DeniesWithoutRunID(t *testing.T) {
	g := NewPolicyGate(PolicyConfig{RequireIdentity: true})
	// No RunID → denied even in dev/allow-all mode.
	d, err := g.Evaluate(context.Background(), PolicyInput{Principal: "svc.worker", ToolName: "graph_query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Allow || len(d.Violations) != 1 || d.Violations[0] != "missing_identity" {
		t.Fatalf("decision = %+v, want deny [missing_identity]", d)
	}
	// A run-bound identity passes the identity check (allowed in dev mode).
	d, err = g.Evaluate(context.Background(), PolicyInput{Principal: "svc.worker", RunID: "run-1", ToolName: "graph_query"})
	if err != nil || !d.Allow {
		t.Fatalf("run-bound = %+v err=%v, want allow", d, err)
	}
}

func TestPolicyGate_TimeBox_DeniesExpiredIdentity(t *testing.T) {
	frozen := time.Date(2026, 6, 25, 13, 0, 0, 0, time.UTC)
	g := NewPolicyGate(PolicyConfig{Clock: func() time.Time { return frozen }})

	expired := PolicyInput{
		Principal: "svc.worker", RunID: "run-1", ToolName: "graph_query",
		ExpiresAt: frozen.Add(-time.Minute), // already past
	}
	d, err := g.Evaluate(context.Background(), expired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Allow || len(d.Violations) != 1 || d.Violations[0] != "identity_expired" {
		t.Fatalf("expired = %+v, want deny [identity_expired]", d)
	}

	// An identity that is still valid is allowed (dev mode).
	valid := expired
	valid.ExpiresAt = frozen.Add(time.Hour)
	if d, err := g.Evaluate(context.Background(), valid); err != nil || !d.Allow {
		t.Fatalf("valid = %+v err=%v, want allow", d, err)
	}
}

func TestDenyDecision_Fields(t *testing.T) {
	input := PolicyInput{Principal: "alice", ProjectID: "proj.a", ToolName: "pose_get_spec"}
	d := DenyDecision(input, "policy_error")
	if d.Allow {
		t.Error("DenyDecision must set Allow=false")
	}
	if d.Principal != "alice" || d.ProjectID != "proj.a" || d.ToolName != "pose_get_spec" {
		t.Errorf("identity not carried: %+v", d)
	}
	if len(d.Violations) != 1 || d.Violations[0] != "policy_error" {
		t.Errorf("Violations = %v", d.Violations)
	}
}

func TestPolicyDecision_Metadata(t *testing.T) {
	d := PolicyDecision{Principal: "bob", ProjectID: "proj.b", ToolName: "pose_list_specs", Violations: []string{"tool_denied"}}
	m := d.Metadata()
	if m["principal"] != "bob" || m["tool"] != "pose_list_specs" {
		t.Errorf("metadata = %v", m)
	}
	vs, _ := m["violations"].([]string)
	if len(vs) != 1 || vs[0] != "tool_denied" {
		t.Errorf("violations = %v", m["violations"])
	}
}

func TestPolicyDecision_Metadata_NilViolationsBecomesEmpty(t *testing.T) {
	m := PolicyDecision{Allow: true}.Metadata()
	vs, ok := m["violations"].([]string)
	if !ok || len(vs) != 0 {
		t.Errorf("violations = %v (%T), want empty slice", m["violations"], m["violations"])
	}
}

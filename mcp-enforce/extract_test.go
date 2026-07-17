package mcpenforce

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestPrincipalFromHeader(t *testing.T) {
	h := http.Header{}
	if got := PrincipalFromHeader(h); got != "" {
		t.Errorf("empty header = %q, want \"\"", got)
	}
	h.Set("X-Principal", "fallback")
	if got := PrincipalFromHeader(h); got != "fallback" {
		t.Errorf("fallback = %q, want \"fallback\"", got)
	}
	h.Set("X-MCP-Principal", "  primary  ")
	if got := PrincipalFromHeader(h); got != "primary" {
		t.Errorf("primary (trimmed) = %q, want \"primary\"", got)
	}
}

func TestProjectIDFromArguments(t *testing.T) {
	if got := ProjectIDFromArguments(json.RawMessage(`{"project_id":"proj.a","x":1}`)); got != "proj.a" {
		t.Errorf("got %q, want proj.a", got)
	}
	if got := ProjectIDFromArguments(json.RawMessage(`{"x":1}`)); got != "" {
		t.Errorf("absent = %q, want \"\"", got)
	}
	if got := ProjectIDFromArguments(json.RawMessage(`not json`)); got != "" {
		t.Errorf("malformed = %q, want \"\"", got)
	}
}

func TestProjectScopeFromArguments(t *testing.T) {
	projectID, projectIDs, invalid := ProjectScopeFromArguments(json.RawMessage(`{"project_id":" proj.legacy ","project_ids":["proj.b"," ","proj.a","proj.b"]}`))
	if invalid {
		t.Fatal("valid scope marked invalid")
	}
	if projectID != "proj.legacy" {
		t.Fatalf("projectID = %q", projectID)
	}
	want := []string{"proj.b", "proj.a"}
	if len(projectIDs) != len(want) || projectIDs[0] != want[0] || projectIDs[1] != want[1] {
		t.Fatalf("projectIDs = %#v, want %#v", projectIDs, want)
	}
	projectID, projectIDs, invalid = ProjectScopeFromArguments(json.RawMessage(`{"project_ids":"invalid"}`))
	if projectID != "" || projectIDs != nil || !invalid {
		t.Fatalf("malformed scope = %q, %#v, invalid=%v", projectID, projectIDs, invalid)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("POSE_MCP_OPA_URL", "http://opa:8181")
	t.Setenv("POSE_MCP_OPA_TIMEOUT", "3")
	t.Setenv("POSE_MCP_REQUIRE_PRINCIPAL", "true")

	cfg := ConfigFromEnv("POSE_MCP_", "pose/mcp/allow")
	if cfg.OPAURL != "http://opa:8181" {
		t.Errorf("OPAURL = %q", cfg.OPAURL)
	}
	if cfg.OPAPath != "pose/mcp/allow" {
		t.Errorf("OPAPath = %q, want default pose/mcp/allow", cfg.OPAPath)
	}
	if cfg.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", cfg.Timeout)
	}
	if !cfg.RequirePrincipal {
		t.Error("RequirePrincipal = false, want true")
	}
}

func TestConfigFromEnv_PathOverride(t *testing.T) {
	t.Setenv("GF_MCP_OPA_PATH", "graphforge/mcp/allow")
	cfg := ConfigFromEnv("GF_MCP_", "mcp/allow")
	if cfg.OPAPath != "graphforge/mcp/allow" {
		t.Errorf("OPAPath = %q, want explicit override", cfg.OPAPath)
	}
	if cfg.RequirePrincipal {
		t.Error("RequirePrincipal = true, want false when unset")
	}
}

func TestIsTruthy(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on", " true "} {
		if !isTruthy(v) {
			t.Errorf("isTruthy(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "off", "maybe"} {
		if isTruthy(v) {
			t.Errorf("isTruthy(%q) = true, want false", v)
		}
	}
}

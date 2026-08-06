package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcpenforce "github.com/harne8/mcp-enforce"
	"github.com/harne8/pose-mcp/internal/pose"
)

func callMCPContext(t *testing.T, ts *httptest.Server, token, projectID string) rpcResult {
	t.Helper()
	arguments := `{}`
	if projectID != "" {
		arguments = `{"project_id":` + strconvQuote(projectID) + `}`
	}
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_mcp_context","arguments":` + arguments + `}}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MCP-Principal", "svc.worker")
	if token != "" {
		req.Header.Set(mcpenforce.IdentityHeader, token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out rpcResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func strconvQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func contextStructured(t *testing.T, out rpcResult) map[string]any {
	t.Helper()
	if out.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", out.Error)
	}
	if isErr, _ := out.Result["isError"].(bool); isErr {
		t.Fatalf("unexpected tool error: %+v", out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured == nil {
		t.Fatalf("missing structuredContent: %+v", out.Result)
	}
	return structured
}

func TestMCPContextReportsActivePathFreeConnection(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      "/secret/root-a",
		DefaultProjectID: "proj.a",
		Explicit:         map[string]string{"proj.b": "/secret/root-b"},
	})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	first := contextStructured(t, callMCPContext(t, ts, "", ""))
	second := contextStructured(t, callMCPContext(t, ts, "", ""))
	if first["connection_active"] != true || first["transport"] != "streamable-http" {
		t.Fatalf("unexpected connection metadata: %+v", first)
	}
	if first["selection_mode"] != "legacy-default-multi-project" || first["default_project_id"] != "proj.a" {
		t.Fatalf("unexpected selection metadata: %+v", first)
	}
	if first["server_instance_id"] == "" || first["server_instance_id"] != second["server_instance_id"] {
		t.Fatalf("server instance id must be non-empty and stable: first=%v second=%v", first["server_instance_id"], second["server_instance_id"])
	}
	ids, _ := first["available_project_ids"].([]any)
	if len(ids) != 2 || ids[0] != "proj.a" || ids[1] != "proj.b" {
		t.Fatalf("available_project_ids = %v, want [proj.a proj.b]", ids)
	}
	raw, _ := json.Marshal(first)
	for _, root := range []string{"/secret/root-a", "/secret/root-b"} {
		if strings.Contains(string(raw), root) {
			t.Errorf("context leaked root %q: %s", root, raw)
		}
	}
}

func TestMCPContextProbesUnknownProjectWithoutStoreResolution(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      t.TempDir(),
		DefaultProjectID: "proj.a",
	})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	structured := contextStructured(t, callMCPContext(t, ts, "", "proj.ghost"))
	requested, _ := structured["requested_project"].(map[string]any)
	if requested["project_id"] != "proj.ghost" || requested["status"] != "unknown" {
		t.Fatalf("requested_project = %+v", requested)
	}
	remediation, _ := structured["remediation"].([]any)
	if len(remediation) < 2 {
		t.Fatalf("unknown project must include structured remediation: %+v", structured)
	}
}

func TestUnknownProjectErrorOffersAuthorizedAlternativesWithoutPaths(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      "/secret/root-a",
		DefaultProjectID: "proj.a",
		Explicit:         map[string]string{"proj.b": "/secret/root-b"},
	})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha","project_id":"proj.ghost"}}}`)
	if isErr, _ := out.Result["isError"].(bool); !isErr {
		t.Fatalf("unknown project must be a tool error: %+v", out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	ids, _ := structured["available_project_ids"].([]any)
	if len(ids) != 2 || ids[0] != "proj.a" || ids[1] != "proj.b" {
		t.Fatalf("available alternatives = %v, want [proj.a proj.b]", ids)
	}
	raw, _ := json.Marshal(structured)
	for _, root := range []string{"/secret/root-a", "/secret/root-b"} {
		if strings.Contains(string(raw), root) {
			t.Errorf("project_unknown remediation leaked root %q: %s", root, raw)
		}
	}
}

func TestMCPContextIdentityFiltersProjectsAndDeniesAnonymous(t *testing.T) {
	secret := []byte("context-test-secret")
	frozen := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      t.TempDir(),
		DefaultProjectID: "proj.a",
		Explicit:         map[string]string{"proj.b": t.TempDir()},
	})
	gate := NewPolicyGate(PolicyConfig{RequireIdentity: true, Clock: func() time.Time { return frozen }})
	ts := httptest.NewServer(NewWithRootsAndPolicy(roots, gate).WithIdentitySecret(secret).Handler("", ""))
	t.Cleanup(ts.Close)

	denied := callMCPContext(t, ts, "", "")
	if denied.Error == nil || denied.Error.Code != -32004 {
		t.Fatalf("anonymous context call must be policy denied: %+v", denied)
	}
	token, err := mcpenforce.MintToken(mcpenforce.Identity{
		RunID: "run-context", ProjectID: "proj.a", ExpiresAt: frozen.Add(time.Hour),
	}, secret)
	if err != nil {
		t.Fatal(err)
	}
	structured := contextStructured(t, callMCPContext(t, ts, token, ""))
	ids, _ := structured["available_project_ids"].([]any)
	if len(ids) != 1 || ids[0] != "proj.a" {
		t.Fatalf("identity must see only its authorized project: %v", ids)
	}
	raw, _ := json.Marshal(structured)
	if strings.Contains(string(raw), "proj.b") {
		t.Fatalf("unauthorized project leaked through context: %s", raw)
	}
}

func TestMCPContextWithoutDefaultRootDemandsExplicitSelection(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{Explicit: map[string]string{"proj.a": t.TempDir()}})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	structured := contextStructured(t, callMCPContext(t, ts, "", ""))
	if structured["selection_mode"] != "explicit-required" {
		t.Fatalf("selection_mode = %v, want explicit-required", structured["selection_mode"])
	}
	remediation, _ := structured["remediation"].([]any)
	for _, raw := range remediation {
		if entry, _ := raw.(map[string]any); entry["code"] == "select-project-explicitly" {
			return
		}
	}
	t.Fatalf("a connection without a default root must tell the caller to pass project_id: %+v", structured)
}

func TestMCPContextBoundsDiscoveryProbesAndReportsTruncation(t *testing.T) {
	explicit := map[string]string{}
	for i := range maxDiscoveryProbes + 6 {
		explicit[fmt.Sprintf("proj.%03d", i)] = t.TempDir()
	}
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: t.TempDir(), Explicit: explicit})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	structured := contextStructured(t, callMCPContext(t, ts, "", ""))
	ids, _ := structured["available_project_ids"].([]any)
	if len(ids) != maxDiscoveryProbes {
		t.Fatalf("discovery evaluated %d projects, want the %d ceiling", len(ids), maxDiscoveryProbes)
	}
	if structured["available_project_ids_truncated"] != true {
		t.Fatalf("a truncated registry must say so instead of looking complete: %+v", structured)
	}
	if ids[0] != "proj.000" {
		t.Fatalf("truncation must stay deterministic, got first id %v", ids[0])
	}
}

func TestMCPContextStdioTransport(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: t.TempDir(), DefaultProjectID: "proj.a"})
	s := NewWithRoots(roots)
	params := json.RawMessage(`{"name":"pose_mcp_context","arguments":{}}`)
	resp := s.dispatchRPC(t.Context(), rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/call", Params: params})
	resultMap, _ := resp.Result.(map[string]any)
	structured, _ := resultMap["structuredContent"].(map[string]any)
	if structured["transport"] != "stdio" {
		t.Fatalf("transport = %v, want stdio", structured["transport"])
	}
}

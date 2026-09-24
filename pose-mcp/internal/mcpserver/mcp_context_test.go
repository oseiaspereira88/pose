package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

func writeMultiRepoMCPProject(t *testing.T, root, slug, status string, schema int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := fmt.Sprintf(`{"schema_version":%d,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1}`, schema) + "\n"
	if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "policy", "review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	if slug != "" {
		body := "---\nslug: " + slug + "\nstatus: " + status + "\ncreated_at: 2026-09-24\n---\n# " + slug + "\n"
		if err := os.WriteFile(filepath.Join(root, ".pose", "specs", "2026-09-24-"+slug+".md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "pose-mcp-test@example.invalid"}, {"config", "user.name", "POSE MCP fixture"}, {"add", "-A"}, {"commit", "-qm", "fixture"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
}

func TestMultiRepoAgentSurfaceReportsPathFreeQualifiedTaskContext(t *testing.T) {
	parent, executor := t.TempDir(), t.TempDir()
	writeMultiRepoMCPProject(t, parent, "", "", 4)
	writeMultiRepoMCPProject(t, executor, "checkout", "in-progress", 4)
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      parent,
		DefaultProjectID: "proj.parent",
		Explicit:         map[string]string{"proj.executor": executor},
	})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_mcp_context","arguments":{"project_id":"proj.parent","task_ref":"xref:proj.executor/spec:checkout"}}}`
	_, result := post(t, ts, request)
	structured := contextStructured(t, result)
	projectContext, _ := structured["project_context"].(map[string]any)
	if projectContext["selected_project_id"] != "proj.parent" || projectContext["task_ref"] != "xref:proj.executor/spec:checkout" || projectContext["context_revision"] == "" {
		t.Fatalf("MCP project context omitted selection or freshness token: %+v", projectContext)
	}
	authority, _ := projectContext["authority"].(map[string]any)
	resolution, _ := projectContext["task_resolution"].(map[string]any)
	if authority["project_id"] != "proj.executor" || authority["slug"] != "checkout" || resolution["resolution_state"] != "resolved" || resolution["source_revision"] == "" {
		t.Fatalf("MCP did not resolve canonical authority and revision: %+v", projectContext)
	}
	if projectContext["coordinator_relation"] != "selected-project-to-qualified-authority" {
		t.Fatalf("coordinator relationship is missing: %+v", projectContext)
	}
	raw, _ := json.Marshal(structured)
	for _, root := range []string{parent, executor} {
		if strings.Contains(string(raw), root) {
			t.Fatalf("MCP project context leaked root %q: %s", root, raw)
		}
	}
}

func TestMultiRepoAgentNegativeBindingChangeRequiresFreshMCPConnection(t *testing.T) {
	parent, executor := t.TempDir(), t.TempDir()
	writeMultiRepoMCPProject(t, parent, "", "", 4)
	writeMultiRepoMCPProject(t, executor, "checkout", "in-progress", 4)
	clone := filepath.Join(t.TempDir(), "proj.executor")
	if output, err := exec.Command("git", "clone", "-q", executor, clone).CombinedOutput(); err != nil {
		t.Fatalf("clone executor binding: %v: %s", err, output)
	}
	firstRoots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "proj.parent", Explicit: map[string]string{"proj.executor": executor}})
	firstServer := httptest.NewServer(NewWithRoots(firstRoots).Handler("", ""))
	t.Cleanup(firstServer.Close)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_mcp_context","arguments":{"project_id":"proj.parent","task_ref":"xref:proj.executor/spec:checkout"}}}`
	_, firstResponse := post(t, firstServer, request)
	first := contextStructured(t, firstResponse)
	firstContext, _ := first["project_context"].(map[string]any)
	firstResolution, _ := firstContext["task_resolution"].(map[string]any)
	secondRoots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "proj.parent", Explicit: map[string]string{"proj.executor": clone}})
	secondServer := httptest.NewServer(NewWithRoots(secondRoots).Handler("", ""))
	t.Cleanup(secondServer.Close)
	_, secondResponse := post(t, secondServer, request)
	second := contextStructured(t, secondResponse)
	secondContext, _ := second["project_context"].(map[string]any)
	secondResolution, _ := secondContext["task_resolution"].(map[string]any)
	if firstContext["context_revision"] == secondContext["context_revision"] || firstResolution["source_revision"] != secondResolution["source_revision"] {
		t.Fatalf("MCP context did not detect binding change independently from task revision: first=%+v second=%+v", firstContext, secondContext)
	}
	if first["server_version"] == "" || second["server_version"] == "" || first["server_instance_id"] == second["server_instance_id"] {
		t.Fatalf("connection version/instance was not exposed independently: first=%+v second=%+v", first, second)
	}
	raw, _ := json.Marshal(second)
	for _, root := range []string{parent, executor, clone} {
		if strings.Contains(string(raw), root) {
			t.Fatalf("MCP binding context leaked root %q: %s", root, raw)
		}
	}
}

func TestMultiRepoAgentNegativeContextDeniesUnauthorizedAuthorityAndUnknownContract(t *testing.T) {
	parent, executor := t.TempDir(), t.TempDir()
	writeMultiRepoMCPProject(t, parent, "", "", 4)
	writeMultiRepoMCPProject(t, executor, "checkout", "in-progress", 4)
	opa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Input struct {
				ProjectID string `json:"project_id"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid input", http.StatusBadRequest)
			return
		}
		allow := input.Input.ProjectID == "" || input.Input.ProjectID == "proj.parent"
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"allow": allow}})
	}))
	t.Cleanup(opa.Close)
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "proj.parent", Explicit: map[string]string{"proj.executor": executor}})
	ts := httptest.NewServer(NewWithRootsAndPolicy(roots, NewPolicyGate(PolicyConfig{OPAURL: opa.URL, HTTPClient: opa.Client()})).Handler("", ""))
	t.Cleanup(ts.Close)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_mcp_context","arguments":{"project_id":"proj.parent","task_ref":"xref:proj.executor/spec:checkout"}}}`
	_, denied := post(t, ts, request)
	if denied.Error == nil && denied.Result["isError"] != true {
		t.Fatalf("unauthorized authority was resolved: %+v", denied)
	}
	deniedRaw, _ := json.Marshal(denied)
	for _, root := range []string{parent, executor} {
		if strings.Contains(string(deniedRaw), root) {
			t.Fatalf("denied task context leaked root %q: %s", root, deniedRaw)
		}
	}

	// A known project with an adopted schema newer than this engine's contract
	// fails explicitly instead of returning a misleading local fallback.
	unsupported := t.TempDir()
	writeMultiRepoMCPProject(t, unsupported, "checkout", "in-progress", 99)
	unknownRoots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "proj.parent", Explicit: map[string]string{"proj.executor": unsupported}})
	unknownServer := httptest.NewServer(NewWithRoots(unknownRoots).Handler("", ""))
	t.Cleanup(unknownServer.Close)
	_, result := post(t, unknownServer, request)
	contextResult := contextStructured(t, result)
	projectContext, _ := contextResult["project_context"].(map[string]any)
	resolution, _ := projectContext["task_resolution"].(map[string]any)
	if resolution["resolution_state"] != "unsupported-artifact-contract" {
		t.Fatalf("unsupported contract did not fail closed: %+v", projectContext)
	}
}

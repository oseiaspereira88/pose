package mcpserver

// MCP project-scope contract behavior (spec pose-mcp-project-scope-contract):
// R1 every pose_* tool advertises the same project_id schema; R2 unknown vs.
// ambiguous project selection surface as distinct structured errors; R3
// neither ever leaks the resolved filesystem root.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestQualifiedArtifactMCPReadinessAuthorizesEveryTarget(t *testing.T) {
	parent, child := t.TempDir(), t.TempDir()
	write := func(root, slug, body string) {
		t.Helper()
		dir := filepath.Join(root, ".pose", "specs")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "2026-09-21-"+slug+".md"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(parent, "consumer", "---\nslug: consumer\nstatus: draft\ndepends_on: xref:child/spec:producer\n---\n")
	write(parent, "producer", "---\nslug: producer\nstatus: done\n---\n")
	write(child, "producer", "---\nslug: producer\nstatus: done\n---\n")
	var allowChild atomic.Bool
	allowChild.Store(true)
	opa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var input struct {
			Input struct {
				Project string `json:"project_id"`
			} `json:"input"`
		}
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid", 400)
			return
		}
		allow := input.Input.Project == "parent" || allowChild.Load()
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"allow": allow}})
	}))
	t.Cleanup(opa.Close)
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "parent", Explicit: map[string]string{"child": child}})
	gate := NewPolicyGate(PolicyConfig{OPAURL: opa.URL, HTTPClient: opa.Client()})
	ts := httptest.NewServer(NewWithRootsAndPolicy(roots, gate).Handler("", ""))
	t.Cleanup(ts.Close)
	call := func() map[string]any {
		_, result := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_spec_readiness","arguments":{"project_id":"parent","slug":"consumer"}}}`)
		if result.Result["isError"] != false {
			t.Fatalf("MCP: %+v", result)
		}
		data, ok := result.Result["structuredContent"].(map[string]any)
		if !ok {
			t.Fatalf("no structured readiness: %+v", result)
		}
		return data
	}
	if got := call(); got["ready"] != true {
		t.Fatalf("authorized readiness: %+v", got)
	}
	allowChild.Store(false)
	got := call()
	raw, _ := json.Marshal(got)
	if got["ready"] != false || !strings.Contains(string(raw), "unauthorized-project") {
		t.Fatalf("authorization bypass: %s", raw)
	}
	if strings.Contains(string(raw), child) || strings.Contains(string(raw), parent) {
		t.Fatalf("root disclosure: %s", raw)
	}
}

func TestFederatedRoadmapToolAuthorizesEveryDependency(t *testing.T) {
	parent, child := t.TempDir(), t.TempDir()
	for _, root := range []string{parent, child} {
		if err := os.MkdirAll(filepath.Join(root, ".pose", "roadmaps"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(parent, ".pose", "policy"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".pose", "roadmaps", "program.md"), []byte("---\nslug: program\nstatus: active\nconsumes: xref:child/roadmap:source\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".pose", "policy", "federation.json"), []byte("{\"schema_version\":1,\"enabled\":true,\"adopted_at\":\"2026-09-24\",\"trusted_projects\":{}}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, ".pose", "roadmaps", "source.md"), []byte("---\nslug: source\nstatus: done\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	opa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var input struct {
			Input struct {
				Project string `json:"project_id"`
			} `json:"input"`
		}
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"allow": input.Input.Project == "parent"}})
	}))
	t.Cleanup(opa.Close)
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "parent", Explicit: map[string]string{"child": child}})
	ts := httptest.NewServer(NewWithRootsAndPolicy(roots, NewPolicyGate(PolicyConfig{OPAURL: opa.URL, HTTPClient: opa.Client()})).Handler("", ""))
	t.Cleanup(ts.Close)
	_, result := post(t, ts, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"pose_federated_roadmap_acceptance","arguments":{"project_id":"parent","slug":"program"}}}`)
	if result.Result["isError"] != false {
		t.Fatalf("MCP federated acceptance failed: %+v", result)
	}
	data, ok := result.Result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing structured acceptance result: %+v", result)
	}
	raw, _ := json.Marshal(data)
	if !strings.Contains(string(raw), "unauthorized-project") || !strings.Contains(string(raw), "ready\":false") {
		t.Fatalf("unauthorized dependency did not block acceptance: %s", raw)
	}
	if strings.Contains(string(raw), child) || strings.Contains(string(raw), parent) {
		t.Fatalf("federated result disclosed a filesystem root: %s", raw)
	}
	_, closeoutResult := post(t, ts, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"project_id":"parent","scope":"roadmap:program"}}}`)
	if closeoutResult.Result["isError"] != false {
		t.Fatalf("MCP federated closeout failed: %+v", closeoutResult)
	}
	closeout, ok := closeoutResult.Result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing structured closeout state: %+v", closeoutResult)
	}
	closeoutRaw, _ := json.Marshal(closeout)
	if closeout["terminal"] != false || !strings.Contains(string(closeoutRaw), "unauthorized-project") {
		t.Fatalf("closeout state did not preserve federated blockers: %s", closeoutRaw)
	}
	if strings.Contains(string(closeoutRaw), child) || strings.Contains(string(closeoutRaw), parent) {
		t.Fatalf("federated closeout disclosed a filesystem root: %s", closeoutRaw)
	}
}

// requestScopedTools act on an already-resolved request_id (spec
// pose-safe-validate-orchestration) and never call StoreFor — they are not
// "project-capable" in the R1 sense, the same way conductor_run_* is not.
var requestScopedTools = map[string]bool{
	"pose_validate_approve": true, "pose_validate_submit": true,
	"pose_validate_status": true, "pose_validate_cancel": true,
}

func TestProjectIDSchemaConsistencyAcrossCatalog(t *testing.T) {
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		if strings.HasPrefix(name, "conductor_") || requestScopedTools[name] {
			continue // no POSE store involved
		}
		schema, _ := def["inputSchema"].(map[string]any)
		props, _ := schema["properties"].(map[string]any)
		field, ok := props["project_id"].(map[string]any)
		if !ok {
			t.Errorf("tool %q has no project_id property in its schema", name)
			continue
		}
		if field["type"] != "string" {
			t.Errorf("tool %q project_id.type = %v, want string", name, field["type"])
		}
		if field["description"] != sharedProjectIDDescription {
			t.Errorf("tool %q project_id.description diverges from the shared contract: %v", name, field["description"])
		}
		required, _ := schema["required"].([]string)
		for _, r := range required {
			if r == "project_id" {
				t.Errorf("tool %q must not require project_id (a default is convenience only)", name)
			}
		}
	}
}

func TestUnknownProjectIDReturnsStructuredError(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha","project_id":"proj.ghost"}}}`)
	isErr, _ := out.Result["isError"].(bool)
	if !isErr {
		t.Fatal("unknown project_id must produce isError=true")
	}
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["error_code"] != "project_unknown" || sc["project_id"] != "proj.ghost" {
		t.Errorf("structuredContent = %+v, want error_code=project_unknown project_id=proj.ghost", sc)
	}
	remediation, _ := sc["remediation"].(map[string]any)
	if remediation["context_tool"] != "pose_mcp_context" || remediation["action"] != "reload_or_restart_connection" {
		t.Errorf("project_unknown remediation = %+v", remediation)
	}
	if _, ok := sc["available_project_ids"]; !ok {
		t.Errorf("project_unknown must include authorized alternatives: %+v", sc)
	}
}

func TestAmbiguousProjectSelectionReturnsStructuredError(t *testing.T) {
	// No default root configured: an empty project_id is ambiguous, not "the"
	// project — this must be distinguishable from project_unknown.
	roots := pose.NewRoots(pose.RootsConfig{})
	srv := NewWithRoots(roots)
	ts := httptest.NewServer(srv.Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{}}}`)
	isErr, _ := out.Result["isError"].(bool)
	if !isErr {
		t.Fatal("ambiguous project selection must produce isError=true")
	}
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["error_code"] != "project_ambiguous" || sc["reason"] != "no-default" {
		t.Errorf("structuredContent = %+v, want error_code=project_ambiguous reason=no-default", sc)
	}
}

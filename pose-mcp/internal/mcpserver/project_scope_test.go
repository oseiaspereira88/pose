package mcpserver

// MCP project-scope contract behavior (spec pose-mcp-project-scope-contract):
// R1 every pose_* tool advertises the same project_id schema; R2 unknown vs.
// ambiguous project selection surface as distinct structured errors; R3
// neither ever leaks the resolved filesystem root.

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

const sharedProjectIDDescription = "Optional project to scope the .pose root (multi-project); omit for the default root"

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

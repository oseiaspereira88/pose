package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestLegacyNewUsesAuthorizedDerivedIdentityForItsDefaultRoot(t *testing.T) {
	root := t.TempDir()
	specPath := filepath.Join(root, ".pose", "specs", "alpha", "spec.md")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte("---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n\n# Alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	expectedProject, err := pose.ResolveDefaultProjectID(root)
	if err != nil {
		t.Fatal(err)
	}

	server := New(pose.Store{Root: root})
	ts := httptest.NewServer(server.Handler("", ""))
	t.Cleanup(ts.Close)
	_, response := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:alpha"}}}`)
	if response.Error != nil || response.Result["isError"] == true {
		t.Fatalf("legacy New default-root read failed: error=%+v result=%v", response.Error, response.Result)
	}
	structured, _ := response.Result["structuredContent"].(map[string]any)
	readView, _ := structured["read_view"].(map[string]any)
	bindings, _ := readView["bindings"].([]any)
	if len(bindings) == 0 {
		t.Fatalf("MCP closeout omitted the default-root read-view binding: %v", structured)
	}
	first, _ := bindings[0].(map[string]any)
	if first["project_id"] != expectedProject {
		t.Fatalf("default-root binding project_id = %v, want derived/declared %q", first["project_id"], expectedProject)
	}
}

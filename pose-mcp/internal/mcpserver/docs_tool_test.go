package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func newDocsTestServer(t *testing.T, manifestJSON string) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	if manifestJSON != "" {
		path := filepath.Join(root, ".pose", "docs.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(manifestJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

func TestToolsCall_DocsState_NoManifestReturnsPresentFalse(t *testing.T) {
	ts := newDocsTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_docs_state","arguments":{}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["manifest_present"] != false {
		t.Fatalf("structuredContent = %+v, want manifest_present=false", sc)
	}
}

func TestToolsCall_DocsState_ReturnsLiveCheckResult(t *testing.T) {
	manifest := `{"schema_version":1,"roots":["docs"],"entries":[{"path":"docs/missing.md","doc_type":"reference"}]}`
	ts := newDocsTestServer(t, manifest)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_docs_state","arguments":{}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["manifest_present"] != true {
		t.Fatalf("structuredContent = %+v, want manifest_present=true", sc)
	}
	result, _ := sc["result"].(map[string]any)
	totals, _ := result["totals"].(map[string]any)
	if totals["errors"] != float64(1) {
		t.Fatalf("totals = %+v, want 1 error for the missing declared doc", totals)
	}
}

func TestToolsCall_DocsState_InCatalog(t *testing.T) {
	found := false
	for _, def := range toolDefinitions() {
		if def["name"] == "pose_docs_state" {
			found = true
			schema, _ := def["inputSchema"].(map[string]any)
			if schema["type"] != "object" {
				t.Fatalf("pose_docs_state schema = %+v", schema)
			}
		}
	}
	if !found {
		t.Fatal("pose_docs_state not found in toolDefinitions()")
	}
}

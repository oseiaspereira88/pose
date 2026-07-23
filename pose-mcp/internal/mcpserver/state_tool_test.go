package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func newStateTestServer(t *testing.T, withArtifact bool) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	if withArtifact {
		staleGeneratedAt := time.Now().UTC().AddDate(0, 0, -31).Format(time.RFC3339)
		content := "---\nschema_version: 1\ngenerated_at: " + staleGeneratedAt + "\nbaseline_commit: abc1234\n---\n\n" +
			"# Project State\n\n" +
			"## Resumo executivo\n<!-- state:curated -->\n\nProjeto de teste.\n\n" +
			"## Direção atual\n<!-- state:curated -->\n\nFoco em X.\n\n" +
			"## Specs & Roadmaps\n<!-- state:derived hash:" + pose.ContentHash12("- specs: total=1") + " -->\n\n- specs: total=1\n\n" +
			"## Follow-ups\n<!-- state:derived hash:" + pose.ContentHash12("- abertos: 0") + " -->\n\n- abertos: 0\n"
		path := filepath.Join(root, ".pose", "state", "project-state.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

func TestToolsCall_ProjectState_NotInitialized(t *testing.T) {
	ts := newStateTestServer(t, false)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_project_state","arguments":{}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["initialized"] != false {
		t.Fatalf("structuredContent = %+v, want initialized=false", sc)
	}
}

func TestToolsCall_ProjectState_ReturnsSectionsAndStaleness(t *testing.T) {
	ts := newStateTestServer(t, true)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_project_state","arguments":{}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %v", sc["schema_version"])
	}
	sections, _ := sc["sections"].([]any)
	if len(sections) != 4 {
		t.Fatalf("sections = %d, want 4: %+v", len(sections), sc)
	}
	staleness, _ := sc["staleness"].(map[string]any)
	if staleness["stale"] != true || staleness["reason"] != "age_exceeded" {
		// A fixture stamped 2026-07-23T10:00:00Z is beyond the default 7-day
		// window as soon as this test runs — proves the field is live, not
		// frozen at generation time.
		t.Fatalf("staleness = %+v, want stale by age", staleness)
	}
}

func TestToolsCall_ProjectState_SectionFilter(t *testing.T) {
	ts := newStateTestServer(t, true)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_project_state","arguments":{"section":"Follow-ups"}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	sections, _ := sc["sections"].([]any)
	if len(sections) != 1 {
		t.Fatalf("sections = %+v, want exactly 1", sections)
	}
	sec, _ := sections[0].(map[string]any)
	if sec["name"] != "Follow-ups" {
		t.Fatalf("filtered section = %+v", sec)
	}

	_, badOut := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_project_state","arguments":{"section":"Does Not Exist"}}}`)
	if badOut.Result["isError"] != true {
		t.Fatalf("unknown section must be a tool error, got %+v", badOut.Result)
	}
}

func TestToolsCall_ProjectState_InCatalog(t *testing.T) {
	found := false
	for _, def := range toolDefinitions() {
		if def["name"] == "pose_project_state" {
			found = true
			schema, _ := def["inputSchema"].(map[string]any)
			props, _ := schema["properties"].(map[string]any)
			if _, ok := props["section"]; !ok {
				t.Error("pose_project_state schema is missing the section property")
			}
		}
	}
	if !found {
		t.Fatal("pose_project_state not advertised in the tool catalog")
	}
}

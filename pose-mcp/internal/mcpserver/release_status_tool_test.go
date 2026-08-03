package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestReleaseStatusToolReadsProjectScopedLedger(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/changelogs/unreleased/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAlpha.\n")
	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":95,"method":"tools/call","params":{"name":"pose_release_status","arguments":{}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("release status tool failed: %+v %v", out.Error, out.Result)
	}
	structured := out.Result["structuredContent"].(map[string]any)
	if pending, ok := structured["pending"].([]any); !ok || len(pending) != 1 {
		t.Fatalf("pending projection=%v", structured)
	}
}

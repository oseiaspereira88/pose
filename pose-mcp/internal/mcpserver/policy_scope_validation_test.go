package mcpserver

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestPolicyScopeMCPRefusesInvalidConfiguration(t *testing.T) {
	for _, withState := range []bool{false, true} {
		root := t.TempDir()
		path := filepath.Join(root, ".pose/policy/review.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"schema_version":2,"enabled":true,"reviewer_independence":{"spce":"different-actor"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		if withState {
			path = filepath.Join(root, ".pose/state/project-state.md")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("# Project State\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
		t.Cleanup(ts.Close)
		_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_project_state","arguments":{}}}`)
		if out.Result["isError"] != true || !strings.Contains(fmt.Sprint(out.Result), `reviewer_independence["spce"]`) {
			t.Fatalf("invalid policy silently hidden: %+v", out.Result)
		}
		if _, ok := out.Result["effective_governance"]; ok {
			t.Fatal("invalid policy reported protection")
		}
	}
}

package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-state-attention: the MCP tool returns the same projection as the
// CLI for the same tree, with coverage and attention.
func TestToolsCall_Obligations_ReturnsReportWithCoverage(t *testing.T) {
	root := t.TempDir()
	for rel, body := range map[string]string{
		".pose/specs/2026-10-01-base.md":     "---\nslug: base\nstatus: draft\n---\n\n# Spec: base\n",
		".pose/specs/2026-10-02-consumer.md": "---\nslug: consumer\nstatus: draft\ndepends_on: base\n---\n\n# Spec: consumer\n",
	} {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_obligations","arguments":{"kind":"dependency"}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	obligations, _ := sc["obligations"].([]any)
	coverage, _ := sc["coverage"].([]any)
	if len(obligations) != 1 || len(coverage) == 0 || sc["complete"] != false || sc["attention"] == nil {
		t.Fatalf("pose_obligations: %+v", sc)
	}
	direct, err := pose.Store{Root: root}.ProjectObligations(pose.ObligationQuery{Category: pose.ObligationDependency})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := obligations[0].(map[string]any)
	if len(direct.Obligations) != 1 || first["id"] != direct.Obligations[0].ID {
		t.Fatalf("MCP and the domain disagree on ids: %+v vs %+v", first, direct.Obligations)
	}
}

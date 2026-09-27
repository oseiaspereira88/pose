package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestABMAtomicStartMCPStatusIsReadOnly(t *testing.T) {
	root := t.TempDir()
	for path, body := range map[string]string{
		".pose/policy/review.json":       `{"schema_version":1,"enabled":false,"atomic_start_version":1}`,
		".pose/specs/2026-09-27-work.md": "---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: One.\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	server := New(pose.Store{Root: root})
	out, err := server.dispatch(context.Background(), "pose_start_status", json.RawMessage(`{"slug":"work"}`))
	if err != nil {
		t.Fatal(err)
	}
	result := out.(map[string]any)
	plan, _ := result["preview"].(pose.StartPlan)
	status, _ := result["status"].(pose.StartStatus)
	if plan.Digest == "" || status.Phase != "not-started" || !status.CapabilityAdopt {
		t.Fatalf("pose_start_status = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "starts")); !os.IsNotExist(err) {
		t.Fatal("the MCP status wrote start state")
	}
}

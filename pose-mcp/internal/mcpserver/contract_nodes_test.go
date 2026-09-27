package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestABMContractNodesMCPReturnsProjectionAndCapability(t *testing.T) {
	root := t.TempDir()
	for path, body := range map[string]string{
		".pose/policy/review.json":  `{"schema_version":1,"enabled":false}`,
		".pose/specs/nodes/spec.md": "---\nslug: nodes\nstatus: in-progress\ncreated_at: 2026-09-27\n---\n\n# Spec: nodes\n\n## 2. Requirements\n\n- R1: One.\n\n## 5. Decisions\n\n### Decision D1\n- Basis: R1\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	server := New(pose.Store{Root: root})
	out, err := server.dispatch(context.Background(), "pose_spec_amendments", json.RawMessage(`{"slug":"nodes"}`))
	if err != nil {
		t.Fatal(err)
	}
	result := out.(map[string]any)
	projection, ok := result["contract_nodes"].(pose.ContractNodesProjection)
	if !ok || len(projection.Nodes) != 2 || result["capability_adopted"] != false {
		t.Fatalf("pose_spec_amendments without capability: %+v", result)
	}
}

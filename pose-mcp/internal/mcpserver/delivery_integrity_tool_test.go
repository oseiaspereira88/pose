package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestDeliveryIntegrityToolReadsProjectScopedGraphAndReversePath(t *testing.T) {
	root := t.TempDir()
	graph := posemodel.DeliveryIntegrityGraph{SchemaVersion: 1, InputDigest: "sha256:test", Reverse: map[string][]string{"internal/core.go": {"alpha"}}, Nodes: []posemodel.DeliveryIntegrityNode{}, Edges: []posemodel.DeliveryIntegrityEdge{}, Claims: []posemodel.ArtifactClaim{}, ChangeSets: []posemodel.ChangeSet{}, Findings: []posemodel.DeliveryIntegrityFinding{}}
	raw, _ := json.Marshal(graph)
	path := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":93,"method":"tools/call","params":{"name":"pose_delivery_integrity","arguments":{"path":"internal/core.go"}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("delivery tool failed: error=%+v result=%v", out.Error, out.Result)
	}
	structured := out.Result["structuredContent"].(map[string]any)
	reverse := structured["reverse"].(map[string]any)
	if _, ok := reverse["internal/core.go"]; !ok {
		t.Fatalf("reverse lookup missing: %v", structured)
	}
}

func TestDeliveryIntegrityToolRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":94,"method":"tools/call","params":{"name":"pose_delivery_integrity","arguments":{"path":"../../etc/passwd"}}}`)
	if out.Error != nil || out.Result["isError"] != true {
		t.Fatalf("path escape did not fail closed: error=%+v result=%v", out.Error, out.Result)
	}
}

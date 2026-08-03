package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestSurfaceAssuranceToolUsesSharedProjectScopedGraph(t *testing.T) {
	root := t.TempDir()
	graph := posemodel.DeliveryIntegrityGraph{SchemaVersion: 1, InputDigest: "sha256:graph", ProvenanceDigest: "sha256:provenance", Nodes: []posemodel.DeliveryIntegrityNode{}, Edges: []posemodel.DeliveryIntegrityEdge{}, Claims: []posemodel.ArtifactClaim{}, ChangeSets: []posemodel.ChangeSet{}, Reverse: map[string][]string{}, Findings: []posemodel.DeliveryIntegrityFinding{}, Deliveries: []posemodel.DeliveryTarget{{Spec: "alpha", Ref: "surface:dashboard", Kind: "surface", ID: "dashboard"}}, Paths: map[string][]string{"surface:dashboard": {"spec:alpha", "delivery:surface:dashboard"}}}
	raw, _ := json.Marshal(graph)
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":95,"method":"tools/call","params":{"name":"pose_surface_assurance","arguments":{"ref":"surface:dashboard"}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("surface tool failed: error=%+v result=%v", out.Error, out.Result)
	}
	structured := out.Result["structuredContent"].(map[string]any)
	if len(structured["deliveries"].([]any)) != 1 || structured["provenance_digest"] != "sha256:provenance" {
		t.Fatalf("wrong projection: %v", structured)
	}
}

func TestSurfaceAssuranceToolRejectsInvalidTypedRef(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":96,"method":"tools/call","params":{"name":"pose_surface_assurance","arguments":{"ref":"../../secret"}}}`)
	if out.Error != nil || out.Result["isError"] != true {
		t.Fatalf("invalid ref accepted: error=%+v result=%v", out.Error, out.Result)
	}
}

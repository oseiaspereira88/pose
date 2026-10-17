package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// publishedIndexRoot publishes a whole set whose delivery graph is graph.
func publishedIndexRoot(t *testing.T, graph posemodel.DeliveryIntegrityGraph) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := graph.IndexJSON()
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string][]byte{}
	for _, name := range posemodel.IndexGenerationFiles {
		outputs[name] = []byte("{}\n")
	}
	outputs["delivery-integrity.json"] = append(raw, '\n')
	if _, err := (posemodel.Store{Root: root}).PublishIndexGeneration(outputs, nil); err != nil {
		t.Fatal(err)
	}
	return root
}

// The MCP tool and the CLI read the set through the same contract: a set that does
// not verify is an error result naming it, never an empty graph.
func TestDeliveryIntegrityToolReadsAPublishedGenerationAndRefusesAMixedOne(t *testing.T) {
	graph := posemodel.DeliveryIntegrityGraph{SchemaVersion: posemodel.DeliveryIntegritySchemaVersion, InputDigest: "sha256:mcp", Reverse: map[string][]string{"internal/core.go": {"alpha"}}}
	root := publishedIndexRoot(t, graph)
	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	call := `{"jsonrpc":"2.0","id":95,"method":"tools/call","params":{"name":"pose_delivery_integrity","arguments":{"path":"internal/core.go"}}}`

	_, out := post(t, ts, call)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("a published generation was refused: error=%+v result=%v", out.Error, out.Result)
	}
	structured := out.Result["structuredContent"].(map[string]any)
	if structured["input_digest"] != "sha256:mcp" {
		t.Fatalf("the tool did not return the generation's graph: %v", structured)
	}

	// Another writer replaces the file behind the descriptor: the set is mixed.
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "delivery-integrity.json"), []byte(`{"schema_version":1,"input_digest":"sha256:other"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".pose", "indexes", "generations")); err != nil {
		t.Fatal(err)
	}
	_, out = post(t, ts, call)
	if out.Error != nil || out.Result["isError"] != true {
		t.Fatalf("a mixed set was served as a graph: error=%+v result=%v", out.Error, out.Result)
	}
	raw, _ := json.Marshal(out.Result)
	if !strings.Contains(string(raw), "index-generation-unavailable") {
		t.Fatalf("the refusal does not name the unavailable set: %s", raw)
	}
}

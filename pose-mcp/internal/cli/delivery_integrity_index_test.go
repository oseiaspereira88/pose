package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// `pose index` writes the compact schema, and the store reads back what it wrote.
func TestIndexWritesTheCompactDeliveryIntegritySchema(t *testing.T) {
	root, _, _ := artifactGitFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdIndex(root, nil, &out, &errOut); code != 0 {
		t.Fatalf("index code=%d err=%s", code, errOut.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "delivery-integrity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var head struct {
		SchemaVersion int      `json:"schema_version"`
		Implied       []string `json:"implied_edges"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		t.Fatal(err)
	}
	if head.SchemaVersion != posemodel.DeliveryIntegrityIndexSchemaVersion || len(head.Implied) != 1 {
		t.Fatalf("schema=%d implied=%v", head.SchemaVersion, head.Implied)
	}
	graph, err := (posemodel.Store{Root: root}).GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if graph.SchemaVersion != posemodel.DeliveryIntegritySchemaVersion {
		t.Fatalf("expanded graph schema=%d", graph.SchemaVersion)
	}
	changes := 0
	for _, edge := range graph.Edges {
		if edge.Type == "changes" {
			changes++
		}
	}
	if changes == 0 {
		t.Fatal("the changes edges were not restored on read")
	}
}

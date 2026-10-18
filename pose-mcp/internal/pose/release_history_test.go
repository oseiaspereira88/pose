package pose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEveryHistoricalReleaseManifestOfThisRepositoryStillLoads(t *testing.T) {
	root, _ := filepath.Abs("../../..")
	entries, _ := os.ReadDir(filepath.Join(root, ".pose", "releases"))
	bad := 0
	for _, e := range entries {
		if !e.IsDir() || ValidateReleaseVersion(e.Name()) != nil {
			continue
		}
		if _, err := LoadReleaseManifest(root, e.Name()); err != nil {
			bad++
			t.Logf("%s: %v", e.Name(), err)
		}
	}
	if bad > 0 {
		t.Fatalf("%d historical manifests refused", bad)
	}
	t.Logf("%d releases read", len(entries))
}

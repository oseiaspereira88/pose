package pose

import (
	"os"
	"path/filepath"
	"strings"
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

func TestEveryReleaseTagOfThisRepositoryIsIntactForThisEngine(t *testing.T) {
	root, _ := filepath.Abs("../../..")
	out, err := gitOutput(root, "tag", "--list", "v*")
	if err != nil || strings.TrimSpace(out) == "" {
		t.Skip("this tree has no release tags to read")
	}
	for _, tag := range strings.Fields(out) {
		if ValidateReleaseVersion(tag) != nil {
			continue
		}
		snap, snapErr := ReadReleaseSnapshot(root, tag)
		if snapErr != nil {
			continue // a tag older than the governed releases holds no snapshot
		}
		if !snap.Intact() {
			t.Errorf("%s: a legitimate historical release is refused: issues=%v gaps=%v", tag, snap.Issues, snap.Gaps)
		}
	}
}

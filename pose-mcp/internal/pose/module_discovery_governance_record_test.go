package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-index-recovery-and-close-scope-consistency.

// R1: the governance record under .pose, at the root and inside a submodule or nested project, is
// not walked: it cannot make the walk exceed its bound, and it adds no module, README or file to
// the inventory, however many records it holds.
func TestModuleDiscoveryDoesNotWalkTheGovernanceRecord(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"service/go.mod": "module example.test/service\n", "README.md": "# project\n"})
	for _, dir := range []string{".pose", "nested/.pose"} {
		records := filepath.Join(root, dir, "review-bundles")
		if err := os.MkdirAll(records, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < moduleDiscoveryMaxEntries/2+200; i++ {
			if err := os.WriteFile(filepath.Join(records, fmt.Sprintf("rvb-%05d.json", i)), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(root, dir, "README.md"), []byte("# governance\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inventory, err := ScanModuleDiscoveryIn(NativeFiles(), root)
	if err != nil {
		t.Fatalf("a large governance record stopped the module discovery: %v", err)
	}
	var paths []string
	for _, m := range inventory.Manifests {
		paths = append(paths, m.Path)
	}
	if len(paths) != 1 || paths[0] != "service/go.mod" {
		t.Fatalf("manifests = %v, want the service's go.mod alone", paths)
	}
	for _, readme := range inventory.Readmes {
		if strings.Contains(readme, ".pose") {
			t.Fatalf("a README of the governance record is an inventory entry: %v", inventory.Readmes)
		}
	}
}

// R2: when the bound is exceeded by what is not governance record, the failure says what was
// counted and what to do, instead of the bare name of the budget.
func TestModuleDiscoveryNamesItsBoundWhenItIsExceeded(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"service/go.mod": "module example.test/service\n"})
	dir := filepath.Join(root, "generated")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < moduleDiscoveryMaxEntries+100; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%05d.txt", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := ScanModuleDiscoveryIn(NativeFiles(), root)
	if err == nil || !strings.Contains(err.Error(), "module discovery visited more than") || !strings.Contains(err.Error(), ".gitignore") {
		t.Fatalf("the bound is not named: %v", err)
	}
}

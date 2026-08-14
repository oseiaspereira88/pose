package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanModulesIgnoresQwenWorktrees(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "pose-mcp", "go.mod"),
		filepath.Join(root, ".qwen", "worktrees", "review", "pose-mcp", "go.mod"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	modules, manifests, _, _, _ := scanModules(root)
	if len(modules) != 1 || modules[0].Path != "pose-mcp" {
		t.Fatalf("modules = %#v, want only pose-mcp", modules)
	}
	if len(manifests) != 1 || manifests[0] != "pose-mcp/go.mod" {
		t.Fatalf("manifests = %#v, want only pose-mcp/go.mod", manifests)
	}
}

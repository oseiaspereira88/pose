package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_ModuleOverridesRootModule(t *testing.T) {
	root := t.TempDir()
	matrixDir := filepath.Join(root, ".pose", "indexes")
	if err := os.MkdirAll(matrixDir, 0o755); err != nil {
		t.Fatalf("mkdir .pose/indexes: %v", err)
	}

	// Create a root go.mod
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	matrix := `{
  "defaults": { "mode": "strict" },
  "stacks": {
    "go": {
      "checks": [
        { "name": "vet", "program": "go", "args": ["vet", "./..."], "severity": "optional" }
      ]
    }
  },
  "moduleOverrides": {
    ".": {
      "stack": "go",
      "mode": "strict"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(matrixDir, "validation-matrix.json"), []byte(matrix), 0o644); err != nil {
		t.Fatalf("write validation-matrix.json: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := cmdValidate(root, []string{}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("cmdValidate with moduleOverrides[\".\"] exited with code %d, stderr: %s", code, stderr.String())
	}

	if strings.Contains(stderr.String(), "outside the project") {
		t.Fatalf("moduleOverrides[\".\"] produced path outside project error: %s", stderr.String())
	}
}

func TestSanitizeGoCheckArgs_FiltersNodeModules(t *testing.T) {
	root := t.TempDir()
	nodeModulesDir := filepath.Join(root, "frontend", "node_modules", "somepkg")
	if err := os.MkdirAll(nodeModulesDir, 0o755); err != nil {
		t.Fatalf("mkdir node_modules: %v", err)
	}

	args := []string{"test", "./..."}
	sanitized := sanitizeGoCheckArgs(root, "go", args)

	// Since go list will return packages or empty list for temp dir, verify node_modules is not in args
	for _, arg := range sanitized {
		if strings.Contains(arg, "node_modules") {
			t.Fatalf("sanitizeGoCheckArgs failed to filter out node_modules: %v", sanitized)
		}
	}
}

func TestDiscoverValidationModules_IgnoresQwenWorktrees(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "pose-mcp", "go.mod"),
		filepath.Join(root, ".qwen", "worktrees", "review", "docs-site", "pyproject.toml"),
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

	modules, err := discoverValidationModules(root)
	if err != nil {
		t.Fatalf("discover validation modules: %v", err)
	}
	if len(modules) != 1 || modules[0].Rel != "pose-mcp" || modules[0].Stack != "go" {
		t.Fatalf("modules = %#v, want only pose-mcp Go module", modules)
	}
}

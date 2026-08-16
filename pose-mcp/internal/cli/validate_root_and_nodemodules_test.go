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

// TestDiscoverValidationModules_IgnoresFixtureDirectories regression-covers
// spec pose-fixture-directory-discovery-exclusion: a synthetic go.mod under
// an adoption-kit example's fixture/testdata directory was previously
// discovered as a real deliverable module, which recomputed pose-dist's own
// provenance digest and invalidated closed specs' review evidence on every
// reindex.
func TestDiscoverValidationModules_IgnoresFixtureDirectories(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "service", "go.mod"),
		filepath.Join(root, "examples", "kit", "fixture", "service", "go.mod"),
		filepath.Join(root, "examples", "kit", "fixtures", "other", "go.mod"),
		filepath.Join(root, "pose-mcp", "internal", "testdata", "sample", "go.mod"),
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
	if len(modules) != 1 || modules[0].Rel != "service" {
		t.Fatalf("modules = %#v, want only the real \"service\" module (fixture/fixtures/testdata excluded)", modules)
	}
}

// TestDiscoverValidationModules_ClassifiesAndroidSeparatelyFromJava
// regression-covers spec pose-android-stack-detection: a Gradle module
// carrying AndroidManifest.xml must classify as "android", not the generic
// "java" every other Gradle/Maven module gets.
func TestDiscoverValidationModules_ClassifiesAndroidSeparatelyFromJava(t *testing.T) {
	root := t.TempDir()
	androidManifest := filepath.Join(root, "app", "src", "main", "AndroidManifest.xml")
	if err := os.MkdirAll(filepath.Dir(androidManifest), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(androidManifest, []byte("<manifest/>\n"), 0o644); err != nil {
		t.Fatalf("write AndroidManifest.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "build.gradle.kts"), []byte("// android app\n"), 0o644); err != nil {
		t.Fatalf("write build.gradle.kts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "build.gradle"), []byte("// plain jvm backend\n"), 0o644); err != nil {
		t.Fatalf("write build.gradle: %v", err)
	}

	modules, err := discoverValidationModules(root)
	if err != nil {
		t.Fatalf("discover validation modules: %v", err)
	}
	stacks := map[string]string{}
	for _, m := range modules {
		stacks[m.Rel] = m.Stack
	}
	if stacks["app"] != "android" {
		t.Errorf("app module stack = %q, want \"android\"", stacks["app"])
	}
	if stacks["backend"] != "java" {
		t.Errorf("backend module stack = %q, want \"java\" (no AndroidManifest.xml)", stacks["backend"])
	}
}

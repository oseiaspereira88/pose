package pose_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestDiscoverComponent(t *testing.T) {
	dir := t.TempDir()
	compDir := filepath.Join(dir, "my-component")
	if err := os.MkdirAll(compDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create test file
	code := "package main\n\nfunc main() {\n  // TODO: test\n  println(\"hello\")\n}\n"
	if err := os.WriteFile(filepath.Join(compDir, "main.go"), []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	store := pose.Store{Root: dir}
	state, err := store.DiscoverComponent("my-component")
	if err != nil {
		t.Fatalf("DiscoverComponent failed: %v", err)
	}

	if state.ComponentSlug != "my-component" {
		t.Errorf("expected slug 'my-component', got %q", state.ComponentSlug)
	}
	if state.Metrics.LOCProduction <= 0 {
		t.Errorf("expected LOCProduction > 0, got %d", state.Metrics.LOCProduction)
	}
	if state.TechnicalDebt.TODOs != 1 {
		t.Errorf("expected 1 TODO, got %d", state.TechnicalDebt.TODOs)
	}

	// Save and load state
	if err := store.SaveComponentState(state); err != nil {
		t.Fatalf("SaveComponentState failed: %v", err)
	}

	loaded, err := store.LoadComponentState("my-component")
	if err != nil {
		t.Fatalf("LoadComponentState failed: %v", err)
	}

	if loaded.ComponentSlug != state.ComponentSlug {
		t.Errorf("loaded slug mismatch: %s vs %s", loaded.ComponentSlug, state.ComponentSlug)
	}
}

// TestGitIgnoredPathsReportsIgnoredDirectories regression-covers spec
// pose-discovery-gitignore-and-root-alias-fix.
func TestGitIgnoredPathsReportsIgnoredDirectories(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/vendored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "vendored"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "kept"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "add", ".gitignore").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	ignored := pose.GitIgnoredPaths(root)
	if !ignored["vendored/"] {
		t.Errorf("GitIgnoredPaths(%q) = %v, want \"vendored/\" present", root, ignored)
	}
	if ignored["kept/"] {
		t.Errorf("GitIgnoredPaths(%q) incorrectly marked kept/ as ignored: %v", root, ignored)
	}
}

// TestGitIgnoredPathsDegradesGracefullyOutsideAGitRepo confirms discovery
// never fails just because GitIgnoredPaths ran somewhere git can't resolve
// — it returns an empty set instead of an error.
func TestGitIgnoredPathsDegradesGracefullyOutsideAGitRepo(t *testing.T) {
	root := t.TempDir()
	ignored := pose.GitIgnoredPaths(root)
	if len(ignored) != 0 {
		t.Errorf("GitIgnoredPaths outside a git repo = %v, want empty", ignored)
	}
}

// TestFindComponentDirectoriesRespectsGitignore regression-covers spec
// pose-discovery-gitignore-and-root-alias-fix for the fallback walker
// FindComponentDirectories itself uses (not only cli's two walkers).
func TestFindComponentDirectoriesRespectsGitignore(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/vendored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "service"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "service", "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "vendored"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vendored", "go.mod"), []byte("module vendored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "add", ".gitignore").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	targets := (pose.Store{Root: root}).FindComponentDirectories()
	for _, target := range targets {
		if target == "vendored" {
			t.Errorf("FindComponentDirectories included gitignored \"vendored\": %v", targets)
		}
	}
	found := false
	for _, target := range targets {
		if target == "service" {
			found = true
		}
	}
	if !found {
		t.Errorf("FindComponentDirectories missed the real \"service\" module: %v", targets)
	}
}

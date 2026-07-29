package pose_test

import (
	"os"
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

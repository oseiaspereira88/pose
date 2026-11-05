package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestGovernanceWriteGuardIndexBuildRefusesBeforeOutputGeneration(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".pose", ".pose/specs", ".pose/roadmaps", ".pose/policy", ".pose/changelogs/unreleased", ".pose/releases"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	err := posemodel.WithGovernanceWriteGuard("test.hold-index-root", []posemodel.Store{{Root: root}}, func([]posemodel.Store) error {
		if code := cmdIndexIn(posemodel.NativeFiles(), root, nil, &stdout, &stderr); code != 1 {
			return fmt.Errorf("index under root contention returned %d", code)
		}
		if _, err := os.Lstat(filepath.Join(root, ".pose", "indexes")); !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("index output directory exists after refusal: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "governance-write-guard-busy") {
		t.Fatalf("root contention was not reported: %q", stderr.String())
	}
}

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

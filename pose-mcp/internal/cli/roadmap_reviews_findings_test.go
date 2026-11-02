package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func failDirectorySync(t *testing.T, suffix string) {
	t.Helper()
	posemodel.PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && strings.HasSuffix(path, suffix) {
			return errors.New("injected")
		}
		return nil
	}
	t.Cleanup(func() { posemodel.PublishSyncHook = nil })
}

func TestAnUnsyncedCloseSaysItClosed(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	failDirectorySync(t, "specs/bundle/spec.md")
	code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer")
	if code != 0 {
		t.Fatalf("a close that took effect reported failure: %d %s", code, out)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose/specs/bundle/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "status: done") {
		t.Fatalf("the transition did not take effect: %s", out)
	}
}

func TestAnUnsyncedRefreshDoesNotReadAsFailed(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose", "state", "project-state.md"), "---\nschema_version: 1\n---\n")
	failDirectorySync(t, "project-state.md")
	if err := stateRefreshConsumer(root, HookEvent{Kind: "spec-closed", Target: "x"}); err != nil {
		t.Fatalf("an unsynced refresh failed: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "refresh_pending: spec-closed") {
		t.Fatalf("a current state was marked pending: %s", raw)
	}
}

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
	if !strings.Contains(out, "transition-durability") {
		t.Fatalf("the close did not say its durability is unconfirmed: %s", out)
	}
}

func unsyncedRefreshFixture(t *testing.T, policy string) string {
	t.Helper()
	root := t.TempDir()
	if policy != "" {
		mustWrite(t, filepath.Join(root, ".pose", "policy", "state.json"), policy)
	}
	mustWrite(t, filepath.Join(root, ".pose", "state", "project-state.md"), "---\nschema_version: 1\n---\n")
	failDirectorySync(t, "project-state.md")
	return root
}

func TestAnUnsyncedRefreshDoesNotReadAsFailed(t *testing.T) {
	root := unsyncedRefreshFixture(t, "")
	if err := stateRefreshConsumer(root, HookEvent{Kind: "spec_closeout", Target: "x"}); err != nil {
		t.Fatalf("an unsynced refresh failed: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "refresh_pending: spec_closeout") {
		t.Fatalf("a current state was marked pending: %s", raw)
	}
	last, ok := lastStateRefresh(root)
	if !ok || last.Result != "unsynced" || !strings.Contains(last.Error, "publish-unsynced") {
		t.Fatalf("the refresh log says %+v", last)
	}
	store := posemodel.Store{Root: root}
	if history, err := os.ReadFile(store.StateHistoryPath()); err != nil || len(history) == 0 {
		t.Fatalf("the history of a published refresh is missing: %v", err)
	}
}

func TestAManualUnsyncedRefreshSaysSoAndSucceeds(t *testing.T) {
	root := unsyncedRefreshFixture(t, "")
	var out, errOut strings.Builder
	if code := cmdStateRefresh(root, nil, &out, &errOut); code != 0 {
		t.Fatalf("code %d: %s %s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String()+errOut.String(), "state.unsynced") {
		t.Fatalf("the refresh did not say so: %s", out.String())
	}
}

func TestAStrictPolicyRefusesAnUnsyncedRefresh(t *testing.T) {
	root := unsyncedRefreshFixture(t, `{"strict_refresh":true}`)
	if err := stateRefreshConsumer(root, HookEvent{Kind: "spec_closeout", Target: "x"}); !errors.Is(err, posemodel.ErrPublishUnsynced) {
		t.Fatalf("a strict policy accepted an unconfirmed refresh: %v", err)
	}
}

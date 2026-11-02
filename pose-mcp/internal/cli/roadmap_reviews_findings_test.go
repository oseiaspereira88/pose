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

func TestAStrictReplayDoesNotReadAsDurable(t *testing.T) {
	event := HookEvent{Kind: "spec_closeout", Target: "x"}
	root := unsyncedRefreshFixture(t, `{"strict_refresh":true}`)
	for i := 0; i < 2; i++ {
		if err := stateRefreshConsumer(root, event); !errors.Is(err, posemodel.ErrPublishUnsynced) {
			t.Fatalf("call %d accepted an unconfirmed refresh: %v", i, err)
		}
	}
	posemodel.PublishSyncHook = nil
	if err := stateRefreshConsumer(root, event); err != nil {
		t.Fatalf("a durable refresh was refused: %v", err)
	}
	if last, ok := lastStateRefresh(root); !ok || last.Result != "ok" {
		t.Fatalf("the refresh log says %+v", last)
	}
}

func TestAPolicyMadeStrictLaterAsksAgain(t *testing.T) {
	event := HookEvent{Kind: "spec_closeout", Target: "x"}
	root := unsyncedRefreshFixture(t, "")
	if err := stateRefreshConsumer(root, event); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "state.json"), `{"strict_refresh":true}`)
	if err := stateRefreshConsumer(root, event); !errors.Is(err, posemodel.ErrPublishUnsynced) {
		t.Fatalf("a strict policy accepted a replay: %v", err)
	}
}

func TestACombinedRefreshFailureSaysBoth(t *testing.T) {
	for name, path := range map[string]func(root string) string{
		"history": func(root string) string { return posemodel.Store{Root: root}.StateHistoryPath() },
		"log":     stateRefreshLogPath,
	} {
		t.Run(name, func(t *testing.T) {
			root := unsyncedRefreshFixture(t, "")
			if err := os.MkdirAll(path(root), 0o755); err != nil {
				t.Fatal(err)
			}
			var out, errOut strings.Builder
			if code := cmdStateRefresh(root, nil, &out, &errOut); code != 1 {
				t.Fatalf("code %d: %s", code, out.String())
			}
			said := out.String() + errOut.String()
			if !strings.Contains(said, "state.published") || !strings.Contains(said, "durability is unconfirmed") {
				t.Fatalf("a material outcome was lost: %s", said)
			}
			if err := stateRefreshConsumer(root, HookEvent{Kind: "spec_closeout", Target: "x"}); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
			if err != nil {
				t.Fatal(err)
			}
			marked := strings.Contains(string(raw), "refresh_pending: spec_closeout published, refresh log not appended, durability unconfirmed")
			if name == "log" && !marked {
				t.Fatalf("with its log gone a refresh told nobody: %s", raw)
			}
			if name == "history" && strings.Contains(string(raw), "refresh_pending: spec_closeout") {
				t.Fatalf("a current state was marked pending: %s", raw)
			}
		})
	}
}

func TestASkippedReplayIsNoDurabilityUnderAStrictPolicy(t *testing.T) {
	event := HookEvent{Kind: "spec_closeout", Target: "x"}
	root := unsyncedRefreshFixture(t, "")
	for i := 0; i < 2; i++ {
		if err := stateRefreshConsumer(root, event); err != nil {
			t.Fatal(err)
		}
	}
	if last, ok := lastStateRefresh(root); !ok || last.Result != "skipped" {
		t.Fatalf("the fixture did not replay: %+v", last)
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "state.json"), `{"strict_refresh":true}`)
	if err := stateRefreshConsumer(root, event); !errors.Is(err, posemodel.ErrPublishUnsynced) {
		t.Fatalf("a skipped replay stood for durability: %v", err)
	}
	posemodel.PublishSyncHook = nil
	if err := stateRefreshConsumer(root, event); err != nil {
		t.Fatal(err)
	}
	if err := stateRefreshConsumer(root, event); err != nil {
		t.Fatal(err)
	}
	if last, _ := lastStateRefresh(root); last.Result != "skipped" {
		t.Fatalf("a durable refresh was not deduplicated: %+v", last)
	}
}

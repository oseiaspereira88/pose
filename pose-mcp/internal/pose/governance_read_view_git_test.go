package pose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGovernanceReadViewGitCapturesAndAuditsExactCommand(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/project.json": `{"schema_version":1,"project_id":"proj.001","name":"git view fixture"}`, "source.go": "package source\n"})
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "git-test")
	first, err := captureGovernanceGit(store.fsys(), root, "git-test", "rev-parse", "--verify", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(first)) == "" || view.operation.roots[0].revision != strings.TrimSpace(string(first)) {
		t.Fatalf("source revision was not captured: output=%q binding=%q", first, view.operation.roots[0].revision)
	}
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte("package source\nfunc Changed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshotTestGit(t, root, "add", "source.go")
	snapshotTestGit(t, root, "commit", "-q", "-m", "advance head")
	replayed, err := captureGovernanceGit(store.fsys(), root, "git-test", "rev-parse", "--verify", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if string(replayed) != string(first) {
		t.Fatalf("repeated Git input escaped its captured generation: first=%q replayed=%q", first, replayed)
	}
	audit := view.audit()
	if !audit.ChangedProducers["git-test"] || len(audit.ChangedPaths) == 0 {
		t.Fatalf("Git mutation was not exposed by final audit: %+v", audit)
	}
}

func TestGovernanceReadViewRawGitAuditSurvivesCallerCancel(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.go": "package source\n"})
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "raw-git-context")
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := CaptureGovernanceRawHistoryGitContextIn(store.fsys(), ctx, root, "raw-git-context", 1024, "rev-parse", "--verify", "HEAD"); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	audit := view.audit()
	if audit.ChangedProducers["raw-git-context"] || len(audit.UnknownProducers["raw-git-context"]) != 0 {
		t.Fatalf("caller cancellation prevented the bounded raw-history audit rerun: %+v", audit)
	}
}

func TestGovernanceReadViewGitBrokenTagWarningIsUnknown(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.go": "package source\n"})
	brokenRef := filepath.Join(root, ".git", "refs", "tags", "broken")
	if err := os.MkdirAll(filepath.Dir(brokenRef), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(brokenRef, []byte("not-an-object-id\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "release-status")
	if _, err := captureReleaseTagBindings(store.fsys(), root, "release-status"); err == nil {
		t.Fatal("a successful tag listing with a broken-ref warning was treated as complete")
	}
	metadata, _ := view.metadata(store.FederatedProjectID, "", "", view.audit())
	if metadata.Freshness != ReadViewFreshnessUnknown || len(metadata.Limitations) == 0 {
		t.Fatalf("broken release ref did not make the read view unknown: %+v", metadata)
	}
}

func TestGovernanceReadViewGitReservesSharedBudgetBeforeRetainingOutput(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.go": "package source\n"})
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "git-budget")

	shim := t.TempDir()
	started := filepath.Join(t.TempDir(), "started")
	release := filepath.Join(t.TempDir(), "release")
	gitShim := filepath.Join(shim, "git")
	script := "#!/bin/sh\n: > \"$GOVERNANCE_GIT_STARTED\"\nwhile [ ! -e \"$GOVERNANCE_GIT_RELEASE\" ]; do sleep 0.01; done\nprintf '%064d' 0\n"
	if err := os.WriteFile(gitShim, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOVERNANCE_GIT_STARTED", started)
	t.Setenv("GOVERNANCE_GIT_RELEASE", release)
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))

	const competingInput = int64(64)
	view.operation.budgetMu.Lock()
	view.operation.inputBytes = governanceReadViewMaxInputBytes - competingInput
	view.operation.budgetMu.Unlock()

	type captureResult struct {
		out []byte
		err error
	}
	result := make(chan captureResult, 1)
	go func() {
		out, err := captureGovernanceGit(store.fsys(), root, "git-budget", "status")
		result <- captureResult{out: out, err: err}
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Git shim did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !view.operation.reserveInput(competingInput) {
		t.Fatal("the concurrent Files producer could not reserve its remaining bytes")
	}
	if err := os.WriteFile(release, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case captured := <-result:
		if !errors.Is(captured.err, errGovernanceReadViewBudget) {
			t.Fatalf("concurrent Git output was not refused at the shared budget: output=%q err=%v", captured.out, captured.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Git capture did not stop after the shared budget was exhausted")
	}

	view.operation.budgetMu.Lock()
	inputBytes := view.operation.inputBytes
	view.operation.budgetMu.Unlock()
	if inputBytes != governanceReadViewMaxInputBytes {
		t.Fatalf("operation input accounting = %d, want exactly %d", inputBytes, governanceReadViewMaxInputBytes)
	}
	key := governanceGitObservationKey(root, []string{"status"}, governanceReadViewMaxInputBytes)
	state := view.operation.git
	state.mu.Lock()
	observation := state.observations[key]
	state.mu.Unlock()
	if observation != nil {
		t.Fatal("over-budget Git output was retained as a complete observation")
	}
}

func TestGovernanceReadViewGitSwallowedFailureMarksProducerUnknown(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.go": "package source\n"})
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "git-failure")
	shim := t.TempDir()
	gitShim := filepath.Join(shim, "git")
	if err := os.WriteFile(gitShim, []byte("#!/bin/sh\necho unavailable >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))

	if paths, ok := commitChangedPathsGovernedIn(store.fsys(), root, "git-failure", strings.Repeat("a", 40)); ok || paths != nil {
		t.Fatalf("failed Git path inventory was treated as complete: paths=%v ok=%t", paths, ok)
	}
	audit := view.audit()
	if len(audit.UnknownProducers["git-failure"]) == 0 {
		t.Fatalf("swallowed Git failure did not disclose producer uncertainty: %+v", audit)
	}
}

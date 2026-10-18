package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-git-and-dispatch-snapshot-integrity: what the dispatch consumed is what was
// sealed, and a result that no longer governs its scope is history, not fulfilment.

func dispatchOnce(t *testing.T, root, adapter string) (int, string) {
	t.Helper()
	return runPose(t, root, "review", "dispatch", "spec:alpha", "--via", adapter, "--apply")
}

// V2 and V4 through the dispatch: a reviewer that hides an edit from Git, or that rewrites
// the validation matrix or the policy under a derived prefix, did not review the sealed content.
func TestDispatchJudgesTheContentNotWhatGitWasToldToIgnore(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"hider":  {Command: []string{"sh", "-c", "cat >/dev/null; git update-index --assume-unchanged pose-mcp/lib.go && echo changed >> pose-mcp/lib.go"}, Vendor: "test", Model: "h-1", Timeout: "30s"},
		"skip":   {Command: []string{"sh", "-c", "cat >/dev/null; git update-index --skip-worktree pose-mcp/lib.go && echo changed >> pose-mcp/lib.go"}, Vendor: "test", Model: "s-1", Timeout: "30s"},
		"matrix": {Command: []string{"sh", "-c", "cat >/dev/null; mkdir -p .pose/indexes && echo '{}' > .pose/indexes/validation-matrix.json"}, Vendor: "test", Model: "m-1", Timeout: "30s"},
		"policy": {Command: []string{"sh", "-c", "cat >/dev/null; mkdir -p .pose/results && echo x >> .pose/policy/review.json"}, Vendor: "test", Model: "p-1", Timeout: "30s"},
		"mover":  {Command: []string{"sh", "-c", "cat >/dev/null; mkdir -p .pose/state && git mv pose-mcp/lib.go .pose/state/lib.go"}, Vendor: "test", Model: "mv-1", Timeout: "30s"},
		"ok":     {Command: []string{"sh", "-c", "cat >/dev/null; mkdir -p .pose/state && echo {} > .pose/state/technical-debt.json; echo decision: approved"}, Vendor: "test", Model: "ok-1", Timeout: "30s"},
	})
	for adapter, want := range map[string]string{"hider": "pose-mcp/lib.go", "skip": "pose-mcp/lib.go", "matrix": "validation-matrix.json", "policy": "review.json", "mover": "pose-mcp/lib.go"} {
		code, out := dispatchOnce(t, root, adapter)
		if code == 0 || !strings.Contains(out, "review_dispatch.status=failed") || !strings.Contains(out, "changed files in the sealed worktree") || !strings.Contains(out, want) {
			t.Fatalf("%s: %d %s", adapter, code, out)
		}
	}
	if code, out := dispatchOnce(t, root, "ok"); code != 0 || !strings.Contains(out, "review_dispatch.status=completed") {
		t.Fatalf("the intact control stopped being eligible: %d %s", code, out)
	}
}

// V5: a policy changed while the run was going leaves the run as history.
func TestARunWhosePolicyChangedWhileItRanIsStale(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"racer": {Command: []string{"sh", "-c", "cat >/dev/null; echo '\n' >> '" + root + "/.pose/policy/review.json'; echo decision: approved"}, Vendor: "test", Model: "r-1", Timeout: "30s"},
	})
	code, out := dispatchOnce(t, root, "racer")
	if code == 0 || !strings.Contains(out, "review_dispatch.status=stale") || !strings.Contains(out, "policy changed") {
		t.Fatalf("%d %s", code, out)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Status != posemodel.ReviewRunStale || runs[0].Output == "" {
		t.Fatalf("the stale run must stay on the record as history: %+v", runs)
	}
}

// V6: a commit that touches none of the sealed inputs does not invalidate the run.
func TestAnUnrelatedCommitDuringTheRunDoesNotInvalidateIt(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"committer": {Command: []string{"sh", "-c", "cat >/dev/null; cd '" + root + "' && echo docs > UNRELATED.md && git add UNRELATED.md && git -c user.name=t -c user.email=t@t -c commit.gpgsign=false commit -q -m docs; echo decision: approved"}, Vendor: "test", Model: "c-1", Timeout: "30s"},
	})
	if code, out := dispatchOnce(t, root, "committer"); code != 0 || !strings.Contains(out, "review_dispatch.status=completed") {
		t.Fatalf("an independent documentation commit invalidated the run: %d %s", code, out)
	}
}

// V2: a checkout that does not materialize the sealed content is not handed to the reviewer.
func TestTheWorktreeIsComparedWithTheSealedCommitBeforeTheRun(t *testing.T) {
	root := dispatchInstance(t)
	if err := os.WriteFile(filepath.Join(root, ".git", "info", "attributes"), []byte("pose-mcp/lib.go filter=mangle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", root, "config", "filter.mangle.smudge", "sed s/package/PACKAGE/").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"ok": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "test", Model: "ok-1", Timeout: "30s"},
	})
	code, out := dispatchOnce(t, root, "ok")
	if code == 0 || !strings.Contains(out, "does not hold the sealed content") || !strings.Contains(out, "pose-mcp/lib.go") {
		t.Fatalf("a worktree that differs from the sealed commit was handed over: %d %s", code, out)
	}
}

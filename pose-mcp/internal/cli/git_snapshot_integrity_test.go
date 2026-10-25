package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func commitSnapshotFixture(t *testing.T, root string) {
	t.Helper()
	artifactGit(t, root, "add", "-A")
	artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture update")
}

func TestReviewDispatchOriginIsSemanticAndBoundToWorktree(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"ok": {Command: []string{"sh", "-c", "cat >/dev/null; echo reviewed"}, Vendor: "test", Model: "test"}})
	store := posemodel.Store{Root: root}
	plan, err := store.PlanReviewDispatch("spec:alpha", "review", "", "ok")
	if err != nil {
		t.Fatal(err)
	}
	// A separate clean worktree must never stand in for the selected origin.
	other := filepath.Join(t.TempDir(), "other")
	artifactGit(t, root, "worktree", "add", "--detach", other, "HEAD")
	t.Cleanup(func() { artifactGit(t, root, "worktree", "remove", "--force", other) })
	run, err := (posemodel.Store{Root: other}).RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != posemodel.ReviewRunFailed || !strings.Contains(run.Failure, "worktree identity") {
		t.Fatalf("other origin: %+v %v", run, err)
	}
	// Commit only an unrelated document, preserving the scope's attributed head.
	writeCloseoutCLIFile(t, root, "unrelated.md", "documentation only\n")
	artifactGit(t, root, "add", "unrelated.md")
	artifactGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "unrelated docs")
	run, err = store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != posemodel.ReviewRunCompleted {
		t.Fatalf("unrelated commit: %+v %v", run, err)
	}
	// An edit hidden from status still changes the material input in the origin.
	artifactGit(t, root, "update-index", "--assume-unchanged", "pose-mcp/lib.go")
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package tampered\n")
	run, err = store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != posemodel.ReviewRunFailed {
		t.Fatalf("hidden origin edit: %+v %v", run, err)
	}
}

func TestReviewDispatchOriginMutationDuringRunIsHistoricalFailure(t *testing.T) {
	root := dispatchInstance(t)
	// The path is a synthetic temp directory with no shell metacharacters.
	command := "cat >/dev/null; printf 'package changed\\n' > '" + filepath.Join(root, "pose-mcp/lib.go") + "'; echo reviewer-result"
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"edit": {Command: []string{"sh", "-c", command}, Vendor: "test", Model: "test"}})
	code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "edit", "--apply")
	if code == 0 || !strings.Contains(out, "review_dispatch.status=failed") {
		t.Fatalf("origin changed: %d %s", code, out)
	}
	runs, err := (posemodel.Store{Root: root}).ListReviewRuns("")
	if err != nil || len(runs) != 1 || !strings.Contains(runs[0].Output, "reviewer-result") || runs[0].TranscriptDigest == "" {
		t.Fatalf("history lost: %+v %v", runs, err)
	}
}

func TestReviewDispatchRejectsHiddenChangesAndArbitraryEvidence(t *testing.T) {
	for name, command := range map[string]string{
		"assume":  "git update-index --assume-unchanged pose-mcp/lib.go && echo changed >> pose-mcp/lib.go",
		"skip":    "git update-index --skip-worktree pose-mcp/lib.go && echo changed >> pose-mcp/lib.go",
		"ignored": "mkdir scratch && echo changed > scratch/ignored.txt",
		"results": "mkdir -p .pose/results && echo forged > .pose/results/arbitrary.json",
		"matrix":  "mkdir -p .pose/indexes && echo forged > .pose/indexes/validation-matrix.json",
		"bundle":  "mkdir -p .pose/review-bundles && echo forged > .pose/review-bundles/arbitrary.json",
	} {
		t.Run(name, func(t *testing.T) {
			root := dispatchInstance(t)
			writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"edit": {Command: []string{"sh", "-c", "cat >/dev/null; " + command}, Vendor: "test", Model: "test"}})
			code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "edit", "--apply")
			if code == 0 || !strings.Contains(out, "review_dispatch.status=failed") || !strings.Contains(out, "changed files") {
				t.Fatalf("accepted: %d %s", code, out)
			}
		})
	}
}

func TestGitSnapshotCloseoutFailsOnUnknownGitAndHiddenChanges(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	if err := os.Rename(filepath.Join(root, ".git"), filepath.Join(root, "saved-git")); err != nil {
		t.Fatal(err)
	}
	code, out := runPose(t, root, "close", "spec:bundle", "--plan", "--json")
	if code == 0 || !strings.Contains(out, "Git snapshot unavailable") {
		t.Fatalf("unknown was clean: %d %s", code, out)
	}
	if err := os.Rename(filepath.Join(root, "saved-git"), filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	artifactGit(t, root, "update-index", "--assume-unchanged", "pose-mcp/main.go")
	writeCloseoutCLIFile(t, root, "pose-mcp/main.go", "package changed\n")
	code, out = runPose(t, root, "close", "spec:bundle", "--plan")
	if code != 0 || !strings.Contains(out, "context=blocked") {
		t.Fatalf("hidden was clean: %d %s", code, out)
	}
}

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
		"ok":     {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "test", Model: "ok-1", Timeout: "30s"},
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
	if code == 0 || !strings.Contains(out, "review_dispatch.status=failed") || !strings.Contains(out, "policy changed") {
		t.Fatalf("%d %s", code, out)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Status != posemodel.ReviewRunFailed || runs[0].Output == "" {
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
	if code == 0 || !strings.Contains(out, "sealed") || !strings.Contains(out, "pose-mcp/lib.go") {
		t.Fatalf("a worktree that differs from the sealed commit was handed over: %d %s", code, out)
	}
}

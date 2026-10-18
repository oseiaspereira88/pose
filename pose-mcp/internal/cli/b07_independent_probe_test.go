package cli

import (
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestB07IndependentSourceChangedDuringRun(t *testing.T) {
	for _, script := range []string{"echo changed >> 'ROOT/pose-mcp/lib.go'", "sed -i s/works/changed/ 'ROOT/.pose/specs/alpha/spec.md'", "rm -rf 'ROOT/.git'"} {
		root := dispatchInstance(t)
		script = strings.ReplaceAll(script, "ROOT", root)
		writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; " + script + "; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
		code, out := dispatchOnce(t, root, "probe")
		t.Logf("mutation=%s code=%d output=%s", script, code, out)
		if strings.Contains(out, "review_dispatch.status=completed") {
			t.Errorf("source inputs lost or changed but run completed")
		}
	}
}
func TestB07IndependentPolicySymlink(t *testing.T) {
	root := dispatchInstance(t)
	p := filepath.Join(root, ".pose/policy/review.json")
	raw, _ := os.ReadFile(p)
	target := filepath.Join(t.TempDir(), "review.json")
	os.WriteFile(target, raw, 0644)
	os.Remove(p)
	os.Symlink(target, p)
	// Reseal: a symlink policy is a consumed input and its target is readable.
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(out)
	}
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; echo ' ' >> '" + target + "'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "review_dispatch.status=completed") {
		t.Error("changed consumed policy symlink target ignored")
	}
}
func TestB07IndependentBundleInputsNotCommit(t *testing.T) {
	root := dispatchInstance(t)
	p := filepath.Join(root, ".pose/specs/alpha/spec.md")
	raw, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(raw), "R1: works", "R1: sealed-new-requirement", 1)), 0644)
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(out)
	}
	current, e := (posemodel.Store{Root: root}).CurrentReviewBundle("spec:alpha")
	if e != nil || current == nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(filepath.Join(root, ".pose/review-bundles"))
	for _, entry := range entries {
		if entry.Name() != current.BundleID+".json" {
			os.Remove(filepath.Join(root, ".pose/review-bundles", entry.Name()))
		}
	}
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; cat .pose/specs/alpha/spec.md; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	runs, _ := (posemodel.Store{Root: root}).ListReviewRuns("")
	t.Logf("code=%d out=%s runs=%+v", code, out, runs)
	if strings.Contains(out, "review_dispatch.status=completed") && len(runs) > 0 && !strings.Contains(runs[len(runs)-1].Output, "sealed-new-requirement") {
		t.Error("reviewer consumed old commit requirements instead of sealed bundle inputs")
	}
}

func TestB07ARunDoesNotStartOnInputsThatAlreadyMoved(t *testing.T) {
	root := dispatchInstance(t)
	marker := filepath.Join(t.TempDir(), "started")
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; touch " + marker + "; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	store := posemodel.Store{Root: root}
	plan, err := store.PlanReviewDispatch("spec:alpha", "review", "", "probe")
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(root, "pose-mcp/lib.go")
	raw, _ := os.ReadFile(f)
	os.WriteFile(f, append(raw, []byte("// moved\n")...), 0o644)
	run, err := store.RunReviewDispatch(plan, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status == posemodel.ReviewRunCompleted || !strings.Contains(run.Failure, "not started") {
		t.Fatalf("a run planned against inputs that moved must not start: %+v", run)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the reviewer was started")
	}
}

// L01 of the second review: a submodule the bundle does not name is examined neither before nor
// after the run.
func TestB07ASubmoduleTheBundleDoesNotNameIsLeftAloneBeforeAndAfterTheRun(t *testing.T) {
	root := dispatchInstanceWith(t, func(root string, git func(args ...string) string) {
		// An unrelated pin whose repository this checkout never materializes.
		git("update-index", "--add", "--cacheinfo", "160000,1111111111111111111111111111111111111111,vendor/other")
	})
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	if !strings.Contains(out, "review_dispatch.status=completed") {
		t.Fatalf("a submodule that is not part of the bundle refused the run (%d): %s", code, out)
	}
}

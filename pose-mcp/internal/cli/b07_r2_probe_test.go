package cli

import (
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestB07R2UnreadableSourceStatus(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; printf broken > '" + root + "/.git/index'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d out=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("source Git status unknown but run completed")
	}
}
func TestB07R2PublishWindow(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	s := posemodel.Store{Root: root}
	p, e := s.PlanReviewDispatch("spec:alpha", "review", "", "probe")
	if e != nil {
		t.Fatal(e)
	}
	calls := 0
	now := func() time.Time {
		calls++
		if calls == 2 {
			os.WriteFile(filepath.Join(root, "pose-mcp/lib.go"), []byte("changed after check\n"), 0644)
		}
		return time.Now()
	}
	run, e := s.RunReviewDispatch(p, now)
	t.Logf("status=%s calls=%d err=%v", run.Status, calls, e)
	if e == nil && run.Status == posemodel.ReviewRunCompleted {
		t.Error("changed inputs at result publication retain completed effect")
	}
}
func TestB07R2SourceMode(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; git -C '" + root + "' config core.filemode false; chmod +x '" + root + "/pose-mcp/lib.go'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d out=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("source input mode changed but run completed")
	}
}
func TestB07R2PolicyChangeWithEqualDigest(t *testing.T) {
	root := dispatchInstance(t)
	dir := filepath.Join(root, ".pose/policy")
	os.WriteFile(filepath.Join(dir, "zz-a"), []byte("x\x00zz-z.json\x00{}"), 0644)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; printf x > '" + dir + "/zz-a'; printf '{}' > '" + dir + "/zz-z.json'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d out=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("policy file set changed without stale because hash framing is ambiguous")
	}
}

func TestB07R2SealedSubjectNotCheckout(t *testing.T) {
	root := dispatchInstance(t)
	if out, e := exec.Command("git", "-C", root, "update-index", "--assume-unchanged", "pose-mcp/lib.go").CombinedOutput(); e != nil {
		t.Fatalf("%s %v", out, e)
	}
	os.WriteFile(filepath.Join(root, "pose-mcp/lib.go"), []byte("package posemcp // sealed-new-subject\n"), 0644)
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatal(out)
	}
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; cat pose-mcp/lib.go; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	runs, e := (posemodel.Store{Root: root}).ListReviewRuns("")
	if e != nil || len(runs) == 0 {
		t.Fatalf("%v %s", e, out)
	}
	run := runs[len(runs)-1]
	t.Logf("code=%d status=%s adapterRead=%q", code, run.Status, run.Output)
	if run.Status == posemodel.ReviewRunCompleted && !strings.Contains(run.Output, "sealed-new-subject") {
		t.Error("bundle seals current implementation but adapter reads old committed implementation")
	}
}

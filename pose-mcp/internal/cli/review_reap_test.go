//go:build linux

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-review-run-reaps-lingering-reviewer-processes R1, R2, R4: a
// reviewer that ends while a command it started in a session of its own is
// still running leaves a run that completes, with that command ended before
// the cleanup and named in the run record.
func TestDispatchEndsWhatTheReviewerLeftRunning(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid is not available")
	}
	root := dispatchInstance(t)
	pidFile := filepath.Join(t.TempDir(), "leftover.pid")
	script := `cat >/dev/null; setsid sh -c 'echo $$ > "` + pidFile + `"; while :; do touch "$TMPDIR/leftover" 2>/dev/null; sleep 0.05; done' </dev/null >/dev/null 2>&1 & ` +
		`i=0; while [ ! -s "` + pidFile + `" ] && [ $i -lt 100 ]; do sleep 0.05; i=$((i+1)); done; printf 'decision: approved\n'`
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", script}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}}})
	code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake")
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the leftover never started: %v (%d %s)", err, code, out)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
		if stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil && strings.Contains(string(stat), ") Z ") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil && !strings.Contains(string(stat), ") Z ") {
		t.Fatalf("the process the reviewer left running survived the run: %s", stat)
	}
	runs, _, _ := (pose.Store{Root: root}).ReadReviewRuns("")
	if len(runs) != 1 {
		t.Fatalf("runs: %d (%d %s)", len(runs), code, out)
	}
	run := runs[0]
	if run.Status != pose.ReviewRunCompleted || run.Safety == nil || run.Safety.Operation.Outcome != pose.OutcomeCompleted {
		t.Fatalf("the run did not complete with its cleanup: status=%s failure=%s safety=%+v", run.Status, run.Failure, run.Safety)
	}
	if !strings.Contains(strings.Join(run.Safety.Limitations, "\n"), "process(es) the run left running") {
		t.Fatalf("the run record does not name the ended process: %v", run.Safety.Limitations)
	}
}

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	builtPoseOnce sync.Once
	builtPose     string
	builtPoseErr  error
)

// poseBinary builds the pose command once: a detached run and its waiters are
// separate processes.
func poseBinary(t *testing.T) string {
	t.Helper()
	builtPoseOnce.Do(func() {
		dir, err := os.MkdirTemp("", "pose-delegate-bin-")
		if err != nil {
			builtPoseErr = err
			return
		}
		builtPose = filepath.Join(dir, "pose")
		out, err := exec.Command("go", "build", "-o", builtPose, "../../cmd/pose").CombinedOutput()
		if err != nil {
			builtPoseErr = err
			t.Log(string(out))
		}
	})
	if builtPoseErr != nil {
		t.Fatal(builtPoseErr)
	}
	return builtPose
}

func runBuilt(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(poseBinary(t), args...)
	cmd.Dir = root
	out, err := cmd.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, string(out)
}

// slowApprovingAgent is a policy whose agent approves once its gate opens:
// the test decides how long a run stays running (spec
// pose-cli-test-wall-time R1).
func slowApprovingAgent(t *testing.T) (string, *gate) {
	t.Helper()
	g := newGate(t)
	return `{"schema_version": 2, "agents": {
  "slow": {"vendor": "test", "model": "s-1", "cli": "sh", "commands": {"read": ["sh", "-c", ` + strconv.Quote("cat >/dev/null; "+g.wait+"; echo decision: approved") + `]}}
}, "purposes": {"review": {"max_runs_per_bundle": 9}}}`, g
}

// R1, R2, R3, R8: --detach returns the attempt at once; every waiter is
// released by the run's end with the same code; a wait on a terminal run
// returns at once; a wait that times out exits 50.
func TestDelegateDetachAndWait(t *testing.T) {
	root := dispatchInstance(t)
	policy, held := slowApprovingAgent(t)
	writeDelegation(t, root, policy)
	started := time.Now()
	code, out := runBuilt(t, root, "delegate", "review", "spec:alpha", "--detach", "--json")
	var detached struct {
		Schema, Attempt, Log string
	}
	if code != 0 || json.Unmarshal([]byte(out), &detached) != nil || detached.Schema != delegateDetachSchema || detached.Attempt == "" {
		t.Fatalf("detach: %d %s", code, out)
	}
	if time.Since(started) > 30*time.Second {
		t.Fatalf("detach waited for the run: %s", time.Since(started))
	}
	codes := make([]int, 3)
	lines := make([]string, 3)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], lines[i] = runBuilt(t, root, "delegate", "wait", detached.Attempt, "--json")
		}(i)
	}
	// The waiters are waiting on a running run; then it finishes.
	time.Sleep(200 * time.Millisecond)
	held.open()
	wg.Wait()
	for i, c := range codes {
		if c != 0 || !strings.Contains(lines[i], `"decision":"approved"`) || strings.Count(strings.TrimSpace(lines[i]), "\n") != 0 {
			t.Fatalf("waiter %d: %d %s", i, c, lines[i])
		}
	}
	if code, out := runBuilt(t, root, "delegate", "wait", detached.Attempt); code != 0 || !strings.Contains(out, "state=completed") {
		t.Fatalf("wait on a terminal run: %d %s", code, out)
	}
	held.close()
	code, out = runBuilt(t, root, "delegate", "review", "spec:alpha", "--detach", "--reason", "a second look", "--json")
	if code != 0 || json.Unmarshal([]byte(out), &detached) != nil {
		t.Fatalf("second detach: %d %s", code, out)
	}
	if code, out := runBuilt(t, root, "delegate", "wait", detached.Attempt, "--timeout", "200ms", "--json"); code != 50 || !strings.Contains(out, `"state":"running"`) {
		t.Fatalf("timed-out wait: %d %s", code, out)
	}
	if code, out := runBuilt(t, root, "delegate", "ls", detached.Attempt); code != 0 || !strings.Contains(out, "heartbeat=") {
		t.Fatalf("ls of a running run: %d %s", code, out)
	}
	held.open()
	if code, _ := runBuilt(t, root, "delegate", "wait", detached.Attempt); code != 0 {
		t.Fatalf("second run: %d", code)
	}
	if code, _ := runBuilt(t, root, "delegate", "wait"); code != 2 {
		t.Fatalf("wait without a run: %d", code)
	}
	if status, _ := exec.Command("git", "-C", root, "status", "--porcelain", ".pose/review-private").Output(); len(status) != 0 {
		t.Fatalf("the supervisor's log is not ignored: %s", status)
	}
}

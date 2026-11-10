package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/version"
	"github.com/harne8/pose-mcp/internal/worktreeruntime"
)

// Spec pose-worktree-runtime-resident-cache.

type runtimeFixture struct {
	root string
	run  func(args ...string) (doc map[string]any, stdout string, code int)
	stop func()
}

func newRuntimeFixture(t *testing.T) *runtimeFixture {
	t.Helper()
	if !worktreeruntime.Supported() {
		t.Skip("no protected local endpoint on this platform")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("no git: %v %s", err, out)
	}
	dir, err := os.MkdirTemp(os.Getenv("TMPDIR"), "rth")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	t.Setenv(worktreeruntime.EnvDir, dir+"/r")
	original := runtimeSpawn
	runtimeSpawn = func(r string, idle time.Duration) error {
		go cmdRuntime(r, []string{"serve", "--idle", idle.String()}, io.Discard, io.Discard)
		return nil
	}
	t.Cleanup(func() { runtimeSpawn = original })
	f := &runtimeFixture{root: root}
	f.run = func(args ...string) (map[string]any, string, int) {
		var out, errb bytes.Buffer
		code := cmdRuntimeRun(root, append([]string{"--envelope"}, args...), &out, &errb)
		var doc map[string]any
		_ = json.Unmarshal(out.Bytes(), &doc)
		return doc, errb.String(), code
	}
	f.stop = func() { cmdRuntime(root, []string{"stop"}, io.Discard, io.Discard) }
	t.Cleanup(f.stop)
	return f
}

func (f *runtimeFixture) start(t *testing.T) {
	t.Helper()
	if code := cmdRuntime(f.root, []string{"start"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("start exited %d", code)
	}
}

// R2/R3: a hosted command run through the runtime answers what it answers standalone (the
// same exit code and output), says where it ran, and a command the runtime does not host, or a
// caller whose environment differs, runs standalone with the reason.
func TestAHostedCommandAnswersThroughTheRuntimeAsItDoesStandalone(t *testing.T) {
	f := newRuntimeFixture(t)
	standalone, _, _ := f.run("roadmap-check")
	if standalone["via"] != "standalone" || standalone["fallback_reason"] == nil {
		t.Fatalf("with no runtime: %v", standalone)
	}
	f.start(t)
	hosted, _, code := f.run("roadmap-check")
	if hosted["via"] != "runtime" || hosted["epoch"] == "" {
		t.Fatalf("through the runtime: %v", hosted)
	}
	if hosted["stdout"] != standalone["stdout"] || hosted["stderr"] != standalone["stderr"] || hosted["exit_code"] != standalone["exit_code"] || code != int(standalone["exit_code"].(float64)) {
		t.Fatalf("the answers differ:\nruntime:    %v\nstandalone: %v", hosted, standalone)
	}
	// A command the runtime does not host is not sent to it.
	if doc, _, _ := f.run("state"); doc["via"] != "standalone" || !strings.Contains(doc["fallback_reason"].(string), "not one the runtime hosts") {
		t.Fatalf("an unhosted command: %v", doc)
	}
	// A caller in another environment is refused by the runtime and runs standalone.
	t.Setenv("POSE_RUNTIME_TEST_MARKER", "different")
	if doc, _, _ := f.run("roadmap-check"); doc["via"] != "standalone" || !strings.Contains(doc["fallback_reason"].(string), "environment") {
		t.Fatalf("a different environment: %v", doc)
	}
}

// R4: a hosted command whose deadline passed, or whose answer did not come back, may have run:
// it is reported as unknown and not repeated standalone.
func TestACommandThatMayHaveRunIsNotRepeatedStandalone(t *testing.T) {
	f := newRuntimeFixture(t)
	f.start(t)
	original := runtimeHostedRun.Load()
	var ran atomic.Int64
	slow := hostedRunFunc(func(args []string, out, errb io.Writer) int {
		ran.Add(1)
		time.Sleep(400 * time.Millisecond)
		return (*original)(args, out, errb)
	})
	runtimeHostedRun.Store(&slow)
	defer func() {
		time.Sleep(600 * time.Millisecond)
		runtimeHostedRun.Store(original)
	}()
	doc, _, code := f.run("--timeout", "50ms", "roadmap-check")
	if code != 1 || doc["outcome"] != "unknown" || doc["via"] != "runtime" {
		t.Fatalf("a deadline: %v exit %d", doc, code)
	}
	if !strings.Contains(doc["stderr"].(string), "was not repeated here") {
		t.Fatalf("the report does not say it was not repeated: %v", doc["stderr"])
	}
	time.Sleep(600 * time.Millisecond)
	if ran.Load() != 1 {
		t.Fatalf("the command ran %d times: it was repeated", ran.Load())
	}
}

// The digest names what decides an answer and ignores what does not.
func TestTheEnvironmentDigestFollowsWhatDecidesAnAnswer(t *testing.T) {
	base := []string{"POSE_PROJECT_ROOTS=/a", "LC_ALL=C", "PATH=/usr/bin", "PWD=/x", "CLAUDECODE=1"}
	d := runtimeEnvDigest(base)
	if d != runtimeEnvDigest([]string{"PWD=/y", "CLAUDECODE=1", "PATH=/bin", "LC_ALL=C", "POSE_PROJECT_ROOTS=/a"}) {
		t.Fatal("the digest depends on the order or on variables that decide nothing")
	}
	for _, changed := range [][]string{
		{"POSE_PROJECT_ROOTS=/b", "LC_ALL=C", "CLAUDECODE=1"},
		{"POSE_PROJECT_ROOTS=/a", "LC_ALL=pt_BR", "CLAUDECODE=1"},
		{"POSE_PROJECT_ROOTS=/a", "LC_ALL=C"},
	} {
		if runtimeEnvDigest(changed) == d {
			t.Fatalf("the digest ignores a change that decides an answer: %v", changed)
		}
	}
	_ = context.Background
}

// R10: the runtime itself refuses what it does not host and a caller in another environment,
// before it starts anything, whatever the client filtered.
func TestTheRuntimeRefusesWhatItDoesNotHostAndAnotherEnvironment(t *testing.T) {
	f := newRuntimeFixture(t)
	f.start(t)
	id, err := worktreeruntime.ResolveIdentity(f.root, version.Version, runtimeBuild())
	if err != nil {
		t.Skip(err)
	}
	client := worktreeruntime.Client{Identity: id, Dir: worktreeruntime.DefaultDir()}
	good := runtimeEnvDigest(os.Environ())
	if r, err := client.Run(context.Background(), []string{"state"}, good, 0); err != nil || r.Outcome != worktreeruntime.OutcomeRefused {
		t.Fatalf("an unhosted command: %+v %v", r, err)
	}
	if r, err := client.Run(context.Background(), []string{"roadmap-check"}, "another", 0); err != nil || r.Outcome != worktreeruntime.OutcomeRefused || !strings.Contains(r.Reason, "environment") {
		t.Fatalf("another environment: %+v %v", r, err)
	}
	if r, err := client.Run(context.Background(), nil, good, 0); err != nil || r.Outcome != worktreeruntime.OutcomeRefused {
		t.Fatalf("no command: %+v %v", r, err)
	}
}

// A command left running after its deadline still counts as work in the runtime: the status
// shows it in flight, and a stop waits for it and reports what it abandoned.
func TestACommandRunningPastItsDeadlineStillCountsAsInFlight(t *testing.T) {
	f := newRuntimeFixture(t)
	f.start(t)
	original := runtimeHostedRun.Load()
	slow := hostedRunFunc(func(args []string, out, errb io.Writer) int {
		time.Sleep(700 * time.Millisecond)
		return (*original)(args, out, errb)
	})
	runtimeHostedRun.Store(&slow)
	defer func() {
		time.Sleep(900 * time.Millisecond)
		runtimeHostedRun.Store(original)
	}()
	f.run("--timeout", "50ms", "roadmap-check")
	id, err := worktreeruntime.ResolveIdentity(f.root, version.Version, runtimeBuild())
	if err != nil {
		t.Skip(err)
	}
	client := worktreeruntime.Client{Identity: id, Dir: worktreeruntime.DefaultDir()}
	status, err := client.Status(context.Background())
	if err != nil || status.InFlight < 1 {
		t.Fatalf("the command running past its deadline is not in flight: %+v %v", status, err)
	}
	stopped, err := client.Stop(context.Background(), 50*time.Millisecond)
	if err != nil || stopped.Abandoned < 1 {
		t.Fatalf("a stop did not report the running command abandoned: %+v %v", stopped, err)
	}
}

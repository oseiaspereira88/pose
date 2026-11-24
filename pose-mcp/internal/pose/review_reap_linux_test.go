//go:build linux

package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Review round 1, N3: a pid that no longer names a process of the run, as
// after reuse by an unrelated process, is never signalled.
func TestAPidThatLostTheRunsEntryIsNotKilled(t *testing.T) {
	other := exec.Command("sleep", "30")
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200 && reviewProcessName(other.Process.Pid) != "sleep"; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	t.Cleanup(func() { _ = other.Process.Kill(); _ = other.Wait() })
	entry := []byte(SealedCheckoutEnv + "=/nonexistent/run-checkout")
	if err := killPinnedMarkedProcess(other.Process.Pid, entry); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if zombie, err := reviewProcessZombie(other.Process.Pid); err != nil || zombie {
		t.Fatalf("a process without the run's entry was killed: zombie=%t %v", zombie, err)
	}
	// The same process carrying the entry is ended.
	marked := exec.Command("sleep", "30")
	marked.Env = append(marked.Environ(), string(entry))
	if err := marked.Start(); err != nil {
		t.Fatal(err)
	}
	// Until exec, /proc shows the parent's environment.
	for i := 0; i < 200; i++ {
		if ok, _ := reviewProcessMarked(marked.Process.Pid, entry); ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := killPinnedMarkedProcess(marked.Process.Pid, entry); err != nil {
		t.Fatal(err)
	}
	if err := marked.Wait(); err == nil {
		t.Fatal("the marked process was not ended")
	}
}

// opaqueProcess starts a process of this user whose environment cannot be
// read (it makes itself non-dumpable), or skips without python3.
func opaqueProcess(t *testing.T) *exec.Cmd {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}
	cmd := exec.Command("python3", "-c", "import ctypes,time; ctypes.CDLL(None).prctl(4,0,0,0,0); time.sleep(30)")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	for i := 0; i < 300; i++ {
		if _, err := reviewProcessMarked(cmd.Process.Pid, []byte("x")); err != nil && !reviewProcessGoneErr(err) {
			return cmd
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Skip("the process never became non-inspectable")
	return nil
}

// Review round 2, N2: a non-inspectable process that existed before the run
// is not the run's; one that started during it may be, and the scan fails
// instead of reporting the run's processes ended.
func TestANonInspectableProcessStartedDuringTheRunFailsTheScan(t *testing.T) {
	before := opaqueProcess(t)
	baseline := reviewOpaqueProcessBaseline()
	if _, ok := baseline[before.Process.Pid]; !ok {
		t.Fatalf("the baseline misses process %d", before.Process.Pid)
	}
	if _, opaque, err := reapSealedCheckoutProcesses("/nonexistent/run-checkout", baseline); err != nil || opaque == 0 {
		t.Fatalf("a process that predates the run failed the scan or was not counted: %d %v", opaque, err)
	}
	during := opaqueProcess(t)
	if _, _, err := reapSealedCheckoutProcesses("/nonexistent/run-checkout", baseline); err == nil || !strings.Contains(err.Error(), "started during the run") {
		t.Fatalf("process %d, started during the run, did not fail the scan: %v", during.Process.Pid, err)
	}
}

// Review round 6, N2: a descendant that leaves the group, starts its own
// session and clears its environment is still ended with the run.
func TestAnOrphanThatClearedItsEnvironmentIsEnded(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid is not available")
	}
	pidFile := filepath.Join(t.TempDir(), "orphan.pid")
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Command = []string{"sh", "-c", `cat >/dev/null; setsid env -i /bin/sh -c 'echo $$ > "$1"; while :; do sleep 0.05; done' sh "$1" </dev/null >/dev/null 2>&1 & ` +
			`i=0; while [ ! -s "$1" ] && [ $i -lt 100 ]; do sleep 0.05; i=$((i+1)); done; printf 'decision: approved\n'`, "sh", pidFile}
	})
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the orphan never started: %v", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if zombie, err := reviewProcessZombie(pid); err == nil && !zombie {
		t.Fatalf("the orphan that cleared its environment survived the run (status %s: %s)", f.run.Status, f.run.Failure)
	}
	if f.run.Status != ReviewRunCompleted || !strings.Contains(strings.Join(f.run.Safety.Limitations, "\n"), "(orphaned)") {
		t.Fatalf("the run did not complete naming the orphan: %s %s %v", f.run.Status, f.run.Failure, f.run.Safety.Limitations)
	}
	// Review round 9, N1: the recorded limitation describes the shim, not an
	// environment match.
	if !strings.Contains(strings.Join(f.run.Safety.Limitations, "\n"), "the review run shim ends every process left in the run's subtree") {
		t.Fatalf("the limitation does not describe the shim: %v", f.run.Safety.Limitations)
	}
	// Review round 7, N4: the record names the pid, never the process name
	// the reviewer chose.
	for _, limit := range f.run.Safety.Limitations {
		if strings.Contains(limit, "(orphaned)") && strings.Contains(limit, " sh ") {
			t.Fatalf("the recorded limitation carries a process name: %q", limit)
		}
	}
	if !strings.Contains(strings.Join(f.run.Safety.Limitations, "\n"), strconv.Itoa(pid)+" (orphaned)") {
		t.Fatalf("the recorded limitation does not name the pid: %v", f.run.Safety.Limitations)
	}
}

// waitGone waits for pid to end (or be a zombie) and reports whether it did.
func waitGone(pid int, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if zombie, err := reviewProcessZombie(pid); err != nil || zombie {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// Review round 8, N2: the shim's report never goes through a path the
// reviewer can write, and the reviewer does not inherit the report pipe.
func TestTheAdapterDoesNotInheritTheShimReport(t *testing.T) {
	fds := filepath.Join(t.TempDir(), "fds")
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Command = []string{"sh", "-c", `cat >/dev/null; ls /proc/$$/fd > "$1"; env | grep -c POSE_REVIEW_SHIM_REPORT >> "$1"; ls "$TMPDIR" >> "$1"; printf 'decision: approved\n'`, "sh", fds}
	})
	raw, err := os.ReadFile(fds)
	if err != nil || f.run.Status != ReviewRunCompleted {
		t.Fatalf("the probe did not run: %v %s %s", err, f.run.Status, f.run.Failure)
	}
	lines := strings.Fields(string(raw))
	for _, l := range lines {
		if l == "3" || strings.Contains(l, "shim-report") {
			t.Fatalf("the adapter sees the shim's report (%q): %s", l, raw)
		}
	}
}

// Review round 8, N3: a timeout asks the shim to end the run's subtree, so an
// orphan that cleared its environment does not outlive it.
func TestATimeoutEndsTheRunsOrphans(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid is not available")
	}
	pidFile := filepath.Join(t.TempDir(), "orphan.pid")
	_ = provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Timeout = "2s"
		a.Command = []string{"sh", "-c", `cat >/dev/null; setsid env -i /bin/sh -c 'echo $$ > "$1"; while :; do sleep 0.05; done' sh "$1" </dev/null >/dev/null 2>&1 & sleep 30`, "sh", pidFile}
	})
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the orphan never started: %v", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if !waitGone(pid, 2*time.Second) {
		t.Fatal("an orphan outlived the run's timeout")
	}
}

// Review round 8, N4: when the dispatcher ends, the shim ends the run's
// subtree instead of leaving the reviewer running behind a dead owner.
func TestTheShimEndsTheRunWhenItsDispatcherEnds(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid is not available")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	orphanPid, shimPid := filepath.Join(dir, "orphan.pid"), filepath.Join(dir, "shim.pid")
	adapter := filepath.Join(dir, "adapter.sh")
	if err := os.WriteFile(adapter, []byte("#!/bin/sh\nsetsid env -i /bin/sh -c 'echo $$ > \"$1\"; while :; do sleep 0.05; done' sh \"$1\" </dev/null >/dev/null 2>&1 &\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A stand-in dispatcher starts the shim and ends once the run is going.
	script := `export ` + ReviewShimReportEnv + `=$$; "$1" ` + ReviewShimArg + ` /bin/sh sh "$4" "$2" & echo $! > "$3"; ` +
		`i=0; while [ ! -s "$2" ] && [ $i -lt 100 ]; do sleep 0.05; i=$((i+1)); done; exit 0`
	// No output pipe: waiting for one would wait for the shim that inherits it.
	dispatcher := exec.Command("sh", "-c", script, "sh", self, orphanPid, shimPid, adapter)
	started := time.Now()
	if err := dispatcher.Run(); err != nil {
		t.Fatalf("the stand-in dispatcher failed: %v", err)
	}
	if time.Since(started) > 10*time.Second {
		t.Fatal("the stand-in dispatcher waited for the run")
	}
	raw, err := os.ReadFile(orphanPid)
	if err != nil {
		t.Fatalf("the run never started: %v", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	if !waitGone(pid, 5*time.Second) {
		t.Fatal("the run outlived its dispatcher")
	}
}

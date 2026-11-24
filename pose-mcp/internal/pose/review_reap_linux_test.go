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
}

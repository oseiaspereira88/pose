//go:build linux

package pose

import (
	"os/exec"
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

//go:build windows

package pose

import (
	"os/exec"
	"testing"
	"time"
)

// Spec pose-review-run-reaps-lingering-reviewer-processes, review round 13,
// N1: on Windows the reviewer's tree runs in a job object, and releasing it
// ends the descendants the reviewer started.
func TestTheReviewersJobEndsItsDescendants(t *testing.T) {
	// cmd starts ping as a child and waits for it: both must end with the job.
	cmd := exec.Command("cmd", "/c", "ping -n 60 127.0.0.1 >NUL")
	prepareReviewProcessTree(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := attachReviewProcessTree(cmd); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	started := time.Now()
	if err := releaseReviewProcessTree(cmd); err != nil {
		t.Fatalf("the job did not end its processes: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the reviewer outlived its job")
	}
	if time.Since(started) > 10*time.Second {
		t.Fatal("ending the job took too long")
	}
}

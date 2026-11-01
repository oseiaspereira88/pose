//go:build !windows

package pose

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
)

func TestProcessThatExitedBeforeItsReadIsGone(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux covered group only")
	}
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Open(fmt.Sprintf("/proc/%d/stat", cmd.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer stat.Close()
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	_, err = stat.Read(make([]byte, 64))
	if err == nil || !processGone(err) {
		t.Fatalf("a read of an exited process answered %v", err)
	}
}

func TestProcessGoneIsOnlyThoseTwoAnswers(t *testing.T) {
	if processGone(nil) {
		t.Fatal("no error is not a process gone")
	}
	for _, err := range []error{os.ErrPermission, syscall.EIO, errors.New("short read")} {
		if processGone(err) {
			t.Fatalf("%v read as a process gone", err)
		}
	}
	if !processGone(os.ErrNotExist) || !processGone(fmt.Errorf("read: %w", syscall.ESRCH)) {
		t.Fatal("an exited process was not read as gone")
	}
}

func TestInventoryHoldsWhileProcessesExit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux covered group only")
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				_ = exec.Command("true").Run()
			}
		}
	}()
	defer func() {
		close(stop)
		<-done
	}()
	for i := 0; i < 200; i++ {
		if err := verifyReviewProcessGroup(-1); err != nil {
			t.Fatal(err)
		}
	}
}

//go:build !windows

package pose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// isolateProcessGroup starts the adapter in its own process group, and makes
// the context's cancellation kill that whole group.
func isolateProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

// killProcessGroup ends whatever the adapter left running.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	if err != nil {
		return err
	}
	return verifyReviewProcessGroup(cmd.Process.Pid)
}
func openReviewAdapterFile(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
func reviewSetupCommand(ctx context.Context, source *exec.Cmd) *exec.Cmd {
	cmd := exec.CommandContext(ctx, source.Path, source.Args[1:]...)
	cmd.Env = source.Env
	isolateProcessGroup(cmd)
	cmd.WaitDelay = time.Second
	return cmd
}

// Verify the covered Linux group after SIGKILL. Zombies have terminated and
// cannot hold pipes; reaping is their parent's duty. Other Unix ports currently
// report signal delivery only and cannot require verified group termination.
func verifyReviewProcessGroup(pid int) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	deadline := time.Now().Add(time.Second)
	for {
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return fmt.Errorf("process-group termination could not be observed")
		}
		if len(entries) > 100000 {
			return fmt.Errorf("process inventory exceeds coverage")
		}
		alive := false
		for _, entry := range entries {
			if _, err := strconv.Atoi(entry.Name()); err != nil {
				continue
			}
			raw, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("process inventory incomplete")
			}
			end := strings.LastIndex(string(raw), ")")
			if end < 0 {
				return fmt.Errorf("process inventory malformed")
			}
			fields := strings.Fields(string(raw[end+1:]))
			if len(fields) < 3 {
				return fmt.Errorf("process inventory malformed")
			}
			group, err := strconv.Atoi(fields[2])
			if err != nil {
				return fmt.Errorf("process group malformed")
			}
			if group == pid && fields[0] != "Z" && fields[0] != "X" {
				alive = true
				break
			}
		}
		if !alive {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("process group remained alive after termination")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func reviewProcessGroupAvailable() error { return verifyReviewProcessGroup(-1) }
func openReviewRootFile(root *os.Root, path string) (*os.File, error) {
	return root.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}

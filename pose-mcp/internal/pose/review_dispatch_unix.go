//go:build !windows

package pose

import (
	"os/exec"
	"syscall"
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
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// detachProcess starts cmd in a session of its own, so the caller's terminal
// closing does not end it.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// stopDetached kills the detached run and everything in its session.
func stopDetached(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
	}
}

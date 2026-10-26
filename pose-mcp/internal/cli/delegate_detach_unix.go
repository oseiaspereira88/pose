//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
	"time"
)

// detachProcess starts cmd in a session of its own, so the caller's terminal
// closing does not end it.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// stopDetached asks the detached run to stop, as an interrupt does, so it
// kills its agent's process group and records the run; only a run that does
// not end within the grace is killed with its session. It returns once the
// run has exited.
func stopDetached(cmd *exec.Cmd, exited <-chan struct{}) {
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-exited:
		return
	case <-time.After(delegateStopGrace):
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Process.Kill()
	<-exited
}

//go:build windows

package cli

import (
	"os/exec"
	"syscall"
)

// detachProcess starts cmd detached from the caller's console.
func detachProcess(cmd *exec.Cmd) {
	const detachedProcess, newProcessGroup = 0x00000008, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | newProcessGroup}
}

// stopDetached kills the detached run; Windows has no interrupt to ask it
// to stop its agent first.
func stopDetached(cmd *exec.Cmd, exited <-chan struct{}) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		<-exited
	}
}

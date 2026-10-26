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

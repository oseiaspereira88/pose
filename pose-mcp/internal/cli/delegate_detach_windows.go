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

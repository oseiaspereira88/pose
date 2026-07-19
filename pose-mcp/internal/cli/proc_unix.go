//go:build unix

package cli

import (
	"os/exec"
	"syscall"
)

// setProcessGroup places the child in its own process group so cancellation
// can terminate descendants too (spec pose-validation-runtime-guardrails).
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup terminates the child's whole process group.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

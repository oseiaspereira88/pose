//go:build !unix

package cli

import "os/exec"

// setProcessGroup is a no-op where process groups are unsupported; the
// platform limitation is documented in the guardrails ADR.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup falls back to killing the direct child only.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

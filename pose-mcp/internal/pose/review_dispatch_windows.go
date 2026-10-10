//go:build windows

package pose

import "os/exec"

// On Windows the context kills the adapter's process; WaitDelay releases the
// pipes its children may still hold.
func isolateProcessGroup(cmd *exec.Cmd) {}

func killProcessGroup(cmd *exec.Cmd) {}

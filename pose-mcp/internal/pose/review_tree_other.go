//go:build !windows

package pose

import "os/exec"

// On Unix the process group, and on Linux the review run shim, end the
// reviewer's tree; these are the Windows job object's hooks.
func prepareReviewProcessTree(*exec.Cmd)       {}
func attachReviewProcessTree(*exec.Cmd) error  { return nil }
func releaseReviewProcessTree(*exec.Cmd) error { return nil }

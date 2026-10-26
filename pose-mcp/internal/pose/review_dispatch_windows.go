//go:build windows

package pose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// On Windows the context kills the adapter's process; WaitDelay releases the
// pipes its children may still hold.
func isolateProcessGroup(cmd *exec.Cmd) {}

func killProcessGroup(cmd *exec.Cmd) error                { return nil }
func openReviewAdapterFile(path string) (*os.File, error) { return os.Open(path) }
func reviewSetupCommand(ctx context.Context, source *exec.Cmd) *exec.Cmd {
	cmd := exec.CommandContext(ctx, source.Path, source.Args[1:]...)
	cmd.Env = source.Env
	return cmd
}

func reviewProcessGroupAvailable() error                              { return fmt.Errorf("process-group termination unsupported") }
func openReviewRootFile(root *os.Root, path string) (*os.File, error) { return root.Open(path) }

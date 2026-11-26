//go:build !linux

package pose

import (
	"os/exec"
	"time"
)

// readOnlyIndexFor reads a private copy where no memory file exists.
func readOnlyIndexFor(cmd *exec.Cmd, raw []byte, modTime time.Time) (func(), error) {
	return privateIndexCopy(cmd, raw, modTime)
}

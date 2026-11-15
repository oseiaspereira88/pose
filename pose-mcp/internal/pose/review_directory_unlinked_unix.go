//go:build unix

package pose

import (
	"os"
	"syscall"
)

// reviewDirectoryUnlinked reports a directory, read through a descriptor
// held across its removal, that is no longer linked anywhere (spec
// pose-review-cleanup-removes-go-build-leftovers).
func reviewDirectoryUnlinked(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Nlink == 0
}

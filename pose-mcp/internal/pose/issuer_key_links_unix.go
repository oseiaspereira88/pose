//go:build !windows

package pose

import (
	"os"
	"syscall"
)

// extraHardLinks reports whether a file has more than one directory entry.
func extraHardLinks(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Nlink > 1
}

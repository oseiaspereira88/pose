//go:build linux

package mcpserver

import (
	"fmt"
	"os"
	"syscall"
)

// fileIdentity adds what a metadata-preserving edit cannot restore: the inode and
// the status-change time, which any write or utimes call moves even when the
// modification time is put back.
func fileIdentity(info os.FileInfo) string {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("#%d.%d:%d", st.Dev, st.Ino, st.Ctim.Nano())
	}
	return ""
}

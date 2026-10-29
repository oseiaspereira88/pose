//go:build !windows

package pose

import (
	"os"
	"syscall"
)

// stateDirOwner reports who owns a directory of the task-state path, and
// whether that is the user (spec pose-delegation-task-detach-and-state R6,
// R7); root also counts for the base and its ancestors.
func stateDirOwner(info os.FileInfo) (user, root bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false, false
	}
	return int(st.Uid) == os.Getuid(), st.Uid == 0
}

// stateDirWritableByOthers reports whether a group or other user could
// rename the entries of a directory: write permission, which the sticky bit
// excuses only where sticky counts (the base and its ancestors, R6).
func stateDirWritableByOthers(info os.FileInfo, sticky bool) bool {
	return info.Mode().Perm()&0o022 != 0 && !(sticky && info.Mode()&os.ModeSticky != 0)
}

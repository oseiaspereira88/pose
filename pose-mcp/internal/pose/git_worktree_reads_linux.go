//go:build linux

package pose

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

// readOnlyIndexFor hands raw to cmd as an anonymous memory file. Git
// resolves the link /proc/self/fd/N to "/memfd:pose-index/none (deleted)"
// and tries to create its lock next to it, in a directory that does not
// exist, so the lock fails for any user, root included, and the refresh is
// not written. No file is created.
func readOnlyIndexFor(cmd *exec.Cmd, raw []byte, modTime time.Time) (func(), error) {
	fd, err := memfdCreate("pose-index/none", unix.MFD_CLOEXEC)
	if err != nil {
		// No memory file here (an old kernel, a seccomp filter): the private
		// copy keeps the index untouched too (review round 5, N27).
		return privateIndexCopy(cmd, raw, modTime)
	}
	f := os.NewFile(uintptr(fd), "pose-index")
	if _, err := memfdWrite(f, raw); err != nil {
		f.Close()
		return privateIndexCopy(cmd, raw, modTime) // review round 1, N8
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return privateIndexCopy(cmd, raw, modTime)
	}
	ts := unix.NsecToTimespec(modTime.UnixNano())
	if err := utimesNanoAt(unix.AT_FDCWD, fmt.Sprintf("/proc/self/fd/%d", fd), []unix.Timespec{ts, ts}, 0); err != nil {
		// Without /proc the memory file cannot carry the time, nor be named
		// to Git: the private copy is read instead (review round 5, N27).
		f.Close()
		return privateIndexCopy(cmd, raw, modTime)
	}
	cmd.ExtraFiles = append(cmd.ExtraFiles, f)
	cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_INDEX_FILE=/proc/self/fd/%d", 2+len(cmd.ExtraFiles)))
	return func() { f.Close() }, nil
}

// memfdCreate and utimesNanoAt are the calls the memory file needs; a test
// replaces them to exercise the fallback (review round 1, N4).
var (
	memfdCreate  = unix.MemfdCreate
	utimesNanoAt = unix.UtimesNanoAt
	memfdWrite   = func(f *os.File, b []byte) (int, error) { return f.Write(b) }
)

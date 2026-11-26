//go:build linux

package pose

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// readOnlyIndexFor hands raw to cmd as an anonymous memory file. Git
// resolves the link /proc/self/fd/N to "/memfd:pose-index/none (deleted)"
// and tries to create its lock next to it, in a directory that does not
// exist, so the lock fails for any user, root included, and the refresh is
// not written. No file is created.
func readOnlyIndexFor(cmd *exec.Cmd, raw []byte) (func(), error) {
	fd, err := unix.MemfdCreate("pose-index/none", unix.MFD_CLOEXEC)
	if err != nil {
		return func() {}, fmt.Errorf("pose: an in-memory index cannot be made: %w", err)
	}
	f := os.NewFile(uintptr(fd), "pose-index")
	if _, err := f.Write(raw); err != nil {
		f.Close()
		return func() {}, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return func() {}, err
	}
	cmd.ExtraFiles = append(cmd.ExtraFiles, f)
	cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_INDEX_FILE=/proc/self/fd/%d", 2+len(cmd.ExtraFiles)))
	return func() { f.Close() }, nil
}

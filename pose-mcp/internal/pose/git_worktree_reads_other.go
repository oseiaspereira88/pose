//go:build !linux

package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// readOnlyIndexFor hands raw to cmd as a private copy: Git refreshes and may
// rewrite the copy, never the checkout's index.
func readOnlyIndexFor(cmd *exec.Cmd, raw []byte, modTime time.Time) (func(), error) {
	dir, err := os.MkdirTemp("", "pose-index-*")
	if err != nil {
		return func() {}, err
	}
	done := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "index")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		done()
		return func() {}, err
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		done()
		return func() {}, err
	}
	// A split index would write a new shared index into .git.
	cmd.Args = append([]string{cmd.Args[0], "-c", "core.splitIndex=false"}, cmd.Args[1:]...)
	cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+path)
	return done, nil
}

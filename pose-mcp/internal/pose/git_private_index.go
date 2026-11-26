package pose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Spec pose-validate-reads-leave-the-index-alone R1, D1: `git diff` against
// the work tree refreshes the index's stat cache and writes it back even with
// GIT_OPTIONAL_LOCKS=0, and diff.autoRefreshIndex=false would change its
// answer (a file whose stat alone changed would be listed). Such a diff runs
// over a private copy of the index instead: Git refreshes and rewrites the
// copy, the answer is the same, and the checkout's .git/index is untouched.

// privateIndexLimit bounds the index copied for one read.
const privateIndexLimit = 512 << 20

// WithPrivateIndex prepares cmd, a git command reading the repository at
// root, to use a private copy of root's index. The returned function removes
// the copy; call it after the command ends.
func WithPrivateIndex(cmd *exec.Cmd, root string) (func(), error) {
	out, err := RawGit("-C", root, "rev-parse", "--git-path", "index").Output()
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be located: %w", root, err)
	}
	index := strings.TrimSuffix(string(out), "\n")
	if !filepath.IsAbs(index) {
		index = filepath.Join(root, index)
	}
	dir, err := os.MkdirTemp("", "pose-index-*")
	if err != nil {
		return func() {}, err
	}
	done := func() { _ = os.RemoveAll(dir) }
	copyPath := filepath.Join(dir, "index")
	raw, err := readBoundedRegular(index, privateIndexLimit)
	switch {
	case err == nil:
		if err := os.WriteFile(copyPath, raw, 0o600); err != nil {
			done()
			return func() {}, err
		}
	case os.IsNotExist(err):
		// No index yet: an absent private index reads the same way.
	default:
		done()
		return func() {}, fmt.Errorf("pose: the index of %s cannot be copied: %w", root, err)
	}
	// A split index would write a new shared index into .git; the copy is
	// written whole instead.
	cmd.Args = append([]string{cmd.Args[0], "-c", "core.splitIndex=false"}, cmd.Args[1:]...)
	cmd.Env = append(cmd.Env, "GIT_INDEX_FILE="+copyPath)
	return done, nil
}

// RawGitOutputWithPrivateIndex runs RawGit(args...) over a private copy of
// root's index and returns its standard output.
func RawGitOutputWithPrivateIndex(root string, args ...string) ([]byte, error) {
	cmd := RawGit(args...)
	done, err := WithPrivateIndex(cmd, root)
	defer done()
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

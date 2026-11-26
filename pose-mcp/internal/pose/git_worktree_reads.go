package pose

import (
	"fmt"
	"os/exec"
	"strings"
)

// Spec pose-validate-reads-leave-the-index-alone R1, D1: `git diff` against
// the work tree refreshes the index's stat cache and writes it back even with
// GIT_OPTIONAL_LOCKS=0, and diff.autoRefreshIndex=false would change its
// answer (a file whose stat alone changed would be listed). So the diff runs
// as it is, but reads the index through a descriptor whose lock Git cannot
// create: it refreshes in memory, cannot write, and skips the write silently
// (refresh_index_quietly), so its answer is the plain diff's and nothing is
// written anywhere. Where that is not available, it reads a private copy.

// diffIndexLimit bounds the index one read loads.
const diffIndexLimit = 512 << 20

// ReadOnlyIndex prepares cmd, a git command reading the repository at root,
// to read root's index without being able to write it. The returned function
// releases what it holds; call it after the command ends.
func ReadOnlyIndex(cmd *exec.Cmd, root string) (func(), error) {
	out, err := RawGit("-C", root, "rev-parse", "--path-format=absolute", "--git-path", "index").Output()
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be located: %w", root, err)
	}
	raw, err := readBoundedRegular(strings.TrimSuffix(string(out), "\n"), diffIndexLimit)
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be read: %w", root, err)
	}
	return readOnlyIndexFor(cmd, raw)
}

// RawGitDiffReadOnly runs RawGit(args...), a diff of the repository at root,
// reading its index without writing it, and returns its standard output.
func RawGitDiffReadOnly(root string, args ...string) ([]byte, error) {
	cmd := RawGit(args...)
	done, err := ReadOnlyIndex(cmd, root)
	defer done()
	if err != nil {
		return nil, err
	}
	return cmd.Output()
}

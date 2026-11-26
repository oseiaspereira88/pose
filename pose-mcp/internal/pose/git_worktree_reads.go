package pose

import (
	"fmt"
	"io"
	"os"
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
	path := strings.TrimSuffix(string(out), "\n")
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		// No index yet: there is no stat cache to refresh, so the plain read
		// writes nothing (review round 2, N11).
		return func() {}, nil
	}
	// One open file is checked and read, within the bound (review round 3,
	// N17): a file swapped in between cannot escape it.
	f, err := os.Open(path)
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be read: %w", root, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return func() {}, fmt.Errorf("pose: the index of %s is not a regular file", root)
	}
	raw, err := io.ReadAll(io.LimitReader(f, diffIndexLimit+1))
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be read: %w", root, err)
	}
	if len(raw) > diffIndexLimit {
		return func() {}, fmt.Errorf("pose: the index of %s is larger than %d bytes", root, diffIndexLimit)
	}
	// The copy keeps the index's modification time: Git trusts an entry on
	// its stat only when the file is older than the index, so a fresher copy
	// would hide a change made in the index's own second (review round 2,
	// N10, racy-clean entries).
	return readOnlyIndexFor(cmd, raw, info.ModTime())
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

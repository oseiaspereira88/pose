package pose

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Spec pose-validate-reads-leave-the-index-alone R1, D1: `git diff` against
// the work tree refreshes the index's stat cache and writes it back even with
// GIT_OPTIONAL_LOCKS=0, and diff.autoRefreshIndex=false would change its
// answer (a file whose stat alone changed would be listed). So the diff runs
// as it is, but reads the index through a descriptor whose lock Git cannot
// create: it refreshes in memory, cannot write, and skips the write silently
// (refresh_index_quietly), so its answer is the plain diff's and nothing is
// written anywhere. Where that is not available (another system, or a
// memory file that cannot be made), it reads a private copy.

// diffIndexLimit bounds the index one read loads.
const diffIndexLimit = 512 << 20

// ErrIndexUnreadable marks a read that failed on the index, not on the
// revisions it compares (review round 5, N23).
var ErrIndexUnreadable = errors.New("the index cannot be read")

// indexLookupTimeout bounds the lookup of the index's path.
const indexLookupTimeout = 30 * time.Second

// ReadOnlyIndex prepares cmd, a git command reading the repository at root,
// to read root's index without being able to write it. The returned function
// releases what it holds; call it after the command ends.
func ReadOnlyIndex(cmd *exec.Cmd, root string) (func(), error) {
	// `--git-path` without `--path-format`, which Git before 2.31 would echo
	// back as a path that does not exist (review round 5, N26): the answer is
	// relative to root unless absolute.
	ctx, cancel := context.WithTimeout(context.Background(), indexLookupTimeout)
	defer cancel()
	out, err := gitCommandContext(ctx, "--no-replace-objects", "-C", root, "rev-parse", "--git-path", "index").Output()
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be located: %w: %w", root, ErrIndexUnreadable, err)
	}
	path := strings.TrimSuffix(string(out), "\n")
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	// Opened without blocking, so a FIFO or device there cannot hang the
	// read, and judged on the open file, so an index another process
	// replaces atomically is read whole, as Git reads it (review rounds 4
	// N21, 5 N22).
	f, err := openIndexNoBlock(path)
	if os.IsNotExist(err) {
		// No index yet: there is no stat cache to refresh, so the plain read
		// writes nothing (review round 2, N11).
		return func() {}, nil
	}
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be read: %w: %w", root, ErrIndexUnreadable, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return func() {}, fmt.Errorf("pose: the index of %s is not a regular file: %w", root, ErrIndexUnreadable)
	}
	raw, err := io.ReadAll(io.LimitReader(f, diffIndexLimit+1))
	if err != nil {
		return func() {}, fmt.Errorf("pose: the index of %s cannot be read: %w: %w", root, ErrIndexUnreadable, err)
	}
	if len(raw) > diffIndexLimit {
		return func() {}, fmt.Errorf("pose: the index of %s is larger than %d bytes: %w", root, diffIndexLimit, ErrIndexUnreadable)
	}
	// The copy keeps the index's modification time: Git trusts an entry on
	// its stat only when the file is older than the index, so a fresher copy
	// would hide a change made in the index's own second (review round 2,
	// N10, racy-clean entries).
	return readOnlyIndexFor(cmd, raw, info.ModTime())
}

// privateIndexCopy hands raw to cmd as a private copy: Git refreshes and may
// rewrite the copy, never the checkout's index.
func privateIndexCopy(cmd *exec.Cmd, raw []byte, modTime time.Time) (func(), error) {
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

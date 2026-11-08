package pose

import (
	"bytes"
	"errors"
	"strings"
)

// One Git read dominated `pose check --strict` by process count (spec
// pose-check-spawns-fewer-git-processes): one `git status` per subject path asked
// whether it was dirty. Measured on pose-dist at 2026-10-09: 9,211 of 29,500 Git
// processes. The other, `git diff-tree` per (commit, path) pair, is read once per
// commit by the governed capture below.

// commitPathsMaxOutput bounds what one `git diff-tree` may return; a commit that
// changed more is "Git cannot answer", which every caller already treats as unknown
// rather than as a clean commit and which marks the governed read unknown.
const (
	commitPathsMaxOutput = 4 << 20
	commitPathsMaxPaths  = 100_000
)

var errCommitPathsTooLarge = errors.New("pose: a commit changed more paths than the reader accepts")

// parseCommitPaths turns the NUL-separated output of `git diff-tree -z` into a set
// in place, without an intermediate slice, refusing more than commitPathsMaxPaths
// entries: the map is bounded by 4 MiB of path bytes and that many entries.
func parseCommitPaths(raw []byte) (map[string]bool, error) {
	paths := map[string]bool{}
	for len(raw) > 0 {
		end := bytes.IndexByte(raw, 0)
		if end < 0 {
			end = len(raw)
		}
		if end > 0 {
			if len(paths) >= commitPathsMaxPaths {
				return nil, errCommitPathsTooLarge
			}
			paths[string(raw[:end])] = true
		}
		if end == len(raw) {
			break
		}
		raw = raw[end+1:]
	}
	return paths, nil
}

// workingTreeSnapshot is one `git status` of the whole tree, taken for the
// duration of one operation. The working tree changes between operations, so
// unlike commitPaths it is never shared across them.
type workingTreeSnapshot struct {
	ok      bool
	entries map[string]string
	// unknown is why the status could not be read, when it could not and the directory is
	// a repository: every path is then unknown, and an unknown path is not a clean one.
	unknown string
}

func takeWorkingTreeSnapshotIn(files Files, root string) *workingTreeSnapshot {
	snapshot := &workingTreeSnapshot{entries: map[string]string{}}
	entries, err := ReadGitStatusIn(files, root)
	if errors.Is(err, ErrGitUnavailable) {
		// Unit fixtures and exported source trees may not have Git metadata.
		return snapshot
	}
	if err != nil {
		snapshot.unknown = err.Error()
		return snapshot
	}
	snapshot.ok = true
	for _, entry := range entries {
		status := strings.TrimSpace(entry.XY)
		snapshot.entries[entry.Path] = status
		if entry.OrigPath != "" {
			snapshot.entries[entry.OrigPath] = status
		}
	}
	return snapshot
}

// takeWorkingTreeSnapshot reads the filesystem directly; takeWorkingTreeSnapshotIn reads the operation's files.
func takeWorkingTreeSnapshot(root string) *workingTreeSnapshot {
	return takeWorkingTreeSnapshotIn(NativeFiles(), root)
}

// change answers what `git status -- path` would: whether path, or anything
// under it, differs from HEAD, with the first status found.
func (w *workingTreeSnapshot) change(path string) (bool, string) {
	if w != nil && w.unknown != "" {
		return true, " (git status unknown: " + w.unknown + ")"
	}
	if w == nil || !w.ok {
		return false, ""
	}
	if status, ok := w.entries[path]; ok {
		return true, statusDetail(status)
	}
	prefix := strings.TrimSuffix(path, "/") + "/"
	for entry, status := range w.entries {
		if strings.HasPrefix(entry, prefix) {
			return true, statusDetail(status)
		}
	}
	return false, ""
}

func statusDetail(status string) string {
	if status == "" {
		return ""
	}
	return " (git status " + status + ")"
}

package pose

import (
	"bytes"
	"context"
	"errors"
	"strings"
)

// Two Git reads dominated `pose check --strict` by process count (spec
// pose-check-spawns-fewer-git-processes): one `git diff-tree` per (commit,
// path) pair asked whether a shared commit changed a path, and one
// `git status` per subject path asked whether it was dirty. Measured on
// pose-dist at 2026-10-09: 12,339 and 9,211 of 29,500 Git processes.

// commitPaths caches the paths each commit changed. A commit's content is
// immutable, so the answer never goes stale, even in a long-lived server. It is
// bounded in bytes and metered per repository root (spec
// pose-bounded-isolated-graph-caches R4/R6): the previous cap counted entries and
// dropped everything at once, and the paths of one commit can be many or few.
var commitPaths = newBoundedCache(64<<20, 32<<20, 0, func(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
})

// commitChangedPaths returns the paths commit changed, read once per process (and
// once when several callers ask together). ok is false when Git cannot answer; a
// failure is not remembered.
// commitPathsMaxOutput bounds what one `git diff-tree` may return to the cache; a
// commit that changed more is "Git cannot answer", which every caller already treats
// as unknown rather than as a clean commit. The reservation is what the output and
// the map built from it can hold at that bound.
const (
	commitPathsMaxOutput = 4 << 20
	commitPathsReserve   = 4 * commitPathsMaxOutput
)

var errCommitPathsTooLarge = errors.New("pose: a commit changed more paths than the cache reads")

// limitedBuffer keeps at most max bytes and refuses the rest, so the read of a huge
// commit is stopped instead of held.
type limitedBuffer struct {
	buf  bytes.Buffer
	max  int
	over bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.max {
		b.over = true
		return 0, errCommitPathsTooLarge
	}
	return b.buf.Write(p)
}

// commitChangedPaths returns the paths commit changed, read once per process (and
// once when several callers ask together). ok is false when Git cannot answer; a
// failure is not remembered.
func commitChangedPaths(root, commit string) (map[string]bool, bool) {
	paths, err := commitPaths.Get(context.Background(), root, root+"\x00"+commit, commitPathsReserve, func() (map[string]bool, int64, error) {
		out := &limitedBuffer{max: commitPathsMaxOutput}
		cmd := gitCommand("-C", root, "diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", "-z", commit)
		cmd.Stdout = out
		if err := cmd.Run(); err != nil {
			if out.over {
				return nil, 0, errCommitPathsTooLarge
			}
			return nil, 0, err
		}
		paths := map[string]bool{}
		var size int64 = 64
		for _, path := range strings.Split(out.buf.String(), "\x00") {
			if path != "" {
				paths[path] = true
				size += int64(len(path)) + 64
			}
		}
		return paths, size, nil
	})
	if err != nil {
		return nil, false
	}
	return paths, true
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

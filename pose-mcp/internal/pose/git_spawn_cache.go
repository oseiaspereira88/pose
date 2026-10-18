package pose

import (
	"errors"
	"strings"
	"sync"
)

// Two Git reads dominated `pose check --strict` by process count (spec
// pose-check-spawns-fewer-git-processes): one `git diff-tree` per (commit,
// path) pair asked whether a shared commit changed a path, and one
// `git status` per subject path asked whether it was dirty. Measured on
// pose-dist at 2026-10-09: 12,339 and 9,211 of 29,500 Git processes.

// commitPaths caches the paths each commit changed. A commit's content is
// immutable, so the answer never goes stale, even in a long-lived server.
var commitPaths = struct {
	sync.Mutex
	byKey map[string]map[string]bool
}{byKey: map[string]map[string]bool{}}

const commitPathsCacheLimit = 20000

// commitChangedPaths returns the paths commit changed, read once per process.
// ok is false when Git cannot answer.
func commitChangedPaths(root, commit string) (map[string]bool, bool) {
	key := root + "\x00" + commit
	commitPaths.Lock()
	paths, cached := commitPaths.byKey[key]
	commitPaths.Unlock()
	if cached {
		return paths, true
	}
	raw, err := gitCommand("-C", root, "diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", "-z", commit).Output()
	if err != nil {
		return nil, false
	}
	paths = map[string]bool{}
	for _, path := range strings.Split(string(raw), "\x00") {
		if path != "" {
			paths[path] = true
		}
	}
	commitPaths.Lock()
	if len(commitPaths.byKey) >= commitPathsCacheLimit {
		commitPaths.byKey = map[string]map[string]bool{}
	}
	commitPaths.byKey[key] = paths
	commitPaths.Unlock()
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

func takeWorkingTreeSnapshot(root string) *workingTreeSnapshot {
	snapshot := &workingTreeSnapshot{entries: map[string]string{}}
	entries, err := ReadGitStatus(root)
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

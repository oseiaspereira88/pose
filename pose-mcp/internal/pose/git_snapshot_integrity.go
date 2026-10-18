package pose

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Spec pose-git-and-dispatch-snapshot-integrity: Git state that could not be read is
// unknown, never clean, and a review worktree is judged by its content, not by what Git
// has been told to ignore.

// ErrGitUnknown is the cause of every failure to read Git state: a status, a HEAD or a
// tree that could not be read says nothing about whether the tree is clean.
var ErrGitUnknown = errors.New("pose: git state is unknown")

// ErrGitUnavailable means the directory is not a Git repository at all: there is no
// history to read, which is a different fact from history that could not be read. Exported
// source trees and unit fixtures are in this state; a consumer that needs a commit to
// describe still has none.
var ErrGitUnavailable = errors.New("pose: not a git repository")

// GitUnknownError names what could not be read and why.
type GitUnknownError struct {
	Op    string
	Cause string
}

func (e *GitUnknownError) Error() string {
	return "pose: git state is unknown (" + e.Op + "): " + e.Cause
}

func (e *GitUnknownError) Unwrap() error { return ErrGitUnknown }

// hasGitMetadata reports whether root or one of its ancestors holds a .git entry: a
// directory that has one is a repository, however damaged.
func hasGitMetadata(root string) bool {
	dir, err := filepath.Abs(root)
	if err != nil {
		return true
	}
	for {
		if _, lerr := os.Lstat(filepath.Join(dir, ".git")); lerr == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func gitUnknown(root, op string, err error, stderr []byte) error {
	cause := ""
	if err != nil {
		cause = err.Error()
	}
	if msg := strings.TrimSpace(string(stderr)); msg != "" {
		cause = msg
	}
	// "not a git repository" is also what a repository with a damaged HEAD says; only a
	// directory without any .git entry is not one.
	if strings.Contains(cause, "not a git repository") && !hasGitMetadata(root) {
		return fmt.Errorf("%w (%s)", ErrGitUnavailable, op)
	}
	return &GitUnknownError{Op: op, Cause: cause}
}

// GitStatusEntry is one entry of `git status --porcelain=v1 -z`: paths are exact bytes,
// never quoted, and a rename or copy carries both of its paths.
type GitStatusEntry struct {
	XY       string
	Path     string
	OrigPath string
}

// ReadGitStatus reads the working-tree status of root (limited to pathspecs when given).
// A failure is an ErrGitUnknown, not an empty status.
func ReadGitStatus(root string, pathspecs ...string) ([]GitStatusEntry, error) {
	args := []string{"-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	cmd := gitCommand(args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		return nil, gitUnknown(root, "status", err, stderr.Bytes())
	}
	return parseGitStatusZ(raw), nil
}

func parseGitStatusZ(raw []byte) []GitStatusEntry {
	fields := strings.Split(string(raw), "\x00")
	var entries []GitStatusEntry
	for i := 0; i < len(fields); i++ {
		record := fields[i]
		if len(record) < 4 {
			continue
		}
		entry := GitStatusEntry{XY: record[:2], Path: record[3:]}
		// A rename or copy is followed by its source path.
		if (record[0] == 'R' || record[0] == 'C' || record[1] == 'R' || record[1] == 'C') && i+1 < len(fields) {
			i++
			entry.OrigPath = fields[i]
		}
		entries = append(entries, entry)
	}
	return entries
}

// ReadGitHead returns the commit HEAD names, or ErrGitUnknown.
func ReadGitHead(root string) (string, error) {
	cmd := gitCommand("-C", root, "rev-parse", "--verify", "HEAD")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", gitUnknown(root, "rev-parse HEAD", err, stderr.Bytes())
	}
	return strings.TrimSpace(string(out)), nil
}

// derivedStateFiles are the files `pose` writes directly under .pose/state: the list of
// what each producer writes, so a sealed note placed beside them is not a regeneration.
var derivedStateFiles = map[string]bool{
	"contributor.json": true, "history.jsonl": true, "integrations.json": true,
	"machinery-manifest.json": true, "project-state.md": true, "refresh-log.jsonl": true,
	"technical-debt.json": true,
}

// derivedIndexFile reports that name is an index file a producer rewrites in place; the
// validation matrix, the extension lock and anything else under .pose/indexes are not.
func derivedIndexFile(name string) bool {
	if name == indexDescriptorName || name == "module-metadata.json" || name == "task-map.json" {
		return true
	}
	for _, file := range IndexGenerationFiles {
		if file == name {
			return true
		}
	}
	return false
}

// IsDerivedRegeneration reports whether a change to path may be a tool's regeneration of
// derived state rather than a change to sealed content. It is a list of what each producer
// writes, not a prefix: the validation matrix, the policy and sealed evidence live beside
// derived files and are never regenerated by a review. The path is judged as Git gave it:
// on POSIX a backslash is a character of a file name, not a separator.
//
// tracked says path exists in the sealed commit: an existing sealed record is never
// rewritten, only new ones are written, while an index or an assessment is rewritten in
// place. Only the presence of a file is judged here; a removal is never a regeneration
// (WorktreeDivergence refuses it before asking).
func IsDerivedRegeneration(p string, tracked bool) bool {
	if p == "" || path.Clean(p) != p {
		return false
	}
	switch {
	case strings.HasPrefix(p, ".pose/assessments/"):
		name := strings.TrimPrefix(p, ".pose/assessments/")
		return !strings.Contains(name, "/") && name != "README.md" && (strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json"))
	case strings.HasPrefix(p, ".pose/state/"):
		rest := strings.TrimPrefix(p, ".pose/state/")
		if comp, ok := strings.CutPrefix(rest, "components/"); ok {
			return comp != "" && !strings.Contains(comp, "/") && strings.HasSuffix(comp, ".json")
		}
		return derivedStateFiles[rest]
	case strings.HasPrefix(p, ".pose/indexes/"):
		rest := strings.TrimPrefix(p, ".pose/indexes/")
		if gen, ok := strings.CutPrefix(rest, "generations/"); ok {
			// generations/<id>/<file>.json: an immutable copy of a published index.
			parts := strings.Split(gen, "/")
			return !tracked && len(parts) == 2 && parts[0] != "" && strings.HasSuffix(parts[1], ".json")
		}
		return derivedIndexFile(rest)
	case strings.HasPrefix(p, ".pose/results/"), strings.HasPrefix(p, ".pose/review-bundles/"),
		strings.HasPrefix(p, ".pose/review-runs/"), strings.HasPrefix(p, ".pose/reports/history/"):
		return !tracked
	}
	return false
}

// WorktreeDivergence lists the paths of the worktree whose content is not the content of
// commit, comparing the files themselves with the commit's tree: a path Git was told to
// ignore changes in (`assume-unchanged`, `skip-worktree`) is read like any other, and a
// missing file, a replaced type or a link is a divergence. Paths for which allowed
// returns true may be rewritten in place as regular files and are otherwise skipped; a
// removal is a divergence whatever allowed says. A submodule of the commit must be
// materialized at its commit. A tree that cannot be read is ErrGitUnknown.
func WorktreeDivergence(worktree, commit string, allowed func(path string, tracked bool) bool) ([]string, error) {
	return WorktreeDivergenceFor(worktree, commit, allowed, nil)
}

// WorktreeDivergenceFor is WorktreeDivergence for a caller that knows which submodules
// matter: a gitlink for which gitlinkMatters returns false is another repository and is
// not examined. A nil gitlinkMatters examines every one.
func WorktreeDivergenceFor(worktree, commit string, allowed func(path string, tracked bool) bool, gitlinkMatters func(path string) bool) ([]string, error) {
	// Paths are the commit's, relative to the root of the repository: a directory inside
	// the worktree is compared as the worktree it belongs to.
	if out, err := gitCommand("-C", worktree, "rev-parse", "--show-toplevel").Output(); err == nil {
		if top := strings.TrimSuffix(string(out), "\n"); top != "" {
			worktree = top
		}
	}
	format := "sha1"
	if out, err := gitCommand("-C", worktree, "rev-parse", "--show-object-format").Output(); err == nil && strings.TrimSpace(string(out)) == "sha256" {
		format = "sha256"
	}
	cmd := gitCommand("-C", worktree, "ls-tree", "-r", "-z", "--full-tree", commit)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	tree, err := cmd.Output()
	if err != nil {
		return nil, gitUnknown(worktree, "ls-tree", err, stderr.Bytes())
	}
	var diverged []string
	seen := map[string]bool{}
	checkedDirs := map[string]bool{}
	for _, record := range strings.Split(string(tree), "\x00") {
		if record == "" {
			continue
		}
		meta, name, ok := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 {
			return nil, &GitUnknownError{Op: "ls-tree", Cause: "unreadable tree entry"}
		}
		mode, kind, object := fields[0], fields[1], fields[2]
		seen[name] = true
		full := filepath.Join(worktree, filepath.FromSlash(name))
		if kind == "commit" {
			if gitlinkMatters == nil || gitlinkMatters(name) {
				if !gitlinkMaterialized(full, object) {
					diverged = append(diverged, name)
				}
			}
			continue
		}
		info, lerr := os.Lstat(full)
		if lerr != nil {
			diverged = append(diverged, name)
			continue
		}
		// A directory above the file that is a link leads out of the worktree, whatever Git
		// was told to ignore: the leaf would be read from somewhere else.
		if linkAbove(worktree, name, checkedDirs) {
			diverged = append(diverged, name)
			continue
		}
		if allowed != nil && allowed(name, true) {
			// Rewritten in place: still the same kind of thing, a regular file.
			if !info.Mode().IsRegular() {
				diverged = append(diverged, name)
			}
			continue
		}
		switch mode {
		case "120000":
			target, rerr := os.Readlink(full)
			if info.Mode()&os.ModeSymlink == 0 || rerr != nil || gitBlobID(format, []byte(target)) != object {
				diverged = append(diverged, name)
			}
		default:
			if !info.Mode().IsRegular() {
				diverged = append(diverged, name)
				continue
			}
			content, rerr := os.ReadFile(full)
			if rerr != nil {
				// A file that could not be read is unknown, not an edited one.
				return nil, &GitUnknownError{Op: "read " + name, Cause: rerr.Error()}
			}
			// Windows does not keep an executable bit: its absence there says nothing.
			executable := info.Mode().Perm()&0o111 != 0
			if gitBlobID(format, content) != object || (runtime.GOOS != "windows" && executable != (mode == "100755")) {
				diverged = append(diverged, name)
			}
		}
	}
	// Files that are not in the commit and that Git does not ignore, whether or not they
	// were added to the index.
	ucmd := gitCommand("-C", worktree, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	var ustderr bytes.Buffer
	ucmd.Stderr = &ustderr
	others, err := ucmd.Output()
	if err != nil {
		return nil, gitUnknown(worktree, "ls-files", err, ustderr.Bytes())
	}
	for _, name := range strings.Split(string(others), "\x00") {
		if name == "" || seen[name] {
			continue
		}
		if allowed != nil && allowed(name, false) {
			if info, lerr := os.Lstat(filepath.Join(worktree, filepath.FromSlash(name))); lerr == nil && info.Mode().IsRegular() {
				continue
			}
		}
		diverged = append(diverged, name)
	}
	sort.Strings(diverged)
	return diverged, nil
}

// gitlinkMaterialized reports that dir is a real directory holding the submodule at commit:
// a repository of its own (not a link to one, not a directory that would answer with its
// parent's), at that HEAD, whose files are the files of that commit. The pin alone proves
// only where the history is, not what the checkout reads.
func gitlinkMaterialized(dir, commit string) bool {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	if g, gerr := os.Lstat(filepath.Join(dir, ".git")); gerr != nil || g.Mode()&os.ModeSymlink != 0 {
		return false
	}
	out, err := gitCommand("-C", dir, "rev-parse", "--verify", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(out)) != commit {
		return false
	}
	diverged, derr := WorktreeDivergenceFor(dir, commit, nil, nil)
	return derr == nil && len(diverged) == 0
}

// gitBlobID is the object id Git gives a blob with this content.
func gitBlobID(format string, content []byte) string {
	var h hash.Hash
	if format == "sha256" {
		h = sha256.New()
	} else {
		h = sha1.New()
	}
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// gitRedirectingEnv are the variables that point Git at a repository, a worktree, an index or
// an object store other than the directory it is run in.
var gitRedirectingEnv = map[string]bool{
	"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true, "GIT_OBJECT_DIRECTORY": true,
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_COMMON_DIR": true, "GIT_NAMESPACE": true,
	"GIT_CEILING_DIRECTORIES": true, "GIT_PREFIX": true, "GIT_QUARANTINE_PATH": true, "GIT_REPLACE_REF_BASE": true,
}

// gitCommand is a git subprocess that reads the repository of the directory it is given with
// -C, whatever the environment of the process says: an inherited GIT_DIR or GIT_WORK_TREE would
// answer for another materialization than the one that was asked about.
func gitCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	env := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !gitRedirectingEnv[name] {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	return cmd
}

// linkAbove reports that a directory between the worktree and the file is not a real directory.
// Directories already examined are remembered in checked (true: a link was found there).
func linkAbove(worktree, name string, checked map[string]bool) bool {
	parts := strings.Split(name, "/")
	dir := ""
	for _, part := range parts[:len(parts)-1] {
		dir = path.Join(dir, part)
		if bad, seen := checked[dir]; seen {
			if bad {
				return true
			}
			continue
		}
		info, err := os.Lstat(filepath.Join(worktree, filepath.FromSlash(dir)))
		checked[dir] = err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir()
		if checked[dir] {
			return true
		}
	}
	return false
}

package pose

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Snapshot disk, including ignored files. Git status alone cannot witness
// creation of an ignored output. WalkDir never follows a directory symlink.
func dispatchDiskSnapshot(root string) (map[string]string, error) {
	entries := map[string]string{}
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		// The per-root authority lock is runtime coordination state, not a
		// project input. Ignore only the exact regular file: a linked or special
		// replacement remains visible to the snapshot and fails closed.
		if filepath.ToSlash(rel) == ".pose/transfers/.authority-transfer.lock" && info.Mode().IsRegular() {
			return nil
		}
		if len(entries) >= 100000 {
			return fmt.Errorf("disk snapshot exceeds 100000 entries")
		}
		hash := sha256.New()
		fmt.Fprintf(hash, "%s\x00", info.Mode().String())
		switch {
		case info.Mode().IsRegular():
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			n, readErr := io.Copy(hash, io.LimitReader(file, (256<<20)-total+1))
			closeErr := file.Close()
			total += n
			if readErr != nil {
				return readErr
			}
			if closeErr != nil {
				return closeErr
			}
			if total > 256<<20 {
				return fmt.Errorf("disk snapshot exceeds 256 MiB")
			}
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			hash.Write([]byte(target))
		case info.IsDir():
		default:
			return fmt.Errorf("unsupported file type at %q", rel)
		}
		entries[filepath.ToSlash(rel)] = hex.EncodeToString(hash.Sum(nil))
		return nil
	})
	if err != nil {
		return nil, &GitSnapshotError{"disk snapshot", err}
	}
	return entries, nil
}

// GitSnapshotError means a repository observation was unavailable, not clean.
type GitSnapshotError struct {
	Operation string
	Cause     error
}

func (e *GitSnapshotError) Error() string {
	return fmt.Sprintf("pose: Git snapshot unavailable (%s): %v", e.Operation, e.Cause)
}
func (e *GitSnapshotError) Unwrap() error { return e.Cause }

func snapshotGit(root string, args ...string) ([]byte, error) {
	cmd := gitCommand(append([]string{"-C", root}, args...)...)
	// Asking about the worktree writes nothing under .git: without this `git status` refreshes
	// and rewrites the index, which a governed read view that watches .git reads as an input that
	// changed during the operation (spec pose-git-status-leaves-the-git-directory-alone).
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return nil, &GitSnapshotError{Operation: strings.Join(args, " "), Cause: err}
	}
	return out, nil
}

// gitChangedPaths uses the NUL protocol; porcelain's quoted display is not a path.
func gitChangedPaths(root string) ([]string, error) {
	out, err := snapshotGit(root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return nil, err
	}
	entries, err := parseGitStatusEntries(out)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		if untrackedAuthorityTransferLock(entry) {
			continue
		}
		paths = append(paths, entry.Path)
		if entry.OrigPath != "" {
			paths = append(paths, entry.OrigPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func untrackedAuthorityTransferLock(entry GitStatusEntry) bool {
	return entry.XY == "??" && entry.Path == ".pose/transfers/.authority-transfer.lock"
}

func parseGitChangedPaths(out []byte) ([]string, error) {
	entries, err := parseGitStatusEntries(out)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, entry := range entries {
		paths = append(paths, entry.Path)
		if entry.OrigPath != "" {
			paths = append(paths, entry.OrigPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// gitMaterializedChanges checks the committed tree against disk independently
// of index flags. No refresh/update-index mutation is needed to see hidden edits.
func gitMaterializedChanges(root, commit string) ([]string, error) {
	return gitMaterializedChangesFor(root, commit, nil)
}
func gitMaterializedChangesFor(root, commit string, matters func(string) bool) ([]string, error) {
	return gitMaterializedChangesForIn(NativeFiles(), root, commit, matters, "")
}

func gitMaterializedChangesForIn(files Files, root, commit string, matters func(string) bool, producer string) ([]string, error) {
	files = FilesOrNative(files)
	if producer != "" {
		producer = governanceProducerFromFiles(files, producer)
	}
	runGit := snapshotGit
	worktreeIdentity := dispatchWorktreeIdentity
	if producer != "" {
		runGit = func(root string, args ...string) ([]byte, error) {
			return captureGovernanceGit(files, root, producer, args...)
		}
		worktreeIdentity = func(root string) (string, error) {
			return dispatchWorktreeIdentityGoverned(files, root, producer)
		}
	}
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, &GitSnapshotError{"root", err}
	}
	if _, err := worktreeIdentity(root); err != nil {
		return nil, err
	}
	out, err := runGit(root, "ls-tree", "-r", "-z", commit)
	if err != nil {
		return nil, err
	}
	var changed []string
	var total int64
	count := 0
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		count++
		if count > 100000 {
			return nil, &GitSnapshotError{"tree snapshot", fmt.Errorf("exceeds 100000 entries")}
		}
		i := bytes.IndexByte(record, '\t')
		if i < 0 {
			return nil, &GitSnapshotError{"tree protocol", fmt.Errorf("missing path")}
		}
		fields := strings.Fields(string(record[:i]))
		if len(fields) != 3 {
			return nil, &GitSnapshotError{"tree protocol", fmt.Errorf("invalid entry")}
		}
		path := string(record[i+1:])
		if fields[0] == "160000" && matters != nil && !matters(path) {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(path))
		// Lstat every parent: a link in a parent must never read outside root.
		unsafe := false
		for parent := filepath.Dir(full); parent != root; parent = filepath.Dir(parent) {
			if parent == filepath.Dir(parent) {
				return nil, &GitSnapshotError{"path", fmt.Errorf("path outside worktree: %q", path)}
			}
			var info os.FileInfo
			var e error
			if producer != "" {
				info, e = files.Lstat(parent)
			} else {
				info, e = os.Lstat(parent)
			}
			if e != nil || !info.IsDir() {
				unsafe = true
				break
			}
		}
		if unsafe {
			changed = append(changed, path)
			continue
		}
		var info os.FileInfo
		var e error
		if producer != "" {
			info, e = files.Lstat(full)
		} else {
			info, e = os.Lstat(full)
		}
		if os.IsNotExist(e) {
			changed = append(changed, path)
			continue
		}
		if e != nil {
			return nil, &GitSnapshotError{"lstat " + path, e}
		}
		if fields[0] == "160000" {
			if !info.IsDir() {
				changed = append(changed, path)
				continue
			}
			if _, err := worktreeIdentity(full); err != nil {
				return nil, err
			}
			head, e := runGit(full, "rev-parse", "--verify", "HEAD")
			if e != nil {
				return nil, e
			}
			var dirty []string
			if producer != "" {
				dirty, e = gitChangedPathsGovernedIn(files, full, producer)
			} else {
				dirty, e = gitChangedPaths(full)
			}
			if e != nil {
				return nil, e
			}
			if strings.TrimSpace(string(head)) != fields[2] || len(dirty) != 0 {
				changed = append(changed, path)
			} else {
				var nested []string
				if producer != "" {
					nested, err = gitMaterializedChangesForIn(files, full, fields[2], matters, producer)
				} else {
					nested, err = gitMaterializedChanges(full, fields[2])
				}
				if err != nil {
					return nil, err
				}
				if len(nested) != 0 {
					changed = append(changed, path)
				}
			}
			continue
		}
		var raw []byte
		if fields[0] == "120000" {
			if info.Mode()&os.ModeSymlink == 0 {
				changed = append(changed, path)
				continue
			}
			var target string
			target, e = os.Readlink(full)
			raw = []byte(target)
		} else {
			if !info.Mode().IsRegular() {
				changed = append(changed, path)
				continue
			}
			// Verify executable materialization even when index flags hide it.
			// Windows does not materialize POSIX executable bits.
			if runtime.GOOS != "windows" && (info.Mode().Perm()&0111 != 0) != (fields[0] == "100755") {
				changed = append(changed, path)
			}
			if info.Size() > (256<<20)-total {
				return nil, &GitSnapshotError{"read " + path, fmt.Errorf("snapshot exceeds 256 MiB")}
			}
			var file *os.File
			file, e = os.Open(full)
			if e == nil {
				raw, e = io.ReadAll(io.LimitReader(file, (256<<20)-total+1))
				closeErr := file.Close()
				if e == nil {
					e = closeErr
				}
				total += int64(len(raw))
				if total > 256<<20 {
					return nil, &GitSnapshotError{"read " + path, fmt.Errorf("snapshot exceeds 256 MiB")}
				}
			}
		}
		if e != nil {
			return nil, &GitSnapshotError{"read " + path, e}
		}
		cmd := gitCommand("-C", root, "hash-object", "--stdin", "--path="+path)
		if fields[0] == "120000" {
			cmd = gitCommand("-C", root, "hash-object", "--stdin", "--no-filters")
		}
		cmd.Stdin = bytes.NewReader(raw)
		hash, e := cmd.Output()
		if e != nil {
			return nil, &GitSnapshotError{"hash " + path, e}
		}
		if strings.TrimSpace(string(hash)) != fields[2] {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed, nil
}

// These readers are shared by the bundle, obligation and execution snapshots.
var ErrGitUnknown = errors.New("pose: git state is unknown")
var ErrGitUnavailable = errors.New("pose: not a git repository")

func (e *GitSnapshotError) Is(target error) bool { return target == ErrGitUnknown }

func hasGitMetadata(root string) bool {
	dir, err := filepath.Abs(root)
	if err != nil {
		return true
	}
	if gitMetadataAbove(dir) {
		return true
	}
	// git walks the physical path, and a link can lead out of the lexical one.
	physical, err := filepath.EvalSymlinks(dir)
	return err == nil && physical != dir && gitMetadataAbove(physical)
}

func gitMetadataAbove(dir string) bool {
	for {
		if gitMetadataAt(dir) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// gitMetadataAt reports that dir holds something of a repository, however broken. An empty
// `.git` directory holds nothing: git walks past it, and a sandbox masks the name with one in
// every root it makes writable. A bare repository has no `.git` and is its own directory.
func gitMetadataAt(dir string) bool {
	if isBareRepository(dir) {
		return true
	}
	path := filepath.Join(dir, ".git")
	info, err := os.Lstat(path)
	if err != nil {
		return !errors.Is(err, os.ErrNotExist)
	}
	if !info.IsDir() {
		return true
	}
	entries, err := os.ReadDir(path)
	return err != nil || len(entries) > 0
}

// isBareRepository applies the test git applies to a directory it is asked to treat as one.
func isBareRepository(dir string) bool {
	head, err := os.Lstat(filepath.Join(dir, "HEAD"))
	if err != nil || head.IsDir() {
		return false
	}
	for _, name := range []string{"objects", "refs"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

type GitStatusEntry struct{ XY, Path, OrigPath string }

func parseGitStatusEntries(out []byte) ([]GitStatusEntry, error) {
	var entries []GitStatusEntry
	for len(out) > 0 {
		i := bytes.IndexByte(out, 0)
		if i <= 3 || out[2] != ' ' {
			return nil, &GitSnapshotError{"status protocol", fmt.Errorf("malformed porcelain record")}
		}
		record := out[:i]
		out = out[i+1:]
		entry := GitStatusEntry{XY: string(record[:2]), Path: string(record[3:])}
		if record[0] == 'R' || record[0] == 'C' || record[1] == 'R' || record[1] == 'C' {
			i = bytes.IndexByte(out, 0)
			if i <= 0 {
				return nil, &GitSnapshotError{"status protocol", fmt.Errorf("missing rename source")}
			}
			entry.OrigPath = string(out[:i])
			out = out[i+1:]
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func ReadGitStatusIn(files Files, root string, pathspecs ...string) ([]GitStatusEntry, error) {
	args := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	preview := FilesOrNative(files).Preview()
	run := snapshotGit
	if preview != nil {
		run = previewGit
	}
	out, err := run(root, args...)
	if err != nil {
		if !hasGitMetadata(root) {
			return nil, fmt.Errorf("%w: %v", ErrGitUnavailable, err)
		}
		return nil, err
	}
	entries, err := parseGitStatusEntries(out)
	if err != nil || preview == nil {
		return entries, err
	}
	return projectGitStatus(preview, root, entries, pathspecs)
}

// ReadGitStatus reads and writes the filesystem directly; ReadGitStatusIn takes the
// operation's files.
func ReadGitStatus(root string, pathspecs ...string) ([]GitStatusEntry, error) {
	return ReadGitStatusIn(NativeFiles(), root, pathspecs...)
}

func ReadGitHead(root string) (string, error) {
	out, err := snapshotGit(root, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsDerivedRegeneration lists candidate native producer outputs. It grants no
// authority to an adapter: dispatch passes no allowance to WorktreeDivergence.
func IsDerivedRegeneration(path string, tracked bool) bool {
	if filepath.ToSlash(filepath.Clean(path)) != path {
		return false
	}
	if !tracked && path == ".pose/transfers/.authority-transfer.lock" {
		return true
	}
	if path == ".pose/state/project-state.md" || path == ".pose/state/project-state.json" || path == ".pose/assessments/technical-debt.md" || path == ".pose/state/technical-debt.json" {
		return true
	}
	for _, name := range append(append([]string{}, IndexGenerationFiles...), indexDescriptorName) {
		if path == ".pose/indexes/"+name {
			return true
		}
	}
	if !tracked && path == ".pose/results/delivery-validation.json" {
		return true
	}
	return false
}

// WorktreeDivergence uses one content reader even for paths hidden from status.
func WorktreeDivergence(root, commit string, allowed func(string, bool) bool) ([]string, error) {
	return WorktreeDivergenceFor(root, commit, allowed, nil)
}
func WorktreeDivergenceFor(root, commit string, allowed func(string, bool) bool, matters func(string) bool) ([]string, error) {
	top, err := snapshotGit(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	if !sameOrAbove(strings.TrimSpace(string(top)), root) {
		return nil, fmt.Errorf("Git selected another worktree")
	}
	root = strings.TrimSpace(string(top))
	changed, err := gitMaterializedChangesFor(root, commit, matters)
	if err != nil {
		return nil, err
	}
	status, err := ReadGitStatus(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range status {
		// Fixture worktrees need not carry the root .gitignore. Keep an
		// untracked lock out of material status in that case, while still
		// treating a staged/modified lock as a real repository change.
		if untrackedAuthorityTransferLock(entry) {
			continue
		}
		changed = append(changed, entry.Path)
		if entry.OrigPath != "" {
			changed = append(changed, entry.OrigPath)
		}
	}
	tracked := map[string]bool{}
	if allowed != nil {
		out, err := snapshotGit(root, "ls-tree", "-r", "-z", "--name-only", commit)
		if err != nil {
			return nil, err
		}
		for _, path := range bytes.Split(out, []byte{0}) {
			tracked[string(path)] = true
		}
	}
	seen := map[string]bool{}
	result := []string{}
	for _, path := range changed {
		_, regularErr := dispatchRegularInputPath(root, path)
		if seen[path] || (allowed != nil && allowed(path, tracked[path]) && regularErr == nil) {
			continue
		}
		seen[path] = true
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func gitCommand(args ...string) *exec.Cmd {
	return gitCommandContext(context.Background(), args...)
}

// gitCommandContext is gitCommand under ctx. Both read raw object history: a
// replace ref or a graft changes what a commit is, and every gate has to judge
// the same commit (spec pose-history-readers-read-raw R1, D1).
func gitCommandContext(ctx context.Context, args ...string) *exec.Cmd {
	// The graft file below is the null device, and Git hints at any graft file on
	// stderr, which a caller that reads stderr as uncertainty would take for a failure.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "advice.graftFileDeprecated=false"}, args...)...)
	env := make([]string, 0, len(os.Environ())+2)
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !gitRedirectingEnv[name] {
			env = append(env, kv)
		}
	}
	cmd.Env = append(env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE="+os.DevNull)
	return cmd
}

const (
	gitEmptyTreeSHA1   = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	gitEmptyTreeSHA256 = "6ef19b41225c5369f1c104d45d8d85efa9b057b53b14b4b9b939dd74decc5321"
)

// GitEmptyTree is the empty tree in commit's object format: the base of a change
// set that starts at a repository's first commit (spec
// pose-history-readers-read-raw R2, D2).
func GitEmptyTree(commit string) string {
	if len(commit) == len(gitEmptyTreeSHA256) {
		return gitEmptyTreeSHA256
	}
	return gitEmptyTreeSHA1
}

// IsGitEmptyTree says that revision is the base of such a change set.
func IsGitEmptyTree(revision string) bool {
	return revision == gitEmptyTreeSHA1 || revision == gitEmptyTreeSHA256
}

// ChangeSetRange is the revision range of a change set for rev-list and log.
func ChangeSetRange(base, head string) string {
	if IsGitEmptyTree(base) {
		return head
	}
	return base + ".." + head
}

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

func sameOrAbove(top, dir string) bool {
	rt, err := filepath.EvalSymlinks(top)
	if err != nil {
		return false
	}
	rd, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rt, rd)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

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

var gitRedirectingEnv = map[string]bool{
	"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true, "GIT_OBJECT_DIRECTORY": true,
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_COMMON_DIR": true, "GIT_NAMESPACE": true,
	"GIT_CEILING_DIRECTORIES": true, "GIT_PREFIX": true, "GIT_QUARANTINE_PATH": true, "GIT_REPLACE_REF_BASE": true,
}

// gitCommand is a git subprocess that reads the repository of the directory it is given with
// -C, whatever the environment of the process says: an inherited GIT_DIR or GIT_WORK_TREE would
// answer for another materialization than the one that was asked about.

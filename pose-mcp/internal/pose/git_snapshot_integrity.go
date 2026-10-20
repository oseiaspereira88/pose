package pose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
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
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
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
	return parseGitChangedPaths(out)
}

func parseGitChangedPaths(out []byte) ([]string, error) {
	var paths []string
	for len(out) > 0 {
		i := bytes.IndexByte(out, 0)
		if i < 3 || out[2] != ' ' {
			return nil, &GitSnapshotError{"status protocol", fmt.Errorf("malformed porcelain record")}
		}
		record := out[:i]
		out = out[i+1:]
		paths = append(paths, string(record[3:]))
		if record[0] == 'R' || record[0] == 'C' || record[1] == 'R' || record[1] == 'C' {
			i = bytes.IndexByte(out, 0)
			if i <= 0 {
				return nil, &GitSnapshotError{"status protocol", fmt.Errorf("missing rename source")}
			}
			paths = append(paths, string(out[:i]))
			out = out[i+1:]
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// gitMaterializedChanges checks the committed tree against disk independently
// of index flags. No refresh/update-index mutation is needed to see hidden edits.
func gitMaterializedChanges(root, commit string) ([]string, error) {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, &GitSnapshotError{"root", err}
	}
	out, err := snapshotGit(root, "ls-tree", "-r", "-z", commit)
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
		full := filepath.Join(root, filepath.FromSlash(path))
		// Lstat every parent: a link in a parent must never read outside root.
		unsafe := false
		for parent := filepath.Dir(full); parent != root; parent = filepath.Dir(parent) {
			if parent == filepath.Dir(parent) {
				return nil, &GitSnapshotError{"path", fmt.Errorf("path outside worktree: %q", path)}
			}
			info, e := os.Lstat(parent)
			if e != nil || !info.IsDir() {
				unsafe = true
				break
			}
		}
		if unsafe {
			changed = append(changed, path)
			continue
		}
		info, e := os.Lstat(full)
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
			head, e := snapshotGit(full, "rev-parse", "--verify", "HEAD")
			if e != nil {
				return nil, e
			}
			dirty, e := gitChangedPaths(full)
			if e != nil {
				return nil, e
			}
			if strings.TrimSpace(string(head)) != fields[2] || len(dirty) != 0 {
				changed = append(changed, path)
			} else {
				nested, err := gitMaterializedChanges(full, fields[2])
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
		cmd := exec.Command("git", "-C", root, "hash-object", "--stdin", "--path="+path)
		if fields[0] == "120000" {
			cmd = exec.Command("git", "-C", root, "hash-object", "--stdin", "--no-filters")
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

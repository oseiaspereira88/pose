package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Spec pose-validate-reads-leave-the-index-alone R1, D1: `git diff` against
// the work tree refreshes the index's stat cache and writes it back even with
// GIT_OPTIONAL_LOCKS=0, and diff.autoRefreshIndex=false would change its
// answer (a file whose stat alone changed would be listed). `git status`
// with optional locks off refreshes in memory and writes nothing, so a read
// that compares with the work tree asks status instead, and a comparison
// with an older commit adds the commits' own diff, which touches no index.

// WorktreeChange is one path that differs between HEAD and the work tree,
// with the change as `git diff HEAD --name-status` names it (M, D, T or A).
type WorktreeChange struct {
	Status string
	Path   string
}

// WorktreeChanges returns the tracked paths under pathspec that differ
// between HEAD and the work tree, without writing the index. Paths added to
// the index only are reported as A.
func WorktreeChanges(root string, pathspec ...string) ([]WorktreeChange, error) {
	args := append([]string{"-C", root, "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=no", "--ignore-submodules=none", "--"}, pathspec...)
	out, err := RawGit(args...).Output()
	if err != nil {
		return nil, fmt.Errorf("pose: git status failed: %w", err)
	}
	var changes []WorktreeChange
	for _, record := range strings.Split(string(out), "\x00") {
		if len(record) < 4 {
			continue
		}
		x, y, path := record[0], record[1], record[3:]
		status := "M"
		switch {
		case x == 'A':
			if y == 'D' {
				continue // added, then removed from the work tree: as in HEAD
			}
			status = "A"
		case x == 'D' || y == 'D':
			status = "D"
		case x == 'T' || y == 'T':
			status = "T"
		}
		changes = append(changes, WorktreeChange{Status: status, Path: path})
	}
	return changes, nil
}

// WorktreeChangedNames returns the names `git diff --name-only <base>` would
// give against the work tree, without writing the index: the paths changed
// between base and HEAD, and the paths the work tree changes from HEAD. A
// path in both is listed only when the work tree still differs from base, as
// the diff would (compared by content, as `git hash-object` reads it).
func WorktreeChangedNames(root, base string, diffArgs ...string) ([]string, error) {
	committed := map[string]bool{}
	var names []string
	if base != "" && base != "HEAD" {
		args := append([]string{"-C", root}, diffArgs...)
		args = append(args, "diff", "--name-only", "-z", "--no-renames", base, "HEAD", "--")
		out, err := RawGit(args...).Output()
		if err != nil {
			return nil, fmt.Errorf("pose: git diff %s HEAD failed: %w", base, err)
		}
		for _, name := range strings.Split(string(out), "\x00") {
			if name != "" {
				committed[name] = true
			}
		}
	}
	changes, err := WorktreeChanges(root)
	if err != nil {
		return nil, err
	}
	inWorktree := map[string]bool{}
	for _, c := range changes {
		inWorktree[c.Path] = true
		if !committed[c.Path] {
			names = append(names, c.Path)
			continue
		}
		same, err := worktreeMatchesCommit(root, base, c.Path)
		if err != nil {
			return nil, err
		}
		if !same {
			names = append(names, c.Path)
		}
	}
	for name := range committed {
		if !inWorktree[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// worktreeMatchesCommit reports whether path in the work tree has the
// content it has at commit, both absent counting as a match; nothing is
// written (hash-object without -w).
func worktreeMatchesCommit(root, commit, path string) (bool, error) {
	want, werr := RawGit("-C", root, "rev-parse", "--verify", "-q", commit+":"+path).Output()
	full := filepath.Join(root, filepath.FromSlash(path))
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return werr != nil, nil
		}
		return false, err
	}
	if werr != nil {
		return false, nil
	}
	var got []byte
	if info.IsDir() {
		// A submodule: its checked-out commit is what the gitlink records.
		got, err = RawGit("-C", full, "rev-parse", "HEAD").Output()
	} else {
		got, err = RawGit("-C", root, "hash-object", "--path="+path, "--", path).Output()
	}
	if err != nil {
		return false, fmt.Errorf("pose: git hash-object %s failed: %w", path, err)
	}
	return strings.TrimSpace(string(got)) == strings.TrimSpace(string(want)), nil
}

package cli

// Legacy test snapshots only; production preview never copies or hashes a tree.
import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func hashTree(root string, skipGit bool) (map[string]string, error) {
	out := map[string]string{}
	err := hashTreeInto(out, root, "", skipGit, map[string]bool{})
	return out, err
}
func hashTreeInto(out map[string]string, dir, prefix string, skipGit bool, visiting map[string]bool) error {
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil
	}
	if visiting[real] {
		return nil
	}
	visiting[real] = true
	defer delete(visiting, real)
	return filepath.WalkDir(real, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(real, path)
		rel = filepath.Join(prefix, rel)
		if skipGit && (rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator))) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			info, err := os.Stat(path)
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return hashTreeInto(out, path, rel, skipGit, visiting)
			}
			if !info.Mode().IsRegular() {
				return nil
			}
		} else if !d.Type().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrPermission) {
			out[filepath.ToSlash(rel)] = "unreadable"
			return nil
		}
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
}

type treeChange struct{ verb, path string }

func diffTrees(before, after map[string]string) []treeChange {
	out := []treeChange{}
	for path, content := range after {
		if old, ok := before[path]; !ok {
			out = append(out, treeChange{"create", path})
		} else if old != content {
			out = append(out, treeChange{"modify", path})
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			out = append(out, treeChange{"remove", path})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

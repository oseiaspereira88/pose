package pose

// Git answers in a preview (spec pose-update-dry-run-reports-the-whole-update
// D2). Git reads the repository on disk; what an update stages is not there.
// Answers about committed objects and the index are the same either way. An
// answer about the working tree — `git status`, the ignored paths — is
// projected: the status git gives for the original tree, with each staged
// change applied to it as git would see it. The preview's git never refreshes
// the index (GIT_OPTIONAL_LOCKS=0), so asking writes nothing under .git.

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// previewGit runs a read-only git query for a preview.
func previewGit(root string, args ...string) ([]byte, error) {
	cmd := gitCommand(append([]string{"-C", root}, args...)...)
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return nil, &GitSnapshotError{Operation: strings.Join(args, " "), Cause: err}
	}
	return out, nil
}

// pathspecCovers reports whether a literal pathspec list selects path.
func pathspecCovers(pathspecs []string, path string) bool {
	if len(pathspecs) == 0 {
		return true
	}
	for _, spec := range pathspecs {
		spec = strings.TrimSuffix(filepath.ToSlash(spec), "/")
		if spec == "" || spec == "." || path == spec || strings.HasPrefix(path, spec+"/") {
			return true
		}
	}
	return false
}

type indexEntry struct{ mode, object string }

// projectGitStatus applies the preview's staged changes to the status git
// reported for the original working tree.
func projectGitStatus(p *PreviewFiles, root string, entries []GitStatusEntry, pathspecs []string) ([]GitStatusEntry, error) {
	changes := []PreviewChange{}
	for _, change := range p.Changes(root) {
		if !change.Outside && !strings.Contains(change.Verb, "directory") && pathspecCovers(pathspecs, change.Path) {
			changes = append(changes, change)
		}
	}
	if len(changes) == 0 {
		return entries, nil
	}
	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		paths = append(paths, change.Path)
	}
	index, err := gitIndexEntries(root, paths)
	if err != nil {
		return nil, err
	}
	ignored, err := gitCheckIgnore(root, paths)
	if err != nil {
		return nil, err
	}
	format := "sha1"
	if out, err := previewGit(root, "rev-parse", "--show-object-format"); err == nil {
		format = strings.TrimSpace(string(out))
	}
	byPath := map[string]GitStatusEntry{}
	for _, entry := range entries {
		byPath[entry.Path] = entry
	}
	for _, change := range changes {
		current, listed := byPath[change.Path]
		tracked, isTracked := index[change.Path]
		if !isTracked {
			switch {
			case strings.HasPrefix(change.Verb, "remove"):
				delete(byPath, change.Path)
			case !listed && !ignored[change.Path]:
				byPath[change.Path] = GitStatusEntry{XY: "??", Path: change.Path}
			}
			continue
		}
		staged := byte(' ')
		if listed && current.XY != "??" {
			staged = current.XY[0]
		}
		worktree := byte('M')
		if strings.HasPrefix(change.Verb, "remove") {
			worktree = 'D'
		} else if mode, object, ok := p.worktreeObject(filepath.Join(root, filepath.FromSlash(change.Path)), format); ok {
			switch {
			case mode == tracked.mode && object == tracked.object:
				worktree = ' '
			case (mode == "120000") != (tracked.mode == "120000"):
				worktree = 'T'
			}
		}
		if staged == ' ' && worktree == ' ' {
			delete(byPath, change.Path)
			continue
		}
		byPath[change.Path] = GitStatusEntry{XY: string([]byte{staged, worktree}), Path: change.Path, OrigPath: current.OrigPath}
	}
	out := make([]GitStatusEntry, 0, len(byPath))
	for _, entry := range byPath {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// worktreeObject is the git object the view holds at path: its mode and
// object id, as `git hash-object` would compute them.
func (p *PreviewFiles) worktreeObject(path, format string) (string, string, bool) {
	info, err := p.Lstat(path)
	if err != nil {
		return "", "", false
	}
	var content []byte
	mode := "100644"
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		target, err := p.Readlink(path)
		if err != nil {
			return "", "", false
		}
		content, mode = []byte(target), "120000"
	case info.Mode().IsRegular():
		raw, err := p.ReadFile(path)
		if err != nil {
			return "", "", false
		}
		content = raw
		if info.Mode().Perm()&0o111 != 0 {
			mode = "100755"
		}
	default:
		return "", "", false
	}
	header := []byte(fmt.Sprintf("blob %d\x00", len(content)))
	if format == "sha256" {
		sum := sha256.Sum256(append(header, content...))
		return mode, hex.EncodeToString(sum[:]), true
	}
	sum := sha1.Sum(append(header, content...))
	return mode, hex.EncodeToString(sum[:]), true
}

func gitIndexEntries(root string, paths []string) (map[string]indexEntry, error) {
	out, err := previewGit(root, append([]string{"ls-files", "--stage", "-z", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	entries := map[string]indexEntry{}
	for _, record := range strings.Split(string(out), "\x00") {
		meta, path, ok := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		if ok && len(fields) == 3 {
			entries[filepath.ToSlash(path)] = indexEntry{mode: fields[0], object: fields[1]}
		}
	}
	return entries, nil
}

// gitCheckIgnore returns which of paths the repository's ignore rules
// exclude; a path that does not exist yet is judged by name, as git does.
func gitCheckIgnore(root string, paths []string) (map[string]bool, error) {
	cmd := gitCommand("-C", root, "check-ignore", "-z", "--stdin")
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	cmd.Stdin = bytes.NewReader([]byte(strings.Join(paths, "\x00") + "\x00"))
	out, err := cmd.Output()
	ignored := map[string]bool{}
	if err != nil {
		// Exit status 1 means none is ignored.
		if exit, ok := err.(interface{ ExitCode() int }); ok && exit.ExitCode() == 1 {
			return ignored, nil
		}
		return nil, &GitSnapshotError{Operation: "check-ignore", Cause: err}
	}
	for _, path := range strings.Split(string(out), "\x00") {
		if path != "" {
			ignored[filepath.ToSlash(path)] = true
		}
	}
	return ignored, nil
}

// projectGitIgnored adds the staged entries the repository's ignore rules
// exclude to what `git ls-files --others --ignored --directory` reported for
// the original tree: the update creates them, and git then lists them.
func projectGitIgnored(p *PreviewFiles, root string, ignored map[string]bool) {
	candidates := []string{}
	for _, change := range p.Changes(root) {
		if !change.Outside && strings.HasPrefix(change.Verb, "create") {
			candidates = append(candidates, change.Path)
		}
	}
	if len(candidates) == 0 {
		return
	}
	excluded, err := gitCheckIgnore(root, candidates)
	if err != nil {
		p.Incomplete("the ignore rules for paths the update creates could not be read: " + err.Error())
		return
	}
	for path := range excluded {
		ignored[path] = true
		ignored[path+"/"] = true
	}
}

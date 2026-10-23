package pose

// The filesystem of one operation (spec pose-update-dry-run-reports-the-whole-update
// D2, ADR 2026-10-22 "Update preview records the shared execution in an
// operation-scoped view").
//
// `pose update --dry-run` used to run the update on a disposable copy of the
// instance. Ten review rounds found a layout the copy reproduced differently —
// links, modes, unreadable files, a directory that can be searched but not
// listed — and the copy never held the source tree module discovery reads. The
// update now runs once, through Files: applying, every operation reaches the
// filesystem as it always did; previewing, reads see the original tree with
// every staged change on top, and no change leaves memory.
//
// Files is passed explicitly: a Store carries it in FS, and the update path
// hands it to every reader and writer it calls. There is no process-wide
// switch, so a preview and an apply can run side by side.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Files is what one operation reads and changes. Paths are OS paths; a
// relative path is taken from the working directory, as os does.
//
// The changes are distinct operations because the filesystem treats them
// differently: WriteFile overwrites in place and follows a final link, so it
// needs the file to be writable; Publish replaces the directory entry whole,
// so it replaces a link instead of writing through it and needs the directory
// to be writable; CreateExclusive refuses any existing entry, a dangling link
// included.
type Files interface {
	ReadFile(name string) ([]byte, error)
	Open(name string) (fs.File, error)
	Stat(name string) (fs.FileInfo, error)
	Lstat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Readlink(name string) (string, error)
	EvalSymlinks(name string) (string, error)

	// WriteFile is os.WriteFile.
	WriteFile(name string, data []byte, perm fs.FileMode) error
	// CreateExclusive creates name holding data, failing when any entry
	// exists there; sync makes the bytes durable before it returns.
	CreateExclusive(name string, data []byte, perm fs.FileMode, sync bool) error
	// Publish is PublishFile.
	Publish(name string, data []byte, perm fs.FileMode, class string, expect PublishExpectation) error
	MkdirAll(name string, perm fs.FileMode) error
	Mkdir(name string, perm fs.FileMode) error
	Remove(name string) error
	RemoveAll(name string) error
	Symlink(oldname, newname string) error

	// Preview is the preview this operation records into, nil when it
	// applies. Code that takes a lock or spawns a process asks it, because
	// neither is a file a view can stage.
	Preview() *PreviewFiles
	// Now is the instant the operation captured, so every date it writes —
	// a policy stamp, a request id — is the same whether it applies or
	// previews.
	Now() time.Time
}

// NativeFiles applies every operation to the filesystem, reading the clock
// each time it is asked.
func NativeFiles() Files { return nativeFiles{} }

// NativeFilesAt applies every operation to the filesystem, dated at.
func NativeFilesAt(at time.Time) Files { return nativeFiles{at: at} }

// FilesOrNative returns files, or the native filesystem when it is nil, so
// a zero Store keeps reading the disk.
func FilesOrNative(files Files) Files {
	if files == nil {
		return nativeFiles{}
	}
	return files
}

type nativeFiles struct{ at time.Time }

func (n nativeFiles) Now() time.Time {
	if n.at.IsZero() {
		return time.Now()
	}
	return n.at
}

func (nativeFiles) ReadFile(name string) ([]byte, error)       { return os.ReadFile(name) }
func (nativeFiles) Open(name string) (fs.File, error)          { return os.Open(name) }
func (nativeFiles) Stat(name string) (fs.FileInfo, error)      { return os.Stat(name) }
func (nativeFiles) Lstat(name string) (fs.FileInfo, error)     { return os.Lstat(name) }
func (nativeFiles) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }
func (nativeFiles) Readlink(name string) (string, error)       { return os.Readlink(name) }
func (nativeFiles) EvalSymlinks(name string) (string, error)   { return filepath.EvalSymlinks(name) }
func (nativeFiles) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (nativeFiles) CreateExclusive(name string, data []byte, perm fs.FileMode, sync bool) error {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err == nil && sync {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}
func (nativeFiles) Publish(name string, data []byte, perm fs.FileMode, class string, expect PublishExpectation) error {
	return PublishFile(name, data, perm, class, expect)
}
func (nativeFiles) MkdirAll(name string, perm fs.FileMode) error { return os.MkdirAll(name, perm) }
func (nativeFiles) Mkdir(name string, perm fs.FileMode) error    { return os.Mkdir(name, perm) }
func (nativeFiles) Remove(name string) error                     { return os.Remove(name) }
func (nativeFiles) RemoveAll(name string) error                  { return os.RemoveAll(name) }
func (nativeFiles) Symlink(oldname, newname string) error        { return os.Symlink(oldname, newname) }
func (nativeFiles) Preview() *PreviewFiles                       { return nil }

// GlobIn is filepath.Glob over files.
func GlobIn(files Files, pattern string) ([]string, error) {
	files = FilesOrNative(files)
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, err
	}
	if !hasGlobMeta(pattern) {
		if _, err := files.Lstat(pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}
	dir, file := filepath.Split(pattern)
	dir = cleanGlobPath(dir)
	if !hasGlobMeta(dir) {
		return globDir(files, dir, file, nil), nil
	}
	if dir == pattern {
		return nil, filepath.ErrBadPattern
	}
	dirs, err := GlobIn(files, dir)
	if err != nil {
		return nil, err
	}
	matches := []string{}
	for _, d := range dirs {
		matches = globDir(files, d, file, matches)
	}
	return matches, nil
}

func hasGlobMeta(path string) bool { return strings.ContainsAny(path, `*?[\`) }

func cleanGlobPath(path string) string {
	switch path {
	case "":
		return "."
	case string(filepath.Separator):
		return path
	}
	return path[:len(path)-1]
}

func globDir(files Files, dir, pattern string, matches []string) []string {
	info, err := files.Stat(dir)
	if err != nil || !info.IsDir() {
		return matches
	}
	entries, err := files.ReadDir(dir)
	if err != nil {
		return matches
	}
	for _, entry := range entries {
		if ok, _ := filepath.Match(pattern, entry.Name()); ok {
			matches = append(matches, filepath.Join(dir, entry.Name()))
		}
	}
	return matches
}

// WalkDirIn is filepath.WalkDir over files: links are reported, not
// followed, and entries come in lexical order.
func WalkDirIn(files Files, root string, fn fs.WalkDirFunc) error {
	files = FilesOrNative(files)
	info, err := files.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDirIn(files, root, fs.FileInfoToDirEntry(info), fn)
	}
	if err == filepath.SkipDir || err == filepath.SkipAll {
		return nil
	}
	return err
}

func walkDirIn(files Files, path string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			err = nil
		}
		return err
	}
	entries, err := files.ReadDir(path)
	if err != nil {
		err = fn(path, d, err)
		if err != nil {
			if err == filepath.SkipDir && d.IsDir() {
				err = nil
			}
			return err
		}
	}
	for _, entry := range entries {
		if err := walkDirIn(files, filepath.Join(path, entry.Name()), entry, fn); err != nil {
			if err == filepath.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// ErrPreviewUnsupported marks an operation a preview cannot stage. A preview
// that meets one is incomplete: it never reports what it could not evaluate as
// done or as changing nothing.
var ErrPreviewUnsupported = errors.New("preview-unsupported")

// PreviewFiles records an operation instead of applying it. Reads see the
// original filesystem, read lazily and by name — a directory that can be
// searched but not listed is read exactly as the operation reads it — with
// every staged change on top. Nothing is written: no temporary file, no lock,
// no link target, inside the instance or outside it.
//
// An entry is keyed by its resolved location, so a write through one alias
// is read back through any other, and replacing a link changes the link, not
// what it pointed to. A change the operating system would refuse now, as far
// as access(2) can tell, is refused with the error the operation would get;
// one it might refuse later (quota, a concurrent writer, a failing disk) is
// not predicted.
type PreviewFiles struct {
	mu         sync.Mutex
	nodes      map[string]*previewNode
	children   map[string]map[string]bool
	physical   map[string]physicalEntry
	ledger     []PreviewOperation
	gaps       []string
	now        time.Time
	umask      fs.FileMode
	umaskKnown bool
	// access answers whether the effective user may write to a physical path;
	// observedWriteAccess unless a test injects a refusal.
	access func(path string, dir bool) error
}

// PreviewOperation is one change a preview staged or refused, in order.
type PreviewOperation struct {
	Op        string `json:"op"`
	Path      string `json:"path"`
	Location  string `json:"location"`
	Target    string `json:"target,omitempty"`
	Class     string `json:"class,omitempty"`
	Refused   string `json:"refused,omitempty"`
	Cause     error  `json:"-"`
	Unchanged bool   `json:"unchanged,omitempty"`
}

type previewKind int

const (
	previewFile previewKind = iota + 1
	previewDir
	previewLink
	previewAbsent
)

type previewNode struct {
	kind   previewKind
	data   []byte
	perm   fs.FileMode
	target string
	// Physical identity survives direct overwrite, but not atomic replacement.
	origin string
}

type physicalEntry struct {
	info fs.FileInfo
	err  error
}

// NewPreviewFiles returns an empty preview; now dates every staged entry.
func NewPreviewFiles(now time.Time) *PreviewFiles {
	mask, known := observedUmask()
	return &PreviewFiles{
		nodes:      map[string]*previewNode{},
		children:   map[string]map[string]bool{},
		physical:   map[string]physicalEntry{},
		now:        now,
		umask:      mask,
		umaskKnown: known,
		access:     observedWriteAccess,
	}
}

// RefuseWritesUnder makes every change under dir fail with err, as if the
// operating system refused it. Only tests call it, to stand for a read-only
// mount or an ACL the process cannot see in permission bits.
func (p *PreviewFiles) RefuseWritesUnder(dir string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	dir = absClean(dir)
	previous := p.access
	p.access = func(path string, isDir bool) error {
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			return err
		}
		return previous(path, isDir)
	}
}

// Preview implements Files.
func (p *PreviewFiles) Preview() *PreviewFiles { return p }

// Now is the instant the preview dates its staged entries with.
func (p *PreviewFiles) Now() time.Time { return p.now }

// Incomplete records that the operation reached something the preview cannot
// evaluate faithfully. A preview with a gap is never reported as complete.
func (p *PreviewFiles) Incomplete(reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, gap := range p.gaps {
		if gap == reason {
			return
		}
	}
	p.gaps = append(p.gaps, reason)
}

// Unsupported records a gap and returns the error the operation receives
// instead of a fabricated success.
func (p *PreviewFiles) Unsupported(what string) error {
	p.Incomplete(what)
	return fmt.Errorf("%w: %s", ErrPreviewUnsupported, what)
}

// Gaps lists what the preview could not evaluate, in the order met.
func (p *PreviewFiles) Gaps() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string{}, p.gaps...)
}

// Operations returns the ledger: every change staged or refused, in order.
func (p *PreviewFiles) Operations() []PreviewOperation {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]PreviewOperation{}, p.ledger...)
}

// Staged reports whether any change is staged at or under path.
func (p *PreviewFiles) Staged(path string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	path = absClean(path)
	for key := range p.nodes {
		if key == path || strings.HasPrefix(key, path+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func absClean(name string) string {
	if abs, err := filepath.Abs(name); err == nil {
		return abs
	}
	return filepath.Clean(name)
}

// entry is what one location holds in the view.
type entry struct {
	kind   previewKind
	staged *previewNode
	info   fs.FileInfo // the original's, when not staged
}

func (e entry) isDir() bool  { return e.kind == previewDir }
func (e entry) isLink() bool { return e.kind == previewLink }

// lstatPhysical reads the original entry once per preview: the view is one
// observation of the filesystem, not a mixture of instants.
func (p *PreviewFiles) lstatPhysical(path string) (fs.FileInfo, error) {
	if cached, ok := p.physical[path]; ok {
		return cached.info, cached.err
	}
	info, err := os.Lstat(path)
	p.physical[path] = physicalEntry{info, err}
	return info, err
}

// physicalVisible reports whether the original entries under dir show
// through: not when dir, or a directory above it, was staged in place of the
// original (created, or removed and created again).
func (p *PreviewFiles) physicalVisible(dir string) bool {
	for current := dir; ; current = filepath.Dir(current) {
		if node, ok := p.nodes[current]; ok && (node.kind == previewDir || node.kind == previewAbsent || node.kind == previewFile || node.kind == previewLink) {
			return false
		}
		parent := filepath.Dir(current)
		if parent == current {
			return true
		}
	}
}

// lookup returns the entry at a resolved location, without following it.
func (p *PreviewFiles) lookup(location string) (entry, bool, error) {
	if node, ok := p.nodes[location]; ok {
		if node.kind == previewAbsent {
			return entry{}, false, nil
		}
		return entry{kind: node.kind, staged: node}, true, nil
	}
	parent := filepath.Dir(location)
	if parent != location && !p.physicalVisible(parent) {
		return entry{}, false, nil
	}
	info, err := p.lstatPhysical(location)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return entry{}, false, nil
		}
		return entry{}, false, err
	}
	kind := previewFile
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		kind = previewLink
	case info.IsDir():
		kind = previewDir
	}
	return entry{kind: kind, info: info}, true, nil
}

func (p *PreviewFiles) readlinkEntry(location string, e entry) (string, error) {
	if e.staged != nil {
		return e.staged.target, nil
	}
	return os.Readlink(location)
}

const maxPreviewLinks = 40

// resolve walks name component by component, through staged and original
// links, and returns the location it names. follow says whether a link in the
// last component is followed. exists is false when the last component is
// missing; its location is then where it would be created — through a
// dangling link, that is the link's target.
func (p *PreviewFiles) resolve(name string, follow bool) (string, entry, bool, error) {
	abs := name
	if !filepath.IsAbs(abs) {
		wd, err := os.Getwd()
		if err != nil {
			return "", entry{}, false, err
		}
		abs = wd + string(filepath.Separator) + name
	}
	volume := filepath.VolumeName(abs)
	sep := string(filepath.Separator)
	pending := splitComponents(strings.TrimPrefix(abs, volume))
	current := volume + sep
	links := 0
	for i := 0; i < len(pending); i++ {
		// Every component, including dot-dot, requires searching the
		// directory reached by the preceding component.
		if node := p.nodes[current]; node != nil && node.kind == previewDir {
			if err := p.stagedAccess(current, node, 0o100); err != nil {
				return current, entry{}, false, err
			}
		}
		component := pending[i]
		// Dot components still search the physical directory. Reducing
		// locked/.. lexically would bypass the kernel's current refusal.
		if (component == "." || component == "..") && p.nodes[current] == nil {
			known, err := observedReadWriteAccess(current, 0o100)
			if !known {
				err = p.unsupportedLocked("physical directory search unavailable: " + current)
			}
			if err != nil {
				return current, entry{}, false, err
			}
		}
		last := i == len(pending)-1
		switch component {
		case "", ".":
			if last {
				e, ok, err := p.lookup(current)
				return current, e, ok, err
			}
			continue
		case "..":
			current = filepath.Dir(current)
			if last {
				e, ok, err := p.lookup(current)
				return current, e, ok, err
			}
			continue
		}
		candidate := filepath.Join(current, component)
		e, ok, err := p.lookup(candidate)
		if err != nil {
			return candidate, entry{}, false, err
		}
		if !ok {
			if last {
				return candidate, entry{}, false, nil
			}
			return candidate, entry{}, false, fs.ErrNotExist
		}
		if e.isLink() && (!last || follow) {
			links++
			if links > maxPreviewLinks {
				return candidate, entry{}, false, syscall.ELOOP
			}
			target, err := p.readlinkEntry(candidate, e)
			if err != nil {
				return candidate, entry{}, false, err
			}
			rest := append(splitComponents(target), pending[i+1:]...)
			if filepath.IsAbs(target) {
				volume = filepath.VolumeName(target)
				current = volume + sep
				rest = append(splitComponents(strings.TrimPrefix(target, volume)), pending[i+1:]...)
			}
			if len(rest) == 0 {
				rest = []string{"."}
			}
			pending, i = rest, -1
			continue
		}
		if last {
			return candidate, e, true, nil
		}
		if !e.isDir() {
			return candidate, entry{}, false, syscall.ENOTDIR
		}
		current = candidate
	}
	e, ok, err := p.lookup(current)
	return current, e, ok, err
}

func splitComponents(path string) []string {
	parts := []string{}
	slashed := filepath.ToSlash(path)
	for _, part := range strings.Split(slashed, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	// A trailing separator requires traversing the preceding entry as a
	// directory, including when the path came from a literal symlink target.
	if strings.HasSuffix(slashed, "/") {
		parts = append(parts, ".")
	}
	return parts
}

func pathError(op, name string, err error) error {
	if err == nil {
		return nil
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return &fs.PathError{Op: op, Path: name, Err: pe.Err}
	}
	return &fs.PathError{Op: op, Path: name, Err: err}
}

func (p *PreviewFiles) ReadFile(name string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.readFileLocked(name)
}

func (p *PreviewFiles) readFileLocked(name string) ([]byte, error) {
	location, e, ok, err := p.resolve(name, true)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	if !ok {
		return nil, pathError("open", name, fs.ErrNotExist)
	}
	if e.isDir() {
		return nil, pathError("read", name, syscall.EISDIR)
	}
	if e.staged != nil {
		if err := p.stagedAccess(location, e.staged, 0o400); err != nil {
			return nil, pathError("open", name, err)
		}
		return append([]byte{}, e.staged.data...), nil
	}
	raw, err := os.ReadFile(location)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	return raw, nil
}

func (p *PreviewFiles) Open(name string) (fs.File, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	location, e, ok, err := p.resolve(name, true)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	if !ok {
		return nil, pathError("open", name, fs.ErrNotExist)
	}
	if e.staged == nil {
		file, err := os.Open(location)
		if err != nil {
			return nil, pathError("open", name, err)
		}
		return file, nil
	}
	if err := p.stagedAccess(location, e.staged, 0o400); err != nil {
		return nil, pathError("open", name, err)
	}
	if e.isDir() {
		return nil, pathError("open", name, p.unsupportedLocked("opening a staged directory as a file"))
	}
	return &stagedFile{Reader: bytes.NewReader(append([]byte{}, e.staged.data...)), info: p.stagedInfo(filepath.Base(location), e.staged)}, nil
}

func (p *PreviewFiles) unsupportedLocked(what string) error {
	for _, gap := range p.gaps {
		if gap == what {
			return fmt.Errorf("%w: %s", ErrPreviewUnsupported, what)
		}
	}
	p.gaps = append(p.gaps, what)
	return fmt.Errorf("%w: %s", ErrPreviewUnsupported, what)
}

type stagedFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *stagedFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *stagedFile) Close() error               { return nil }

func (p *PreviewFiles) Stat(name string) (fs.FileInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.statLocked(name, true)
}

func (p *PreviewFiles) Lstat(name string) (fs.FileInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.statLocked(name, false)
}

func (p *PreviewFiles) statLocked(name string, follow bool) (fs.FileInfo, error) {
	op := "lstat"
	if follow {
		op = "stat"
	}
	location, e, ok, err := p.resolve(name, follow)
	if err != nil {
		return nil, pathError(op, name, err)
	}
	if !ok {
		return nil, pathError(op, name, fs.ErrNotExist)
	}
	if e.staged != nil {
		return p.stagedInfo(filepath.Base(location), e.staged), nil
	}
	return renamedInfo{e.info, filepath.Base(absClean(name))}, nil
}

// renamedInfo reports the name the caller used, as os.Stat does, for an
// entry reached through links.
type renamedInfo struct {
	fs.FileInfo
	name string
}

func (r renamedInfo) Name() string { return r.name }

func (p *PreviewFiles) stagedInfo(name string, node *previewNode) fs.FileInfo {
	return stagedInfo{name: name, node: node, mod: p.now}
}

type stagedInfo struct {
	name string
	node *previewNode
	mod  time.Time
}

func (s stagedInfo) Name() string { return s.name }
func (s stagedInfo) Size() int64  { return int64(len(s.node.data)) }
func (s stagedInfo) Mode() fs.FileMode {
	switch s.node.kind {
	case previewDir:
		return fs.ModeDir | s.node.perm
	case previewLink:
		return fs.ModeSymlink | 0o777
	}
	return s.node.perm
}
func (s stagedInfo) ModTime() time.Time { return s.mod }
func (s stagedInfo) IsDir() bool        { return s.node.kind == previewDir }
func (s stagedInfo) Sys() any           { return nil }

func (p *PreviewFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.readDirLocked(name)
}

func (p *PreviewFiles) readDirLocked(name string) ([]fs.DirEntry, error) {
	location, e, ok, err := p.resolve(name, true)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	if !ok {
		return nil, pathError("open", name, fs.ErrNotExist)
	}
	if !e.isDir() {
		return nil, pathError("readdirent", name, syscall.ENOTDIR)
	}
	if e.staged != nil {
		if err := p.stagedAccess(location, e.staged, 0o400); err != nil {
			return nil, pathError("open", name, err)
		}
	}
	byName := map[string]fs.DirEntry{}
	if e.staged == nil {
		// The original is listed as the operation would list it: a directory
		// it cannot list fails here too, with the same error.
		original, err := os.ReadDir(location)
		if err != nil {
			return nil, pathError("open", name, err)
		}
		for _, child := range original {
			byName[child.Name()] = child
		}
	}
	for child := range p.children[location] {
		node := p.nodes[filepath.Join(location, child)]
		if node == nil {
			continue
		}
		if node.kind == previewAbsent {
			delete(byName, child)
			continue
		}
		byName[child] = fs.FileInfoToDirEntry(p.stagedInfo(child, node))
	}
	out := make([]fs.DirEntry, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func (p *PreviewFiles) Readlink(name string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	location, e, ok, err := p.resolve(name, false)
	if err != nil {
		return "", pathError("readlink", name, err)
	}
	if !ok {
		return "", pathError("readlink", name, fs.ErrNotExist)
	}
	if !e.isLink() {
		return "", pathError("readlink", name, syscall.EINVAL)
	}
	return p.readlinkEntry(location, e)
}

func (p *PreviewFiles) EvalSymlinks(name string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	location, _, ok, err := p.resolve(name, true)
	if err != nil {
		return "", pathError("lstat", name, err)
	}
	if !ok {
		return "", pathError("lstat", name, fs.ErrNotExist)
	}
	if !filepath.IsAbs(name) {
		if wd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(wd, location); err == nil && !strings.HasPrefix(rel, "..") {
				return rel, nil
			}
		}
	}
	return location, nil
}

// stage records node at location and keeps the parent's child list.
func (p *PreviewFiles) stage(location string, node *previewNode) {
	p.nodes[location] = node
	parent := filepath.Dir(location)
	if p.children[parent] == nil {
		p.children[parent] = map[string]bool{}
	}
	p.children[parent][filepath.Base(location)] = true
}

func (p *PreviewFiles) record(op PreviewOperation, err error) error {
	if err != nil {
		op.Refused = err.Error()
		op.Cause = err
	}
	p.ledger = append(p.ledger, op)
	return err
}

// stagedAccess uses native access for preserved physical identities. New
// entries belong to the effective user. Missing owner rights cannot establish
// refusal: capabilities/root or inherited ACLs may grant access. Disclose that
// limit instead of predicting success or denying a privileged operation.
func (p *PreviewFiles) stagedAccess(location string, node *previewNode, need fs.FileMode) error {
	if node.origin != "" {
		if known, err := observedReadWriteAccess(node.origin, need); known {
			return err
		}
		return p.unsupportedLocked("physical staged access unavailable: " + location)
	}
	if !stagedOwnerAccessKnown() || node.perm&need != need {
		return p.unsupportedLocked("staged effective permission evaluation unavailable: " + location)
	}
	return nil
}

// writableDir checks that the operation may add, replace or remove an entry
// in the directory at location.
func (p *PreviewFiles) writableDir(location string) error {
	if node, ok := p.nodes[location]; ok {
		return p.stagedAccess(location, node, 0o300)
	}
	return p.access(location, true)
}

// Directory write/search access does not establish permission to replace an
// entry under a sticky parent. Ownership and effective privilege are separate
// facts; retain uncertainty instead of predicting removal or replacement.
func (p *PreviewFiles) replacementAccess(location string) error {
	parent := filepath.Dir(location)
	e, ok, err := p.lookup(parent)
	if err != nil {
		return err
	}
	if ok && e.info != nil && e.info.Mode()&fs.ModeSticky != 0 {
		return p.unsupportedLocked("sticky-directory replacement authority unavailable: " + location)
	}
	return nil
}

// writableFile checks that the operation may write into the file at location.
func (p *PreviewFiles) writableFile(location string, e entry) error {
	if e.staged != nil {
		return p.stagedAccess(location, e.staged, 0o200)
	}
	return p.access(location, false)
}

func (p *PreviewFiles) WriteFile(name string, data []byte, perm fs.FileMode) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := PreviewOperation{Op: "write", Path: name}
	location, e, ok, err := p.resolve(name, true)
	op.Location = location
	if err != nil {
		return p.record(op, pathError("open", name, err))
	}
	if ok && e.isDir() {
		return p.record(op, pathError("open", name, syscall.EISDIR))
	}
	mode := perm.Perm() &^ p.umask
	origin := ""
	if ok {
		if e.staged == nil && (e.info == nil || !e.info.Mode().IsRegular()) {
			return p.record(op, p.unsupportedLocked("direct-write special-file effects unavailable: "+location))
		}
		if err := p.writableFile(location, e); err != nil {
			return p.record(op, pathError("open", name, err))
		}
		mode = p.permOf(location, e)
		if e.staged != nil {
			origin = e.staged.origin
		} else {
			links, known := physicalLinkCount(e.info)
			if !known || links > 1 {
				return p.record(op, p.unsupportedLocked("direct overwrite inode aliases unknown or multiply linked: "+location))
			}
			origin = location
		}
	} else if err := p.writableDir(filepath.Dir(location)); err != nil {
		return p.record(op, pathError("open", name, err))
	}
	if !ok && !p.umaskKnown {
		return p.record(op, p.unsupportedLocked("creation umask unavailable: "+location))
	}
	p.stage(location, &previewNode{kind: previewFile, data: append([]byte{}, data...), perm: mode, origin: origin})
	return p.record(op, nil)
}

func (p *PreviewFiles) permOf(location string, e entry) fs.FileMode {
	if e.staged != nil {
		return e.staged.perm
	}
	if e.info != nil {
		return e.info.Mode().Perm()
	}
	return 0o644
}

func (p *PreviewFiles) CreateExclusive(name string, data []byte, perm fs.FileMode, sync bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := PreviewOperation{Op: "create-exclusive", Path: name}
	location, _, ok, err := p.resolve(name, false)
	op.Location = location
	if err != nil {
		return p.record(op, pathError("open", name, err))
	}
	if ok {
		return p.record(op, pathError("open", name, fs.ErrExist))
	}
	if err := p.writableDir(filepath.Dir(location)); err != nil {
		return p.record(op, pathError("open", name, err))
	}
	if !p.umaskKnown {
		return p.record(op, p.unsupportedLocked("creation umask unavailable: "+location))
	}
	p.stage(location, &previewNode{kind: previewFile, data: append([]byte{}, data...), perm: perm.Perm() &^ p.umask})
	return p.record(op, nil)
}

// Publish stages what PublishFile publishes, checking the expectation against
// the view: the entry at name is replaced whole, a link included.
func (p *PreviewFiles) Publish(name string, data []byte, perm fs.FileMode, class string, expect PublishExpectation) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := PreviewOperation{Op: "publish", Path: name, Class: class}
	dir := filepath.Dir(name)
	if err := p.mkdirAllLocked(dir, 0o755); err != nil {
		return p.record(op, err)
	}
	dirLocation, _, _, err := p.resolve(dir, true)
	if err != nil {
		return p.record(op, pathError("open", name, err))
	}
	location := filepath.Join(dirLocation, filepath.Base(name))
	op.Location = location
	if err := p.writableDir(dirLocation); err != nil {
		return p.record(op, pathError("open", name, err))
	}
	switch expect.Mode {
	case ExpectAbsent:
		if _, exists, err := p.lookup(location); err != nil {
			return p.record(op, err)
		} else if exists {
			current, readErr := p.readFileLocked(name)
			if readErr != nil {
				return p.record(op, readErr)
			}
			if digestHex(current) != digestHex(data) {
				return p.record(op, &PublishConflictError{Path: name, Expected: "absent", Found: digestHex(current)})
			}
			op.Unchanged = true
			return p.record(op, nil)
		}
	case ExpectDigest:
		current, err := p.readFileLocked(name)
		found := "absent"
		if err == nil {
			found = digestHex(current)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return p.record(op, err)
		}
		if found != expect.Digest {
			return p.record(op, &PublishConflictError{Path: name, Expected: expect.Digest, Found: found})
		}
	}
	if existing, ok, err := p.lookup(location); err != nil {
		return p.record(op, err)
	} else if ok && existing.isDir() {
		return p.record(op, pathError("rename", name, syscall.EISDIR))
	} else if ok {
		if err := p.replacementAccess(location); err != nil {
			return p.record(op, err)
		}
	}
	p.stage(location, &previewNode{kind: previewFile, data: append([]byte{}, data...), perm: perm.Perm()})
	return p.record(op, nil)
}

func (p *PreviewFiles) MkdirAll(name string, perm fs.FileMode) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mkdirAllLocked(name, perm)
}

// mkdirAllLocked is os.MkdirAll: an existing directory, or a link to one, is
// accepted; anything else in the way is an error.
func (p *PreviewFiles) mkdirAllLocked(name string, perm fs.FileMode) error {
	if info, err := p.statLocked(name, true); err == nil {
		if info.IsDir() {
			return nil
		}
		return pathError("mkdir", name, syscall.ENOTDIR)
	}
	abs := absClean(name)
	if parent := filepath.Dir(abs); parent != abs {
		if err := p.mkdirAllLocked(parent, perm); err != nil {
			return err
		}
	}
	err := p.mkdirLocked(name, perm)
	if err != nil {
		if info, statErr := p.statLocked(name, false); statErr == nil && info.IsDir() {
			return nil
		}
	}
	return err
}

func (p *PreviewFiles) Mkdir(name string, perm fs.FileMode) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mkdirLocked(name, perm)
}

func (p *PreviewFiles) mkdirLocked(name string, perm fs.FileMode) error {
	op := PreviewOperation{Op: "mkdir", Path: name}
	location, _, ok, err := p.resolve(name, false)
	op.Location = location
	if err != nil {
		return p.record(op, pathError("mkdir", name, err))
	}
	if ok {
		return p.record(op, pathError("mkdir", name, fs.ErrExist))
	}
	if err := p.writableDir(filepath.Dir(location)); err != nil {
		return p.record(op, pathError("mkdir", name, err))
	}
	if !p.umaskKnown {
		return p.record(op, p.unsupportedLocked("creation umask unavailable: "+location))
	}
	p.stage(location, &previewNode{kind: previewDir, perm: perm.Perm() &^ p.umask})
	return p.record(op, nil)
}

func (p *PreviewFiles) Remove(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := PreviewOperation{Op: "remove", Path: name}
	location, e, ok, err := p.resolve(name, false)
	op.Location = location
	if err != nil {
		return p.record(op, pathError("remove", name, err))
	}
	if !ok {
		return p.record(op, pathError("remove", name, fs.ErrNotExist))
	}
	if err := p.replacementAccess(location); err != nil {
		return p.record(op, err)
	}
	if e.isDir() {
		children, err := p.readDirLocked(location)
		if err != nil {
			// Removal can succeed without read access to an empty directory.
			// Failed enumeration cannot establish the native removal result.
			return p.record(op, pathError("remove", name, p.unsupportedLocked("directory removal emptiness unavailable: "+location)))
		}
		if len(children) > 0 {
			return p.record(op, pathError("remove", name, syscall.ENOTEMPTY))
		}
	}
	if err := p.writableDir(filepath.Dir(location)); err != nil {
		return p.record(op, pathError("remove", name, err))
	}
	p.removeLocked(location)
	return p.record(op, nil)
}

// removeLocked stages the absence of location and drops what was staged
// under it.
func (p *PreviewFiles) removeLocked(location string) {
	prefix := location + string(filepath.Separator)
	for key := range p.nodes {
		if strings.HasPrefix(key, prefix) {
			delete(p.nodes, key)
		}
	}
	for key := range p.children {
		if key == location || strings.HasPrefix(key, prefix) {
			delete(p.children, key)
		}
	}
	p.stage(location, &previewNode{kind: previewAbsent})
}

func (p *PreviewFiles) RemoveAll(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := PreviewOperation{Op: "remove-all", Path: name}
	location, e, ok, err := p.resolve(name, false)
	op.Location = location
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return p.record(op, nil)
		}
		return p.record(op, pathError("unlinkat", name, err))
	}
	if !ok {
		return p.record(op, nil)
	}
	if err := p.replacementAccess(location); err != nil {
		return p.record(op, err)
	}
	if err := p.writableDir(filepath.Dir(location)); err != nil {
		return p.record(op, pathError("unlinkat", name, err))
	}
	if e.isDir() {
		if err := p.removableTree(location); err != nil {
			return p.record(op, pathError("unlinkat", name, err))
		}
	}
	p.removeLocked(location)
	return p.record(op, nil)
}

// removableTree checks that every directory under location can be listed
// and emptied, as os.RemoveAll needs.
func (p *PreviewFiles) removableTree(location string) error {
	children, err := p.readDirLocked(location)
	if err != nil {
		return p.unsupportedLocked("recursive removal contents unavailable: " + location)
	}
	if len(children) == 0 {
		return nil
	}
	if err := p.writableDir(location); err != nil {
		return err
	}
	for _, child := range children {
		if err := p.replacementAccess(filepath.Join(location, child.Name())); err != nil {
			return err
		}
		if child.IsDir() {
			if err := p.removableTree(filepath.Join(location, child.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *PreviewFiles) Symlink(oldname, newname string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := PreviewOperation{Op: "symlink", Path: newname, Target: oldname}
	location, _, ok, err := p.resolve(newname, false)
	op.Location = location
	if err != nil {
		return p.record(op, &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: unwrapPathErr(err)})
	}
	if ok {
		return p.record(op, &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: syscall.EEXIST})
	}
	if err := p.writableDir(filepath.Dir(location)); err != nil {
		return p.record(op, &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err})
	}
	p.stage(location, &previewNode{kind: previewLink, target: oldname})
	return p.record(op, nil)
}

func unwrapPathErr(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// PreviewChange is one net difference between the original filesystem and
// the view: what applying the operation leaves changed.
type PreviewChange struct {
	// Verb: create, modify, change mode, remove, create directory, remove
	// directory, create link, replace link, replace link with file, replace
	// file with link, replace directory.
	Verb string `json:"verb"`
	// Path is relative to the root the changes were asked for, slash-separated,
	// or absolute when the location is outside it.
	Path    string `json:"path"`
	Outside bool   `json:"outside,omitempty"`
	// Through names the path the operation wrote by, when it differs from the
	// location (a write through a link).
	Through string `json:"through,omitempty"`
}

// Changes returns the net differences, in path order. A location written
// several times is one change; one written back to its original bytes and
// mode is none.
func (p *PreviewFiles) Changes(root string) []PreviewChange {
	p.mu.Lock()
	defer p.mu.Unlock()
	rootLocation := absClean(root)
	if resolved, _, ok, err := p.resolve(root, true); err == nil && ok {
		rootLocation = resolved
	}
	// A change made through a link names the path the operation used: the
	// location relative to the resolved root differs from that path relative
	// to the root as given.
	through := map[string]string{}
	for _, op := range p.ledger {
		if op.Refused != "" || op.Location == "" {
			continue
		}
		logical, logicalErr := filepath.Rel(absClean(root), absClean(op.Path))
		located, locatedErr := filepath.Rel(rootLocation, op.Location)
		if logicalErr == nil && locatedErr == nil && logical == located {
			continue
		}
		if _, seen := through[op.Location]; !seen {
			if logicalErr == nil && !strings.HasPrefix(logical, "..") {
				through[op.Location] = filepath.ToSlash(logical)
			} else {
				through[op.Location] = op.Path
			}
		}
	}
	locations := make([]string, 0, len(p.nodes))
	for location := range p.nodes {
		locations = append(locations, location)
	}
	sort.Strings(locations)
	out := []PreviewChange{}
	for _, location := range locations {
		node := p.nodes[location]
		verb := p.netChange(location, node)
		if verb == "" {
			continue
		}
		change := PreviewChange{Verb: verb, Path: location}
		if rel, err := filepath.Rel(rootLocation, location); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			change.Path = filepath.ToSlash(rel)
		} else {
			change.Outside = true
		}
		change.Through = through[location]
		out = append(out, change)
		if verb == "remove directory" {
			out = append(out, p.removedBelow(location, change.Path)...)
		}
	}
	// A directory removed and created again hides what it held: each original
	// entry not staged again is removed.
	for _, location := range locations {
		node := p.nodes[location]
		if node.kind != previewDir {
			continue
		}
		info, err := p.lstatPhysical(location)
		if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			continue
		}
		entries, err := os.ReadDir(location)
		if err != nil {
			continue
		}
		rel := location
		if r, err := filepath.Rel(rootLocation, location); err == nil && !strings.HasPrefix(r, "..") {
			rel = filepath.ToSlash(r)
		}
		for _, original := range entries {
			child := filepath.Join(location, original.Name())
			if _, staged := p.nodes[child]; staged {
				continue
			}
			if original.IsDir() {
				out = append(out, PreviewChange{Verb: "remove directory", Path: rel + "/" + original.Name()})
				out = append(out, p.removedBelow(child, rel+"/"+original.Name())...)
				continue
			}
			out = append(out, PreviewChange{Verb: "remove", Path: rel + "/" + original.Name()})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func (p *PreviewFiles) netChange(location string, node *previewNode) string {
	// The original at this location: what was there before any change.
	var original fs.FileInfo
	if info, err := p.lstatPhysical(location); err == nil {
		original = info
	}
	switch node.kind {
	case previewAbsent:
		switch {
		case original == nil:
			return ""
		case original.IsDir():
			return "remove directory"
		}
		return "remove"
	case previewDir:
		switch {
		case original == nil:
			return "create directory"
		case original.IsDir():
			return ""
		}
		return "replace " + originalKind(original) + " with directory"
	case previewLink:
		switch {
		case original == nil:
			return "create link"
		case original.Mode()&fs.ModeSymlink != 0:
			if target, err := os.Readlink(location); err == nil && target == node.target {
				return ""
			}
			return "replace link"
		}
		return "replace " + originalKind(original) + " with link"
	}
	switch {
	case original == nil:
		return "create"
	case original.Mode()&fs.ModeSymlink != 0 || original.IsDir():
		return "replace " + originalKind(original) + " with file"
	}
	raw, err := os.ReadFile(location)
	if err != nil || !bytes.Equal(raw, node.data) {
		return "modify"
	}
	if original.Mode().Perm() != node.perm {
		return "change mode"
	}
	return ""
}

func originalKind(info fs.FileInfo) string {
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		return "link"
	case info.IsDir():
		return "directory"
	}
	return "file"
}

// removedBelow lists the original entries a removed directory held.
func (p *PreviewFiles) removedBelow(location, rel string) []PreviewChange {
	out := []PreviewChange{}
	entries, err := os.ReadDir(location)
	if err != nil {
		return out
	}
	for _, entry := range entries {
		childRel := rel + "/" + entry.Name()
		child := filepath.Join(location, entry.Name())
		if entry.IsDir() {
			out = append(out, PreviewChange{Verb: "remove directory", Path: childRel})
			out = append(out, p.removedBelow(child, childRel)...)
			continue
		}
		out = append(out, PreviewChange{Verb: "remove", Path: childRel})
	}
	return out
}

var _ io.Reader = (*stagedFile)(nil)

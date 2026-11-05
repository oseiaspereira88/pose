package pose

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func governancePathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// governanceRootVectorFiles routes resolver filesystem calls to the guarded
// Files of the root that owns each absolute path. It exposes no fallback to
// NativeFiles, so a resolver cannot read outside its operation's locked root
// vector by dropping or changing its project selector.
type governanceRootVectorFiles struct {
	roots []governanceRootFile
	clock Files
}

type governanceRootFile struct {
	path  string
	files Files
}

func newGovernanceRootVectorFiles(stores map[string]Store, localID string) Files {
	vector := &governanceRootVectorFiles{}
	if local, ok := stores[localID]; ok {
		vector.clock = local.fsys()
	}
	for _, store := range stores {
		root, err := canonicalGovernanceRoot(store.Root)
		if err != nil {
			continue
		}
		vector.roots = append(vector.roots, governanceRootFile{path: root, files: store.fsys()})
	}
	sort.Slice(vector.roots, func(i, j int) bool { return len(vector.roots[i].path) > len(vector.roots[j].path) })
	return vector
}

func (v *governanceRootVectorFiles) forPath(name string) (Files, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, ErrGovernanceWriteGuardRootBinding
	}
	abs = filepath.Clean(abs)
	for _, root := range v.roots {
		if governancePathWithin(root.path, abs) {
			return root.files, nil
		}
	}
	return nil, ErrGovernanceWriteGuardRootBinding
}

func (v *governanceRootVectorFiles) ReadFile(name string) ([]byte, error) {
	f, err := v.forPath(name)
	if err != nil {
		return nil, err
	}
	return f.ReadFile(name)
}
func (v *governanceRootVectorFiles) Open(name string) (fs.File, error) {
	f, err := v.forPath(name)
	if err != nil {
		return nil, err
	}
	return f.Open(name)
}
func (v *governanceRootVectorFiles) Stat(name string) (fs.FileInfo, error) {
	f, err := v.forPath(name)
	if err != nil {
		return nil, err
	}
	return f.Stat(name)
}
func (v *governanceRootVectorFiles) Lstat(name string) (fs.FileInfo, error) {
	f, err := v.forPath(name)
	if err != nil {
		return nil, err
	}
	return f.Lstat(name)
}
func (v *governanceRootVectorFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := v.forPath(name)
	if err != nil {
		return nil, err
	}
	return f.ReadDir(name)
}
func (v *governanceRootVectorFiles) Readlink(name string) (string, error) {
	f, err := v.forPath(name)
	if err != nil {
		return "", err
	}
	return f.Readlink(name)
}
func (v *governanceRootVectorFiles) EvalSymlinks(name string) (string, error) {
	f, err := v.forPath(name)
	if err != nil {
		return "", err
	}
	return f.EvalSymlinks(name)
}
func (v *governanceRootVectorFiles) WriteFile(name string, data []byte, perm fs.FileMode) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.WriteFile(name, data, perm)
}
func (v *governanceRootVectorFiles) CreateExclusive(name string, data []byte, perm fs.FileMode, syncFile bool) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.CreateExclusive(name, data, perm, syncFile)
}
func (v *governanceRootVectorFiles) Publish(name string, data []byte, perm fs.FileMode, class string, expect PublishExpectation) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.Publish(name, data, perm, class, expect)
}
func (v *governanceRootVectorFiles) MkdirAll(name string, perm fs.FileMode) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.MkdirAll(name, perm)
}
func (v *governanceRootVectorFiles) Mkdir(name string, perm fs.FileMode) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.Mkdir(name, perm)
}
func (v *governanceRootVectorFiles) Remove(name string) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.Remove(name)
}
func (v *governanceRootVectorFiles) RemoveAll(name string) error {
	f, err := v.forPath(name)
	if err != nil {
		return err
	}
	return f.RemoveAll(name)
}
func (v *governanceRootVectorFiles) Symlink(oldname, newname string) error {
	f, err := v.forPath(newname)
	if err != nil {
		return err
	}
	return f.Symlink(oldname, newname)
}
func (v *governanceRootVectorFiles) Preview() *PreviewFiles {
	if v.clock == nil {
		return nil
	}
	return v.clock.Preview()
}
func (v *governanceRootVectorFiles) Now() time.Time {
	if v.clock == nil {
		return time.Now()
	}
	return v.clock.Now()
}

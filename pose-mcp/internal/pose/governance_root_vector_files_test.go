package pose

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// New recovery regression coverage; the helper's body is restored literally.
func TestGovernancePathWithinBoundaries(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	for _, tt := range []struct {
		name, root, path string
		want             bool
	}{
		{"same", root, root, true},
		{"child", root, filepath.Join(root, ".pose", "specs"), true},
		{"dot-prefixed-child", root, filepath.Join(root, "..child"), true},
		{"clean-child", root, root + string(filepath.Separator) + "child" + string(filepath.Separator) + "..", true},
		{"parent", root, filepath.Dir(root), false},
		{"prefix-sibling", root, root + "-other", false},
		{"escape", root, filepath.Join(root, "..", "other", "file"), false},
		{"relative-target", root, "relative", false},
		{"relative-root", "relative", root, false},
		{"filesystem-root", filepath.VolumeName(root) + string(filepath.Separator), root, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := governancePathWithin(tt.root, tt.path); got != tt.want {
				t.Fatalf("governancePathWithin(%q, %q) = %v, want %v", tt.root, tt.path, got, tt.want)
			}
		})
	}
	if runtime.GOOS == "windows" {
		for _, pair := range [][2]string{{`C:\project`, `D:\project\file`}, {`\\server\share\project`, `\\server\other\project\file`}} {
			if governancePathWithin(pair[0], pair[1]) {
				t.Fatalf("different volumes accepted: %q", pair)
			}
		}
	}
}

func TestGovernanceRootVectorFilesRouting(t *testing.T) {
	parent := t.TempDir()
	nested := filepath.Join(parent, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	parentRoot, err := canonicalGovernanceRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	nestedRoot, err := canonicalGovernanceRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	parentFS := &governanceGuardTestFilesWrapper{Files: NativeFiles()}
	nestedFS := &governanceGuardTestFilesWrapper{Files: NativeFiles()}
	v := newGovernanceRootVectorFiles(map[string]Store{
		"parent": {Root: parent, FS: parentFS},
		"nested": {Root: nested, FS: nestedFS},
	}, "parent").(*governanceRootVectorFiles)
	for _, tt := range []struct {
		name string
		want Files
	}{
		{parentRoot, parentFS},
		{filepath.Join(parentRoot, "file"), parentFS},
		{nestedRoot, nestedFS},
		{filepath.Join(nestedRoot, "file"), nestedFS},
		{filepath.Join(parentRoot, "nested-other", "file"), parentFS},
	} {
		got, err := v.forPath(tt.name)
		if err != nil || got != tt.want {
			t.Fatalf("route %q = %T, %v; want selected root's Files", tt.name, got, err)
		}
	}
	outside := filepath.Join(t.TempDir(), "existing")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{parentRoot + "-other", filepath.Dir(parentRoot), outside} {
		if got, err := v.forPath(name); got != nil || !errors.Is(err, ErrGovernanceWriteGuardRootBinding) {
			t.Fatalf("outside route %q = %v, %v", name, got, err)
		}
	}
	if _, err := v.ReadFile(outside); !errors.Is(err, ErrGovernanceWriteGuardRootBinding) {
		t.Fatalf("outside read fell back to native Files: %v", err)
	}
	if err := v.WriteFile(outside, []byte("changed"), 0o600); !errors.Is(err, ErrGovernanceWriteGuardRootBinding) {
		t.Fatalf("outside write fell back to native Files: %v", err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "outside" {
		t.Fatalf("outside file changed: %q, %v", data, err)
	}
	if got, err := (&governanceRootVectorFiles{}).forPath(parentRoot); got != nil || !errors.Is(err, ErrGovernanceWriteGuardRootBinding) {
		t.Fatalf("empty root vector did not fail closed: %v, %v", got, err)
	}
}

func TestGovernanceRootVectorFilesRetainsReleasedGuard(t *testing.T) {
	store := governanceWriteGuardTestStore(t)
	var retained Files
	path := filepath.Join(store.Root, ".pose", "guarded.txt")
	err := WithGovernanceWriteGuard("test.root-vector", []Store{store}, func(guarded []Store) error {
		retained = newGovernanceRootVectorFiles(map[string]Store{"local": guarded[0]}, "local")
		return retained.WriteFile(path, []byte("guarded"), 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := retained.WriteFile(path, []byte("after-release"), 0o600); !errors.Is(err, ErrGovernanceWriteGuardReleased) {
		t.Fatalf("root routing bypassed released guard: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "guarded" {
		t.Fatalf("guarded content = %q, %v", data, err)
	}
}

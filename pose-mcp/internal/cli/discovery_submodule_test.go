package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// submoduleWithIgnoredCache mirrors the fixture in
// internal/pose/discovery_submodule_test.go: a submodule at mod/ whose own
// .gitignore excludes cache/, with a manifest and a README left in
// mod/cache/, next to a real module at mod/tool/.
func submoduleWithIgnoredCache(t *testing.T) string {
	t.Helper()
	git := func(dir string, args ...string) {
		t.Helper()
		base := []string{"-C", dir, "-c", "user.name=pose-test", "-c", "user.email=pose-test@example.invalid", "-c", "protocol.file.allow=always", "-c", "commit.gpgsign=false"}
		if out, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	upstream := t.TempDir()
	git(upstream, "init", "-q")
	write(filepath.Join(upstream, ".gitignore"), "cache/\n")
	write(filepath.Join(upstream, "tool", "go.mod"), "module example.com/tool\n")
	git(upstream, "add", ".")
	git(upstream, "commit", "-q", "-m", "upstream")

	root := t.TempDir()
	git(root, "init", "-q")
	git(root, "submodule", "add", "-q", upstream, "mod")
	write(filepath.Join(root, "mod", "cache", "go.mod"), "module example.com/cache\n")
	write(filepath.Join(root, "mod", "cache", "README.md"), "cache\n")
	return root
}

// Found on an adopting repository: a pytest cache inside the POSE submodule
// appeared in repo-map.json on one machine and not in a clean clone, so
// `pose index` produced a diff that depended on who ran it (spec
// pose-discovery-gitignore-inside-submodules).
func TestScanModules_RespectsGitignoreInsideSubmodule(t *testing.T) {
	root := submoduleWithIgnoredCache(t)

	modules, manifests, _, _, readmes := scanModules(root)
	for _, module := range modules {
		if module.Path == "mod/cache" {
			t.Fatalf("modules = %#v, want no mod/cache (ignored by the submodule)", modules)
		}
	}
	for _, path := range append(append([]string{}, manifests...), readmes...) {
		if strings.HasPrefix(path, "mod/cache/") {
			t.Fatalf("manifests = %#v, readmes = %#v, want nothing under mod/cache/", manifests, readmes)
		}
	}
	found := false
	for _, module := range modules {
		found = found || module.Path == "mod/tool"
	}
	if !found {
		t.Fatalf("modules = %#v, want the submodule's tracked mod/tool still discovered", modules)
	}
}

func TestDiscoverValidationModules_RespectsGitignoreInsideSubmodule(t *testing.T) {
	root := submoduleWithIgnoredCache(t)

	modules, err := discoverValidationModules(root)
	if err != nil {
		t.Fatalf("discover validation modules: %v", err)
	}
	rels := []string{}
	for _, module := range modules {
		rels = append(rels, module.Rel)
	}
	if len(rels) != 1 || rels[0] != "mod/tool" {
		t.Fatalf("modules = %v, want only mod/tool (mod/cache is ignored by the submodule)", rels)
	}
}

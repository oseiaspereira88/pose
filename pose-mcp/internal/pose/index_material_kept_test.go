package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// keptMaterialFixture is a consumer and a dependency repository, each with
// one committed file, and a store whose resolver authorizes both. compute
// counts the builds of the material.
func keptMaterialFixture(t *testing.T) (Store, string, string, *int) {
	t.Helper()
	repo := func() string {
		root := t.TempDir()
		for _, args := range [][]string{{"init", "-q"}} {
			if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v %s", args, err, out)
			}
		}
		if err := os.WriteFile(filepath.Join(root, "spec.md"), []byte("one\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		commitMaterialFixture(t, root)
		return root
	}
	consumer, dependency := repo(), repo()
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"proj.engine": dependency}})}
	builds := 0
	original := indexMaterialCompute
	indexMaterialCompute = func(Store) (IndexFederatedMaterial, error) {
		builds++
		return IndexFederatedMaterial{Projects: []string{"proj.engine"}}, nil
	}
	keptIndexMaterial.clear()
	t.Cleanup(func() { indexMaterialCompute = original; keptIndexMaterial.clear() })
	return Store{Root: consumer, FederatedProjectID: "proj.local", FederatedResolver: &resolver}, consumer, dependency, &builds
}

func commitMaterialFixture(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "c"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

// Spec pose-index-material-is-kept-per-repository-state, R1: the material
// of one state of every authorized repository is built once per process.
func TestIndexMaterialIsBuiltOncePerRepositoryState(t *testing.T) {
	store, _, _, builds := keptMaterialFixture(t)
	first, ok1 := IndexInputBindingForStoreIn(store, "federated-material")
	second, ok2 := IndexInputBindingForStoreIn(store, "federated-material")
	if !ok1 || !ok2 || first != second || *builds != 1 {
		t.Fatalf("builds=%d digests %q %q", *builds, first, second)
	}
}

// R2: any change to an authorized repository's state builds it again: a
// commit, a new untracked file, and a dirty file edited a second time, which
// keeps the same status line.
func TestIndexMaterialFollowsEveryRepositoryState(t *testing.T) {
	for name, change := range map[string]func(t *testing.T, consumer, dependency string){
		"dependency commit": func(t *testing.T, _, dependency string) {
			if err := os.WriteFile(filepath.Join(dependency, "spec.md"), []byte("two\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			commitMaterialFixture(t, dependency)
		},
		"untracked file": func(t *testing.T, consumer, _ string) {
			if err := os.WriteFile(filepath.Join(consumer, "new.md"), []byte("new\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"dirty file edited again": func(t *testing.T, _, dependency string) {
			if err := os.WriteFile(filepath.Join(dependency, "spec.md"), []byte("three\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			store, consumer, dependency, builds := keptMaterialFixture(t)
			if name == "dirty file edited again" {
				if err := os.WriteFile(filepath.Join(dependency, "spec.md"), []byte("dirty\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			IndexInputBindingForStoreIn(store, "federated-material")
			change(t, consumer, dependency)
			IndexInputBindingForStoreIn(store, "federated-material")
			if *builds != 2 {
				t.Fatalf("a %s reused the kept material: %d builds", name, *builds)
			}
		})
	}
}

// R3: a store without an explicit resolver, which names no complete set of
// repositories, builds the material every time.
func TestIndexMaterialWithoutAResolverIsNotKept(t *testing.T) {
	store, _, _, builds := keptMaterialFixture(t)
	store.FederatedResolver = nil
	IndexInputBindingForStoreIn(store, "federated-material")
	IndexInputBindingForStoreIn(store, "federated-material")
	if *builds != 2 {
		t.Fatalf("builds=%d, want 2", *builds)
	}
}

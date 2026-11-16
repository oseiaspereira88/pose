package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func commitMaterialFixture(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "c"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

// keptMaterialFixture is a consumer repository with one committed file the
// material reads, and a store whose resolver authorizes a dependency. The
// material's build is replaced by one that reads the file through the
// operation's Files and counts its builds; unknown makes a build leave its
// closure unknown.
func keptMaterialFixture(t *testing.T) (Store, string, *int, *bool) {
	t.Helper()
	repo := func(body string) string {
		root := t.TempDir()
		if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, out)
		}
		if err := os.WriteFile(filepath.Join(root, "spec.md"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		commitMaterialFixture(t, root)
		return root
	}
	consumer, dependency := repo("one\n"), repo("dependency\n")
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"proj.local": consumer, "proj.engine": dependency}})}
	builds, unknown := 0, false
	original := indexMaterialCompute
	indexMaterialCompute = func(s Store) (IndexFederatedMaterial, error) {
		builds++
		raw, err := s.fsys().ReadFile(filepath.Join(s.Root, "spec.md"))
		if err != nil {
			return IndexFederatedMaterial{}, err
		}
		if unknown {
			markGovernanceReadUnknownForProject(s.fsys(), s.Root, "", keptMaterialProducer, "fixture left the read unknown")
		}
		return IndexFederatedMaterial{Projection: IndexFederatedMaterialProjection{SchemaVersion: len(raw) + int(raw[0])}}, nil
	}
	keptIndexMaterial.Lock()
	keptIndexMaterial.entries = map[string]keptMaterialEntry{}
	keptIndexMaterial.Unlock()
	t.Cleanup(func() {
		indexMaterialCompute = original
		keptIndexMaterial.Lock()
		keptIndexMaterial.entries = map[string]keptMaterialEntry{}
		keptIndexMaterial.Unlock()
	})
	return Store{Root: consumer, FederatedProjectID: "proj.local", FederatedResolver: &resolver}, consumer, &builds, &unknown
}

// inOperation runs the material binding in a fresh read-view operation, as
// one MCP call does, and returns the digest and the view it ran in.
func inOperation(t *testing.T, store Store) (string, *governanceReadView) {
	t.Helper()
	base, view := store.beginGovernanceReadView()
	scoped := base.withGovernanceReadView(view, "review")
	scoped.FederatedResolver = store.FederatedResolver
	scoped.FederatedProjectID = store.FederatedProjectID
	digest, ok := IndexInputBindingForStoreIn(scoped, "federated-material")
	if !ok {
		t.Fatal("the material binding is unknown")
	}
	return digest, view
}

// Spec pose-index-material-is-kept-per-repository-state, R1: a later
// operation reuses the kept material when its closure reads the same, and
// the closure's reads enter that operation's own read view.
func TestIndexMaterialIsReusedWhenItsClosureReadsTheSame(t *testing.T) {
	store, consumer, builds, _ := keptMaterialFixture(t)
	first, _ := inOperation(t, store)
	second, view := inOperation(t, store)
	if first != second || *builds != 1 {
		t.Fatalf("builds=%d digests %q %q", *builds, first, second)
	}
	view.mu.Lock()
	_, observed := view.files[view.pathKey(filepath.Join(consumer, "spec.md"))]
	view.mu.Unlock()
	if !observed {
		t.Fatal("the reused material's reads are not in the current operation's view")
	}
}

// R2: a read of the closure that returns other bytes builds the material
// again, including an edit that keeps the file's size.
func TestIndexMaterialIsBuiltAgainWhenItsClosureChanged(t *testing.T) {
	store, consumer, builds, _ := keptMaterialFixture(t)
	first, _ := inOperation(t, store)
	if err := os.WriteFile(filepath.Join(consumer, "spec.md"), []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, _ := inOperation(t, store)
	if *builds != 2 || first == second {
		t.Fatalf("a changed input reused the kept material: builds=%d", *builds)
	}
}

// R3: a build that left its closure unknown is not kept, and a store with no
// read-view operation builds every time.
func TestIndexMaterialWithAnUnknownClosureIsNotKept(t *testing.T) {
	store, _, builds, unknown := keptMaterialFixture(t)
	*unknown = true
	inOperation(t, store)
	afterFirst := *builds
	inOperation(t, store)
	keptIndexMaterial.Lock()
	kept := len(keptIndexMaterial.entries)
	keptIndexMaterial.Unlock()
	// The isolated build cannot be proved, so each operation answers on a
	// build of its own, and nothing is kept.
	if kept != 0 || *builds <= afterFirst {
		t.Fatalf("an unknown closure was kept or reused: kept=%d builds %d then %d", kept, afterFirst, *builds)
	}
	plain, _, plainBuilds, _ := keptMaterialFixture(t)
	IndexInputBindingForStoreIn(plain, "federated-material")
	IndexInputBindingForStoreIn(plain, "federated-material")
	if *plainBuilds != 2 {
		t.Fatalf("builds without an operation=%d, want 2", *plainBuilds)
	}
}

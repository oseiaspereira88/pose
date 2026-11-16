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

// federatedKeptFixture is a consumer whose spec and roadmap depend on a
// foreign project's spec, built by the real federated builder; builds counts
// them.
func federatedKeptFixture(t *testing.T) (Store, string, string, *int) {
	t.Helper()
	local, foreign := t.TempDir(), t.TempDir()
	for root, project := range map[string]string{local: "proj.local", foreign: "proj.foreign"} {
		for _, dir := range []string{"specs", "roadmaps"} {
			if err := os.MkdirAll(filepath.Join(root, ".pose", dir), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		qualifiedFile(t, root, "project.json", `{"schema_version":1,"project_id":"`+project+`"}`)
	}
	qualifiedFile(t, local, "specs/consumer.md", "---\nslug: consumer\nstatus: in-progress\ndepends_on: xref:proj.foreign/spec:engine\n---\n# Consumer\n")
	qualifiedFile(t, local, "roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:proj.foreign/spec:engine\n---\n")
	qualifiedFile(t, foreign, "specs/engine.md", "---\nslug: engine\nstatus: done\n---\n# Engine v1\n")
	for _, root := range []string{local, foreign} {
		if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, out)
		}
		commitMaterialFixture(t, root)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", `{"proj.local":"`+local+`","proj.foreign":"`+foreign+`"}`)
	resolver, project, err := EnvironmentArtifactResolverIn(NativeFiles(), local, "")
	if err != nil || project != "proj.local" {
		t.Fatalf("resolver = %q, %v", project, err)
	}
	builds := 0
	original := indexMaterialCompute
	indexMaterialCompute = func(s Store) (IndexFederatedMaterial, error) {
		builds++
		return BuildIndexFederatedMaterialIn(s)
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
	return Store{Root: local, FS: NativeFiles(), FederatedProjectID: project, FederatedResolver: &resolver}, local, foreign, &builds
}

// coordinatorBinding asks a fresh coordinator operation over store for the
// material binding of root, as an index check does, and returns the digest
// and the operation.
func coordinatorBinding(t *testing.T, store Store, root string) (string, *governanceReadViewOperation) {
	t.Helper()
	_, view := store.beginGovernanceReadView()
	if view == nil || view.operation == nil {
		t.Fatal("no coordinator operation")
	}
	digest, ok := view.operation.indexInputBinding(view.filesFor("review"), root, "review", "federated-material")
	if !ok {
		t.Fatalf("the material binding of %s is unknown", root)
	}
	return digest, view.operation
}

func operationObserved(op *governanceReadViewOperation, path string) bool {
	for _, v := range operationViews(op) {
		v.mu.Lock()
		_, found := v.files[v.pathKey(path)]
		v.mu.Unlock()
		if found {
			return true
		}
	}
	return false
}

// Review F6: the real builder's foreign reads are in the kept closure, are
// replayed into the reusing operation, and a foreign edit that keeps the
// file's size builds again.
func TestIndexMaterialClosureCoversTheForeignDependency(t *testing.T) {
	store, local, foreign, builds := federatedKeptFixture(t)
	engine := filepath.Join(foreign, ".pose", "specs", "engine.md")
	first, _ := coordinatorBinding(t, store, local)
	second, op := coordinatorBinding(t, store, local)
	if first != second || *builds != 1 {
		t.Fatalf("builds=%d digests %q %q", *builds, first, second)
	}
	if !operationObserved(op, engine) {
		t.Fatal("the foreign dependency is not in the reusing operation's closure")
	}
	qualifiedFile(t, foreign, "specs/engine.md", "---\nslug: engine\nstatus: done\n---\n# Engine v2\n")
	third, _ := coordinatorBinding(t, store, local)
	if third == first || *builds != 2 {
		t.Fatalf("a foreign edit reused the kept material: builds=%d", *builds)
	}
}

// Review F7: a foreign root validated inside a coordinator operation is built
// over its own root, kept, and reused.
func TestIndexMaterialOfAForeignRootIsKeptForThatRoot(t *testing.T) {
	store, _, foreign, builds := federatedKeptFixture(t)
	first, _ := coordinatorBinding(t, store, foreign)
	if *builds != 1 {
		t.Fatalf("the foreign root's first binding built %d times, want 1", *builds)
	}
	second, _ := coordinatorBinding(t, store, foreign)
	if first != second || *builds != 1 {
		t.Fatalf("the foreign root's material was not kept: builds=%d", *builds)
	}
}

// Review F8: the authorization decisions a build consulted are asked again
// on every hit; a changed decision builds again, and the registry's lookup
// outcomes are part of the key.
func TestIndexMaterialIsNotKeptAcrossAuthorizationOrRegistry(t *testing.T) {
	store, local, _, builds := federatedKeptFixture(t)
	allowForeign, asked := true, 0
	authorized := *store.FederatedResolver
	authorized.Authorize = func(project string) bool {
		asked++
		return project != "proj.foreign" || allowForeign
	}
	withAuthorize := store
	withAuthorize.FederatedResolver = &authorized
	first, _ := coordinatorBinding(t, withAuthorize, local)
	askedBefore := asked
	second, _ := coordinatorBinding(t, withAuthorize, local)
	if first != second || *builds != 1 || asked == askedBefore {
		t.Fatalf("same decisions: builds=%d, asked again=%v", *builds, asked != askedBefore)
	}
	allowForeign = false
	third, _ := coordinatorBinding(t, withAuthorize, local)
	if third == first || *builds < 2 {
		t.Fatalf("a revoked authorization reused the kept material: builds=%d", *builds)
	}
	// A referenced project that no root registers: its decision reads no
	// file, so only asking it again can tell unauthorized from unknown.
	qualifiedFile(t, local, "specs/ghost.md", "---\nslug: ghost\nstatus: in-progress\ndepends_on: xref:proj.ghost/spec:missing\n---\n# Ghost\n")
	commitMaterialFixture(t, local)
	allowForeign, allowGhost := true, true
	authorized.Authorize = func(project string) bool {
		switch project {
		case "proj.ghost":
			return allowGhost
		case "proj.foreign":
			return allowForeign
		}
		return true
	}
	ghostAllowed, _ := coordinatorBinding(t, withAuthorize, local)
	buildsBefore := *builds
	allowGhost = false
	ghostDenied, _ := coordinatorBinding(t, withAuthorize, local)
	if *builds == buildsBefore || ghostDenied == ghostAllowed {
		t.Fatalf("a changed decision for an unregistered project reused the kept material: builds %d -> %d", buildsBefore, *builds)
	}
	keyBefore, ok := keptMaterialKey(store)
	if !ok {
		t.Fatal("a plain resolver has no key")
	}
	t.Setenv("POSE_PROJECT_ROOTS", `{"proj.local":"`+local+`"}`)
	narrow, _, err := EnvironmentArtifactResolverIn(NativeFiles(), local, "")
	if err != nil {
		t.Fatal(err)
	}
	narrowed := store
	narrowed.FederatedResolver = &narrow
	if keyAfter, _ := keptMaterialKey(narrowed); keyAfter == keyBefore {
		t.Fatal("a registry without the foreign project has the same key")
	}
}

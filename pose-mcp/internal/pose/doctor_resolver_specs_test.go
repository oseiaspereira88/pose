package pose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func doctorResolverFixture(t *testing.T) (Store, string, string) {
	t.Helper()
	main, source := t.TempDir(), t.TempDir()
	for _, root := range []string{main, source} {
		for i := 0; i < 6; i++ {
			qualifiedFile(t, root, fmt.Sprintf("specs/member-%d.md", i), fmt.Sprintf("---\nslug: member-%d\nstatus: done\n---\n", i))
		}
	}
	federatedTestProject(t, main, "proj.main")
	federatedTestProject(t, source, "proj.source")
	resolver := federatedTestResolver("proj.main", "proj.source", main, source)
	return Store{Root: main, FederatedProjectID: "proj.main", FederatedResolver: &resolver}, main, source
}

func TestDoctorResolverCatalogueIsSharedAndFresh(t *testing.T) {
	store, main, source := doctorResolverFixture(t)
	counts := map[string]int{}
	var mu sync.Mutex
	previous := specCatalogueListed
	specCatalogueListed = func(root string) { mu.Lock(); counts[filepath.Clean(root)]++; mu.Unlock() }
	t.Cleanup(func() { specCatalogueListed = previous })
	run := func(want string) {
		t.Helper()
		before := map[string]int{main: counts[main], source: counts[source]}
		if err := store.WithReviewOperationView(func(shared Store) {
			for n := 0; n < 20; n++ {
				for _, project := range []string{"proj.main", "proj.source"} {
					got := shared.FederatedResolver.Resolve("proj.main", "xref:"+project+"/spec:member-0")
					expected := "done"
					if project == "proj.source" {
						expected = want
					}
					if !got.Resolved || got.Status != expected {
						t.Fatalf("%s: %+v", project, got)
					}
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		for _, root := range []string{main, source} {
			if got := counts[root] - before[root]; got != 1 {
				t.Fatalf("catalogue parses for %s = %d, want 1 per operation", root, got)
			}
		}
	}
	run("done")
	path := filepath.Join(source, ".pose/specs/member-0.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "done", "draft", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	run("draft")
}

func TestDoctorResolverCatalogueChangeFailsFinalAudit(t *testing.T) {
	store, _, source := doctorResolverFixture(t)
	err := store.WithReviewOperationView(func(shared Store) {
		ref := "xref:proj.source/spec:member-0"
		first := shared.FederatedResolver.Resolve("proj.main", ref)
		if !first.Resolved || first.Status != "done" {
			t.Fatalf("first=%+v", first)
		}
		path := filepath.Join(source, ".pose/specs/member-0.md")
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		info, e := os.Stat(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, []byte(strings.Replace(string(raw), "done", "open", 1)), 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.Chtimes(path, info.ModTime(), info.ModTime()); e != nil {
			t.Fatal(e)
		}
		shared.FederatedResolver.Resolve("proj.main", ref)
	})
	if !errors.Is(err, ErrGovernanceReadViewChanged) {
		t.Fatalf("same-size/mtime change accepted: %v", err)
	}
}

// A shared parse must not select a missing or conflicting identity, or bypass
// the resolver's authorization; compare the entire answer with the live path.
func TestDoctorResolverCataloguePreservesResolution(t *testing.T) {
	for _, mode := range []string{"ordinary", "duplicate", "unauthorized"} {
		t.Run(mode, func(t *testing.T) {
			store, _, source := doctorResolverFixture(t)
			if mode == "duplicate" {
				qualifiedFile(t, source, "specs/2026-11-17-member-0.md", "---\nslug: member-0\nstatus: done\n---\n")
			}
			if mode == "unauthorized" {
				store.FederatedResolver.Authorize = func(project string) bool { return project != "proj.source" }
			}
			refs := []string{"xref:proj.source/spec:member-0", "xref:proj.main/spec:member-1", "xref:proj.source/spec:missing", "xref:proj.absent/spec:member-0"}
			want := make([]ArtifactResolution, len(refs))
			for i, ref := range refs {
				want[i] = store.FederatedResolver.Resolve("proj.main", ref)
			}
			err := store.WithReviewOperationView(func(shared Store) {
				for round := 0; round < 2; round++ {
					for i, ref := range refs {
						got := shared.FederatedResolver.Resolve("proj.main", ref)
						if !reflect.DeepEqual(got, want[i]) {
							t.Fatalf("%s: shared=%+v live=%+v", ref, got, want[i])
						}
					}
				}
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

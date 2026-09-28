package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func reviewArchiveFixture(t *testing.T) (string, Store, DeliveryIntegrityGraph, ReviewBundleSubject) {
	t.Helper()
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, fragmentPending, "reviewed release fragment\r\n")
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.name", "Fixture")
	designDeltaGit(t, root, "config", "user.email", "fixture@example.invalid")
	designDeltaGit(t, root, "add", ".")
	designDeltaGit(t, root, "commit", "-qm", "Fixture")
	graph := DeliveryIntegrityGraph{ChangeSets: []ChangeSet{{ID: "cs-backend", Spec: "backend", ResolvedBase: "base", ResolvedHead: "head", Paths: []ObservedPath{{Action: "created", Path: fragmentPending}}}}}
	subject, _, blockers, err := store.reviewBundleSubject(ScopeRef{Kind: "spec", Slug: "backend"}, []ReviewPlanComponent{{Path: "api"}}, graph, nil)
	if err != nil || len(blockers) != 0 {
		t.Fatalf("pending subject: %v %v", err, blockers)
	}
	archiveReviewFragment(t, root, "v1.2.0", "backend")
	return root, store, graph, subject
}

func archiveReviewFragment(t *testing.T, root, version, spec string) {
	t.Helper()
	archive := ".pose/changelogs/" + version + "/alpha.md"
	manifest := ".pose/releases/" + version + "/manifest.json"
	raw := []byte("reviewed release fragment\r\n")
	writeReviewFixture(t, root, archive, string(raw))
	data, err := json.Marshal(ReleaseManifest{SchemaVersion: 1, Version: version, Fragments: []ReleaseFragment{{Spec: spec, Path: "alpha.md", Digest: ReleaseDigest(string(raw))}}})
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, manifest, string(data))
	if err := os.Remove(filepath.Join(root, fragmentPending)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	designDeltaGit(t, root, "add", ".")
	designDeltaGit(t, root, "commit", "-qm", "Archive release")
}

func TestReviewReleaseArchivePreservesSubject(t *testing.T) {
	_, store, graph, before := reviewArchiveFixture(t)
	after, _, blockers, err := store.reviewBundleSubject(ScopeRef{Kind: "spec", Slug: "backend"}, []ReviewPlanComponent{{Path: "api"}}, graph, nil)
	if err != nil || len(blockers) != 0 {
		t.Fatalf("committed archive must preserve subject: %v %v", err, blockers)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("archival changed semantic subject: before=%+v after=%+v", before, after)
	}
}

func TestReviewReleaseArchiveRejectsUnattestedContent(t *testing.T) {
	cases := map[string]func(*testing.T, string){
		"dirty archive":      func(t *testing.T, root string) { writeReviewFixture(t, root, fragmentArchived, "changed") },
		"untracked archive":  func(t *testing.T, root string) { designDeltaGit(t, root, "rm", "--cached", fragmentArchived) },
		"untracked manifest": func(t *testing.T, root string) { designDeltaGit(t, root, "rm", "--cached", fragmentManifest) },
		"forged uncommitted manifest and archive": func(t *testing.T, root string) {
			writeReviewFixture(t, root, fragmentArchived, "forged")
			raw, _ := json.Marshal(ReleaseManifest{SchemaVersion: 1, Version: "v1.2.0", Fragments: []ReleaseFragment{{Spec: "backend", Path: "alpha.md", Digest: ReleaseDigest("forged")}}})
			writeReviewFixture(t, root, fragmentManifest, string(raw))
		},
		"wrong spec": func(t *testing.T, root string) {
			raw, _ := json.Marshal(ReleaseManifest{SchemaVersion: 1, Version: "v1.2.0", Fragments: []ReleaseFragment{{Spec: "other", Path: "alpha.md", Digest: ReleaseDigest("reviewed release fragment\r\n")}}})
			writeReviewFixture(t, root, fragmentManifest, string(raw))
			designDeltaGit(t, root, "add", ".")
			designDeltaGit(t, root, "commit", "-qm", "Wrong owner")
		},
		"duplicate release": func(t *testing.T, root string) { archiveReviewFragment(t, root, "v1.3.0", "backend") },
		"archive escapes root": func(t *testing.T, root string) {
			outside := filepath.Join(t.TempDir(), "fragment.md")
			if err := os.WriteFile(outside, []byte("reviewed release fragment\r\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, fragmentArchived)); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(root, fragmentArchived)); err != nil {
				t.Fatal(err)
			}
			designDeltaGit(t, root, "add", ".")
			designDeltaGit(t, root, "commit", "-qm", "Symlink")
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			root, store, graph, _ := reviewArchiveFixture(t)
			mutate(t, root)
			_, _, blockers, err := store.reviewBundleSubject(ScopeRef{Kind: "spec", Slug: "backend"}, []ReviewPlanComponent{{Path: "api"}}, graph, nil)
			if err == nil && len(blockers) == 0 {
				t.Fatal("unattested archival accepted")
			}
		})
	}
}

func TestReviewReleaseArchiveDoesNotResolveOtherMissingPaths(t *testing.T) {
	_, store, _, _ := reviewArchiveFixture(t)
	graph := DeliveryIntegrityGraph{ChangeSets: []ChangeSet{{ID: "cs-backend", Spec: "backend", Paths: []ObservedPath{{Action: "created", Path: ".pose/knowledge/missing.md"}}}}}
	_, _, blockers, err := store.reviewBundleSubject(ScopeRef{Kind: "spec", Slug: "backend"}, []ReviewPlanComponent{{Path: "api"}}, graph, nil)
	if err == nil && len(blockers) == 0 {
		t.Fatal("missing unrelated subject was accepted")
	}
}

package pose

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDocsSyncBundle_NoManifestErrors(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root}
	if _, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo"); err == nil {
		t.Fatal("expected error without a manifest")
	}
}

func TestBuildDocsSyncBundle_ExcludesSensitiveAndMissing(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody A\n")
	writeDocsFile(t, root, "docs/b.md", "---\ntitle: B\ndoc_type: reference\n---\nbody B\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
		{Path: "docs/b.md", DocType: "reference", Sensitive: true},
		{Path: "docs/missing.md", DocType: "reference"},
	}}
	writeManifestFile(t, root, manifest)

	store := Store{Root: root}
	bundle, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Docs) != 1 || bundle.Docs[0].Path != "docs/a.md" {
		t.Fatalf("expected only docs/a.md in the bundle, got %+v", bundle.Docs)
	}
	if bundle.ExcludedSensitive != 1 {
		t.Fatalf("expected 1 excluded sensitive doc, got %d", bundle.ExcludedSensitive)
	}
	if bundle.Docs[0].Content != "---\ntitle: A\ndoc_type: reference\n---\nbody A\n" {
		t.Fatalf("unexpected content: %q", bundle.Docs[0].Content)
	}
	if bundle.Docs[0].ContentHash == "" {
		t.Fatal("expected a non-empty content hash")
	}
}

func TestBuildDocsSyncBundle_HashIsDeterministicAcrossCalls(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	writeManifestFile(t, root, manifest)

	store := Store{Root: root}
	b1, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	b2, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	if b1.BundleHash != b2.BundleHash {
		t.Fatalf("expected stable bundle hash across calls, got %q vs %q", b1.BundleHash, b2.BundleHash)
	}
}

func TestBuildDocsSyncBundle_IncludesReviewPendingAndCheckCounts(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "# no frontmatter\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	writeManifestFile(t, root, manifest)
	writeDocsReviewJSONL(t, root, []DocsReviewEvent{
		{At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x"},
	})

	store := Store{Root: root}
	bundle, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Docs) != 1 {
		t.Fatalf("expected 1 doc, got %+v", bundle.Docs)
	}
	doc := bundle.Docs[0]
	if !doc.ReviewPending {
		t.Fatal("expected review_pending=true")
	}
	if doc.CheckWarnings != 1 {
		t.Fatalf("expected 1 missing_frontmatter warning, got %+v", doc)
	}
}

func writeManifestFile(t *testing.T, root string, manifest *DocsManifest) {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".pose", "docs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

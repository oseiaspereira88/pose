package pose

import (
	"bytes"
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

// harne8-portal-wiki-surface R18: a barreira de sensibilidade não é
// "o path não aparece na lista" — é que nenhum byte do documento sai do
// repositório. Este teste serializa o bundle inteiro, que é o que de fato
// viaja para o Conductor, e procura o conteúdo lá dentro.
func TestBuildDocsSyncBundle_SensitiveContentNeverReachesTheWire(t *testing.T) {
	root := t.TempDir()
	const segredo = "origin em localhost:8788 atras do tunnel harne8-platform-prod"
	writeDocsFile(t, root, "docs/publica.md", "---\ntitle: Publica\ndoc_type: reference\n---\nconteudo publico\n")
	writeDocsFile(t, root, "docs/runbook.md", "---\ntitle: Runbook\ndoc_type: howto\n---\n"+segredo+"\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/publica.md", DocType: "reference"},
		{Path: "docs/runbook.md", DocType: "howto", Sensitive: true},
	}}
	writeManifestFile(t, root, manifest)

	store := Store{Root: root}
	bundle, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{segredo, "localhost:8788", "harne8-platform-prod", "docs/runbook.md"} {
		if bytes.Contains(wire, []byte(forbidden)) {
			t.Fatalf("bundle serializado carrega %q de um doc sensível:\n%s", forbidden, wire)
		}
	}
	if bundle.ExcludedSensitive != 1 {
		t.Fatalf("esperava 1 doc retido, veio %d", bundle.ExcludedSensitive)
	}
	// A retenção é contada, não silenciosa: sem esse número a projeção
	// parece completa para quem a consome.
	if len(bundle.Docs) != 1 {
		t.Fatalf("esperava só o doc público no bundle, veio %+v", bundle.Docs)
	}
}

// Marcar sensitive depois do fato retira o documento de exports seguintes:
// a classificação é reversível na direção que importa para vazamento.
func TestBuildDocsSyncBundle_MarcarSensitiveRetiraDoBundle(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody A\n")
	entry := DocsEntry{Path: "docs/a.md", DocType: "reference"}
	writeManifestFile(t, root, &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{entry}})

	store := Store{Root: root}
	antes, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(antes.Docs) != 1 {
		t.Fatalf("esperava o doc no bundle antes da marcação, veio %+v", antes.Docs)
	}

	entry.Sensitive = true
	writeManifestFile(t, root, &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{entry}})
	depois, err := store.BuildDocsSyncBundle(context.Background(), "proj.demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(depois.Docs) != 0 || depois.ExcludedSensitive != 1 {
		t.Fatalf("marcar sensitive deveria retirar do bundle, veio %+v (excluded=%d)", depois.Docs, depois.ExcludedSensitive)
	}
	if antes.BundleHash == depois.BundleHash {
		t.Fatal("o hash do bundle deveria mudar quando um doc é retido")
	}
}

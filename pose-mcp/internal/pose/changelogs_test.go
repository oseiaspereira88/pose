package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func changelogStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/changelogs/unreleased/spec-b.md", `---
spec: spec-b
category: fixed
breaking: false
refs: PR#7
---

Corrige o parser de eventos.
`)
	write(".pose/changelogs/unreleased/spec-a.md", `---
spec: spec-a
category: added
breaking: true
refs:
---

<!-- comentário de template ignorado -->
Nova API pública de rollup.
`)
	write(".pose/changelogs/v0.1.0.md", "# v0.1.0\n\n## Added\n- primeira release\n")
	return Store{Root: root}
}

func TestGetChangelogUnreleasedAndReleases(t *testing.T) {
	s := changelogStore(t)
	c, err := s.GetChangelog("")
	if err != nil {
		t.Fatalf("GetChangelog: %v", err)
	}
	if len(c.Unreleased) != 2 {
		t.Fatalf("unreleased = %d, want 2", len(c.Unreleased))
	}
	// ordenados por spec
	if c.Unreleased[0].Spec != "spec-a" || !c.Unreleased[0].Breaking || c.Unreleased[0].Category != "added" {
		t.Fatalf("frag[0] = %+v", c.Unreleased[0])
	}
	if !strings.Contains(c.Unreleased[0].Body, "Nova API") || strings.Contains(c.Unreleased[0].Body, "comentário") {
		t.Fatalf("body deve ignorar comentários de template: %q", c.Unreleased[0].Body)
	}
	if len(c.Releases) != 1 || c.Releases[0] != "v0.1.0" {
		t.Fatalf("releases = %v, want [v0.1.0]", c.Releases)
	}
}

func TestGetChangelogVersionBody(t *testing.T) {
	s := changelogStore(t)
	c, err := s.GetChangelog("v0.1.0")
	if err != nil {
		t.Fatalf("GetChangelog(v0.1.0): %v", err)
	}
	if c.Version != "v0.1.0" || !strings.Contains(c.VersionBody, "primeira release") {
		t.Fatalf("version body = %+v", c)
	}
	if _, err := s.GetChangelog("v9.9.9"); err == nil {
		t.Fatal("versão inexistente deve dar erro")
	}
	if _, err := s.GetChangelog("../etc"); err == nil {
		t.Fatal("versão com traversal deve dar erro")
	}
}

func TestGetChangelogEmptyRepo(t *testing.T) {
	s := Store{Root: t.TempDir()}
	c, err := s.GetChangelog("")
	if err != nil {
		t.Fatalf("GetChangelog: %v", err)
	}
	if len(c.Unreleased) != 0 || len(c.Releases) != 0 {
		t.Fatalf("repo vazio deve retornar vazio: %+v", c)
	}
}

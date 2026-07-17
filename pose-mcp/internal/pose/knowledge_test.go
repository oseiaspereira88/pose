package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListKnowledge_FiltersRestricted(t *testing.T) {
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

	write(".pose/knowledge/handbook.md", "---\nslug: handbook\ntype: handoff\nowner: @platform\nsensitivity: public-internal\n---\n\n# Handbook\n\nTeam processes.")
	write(".pose/knowledge/secret.md", "---\nslug: secret\ntype: decision-log\nowner: @security\nsensitivity: restricted\n---\n\n# Secret Decision\n\nSensitive details.")

	store := Store{Root: root}
	list, err := store.ListKnowledge()
	if err != nil {
		t.Fatalf("ListKnowledge: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("count = %d, want 1", len(list))
	}
	if list[0].Slug != "handbook" {
		t.Errorf("slug = %q, want handbook", list[0].Slug)
	}
	if list[0].Body != "" {
		t.Errorf("body should be empty on list")
	}
}

func TestGetKnowledge_PublicInternal(t *testing.T) {
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

	write(".pose/knowledge/handbook.md", "---\nslug: handbook\ntype: handoff\nowner: @platform\nsensitivity: public-internal\ncreated_at: 2026-06-01\n---\n\n# Handbook\n\nTeam processes.")

	store := Store{Root: root}
	ke, err := store.GetKnowledge("handbook")
	if err != nil {
		t.Fatalf("GetKnowledge: %v", err)
	}
	if ke.Slug != "handbook" {
		t.Errorf("slug = %q, want handbook", ke.Slug)
	}
	if ke.Type != "handoff" {
		t.Errorf("type = %q, want handoff", ke.Type)
	}
	if ke.Sensitivity != "public-internal" {
		t.Errorf("sensitivity = %q, want public-internal", ke.Sensitivity)
	}
	if !strings.Contains(ke.Body, "Team processes") {
		t.Errorf("body missing expected content: %q", ke.Body)
	}
}

func TestGetKnowledge_Restricted(t *testing.T) {
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

	write(".pose/knowledge/secret.md", "---\nslug: secret\ntype: decision-log\nowner: @security\nsensitivity: restricted\n---\n\n# Secret\n\nDo not share.")

	store := Store{Root: root}
	_, err := store.GetKnowledge("secret")
	if err == nil {
		t.Fatal("expected error for restricted entry")
	}
	if !strings.Contains(err.Error(), "restricted") {
		t.Errorf("error message missing 'restricted': %v", err)
	}
}

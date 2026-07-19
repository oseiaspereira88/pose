package pose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListExtensionsEmptyWhenNoLockFile(t *testing.T) {
	s := Store{Root: t.TempDir()}
	items, err := s.ListExtensions()
	if err != nil || len(items) != 0 {
		t.Fatalf("items=%v err=%v, want empty/nil", items, err)
	}
}

func TestListExtensionsReadsLockFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "indexes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	lock := `{"schema_version":1,"extensions":{"acme-skill":{"version":"1.0.0","kind":"skill","installed_at":"2026-07-19T00:00:00Z","digest":"abc","files":{".agents/skills/acme-skill/SKILL.md":"deadbeef"},"signature_verified":true}}}`
	if err := os.WriteFile(filepath.Join(dir, "extensions.lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Store{Root: root}
	items, err := s.ListExtensions()
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[0].ID != "acme-skill" || !items[0].SignatureVerified || len(items[0].Files) != 1 {
		t.Errorf("item = %+v", items[0])
	}
}

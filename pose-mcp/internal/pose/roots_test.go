package pose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoots_StoreFor(t *testing.T) {
	roots := NewRoots(RootsConfig{
		DefaultRoot:      "/default",
		DefaultProjectID: "proj.crisol",
		Explicit:         map[string]string{"proj.foo": "/foo"},
	})

	t.Run("empty resolves to default", func(t *testing.T) {
		s, err := roots.StoreFor("")
		if err != nil || s.Root != "/default" {
			t.Fatalf("got (%q,%v), want /default", s.Root, err)
		}
	})
	t.Run("default project id maps to default root", func(t *testing.T) {
		s, err := roots.StoreFor("proj.crisol")
		if err != nil || s.Root != "/default" {
			t.Fatalf("got (%q,%v), want /default", s.Root, err)
		}
	})
	t.Run("known project resolves to its root", func(t *testing.T) {
		s, err := roots.StoreFor("proj.foo")
		if err != nil || s.Root != "/foo" {
			t.Fatalf("got (%q,%v), want /foo", s.Root, err)
		}
	})
	t.Run("unknown project errors (no silent fallback)", func(t *testing.T) {
		if _, err := roots.StoreFor("proj.ghost"); err == nil {
			t.Fatal("want error for unknown project_id")
		}
	})
}

func TestRoots_ExplicitWinsOverDefaultMapping(t *testing.T) {
	roots := NewRoots(RootsConfig{
		DefaultRoot:      "/default",
		DefaultProjectID: "proj.crisol",
		Explicit:         map[string]string{"proj.crisol": "/explicit"},
	})
	s, err := roots.StoreFor("proj.crisol")
	if err != nil || s.Root != "/explicit" {
		t.Fatalf("got (%q,%v), want /explicit (explicit wins)", s.Root, err)
	}
}

func TestRoots_RescanOnMissFindsNewProject(t *testing.T) {
	base := t.TempDir()
	roots := NewRoots(RootsConfig{ProjectsDir: base})
	roots.rescanWindow = 0 // disable throttle for the test

	// Not present yet.
	if _, err := roots.StoreFor("late"); err == nil {
		t.Fatal("want error before the project exists")
	}
	// Materialize a project dir with .pose after construction.
	root := filepath.Join(base, "late")
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A subsequent miss rescans and now resolves it (no restart).
	s, err := roots.StoreFor("late")
	if err != nil || s.Root != root {
		t.Fatalf("got (%q,%v), want %q after rescan", s.Root, err, root)
	}
}

func TestRoots_RescanThrottled(t *testing.T) {
	base := t.TempDir()
	roots := NewRoots(RootsConfig{ProjectsDir: base})
	// Throttle window is large by default; a fresh dir created right after a miss
	// should NOT be picked up until the window elapses.
	root := filepath.Join(base, "fast")
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	// First miss just rebuilt at construction (<2s ago), so maybeRescan is a no-op.
	if _, err := roots.StoreFor("fast"); err == nil {
		t.Fatal("want throttle to suppress immediate rescan")
	}
}

func TestScanProjectsDir(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "withpose", ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "nopose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "afile"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty prefix: dirname IS the project_id.
	got, err := ScanProjectsDir(base, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["withpose"] != filepath.Join(base, "withpose") {
		t.Fatalf("got %v, want {withpose: .../withpose}", got)
	}
}

func TestScanProjectsDir_MissingBaseIsEmpty(t *testing.T) {
	got, err := ScanProjectsDir(filepath.Join(t.TempDir(), "does-not-exist"), "")
	if err != nil || len(got) != 0 {
		t.Fatalf("got (%v,%v), want empty/no error", got, err)
	}
	empty, err := ScanProjectsDir("", "")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty base: got (%v,%v)", empty, err)
	}
}

func TestParseRootsJSON(t *testing.T) {
	got, err := ParseRootsJSON(`{"proj.a":"/a","proj.b":"/b"}`)
	if err != nil || got["proj.a"] != "/a" || got["proj.b"] != "/b" {
		t.Fatalf("got (%v,%v)", got, err)
	}
	empty, err := ParseRootsJSON("")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty: got (%v,%v)", empty, err)
	}
	if _, err := ParseRootsJSON("{not json"); err == nil {
		t.Fatal("want error for invalid json")
	}
}

package pose

import (
	"os"
	"path/filepath"
	"testing"
)

func pointerFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/demo-spec/spec.md", "---\nslug: demo-spec\nstatus: done\n---\n\n# Spec: demo-spec\n")
	write(".pose/knowledge/2026-07-21-note-demo-note.md", "---\nslug: demo-note\ntype: note\n---\n\nbody\n")
	write(".pose/adr/2026-07-21-demo-decision.md", "# ADR: demo\n")
	write(".pose/reports/2026-07-21-demo-review.md", "# Report\n")
	write("README.md", "# demo\n")
	return root
}

func TestResolvePointer_AllKindsResolve(t *testing.T) {
	store := Store{Root: pointerFixtureRoot(t)}
	cases := []string{
		"spec:demo-spec",
		"knowledge:demo-note",
		"adr:2026-07-21-demo-decision.md",
		"report:2026-07-21-demo-review.md",
		"doc:README.md",
		"commit:abc1234",
		"check:go test ./...",
		"url:https://example.com",
		"component:conductor-internal-policy",
	}
	for _, ref := range cases {
		if ok, reason := store.ResolvePointer(ref); !ok {
			t.Errorf("ResolvePointer(%q) = false, %q; want true", ref, reason)
		}
	}
}

func TestResolvePointer_RejectsBrokenAndMalformed(t *testing.T) {
	store := Store{Root: pointerFixtureRoot(t)}
	cases := []string{
		"spec:ghost-spec",
		"knowledge:ghost-note",
		"adr:ghost.md",
		"report:ghost.md",
		"doc:ghost.md",
		"doc:../outside.md",
		"commit:not-hex",
		"url:http://insecure",
		"nope:whatever",
		"no-colon-at-all",
		"empty:",
	}
	for _, ref := range cases {
		if ok, reason := store.ResolvePointer(ref); ok {
			t.Errorf("ResolvePointer(%q) = true; want false (a reason)", ref)
		} else if reason == "" {
			t.Errorf("ResolvePointer(%q) returned false with an empty reason", ref)
		}
	}
}

func TestResolvePointer_ComponentNeverExistenceChecked(t *testing.T) {
	store := Store{Root: pointerFixtureRoot(t)}
	// component: identity lives in GraphForge, never checked locally — any
	// non-empty value after the colon must resolve.
	if ok, reason := store.ResolvePointer("component:anything-goes"); !ok {
		t.Errorf("component pointer must never be existence-checked locally, got false: %q", reason)
	}
}

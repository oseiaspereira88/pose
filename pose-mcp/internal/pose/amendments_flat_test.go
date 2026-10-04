package pose

import (
	"os"
	"path/filepath"
	"testing"
)

// Spec pose-flat-spec-amendments.
func TestAmendmentsPathIsPerSpecForFlatAndUnchangedForFolders(t *testing.T) {
	root := t.TempDir()
	specs := filepath.Join(root, ".pose", "specs")
	folder := filepath.Join(specs, "2026-08-07-folder")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		filepath.Join(folder, "spec.md"):          filepath.Join(folder, "amendments.jsonl"),
		folder:                                    filepath.Join(folder, "amendments.jsonl"),
		filepath.Join(specs, "2026-10-04-a.md"):   filepath.Join(specs, "2026-10-04-a.amendments.jsonl"),
		filepath.Join(specs, "2026-10-04-b.md"):   filepath.Join(specs, "2026-10-04-b.amendments.jsonl"),
		filepath.Join(specs, "legacy", "spec.md"): filepath.Join(specs, "legacy", "amendments.jsonl"),
	}
	seen := map[string]string{}
	for spec, want := range cases {
		got := AmendmentsPath(spec)
		if got != want {
			t.Errorf("AmendmentsPath(%s) = %s, want %s", spec, got, want)
		}
		if other, dup := seen[got]; dup && other != spec && filepath.Dir(other) != spec && filepath.Dir(spec) != other {
			t.Errorf("%s and %s share journal %s", spec, other, got)
		}
		seen[got] = spec
	}
	if AmendmentsPath(filepath.Join(specs, "2026-10-04-a.md")) == AmendmentsPath(filepath.Join(specs, "2026-10-04-b.md")) {
		t.Fatal("two flat specs in one directory share a journal")
	}
}

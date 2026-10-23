package pose

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateOperationReleaseReadsStagedInventoryAndProvesAbsence(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	// Preserve the intended bytes, then remove their physical counterparts before
	// staging: neither successful reads nor absence checks can borrow from disk.
	paths := []string{".pose/policy/release.json", ".pose/releases/v1.0.0/manifest.json", ".pose/changelogs/v1.0.0/alpha.md", ".pose/changelogs/v1.0.0.md", "VERSION"}
	bytes := map[string][]byte{}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(r.root, rel))
		if err != nil {
			t.Fatal(err)
		}
		bytes[rel] = raw
	}
	for _, rel := range []string{".pose/releases", ".pose/changelogs/v1.0.0", ".pose/changelogs/v1.0.0.md", ".pose/policy/release.json", "VERSION"} {
		if err := os.RemoveAll(filepath.Join(r.root, rel)); err != nil {
			t.Fatal(err)
		}
	}
	p := NewPreviewFiles(time.Now())
	for _, rel := range paths {
		path := filepath.Join(r.root, rel)
		if err := p.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := p.WriteFile(path, bytes[rel], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	status, err := (Store{Root: r.root, FS: p}).GetReleaseStatus("")
	if err != nil || len(status.Releases) != 1 || !status.Releases[0].Sound() {
		t.Fatalf("staged release inventory: %v %+v", err, status)
	}
	archived := LoadArchivedFragmentsIn(p, r.root)
	if len(archived) != 1 || !archived[0].Intact {
		t.Fatalf("staged archive witness: %+v", archived)
	}
	if absent, err := ProvenAbsentIn(p, filepath.Join(r.root, ".pose/releases/v1.0.0/manifest.json")); err != nil || absent {
		t.Fatalf("staged manifest called absent: %v %v", absent, err)
	}
	if _, err := LoadReleasePolicy(r.root); !errors.Is(err, ErrReleasePolicyMissing) {
		t.Fatalf("physical policy should remain absent: %v", err)
	}
	if err := p.Remove(filepath.Join(r.root, ".pose/policy/release.json")); err != nil {
		t.Fatal(err)
	}
	if err := p.Symlink("missing-release.json", filepath.Join(r.root, ".pose/policy/release.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReleasePolicyIn(p, r.root); err == nil || errors.Is(err, ErrReleasePolicyMissing) {
		t.Fatalf("staged broken policy link became proven absence: %v", err)
	}
}

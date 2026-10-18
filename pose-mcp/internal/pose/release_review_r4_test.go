package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the fourth independent review of pose-release-manifest-tag-integrity.

// R4-F01: a fragment body that spans lines is text, and the manifest that freezes it loads.
func TestR4MultilineBodyIsFrozenAndLoads(t *testing.T) {
	body := "---\nspec: alpha\ncategory: fixed\nbreaking: false\n---\n\nFixes alpha.\n\nSecond paragraph.\n"
	fragments := []ReleaseFragment{{Spec: "alpha", Category: "fixed", Path: "alpha.md", Body: "Fixes alpha.\n\nSecond paragraph.", Digest: ReleaseDigest(body)}}
	manifest := NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", fragments, fixtureReleasePolicy(), ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": "text", "value": "v1.0.0"})
	if _, issues := ValidateReleaseManifestBytes(CanonicalJSON(manifest), "v1.0.0"); len(issues) != 0 {
		t.Fatalf("the manifest a producer wrote is refused: %v", issues)
	}
}

// R4-F02: a link in the tagged tree is not a file.
func TestR4ALinkInPlaceOfTheVersionSourceIsNotItsContent(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	os.Remove(filepath.Join(r.root, "VERSION"))
	relWrite(t, r.root, "real-version", "9.9.9\n")
	if err := os.Symlink("real-version", filepath.Join(r.root, "VERSION")); err != nil {
		t.Skip("symlinks are not available")
	}
	if snap := snapshotOf(t, tagged(t, r)); snap.Intact() || !hasText(snap.Gaps, "not a regular file") {
		t.Fatalf("the bytes of a link were read as a version file: %+v", snap)
	}
}

// R4-F04: a candidate that lost the files it froze is not sound.
func TestR4APreparedStatusSeesTheFilesItLost(t *testing.T) {
	for name, victim := range map[string]string{"notes": ".pose/changelogs/v1.0.0.md", "fragment": ".pose/changelogs/v1.0.0/alpha.md"} {
		r := newReleaseRepo(t, "v1.0.0")
		os.Remove(filepath.Join(r.root, filepath.FromSlash(victim)))
		status, err := (Store{Root: r.root}).GetReleaseStatus("v1.0.0")
		if err != nil || len(status.Releases) != 1 || status.Releases[0].Sound() {
			t.Errorf("%s: a candidate that lost its file was sound: %+v %v", name, status, err)
		}
	}
}

// R4-F05: a verified release that is not sound still owes.
func TestR4AnUnsoundVerifiedReleaseStillOwes(t *testing.T) {
	r := tagged(t, newReleaseRepo(t, "v1.0.0"))
	relWrite(t, r.root, ".pose/releases/v1.0.0/events.jsonl", `{"version":"v1.0.0","state":"tagged"}
{"version":"v1.0.0","state":"published","evidence_digest":"sha256:p"}
{"version":"v1.0.0","state":"verified","evidence":{"publication_digest":"sha256:p"}}
`)
	os.Remove(filepath.Join(r.root, ".pose/changelogs/v1.0.0.md"))
	items, err := (Store{Root: r.root}).releaseQueueObligations("proj.test")
	if err != nil || len(items) != 1 {
		t.Fatalf("a verified release with a lost file was resolved: %+v %v", items, err)
	}
}

// R4-F06: a manifest of the working tree that is not the cut of the tag attests nothing.
func TestR4ArchivalIsBoundToTheCutOfTheTag(t *testing.T) {
	r := tagged(t, newReleaseRepo(t, "v1.0.0"))
	if got := LoadArchivedFragments(r.root); len(got) != 1 || !got[0].Intact {
		t.Fatalf("the control witness is not intact: %+v", got)
	}
	body := "---\nspec: beta\ncategory: fixed\nbreaking: false\n---\n\nFixes beta.\n"
	other := []ReleaseFragment{{Spec: "beta", Category: "fixed", Path: "alpha.md", Body: "Fixes beta.", Digest: ReleaseDigest(body)}}
	r.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", other, fixtureReleasePolicy(), ChangelogPolicy{}, r.manifest.VersionEvidence)
	r.body = body
	r.notes = RenderReleaseNotes("v1.0.0", other, ChangelogPolicy{})
	r.writeAll(t)
	r.commit(t, "another cut under the same tag")
	got := LoadArchivedFragments(r.root)
	if len(got) != 1 || got[0].Intact {
		t.Fatalf("a cut that is not the one the tag holds attested a claim: %+v", got)
	}
}

// R4-F07: what Git could not read is not what is not there.
func TestR4ALostReleaseDirectoryAndPolicyAreNotAbsence(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	os.RemoveAll(filepath.Join(r.root, ".pose", "releases"))
	if err := os.Symlink(filepath.Join(r.root, "gone"), filepath.Join(r.root, ".pose", "releases")); err != nil {
		t.Skip("symlinks are not available")
	}
	if status, err := (Store{Root: r.root}).GetReleaseStatus(""); err == nil {
		t.Fatalf("a dangling release directory read as no releases: %+v", status)
	}
	// A primary policy that is there and cannot be read does not fall back to the legacy one.
	p := t.TempDir()
	os.MkdirAll(filepath.Join(p, ".pose", "policy", "release.json"), 0o755)
	relWrite(t, p, ".pose/release-policy.json", `{"schema_version":1}`)
	if _, err := LoadReleasePolicy(p); err == nil || !strings.Contains(err.Error(), "could not be read") {
		t.Fatalf("the legacy policy replaced a primary one that could not be read: %v", err)
	}
}

// R4-F08: a cutoff is a version, and a changelog policy of a schema this engine does not read is a gap.
func TestR4TheCutoffIsAVersionAndTheChangelogSchemaIsSupported(t *testing.T) {
	if err := ValidateReleasePolicy(ReleasePolicy{SchemaVersion: 1, LegacyCutoff: "not-a-version"}); err == nil {
		t.Fatal("a cutoff that is not a version was accepted")
	}
	if err := ValidateReleasePolicy(ReleasePolicy{SchemaVersion: 1, LegacyCutoff: "v1.2.0"}); err != nil {
		t.Fatalf("a cutoff that is a version was refused: %v", err)
	}
	r := newReleaseRepo(t, "v1.0.0")
	relWrite(t, r.root, ".pose/policy/changelog.json", `{"schema_version":999}`)
	if snap := snapshotOf(t, tagged(t, r)); !hasText(snap.Gaps, "schema_version 999") {
		t.Fatalf("a changelog policy of a future schema was taken as complete proof: %+v", snap.Gaps)
	}
}

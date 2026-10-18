package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the second independent review of pose-release-manifest-tag-integrity.

// R2-F01: the objects inside the manifest are held to the contract of the root.
func TestR2NestedFragmentObjectsAreStrict(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	good := string(CanonicalJSON(r.manifest))
	cases := map[string]string{
		"breaking absent": strings.Replace(good, `"breaking": false,
      "body"`, `"body"`, 1),
		"upper case name": strings.Replace(good, `"spec": "alpha"`, `"SPEC": "alpha"`, 1),
		"refs null":       strings.Replace(good, `"body": "Fixes alpha."`, `"refs": null, "body": "Fixes alpha."`, 1),
		"breaking null":   replaceLastIn(good, `"breaking": false`, `"breaking": null`, 1),
		"extra field":     strings.Replace(good, `"body": "Fixes alpha."`, `"body": "Fixes alpha.", "extra": 1`, 1),
	}
	for name, raw := range cases {
		if raw == good {
			t.Fatalf("%s: the case does not change the manifest", name)
		}
		if _, issues := ValidateReleaseManifestBytes([]byte(raw), "v1.0.0"); len(issues) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, issues := ValidateReleaseManifestBytes([]byte(good), "v1.0.0"); len(issues) != 0 {
		t.Fatalf("the control is refused: %v", issues)
	}
}

func replaceLastIn(s, old, new string, _ int) string {
	// The first occurrence is in the fragment: the manifest-level flag comes after the fragments.
	i := strings.Index(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

// R2-F02: the archival witness is the spec the blob names, not the one the manifest attributes it to.
func TestR2ArchivalWitnessIsTheSpecOfTheBlob(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	if got := LoadArchivedFragments(r.root); len(got) != 1 || !got[0].Intact {
		t.Fatalf("the control witness is not intact: %+v", got)
	}
	forged := append([]ReleaseFragment(nil), r.manifest.Fragments...)
	forged[0].Spec = "beta"
	r.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", forged, fixtureReleasePolicy(), ChangelogPolicy{}, r.manifest.VersionEvidence)
	r.writeAll(t)
	got := LoadArchivedFragments(r.root)
	if len(got) != 1 || got[0].Spec != "beta" || got[0].Intact {
		t.Fatalf("a fragment of alpha was attested for beta: %+v", got)
	}
}

// R2-F03: the status and the check agree on a working tree that lost the files of the cut.
func TestR2StatusSeesTheFilesOfTheCutThatWereLost(t *testing.T) {
	for name, victim := range map[string]string{"notes": ".pose/changelogs/v1.0.0.md", "fragment": ".pose/changelogs/v1.0.0/alpha.md"} {
		r := tagged(t, newReleaseRepo(t, "v1.0.0"))
		os.Remove(filepath.Join(r.root, filepath.FromSlash(victim)))
		status, err := (Store{Root: r.root}).GetReleaseStatus("v1.0.0")
		if err != nil || len(status.Releases) != 1 || status.Releases[0].Sound() || len(status.Releases[0].Gaps) == 0 {
			t.Errorf("%s: a working tree that lost the file was sound: %+v %v", name, status, err)
		}
	}
}

// R2-F04: the evidence names what the policy of the cut says, and the source says the version.
func TestR2VersionEvidenceFollowsThePolicyOfTheCut(t *testing.T) {
	// A JSON source read with the key of the policy of the tag.
	j := newReleaseRepo(t, "v1.0.0")
	policy := ReleasePolicy{SchemaVersion: 1, VersionSource: &ReleaseVersionSource{Path: "package.json", Kind: ReleaseVersionSourceJSON, Key: "version"}}
	j.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", j.manifest.Fragments, policy, ChangelogPolicy{}, map[string]string{"source": "package.json", "kind": "json", "value": "v1.0.0"})
	j.writeAll(t)
	relWrite(t, j.root, ".pose/policy/release.json", string(CanonicalJSON(policy)))
	relWrite(t, j.root, "package.json", `{"version":"9.9.9"}`)
	if snap := snapshotOf(t, tagged(t, j)); !hasText(snap.Gaps, "version source says") {
		t.Fatalf("a JSON source that says another version was accepted: %+v %v", snap.Gaps, snap.Limits)
	}
	// The same file saying the right version is intact and needs no limit.
	ok := newReleaseRepo(t, "v1.0.0")
	ok.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", ok.manifest.Fragments, policy, ChangelogPolicy{}, map[string]string{"source": "package.json", "kind": "json", "value": "v1.0.0"})
	ok.writeAll(t)
	relWrite(t, ok.root, ".pose/policy/release.json", string(CanonicalJSON(policy)))
	relWrite(t, ok.root, "package.json", `{"version":"v1.0.0"}`)
	if snap := snapshotOf(t, tagged(t, ok)); !snap.Intact() || len(snap.Limits) != 0 {
		t.Fatalf("a JSON source that can be recomputed was not: %+v %v", snap.Gaps, snap.Limits)
	}
	// No version_source in the policy: only the engine's own version file is evidence.
	n := newReleaseRepo(t, "v1.0.0")
	bare := ReleasePolicy{SchemaVersion: 1}
	n.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", n.manifest.Fragments, bare, ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": "", "value": "v1.0.0"})
	n.writeAll(t)
	relWrite(t, n.root, ".pose/policy/release.json", string(CanonicalJSON(bare)))
	if snap := snapshotOf(t, tagged(t, n)); !hasText(snap.Gaps, "engine's own version file") {
		t.Fatalf("an arbitrary file was taken as the engine's version file: %+v", snap)
	}
}

// R2-F06: the policies of the cut are held to their contracts.
func TestR2PoliciesOfTheCutAreValidated(t *testing.T) {
	p := newReleaseRepo(t, "v1.0.0")
	adopted := fixtureReleasePolicy()
	adopted.AdoptedAt = "2026-08-03" // adopted without provider and repository
	p.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", p.manifest.Fragments, adopted, ChangelogPolicy{}, p.manifest.VersionEvidence)
	p.writeAll(t)
	relWrite(t, p.root, ".pose/policy/release.json", string(CanonicalJSON(adopted)))
	if snap := snapshotOf(t, tagged(t, p)); !hasText(snap.Gaps, "does not satisfy its contract") {
		t.Fatalf("a policy that its own loader refuses was accepted: %+v", snap.Gaps)
	}
	c := newReleaseRepo(t, "v1.0.0")
	invented := append([]ReleaseFragment(nil), c.manifest.Fragments...)
	invented[0].Category = "invented"
	relWrite(t, c.root, ".pose/changelogs/v1.0.0/alpha.md", "---\nspec: alpha\ncategory: invented\nbreaking: false\n---\n\nFixes alpha.\n")
	invented[0].Digest = ReleaseDigest("---\nspec: alpha\ncategory: invented\nbreaking: false\n---\n\nFixes alpha.\n")
	c.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", invented, fixtureReleasePolicy(), ChangelogPolicy{}, c.manifest.VersionEvidence)
	relWrite(t, c.root, ".pose/releases/v1.0.0/manifest.json", string(CanonicalJSON(c.manifest)))
	relWrite(t, c.root, ".pose/changelogs/v1.0.0.md", RenderReleaseNotes("v1.0.0", invented, ChangelogPolicy{Categories: append(DefaultChangelogCategories(), "invented")}))
	if snap := snapshotOf(t, tagged(t, c)); !hasText(snap.Gaps, "does not accept") {
		t.Fatalf("a category the changelog policy does not accept was taken: %+v", snap.Gaps)
	}
}

// R2-F07: the notes are the rendering of the frozen fragments.
func TestR2NotesAreTheRenderingOfTheFragments(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	r.notes = "# An unrelated release\n"
	r.manifest.NotesDigest = ReleaseDigest(r.notes)
	r.writeAll(t)
	if snap := snapshotOf(t, tagged(t, r)); !hasText(snap.Gaps, "not the rendering") {
		t.Fatalf("arbitrary notes passed once their digest followed: %+v", snap.Gaps)
	}
	// The right notes under a digest that is not theirs.
	d := newReleaseRepo(t, "v1.0.0")
	d.manifest.NotesDigest = ReleaseDigest("some other notes")
	d.writeAll(t)
	if snap := snapshotOf(t, tagged(t, d)); !hasText(snap.Gaps, "digest differs") {
		t.Fatalf("a notes digest that is not the digest of the notes passed: %+v", snap.Gaps)
	}
}

// R2-F05: a git that cannot answer is not a tag that is not there.
func TestR2GitFailuresAreNotAbsence(t *testing.T) {
	r := tagged(t, newReleaseRepo(t, "v1.0.0"))
	real, err := lookGit()
	if err != nil {
		t.Skip("git is not on the path")
	}
	shim := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n*\"tag --list\"*) echo boom >&2; exit 2;;\nesac\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shim, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if status, err := (Store{Root: r.root}).GetReleaseStatus(""); err == nil {
		t.Fatalf("a failing tag listing read as no releases: %+v", status)
	}
}

func lookGit() (string, error) { return exec.LookPath("git") }

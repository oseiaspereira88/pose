package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the independent review of pose-release-manifest-tag-integrity.

func snapshotOf(t *testing.T, r *releaseRepo) ReleaseSnapshot {
	t.Helper()
	snap, err := ReadReleaseSnapshot(r.root, r.version)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func tagged(t *testing.T, r *releaseRepo) *releaseRepo {
	t.Helper()
	relGit(t, r.root, "add", "-A")
	relGit(t, r.root, "commit", "-q", "--allow-empty", "-m", "cut "+r.version)
	relGit(t, r.root, "tag", r.version)
	return r
}

func hasText(list []string, text string) bool {
	for _, item := range list {
		if strings.Contains(item, text) {
			return true
		}
	}
	return false
}

// F01: one document, exact names, every required field, no repeated key.
func TestF01ManifestIsStrictJSON(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	good := string(CanonicalJSON(r.manifest))
	cases := map[string]string{
		"trailing brace":       good + "}",
		"trailing bracket":     good + "]",
		"repeated key":         strings.Replace(good, `"schema_version": 1,`, `"schema_version": 2, "schema_version": 1,`, 1),
		"upper case name":      strings.Replace(good, `"version": "v1.0.0",`, `"VERSION": "v1.0.0",`, 1),
		"breaking absent":      replaceLast(good, `"breaking": false,`, ``),
		"breaking null":        replaceLast(good, `"breaking": false,`, `"breaking": null,`),
		"fragments null":       strings.Replace(good, `"fragments": [`, `"fragments": null, "x": [`, 1),
		"version evidence nil": strings.Replace(good, `"version_evidence"`, `"version_evidence_x"`, 1),
	}
	for name, raw := range cases {
		if m, issues := ValidateReleaseManifestBytes([]byte(raw), "v1.0.0"); len(issues) == 0 {
			t.Errorf("%s: accepted (%v)", name, m != nil)
		}
	}
	if _, issues := ValidateReleaseManifestBytes([]byte(good), "v1.0.0"); len(issues) != 0 {
		t.Fatalf("the control is refused: %v", issues)
	}
}

// F02: the fields the manifest repeats are recomposed from the fragment blobs.
func TestF02ProjectionsAreRecomposedFromTheFragmentBlobs(t *testing.T) {
	for name, edit := range map[string]func(f *ReleaseFragment){
		"category": func(f *ReleaseFragment) { f.Category = "added" },
		"body":     func(f *ReleaseFragment) { f.Body = "Something else." },
		"breaking": func(f *ReleaseFragment) { f.Breaking = true },
		"refs":     func(f *ReleaseFragment) { f.Refs = "#1" },
	} {
		r := newReleaseRepo(t, "v1.0.0")
		altered := append([]ReleaseFragment(nil), r.manifest.Fragments...)
		edit(&altered[0])
		r.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", altered, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, r.manifest.VersionEvidence)
		r.notes = RenderReleaseNotes("v1.0.0", altered, ChangelogPolicy{})
		r.writeAll(t)
		if snap := snapshotOf(t, tagged(t, r)); snap.Intact() || !hasText(snap.Gaps, name) && !hasText(snap.Gaps, "references") && !hasText(snap.Gaps, "breaking") {
			t.Errorf("%s: a manifest that describes another delivery than its fragment was intact: %+v", name, snap.Gaps)
		}
	}
	// A blob that differs from the frozen digest without changing what the manifest repeats.
	c := newReleaseRepo(t, "v1.0.0")
	relWrite(t, c.root, ".pose/changelogs/v1.0.0/alpha.md", c.body+"\n<!-- an edit that changes no field -->\n")
	if snap := snapshotOf(t, tagged(t, c)); !hasText(snap.Gaps, "digest the manifest froze") {
		t.Fatalf("a fragment edited after the freeze passed: %+v", snap.Gaps)
	}
	d := newReleaseRepo(t, "v1.0.0")
	twice := []ReleaseFragment{d.manifest.Fragments[0], d.manifest.Fragments[0]}
	twice[1].Spec = "beta"
	d.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", twice, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, d.manifest.VersionEvidence)
	d.writeAll(t)
	if snap := snapshotOf(t, tagged(t, d)); !hasText(snap.Gaps, "share the file") {
		t.Fatalf("two specs on one file passed: %+v", snap.Gaps)
	}
}

// F03: the policy digest is tied to the policy of the cut.
func TestF03PolicyDigestFollowsThePolicyOfTheTag(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	r.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", r.manifest.Fragments, ReleasePolicy{SchemaVersion: 1, AllowEmpty: true}, ChangelogPolicy{}, r.manifest.VersionEvidence)
	r.writeAll(t)
	if snap := snapshotOf(t, tagged(t, r)); !hasText(snap.Gaps, "policy digest") {
		t.Fatalf("a policy digest of another policy passed: %+v", snap)
	}
	n := newReleaseRepo(t, "v1.0.0")
	relGit(t, n.root, "rm", "-q", "-f", ".pose/policy/release.json")
	if snap := snapshotOf(t, tagged(t, n)); !hasText(snap.Gaps, "no release policy") {
		t.Fatalf("a tree without a policy passed: %+v", snap)
	}
}

// F04: the version evidence is the version of the release, and its source is in the tree.
func TestF04VersionEvidenceMustBeTheReleaseAndItsSourceMustBeThere(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	for name, evidence := range map[string]map[string]string{
		"another version": {"source": "VERSION", "kind": "text", "value": "v9.9.9"},
		"empty":           {},
		"invented kind":   {"source": "VERSION", "kind": "yaml", "value": "v1.0.0"},
		"no source":       {"kind": "text", "value": "v1.0.0"},
		"extra key":       {"source": "VERSION", "kind": "text", "value": "v1.0.0", "x": "y"},
	} {
		m := NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", r.manifest.Fragments, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, evidence)
		if _, issues := ValidateReleaseManifestBytes(CanonicalJSON(m), "v1.0.0"); len(issues) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
	j := newReleaseRepo(t, "v1.0.0")
	j.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", j.manifest.Fragments, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, map[string]string{"source": "gone.json", "kind": "json", "value": "v1.0.0"})
	j.writeAll(t)
	if snap := snapshotOf(t, tagged(t, j)); !hasText(snap.Gaps, "missing from the tagged tree") {
		t.Fatalf("a version source that is not in the tree passed: %+v", snap)
	}
}

// F05: a status over a snapshot that is not intact says so and does not advance the release.
func TestF05StatusDoesNotTrustASnapshotThatIsNotIntact(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relWrite(t, r.root, ".pose/changelogs/v1.0.0.md", "corrupted notes\n")
	tagged(t, r)
	status, err := (Store{Root: r.root}).GetReleaseStatus("v1.0.0")
	if err != nil || len(status.Releases) != 1 {
		t.Fatalf("%+v %v", status, err)
	}
	release := status.Releases[0]
	if release.State == "tagged" || len(release.Gaps) == 0 || release.Sound() {
		t.Fatalf("a release whose tag holds corrupted notes was trusted: %+v", release)
	}
}

// F06: absence, a read failure and a lost directory are different answers.
func TestF06AbsenceIsNotALoss(t *testing.T) {
	// An archive entry that is a file, not a directory.
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "rm", "-r", "-q", "-f", ".pose/changelogs/v1.0.0")
	relWrite(t, r.root, ".pose/changelogs/v1.0.0", "a file where the archive belongs")
	r.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", []ReleaseFragment{}, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, r.manifest.VersionEvidence)
	r.notes = RenderReleaseNotes("v1.0.0", []ReleaseFragment{}, ChangelogPolicy{})
	relWrite(t, r.root, ".pose/changelogs/v1.0.0.md", r.notes)
	relWrite(t, r.root, ".pose/releases/v1.0.0/manifest.json", string(CanonicalJSON(r.manifest)))
	if snap := snapshotOf(t, tagged(t, r)); !hasText(snap.Gaps, "not a directory") {
		t.Fatalf("an archive that is a file passed as an empty archive: %+v", snap)
	}
	// A legitimately empty release leaves no archive directory: that is absence, not a loss.
	e := newReleaseRepo(t, "v1.0.0")
	relGit(t, e.root, "rm", "-r", "-q", "-f", ".pose/changelogs/v1.0.0")
	e.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", []ReleaseFragment{}, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, e.manifest.VersionEvidence)
	e.notes = RenderReleaseNotes("v1.0.0", []ReleaseFragment{}, ChangelogPolicy{})
	relWrite(t, e.root, ".pose/changelogs/v1.0.0.md", e.notes)
	relWrite(t, e.root, ".pose/releases/v1.0.0/manifest.json", string(CanonicalJSON(e.manifest)))
	if snap := snapshotOf(t, tagged(t, e)); !snap.Intact() {
		t.Fatalf("an empty release with no archive directory was refused: %+v", snap)
	}
	// The release directory cannot be listed: not "no releases".
	f := newReleaseRepo(t, "v1.0.0")
	os.RemoveAll(filepath.Join(f.root, ".pose", "releases"))
	relWrite(t, f.root, ".pose/releases", "a file")
	if _, err := (Store{Root: f.root}).GetReleaseStatus(""); err == nil {
		t.Fatal("a release directory that is a file read as an empty project")
	}
	// The directory is gone from the working tree and the tag still holds the release.
	g := tagged(t, newReleaseRepo(t, "v1.0.0"))
	os.RemoveAll(filepath.Join(g.root, ".pose", "releases"))
	status, err := (Store{Root: g.root}).GetReleaseStatus("")
	if err != nil || len(status.Releases) != 1 || status.Releases[0].Snapshot == nil || status.Releases[0].Sound() {
		t.Fatalf("a release lost from the working tree vanished from the record: %+v %v", status, err)
	}
}

// F10: Semantic Versioning precedence everywhere, and a previous version that is older.
func TestF10VersionPrecedenceIsSemanticAndPreviousIsOlder(t *testing.T) {
	order := []string{"v1.0.0-alpha", "v1.0.0-alpha.1", "v1.0.0-alpha.beta", "v1.0.0-beta", "v1.0.0-beta.2", "v1.0.0-beta.11", "v1.0.0-rc.1", "v1.0.0", "v1.0.1", "v1.1.0", "v2.0.0"}
	for i := 0; i < len(order)-1; i++ {
		if CompareReleasePrecedence(order[i], order[i+1]) >= 0 || CompareReleasePrecedence(order[i+1], order[i]) <= 0 {
			t.Errorf("%s should precede %s", order[i], order[i+1])
		}
	}
	for _, bad := range []string{"v1.0.0-", "v1.0.0-..", "v1.0.0-01", "v1.0.0-a..b", "v1.0.0+build", "1.0.0"} {
		if ValidateReleaseVersion(bad) == nil {
			t.Errorf("%q accepted as a release version", bad)
		}
	}
	r := newReleaseRepo(t, "v1.0.0")
	m := NewReleaseManifest("v1.0.0", "v9.9.9", "2026-10-02T00:00:00Z", r.manifest.Fragments, ReleasePolicy{SchemaVersion: 1}, ChangelogPolicy{}, r.manifest.VersionEvidence)
	if _, issues := ValidateReleaseManifestBytes(CanonicalJSON(m), "v1.0.0"); len(issues) == 0 {
		t.Fatal("a previous version newer than the release was accepted")
	}
}

func replaceLast(s, old, new string) string {
	i := strings.LastIndex(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

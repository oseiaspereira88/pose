package pose

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-release-manifest-tag-integrity.

type releaseRepo struct {
	root     string
	version  string
	body     string
	notes    string
	manifest ReleaseManifest
}

func relGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func relWrite(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newReleaseRepo prepares a release the way `pose release prepare` leaves it: manifest, notes, the
// frozen fragment and the version source, committed.
func newReleaseRepo(t *testing.T, version string) *releaseRepo {
	t.Helper()
	r := &releaseRepo{root: t.TempDir(), version: version}
	r.body = "---\nspec: alpha\ncategory: fixed\nbreaking: false\n---\n\nFixes alpha.\n"
	fragments := []ReleaseFragment{{Spec: "alpha", Category: "fixed", Path: "alpha.md", Body: "Fixes alpha.", Digest: ReleaseDigest(r.body)}}
	r.manifest = NewReleaseManifest(version, "", "2026-10-02T00:00:00Z", fragments, ReleasePolicy{}, ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": "text", "value": version})
	r.notes = RenderReleaseNotes(version, fragments, ChangelogPolicy{})
	relGit(t, r.root, "init", "-q")
	r.writeAll(t)
	relGit(t, r.root, "add", "-A")
	relGit(t, r.root, "commit", "-q", "-m", "prepare "+version)
	return r
}

func (r *releaseRepo) writeAll(t *testing.T) {
	t.Helper()
	relWrite(t, r.root, "VERSION", strings.TrimPrefix(r.version, "v")+"\n")
	relWrite(t, r.root, ".pose/changelogs/"+r.version+"/alpha.md", r.body)
	relWrite(t, r.root, ".pose/changelogs/"+r.version+".md", r.notes)
	relWrite(t, r.root, ".pose/releases/"+r.version+"/manifest.json", string(CanonicalJSON(r.manifest)))
}

func (r *releaseRepo) commit(t *testing.T, msg string) {
	t.Helper()
	relGit(t, r.root, "add", "-A")
	relGit(t, r.root, "commit", "-q", "-m", msg)
}

// V1 (P04): a parseable object is not a manifest; each redundant field must follow from the fragments.
func TestAManifestIsValidOnlyIfItsShapeAndProjectionsAgree(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	good := CanonicalJSON(r.manifest)
	if m, issues := ValidateReleaseManifestBytes(good, "v1.0.0"); m == nil || len(issues) != 0 {
		t.Fatalf("the control manifest has issues: %v", issues)
	}
	mutate := func(edit func(m *ReleaseManifest)) []byte {
		m := r.manifest
		m.Specs = append([]string(nil), m.Specs...)
		m.Categories = map[string]int{}
		for k, v := range r.manifest.Categories {
			m.Categories[k] = v
		}
		m.Fragments = append([]ReleaseFragment(nil), m.Fragments...)
		edit(&m)
		return CanonicalJSON(m)
	}
	cases := map[string]struct {
		raw   []byte
		field string
	}{
		"specs projection":      {mutate(func(m *ReleaseManifest) { m.Specs = []string{"beta"} }), "specs"},
		"categories projection": {mutate(func(m *ReleaseManifest) { m.Categories = map[string]int{"added": 1} }), "categories"},
		"breaking projection":   {mutate(func(m *ReleaseManifest) { m.Breaking = true }), "breaking"},
		"input digest":          {mutate(func(m *ReleaseManifest) { m.ReleaseInputDigest = ReleaseDigest("x") }), "release_input_digest"},
		"fragment edited":       {mutate(func(m *ReleaseManifest) { m.Fragments[0].Body = "tampered" }), "release_input_digest"},
		"schema version":        {mutate(func(m *ReleaseManifest) { m.SchemaVersion = 2 }), "schema_version"},
		"version":               {mutate(func(m *ReleaseManifest) { m.Version = "v9.9.9" }), "version"},
		"prepared_at":           {mutate(func(m *ReleaseManifest) { m.PreparedAt = "yesterday" }), "prepared_at"},
		"notes digest shape":    {mutate(func(m *ReleaseManifest) { m.NotesDigest = "abc" }), "notes_digest"},
		"fragment path":         {mutate(func(m *ReleaseManifest) { m.Fragments[0].Path = "../../etc/passwd" }), "fragments[0].path"},
		"evidence path":         {mutate(func(m *ReleaseManifest) { m.VersionEvidence = map[string]string{"source": "../outside"} }), "version_evidence.source"},
		"duplicate fragment": {mutate(func(m *ReleaseManifest) {
			m.Fragments = append(m.Fragments, m.Fragments[0])
			m.Specs = append(m.Specs, "alpha")
			m.Categories["fixed"] = 2
		}), "fragments[1].spec"},
		"unknown field":   {[]byte(strings.Replace(string(good), `"version": "v1.0.0"`, `"version": "v1.0.0", "invented": true`, 1)), "manifest"},
		"not a manifest":  {[]byte(`{"version":"v1.0.0"}`), "schema_version"},
		"trailing object": {append(append([]byte{}, good...), []byte(`{"x":1}`)...), "manifest"},
	}
	for name, c := range cases {
		_, issues := ValidateReleaseManifestBytes(c.raw, "v1.0.0")
		found := false
		for _, issue := range issues {
			found = found || issue.Field == c.field
		}
		if !found {
			t.Errorf("%s: no issue on %s: %v", name, c.field, issues)
		}
	}
}

// V2 (P06): a tag whose tree holds no manifest is refused whatever the working tree holds.
func TestATagWithoutTheSnapshotInItsTreeIsRefusedWhateverTheWorktreeHolds(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "rm", "-q", "-r", ".pose/releases", ".pose/changelogs")
	relGit(t, r.root, "commit", "-q", "-m", "tag a commit without the snapshot")
	relGit(t, r.root, "tag", "v1.0.0")
	r.writeAll(t) // the working tree holds a perfectly valid snapshot, untracked
	if _, err := ReadReleaseSnapshot(r.root, "v1.0.0"); !errors.Is(err, ErrReleaseSnapshotMissing) {
		t.Fatalf("a tag with no snapshot was read from the working tree: %v", err)
	}
	status, err := (Store{Root: r.root}).GetReleaseStatus("v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Releases) != 1 || status.Releases[0].State == "tagged" || !strings.Contains(strings.Join(status.Releases[0].Gaps, " "), "no release snapshot") {
		t.Fatalf("status = %+v", status.Releases)
	}
	if _, err := ReadReleaseSnapshot(r.root, "v7.7.7"); !errors.Is(err, ErrReleaseTagMissing) {
		t.Fatalf("a name that is not a tag: %v", err)
	}
}

// V3: lightweight and annotated tags resolve to the commit they name.
func TestLightweightAndAnnotatedTagsResolveToTheirCommit(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	head := relGit(t, r.root, "rev-parse", "HEAD")
	relGit(t, r.root, "tag", "v1.0.0")
	commit, annotated, err := ResolveReleaseTag(r.root, "v1.0.0")
	if err != nil || commit != head || annotated {
		t.Fatalf("lightweight: %s %v %v", commit, annotated, err)
	}
	relGit(t, r.root, "tag", "-d", "v1.0.0")
	relGit(t, r.root, "tag", "-a", "v1.0.0", "-m", "release")
	commit, annotated, err = ResolveReleaseTag(r.root, "v1.0.0")
	if err != nil || commit != head || !annotated {
		t.Fatalf("annotated: %s %v %v", commit, annotated, err)
	}
	snap, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if err != nil || !snap.Intact() || !snap.Annotated || snap.Commit != head {
		t.Fatalf("snapshot: %+v %v", snap, err)
	}
}

// V4: the notes, the fragments and the version evidence are read from the tagged tree.
func TestTheSnapshotIsVerifiedAgainstTheTaggedTreeNotTheWorktree(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "tag", "v1.0.0")
	// The working tree is tampered after the tag: the snapshot of the tag is still intact.
	relWrite(t, r.root, ".pose/changelogs/v1.0.0.md", "tampered notes\n")
	relWrite(t, r.root, ".pose/changelogs/v1.0.0/alpha.md", "tampered fragment\n")
	if snap, err := ReadReleaseSnapshot(r.root, "v1.0.0"); err != nil || !snap.Intact() {
		t.Fatalf("a worktree edit after the tag changed the tag's snapshot: %+v %v", snap, err)
	}
	// A tree that was tagged with the notes, a fragment or the version source wrong is not intact.
	tampers := map[string]func(root string){
		"notes":            func(root string) { relWrite(t, root, ".pose/changelogs/v1.0.0.md", "other notes\n") },
		"fragment":         func(root string) { relWrite(t, root, ".pose/changelogs/v1.0.0/alpha.md", "other fragment\n") },
		"missing fragment": func(root string) { _ = os.Remove(filepath.Join(root, ".pose", "changelogs", "v1.0.0", "alpha.md")) },
		"extra fragment":   func(root string) { relWrite(t, root, ".pose/changelogs/v1.0.0/beta.md", "beta\n") },
		"version source":   func(root string) { relWrite(t, root, "VERSION", "9.9.9\n") },
	}
	for name, tamper := range tampers {
		clone := newReleaseRepo(t, "v1.0.0")
		tamper(clone.root)
		clone.commit(t, "tamper "+name)
		relGit(t, clone.root, "tag", "v1.0.0")
		snap, err := ReadReleaseSnapshot(clone.root, "v1.0.0")
		if err != nil || snap.Intact() || len(snap.Gaps) == 0 {
			t.Errorf("%s: a tagged tree that disagrees with its manifest read as intact: %+v %v", name, snap, err)
		}
	}
}

// V5 and V6: provider evidence and the next cycle after the cut do not make a correct tag stale.
func TestACorrectCutStaysIntactAfterEvidenceAndTheNextCycle(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "tag", "-a", "v1.0.0", "-m", "release")
	// Provider evidence is recorded after the tag, and the next cycle opens: the tagged commit
	// is no longer HEAD, and neither commit touches the cut.
	relWrite(t, r.root, ".pose/releases/v1.0.0/events.jsonl", `{"version":"v1.0.0","state":"tagged"}`+"\n")
	r.commit(t, "record evidence")
	relWrite(t, r.root, "VERSION", "1.1.0\n")
	relWrite(t, r.root, ".pose/changelogs/unreleased/beta.md", "---\nspec: beta\ncategory: added\nbreaking: false\n---\n\nBeta.\n")
	r.commit(t, "open the next cycle")
	snap, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if err != nil || !snap.Intact() {
		t.Fatalf("a correct cut went stale after a later commit: %+v %v", snap, err)
	}
	status, err := (Store{Root: r.root}).GetReleaseStatus("v1.0.0")
	if err != nil || len(status.Releases) != 1 || status.Releases[0].Snapshot == nil || !status.Releases[0].Snapshot.Intact() {
		t.Fatalf("status = %+v %v", status, err)
	}
}

// R3: a source that cannot be re-verified is a limit, not a failure and not a pass.
func TestAVersionSourceThatCannotBeRecomputedIsALimit(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	r.manifest = NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", r.manifest.Fragments, ReleasePolicy{}, ChangelogPolicy{}, map[string]string{"source": "package.json", "kind": "json", "value": "v1.0.0"})
	relWrite(t, r.root, ".pose/releases/v1.0.0/manifest.json", string(CanonicalJSON(r.manifest)))
	r.commit(t, "json version source")
	relGit(t, r.root, "tag", "v1.0.0")
	snap, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if err != nil || !snap.Intact() || len(snap.Limits) == 0 {
		t.Fatalf("snapshot = %+v %v", snap, err)
	}
	var raw map[string]any
	if json.Unmarshal(CanonicalJSON(snap), &raw) != nil || raw["limits"] == nil {
		t.Fatal("the limit is not disclosed in the JSON")
	}
}

package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// R6-F01: only the block the registry derives is tolerated; a fragment category that is named
// "compatibility" is the delivery content and is compared.
func TestR6TheRegistryBlockIsToleratedAndAFragmentSectionWithTheSameHeadingIsNot(t *testing.T) {
	policy := ChangelogPolicy{SchemaVersion: 1, Categories: []string{"compatibility", "fixed"}}
	fragments := []ReleaseFragment{{Spec: "alpha", Category: "compatibility", Body: "Required content."}}
	rendered := RenderReleaseNotes("v1.0.0", fragments, policy)
	if !strings.Contains(rendered, "## Compatibility\n\n- Required content.") {
		t.Fatalf("the fixture does not render the fragment under the heading: %q", rendered)
	}
	forged := strings.Replace(rendered, "Required content.", "False content.", 1)
	if ok, _ := NotesAreTheRendering(forged, "v1.0.0", fragments, policy); ok {
		t.Fatal("notes whose compatibility section is a fragment's, with other text, passed as the rendering")
	}
	withRegistry := strings.Replace(rendered, "# POSE v1.0.0\n\n", "# POSE v1.0.0\n\n## Compatibility\n\nThis release introduces a contract this engine no longer lists.\n\n- `x` — y\n\n", 1)
	if ok, limit := NotesAreTheRendering(withRegistry, "v1.0.0", fragments, policy); !ok || limit == "" {
		t.Fatalf("a registry block that only another engine renders was a contradiction: ok=%v limit=%q", ok, limit)
	}
	forgedUnderRegistry := strings.Replace(withRegistry, "Required content.", "False content.", 1)
	if ok, _ := NotesAreTheRendering(forgedUnderRegistry, "v1.0.0", fragments, policy); ok {
		t.Fatal("the registry block hid a change of the fragments' text")
	}
}

// R6-F02: an ancestor that is not a tree hides what is below it; it is not absence.
func TestR6AnAncestorThatIsNotATreeIsNotAbsence(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "rm", "-rq", "--cached", ".pose/policy")
	os.RemoveAll(filepath.Join(r.root, ".pose", "policy"))
	if err := os.Symlink("../elsewhere", filepath.Join(r.root, ".pose", "policy")); err != nil {
		t.Skip("symlinks are not available")
	}
	r.commit(t, "the policy directory becomes a link")
	if _, _, _, found, err := gitTreeEntry(r.root, "HEAD", ".pose/policy/release.json"); err == nil || found {
		t.Fatalf("a link in the place of an ancestor read as absence: found=%v err=%v", found, err)
	}
	if _, err := gitBlobAt(r.root, "HEAD", ".pose/policy/changelog.json"); err == nil || os.IsNotExist(err) {
		t.Fatalf("the changelog policy under a link was taken as not there: %v", err)
	}
	// Absence stays absence: the ancestor is a tree and the entry is not in it, or a level is not there.
	if _, _, _, found, err := gitTreeEntry(r.root, "HEAD", ".pose/changelogs/none/never.md"); err != nil || found {
		t.Fatalf("a path under missing directories is absent, not an error: found=%v err=%v", found, err)
	}
}

// R6-F04: a relative path is looked at from the working directory it was written for.
func TestR6ARelativePathIsSearchedFromTheWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "lost"), filepath.Join(root, ".pose")); err != nil {
		t.Skip("symlinks are not available")
	}
	t.Chdir(root)
	if absent, err := ProvenAbsent(filepath.Join(".pose", "releases")); err == nil || absent {
		t.Fatalf("a lost link above a relative path read as absence: absent=%v err=%v", absent, err)
	}
	if _, err := (Store{Root: "."}).GetReleaseStatus(""); err == nil {
		t.Fatal("a relative root whose .pose is a lost link read as no releases")
	}
}

// R6-F06: a cut the project has passed is said to be superseded; one that can still happen is not.
func TestR6ASupersededCandidate(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	manifest, err := LoadReleaseManifest(r.root, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if ReleaseCandidateSuperseded(r.root, manifest) {
		t.Fatal("a cut whose version is the project's version is superseded")
	}
	relWrite(t, r.root, "VERSION", "2.0.0\n")
	if !ReleaseCandidateSuperseded(r.root, manifest) {
		t.Fatal("a cut older than the project's version is not superseded")
	}
	relWrite(t, r.root, "VERSION", "0.9.0\n")
	if ReleaseCandidateSuperseded(r.root, manifest) {
		t.Fatal("a cut newer than the project's version was called superseded")
	}
	relWrite(t, r.root, "VERSION", "not a version\n")
	if ReleaseCandidateSuperseded(r.root, manifest) {
		t.Fatal("a version file that cannot be read superseded a cut")
	}
}

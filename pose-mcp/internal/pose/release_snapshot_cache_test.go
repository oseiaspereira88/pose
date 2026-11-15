package pose

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Spec pose-federated-proof-reads-each-input-once, R5: the snapshot of a
// tag at a commit is read from git once per process; a second read of that
// commit needs no git.
func TestAReleaseSnapshotIsReadOncePerCommit(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "tag", "v1.0.0")
	commit, annotated, err := ResolveReleaseTag(r.root, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	first, err := readReleaseSnapshotAtCommit(r.root, "v1.0.0", commit, annotated)
	if err != nil || !first.Intact() {
		t.Fatalf("first read: %+v %v", first, err)
	}
	// Without the repository's objects, only the kept snapshot can answer.
	git := filepath.Join(r.root, ".git")
	if err := os.Rename(git, git+".away"); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(git+".away", git)
	second, err := readReleaseSnapshotAtCommit(r.root, "v1.0.0", commit, annotated)
	if err != nil || !second.Intact() || second.ManifestDigest != first.ManifestDigest {
		t.Fatalf("the second read of the same commit went back to git: %+v %v", second, err)
	}
}

// R5: each reader receives its own copy; changing one does not change what
// the next reader receives.
func TestAKeptReleaseSnapshotIsCopiedToEachReader(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "tag", "v1.0.0")
	first, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if err != nil || first.Manifest == nil || len(first.Manifest.Fragments) == 0 {
		t.Fatalf("first read: %+v %v", first, err)
	}
	first.Gaps = append(first.Gaps, "changed by a reader")
	first.Manifest.Fragments[0].Spec = "changed"
	second, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if err != nil || len(second.Gaps) != 0 || second.Manifest.Fragments[0].Spec != "alpha" {
		t.Fatalf("a reader's change reached the next reader: %+v %v", second, err)
	}
}

// R5: a tag moved to another commit is read at that commit.
func TestAMovedReleaseTagIsReadAtItsNewCommit(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "tag", "v1.0.0")
	before, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if err != nil || !before.Intact() {
		t.Fatalf("before: %+v %v", before, err)
	}
	relWrite(t, r.root, ".pose/releases/v1.0.0/manifest.json", "{}\n")
	r.commit(t, "break the manifest")
	relGit(t, r.root, "tag", "-f", "v1.0.0")
	after, err := ReadReleaseSnapshot(r.root, "v1.0.0")
	if after.Commit == before.Commit || after.Intact() {
		t.Fatalf("the moved tag was read from the old commit's snapshot: %+v %v", after, err)
	}
}

// R5: a git read that fails is asked again, not kept as the commit's answer.
func TestAFailedReleaseSnapshotReadIsNotKept(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	relGit(t, r.root, "tag", "v1.0.0")
	commit, annotated, err := ResolveReleaseTag(r.root, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	git := filepath.Join(r.root, ".git")
	if err := os.Rename(git, git+".away"); err != nil {
		t.Fatal(err)
	}
	if _, err := readReleaseSnapshotAtCommit(r.root, "v1.0.0", commit, annotated); err == nil {
		t.Fatal("a read without the repository succeeded")
	}
	if err := os.Rename(git+".away", git); err != nil {
		t.Fatal(err)
	}
	if snap, err := readReleaseSnapshotAtCommit(r.root, "v1.0.0", commit, annotated); err != nil || !snap.Intact() {
		t.Fatalf("the failed read was kept: %+v %v", snap, err)
	}
}

// R6: a tag naming something that is not a commit is refused the same way
// on every read, and a tag recreated as annotated reads as annotated.
func TestAReleaseTagPeelIsKeptPerObject(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	tree := relGit(t, r.root, "rev-parse", "HEAD^{tree}")
	relGit(t, r.root, "tag", "v1.0.0", tree)
	for i := 0; i < 2; i++ {
		if _, _, err := ResolveReleaseTag(r.root, "v1.0.0"); !errors.Is(err, ErrReleaseTagNotCommit) {
			t.Fatalf("read %d: a tag of a tree resolved: %v", i, err)
		}
	}
	relGit(t, r.root, "tag", "-d", "v1.0.0")
	relGit(t, r.root, "tag", "v1.0.0")
	if _, annotated, err := ResolveReleaseTag(r.root, "v1.0.0"); err != nil || annotated {
		t.Fatalf("lightweight: %v %v", annotated, err)
	}
	relGit(t, r.root, "tag", "-d", "v1.0.0")
	relGit(t, r.root, "tag", "-a", "v1.0.0", "-m", "release")
	if _, annotated, err := ResolveReleaseTag(r.root, "v1.0.0"); err != nil || !annotated {
		t.Fatalf("annotated: %v %v", annotated, err)
	}
}

// R7: a tag resolved from the one listing of the tags' refs answers as
// ResolveReleaseTag answers it alone: lightweight, annotated, missing, and
// not a commit.
func TestAListedReleaseTagResolvesAsAloneOne(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	r.addVersion(t, "v1.1.0")
	r.addVersion(t, "v1.2.0")
	relGit(t, r.root, "tag", "v1.0.0")
	relGit(t, r.root, "tag", "-a", "v1.1.0", "-m", "release")
	relGit(t, r.root, "tag", "v1.2.0", relGit(t, r.root, "rev-parse", "HEAD^{tree}"))
	refs, ok := releaseTagRefs(r.root)
	if !ok {
		t.Fatal("the tags could not be listed")
	}
	for _, tag := range []string{"v1.0.0", "v1.1.0", "v1.2.0", "v9.9.9"} {
		aloneCommit, aloneAnnotated, aloneErr := ResolveReleaseTag(r.root, tag)
		commit, annotated, err := resolveListedReleaseTag(r.root, tag, refs)
		if commit != aloneCommit || annotated != aloneAnnotated || errors.Is(err, ErrReleaseTagMissing) != errors.Is(aloneErr, ErrReleaseTagMissing) || errors.Is(err, ErrReleaseTagNotCommit) != errors.Is(aloneErr, ErrReleaseTagNotCommit) || (err == nil) != (aloneErr == nil) {
			t.Fatalf("%s: listed %q %v %v, alone %q %v %v", tag, commit, annotated, err, aloneCommit, aloneAnnotated, aloneErr)
		}
	}
}

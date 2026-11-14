package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"testing"
)

// Spec pose-read-view-streamed-object-digests R3: streaming each commit's
// patch into the digest keeps the digest the retained capture computed, the
// SHA-256 of the concatenated `git show --binary` of the commits in order.
func TestAStreamedChangeSetKeepsItsDiffDigest(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.txt", "one\n")
	first := repo.commit("first\n\nPOSE-Spec: alpha")
	repo.write("a.txt", "two\n")
	repo.write("b.bin", "\x00\x01\x02binary\xff\n")
	second := repo.commit("second\n\nPOSE-Spec: alpha")
	set, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	for _, sha := range []string{first, second} {
		out, err := exec.Command("git", "--no-replace-objects", "-C", repo.root, "show", "--binary", "--no-ext-diff", "--format=", sha, "--").Output()
		if err != nil {
			t.Fatal(err)
		}
		digest.Write(out)
	}
	if want := "sha256:" + hex.EncodeToString(digest.Sum(nil)); set.DiffDigest != want {
		t.Fatalf("streamed diff digest %s, the retained capture's %s", set.DiffDigest, want)
	}
}

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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

// Review round 2 (N1): a failed streamed patch read keeps the typed unknown
// and its forensic cause in the collection gap.
func TestAFailedStreamedPatchReadKeepsItsCause(t *testing.T) {
	root := laterRemovalFixture(t)
	gamma := artifactGit(t, root, "log", "--format=%H", "-n", "1", "--grep", "^POSE-Spec: gamma$")
	failingGit(t, "show", gamma)
	_, _, sets, _, _, collected, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
	if err != nil || !strings.Contains(collected["gamma"], "synthetic-history-failure") {
		t.Fatalf("the failed streamed show lost its cause: %v %v", collected, err)
	}
	for _, set := range sets {
		if set.Spec == "gamma" {
			t.Fatalf("a failed streamed show produced a change set: %+v", set)
		}
	}
	if err := posemodel.StreamGovernanceRawHistoryGitIn(posemodel.NativeFiles(), context.Background(), root, "index.delivery-integrity", 1<<20, io.Discard, "show", "--binary", "--no-ext-diff", "--format=", gamma, "--"); !errors.Is(err, posemodel.ErrGitUnknown) {
		t.Fatalf("a failed streamed read is not typed unknown: %v", err)
	}
}

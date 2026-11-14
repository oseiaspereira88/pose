package cli

import (
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-index-streams-each-commit-once: one pass over the trailer log
// gives every spec the digest its own resolution computes, and reads each
// commit's patch once.
func TestAStreamedSinglePassKeepsEverySpecsDigest(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.txt", "one\n")
	repo.commit("first\n\nPOSE-Spec: alpha")
	repo.write("a.txt", "two\n")
	repo.write("b.bin", "\x00\x01binary\n")
	repo.commit("shared\n\nPOSE-Spec: alpha\nPOSE-Spec: beta")
	repo.write("c.txt", "three\n")
	repo.commit("beta only\n\nPOSE-Spec: beta")
	repo.write("d.txt", "four\n")
	repo.commit("repeated\n\nPOSE-Spec: gamma\nPOSE-Spec: gamma")
	repo.write("a.txt", "five\n")
	repo.commit("alpha again\n\nPOSE-Spec: alpha")

	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shim, count := t.TempDir(), filepath.Join(t.TempDir(), "shows")
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = show ] && printf x >> '" + count + "'; done\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shim, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, order, err := allCommitsWithSpecTrailersOrderedIn(posemodel.NativeFiles(), repo.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	digests, failed := trailerPatchDigests(posemodel.NativeFiles(), repo.root, order, map[string]bool{"alpha": true, "beta": true, "gamma": true})
	shows, _ := os.ReadFile(count)
	if len(failed) != 0 || len(shows) != 4 {
		t.Fatalf("failed=%v; %d patch reads for 4 distinct attributed commits outside the repeated spec", failed, len(shows))
	}
	if _, ok := digests["gamma"]; ok {
		t.Fatal("a spec naming one commit twice was streamed in the single pass")
	}
	for _, spec := range []string{"alpha", "beta"} {
		set, err := resolveGitChangeSet(repo.root, spec, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if got := "sha256:" + hex.EncodeToString(digests[spec]); got != set.DiffDigest {
			t.Fatalf("%s: single pass %s, its own resolution %s", spec, got, set.DiffDigest)
		}
	}
	if !strings.HasPrefix(string(shows), "x") {
		t.Fatal("no patch was read")
	}
}

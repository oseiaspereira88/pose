package pose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunProvenanceExecutableHasIndependentMeasuredBudget(t *testing.T) {
	// Measured installed native CLI; unlike transcript retention, hashing is
	// streamed. Keep a regression for this actual supported binary size.
	if reviewExecutableMaxBytes < 337814032 {
		t.Fatal("the native Claude Code executable cannot fit the executable proof bound")
	}
	if reviewTranscriptMaxBytes != 256*1024*1024 {
		t.Fatal("executable support must not increase transcript retention")
	}
}

func TestRunProvenanceExecutableBudgetBoundaryAndLinks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "agent")
	if err := os.WriteFile(path, []byte("abc"), 0700); err != nil {
		t.Fatal(err)
	}
	const want = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got, err := reviewExecutableDigestWithin(path, 3); err != nil || got != want {
		t.Fatalf("exact bound: digest=%s err=%v", got, err)
	}
	if _, err := reviewExecutableDigestWithin(path, 2); err == nil {
		t.Fatal("oversized executable was accepted")
	}
	link := filepath.Join(root, "selected-agent")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if got, err := reviewExecutableDigestWithin(link, 3); err != nil || got != want {
		t.Fatalf("selected symlink: digest=%s err=%v", got, err)
	}
	if err := os.WriteFile(path, []byte("abd"), 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := reviewExecutableDigestWithin(link, 3); err != nil || got == want {
		t.Fatalf("replacement bytes retained the earlier proof: digest=%s err=%v", got, err)
	}
	for _, bad := range []string{root, filepath.Join(root, "missing")} {
		if _, err := reviewExecutableDigestWithin(bad, 3); err == nil {
			t.Fatalf("invalid executable accepted: %s", bad)
		}
	}
	for _, limit := range []int64{-1, 0, reviewExecutableMaxBytes + 1} {
		if _, err := reviewExecutableDigestWithin(path, limit); err == nil {
			t.Fatalf("invalid bound accepted: %d", limit)
		}
	}
}

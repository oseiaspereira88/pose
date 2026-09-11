package cli

// An update that delivered everything reports that it did (spec
// pose-update-reports-what-it-delivered).
//
// `pose update --force` refreshes the scaffold through `pose install`, which
// indexed the instance before its final gate and returned the moment the index
// failed. A corrupt changelog fragment — instance state, not machinery — made a
// run that had already delivered every tree end in "scaffold refresh failed",
// without saying what had been written or that the fragment predated the run.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateForceWithACorruptFragmentReportsWhatWasDelivered(t *testing.T) {
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	fragments := filepath.Join(repo, ".pose", "changelogs", "unreleased")
	if err := os.MkdirAll(fragments, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fragments, "broken.md"), []byte("not a fragment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A machinery file the refresh must put back, so delivery is observed
	// rather than assumed.
	workflow := filepath.Join(repo, ".pose", "workflows", "feature.md")
	if err := os.Remove(workflow); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errB.Reset()
	code := cmdUpdate(repo, []string{"--force", "--no-self"}, &out, &errB)
	stderr := errB.String()
	if code == 0 {
		t.Fatalf("an instance with a corrupt fragment must still fail its strict gate; stderr=%s", stderr)
	}
	if _, err := os.Stat(workflow); err != nil {
		t.Fatalf("the refresh did not deliver machinery before failing: %v", err)
	}
	for _, want := range []string{"broken.md", "already failed the same gate before this run", "does not roll back"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not say %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "scaffold refresh failed") {
		t.Errorf("the run delivered its files and still reported the refresh as failed:\n%s", stderr)
	}
}

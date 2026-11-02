package pose

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestAReplacedTagTreeIsRefused(t *testing.T) {
	r := newReleaseRepo(t, "v1.0.0")
	good := relGit(t, r.root, "rev-parse", "HEAD")
	relGit(t, r.root, "rm", "-q", "-r", ".pose/releases", ".pose/changelogs")
	r.commit(t, "tag without release snapshot")
	bad := relGit(t, r.root, "rev-parse", "HEAD")
	relGit(t, r.root, "tag", "v1.0.0")
	r.writeAll(t)
	relGit(t, r.root, "replace", bad, good)
	if _, err := ReadReleaseSnapshot(r.root, "v1.0.0"); !errors.Is(err, ErrReleaseSnapshotMissing) {
		t.Fatalf("a replace ref supplied the snapshot of a tag: %v", err)
	}
}

// Named exceptions, each with its reason. Everything else goes through the raw reader.
var directGitExceptions = map[string]string{
	"internal/pose/git_snapshot_integrity.go": "the raw reader itself",
	"internal/pose/review_attempt_ledger.go":  "runs the arguments and environment of gitCommand under its own deadline",
	"internal/usage/usage.go":                 "asks for the common directory, reads no object, and cannot import this package",
}

func TestEngineStartsGitOnlyThroughTheRawReader(t *testing.T) {
	direct := regexp.MustCompile(`exec\.Command(Context)?\([^)]*"git"`)
	root := filepath.Join("..", "..")
	seen := map[string]bool{}
	for _, tree := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if !direct.Match(raw) {
				return nil
			}
			seen[rel] = true
			if directGitExceptions[rel] == "" {
				t.Errorf("%s starts git directly: use RawHistoryGit, or name the exception and its reason here", rel)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for rel := range directGitExceptions {
		if !seen[rel] {
			t.Errorf("the exception for %s no longer applies", rel)
		}
	}
}

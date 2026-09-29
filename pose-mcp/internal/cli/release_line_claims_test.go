package cli

// Release-line claims (spec public-claims-reads-release-lines). Every docs
// page opened with "Applies to: POSE 5.x (current stable)" for a whole major
// after 6.0.0 shipped, and public-claims passed: its prose pattern needed a
// minor digit, so "5.x" was not a claim at all as far as the gate could see.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPublicClaimsReadsAMajorReleaseLine(t *testing.T) {
	root := t.TempDir()
	writeClaimsFixture(t, root, `{"path":"page.md","version_claims":"current-only"},
	 {"path":"home.md","version_claims":"none"}`)
	writeSurface(t, root, "home.md", "POSE is an SDD framework.\n")

	// The released version is 1.7.10: the current major line passes.
	writeSurface(t, root, "page.md", "**Applies to:** POSE 1.x (current stable)\n")
	if code, out := runClaims(t, root, "--strict"); code != 0 {
		t.Fatalf("current major line should pass; exit=%d out=%s", code, out)
	}

	// The previous major is the defect that shipped.
	writeSurface(t, root, "page.md", "**Applies to:** POSE 0.x (current stable)\n")
	code, out := runClaims(t, root, "--strict")
	if code == 0 {
		t.Fatalf("a previous major line must fail; out=%s", out)
	}
	if !strings.Contains(out, "declares version 0 but the released version is 1.7.10") {
		t.Errorf("finding does not name the claimed line and the release: %s", out)
	}

	// An evergreen surface may not name a line either.
	writeSurface(t, root, "page.md", "**Applies to:** POSE 1.x (current stable)\n")
	writeSurface(t, root, "home.md", "POSE 1.x is an SDD framework.\n")
	if code, out := runClaims(t, root, "--strict"); code == 0 {
		t.Fatalf("an evergreen surface naming a release line must fail; out=%s", out)
	}
}

// A page that states which release it applies to is a version claim, so the
// contract has to check it; otherwise the next major repeats the drift.
func TestEveryDocsPageThatNamesAReleaseLineIsADeclaredSurface(t *testing.T) {
	root, err := repoRootForTest()
	if err != nil {
		t.Skip(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "public", "claims.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract publicClaimsContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	declared := map[string]string{}
	for _, s := range contract.Surfaces {
		declared[s.Path] = s.VersionClaims
	}
	line := regexp.MustCompile(`\*\*Applies to:\*\* POSE `)
	pages := 0
	err = filepath.WalkDir(filepath.Join(root, "docs-site", "docs"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !line.Match(body) {
			return nil
		}
		pages++
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if mode, ok := declared[rel]; !ok {
			t.Errorf("%s names the release it applies to but is not a declared public-claims surface", rel)
		} else if mode != versionClaimsCurrent {
			t.Errorf("%s names a release line but is declared %q; only %q allows the current one", rel, mode, versionClaimsCurrent)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if pages == 0 {
		t.Fatal("no docs page carries an Applies-to banner; the extraction stopped matching")
	}
}

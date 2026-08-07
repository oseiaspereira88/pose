package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

func writeReleaseFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func releaseGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Pose Test", "GIT_AUTHOR_EMAIL=pose@example.invalid", "GIT_COMMITTER_NAME=Pose Test", "GIT_COMMITTER_EMAIL=pose@example.invalid")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestReleasePrepareConsumesOnlyPendingSnapshotAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	target := "v" + version.ReleaseBase()
	writeReleaseFixture(t, root, ".pose/release-policy.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo"}`)
	writeReleaseFixture(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\n---\n")
	writeReleaseFixture(t, root, ".pose/changelogs/unreleased/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAdds alpha.\n")
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target}, &out, &errOut); code != 0 {
		t.Fatalf("dry-run=%d %s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/changelogs/unreleased/alpha.md")); err != nil {
		t.Fatal("dry-run mutated pending fragment")
	}
	out.Reset()
	errOut.Reset()
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("apply=%d %s", code, errOut.String())
	}
	notesBefore, err := os.ReadFile(filepath.Join(root, ".pose/changelogs", target+".md"))
	if err != nil {
		t.Fatal(err)
	}
	writeReleaseFixture(t, root, ".pose/specs/beta/spec.md", "---\nslug: beta\nstatus: done\n---\n")
	writeReleaseFixture(t, root, ".pose/changelogs/unreleased/beta.md", "---\nspec: beta\ncategory: fixed\nbreaking: false\n---\n\nFixes beta.\n")
	notesAfter, _ := os.ReadFile(filepath.Join(root, ".pose/changelogs", target+".md"))
	if !bytes.Equal(notesBefore, notesAfter) || !strings.Contains(string(notesBefore), "Adds alpha") || strings.Contains(string(notesBefore), "Fixes beta") {
		t.Fatal("new pending work altered prior notes")
	}
	if gaps, _ := checkRelease(root, target); len(gaps) != 0 {
		t.Fatalf("prepared release gaps=%v", gaps)
	}
}

func TestReleaseBackfillReportsArchiveWithoutFabricatingManifest(t *testing.T) {
	root := t.TempDir()
	releaseGit(t, root, "init", "-q")
	writeReleaseFixture(t, root, ".pose/changelogs/v0.9.0/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAlpha.\n")
	releaseGit(t, root, "add", ".")
	releaseGit(t, root, "commit", "-m", "release history")
	releaseGit(t, root, "tag", "v0.9.0")
	var out, errOut bytes.Buffer
	if code := cmdReleaseBackfill(root, []string{"--from-git"}, &out, &errOut); code != 0 {
		t.Fatalf("backfill=%d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "archive=true") || !strings.Contains(out.String(), "manifest=false") {
		t.Fatalf("unexpected backfill: %s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/releases/v0.9.0/manifest.json")); !os.IsNotExist(err) {
		t.Fatal("backfill fabricated manifest")
	}
}

func TestReleaseEvidenceRejectsCredentialsAndUnsafeAssetNames(t *testing.T) {
	evidence := posemodel.ReleaseEvidence{SchemaVersion: 1, Provider: "github", Repository: "owner/repo", Version: "v1.0.0", Tag: "v1.0.0", Commit: strings.Repeat("a", 40), PublishedAt: "2026-08-03T00:00:00Z", URL: "https://token@example.com/release", Assets: map[string]string{"../pose": "sha256:" + strings.Repeat("b", 64)}}
	if err := validateReleaseEvidence("published", "v1.0.0", evidence); err == nil {
		t.Fatal("unsafe provider evidence accepted")
	}
}

// A spec that declared its changelog fragment as an artifact must keep pointing
// at it after the cut. Before this, every release broke the structural gate for
// the specs it consumed: the declared unreleased path no longer existed
// (spec pose-release-cycle-debt-closure, R2).
func TestReleasePrepareRepointsConsumedSpecArtifactClaims(t *testing.T) {
	root := t.TempDir()
	target := "v" + version.ReleaseBase()
	writeReleaseFixture(t, root, ".pose/release-policy.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo"}`)
	writeReleaseFixture(t, root, ".pose/specs/alpha/spec.md",
		"---\nslug: alpha\nstatus: done\n---\n\n### Artifacts\n"+
			"- created: .pose/changelogs/unreleased/alpha.md\n"+
			"- modified: internal/alpha.go\n")
	writeReleaseFixture(t, root, ".pose/changelogs/unreleased/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAdds alpha.\n")
	// A spec that never mentions the fragment must not be touched at all.
	untouched := "---\nslug: beta\nstatus: done\n---\n\n### Artifacts\n- modified: internal/beta.go\n"
	writeReleaseFixture(t, root, ".pose/specs/beta/spec.md", untouched)

	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("apply=%d %s", code, errOut.String())
	}

	claimed, err := os.ReadFile(filepath.Join(root, ".pose/specs/alpha/spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	archived := ".pose/changelogs/" + target + "/alpha.md"
	if !strings.Contains(string(claimed), "- created: "+archived) {
		t.Errorf("the consumed spec must claim the archived path, got:\n%s", claimed)
	}
	if strings.Contains(string(claimed), ".pose/changelogs/unreleased/alpha.md") {
		t.Error("the stale unreleased claim must be gone")
	}
	if !strings.Contains(string(claimed), "- modified: internal/alpha.go") {
		t.Error("unrelated claims must survive untouched")
	}
	// The claim must resolve on disk, which is the whole point.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(archived))); err != nil {
		t.Errorf("the repointed claim does not resolve: %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(root, ".pose/specs/beta/spec.md")); string(after) != untouched {
		t.Error("a spec that does not claim the fragment must not be rewritten")
	}
}

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

// Prepare never edits a spec (spec pose-release-archival-attested-by-the-ledger).
// It used to rewrite a consumed spec's fragment claim into a rename no commit of
// the spec performed: every released spec then failed artifact-check, and a
// spec closed before the cut had its sealed review subject changed by the cut.
func TestReleasePrepareLeavesEverySpecByteIdentical(t *testing.T) {
	root := t.TempDir()
	target := "v" + version.ReleaseBase()
	writeReleaseFixture(t, root, ".pose/release-policy.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo"}`)
	claimed := "---\nslug: alpha\nstatus: done\n---\n\n### Artifacts\n" +
		"- created: .pose/changelogs/unreleased/alpha.md\n" +
		"- modified: internal/alpha.go\n"
	writeReleaseFixture(t, root, ".pose/specs/alpha/spec.md", claimed)
	writeReleaseFixture(t, root, ".pose/changelogs/unreleased/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAdds alpha.\n")
	untouched := "---\nslug: beta\nstatus: done\n---\n\n### Artifacts\n- modified: internal/beta.go\n"
	writeReleaseFixture(t, root, ".pose/specs/beta/spec.md", untouched)

	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("apply=%d %s", code, errOut.String())
	}
	for path, want := range map[string]string{".pose/specs/alpha/spec.md": claimed, ".pose/specs/beta/spec.md": untouched} {
		if got, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(path))); string(got) != want {
			t.Errorf("prepare edited %s:\n%s", path, got)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/changelogs", target, "alpha.md")); err != nil {
		t.Errorf("the fragment must still be archived: %v", err)
	}
}

// releasedSpecFixture commits a spec that declares its fragment, with the
// spec's trailer, then cuts a release and commits the cut without one — the
// way a release commit is made.
func releasedSpecFixture(t *testing.T) (root, target string) {
	t.Helper()
	root = t.TempDir()
	target = "v" + version.ReleaseBase()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "config", "user.name", "POSE Tests")
	writeArtifactTestFile(t, root, "README.md", "fixture\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "baseline")
	writeArtifactTestFile(t, root, ".pose/release-policy.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo"}`)
	// The fragments are governed, so an archived one left unclaimed would be an
	// orphan; the release notes are the release's, not a spec's, and excluded.
	writeArtifactTestFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["internal",".pose/changelogs"],"exclusions":[".pose/changelogs/`+target+`.md"],"severities":{"existence":"error","action-mismatch":"error","undeclared":"error","orphan":"error"}}`)
	writeArtifactTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-03\n---\n\n# Spec: alpha\n\n## 3. Technical Plan\n\n### Artifacts\n- created: internal/alpha.go\n- created: .pose/changelogs/unreleased/alpha.md\n")
	writeArtifactTestFile(t, root, "internal/alpha.go", "package internal\n")
	writeArtifactTestFile(t, root, ".pose/changelogs/unreleased/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAdds <alpha> & more.\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "implement alpha", "-m", "POSE-Spec: alpha")
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("prepare=%d %s", code, errOut.String())
	}
	artifactGit(t, root, "add", "-A", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "chore(release): prepare "+target)
	return root, target
}

// The released spec passes artifact-check after the cut, and says where its
// fragment went. Against the rewriting prepare this failed with action-mismatch.
func TestAReleasedSpecStillPassesArtifactCheckAfterTheCut(t *testing.T) {
	root, target := releasedSpecFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--strict"}, &out, &errOut); code != 0 {
		t.Fatalf("artifact-check=%d err=%s out=%s", code, errOut.String(), out.String())
	}
	want := "artifact.archived=.pose/changelogs/unreleased/alpha.md -> .pose/changelogs/" + target + "/alpha.md (release " + target + ")"
	if !strings.Contains(out.String(), want) {
		t.Errorf("artifact-check must say where the fragment went; want %q in:\n%s", want, out.String())
	}
	// The archived fragment is governed here, and claimed through the release,
	// so the whole-repository graph must not call it an orphan either.
	graph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range graph.Findings {
		if f.Spec == "alpha" || strings.Contains(f.Path, "alpha.md") {
			t.Errorf("unexpected finding: %+v", f)
		}
	}
}

// A spec an earlier release rewrote keeps passing, through the manifest of the
// version its rename names, with nothing migrated.
func TestASpecAnEarlierReleaseRewroteStillPasses(t *testing.T) {
	root, target := releasedSpecFixture(t)
	writeArtifactTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-08-03\n---\n\n# Spec: alpha\n\n## 3. Technical Plan\n\n### Artifacts\n- created: internal/alpha.go\n- renamed: .pose/changelogs/unreleased/alpha.md -> .pose/changelogs/"+target+"/alpha.md\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "an older engine rewrote the claim")
	var out, errOut bytes.Buffer
	if code := cmdArtifactCheck(root, []string{"--spec", "alpha", "--strict"}, &out, &errOut); code != 0 {
		t.Fatalf("artifact-check=%d err=%s out=%s", code, errOut.String(), out.String())
	}
}

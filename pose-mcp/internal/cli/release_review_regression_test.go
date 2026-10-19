package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// Regression tests for the independent review of pose-release-manifest-tag-integrity.

func relGitCLI(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func checkDoc(t *testing.T, root string) map[string]any {
	t.Helper()
	_, out := runPose(t, root, "release", "check", "--version", "v1.0.0", "--strict", "--json")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return doc
}

// F07: a branch that carries the name of a release is not its tag.
func TestF07ABranchWithTheNameOfATagIsNotATag(t *testing.T) {
	root := cliReleaseRepo(t)
	relGitCLI(t, root, "branch", "v1.0.0")
	doc := checkDoc(t, root)
	if doc["tagged"] == true {
		t.Fatalf("a branch was accepted as the release tag: %v", doc)
	}
}

// F08: a policy that moved after the tag does not invalidate the cut the tag holds.
func TestF08ALaterPolicyDoesNotInvalidateATaggedCut(t *testing.T) {
	root := cliReleaseRepo(t)
	relGitCLI(t, root, "tag", "v1.0.0")
	writeCloseoutCLIFile(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"","tag_pattern":"","allow_empty":true,"legacy_cutoff":"","provider":"","repository":"","require_verification":false,"version_source":{"path":"VERSION","kind":"text"}}`)
	relGitCLI(t, root, "add", "-A")
	relGitCLI(t, root, "commit", "-q", "-m", "later policy")
	doc := checkDoc(t, root)
	if doc["valid"] != true || doc["tagged"] != true {
		t.Fatalf("a correct tagged cut was invalidated by a later policy: %v", doc)
	}
}

// F09: the cut of the working tree is the cut of the tag.
func TestF09TheWorkingTreeMustBeTheCutOfTheTag(t *testing.T) {
	root := cliReleaseRepo(t)
	relGitCLI(t, root, "tag", "v1.0.0")
	body := "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nSomething else.\n"
	fragments := []posemodel.ReleaseFragment{{Spec: "alpha", Category: "added", Path: "alpha.md", Body: "Something else.", Digest: posemodel.ReleaseDigest(body)}}
	other := posemodel.NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", fragments, posemodel.ReleasePolicy{SchemaVersion: 1, VersionSource: &posemodel.ReleaseVersionSource{Path: "VERSION", Kind: "text"}}, posemodel.ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": "text", "value": "v1.0.0"})
	writeCloseoutCLIFile(t, root, ".pose/changelogs/v1.0.0/alpha.md", body)
	writeCloseoutCLIFile(t, root, ".pose/changelogs/v1.0.0.md", posemodel.RenderReleaseNotes("v1.0.0", fragments, posemodel.ChangelogPolicy{}))
	writeCloseoutCLIFile(t, root, ".pose/releases/v1.0.0/manifest.json", string(posemodel.CanonicalJSON(other)))
	doc := checkDoc(t, root)
	if doc["valid"] == true {
		t.Fatalf("two different cuts under one tag passed: %v", doc)
	}
}

// F05: an event cannot state that a tag is the release when the tag's tree is not an intact one.
func TestF05RecordRefusesATagWhoseTreeIsNotIntact(t *testing.T) {
	// Control: the same record about an intact tag is accepted.
	good := cliReleaseRepo(t)
	relGitCLI(t, good, "tag", "v1.0.0")
	goodHead := strings.TrimSpace(gitOut(t, good, "rev-parse", "HEAD"))
	writeCloseoutCLIFile(t, good, "evidence.json", `{"schema_version":1,"provider":"github","repository":"o/r","version":"v1.0.0","tag":"v1.0.0","commit":"`+goodHead+`"}`)
	if code, out := runPose(t, good, "release", "record", "--version", "v1.0.0", "--event", "tagged", "--evidence", "evidence.json"); code != 0 {
		t.Fatalf("the control record was refused: %s", out)
	}
	root := cliReleaseRepo(t)
	writeCloseoutCLIFile(t, root, ".pose/changelogs/v1.0.0.md", "corrupted\n")
	relGitCLI(t, root, "add", "-A")
	relGitCLI(t, root, "commit", "-q", "-m", "corrupt the notes")
	relGitCLI(t, root, "tag", "v1.0.0")
	// The working tree is the correct cut again: only the tag's own tree is broken.
	relGitCLI(t, root, "checkout", "HEAD~1", "--", ".pose/changelogs/v1.0.0.md")
	head := strings.TrimSpace(gitOut(t, root, "rev-parse", "HEAD"))
	evidence := `{"schema_version":1,"provider":"github","repository":"o/r","version":"v1.0.0","tag":"v1.0.0","commit":"` + head + `"}`
	writeCloseoutCLIFile(t, root, "evidence.json", evidence)
	if code, out := runPose(t, root, "release", "record", "--version", "v1.0.0", "--event", "tagged", "--evidence", "evidence.json"); code == 0 {
		t.Fatalf("a tag with corrupted notes was recorded as the release: %s", out)
	}
	if _, err := os.Stat(root + "/.pose/releases/v1.0.0/events.jsonl"); err == nil {
		t.Fatal("an event was written")
	}
}

func gitOut(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func preparedEngineRelease(t *testing.T) (root, target string) {
	return preparedEngineReleaseWith(t, false)
}

// preparedEngineReleaseWith prepares a release of the engine repository; empty prepares one with no
// fragment, which the policy then allows.
func preparedEngineReleaseWith(t *testing.T, empty bool) (root, target string) {
	t.Helper()
	root = t.TempDir()
	target = "v" + version.ReleaseBase()
	allow := "false"
	if empty {
		allow = "true"
	}
	writeReleaseFixture(t, root, ".pose/release-policy.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo","allow_empty":`+allow+`}`)
	markAsEngineRepository(t, root)
	if !empty {
		writeReleaseFixture(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\n---\n")
		writeReleaseFixture(t, root, ".pose/changelogs/unreleased/alpha.md", "---\nspec: alpha\ncategory: added\nbreaking: false\n---\n\nAdds alpha.\n")
	}
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("prepare=%d %s", code, errOut.String())
	}
	return root, target
}

// R2-F08: a manifest that is there and is not valid is not written over.
func TestR2PrepareDoesNotOverwriteAManifestItCannotRead(t *testing.T) {
	root, target := preparedEngineReleaseWith(t, true)
	path := filepath.Join(root, ".pose", "releases", target, "manifest.json")
	raw, _ := os.ReadFile(path)
	broken := strings.Replace(string(raw), `"breaking": false,
  "notes_digest"`, `"notes_digest"`, 1)
	if broken == string(raw) {
		t.Fatal("the case does not change the manifest")
	}
	os.WriteFile(path, []byte(broken), 0o644)
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "is not overwritten") {
		t.Fatalf("prepare did not refuse an invalid manifest by name (%d): out=%s err=%s", code, out.String(), errOut.String())
	}
	if after, _ := os.ReadFile(path); string(after) != broken {
		t.Fatal("the invalid manifest was overwritten")
	}
}

// R2-F04: the evidence of a candidate names the source and the kind of the policy of today.
func TestR2PreparedEvidenceNamesTheSourceOfThePolicy(t *testing.T) {
	root, target := preparedEngineRelease(t)
	path := filepath.Join(root, ".pose", "releases", target, "manifest.json")
	raw, _ := os.ReadFile(path)
	var manifest posemodel.ReleaseManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	evidence := map[string]string{"source": "missing.txt", "value": manifest.VersionEvidence["value"]}
	policy, _ := posemodel.LoadReleasePolicy(root)
	forged := posemodel.NewReleaseManifest(target, manifest.PreviousVersion, manifest.PreparedAt, manifest.Fragments, policy, posemodel.LoadChangelogPolicy(root), evidence)
	os.WriteFile(path, posemodel.CanonicalJSON(forged), 0o644)
	gaps, _ := checkRelease(root, target)
	if !strings.Contains(strings.Join(gaps, ";"), "version evidence of the manifest names") {
		t.Fatalf("a source the policy does not name passed: %v", gaps)
	}
}

// R4-F03: prepared notes are the rendering of the frozen fragments, and `release notes` respects it.
func TestR4PreparedNotesAreTheRenderingAndTheReaderRefusesOtherwise(t *testing.T) {
	root, target := preparedEngineRelease(t)
	manifestPath := filepath.Join(root, ".pose", "releases", target, "manifest.json")
	raw, _ := os.ReadFile(manifestPath)
	var manifest posemodel.ReleaseManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	other := "# Unrelated notes\n"
	manifest.NotesDigest = posemodel.ReleaseDigest(other)
	os.WriteFile(manifestPath, posemodel.CanonicalJSON(manifest), 0o644)
	os.WriteFile(filepath.Join(root, ".pose", "changelogs", target+".md"), []byte(other), 0o644)
	if gaps, _ := checkRelease(root, target); !strings.Contains(strings.Join(gaps, ";"), "not the rendering") {
		t.Fatalf("notes that are not the rendering passed once their digest followed: %v", gaps)
	}
	var out, errOut bytes.Buffer
	if code := cmdReleaseNotesSnapshot(root, []string{"--version", target}, &out, &errOut); code == 0 || out.Len() != 0 {
		t.Fatalf("the reader printed notes the check refuses (%d): %s", code, out.String())
	}
}

// R4-F05: events that say the release was tagged, in a repository without the tag.
func TestR4EventsThatNameATagThatIsNotThereAreAGap(t *testing.T) {
	root := cliReleaseRepo(t)
	writeCloseoutCLIFile(t, root, ".pose/releases/v1.0.0/events.jsonl", `{"schema_version":1,"version":"v1.0.0","state":"tagged","recorded_at":"2026-10-02T00:00:00Z","evidence":{"schema_version":1,"provider":"github","repository":"o/r","version":"v1.0.0","tag":"v1.0.0","commit":"c"},"evidence_digest":"sha256:x"}`+"\n")
	doc := checkDoc(t, root)
	if doc["valid"] == true {
		t.Fatalf("events naming a tag that does not exist were valid: %v", doc)
	}
}

// R5-F04: a manifest that is a link to nowhere is not a place to write through.
func TestR5PrepareDoesNotWriteThroughALostManifestLink(t *testing.T) {
	root, target := preparedEngineReleaseWith(t, true)
	path := filepath.Join(root, ".pose", "releases", target, "manifest.json")
	os.Remove(path)
	lostTarget := filepath.Join(root, "lost.json")
	if err := os.Symlink(lostTarget, path); err != nil {
		t.Skip("symlinks are not available")
	}
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", target, "--apply"}, &out, &errOut); code == 0 || !strings.Contains(errOut.String(), "is not overwritten") {
		t.Fatalf("prepare wrote through a lost link (%d): out=%s err=%s", code, out.String(), errOut.String())
	}
	if _, err := os.Stat(lostTarget); err == nil {
		t.Fatal("the target of the link was created")
	}
}

func releaseCheckerFor(root string) (*nativeChecker, *bytes.Buffer) {
	out := &bytes.Buffer{}
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: out}
	appendReleasePolicyChecks(checker)
	return checker, out
}

// R6-F03: a release inventory that is lost is not "no releases", in the checker and in the plan.
func TestR6ALostReleaseInventoryIsNotNoReleases(t *testing.T) {
	root, target := preparedEngineReleaseWith(t, true)
	if err := os.RemoveAll(filepath.Join(root, ".pose", "releases")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "lost"), filepath.Join(root, ".pose", "releases")); err != nil {
		t.Skip("symlinks are not available")
	}
	if checker, out := releaseCheckerFor(root); checker.errors == 0 || !strings.Contains(out.String(), "release inventory") {
		t.Fatalf("the checker called a lost inventory sound: errors=%d %q", checker.errors, out.String())
	}
	if _, _, previous, _, err := releaseInputs(root, target); err == nil {
		t.Fatalf("the plan took a lost inventory for no earlier release (previous=%q)", previous)
	}
}

// R6-F05: the categories of a cut are the ones its policy held, not the ones the project allows today.
func TestR6ACategoryAllowedWhenTheCutWasMadeIsNotRevalidated(t *testing.T) {
	root, target := preparedEngineReleaseWith(t, false)
	releaseGit(t, root, "init", "-q")
	releaseGit(t, root, "add", ".")
	releaseGit(t, root, "commit", "-q", "-m", "release "+target)
	releaseGit(t, root, "tag", target)
	writeReleaseFixture(t, root, ".pose/policy/changelog.json", `{"schema_version":1,"categories":["fixed"]}`)
	if _, out := releaseCheckerFor(root); strings.Contains(out.String(), "invalid `category") || strings.Contains(out.String(), "invalid category") {
		t.Fatalf("the checker revalidated the categories of a cut under today's policy: %q", out.String())
	}
}

// R6-F06: a prepared cut the project's version has passed is said, not failed; one that can still
// happen is judged.
func TestR6ASupersededPreparedCutIsAWarningAndALiveOneIsJudged(t *testing.T) {
	root := t.TempDir()
	writeReleaseFixture(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo","allow_empty":true,"version_source":{"path":"VERSION","kind":"text"}}`)
	writeReleaseFixture(t, root, "VERSION", "1.0.0\n")
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", "v1.0.0", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("prepare=%d %s", code, errOut.String())
	}
	releaseGit(t, root, "init", "-q")
	if checker, text := releaseCheckerFor(root); checker.errors != 0 {
		t.Fatalf("a sound prepared cut failed: %q", text.String())
	}
	if err := os.Remove(filepath.Join(root, ".pose", "changelogs", "v1.0.0.md")); err != nil {
		t.Fatal(err)
	}
	if checker, _ := releaseCheckerFor(root); checker.errors == 0 {
		t.Fatal("a live prepared cut that lost its notes was not judged")
	}
	writeReleaseFixture(t, root, "VERSION", "2.0.0\n")
	checker, text := releaseCheckerFor(root)
	if checker.errors != 0 || !strings.Contains(text.String(), "has passed") {
		t.Fatalf("a superseded cut was not said as a warning: errors=%d %q", checker.errors, text.String())
	}
}

// adoptedTaggedRelease prepares v1.0.0 under an adopted policy and tags a commit holding the cut.
func adoptedTaggedRelease(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeReleaseFixture(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"2026-08-03","provider":"github","repository":"owner/repo","allow_empty":true,"version_source":{"path":"VERSION","kind":"text"}}`)
	writeReleaseFixture(t, root, "VERSION", "1.0.0\n")
	var out, errOut bytes.Buffer
	if code := cmdReleasePrepare(root, []string{"--version", "v1.0.0", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("prepare=%d %s", code, errOut.String())
	}
	releaseGit(t, root, "init", "-q")
	releaseGit(t, root, "add", ".")
	releaseGit(t, root, "commit", "-q", "-m", "release v1.0.0")
	releaseGit(t, root, "tag", "v1.0.0")
	return root
}

// R7-F01: the tag witnesses a release whose directory the working tree lost.
func TestR7ATaggedReleaseWhoseDirectoryWasRemovedIsJudged(t *testing.T) {
	for name, remove := range map[string]string{"one release": "v1.0.0", "whole inventory": ""} {
		t.Run(name, func(t *testing.T) {
			root := adoptedTaggedRelease(t)
			if checker, out := releaseCheckerFor(root); checker.errors != 0 {
				t.Fatalf("control: an intact tagged release failed: %q", out.String())
			}
			if err := os.RemoveAll(filepath.Join(root, ".pose", "releases", remove)); err != nil {
				t.Fatal(err)
			}
			if checker, out := releaseCheckerFor(root); checker.errors == 0 || !strings.Contains(out.String(), "not a sound cut in the working tree") {
				t.Fatalf("a tagged release that lost its directory passed: errors=%d %q", checker.errors, out.String())
			}
		})
	}
}

// R7-F02: a superseded cut is a warning only when it was never tagged and its journal is readable.
func TestR7SupersededDoesNotMaskALostTagOrAnUnreadableJournal(t *testing.T) {
	recorded := func(t *testing.T) string {
		root := adoptedTaggedRelease(t)
		head := strings.TrimSpace(gitOut(t, root, "rev-parse", "HEAD"))
		writeReleaseFixture(t, root, "evidence.json", `{"schema_version":1,"provider":"github","repository":"owner/repo","version":"v1.0.0","tag":"v1.0.0","commit":"`+head+`"}`)
		var out, errOut bytes.Buffer
		if code := cmdReleaseRecord(root, []string{"--version", "v1.0.0", "--event", "tagged", "--evidence", "evidence.json"}, &out, &errOut); code != 0 {
			t.Fatalf("record=%d %s", code, errOut.String())
		}
		releaseGit(t, root, "tag", "-d", "v1.0.0")
		writeReleaseFixture(t, root, "VERSION", "2.0.0\n")
		return root
	}
	t.Run("lost tag already recorded", func(t *testing.T) {
		root := recorded(t)
		if checker, out := releaseCheckerFor(root); checker.errors == 0 {
			t.Fatalf("a recorded tag that vanished was called superseded: %q", out.String())
		}
	})
	t.Run("unreadable journal", func(t *testing.T) {
		root := recorded(t)
		writeReleaseFixture(t, root, ".pose/releases/v1.0.0/events.jsonl", "{broken\n")
		if checker, out := releaseCheckerFor(root); checker.errors == 0 {
			t.Fatalf("an unreadable journal was called superseded: %q", out.String())
		}
	})
}

// R8-F01: a release policy that disappeared does not switch the checker of governed releases off.
func TestR8MissingAdoptedPolicyDoesNotDisableTheReleaseChecker(t *testing.T) {
	dropPolicy := func(t *testing.T, root string) {
		t.Helper()
		for _, rel := range []string{".pose/policy/release.json", ".pose/release-policy.json"} {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
	}
	t.Run("cut in the working tree", func(t *testing.T) {
		root := adoptedTaggedRelease(t)
		releaseGit(t, root, "tag", "-d", "v1.0.0")
		dropPolicy(t, root)
		if checker, out := releaseCheckerFor(root); checker.errors == 0 || !strings.Contains(out.String(), "policy is missing") {
			t.Fatalf("a lost policy switched the checker off: errors=%d %q", checker.errors, out.String())
		}
	})
	t.Run("tag witness only", func(t *testing.T) {
		root := adoptedTaggedRelease(t)
		dropPolicy(t, root)
		if err := os.RemoveAll(filepath.Join(root, ".pose", "releases")); err != nil {
			t.Fatal(err)
		}
		if checker, out := releaseCheckerFor(root); checker.errors == 0 || !strings.Contains(out.String(), "policy is missing") {
			t.Fatalf("a lost policy and inventory switched the checker off: errors=%d %q", checker.errors, out.String())
		}
	})
	t.Run("a project that never adopted stays quiet", func(t *testing.T) {
		root := t.TempDir()
		releaseGit(t, root, "init", "-q")
		writeReleaseFixture(t, root, "VERSION", "1.0.0\n")
		releaseGit(t, root, "add", ".")
		releaseGit(t, root, "commit", "-q", "-m", "legacy")
		releaseGit(t, root, "tag", "v1.0.0")
		if checker, out := releaseCheckerFor(root); checker.errors != 0 {
			t.Fatalf("a project without governed releases was judged: %q", out.String())
		}
	})
}

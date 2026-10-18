package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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
	writeCloseoutCLIFile(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"","tag_pattern":"","allow_empty":true,"legacy_cutoff":"","provider":"","repository":"","require_verification":false}`)
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
	other := posemodel.NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", fragments, posemodel.ReleasePolicy{SchemaVersion: 1}, posemodel.ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": "text", "value": "v1.0.0"})
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

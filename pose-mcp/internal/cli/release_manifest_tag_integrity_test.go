package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-release-manifest-tag-integrity: `release check` and `release status` read the tag's own tree.

func cliReleaseRepo(t *testing.T) string { return cliReleaseRepoKind(t, "text") }

func cliReleaseRepoKind(t *testing.T, kind string) string {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	body := "---\nspec: alpha\ncategory: fixed\nbreaking: false\n---\n\nFixes alpha.\n"
	fragments := []posemodel.ReleaseFragment{{Spec: "alpha", Category: "fixed", Path: "alpha.md", Body: "Fixes alpha.", Digest: posemodel.ReleaseDigest(body)}}
	manifest := posemodel.NewReleaseManifest("v1.0.0", "", "2026-10-02T00:00:00Z", fragments, posemodel.ReleasePolicy{SchemaVersion: 1}, posemodel.ChangelogPolicy{}, map[string]string{"source": "VERSION", "kind": kind, "value": "v1.0.0"})
	git("init", "-q")
	writeCloseoutCLIFile(t, root, ".pose/policy/release.json", `{"schema_version":1,"adopted_at":"","tag_pattern":"","allow_empty":false,"legacy_cutoff":"","provider":"","repository":"","require_verification":false}`)
	writeCloseoutCLIFile(t, root, "VERSION", "1.0.0\n")
	writeCloseoutCLIFile(t, root, ".pose/changelogs/v1.0.0/alpha.md", body)
	writeCloseoutCLIFile(t, root, ".pose/changelogs/v1.0.0.md", posemodel.RenderReleaseNotes("v1.0.0", fragments, posemodel.ChangelogPolicy{}))
	writeCloseoutCLIFile(t, root, ".pose/releases/v1.0.0/manifest.json", string(posemodel.CanonicalJSON(manifest)))
	git("add", "-A")
	git("commit", "-q", "-m", "prepare v1.0.0")
	return root
}

func TestReleaseCheckReadsTheTagsOwnTree(t *testing.T) {
	root := cliReleaseRepo(t)
	run := func(root string) map[string]any {
		t.Helper()
		_, out := runPose(t, root, "release", "check", "--version", "v1.0.0", "--json")
		var doc map[string]any
		if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return doc
	}
	if out, err := exec.Command("git", "-C", root, "tag", "v1.0.0").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	// The fixture has no release policy, so the working-tree checks have their own gaps; the
	// tag's snapshot, read from its own tree, has none.
	doc := run(root)
	gapsOf := func(doc map[string]any) string {
		gaps, _ := doc["gaps"].([]any)
		parts := []string{}
		for _, g := range gaps {
			parts = append(parts, g.(string))
		}
		return strings.Join(parts, "; ")
	}
	if doc["tagged"] != true || strings.Contains(gapsOf(doc), "snapshot") {
		t.Fatalf("an intact tagged release: %v", doc)
	}
	// A tag on a commit that holds no snapshot, while the working tree is a valid prepared release.
	bare := cliReleaseRepo(t)
	for _, args := range [][]string{{"rm", "-q", "-r", ".pose"}, {"commit", "-q", "-m", "no snapshot"}, {"tag", "v1.0.0"}} {
		cmd := exec.Command("git", append([]string{"-C", bare, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	for _, f := range []string{".pose/changelogs/v1.0.0/alpha.md", ".pose/changelogs/v1.0.0.md", ".pose/releases/v1.0.0/manifest.json"} {
		src, _ := os.ReadFile(filepath.Join(root, f))
		writeCloseoutCLIFile(t, bare, f, string(src))
	}
	doc = run(bare)
	found := strings.Contains(gapsOf(doc), "no release snapshot in its tree")
	if doc["valid"] == true || !found {
		t.Fatalf("a tag with no snapshot in its tree passed on the strength of the working tree: %v", doc)
	}
	code, text := runPose(t, bare, "release", "check", "--version", "v1.0.0", "--strict")
	if code == 0 || !strings.Contains(text, "no release snapshot in its tree") {
		t.Fatalf("strict check: %d %s", code, text)
	}
}

func TestReleaseStatusReportsTheSnapshotOfTheTag(t *testing.T) {
	root := cliReleaseRepo(t)
	if out, err := exec.Command("git", "-C", root, "tag", "-a", "v1.0.0", "-m", "release", "-c", "tag.gpgsign=false").CombinedOutput(); err != nil {
		cmd := exec.Command("git", "-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "tag", "-a", "v1.0.0", "-m", "release")
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			t.Fatalf("%v %s %s", err2, out, out2)
		}
	}
	code, out := runPose(t, root, "release", "status", "--json")
	if code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	var doc struct {
		Releases []struct {
			Version  string `json:"version"`
			State    string `json:"state"`
			Snapshot *struct {
				Annotated bool `json:"annotated"`
			} `json:"snapshot"`
		} `json:"releases"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil || len(doc.Releases) != 1 {
		t.Fatalf("%v\n%s", err, out)
	}
	if r := doc.Releases[0]; r.State != "tagged" || r.Snapshot == nil || !r.Snapshot.Annotated {
		t.Fatalf("release = %+v", r)
	}
}

// A version source the manifest cannot recompute is disclosed as a limit in the JSON, not as a failure.
func TestReleaseCheckDisclosesTheVersionSourcesItCouldNotRecompute(t *testing.T) {
	root := cliReleaseRepoKind(t, "json")
	if out, err := exec.Command("git", "-C", root, "tag", "v1.0.0").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	_, out := runPose(t, root, "release", "check", "--version", "v1.0.0", "--json")
	var doc struct {
		Limits []string `json:"snapshot_limits"`
		Gaps   []string `json:"gaps"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(doc.Limits) == 0 || strings.Contains(strings.Join(doc.Gaps, ";"), "snapshot") {
		t.Fatalf("limits=%v gaps=%v", doc.Limits, doc.Gaps)
	}
}

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Spec pose-update-dry-run-reports-the-whole-update: on an instance last
// updated by an older engine, the dry-run lists exactly what the real update
// then changes, writes nothing, and doctor and check name the stale stamp
// until the update runs.
func TestUpdateDryRunListsWhatTheUpdateChanges(t *testing.T) {
	repo := olderInstance(t)
	manifest := filepath.Join(repo, ".pose", "state", "machinery-manifest.json")
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["engine_version"] = "6.1.0"
	stamped, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(manifest, append(stamped, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	// A drifted managed file the update merges back.
	template := filepath.Join(repo, ".pose", "templates", "spec.md")
	original, err := os.ReadFile(template)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(template, []byte(strings.Replace(string(original), "## 1.", "## 1 (drifted).", 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	if code, out := runPose(t, repo, "doctor"); code != 0 || !strings.Contains(out, "last updated by POSE 6.1.0") {
		t.Fatalf("doctor does not name the stale stamp: %d %s", code, out)
	}
	if _, out := runPose(t, repo, "check"); !strings.Contains(out, "last updated by POSE 6.1.0") {
		t.Fatalf("check does not name the stale stamp: %s", out)
	}

	before, err := hashTree(repo, true)
	if err != nil {
		t.Fatal(err)
	}
	code, out := runPose(t, repo, "update", "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run: %s", out)
	}
	if !strings.Contains(out, "would stamp engine_version: 6.1.0 ->") || !strings.Contains(out, "nothing was applied") {
		t.Fatalf("dry-run output: %s", out)
	}
	afterDry, _ := hashTree(repo, true)
	if changes := diffTrees(before, afterDry); len(changes) != 0 {
		t.Fatalf("the dry-run wrote: %+v", changes)
	}
	predicted := []string{}
	for _, line := range strings.Split(out, "\n") {
		for _, verb := range []string{"create", "modify", "remove"} {
			if rest, ok := strings.CutPrefix(line, "[DRY-RUN] would "+verb+": "); ok {
				predicted = append(predicted, verb+" "+rest)
			}
		}
		var opened int
		if _, err := fmt.Sscanf(line, "[DRY-RUN] would open %d action request(s)", &opened); err == nil {
			for i := 0; i < opened; i++ {
				predicted = append(predicted, "create .pose/actions/act-<id>.jsonl")
			}
		}
	}

	if code, out := runPose(t, repo, "update", "--no-self"); code != 0 {
		t.Fatalf("update: %s", out)
	}
	afterReal, _ := hashTree(repo, true)
	actual := []string{}
	for _, change := range diffTrees(before, afterReal) {
		if change.verb == "create" && isActionRequestFile(change.path) {
			change.path = ".pose/actions/act-<id>.jsonl"
		}
		actual = append(actual, change.verb+" "+change.path)
	}
	sort.Strings(predicted)
	sort.Strings(actual)
	if strings.Join(predicted, "\n") != strings.Join(actual, "\n") {
		t.Fatalf("the dry-run predicted\n%s\nand the update changed\n%s", strings.Join(predicted, "\n"), strings.Join(actual, "\n"))
	}
	if len(actual) < 2 {
		t.Fatalf("the fixture exercised too little: %v", actual)
	}
	if _, out := runPose(t, repo, "doctor"); strings.Contains(out, "last updated by POSE") {
		t.Fatalf("doctor still warns after the update: %s", out)
	}
	if code, out := runPose(t, repo, "update", "--dry-run"); code != 0 || !strings.Contains(out, "the update would change nothing") {
		t.Fatalf("a current instance's dry-run does not say nothing would change: %s", out)
	}
}

// --force goes through install, which needs a real repository in the copy.
func TestUpdateDryRunWithForceRunsOnTheCopy(t *testing.T) {
	repo := olderInstance(t)
	before, _ := hashTree(repo, true)
	code, out := runPose(t, repo, "update", "--dry-run", "--force")
	if code != 0 || strings.Contains(out, "would fail") || !strings.Contains(out, "Result: DRY-RUN") {
		t.Fatalf("dry-run --force: %d %s", code, out)
	}
	after, _ := hashTree(repo, true)
	if changes := diffTrees(before, after); len(changes) != 0 {
		t.Fatalf("the dry-run --force wrote: %+v", changes)
	}
}

// A symlinked .pose/policy is read through by the update, so the dry-run
// copy follows it instead of predicting a policy-less update (found in review).
func TestUpdateDryRunFollowsASymlinkedPolicy(t *testing.T) {
	repo := olderInstance(t)
	want := dryRunOutput(t, repo)
	shared := filepath.Join(t.TempDir(), "policy")
	if err := os.Rename(filepath.Join(repo, ".pose/policy"), shared); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(shared, filepath.Join(repo, ".pose/policy")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if got := dryRunOutput(t, repo); got != want {
		t.Fatalf("the dry-run differs through a symlinked policy:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func dryRunOutput(t *testing.T, repo string) string {
	t.Helper()
	code, out := runPose(t, repo, "update", "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run: %d %s", code, out)
	}
	lines := []string{}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "would ") || strings.HasPrefix(line, "Result:") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// The copy keeps links as links: an update that refuses a symlinked managed
// directory makes the dry-run say the update would fail, and nothing the
// dry-run runs writes through the link into the instance (found in review).
func TestUpdateDryRunMeetsTheLinksTheUpdateMeets(t *testing.T) {
	repo := olderInstance(t)
	shared := filepath.Join(t.TempDir(), "templates")
	if err := os.Rename(filepath.Join(repo, ".pose/templates"), shared); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(shared, filepath.Join(repo, ".pose/templates")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	before, _ := hashTree(shared, false)
	repoBefore, _ := hashTree(repo, true)
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	if after, _ := hashTree(shared, false); len(diffTrees(before, after)) != 0 {
		t.Fatal("the dry-run wrote through the link")
	}
	if after, _ := hashTree(repo, true); len(diffTrees(repoBefore, after)) != 0 {
		t.Fatal("the dry-run wrote into the instance")
	}
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if (dryCode == 0) != (realCode == 0) || (realCode != 0) != strings.Contains(dryOut, "would fail") {
		t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
}

// A directory reached through a link and by its own path, holding links and
// read-only files, is copied once (found in review: .claude/skills ->
// ../.agents/skills made the dry-run fail where the update succeeds).
func TestUpdateDryRunCopiesALinkedDirectoryOnce(t *testing.T) {
	repo := olderInstance(t)
	skills := filepath.Join(repo, ".agents/skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(skills, "local", "notes.md"), "local\n")
	if err := os.Chmod(filepath.Join(skills, "local", "notes.md"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("notes.md", filepath.Join(skills, "local", "alias.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_ = os.RemoveAll(filepath.Join(repo, ".claude/skills"))
	if err := os.MkdirAll(filepath.Join(repo, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../.agents/skills", filepath.Join(repo, ".claude/skills")); err != nil {
		t.Fatal(err)
	}
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if dryCode != realCode || strings.Contains(dryOut, "preparing the dry-run copy") {
		t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
}

// A link to one file inside a directory, copied before a link to the
// directory itself, does not stop the rest of the directory from being
// copied (found in review).
func TestUpdateDryRunCopiesADirectoryAfterALinkIntoIt(t *testing.T) {
	repo := olderInstance(t)
	want := dryRunOutput(t, repo)
	config := filepath.Join(repo, "config")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(repo, ".pose/policy"), filepath.Join(config, "policy")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../config/policy", filepath.Join(repo, ".pose/policy")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(config, "policy"))
	if len(entries) < 2 {
		t.Fatalf("fixture: the policy has %d file(s)", len(entries))
	}
	if err := os.Symlink("../config/policy/"+entries[0].Name(), filepath.Join(repo, ".pose/a-policy-alias")); err != nil {
		t.Fatal(err)
	}
	if got := dryRunOutput(t, repo); got != want {
		t.Fatalf("a link into the policy hid the rest of it:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

// A link to a directory that contains it is linked, not copied into itself,
// and the dry-run agrees with the update (found in review).
func TestUpdateDryRunLinksADirectoryThatContainsTheLink(t *testing.T) {
	for name, link := range map[string][2]string{"self": {".pose/self", "."}, "parent": {".agents/up", ".."}} {
		t.Run(name, func(t *testing.T) {
			repo := olderInstance(t)
			if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(link[0])), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(link[1], filepath.Join(repo, link[0])); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
			realCode, realOut := runPose(t, repo, "update", "--no-self")
			if dryCode != realCode || strings.Contains(dryOut, "preparing the dry-run copy") {
				t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
			}
		})
	}
}

// A broken link never points the copy at its original target: the update
// may create that target, and the dry-run writes nothing (found in review).
func TestUpdateDryRunNeverWritesThroughABrokenLink(t *testing.T) {
	repo := olderInstance(t)
	outside := filepath.Join(t.TempDir(), "outside-security.md")
	inside := filepath.Join(repo, ".pose", "missing-inside.md")
	_ = os.Remove(filepath.Join(repo, ".pose/rules/security.md"))
	if err := os.MkdirAll(filepath.Join(repo, ".pose/rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, ".pose/rules/security.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink("../missing-inside.md", filepath.Join(repo, ".pose/rules/inside.md")); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(repo, ".pose/state/machinery-manifest.json"))
	before, _ := hashTree(repo, true)
	runPose(t, repo, "update", "--dry-run")
	for _, path := range []string{outside, inside} {
		if _, err := os.Lstat(path); err == nil {
			t.Fatalf("the dry-run created %s through a broken link", path)
		}
	}
	if after, _ := hashTree(repo, true); len(diffTrees(before, after)) != 0 {
		t.Fatal("the dry-run wrote into the instance")
	}
}

// A broken link whose missing target sits in an existing directory: the
// update writes through it, so the dry-run must agree that it succeeds, and
// still write nothing outside the copy (found in review).
func TestUpdateDryRunAgreesOnABrokenLinkIntoAnExistingDirectory(t *testing.T) {
	repo := olderInstance(t)
	outside := filepath.Join(t.TempDir(), "outside-security.md")
	_ = os.Remove(filepath.Join(repo, ".pose/rules/security.md"))
	if err := os.Symlink(outside, filepath.Join(repo, ".pose/rules/security.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_ = os.Remove(filepath.Join(repo, ".pose/state/machinery-manifest.json"))
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	if _, err := os.Lstat(outside); err == nil {
		t.Fatal("the dry-run created the external target")
	}
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if dryCode != realCode || (realCode == 0) == strings.Contains(dryOut, "would fail") {
		t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
}

// A broken link at the top of the instance is copied too: the update fails
// through it, and so must the dry-run (found in review).
func TestUpdateDryRunKeepsABrokenTopLevelLink(t *testing.T) {
	repo := olderInstance(t)
	_ = os.Remove(filepath.Join(repo, "AGENTS.md"))
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing", "AGENTS.md"), filepath.Join(repo, "AGENTS.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run", "--force")
	realCode, realOut := runPose(t, repo, "update", "--no-self", "--force")
	if dryCode != realCode {
		t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
}

// A read-only directory stays read-only in the copy: the update fails on it,
// and the dry-run must say so (found in review).
func TestUpdateDryRunKeepsDirectoryPermissions(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	repo := olderInstance(t)
	pose := filepath.Join(repo, ".pose")
	if err := os.Chmod(pose, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pose, 0o755) })
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if dryCode != realCode || (realCode != 0) != strings.Contains(dryOut, "would fail") {
		t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
	leftovers, _ := filepath.Glob(filepath.Join(os.TempDir(), "pose-update-dry-run-*"))
	for _, dir := range leftovers {
		if info, err := os.Stat(dir); err == nil && time.Since(info.ModTime()) < time.Minute {
			t.Fatalf("the dry-run left its copy behind: %s", dir)
		}
	}
}

// A file nobody may read does not stop the dry-run when the update does not
// read it, inside the instance's root or its managed directories (found in
// review).
func TestUpdateDryRunToleratesUnreadableFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	repo := olderInstance(t)
	for _, rel := range []string{"unrelated.txt", ".pose/notes/private.md"} {
		mustWrite(t, filepath.Join(repo, rel), "secret\n")
		if err := os.Chmod(filepath.Join(repo, rel), 0o000); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(repo, rel)
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	}
	locked := filepath.Join(repo, ".pose", "locked")
	mustWrite(t, filepath.Join(locked, "inside.md"), "x\n")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	dryCode, dryOut := runPose(t, repo, "update", "--dry-run")
	realCode, realOut := runPose(t, repo, "update", "--no-self")
	if dryCode != realCode || strings.Contains(dryOut, "preparing the dry-run copy") || strings.Contains(dryOut, "reading the dry-run copy") {
		t.Fatalf("dry-run %d and update %d disagree:\n%s\n---\n%s", dryCode, realCode, dryOut, realOut)
	}
}

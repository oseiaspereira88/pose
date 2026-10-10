package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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
	}

	if code, out := runPose(t, repo, "update", "--no-self"); code != 0 {
		t.Fatalf("update: %s", out)
	}
	afterReal, _ := hashTree(repo, true)
	actual := []string{}
	for _, change := range diffTrees(before, afterReal) {
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

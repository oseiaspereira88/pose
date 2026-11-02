package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func rawHistoryGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestReleaseCheckRefusesAReplacedTagTree(t *testing.T) {
	root := cliReleaseRepo(t)
	good := rawHistoryGit(t, root, "rev-parse", "HEAD")
	rawHistoryGit(t, root, "rm", "-q", "-r", "--cached", ".pose")
	rawHistoryGit(t, root, "commit", "-q", "-m", "no snapshot")
	bad := rawHistoryGit(t, root, "rev-parse", "HEAD")
	rawHistoryGit(t, root, "tag", "v1.0.0")
	check := func() (int, []string) {
		t.Helper()
		code, out := runPose(t, root, "release", "check", "--version", "v1.0.0", "--strict", "--json")
		var doc struct {
			Valid bool     `json:"valid"`
			Gaps  []string `json:"gaps"`
		}
		if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		if doc.Valid {
			t.Fatalf("a tag with no snapshot in its tree read valid: %s", out)
		}
		return code, doc.Gaps
	}
	code, before := check()
	rawHistoryGit(t, root, "replace", bad, good)
	replaced, after := check()
	if code == 0 || replaced == 0 {
		t.Fatalf("strict check passed: %d %d", code, replaced)
	}
	if strings.Join(before, ";") != strings.Join(after, ";") || !strings.Contains(strings.Join(after, ";"), "snapshot") {
		t.Fatalf("the replace ref changed what the check saw: %v, %v", before, after)
	}
}

func TestReadyTransitionsReadRawHistory(t *testing.T) {
	root := t.TempDir()
	spec := filepath.Join(root, ".pose", "specs", "alpha", "spec.md")
	mustWrite(t, spec, "---\nslug: alpha\nstatus: draft\ncreated_at: 2026-11-02\n---\n\n# Spec: alpha\n")
	artifactGit(t, root, "init", "-q")
	rawHistoryGit(t, root, "add", "-A")
	rawHistoryGit(t, root, "commit", "-q", "-m", "draft")
	draft := rawHistoryGit(t, root, "rev-parse", "HEAD")
	raw, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, spec, strings.Replace(string(raw), "status: draft", "status: in-progress", 1))
	changed := func() []string {
		t.Helper()
		checker := &nativeChecker{root: root}
		return checker.readyTransitionCandidates()
	}
	if got := changed(); len(got) != 1 {
		t.Fatalf("the transition was not seen: %v", got)
	}
	rawHistoryGit(t, root, "add", "-A")
	rawHistoryGit(t, root, "commit", "-q", "-m", "started")
	started := rawHistoryGit(t, root, "rev-parse", "HEAD")
	rawHistoryGit(t, root, "reset", "-q", "--soft", draft)
	rawHistoryGit(t, root, "replace", draft, started)
	if got := changed(); len(got) != 1 {
		t.Fatalf("a replace ref hid the transition: %v", got)
	}
	other := t.TempDir()
	artifactGit(t, other, "init", "-q")
	rawHistoryGit(t, other, "commit", "-q", "--allow-empty", "-m", "clean")
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	if got := changed(); len(got) != 1 {
		t.Fatalf("an inherited GIT_DIR hid the transition: %v", got)
	}
}

func TestReportStagesIntoTheSelectedIndex(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "fixture\n")
	artifactGit(t, root, "init", "-q")
	rawHistoryGit(t, root, "add", "-A")
	rawHistoryGit(t, root, "commit", "-q", "-m", "fixture")
	index := filepath.Join(t.TempDir(), "transaction.index")
	rawHistoryGit(t, root, "read-tree", "--index-output="+index, "HEAD")
	t.Setenv("GIT_INDEX_FILE", index)
	if code, out := runPose(t, root, "report", "--task", "review staging", "--git-stage"); code != 0 {
		t.Fatalf("report %d %s", code, out)
	}
	selected := rawHistoryGit(t, root, "ls-files", "--", ".pose/reports/history")
	os.Unsetenv("GIT_INDEX_FILE")
	unselected := rawHistoryGit(t, root, "ls-files", "--", ".pose/reports/history")
	if selected == "" || unselected != "" {
		t.Fatalf("the report went to the wrong index: selected %q, default %q", selected, unselected)
	}
}

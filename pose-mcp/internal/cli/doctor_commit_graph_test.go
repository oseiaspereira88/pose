package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func doctorCommitGraphFinding(t *testing.T, repo string) doctorFinding {
	t.Helper()
	var finding doctorFinding
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		Main([]string{"doctor", "--json"}, &out, &errB)
		var report doctorJSON
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out.String())
		}
		for _, f := range report.Findings {
			if f.Check == "git.commit-graph" {
				finding = f
			}
		}
	})
	if finding.Check == "" {
		t.Fatal("doctor reported no git.commit-graph check")
	}
	return finding
}

func gitInRepo(t *testing.T, repo string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func removeCommitGraph(t *testing.T, repo string) {
	t.Helper()
	info := filepath.Join(repo, ".git", "objects", "info")
	for _, name := range []string{"commit-graph", "commit-graphs"} {
		if err := os.RemoveAll(filepath.Join(info, name)); err != nil {
			t.Fatal(err)
		}
	}
}

// Spec pose-doctor-recommends-the-commit-graph: R1 a long history without a
// commit-graph with changed-path filters is a next step with a fix; R2 the
// fix writes it and the check turns ok; R3 a graph without the filters is
// recommended again, and a short history needs none.
func TestDoctorRecommendsTheCommitGraph(t *testing.T) {
	repo := doctorFixture(t)
	original := commitGraphHistoryThreshold
	t.Cleanup(func() { commitGraphHistoryThreshold = original })
	for i := 0; i < 4; i++ {
		gitInRepo(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "history")
	}
	removeCommitGraph(t, repo)

	commitGraphHistoryThreshold = 1 << 30
	if f := doctorCommitGraphFinding(t, repo); f.Level != "ok" {
		t.Fatalf("a short history is not ok: %+v", f)
	}

	commitGraphHistoryThreshold = 3
	missing := doctorCommitGraphFinding(t, repo)
	if missing.Level != "next" || missing.FixCode != "git.commit-graph" || missing.RemediationClass != remediationFixable {
		t.Fatalf("a long history without a commit-graph: %+v", missing)
	}

	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		Main([]string{"doctor", "--fix", "--yes", "--only", "git.commit-graph"}, &out, &errB)
	})
	if fixed := doctorCommitGraphFinding(t, repo); fixed.Level != "ok" {
		t.Fatalf("the fix did not satisfy the check: %+v", fixed)
	}
	state, err := readCommitGraphState(repo)
	if err != nil || !state.present || !state.changedPaths {
		t.Fatalf("the fix wrote no commit-graph with changed paths: %+v %v", state, err)
	}

	removeCommitGraph(t, repo)
	gitInRepo(t, repo, "commit-graph", "write", "--reachable", "--no-changed-paths")
	if plain := doctorCommitGraphFinding(t, repo); plain.Level != "next" {
		t.Fatalf("a commit-graph without changed-path filters is not recommended again: %+v", plain)
	}

	removeCommitGraph(t, repo)
	gitInRepo(t, repo, "commit-graph", "write", "--reachable", "--changed-paths", "--split")
	if split := doctorCommitGraphFinding(t, repo); split.Level != "ok" {
		t.Fatalf("a split commit-graph with changed paths is not recognized: %+v", split)
	}
}

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Spec pose-doctor-recommends-the-commit-graph: R1 a long history Git walks
// whole is a next step with a fix; R2 the fix writes the filters and the
// check turns ok; R3 a short history needs none. The answer is Git's own, so
// it follows whichever graph Git loads.
func TestDoctorRecommendsTheCommitGraph(t *testing.T) {
	repo := doctorFixture(t)
	original := commitGraphHistoryThreshold
	t.Cleanup(func() { commitGraphHistoryThreshold = original })
	for i := 0; i < 4; i++ {
		// Each commit changes a file of its own, so a lookup of another path
		// has commits the filters can rule out.
		if err := os.WriteFile(filepath.Join(repo, "zz-history.txt"), []byte(strings.Repeat("x", i+1)), 0o644); err != nil {
			t.Fatal(err)
		}
		gitInRepo(t, repo, "add", "zz-history.txt")
		gitInRepo(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "history")
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

	// Review F3: with a single graph and a chain, Git loads the single graph.
	gitInRepo(t, repo, "commit-graph", "write", "--reachable", "--no-changed-paths")
	if both := doctorCommitGraphFinding(t, repo); both.Level != "next" {
		t.Fatalf("an unfiltered single graph beside a filtered chain is judged by the chain: %+v", both)
	}
	removeCommitGraph(t, repo)
	gitInRepo(t, repo, "commit-graph", "write", "--reachable", "--changed-paths")

	// Review F2: a repository whose graph lives only in an alternate.
	shared := filepath.Join(t.TempDir(), "shared")
	if out, err := exec.Command("git", "clone", "-q", "--shared", repo, shared).CombinedOutput(); err != nil {
		t.Fatalf("clone --shared: %v %s", err, out)
	}
	removeCommitGraph(t, shared)
	state, err := readCommitGraphState(shared)
	if err != nil || !state.filtersUsed {
		t.Fatalf("a graph in an alternate is not seen: %+v %v", state, err)
	}
}

// Review F5: a repository whose history cannot be read is a warning, not a
// silent pass; a root that is not a repository gets no check.
func TestDoctorCommitGraphReportsAnUnreadableHistory(t *testing.T) {
	if _, err := readCommitGraphState(t.TempDir()); !errors.Is(err, errCommitGraphNotApplicable) {
		t.Fatalf("a directory outside Git: %v, want not applicable", err)
	}
	repo := doctorFixture(t)
	original := commitGraphHistoryThreshold
	t.Cleanup(func() { commitGraphHistoryThreshold = original })
	commitGraphHistoryThreshold = 1
	commit := func() {
		gitInRepo(t, repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "history")
	}
	commit()
	commit()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD~1").Output()
	if err != nil {
		t.Fatal(err)
	}
	parent := strings.TrimSpace(string(out))
	if err := os.Remove(filepath.Join(repo, ".git", "objects", parent[:2], parent[2:])); err != nil {
		t.Fatalf("the parent commit is not a loose object: %v", err)
	}
	if f := doctorCommitGraphFinding(t, repo); f.Level != "warn" || !strings.Contains(f.Message, "could not be read") {
		t.Fatalf("an unreadable history: %+v", f)
	}
}

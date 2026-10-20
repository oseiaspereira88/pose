package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestB07R4PartialSupersedingRun(t *testing.T) {
	s, b, now := retentionFixture(t)
	base := ReviewRun{SchemaVersion: 1, Status: ReviewRunCompleted, Scope: "spec:backend", BundleID: b.BundleID, BundleDigest: b.BundleDigest, SealedCommit: b.Payload.Subject.Head, PolicyDigest: directoryDigest(filepath.Join(s.Root, ".pose/policy")), StartedAt: now.Format(time.RFC3339Nano), EndedAt: now.Format(time.RFC3339Nano)}
	run, e := s.recordReviewRun(base, nil)
	if e != nil {
		t.Fatal(e)
	}
	if why := s.ReviewRunStillGoverns(run); why != "" {
		t.Fatalf("control %s", why)
	}
	late := base
	late.Status = ReviewRunStale
	late.Supersedes = run.RunID
	late.EndedAt = now.Add(time.Second).Format(time.RFC3339Nano)
	sup, e := s.recordReviewRun(late, nil)
	if e != nil {
		t.Fatal(e)
	}
	if why := s.ReviewRunStillGoverns(run); why == "" {
		t.Fatal("supersession control accepted")
	}
	raw, _ := json.Marshal(map[string]string{"run_id": sup.RunID})
	os.WriteFile(filepath.Join(s.Root, sup.Path), raw, 0644)
	_, lost, _ := s.ReadReviewRuns("")
	why := s.ReviewRunStillGoverns(run)
	t.Logf("partial record lost=%v governs=%q", lost, why)
	if why == "" {
		t.Error("partial superseding record silently restores completed run")
	}
}
func TestB07R4RunTranscript(t *testing.T) {
	s, b, now := retentionFixture(t)
	base := ReviewRun{SchemaVersion: 1, Status: ReviewRunCompleted, Scope: "spec:backend", BundleID: b.BundleID, BundleDigest: b.BundleDigest, SealedCommit: b.Payload.Subject.Head, PolicyDigest: directoryDigest(filepath.Join(s.Root, ".pose/policy")), StartedAt: now.Format(time.RFC3339Nano), EndedAt: now.Format(time.RFC3339Nano)}
	run, e := s.recordReviewRun(base, []byte("rejected"))
	if e != nil {
		t.Fatal(e)
	}
	if why := s.ReviewRunStillGoverns(run); why != "" {
		t.Fatalf("control %s", why)
	}
	os.Remove(filepath.Join(s.Root, run.TranscriptPath))
	why := s.ReviewRunStillGoverns(run)
	t.Logf("missing transcript governs=%q", why)
	if why == "" {
		t.Error("lost transcript still governs")
	}
}
func TestB07R4DerivedAncestor(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a": "a"})
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	outside := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".pose"), 0755)
	os.Symlink(outside, filepath.Join(root, ".pose/results"))
	os.WriteFile(filepath.Join(outside, "new.json"), []byte("{}"), 0644)
	d, e := WorktreeDivergence(root, head, IsDerivedRegeneration)
	t.Logf("untracked ancestor divergence=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("derived ancestor escapes worktree")
	}
}
func TestB07R4DerivedGitlinkAncestor(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a": "a"})
	sub := snapshotRepo(t, map[string]string{"b": "b"})
	pin := snapshotTestGit(t, sub, "rev-parse", "HEAD")
	os.MkdirAll(filepath.Join(root, "sub"), 0755)
	os.Rename(sub, filepath.Join(root, "sub/repo"))
	snapshotTestGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+pin+",sub/repo")
	snapshotTestGit(t, root, "commit", "-qm", "sub")
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	if d, e := WorktreeDivergence(root, head, nil); e != nil || len(d) > 0 {
		t.Fatalf("control %v %v", d, e)
	}
	outside := filepath.Join(t.TempDir(), "sub")
	os.Rename(filepath.Join(root, "sub"), outside)
	os.Symlink(outside, filepath.Join(root, "sub"))
	os.WriteFile(filepath.Join(root, ".git/info/exclude"), []byte("sub\n"), 0644)
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("gitlink ancestor divergence=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("gitlink ancestor escapes worktree")
	}
}

// The records of the directories that hold records are named like the records of their producer.
// Binding an assessment or a component state to the tool and the inputs that produced it is the
// producer ledger of another batch (B20); until then those two directories are judged by name.
func TestB07R4ArbitraryRegeneration(t *testing.T) {
	for _, name := range []string{".pose/results/arbitrary.txt", ".pose/indexes/generations/not-a-generation/arbitrary.json", ".pose/review-bundles/arbitrary.txt", ".pose/review-runs/arbitrary.txt", ".pose/reports/history/arbitrary.txt"} {
		t.Run(name, func(t *testing.T) {
			root := snapshotRepo(t, map[string]string{"a": "a"})
			head := snapshotTestGit(t, root, "rev-parse", "HEAD")
			p := filepath.Join(root, name)
			os.MkdirAll(filepath.Dir(p), 0755)
			os.WriteFile(p, []byte("not produced by any registered tool"), 0644)
			d, e := WorktreeDivergence(root, head, IsDerivedRegeneration)
			dirty, de := (Store{Root: root}).ungeneratedChanges("alpha", snapshotTestGit(t, root, "rev-parse", "HEAD"))
			t.Logf("path=%s divergence=%v/%v closeout=%v/%v", name, d, e, dirty, de)
			if e == nil && len(d) == 0 {
				t.Error("arbitrary output accepted without producer or input authorization")
			}
		})
	}
}
func r4bRun(t *testing.T) (Store, ReviewRun) {
	t.Helper()
	s, b, now := retentionFixture(t)
	base := ReviewRun{SchemaVersion: 1, Status: ReviewRunCompleted, Scope: "spec:backend", BundleID: b.BundleID, BundleDigest: b.BundleDigest, SealedCommit: b.Payload.Subject.Head, PolicyDigest: directoryDigest(filepath.Join(s.Root, ".pose/policy")), StartedAt: now.Format(time.RFC3339Nano), EndedAt: now.Format(time.RFC3339Nano)}
	r, e := s.recordReviewRun(base, []byte("original transcript"))
	if e != nil {
		t.Fatal(e)
	}
	if why := s.ReviewRunStillGoverns(r); why != "" {
		t.Fatal("control: " + why)
	}
	return s, r
}
func TestB07R4BRunFields(t *testing.T) {
	for _, field := range []string{"schema_version", "output", "status", "supersedes", "run_id", "transcript_digest", "transcript_path", "ended_at", "exit_code"} {
		t.Run(field, func(t *testing.T) {
			s, r := r4bRun(t)
			raw, _ := os.ReadFile(filepath.Join(s.Root, r.Path))
			var obj map[string]any
			json.Unmarshal(raw, &obj)
			switch field {
			case "schema_version":
				obj[field] = 99
			case "exit_code":
				obj[field] = 97
			case "output":
				obj[field] = "forged approval"
			case "ended_at":
				obj[field] = "not an instant"
			case "status":
				obj[field] = "failed"
			case "run_id":
				obj[field] = "other"
			case "transcript_path":
				obj[field] = "../outside"
			default:
				obj[field] = "different"
			}
			raw, _ = json.Marshal(obj)
			os.WriteFile(filepath.Join(s.Root, r.Path), raw, 0644)
			runs, lost, e := s.ReadReviewRuns("")
			if e != nil {
				t.Fatal(e)
			}
			candidate := r
			if field != "status" && len(runs) > 0 {
				candidate = runs[0]
			}
			why := s.ReviewRunStillGoverns(candidate)
			t.Logf("field=%s lost=%v reason=%q", field, lost, why)
			if why == "" {
				t.Error("invalid persisted run still governs")
			}
		})
	}
}
func TestB07R4BRunRootLink(t *testing.T) {
	s, r := r4bRun(t)
	outside := filepath.Join(t.TempDir(), "runs")
	os.Rename(filepath.Join(s.Root, reviewRunsDir), outside)
	os.Symlink(outside, filepath.Join(s.Root, reviewRunsDir))
	why := s.ReviewRunStillGoverns(r)
	t.Logf("linked runs root reason=%q", why)
	if why == "" {
		t.Error("linked run storage accepted")
	}
}
func TestB07R4BRunOrdering(t *testing.T) {
	s, r := r4bRun(t)
	a := r
	a.RunID = ""
	a.StartedAt = "2026-10-18T00:00:00Z"
	a.EndedAt = a.StartedAt
	first, e := s.recordReviewRun(a, nil)
	if e != nil {
		t.Fatal(e)
	}
	a.RunID = ""
	a.StartedAt = "2026-10-18T00:00:00.1Z"
	a.EndedAt = a.StartedAt
	second, e := s.recordReviewRun(a, nil)
	if e != nil {
		t.Fatal(e)
	}
	runs, _, _ := s.ReadReviewRuns("")
	i, j := -1, -1
	for k, r := range runs {
		if r.RunID == first.RunID {
			i = k
		}
		if r.RunID == second.RunID {
			j = k
		}
	}
	t.Logf("earlier index=%d later index=%d", i, j)
	if i > j {
		t.Error("runs ordered by timestamp text rather than instant")
	}
}
func TestB07R4BUnknownTopLevel(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a": "sealed"})
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	sub := filepath.Join(root, "copy")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "a"), []byte("sealed"), 0644)
	os.WriteFile(filepath.Join(root, ".git/info/exclude"), []byte("copy/\n"), 0644)
	snapshotTestGit(t, root, "update-index", "--assume-unchanged", "a")
	os.WriteFile(filepath.Join(root, "a"), []byte("changed"), 0644)
	if d, e := WorktreeDivergence(sub, head, nil); e != nil || len(d) == 0 {
		t.Fatalf("known-root control %v/%v", d, e)
	}
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	body := "#!/bin/sh\nif test \"$3\" = rev-parse && test \"$4\" = --show-toplevel; then exit 97; fi\nexec '" + real + "' \"$@\"\n"
	os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	d, e := WorktreeDivergence(sub, head, nil)
	t.Logf("unknown root divergence=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("unknown worktree root converted to clean copy")
	}
}
func TestB07R4BMissingRun(t *testing.T) {
	s, r := r4bRun(t)
	os.Remove(filepath.Join(s.Root, r.Path))
	why := s.ReviewRunStillGoverns(r)
	t.Logf("missing current record reason=%q", why)
	if why == "" {
		t.Error("run without persisted record still governs")
	}
}
func TestB07R4BOtherWorktreeConfig(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a": "sealed"})
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	clean := filepath.Join(t.TempDir(), "clean")
	snapshotTestGit(t, root, "clone", "-q", root, clean)
	os.WriteFile(filepath.Join(root, "a"), []byte("changed"), 0644)
	snapshotTestGit(t, root, "config", "core.worktree", clean)
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("redirected configured worktree divergence=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("dirty root certified with another configured worktree")
	}
}

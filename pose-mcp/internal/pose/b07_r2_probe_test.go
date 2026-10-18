package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestB07R2DirtyGitlink(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"root.txt": "r"})
	sub := filepath.Join(root, "module")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "lib.go"), []byte("sealed"), 0644)
	snapshotGit(t, sub, "init", "-q")
	snapshotGit(t, sub, "add", ".")
	snapshotGit(t, sub, "commit", "-qm", "sub")
	pin := snapshotGit(t, sub, "rev-parse", "HEAD")
	snapshotGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+pin+",module")
	snapshotGit(t, root, "commit", "-qm", "pin")
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	if d, e := WorktreeDivergence(root, head, nil); e != nil || len(d) != 0 {
		t.Fatalf("control %v %v", d, e)
	}
	snapshotGit(t, sub, "update-index", "--assume-unchanged", "lib.go")
	os.WriteFile(filepath.Join(sub, "lib.go"), []byte("changed"), 0644)
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("dirty gitlink diverged=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("materialized gitlink with changed content accepted")
	}
	os.WriteFile(filepath.Join(sub, "lib.go"), []byte("sealed"), 0644)
	outside := filepath.Join(t.TempDir(), "moved")
	os.Rename(sub, outside)
	os.Symlink(outside, sub)
	d, e = WorktreeDivergence(root, head, nil)
	t.Logf("linked gitlink diverged=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("gitlink symlink escaping worktree accepted")
	}

}
func TestB07R2CloseoutType(t *testing.T) {
	for _, name := range []string{".pose/state/project-state.md", ".pose/results/delivery-validation.json", ".pose/closeout-plans/arbitrary.json"} {
		root := snapshotRepo(t, map[string]string{name: "sealed"})
		p := filepath.Join(root, name)
		os.Remove(p)
		os.Symlink(filepath.Join(t.TempDir(), "outside"), p)
		d, e := (Store{Root: root}).ungeneratedChanges()
		t.Logf("path=%s dirty=%v err=%v", name, d, e)
		if e == nil && len(d) == 0 {
			t.Errorf("closeout accepts type replacement %s", name)
		}
	}
}
func TestB07R2PersistentUnknown(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/project.json": `{"schema_version":1,"project_id":"proj.probe"}`, "a": "a"})
	s := Store{Root: root}
	os.WriteFile(filepath.Join(root, ".git/index"), []byte("broken"), 0644)
	before := s.CurrentObligationSnapshot()
	changed := s.ObligationSnapshotChanges(before)
	t.Logf("coherent=%v changes=%v", before.Coherent, changed)
	if !before.Coherent && len(changed) == 0 {
		t.Error("persistent unknown returns empty revalidation")
	}
}
func TestB07R2NegativeAcrossBundles(t *testing.T) {
	s, old, now := retentionFixture(t)
	profile := filepath.Join(s.Root, ".pose/review-profiles/spec-closeout.json")
	raw, _ := os.ReadFile(profile)
	var next ReviewBundle
	found := false
	for i := 1; i <= 80; i++ {
		os.WriteFile(profile, append(append([]byte{}, raw...), []byte(strings.Repeat(" ", i))...), 0644)
		b, e := s.SealReviewBundle("spec:backend", now)
		if e != nil {
			t.Fatal(e)
		}
		if b.BundleID < old.BundleID {
			next = b
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no lower hash")
	}
	reject := approvedBundleAttestation(next, "agent:reject")
	reject.Decision = "changes-requested"
	if _, e := s.RecordReviewAttestation(reject, now.Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	closeBackendSpec(t, s)
	addMatrixCheck(t, s)
	v, e := s.VerifyReviewBundle("spec:backend")
	t.Logf("old=%s negativeBundle=%s approved=%v state=%s err=%v", old.BundleID, next.BundleID, v.Approved, v.State, e)
	if e == nil && v.Approved {
		t.Error("closed scope retains approval over negative bundle of same instant")
	}
}

func TestB07R2PolicyDigestFraming(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "a"), []byte("x\x00z.json\x00{}"), 0644)
	before := directoryDigest(d)
	os.WriteFile(filepath.Join(d, "a"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(d, "z.json"), []byte("{}"), 0644)
	after := directoryDigest(d)
	t.Logf("before=%s after=%s", before, after)
	if before == after {
		t.Error("different policy files serialize to identical digest input")
	}
}
func TestB07R2IrrelevantGitlink(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a": "a"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	snapshotGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+head+",module")
	snapshotGit(t, root, "commit", "-qm", "pin")
	head = snapshotGit(t, root, "rev-parse", "HEAD")
	pre, pe := WorktreeDivergenceFor(root, head, nil, func(string) bool { return false })
	post, po := WorktreeDivergence(root, head, IsDerivedRegeneration)
	t.Logf("irrelevant gitlink pre=%v/%v post=%v/%v", pre, pe, post, po)
}
func TestB07R2OtherWorktreeEnvironment(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"lib.go": "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	other := t.TempDir()
	os.WriteFile(filepath.Join(other, "lib.go"), []byte("sealed"), 0644)
	os.WriteFile(filepath.Join(root, "lib.go"), []byte("changed"), 0644)
	t.Setenv("GIT_DIR", filepath.Join(root, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	d, e := WorktreeDivergence(root, head, nil)
	status, se := ReadGitStatus(root)
	t.Logf("requested=%s env-worktree=%s divergence=%v/%v status=%v/%v", root, other, d, e, status, se)
	if e == nil && len(d) == 0 {
		t.Error("changed worktree verified clean using another materialization from environment")
	}
}

func TestB07R2CloseoutUnknownHead(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/specs/alpha/spec.md": "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\n---\n# alpha\n", ".pose/policy/review.json": `{"schema_version":2,"enabled":false}`})
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	body := "#!/bin/sh\ncase \"$*\" in *\"rev-parse --verify HEAD\"*) exit 97;; esac\nexec '" + real + "' \"$@\"\n"
	os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	p, e := (Store{Root: root}).PlanCloseout("spec:alpha")
	t.Logf("revision=%q steps=%+v err=%v", p.Inputs.Revision, p.Steps, e)
	if e == nil && p.Inputs.Revision == "" && len(p.Steps) > 0 && p.Steps[0].State != CloseoutStateBlocked {
		t.Error("closeout context ready with unknown HEAD")
	}
}
func TestB07R2LegacyNegativeSameInstant(t *testing.T) {
	root, s := reviewFixture(t)
	recordApprovedReview(t, s, "rvw-z", "spec:alpha", "spec-closeout@1", "requirements", "")
	recordApprovedReview(t, s, "rvw-a", "spec:alpha", "spec-closeout@1", "requirements", "")
	p := filepath.Join(root, ".pose/reviews/rvw-a.md")
	raw, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(raw), "decision: approved", "decision: changes-requested", 1)), 0644)
	v, e := s.ReviewCheck("spec:alpha")
	t.Logf("approved=%v selected=%s err=%v", v.Approved, v.Current.ReviewID, e)
	if e == nil && v.Approved {
		t.Error("legacy same-instant rejection hidden by textual id")
	}
}
func TestB07R2SnapshotCleanToDirty(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/project.json": `{"schema_version":1,"project_id":"proj.probe"}`, ".pose/specs/alpha/spec.md": "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\n---\n# alpha\n"})
	s := Store{Root: root}
	before := s.CurrentObligationSnapshot()
	os.WriteFile(filepath.Join(root, ".pose/specs/alpha/spec.md"), []byte("changed"), 0644)
	after := s.CurrentObligationSnapshot()
	changes := s.ObligationSnapshotChanges(before)
	t.Logf("dirty before=%v after=%v changes=%v", before.WorktreeDirty, after.WorktreeDirty, changes)
	if !before.WorktreeDirty && after.WorktreeDirty && len(changes) == 0 {
		t.Error("clean-to-dirty transition omitted from snapshot revalidation")
	}
}

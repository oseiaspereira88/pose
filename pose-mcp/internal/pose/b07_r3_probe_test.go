package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestB07R3AncestorSymlink(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"src/lib.go": "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	if d, e := WorktreeDivergence(root, head, nil); e != nil || len(d) > 0 {
		t.Fatalf("control %v %v", d, e)
	}
	outside := filepath.Join(t.TempDir(), "src")
	if e := os.Rename(filepath.Join(root, "src"), outside); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(root, "src")); e != nil {
		t.Fatal(e)
	}
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("ancestor symlink divergence=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("ancestor symlink escaping root accepted")
	}
}
func TestB07R3CheckpointScope(t *testing.T) {
	name := ".pose/closeout-plans/not-this-scope.json"
	root := snapshotRepo(t, map[string]string{name: "sealed"})
	os.WriteFile(filepath.Join(root, name), []byte("changed"), 0644)
	d, e := (Store{Root: root}).ungeneratedChanges("spec:alpha")
	t.Logf("arbitrary checkpoint dirty=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("modified unrelated checkpoint accepted as generated")
	}
}
func TestB07R3TiedBundleLostAttestation(t *testing.T) {
	s, old, now := retentionFixture(t)
	p := filepath.Join(s.Root, ".pose/review-profiles/spec-closeout.json")
	raw, _ := os.ReadFile(p)
	var next ReviewBundle
	for i := 1; i <= 100; i++ {
		os.WriteFile(p, append(append([]byte{}, raw...), []byte(strings.Repeat(" ", i))...), 0644)
		b, e := s.SealReviewBundle("spec:backend", now)
		if e != nil {
			t.Fatal(e)
		}
		if b.BundleID < old.BundleID {
			next = b
			break
		}
	}
	if next.BundleID == "" {
		t.Fatal("no lower hash")
	}
	reject := approvedBundleAttestation(next, "agent:reject")
	reject.Decision = "changes-requested"
	att, e := s.RecordReviewAttestation(reject, now.Add(time.Minute))
	if e != nil {
		t.Fatal(e)
	}
	closeBackendSpec(t, s)
	addMatrixCheck(t, s)
	v, e := s.VerifyReviewBundle("spec:backend")
	if e != nil || v.Approved {
		t.Fatalf("negative control approved=%v err=%v", v.Approved, e)
	}
	p = filepath.Join(s.Root, filepath.FromSlash(att.Path))
	data, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	data = []byte(strings.Replace(string(data), `"decision": "changes-requested"`, `"decision": 17`, 1))
	os.WriteFile(p, data, 0644)
	read, _ := s.ReadReviewAttestations(next.BundleID)
	t.Logf("loss=%+v", read.Coverage)
	v, e = s.VerifyReviewBundle("spec:backend")
	t.Logf("lost rejection approved=%v state=%s err=%v", v.Approved, v.State, e)
	if e == nil && v.Approved {
		t.Error("approval retained over lost tied sibling attestation")
	}
}
func TestB07R3SnapshotRecoversUnknown(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/project.json": `{"schema_version":1,"project_id":"proj.probe"}`, "a": "a"})
	s := Store{Root: root}
	p := filepath.Join(root, ".git/index")
	raw, _ := os.ReadFile(p)
	os.WriteFile(p, []byte("broken"), 0644)
	before := s.CurrentObligationSnapshot()
	os.WriteFile(p, raw, 0644)
	changed := s.ObligationSnapshotChanges(before)
	t.Logf("before coherent=%v now=%v changes=%v", before.Coherent, s.CurrentObligationSnapshot().Coherent, changed)
	if !before.Coherent && len(changed) == 0 {
		t.Error("unknown baseline revalidated as unchanged after recovery")
	}
}
func TestB07R3RunFields(t *testing.T) {
	s, b, _ := retentionFixture(t)
	run := ReviewRun{RunID: "rrn-probe", Status: ReviewRunCompleted, Scope: "spec:backend", BundleID: b.BundleID, BundleDigest: "sha256:wrong", SealedCommit: "wrong", PolicyDigest: directoryDigest(filepath.Join(s.Root, ".pose/policy"))}
	reason := s.ReviewRunStillGoverns(run)
	t.Logf("wrong digest and commit reason=%q", reason)
	if reason == "" {
		t.Error("run with wrong digest and commit governs")
	}
}
func TestB07R3LostSupersedingRun(t *testing.T) {
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
	late.Failure = "stale"
	late.EndedAt = now.Add(time.Second).Format(time.RFC3339Nano)
	sup, e := s.recordReviewRun(late, nil)
	if e != nil {
		t.Fatal(e)
	}
	if why := s.ReviewRunStillGoverns(run); why == "" {
		t.Fatal("supersession control accepted")
	}
	p := filepath.Join(s.Root, sup.Path)
	raw, _ := os.ReadFile(p)
	os.WriteFile(p, append(raw, []byte("broken")...), 0644)
	why := s.ReviewRunStillGoverns(run)
	t.Logf("lost superseding run reason=%q", why)
	if why == "" {
		t.Error("lost superseding run treated as absent")
	}
}
func TestB07R3ConfigRedirect(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"lib.go": "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	other := t.TempDir()
	os.WriteFile(filepath.Join(other, "lib.go"), []byte("sealed"), 0644)
	os.WriteFile(filepath.Join(root, "lib.go"), []byte("changed"), 0644)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.worktree")
	t.Setenv("GIT_CONFIG_VALUE_0", other)
	d, e := WorktreeDivergence(root, head, nil)
	entries, se := ReadGitStatus(root)
	t.Logf("config redirect divergence=%v/%v status=%v/%v", d, e, entries, se)
	if e == nil && len(d) == 0 {
		t.Error("changed root verified against config-redirected worktree")
	}
}
func TestB07R3QuotedGitlink(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a": "a"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	p := "pose-mcp/module é"
	snapshotGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+head+","+p)
	snapshotGit(t, root, "commit", "-qm", "pin")
	got := reviewBundleGitlinks(root, []string{p})
	t.Logf("quoted gitlink mapping=%v", got)
	if got[p] != "" {
		t.Log("mapping is present")
	}
}
func TestB07R3CustomResultsPath(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/results/custom.json": "old", ".pose/policy/delivery.json": `{"schema_version":1,"enabled":true,"results_path":".pose/results/custom.json"}`})
	os.WriteFile(filepath.Join(root, ".pose/results/custom.json"), []byte("new"), 0644)
	d, e := (Store{Root: root}).ungeneratedChanges("spec:alpha")
	t.Logf("custom producer output dirty=%v err=%v", d, e)
	if e != nil || len(d) != 1 {
		t.Fatalf("expected conservative refusal %v %v", d, e)
	}
}
func TestB07R3RootWhitespace(t *testing.T) {
	original := snapshotRepo(t, map[string]string{"lib.go": "sealed"})
	base := t.TempDir()
	root := filepath.Join(base, "repo ")
	if e := os.Rename(original, root); e != nil {
		t.Fatal(e)
	}
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	snapshotGit(t, base, "clone", "-q", root, filepath.Join(base, "repo"))
	os.WriteFile(filepath.Join(root, "lib.go"), []byte("changed"), 0644)
	entries, se := ReadGitStatus(root)
	if se != nil || len(entries) == 0 {
		t.Fatalf("dirty control %v %v", entries, se)
	}
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("trailing-space root divergence=%v err=%v", d, e)
	if e == nil && len(d) == 0 {
		t.Error("root whitespace trimmed into another clean repository")
	}
}

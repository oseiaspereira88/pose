package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestB07IndependentCloseoutPrefix(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/indexes/validation-matrix.json": "{}", ".pose/results/sealed.json": "sealed"})
	for _, p := range []string{".pose/indexes/validation-matrix.json", ".pose/results/sealed.json"} {
		os.WriteFile(filepath.Join(root, p), []byte("changed"), 0644)
	}
	dirty, err := (Store{Root: root}).ungeneratedChanges("spec:alpha")
	t.Logf("dirty=%v err=%v", dirty, err)
	if len(dirty) == 0 && err == nil {
		t.Error("close context accepts changed matrix and sealed evidence")
	}
}
func TestB07IndependentSubdirectory(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"dir/a.txt": "a", "root.txt": "r"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	d, e := WorktreeDivergence(filepath.Join(root, "dir"), head, nil)
	t.Logf("diverged=%v err=%v", d, e)
	if len(d) > 0 || e != nil {
		t.Error("clean subdirectory compared with repository-root paths")
	}
}
func TestB07IndependentDerivedRules(t *testing.T) {
	for _, p := range []string{".pose/indexes/not-a-produced-index.json", ".pose/assessments/sealed.txt", ".pose/state/arbitrary.txt"} {
		if IsDerivedRegeneration(p, true) {
			t.Errorf("arbitrary tracked path allowed: %s", p)
		}
	}
	root := snapshotRepo(t, map[string]string{".pose/state/sealed.txt": "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	os.Remove(filepath.Join(root, ".pose/state/sealed.txt"))
	d, e := WorktreeDivergence(root, head, IsDerivedRegeneration)
	t.Logf("deleted tracked sealed path diverged=%v err=%v", d, e)
	if len(d) == 0 && e == nil {
		t.Error("deletion exempted as regeneration")
	}
}
func TestB07IndependentSameSecondSupersession(t *testing.T) {
	store, _, now := retentionFixture(t)
	// Try several different second-resolution seals: new bundle hashes need not sort later.
	for i := 0; i < 20; i++ {
		old, e := store.SealReviewBundle("spec:backend", now)
		if e != nil {
			t.Fatal(e)
		}
		addMatrixCheck(t, store)
		// vary a semantic input as well, since addMatrixCheck is idempotent.
		p := filepath.Join(store.Root, ".pose/review-profiles/spec-closeout.json")
		raw, _ := os.ReadFile(p)
		os.WriteFile(p, append(raw, byte(' ')), 0644)
		next, e := store.SealReviewBundle("spec:backend", now.Add(time.Duration(i%2)*time.Millisecond))
		if e != nil {
			t.Fatal(e)
		}
		if old.BundleID == next.BundleID {
			continue
		}
		plan := DispatchPlan{Bundle: old, Brief: ReviewBrief{Scope: "spec:backend"}, PolicyDigest: directoryDigest(filepath.Join(store.Root, ".pose/policy"))}
		reason := store.dispatchNoLongerGoverns(plan)
		t.Logf("old=%s new=%s reason=%q", old.BundleID, next.BundleID, reason)
		if old.BundleID > next.BundleID && !strings.Contains(reason, "superseded") {
			t.Error("same-second new bundle missed by hash text ordering")
			return
		}
	}
}
func TestB07IndependentSnapshotRevalidation(t *testing.T) {
	root := snapshotRepo(t, map[string]string{".pose/project.json": `{"schema_version":1,"project_id":"proj.probe"}`, "a.txt": "a"})
	s := Store{Root: root}
	before := s.CurrentObligationSnapshot()
	os.WriteFile(filepath.Join(root, ".git/index"), []byte("broken"), 0644)
	after := s.CurrentObligationSnapshot()
	changed := s.ObligationSnapshotChanges(before)
	t.Logf("coherent before=%v after=%v changes=%v digestEqual=%v", before.Coherent, after.Coherent, changed, before.Digest == after.Digest)
	if before.Coherent && !after.Coherent && len(changed) == 0 {
		t.Error("write-gate revalidation ignores loss of coherence")
	}
}
func TestB07IndependentParentLink(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"dir/a.txt": "a"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	outside := filepath.Join(t.TempDir(), "moved")
	os.Rename(filepath.Join(root, "dir"), outside)
	os.Symlink(outside, filepath.Join(root, "dir"))
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("parent symlink diverged=%v err=%v", d, e)
	if len(d) == 0 && e == nil {
		t.Error("worktree escapes through parent symlink")
	}
}
func TestB07IndependentGitlink(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.txt": "a"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	snapshotGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+head+",module")
	snapshotGit(t, root, "commit", "-qm", "gitlink")
	head = snapshotGit(t, root, "rev-parse", "HEAD")
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("absent sealed gitlink diverged=%v err=%v", d, e)
	if len(d) == 0 && e == nil {
		t.Error("missing sealed gitlink accepted")
	}
}
func TestB07IndependentBackslashAlias(t *testing.T) {
	name := `.pose\state\source.go`
	root := snapshotRepo(t, map[string]string{name: "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	os.WriteFile(filepath.Join(root, name), []byte("changed"), 0644)
	d, e := WorktreeDivergence(root, head, IsDerivedRegeneration)
	t.Logf("literal backslash name=%q diverged=%v err=%v", name, d, e)
	if len(d) == 0 && e == nil {
		t.Error("literal POSIX filename gains derived exemption by separator rewriting")
	}
}
func TestB07IndependentNegativeSameSecond(t *testing.T) {
	store, bundle, now := retentionFixture(t)
	atts, _ := store.ListReviewAttestations(bundle.BundleID)
	approved := atts[len(atts)-1]
	for i := 0; i < 40; i++ {
		reject := approvedBundleAttestation(bundle, "agent:reject-"+strings.Repeat("x", i+1))
		reject.Decision = "changes-requested"
		rec, e := store.RecordReviewAttestation(reject, now.Add(time.Minute))
		if e != nil {
			t.Fatal(e)
		}
		if rec.AttestationID < approved.AttestationID {
			v, e := store.VerifyReviewBundle("spec:backend")
			t.Logf("approved=%s rejectedLater=%s selected=%v verificationApproved=%v err=%v", approved.AttestationID, rec.AttestationID, v.Attestation.Decision, v.Approved, e)
			if v.Approved {
				t.Error("later negative attestation ignored by hash order")
			}
			return
		}
	}
	t.Fatal("could not construct hash ordering")
}
func TestB07IndependentReadErrorDiagnostic(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.txt": "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	p := filepath.Join(root, "a.txt")
	os.Chmod(p, 0)
	defer os.Chmod(p, 0644)
	if _, e := os.ReadFile(p); e == nil {
		t.Skip("environment can read mode 000")
	}
	d, e := WorktreeDivergence(root, head, nil)
	t.Logf("unreadable input diverged=%v typedError=%v", d, e)
	if e == nil {
		t.Error("input read cause lost; fail closed as divergence")
	}
}
func TestB07IndependentUnknownPolicyDigest(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	os.RemoveAll(filepath.Join(store.Root, ".pose/policy"))
	plan := DispatchPlan{Bundle: bundle, Brief: ReviewBrief{Scope: "spec:backend"}, PolicyDigest: directoryDigest(filepath.Join(store.Root, ".pose/policy"))}
	reason := store.dispatchNoLongerGoverns(plan)
	t.Logf("policyDigest=%q reason=%q", plan.PolicyDigest, reason)
	if reason == "" {
		t.Error("unavailable baseline accepted as current policy")
	}
	plan.PolicyDigest = ""
	reason = store.dispatchNoLongerGoverns(plan)
	if reason == "" {
		t.Error("absent policy digest disables revalidation")
	}
}
func TestB07IndependentLostSupersedingBundle(t *testing.T) {
	store, bundle, now := retentionFixture(t)
	addMatrixCheck(t, store)
	next, e := store.SealReviewBundle("spec:backend", now.Add(time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(store.Root, filepath.FromSlash(next.Path)), []byte("{"), 0644)
	plan := DispatchPlan{Bundle: bundle, Brief: ReviewBrief{Scope: "spec:backend"}, PolicyDigest: directoryDigest(filepath.Join(store.Root, ".pose/policy"))}
	reason := store.dispatchNoLongerGoverns(plan)
	t.Logf("lost newer bundle=%s reason=%q", next.BundleID, reason)
	if !strings.Contains(reason, "could not be read") {
		t.Errorf("lost superseding bundle read as absent: %q", reason)
	}
}

// Regression tests added with the fixes of the independent review: each one holds a
// guarantee that another guard would otherwise hide from the mutation driver.
func TestB07DerivedNamesAreExactAndNeverRemoved(t *testing.T) {
	name := ".pose/state/project-state.md"
	root := snapshotRepo(t, map[string]string{name: "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	os.WriteFile(filepath.Join(root, name), []byte("regenerated"), 0o644)
	if d, e := WorktreeDivergence(root, head, IsDerivedRegeneration); e != nil || len(d) != 0 {
		t.Fatalf("a producer's own file rewritten in place is a regeneration: %v %v", d, e)
	}
	os.Remove(filepath.Join(root, name))
	if d, e := WorktreeDivergence(root, head, IsDerivedRegeneration); e != nil || len(d) != 1 {
		t.Fatalf("the removal of a derived file is a divergence: %v %v", d, e)
	}
}

func TestB07BackslashIsACharacterOfAPosixName(t *testing.T) {
	if filepath.Separator == '\\' {
		t.Skip("windows")
	}
	name := `.pose\state\project-state.md`
	root := snapshotRepo(t, map[string]string{name: "sealed"})
	head := snapshotGit(t, root, "rev-parse", "HEAD")
	os.WriteFile(filepath.Join(root, name), []byte("changed"), 0o644)
	if d, e := WorktreeDivergence(root, head, IsDerivedRegeneration); e != nil || len(d) != 1 {
		t.Fatalf("a literal backslash name does not borrow the exemption of the path it resembles: %v %v", d, e)
	}
}

func TestB07ALinkInAPolicyDirectoryIsNotReadWhole(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0o644)
	if !directoryDigestKnown(directoryDigest(dir)) {
		t.Fatal("a plain directory reads whole")
	}
	os.Symlink(filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json"))
	if got := directoryDigest(dir); got != directoryUnavailable {
		t.Fatalf("a link is not a file that was read, got %q", got)
	}
}

// The closeout writes the checkpoint of a scope: one JSON file directly in its directory.
func TestB07ACloseoutCheckpointIsOneJSONFileOfTheScope(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.txt": "a"})
	write := func(rel string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/closeout-plans/spec-alpha.json")
	if dirty, err := (Store{Root: root}).ungeneratedChanges("spec:alpha"); err != nil || len(dirty) != 0 {
		t.Fatalf("the checkpoint of a scope is generated: %v %v", dirty, err)
	}
	write(".pose/closeout-plans/notes.txt")
	write(".pose/closeout-plans/sub/deep.json")
	dirty, err := (Store{Root: root}).ungeneratedChanges("spec:alpha")
	if err != nil || len(dirty) != 2 {
		t.Fatalf("an arbitrary file under the checkpoint directory was taken as regeneration: %v %v", dirty, err)
	}
}

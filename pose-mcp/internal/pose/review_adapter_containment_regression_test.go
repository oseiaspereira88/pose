package pose

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAdapterContainmentShortSessionExpansionCannotGovern(t *testing.T) {
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Containment.StreamBytes = containmentInt(2048)
		a.Containment.OutputBytes = containmentInt(2048)
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'session: x\ndecision: approved\n'; head -c 1024 /dev/zero | tr '\000' x`}
	})
	if f.run.Status != ReviewRunFailed || f.run.Safety.CaptureComplete {
		t.Fatalf("expanding redaction retained a complete review: %+v", f.run)
	}
	raw, err := os.ReadFile(filepath.Join(f.store.Root, filepath.FromSlash(f.run.TranscriptPath)))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > int(f.run.Safety.StreamBytes) || len(f.run.Output) > int(f.run.Safety.OutputBytes) {
		t.Fatal("redacted retention exceeded configured stream/output bounds")
	}
	proof, err := f.store.SignReviewRunProof(f.run.RunID, f.key, time.Hour, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := f.store.VerifyReviewRunProof(f.run.RunID, proof.ProofID)
	if err != nil || audit.CompleteReview || audit.State != ReviewRunFailed {
		t.Fatal("failed capture authorized review", audit, err)
	}
}

func TestAdapterContainmentCredentialSessionResumesWithPrivateOriginal(t *testing.T) {
	const secret = "synthetic-session-credential-123"
	t.Setenv("FIXTURE_TOKEN", secret)
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Containment.Credentials = []string{"FIXTURE_TOKEN"}
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
		a.Limit = &ReviewerLimit{ExitCodes: []int{75}}
		a.Resume = &ReviewerResume{Command: []string{"sh", "-c", `cat >/dev/null; test "$1" = "$FIXTURE_TOKEN" || exit 1; printf 'decision: approved\n'`, "sh", "{session}"}}
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'session: %s\n' "$FIXTURE_TOKEN"; exit 75`}
	})
	stored, err := f.store.readReviewSession(f.run.SessionDigest)
	if err != nil || stored != secret {
		t.Fatal("redaction changed private continuation reference", err)
	}
	for _, path := range []string{f.run.Path, f.run.TranscriptPath} {
		raw, err := os.ReadFile(filepath.Join(f.store.Root, filepath.FromSlash(path)))
		if err != nil || bytes.Contains(raw, []byte(secret)) {
			t.Fatal("private credential/session leaked to public retention", path, err)
		}
	}
	plan, err := f.store.PlanReviewResume(f.run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || resumed.Status != ReviewRunCompleted || resumed.AttemptID != f.run.AttemptID || resumed.Resumes != f.run.RunID {
		t.Fatal("continuation did not receive its original private session", resumed.Failure, err)
	}
	if strings.Contains(resumed.Output, secret) {
		t.Fatal("resumed public output leaked credential/session")
	}
}

func TestAdapterContainmentOverlappingSessionDoesNotExposeCredentialSuffix(t *testing.T) {
	session := "fixture-session-ref"
	secret := session + "-secret-suffix-123456"
	raw := []byte(session + " " + secret)
	retained, err := redactReviewCapture(raw, []string{secret}, session, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != "[private-session] [credential]" {
		t.Fatal("overlapping session prevented complete credential redaction", string(retained))
	}
}

type containmentCleanupFixture struct {
	store                Store
	tmp, worktree        string
	owned, worktreeOwned os.FileInfo
	marker               []byte
	runtimeRoot          *os.Root
	ownership            *reviewCleanupOwnership
}

func makeContainmentCleanupFixture(t *testing.T) containmentCleanupFixture {
	t.Helper()
	repo := t.TempDir()
	gitbin, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command(gitbin, append([]string{"-C", repo, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "tracked"), []byte("operation data"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-qm", "fixture")
	tmp := filepath.Join(t.TempDir(), "operation")
	if err := os.Mkdir(tmp, 0700); err != nil {
		t.Fatal(err)
	}
	owned, err := os.Lstat(tmp)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	worktree := filepath.Join(tmp, "worktree")
	git("worktree", "add", "--detach", worktree, "HEAD")
	worktreeOwned, err := os.Lstat(worktree)
	if err != nil {
		t.Fatal(err)
	}
	marker, err := os.ReadFile(filepath.Join(worktree, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Root: repo}
	ownership, err := store.captureReviewCleanupOwnership(root, worktree, marker)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ownership.registry.Close() })
	return containmentCleanupFixture{store: store, tmp: tmp, worktree: worktree, owned: owned, worktreeOwned: worktreeOwned, marker: marker, runtimeRoot: root, ownership: ownership}
}
func (f containmentCleanupFixture) cleanup() error {
	return f.store.cleanupReviewResources(f.tmp, f.owned, f.worktree, f.worktreeOwned, f.marker, f.ownership)
}

func TestAdapterContainmentCleanupCapturedRootsRefuseReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux descriptor and symlink qualification")
	}
	for _, kind := range []string{"runtime-symlink", "runtime-missing", "registration-symlink"} {
		t.Run(kind, func(t *testing.T) {
			f := makeContainmentCleanupFixture(t)
			victim := t.TempDir()
			sentinel := filepath.Join(victim, "keep")
			if err := os.WriteFile(sentinel, []byte("foreign data"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "runtime-symlink", "runtime-missing":
				if err := os.Rename(f.tmp, f.tmp+"-owned"); err != nil {
					t.Fatal(err)
				}
				if kind == "runtime-symlink" {
					if err := os.Symlink(victim, f.tmp); err != nil {
						t.Fatal(err)
					}
				}
			case "registration-symlink":
				path := filepath.Join(f.store.Root, ".git", "worktrees", f.ownership.registration)
				if err := os.Rename(path, path+"-owned"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(victim, path); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.cleanup(); err == nil {
				t.Fatal("replaced/missing owned resource reported successful cleanup")
			}
			if raw, err := os.ReadFile(sentinel); err != nil || string(raw) != "foreign data" {
				t.Fatal("foreign resource changed", err)
			}
		})
	}
	// The actual recursive primitive uses the descriptor even after its path is
	// redirected. This tests the former post-validation window without a Git
	// wrapper: there is now no path-based recursive Git command to intercept.
	f := makeContainmentCleanupFixture(t)
	victim := t.TempDir()
	sentinel := filepath.Join(victim, "keep")
	if err := os.WriteFile(sentinel, []byte("foreign data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(f.tmp, f.tmp+"-owned"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, f.tmp); err != nil {
		t.Fatal(err)
	}
	if err := removeReviewOwnedContents(f.runtimeRoot, 100000, 256<<20); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("descriptor-confined removal escaped its root", err)
	}
	if _, err := os.Stat(filepath.Join(f.tmp+"-owned", "worktree")); !os.IsNotExist(err) {
		t.Fatal("owned descriptor was not exercised", err)
	}
}

func TestAdapterContainmentCleanupOnlyOwnRegistrationAndIdempotence(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux descriptor qualification")
	}
	f := makeContainmentCleanupFixture(t)
	unrelated := filepath.Join(f.store.Root, ".git", "worktrees", "unrelated-registration")
	if err := os.Mkdir(unrelated, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(unrelated, "keep")
	if err := os.WriteFile(sentinel, []byte("unrelated registry data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := f.cleanup(); err != nil {
		t.Fatal(err)
	}
	if err := f.cleanup(); err != nil {
		t.Fatal("completed cleanup was not idempotent", err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal("cleanup pruned unrelated registration", err)
	}
	if _, err := os.Stat(filepath.Join(f.store.Root, ".git", "worktrees", f.ownership.registration)); !os.IsNotExist(err) {
		t.Fatal("own registration was not removed", err)
	}
}

func TestAdapterContainmentFailedCleanupContinuationRemainsFailed(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Unix fixture command and registration qualification")
	}
	for _, interrupted := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed-domain", true: "interrupted-domain"}[interrupted], func(t *testing.T) {
			exit := "0"
			if interrupted {
				exit = "75"
			}
			f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
				a.Containment = containmentPolicy()
				a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
				a.Limit = &ReviewerLimit{ExitCodes: []int{75}}
				a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'session: fixture-private-reference\n'; exit 75`}
				a.Resume = &ReviewerResume{Command: []string{"sh", "-c", `cat >/dev/null; registration=$(git rev-parse --absolute-git-dir) || exit 2; printf 'fixture owned lock\n' > "$registration/locked"; printf 'decision: approved\n'; exit ` + exit, "sh", "{session}"}}
			})
			plan, err := f.store.PlanReviewResume(f.run.RunID)
			if err != nil {
				t.Fatal(err)
			}
			next, err := f.store.RunReviewDispatch(plan, time.Now)
			if err == nil || next.RunID == "" || next.Safety.Operation.Outcome != OutcomeFailed {
				t.Fatal("cleanup failure was not retained", next, err)
			}
			if why := f.store.ReviewRunStillGoverns(next); why == "" {
				t.Fatal("cleanup failure governed review")
			}
			retry, err := f.store.PlanReviewResume(f.run.RunID)
			if err != nil || retry.Existing == nil {
				t.Fatal("retry lost its immutable result", err)
			}
			repeated, err := f.store.RunReviewDispatch(retry, time.Now)
			if err == nil || repeated.RunID != next.RunID {
				t.Fatal("idempotent retry hid cleanup failure or respawned", repeated.RunID, err)
			}
			if interrupted {
				if _, err := f.store.PlanReviewResume(next.RunID); err == nil {
					t.Fatal("new continuation admitted beside possible orphan")
				}
			}
			runs, losses, err := f.store.ReadReviewRuns("")
			if err != nil || len(losses) > 0 || len(runs) != 2 {
				t.Fatal("cleanup retry altered run ledger", len(runs), losses, err)
			}
		})
	}
}

func TestAdapterContainmentEscapedFiniteChildIsDisclosedNotContained(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux setsid limitation fixture")
	}
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid unavailable for explicit escape fixture")
	}
	outside := t.TempDir()
	trigger := filepath.Join(outside, "trigger")
	escaped := filepath.Join(outside, "escaped")
	ready := filepath.Join(outside, "ready")
	t.Cleanup(func() { _ = os.WriteFile(trigger, []byte("release finite child"), 0600) })
	f := provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Containment = containmentPolicy()
		a.Command = []string{"sh", "-c", `cat >/dev/null; setsid sh -c 'printf ready > "$3"; n=0; while test "$n" -lt 200; do if test -e "$1"; then printf escaped > "$2"; exit 0; fi; n=$((n+1)); sleep 0.01; done' sh "$1" "$2" "$3" >/dev/null 2>&1 & n=0; while ! test -e "$3"; do n=$((n+1)); test "$n" -lt 100 || exit 2; sleep 0.01; done; printf 'decision: approved\n'`, "sh", trigger, escaped, ready}
	})
	if f.run.Status != ReviewRunCompleted || f.run.Safety.Capabilities["escaped-descendant-termination"] != "unavailable" {
		t.Fatal("escaped process was claimed contained", f.run.Failure, f.run.Safety)
	}
	if _, err := os.Stat(escaped); !os.IsNotExist(err) {
		t.Fatal("escape fixture ran before dispatch returned", err)
	}
	if err := os.WriteFile(trigger, []byte("release finite child"), 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(escaped); err == nil && string(raw) == "escaped" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("escape fixture did not exercise the disclosed limitation")
}

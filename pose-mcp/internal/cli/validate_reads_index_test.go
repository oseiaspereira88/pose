package cli

import (
	"bytes"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// staleIndexRepo is a repository whose index holds a stale stat cache for a
// file whose content did not change, so any read that refreshes the index
// would write it back.
func staleIndexRepo(t *testing.T) (string, []byte) {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.txt", ".pose/specs/2026-11-26-alpha.md"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	later := time.Now().Add(2 * time.Second)
	for _, f := range []string{"a.txt", ".pose/specs/2026-11-26-alpha.md"} {
		if err := os.Chtimes(filepath.Join(root, f), later, later); err != nil {
			t.Fatal(err)
		}
	}
	index, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	return root, index
}

// Spec pose-validate-reads-leave-the-index-alone R1, R2: the reads validate,
// check and report make leave a stale index byte-identical, and still answer
// that nothing changed.
func TestValidationReadsLeaveTheIndexAlone(t *testing.T) {
	root, before := staleIndexRepo(t)
	if !materialWorktreeClean(root) {
		t.Fatal("a stat-only change made the work tree dirty")
	}
	sel, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{})
	if err != nil || len(sel.Changed) != 0 {
		t.Fatalf("the scope diff against the work tree: %v %v", sel.Changed, err)
	}
	if files, err := reportChangedFiles(root, "HEAD"); err != nil || len(files) != 0 {
		t.Fatalf("report listed a stat-only change: %v", files)
	}
	if got := (&nativeChecker{root: root}).readyTransitionCandidates(); len(got) != 0 {
		t.Fatalf("the ready transition read listed a stat-only change: %v", got)
	}
	after, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("a validation read rewrote .git/index")
	}
}

// Spec pose-validate-reads-leave-the-index-alone R3: while those reads run
// concurrently, a reader watching .git/index never sees it change.
func TestConcurrentValidationReadsNeverChangeTheIndex(t *testing.T) {
	root, before := staleIndexRepo(t)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				materialWorktreeClean(root)
				_, _ = computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{})
				_, _ = reportChangedFiles(root, "HEAD")
				(&nativeChecker{root: root}).readyTransitionCandidates()
				// The review history reader runs concurrently too (review
				// round 1, N5).
				_, _ = posemodel.Store{Root: root}.ReviewRunHistoryFindings()
			}
		}()
	}
	changed := make(chan string, 1)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			now, err := os.ReadFile(filepath.Join(root, ".git", "index"))
			if err == nil && !bytes.Equal(before, now) {
				changed <- "the index changed while validation reads ran"
				return
			}
			if _, err := os.Lstat(filepath.Join(root, ".git", "index.lock")); err == nil {
				changed <- "a validation read took the index lock"
				return
			}
		}
	}()
	wg.Wait()
	close(stop)
	select {
	case why := <-changed:
		t.Fatal(why)
	default:
	}
}

// Review round 2, N11: without an index file the reads still answer: the
// untracked spec is a ready-transition candidate and the report lists it.
func TestValidationReadsAnswerWithoutAnIndex(t *testing.T) {
	root, _ := staleIndexRepo(t)
	if err := os.Remove(filepath.Join(root, ".git", "index")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "specs", "2026-11-26-beta.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range (&nativeChecker{root: root}).readyTransitionCandidates() {
		if strings.HasSuffix(c, "2026-11-26-beta.md") {
			found = true
		}
	}
	if !found {
		t.Fatal("the ready transition read lost the new spec without an index")
	}
	if files, err := reportChangedFiles(root, ""); err != nil || len(files) == 0 {
		t.Fatal("report listed nothing without an index")
	}
	if _, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{}); err != nil {
		t.Fatalf("scope selection failed without an index: %v", err)
	}
}

// Spec pose-validate-read-only-index-followups R2: a failed index read is
// reported with its cause by the check, scope selection and the report.
func TestAFailedIndexReadIsReported(t *testing.T) {
	root, _ := staleIndexRepo(t)
	index := filepath.Join(root, ".git", "index")
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(index, 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &out}
	checker.readyTransitionCandidates()
	if checker.errors == 0 || !strings.Contains(out.String(), "cannot be listed") {
		t.Fatalf("the check dropped a failed index read: %d %s", checker.errors, out.String())
	}
	_, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{})
	if err == nil || strings.Contains(err.Error(), "revision unknown?") || !strings.Contains(err.Error(), "index") {
		t.Fatalf("scope selection mislabelled a failed index read: %v", err)
	}
	if _, err := reportChangedFiles(root, "HEAD"); err == nil {
		t.Fatal("report hid a failed index read")
	}
}

// Review round 1, N1–N2 of spec pose-validate-read-only-index-followups: a
// corrupt index, which git itself refuses, is reported as an index failure
// with what git said, not as an unknown revision.
func TestACorruptIndexIsReportedWithItsCause(t *testing.T) {
	root, _ := staleIndexRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("not an index at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &out}
	checker.readyTransitionCandidates()
	if checker.errors == 0 || !strings.Contains(out.String(), "index") {
		t.Fatalf("the check did not name the corrupt index: %d %s", checker.errors, out.String())
	}
	_, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{})
	if err == nil || strings.Contains(err.Error(), "revision unknown?") || !strings.Contains(err.Error(), "index") {
		t.Fatalf("scope selection mislabelled a corrupt index: %v", err)
	}
	if _, err := reportChangedFiles(root, "HEAD"); err == nil || !strings.Contains(err.Error(), "index") {
		t.Fatalf("report did not name the corrupt index: %v", err)
	}
}

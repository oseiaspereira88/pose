package cli

import (
	"bytes"
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
	if files := reportChangedFiles(root, "HEAD"); len(files) != 0 {
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
				reportChangedFiles(root, "HEAD")
				(&nativeChecker{root: root}).readyTransitionCandidates()
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
	if files := reportChangedFiles(root, ""); len(files) == 0 {
		t.Fatal("report listed nothing without an index")
	}
	if _, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{}); err != nil {
		t.Fatalf("scope selection failed without an index: %v", err)
	}
}

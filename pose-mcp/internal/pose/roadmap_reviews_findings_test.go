package pose

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestAPremiseOutlivesItsDelivery(t *testing.T) {
	probe, _ := premiseFixture(t, "doc:api/payments.proto")
	pin := digestPrefix(t, probe, "api/payments.proto")
	_, s := premiseFixture(t, "doc:api/payments.proto@"+pin)
	if err := os.WriteFile(filepath.Join(s.Root, "api/payments.proto"), []byte("syntax = \"proto3\";\nmessage Charge {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := premiseObligationsFor(t, s); len(got) != 1 {
		t.Fatalf("the fixture does not signal: %+v", got)
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, sp := range specs {
		raw, err := os.ReadFile(sp.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sp.Path, []byte(ReplaceFrontmatterValue(string(raw), "status", "done")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := premiseObligationsFor(t, s)
	if len(got) != 1 || got[0].Effects[0].Mode != EffectAdvisory {
		t.Fatalf("a premise stopped being watched at delivery: %+v", got)
	}
}

func TestAnEmptyGitDirectoryIsNoRepository(t *testing.T) {
	base := t.TempDir()
	below := filepath.Join(base, "fixture")
	if err := os.MkdirAll(below, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(base, ".git")
	if err := os.Mkdir(marker, 0o755); err != nil {
		t.Fatal(err)
	}
	if hasGitMetadata(below) || !GitNotARepository(below) {
		t.Fatalf("an empty directory read as a repository: metadata=%v", hasGitMetadata(below))
	}
	// Anything in it is a repository git may be failing to read, and stays unknown.
	if err := os.WriteFile(filepath.Join(marker, "HEAD"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasGitMetadata(below) || GitNotARepository(below) {
		t.Fatal("a broken repository read as none")
	}
	if err := os.RemoveAll(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasGitMetadata(below) || GitNotARepository(below) {
		t.Fatal("a gitfile read as no repository")
	}
}

func TestEveryRepositoryGitFindsIsARepository(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(base, "repo")
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	designDeltaGit(t, repo, "init", "-q")
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(sub, alias); err != nil {
		t.Skip(err)
	}
	bare := filepath.Join(base, "bare.git")
	if err := os.Mkdir(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	designDeltaGit(t, bare, "init", "-q", "--bare")
	for name, dir := range map[string]string{"link out of the lexical path": alias, "bare": bare} {
		if !hasGitMetadata(dir) || GitNotARepository(dir) {
			t.Errorf("%s: git finds a repository, the engine does not", name)
		}
	}
}

func TestAMarkerThatIsNoRepositoryEndsTheIgnoreWalk(t *testing.T) {
	for name, content := range map[string]bool{"empty": false, "broken": true} {
		t.Run(name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			designDeltaGit(t, root, "init", "-q")
			designDeltaGit(t, root, "config", "user.name", "Fixture")
			designDeltaGit(t, root, "config", "user.email", "fixture@example.invalid")
			designDeltaGit(t, root, "commit", "-q", "--allow-empty", "-m", "base")
			head := designDeltaGit(t, root, "rev-parse", "HEAD")
			designDeltaGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+head+",sm")
			if err := os.MkdirAll(filepath.Join(root, "sm", ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			if content {
				writeReviewFixture(t, root, "sm/.git/config", "broken")
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				collectGitIgnoredPaths(NativeFiles(), root, "", map[string]bool{})
			}()
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				t.Fatal("the walk went back into the same repository")
			}
		})
	}
}

func TestTheIgnoreWalkKnowsARepositoryByIdentity(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gitRoot := func(dir string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		designDeltaGit(t, dir, "init", "-q")
		designDeltaGit(t, dir, "config", "user.name", "Fixture")
		designDeltaGit(t, dir, "config", "user.email", "fixture@example.invalid")
		designDeltaGit(t, dir, "commit", "-q", "--allow-empty", "-m", "base")
	}
	// A space ends the name of the enclosing repository: it is part of the path.
	spaced := filepath.Join(base, "repo ")
	gitRoot(spaced)
	head := designDeltaGit(t, spaced, "rev-parse", "HEAD")
	designDeltaGit(t, spaced, "update-index", "--add", "--cacheinfo", "160000,"+head+",sm")
	if err := os.MkdirAll(filepath.Join(spaced, "sm", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !resolvesToEnclosingRepository(filepath.Join(spaced, "sm")) {
		t.Fatal("a marker below a spaced root read as its own repository")
	}
	// An initialized submodule keeps its rules, however the root is spelled.
	parent := filepath.Join(base, "parent")
	gitRoot(parent)
	gitRoot(filepath.Join(parent, "sm"))
	writeReviewFixture(t, parent, "sm/.gitignore", "ignored/\n")
	writeReviewFixture(t, parent, "sm/ignored/out", "x")
	subHead := designDeltaGit(t, filepath.Join(parent, "sm"), "rev-parse", "HEAD")
	designDeltaGit(t, parent, "update-index", "--add", "--cacheinfo", "160000,"+subHead+",sm")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(wd, parent)
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(parent, link); err != nil {
		t.Skip(err)
	}
	for name, root := range map[string]string{"absolute": parent, "relative": relative, "through a link": link} {
		ignored := map[string]bool{}
		collectGitIgnoredPaths(NativeFiles(), root, "", ignored)
		if !ignored["sm/ignored"] {
			t.Errorf("%s: the rules of the submodule were lost: %v", name, ignored)
		}
	}
}

func TestTheIgnoreWalkEntersARepositoryOnce(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	writeReviewFixture(t, root, ".gitignore", "ignored/\n")
	writeReviewFixture(t, root, "ignored/out", "x")
	ignored := map[string]bool{}
	collectGitIgnoredPaths(NativeFiles(), root, "", ignored)
	if !ignored["ignored"] {
		t.Fatalf("the fixture ignores nothing: %v", ignored)
	}
	again := map[string]bool{}
	walkGitIgnoredPaths(NativeFiles(), root, "", again, map[string]bool{physicalIdentity(root): true})
	if len(again) != 0 {
		t.Fatalf("a visited repository was walked again: %v", again)
	}
}

func TestACompleteFallbackReportsItsDurability(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	linkFile = func(string, string) error { return &os.LinkError{Op: "link", Err: nativeNoLinks()} }
	t.Cleanup(func() { linkFile = os.Link })
	PublishSyncHook = func(kind, _ string) error {
		if kind == "exclusive" {
			return syscall.EIO
		}
		return nil
	}
	t.Cleanup(func() { PublishSyncHook = nil })
	err := PublishFile(path, []byte("sealed\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	if !errors.Is(err, ErrPublishUnsynced) {
		t.Fatalf("a complete record read as not published: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "sealed\n" {
		t.Fatalf("got %q", got)
	}
	if tmp := leftovers(t, dir); len(tmp) != 0 {
		t.Fatalf("staging left: %v", tmp)
	}
}

func TestABriefAcceptedWholeIsDelivered(t *testing.T) {
	cut := errors.New("broken pipe")
	for name, c := range map[string]struct{ copied, closed, want error }{
		"closed by us":             {nil, nil, nil},
		"closed by Wait first":     {nil, os.ErrClosed, nil},
		"close failed":             {nil, cut, cut},
		"write failed":             {cut, nil, cut},
		"write failed, then close": {cut, os.ErrClosed, cut},
	} {
		if got := briefDeliveryError(c.copied, c.closed); !errors.Is(got, c.want) || (c.want == nil) != (got == nil) {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
	for i := 0; i < 200; i++ {
		// A child that does not read races the writer: the pipe took the bytes, or it
		// was gone first. Either answer can be true, and it has to match the bytes.
		_, count, complete, _, err := runProvenanceCommand(exec.Command("true"), "brief")
		if accepted := count == 5; accepted != complete || accepted != (err == nil) {
			t.Fatalf("run %d: count=%d complete=%v err=%v", i, count, complete, err)
		}
	}
}

func nativeNoLinks() error {
	if runtime.GOOS == "windows" {
		return syscall.Errno(50)
	}
	return syscall.ENOTSUP
}

func TestOnlyAMissingLinkFallsBack(t *testing.T) {
	cases := map[string]struct {
		err      error
		fallback bool
	}{
		"no links here": {nativeNoLinks(), true},
		"no space":      {syscall.ENOSPC, false},
		"no permission": {syscall.EACCES, false},
		"generic":       {errors.New("failed"), false},
	}
	if runtime.GOOS != "windows" {
		cases["a filesystem or policy refusing links"] = struct {
			err      error
			fallback bool
		}{syscall.EPERM, true}
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "record.json")
			linkFile = func(string, string) error { return &os.LinkError{Op: "link", Err: c.err} }
			t.Cleanup(func() { linkFile = os.Link })
			err := PublishFile(path, []byte("sealed\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
			if _, statErr := os.Stat(path); c.fallback != (err == nil && statErr == nil) {
				t.Fatalf("fallback=%v: err=%v stat=%v", c.fallback, err, statErr)
			}
		})
	}
}

func TestTheFallbackAnswersForItsDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	linkFile = func(string, string) error { return &os.LinkError{Op: "link", Err: nativeNoLinks()} }
	t.Cleanup(func() { linkFile = os.Link })
	PublishSyncHook = func(kind, _ string) error {
		if kind == "dir" {
			return errors.New("injected")
		}
		return nil
	}
	t.Cleanup(func() { PublishSyncHook = nil })
	err := PublishFile(path, []byte("sealed\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	if !errors.Is(err, ErrPublishUnsynced) {
		t.Fatalf("a fallback skipped the sync of its directory: %v", err)
	}
	if tmp := leftovers(t, dir); len(tmp) != 0 {
		t.Fatalf("staging left: %v", tmp)
	}
}

func TestAWithdrawnPremiseAsksForNothing(t *testing.T) {
	probe, _ := premiseFixture(t, "doc:api/payments.proto")
	pin := digestPrefix(t, probe, "api/payments.proto")
	_, s := premiseFixture(t, "doc:api/payments.proto@"+pin)
	if err := os.WriteFile(filepath.Join(s.Root, "api/payments.proto"), []byte("syntax = \"proto3\";\nmessage Charge {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		t.Fatal(err)
	}
	edit := func(from, to string) {
		t.Helper()
		for _, sp := range specs {
			raw, err := os.ReadFile(sp.Path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), from) {
				t.Fatalf("the fixture has no %q", from)
			}
			if err := os.WriteFile(sp.Path, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	edit("### Decision D1\n", "### Decision D1\n- Status: withdrawn\n")
	got := premiseObligationsFor(t, s)
	if len(got) != 1 || len(got[0].Targets) != 1 {
		t.Fatalf("a withdrawn decision is still asked about: %+v", got)
	}
	edit("- Status: verified\n", "- Status: withdrawn\n")
	if got := premiseObligationsFor(t, s); len(got) != 0 {
		t.Fatalf("a withdrawn premise is still asked about: %+v", got)
	}
}

func TestTheCleanupPlanHoldsBothCorpora(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "reports", "2026-11-02-pose-v7-legacy-cleanup-both-corpora.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"| Class | pose-dist", "Harne8", "before and after the run"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the report does not hold %q", want)
		}
	}
}

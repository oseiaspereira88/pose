package cli

import (
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestB07R3DispatchAncestorLink(t *testing.T) {
	root := dispatchInstance(t)
	outside := filepath.Join(t.TempDir(), "moved")
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; mv pose-mcp '" + outside + "'; ln -s '" + outside + "' pose-mcp; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("dispatch completed after ancestor escaped worktree")
	}
}
func TestB07R3SourceModeUnknown(t *testing.T) {
	root := dispatchInstance(t)
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	body := "#!/bin/sh\ncase \"$*\" in *\"-C " + root + " ls-tree\"*) exit 97;; esac\nexec '" + real + "' \"$@\"\n"
	os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; git -C '" + root + "' config core.filemode false; chmod +x '" + root + "/pose-mcp/lib.go'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("unreadable source mode accepted")
	}
}
func TestB07R3SourceType(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; git -C '" + root + "' update-index --assume-unchanged pose-mcp/lib.go; cp '" + root + "/pose-mcp/lib.go' '" + root + "/copy.txt'; printf '\\ncopy.txt\\n' >> '" + root + "/.git/info/exclude'; rm '" + root + "/pose-mcp/lib.go'; ln -s ../copy.txt '" + root + "/pose-mcp/lib.go'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("source regular file replaced by symlink accepted")
	}
}
func TestB07R3DispatchIgnoredAncestorLink(t *testing.T) {
	root := dispatchInstance(t)
	outside := filepath.Join(t.TempDir(), "moved")
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; mv pose-mcp '" + outside + "'; ln -s '" + outside + "' pose-mcp; printf '\\npose-mcp\\n' >> .git-ignore-unused; rm .git-ignore-unused; git rev-parse --git-path info/exclude | xargs sh -c 'printf \"\\npose-mcp\\n\" >> \"$0\"'; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	t.Logf("code=%d output=%s", code, out)
	if strings.Contains(out, "delegate.status=completed") {
		t.Error("ignored ancestor link escapes worktree")
	}
}
func TestB07R3PersistedRunIdentity(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"probe": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "test", Model: "probe", Timeout: "30s"}})
	code, out := dispatchOnce(t, root, "probe")
	if code != 0 {
		t.Fatalf("control: %s", out)
	}
	s := posemodel.Store{Root: root}
	runs, e := s.ListReviewRuns("")
	if e != nil || len(runs) != 1 {
		t.Fatalf("%v %v", runs, e)
	}
	r := runs[0]
	if why := s.ReviewRunStillGoverns(r); why != "" {
		t.Fatalf("control: %s", why)
	}
	p := filepath.Join(root, r.Path)
	raw, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	raw = []byte(strings.Replace(string(raw), r.BundleDigest, "sha256:wrong", 1))
	os.WriteFile(p, raw, 0644)
	// A record edited after it was written is not read as a run: it is lost, and a copy of the run that
	// it used to be does not govern while the loss stands.
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(runs) != 0 || len(lost) < 1 || !strings.Contains(strings.Join(lost, ";"), "identity") {
		t.Fatalf("an edited record was read as a run: runs=%v lost=%v %v", runs, lost, e)
	}
	if why := s.ReviewRunStillGoverns(r); why == "" {
		t.Error("a run whose record was edited governs")
	}
}

package pose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Spec pose-governance-audit-skips-pure-git-reads.

const pureTestID = "0123456789abcdef0123456789abcdef01234567"

// R1: only the exact forms that cannot depend on mutable state are pure.
func TestOnlyReadsThatCannotDependOnMutableStateAreClassifiedPure(t *testing.T) {
	type want struct{ object, head bool }
	for _, tc := range []struct {
		args []string
		want want
	}{
		{[]string{"show", pureTestID + ":.pose/a.md"}, want{true, false}},
		{[]string{"show", "HEAD:.pose/a.md"}, want{}},
		{[]string{"show", "main:.pose/a.md"}, want{}},
		{[]string{"show", pureTestID}, want{}},
		{[]string{"cat-file", "blob", pureTestID}, want{true, false}},
		{[]string{"cat-file", "blob", "HEAD"}, want{}},
		{[]string{"cat-file", "-e", pureTestID}, want{}},
		{[]string{"log", "-1", "--format=%H", "--", "a.md"}, want{false, true}},
		{[]string{"log", "-1", "--format=%H", "--"}, want{}},
		{[]string{"log", "-1", "--format=%d", "--", "a.md"}, want{}},
		{[]string{"log", "-1", "--all", "--format=%H", "--", "a.md"}, want{}},
		{[]string{"log", "--all", "-1", "--format=%H", "--", "a.md"}, want{}},
		{[]string{"ls-tree", "-z", pureTestID, "--", "a"}, want{true, false}},
		{[]string{"ls-tree", pureTestID}, want{true, false}},
		{[]string{"ls-tree", "-z", "HEAD", "--", "a"}, want{}},
		{[]string{"ls-tree", "-z", "refs/heads/main"}, want{}},
		{[]string{"ls-tree", "--abbrev", pureTestID}, want{}},
		{[]string{"ls-tree", "-z", pureTestID, "extra"}, want{}},
		{[]string{"rev-parse", "--verify", "HEAD"}, want{}},
		{[]string{"merge-base", "--is-ancestor", pureTestID, pureTestID}, want{}},
		{nil, want{}},
	} {
		object, head := governancePureGitRead(tc.args)
		if object != tc.want.object || head != tc.want.head {
			t.Errorf("%v classified (%v, %v), want (%v, %v)", tc.args, object, head, tc.want.object, tc.want.head)
		}
	}
}

// R2/R3: the final audit does not rerun a pure read while its commit stands, reruns every
// other command, and reruns a head-relative read, and finds the change, when HEAD moved.
func TestTheFinalAuditSkipsPureReadsWhileTheirCommitStandsAndOnlyThen(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	reruns := map[string]*atomic.Int64{}
	original := governanceAuditGit
	governanceAuditGit = func(ctx context.Context, files Files, r string, args []string, limit int64) ([]byte, []byte, error) {
		key := strings.Join(args, " ")
		if reruns[key] == nil {
			reruns[key] = &atomic.Int64{}
		}
		reruns[key].Add(1)
		return original(ctx, files, r, args, limit)
	}
	defer func() { governanceAuditGit = original }()

	capture := func() (*governanceReadView, Store, string) {
		base, view := (Store{Root: root}).beginGovernanceReadView()
		store := base.withGovernanceReadView(view, "pure-test")
		head, err := captureGovernanceGit(store.fsys(), root, "pure-test", "rev-parse", "--verify", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		id := strings.TrimSpace(string(head))
		for _, args := range [][]string{
			{"log", "-1", "--format=%H", "--", "a.md"},
			{"show", id + ":a.md"},
			{"ls-tree", "-z", "HEAD", "--", "a.md"},
		} {
			if _, err := captureGovernanceGit(store.fsys(), root, "pure-test", args...); err != nil {
				t.Fatalf("%v: %v", args, err)
			}
		}
		return view, store, id
	}
	count := func(prefix string) int64 {
		var n int64
		for key, c := range reruns {
			if strings.HasPrefix(key, prefix) {
				n += c.Load()
			}
		}
		return n
	}

	view, _, _ := capture()
	if audit := view.audit(); len(audit.ChangedPaths) != 0 || audit.ChangedProducers["pure-test"] {
		t.Fatalf("nothing changed, the audit reports it did: %+v", audit)
	}
	if got := count("log -1"); got != 0 {
		t.Fatalf("the head-relative log was rerun %d times with HEAD where it was", got)
	}
	if got := count("show "); got != 0 {
		t.Fatalf("the object-id show was rerun %d times", got)
	}
	if count("rev-parse") == 0 || count("ls-tree") == 0 {
		t.Fatalf("a command that is not pure was not rerun: %v", reruns)
	}

	// HEAD moves, a.md changes: the log, rerun, shows a different last commit.
	view, _, _ = capture()
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("one changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshotTestGit(t, root, "add", "a.md")
	snapshotTestGit(t, root, "commit", "-q", "-m", "change a")
	before := count("log -1")
	audit := view.audit()
	if count("log -1") != before+1 {
		t.Fatalf("the head-relative log was not rerun after HEAD moved")
	}
	if !audit.ChangedProducers["pure-test"] || len(audit.ChangedPaths) == 0 {
		t.Fatalf("the change after HEAD moved was not found: %+v", audit)
	}
}

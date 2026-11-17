package pose

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// auditAfterDeadline captures one read under a 2 s deadline through a real
// read view, lets the deadline pass, optionally changes the repository, and
// returns the final audit.
func auditAfterDeadline(t *testing.T, read func(Files, context.Context, string) error, change func(root string)) governanceReadViewAudit {
	t.Helper()
	root, _ := snapshotBudgetRepo(t)
	_, view := Store{Root: root}.beginGovernanceReadView()
	if view == nil {
		t.Fatal("no read view")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := read(view.filesFor("audit-timeout"), ctx, root); err != nil {
		t.Fatal(err)
	}
	cancel()
	time.Sleep(2500 * time.Millisecond)
	if change != nil {
		change(root)
	}
	return view.audit()
}

// ls-tree of HEAD is not a pure read: the audit repeats it.
func captureHeadListing(files Files, ctx context.Context, root string) error {
	_, err := CaptureGovernanceGitContextIn(files, ctx, root, "audit-timeout", 1<<20, "ls-tree", "-r", "-z", "HEAD")
	return err
}

func streamHeadShow(files Files, ctx context.Context, root string) error {
	return StreamGovernanceRawHistoryGitIn(files, ctx, root, "audit-timeout", 1<<20, io.Discard, "show", "--format=", "HEAD", "--")
}

// Spec pose-read-view-audit-repeats-a-read-under-its-own-time, R1 and R2: a
// captured command that the audit repeats, unchanged, is not reported after
// its own deadline.
func TestReadViewAuditRepeatsACommandUnderItsOwnTime(t *testing.T) {
	audit := auditAfterDeadline(t, captureHeadListing, nil)
	if len(audit.ChangedPaths) != 0 || len(audit.UnknownProducers) != 0 {
		t.Fatalf("an unchanged command audited after its deadline: changed=%v unknown=%v", audit.ChangedPaths, audit.UnknownProducers)
	}
}

// R1 and R2 for a streamed read.
func TestReadViewAuditRepeatsAStreamUnderItsOwnTime(t *testing.T) {
	audit := auditAfterDeadline(t, streamHeadShow, nil)
	if len(audit.ChangedPaths) != 0 || len(audit.UnknownProducers) != 0 {
		t.Fatalf("an unchanged stream audited after its deadline: changed=%v unknown=%v", audit.ChangedPaths, audit.UnknownProducers)
	}
}

// The command is really repeated: a HEAD that moved after the deadline is
// reported as a change.
func TestReadViewAuditStillSeesAChangeAfterTheDeadline(t *testing.T) {
	audit := auditAfterDeadline(t, captureHeadListing, func(root string) {
		if err := os.WriteFile(filepath.Join(root, "data-new.txt"), []byte("new\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		federatedTestGit(t, root, "add", "-A")
		federatedTestGit(t, root, "commit", "-q", "-m", "moved")
	})
	if len(audit.ChangedPaths) == 0 {
		t.Fatal("a HEAD that moved after the deadline was not reported")
	}
}

// R3: a repeated read still runs under a deadline, the time it had.
func TestAuditTimeoutIsTheTimeTheReadHad(t *testing.T) {
	start := time.Now()
	if got := auditTimeout(start.Add(90*time.Second), start); got != 90*time.Second {
		t.Fatalf("got %v, want 90s", got)
	}
	if got := auditTimeout(start.Add(-time.Second), start); got <= 0 {
		t.Fatalf("an expired deadline gave no time: %v", got)
	}
}

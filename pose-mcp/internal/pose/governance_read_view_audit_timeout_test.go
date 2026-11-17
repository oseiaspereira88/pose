package pose

import (
	"context"
	"io"
	"testing"
	"time"
)

// Spec pose-read-view-audit-repeats-a-read-under-its-own-time: a read
// captured under a short deadline is audited under the same length of time,
// counted from the audit, so an operation that outlives the deadline does
// not report its unchanged reads as changed.
func TestReadViewAuditRepeatsAReadUnderItsOwnTime(t *testing.T) {
	root, head := snapshotBudgetRepo(t)
	_, view := Store{Root: root}.beginGovernanceReadView()
	if view == nil {
		t.Fatal("no read view")
	}
	files := view.filesFor("audit-timeout")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if _, err := CaptureGovernanceGitContextIn(files, ctx, root, "audit-timeout", 1<<20, "ls-tree", "-r", "-z", head); err != nil {
		t.Fatal(err)
	}
	if err := StreamGovernanceRawHistoryGitIn(files, ctx, root, "audit-timeout", 1<<20, io.Discard, "show", "--format=", head, "--"); err != nil {
		t.Fatal(err)
	}
	cancel()
	time.Sleep(2500 * time.Millisecond)
	audit := view.audit()
	if len(audit.ChangedPaths) != 0 || len(audit.UnknownProducers) != 0 {
		t.Fatalf("unchanged reads audited after their deadline: changed=%v unknown=%v", audit.ChangedPaths, audit.UnknownProducers)
	}
}

// A repeated read still runs under a deadline: the time it had.
func TestAuditTimeoutIsTheTimeTheReadHad(t *testing.T) {
	start := time.Now()
	if got := auditTimeout(start.Add(90*time.Second), start); got != 90*time.Second {
		t.Fatalf("got %v, want 90s", got)
	}
	if got := auditTimeout(start.Add(-time.Second), start); got <= 0 {
		t.Fatalf("an expired deadline gave no time: %v", got)
	}
}

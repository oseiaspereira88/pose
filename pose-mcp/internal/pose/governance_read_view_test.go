package pose

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGovernanceReadViewDirtyToDirtySnapshot(t *testing.T) {
	governanceReadViewProbeUseKnownEngineVersion(t)
	store := obligationFixture(t)
	path := filepath.Join(store.Root, ".pose/specs/2026-10-01-base.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	firstBytes := bytes.Replace(original, []byte("R1: x"), []byte("R1: z"), 1)
	if bytes.Equal(firstBytes, original) {
		t.Fatal("fixture spec did not contain the expected requirement")
	}
	if err := os.WriteFile(path, firstBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	dirtyInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	first, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Snapshot.WorktreeDirty {
		t.Fatalf("first operation did not observe a dirty worktree: %+v", first.Snapshot)
	}
	if changed := store.ObligationSnapshotChanges(first.Snapshot); len(changed) != 0 {
		t.Fatalf("stable inputs changed after the first operation: %v", changed)
	}
	if first.Snapshot.ReadView == nil || first.Snapshot.ReadView.Freshness != ReadViewFreshnessCurrent {
		t.Fatalf("stable projection did not retain a current read-view proof: %+v", first.Snapshot.ReadView)
	}

	secondBytes := bytes.Replace(firstBytes, []byte("R1: z"), []byte("R1: y"), 1)
	if bytes.Equal(secondBytes, firstBytes) || len(secondBytes) != len(firstBytes) {
		t.Fatal("second spec edit must change bytes without changing their length")
	}
	if err := os.WriteFile(path, secondBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, dirtyInfo.ModTime(), dirtyInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if !info.ModTime().Equal(dirtyInfo.ModTime()) {
		t.Fatalf("mtime was not restored: got %v, want %v", info.ModTime(), dirtyInfo.ModTime())
	}

	second, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Snapshot.WorktreeDirty {
		t.Fatalf("second operation did not observe a dirty worktree: %+v", second.Snapshot)
	}
	if first.Snapshot.Digest == second.Snapshot.Digest {
		t.Fatalf("same-size spec edit kept the operation identity: %s", first.Snapshot.Digest)
	}
	if first.Snapshot.ReadView == nil || second.Snapshot.ReadView == nil || first.Snapshot.ReadView.ID == second.Snapshot.ReadView.ID {
		t.Fatalf("same-size spec edit kept the read-view identity: first=%+v second=%+v", first.Snapshot.ReadView, second.Snapshot.ReadView)
	}
	if changed := store.ObligationSnapshotChanges(first.Snapshot); len(changed) == 0 {
		t.Fatal("same-size spec edit with restored mtime was not reported as a relevant snapshot change")
	}
}

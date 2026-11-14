package pose

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Spec pose-read-view-streamed-object-digests.

func streamTestRepo(t *testing.T) (string, string) {
	t.Helper()
	root := snapshotRepo(t, map[string]string{".pose/project.json": `{"schema_version":1,"project_id":"proj.001","name":"stream fixture"}`, "a.md": "one\n"})
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshotTestGit(t, root, "commit", "-q", "-am", "second")
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return root, strings.TrimSpace(string(out))
}

func streamArgs(id string) []string {
	return []string{"show", "--binary", "--no-ext-diff", "--format=", id, "--"}
}

// R1: the sink gets the exact bytes, the view keeps only digest and length,
// and the retained captured-input budget is untouched.
func TestAStreamedObjectReadRetainsOnlyItsDigest(t *testing.T) {
	root, id := streamTestRepo(t)
	want, err := exec.Command("git", append([]string{"-C", root}, streamArgs(id)...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "stream-test")
	var sink bytes.Buffer
	if err := StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-test", 1<<20, &sink, streamArgs(id)...); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sink.Bytes(), want) {
		t.Fatalf("the sink did not get the exact output:\n%q\nwant\n%q", sink.Bytes(), want)
	}
	state := view.operation.git
	for key := range state.observations {
		if strings.Contains(key, `"show"`) {
			t.Fatalf("the streamed read was also retained as a captured observation: %q", key)
		}
	}
	if len(state.streams) != 1 {
		t.Fatalf("streams=%d", len(state.streams))
	}
	for _, item := range state.streams {
		if item.length != int64(len(want)) || item.digest != digestBytes(want) {
			t.Fatalf("streamed observation %+v", item)
		}
	}
	view.operation.budgetMu.Lock()
	input, streamed := view.operation.inputBytes, view.operation.streamedBytes
	view.operation.budgetMu.Unlock()
	if streamed != int64(len(want)) || input >= int64(len(want)) {
		t.Fatalf("streamed=%d input=%d for %d bytes", streamed, input, len(want))
	}
	if audit := view.audit(); audit.ChangedProducers["stream-test"] || len(audit.UnknownProducers["stream-test"]) != 0 {
		t.Fatalf("an unchanged streamed read did not pass the audit: %+v", audit)
	}
}

// R2: the streamed budget is exact at its boundary and an overflow is
// unknown, never truncated.
func TestTheStreamedBudgetIsExactAndOverflowIsUnknown(t *testing.T) {
	root, id := streamTestRepo(t)
	want, _ := exec.Command("git", append([]string{"-C", root}, streamArgs(id)...)...).Output()
	for _, overflow := range []bool{false, true} {
		base, view := (Store{Root: root}).beginGovernanceReadView()
		store := base.withGovernanceReadView(view, "stream-budget")
		prior := int64(governanceReadViewMaxStreamedBytes - len(want))
		if overflow {
			prior++
		}
		view.operation.budgetMu.Lock()
		view.operation.streamedBytes = prior
		view.operation.budgetMu.Unlock()
		var sink bytes.Buffer
		err := StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-budget", 1<<20, &sink, streamArgs(id)...)
		if overflow != errors.Is(err, errGovernanceReadViewBudget) {
			t.Fatalf("overflow=%v err=%v", overflow, err)
		}
		metadata, _ := view.metadata(store.FederatedProjectID, "", "", view.audit())
		if overflow != strings.Contains(strings.Join(metadata.Limitations, "\n"), "streamed byte budget") {
			t.Fatalf("overflow=%v limitations=%v", overflow, metadata.Limitations)
		}
	}
	if err := StreamGovernanceRawHistoryGitIn(nil, context.Background(), root, "p", governanceReadViewMaxStreamedRead+1, io.Discard, streamArgs(id)...); err == nil {
		t.Fatal("a per-read limit above the bound was accepted")
	}
}

// R4: the same read again in the operation is streamed again to its new
// consumer, and a different result leaves the producer unknown.
func TestASecondStreamedReadIsReadAgainAndCompared(t *testing.T) {
	root, id := streamTestRepo(t)
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "stream-again")
	var first, second bytes.Buffer
	if err := StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-again", 1<<20, &first, streamArgs(id)...); err != nil {
		t.Fatal(err)
	}
	if err := StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-again", 1<<20, &second, streamArgs(id)...); err != nil || !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("the second consumer did not get the bytes: %v", err)
	}
	original := governanceStreamRun
	governanceStreamRun = func(ctx context.Context, files Files, r string, args []string, limit int64, admit func(int64) bool, sink io.Writer) (string, int64, []byte, error) {
		return original(ctx, files, r, args, limit, admit, &prefixed{sink: sink})
	}
	defer func() { governanceStreamRun = original }()
	changed := &bytes.Buffer{}
	tampered := func(ctx context.Context, files Files, r string, args []string, limit int64, admit func(int64) bool, sink io.Writer) (string, int64, []byte, error) {
		_, _ = sink.Write([]byte("other"))
		return digestBytes([]byte("other")), 5, nil, nil
	}
	governanceStreamRun = tampered
	if err := StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-again", 1<<20, changed, streamArgs(id)...); err == nil {
		t.Fatal("a streamed read that differed within the operation was accepted")
	}
	metadata, _ := view.metadata(store.FederatedProjectID, "", "", view.audit())
	if !strings.Contains(strings.Join(metadata.Limitations, "\n"), "differed within the operation") {
		t.Fatalf("limitations=%v", metadata.Limitations)
	}
}

type prefixed struct{ sink io.Writer }

func (p *prefixed) Write(data []byte) (int, error) { return p.sink.Write(data) }

// The final audit streams every read again (review round 1, F1 and F3): an
// unchanged read passes, and a change of rendering that leaves the commit
// and the configuration alone, an attribute, is found.
func TestTheStreamedAuditStreamsEveryReadAgain(t *testing.T) {
	root, id := streamTestRepo(t)
	var reruns atomic.Int64
	original := governanceAuditGitStream
	governanceAuditGitStream = func(ctx context.Context, files Files, r string, args []string, limit int64, admit func(int64) bool, sink io.Writer) (string, int64, []byte, error) {
		reruns.Add(1)
		return original(ctx, files, r, args, limit, admit, sink)
	}
	defer func() { governanceAuditGitStream = original }()
	read := func() *governanceReadView {
		base, view := (Store{Root: root}).beginGovernanceReadView()
		store := base.withGovernanceReadView(view, "stream-audit")
		if err := StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-audit", 1<<20, io.Discard, streamArgs(id)...); err != nil {
			t.Fatal(err)
		}
		return view
	}
	if audit := read().audit(); reruns.Load() != 1 || audit.ChangedProducers["stream-audit"] {
		t.Fatalf("an unchanged read was not streamed again or did not pass: reruns=%d %+v", reruns.Load(), audit)
	}
	view := read()
	if err := os.WriteFile(filepath.Join(root, ".git", "info", "attributes"), []byte("a.md -diff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if audit := view.audit(); reruns.Load() != 2 || !audit.ChangedProducers["stream-audit"] {
		t.Fatalf("an attribute that changed the rendered patch was not found: reruns=%d %+v", reruns.Load(), audit)
	}
}

// Two first reads of the same command that run concurrently and differ
// leave the producer unknown (review round 1, F2).
func TestConcurrentFirstStreamedReadsAreCompared(t *testing.T) {
	root, id := streamTestRepo(t)
	base, view := (Store{Root: root}).beginGovernanceReadView()
	store := base.withGovernanceReadView(view, "stream-race")
	var calls atomic.Int64
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	original := governanceStreamRun
	governanceStreamRun = func(ctx context.Context, files Files, r string, args []string, limit int64, admit func(int64) bool, sink io.Writer) (string, int64, []byte, error) {
		n := calls.Add(1)
		started <- struct{}{}
		<-release
		body := []byte("read " + string(rune('0'+n)))
		_, _ = sink.Write(body)
		return digestBytes(body), int64(len(body)), nil, nil
	}
	defer func() { governanceStreamRun = original }()
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			errs <- StreamGovernanceRawHistoryGitIn(store.fsys(), context.Background(), root, "stream-race", 1<<20, io.Discard, streamArgs(id)...)
		}()
	}
	<-started
	<-started
	close(release)
	failed := 0
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			failed++
		}
	}
	metadata, _ := view.metadata(store.FederatedProjectID, "", "", view.audit())
	if failed != 1 || !strings.Contains(strings.Join(metadata.Limitations, "\n"), "differed within the operation") {
		t.Fatalf("concurrent differing first reads both succeeded: failed=%d limitations=%v", failed, metadata.Limitations)
	}
}

package pose

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// interleavedReviewPolicyFiles serves the accepted policy to each operation
// attempt, then revokes it exactly when the final audit reopens the file. The
// retry starts from the accepted policy again, so both attempts must be
// rejected by their own final audits.
type interleavedReviewPolicyFiles struct {
	Files
	path       string
	accepted   []byte
	revoked    []byte
	opens      int
	auditOpens int
}

func (f *interleavedReviewPolicyFiles) Open(path string) (fs.File, error) {
	if filepath.Clean(path) == f.path {
		f.opens++
		served := f.accepted
		if f.opens%2 == 0 {
			f.auditOpens++
			served = f.revoked
		}
		file, err := f.Files.Open(path)
		if err != nil {
			return nil, err
		}
		return &interleavedReviewPolicyFile{File: file, reader: bytes.NewReader(served)}, nil
	}
	return f.Files.Open(path)
}

// interleavedReviewPolicyFile keeps the underlying file identity stable while
// serving different bytes to the final audit. This places the change before
// the audit's preflight Stat, so the intended changed-path audit runs instead
// of a metadata mismatch short-circuiting the operation.
type interleavedReviewPolicyFile struct {
	fs.File
	reader *bytes.Reader
}

func (f *interleavedReviewPolicyFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func interleavedReviewPolicyStore(t *testing.T, store Store) (*interleavedReviewPolicyFiles, Store) {
	t.Helper()
	path := filepath.Join(store.Root, ".pose", "policy", "review.json")
	accepted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	revokedText := strings.Replace(string(accepted), `"enabled": true`, `"enabled": null`, 1)
	if revokedText == string(accepted) {
		t.Fatal("fixture policy has no enabled=true field to revoke")
	}
	files := &interleavedReviewPolicyFiles{
		Files:    NativeFiles(),
		path:     filepath.Clean(path),
		accepted: accepted,
		revoked:  []byte(revokedText),
	}
	return files, Store{Root: store.Root, FS: files}
}

func TestRetainedCompletedReviewBlocksWhenBothReadViewAttemptsChangePolicy(t *testing.T) {
	store, _, _ := retentionFixture(t)
	closeBackendSpec(t, store)
	stable, err := store.GetCloseoutState("spec:backend")
	if err != nil || !stable.Terminal || !stable.Review.Approved || !stable.Review.Fresh {
		t.Fatalf("fixture did not retain its completed approval before interleaving: state=%+v err=%v", stable, err)
	}

	files, changing := interleavedReviewPolicyStore(t, store)
	state, err := changing.GetCloseoutState("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if files.opens < 4 || files.auditOpens < 2 {
		t.Fatalf("fixture did not interleave both attempts: policy opens=%d audit opens=%d", files.opens, files.auditOpens)
	}
	if state.Terminal || state.Review.Approved || state.Review.Fresh || state.ReadView == nil || state.ReadView.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("a retained approval survived policy changes in both read-view attempts: %+v", state)
	}
	if !strings.Contains(strings.Join(state.Blockers, " "), "governance read view is unknown") ||
		!strings.Contains(strings.Join(state.Limitations, " "), "observed input changed before final validation") {
		t.Fatalf("closeout did not disclose the unstable policy input: blockers=%+v limitations=%+v", state.Blockers, state.Limitations)
	}
}

func TestRetainedReviewCheckBlocksWhenBothReadViewAttemptsChangePolicy(t *testing.T) {
	store, _, _ := retentionFixture(t)
	closeBackendSpec(t, store)
	files, changing := interleavedReviewPolicyStore(t, store)
	eval, err := changing.ReviewCheck("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if files.opens < 4 || files.auditOpens < 2 {
		t.Fatalf("fixture did not interleave both attempts: policy opens=%d audit opens=%d", files.opens, files.auditOpens)
	}
	if eval.Approved || eval.Fresh || eval.ReadView == nil || eval.ReadView.Freshness != ReadViewFreshnessUnknown {
		t.Fatalf("a retained approval survived policy changes in both read-view attempts: %+v", eval)
	}
	if !strings.Contains(strings.Join(eval.Blockers, " "), "governance read view is unknown") {
		t.Fatalf("review check did not disclose the unstable policy input: blockers=%+v", eval.Blockers)
	}
}

func TestLegacyDoneCloseoutRemainsReadOnlyTerminalWhenHeadIsUnavailable(t *testing.T) {
	root, store := reviewFixture(t)
	writeReviewFixture(t, root, ".pose/specs/legacy/spec.md", "---\nslug: legacy\nstatus: done\ncreated_at: 2026-08-01\ncompleted_at: 2026-08-01\n---\n\n# Spec: legacy\n\n## 2. Requirements\n- R1: shipped before review adoption\n")
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}

	state, err := store.GetCloseoutState("spec:legacy")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Terminal || state.NextAction != "none" || state.Review.Required || state.Review.Approved || state.Review.Fresh {
		t.Fatalf("missing HEAD changed the historical read-only terminal result: %+v", state)
	}
	if state.ReadView == nil || state.ReadView.Freshness != ReadViewFreshnessUnknown ||
		!strings.Contains(strings.Join(state.Limitations, " "), "governance read view is unknown") {
		t.Fatalf("missing HEAD was not disclosed as an unknown read view: %+v", state)
	}
}

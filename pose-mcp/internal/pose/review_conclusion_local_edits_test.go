package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// editSealedSubject changes a file the bundle sealed, in the working tree
// only, and returns how to restore it.
func editSealedSubject(t *testing.T, f runProvenanceFixture) func() {
	t.Helper()
	for _, entry := range f.bundle.Payload.Subject.Entries {
		if entry.Class == "submodule" || entry.Action == "removed" {
			continue
		}
		path := entry.Path
		if entry.NewPath != "" {
			path = entry.NewPath
		}
		full := filepath.Join(f.store.Root, filepath.FromSlash(path))
		before, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		if err := os.WriteFile(full, append(append([]byte{}, before...), []byte("\n// a fix for the next round\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		return func() {
			if err := os.WriteFile(full, before, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatal("the bundle seals no editable file")
	return nil
}

// Spec pose-review-conclude-ignores-unrelated-local-edits R1, R3: a run that
// requests changes is concluded while the implementer already edits the
// sealed subject in the working tree.
func TestARequestForChangesConcludesOverLocalEdits(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.Decision = "changes-requested"
	findings := []ReviewDraftFinding{{ID: "B1", Severity: "high", Disposition: "blocker", File: "x.go", Line: 3, Summary: "a defect"}}
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, findings)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "changes requested over local edits")
	defer editSealedSubject(t, f)()
	conclusion, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || conclusion.State != ReviewConclusionRecorded || conclusion.Decision != "changes-requested" {
		t.Fatalf("a request for changes was refused over local edits: %+v %v", conclusion, err)
	}
}

// R2, R3: an approval refused only because the working tree no longer holds
// its bundle's subject leaves no record, and the same run concludes once the
// tree matches again.
func TestAnApprovalOverLocalEditsIsNotRecordedAndConcludesLater(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "approval over local edits")
	restore := editSealedSubject(t, f)
	refused, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || refused.State != ReviewConclusionRefused || !strings.Contains(strings.Join(refused.Reasons, ";"), "not recorded") {
		t.Fatalf("an approval over local edits: %+v %v", refused, err)
	}
	if existing, _ := f.store.ReadReviewConclusion(run.RunID); existing != nil {
		t.Fatalf("the refusal over local edits was recorded: %+v", existing)
	}
	restore()
	conclusion, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || conclusion.State != ReviewConclusionRecorded || conclusion.Attestation == "" {
		t.Fatalf("the approval did not conclude once the tree matched: %+v %v", conclusion, err)
	}
}

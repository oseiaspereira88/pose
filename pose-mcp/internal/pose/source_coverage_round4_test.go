package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-source-coverage-integrity: the cases the fourth independent review reproduced.

// R4-F01 and R4-F05: an item cut anywhere is a loss, and the diagnostic does not copy the line.
func TestAnItemCutAnywhereIsALossWithoutTheRawLine(t *testing.T) {
	_, store := reviewFixture(t)
	for _, tail := range []string{"- F1", "- F1 [", "- F1 [op", "- F1 [open CONFIDENTIAL-SENTINEL", "-"} {
		_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
		writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-ffff6666.md", "---\nschema_version: 1\nreview_id: rvw-20260803T090000Z-ffff6666\nscope: spec:alpha\ndecision: approved\nreviewed_at: 2026-08-03T09:00:00Z\n---\n\n## Findings\n\n"+tail)
		read, err := store.ReadReviewAttempts("")
		if err != nil {
			t.Fatal(err)
		}
		if len(read.Attempts) != 0 || read.Coverage.Complete() {
			t.Fatalf("%q: attempts=%d coverage=%+v", tail, len(read.Attempts), read.Coverage)
		}
		if strings.Contains(read.Coverage.Losses[0].Detail, "CONFIDENTIAL") || strings.Contains(read.Coverage.Losses[0].Detail, "F1") {
			t.Fatalf("%q: the diagnostic copied the line: %q", tail, read.Coverage.Losses[0].Detail)
		}
	}
}

// R4-F02: the public writer accepts any RFC3339 offset; the latest attestation is the latest instant.
func TestAttestationsAreOrderedByInstant(t *testing.T) {
	store, bundle, now := retentionFixture(t)
	rejected := approvedBundleAttestation(bundle, "agent:second-review")
	rejected.Decision = "rejected"
	// 04:02-08:00 is 12:02Z, one minute after the approval at 12:01Z.
	at := now.Add(2 * time.Minute).In(time.FixedZone("x", -8*3600))
	rejected.AttestedAt = at.Format(time.RFC3339)
	if _, err := store.RecordReviewAttestation(rejected, at); err != nil {
		t.Fatal(err)
	}
	read, err := store.ReadReviewAttestations(bundle.BundleID)
	if err != nil || len(read.Attestations) < 2 {
		t.Fatalf("%v %v", read, err)
	}
	if last := read.Attestations[len(read.Attestations)-1]; last.Decision != "rejected" {
		t.Fatalf("the last attestation by instant is the rejection, got %s", last.Decision)
	}
	v, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved {
		t.Fatalf("an approval stood over a later rejection written with another offset: %v", v.Blockers)
	}
}

// R4-F04: a finished spec whose open follow-up has no description is reported.
func TestAnEmptyOpenFollowupOfADoneSpecIsALoss(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/2026-10-01-empty.md", "---\nslug: empty\nstatus: done\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open]\n")
	read := ReadSpecFollowups(root)
	if read.Coverage.CompleteFor("empty") {
		t.Fatalf("an emptied follow-up disappeared: %+v", read)
	}
	root = t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/2026-10-01-draft.md", "---\nslug: draft\nstatus: draft\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open]\n")
	if read := ReadSpecFollowups(root); !read.Coverage.Complete() {
		t.Fatalf("the template placeholder of a draft is expected: %+v", read.Coverage)
	}
}

// R4-F06: a future format is unsupported, not corrupt.
func TestAFutureSchemaIsUnsupportedNotCorrupt(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	for _, c := range []struct{ dir, id string }{
		{".pose/review-bundles", bundle.BundleID},
	} {
		path := filepath.Join(store.Root, c.dir, c.id+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		doc["schema_version"] = 99
		out, _ := json.Marshal(doc)
		if err := os.WriteFile(path, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read, err := store.ReadReviewBundles("")
	if err != nil || read.Coverage.Complete() {
		t.Fatalf("%+v %v", read.Coverage, err)
	}
	if read.Coverage.Losses[0].Kind != LossUnsupported {
		t.Fatalf("kind = %s", read.Coverage.Losses[0].Kind)
	}
}

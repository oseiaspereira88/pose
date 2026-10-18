package pose

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-source-coverage-integrity: the cases the fifth independent review reproduced.

// R5-F01: a bundle without a valid sealed_at has no place in the history; it is a loss.
func TestABundleWithoutASealedInstantIsALossNotTheOldest(t *testing.T) {
	store, _, now := retentionFixture(t)
	addMatrixCheck(t, store)
	second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	rejected := approvedBundleAttestation(second, "agent:second-review")
	rejected.Decision = "rejected"
	if _, err := store.RecordReviewAttestation(rejected, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, ".pose/review-bundles", second.BundleID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	delete(doc, "sealed_at")
	out, _ := json.Marshal(doc)
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	// A bundle whose header names the scope is certainly part of its history: the scoped
	// read refuses; the global read reports it as a loss.
	if _, err := store.ReadReviewBundles("spec:backend"); err == nil {
		t.Fatal("a bundle with no sealed_at was read as part of the history")
	}
	if read, err := store.ReadReviewBundles(""); err != nil || read.Coverage.Complete() {
		t.Fatalf("a bundle with no sealed_at read as complete: %+v %v", read.Coverage, err)
	}
	closeBackendSpec(t, store)
	writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	// Either the verification refuses (the history has a record that cannot be placed) or it
	// does not approve; it never retains the earlier approval over it.
	if v, err := store.VerifyReviewBundle("spec:backend"); err == nil && v.Approved {
		t.Fatalf("an earlier approval stood over a rejection whose place in time was lost: %v", v.Blockers)
	}
}

// R5-F02: recorded work that lost its text is reported; only the bare placeholder is tolerated.
func TestAnAnnotatedFollowupThatLostItsTextIsALoss(t *testing.T) {
	doc := func(status, bullet string) string {
		return "---\nslug: s\nstatus: " + status + "\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n" + bullet + "\n"
	}
	for _, c := range []struct {
		status, bullet string
		complete       bool
	}{
		{"in-progress", "- [open] (owner:@core crit:high review:2020-01-01)", false},
		{"in-progress", "- [open]", true},
		{"draft", "- [open]", true},
		{"done", "- [open]", false},
		{"done", "- [open] (owner:@core crit:low review:2099-01-01)", false},
	} {
		root := t.TempDir()
		writeCoverageFile(t, root, ".pose/specs/2026-10-01-s.md", doc(c.status, c.bullet))
		if got := ReadSpecFollowups(root).Coverage.Complete(); got != c.complete {
			t.Errorf("%s %q: complete=%v, want %v", c.status, c.bullet, got, c.complete)
		}
	}
}

// R5-F03: a persisted component state is a component only if it names one in a format that is read.
func TestAPersistedComponentStateIsValidatedBeforeItIsUsed(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root}
	for name, c := range map[string]struct {
		body string
		want error
	}{
		"null":      {`null`, ErrComponentStateInvalid},
		"no slug":   {`{"path":"x"}`, ErrComponentStateInvalid},
		"future":    {`{"schema_version":99,"component_slug":"other"}`, ErrComponentStateUnsupported},
		"truncated": {`{"component_slug":"oth`, ErrComponentStateParse},
	} {
		writeCoverageFile(t, root, ".pose/state/components/other.json", c.body)
		if _, err := store.LoadComponentState("other"); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	writeCoverageFile(t, root, ".pose/state/components/other.json", `{"component_slug":"other","path":"other"}`)
	if state, err := store.LoadComponentState("other"); err != nil || state.ComponentSlug != "other" {
		t.Fatalf("a valid state stopped loading: %v %v", state, err)
	}
}

// R5-F04: every consumer of the bundle history orders it by instant.
func TestNoConsumerOrdersTheBundleHistoryByText(t *testing.T) {
	for _, file := range []string{"governance_replay.go", "governance_waits.go", "review_bundle.go"} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if strings.Contains(line, "SealedAt <") || strings.Contains(line, "SealedAt >") || strings.Contains(line, "AttestedAt <") || strings.Contains(line, "AttestedAt >") {
				t.Errorf("%s:%d orders a history by text: %s", file, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// R6-F01: a record that names its scope twice is not the other scope's.
func TestARecordThatNamesItsScopeTwiceIsALossNotAnotherScopes(t *testing.T) {
	_, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-20260802T120000Z-aaaa1111", "spec:alpha", "spec-closeout@1", "requirements", "")
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-bbbb2222.md", "---\nschema_version: 1\nreview_id: rvw-20260803T090000Z-bbbb2222\nscope: spec:alpha\nscope: spec:other\ndecision: rejected\nreviewed_at: 2026-08-03T09:00:00Z\n---\n")
	read, err := store.ReadReviewAttempts("spec:alpha")
	if err != nil || read.Coverage.CompleteFor("spec:alpha") {
		t.Fatalf("a repeated scope was filtered out as another scope's: %+v %v", read.Coverage, err)
	}
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil || eval.Approved {
		t.Fatalf("an earlier approval stood over a rejection whose scope is ambiguous: approved=%v err=%v", eval.Approved, err)
	}
	global, err := store.ReadReviewAttempts("")
	if err != nil || global.Coverage.Complete() {
		t.Fatalf("the global read accepted a record that names its scope twice: %+v", global.Coverage)
	}
}

// R7-F01 and R7-F02: every field the attempt is read by, once; a scope named twice is nobody's.
func TestARepeatedAttemptFieldIsALossUnattributedToAnyScope(t *testing.T) {
	for _, field := range []string{"schema_version: 1", "reviewer: agent:a", "profile: spec-closeout@1", "supersedes: x", "evidence_refs: [a]"} {
		_, store := reviewFixture(t)
		_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
		name := strings.SplitN(field, ":", 2)[0]
		writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-cccc3333.md", "---\nschema_version: 1\nreview_id: rvw-20260803T090000Z-cccc3333\nscope: spec:alpha\ndecision: approved\nreviewed_at: 2026-08-03T09:00:00Z\n"+field+"\n"+field+"\n---\n")
		read, err := store.ReadReviewAttempts("")
		if err != nil || len(read.Attempts) != 0 || read.Coverage.Complete() {
			t.Errorf("%s: a repeated field was read: attempts=%d coverage=%+v", name, len(read.Attempts), read.Coverage)
		}
	}
	_, store := reviewFixture(t)
	_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-dddd4444.md", "---\nschema_version: 1\nreview_id: x\nscope: spec:alpha\nscope: spec:other\ndecision: approved\nreviewed_at: 2026-08-03T09:00:00Z\n---\n")
	read, err := store.ReadReviewAttempts("")
	if err != nil || len(read.Coverage.Losses) != 1 || read.Coverage.Losses[0].Scope != "" {
		t.Fatalf("a scope named twice must not be attributed to either: %+v %v", read.Coverage, err)
	}
	if read.Coverage.CompleteFor("spec:alpha") || read.Coverage.CompleteFor("spec:other") {
		t.Fatal("an unattributed loss concerns every scope")
	}
}

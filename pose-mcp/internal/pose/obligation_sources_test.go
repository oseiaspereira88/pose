package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeSourceFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validSourceObligations(t *testing.T) func([]Obligation, error) []Obligation {
	return func(items []Obligation, err error) []Obligation {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range items {
			if err := ValidateObligation(o); err != nil {
				t.Fatalf("invalid obligation: %v", err)
			}
			if !strings.Contains(o.Source.Ref.String(), "/project:proj.test#") {
				t.Fatalf("a project-level obligation does not originate on the project: %s", o.Source.Ref.String())
			}
		}
		return items
	}
}

// An open docs review mark is owed by the doc's owner until resolved.
func TestAttentionSourceDocsReviewMarksAreOwed(t *testing.T) {
	root := t.TempDir()
	writeSourceFixture(t, root, ".pose/docs.json", `{"schema_version":1,"entries":[{"path":"docs/guide.md","doc_type":"howto","owner":"@docs-team"}]}`)
	writeSourceFixture(t, root, ".pose/docs-review.jsonl", `{"at":"2026-10-01T00:00:00Z","doc":"docs/guide.md","kind":"marked","trigger":"spec:demo"}
{"at":"2026-10-02T00:00:00Z","doc":"docs/old.md","kind":"marked","trigger":"spec:demo"}
{"at":"2026-10-03T00:00:00Z","doc":"docs/old.md","kind":"resolved","outcome":"no_change_needed","reason":"unaffected"}
`)
	items := validSourceObligations(t)(Store{Root: root}.docsReviewObligations("proj.test"))
	if len(items) != 1 || items[0].Source.Detail != "docs/guide.md" || items[0].Recipient.Principal != "@docs-team" || items[0].Waiting != WaitingActor {
		t.Fatalf("docs review obligations = %+v", items)
	}
}

// A mechanism with a stale trigger is owed a reassessment; a retired one is not.
func TestAttentionSourceCapabilityTriggersAreOwed(t *testing.T) {
	root := t.TempDir()
	writeSourceFixture(t, root, ".pose/capabilities/assessment.md", strings.Replace(validAssessment, "- gaps: RBAC mapping open; retrieval is lexical\n", "- gaps: RBAC mapping open; retrieval is lexical\n- stale: since=2026-10-02T00:00:00Z;trigger=spec:demo\n", 1))
	store := Store{Root: root}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		t.Fatal(err)
	}
	stale := 0
	for _, m := range assessment.Mechanisms {
		if len(m.StaleTriggers) > 0 && !m.Retired {
			stale++
		}
	}
	items := validSourceObligations(t)(store.capabilityTriggerObligations("proj.test"))
	if len(items) != stale || stale == 0 {
		t.Fatalf("capability obligations = %+v (stale mechanisms %d)", items, stale)
	}
}

// Only releases newer than the newest verified one are in flight.
func TestAttentionSourceReleaseQueueSkipsSupersededReleases(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"v1.0.0", "v1.1.0", "v1.2.0"} {
		writeSourceFixture(t, root, ".pose/releases/"+v+"/manifest.json", `{"version":"`+v+`"}`)
	}
	writeSourceFixture(t, root, ".pose/releases/v1.1.0/events.jsonl", `{"version":"v1.1.0","state":"tagged"}
{"version":"v1.1.0","state":"published","evidence_digest":"sha256:p"}
{"version":"v1.1.0","state":"verified","evidence":{"publication_digest":"sha256:p"}}
`)
	writeSourceFixture(t, root, ".pose/releases/v1.0.0/events.jsonl", `{"version":"v1.0.0","state":"failed"}
`)
	items := validSourceObligations(t)(Store{Root: root}.releaseQueueObligations("proj.test"))
	if len(items) != 1 || items[0].Source.Detail != "v1.2.0" || items[0].ReasonCode != "release-prepared" {
		t.Fatalf("release queue = %+v", items)
	}
	// A later release that was tagged abandons the prepared one.
	writeSourceFixture(t, root, ".pose/releases/v1.3.0/manifest.json", `{"version":"v1.3.0"}`)
	writeSourceFixture(t, root, ".pose/releases/v1.3.0/events.jsonl", `{"version":"v1.3.0","state":"tagged"}
`)
	items = validSourceObligations(t)(Store{Root: root}.releaseQueueObligations("proj.test"))
	if len(items) != 1 || items[0].Source.Detail != "v1.3.0" || items[0].ReasonCode != "release-tagged" {
		t.Fatalf("release queue after a later tag = %+v", items)
	}
}

// An accepted risk past its review date in the latest record is owed; one in a
// superseded record, or not yet due, is not.
func TestAttentionSourceOverdueFindingsAreOwed(t *testing.T) {
	root := t.TempDir()
	record := func(name, at, findings string) {
		writeSourceFixture(t, root, ".pose/reviews/"+name+".md", "---\nschema_version: 1\nreview_id: "+name+"\nscope: spec:alpha\nscope_digest: sha256:x\nprofile: spec-closeout@1\nreviewer: agent:r\ndecision: approved\nreviewed_at: "+at+"\nsupersedes:\nevidence_refs: []\n---\n\n## Criteria\n- scope [passed] evidence:x\n\n## Findings\n"+findings)
	}
	record("rvw-old", "2026-09-01T00:00:00Z", "- stale-risk [accepted-risk] severity:low action:a owner:@old rationale:r review_by:2026-09-02\n")
	record("rvw-new", "2026-10-01T00:00:00Z", "- due-risk [accepted-risk] severity:low action:a owner:@team rationale:r review_by:2026-10-05\n- later-risk [accepted-risk] severity:low action:a owner:@team rationale:r review_by:2026-12-01\n")
	items := validSourceObligations(t)(Store{Root: root}.findingObligations("proj.test", time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)))
	if len(items) != 1 || !strings.HasSuffix(items[0].Source.Detail, "/due-risk") || items[0].Recipient.Principal != "@team" {
		t.Fatalf("finding obligations = %+v", items)
	}
}

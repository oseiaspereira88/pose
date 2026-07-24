package pose

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeDocsReviewJSONL(t *testing.T, root string, events []DocsReviewEvent) {
	t.Helper()
	path := filepath.Join(root, ".pose", "docs-review.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadDocsReviewEvents_MissingFileIsEmptyNotError(t *testing.T) {
	root := t.TempDir()
	events, err := LoadDocsReviewEvents(filepath.Join(root, ".pose", "docs-review.jsonl"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %+v", events)
	}
}

func TestLoadDocsReviewEvents_RoundTrips(t *testing.T) {
	root := t.TempDir()
	writeDocsReviewJSONL(t, root, []DocsReviewEvent{
		{At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x", Hits: []string{"site/a"}},
	})
	events, err := LoadDocsReviewEvents(filepath.Join(root, ".pose", "docs-review.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Doc != "docs/a.md" || events[0].Kind != "marked" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestLoadDocsReviewEvents_RejectsMalformedLine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".pose", "docs-review.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDocsReviewEvents(path); err == nil {
		t.Fatal("expected error for malformed JSONL line")
	}
}

func TestPendingDocsReviews_MarkThenResolveClearsIt(t *testing.T) {
	events := []DocsReviewEvent{
		{At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x"},
		{At: "2026-07-24T01:00:00Z", Doc: "docs/a.md", Kind: "resolved", Outcome: "updated", Commit: "abc1234"},
	}
	pending := PendingDocsReviews(events)
	if len(pending["docs/a.md"]) != 0 {
		t.Fatalf("expected no pending triggers after resolve, got %+v", pending)
	}
}

func TestPendingDocsReviews_MultipleTriggersAccumulate(t *testing.T) {
	events := []DocsReviewEvent{
		{At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x"},
		{At: "2026-07-24T01:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:y"},
	}
	pending := PendingDocsReviews(events)
	if len(pending["docs/a.md"]) != 2 {
		t.Fatalf("expected 2 accumulated triggers, got %+v", pending["docs/a.md"])
	}
}

func TestPendingDocsReviews_SameTriggerReplacesInPlace(t *testing.T) {
	events := []DocsReviewEvent{
		{At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x", Hits: []string{"site/a"}},
		{At: "2026-07-24T02:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x", Hits: []string{"site/a", "site/b"}},
	}
	pending := PendingDocsReviews(events)
	got := pending["docs/a.md"]
	if len(got) != 1 || !reflect.DeepEqual(got[0].Hits, []string{"site/a", "site/b"}) {
		t.Fatalf("expected single replaced trigger with updated hits, got %+v", got)
	}
}

func TestMatchesOwns_DirectoryPrefixAndExactAndGlob(t *testing.T) {
	owns := []string{"site", "docs/exact.md", "pose-mcp/internal/*.go"}
	cases := map[string]bool{
		"site/README.md":               true,
		"site/assets/styles.css":       true,
		"docs/exact.md":                true,
		"docs/other.md":                false,
		"pose-mcp/internal/foo.go":     true,
		"pose-mcp/internal/cli/foo.go": false,
		"unrelated/file.md":            false,
	}
	for file, want := range cases {
		if got := MatchesOwns(file, owns); got != want {
			t.Errorf("MatchesOwns(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestCheckDocs_ReviewPendingProjectsFromJSONL(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody\n")
	writeDocsReviewJSONL(t, root, []DocsReviewEvent{
		{At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:x", Hits: []string{"site/a"}},
	})
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if len(result.ReviewPending) != 1 || result.ReviewPending[0].Doc != "docs/a.md" {
		t.Fatalf("expected review_pending to project the mark, got %+v", result.ReviewPending)
	}
}

func TestCheckDocs_NoReviewLogIsEmptyReviewPending(t *testing.T) {
	root := t.TempDir()
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if len(result.ReviewPending) != 0 {
		t.Fatalf("expected empty review_pending, got %+v", result.ReviewPending)
	}
}

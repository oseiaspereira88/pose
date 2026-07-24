package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func docsReviewTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runStateGit(t, root, "init")
	runStateGit(t, root, "config", "user.email", "test@example.com")
	runStateGit(t, root, "config", "user.name", "Docs Review Test")
	return root
}

func writeDocsReviewManifest(t *testing.T, root string) {
	t.Helper()
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{
		{Path: "docs/a.md", DocType: "reference", Owns: []string{"site"}, Owner: "@alice"},
		{Path: "docs/b.md", DocType: "reference"},
	}}
	raw, _ := json.Marshal(manifest)
	path := filepath.Join(root, ".pose", "docs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDocsReviewConsumer_NoManifestIsNoOp(t *testing.T) {
	root := docsReviewTestRoot(t)
	if err := docsReviewConsumer(root, HookEvent{Kind: "spec_closeout", Target: "demo", Commit: "0000000", At: time.Now()}); err != nil {
		t.Fatalf("no-manifest case must never error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "docs-review.jsonl")); !os.IsNotExist(err) {
		t.Fatal("docs-review.jsonl must not be created as a side effect")
	}
}

func TestDocsReviewConsumer_PathsFallbackMarksDoc(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)

	siteFile := filepath.Join(root, "site", "README.md")
	if err := os.MkdirAll(filepath.Dir(siteFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(siteFile, []byte("# site\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runStateGit(t, root, "add", "site/README.md")
	runStateGit(t, root, "commit", "-m", "add site readme")
	commit := gitHeadCommit(root)

	if err := docsReviewConsumer(root, HookEvent{Kind: "spec_closeout", Target: "demo", Commit: commit, At: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := pose.Store{Root: root}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Doc != "docs/a.md" || events[0].Kind != "marked" {
		t.Fatalf("expected exactly 1 marked event for docs/a.md, got %+v", events)
	}

	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "refresh-log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"consumer":"docs-review"`)) {
		t.Fatalf("expected a docs-review refresh-log entry, got %s", raw)
	}
}

func TestDocsReviewConsumer_NoMappingLogsUnavailable(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	if err := docsReviewConsumer(root, HookEvent{Kind: "spec_closeout", Target: "demo", Commit: "0000000", At: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "refresh-log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("docs_mapping_unavailable")) {
		t.Fatalf("expected docs_mapping_unavailable signal, got %s", raw)
	}
}

func TestDocsReviewConsumer_RerunWithSameCommitDoesNotDuplicateMark(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	siteFile := filepath.Join(root, "site", "README.md")
	if err := os.MkdirAll(filepath.Dir(siteFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(siteFile, []byte("# site\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runStateGit(t, root, "add", "site/README.md")
	runStateGit(t, root, "commit", "-m", "add site readme")
	commit := gitHeadCommit(root)

	ev := HookEvent{Kind: "spec_closeout", Target: "demo", Commit: commit, At: time.Now()}
	if err := docsReviewConsumer(root, ev); err != nil {
		t.Fatal(err)
	}
	if err := docsReviewConsumer(root, ev); err != nil {
		t.Fatal(err)
	}
	store := pose.Store{Root: root}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected the identical re-mark to be deduped, got %d events: %+v", len(events), events)
	}
}

func TestCollectDocsReviewFollowups_UsesEntryOwnerAndAppearsInAggregate(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	if err := appendDocsReviewEvent(root, pose.DocsReviewEvent{
		At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:demo", Hits: []string{"site/README.md"},
	}); err != nil {
		t.Fatal(err)
	}
	entries := collectDocsReviewFollowups(root)
	if len(entries) != 1 {
		t.Fatalf("expected 1 synthetic followup, got %+v", entries)
	}
	if entries[0].Owner != "@alice" {
		t.Fatalf("expected owner from the manifest entry, got %q", entries[0].Owner)
	}
	if entries[0].Spec != "docs:docs/a.md" {
		t.Fatalf("unexpected origin: %q", entries[0].Spec)
	}
}

func TestCollectDocsReviewFollowups_NoMarksIsEmpty(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	if entries := collectDocsReviewFollowups(root); len(entries) != 0 {
		t.Fatalf("expected no followups, got %+v", entries)
	}
}

func TestDocsReviewResolve_UpdatedClearsMark(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	if err := appendDocsReviewEvent(root, pose.DocsReviewEvent{
		At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:demo",
	}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := docsReviewResolve(root, []string{"docs/a.md"}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	store := pose.Store{Root: root}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		t.Fatal(err)
	}
	pending := pose.PendingDocsReviews(events)
	if len(pending["docs/a.md"]) != 0 {
		t.Fatalf("expected mark cleared after resolve, got %+v", pending)
	}
	if events[len(events)-1].Outcome != "updated" || events[len(events)-1].Commit == "" {
		t.Fatalf("expected outcome=updated with a captured commit, got %+v", events[len(events)-1])
	}
}

func TestDocsReviewResolve_NoChangeRequiresReason(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	if err := appendDocsReviewEvent(root, pose.DocsReviewEvent{
		At: "2026-07-24T00:00:00Z", Doc: "docs/a.md", Kind: "marked", Trigger: "spec:demo",
	}); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := docsReviewResolve(root, []string{"docs/a.md", "--no-change"}, &stdout, &stderr, localeEN); code != 2 {
		t.Fatalf("expected exit 2 without --reason, got %d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := docsReviewResolve(root, []string{"docs/a.md", "--no-change", "--reason", "already accurate"}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0 with --reason, got %d, stderr=%s", code, stderr.String())
	}
}

func TestDocsReviewResolve_NoPendingErrors(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	var stdout, stderr bytes.Buffer
	if code := docsReviewResolve(root, []string{"docs/a.md"}, &stdout, &stderr, localeEN); code != 1 {
		t.Fatalf("expected exit 1 for a doc with no pending review, got %d", code)
	}
}

func TestDocsReviewRequest_ManualMarksDoc(t *testing.T) {
	root := docsReviewTestRoot(t)
	writeDocsReviewManifest(t, root)
	var stdout, stderr bytes.Buffer
	if code := docsReviewRequest(root, []string{"docs/b.md", "--reason", "novo endpoint"}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	store := pose.Store{Root: root}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Trigger != "manual:novo endpoint" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestDocsReviewRequest_AllStaleMarksStaleDocs(t *testing.T) {
	root := docsReviewTestRoot(t)
	manifest := pose.DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []pose.DocsEntry{
		{Path: "docs/a.md", DocType: "reference", ReviewAfter: "2020-01-01"},
	}}
	raw, _ := json.Marshal(manifest)
	writeCLIDocsFixture(t, root, ".pose/docs.json", string(raw))
	writeCLIDocsFixture(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody\n")

	var stdout, stderr bytes.Buffer
	if code := docsReviewRequest(root, []string{"--all-stale"}, &stdout, &stderr, localeEN); code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, stderr.String())
	}
	store := pose.Store{Root: root}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Trigger != "stale-bridge" || events[0].Doc != "docs/a.md" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestResolveDocsViaComponentsHit_ComponentRefMatchesOwns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		structured, _ := json.Marshal(graphForgeHitResult{Hits: []map[string]any{
			{"component_id": "site-frontend", "level": "direct"},
		}})
		resp := jsonRPCToolCallResult{Result: &struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
			IsError           bool            `json:"isError"`
		}{StructuredContent: structured}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	caller := httpComponentsHitCaller{url: srv.URL, projectID: "demo"}
	manifest := &pose.DocsManifest{Entries: []pose.DocsEntry{
		{Path: "docs/a.md", Owns: []string{"component:site-frontend"}},
	}}
	hits, ok := resolveDocsViaComponentsHit(caller, HookEvent{Target: "demo-spec"}, manifest, docsReviewPolicy{MinHits: 1, Level: "direct"})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if _, present := hits["docs/a.md"]; !present {
		t.Fatalf("expected docs/a.md to be hit via component ref, got %+v", hits)
	}
}

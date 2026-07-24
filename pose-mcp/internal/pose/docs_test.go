package pose

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func writeDocsFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseDocsManifest_ValidRoundTrips(t *testing.T) {
	raw := `{"schema_version":1,"roots":["docs"],"entries":[{"path":"docs/a.md","doc_type":"reference"}]}`
	m, err := ParseDocsManifest([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.SchemaVersion != 1 || len(m.Entries) != 1 || m.Entries[0].Path != "docs/a.md" {
		t.Fatalf("unexpected manifest: %+v", m)
	}
}

func TestParseDocsManifest_RejectsMissingSchemaVersion(t *testing.T) {
	if _, err := ParseDocsManifest([]byte(`{"roots":["docs"],"entries":[]}`)); err == nil {
		t.Fatal("expected error for missing schema_version")
	}
}

func TestParseDocsManifest_RejectsNewerSchema(t *testing.T) {
	if _, err := ParseDocsManifest([]byte(`{"schema_version":99,"roots":["docs"],"entries":[]}`)); err == nil {
		t.Fatal("expected error for a schema_version newer than supported")
	}
}

func TestParseDocsManifest_RejectsDuplicatePaths(t *testing.T) {
	raw := `{"schema_version":1,"roots":["docs"],"entries":[{"path":"docs/a.md","doc_type":"reference"},{"path":"docs/a.md","doc_type":"howto"}]}`
	if _, err := ParseDocsManifest([]byte(raw)); err == nil {
		t.Fatal("expected error for duplicate entry path")
	}
}

func TestParseDocsManifest_RejectsMalformedJSON(t *testing.T) {
	if _, err := ParseDocsManifest([]byte(`{not json`)); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestCheckDocs_MissingDeclaredEntry(t *testing.T) {
	root := t.TempDir()
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/missing.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if result.Totals.Errors != 1 {
		t.Fatalf("expected 1 error, got totals=%+v issues=%+v", result.Totals, result.Issues)
	}
	if result.Issues[0].Rule != "missing" {
		t.Fatalf("expected rule=missing, got %+v", result.Issues[0])
	}
}

func TestCheckDocs_UndeclaredFileWarns(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/orphan.md", "---\ntitle: Orphan\ndoc_type: reference\n---\nbody\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if result.Totals.Undeclared != 1 || result.Totals.Warnings != 1 {
		t.Fatalf("expected 1 undeclared warning, got totals=%+v", result.Totals)
	}
	if result.Issues[0].Rule != "undeclared" || result.Issues[0].Path != "docs/orphan.md" {
		t.Fatalf("unexpected issue: %+v", result.Issues[0])
	}
}

func TestCheckDocs_MissingFrontmatterWarns(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "# No frontmatter here\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "missing_frontmatter" && issue.Path == "docs/a.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing_frontmatter issue, got %+v", result.Issues)
	}
}

func TestCheckDocs_BrokenRelativeLinkErrors(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nSee [b](./b.md) for more.\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "broken_link" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected broken_link issue, got %+v", result.Issues)
	}
}

func TestCheckDocs_ValidRelativeLinkDoesNotError(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nSee [b](./b.md).\n")
	writeDocsFile(t, root, "docs/b.md", "---\ntitle: B\ndoc_type: reference\n---\nbody\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"}, {Path: "docs/b.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	for _, issue := range result.Issues {
		if issue.Rule == "broken_link" {
			t.Fatalf("unexpected broken_link issue: %+v", issue)
		}
	}
}

func TestCheckDocs_BrokenTypedReferenceErrors(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nSee spec:does-not-exist.\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "broken_reference" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected broken_reference issue, got %+v", result.Issues)
	}
}

func TestCheckDocs_SecurityScanFlagsSecretShapedContent(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nAKIAABCDEFGHIJKLMNOP\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "security" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected security issue, got %+v", result.Issues)
	}
}

func TestCheckDocs_ReviewAfterPastDueIsStale(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody\n")
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference", ReviewAfter: yesterday},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if result.Totals.Stale != 1 {
		t.Fatalf("expected 1 stale doc, got totals=%+v", result.Totals)
	}
}

func TestCheckDocs_DefaultReviewDaysUsesGitHistory(t *testing.T) {
	root := t.TempDir()
	runStateTestGit(t, root, "init")
	runStateTestGit(t, root, "config", "user.email", "test@example.com")
	runStateTestGit(t, root, "config", "user.name", "Docs Test")
	writeDocsFile(t, root, "docs/a.md", "---\ntitle: A\ndoc_type: reference\n---\nbody\n")
	runStateTestGit(t, root, "add", "docs/a.md")
	cmd := exec.Command("git", "-C", root, "commit", "-m", "add doc")
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2020-01-01T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}

	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, DefaultReviewDays: 30, Entries: []DocsEntry{
		{Path: "docs/a.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if result.Totals.Stale != 1 {
		t.Fatalf("expected 1 stale doc from old git history, got totals=%+v issues=%+v", result.Totals, result.Issues)
	}
}

func TestCheckDocs_SeverityOffSuppressesRule(t *testing.T) {
	root := t.TempDir()
	writeDocsFile(t, root, "docs/orphan.md", "---\ntitle: Orphan\ndoc_type: reference\n---\nbody\n")
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Severities: map[string]string{"undeclared": "off"}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if len(result.Issues) != 0 || result.Totals.Warnings != 0 {
		t.Fatalf("expected undeclared rule suppressed, got %+v", result)
	}
}

func TestCheckDocs_SeverityOverrideChangesErrorToWarning(t *testing.T) {
	root := t.TempDir()
	manifest := &DocsManifest{SchemaVersion: 1, Roots: []string{"docs"}, Severities: map[string]string{"missing": "warning"}, Entries: []DocsEntry{
		{Path: "docs/missing.md", DocType: "reference"},
	}}
	store := Store{Root: root}
	result := store.CheckDocs(context.Background(), manifest)
	if result.Totals.Errors != 0 || result.Totals.Warnings != 1 {
		t.Fatalf("expected severity override to warning, got totals=%+v", result.Totals)
	}
}

func TestDocsProfileRoots_KnownProfiles(t *testing.T) {
	if !ValidDocsProfile("monorepo") || ValidDocsProfile("not-a-profile") {
		t.Fatal("validDocsProfile did not distinguish known from unknown profiles")
	}
	if roots := DocsProfileRoots("cli"); len(roots) == 0 {
		t.Fatal("expected non-empty default roots for cli profile")
	}
}

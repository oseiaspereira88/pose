package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExcerptMarkdownTitleAndParagraph(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Widget Factory\n\nWidget Factory builds widgets from raw gears.\nIt has a CLI and a web dashboard.\n\n## Installation\n\nrun make install\n")
	got := excerptMarkdown(filepath.Join(dir, "README.md"))
	want := "Widget Factory: Widget Factory builds widgets from raw gears. It has a CLI and a web dashboard."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExcerptMarkdownSkipsBadgesAndHTMLComments(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "[![CI](https://example.com/badge.svg)](https://example.com)\n![logo](logo.png)\n\n<!-- internal note\n   spanning lines -->\n\n# Widget Factory\n\nBuilds widgets.\n")
	got := excerptMarkdown(filepath.Join(dir, "README.md"))
	if got != "Widget Factory: Builds widgets." {
		t.Errorf("got %q", got)
	}
}

func TestExcerptMarkdownBadgesOnlyYieldsNothing(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "[![CI](https://example.com/badge.svg)](https://example.com)\n![logo](logo.png)\n")
	if got := excerptMarkdown(filepath.Join(dir, "README.md")); got != "" {
		t.Errorf("expected empty excerpt from a badges-only README, got %q", got)
	}
}

func TestExcerptMarkdownAbsentFile(t *testing.T) {
	dir := t.TempDir()
	if got := excerptMarkdown(filepath.Join(dir, "README.md")); got != "" {
		t.Errorf("expected empty excerpt for an absent file, got %q", got)
	}
}

func TestExcerptMarkdownNoHeadingUsesFirstParagraph(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "Just a plain description with no heading at all.\nSecond line of the same paragraph.\n\nA second paragraph that should not be included.\n")
	got := excerptMarkdown(filepath.Join(dir, "README.md"))
	if got != "Just a plain description with no heading at all. Second line of the same paragraph." {
		t.Errorf("got %q", got)
	}
}

func TestTruncateSummaryCutsOnWordBoundary(t *testing.T) {
	s := strings.Repeat("word ", 200)
	got := truncateSummary(s, 50)
	if len(strings.TrimSuffix(got, "…")) > 50 {
		t.Errorf("truncated result exceeds the limit before the ellipsis: %d chars: %q", len(got), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected a truncation marker, got %q", got)
	}
	if strings.HasSuffix(strings.TrimSuffix(got, "…"), "wor") {
		t.Errorf("cut mid-word: %q", got)
	}
}

func TestTruncateSummaryUnderLimitUnchanged(t *testing.T) {
	if got := truncateSummary("short", 50); got != "short" {
		t.Errorf("got %q", got)
	}
}

func TestExtractProjectContextCombinesReadmeAndClaude(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Widget Factory\n\nBuilds widgets.\n")
	mustWrite(t, filepath.Join(dir, "CLAUDE.md"), "# Agent notes\n\nRun tests with make test before every commit.\n")
	got := extractProjectContext(dir)
	if !strings.Contains(got, "Widget Factory: Builds widgets.") {
		t.Errorf("missing README excerpt: %q", got)
	}
	if !strings.Contains(got, "Agent notes: Run tests with make test before every commit.") {
		t.Errorf("missing CLAUDE.md excerpt: %q", got)
	}
}

func TestExtractProjectContextNeitherFilePresent(t *testing.T) {
	dir := t.TempDir()
	if got := extractProjectContext(dir); got != "" {
		t.Errorf("expected empty summary, got %q", got)
	}
}

func TestInjectExtractedProjectContextReplacesPlaceholderPrefixOnly(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "README.md"), "# Widget Factory\n\nBuilds widgets.\n")
	content := "# AGENTS.md — {{PROJECT_NAME}}\n\n## Project context\n\n{{PROJECT_NAME}}: describe the repository's purpose and its main components.\n\n## Instance notes\n"
	got := injectExtractedProjectContext(content, dir)
	if !strings.Contains(got, "{{PROJECT_NAME}}: Widget Factory: Builds widgets.") {
		t.Errorf("placeholder line not replaced: %s", got)
	}
	if strings.Count(got, "{{PROJECT_NAME}}") != 2 { // title line + injected line
		t.Errorf("unexpected number of {{PROJECT_NAME}} tokens left for the replacer: %s", got)
	}
}

func TestInjectExtractedProjectContextNoOpWithoutReadme(t *testing.T) {
	dir := t.TempDir()
	content := "{{PROJECT_NAME}}: describe the repository's purpose and its main components.\n"
	if got := injectExtractedProjectContext(content, dir); got != content {
		t.Errorf("expected no change without a README/CLAUDE.md, got %q", got)
	}
}

// --- cmdInstall integration ---

func TestInstallExtractsProjectContextFromReadme(t *testing.T) {
	repo := newGitRepo(t)
	mustWrite(t, filepath.Join(repo, "README.md"), "# Widget Factory\n\nWidget Factory turns raw gears into finished widgets via a CLI and a web dashboard.\n")

	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}

	agents, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(agents)
	if strings.Contains(text, "describe the repository's purpose and its main components") {
		t.Errorf("placeholder was not replaced:\n%s", text)
	}
	if !strings.Contains(text, "Widget Factory turns raw gears into finished widgets") {
		t.Errorf("extracted README content missing from AGENTS.md:\n%s", text)
	}
}

func TestInstallWithoutReadmeKeepsPlaceholderUnchanged(t *testing.T) {
	repo := newGitRepo(t)

	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}

	agents, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "describe the repository's purpose and its main components") {
		t.Errorf("expected the unfilled placeholder to survive an install with no README.md/CLAUDE.md:\n%s", agents)
	}
}

func TestInstallPreservesHandEditedProjectContextOnRerun(t *testing.T) {
	repo := newGitRepo(t)
	mustWrite(t, filepath.Join(repo, "README.md"), "# Widget Factory\n\nBuilds widgets from raw gears.\n")

	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}

	agentsPath := filepath.Join(repo, "AGENTS.md")
	agents, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	// The operator further hand-edits the extracted line, exactly as the
	// section's own comment invites them to.
	edited := strings.Replace(string(agents), "Widget Factory: Builds widgets from raw gears.", "widget-factory: a hand-curated description the team wrote themselves.", 1)
	if edited == string(agents) {
		t.Fatalf("test setup: extracted line not found to edit:\n%s", agents)
	}
	if err := os.WriteFile(agentsPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// A README.md change between runs must not overwrite what the operator
	// already wrote — the section is instance-owned once populated.
	mustWrite(t, filepath.Join(repo, "README.md"), "# Widget Factory\n\nCompletely different upstream description.\n")

	out.Reset()
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("re-run exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}
	after, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "a hand-curated description the team wrote themselves") {
		t.Errorf("re-run reverted the operator's hand edit:\n%s", after)
	}
	if strings.Contains(string(after), "Completely different upstream description") {
		t.Errorf("re-run overwrote the instance-owned section from the changed README:\n%s", after)
	}
}

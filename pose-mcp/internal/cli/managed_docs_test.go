package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

const canonicalManual = `# POSE

## 1) What it is

Engine-owned intro, version 2.

## 9) Instance limitations

<!-- pose:instance-owned -->

- Document limitations as the instance evolves.

## 12) Executive summary

Engine-owned summary, version 2.
`

func TestMergeManagedDocRefreshesEngineSectionsAndKeepsInstanceOnes(t *testing.T) {
	local := `# POSE

## 1) What it is

Engine-owned intro, version 1.

## 9) Instance limitations

<!-- pose:instance-owned -->

- The payments module has no coverage in module-metadata.json.

## 12) Executive summary

Engine-owned summary, version 1.
`
	merged, preserved := MergeManagedDoc(canonicalManual, local)
	if !preserved {
		t.Fatal("merge must report that instance content was preserved")
	}
	if !strings.Contains(merged, "Engine-owned intro, version 2.") ||
		!strings.Contains(merged, "Engine-owned summary, version 2.") {
		t.Errorf("engine-owned sections did not refresh:\n%s", merged)
	}
	if !strings.Contains(merged, "payments module has no coverage") {
		t.Errorf("instance-owned body was lost:\n%s", merged)
	}
	if strings.Contains(merged, "version 1.") {
		t.Errorf("stale engine content survived:\n%s", merged)
	}
}

func TestMergeManagedDocKeepsSectionsTheEngineDoesNotKnow(t *testing.T) {
	local := canonicalManual + `
## 13) Team-specific appendix

Locally invented section that the engine never ships.
`
	merged, preserved := MergeManagedDoc(canonicalManual, local)
	if !preserved {
		t.Fatal("an unknown local section must count as preserved content")
	}
	if !strings.Contains(merged, "Locally invented section") {
		t.Errorf("a section the instance added was dropped:\n%s", merged)
	}
}

func TestMergeManagedDocIsIdempotent(t *testing.T) {
	once, _ := MergeManagedDoc(canonicalManual, canonicalManual)
	twice, _ := MergeManagedDoc(canonicalManual, once)
	if once != twice {
		t.Errorf("merge is not idempotent:\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
	if once != canonicalManual {
		t.Errorf("merging a manual with itself must be a no-op:\n%s", once)
	}
}

func TestMergeManagedDocPrefersFirstDuplicateHeading(t *testing.T) {
	local := `# POSE

## 9) Instance limitations

<!-- pose:instance-owned -->

- First body wins.

## 9) Instance limitations

<!-- pose:instance-owned -->

- Second body must not silently replace it.
`
	merged, _ := MergeManagedDoc(canonicalManual, local)
	if !strings.Contains(merged, "First body wins.") || strings.Contains(merged, "Second body must not") {
		t.Errorf("duplicate headings resolved to the wrong body:\n%s", merged)
	}
}

func TestRefreshManagedDocsUpdatesAnInstalledManual(t *testing.T) {
	root := t.TempDir()
	// A manual stale enough to prove the refresh happened, carrying an
	// instance-owned body that must survive it.
	stale := `# POSE

## 1) What it is

POSE is the operating standard for agent work in **acme**.

Ancient engine prose that the shipped manual no longer contains anywhere.

## 9) Instance limitations

<!-- pose:instance-owned -->

- Instance note that must survive the refresh.
`
	if err := os.WriteFile(filepath.Join(root, "POSE.md"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := refreshManagedDocs(root, "", &out, localeEN); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "POSE.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.Contains(text, "Ancient engine prose") {
		t.Errorf("engine-owned content did not refresh:\n%s", text)
	}
	if !strings.Contains(text, "Instance note that must survive the refresh.") {
		t.Errorf("instance-owned content was lost:\n%s", text)
	}
	if strings.Contains(text, "{{PROJECT_NAME}}") {
		t.Errorf("refresh reintroduced an unresolved scaffold placeholder:\n%s", text)
	}
	if !strings.Contains(text, "**acme**") {
		t.Errorf("refresh lost the instance's own project name:\n%s", text)
	}
}

func TestRefreshManagedDocsKeepsTheInstalledLocale(t *testing.T) {
	// An instance installed with --locale pt-BR must not be rewritten in
	// English just because the upgrade ran in an English shell: the locale is
	// a property of the installed manual, not of the invoking environment.
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--locale", "pt-BR", "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	before, err := os.ReadFile(filepath.Join(repo, "POSE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(before), "## 1) O que é") {
		t.Fatalf("fixture is not the pt-BR manual")
	}

	var refreshOut strings.Builder
	if err := refreshManagedDocs(repo, "", &refreshOut, localeEN); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(filepath.Join(repo, "POSE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "## 1) What it is") {
		t.Error("a pt-BR instance was rewritten in English")
	}
	if !strings.Contains(string(after), "## 1) O que é") {
		t.Error("the pt-BR manual lost its own headings")
	}
	if string(before) != string(after) {
		t.Error("refreshing a freshly installed manual must be a no-op")
	}
}

func TestRefreshManagedDocsIgnoresAbsentManual(t *testing.T) {
	root := t.TempDir()
	var out strings.Builder
	if err := refreshManagedDocs(root, "", &out, localeEN); err != nil {
		t.Fatalf("an absent manual is pose install's job, not an error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "POSE.md")); !os.IsNotExist(err) {
		t.Error("refresh must not create a manual where none existed")
	}
}

// TestResolveDocLocaleHonorsExplicitEnglish regression-covers spec
// pose-locale-switch-section-identity, root cause 1: an explicit "en"
// preference must win even against a pt-BR-detected existing manual —
// before this fix, resolveDocLocale's own short-circuit only recognized a
// non-"en" explicit preference, so "en" silently fell through to
// auto-detection and got overridden back to whatever the existing content
// already was.
func TestResolveDocLocaleHonorsExplicitEnglish(t *testing.T) {
	dist := scaffold.Dist()
	ptBRExisting, err := fs.ReadFile(dist, "locales/pt-BR/POSE.md")
	if err != nil {
		t.Fatalf("reading shipped pt-BR POSE.md: %v", err)
	}
	if got := resolveDocLocale(dist, "POSE.md", string(ptBRExisting), "en", true); got != "" {
		t.Errorf("explicit en against pt-BR existing content: resolveDocLocale = %q, want \"\" (English)", got)
	}
	// A non-explicit "en" (a caller's zero-value default, not a real ask)
	// must still fall through to auto-detection, or a caller like
	// `pose install`'s own locale auto-detect (which passes "en" as its
	// unset default, explicit=false) would incorrectly force English onto
	// an already-localized target on every unrelated rerun.
	if got := resolveDocLocale(dist, "POSE.md", string(ptBRExisting), "en", false); got != "pt-BR" {
		t.Errorf("non-explicit en against pt-BR existing content: resolveDocLocale = %q, want \"pt-BR\" (auto-detected)", got)
	}
}

// TestRefreshManagedDocsSwitchesLocaleWithoutDuplicating regression-covers
// spec pose-locale-switch-section-identity, root cause 2: switching
// `--locale` on a plain `pose update` (no --force) must replace the
// manual's language, not concatenate both — literal heading matching
// treated a translated section as unknown content and appended it instead
// of recognizing it as the same section.
func TestRefreshManagedDocsSwitchesLocaleWithoutDuplicating(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--locale", "pt-BR", "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}

	var refreshOut strings.Builder
	if err := refreshManagedDocs(repo, "en", &refreshOut, localeEN); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(filepath.Join(repo, "POSE.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(after)
	if strings.Contains(text, "## 1) O que é") {
		t.Errorf("an explicit --locale en switch left the old pt-BR heading behind:\n%s", text)
	}
	if !strings.Contains(text, "## 1) What it is") {
		t.Errorf("an explicit --locale en switch did not produce the English heading:\n%s", text)
	}
	if strings.Count(text, "## 1)") > 1 {
		t.Errorf("locale switch duplicated section 1 instead of replacing it:\n%s", text)
	}
}

// TestRefreshManagedDocsWarnsAndBacksUpDroppedContent regression-covers the
// corrected scope of the original finding on `pose update`'s doc refresh: a
// hand-edit inside an engine-owned section body (no heading of its own) is
// legitimately overwritten by the refresh, but the operator must be told —
// before this fix, refreshManagedDocs backed up unconditionally on any
// change yet logged the generic "merged ... preserved" line even when
// content was in fact dropped, reading as if nothing had been lost.
func TestRefreshManagedDocsWarnsAndBacksUpDroppedContent(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--locale", "en", "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	agentsPath := filepath.Join(repo, "AGENTS.md")
	original, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	const customLine = "- Cloudflare Workers backend: `.pose/rules/backend-worker.md`\n"
	marker := "## Domain rules"
	idx := strings.Index(string(original), marker)
	if idx < 0 {
		t.Fatalf("fixture AGENTS.md has no %q section:\n%s", marker, original)
	}
	// Insert the custom line just after the section's next newline, inside
	// its body — not under its own heading.
	bodyStart := idx + strings.Index(string(original)[idx:], "\n") + 1
	edited := string(original)[:bodyStart] + customLine + string(original)[bodyStart:]
	if err := os.WriteFile(agentsPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	var refreshOut strings.Builder
	if err := refreshManagedDocs(repo, "en", &refreshOut, localeEN); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(refreshOut.String(), "backed up customized") {
		t.Errorf("refresh must warn that customized content was backed up, got:\n%s", refreshOut.String())
	}
	backup, err := os.ReadFile(agentsPath + ".pose-backup")
	if err != nil {
		t.Fatalf(".pose-backup was not written: %v", err)
	}
	if !strings.Contains(string(backup), customLine) {
		t.Errorf("backup does not contain the dropped custom line:\n%s", backup)
	}
}

func TestMergeDropsLocalContentDetectsTextWithoutItsOwnHeading(t *testing.T) {
	// A note appended to the end of the file lands in an engine-owned section
	// body, so the refresh legitimately overwrites it — the caller must be told
	// so it can keep a backup instead of losing it silently.
	local := canonicalManual + "\n<!-- an instance note with no heading of its own -->\n"
	if !MergeDropsLocalContent(canonicalManual, local) {
		t.Error("a note inside an engine-owned section body must be reported as dropped")
	}

	withOwnSection := canonicalManual + "\n## Instance-only section\n\nkept verbatim\n"
	if MergeDropsLocalContent(canonicalManual, withOwnSection) {
		t.Error("a section the engine does not know is appended, never dropped")
	}

	if MergeDropsLocalContent(canonicalManual, canonicalManual) {
		t.Error("an untouched manual drops nothing")
	}
}

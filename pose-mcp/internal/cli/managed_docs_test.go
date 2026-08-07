package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

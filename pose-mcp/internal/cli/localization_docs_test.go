package cli

// Localization and documentation contract (spec pose-localization-docs-contract):
// every documented `pose <command>` invocation must be recognized by the
// real CLI dispatcher (R1), and an unsupported --locale falls back to
// English cleanly rather than a partial mix (R2, the fallback half —
// the parity half is TestEditorialDefaultsAreEnglishAndPtBROverlayIsComplete
// in internal/scaffold and the templates assertion in
// TestNativeScaffoldsCreateContractArtifacts).

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// dispatchedCommands derives the set of recognized top-level subcommands
// directly from cli.go's own switch statement — not a hand-maintained
// duplicate list, so it can never itself drift from what Main() accepts.
func dispatchedCommands(t *testing.T) map[string]bool {
	t.Helper()
	src, err := os.ReadFile("cli.go")
	if err != nil {
		t.Skipf("cannot read cli.go for self-inspection: %v", err)
	}
	caseRE := regexp.MustCompile(`(?m)^\tcase ((?:"[a-zA-Z0-9_-]+", *)*"[a-zA-Z0-9_-]+"):`)
	literalRE := regexp.MustCompile(`"([a-zA-Z0-9_-]+)"`)
	commands := map[string]bool{}
	for _, m := range caseRE.FindAllStringSubmatch(string(src), -1) {
		for _, lit := range literalRE.FindAllStringSubmatch(m[1], -1) {
			commands[lit[1]] = true
		}
	}
	if len(commands) < 20 {
		t.Fatalf("self-inspection found suspiciously few commands (%d) — regex drifted from cli.go's shape", len(commands))
	}
	return commands
}

// extractDocumentedCommands scans a doc file for `pose <word>` mentions
// (fenced-block examples and inline code alike) and returns the candidate
// subcommand tokens, deduplicated.
func extractDocumentedCommands(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`\bpose[ \t]+([a-z][a-z0-9-]*)\b`)
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(string(content), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

func TestDocumentedCommandsAreRecognizedByTheCLI(t *testing.T) {
	repoRoot, err := repoRootForTest()
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	commands := dispatchedCommands(t)
	// A few English prose words legitimately follow "pose " without being a
	// subcommand (product references, not invocations); not real drift.
	allow := map[string]bool{"cli": true, "mcp": true, "binary": true, "distribution": true, "installs": true, "config": true, "native": true}

	docs := []string{filepath.Join(repoRoot, "README.md")}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "docs-site", "docs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			docs = append(docs, filepath.Join(repoRoot, "docs-site", "docs", e.Name()))
		}
	}

	checked := 0
	for _, doc := range docs {
		for _, cmd := range extractDocumentedCommands(t, doc) {
			if allow[cmd] {
				continue
			}
			checked++
			if !commands[cmd] {
				t.Errorf("%s documents `pose %s`, which is not a command the CLI recognizes", filepath.Base(doc), cmd)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no documented `pose <command>` invocations were found — extraction regex likely drifted from the docs")
	}
}

// docTypeRE matches the Diátaxis classification line every docs-site page
// carries right after its H1 (spec pose-localization-docs-contract, R3):
// "**Doc type:** <Tutorial|How-to|Reference|Explanation> · **Applies to:** POSE ..."
var docTypeRE = regexp.MustCompile(`\*\*Doc type:\*\* (Tutorial|How-to|Reference|Explanation) .+\*\*Applies to:\*\* POSE`)

func TestDocsAreDiataxisClassifiedWithVersionApplicability(t *testing.T) {
	repoRoot, err := repoRootForTest()
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	dir := filepath.Join(repoRoot, "docs-site", "docs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		checked++
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !docTypeRE.Match(content) {
			t.Errorf("%s is missing a Diátaxis doc-type + version-applicability line (Tutorial|How-to|Reference|Explanation)", e.Name())
		}
	}
	if checked == 0 {
		t.Fatal("no docs-site pages found")
	}
}

// sudoInExampleRE flags a documented command requiring elevated privileges
// — POSE's own design principle is that no command needs it; a `sudo`
// appearing in a copyable example is a real permissions smell, not
// something a reader should paste without noticing.
var sudoInExampleRE = regexp.MustCompile(`\bsudo\b`)

// TestDocsHaveNoUnsafeOrSecretShapedExamples is the Security requirement's
// deterministic scan (secrets, unsafe downloads, permissions) reusing the
// exact same offline patterns pose-agent-skills-conformance already
// applies to skill content (unsafeSkillPatterns, secretLikePatterns) —
// docs are prose an author can paste an unsafe example into just as
// easily as a skill file.
func TestDocsHaveNoUnsafeOrSecretShapedExamples(t *testing.T) {
	repoRoot, err := repoRootForTest()
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	docs := []string{filepath.Join(repoRoot, "README.md")}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "docs-site", "docs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			docs = append(docs, filepath.Join(repoRoot, "docs-site", "docs", e.Name()))
		}
	}
	checked := 0
	for _, doc := range docs {
		checked++
		content, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range unsafeSkillPatterns {
			if re.Match(content) {
				t.Errorf("%s: matches unsafe pattern %s", filepath.Base(doc), re.String())
			}
		}
		for _, re := range secretLikePatterns {
			if re.Match(content) {
				t.Errorf("%s: matches secret-shaped pattern %s", filepath.Base(doc), re.String())
			}
		}
		if sudoInExampleRE.Match(content) {
			t.Errorf("%s: documents a command requiring sudo — POSE never needs elevated privileges", filepath.Base(doc))
		}
	}
	if checked == 0 {
		t.Fatal("no docs were scanned")
	}
}

func TestInstallFallsBackToEnglishWhenLocaleUnsupported(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp", "--locale", "xx-ZZ"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	if !strings.Contains(out.String(), "not available") {
		t.Errorf("expected a fallback notice: %s", out.String())
	}
	for _, rel := range []string{
		filepath.Join("AGENTS.md"),
		filepath.Join(".pose", "templates", "knowledge.md"),
		filepath.Join(".agents", "skills", "pose-feature", "SKILL.md"),
	} {
		b, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		if hasPortugueseAccent(string(b)) {
			t.Errorf("%s: unsupported locale must fall back entirely to English, found non-English content", rel)
		}
	}
}

// TestInstallReinstallDetectsExistingLocaleWithoutFlag pins the fix for
// github.com/oseiaspereira88/pose#18: rerunning `pose install <target>` on
// an already pt-BR-localized instance without --locale must preserve pt-BR
// (detected from the existing POSE.md), not silently revert to English.
// This is also what pose update --force exercises, since its force branch
// delegates entirely to cmdInstall.
func TestInstallReinstallDetectsExistingLocaleWithoutFlag(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp", "--locale", "pt-BR"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	agents := filepath.Join(repo, "AGENTS.md")
	before, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPortugueseAccent(string(before)) {
		t.Fatalf("fixture did not install pt-BR content: %s", before)
	}

	out.Reset()
	errB.Reset()
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	after, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPortugueseAccent(string(after)) {
		t.Errorf("reinstall without --locale reverted an already pt-BR-localized project to English:\n%s", after)
	}
}

// TestInstallFreshWithoutLocaleStillDefaultsToEnglish pins R2: a target with
// no pre-existing POSE.md has nothing to detect a locale from, so it must
// keep defaulting to English when --locale is omitted.
func TestInstallFreshWithoutLocaleStillDefaultsToEnglish(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	b, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if hasPortugueseAccent(string(b)) {
		t.Errorf("fresh install without --locale must default to English, found non-English content: %s", b)
	}
}

// TestInstallExplicitLocaleOverridesDetection pins R3: an explicit --locale
// always wins over detection, including the edge case --locale en on an
// already pt-BR-localized instance — the case where a naive fix (always
// calling the detector unconditionally) would silently ignore the explicit
// flag, since resolveDocLocale's own short-circuit only special-cases
// non-en preferred values. Uses --force on the second call: AGENTS.md/
// POSE.md merge by heading (MergeManagedDoc) when unforced, and a
// Portuguese heading never matches its English counterpart, so a plain
// merge appends the old section instead of replacing it — a real, separate,
// pre-existing limitation of the merge algorithm, not of locale resolution.
// --force skips the merge and overwrites wholesale, which is what isolates
// the assertion to what this fix actually controls: which locale's content
// deliverMachinery/the docs step reads from.
func TestInstallExplicitLocaleOverridesDetection(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp", "--locale", "pt-BR"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	out.Reset()
	errB.Reset()
	if code := cmdInstall([]string{repo, "--skip-mcp", "--force", "--locale", "en"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall exit=%d out=%s err=%s", code, out.String(), errB.String())
	}
	b, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if hasPortugueseAccent(string(b)) {
		t.Errorf("explicit --locale en did not override detection on an already pt-BR-localized instance: %s", b)
	}
}

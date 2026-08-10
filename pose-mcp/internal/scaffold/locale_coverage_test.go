package scaffold

// Locale coverage (spec pose-locale-coverage-contract).
//
// Parity was extended three times, each after a human noticed by reading:
// manuals, then skill entries, then the skill index. Every time the fix guarded
// that one document and left the next one uncovered — a 24-file blind spot
// across rules, workflows and templates, reported by a user rather than caught.
//
// So the unit of coverage is no longer a document but the locale tree: every
// translated file must have an English source and a declared comparison. A new
// translated document is guarded by default, or this fails because it does not
// know how to compare it.
//
// The comparisons are deliberately structural. Prose differs by language and
// the skills differ in shape by design (terse English, example-rich pt-BR), so
// asserting sameness of text would be noise. What must hold is that both sides
// describe the same thing: the same heading tree, and the same POSE commands.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// comparison declares how a translated file is checked against its source.
type comparison int

const (
	// structural: same heading tree, same POSE commands taught.
	structural comparison = iota
	// commandsOnly: heading trees legitimately differ; commands must not.
	commandsOnly
	// presenceOnly: a routing table checked by its own dedicated test.
	presenceOnly
)

// localeComparisons maps a repository-relative path to how it is compared.
// A translated file absent from this map fails: coverage is declared, never
// assumed.
var localeComparisons = map[string]comparison{
	"POSE.md":                  structural,
	"AGENTS.md":                structural,
	".agents/skills/README.md": presenceOnly,
}

func init() {
	// Skills keep their shape difference; only the commands must match.
	for _, slug := range []string{
		"pose-adr", "pose-bugfix", "pose-doc-update", "pose-feature",
		"pose-knowledge", "pose-recurrence-escalation", "pose-release-closeout",
		"pose-review", "pose-spec-closeout", "pose-surface-closeout",
		"pose-test-plan",
	} {
		localeComparisons[".agents/skills/"+slug+"/SKILL.md"] = commandsOnly
	}
	// Rules, workflows and templates are translations of the same document and
	// must keep the same structure.
	for _, p := range []string{
		".pose/rules/_base-recurrence.md", ".pose/rules/backend-go.md",
		".pose/rules/delivery-evidence.md", ".pose/rules/delivery-surface.md",
		".pose/rules/documentation-style.md", ".pose/rules/frontend-react.md",
		".pose/rules/knowledge-governance.md",
		".pose/rules/release-integrity.md", ".pose/rules/security.md",
		".pose/templates/changelog-fragment.md", ".pose/templates/doc-audit-report.md",
		".pose/templates/knowledge.md", ".pose/templates/review.md",
		".pose/templates/roadmap.md", ".pose/templates/spec.md",
		".pose/workflows/bugfix.md", ".pose/workflows/documentation-update.md",
		".pose/workflows/feature.md", ".pose/workflows/recurrence-escalation.md",
		".pose/workflows/refactor.md", ".pose/workflows/release.md",
		".pose/workflows/review.md", ".pose/workflows/ui-surface.md",
	} {
		localeComparisons[p] = structural
	}
}

var headingLevelRe = regexp.MustCompile(`(?m)^(#{1,6}) `)

func headingLevels(content string) []string {
	m := headingLevelRe.FindAllStringSubmatch(content, -1)
	out := make([]string, 0, len(m))
	for _, h := range m {
		out = append(out, h[1])
	}
	return out
}

func TestLocaleCoverage(t *testing.T) {
	root := poseDistDir(t)
	localesDir := filepath.Join(root, "locales")
	locales, err := os.ReadDir(localesDir)
	if err != nil {
		t.Fatalf("reading locales/: %v", err)
	}

	checked := 0
	for _, loc := range locales {
		if !loc.IsDir() {
			continue
		}
		base := filepath.Join(localesDir, loc.Name())
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return err
			}
			rel, _ := filepath.Rel(base, path)
			rel = filepath.ToSlash(rel)

			source := filepath.Join(root, rel)
			if _, statErr := os.Stat(source); statErr != nil {
				t.Errorf("%s/%s has no English source at %s — a translation of nothing", loc.Name(), rel, rel)
				return nil
			}
			how, declared := localeComparisons[rel]
			if !declared {
				t.Errorf("%s/%s has no declared comparison — add it to localeComparisons so it is guarded, rather than leaving it unchecked",
					loc.Name(), rel)
				return nil
			}
			checked++

			src := readManual(t, source)
			tgt := readManual(t, path)

			if how == structural {
				a, b := headingLevels(src), headingLevels(tgt)
				if !equalShape(a, b) {
					t.Errorf("%s/%s: heading tree differs from the source (%d vs %d headings) — the translation has drifted structurally",
						loc.Name(), rel, len(a), len(b))
				}
			}
			if how == structural || how == commandsOnly {
				s, g := taughtCommands(src), taughtCommands(tgt)
				if missing := diffTokens(s, g); len(missing) > 0 {
					t.Errorf("%s/%s: source teaches %d POSE command(s) the translation does not: %s",
						loc.Name(), rel, len(missing), strings.Join(capTokens(missing), ", "))
				}
				if extra := diffTokens(g, s); len(extra) > 0 {
					t.Errorf("%s/%s: translation teaches %d POSE command(s) the source does not: %s",
						loc.Name(), rel, len(extra), strings.Join(capTokens(extra), ", "))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", base, err)
		}
	}
	if checked == 0 {
		t.Error("no translated file was compared — the discovery is broken, not the translations")
	}

	// A declaration for a file that no longer exists hides shrinking coverage.
	for rel := range localeComparisons {
		found := false
		for _, loc := range locales {
			if !loc.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(localesDir, loc.Name(), rel)); err == nil {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("localeComparisons declares %s but no locale translates it — remove the declaration", rel)
		}
	}
}

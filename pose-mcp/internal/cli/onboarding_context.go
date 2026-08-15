package cli

// Brownfield onboarding context extraction (spec
// pose-onboarding-context-extraction): populates AGENTS.md's "Project
// context" placeholder from a target repository's own README.md/CLAUDE.md
// instead of leaving it for the operator to fill by hand.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// maxProjectContextChars bounds the extracted summary to roughly the
// "3-6 lines" the canonical placeholder asks a human to write by hand —
// long enough to be useful, short enough to stay a summary rather than a
// copy of the source file.
const maxProjectContextChars = 600

var (
	mdImageOrBadgeLine = regexp.MustCompile(`^(\[!\[.*\]\(.*\)\]\(.*\)|!\[.*\]\(.*\))$`)
	mdHeadingLine      = regexp.MustCompile(`^#{1,6}\s+(.*)$`)
)

// extractProjectContext reads the target repository's own README.md and
// CLAUDE.md, when present, and returns a conservative plain-text summary
// suitable for AGENTS.md's "Project context" section: the first heading and
// first real paragraph of each, excerpted verbatim rather than summarized
// or generated. Returns "" when neither file exists or neither yields usable
// prose (e.g. a README that is only badges/images), so callers can fall
// back to the unfilled placeholder unchanged.
func extractProjectContext(root string) string {
	readme := excerptMarkdown(filepath.Join(root, "README.md"))
	claude := excerptMarkdown(filepath.Join(root, "CLAUDE.md"))
	switch {
	case readme == "" && claude == "":
		return ""
	case readme == "":
		return truncateSummary(claude, maxProjectContextChars)
	case claude == "" || claude == readme:
		return truncateSummary(readme, maxProjectContextChars)
	default:
		return truncateSummary(readme+" "+claude, maxProjectContextChars)
	}
}

// excerptMarkdown extracts a Markdown file's first heading and the first
// paragraph following it (or, absent a heading, the document's first
// paragraph) as flowing plain text. Badge/image-only lines, HTML comments
// and horizontal rules are skipped rather than excerpted. Returns "" when
// the file is absent or has no usable prose.
func excerptMarkdown(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	i := 0
	skipNoise := func() {
		for i < len(lines) {
			line := strings.TrimSpace(lines[i])
			switch {
			case line == "":
				i++
			case mdImageOrBadgeLine.MatchString(line):
				i++
			case line == "---" || line == "***" || line == "___":
				i++
			case strings.HasPrefix(line, "<!--"):
				for i < len(lines) && !strings.Contains(lines[i], "-->") {
					i++
				}
				i++
			default:
				return
			}
		}
	}
	skipNoise()

	var title string
	if i < len(lines) {
		if m := mdHeadingLine.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil {
			title = strings.TrimSpace(m[1])
			i++
			skipNoise()
		}
	}

	var paragraph []string
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || mdHeadingLine.MatchString(line) || strings.HasPrefix(line, "```") {
			break
		}
		if mdImageOrBadgeLine.MatchString(line) {
			i++
			continue
		}
		paragraph = append(paragraph, line)
		i++
	}
	body := strings.Join(paragraph, " ")

	switch {
	case title != "" && body != "":
		return title + ": " + body
	case title != "":
		return title
	default:
		return body
	}
}

// truncateSummary caps s at max characters, cutting on a word boundary
// rather than mid-word, and marks a cut with a trailing ellipsis so the
// result never silently reads as complete when it isn't.
func truncateSummary(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if idx := strings.LastIndexByte(cut, ' '); idx > 0 {
		cut = cut[:idx]
	}
	return strings.TrimSpace(cut) + "…"
}

// projectContextPlaceholderPrefix is the literal line prefix every shipped
// AGENTS.md translation uses for its "Project context" placeholder — the
// description after the colon is the only part that varies by locale, so
// matching on this prefix lets injectExtractedProjectContext work without
// knowing which locale it was handed.
const projectContextPlaceholderPrefix = "{{PROJECT_NAME}}: "

// injectExtractedProjectContext replaces AGENTS.md's generic
// "{{PROJECT_NAME}}: describe..." placeholder line with a summary excerpted
// from the target repository's own README.md/CLAUDE.md, when one exists.
// Must run before the {{PROJECT_NAME}} token itself is substituted, so the
// injected summary still receives the real project name via the same
// replacement pass. A target with neither file, or with only boilerplate
// (badges/images), returns content unchanged — a fresh non-brownfield
// repository sees no behavior change.
func injectExtractedProjectContext(content, target string) string {
	summary := extractProjectContext(target)
	if summary == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, projectContextPlaceholderPrefix) {
			lines[i] = projectContextPlaceholderPrefix + summary
			return strings.Join(lines, "\n")
		}
	}
	return content
}

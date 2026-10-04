package pose

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SpecFollowup is one follow-up bullet of a spec's Final Report. The parser
// lives here so the CLI backlog and the obligation projection read follow-ups
// the same way (spec pose-obligation-projection); `pose followups` adds its
// synthesized sources on top.
type SpecFollowup struct {
	Spec           string
	SpecStatus     string
	Path           string
	Ordinal        int
	RawDisposition string
	Target         string
	Text           string
	Owner          string
	Criticality    string
	Review         string
	By             string
	MetaErr        string
}

var (
	followupBulletRE      = regexp.MustCompile(`^\s*-\s+(.*\S)\s*$`)
	followupDispositionRE = regexp.MustCompile(`^\[\s*([a-z-]+)(?:\s*:\s*([^\]]+))?\s*\]\s*(.*)$`)
	followupHTMLCommentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
	FollowupMetaGroupRE   = regexp.MustCompile(`\(([^()]*\bowner:[^()]*)\)\s*$`)
	FollowupReviewDateRE  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	followupCriticalities = map[string]bool{"low": true, "medium": true, "high": true}
)

// ParseFollowupMeta extracts the trailing ownership group. metaErr describes
// a malformed group ("" when valid or absent); an absent group reads as
// unowned.
func ParseFollowupMeta(text string) (stripped, owner, crit, review, by, metaErr string) {
	m := FollowupMetaGroupRE.FindStringSubmatchIndex(text)
	if m == nil {
		return strings.TrimSpace(text), "unowned", "", "", "", ""
	}
	group := text[m[2]:m[3]]
	stripped = strings.TrimSpace(text[:m[0]])
	owner = "unowned"
	for _, field := range strings.Fields(group) {
		key, value, ok := strings.Cut(field, ":")
		if !ok || value == "" {
			return stripped, owner, crit, review, by, "malformed ownership field '" + field + "' (use key:value)"
		}
		switch key {
		case "owner":
			owner = value
		case "crit":
			if !followupCriticalities[value] {
				return stripped, owner, crit, review, by, "invalid crit '" + value + "' (use low|medium|high)"
			}
			crit = value
		case "review":
			if !FollowupReviewDateRE.MatchString(value) {
				return stripped, owner, crit, review, by, "invalid review date '" + value + "' (use YYYY-MM-DD)"
			}
			review = value
		case "by":
			by = value
		default:
			return stripped, owner, crit, review, by, "unknown ownership field '" + key + "' (use owner|crit|review|by)"
		}
	}
	if crit == "" || review == "" {
		return stripped, owner, crit, review, by, "incomplete ownership group (declare owner, crit and review together)"
	}
	return stripped, owner, crit, review, by, ""
}

// ParseSpecFollowups reads every spec's Final Report follow-ups, flat and
// folder layouts alike, in path order.
func ParseSpecFollowups(root string) []SpecFollowup {
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*", "spec.md"))
	flatPaths, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*.md"))
	for _, p := range flatPaths {
		if !strings.EqualFold(filepath.Base(p), "README.md") {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	entries := []SpecFollowup{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		body := followupHTMLCommentRE.ReplaceAllString(string(raw), "")
		fm, _ := SplitFrontmatter(string(raw))
		specSlug := strings.TrimSpace(fm["slug"])
		if specSlug == "" {
			if filepath.Base(path) == "spec.md" {
				specSlug = filepath.Base(filepath.Dir(path))
			} else {
				specSlug = strings.TrimSuffix(filepath.Base(path), ".md")
			}
		}
		status := followupFrontmatterStatus(body)
		inFinal, inFollowups := false, false
		ordinal := 0
		// A follow-up may wrap across lines: everything until the next bullet,
		// heading or blank line belongs to it (spec
		// pose-release-cycle-debt-closure, R4).
		var pending string
		flush := func() {
			if pending == "" {
				return
			}
			text, disposition, target := pending, "", ""
			if parsed := followupDispositionRE.FindStringSubmatch(text); parsed != nil {
				disposition, target, text = parsed[1], strings.TrimSpace(parsed[2]), strings.TrimSpace(parsed[3])
			}
			stripped, owner, crit, review, by, metaErr := ParseFollowupMeta(text)
			if stripped != "" {
				ordinal++
				entries = append(entries, SpecFollowup{
					Spec: specSlug, SpecStatus: status, Path: path, Ordinal: ordinal,
					RawDisposition: disposition, Target: target, Text: stripped,
					Owner: owner, Criticality: crit, Review: review, By: by, MetaErr: metaErr,
				})
			}
			pending = ""
		}
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "## ") {
				flush()
				heading := strings.TrimSpace(strings.TrimLeft(line, "#0123456789. "))
				inFinal = strings.HasPrefix(strings.ToLower(heading), "final report")
				inFollowups = false
				continue
			}
			if inFinal && strings.HasPrefix(line, "### ") {
				flush()
				inFollowups = strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "###"))), "follow-up")
				continue
			}
			if !inFollowups {
				continue
			}
			if match := followupBulletRE.FindStringSubmatch(line); match != nil {
				flush()
				pending = match[1]
				continue
			}
			if pending != "" {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					pending += " " + trimmed
				} else {
					flush()
				}
			}
		}
		flush()
	}
	return entries
}

func followupFrontmatterStatus(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "unset"
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(line, "status:") {
			return strings.TrimSpace(strings.SplitN(strings.TrimPrefix(line, "status:"), "#", 2)[0])
		}
	}
	return "unset"
}

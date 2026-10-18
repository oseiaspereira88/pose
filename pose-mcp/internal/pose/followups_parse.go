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
	// DispositionErr is set when the bullet carries a [disposition] this engine does not
	// know: it is read as unresolved, with this diagnostic.
	DispositionErr string
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

// SpecFollowupRead is what reading the follow-ups of every spec found: the items,
// and what the read could not see (spec pose-source-coverage-integrity). An empty
// Items with losses is not "nothing is owed".
type SpecFollowupRead struct {
	Items    []SpecFollowup
	Coverage SourceCoverage
}

// KnownFollowupDispositions are the dispositions a follow-up can carry. One that is not
// here is not a resolution: it is a typo, and the follow-up still owes one.
var KnownFollowupDispositions = map[string]bool{"open": true, "spawned": true, "covered": true, "duplicate": true, "done": true, "wont-do": true}

// FollowupUnresolved is the one definition of "still owes a disposition": none at all (the
// legacy format), an explicit open, or one this engine does not know.
func FollowupUnresolved(disposition string) bool {
	return disposition == "" || disposition == "open" || !KnownFollowupDispositions[disposition]
}

// Unresolved reports a follow-up that still owes a disposition. One definition, so the
// backlog, the candidates and the obligation projection count the same set.
func (f SpecFollowup) Unresolved() bool { return FollowupUnresolved(f.RawDisposition) }

// ParseSpecFollowups reads every spec's Final Report follow-ups, flat, folder and
// split layouts alike, in path order. It drops what it cannot read; callers that
// must not mistake that for "nothing is owed" use ReadSpecFollowups.
func ParseSpecFollowups(root string) []SpecFollowup {
	return ReadSpecFollowups(root).Items
}

// ReadSpecFollowups reads every spec's Final Report follow-ups, in the flat, folder and
// legacy split layouts the spec listing reads, in path order, and reports each spec it
// could not read or interpret, attributed to the spec its path names.
func ReadSpecFollowups(root string) SpecFollowupRead {
	coverage := SourceCoverage{Losses: []SourceLoss{}}
	entries := []SpecFollowup{}
	specsDir := filepath.Join(root, ".pose", "specs")
	rel := func(path string) string {
		r, err := filepath.Rel(root, path)
		if err != nil {
			return path
		}
		return filepath.ToSlash(r)
	}
	lose := func(path, slug, kind, detail string) {
		coverage.Add(SourceLoss{Producer: "followups", Artifact: rel(path), Scope: slug, AttributedBy: "path", Kind: kind, Detail: detail})
	}
	dirEntries, err := os.ReadDir(specsDir)
	if err != nil {
		// Absent is only absent when nothing is there: a link to nowhere reads as
		// "does not exist" and is a lost directory.
		if _, lerr := os.Lstat(specsDir); !os.IsNotExist(err) || lerr == nil {
			coverage.Add(SourceLoss{Producer: "followups", Artifact: ".pose/specs", Kind: LossReadError, Detail: err.Error()})
		}
		coverage.Normalize()
		return SpecFollowupRead{Items: entries, Coverage: coverage}
	}
	type source struct {
		path, slug, text, status string
		ok                       bool
	}
	var sources []source
	for _, e := range dirEntries {
		path := filepath.Join(specsDir, e.Name())
		switch {
		case e.IsDir() || (e.Type()&os.ModeSymlink != 0 && isDirectory(path)):
			slug := specSlugFromPath(path + "/spec.md")
			specPath := filepath.Join(path, "spec.md")
			// Presence is decided by Lstat: an entry that is there but cannot be resolved
			// (a dangling link) is a spec that was lost, not one that does not exist.
			if _, lerr := os.Lstat(specPath); lerr == nil {
				raw, readErr := os.ReadFile(specPath)
				if readErr != nil {
					lose(specPath, slug, LossReadError, readErr.Error())
					continue
				}
				sources = append(sources, source{path: specPath, slug: slug, text: string(raw), ok: true})
				continue
			} else if !os.IsNotExist(lerr) {
				lose(specPath, slug, LossReadError, lerr.Error())
				continue
			}
			// The legacy split layout the spec listing reads: the Final Report lives in
			// one of the section files, and the folder is the spec.
			var files []string
			broken := false
			for _, name := range splitSpecSectionFiles {
				section := filepath.Join(path, name)
				if _, lerr := os.Lstat(section); lerr != nil {
					if !os.IsNotExist(lerr) {
						lose(section, slug, LossReadError, lerr.Error())
						broken = true
					}
					continue
				}
				if fi, serr := os.Stat(section); serr != nil {
					lose(section, slug, LossReadError, serr.Error())
					broken = true
				} else if !fi.IsDir() {
					files = append(files, section)
				}
			}
			if len(files) > 0 {
				body, readErr := readSplitSpecBody(files)
				if readErr != nil {
					lose(path, slug, LossReadError, readErr.Error())
					continue
				}
				sources = append(sources, source{path: path, slug: slug, text: body, status: splitSpecStatus(files), ok: true})
			} else if _, listErr := os.ReadDir(path); listErr != nil && !broken {
				// A folder this process cannot list may hold a spec: absence is not proven.
				lose(path, slug, LossReadError, listErr.Error())
			}
		case e.Type()&os.ModeSymlink != 0 && !strings.HasSuffix(e.Name(), ".md"):
			// A link that stood for a spec folder and no longer resolves: it is neither
			// a directory nor a flat spec, and it is not absent.
			if _, statErr := os.Stat(path); statErr != nil {
				lose(path, specSlugFromPath(path+"/spec.md"), LossReadError, statErr.Error())
			}
		case strings.HasSuffix(e.Name(), ".md") && !strings.EqualFold(e.Name(), "README.md"):
			slug := specSlugFromPath(path)
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				lose(path, slug, LossReadError, readErr.Error())
				continue
			}
			sources = append(sources, source{path: path, slug: slug, text: string(raw), ok: true})
		}
	}
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].path < sources[j].path })
	for _, src := range sources {
		text := src.text
		if first, _, _ := strings.Cut(text, "\n"); strings.TrimSpace(first) == "---" && !frontmatterClosed(text) {
			// The header never ends, so the status and the slug are unknown and the body
			// is not where the spec says it is.
			lose(src.path, src.slug, LossParseError, "the frontmatter is not closed")
		}
		body := followupHTMLCommentRE.ReplaceAllString(text, "")
		fm, _ := SplitFrontmatter(text)
		specSlug := strings.TrimSpace(fm["slug"])
		if specSlug == "" {
			specSlug = src.slug
		}
		status := src.status
		if status == "" {
			status = followupFrontmatterStatus(body)
		}
		path := src.path
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
			if stripped == "" && status == "done" {
				// A bullet that names a disposition and no follow-up, in a spec that claims to
				// be finished, is an item cut short or emptied: it is reported, not dropped, and
				// no description is invented for it. (The template's "- [open]" placeholder in
				// a spec still being written is expected and says nothing.)
				lose(path, specSlug, LossInvalid, "a follow-up bullet has no description")
			}
			if stripped != "" {
				ordinal++
				dispositionErr := ""
				if disposition != "" && !KnownFollowupDispositions[disposition] {
					dispositionErr = "unknown disposition '" + disposition + "' (use open|spawned|covered|duplicate|done|wont-do); it is not a resolution"
				}
				entries = append(entries, SpecFollowup{
					Spec: specSlug, SpecStatus: status, Path: path, Ordinal: ordinal,
					RawDisposition: disposition, Target: target, Text: stripped,
					Owner: owner, Criticality: crit, Review: review, By: by, MetaErr: metaErr, DispositionErr: dispositionErr,
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
	coverage.Normalize()
	return SpecFollowupRead{Items: entries, Coverage: coverage}
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// frontmatterClosed reports that a text that opens a header closes it.
func frontmatterClosed(text string) bool {
	_, rest, _ := strings.Cut(text, "\n")
	for rest != "" {
		line, tail, _ := strings.Cut(rest, "\n")
		if strings.TrimSpace(line) == "---" {
			return true
		}
		rest = tail
	}
	return false
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

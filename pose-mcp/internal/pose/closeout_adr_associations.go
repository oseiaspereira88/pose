package pose

// ADR associations of a roadmap, for the when-justified roadmap rule of the
// closeout review evaluator (spec pose-closeout-review-policy-by-cost-benefit).
//
// An association is explicit, never inferred from prose:
//   - forward: the roadmap's `adrs: [adr:<file>.md]` frontmatter, or a local
//     Markdown link from its body to a file in .pose/adr/;
//   - backward: an ADR's `roadmaps: [roadmap:<slug>]` (or the qualified
//     `xref:<project>/roadmap:<slug>`) frontmatter, or a local Markdown link
//     from the ADR to .pose/roadmaps/<slug>.md.
//
// Only the owning project's .pose/adr/ is inventoried; nothing searches other
// projects. A malformed declaration, a link to a missing file or an
// incomplete inventory is unknown, which can never prove absence.

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	closeoutADRInventoryLimit = 4096
	closeoutADRSizeLimit      = 1 << 20
)

var (
	// [text](dest "title") and [text](<dest>) inline links.
	markdownInlineLinkRE = regexp.MustCompile(`\]\(\s*(<[^>\n]*>|[^)\s]+)(?:\s+(?:"[^"]*"|'[^']*'|\([^)]*\)))?\s*\)`)
	// [label]: dest "title" reference definitions.
	markdownReferenceLinkRE = regexp.MustCompile(`(?m)^\s{0,3}\[[^\]\n]+\]:\s*(<[^>\n]*>|\S+)`)
)

func (s Store) adrDir() string { return filepath.Join(s.Root, ".pose", "adr") }

// markdownLinkDestinations lists the link destinations of a Markdown body,
// angle brackets and fragments removed. External URLs are kept out.
func markdownLinkDestinations(body string) ([]string, []string) {
	// Code and comments are read like prose: a link shown as an example
	// still counts, so the reader over-enforces rather than dropping a real
	// link it misparsed (review round 14, N46; follow-up F6).
	out, undecodable := []string{}, []string{}
	add := func(raw string) {
		dest := strings.TrimSpace(raw)
		dest = strings.TrimSuffix(strings.TrimPrefix(dest, "<"), ">")
		// The destination as Markdown resolves it: character references and
		// backslash escapes first, then the fragment and query, then URL
		// escapes. One that does not decode is reported, never skipped
		// (review round 15, N48).
		dest = markdownBackslashEscapeRE.ReplaceAllString(html.UnescapeString(dest), "$1")
		if i := strings.IndexAny(dest, "#?"); i >= 0 {
			dest = dest[:i]
		}
		decoded, err := url.PathUnescape(dest)
		if err != nil {
			undecodable = append(undecodable, strings.TrimSpace(raw))
			return
		}
		dest = decoded
		if dest == "" || strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") || strings.HasPrefix(dest, "/") {
			return
		}
		out = append(out, dest)
	}
	for _, m := range markdownInlineLinkRE.FindAllStringSubmatch(body, -1) {
		add(m[1])
	}
	for _, m := range markdownReferenceLinkRE.FindAllStringSubmatch(body, -1) {
		add(m[1])
	}
	return out, undecodable
}

// markdownBackslashEscapeRE is a CommonMark backslash escape of ASCII
// punctuation.
var markdownBackslashEscapeRE = regexp.MustCompile(`\\([!-/:-@\[-` + "`" + `{-~])`)

// associationList reads one association key (adrs: on a roadmap, roadmaps: on
// an ADR) from the raw frontmatter of a governance file, strictly: a key
// declared more than once, declared with no value, outside a [list], or with
// an empty item is reported, never read as an absent or empty association. An
// explicit [] declares none (review round 12, N39).
func associationList(raw, key, owner string) ([]string, bool, string) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false, ""
	}
	values := []string{}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		// The key and value as SplitFrontmatter reads them: the key trimmed,
		// a template comment removed (review round 13, N45).
		if name, value, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(name) == key {
			values = append(values, cleanValue(value))
		}
	}
	switch {
	case len(values) == 0:
		return nil, false, ""
	case len(values) > 1:
		return nil, true, fmt.Sprintf("%s declares %s: %d times; one declaration is required", owner, key, len(values))
	}
	value := values[0]
	if value == "" {
		return nil, true, fmt.Sprintf("%s declares %s: with no value; declare [] for none", owner, key)
	}
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, true, fmt.Sprintf("%s declares %s: outside a [list]", owner, key)
	}
	inner := strings.TrimSpace(value[1 : len(value)-1])
	if inner == "" {
		return []string{}, true, ""
	}
	items := []string{}
	for _, item := range strings.Split(inner, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, true, fmt.Sprintf("%s declares %s: with an empty item", owner, key)
		}
		items = append(items, item)
	}
	return items, true, ""
}

// validADRBasename accepts exactly a file name ending in .md directly inside
// .pose/adr/: no directory, traversal, hidden file or URI.
func validADRBasename(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.ContainsAny(name, `/\:`) &&
		!strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".md") && name != ".md"
}

// regularADR reports a regular, non-symlinked file of .pose/adr/ within the root.
func (s Store) regularADR(name string) error {
	path := filepath.Join(s.adrDir(), name)
	info, err := s.fsys().Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !artifactPathWithin(s.fsys(), s.Root, path) {
		return fmt.Errorf("not a regular file inside .pose/adr")
	}
	return nil
}

// closeoutADRAssociations returns the sorted ADR basenames associated with a
// roadmap, and the diagnostics that make the answer unknown.
func (s Store) closeoutADRAssociations(rm *Roadmap, project string) ([]string, []string) {
	found := map[string]bool{}
	diagnostics := []string{}
	rawRoadmap, readErr := s.fsys().ReadFile(rm.Path)
	if readErr != nil {
		diagnostics = append(diagnostics, "roadmap "+rm.Slug+" could not be read for its adrs: "+readErr.Error())
	}
	declared, _, declarationProblem := associationList(string(rawRoadmap), "adrs", "roadmap "+rm.Slug)
	if declarationProblem != "" {
		diagnostics = append(diagnostics, declarationProblem)
	}
	for _, item := range declared {
		name, ok := strings.CutPrefix(item, "adr:")
		if !ok || !validADRBasename(name) {
			diagnostics = append(diagnostics, fmt.Sprintf("roadmap %s declares %q, not adr:<file>.md", rm.Slug, item))
			continue
		}
		if err := s.regularADR(name); err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("roadmap %s declares adr:%s, which cannot be read: %v", rm.Slug, name, err))
			continue
		}
		found[name] = true
	}
	body := rm.Body
	if body == "" {
		if full, err := s.GetRoadmap(rm.Slug); err == nil {
			body = full.Body
		}
	}
	roadmapDir := filepath.Dir(rm.Path)
	links, undecodable := markdownLinkDestinations(body)
	for _, dest := range undecodable {
		diagnostics = append(diagnostics, fmt.Sprintf("roadmap %s links to %s, whose escapes do not decode; the association reader cannot tell where it points", rm.Slug, dest))
	}
	for _, dest := range links {
		target := filepath.Clean(filepath.Join(roadmapDir, filepath.FromSlash(dest)))
		if filepath.Dir(target) == filepath.Clean(s.adrDir()) && strings.ContainsAny(dest, "()") {
			diagnostics = append(diagnostics, fmt.Sprintf("roadmap %s links to %s, a destination with parentheses the association reader does not read whole; rename the ADR or the link", rm.Slug, dest))
			continue
		}
		if filepath.Dir(target) != filepath.Clean(s.adrDir()) || !strings.HasSuffix(target, ".md") {
			continue
		}
		name := filepath.Base(target)
		if err := s.regularADR(name); err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("roadmap %s links to %s, which cannot be read: %v", rm.Slug, dest, err))
			continue
		}
		found[name] = true
	}
	entries, err := s.fsys().ReadDir(s.adrDir())
	if err != nil && !os.IsNotExist(err) {
		diagnostics = append(diagnostics, "the ADR inventory could not be read: "+err.Error())
	}
	if len(entries) > closeoutADRInventoryLimit {
		diagnostics = append(diagnostics, fmt.Sprintf("the ADR inventory has %d entries, above the limit of %d", len(entries), closeoutADRInventoryLimit))
		entries = nil
	}
	roadmapPath := filepath.Clean(filepath.Join(s.roadmapsDir(), rm.Slug+".md"))
	for _, entry := range entries {
		name := entry.Name()
		// Every candidate is read, the associated ones included: its size,
		// its readability and its own declaration are inputs. One that
		// cannot be read could hold a backlink, so it is unknown.
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		if err := s.regularADR(name); err != nil {
			diagnostics = append(diagnostics, "ADR "+name+" is not a regular file of .pose/adr: "+err.Error())
			continue
		}
		path := filepath.Join(s.adrDir(), name)
		if info, err := s.fsys().Lstat(path); err != nil || info.Size() > closeoutADRSizeLimit {
			diagnostics = append(diagnostics, "ADR "+name+" is unreadable or above the size limit")
			continue
		}
		raw, err := s.fsys().ReadFile(path)
		if err != nil {
			diagnostics = append(diagnostics, "ADR "+name+" could not be read: "+err.Error())
			continue
		}
		_, adrBody := SplitFrontmatter(string(raw))
		backward, _, backwardProblem := associationList(string(raw), "roadmaps", "ADR "+name)
		if backwardProblem != "" {
			diagnostics = append(diagnostics, backwardProblem)
		}
		for _, item := range backward {
			ref, err := ParseArtifactRef(item)
			if err != nil || ref.Kind != "roadmap" || ref.Milestone != "" {
				diagnostics = append(diagnostics, fmt.Sprintf("ADR %s declares %q, not roadmap:<slug>", name, item))
				continue
			}
			if ref.Project != "" && ref.Project != project {
				continue // a roadmap of another project: not this association
			}
			if _, err := s.GetRoadmap(ref.Slug); err != nil {
				diagnostics = append(diagnostics, fmt.Sprintf("ADR %s declares %s, which names no roadmap of this project", name, item))
				continue
			}
			if ref.Slug == rm.Slug {
				found[name] = true
			}
		}
		backlinks, undecodableBacklinks := markdownLinkDestinations(adrBody)
		for _, dest := range undecodableBacklinks {
			diagnostics = append(diagnostics, "ADR "+name+" links to "+dest+", whose escapes do not decode; the association reader cannot tell where it points")
		}
		for _, dest := range backlinks {
			target := filepath.Clean(filepath.Join(s.adrDir(), filepath.FromSlash(dest)))
			// A destination the reader may have cut short is unknown, never an
			// absent association (review round 14, N47).
			if filepath.Dir(target) == filepath.Clean(s.roadmapsDir()) && strings.ContainsAny(dest, "()") {
				diagnostics = append(diagnostics, "ADR "+name+" links to "+dest+", a destination with parentheses the association reader does not read whole; rename the roadmap or the link")
				continue
			}
			if filepath.Dir(target) != filepath.Clean(s.roadmapsDir()) || !strings.HasSuffix(target, ".md") {
				continue
			}
			// A link into the roadmaps namespace is an association input: one
			// that does not resolve to a roadmap leaves the answer unknown.
			if info, err := s.fsys().Lstat(target); err != nil || !info.Mode().IsRegular() || !artifactPathWithin(s.fsys(), s.Root, target) {
				diagnostics = append(diagnostics, "ADR "+name+" links to "+dest+", which is not a roadmap of this project")
				continue
			}
			// The target is read as the roadmap it names, as a roadmaps:
			// declaration is: a regular entry that cannot be read or parsed
			// leaves the answer unknown, never an absence. The roadmaps README
			// is no roadmap, as ListRoadmaps reads the namespace.
			if strings.EqualFold(filepath.Base(target), "README.md") {
				continue
			}
			if _, err := s.GetRoadmap(strings.TrimSuffix(filepath.Base(target), ".md")); err != nil {
				diagnostics = append(diagnostics, "ADR "+name+" links to "+dest+", which could not be read as a roadmap of this project")
				continue
			}
			if target == roadmapPath {
				found[name] = true
			}
		}
	}
	adrs := make([]string, 0, len(found))
	for name := range found {
		adrs = append(adrs, name)
	}
	sort.Strings(adrs)
	sort.Strings(diagnostics)
	return adrs, diagnostics
}

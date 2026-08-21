package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

// instanceOwnedMarker tags a canonical manual section whose body belongs to the
// repository instance, not to the engine. `pose install`/`pose upgrade` refresh
// every other section from the shipped manual while preserving the body an
// instance already wrote under a tagged heading.
//
// Before this contract POSE.md and AGENTS.md had no distributable path at all:
// the installer skipped them outright unless --force, and --force overwrote the
// instance-owned sections wholesale, so canonical manual content never reached
// an existing repository (spec pose-manual-distribution-merge).
const instanceOwnedMarker = "<!-- pose:instance-owned -->"

// docSection is one `## `-delimited slice of a manual. Bodies stay as line
// slices so reassembly is exact — joining and re-splitting strings loses the
// blank line that separates a section from the next heading.
type docSection struct {
	Heading string
	Body    []string
}

// splitDocSections splits a manual on level-2 headings. Anything before the
// first `## ` heading is returned as the preamble.
func splitDocSections(doc string) (preamble []string, sections []docSection) {
	current := docSection{}
	var buf []string
	flush := func() {
		if current.Heading == "" {
			preamble = buf
			return
		}
		current.Body = buf
		sections = append(sections, current)
	}
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			current = docSection{Heading: line}
			buf = nil
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return preamble, sections
}

// MergeManagedDoc rebuilds a managed manual from the canonical text while
// keeping what the instance owns. Sections the canonical text tags with
// instanceOwnedMarker take the local body when the instance already has that
// heading; every other section is refreshed. Local sections absent from the
// canonical manual are appended rather than dropped, so a repository never
// loses content it added.
//
// The second return value reports whether anything was preserved from the
// local copy, which lets the caller log a merge instead of an overwrite.
func MergeManagedDoc(canonical, local string) (string, bool) {
	merged, preserved, _ := mergeManagedDoc(canonical, local, nil)
	return merged, preserved
}

// MergeManagedDocAcrossLocale is MergeManagedDoc for the case where local was
// last written in a different locale than canonical. headingTranslation maps
// each of local's headings to the canonical heading it corresponds to (built
// by buildHeadingTranslation), so a section translated between locales is
// recognized as the same section instead of being appended as unrelated
// content the merge has never seen (spec pose-locale-switch-section-identity).
func MergeManagedDocAcrossLocale(canonical, local string, headingTranslation map[string]string) (string, bool) {
	merged, preserved, _ := mergeManagedDoc(canonical, local, headingTranslation)
	return merged, preserved
}

// MergeDropsLocalContent reports whether merging would drop a non-blank line the
// instance wrote. Text written inside an engine-owned section body — a note
// appended to the end of the file, for instance — belongs to no local heading,
// so the refresh legitimately overwrites it. Losing it silently is what is not
// acceptable: the caller keeps a `.pose-backup` copy when this returns true
// (spec pose-managed-doc-content-preservation).
func MergeDropsLocalContent(canonical, local string) bool {
	_, _, dropped := mergeManagedDoc(canonical, local, nil)
	return dropped
}

// MergeAcrossLocaleDropsLocalContent is MergeDropsLocalContent for a merge
// that also translates headings across a locale switch.
func MergeAcrossLocaleDropsLocalContent(canonical, local string, headingTranslation map[string]string) bool {
	_, _, dropped := mergeManagedDoc(canonical, local, headingTranslation)
	return dropped
}

// buildHeadingTranslation pairs canonicalFrom's sections with canonicalTo's by
// position — both are shipped translations of the same manual, kept in
// lockstep by the locale-parity gate, so section N in one is section N in the
// other — and returns a map from canonicalFrom's heading text to canonicalTo's
// at the same position. Mismatched section counts return nil: pairing by
// position would silently mislabel sections rather than merely fail to
// translate, so an untranslatable manual falls back to literal heading
// matching (same-language behavior) instead.
func buildHeadingTranslation(canonicalFrom, canonicalTo string) map[string]string {
	_, fromSections := splitDocSections(canonicalFrom)
	_, toSections := splitDocSections(canonicalTo)
	if len(fromSections) == 0 || len(fromSections) != len(toSections) {
		return nil
	}
	translation := make(map[string]string, len(fromSections))
	for i, section := range fromSections {
		translation[section.Heading] = toSections[i].Heading
	}
	return translation
}

func mergeManagedDoc(canonical, local string, headingTranslation map[string]string) (string, bool, bool) {
	canonicalPre, canonicalSections := splitDocSections(canonical)
	_, localSections := splitDocSections(local)

	// effectiveHeading is what a local section is matched against: its own
	// heading in the common same-locale case, or the canonical translation of
	// it when the instance is merging across a locale switch — otherwise a
	// translated section reads as content the engine has never seen and gets
	// appended as a duplicate instead of recognized as the same section
	// (spec pose-locale-switch-section-identity).
	effectiveHeading := func(heading string) string {
		if translated, ok := headingTranslation[heading]; ok {
			return translated
		}
		return heading
	}

	localByHeading := make(map[string]docSection, len(localSections))
	for _, section := range localSections {
		key := effectiveHeading(section.Heading)
		// First occurrence wins: a duplicated heading in the local copy must
		// not let a later one silently replace the earlier body.
		if _, seen := localByHeading[key]; !seen {
			localByHeading[key] = section
		}
	}

	preserved := false
	merged := make([]docSection, 0, len(canonicalSections))
	kept := make(map[string]bool, len(canonicalSections))
	for _, section := range canonicalSections {
		kept[section.Heading] = true
		if !sectionIsInstanceOwned(section) {
			merged = append(merged, section)
			continue
		}
		localSection, ok := localByHeading[section.Heading]
		if !ok {
			merged = append(merged, section)
			continue
		}
		preserved = true
		merged = append(merged, docSection{Heading: section.Heading, Body: localSection.Body})
	}

	// Sections the instance added on its own are unknown to the engine; keep
	// them at the end instead of discarding them. A section already matched
	// above via its translated heading is not "unknown" — skip it here too,
	// or it would be both merged in place and appended again as a duplicate.
	for _, section := range localSections {
		if !kept[effectiveHeading(section.Heading)] {
			preserved = true
			merged = append(merged, section)
		}
	}

	out := append([]string{}, canonicalPre...)
	for _, section := range merged {
		out = append(out, section.Heading)
		out = append(out, section.Body...)
	}
	result := strings.Join(out, "\n")
	return result, preserved, droppedLocalContent(local, result)
}

// droppedLocalContent reports whether a non-blank line present in the local
// manual is absent from the merged result. Comparing line multisets is enough
// here: the merge only ever reorders, refreshes or appends whole sections, so a
// local line that survives appears verbatim.
func droppedLocalContent(local, merged string) bool {
	remaining := map[string]int{}
	for _, line := range strings.Split(merged, "\n") {
		remaining[strings.TrimSpace(line)]++
	}
	for _, line := range strings.Split(local, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if remaining[trimmed] == 0 {
			return true
		}
		remaining[trimmed]--
	}
	return false
}

// sectionIsInstanceOwned reports whether the canonical manual tags this
// section's body as belonging to the instance or active contributor mode.
func sectionIsInstanceOwned(section docSection) bool {
	for _, line := range section.Body {
		if strings.Contains(line, instanceOwnedMarker) || strings.Contains(line, contributorModeMarker) {
			return true
		}
	}
	return false
}

// refreshManagedDocs merges the shipped manuals into an existing instance.
// `pose upgrade` calls it on every run — without it the upgrade path delivers
// only the binary and the schema stamp, and no manual change ever reaches a
// repository that is already installed.
func refreshManagedDocs(root, locale string, stdout io.Writer, localeText cliLocale) error {
	dist := scaffold.Dist()
	for _, doc := range []string{"AGENTS.md", "POSE.md"} {
		existing, err := os.ReadFile(filepath.Join(root, doc))
		if err != nil {
			// Absent manual is `pose install`'s job, not the upgrade path's.
			continue
		}
		// The locale is a property of the installed manual, not of the shell
		// running the upgrade: an instance installed with --locale pt-BR must
		// not be silently rewritten in English by a plain `pose upgrade`, and
		// an explicit `--locale en` must land in English even when the
		// instance is currently pt-BR (resolveDocLocale honors an explicit
		// preference for every locale, "en" included).
		existingResolved := resolveDocLocale(dist, doc, string(existing), "", false)
		targetResolved := resolveDocLocale(dist, doc, string(existing), locale, locale != "")
		docsPrefix := ""
		if targetResolved != "" {
			docsPrefix = "locales/" + targetResolved + "/"
		}
		canonical, err := fs.ReadFile(dist, docsPrefix+doc)
		if err != nil {
			canonical, err = fs.ReadFile(dist, doc)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", doc, err)
		}
		// The instance already resolved the scaffold placeholders; reuse its
		// own values so a refresh never reintroduces {{PROJECT_NAME}}.
		content := restoreDocPlaceholders(string(canonical), string(existing), root)

		var merged string
		var preserved, dropsContent bool
		if targetResolved != existingResolved {
			// Crossing a locale switch: local headings are in a different
			// language than canonical's, so literal heading matching would
			// treat every local section as unknown-to-the-engine and append
			// it as a duplicate instead of recognizing it as the same
			// section (spec pose-locale-switch-section-identity).
			existingCanonicalPrefix := ""
			if existingResolved != "" {
				existingCanonicalPrefix = "locales/" + existingResolved + "/"
			}
			existingCanonical, ecErr := fs.ReadFile(dist, existingCanonicalPrefix+doc)
			var translation map[string]string
			if ecErr == nil {
				translation = buildHeadingTranslation(string(existingCanonical), string(canonical))
			}
			merged, preserved = MergeManagedDocAcrossLocale(content, string(existing), translation)
			dropsContent = MergeAcrossLocaleDropsLocalContent(content, string(existing), translation)
		} else {
			merged, preserved = MergeManagedDoc(content, string(existing))
			dropsContent = MergeDropsLocalContent(content, string(existing))
		}
		if merged == string(existing) {
			continue
		}
		// Hard guard: a refresh that cannot resolve every placeholder would
		// hand the instance a manual reading "{{PROJECT_NAME}}". Leaving the
		// file untouched is strictly better than degrading it.
		if strings.Contains(merged, "{{") {
			fmt.Fprintf(stdout, cliText(localeText, "[WARN] skipped %s: unresolved scaffold placeholder\n", "[AVISO] %s ignorado: placeholder de scaffold não resolvido\n"), doc)
			continue
		}
		// Text the instance wrote inside an engine-owned section body has no
		// heading of its own, so the refresh legitimately overwrites it. Keep
		// a copy and say so explicitly — the merge only preserves what it can
		// recognize as a whole section, and a generic "merged" log would
		// otherwise read as if nothing had been lost (spec
		// pose-managed-doc-content-preservation, matching the warning `pose
		// install`'s own merge path already gives for the same case).
		if dropsContent {
			if err := os.WriteFile(filepath.Join(root, doc)+".pose-backup", existing, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(stdout, cliText(localeText, "[WARN] backed up customized: %s -> %s.pose-backup (content outside instance-owned sections was not preserved)\n", "[AVISO] backup de conteúdo customizado: %s -> %s.pose-backup (conteúdo fora das seções da instância não foi preservado)\n"), doc, doc)
		}
		if err := writeAtomic(filepath.Join(root, doc), []byte(merged), 0o644); err != nil {
			return err
		}
		if preserved {
			fmt.Fprintf(stdout, cliText(localeText, "[INFO] merged: %s (instance-owned sections preserved)\n", "[INFO] mesclado: %s (seções da instância preservadas)\n"), doc)
		} else {
			fmt.Fprintf(stdout, cliText(localeText, "[INFO] updated: %s\n", "[INFO] atualizado: %s\n"), doc)
		}
	}
	return nil
}

// resolveDocLocale picks which shipped translation of doc the installed manual
// actually is, by scoring how many of each candidate's headings the local file
// still contains. A non-"en" preference always wins outright when the
// document ships that locale. An "en" preference wins outright only when
// explicit is true — callers that pass "en" merely as their zero-value
// default, with no real ask from the operator, must pass explicit=false, or
// a genuinely pt-BR (or other) instance would be silently forced back to
// English by every unrelated command that happens to auto-detect a locale
// (github.com/oseiaspereira88/pose#18's failure mode, one layer deeper: this
// function itself must not repeat it for callers that DO make an explicit
// "en" ask, e.g. `pose update --locale en` against a pt-BR instance). An
// empty preference always falls through to auto-detection against the
// existing content. Returns "" for the English original.
func resolveDocLocale(dist fs.FS, doc, existing, preferred string, explicit bool) string {
	if preferred == "en" {
		if explicit {
			return ""
		}
	} else if preferred != "" {
		if _, err := fs.Stat(dist, "locales/"+preferred+"/"+doc); err == nil {
			return preferred
		}
	}
	score := func(prefix string) int {
		canonical, err := fs.ReadFile(dist, prefix+doc)
		if err != nil {
			return -1
		}
		_, sections := splitDocSections(string(canonical))
		matches := 0
		for _, section := range sections {
			if strings.Contains(existing, section.Heading) {
				matches++
			}
		}
		return matches
	}
	best, bestScore := "", score("")
	entries, err := fs.ReadDir(dist, "locales")
	if err != nil {
		return best
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Strictly greater keeps English as the tie-breaking default.
		if s := score("locales/" + entry.Name() + "/"); s > bestScore {
			best, bestScore = entry.Name(), s
		}
	}
	return best
}

// restoreDocPlaceholders substitutes the scaffold placeholders in a canonical
// manual with the values the installed instance already carries, read back from
// its own first line mentioning them. When they cannot be recovered the
// placeholders are left untouched rather than guessed.
func restoreDocPlaceholders(canonical, existing, root string) string {
	name, id := detectProjectIdentity(existing, root)
	replacements := []string{}
	if name != "" {
		replacements = append(replacements, "{{PROJECT_NAME}}", name)
	}
	if id != "" {
		replacements = append(replacements, "{{PROJECT_ID}}", id)
	}
	if len(replacements) == 0 {
		return canonical
	}
	return strings.NewReplacer(replacements...).Replace(canonical)
}

// detectProjectIdentity recovers the project name and id an installed manual
// was rendered with. It reads the rendered manuals first — those carry the
// values the instance actually chose — and falls back to `.mcp.json` and the
// repository directory name, which is what `pose install` defaults to.
func detectProjectIdentity(existing, root string) (name, id string) {
	namePrefixes := []struct{ prefix, suffix string }{
		{"POSE is the operating standard for agent work in **", "**."},
		{"POSE é o padrão operacional de trabalho com agentes em **", "**."},
		{"# AGENTS.md — ", ""},
	}
	for _, line := range strings.Split(existing, "\n") {
		if name != "" {
			break
		}
		for _, form := range namePrefixes {
			rest, ok := strings.CutPrefix(line, form.prefix)
			if !ok {
				continue
			}
			rest = strings.TrimSpace(rest)
			if form.suffix == "" {
				name = rest
				break
			}
			if value, ok := strings.CutSuffix(rest, form.suffix); ok {
				name = value
				break
			}
		}
	}
	if raw, err := os.ReadFile(filepath.Join(root, ".mcp.json")); err == nil {
		if _, rest, ok := strings.Cut(string(raw), "\"POSE_DEFAULT_PROJECT_ID\": \""); ok {
			if value, _, ok := strings.Cut(rest, "\""); ok {
				id = value
			}
		}
	}
	if name == "" {
		name = filepath.Base(root)
	}
	if id == "" {
		id = "proj." + name
	}
	return name, id
}

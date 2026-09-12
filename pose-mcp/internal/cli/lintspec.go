package cli

// Native lint-spec gate. It preserves the published lifecycle verdicts and
// stable machine metrics without an external runtime.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

var (
	requiredSections = []string{"Intent", "Requirements", "Technical Plan", "Tasks", "Validation", "Final Report"}
	optionalSections = []string{"Decisions"}
	validStatus      = map[string]bool{"draft": true, "in-progress": true, "done": true, "blocked": true, "superseded": true, "abandoned": true}

	validDispositions       = map[string]bool{"open": true, "spawned": true, "covered": true, "duplicate": true, "done": true, "wont-do": true}
	dispositionNeedsTarget  = map[string]bool{"spawned": true, "covered": true, "duplicate": true, "wont-do": true}
	dispositionSlugTargeted = map[string]bool{"spawned": true, "covered": true, "duplicate": true}

	headingRE      = regexp.MustCompile(`^##\s+\d+\.\s+(.+?)\s*$`)
	subheadingRE   = regexp.MustCompile(`^###\s+(.+?)\s*$`)
	placeholderRE  = regexp.MustCompile(`^\s*<!--.*-->\s*$`)
	emptyBulletRE  = regexp.MustCompile(`^\s*-\s*$`)
	metaLineRE     = regexp.MustCompile(`^\s*-\s*[A-Za-zÀ-ÿ ]+:\s*$`)
	htmlCommentRE  = regexp.MustCompile(`(?s)<!--.*?-->`)
	bulletRE       = regexp.MustCompile(`^\s*-\s+(.*\S)\s*$`)
	dispositionRE  = regexp.MustCompile(`^\[\s*([a-z-]+)\s*(?::\s*(.+?))?\s*\]\s*(.*)$`)
	frontmatterRE  = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	inlineCommRE   = regexp.MustCompile(`\s+#.*$`)
	acceptanceIDRE = regexp.MustCompile(`^\s*-\s*R(\d+)\s*(?:\[(\w+)\])?\s*[:—-]`)
	depSlugRE      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	depMilestoneRE = regexp.MustCompile(`^milestone:[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$`)
	depRoadmapRE   = regexp.MustCompile(`^roadmap:[a-z0-9][a-z0-9._-]*$`)
	// depXrefRE is the cross-repository reference grammar (spec
	// pose-cross-repo-portfolio, R1): xref:<project_id>/<spec-slug> —
	// additive to the local-only forms above, never a substitute for them.
	depXrefRE = regexp.MustCompile(`^xref:[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$`)
)

func lintParseFrontmatter(text string) map[string]string {
	m := frontmatterRE.FindStringSubmatch(text)
	fields := map[string]string{}
	if m == nil {
		return fields
	}
	for _, line := range strings.Split(m[1], "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") || !strings.Contains(line, ":") {
			continue
		}
		key, value, _ := strings.Cut(line, ":")
		value = strings.TrimSpace(inlineCommRE.ReplaceAllString(value, ""))
		fields[strings.TrimSpace(key)] = value
	}
	return fields
}

func isContentLine(line string) bool {
	stripped := strings.TrimSpace(line)
	switch {
	case stripped == "", stripped == "---":
		return false
	case placeholderRE.MatchString(line),
		emptyBulletRE.MatchString(line),
		subheadingRE.MatchString(line),
		metaLineRE.MatchString(line):
		return false
	}
	return true
}

func splitLintSections(text string) map[string][]string {
	sections := map[string][]string{}
	var name string
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if m := headingRE.FindStringSubmatch(line); m != nil {
			if name != "" {
				sections[name] = lines
			}
			name = strings.TrimSpace(m[1])
			lines = nil
			continue
		}
		lines = append(lines, line)
	}
	if name != "" {
		sections[name] = lines
	}
	return sections
}

func classifySection(lines []string) string {
	content, hasAny := 0, false
	for _, l := range lines {
		if isContentLine(l) {
			content++
		}
		if strings.TrimSpace(l) != "" {
			hasAny = true
		}
	}
	if content > 0 {
		return "filled"
	}
	if hasAny {
		return "skeleton"
	}
	return "empty"
}

// warnMisplacedFollowupMeta names the one format POSE reads for follow-up
// ownership, for an item that wrote it some other way.
func warnMisplacedFollowupMeta(stderr io.Writer, locale cliLocale, slug, content string) {
	snippet := content
	if len([]rune(snippet)) > 60 {
		snippet = string([]rune(snippet)[:60]) + "…"
	}
	fmt.Fprintf(stderr, cliText(locale,
		"[WARNING] %s: follow-up ownership is outside the trailing group and is ignored — end the bullet with '(owner:@alias crit:low|medium|high review:YYYY-MM-DD)' → \"%s\"\n",
		"[AVISO] %s: ownership do follow-up está fora do grupo final e é ignorado — termine o bullet com '(owner:@alias crit:low|medium|high review:YYYY-MM-DD)' → \"%s\"\n"),
		slug, snippet)
}

// extractFollowups returns each follow-up bullet with its continuation lines
// joined, the way `pose followups` reads it: a bullet runs until the next
// bullet, heading or blank line. Reading only the first line made the lint
// report a wrapped "(owner:… review:…)" group as unowned while `pose
// followups` read the same item as owned (spec pose-one-follow-up-format).
func extractFollowups(finalReport []string) []string {
	var bullets []string
	in := false
	current := -1
	for _, line := range finalReport {
		if m := subheadingRE.FindStringSubmatch(line); m != nil {
			in = strings.HasPrefix(strings.ToLower(strings.TrimSpace(m[1])), "follow-up")
			current = -1
			continue
		}
		if !in {
			continue
		}
		if m := bulletRE.FindStringSubmatch(line); m != nil {
			bullets = append(bullets, strings.TrimSpace(m[1]))
			current = len(bullets) - 1
			continue
		}
		if current >= 0 {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				bullets[current] += " " + trimmed
			} else {
				current = -1
			}
		}
	}
	return bullets
}

func collectSpecSlugs(specsDir string) map[string]bool {
	slugs := map[string]bool{}
	if filepath.Base(specsDir) != "specs" {
		if _, err := os.Stat(filepath.Join(specsDir, ".pose", "specs")); err == nil {
			specsDir = filepath.Join(specsDir, ".pose", "specs")
		} else if _, err := os.Stat(filepath.Join(specsDir, "specs")); err == nil {
			specsDir = filepath.Join(specsDir, "specs")
		}
	}
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return slugs
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			specMD := filepath.Join(specsDir, name, "spec.md")
			if b, err := os.ReadFile(specMD); err == nil {
				if slug := lintParseFrontmatter(string(b))["slug"]; slug != "" {
					slugs[slug] = true
				} else {
					slugs[name] = true
				}
			}
		} else if strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "README.md") {
			specMD := filepath.Join(specsDir, name)
			if b, err := os.ReadFile(specMD); err == nil {
				if slug := lintParseFrontmatter(string(b))["slug"]; slug != "" {
					slugs[slug] = true
				}
				base := strings.TrimSuffix(name, ".md")
				slugs[base] = true
				if m := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.*)$`).FindStringSubmatch(base); m != nil {
					slugs[m[1]] = true
				}
			}
		}
	}
	return slugs
}

func lintParseDependsOn(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = value[1 : len(value)-1]
	}
	var out []string
	for _, item := range strings.Split(value, ",") {
		if t := strings.TrimSpace(item); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func siblingSpecContent(specsDir, slug string) (string, bool) {
	if filepath.Base(specsDir) != "specs" {
		if _, err := os.Stat(filepath.Join(specsDir, ".pose", "specs")); err == nil {
			specsDir = filepath.Join(specsDir, ".pose", "specs")
		} else if _, err := os.Stat(filepath.Join(specsDir, "specs")); err == nil {
			specsDir = filepath.Join(specsDir, "specs")
		}
	}
	if b, err := os.ReadFile(filepath.Join(specsDir, slug, "spec.md")); err == nil {
		return string(b), true
	}
	if b, err := os.ReadFile(filepath.Join(specsDir, slug+".md")); err == nil {
		return string(b), true
	}
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if strings.HasSuffix(name, "-"+slug) || strings.HasSuffix(name, "_"+slug) {
				if b, err := os.ReadFile(filepath.Join(specsDir, name, "spec.md")); err == nil {
					fm := lintParseFrontmatter(string(b))
					if fm["slug"] == slug || fm["slug"] == "" {
						return string(b), true
					}
				}
			}
		} else if strings.HasSuffix(name, ".md") && !strings.EqualFold(name, "README.md") {
			base := strings.TrimSuffix(name, ".md")
			if base == slug || strings.HasSuffix(base, "-"+slug) || strings.HasSuffix(base, "_"+slug) {
				if b, err := os.ReadFile(filepath.Join(specsDir, name)); err == nil {
					fm := lintParseFrontmatter(string(b))
					if fm["slug"] == slug || fm["slug"] == "" {
						return string(b), true
					}
				}
			}
		}
	}
	for _, e := range entries {
		var path string
		if e.IsDir() {
			path = filepath.Join(specsDir, e.Name(), "spec.md")
		} else if strings.HasSuffix(e.Name(), ".md") && !strings.EqualFold(e.Name(), "README.md") {
			path = filepath.Join(specsDir, e.Name())
		}
		if path != "" {
			if b, err := os.ReadFile(path); err == nil {
				fm := lintParseFrontmatter(string(b))
				if fm["slug"] == slug {
					return string(b), true
				}
			}
		}
	}
	return "", false
}

func siblingSpecStatus(specsDir, slug string) string {
	if content, ok := siblingSpecContent(specsDir, slug); ok {
		return lintParseFrontmatter(content)["status"]
	}
	return ""
}

// lintFollowupDisposition mirrors lint_followup_disposition: returns the
// disposition ("" when absent) and an error message ("" when valid).
func lintFollowupDisposition(content string, knownSlugs map[string]bool, currentSlug string, locale cliLocale) (string, string) {
	m := dispositionRE.FindStringSubmatch(content)
	if m == nil {
		return "", cliText(locale, "missing disposition (expected [open|spawned|covered|duplicate|done|wont-do] prefix)", "sem disposição (esperado prefixo [open|spawned|covered|duplicate|done|wont-do])")
	}
	disposition, target := m[1], strings.TrimSpace(m[2])
	if !validDispositions[disposition] {
		return disposition, fmt.Sprintf(cliText(locale, "invalid disposition: [%s]", "disposição inválida: [%s]"), disposition)
	}
	if dispositionNeedsTarget[disposition] && target == "" {
		kind := "slug"
		if disposition == "wont-do" {
			kind = cliText(locale, "reason", "motivo")
		}
		return disposition, fmt.Sprintf(cliText(locale, "disposition [%s] requires a %s (use [%s: <%s>])", "disposição [%s] exige %s (use [%s: <%s>])"), disposition, kind, disposition, kind)
	}
	if knownSlugs != nil && dispositionSlugTargeted[disposition] {
		if currentSlug != "" && target == currentSlug {
			return disposition, fmt.Sprintf(cliText(locale, "disposition [%s] points to the current spec (%s)", "disposição [%s] aponta para a própria spec (%s)"), disposition, target)
		}
		if !knownSlugs[target] {
			return disposition, fmt.Sprintf(cliText(locale, "disposition [%s: %s] points to a missing spec", "disposição [%s: %s] aponta para spec inexistente"), disposition, target)
		}
	}
	return disposition, ""
}

type ridEntry struct {
	id   string
	crit string
}

func parseRequirementIDs(lines []string) []ridEntry {
	var ids []ridEntry
	for _, line := range lines {
		if m := acceptanceIDRE.FindStringSubmatch(line); m != nil {
			ids = append(ids, ridEntry{"R" + m[1], m[2]})
		}
	}
	return ids
}

func parseISOInstant(value string) (time.Time, bool) {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// lintOneSpec lints a single spec.md, printing the same machine lines and
// stderr diagnostics as the python engine. Returns 0/1 (2 on IO error).
// specFindings routes one spec's lint findings through the renderer. They are
// the command's result, so they belong on stdout: they used to go to stderr,
// where `pose lint-spec 2>/dev/null` dropped them silently while other commands
// kept theirs (spec pose-cli-output-rendering-system R4).
type specFindings struct {
	r    *cliout.Renderer
	slug string
}

func (f specFindings) finding(state cliout.State, code, message string) {
	f.r.Finding(cliout.Finding{State: state, Code: code, Path: f.slug, Message: message})
}

// sectionState reads the section's own requiredness: a missing required section
// is an error, an optional one a warning.
func sectionState(required bool) cliout.State {
	if required {
		return cliout.StateError
	}
	return cliout.StateWarning
}

func lintOneSpec(specPath string, requiredOnly, readyCheck bool, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	raw, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: spec not found: %s\n", "Erro: spec ausente: %s\n"), specPath)
		return 2
	}
	frontmatter := lintParseFrontmatter(string(raw))
	text := htmlCommentRE.ReplaceAllString(string(raw), "")
	sections := splitLintSections(text)
	slug := frontmatter["slug"]
	if slug == "" {
		slug = filepath.Base(filepath.Dir(specPath))
	}
	lint := specFindings{r: render(stdout, stderr), slug: slug}

	if readyCheck {
		failures := 0
		for _, name := range []string{"Intent", "Requirements", "Technical Plan"} {
			lines, ok := sections[name]
			if !ok || classifySection(lines) != "filled" {
				lint.finding(cliout.StateError, "dor", fmt.Sprintf(cliText(locale, "DoR: section %s is missing, empty, or skeletal", "DoR: seção %s ausente/vazia/esquelética"), name))
				failures++
			}
		}
		if len(parseRequirementIDs(sections["Requirements"])) == 0 {
			lint.finding(cliout.StateError, "dor", cliText(locale, "DoR: no acceptance criterion has a stable ID (use '- R<N>: ...' bullets in Requirements)", "DoR: nenhum acceptance criterion com ID estável (use bullets '- R<N>: ...' em Requirements)"))
			failures++
		}
		for _, ref := range lintParseDependsOn(frontmatter["depends_on"]) {
			if depSlugRE.MatchString(ref) || depMilestoneRE.MatchString(ref) || depRoadmapRE.MatchString(ref) || depXrefRE.MatchString(ref) {
				continue
			}
			lint.finding(cliout.StateError, "dor", fmt.Sprintf(cliText(locale, "DoR: invalid depends_on reference: '%s'", "DoR: ref inválida em depends_on: '%s'"), ref))
			failures++
		}
		ready := "true"
		if failures > 0 {
			ready = "false"
		}
		fmt.Fprintf(stdout, "spec.ready=%s\n", ready)
		fmt.Fprintf(stdout, "spec.ready.failures=%d\n", failures)
		if failures > 0 {
			return 1
		}
		return 0
	}

	targets := append([]string{}, requiredSections...)
	if !requiredOnly {
		targets = append(targets, optionalSections...)
	}
	isRequired := map[string]bool{}
	for _, s := range requiredSections {
		isRequired[s] = true
	}

	total, filled, skeleton, empty, requiredMissing := 0, 0, 0, 0, 0
	for _, name := range targets {
		lines, ok := sections[name]
		if !ok {
			if isRequired[name] {
				lint.finding(cliout.StateError, "section", fmt.Sprintf(cliText(locale, "required section missing: %s", "seção obrigatória ausente: %s"), name))
				requiredMissing++
			} else {
				lint.finding(cliout.StateWarning, "section", fmt.Sprintf(cliText(locale, "optional section missing: %s", "seção opcional ausente: %s"), name))
			}
			continue
		}
		total++
		switch classifySection(lines) {
		case "filled":
			filled++
		case "skeleton":
			skeleton++
			lint.finding(sectionState(isRequired[name]), "section", fmt.Sprintf(cliText(locale, "%s: skeletal (placeholders or comments only)", "%s: esqueleto (apenas placeholders/comentários)"), name))
			if isRequired[name] {
				requiredMissing++
			}
		default:
			empty++
			lint.finding(sectionState(isRequired[name]), "section", fmt.Sprintf(cliText(locale, "%s: empty", "%s: vazia"), name))
			if isRequired[name] {
				requiredMissing++
			}
		}
	}

	specStatus := frontmatter["status"]
	if specStatus == "" {
		specStatus = "unset"
	}
	lifecycle := 0
	if specStatus != "unset" && !validStatus[specStatus] {
		lint.finding(cliout.StateError, "frontmatter", fmt.Sprintf(cliText(locale, "invalid frontmatter status: '%s' (use draft|in-progress|done|blocked|superseded|abandoned)", "status inválido no frontmatter: '%s' (use draft|in-progress|done|blocked|superseded|abandoned)"), specStatus))
		lifecycle++
	}

	// Lifecycle dates.
	parsed := map[string]time.Time{}
	for _, field := range []string{"created_at", "completed_at"} {
		value := strings.Trim(strings.TrimSpace(frontmatter[field]), `"'`)
		if value == "" {
			continue
		}
		if t, ok := parseISOInstant(value); ok {
			parsed[field] = t
		} else {
			lint.finding(cliout.StateError, "frontmatter", fmt.Sprintf(cliText(locale, "%s must use ISO 8601: '%s'", "%s deve usar ISO 8601: '%s'"), field, value))
			lifecycle++
		}
	}
	if c, ok1 := parsed["created_at"]; ok1 {
		if d, ok2 := parsed["completed_at"]; ok2 && d.Before(c) {
			lint.finding(cliout.StateError, "frontmatter", cliText(locale, "completed_at is earlier than created_at", "completed_at anterior a created_at"))
			lifecycle++
		}
	}

	// Canonical heading uniqueness.
	nameCount := map[string]int{}
	followupHeadings := 0
	for _, line := range strings.Split(text, "\n") {
		if m := headingRE.FindStringSubmatch(line); m != nil {
			nameCount[strings.ToLower(strings.TrimSpace(m[1]))]++
		}
		if m := subheadingRE.FindStringSubmatch(line); m != nil &&
			strings.HasPrefix(strings.ToLower(strings.TrimSpace(m[1])), "follow-up") {
			followupHeadings++
		}
	}
	var dupNames []string
	for n, c := range nameCount {
		if c > 1 {
			dupNames = append(dupNames, n)
		}
	}
	sort.Strings(dupNames)
	for _, n := range dupNames {
		lint.finding(cliout.StateError, "heading", fmt.Sprintf(cliText(locale, "duplicate canonical heading: %s appears %d times", "heading canônico duplicado: %s aparece %d vezes"), n, nameCount[n]))
		lifecycle++
	}
	if followupHeadings > 1 {
		lint.finding(cliout.StateError, "heading", fmt.Sprintf(cliText(locale, "duplicate canonical heading: Follow-ups appears %d times", "heading canônico duplicado: Follow-ups aparece %d vezes"), followupHeadings))
		lifecycle++
	}

	specsDir := filepath.Dir(specPath)
	if filepath.Base(specsDir) != "specs" {
		specsDir = filepath.Dir(specsDir)
	}
	if specStatus == "in-progress" {
		for _, dep := range lintParseDependsOn(frontmatter["depends_on"]) {
			if strings.Contains(dep, ":") {
				continue
			}
			if st := siblingSpecStatus(specsDir, dep); st != "" && st != "done" {
				lint.finding(cliout.StateWarning, "dependency", fmt.Sprintf(cliText(locale, "in-progress with unsatisfied dependency: '%s' (status: %s)", "in-progress com dependência não satisfeita: '%s' (status: %s)"), dep, st))
			}
		}
	}

	// Duplicate R-IDs.
	ridFailures := 0
	seen := map[string]int{}
	for _, r := range parseRequirementIDs(sections["Requirements"]) {
		seen[r.id]++
	}
	var rids []string
	for id, c := range seen {
		if c > 1 {
			rids = append(rids, id)
		}
	}
	sort.Strings(rids)
	for _, id := range rids {
		lint.finding(cliout.StateError, "requirement", fmt.Sprintf(cliText(locale, "duplicate R-ID: %s appears %d times in Requirements", "R-ID duplicado: %s aparece %d vezes em Requirements"), id, seen[id]))
		ridFailures++
	}

	// Requirement-to-evidence trace (spec pose-requirement-evidence-traceability).
	// Malformed entries and orphans always fail; full coverage is enforced at
	// closeout when the section exists. Legacy done specs without the section
	// get a visible warning (additive migration — see the traceability ADR).
	trace := posepkg.ParseRequirementTrace(text)
	traceFailures := 0
	for _, msg := range trace.Errors {
		lint.finding(cliout.StateError, "requirement-trace", fmt.Sprintf(cliText(locale, "requirement trace: %s", "requirement trace: %s"), msg))
		traceFailures++
	}
	for _, id := range trace.Orphans {
		lint.finding(cliout.StateError, "requirement-trace", fmt.Sprintf(cliText(locale, "requirement trace: %s is traced but not declared in Requirements", "requirement trace: %s rastreado mas não declarado em Requirements"), id))
		traceFailures++
	}
	traceEntries := 0
	for _, r := range trace.Requirements {
		if r.Entry != nil {
			traceEntries++
		}
	}
	if specStatus == "done" {
		if trace.HasSection {
			for _, id := range trace.Missing {
				lint.finding(cliout.StateError, "requirement-trace", fmt.Sprintf(cliText(locale, "requirement trace: %s has no trace entry (declare satisfied, waived or withdrawn)", "requirement trace: %s sem entrada de trace (declare satisfied, waived ou withdrawn)"), id))
				traceFailures++
			}
		} else if len(trace.Requirements) > 0 {
			// Was a warning while eleven pre-contract specs still lacked a
			// trace. They now have one, so a done spec with requirements and no
			// trace is a real gap rather than a legacy artefact
			// (spec pose-governance-gate-activation, R2).
			lint.finding(cliout.StateError, "requirement-trace", cliText(locale, "done without a '### Requirement trace' subsection in Validation (every done spec must trace its R-IDs)", "done sem subseção '### Requirement trace' em Validation (toda spec done deve rastrear seus R-IDs)"))
			traceFailures++
		}
	}
	if root, err := projectRoot(); err == nil {
		if deliveryPolicy, err := posepkg.LoadDeliveryPolicy(root); err != nil {
			lint.finding(cliout.StateError, "delivery", fmt.Sprintf("delivery policy: %v", err))
			traceFailures++
		} else if deliveryPolicy.Enabled {
			store := posepkg.Store{Root: root}
			if full, err := store.GetSpec(slug); err == nil {
				targets, found, parseErr := posepkg.ParseDeliveryTargets(*full)
				if parseErr != nil {
					lint.finding(cliout.StateError, "delivery", fmt.Sprintf("delivery targets: %v", parseErr))
					traceFailures++
				} else if (found || len(full.Delivers) > 0) && specStatus == "done" {
					statuses := map[string]string{}
					if summaries, err := store.ListSpecs("", ""); err == nil {
						for _, item := range summaries {
							statuses[item.Slug] = item.Status
						}
					}
					for _, message := range posepkg.ValidateDeliveryTrace(*full, targets, statuses) {
						lint.finding(cliout.StateError, "delivery", fmt.Sprintf("delivery trace: %s", message))
						traceFailures++
					}
				}
			}
		}
	}

	// Amendment history gate (spec pose-spec-amendment-history): when the
	// append-only event log exists, a done spec must acknowledge the current
	// requirement state — silent post-evidence rewrites are rejected.
	amendFailures, amendEvents := 0, 0
	if events, aerr := posepkg.LoadAmendments(posepkg.AmendmentsPath(specPath)); aerr != nil {
		lint.finding(cliout.StateError, "amendments", fmt.Sprintf(cliText(locale, "amendments.jsonl: %v", "amendments.jsonl: %v"), aerr))
		amendFailures++
	} else if events != nil {
		amendEvents = len(events)
		if specStatus == "done" {
			for _, finding := range posepkg.UnacknowledgedChanges(text, events) {
				lint.finding(cliout.StateError, "amendments", fmt.Sprintf(cliText(locale, "amendment history: %s", "amendment history: %s"), finding))
				amendFailures++
			}
		}
	}

	followups := extractFollowups(sections["Final Report"])
	followupsOpen := 0
	knownSlugs := collectSpecSlugs(specsDir)

	if specStatus == "done" {
		if strings.TrimSpace(frontmatter["completed_at"]) == "" {
			lint.finding(cliout.StateError, "frontmatter", cliText(locale, "status: done requires populated 'completed_at' frontmatter", "status: done exige 'completed_at' preenchido no frontmatter"))
			lifecycle++
		}
		if root, err := projectRoot(); err == nil {
			if policy, err := posepkg.LoadArtifactPolicy(root); err != nil {
				lint.finding(cliout.StateError, "artifacts", fmt.Sprintf("artifact policy: %v", err))
				lifecycle++
			} else if policy.Enabled && strings.TrimSpace(frontmatter["completed_at"]) >= policy.AdoptedAt {
				claims, found, err := posepkg.ParseArtifactClaims(posepkg.Spec{Slug: slug, Body: text}, policy)
				if err != nil || !found || len(claims) == 0 {
					lint.finding(cliout.StateError, "artifacts", fmt.Sprintf("structured Artifacts declaration required after %s: %v", policy.AdoptedAt, err))
					lifecycle++
				}
			}
		}
		for _, content := range followups {
			disposition, errMsg := lintFollowupDisposition(content, knownSlugs, slug, locale)
			if errMsg != "" {
				snippet := content
				if len([]rune(snippet)) > 60 {
					snippet = string([]rune(snippet)[:60]) + "…"
				}
				lint.finding(cliout.StateError, "follow-up", fmt.Sprintf(cliText(locale, "follow-up lacks a valid disposition: %s → \"%s\"", "follow-up sem disposição válida: %s → \"%s\""), errMsg, snippet))
				lifecycle++
			} else if disposition == "open" {
				followupsOpen++
				// Ownership gate (spec pose-followup-ownership-sla): open
				// residual work on a done spec needs an owner, criticality
				// and review date. Legacy entries stay visible as unowned
				// warnings; malformed metadata is an error.
				_, owner, _, _, _, metaErr := parseFollowupMeta(content)
				if metaErr != "" {
					lint.finding(cliout.StateError, "follow-up", fmt.Sprintf(cliText(locale, "open follow-up ownership: %s", "ownership de follow-up aberto: %s"), metaErr))
					lifecycle++
				} else if followupMetaMisplaced(content) {
					warnMisplacedFollowupMeta(stderr, locale, slug, content)
				} else if owner == "unowned" {
					lint.finding(cliout.StateWarning, "follow-up", cliText(locale, "open follow-up is unowned (declare '(owner:@alias crit:low|medium|high review:YYYY-MM-DD)')", "follow-up aberto sem dono (declare '(owner:@alias crit:low|medium|high review:YYYY-MM-DD)')"))
				}
			} else if disposition == "covered" {
				m := dispositionRE.FindStringSubmatch(content)
				if m != nil {
					target := strings.TrimSpace(m[2])
					if target != "" && knownSlugs[target] {
						if targetContent, ok := siblingSpecContent(specsDir, target); ok {
							targetFM := lintParseFrontmatter(targetContent)
							hasAnchor := false
							for _, dep := range lintParseDependsOn(targetFM["depends_on"]) {
								if dep == slug {
									hasAnchor = true
									break
								}
							}
							if !hasAnchor && strings.Contains(targetContent, slug) {
								hasAnchor = true
							}
							if !hasAnchor {
								fmt.Fprintf(stderr, cliText(locale,
									"[WARNING] %s: follow-up [covered: %s] has no verifiable anchor (e.g. depends_on: %s or mention of '%s') in target spec %s\n",
									"[AVISO] %s: follow-up [covered: %s] não possui âncora verificável (ex.: depends_on: %s ou menção a '%s') na spec alvo %s\n",
								), slug, target, slug, slug, target)
							}
						}
					}
				}
			}
		}
	} else {
		for _, content := range followups {
			disposition, _ := lintFollowupDisposition(content, nil, "", locale)
			if disposition == "open" || disposition == "" {
				followupsOpen++
				// Before closeout an unowned item is not reported, but metadata
				// the parser ignores is: it was meant, and nothing reads it.
				if followupMetaMisplaced(content) {
					warnMisplacedFollowupMeta(stderr, locale, slug, content)
				}
			}
		}
	}

	fmt.Fprintf(stdout, "spec.path=%s\n", specPath)
	fmt.Fprintf(stdout, "spec.status=%s\n", specStatus)
	fmt.Fprintf(stdout, "spec.sections.total=%d\n", total)
	fmt.Fprintf(stdout, "spec.sections.filled=%d\n", filled)
	fmt.Fprintf(stdout, "spec.sections.skeleton=%d\n", skeleton)
	fmt.Fprintf(stdout, "spec.sections.empty=%d\n", empty)
	fmt.Fprintf(stdout, "spec.required.missing=%d\n", requiredMissing)
	fmt.Fprintf(stdout, "spec.followups.total=%d\n", len(followups))
	fmt.Fprintf(stdout, "spec.followups.open=%d\n", followupsOpen)
	fmt.Fprintf(stdout, "spec.lifecycle.failures=%d\n", lifecycle)
	fmt.Fprintf(stdout, "spec.requirements.ids=%d\n", len(parseRequirementIDs(sections["Requirements"])))
	fmt.Fprintf(stdout, "spec.requirements.duplicate_failures=%d\n", ridFailures)
	fmt.Fprintf(stdout, "spec.trace.present=%t\n", trace.HasSection)
	fmt.Fprintf(stdout, "spec.trace.entries=%d\n", traceEntries)
	fmt.Fprintf(stdout, "spec.trace.missing=%d\n", len(trace.Missing))
	fmt.Fprintf(stdout, "spec.trace.failures=%d\n", traceFailures)
	fmt.Fprintf(stdout, "spec.amendments.events=%d\n", amendEvents)
	fmt.Fprintf(stdout, "spec.amendments.failures=%d\n", amendFailures)

	if requiredMissing > 0 || lifecycle > 0 || ridFailures > 0 || traceFailures > 0 || amendFailures > 0 {
		return 1
	}
	return 0
}

func cmdLintSpec(args []string, stdout, stderr io.Writer) int {
	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose lint-spec: %v\n", err)
		return 2
	}
	return cmdLintSpecInRoot(root, args, stdout, stderr)
}

func cmdLintSpecInRoot(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	mode := "strict"
	requiredOnly, readyCheck := false, false
	target := ""
	for _, a := range args {
		switch a {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--required-only":
			requiredOnly = true
		case "--ready-check":
			readyCheck = true
		case "--all":
			target = "--all"
		case "-h", "--help":
			fmt.Fprintln(stdout, cliText(locale, "Usage: pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check]", "Uso: pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check]"))
			return 0
		default:
			if strings.HasPrefix(a, "--") {
				fmt.Fprintf(stderr, cliText(locale, "Error: unknown option: %s\n", "Erro: opção desconhecida: %s\n"), a)
				return 2
			}
			if target != "" {
				fmt.Fprintf(stderr, cliText(locale, "Error: unexpected argument: %s\n", "Erro: argumento extra: %s\n"), a)
				return 2
			}
			target = a
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: provide <slug> or --all", "Erro: informe <slug> ou --all"))
		return 2
	}
	specsDir := filepath.Join(root, ".pose", "specs")

	totalLinted, totalFailed := 0, 0
	var failedSpecs []string
	lintOne := func(path string, slug string) {
		totalLinted++
		fmt.Fprintln(stdout, "---")
		if rc := lintOneSpec(path, requiredOnly, readyCheck, stdout, stderr); rc != 0 {
			totalFailed++
			if slug == "" {
				slug = strings.TrimSuffix(filepath.Base(path), ".md")
				if filepath.Base(path) == "spec.md" {
					slug = filepath.Base(filepath.Dir(path))
				}
			}
			failedSpecs = append(failedSpecs, slug)
		}
	}

	if target == "--all" {
		store := posepkg.Store{Root: root}
		specs, err := store.ListSpecs("", "")
		if err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: specs directory not found: %s\n", "Erro: specs dir ausente: %s\n"), specsDir)
			return 2
		}
		for _, sp := range specs {
			specMD := sp.Path
			if !filepath.IsAbs(specMD) {
				specMD = filepath.Join(root, filepath.FromSlash(specMD))
			}
			if _, statErr := os.Stat(specMD); statErr == nil {
				lintOne(specMD, sp.Slug)
			}
		}
	}
	var closeoutHookErr error
	if target != "--all" {
		store := posepkg.Store{Root: root}
		sp, err := store.GetSpec(target)
		if err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: spec not found: %s\n", "Erro: spec não encontrada: %s\n"), target)
			return 2
		}
		specMD := sp.Path
		if !filepath.IsAbs(specMD) {
			specMD = filepath.Join(root, filepath.FromSlash(specMD))
		}
		lintOne(specMD, sp.Slug)
		if totalFailed == 0 && mode == "strict" {
			if fm, err := readFlatFrontmatter(specMD); err == nil && fm["status"] == "done" {
				closeoutHookErr = EmitHook(root, HookEvent{Kind: "spec_closeout", Target: target, Commit: gitHeadCommit(root), At: time.Now().UTC()})
			}
		}
	}

	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "lint.specs.checked=%d\n", totalLinted)
	fmt.Fprintf(stdout, "lint.specs.failed=%d\n", totalFailed)
	if totalFailed > 0 {
		findings := make([]usageFinding, 0, len(failedSpecs))
		for _, slug := range failedSpecs {
			findings = append(findings, usageFinding{ID: "spec:" + slug, Severity: "error"})
		}
		noteUsageFindings(stdout, "fail", findings, true)
		fmt.Fprintf(stdout, "Resultado: FALHA (%d spec(s) com seção obrigatória vazia/esquelética ou gate de ciclo de vida violado)\n", totalFailed)
		if mode == "strict" {
			PrintContributorFailureHint(root, stdout, locale)
			return 1
		}
		fmt.Fprintln(stdout, cliText(locale, "Tolerant mode: record a follow-up to complete specs.", "Modo tolerant: registrar follow-up para completar specs."))
		fmt.Fprintln(stdout, "Resultado: FALHA_TOLERADA")
		return 0
	}
	if closeoutHookErr != nil {
		fmt.Fprintf(stderr, "pose lint-spec: %v\n", closeoutHookErr)
		fmt.Fprintln(stdout, "Resultado: FALHA (state-refresh estrito falhou no pós-closeout)")
		return 1
	}
	noteUsageFindings(stdout, "pass", nil, true)
	fmt.Fprintln(stdout, "Resultado: SUCESSO")
	return 0
}

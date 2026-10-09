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
	"strconv"
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
	if knownSlugs != nil && dispositionSlugTargeted[disposition] && strings.HasPrefix(target, "xref:") {
		if reason := lintQualifiedFollowupTarget(target); reason != "" {
			return disposition, fmt.Sprintf(cliText(locale, "disposition [%s: %s] %s", "disposição [%s: %s] %s"), disposition, target, reason)
		}
		return disposition, ""
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

// lintQualifiedFollowupTarget checks a disposition target in another project
// (spec pose-followup-dispositions-accept-qualified-refs). What can be checked
// is: a bound project must hold the spec. An unbound project is not evidence
// that the spec is missing — the other repository is simply not here — so the
// target is accepted, and verified wherever both projects are bound.
func lintQualifiedFollowupTarget(target string) string {
	ref, err := posepkg.ParseArtifactRef(target)
	if err != nil {
		return "is not a valid qualified reference (xref:<project>/spec:<slug>)"
	}
	if ref.Kind != "spec" {
		return "names a " + ref.Kind + "; only a spec can take over a follow-up"
	}
	root, err := projectRoot()
	if err != nil {
		return ""
	}
	resolver, project, err := posepkg.EnvironmentArtifactResolver(root, "")
	if err != nil {
		return "cannot be checked: " + err.Error()
	}
	resolution := resolver.Resolve(project, target)
	switch {
	case resolution.Resolved:
		return ""
	case resolution.State == "unknown-project" || resolution.State == "unavailable-project" || resolution.State == "unauthorized-project":
		return ""
	default:
		return "does not resolve: " + resolution.State
	}
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

// completedBeforeCreated reports whether a completion precedes its creation.
// Two bare dates carry no zone: POSE stamps them in UTC (`new-spec`, `close`)
// while a person filling one in writes their local date, and the two differ by
// at most one day. A spec created at 23:30 in UTC-3 is stamped with tomorrow's
// UTC date and closed the same evening with today's local one, so one day of
// apparent regression is that skew, not an error. Instants are compared exactly
// (spec calendar-dates-tolerate-utc-stamping).
func completedBeforeCreated(created, completed time.Time, bothDates bool) bool {
	if bothDates {
		return completed.Before(created.AddDate(0, 0, -1))
	}
	return completed.Before(created)
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

// lintProjectRoot is the directory that holds the .pose tree of specPath.
func lintProjectRoot(specPath string) string {
	dir := filepath.Dir(specPath)
	for {
		if filepath.Base(dir) == ".pose" {
			return filepath.Dir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Dir(filepath.Dir(specPath))
		}
		dir = parent
	}
}

func lintOneSpec(specPath string, requiredOnly, readyCheck bool, stdout, stderr io.Writer, designCheck ...bool) int {
	return lintOneSpecWith(render(stdout, stderr), true, specPath, requiredOnly, readyCheck, stderr, len(designCheck) > 0 && designCheck[0])
}

// lintOneSpecWith lints one spec through r. A caller that lints many specs into
// one machine document passes fields=false: the per-spec `spec.*` fields would
// collide in the document's field map, and every finding already names its spec.
func lintOneSpecWith(r *cliout.Renderer, fields bool, specPath string, requiredOnly, readyCheck bool, stderr io.Writer, checkDesign bool) int {
	field := func(name, value string) {
		if fields {
			r.Field(name, value)
		}
	}
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
	lint := specFindings{r: r, slug: slug}
	if frontmatter["status"] == "superseded" {
		if canonical, ok := posepkg.VerifiedTransferStub(lintProjectRoot(specPath), slug, string(raw)); ok {
			// A verified redirect owns no requirements or lifecycle of its
			// own; the canonical task is linted where it lives.
			field("spec.redirect", canonical.String())
			return 0
		}
	}
	lineageFailures := 0
	if links := posepkg.RemediationValues(frontmatter["remediates"]); len(links) > 0 {
		root, lineageErr := projectRootAt(filepath.Dir(specPath))
		if lineageErr == nil {
			lineageErr = (posepkg.Store{Root: root}).ValidateRemediationLineage(posepkg.Spec{Slug: slug, Remediates: links})
		}
		if lineageErr != nil {
			lint.finding(cliout.StateError, "remediation-lineage", lineageErr.Error())
			lineageFailures++
		}
		field("spec.remediation.links", strconv.Itoa(len(links)))
		field("spec.remediation.failures", strconv.Itoa(lineageFailures))
	}
	designFailures := 0
	emitDesignBasis := func() {
		if !checkDesign {
			return
		}
		root, rootErr := projectRootAt(filepath.Dir(specPath))
		if rootErr != nil {
			root = ""
		}
		report := posepkg.ValidateDesignBasis(string(raw), root)
		// Buffered so the fields follow the findings, as they always have.
		designFields := [][2]string{}
		designFields = append(designFields, [2]string{"spec.design_basis.present", strconv.FormatBool(report.HasSection)})
		designFields = append(designFields, [2]string{"spec.design_basis.assumptions", strconv.Itoa(len(report.Assumptions))})
		designFields = append(designFields, [2]string{"spec.design_basis.decisions", strconv.Itoa(len(report.Decisions))})
		designFields = append(designFields, [2]string{"spec.design_basis.diagnostics", strconv.Itoa(len(report.Diagnostics))})
		errors, warnings := 0, 0
		for _, diagnostic := range report.Diagnostics {
			state := cliout.StateWarning
			if diagnostic.Severity == "error" {
				state = cliout.StateError
				errors++
			} else {
				warnings++
			}
			message := fmt.Sprintf("design basis line %d: %s", diagnostic.Line, diagnostic.Message)
			lint.finding(state, "design-basis/"+diagnostic.Code, message)
		}
		designFailures = errors
		designFields = append(designFields, [2]string{"spec.design_basis.errors", strconv.Itoa(errors)})
		designFields = append(designFields, [2]string{"spec.design_basis.warnings", strconv.Itoa(warnings)})
		if report.Digest != "" {
			designFields = append(designFields, [2]string{"spec.design_basis.digest", report.Digest})
		}
		for _, f := range designFields {
			field(f[0], f[1])
		}
	}

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
			if _, err := posepkg.ParseArtifactRef(ref); err == nil {
				continue
			}
			lint.finding(cliout.StateError, "dor", fmt.Sprintf(cliText(locale, "DoR: invalid depends_on reference: '%s'", "DoR: ref inválida em depends_on: '%s'"), ref))
			failures++
		}
		emitDesignBasis()
		failures += designFailures + lineageFailures
		ready := "true"
		if failures > 0 {
			ready = "false"
		}
		field("spec.ready", ready)
		field("spec.ready.failures", strconv.Itoa(failures))
		if failures > 0 {
			return 1
		}
		return 0
	}

	// A spec declares its surface (spec pose-progressive-spec-surface): a
	// minimal surface does not require the planner-local Tasks section. It
	// never relaxes the lifecycle gates below — trace, follow-ups and
	// amendments are checked the same way.
	required := posepkg.RequiredSpecSections(strings.TrimSpace(frontmatter["surface"]))
	targets := append([]string{}, required...)
	if !requiredOnly {
		targets = append(targets, optionalSections...)
		for _, name := range requiredSections {
			if !containsSection(targets, name) {
				targets = append(targets, name)
			}
		}
	}
	isRequired := map[string]bool{}
	for _, s := range required {
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
	// `blocked` is an operational condition, not an outcome (spec
	// pose-blocked-semantics-alignment). Without a declared prerequisite
	// nothing in the spec says what it waits on, so readiness can only report
	// `cause: unknown`. A warning, never a failure: legacy records stay valid.
	if specStatus == "blocked" && strings.TrimSpace(frontmatter["depends_on"]) == "" {
		lint.finding(cliout.StateWarning, "frontmatter", cliText(locale,
			"status: blocked records no cause (depends_on is empty), so readiness reports cause: unknown; declare the prerequisite in depends_on or describe the wait in Known gaps",
			"status: blocked não registra causa (depends_on vazio), então a readiness reporta cause: unknown; declare o pré-requisito em depends_on ou descreva a espera em Known gaps"))
	}

	// Lifecycle dates.
	parsed := map[string]time.Time{}
	dateOnly := map[string]bool{}
	for _, field := range []string{"created_at", "completed_at"} {
		value := strings.Trim(strings.TrimSpace(frontmatter[field]), `"'`)
		if value == "" {
			continue
		}
		if t, ok := parseISOInstant(value); ok {
			parsed[field] = t
			dateOnly[field] = len(value) == len("2006-01-02")
		} else {
			lint.finding(cliout.StateError, "frontmatter", fmt.Sprintf(cliText(locale, "%s must use ISO 8601: '%s'", "%s deve usar ISO 8601: '%s'"), field, value))
			lifecycle++
		}
	}
	if c, ok1 := parsed["created_at"]; ok1 {
		if d, ok2 := parsed["completed_at"]; ok2 && completedBeforeCreated(c, d, dateOnly["created_at"] && dateOnly["completed_at"]) {
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
	// A test: ref must name a test that exists (spec pose-trace-test-refs-resolve).
	// An unresolved ref blocks a spec that can still change, so an invented name
	// no longer closes; on a closed spec it is reported, because history is not
	// rewritten and the --all gate must not turn red on it. The root is the
	// spec's own repository, whatever the working directory.
	if root, err := projectRootAt(filepath.Dir(specPath)); err == nil {
		for _, ref := range posepkg.UnresolvedTestRefs(trace, posepkg.LoadTestCatalog(root)) {
			switch specStatus {
			case "done", "superseded", "abandoned":
				lint.finding(cliout.StateWarning, "requirement-trace", fmt.Sprintf(cliText(locale, "requirement trace: %s names no test in this repository or its submodules", "requirement trace: %s não nomeia nenhum teste deste repositório ou de seus submódulos"), ref))
			default:
				lint.finding(cliout.StateError, "requirement-trace", fmt.Sprintf(cliText(locale, "requirement trace: %s names no test in this repository or its submodules (cite the test function, subtest, test title or test file)", "requirement trace: %s não nomeia nenhum teste deste repositório ou de seus submódulos (cite a função de teste, o subteste, o título ou o arquivo de teste)"), ref))
				traceFailures++
			}
		}
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
			adopted, perr := posepkg.Store{Root: lintProjectRoot(specPath)}.ContractNodesAdopted()
			if perr != nil {
				lint.finding(cliout.StateError, "amendments", "review policy: "+perr.Error())
				amendFailures++
			}
			for _, finding := range posepkg.UnacknowledgedNodeChanges(slug, text, events, adopted) {
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

	field("spec.path", specPath)
	field("spec.status", specStatus)
	field("spec.sections.total", strconv.Itoa(total))
	field("spec.sections.filled", strconv.Itoa(filled))
	field("spec.sections.skeleton", strconv.Itoa(skeleton))
	field("spec.sections.empty", strconv.Itoa(empty))
	field("spec.required.missing", strconv.Itoa(requiredMissing))
	field("spec.followups.total", strconv.Itoa(len(followups)))
	field("spec.followups.open", strconv.Itoa(followupsOpen))
	field("spec.lifecycle.failures", strconv.Itoa(lifecycle))
	field("spec.requirements.ids", strconv.Itoa(len(parseRequirementIDs(sections["Requirements"]))))
	field("spec.requirements.duplicate_failures", strconv.Itoa(ridFailures))
	field("spec.trace.present", strconv.FormatBool(trace.HasSection))
	field("spec.trace.entries", strconv.Itoa(traceEntries))
	field("spec.trace.missing", strconv.Itoa(len(trace.Missing)))
	field("spec.trace.failures", strconv.Itoa(traceFailures))
	field("spec.amendments.events", strconv.Itoa(amendEvents))
	field("spec.amendments.failures", strconv.Itoa(amendFailures))
	emitDesignBasis()

	if requiredMissing > 0 || lifecycle > 0 || ridFailures > 0 || traceFailures > 0 || amendFailures > 0 || designFailures > 0 || lineageFailures > 0 {
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
	requiredOnly, readyCheck, designCheck := false, false, false
	asJSON, quiet := false, false
	jsonOut := ""
	colorMode := cliout.ColorAuto
	target := ""
	usage := cliText(locale, "Usage: pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check] [--design-check] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]",
		"Uso: pose lint-spec <slug>|--all [--strict|--tolerant] [--required-only] [--ready-check] [--design-check] [--json] [--json-out <path>] [--quiet] [--color auto|always|never]")
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--required-only":
			requiredOnly = true
		case "--ready-check":
			readyCheck = true
		case "--design-check":
			designCheck = true
		case "--all":
			target = "--all"
		case "--json":
			asJSON = true
		case "--quiet":
			quiet = true
		case "--json-out":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") || !confinedRelativePath(args[i+1]) {
				render(stdout, stderr).Failure(cliText(locale, "--json-out requires a relative path inside the project", "--json-out exige um caminho relativo dentro do projeto"))
				return 2
			}
			i++
			jsonOut = args[i]
		case "--color":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, cliText(locale, "Error: %s requires a value.\n", "Erro: %s exige um valor.\n"), a)
				return 2
			}
			i++
			parsed, ok := cliout.ParseColorMode(args[i])
			if !ok {
				render(stdout, stderr).UnknownToken(cliText(locale, "value", "valor"), args[i], []string{"auto", "always", "never"})
				return 2
			}
			colorMode = parsed
		case "-h", "--help":
			fmt.Fprintln(stdout, usage)
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

	// out carries the result. With --json it records one document; the
	// per-spec lint writes into it, keeping its fields only for a single spec.
	// With --quiet the per-spec output is discarded and only the verdict line
	// is printed; progress and warnings on stderr are untouched.
	out := renderWithColor(stdout, stderr, colorMode)
	switch {
	case asJSON:
		out.RecordJSON("lint-spec")
	case jsonOut != "":
		out.RecordTee("lint-spec")
	}
	human := !asJSON && !quiet
	perSpec := out
	if quiet && !asJSON {
		perSpec = cliout.NewPlain(io.Discard, stderr)
	}
	result := stdout
	if !human {
		result = io.Discard
	}

	totalLinted, totalFailed := 0, 0
	var failedSpecs []string
	lintOne := func(path string, slug string) {
		totalLinted++
		fmt.Fprintln(result, "---")
		if rc := lintOneSpecWith(perSpec, !asJSON || target != "--all", path, requiredOnly, readyCheck, stderr, designCheck); rc != 0 {
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

	// The summary and verdict lines below are pinned contract lines; the
	// machine document carries the same facts as counts and a verdict.
	out.RecordCount("specs_checked", totalLinted)
	out.RecordCount("specs_failed", totalFailed)
	out.RecordField("mode", mode)
	defer func() {
		if jsonOut != "" {
			if err := writeJSONOut(out, root, jsonOut); err != nil {
				out.Failure(err.Error())
			}
		}
		_ = out.FlushJSON()
	}()
	fmt.Fprintln(result)
	fmt.Fprintf(result, "lint.specs.checked=%d\n", totalLinted)
	fmt.Fprintf(result, "lint.specs.failed=%d\n", totalFailed)
	verdict := func(state cliout.State, line, text string) {
		word := ""
		if state == cliout.StateWarning {
			word = "TOLERATED_FAILURE"
		}
		if asJSON {
			out.Verdict(cliout.Verdict{State: state, Word: word, Text: text})
			return
		}
		out.RecordVerdict(cliout.Verdict{State: state, Word: word, Text: text})
		fmt.Fprintln(stdout, line)
	}
	if totalFailed > 0 {
		findings := make([]usageFinding, 0, len(failedSpecs))
		for _, slug := range failedSpecs {
			findings = append(findings, usageFinding{ID: "spec:" + slug, Severity: "error"})
		}
		noteUsageFindings(stdout, "fail", findings, true)
		if mode == "strict" {
			verdict(cliout.StateFail, fmt.Sprintf("Resultado: FALHA (%d spec(s) com seção obrigatória vazia/esquelética ou gate de ciclo de vida violado)", totalFailed),
				fmt.Sprintf(cliText(locale, "%d spec(s) with an empty or skeletal required section or a violated lifecycle gate", "%d spec(s) com seção obrigatória vazia/esquelética ou gate de ciclo de vida violado"), totalFailed))
			if human {
				PrintContributorFailureHint(root, stderr, locale)
			}
			return 1
		}
		verdict(cliout.StateWarning, fmt.Sprintf("Resultado: FALHA (%d spec(s) com seção obrigatória vazia/esquelética ou gate de ciclo de vida violado)", totalFailed),
			fmt.Sprintf(cliText(locale, "%d spec(s) failed; tolerated in tolerant mode", "%d spec(s) falharam; tolerado no modo tolerant"), totalFailed))
		if !asJSON {
			fmt.Fprintln(result, cliText(locale, "Tolerant mode: record a follow-up to complete specs.", "Modo tolerant: registrar follow-up para completar specs."))
			fmt.Fprintln(stdout, "Resultado: FALHA_TOLERADA")
		}
		return 0
	}
	if closeoutHookErr != nil {
		fmt.Fprintf(stderr, "pose lint-spec: %v\n", closeoutHookErr)
		verdict(cliout.StateFail, "Resultado: FALHA (state-refresh estrito falhou no pós-closeout)",
			cliText(locale, "the strict state refresh after closeout failed", "state-refresh estrito falhou no pós-closeout"))
		return 1
	}
	noteUsageFindings(stdout, "pass", nil, true)
	verdict(cliout.StatePass, "Resultado: SUCESSO", "")
	return 0
}

func containsSection(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

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

func extractFollowups(finalReport []string) []string {
	var bullets []string
	in := false
	for _, line := range finalReport {
		if m := subheadingRE.FindStringSubmatch(line); m != nil {
			in = strings.HasPrefix(strings.ToLower(strings.TrimSpace(m[1])), "follow-up")
			continue
		}
		if !in {
			continue
		}
		if m := bulletRE.FindStringSubmatch(line); m != nil {
			bullets = append(bullets, strings.TrimSpace(m[1]))
		}
	}
	return bullets
}

func collectSpecSlugs(specsDir string) map[string]bool {
	slugs := map[string]bool{}
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return slugs
	}
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(specsDir, e.Name(), "spec.md")); err == nil {
				slugs[e.Name()] = true
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

func siblingSpecStatus(specsDir, slug string) string {
	b, err := os.ReadFile(filepath.Join(specsDir, slug, "spec.md"))
	if err != nil {
		return ""
	}
	return lintParseFrontmatter(string(b))["status"]
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
	slug := filepath.Base(filepath.Dir(specPath))

	if readyCheck {
		failures := 0
		for _, name := range []string{"Intent", "Requirements", "Technical Plan"} {
			lines, ok := sections[name]
			if !ok || classifySection(lines) != "filled" {
				fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: DoR: section %s is missing, empty, or skeletal\n", "[ERRO] %s: DoR: seção %s ausente/vazia/esquelética\n"), slug, name)
				failures++
			}
		}
		if len(parseRequirementIDs(sections["Requirements"])) == 0 {
			fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: DoR: no acceptance criterion has a stable ID (use '- R<N>: ...' bullets in Requirements)\n", "[ERRO] %s: DoR: nenhum acceptance criterion com ID estável (use bullets '- R<N>: ...' em Requirements)\n"), slug)
			failures++
		}
		for _, ref := range lintParseDependsOn(frontmatter["depends_on"]) {
			if depSlugRE.MatchString(ref) || depMilestoneRE.MatchString(ref) || depRoadmapRE.MatchString(ref) {
				continue
			}
			fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: DoR: invalid depends_on reference: '%s'\n", "[ERRO] %s: DoR: ref inválida em depends_on: '%s'\n"), slug, ref)
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
				fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: required section missing: %s\n", "[ERRO] %s: seção obrigatória ausente: %s\n"), slug, name)
				requiredMissing++
			} else {
				fmt.Fprintf(stderr, cliText(locale, "[WARNING] %s: optional section missing: %s\n", "[AVISO] %s: seção opcional ausente: %s\n"), slug, name)
			}
			continue
		}
		total++
		level := cliText(locale, "WARNING", "AVISO")
		if isRequired[name] {
			level = cliText(locale, "ERROR", "ERRO")
		}
		switch classifySection(lines) {
		case "filled":
			filled++
		case "skeleton":
			skeleton++
			fmt.Fprintf(stderr, cliText(locale, "[%s] %s: %s: skeletal (placeholders or comments only)\n", "[%s] %s: %s: esqueleto (apenas placeholders/comentários)\n"), level, slug, name)
			if isRequired[name] {
				requiredMissing++
			}
		default:
			empty++
			fmt.Fprintf(stderr, cliText(locale, "[%s] %s: %s: empty\n", "[%s] %s: %s: vazia\n"), level, slug, name)
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
		fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: invalid frontmatter status: '%s' (use draft|in-progress|done|blocked|superseded|abandoned)\n", "[ERRO] %s: status inválido no frontmatter: '%s' (use draft|in-progress|done|blocked|superseded|abandoned)\n"), slug, specStatus)
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
			fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: %s must use ISO 8601: '%s'\n", "[ERRO] %s: %s deve usar ISO 8601: '%s'\n"), slug, field, value)
			lifecycle++
		}
	}
	if c, ok1 := parsed["created_at"]; ok1 {
		if d, ok2 := parsed["completed_at"]; ok2 && d.Before(c) {
			fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: completed_at is earlier than created_at\n", "[ERRO] %s: completed_at anterior a created_at\n"), slug)
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
		fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: duplicate canonical heading: %s appears %d times\n", "[ERRO] %s: heading canônico duplicado: %s aparece %d vezes\n"), slug, n, nameCount[n])
		lifecycle++
	}
	if followupHeadings > 1 {
		fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: duplicate canonical heading: Follow-ups appears %d times\n", "[ERRO] %s: heading canônico duplicado: Follow-ups aparece %d vezes\n"), slug, followupHeadings)
		lifecycle++
	}

	specsDir := filepath.Dir(filepath.Dir(specPath))
	if specStatus == "in-progress" {
		for _, dep := range lintParseDependsOn(frontmatter["depends_on"]) {
			if strings.Contains(dep, ":") {
				continue
			}
			if st := siblingSpecStatus(specsDir, dep); st != "" && st != "done" {
				fmt.Fprintf(stderr, cliText(locale, "[WARNING] %s: in-progress with unsatisfied dependency: '%s' (status: %s)\n", "[AVISO] %s: in-progress com dependência não satisfeita: '%s' (status: %s)\n"), slug, dep, st)
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
		fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: duplicate R-ID: %s appears %d times in Requirements\n", "[ERRO] %s: R-ID duplicado: %s aparece %d vezes em Requirements\n"), slug, id, seen[id])
		ridFailures++
	}

	followups := extractFollowups(sections["Final Report"])
	followupsOpen := 0
	knownSlugs := collectSpecSlugs(specsDir)

	if specStatus == "done" {
		if strings.TrimSpace(frontmatter["completed_at"]) == "" {
			fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: status: done requires populated 'completed_at' frontmatter\n", "[ERRO] %s: status: done exige 'completed_at' preenchido no frontmatter\n"), slug)
			lifecycle++
		}
		for _, content := range followups {
			disposition, errMsg := lintFollowupDisposition(content, knownSlugs, slug, locale)
			if errMsg != "" {
				snippet := content
				if len([]rune(snippet)) > 60 {
					snippet = string([]rune(snippet)[:60]) + "…"
				}
				fmt.Fprintf(stderr, cliText(locale, "[ERROR] %s: follow-up lacks a valid disposition: %s → \"%s\"\n", "[ERRO] %s: follow-up sem disposição válida: %s → \"%s\"\n"), slug, errMsg, snippet)
				lifecycle++
			} else if disposition == "open" {
				followupsOpen++
			}
		}
	} else {
		for _, content := range followups {
			disposition, _ := lintFollowupDisposition(content, nil, "", locale)
			if disposition == "open" || disposition == "" {
				followupsOpen++
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

	if requiredMissing > 0 || lifecycle > 0 || ridFailures > 0 {
		return 1
	}
	return 0
}

// cmdLintSpec mirrors pose-lint-spec.sh: <slug>|--all, --strict|--tolerant,
// --required-only, --ready-check; aggregate lines and Resultado semantics.
func cmdLintSpec(args []string, stdout, stderr io.Writer) int {
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
	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose lint-spec: %v\n", err)
		return 2
	}
	specsDir := filepath.Join(root, ".pose", "specs")

	totalLinted, totalFailed := 0, 0
	lintOne := func(path string) {
		totalLinted++
		fmt.Fprintln(stdout, "---")
		if rc := lintOneSpec(path, requiredOnly, readyCheck, stdout, stderr); rc != 0 {
			totalFailed++
		}
	}

	if target == "--all" {
		entries, err := os.ReadDir(specsDir)
		if err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: specs directory not found: %s\n", "Erro: specs dir ausente: %s\n"), specsDir)
			return 2
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			specMD := filepath.Join(specsDir, name, "spec.md")
			if _, err := os.Stat(specMD); err == nil {
				lintOne(specMD)
			}
		}
		for _, name := range names {
			if _, err := os.Stat(filepath.Join(specsDir, name, "spec.md")); err != nil {
				fmt.Fprintln(stdout, "---")
				fmt.Fprintf(stderr, cliText(locale, "[WARNING] %s: no consolidated spec.md (pre-unified-template format)\n", "[AVISO] %s: sem spec.md consolidado (formato pré-template-único)\n"), name)
			}
		}
	} else {
		specMD := filepath.Join(specsDir, target, "spec.md")
		if _, err := os.Stat(specMD); err != nil {
			legacy := filepath.Join(specsDir, target+".md")
			if _, err := os.Stat(legacy); err == nil {
				specMD = legacy
			} else {
				fmt.Fprintf(stderr, cliText(locale, "Error: spec not found: %s\n", "Erro: spec não encontrada: %s\n"), specMD)
				return 2
			}
		}
		lintOne(specMD)
	}

	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "lint.specs.checked=%d\n", totalLinted)
	fmt.Fprintf(stdout, "lint.specs.failed=%d\n", totalFailed)
	if totalFailed > 0 {
		fmt.Fprintf(stdout, "Resultado: FALHA (%d spec(s) com seção obrigatória vazia/esquelética ou gate de ciclo de vida violado)\n", totalFailed)
		if mode == "strict" {
			return 1
		}
		fmt.Fprintln(stdout, cliText(locale, "Tolerant mode: record a follow-up to complete specs.", "Modo tolerant: registrar follow-up para completar specs."))
		fmt.Fprintln(stdout, "Resultado: FALHA_TOLERADA")
		return 0
	}
	fmt.Fprintln(stdout, "Resultado: SUCESSO")
	return 0
}

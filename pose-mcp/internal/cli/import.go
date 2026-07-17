package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	importMaxFiles   = 1000
	importMaxEntries = 10000
	importMaxBytes   = 10 << 20
)

var (
	importNow       = time.Now
	importWriteFile = os.WriteFile
)

type importOptions struct {
	format string
	source string
	dryRun bool
}

type importUnit struct {
	format       string
	source       string
	slug         string
	title        string
	intent       string
	requirements []string
	plan         string
	tasks        string
	artifacts    []string
	warnings     []string
}

type markdownSection struct {
	level int
	title string
	body  string
}

func cmdImport(args []string, stdout, stderr io.Writer) int {
	opts, err := parseImportOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "pose import: %v\n", err)
		fmt.Fprintln(stderr, "usage: pose import <spec-kit|openspec> <path> [--dry-run]")
		return 2
	}

	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose import: resolve project root: %v\n", err)
		return 1
	}
	units, err := loadImportUnits(opts)
	if err != nil {
		fmt.Fprintf(stderr, "pose import: %v\n", err)
		return 1
	}
	if err := preflightImport(root, units); err != nil {
		fmt.Fprintf(stderr, "pose import: %v\n", err)
		return 1
	}

	rendered := make([]string, len(units))
	for i := range units {
		rendered[i] = renderImportedSpec(units[i])
	}
	if !opts.dryRun {
		if err := writeImportedSpecs(root, units, rendered); err != nil {
			fmt.Fprintf(stderr, "pose import: %v\n", err)
			return 1
		}
	}

	warningCount := 0
	for _, unit := range units {
		destination := filepath.ToSlash(filepath.Join(".pose", "specs", unit.slug, "spec.md"))
		action := "write"
		if opts.dryRun {
			action = "dry-run"
		}
		fmt.Fprintf(stdout, "import.spec slug=%s format=%s source=%s destination=%s requirements=%d artifacts=%d action=%s\n",
			unit.slug, unit.format, filepath.ToSlash(unit.source), destination, len(unit.requirements), len(unit.artifacts), action)
		for _, warning := range unit.warnings {
			warningCount++
			fmt.Fprintf(stdout, "import.curation slug=%s warning=%s\n", unit.slug, quoteReportValue(warning))
		}
	}
	written := len(units)
	if opts.dryRun {
		written = 0
	}
	fmt.Fprintf(stdout, "import.summary specs=%d warnings=%d written=%d dry_run=%t\n",
		len(units), warningCount, written, opts.dryRun)
	return 0
}

func parseImportOptions(args []string) (importOptions, error) {
	var opts importOptions
	positional := make([]string, 0, 2)
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			if opts.dryRun {
				return opts, errors.New("--dry-run specified more than once")
			}
			opts.dryRun = true
		default:
			if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("unknown option %q", arg)
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) != 2 {
		return opts, errors.New("format and path are required")
	}
	opts.format, opts.source = positional[0], positional[1]
	if opts.format != "spec-kit" && opts.format != "openspec" {
		return opts, fmt.Errorf("unsupported format %q (expected spec-kit or openspec)", opts.format)
	}
	return opts, nil
}

func loadImportUnits(opts importOptions) ([]importUnit, error) {
	source, err := filepath.Abs(opts.source)
	if err != nil {
		return nil, fmt.Errorf("resolve source: %w", err)
	}
	info, err := os.Lstat(source)
	if err != nil {
		return nil, fmt.Errorf("inspect source %q: %w", opts.source, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("source %q is a symlink", opts.source)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("source %q is not a regular file or directory", opts.source)
	}

	paths, err := discoverImportSpecs(opts.format, source, info)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no supported %s specs found under %q", opts.format, opts.source)
	}
	if len(paths) > importMaxFiles {
		return nil, fmt.Errorf("input contains %d specs; limit is %d", len(paths), importMaxFiles)
	}

	sourceBase := source
	if info.Mode().IsRegular() {
		sourceBase = filepath.Dir(source)
	}
	budget := int64(importMaxBytes)
	units := make([]importUnit, 0, len(paths))
	for _, specPath := range paths {
		var unit importUnit
		switch opts.format {
		case "spec-kit":
			unit, err = loadSpecKitUnit(sourceBase, specPath, &budget)
		case "openspec":
			unit, err = loadOpenSpecUnit(sourceBase, specPath, &budget)
		}
		if err != nil {
			return nil, err
		}
		units = append(units, unit)
	}
	sort.Slice(units, func(i, j int) bool { return units[i].slug < units[j].slug })
	return units, nil
}

func discoverImportSpecs(format, source string, info os.FileInfo) ([]string, error) {
	if info.Mode().IsRegular() {
		if strings.ToLower(filepath.Base(source)) != "spec.md" {
			return nil, fmt.Errorf("source file must be named spec.md")
		}
		return []string{source}, nil
	}

	direct := filepath.Join(source, "spec.md")
	if directInfo, err := os.Lstat(direct); err == nil {
		if directInfo.Mode()&os.ModeSymlink != 0 || !directInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", direct)
		}
		return []string{direct}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect %s: %w", direct, err)
	}

	searchRoot := source
	if format == "openspec" {
		if _, err := os.Stat(filepath.Join(source, "proposal.md")); err == nil {
			searchRoot = filepath.Join(source, "specs")
		} else if info, err := os.Stat(filepath.Join(source, "specs")); err == nil && info.IsDir() {
			searchRoot = filepath.Join(source, "specs")
		}
	} else if info, err := os.Stat(filepath.Join(source, "specs")); err == nil && info.IsDir() {
		searchRoot = filepath.Join(source, "specs")
	}

	var paths []string
	entryCount := 0
	markdownCount := 0
	err := filepath.WalkDir(searchRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entryCount++
		if entryCount > importMaxEntries {
			return fmt.Errorf("input contains more than %d filesystem entries", importMaxEntries)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed: %s", path)
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			markdownCount++
			if markdownCount > importMaxFiles {
				return fmt.Errorf("input contains more than %d Markdown files", importMaxFiles)
			}
		}
		if !entry.IsDir() && strings.EqualFold(entry.Name(), "spec.md") {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("spec is not a regular file: %s", path)
			}
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover specs: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func loadSpecKitUnit(sourceRoot, specPath string, budget *int64) (importUnit, error) {
	content, err := readImportFile(specPath, budget)
	if err != nil {
		return importUnit{}, err
	}
	sections, h1 := parseMarkdown(content)
	name := strings.TrimSpace(strings.TrimPrefix(h1, "Feature Specification:"))
	if name == "" {
		name = filepath.Base(filepath.Dir(specPath))
	}
	unit := importUnit{
		format: "spec-kit", source: relativeImportPath(sourceRoot, specPath),
		slug: normalizeImportSlug(filepath.Base(filepath.Dir(specPath))), title: name,
		artifacts: []string{relativeImportPath(sourceRoot, specPath)},
	}
	unit.intent = firstNonEmpty(
		sectionBody(sections, "User Scenarios & Testing"),
		metadataValue(content, "Input"),
		fmt.Sprintf("Imported feature %q from spec-kit.", name),
	)
	unit.requirements = specKitRequirements(content)
	unit.warnings = append(unit.warnings, unmappedTopLevelSections(sections, func(title string) bool {
		switch strings.ToLower(title) {
		case "user scenarios & testing", "requirements", "success criteria", "assumptions":
			return true
		default:
			return false
		}
	})...)
	if len(unit.requirements) == 0 {
		unit.requirements = []string{"Confirm the intended behavior from the imported spec-kit narrative; no FR-* requirement was present in the source."}
		unit.warnings = append(unit.warnings, "no FR-* requirements found; curate the generated R1 criterion")
	}

	featureDir := filepath.Dir(specPath)
	planParts := []string{}
	if body := sectionBody(sections, "Success Criteria"); body != "" {
		planParts = append(planParts, "### Imported Success Criteria\n\n"+body)
	}
	if body := sectionBody(sections, "Assumptions"); body != "" {
		planParts = append(planParts, "### Imported Assumptions\n\n"+body)
	}
	if companion, ok, err := readOptionalImportFile(filepath.Join(featureDir, "plan.md"), budget); err != nil {
		return importUnit{}, err
	} else if ok {
		unit.artifacts = append(unit.artifacts, relativeImportPath(sourceRoot, filepath.Join(featureDir, "plan.md")))
		planParts = append(planParts, "### Imported Implementation Plan\n\n"+companion)
	} else {
		unit.warnings = append(unit.warnings, "plan.md not found; curate the Technical Plan")
	}
	unit.plan = strings.Join(planParts, "\n\n")
	if unit.plan == "" {
		unit.plan = "The source did not provide a structured implementation plan. Curate affected areas, contracts, storage changes, and technical risks before execution."
	}
	if companion, ok, err := readOptionalImportFile(filepath.Join(featureDir, "tasks.md"), budget); err != nil {
		return importUnit{}, err
	} else if ok {
		unit.artifacts = append(unit.artifacts, relativeImportPath(sourceRoot, filepath.Join(featureDir, "tasks.md")))
		unit.tasks = companion
	} else {
		unit.tasks = "- [ ] Curate implementation tasks from the imported requirements.\n- [ ] Define deterministic validation for the imported scope."
		unit.warnings = append(unit.warnings, "tasks.md not found; curate the generated task checklist")
	}
	return validateImportUnit(unit)
}

func loadOpenSpecUnit(sourceRoot, specPath string, budget *int64) (importUnit, error) {
	content, err := readImportFile(specPath, budget)
	if err != nil {
		return importUnit{}, err
	}
	sections, h1 := parseMarkdown(content)
	capability := filepath.Base(filepath.Dir(specPath))
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(h1, "Specification"), "Spec"))
	if name == "" {
		name = capability
	}
	unit := importUnit{
		format: "openspec", source: relativeImportPath(sourceRoot, specPath),
		slug: normalizeImportSlug(capability), title: name,
		artifacts: []string{relativeImportPath(sourceRoot, specPath)},
		intent:    firstNonEmpty(sectionBody(sections, "Purpose"), fmt.Sprintf("Imported capability %q from OpenSpec.", name)),
	}
	unit.requirements = openSpecRequirements(sections)
	unit.warnings = append(unit.warnings, unmappedTopLevelSections(sections, func(title string) bool {
		upper := strings.ToUpper(title)
		return strings.EqualFold(title, "Purpose") || strings.HasSuffix(upper, " REQUIREMENTS") || strings.EqualFold(title, "Requirements")
	})...)
	if len(unit.requirements) == 0 {
		return importUnit{}, fmt.Errorf("OpenSpec file %s has no '### Requirement:' sections", specPath)
	}

	changeRoot := findOpenSpecChangeRoot(specPath)
	planParts := []string{}
	if changeRoot != "" {
		changeName := filepath.Base(changeRoot)
		unit.slug = normalizeImportSlug(changeName + "-" + capability)
		for _, artifact := range []string{"proposal.md", "design.md"} {
			path := filepath.Join(changeRoot, artifact)
			if companion, ok, err := readOptionalImportFile(path, budget); err != nil {
				return importUnit{}, err
			} else if ok {
				unit.artifacts = append(unit.artifacts, relativeImportPath(sourceRoot, path))
				if artifact == "proposal.md" {
					unit.intent = companion
				} else {
					planParts = append(planParts, "### Imported Design\n\n"+companion)
				}
			}
		}
		tasksPath := filepath.Join(changeRoot, "tasks.md")
		if companion, ok, err := readOptionalImportFile(tasksPath, budget); err != nil {
			return importUnit{}, err
		} else if ok {
			unit.artifacts = append(unit.artifacts, relativeImportPath(sourceRoot, tasksPath))
			unit.tasks = companion
		}
	}
	if len(planParts) == 0 {
		unit.plan = "The OpenSpec source describes behavior but does not provide a design artifact. Curate affected areas, contracts, storage changes, and technical risks before execution."
		unit.warnings = append(unit.warnings, "design.md not found; curate the Technical Plan")
	} else {
		unit.plan = strings.Join(planParts, "\n\n")
	}
	if unit.tasks == "" {
		unit.tasks = "- [ ] Derive implementation tasks from the imported behavioral requirements.\n- [ ] Define deterministic validation for every imported scenario."
		unit.warnings = append(unit.warnings, "tasks.md not found; curate the generated task checklist")
	}
	return validateImportUnit(unit)
}

func parseMarkdown(content string) ([]markdownSection, string) {
	heading := regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)[ \t]*#*[ \t]*$`)
	var sections []markdownSection
	var current *markdownSection
	var body strings.Builder
	h1 := ""
	flush := func() {
		if current == nil {
			return
		}
		current.body = strings.TrimSpace(body.String())
		sections = append(sections, *current)
		body.Reset()
	}
	for _, line := range strings.Split(content, "\n") {
		match := heading.FindStringSubmatch(line)
		if match == nil {
			if current != nil {
				body.WriteString(line)
				body.WriteByte('\n')
			}
			continue
		}
		flush()
		current = &markdownSection{level: len(match[1]), title: strings.TrimSpace(match[2])}
		if current.level == 1 && h1 == "" {
			h1 = current.title
		}
	}
	flush()
	return sections, h1
}

func specKitRequirements(content string) []string {
	re := regexp.MustCompile(`(?m)^[ \t]*[-*][ \t]+(FR-[0-9]+):[ \t]*(.+?)[ \t]*$`)
	matches := re.FindAllStringSubmatch(content, -1)
	requirements := make([]string, 0, len(matches))
	for _, match := range matches {
		requirements = append(requirements, match[1]+" — "+match[2])
	}
	return requirements
}

func openSpecRequirements(sections []markdownSection) []string {
	var requirements []string
	delta := "CURRENT"
	for i, section := range sections {
		upper := strings.ToUpper(section.title)
		if section.level == 2 && strings.HasSuffix(upper, " REQUIREMENTS") {
			delta = strings.TrimSuffix(upper, " REQUIREMENTS")
		}
		if section.level != 3 || !strings.HasPrefix(strings.ToLower(section.title), "requirement:") {
			continue
		}
		name := strings.TrimSpace(section.title[len("Requirement:"):])
		bodyParts := []string{section.body}
		for j := i + 1; j < len(sections) && sections[j].level > section.level; j++ {
			child := sections[j]
			bodyParts = append(bodyParts, child.title+": "+child.body)
		}
		body := strings.Join(strings.Fields(strings.Join(bodyParts, " ")), " ")
		if body == "" {
			body = "Source requirement body is empty; curate this criterion."
		}
		requirements = append(requirements, fmt.Sprintf("[%s] %s — %s", delta, name, body))
	}
	return requirements
}

func renderImportedSpec(unit importUnit) string {
	date := importNow().UTC().Format("2006-01-02")
	var b strings.Builder
	fmt.Fprintf(&b, "---\nslug: %s\nstatus: draft\ncreated_at: %s\ncompleted_at:\nsupersedes:\ndepends_on:\npriority: 2\n---\n\n", unit.slug, date)
	fmt.Fprintf(&b, "# Spec: %s\n\n> Imported from %s. Review the curation notes before changing status.\n\n---\n\n", unit.title, unit.format)
	fmt.Fprintf(&b, "## 1. Intent\n\n### Objective and source context\n\n%s\n\n### Constraints\n\n- Preserve observable behavior captured by the source artifacts.\n- Resolve every item listed under Import Provenance before execution.\n\n### Non-goals\n\n- Synchronize subsequent source changes automatically.\n\n---\n\n", unit.intent)
	b.WriteString("## 2. Requirements\n\n### Functional\n\n")
	for i, requirement := range unit.requirements {
		fmt.Fprintf(&b, "- R%d: %s\n", i+1, requirement)
	}
	b.WriteString("\n### Non-functional\n\n- Preserve deterministic, reviewable behavior for this imported scope.\n\n### Security\n\n- Review source-specific trust boundaries and sensitive data before implementation.\n\n### Compatibility\n\n- Preserve compatibility constraints stated in the imported artifacts.\n\n---\n\n")
	fmt.Fprintf(&b, "## 3. Technical Plan\n\n%s\n\n---\n\n", unit.plan)
	fmt.Fprintf(&b, "## 4. Tasks\n\n%s\n\n---\n\n", unit.tasks)
	b.WriteString("## 5. Decisions\n\nNo implementation decision was imported. Record material trade-offs here during curation.\n\n---\n\n")
	b.WriteString("## 6. Validation\n\n### Strategy\n\nMap deterministic checks to every imported requirement before implementation.\n\n### Checks\n\n- Test: define the module test command.\n- Lint: define the module lint command.\n- Build: define the module build command when applicable.\n- Security / contract: define checks for affected boundaries.\n\n---\n\n")
	b.WriteString("## 7. Final Report\n\n### Delivered scope\n\nNot delivered; this imported spec remains draft.\n\n### Residual risks\n\n- Imported semantics require human review before execution.\n\n### Follow-ups\n\n- [open] Complete import curation and run `pose lint-spec ")
	b.WriteString(unit.slug)
	b.WriteString(" --ready-check` before marking this spec in-progress.\n\n---\n\n")
	b.WriteString("## 8. Import Provenance\n\n")
	fmt.Fprintf(&b, "- Format: `%s`\n- Source: `%s`\n- Artifacts consumed:\n", unit.format, filepath.ToSlash(unit.source))
	for _, artifact := range unit.artifacts {
		fmt.Fprintf(&b, "  - `%s`\n", filepath.ToSlash(artifact))
	}
	if len(unit.warnings) == 0 {
		b.WriteString("- Curation notes: no structural gaps detected.\n")
	} else {
		b.WriteString("- Curation notes:\n")
		for _, warning := range unit.warnings {
			fmt.Fprintf(&b, "  - %s\n", warning)
		}
	}
	return b.String()
}

func preflightImport(root string, units []importUnit) error {
	seen := make(map[string]struct{}, len(units))
	for _, unit := range units {
		if _, ok := seen[unit.slug]; ok {
			return fmt.Errorf("multiple source specs normalize to slug %q", unit.slug)
		}
		seen[unit.slug] = struct{}{}
		destination := filepath.Join(root, ".pose", "specs", unit.slug, "spec.md")
		if _, err := os.Lstat(destination); err == nil {
			return fmt.Errorf("destination already exists: %s", destination)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect destination %s: %w", destination, err)
		}
	}
	return nil
}

func writeImportedSpecs(root string, units []importUnit, rendered []string) error {
	createdDirs := make([]string, 0, len(units))
	rollback := func() {
		for i := len(createdDirs) - 1; i >= 0; i-- {
			_ = os.RemoveAll(createdDirs[i])
		}
	}
	for i, unit := range units {
		dir := filepath.Join(root, ".pose", "specs", unit.slug)
		if err := os.Mkdir(dir, 0o755); err != nil {
			rollback()
			return fmt.Errorf("create destination %s: %w", dir, err)
		}
		createdDirs = append(createdDirs, dir)
		if err := importWriteFile(filepath.Join(dir, "spec.md"), []byte(rendered[i]), 0o644); err != nil {
			rollback()
			return fmt.Errorf("write spec %s: %w", unit.slug, err)
		}
	}
	return nil
}

func validateImportUnit(unit importUnit) (importUnit, error) {
	if unit.slug == "" {
		return importUnit{}, fmt.Errorf("source %s does not produce a valid ASCII slug", unit.source)
	}
	if unit.title == "" || unit.intent == "" || len(unit.requirements) == 0 {
		return importUnit{}, fmt.Errorf("source %s is missing required imported content", unit.source)
	}
	sort.Strings(unit.artifacts)
	return unit, nil
}

func readImportFile(path string, budget *int64) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect artifact %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("artifact is not a regular file: %s", path)
	}
	if info.Size() > *budget {
		return "", fmt.Errorf("input exceeds %d-byte limit while reading %s", importMaxBytes, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read artifact %s: %w", path, err)
	}
	*budget -= int64(len(content))
	return string(content), nil
}

func readOptionalImportFile(path string, budget *int64) (string, bool, error) {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("inspect artifact %s: %w", path, err)
	}
	content, err := readImportFile(path, budget)
	return content, err == nil, err
}

func sectionBody(sections []markdownSection, title string) string {
	for i, section := range sections {
		if strings.EqualFold(section.title, title) {
			parts := []string{section.body}
			for j := i + 1; j < len(sections) && sections[j].level > section.level; j++ {
				child := sections[j]
				parts = append(parts, strings.Repeat("#", child.level)+" "+child.title+"\n\n"+child.body)
			}
			return strings.TrimSpace(strings.Join(parts, "\n\n"))
		}
	}
	return ""
}

func unmappedTopLevelSections(sections []markdownSection, known func(string) bool) []string {
	var warnings []string
	for _, section := range sections {
		if section.level == 2 && !known(section.title) {
			warnings = append(warnings, fmt.Sprintf("unmapped source section %q; review it in the original artifact", section.title))
		}
	}
	return warnings
}

func metadataValue(content, key string) string {
	re := regexp.MustCompile(`(?m)^\*{0,2}` + regexp.QuoteMeta(key) + `\*{0,2}:[ \t]*(.+)$`)
	if match := re.FindStringSubmatch(content); match != nil {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeImportSlug(value string) string {
	value = regexp.MustCompile(`^[0-9]+[-_]`).ReplaceAllString(value, "")
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(value) {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func relativeImportPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return rel
}

func findOpenSpecChangeRoot(specPath string) string {
	dir := filepath.Dir(specPath)
	for {
		if filepath.Base(dir) == "specs" {
			candidate := filepath.Dir(dir)
			if _, err := os.Stat(filepath.Join(candidate, "proposal.md")); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func quoteReportValue(value string) string {
	return fmt.Sprintf("%q", strings.Join(strings.Fields(value), " "))
}

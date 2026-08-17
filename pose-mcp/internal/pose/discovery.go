package pose

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GitIgnoredPaths returns the set of paths (relative to root, slash-
// separated, directories carrying a trailing slash) git considers ignored —
// computed once via a single `git ls-files` invocation, not looked up per
// path, so a discovery walker can skip an entire ignored subtree with one
// map lookup per directory instead of one git process per directory. `git
// ls-files --directory` reports an ignored directory itself and does not
// recurse into it, so a deeply-nested ignore rule (`foo/bar/` ignored while
// `foo/` is not) still resolves to exactly the directory that is actually
// ignored, not just top-level entries.
//
// Discovered live: a target repository's own entirely-gitignored tree (a
// vendored or locally-checked-out project the repository deliberately
// excludes from version control) was walked and discovered as real
// governed modules anyway, because none of the three discovery walkers in
// this codebase knew about .gitignore at all (spec
// pose-discovery-gitignore-and-root-alias-fix).
//
// Returns an empty (non-nil) set — never an error — when git is
// unavailable or root is not a repository: discovery degrades to "nothing
// is ignored," the behavior every caller already had before this existed.
func GitIgnoredPaths(root string) map[string]bool {
	ignored := map[string]bool{}
	out, err := exec.Command("git", "-C", root, "ls-files", "--others", "--ignored", "--exclude-standard", "--directory").Output()
	if err != nil {
		return ignored
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ignored[line] = true
		}
	}
	return ignored
}

// ComponentDiscoveryMetrics holds code metrics for a discovered component.
type ComponentDiscoveryMetrics struct {
	LOCProduction int      `json:"loc_production"`
	LOCTests      int      `json:"loc_tests"`
	TotalFiles    int      `json:"total_files"`
	Languages     []string `json:"languages"`
}

// TechnicalDebtMetrics tracks debt markers found in code.
type TechnicalDebtMetrics struct {
	TODOs  int `json:"todos"`
	FIXMEs int `json:"fixmes"`
	Panics int `json:"panics"`
	Stubs  int `json:"stubs"`
}

// ComponentDiscoveryState represents the machine-readable state of a component discovery.
type ComponentDiscoveryState struct {
	SchemaVersion     int                       `json:"schema_version"`
	ComponentSlug     string                    `json:"component_slug"`
	Path              string                    `json:"path"`
	DiscoveredAt      string                    `json:"discovered_at"`
	BaselineCommit    string                    `json:"baseline_commit"`
	Metrics           ComponentDiscoveryMetrics `json:"metrics"`
	TechnicalDebt     TechnicalDebtMetrics      `json:"technical_debt"`
	Metadata          map[string]string         `json:"metadata,omitempty"`
	Status            string                    `json:"status"`
	CompletenessScore float64                   `json:"completeness_score"`
}

// AssessmentsDir returns the path to .pose/assessments
func (s Store) AssessmentsDir() string {
	return filepath.Join(s.Root, ".pose", "assessments")
}

// ComponentStateDir returns the path to .pose/state/components
func (s Store) ComponentStateDir() string {
	return filepath.Join(s.Root, ".pose", "state", "components")
}

// HasAssessments reports whether .pose/assessments exists.
func (s Store) HasAssessments() bool {
	_, err := os.Stat(s.AssessmentsDir())
	return err == nil
}

// ListAssessments returns a list of all markdown assessment filenames in .pose/assessments/
func (s Store) ListAssessments() ([]string, error) {
	dir := s.AssessmentsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var res []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md" {
			res = append(res, e.Name())
		}
	}
	sort.Strings(res)
	return res, nil
}

// GetAssessment returns the content of an assessment markdown file in .pose/assessments/
func (s Store) GetAssessment(slugOrFile string) (map[string]any, error) {
	filename := slugOrFile
	if !strings.HasSuffix(filename, ".md") {
		filename = slugOrFile + ".md"
	}
	if filepath.Base(filename) != filename || filename == ".md" {
		return nil, fmt.Errorf("assessment filename must not escape .pose/assessments")
	}
	path := filepath.Join(s.AssessmentsDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("assessment %q not found: %w", filename, err)
	}
	return map[string]any{
		"filename": filename,
		"path":     path,
		"content":  string(data),
	}, nil
}

// FindComponentDirectories dynamically identifies component paths without hardcoding.
// It checks .pose/indexes/module-metadata.json first, falling back to auto-discovering project manifests.
func (s Store) FindComponentDirectories() []string {
	var targets []string
	seen := make(map[string]bool)

	// 1. Primary: Read .pose/indexes/module-metadata.json
	metaPath := filepath.Join(s.Root, ".pose", "indexes", "module-metadata.json")
	if data, err := os.ReadFile(metaPath); err == nil {
		var meta struct {
			Modules map[string]any `json:"modules"`
		}
		if err := json.Unmarshal(data, &meta); err == nil && len(meta.Modules) > 0 {
			for modPath := range meta.Modules {
				relPath, ok := resolveDeclaredModulePath(s.Root, modPath)
				if ok && !seen[relPath] {
					targets = append(targets, relPath)
					seen[relPath] = true
				}
			}
		}
	}

	if len(targets) > 0 {
		sort.Strings(targets)
		return targets
	}

	// 2. Fallback: Auto-discover subdirectories containing project manifests
	ignored := GitIgnoredPaths(s.Root)
	_ = filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(s.Root, path)
		if rel == "." || rel == "" {
			return nil
		}

		// Skip hidden dirs, common build output dirs, and synthetic
		// testdata/fixture(s) content — see cli/index.go's scanModules for
		// why (spec pose-fixture-directory-discovery-exclusion).
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" || name == "dist" || name == "testdata" || name == "fixture" || name == "fixtures" {
			return filepath.SkipDir
		}
		if ignored[filepath.ToSlash(rel)+"/"] {
			return filepath.SkipDir
		}

		if hasProjectManifest(path) && !seen[rel] {
			targets = append(targets, rel)
			seen[rel] = true
			return filepath.SkipDir
		}
		return nil
	})

	sort.Strings(targets)
	return targets
}

func resolveDeclaredModulePath(root, declared string) (string, bool) {
	if filepath.IsAbs(declared) {
		return "", false
	}
	clean := filepath.Clean(declared)
	if clean == "." {
		return ".", true
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	current := root
	var actual []string
	for _, segment := range strings.Split(filepath.ToSlash(clean), "/") {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", false
		}
		match := ""
		for _, entry := range entries {
			if entry.IsDir() && strings.EqualFold(entry.Name(), segment) {
				match = entry.Name()
				if entry.Name() == segment {
					break
				}
			}
		}
		if match == "" {
			return "", false
		}
		actual = append(actual, match)
		current = filepath.Join(current, match)
	}
	return filepath.ToSlash(filepath.Join(actual...)), true
}

func hasProjectManifest(dir string) bool {
	manifests := []string{
		"go.mod", "Cargo.toml", "package.json", "pyproject.toml",
		"pom.xml", "build.gradle", "Makefile", "Dockerfile", "wrangler.json", "wrangler.jsonc",
	}
	for _, m := range manifests {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			return true
		}
	}
	return false
}

// DiscoverAllComponents dynamically finds and discovers all components in the repository.
func (s Store) DiscoverAllComponents() ([]*ComponentDiscoveryState, error) {
	targets := s.FindComponentDirectories()
	if len(targets) == 0 {
		targets = []string{"."}
	}

	var results []*ComponentDiscoveryState
	for _, t := range targets {
		state, err := s.DiscoverComponent(t)
		if err != nil {
			return nil, err
		}
		if err := s.SaveComponentState(state); err != nil {
			return nil, err
		}
		results = append(results, state)
	}

	if err := s.GenerateConsolidatedAssessment(results); err != nil {
		return nil, err
	}
	return results, nil
}

// DiscoverComponent scans a directory under root and produces a ComponentDiscoveryState.
func (s Store) DiscoverComponent(relPath string) (*ComponentDiscoveryState, error) {
	cleanPath, absPath, err := s.resolveComponentDir(relPath)
	if err != nil {
		return nil, fmt.Errorf("discover component %q: %w", relPath, err)
	}
	slug := slugifyAssessmentPath(cleanPath)

	state := &ComponentDiscoveryState{
		SchemaVersion:     1,
		ComponentSlug:     slug,
		Path:              cleanPath,
		DiscoveredAt:      time.Now().UTC().Format(time.RFC3339),
		BaselineCommit:    s.resolveGitCommit(),
		Status:            "verified",
		CompletenessScore: 1.0,
		Metadata:          make(map[string]string),
	}

	langMap := make(map[string]bool)

	_ = filepath.WalkDir(absPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if path != absPath && shouldSkipAssessmentDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !isDebtSourceExt(ext) {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxAssessmentFileSize {
			return nil
		}
		name := strings.ToLower(entry.Name())
		loc, isTest, generated, debt := inspectCodeFile(path)
		if generated {
			return nil
		}

		// Determine language
		switch ext {
		case ".go":
			langMap["go"] = true
		case ".rs":
			langMap["rust"] = true
		case ".ts", ".tsx":
			langMap["typescript"] = true
		case ".js", ".jsx":
			langMap["javascript"] = true
		case ".py":
			langMap["python"] = true
		case ".java", ".kt":
			langMap["jvm"] = true
		case ".cs":
			langMap["csharp"] = true
		case ".c", ".cc", ".cpp", ".h", ".hpp":
			langMap["c-cpp"] = true
		case ".rb":
			langMap["ruby"] = true
		case ".php":
			langMap["php"] = true
		case ".swift":
			langMap["swift"] = true
		case ".scala":
			langMap["scala"] = true
		case ".sh":
			langMap["shell"] = true
		}

		// Count LOC & debt
		state.Metrics.TotalFiles++
		if isTest || strings.Contains(name, "_test.") || strings.Contains(name, ".test.") || strings.Contains(name, "spec.") {
			state.Metrics.LOCTests += loc
		} else {
			state.Metrics.LOCProduction += loc
		}

		state.TechnicalDebt.TODOs += debt.TODOs
		state.TechnicalDebt.FIXMEs += debt.FIXMEs
		state.TechnicalDebt.Panics += debt.Panics
		state.TechnicalDebt.Stubs += debt.Stubs

		return nil
	})
	state.Metadata = s.componentMetadata(cleanPath)

	for l := range langMap {
		state.Metrics.Languages = append(state.Metrics.Languages, l)
	}
	sort.Strings(state.Metrics.Languages)

	// Compute completeness score dynamically based on ground-truth debt markers
	score := 1.0
	score -= float64(state.TechnicalDebt.TODOs) * 0.005
	score -= float64(state.TechnicalDebt.FIXMEs) * 0.010
	score -= float64(state.TechnicalDebt.Panics) * 0.020
	score -= float64(state.TechnicalDebt.Stubs) * 0.015

	if score < 0.0 {
		score = 0.0
	}
	state.CompletenessScore = score

	if score >= 0.95 {
		state.Status = "verified"
	} else if score >= 0.75 {
		state.Status = "in_progress"
	} else {
		state.Status = "needs_attention"
	}

	return state, nil
}

func (s Store) componentMetadata(componentPath string) map[string]string {
	result := make(map[string]string)
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "indexes", "module-metadata.json"))
	if err != nil {
		return result
	}
	var document struct {
		Modules map[string]map[string]string `json:"modules"`
	}
	if json.Unmarshal(raw, &document) != nil {
		return result
	}
	for path, values := range document.Modules {
		resolved, ok := resolveDeclaredModulePath(s.Root, path)
		if !ok || filepath.ToSlash(resolved) != filepath.ToSlash(componentPath) {
			continue
		}
		for key, value := range values {
			if strings.TrimSpace(value) != "" {
				result[key] = value
			}
		}
		break
	}
	return result
}

func (s Store) resolveGitCommit() string {
	cmd := exec.Command("git", "-C", s.Root, "rev-parse", "--short=12", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "0000000"
	}
	res := strings.TrimSpace(string(out))
	if res == "" {
		return "0000000"
	}
	return res
}

func inspectCodeFile(path string) (loc int, isTest, generated bool, debt TechnicalDebtMetrics) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false, false, debt
	}
	if strings.IndexByte(string(raw), 0) >= 0 || hasGeneratedAssessmentHeader(raw) {
		return 0, false, true, debt
	}
	isTest = isIntegrationTestFile(filepath.ToSlash(path))

	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	lex := debtLexState{rust: strings.EqualFold(filepath.Ext(path), ".rs")}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		loc++
		if isTest {
			continue
		}

		for _, marker := range lex.markers(line) {
			switch marker {
			case "TODO":
				debt.TODOs++
			case "FIXME", "HACK":
				debt.FIXMEs++
			case "PANIC":
				debt.Panics++
			case "STUB":
				debt.Stubs++
			}
		}
	}
	return loc, isTest, false, debt
}

// SaveComponentState writes ComponentDiscoveryState JSON to .pose/state/components/<slug>.json
// and automatically generates the corresponding .pose/assessments/<slug>.md report.
func (s Store) SaveComponentState(state *ComponentDiscoveryState) error {
	if state == nil {
		return fmt.Errorf("component state is required")
	}
	state.ComponentSlug = slugifyAssessmentPath(state.ComponentSlug)
	dir := s.ComponentStateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file := filepath.Join(dir, state.ComponentSlug+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return err
	}

	return s.GenerateComponentAssessmentMarkdown(state)
}

// GenerateComponentAssessmentMarkdown automatically builds .pose/assessments/<slug>.md from discovery data.
func (s Store) GenerateComponentAssessmentMarkdown(state *ComponentDiscoveryState) error {
	dir := s.AssessmentsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	filename := slugifyAssessmentPath(state.ComponentSlug) + ".md"

	langs := strings.Join(state.Metrics.Languages, ", ")
	if langs == "" {
		langs = "n/a"
	}

	md := fmt.Sprintf(`# Component Assessment: %s (`+"`%s`"+`)

> **Mapeamento de Módulo POSE**: `+"`%s`"+`
> **Data de Avaliação**: %s | **Baseline Commit**: %s
> **Métricas**: %d LOC Produção | %d LOC Testes | %d Arquivos Totais
> **Linguagens**: %s
> **Saúde de Código**: %d TODOs | %d FIXMEs | %d Panics | %d Stubs

---

## 1. Visão Geral e Estrutura do Módulo

O componente **%s** reside no caminho `+"`%s`"+` do projeto **%s**.

- **Status de Verificação POSE**: `+"`%s`"+`
- **Pontuação de Completude**: %.0f%%

---

## 2. Matriz de Dívida Técnica & Riscos

- **TODOs**: %d
- **FIXMEs**: %d
- **Panics**: %d
- **Stubs**: %d
`, state.ComponentSlug, state.Path, state.Path, state.DiscoveredAt, state.BaselineCommit, state.Metrics.LOCProduction, state.Metrics.LOCTests, state.Metrics.TotalFiles, langs, state.TechnicalDebt.TODOs, state.TechnicalDebt.FIXMEs, state.TechnicalDebt.Panics, state.TechnicalDebt.Stubs, state.ComponentSlug, state.Path, s.projectLabel(), state.Status, state.CompletenessScore*100, state.TechnicalDebt.TODOs, state.TechnicalDebt.FIXMEs, state.TechnicalDebt.Panics, state.TechnicalDebt.Stubs)

	file := filepath.Join(dir, filename)
	return os.WriteFile(file, []byte(md), 0o644)
}

// LoadComponentState reads ComponentDiscoveryState JSON from .pose/state/components/<slug>.json
func (s Store) LoadComponentState(slug string) (*ComponentDiscoveryState, error) {
	canonical := slugifyAssessmentPath(slug)
	if canonical == "root" && slug != "root" {
		return nil, fmt.Errorf("invalid component slug")
	}
	file := filepath.Join(s.ComponentStateDir(), canonical+".json")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var state ComponentDiscoveryState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// GenerateConsolidatedAssessment builds a project-derived consolidated report.
func (s Store) GenerateConsolidatedAssessment(states []*ComponentDiscoveryState) error {
	dir := s.AssessmentsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var totalProd, totalTests, totalFiles int
	var totalTODOs, totalFIXMEs, totalPanics, totalStubs int
	var sumCompleteness float64

	for _, st := range states {
		totalProd += st.Metrics.LOCProduction
		totalTests += st.Metrics.LOCTests
		totalFiles += st.Metrics.TotalFiles
		totalTODOs += st.TechnicalDebt.TODOs
		totalFIXMEs += st.TechnicalDebt.FIXMEs
		totalPanics += st.TechnicalDebt.Panics
		totalStubs += st.TechnicalDebt.Stubs
		sumCompleteness += st.CompletenessScore
	}

	avgCompleteness := 1.0
	if len(states) > 0 {
		avgCompleteness = sumCompleteness / float64(len(states))
	}

	// Fetch open specs
	var openSpecsCount int
	if openSpecs, err := s.ListSpecs("draft,in_progress,in-progress", ""); err == nil {
		openSpecsCount = len(openSpecs)
	}

	// Fetch integration gaps
	var gapsCount int
	if matrix, err := s.AnalyzeIntegrations(); err == nil && matrix != nil {
		gapsCount = matrix.Summary.IdentifiedGaps
	}

	// Deduct platform macro penalties
	platformCompleteness := avgCompleteness - (float64(openSpecsCount) * 0.02) - (float64(gapsCount) * 0.015)
	if platformCompleteness < 0.0 {
		platformCompleteness = 0.0
	}

	commit := s.resolveGitCommit()
	now := time.Now().UTC().Format(time.RFC3339)

	project := s.projectLabel()
	md := fmt.Sprintf(`# Project Assessment: %s

> **Gerado por**: POSE Discovery Engine (`+"`pose assess discover`"+`)
> **Data de Avaliação**: %s
> **Baseline Commit**: %s

---

## 1. Resumo Executivo do Projeto

- **Total de Componentes Auditados**: %d
- **Linhas de Código de Produção**: %d
- **Linhas de Código de Testes**: %d
- **Total Geral de Linhas de Código**: %d
- **Total de Arquivos Auditados**: %d
- **Completude Dinâmica da Plataforma**: %.1f%%
- **Dívidas Técnicas em Aberto**: %d TODOs | %d FIXMEs | %d Panics | %d Stubs
- **Especificações (Specs) em Aberto**: %d
- **Gaps de Integração Identificados**: %d

---

## 2. Inventário e Métricas dos Componentes

| # | Componente Slug | Caminho do Módulo | Linguagens | LOC Produção | LOC Testes | Arquivos | TODOs | Completude | Status |
|---|---|---|---|---|---|---|---|---|---|
`, project, now, commit, len(states), totalProd, totalTests, totalProd+totalTests, totalFiles, platformCompleteness*100, totalTODOs, totalFIXMEs, totalPanics, totalStubs, openSpecsCount, gapsCount)

	for i, st := range states {
		langs := strings.Join(st.Metrics.Languages, ", ")
		if langs == "" {
			langs = "n/a"
		}
		md += fmt.Sprintf("| %02d | `%s` | `%s` | `%s` | %d | %d | %d | %d | %.0f%% | `%s` |\n",
			i+1, st.ComponentSlug, st.Path, langs, st.Metrics.LOCProduction, st.Metrics.LOCTests, st.Metrics.TotalFiles, st.TechnicalDebt.TODOs, st.CompletenessScore*100, st.Status)
	}

	md += "\n---\n\n## 3. Topologia Observada\n\n"
	if len(states) == 0 {
		md += "Nenhum componente com manifesto ou metadado POSE foi observado.\n"
	} else {
		for _, st := range states {
			langs := strings.Join(st.Metrics.Languages, ", ")
			if langs == "" {
				langs = "n/a"
			}
			md += fmt.Sprintf("- `%s`: caminho `%s`; linguagens %s; status `%s`.\n", st.ComponentSlug, st.Path, langs, st.Status)
		}
	}

	file := filepath.Join(dir, "consolidated.md")
	if err := os.WriteFile(file, []byte(md), 0o644); err != nil {
		return err
	}

	return s.GenerateAssessmentsIndex(states)
}

// GenerateAssessmentsIndex automatically builds .pose/assessments/README.md linking all components.
func (s Store) GenerateAssessmentsIndex(states []*ComponentDiscoveryState) error {
	dir := s.AssessmentsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	md := fmt.Sprintf(`# Component Assessments: %s

> **Gerado Automaticamente por**: POSE Discovery Engine (`+"`pose assess discover`"+`)

## Assessments por Componente (`+fmt.Sprintf("%d Módulos", len(states))+`)

| #  | Componente Slug | Módulo / Path | Linguagens | LOC Produção | LOC Testes | Arquivos | Status | Relatório Markdown |
|----|------------|---------------|-----------|--------------|------------|----------|--------|--------------------|
`, s.projectLabel())

	for i, st := range states {
		filename := slugifyAssessmentPath(st.ComponentSlug) + ".md"

		langs := strings.Join(st.Metrics.Languages, ", ")
		if langs == "" {
			langs = "n/a"
		}

		md += fmt.Sprintf("| %02d | `%s` | `%s` | `%s` | %d | %d | %d | `%s` | [%s](./%s) |\n",
			i+1, st.ComponentSlug, st.Path, langs, st.Metrics.LOCProduction, st.Metrics.LOCTests, st.Metrics.TotalFiles, st.Status, filename, filename)
	}

	md += "\n## Assessments Consolidados & Governança Global\n\n" +
		"- **[Assessment Consolidado do Projeto](./consolidated.md)**\n" +
		"- **[Matriz de Integrações Inter-Componentes](./integrations.md)**\n" +
		"- **[Relatório de Dívida Técnica & Backlog POSE](./technical-debt.md)**\n"

	file := filepath.Join(dir, "README.md")
	return os.WriteFile(file, []byte(md), 0o644)
}

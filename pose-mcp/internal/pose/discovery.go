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
				relPath := modPath
				if _, err := os.Stat(filepath.Join(s.Root, relPath)); err != nil {
					relPath = strings.ToLower(modPath)
				}
				if _, err := os.Stat(filepath.Join(s.Root, relPath)); err == nil && !seen[relPath] {
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
	_ = filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(s.Root, path)
		if rel == "." || rel == "" {
			return nil
		}

		// Skip hidden dirs and common build output dirs
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "target" || name == "dist" {
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
			continue
		}
		_ = s.SaveComponentState(state)
		results = append(results, state)
	}

	_ = s.GenerateConsolidatedAssessment(results)
	return results, nil
}

// DiscoverComponent scans a directory under root and produces a ComponentDiscoveryState.
func (s Store) DiscoverComponent(relPath string) (*ComponentDiscoveryState, error) {
	absPath := filepath.Join(s.Root, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("discover component %q: %w", relPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("discover component %q: path is not a directory", relPath)
	}

	slug := strings.Trim(strings.ReplaceAll(relPath, "/", "-"), "-")
	if slug == "" {
		slug = "root"
	}

	state := &ComponentDiscoveryState{
		SchemaVersion:     1,
		ComponentSlug:     slug,
		Path:              relPath,
		DiscoveredAt:      time.Now().UTC().Format(time.RFC3339),
		BaselineCommit:    s.resolveGitCommit(),
		Status:            "verified",
		CompletenessScore: 1.0,
		Metadata:          make(map[string]string),
	}

	langMap := make(map[string]bool)

	_ = filepath.Walk(absPath, func(path string, f os.FileInfo, err error) error {
		if err != nil || f.IsDir() {
			name := f.Name()
			if name == "node_modules" || name == "vendor" || name == "target" || name == ".git" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		name := strings.ToLower(f.Name())

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
		}

		// Count LOC & debt
		loc, isTest, debt := inspectCodeFile(path)
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

func inspectCodeFile(path string) (loc int, isTest bool, debt TechnicalDebtMetrics) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false, debt
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		loc++

		upper := strings.ToUpper(line)
		if strings.Contains(upper, "TODO") {
			debt.TODOs++
		}
		if strings.Contains(upper, "FIXME") {
			debt.FIXMEs++
		}
		if strings.Contains(line, "panic(") {
			debt.Panics++
		}
		if strings.Contains(line, "stub") || strings.Contains(line, "unimplemented!") {
			debt.Stubs++
		}
	}
	return loc, false, debt
}

// SaveComponentState writes ComponentDiscoveryState JSON to .pose/state/components/<slug>.json
// and automatically generates the corresponding .pose/assessments/<slug>.md report.
func (s Store) SaveComponentState(state *ComponentDiscoveryState) error {
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

	filename := state.ComponentSlug + ".md"
	// Normalize slug if starts with graphforge-
	if strings.HasPrefix(state.ComponentSlug, "graphforge-") {
		filename = strings.TrimPrefix(state.ComponentSlug, "graphforge-") + ".md"
	} else if strings.HasPrefix(state.ComponentSlug, "pose-dist-") {
		filename = strings.TrimPrefix(state.ComponentSlug, "pose-dist-") + ".md"
	}

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

O componente **%s** reside no caminho `+"`%s`"+` no repositório Harne8.

- **Status de Verificação POSE**: `+"`%s`"+`
- **Pontuação de Completude**: %.0f%%

---

## 2. Matriz de Dívida Técnica & Riscos

- **TODOs**: %d
- **FIXMEs**: %d
- **Panics**: %d
- **Stubs**: %d
`, state.ComponentSlug, state.Path, state.Path, state.DiscoveredAt, state.BaselineCommit, state.Metrics.LOCProduction, state.Metrics.LOCTests, state.Metrics.TotalFiles, langs, state.TechnicalDebt.TODOs, state.TechnicalDebt.FIXMEs, state.TechnicalDebt.Panics, state.TechnicalDebt.Stubs, state.ComponentSlug, state.Path, state.Status, state.CompletenessScore*100, state.TechnicalDebt.TODOs, state.TechnicalDebt.FIXMEs, state.TechnicalDebt.Panics, state.TechnicalDebt.Stubs)

	file := filepath.Join(dir, filename)
	return os.WriteFile(file, []byte(md), 0o644)
}

// LoadComponentState reads ComponentDiscoveryState JSON from .pose/state/components/<slug>.json
func (s Store) LoadComponentState(slug string) (*ComponentDiscoveryState, error) {
	file := filepath.Join(s.ComponentStateDir(), slug+".json")
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

// GenerateConsolidatedAssessment builds .pose/assessments/consolidated.md for the Harne8 platform.
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

	md := fmt.Sprintf(`# Harne8 Platform Macro Assessment & Monorepo Consolidation

> **Gerado por**: POSE Discovery Engine (`+"`pose assess discover`"+`)
> **Data de Avaliação**: %s
> **Baseline Commit**: %s

---

## 1. Resumo Executivo da Plataforma Harne8

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

## 2. Inventário e Métricas dos Componentes Harne8

| # | Componente Slug | Caminho do Módulo | Linguagens | LOC Produção | LOC Testes | Arquivos | TODOs | Completude | Status |
|---|---|---|---|---|---|---|---|---|---|
`, now, commit, len(states), totalProd, totalTests, totalProd+totalTests, totalFiles, platformCompleteness*100, totalTODOs, totalFIXMEs, totalPanics, totalStubs, openSpecsCount, gapsCount)

	for i, st := range states {
		langs := strings.Join(st.Metrics.Languages, ", ")
		if langs == "" {
			langs = "n/a"
		}
		md += fmt.Sprintf("| %02d | `%s` | `%s` | `%s` | %d | %d | %d | %d | %.0f%% | `%s` |\n",
			i+1, st.ComponentSlug, st.Path, langs, st.Metrics.LOCProduction, st.Metrics.LOCTests, st.Metrics.TotalFiles, st.TechnicalDebt.TODOs, st.CompletenessScore*100, st.Status)
	}

	md += "\n---\n\n## 3. Arquitetura dos Subsistemas Harne8\n\n" +
		"1. **Conductor & Harness Control Plane (`conductor`, `harness`)**:\n" +
		"   - Orquestração de frota de agentes de IA, acompanhamento de execuções de runs e execução de sandboxes com suporte a SAGAs.\n" +
		"2. **GraphForge Knowledge Subsystem (`graphforge/*`)**:\n" +
		"   - Compilação de código AST (Rust/Go), indexadores (Git/Infra/Test), correlation engine OTel, enricher semântico LLM, motor de planejamento shadow graph e interface tri-engine Canvas 2D/3D (React 19).\n" +
		"3. **Edge Gateway & Portal (`workers/app`, `site`)**:\n" +
		"   - Gateway Cloudflare Workers, autenticação de sessão Portal, distribuição de magic links e site oficial Harne8.\n" +
		"4. **Governança POSE & Enforce (`pose-dist/pose-mcp`, `pose-dist/mcp-enforce`, `contracts`)**:\n" +
		"   - Servidor nativo `harne8-pose-mcp`, motor de aplicação de política OPA `mcp-enforce` e esquemas compartilhados Protobuf/OpenAPI.\n"

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

	md := `# Harne8 Platform — Component Assessments Index

> **Gerado Automaticamente por**: POSE Discovery Engine (` + "`pose assess discover`" + `)

## Assessments por Componente (` + fmt.Sprintf("%d Módulos", len(states)) + `)

| #  | Componente Slug | Módulo / Path | Linguagens | LOC Produção | LOC Testes | Arquivos | Status | Relatório Markdown |
|----|------------|---------------|-----------|--------------|------------|----------|--------|--------------------|
`

	for i, st := range states {
		filename := st.ComponentSlug + ".md"
		if strings.HasPrefix(st.ComponentSlug, "graphforge-") {
			filename = strings.TrimPrefix(st.ComponentSlug, "graphforge-") + ".md"
		} else if strings.HasPrefix(st.ComponentSlug, "pose-dist-") {
			filename = strings.TrimPrefix(st.ComponentSlug, "pose-dist-") + ".md"
		}

		langs := strings.Join(st.Metrics.Languages, ", ")
		if langs == "" {
			langs = "n/a"
		}

		md += fmt.Sprintf("| %02d | `%s` | `%s` | `%s` | %d | %d | %d | `%s` | [%s](./%s) |\n",
			i+1, st.ComponentSlug, st.Path, langs, st.Metrics.LOCProduction, st.Metrics.LOCTests, st.Metrics.TotalFiles, st.Status, filename, filename)
	}

	md += "\n## Assessments Consolidados & Governança Global\n\n" +
		"- **[Assessment Macro Consolidado da Plataforma Harne8](./consolidated.md)**\n" +
		"- **[Matriz de Integrações Inter-Componentes](./integrations.md)**\n" +
		"- **[Relatório de Dívida Técnica & Backlog POSE](./technical-debt.md)**\n"

	file := filepath.Join(dir, "README.md")
	return os.WriteFile(file, []byte(md), 0o644)
}

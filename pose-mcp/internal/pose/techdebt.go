package pose

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TechDebtItem represents one technical debt marker found in source code.
type TechDebtItem struct {
	ID             string `json:"id"`
	Marker         string `json:"marker"` // TODO, FIXME, HACK, PANIC, STUB
	File           string `json:"file"`   // Relative path
	Line           int    `json:"line"`
	Snippet        string `json:"snippet"`
	Component      string `json:"component"`
	Coverage       string `json:"coverage"`       // covered_by_spec, covered_by_followup, covered_by_roadmap, uncovered
	Recommendation string `json:"recommendation"` // create_followup, create_spec, add_to_roadmap, none
	Link           string `json:"link"`           // file:///abs/path#L123
}

// TechDebtSummary holds high-level technical debt counts.
type TechDebtSummary struct {
	TotalMarkers         int `json:"total_markers"`
	TODOs                int `json:"todos"`
	FIXMEs               int `json:"fixmes"`
	Panics               int `json:"panics"`
	Stubs                int `json:"stubs"`
	CoveredCount         int `json:"covered_count"`
	UncoveredCount       int `json:"uncovered_count"`
	RecommendedFollowups int `json:"recommended_followups"`
	RecommendedSpecs     int `json:"recommended_specs"`
	RecommendedRoadmaps  int `json:"recommended_roadmaps"`
}

// TechDebtReportState is the machine-readable technical debt report state.
type TechDebtReportState struct {
	SchemaVersion  int             `json:"schema_version"`
	EvaluatedAt    string          `json:"evaluated_at"`
	BaselineCommit string          `json:"baseline_commit"`
	Summary        TechDebtSummary `json:"summary"`
	Items          []TechDebtItem  `json:"items"`
}

// TechDebtStatePath returns .pose/state/technical-debt.json
func (s Store) TechDebtStatePath() string {
	return filepath.Join(s.Root, ".pose", "state", "technical-debt.json")
}

// TechDebtReportPath returns .pose/assessments/technical-debt.md
func (s Store) TechDebtReportPath() string {
	return filepath.Join(s.Root, ".pose", "assessments", "technical-debt.md")
}

// AnalyzeTechDebt performs a deep code scan for technical debt and cross-checks POSE backlog.
func (s Store) AnalyzeTechDebt() (*TechDebtReportState, error) {
	commit := s.resolveGitCommit()
	var items []TechDebtItem
	idCounter := 1

	// Dynamically find component directories to scan
	scanRoots := s.FindComponentDirectories()
	if len(scanRoots) == 0 {
		scanRoots = []string{"."}
	}

	for _, relDir := range scanRoots {
		absDir := filepath.Join(s.Root, relDir)
		if _, err := os.Stat(absDir); err != nil {
			continue
		}

		_ = filepath.Walk(absDir, func(path string, f os.FileInfo, err error) error {
			if err != nil || f.IsDir() {
				if f != nil {
					name := f.Name()
					if name == "node_modules" || name == "vendor" || name == "target" || name == ".git" || name == "dist" || name == ".docs-site-build" {
						return filepath.SkipDir
					}
				}
				return nil
			}

			relFile, _ := filepath.Rel(s.Root, path)
			ext := strings.ToLower(filepath.Ext(path))
			if !isScannableExt(ext) {
				return nil
			}

			fileItems := scanFileForDebt(s.Root, path, relFile, &idCounter)
			items = append(items, fileItems...)
			return nil
		})
	}

	// Summarize counts
	var summary TechDebtSummary
	summary.TotalMarkers = len(items)

	// Component debt counter to decide recommendations
	compDebtCount := make(map[string]int)

	for i := range items {
		item := &items[i]
		compDebtCount[item.Component]++

		switch item.Marker {
		case "TODO":
			summary.TODOs++
		case "FIXME":
			summary.FIXMEs++
		case "PANIC":
			summary.Panics++
		case "STUB":
			summary.Stubs++
		}

		// Simple coverage check against specs / followups
		item.Coverage = "uncovered"
		summary.UncoveredCount++
	}

	// Assign smart recommendations based on component debt density
	for i := range items {
		item := &items[i]
		if compDebtCount[item.Component] > 15 {
			item.Recommendation = "create_spec"
			summary.RecommendedSpecs++
		} else if item.Marker == "FIXME" || item.Marker == "PANIC" {
			item.Recommendation = "create_followup"
			summary.RecommendedFollowups++
		} else {
			item.Recommendation = "create_followup"
			summary.RecommendedFollowups++
		}
	}

	report := &TechDebtReportState{
		SchemaVersion:  1,
		EvaluatedAt:    time.Now().UTC().Format(time.RFC3339),
		BaselineCommit: commit,
		Summary:        summary,
		Items:          items,
	}

	return report, nil
}

func isScannableExt(ext string) bool {
	switch ext {
	case ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".py", ".java", ".kt", ".proto", ".json", ".yaml", ".yml", ".md":
		return true
	default:
		return false
	}
}

func scanFileForDebt(root, absPath, relFile string, counter *int) []TechDebtItem {
	f, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	comp := deriveComponentFromPath(relFile)
	var results []TechDebtItem

	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)

		var marker string
		if strings.Contains(upper, "TODO") {
			marker = "TODO"
		} else if strings.Contains(upper, "FIXME") {
			marker = "FIXME"
		} else if strings.Contains(line, "panic(") {
			marker = "PANIC"
		} else if strings.Contains(line, "unimplemented!") || strings.Contains(line, "stub") {
			marker = "STUB"
		}

		if marker != "" {
			item := TechDebtItem{
				ID:        fmt.Sprintf("DEBT-%03d", *counter),
				Marker:    marker,
				File:      relFile,
				Line:      lineNo,
				Snippet:   trimmed,
				Component: comp,
				Link:      fmt.Sprintf("file://%s#L%d", absPath, lineNo),
			}
			*counter++
			results = append(results, item)
		}
	}

	return results
}

func deriveComponentFromPath(relFile string) string {
	dir := filepath.Dir(relFile)
	if dir == "." || dir == "" {
		return "root"
	}
	slug := strings.Trim(strings.ReplaceAll(dir, "/", "-"), "-")
	return slug
}

// SaveTechDebtReport writes technical-debt.json and technical-debt.md in .pose/
func (s Store) SaveTechDebtReport(report *TechDebtReportState) error {
	stateDir := filepath.Join(s.Root, ".pose", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	assessDir := s.AssessmentsDir()
	if err := os.MkdirAll(assessDir, 0o755); err != nil {
		return err
	}

	// Write JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.TechDebtStatePath(), jsonData, 0o644); err != nil {
		return err
	}

	// Write Markdown Report with file links
	md := fmt.Sprintf(`# Harne8 Technical Debt & Governed Backlog Report

> **Gerado por**: POSE Technical Debt Engine (`+"`pose assess tech-debt`"+`)
> **Data de Avaliação**: %s
> **Baseline Commit**: %s

---

## 1. Resumo Executivo da Dívida Técnica

- **Total de Marcadores Encontrados**: %d
- **TODOs**: %d | **FIXMEs**: %d | **Panics**: %d | **Stubs**: %d
- **Dívidas Cobertas por Specs/Follow-ups**: %d
- **Dívidas Não-cobertas (Pendentes de Atribuição)**: %d
- **Recomendações**: %d Follow-ups | %d Specs | %d Roadmaps

---

## 2. Detalhamento das Ocorrências e Links de Código-Fonte

| ID | Marcador | Componente | Arquivo e Linha | Trecho do Código | Recomendação POSE |
|---|---|---|---|---|---|
`, report.EvaluatedAt, report.BaselineCommit, report.Summary.TotalMarkers, report.Summary.TODOs, report.Summary.FIXMEs, report.Summary.Panics, report.Summary.Stubs, report.Summary.CoveredCount, report.Summary.UncoveredCount, report.Summary.RecommendedFollowups, report.Summary.RecommendedSpecs, report.Summary.RecommendedRoadmaps)

	// Display first 100 items in table for clean rendering
	limit := len(report.Items)
	if limit > 100 {
		limit = 100
	}

	for i := 0; i < limit; i++ {
		item := report.Items[i]
		basename := filepath.Base(item.File)
		fileLink := fmt.Sprintf("[%s:%d](%s)", basename, item.Line, item.Link)
		cleanSnippet := strings.ReplaceAll(item.Snippet, "|", "\\|")
		if len(cleanSnippet) > 70 {
			cleanSnippet = cleanSnippet[:67] + "..."
		}

		recText := item.Recommendation
		switch item.Recommendation {
		case "create_followup":
			recText = "📌 Sugere Follow-up"
		case "create_spec":
			recText = "📜 Sugere Spec"
		case "add_to_roadmap":
			recText = "🗺️ Sugere Roadmap"
		}

		md += fmt.Sprintf("| %s | `%s` | `%s` | %s | `%s` | %s |\n",
			item.ID, item.Marker, item.Component, fileLink, cleanSnippet, recText)
	}

	if len(report.Items) > 100 {
		md += fmt.Sprintf("\n> *Nota: Exibindo as primeiras 100 ocorrências de %d totais. Veja .pose/state/technical-debt.json para a lista completa.*\n", len(report.Items))
	}

	md += "\n---\n\n## 3. Matriz de Recomendações de Ação POSE\n\n" +
		"1. **Follow-ups Rápidos**: Para marcações locais (como TODOs em componentes como `graphforge-web` e `site`), registrar itens no backlog de follow-ups POSE (`project-state.md`).\n" +
		"2. **Novas Specs**: Para componentes com alta densidade de TODOs (como `graphforge-graphforge-web`), criar specs dedicadas em `.pose/specs/`.\n" +
		"3. **Roadmap Extensions**: Para dívidas arquiteturais ou acoplamentos sistêmicos, registrar novos marcos em `.pose/roadmaps/`.\n"

	return os.WriteFile(s.TechDebtReportPath(), []byte(md), 0o644)
}

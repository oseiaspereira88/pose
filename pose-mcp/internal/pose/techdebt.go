package pose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

type TechDebtItem struct {
	ID             string `json:"id"`
	Marker         string `json:"marker"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Snippet        string `json:"snippet"`
	Component      string `json:"component"`
	Coverage       string `json:"coverage"`
	CoverageRef    string `json:"coverage_ref,omitempty"`
	Recommendation string `json:"recommendation"`
	Link           string `json:"link"`
}

type TechDebtSummary struct {
	TotalMarkers         int `json:"total_markers"`
	TODOs                int `json:"todos"`
	FIXMEs               int `json:"fixmes"`
	Hacks                int `json:"hacks,omitempty"`
	Panics               int `json:"panics"`
	Stubs                int `json:"stubs"`
	CoveredCount         int `json:"covered_count"`
	UncoveredCount       int `json:"uncovered_count"`
	RecommendedFollowups int `json:"recommended_followups"`
	RecommendedSpecs     int `json:"recommended_specs"`
	RecommendedRoadmaps  int `json:"recommended_roadmaps"`
}

type TechDebtReportState struct {
	SchemaVersion  int             `json:"schema_version"`
	EvaluatedAt    string          `json:"evaluated_at"`
	BaselineCommit string          `json:"baseline_commit"`
	Summary        TechDebtSummary `json:"summary"`
	Items          []TechDebtItem  `json:"items"`
	// Scope says what the scan covered (spec pose-source-coverage-integrity): the
	// files it read, the ones it could not read (Coverage.Losses) and the ones it
	// leaves out on purpose (Coverage.Exclusions). Zero markers with Complete false
	// does not mean the project has none, and with exclusions it means none in the
	// files that were scanned.
	ScannedFiles int            `json:"scanned_files"`
	Complete     bool           `json:"complete"`
	Coverage     SourceCoverage `json:"coverage"`
}

type debtCoverageDocument struct {
	Coverage string
	Ref      string
	Text     string
}

var (
	// Comment markers are intentionally case-sensitive. Lowercase words such as
	// Portuguese "todo" and explanatory prose such as "test stub" are not debt
	// declarations; executable stub constructs remain case-insensitive below.
	debtCommentMarkerRE = regexp.MustCompile(`\b(TODO|FIXME|HACK|STUB)\b`)
	debtPanicRE         = regexp.MustCompile(`(?i)\bpanic\s*\(`)
	debtStubRE          = regexp.MustCompile(`(?i)\b(?:notimplementederror|notimplementedexception)\b|\b(?:unimplemented|todo)\s*!\s*\(`)
)

type debtLexState struct {
	blockEnd string
	quote    byte
	rust     bool
}

func rustLifetimeStart(line string, index int) bool {
	if index+1 >= len(line) || !(unicode.IsLetter(rune(line[index+1])) || line[index+1] == '_') {
		return false
	}
	end := index + 2
	for end < len(line) && (unicode.IsLetter(rune(line[end])) || unicode.IsDigit(rune(line[end])) || line[end] == '_') {
		end++
	}
	return end >= len(line) || line[end] != '\''
}

// split strips string literals while retaining comment text. The small lexical
// state prevents marker-shaped strings and multi-line comments from being
// mistaken for executable debt without depending on one project's language.
func (state *debtLexState) split(line string) (string, string) {
	var code, comments strings.Builder
	for i := 0; i < len(line); {
		if state.blockEnd != "" {
			end := strings.Index(line[i:], state.blockEnd)
			if end < 0 {
				comments.WriteString(line[i:])
				break
			}
			comments.WriteString(line[i : i+end])
			i += end + len(state.blockEnd)
			state.blockEnd = ""
			continue
		}
		if state.quote != 0 {
			quote := state.quote
			escaped := false
			for i < len(line) {
				current := line[i]
				i++
				if quote != '`' && current == '\\' && !escaped {
					escaped = true
					continue
				}
				if current == quote && !escaped {
					state.quote = 0
					break
				}
				escaped = false
			}
			continue
		}
		switch {
		case strings.HasPrefix(line[i:], "/*"):
			state.blockEnd = "*/"
			i += 2
		case strings.HasPrefix(line[i:], "<!--"):
			state.blockEnd = "-->"
			i += 4
		case strings.HasPrefix(line[i:], "//"):
			comments.WriteString(line[i+2:])
			return code.String(), comments.String()
		case line[i] == '#' && (i == 0 || unicode.IsSpace(rune(line[i-1]))):
			comments.WriteString(line[i+1:])
			return code.String(), comments.String()
		case strings.HasPrefix(line[i:], "--") && (i == 0 || unicode.IsSpace(rune(line[i-1]))):
			comments.WriteString(line[i+2:])
			return code.String(), comments.String()
		case line[i] == '\'' && state.rust && rustLifetimeStart(line, i):
			code.WriteByte(line[i])
			i++
		case line[i] == '\'' || line[i] == '"' || line[i] == '`':
			state.quote = line[i]
			i++
		default:
			code.WriteByte(line[i])
			i++
		}
	}
	return code.String(), comments.String()
}

func (state *debtLexState) markers(line string) []string {
	code, comments := state.split(line)
	seen := make(map[string]bool)
	markers := []string{}
	for _, match := range debtCommentMarkerRE.FindAllStringSubmatch(comments, -1) {
		marker := strings.ToUpper(match[1])
		if !seen[marker] {
			markers = append(markers, marker)
			seen[marker] = true
		}
	}
	if debtPanicRE.MatchString(code) {
		markers = append(markers, "PANIC")
	}
	if debtStubRE.MatchString(code) && !seen["STUB"] {
		markers = append(markers, "STUB")
	}
	return markers
}

func (s Store) TechDebtStatePath() string {
	return filepath.Join(s.Root, ".pose", "state", "technical-debt.json")
}

func (s Store) TechDebtReportPath() string {
	return filepath.Join(s.Root, ".pose", "assessments", "technical-debt.md")
}

func activeAssessmentStatus(status string) bool {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(status), "_", "-")) {
	case "done", "superseded", "abandoned":
		return false
	default:
		return true
	}
}

func (s Store) debtCoverageDocuments() []debtCoverageDocument {
	documents, _ := s.debtCoverageDocumentsRead()
	return documents
}

// debtCoverageDocumentsRead is debtCoverageDocuments and the backlog documents it could
// not read. A marker is reported "uncovered" when no document names it; a document that
// could not be read may have named it, so the classification is only as complete as this
// read (spec pose-source-coverage-integrity).
func (s Store) debtCoverageDocumentsRead() ([]debtCoverageDocument, []SourceLoss) {
	var documents []debtCoverageDocument
	var losses []SourceLoss
	lose := func(artifact, kind, detail string) {
		losses = append(losses, SourceLoss{Producer: "techdebt-reconciliation", Artifact: artifact, Kind: kind, Detail: detail})
	}
	if specs, err := s.ListSpecs("", ""); err == nil {
		for _, item := range specs {
			if !activeAssessmentStatus(item.Status) {
				continue
			}
			spec, err := s.GetSpec(item.Slug)
			if err == nil {
				documents = append(documents, debtCoverageDocument{
					Coverage: "covered_by_spec", Ref: "spec:" + item.Slug, Text: strings.ToLower(spec.Body),
				})
			} else {
				lose("spec:"+item.Slug, LossReadError, err.Error())
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		lose(".pose/specs", LossReadError, err.Error())
	}
	// Specs the listing cannot see at all (a file it could not read) are still backlog.
	for _, loss := range ReadSpecFollowups(s.Root).Coverage.Losses {
		lose(loss.Artifact, loss.Kind, loss.Detail)
	}
	if roadmaps, err := s.ListRoadmaps(); err == nil {
		for _, item := range roadmaps {
			if !activeAssessmentStatus(item.Status) {
				continue
			}
			roadmap, err := s.GetRoadmap(item.Slug)
			if err == nil {
				documents = append(documents, debtCoverageDocument{
					Coverage: "covered_by_roadmap", Ref: "roadmap:" + item.Slug, Text: strings.ToLower(roadmap.Body),
				})
			} else {
				lose("roadmap:"+item.Slug, LossReadError, err.Error())
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		lose(".pose/roadmaps", LossReadError, err.Error())
	}
	// Roadmaps the listing cannot see are still backlog: a link to nowhere or a file
	// that cannot be opened is listed by the directory and absent from ListRoadmaps.
	if entries, dirErr := os.ReadDir(filepath.Join(s.Root, ".pose", "roadmaps")); dirErr == nil {
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".md") || strings.EqualFold(name, "README.md") {
				continue
			}
			full := filepath.Join(s.Root, ".pose", "roadmaps", name)
			if _, statErr := os.Stat(full); statErr != nil {
				lose(".pose/roadmaps/"+name, LossReadError, statErr.Error())
			} else if f, openErr := os.Open(full); openErr != nil {
				lose(".pose/roadmaps/"+name, LossReadError, openErr.Error())
			} else {
				f.Close()
			}
		}
	} else if _, lerr := os.Lstat(filepath.Join(s.Root, ".pose", "roadmaps")); !errors.Is(dirErr, os.ErrNotExist) || lerr == nil {
		lose(".pose/roadmaps", LossReadError, dirErr.Error())
	}
	statePath := filepath.Join(s.Root, ".pose", "state", "project-state.md")
	if state, err := s.ProjectState(context.Background(), "Follow-ups"); err != nil {
		if _, lerr := os.Lstat(statePath); lerr == nil || !errors.Is(lerr, os.ErrNotExist) {
			// Present, or not provably absent (an unreadable parent says nothing).
			lose(".pose/state/project-state.md", LossReadError, err.Error())
		}
	} else if err == nil {
		for _, section := range state.Sections {
			if section.Name == "Follow-ups" {
				documents = append(documents, debtCoverageDocument{
					Coverage: "covered_by_followup", Ref: "state:follow-ups", Text: strings.ToLower(section.Body),
				})
			}
		}
	}
	sort.Slice(documents, func(i, j int) bool { return documents[i].Ref < documents[j].Ref })
	return documents, losses
}

// documentCoversDebt reports whether a backlog document actually addresses this
// debt marker. Coverage requires evidence that identifies the marker itself —
// its file path or its stable id — never merely its component.
//
// Naming a component used to be enough, which meant any active spec touching a
// module silently marked every debt marker in it as covered and dropped the
// recommended follow-up. A component name is a location, not a commitment.
func documentCoversDebt(document string, item TechDebtItem) bool {
	file := strings.ToLower(filepath.ToSlash(item.File))
	if file != "" && strings.Contains(document, file) {
		return true
	}
	if id := strings.ToLower(strings.TrimSpace(item.ID)); id != "" && strings.Contains(document, id) {
		return true
	}
	return false
}

// techDebtID derives a marker identifier from the marker's own identity — the
// file it lives in, the marker keyword and the text of the line — so the same
// marker keeps the same id across runs.
//
// The ids used to be positional (`DEBT-001`, `DEBT-002`, …), which meant adding
// a marker earlier in the scan renumbered every later one: a spec, review or
// follow-up citing `DEBT-014` silently came to mean a different marker. This is
// the same defect pose-assessment-engine-precision fixed for integration gaps,
// left behind on debt markers (spec pose-governance-gate-activation, R1).
//
// The line number is deliberately excluded: a marker that merely moves down the
// file when unrelated code is inserted above it is still the same debt, and an
// id that changed on every edit would be no more citable than a positional one.
func techDebtID(relPath, marker, snippet string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(relPath)) + "\x00" +
		strings.ToLower(strings.TrimSpace(marker)) + "\x00" +
		strings.TrimSpace(snippet)))
	return "DEBT-" + hex.EncodeToString(sum[:4])
}

// scanAssessmentFileForDebt scans the bytes the file scan already read, line by line.
// It does not reopen the file and has no line limit: a bufio.Scanner stops at a
// token longer than its buffer and reports it only through Err, which the scan
// ignored, so a TODO after a 70 000-character line was lost without a word.
func scanAssessmentFileForDebt(file assessmentFile) []TechDebtItem {
	var result []TechDebtItem
	lex := debtLexState{rust: file.Ext == ".rs"}
	lineNumber := 0
	for _, line := range strings.Split(file.Content, "\n") {
		lineNumber++
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		for _, marker := range lex.markers(line) {
			result = append(result, TechDebtItem{
				ID: techDebtID(file.RelPath, marker, trimmed), Marker: marker,
				File: file.RelPath, Line: lineNumber, Snippet: trimmed, Component: file.Component,
				Coverage: "uncovered", Recommendation: "create_followup",
				Link: fmt.Sprintf("file://%s#L%d", file.AbsPath, lineNumber),
			})
		}
	}
	return result
}

// AnalyzeTechDebt scans project source once and reconciles markers with active POSE backlog evidence.
func (s Store) AnalyzeTechDebt() (*TechDebtReportState, error) {
	files, coverage, err := s.assessmentFilesRead()
	if err != nil {
		return nil, err
	}
	items := []TechDebtItem{}
	scanned := 0
	for _, file := range files {
		if !isDebtSourceExt(file.Ext) || isIntegrationTestFile(file.RelPath) {
			continue
		}
		scanned++
		items = append(items, scanAssessmentFileForDebt(file)...)
	}
	coverage.Normalize()
	documents, documentLosses := s.debtCoverageDocumentsRead()
	for _, loss := range documentLosses {
		coverage.Add(loss)
	}
	coverage.Normalize()
	componentCounts := make(map[string]int)
	componentFiles := make(map[string]map[string]bool)
	for _, item := range items {
		componentCounts[item.Component]++
		if componentFiles[item.Component] == nil {
			componentFiles[item.Component] = make(map[string]bool)
		}
		componentFiles[item.Component][item.File] = true
	}

	var summary TechDebtSummary
	for index := range items {
		item := &items[index]
		summary.TotalMarkers++
		switch item.Marker {
		case "TODO":
			summary.TODOs++
		case "FIXME":
			summary.FIXMEs++
		case "HACK":
			summary.Hacks++
		case "PANIC":
			summary.Panics++
		case "STUB":
			summary.Stubs++
		}
		for _, document := range documents {
			if documentCoversDebt(document.Text, *item) {
				item.Coverage, item.CoverageRef, item.Recommendation = document.Coverage, document.Ref, "none"
				break
			}
		}
		if item.Coverage != "uncovered" {
			summary.CoveredCount++
			continue
		}
		summary.UncoveredCount++
		switch {
		case componentCounts[item.Component] > 50 && len(componentFiles[item.Component]) > 5:
			item.Recommendation = "add_to_roadmap"
			summary.RecommendedRoadmaps++
		case componentCounts[item.Component] > 15:
			item.Recommendation = "create_spec"
			summary.RecommendedSpecs++
		default:
			item.Recommendation = "create_followup"
			summary.RecommendedFollowups++
		}
	}

	return &TechDebtReportState{
		SchemaVersion: 1, EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
		BaselineCommit: s.resolveGitCommit(), Summary: summary, Items: items,
		ScannedFiles: scanned, Complete: coverage.Complete(), Coverage: coverage,
	}, nil
}

// SaveTechDebtReport writes technical-debt.json and technical-debt.md in .pose/.
func (s Store) SaveTechDebtReport(report *TechDebtReportState) error {
	if report == nil {
		return fmt.Errorf("technical-debt report is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.TechDebtStatePath()), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.AssessmentsDir(), 0o755); err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.TechDebtStatePath(), append(jsonData, '\n'), 0o644); err != nil {
		return err
	}

	md := fmt.Sprintf(`# Technical Debt Assessment: %s

> **Gerado por**: POSE Technical Debt Engine (`+"`pose assess tech-debt`"+`)
> **Data de Avaliação**: %s
> **Baseline Commit**: %s

## 1. Resumo Executivo

- **Total de Marcadores**: %d
- **TODOs**: %d | **FIXMEs**: %d | **HACKs**: %d | **Panics**: %d | **Stubs**: %d
- **Cobertos por Backlog Ativo**: %d
- **Não Cobertos**: %d
- **Recomendações**: %d Follow-ups | %d Specs | %d Roadmaps
%s
## 2. Ocorrências

| ID | Marcador | Componente | Arquivo e Linha | Cobertura | Evidência | Recomendação |
|---|---|---|---|---|---|---|
`, s.projectLabel(), report.EvaluatedAt, report.BaselineCommit, report.Summary.TotalMarkers,
		report.Summary.TODOs, report.Summary.FIXMEs, report.Summary.Hacks, report.Summary.Panics,
		report.Summary.Stubs, report.Summary.CoveredCount, report.Summary.UncoveredCount,
		report.Summary.RecommendedFollowups, report.Summary.RecommendedSpecs, report.Summary.RecommendedRoadmaps,
		techDebtScopeMarkdown(report))
	limit := len(report.Items)
	if limit > 100 {
		limit = 100
	}
	for _, item := range report.Items[:limit] {
		fileLink := fmt.Sprintf("[%s:%d](%s)", filepath.Base(item.File), item.Line, item.Link)
		md += fmt.Sprintf("| %s | `%s` | `%s` | %s | `%s` | `%s` | `%s` |\n",
			item.ID, item.Marker, markdownCell(item.Component), fileLink,
			item.Coverage, markdownCell(item.CoverageRef), item.Recommendation)
	}
	if len(report.Items) > limit {
		md += fmt.Sprintf("\n> Exibindo %d de %d ocorrências; consulte `.pose/state/technical-debt.json` para o conjunto completo.\n", limit, len(report.Items))
	}
	md += "\n## 3. Política de Ação\n\n" +
		"- Registre dívida local não coberta como follow-up.\n" +
		"- Abra uma spec quando um componente concentrar dívida recorrente.\n" +
		"- Estenda o roadmap quando a dívida atravessar vários arquivos e exigir coordenação sistêmica.\n" +
		"- Não crie recomendação nova quando uma spec, roadmap ou follow-up ativo já cobrir a ocorrência.\n"
	return os.WriteFile(s.TechDebtReportPath(), []byte(md), 0o644)
}

// techDebtScopeMarkdown says what the scan covered, so a report with no markers does not
// read as a project with none.
func techDebtScopeMarkdown(report *TechDebtReportState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- **Escopo do scan**: %d arquivo(s) de código lidos", report.ScannedFiles)
	if report.Complete {
		b.WriteString("; nenhum arquivo ilegível\n")
	} else {
		fmt.Fprintf(&b, "; **INCOMPLETO** — %s. Zero marcadores não prova ausência de dívida nos arquivos não lidos.\n", report.Coverage.Describe(""))
	}
	for _, e := range report.Coverage.Exclusions {
		fmt.Fprintf(&b, "- **Excluídos (%s)**: %d arquivo(s) — %s\n", e.Pattern, e.Count, e.Reason)
	}
	return b.String()
}

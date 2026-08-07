package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// integrationGapID derives a gap identifier from the contract it describes, so
// the same contract keeps the same id across runs.
//
// The ids used to be positional (`GAP-001`, `GAP-002`, …), which meant adding
// one contract renumbered every later gap: a report, ticket or review comment
// citing `GAP-032` silently came to mean a different contract.
func integrationGapID(protocol, name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(protocol)) + "\x00" + strings.ToLower(strings.TrimSpace(name))))
	return "GAP-" + hex.EncodeToString(sum[:4])
}

// IntegrationGap describes a missing provider or consumer observed in the repository.
type IntegrationGap struct {
	GapID       string `json:"gap_id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Provider    string `json:"provider"`
	Consumer    string `json:"consumer"`
	Description string `json:"description"`
}

// IntegrationContract represents a repository-derived provider-consumer relationship.
type IntegrationContract struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Provider string `json:"provider"`
	Consumer string `json:"consumer"`
	Status   string `json:"status"`
}

type IntegrationSummary struct {
	TotalIntegrations int `json:"total_integrations"`
	ActiveContracts   int `json:"active_contracts"`
	IdentifiedGaps    int `json:"identified_gaps"`
}

type IntegrationMatrix struct {
	SchemaVersion  int                   `json:"schema_version"`
	EvaluatedAt    string                `json:"evaluated_at"`
	BaselineCommit string                `json:"baseline_commit"`
	Summary        IntegrationSummary    `json:"summary"`
	Contracts      []IntegrationContract `json:"contracts"`
	Gaps           []IntegrationGap      `json:"gaps"`
}

type integrationObservation struct {
	Protocol      string
	Token         string
	Name          string
	Aliases       []string
	Providers     map[string]bool
	Consumers     map[string]bool
	ProviderFiles map[string]bool
	ConsumerFiles map[string]bool
}

var (
	httpHandleRouteRE = regexp.MustCompile(`(?i)\b(?:handlefunc|handle|route)\s*\(\s*["'` + "`" + `](/[^"'` + "`" + `\s?#]*)`)
	httpRouterRouteRE = regexp.MustCompile(`(?i)\b(?:router|mux|app|server|group|r)\s*\.\s*(get|post|put|patch|delete|options|head)\s*\(\s*["'` + "`" + `](/[^"'` + "`" + `\s?#]*)`)
	httpAnnotationRE  = regexp.MustCompile(`(?i)@(?:get|post|put|patch|delete|request)mapping\s*\(\s*(?:value\s*=\s*)?["'](/[^"'\s?#]*)`)
	httpPathLiteralRE = regexp.MustCompile(`["'` + "`" + `](/[^"'` + "`" + `\s?#]*)["'` + "`" + `]`)
	httpClientRE      = regexp.MustCompile(`(?i)\b(?:fetch|axios(?:\.(?:get|post|put|patch|delete))?|newrequest(?:withcontext)?|requests?\.(?:get|post|put|patch|delete)|http\.(?:get|post)|client\.(?:get|post|put|patch|delete))\s*\(`)
	quotedTokenRE     = regexp.MustCompile(`["'` + "`" + `]([A-Za-z0-9][A-Za-z0-9._-]{2,})["'` + "`" + `]`)
	topicProviderRE   = regexp.MustCompile(`(?i)\b(?:publish|produce|emit|sendmessage|send_message|writemessage|write_message)\s*\(`)
	topicConsumerRE   = regexp.MustCompile(`(?i)\b(?:subscribe|consume|listen|readmessage|read_message|receivemessage|receive_message)\s*\(`)
	mcpToolNameRE     = regexp.MustCompile(`(?i)["']name["']\s*:\s*["']([A-Za-z][A-Za-z0-9_-]{2,})["']`)
	mcpCallContextRE  = regexp.MustCompile(`(?i)\b(?:tools/call|calltool|call_tool|invoke_tool|tool_name)\b`)
	protoSymbolRE     = regexp.MustCompile(`(?m)^\s*(?:service|message)\s+([A-Za-z][A-Za-z0-9_]*)\b`)
	openAPIYAMLPathRE = regexp.MustCompile(`^\s*(/[^\s:#?]+)\s*:\s*(?:#.*)?$`)
)

func isCommentOnlyObservationLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "--")
}

func (s Store) IntegrationStatePath() string {
	return filepath.Join(s.Root, ".pose", "state", "integrations.json")
}

func (s Store) IntegrationReportPath() string {
	return filepath.Join(s.Root, ".pose", "assessments", "integrations.md")
}

func ensureObservation(observations map[string]*integrationObservation, protocol, token, name string) *integrationObservation {
	key := protocol + "\x00" + token
	if existing := observations[key]; existing != nil {
		return existing
	}
	observation := &integrationObservation{
		Protocol: protocol, Token: token, Name: name,
		Providers: make(map[string]bool), Consumers: make(map[string]bool),
		ProviderFiles: make(map[string]bool), ConsumerFiles: make(map[string]bool),
	}
	observations[key] = observation
	return observation
}

func addProvider(observation *integrationObservation, file assessmentFile) {
	observation.Providers[file.Component] = true
	observation.ProviderFiles[file.RelPath] = true
}

func addConsumer(observation *integrationObservation, file assessmentFile) {
	observation.Consumers[file.Component] = true
	observation.ConsumerFiles[file.RelPath] = true
}

func isIntegrationTestFile(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	return strings.Contains(lower, "/testdata/") || strings.Contains(lower, "/fixtures/") || strings.Contains(lower, "/fixture/") ||
		strings.Contains(lower, "/tests/") || strings.Contains(lower, "/test/") ||
		strings.Contains(base, "_test.") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.")
}

func routeObservations(file assessmentFile, observations map[string]*integrationObservation) {
	for _, line := range strings.Split(file.Content, "\n") {
		if isCommentOnlyObservationLine(line) {
			continue
		}
		if match := httpHandleRouteRE.FindStringSubmatch(line); len(match) > 1 {
			observation := ensureObservation(observations, "rest", match[1], "HTTP "+match[1])
			addProvider(observation, file)
		}
		if match := httpRouterRouteRE.FindStringSubmatch(line); len(match) > 2 {
			method, path := strings.ToUpper(match[1]), match[2]
			observation := ensureObservation(observations, "rest", path, method+" "+path)
			addProvider(observation, file)
		}
		if match := httpAnnotationRE.FindStringSubmatch(line); len(match) > 1 {
			observation := ensureObservation(observations, "rest", match[1], "HTTP "+match[1])
			addProvider(observation, file)
		}
		if operation := httpClientRE.FindStringIndex(line); operation != nil {
			if match := httpPathLiteralRE.FindStringSubmatch(line[operation[1]:]); len(match) > 1 {
				observation := ensureObservation(observations, "rest", match[1], "HTTP "+match[1])
				addConsumer(observation, file)
			}
		}
	}
}

func openAPIObservations(file assessmentFile, observations map[string]*integrationObservation) {
	lowerPath := strings.ToLower(file.RelPath)
	if !strings.Contains(lowerPath, "openapi") && !strings.Contains(lowerPath, "swagger") {
		return
	}
	if file.Ext != ".yaml" && file.Ext != ".yml" && file.Ext != ".json" {
		return
	}
	if file.Ext == ".json" {
		var document struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}
		if json.Unmarshal([]byte(file.Content), &document) != nil {
			return
		}
		for path := range document.Paths {
			if !strings.HasPrefix(path, "/") {
				continue
			}
			observation := ensureObservation(observations, "rest", path, "HTTP "+path)
			addProvider(observation, file)
		}
		return
	}
	for _, line := range strings.Split(file.Content, "\n") {
		if isCommentOnlyObservationLine(line) {
			continue
		}
		match := openAPIYAMLPathRE.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		observation := ensureObservation(observations, "rest", match[1], "HTTP "+match[1])
		addProvider(observation, file)
	}
}

func topicObservations(file assessmentFile, observations map[string]*integrationObservation) {
	for _, line := range strings.Split(file.Content, "\n") {
		if isCommentOnlyObservationLine(line) {
			continue
		}
		for _, role := range []struct {
			expression *regexp.Regexp
			provider   bool
		}{{topicProviderRE, true}, {topicConsumerRE, false}} {
			operation := role.expression.FindStringIndex(line)
			if operation == nil {
				continue
			}
			match := quotedTokenRE.FindStringSubmatch(line[operation[1]:])
			if len(match) < 2 {
				continue
			}
			topic := match[1]
			if !strings.ContainsAny(topic, ".-") {
				continue
			}
			observation := ensureObservation(observations, "message", topic, "Message topic "+topic)
			if role.provider {
				addProvider(observation, file)
			} else {
				addConsumer(observation, file)
			}
		}
	}
}

func mcpProviderObservations(file assessmentFile, observations map[string]*integrationObservation) {
	lowerPath := strings.ToLower(file.RelPath)
	if !strings.Contains(lowerPath, "mcp") && !strings.Contains(lowerPath, "tool") && !strings.Contains(lowerPath, "catalog") {
		return
	}
	lowerContent := strings.ToLower(file.Content)
	if !strings.Contains(lowerContent, `"tools"`) && !strings.Contains(lowerContent, "inputschema") &&
		!strings.Contains(lowerContent, `"description"`) {
		return
	}
	if file.Ext == ".json" {
		var catalog struct {
			Tools []struct {
				Name        string          `json:"name"`
				InputSchema json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		}
		if json.Unmarshal([]byte(file.Content), &catalog) != nil {
			return
		}
		for _, tool := range catalog.Tools {
			if tool.Name == "" || len(tool.InputSchema) == 0 {
				continue
			}
			observation := ensureObservation(observations, "mcp", tool.Name, "MCP tool "+tool.Name)
			addProvider(observation, file)
		}
		return
	}
	matches := mcpToolNameRE.FindAllStringSubmatchIndex(file.Content, -1)
	for index, positions := range matches {
		segmentEnd := len(file.Content)
		if index+1 < len(matches) {
			segmentEnd = matches[index+1][0]
		}
		if segmentEnd-positions[0] > 4096 {
			segmentEnd = positions[0] + 4096
		}
		segment := strings.ToLower(file.Content[positions[1]:segmentEnd])
		if !strings.Contains(segment, "inputschema") {
			continue
		}
		toolName := file.Content[positions[2]:positions[3]]
		observation := ensureObservation(observations, "mcp", toolName, "MCP tool "+toolName)
		addProvider(observation, file)
	}
}

func protoProviderObservation(file assessmentFile, observations map[string]*integrationObservation) {
	if file.Ext != ".proto" {
		return
	}
	token := filepath.ToSlash(file.RelPath)
	observation := ensureObservation(observations, "protobuf", token, "Protobuf "+token)
	addProvider(observation, file)
	observation.Aliases = append(observation.Aliases, filepath.Base(token))
	for _, match := range protoSymbolRE.FindAllStringSubmatch(file.Content, -1) {
		observation.Aliases = append(observation.Aliases, match[1])
	}
}

func inferConsumers(files []assessmentFile, observations map[string]*integrationObservation) {
	for _, observation := range observations {
		for _, file := range files {
			if isIntegrationTestFile(file.RelPath) || observation.ProviderFiles[file.RelPath] || observation.ConsumerFiles[file.RelPath] {
				continue
			}
			switch observation.Protocol {
			case "protobuf":
				for _, alias := range observation.Aliases {
					if alias != "" && strings.Contains(file.Content, alias) {
						addConsumer(observation, file)
						break
					}
				}
			case "mcp":
				if strings.Contains(file.Content, observation.Token) && mcpCallContextRE.MatchString(file.Content) {
					addConsumer(observation, file)
				}
			}
		}
	}
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func observedPart(values map[string]bool) string {
	if len(values) == 0 {
		return "unobserved"
	}
	return strings.Join(sortedSet(values), ", ")
}

// AnalyzeIntegrations derives contracts and gaps from the selected repository.
func (s Store) AnalyzeIntegrations() (*IntegrationMatrix, error) {
	files, err := s.assessmentFiles()
	if err != nil {
		return nil, err
	}
	observations := make(map[string]*integrationObservation)
	for _, file := range files {
		if isIntegrationTestFile(file.RelPath) {
			continue
		}
		protoProviderObservation(file, observations)
		routeObservations(file, observations)
		openAPIObservations(file, observations)
		topicObservations(file, observations)
		mcpProviderObservations(file, observations)
	}
	inferConsumers(files, observations)

	ordered := make([]*integrationObservation, 0, len(observations))
	for _, observation := range observations {
		ordered = append(ordered, observation)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Protocol != ordered[j].Protocol {
			return ordered[i].Protocol < ordered[j].Protocol
		}
		if ordered[i].Name != ordered[j].Name {
			return ordered[i].Name < ordered[j].Name
		}
		return ordered[i].Token < ordered[j].Token
	})

	matrix := &IntegrationMatrix{
		SchemaVersion: 1, EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
		BaselineCommit: s.resolveGitCommit(), Contracts: []IntegrationContract{}, Gaps: []IntegrationGap{},
	}
	for _, observation := range ordered {
		hasProvider, hasConsumer := len(observation.Providers) > 0, len(observation.Consumers) > 0
		status := "gap"
		if hasProvider && hasConsumer {
			status = "active"
			matrix.Summary.ActiveContracts++
		}
		provider, consumer := observedPart(observation.Providers), observedPart(observation.Consumers)
		matrix.Contracts = append(matrix.Contracts, IntegrationContract{
			Name: observation.Name, Protocol: observation.Protocol,
			Provider: provider, Consumer: consumer, Status: status,
		})
		if hasProvider == hasConsumer {
			continue
		}
		gap := IntegrationGap{
			GapID: integrationGapID(observation.Protocol, observation.Name), Provider: provider, Consumer: consumer,
		}
		if hasProvider {
			gap.Title = "No consumer observed for " + observation.Name
			gap.Severity = "medium"
			gap.Description = "A provider declaration was observed, but no repository consumer reference was found."
		} else {
			gap.Title = "No provider observed for " + observation.Name
			gap.Severity = "high"
			gap.Description = "A consumer usage was observed, but no matching repository provider declaration was found."
		}
		matrix.Gaps = append(matrix.Gaps, gap)
	}
	matrix.Summary.TotalIntegrations = len(matrix.Contracts)
	matrix.Summary.IdentifiedGaps = len(matrix.Gaps)
	return matrix, nil
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

// SaveIntegrationMatrix writes integrations.json and integrations.md in .pose/.
func (s Store) SaveIntegrationMatrix(matrix *IntegrationMatrix) error {
	if matrix == nil {
		return fmt.Errorf("integration matrix is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.IntegrationStatePath()), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.AssessmentsDir(), 0o755); err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.IntegrationStatePath(), append(jsonData, '\n'), 0o644); err != nil {
		return err
	}

	md := fmt.Sprintf(`# Integration Assessment: %s

> **Gerado por**: POSE Integration Engine (`+"`pose assess integrate`"+`)
> **Data de Avaliação**: %s
> **Baseline Commit**: %s

## 1. Resumo Executivo

- **Total de Contratos Observados**: %d
- **Contratos com Provedor e Consumidor**: %d
- **Gaps de Integração**: %d

## 2. Matriz de Contratos

| Nome do Contrato | Protocolo | Provedor | Consumidor | Status |
|---|---|---|---|---|
`, s.projectLabel(), matrix.EvaluatedAt, matrix.BaselineCommit, matrix.Summary.TotalIntegrations, matrix.Summary.ActiveContracts, matrix.Summary.IdentifiedGaps)
	for _, contract := range matrix.Contracts {
		md += fmt.Sprintf("| %s | `%s` | `%s` | `%s` | `%s` |\n",
			markdownCell(contract.Name), markdownCell(contract.Protocol), markdownCell(contract.Provider),
			markdownCell(contract.Consumer), markdownCell(contract.Status))
	}
	md += "\n## 3. Gaps Observados\n\n"
	if len(matrix.Gaps) == 0 {
		md += "Nenhum gap estático foi observado.\n"
	}
	for _, gap := range matrix.Gaps {
		md += fmt.Sprintf("### [%s] %s\n- **Severidade**: %s\n- **Provedor**: `%s`\n- **Consumidor**: `%s`\n- **Evidência**: %s\n\n",
			gap.GapID, gap.Title, gap.Severity, gap.Provider, gap.Consumer, gap.Description)
	}
	return os.WriteFile(s.IntegrationReportPath(), []byte(strings.TrimRight(md, "\n")+"\n"), 0o644)
}

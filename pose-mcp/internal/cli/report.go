package cli

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var reportSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

type reportRecord struct {
	GeneratedAt       string `json:"generated_at"`
	Sequence          int    `json:"sequence"`
	Task              string `json:"task"`
	TaskSlug          string `json:"task_slug"`
	ReportType        string `json:"report_type"`
	Spec              string `json:"spec"`
	Workflow          string `json:"workflow"`
	Rules             string `json:"rules"`
	ValidationProfile string `json:"validation_profile"`
	Context           string `json:"context"`
	Risk              string `json:"risk"`
	Outcome           string `json:"outcome"`
	OutcomeSource     string `json:"outcome_source"`
	StableHash        string `json:"stable_hash"`
	ReportPath        string `json:"report_path"`
}

func cmdReport(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	values := map[string]string{"type": "standard", "outcome": "", "context": "not-provided", "validation-profile": "not-provided"}
	gitStage := false
	valueFlags := map[string]bool{
		"task": true, "spec": true, "risk": true, "workflow": true, "rules": true,
		"validate-output": true, "type": true, "context": true,
		"validation-profile": true, "outcome": true, "since": true,
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "--git-stage" {
			gitStage = true
			continue
		}
		if !strings.HasPrefix(args[i], "--") {
			fmt.Fprintln(stderr, cliText(locale, "Usage: pose report --task <description> [options]", "Uso: pose report --task <descrição> [opções]"))
			return 2
		}
		key := strings.TrimPrefix(args[i], "--")
		if !valueFlags[key] {
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: --%s\n", "Erro: argumento inválido: --%s\n"), key)
			return 2
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			fmt.Fprintf(stderr, cliText(locale, "Error: --%s requires a value.\n", "Erro: --%s exige um valor.\n"), key)
			return 2
		}
		i++
		values[key] = args[i]
	}
	task := strings.TrimSpace(values["task"])
	if task == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --task is required.", "Erro: --task é obrigatório."))
		return 2
	}
	if values["type"] != "standard" && values["type"] != "doc-audit" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --type must be 'standard' or 'doc-audit'.", "Erro: --type deve ser 'standard' ou 'doc-audit'."))
		return 2
	}
	outcome, outcomeSource := values["outcome"], "manual"
	validateOutput := values["validate-output"]
	if validateOutput != "" {
		clean := validateOutput
		if !filepath.IsAbs(clean) {
			clean = filepath.Join(root, filepath.FromSlash(clean))
		}
		rel, err := filepath.Rel(root, filepath.Clean(clean))
		if err != nil || !confinedRelativePath(rel) {
			fmt.Fprintln(stderr, cliText(locale, "Error: --validate-output must remain inside the project.", "Erro: --validate-output deve permanecer dentro do projeto."))
			return 2
		}
		validateOutput = clean
	}
	if validateOutput == "" {
		for _, candidate := range []string{filepath.Join(root, ".pose", "reports", "pose-validate.latest.log"), filepath.Join(root, ".pose", "pose-validate.log")} {
			if _, err := os.Stat(candidate); err == nil {
				validateOutput = candidate
				break
			}
		}
	}
	validationCommands, validationResults, derivedOutcome := parseValidationLog(validateOutput)
	if outcome == "" && derivedOutcome != "" {
		outcome, outcomeSource = derivedOutcome, "derived"
	}
	if outcome == "" {
		outcome = "unknown"
	}
	if !map[string]bool{"pass": true, "fail": true, "partial": true, "skipped": true, "unknown": true}[outcome] {
		fmt.Fprintln(stderr, cliText(locale, "Error: invalid --outcome (use pass|fail|partial|skipped|unknown).", "Erro: --outcome inválido (use pass|fail|partial|skipped|unknown)."))
		return 2
	}
	slug := strings.Trim(reportSlugChars.ReplaceAllString(strings.ToLower(task), "-"), "-")
	if slug == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --task does not produce a valid slug.", "Erro: --task não produz um slug válido."))
		return 2
	}
	now := time.Now().UTC()
	reports := filepath.Join(root, ".pose", "reports")
	historyDir := filepath.Join(reports, "history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: creating reports: %v\n", "Erro: criando relatórios: %v\n"), err)
		return 1
	}
	reportPath := filepath.Join(reports, now.Format("2006-01-02")+"-"+values["type"]+"-"+slug+".md")
	historyPath := filepath.Join(historyDir, values["type"]+"-"+slug+".jsonl")
	previous := readLastReportRecord(historyPath)
	stable := map[string]string{
		"task_slug": slug, "spec": values["spec"], "report_type": values["type"],
		"workflow": values["workflow"], "rules": values["rules"],
		"validation_profile": values["validation-profile"], "context": values["context"],
	}
	stableJSON := stableReportJSON(stable)
	digest := sha256.Sum256(stableJSON)
	stableHash := hex.EncodeToString(digest[:])
	sequence, compareStatus, previousAt, changes := 1, "first-run", "", []string{}
	if previous != nil {
		sequence, previousAt = previous.Sequence+1, previous.GeneratedAt
		if previous.StableHash == stableHash {
			compareStatus = "stable"
		} else {
			compareStatus = "changed"
			old := map[string]string{"task_slug": previous.TaskSlug, "spec": previous.Spec, "report_type": previous.ReportType, "workflow": previous.Workflow, "rules": previous.Rules, "validation_profile": previous.ValidationProfile, "context": previous.Context}
			keys := make([]string, 0, len(stable))
			for key := range stable {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if old[key] != stable[key] {
					changes = append(changes, fmt.Sprintf("%s: %q -> %q", key, old[key], stable[key]))
				}
			}
		}
	}
	filesChanged := reportChangedFiles(root, values["since"])
	markdown := renderReportMarkdown(now, values, task, slug, outcome, outcomeSource, filesChanged, validationCommands, validationResults, sequence, stableHash, compareStatus, previousAt, changes)
	if err := os.WriteFile(reportPath, []byte(markdown), 0o644); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing report: %v\n", "Erro: escrevendo relatório: %v\n"), err)
		return 1
	}
	record := reportRecord{now.Format(time.RFC3339), sequence, task, slug, values["type"], values["spec"], values["workflow"], values["rules"], values["validation-profile"], values["context"], values["risk"], outcome, outcomeSource, stableHash, reportPath}
	history, err := os.OpenFile(historyPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing history: %v\n", "Erro: escrevendo histórico: %v\n"), err)
		return 1
	}
	encodeErr := json.NewEncoder(history).Encode(record)
	closeErr := history.Close()
	if encodeErr != nil || closeErr != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: serializing history: %v %v\n", "Erro: serializando histórico: %v %v\n"), encodeErr, closeErr)
		return 1
	}
	if gitStage {
		rel, _ := filepath.Rel(root, historyPath)
		if err := exec.Command("git", "-C", root, "add", "--", filepath.ToSlash(rel)).Run(); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: staging history: %v\n", "Erro: adicionando histórico ao stage: %v\n"), err)
			return 1
		}
	}
	fmt.Fprintln(stdout, reportPath)
	return 0
}

func parseValidationLog(path string) ([]string, []string, string) {
	commands, results, outcome := []string{}, []string{}, ""
	if path == "" {
		return commands, results, outcome
	}
	file, err := os.Open(path)
	if err != nil {
		return commands, results, outcome
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "  -> ") {
			commands = append(commands, strings.TrimPrefix(line, "  -> "))
		}
		if strings.HasPrefix(line, " - [") || strings.HasPrefix(line, "Result:") || strings.HasPrefix(line, "Resultado:") {
			results = append(results, line)
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.Contains(upper, "FAILURE_TOLERATED") || strings.Contains(upper, "FALHA_TOLERADA"):
			outcome = "partial"
		case strings.Contains(upper, "RESULT: SUCCESS") || strings.Contains(upper, "RESULTADO: SUCESSO"):
			outcome = "pass"
		case strings.Contains(upper, "RESULT: FAILURE") || strings.Contains(upper, "RESULTADO: FALHA"):
			outcome = "fail"
		}
	}
	return commands, results, outcome
}

func readLastReportRecord(path string) *reportRecord {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var last *reportRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record reportRecord
		if json.Unmarshal(scanner.Bytes(), &record) == nil {
			copy := record
			last = &copy
		}
	}
	return last
}

func reportChangedFiles(root, since string) []string {
	args := []string{"-C", root}
	if since != "" {
		args = append(args, "diff", "--name-only", since)
	} else {
		args = append(args, "status", "--porcelain=v1")
	}
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	files := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		if since == "" && len(line) >= 4 {
			line = line[3:]
		}
		files = append(files, line)
	}
	return files
}

func renderReportMarkdown(now time.Time, values map[string]string, task, slug, outcome, outcomeSource string, files, commands, results []string, sequence int, hash, compareStatus, previousAt string, changes []string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# POSE Report - %s\n\n## Report Type\n- %s\n\n## Task\n- %s\n- Task slug: %s\n", now.Format("2006-01-02"), values["type"], task, slug)
	if values["spec"] != "" {
		fmt.Fprintf(&builder, "- Spec: %s\n", values["spec"])
	}
	if values["workflow"] != "" {
		fmt.Fprintf(&builder, "- Workflow: %s\n", values["workflow"])
	}
	fmt.Fprintf(&builder, "\n## Outcome\n- Outcome: %s (source: %s)\n\n## Rules Applied\n", outcome, outcomeSource)
	writeReportList(&builder, splitNonEmpty(values["rules"], ","), "_Not provided_")
	builder.WriteString("\n## Files Changed\n")
	writeReportList(&builder, files, "_No files detected_")
	builder.WriteString("\n## Validation Commands\n")
	writeReportList(&builder, commands, "_Fill manually_")
	builder.WriteString("\n## Results\n")
	writeReportList(&builder, results, "_No validation output detected_")
	fmt.Fprintf(&builder, "\n## Execution Metadata\n- Generated at (UTC): %s\n- Context: %s\n- Validation profile: %s\n- Sequence for task/spec: %d\n- Stable comparison hash: %s\n", now.Format(time.RFC3339), values["context"], values["validation-profile"], sequence, hash)
	if previousAt == "" {
		previousAt = "_No previous execution_"
	}
	fmt.Fprintf(&builder, "\n## Historical Comparison\n- Previous execution: %s\n- Status: %s\n- Stable field diffs:\n", previousAt, compareStatus)
	writeReportList(&builder, changes, "_No changes in stable fields_")
	builder.WriteString("\n## Risks\n")
	writeReportList(&builder, splitNonEmpty(values["risk"], "\n"), "_No risks provided_")
	builder.WriteString("\n## Follow-ups\n- _Add next steps if needed._\n\n## Human Review Needed\n- [ ] Review functional impact\n- [ ] Review validation coverage\n- [ ] Approve merge\n")
	return builder.String()
}

func writeReportList(builder *strings.Builder, values []string, empty string) {
	if len(values) == 0 {
		fmt.Fprintf(builder, "- %s\n", empty)
		return
	}
	for _, value := range values {
		fmt.Fprintf(builder, "- %s\n", strings.TrimSpace(value))
	}
}

func splitNonEmpty(value, separator string) []string {
	result := []string{}
	for _, item := range strings.Split(value, separator) {
		if strings.TrimSpace(item) != "" {
			result = append(result, strings.TrimSpace(item))
		}
	}
	return result
}

func stableReportJSON(values map[string]string) []byte {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteByte('{')
	for index, key := range keys {
		if index > 0 {
			builder.WriteString(", ")
		}
		encodedKey, _ := json.Marshal(key)
		encodedValue, _ := json.Marshal(values[key])
		builder.Write(encodedKey)
		builder.WriteString(": ")
		builder.Write(encodedValue)
	}
	builder.WriteByte('}')
	return []byte(builder.String())
}

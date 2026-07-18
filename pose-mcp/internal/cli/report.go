package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var reportSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// cmdReport creates the native baseline report and history record. Advanced
// comparison fields remain delegated until their parity fixture is ported.
func cmdReport(root string, args []string, stdout, stderr io.Writer) int {
	values := map[string]string{"type": "standard", "outcome": "unknown"}
	for i := 0; i < len(args); i++ {
		if args[i] == "--git-stage" {
			continue
		}
		if !strings.HasPrefix(args[i], "--") || i+1 >= len(args) {
			fmt.Fprintln(stderr, "Uso: pose report --task <descrição> [--outcome pass|fail|partial|skipped|unknown]")
			return 2
		}
		key := strings.TrimPrefix(args[i], "--")
		i++
		values[key] = args[i]
	}
	task := strings.TrimSpace(values["task"])
	if task == "" {
		fmt.Fprintln(stderr, "Erro: --task é obrigatório.")
		return 2
	}
	outcome := values["outcome"]
	if !map[string]bool{"pass": true, "fail": true, "partial": true, "skipped": true, "unknown": true}[outcome] {
		fmt.Fprintln(stderr, "Erro: --outcome inválido.")
		return 2
	}
	slug := strings.Trim(reportSlugChars.ReplaceAllString(strings.ToLower(task), "-"), "-")
	if slug == "" {
		fmt.Fprintln(stderr, "Erro: --task não gera slug válido.")
		return 2
	}
	now := time.Now().UTC()
	reports := filepath.Join(root, ".pose", "reports")
	if err := os.MkdirAll(filepath.Join(reports, "history"), 0o755); err != nil {
		fmt.Fprintf(stderr, "Erro: criar reports: %v\n", err)
		return 1
	}
	reportPath := filepath.Join(reports, now.Format("2006-01-02")+"-"+values["type"]+"-"+slug+".md")
	content := fmt.Sprintf("# POSE Report - %s\n\n## Task\n- %s\n\n## Outcome\n- Outcome: %s\n", now.Format("2006-01-02"), task, outcome)
	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "Erro: escrever relatório: %v\n", err)
		return 1
	}
	historyPath := filepath.Join(reports, "history", values["type"]+"-"+slug+".jsonl")
	history, err := os.OpenFile(historyPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: escrever history: %v\n", err)
		return 1
	}
	defer history.Close()
	if err := json.NewEncoder(history).Encode(map[string]string{"generated_at": now.Format(time.RFC3339), "task": task, "task_slug": slug, "outcome": outcome, "report_type": values["type"]}); err != nil {
		fmt.Fprintf(stderr, "Erro: serializar history: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Report criado: %s\n", reportPath)
	return 0
}

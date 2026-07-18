package cli

// Native port of the history-check gate (spec pose-cli-native-gates).
// Implements the tracked-history contract natively.

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func cmdHistoryCheck(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	mode := "tolerant"
	for _, a := range args {
		switch a {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "-h", "--help":
			fmt.Fprintln(stdout, cliText(locale, "Usage: pose history-check [--strict|--tolerant]", "Uso: pose history-check [--strict|--tolerant]"))
			return 0
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), a)
			return 2
		}
	}
	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose history-check: %v\n", err)
		return 2
	}
	historyDir := filepath.Join(root, ".pose", "reports", "history")
	if fi, err := os.Stat(historyDir); err != nil || !fi.IsDir() {
		fmt.Fprintf(stderr, cliText(locale, "Error: history directory not found: %s\n", "Erro: history dir ausente: %s\n"), historyDir)
		return 2
	}
	if err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: not a git repository: %s\n", "Erro: não é um repositório git: %s\n"), root)
		return 2
	}

	entries, err := os.ReadDir(historyDir)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: reading %s: %v\n", "Erro: lendo %s: %v\n"), historyDir, err)
		return 2
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	untracked, modified, clean := 0, 0, 0
	for _, name := range files {
		rel := filepath.ToSlash(filepath.Join(".pose", "reports", "history", name))
		out, _ := exec.Command("git", "-C", root, "status", "--porcelain=v1", "--", rel).Output()
		status := strings.TrimRight(string(out), "\n")
		switch {
		case status == "":
			clean++
		case strings.HasPrefix(status, "??"):
			fmt.Fprintf(stderr, cliText(locale, "[WARNING] untracked JSONL: %s\n", "[AVISO] JSONL untracked: %s\n"), rel)
			untracked++
		case strings.HasPrefix(status, " M "), strings.HasPrefix(status, " D "), strings.HasPrefix(status, " T "):
			fmt.Fprintf(stderr, cliText(locale, "[WARNING] modified unstaged JSONL: %s\n", "[AVISO] JSONL modificado e não-staged: %s\n"), rel)
			modified++
		default:
			// Index has changes (staged) — OK for the gate.
			clean++
		}
	}

	fmt.Fprintf(stdout, "history.untracked=%d\n", untracked)
	fmt.Fprintf(stdout, "history.modified_unstaged=%d\n", modified)
	fmt.Fprintf(stdout, "history.staged_or_clean=%d\n", clean)

	if problems := untracked + modified; problems > 0 {
		fmt.Fprintf(stdout, "Resultado: FALHA (%d JSONL fora do versionamento)\n", problems)
		if mode == "strict" {
			fmt.Fprintln(stderr, cliText(locale, "To fix: git add .pose/reports/history/", "Para corrigir: git add .pose/reports/history/"))
			return 1
		}
		fmt.Fprintln(stdout, cliText(locale, "Tolerant mode: record and version before the next merge.", "Modo tolerant: registrar e versionar antes do próximo merge."))
		fmt.Fprintln(stdout, "Resultado: FALHA_TOLERADA")
		return 0
	}
	fmt.Fprintln(stdout, "Resultado: SUCESSO")
	return 0
}

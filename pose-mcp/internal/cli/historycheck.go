package cli

// Native port of the history-check gate (spec pose-cli-native-gates).
// Implements the tracked-history contract natively.

import (
	"fmt"
	"github.com/harne8/pose-mcp/internal/cli/cliout"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func cmdHistoryCheck(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	mode := "tolerant"
	args, flags, flagErr := splitOutputFlags(args)
	if flagErr != "" {
		fmt.Fprintf(stderr, cliText(locale, "Error: %s\n", "Erro: %s\n"), flagErr)
		return 2
	}
	for _, a := range args {
		switch a {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "-h", "--help":
			fmt.Fprintln(stdout, cliText(locale, "Usage: pose history-check [--strict|--tolerant] [--json] [--quiet] [--color auto|always|never]", "Uso: pose history-check [--strict|--tolerant] [--json] [--quiet] [--color auto|always|never]"))
			return 0
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), a)
			return 2
		}
	}
	gate := newGateOutput("history-check", flags, stdout, stderr)
	defer gate.Close()
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
			gate.r.Finding(cliout.Finding{State: cliout.StateWarning, Code: "untracked", Path: rel, Message: cliText(locale, "history JSONL is not tracked by Git", "JSONL de histórico não versionado no Git")})
			untracked++
		case strings.HasPrefix(status, " M "), strings.HasPrefix(status, " D "), strings.HasPrefix(status, " T "):
			gate.r.Finding(cliout.Finding{State: cliout.StateWarning, Code: "modified-unstaged", Path: rel, Message: cliText(locale, "history JSONL has unstaged changes", "JSONL de histórico com mudanças não staged")})
			modified++
		default:
			// Index has changes (staged) — OK for the gate.
			clean++
		}
	}

	gate.Field("history.untracked", strconv.Itoa(untracked))
	gate.Field("history.modified_unstaged", strconv.Itoa(modified))
	gate.Field("history.staged_or_clean", strconv.Itoa(clean))

	if problems := untracked + modified; problems > 0 {
		text := fmt.Sprintf(cliText(locale, "%d JSONL outside version control", "%d JSONL fora do versionamento"), problems)
		line := fmt.Sprintf("Resultado: FALHA (%d JSONL fora do versionamento)", problems)
		if mode == "strict" {
			gate.Verdict(cliout.StateFail, line, text)
			gate.r.Hint(cliText(locale, "To fix: git add .pose/reports/history/", "Para corrigir: git add .pose/reports/history/"))
			return 1
		}
		if !flags.JSON {
			fmt.Fprintln(stdout, line)
		}
		fmt.Fprintln(gate.result, cliText(locale, "Tolerant mode: record and version before the next merge.", "Modo tolerant: registrar e versionar antes do próximo merge."))
		gate.VerdictWord(cliout.StateWarning, "TOLERATED_FAILURE", "Resultado: FALHA_TOLERADA", text+cliText(locale, ", tolerated", ", tolerado"))
		return 0
	}
	gate.Verdict(cliout.StatePass, "Resultado: SUCESSO", "")
	return 0
}

package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/version"
)

func cmdReportLimitation(root string, args []string, stdout, stderr io.Writer) int {
	commandLocale := cliLocaleValue()
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }

	var title, body, kind string
	submitRemote := false
	kind = "limitation"

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--title", "-t":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, text("pose report-limitation: %s requires a value\n", "pose report-limitation: %s exige um valor\n"), a)
				return 2
			}
			title = args[i+1]
			i++
		case "--body", "--description", "-b":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, text("pose report-limitation: %s requires a value\n", "pose report-limitation: %s exige um valor\n"), a)
				return 2
			}
			body = args[i+1]
			i++
		case "--kind", "--type", "-k":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, text("pose report-limitation: %s requires a value\n", "pose report-limitation: %s exige um valor\n"), a)
				return 2
			}
			kind = strings.ToLower(args[i+1])
			i++
		case "--submit", "-s":
			submitRemote = true
		default:
			return usageError(stderr, text(
				"Usage: pose report-limitation --title \"...\" [--body \"...\"] [--kind limitation|suggestion|bug] [--submit]",
				"Uso: pose report-limitation --title \"...\" [--body \"...\"] [--kind limitation|suggestion|bug] [--submit]",
			))
		}
	}

	if title == "" {
		fmt.Fprintln(stderr, text("pose report-limitation: --title is required", "pose report-limitation: --title é obrigatório"))
		return 2
	}

	if kind != "limitation" && kind != "suggestion" && kind != "bug" {
		kind = "limitation"
	}

	// Prepare metadata context
	engineVer := version.Version
	osInfo := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	goVer := runtime.Version()
	timestamp := time.Now().Format(time.RFC3339)

	fullReport := fmt.Sprintf(`## Description
%s

---
### System Context (Auto-generated)
- **POSE Engine Version:** %s
- **OS/Arch:** %s
- **Go Version:** %s
- **Reported At:** %s
- **Kind:** %s
`, body, engineVer, osInfo, goVer, timestamp, kind)

	// Save local copy in .pose/feedback/
	feedbackDir := filepath.Join(root, ".pose", "feedback")
	if err := os.MkdirAll(feedbackDir, 0o755); err != nil {
		fmt.Fprintf(stderr, text("pose report-limitation: creating feedback directory: %v\n", "pose report-limitation: criando diretório de feedback: %v\n"), err)
		return 1
	}

	slug := slugify(title)
	filename := fmt.Sprintf("%s-%s.md", kind, slug)
	localPath := filepath.Join(feedbackDir, filename)

	localContent := fmt.Sprintf(`---
title: %q
kind: %s
engine_version: %s
reported_at: %s
---

# POSE Engine Report: %s

%s
`, title, kind, engineVer, timestamp, title, fullReport)

	if err := writeAtomic(localPath, []byte(localContent), 0o644); err != nil {
		fmt.Fprintf(stderr, text("pose report-limitation: writing local feedback: %v\n", "pose report-limitation: escrevendo feedback local: %v\n"), err)
		return 1
	}

	fmt.Fprintf(stdout, text("[INFO] Limitation/Feedback recorded locally at: %s\n", "[INFO] Limitação/Feedback registrado localmente em: %s\n"), localPath)

	// Submit remotely via gh CLI or GITHUB_TOKEN if requested or gh is available
	if submitRemote {
		label := feedbackIssueLabel(kind)

		fmt.Fprintln(stdout, text("[INFO] Submitting report upstream to oseiaspereira88/pose on GitHub...", "[INFO] Submetendo relato ao repositório upstream oseiaspereira88/pose no GitHub..."))

		cmd := exec.Command("gh", "issue", "create",
			"--repo", "oseiaspereira88/pose",
			"--title", fmt.Sprintf("[%s] %s", strings.ToUpper(kind), title),
			"--body", fullReport,
			"--label", label,
		)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(stderr, text("[WARN] Could not submit issue automatically via gh CLI: %v\n", "[WARN] Não foi possível submeter issue automaticamente via gh CLI: %v\n"), err)
			fmt.Fprintln(stdout, text("You can copy the local feedback file and post it manually at https://github.com/oseiaspereira88/pose/issues", "Você pode copiar o arquivo local de feedback e publicar manualmente em https://github.com/oseiaspereira88/pose/issues"))
		} else {
			fmt.Fprintln(stdout, text("Result: SUCCESS — Report published to community tracker!", "Resultado: SUCESSO — Relato publicado no rastreador da comunidade!"))
		}
	} else {
		fmt.Fprintln(stdout, text("To submit this upstream to the POSE core repository, run with --submit or post at https://github.com/oseiaspereira88/pose/issues", "Para submeter ao repositório central do POSE, rode com --submit ou publique em https://github.com/oseiaspereira88/pose/issues"))
	}

	return 0
}

// feedbackIssueLabel intentionally uses only labels installed by GitHub's
// standard repository contract. Custom labels previously made --submit fail
// before the issue could be created when upstream lacked feature-proposal or
// engine-limitation.
func feedbackIssueLabel(kind string) string {
	if kind == "bug" {
		return "bug"
	}
	return "enhancement"
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	res := strings.Trim(b.String(), "-")
	if len(res) > 40 {
		res = res[:40]
	}
	if res == "" {
		return "report"
	}
	return res
}

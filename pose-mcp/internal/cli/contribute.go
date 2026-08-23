package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/scaffold"
)

const (
	contributorModeMarker = "<!-- pose:contributor-mode -->"
	contributorStateFile  = ".pose/state/contributor.json"
	contributionsDir      = ".pose/contributions"
)

type contributorState struct {
	Active     bool   `json:"active"`
	EnabledAt  string `json:"enabled_at,omitempty"`
	DisabledAt string `json:"disabled_at,omitempty"`
	Upstream   string `json:"upstream"`
}

type stagedContribution struct {
	Path          string `json:"path"`
	Filename      string `json:"filename"`
	Title         string `json:"title"`
	Type          string `json:"type"`
	CreatedAt     string `json:"created_at"`
	Status        string `json:"status"`
	SubmittedAt   string `json:"submitted_at,omitempty"`
	UpstreamIssue string `json:"upstream_issue,omitempty"`
	DismissReason string `json:"dismiss_reason,omitempty"`
}

func loadContributorState(root string) (*contributorState, error) {
	path := filepath.Join(root, filepath.FromSlash(contributorStateFile))
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &contributorState{Active: false, Upstream: "https://github.com/oseiaspereira88/pose"}, nil
		}
		return nil, err
	}
	var s contributorState
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	if s.Upstream == "" {
		s.Upstream = "https://github.com/oseiaspereira88/pose"
	}
	return &s, nil
}

func writeContributorState(root string, s *contributorState) error {
	path := filepath.Join(root, filepath.FromSlash(contributorStateFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func cmdContribute(root string, args []string, stdout, stderr io.Writer) int {
	commandLocale := cliLocaleValue()
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }

	if len(args) == 0 {
		return usageError(stderr, text(
			"Usage: pose contribute <enable|disable|status|stage|list> [--target <dir>] [--json]",
			"Uso: pose contribute <enable|disable|status|stage|list> [--target <dir>] [--json]",
		))
	}

	target := root
	subcmd := args[0]
	flags := args[1:]

	for i, a := range flags {
		if a == "--target" && i+1 < len(flags) {
			target = flags[i+1]
		}
	}

	switch subcmd {
	case "enable":
		return cmdContributeEnable(target, flags, stdout, stderr, commandLocale)
	case "disable":
		return cmdContributeDisable(target, flags, stdout, stderr, commandLocale)
	case "status":
		return cmdContributeStatus(target, flags, stdout, stderr, commandLocale)
	case "stage":
		return cmdContributeStage(target, flags, stdout, stderr, commandLocale)
	case "list":
		return cmdContributeList(target, flags, stdout, stderr, commandLocale)
	case "submit":
		return cmdContributeSubmit(target, flags, stdout, stderr, commandLocale)
	case "mark-submitted":
		return cmdContributeMarkSubmitted(target, flags, stdout, stderr, commandLocale)
	case "dismiss":
		return cmdContributeDismiss(target, flags, stdout, stderr, commandLocale)
	default:
		fmt.Fprintf(stderr, text("pose contribute: unknown subcommand: %s\n", "pose contribute: subcomando desconhecido: %s\n"), subcmd)
		return 2
	}
}

func cmdContributeEnable(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	state, err := loadContributorState(target)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute enable: %v\n", err)
		return 1
	}

	state.Active = true
	state.EnabledAt = time.Now().UTC().Format(time.RFC3339)
	state.DisabledAt = ""

	if err := writeContributorState(target, state); err != nil {
		fmt.Fprintf(stderr, "pose contribute enable: writing state: %v\n", err)
		return 1
	}

	contribDir := filepath.Join(target, filepath.FromSlash(contributionsDir))
	if err := os.MkdirAll(contribDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "pose contribute enable: creating contributions dir: %v\n", err)
		return 1
	}

	// Update AGENTS.md and POSE.md
	loc := machineryLocale(scaffold.Dist(), target, "en", false)
	if err := injectContributorDocs(target, loc); err != nil {
		fmt.Fprintf(stderr, "pose contribute enable: updating docs: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, text(
		"POSE Contributor Mode is now ENABLED.\nExecuting agents will proactively stage sanitized feedback artifacts under .pose/contributions/.",
		"Modo Contribuidor POSE agora está ATIVO.\nAgentes em execução registrarão artefatos sanitizados de feedback sob .pose/contributions/.",
	))
	return 0
}

func cmdContributeDisable(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	state, err := loadContributorState(target)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute disable: %v\n", err)
		return 1
	}

	state.Active = false
	state.DisabledAt = time.Now().UTC().Format(time.RFC3339)

	if err := writeContributorState(target, state); err != nil {
		fmt.Fprintf(stderr, "pose contribute disable: writing state: %v\n", err)
		return 1
	}

	if err := removeContributorDocs(target); err != nil {
		fmt.Fprintf(stderr, "pose contribute disable: updating docs: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, text(
		"POSE Contributor Mode is now DISABLED.",
		"Modo Contribuidor POSE agora está DESATIVADO.",
	))
	return 0
}

func cmdContributeStatus(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	jsonOut := hasFlag(flags, "--json")
	state, err := loadContributorState(target)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute status: %v\n", err)
		return 1
	}

	staged, _ := listStagedContributions(target)

	if jsonOut {
		type statusJSON struct {
			Active       bool                  `json:"active"`
			EnabledAt    string                `json:"enabled_at,omitempty"`
			Upstream     string                `json:"upstream"`
			StagedCount  int                   `json:"staged_count"`
			StagedFiles  []stagedContribution `json:"staged_files"`
			PrivacyRule  string                `json:"privacy_rule"`
		}
		res := statusJSON{
			Active:      state.Active,
			EnabledAt:   state.EnabledAt,
			Upstream:    state.Upstream,
			StagedCount: len(staged),
			StagedFiles: staged,
			PrivacyRule: "Strict synthetic isolation. No proprietary code, internal hostnames, or credentials allowed.",
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return boolToExit(enc.Encode(res) == nil)
	}

	if state.Active {
		fmt.Fprintln(stdout, text("Status: ACTIVE", "Status: ATIVO"))
		if state.EnabledAt != "" {
			fmt.Fprintf(stdout, text("Enabled at: %s\n", "Ativado em: %s\n"), state.EnabledAt)
		}
	} else {
		fmt.Fprintln(stdout, text("Status: INACTIVE (disabled)", "Status: INATIVO (desativado)"))
	}
	fmt.Fprintf(stdout, text("Upstream repository: %s\n", "Repositório upstream: %s\n"), state.Upstream)
	fmt.Fprintf(stdout, text("Staged contributions: %d under %s/\n", "Contribuições em rascunho: %d sob %s/\n"), len(staged), contributionsDir)
	fmt.Fprintln(stdout, text(
		"Privacy rule: Sanitized synthetic reproductions only. Zero private code transmission.",
		"Regra de privacidade: Apenas reproduções sintéticas sanitizadas. Zero envio de código privado.",
	))
	return 0
}

func cmdContributeStage(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	var title, ctype, body, module string
	ctype = "enhancement"

	for i := 0; i < len(flags); i++ {
		a := flags[i]
		if a == "--title" && i+1 < len(flags) {
			title = flags[i+1]
			i++
		} else if a == "--type" && i+1 < len(flags) {
			ctype = flags[i+1]
			i++
		} else if a == "--body" && i+1 < len(flags) {
			body = flags[i+1]
			i++
		} else if a == "--module" && i+1 < len(flags) {
			module = flags[i+1]
			i++
		}
	}

	if title == "" {
		fmt.Fprintln(stderr, text("pose contribute stage: --title is required", "pose contribute stage: --title é obrigatório"))
		return 2
	}

	slug := slugify(title)
	if slug == "" {
		slug = "contribution"
	}
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.md", ts, slug)
	contribDir := filepath.Join(target, filepath.FromSlash(contributionsDir))
	_ = os.MkdirAll(contribDir, 0o755)

	filePath := filepath.Join(contribDir, filename)

	if body == "" {
		body = text(
			"## Problem / Observation\n\nDescribe the observed friction, limitation, or engine behavior here.\n\n## Reproduction (Synthetic)\n\nProvide minimal synthetic reproduction steps without proprietary code.\n\n## Proposed Solution\n\nDescribe the proposed improvement or fix.",
			"## Problema / Observação\n\nDescreva aqui o atrito, limitação ou comportamento observado no motor.\n\n## Reprodução (Sintética)\n\nForneça passos de reprodução sintética mínima sem código proprietário.\n\n## Solução Proposta\n\nDescreva a melhoria ou correção proposta.",
		)
	}

	content := fmt.Sprintf(`---
title: %q
type: %q
module: %q
created_at: %q
status: staged
upstream: "https://github.com/oseiaspereira88/pose"
privacy: sanitized-synthetic
---

# Contribution Draft: %s

%s
`, title, ctype, module, time.Now().UTC().Format(time.RFC3339), title, body)

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "pose contribute stage: writing file: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, text("Staged contribution recorded: %s\n", "Rascunho de contribuição registrado: %s\n"), filePath)
	return 0
}

func cmdContributeList(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	jsonOut := hasFlag(flags, "--json")
	statusFilter := "all"
	for i := 0; i < len(flags); i++ {
		if flags[i] == "--status" && i+1 < len(flags) {
			statusFilter = strings.ToLower(flags[i+1])
			i++
		}
	}
	items, err := listStagedContributions(target)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute list: %v\n", err)
		return 1
	}

	var filtered []stagedContribution
	for _, it := range items {
		st := it.Status
		if st == "" {
			st = "staged"
		}
		if statusFilter == "all" || strings.EqualFold(st, statusFilter) {
			filtered = append(filtered, it)
		}
	}

	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return boolToExit(enc.Encode(filtered) == nil)
	}

	if len(filtered) == 0 {
		fmt.Fprintln(stdout, text("No contributions found matching criteria in .pose/contributions/", "Nenhuma contribuição encontrada com os critérios em .pose/contributions/"))
		return 0
	}

	fmt.Fprintf(stdout, text("Found %d contribution(s):\n", "Encontrada(s) %d contribuição(ões):\n"), len(filtered))
	for _, it := range filtered {
		st := it.Status
		if st == "" {
			st = "staged"
		}
		extra := ""
		if it.UpstreamIssue != "" {
			extra = " (" + it.UpstreamIssue + ")"
		} else if it.DismissReason != "" {
			extra = " [reason: " + it.DismissReason + "]"
		}
		fmt.Fprintf(stdout, "  - [%s] [%s] %s (%s)%s -> %s\n", st, it.Type, it.Title, it.CreatedAt, extra, it.Filename)
	}
	return 0
}

func cmdContributeMarkSubmitted(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	if len(flags) == 0 {
		fmt.Fprintln(stderr, text("Usage: pose contribute mark-submitted <filename|slug> [--issue-url <url>]", "Uso: pose contribute mark-submitted <filename|slug> [--issue-url <url>]"))
		return 2
	}
	query := flags[0]
	var issueURL string
	for i := 1; i < len(flags); i++ {
		if (flags[i] == "--issue-url" || flags[i] == "--url") && i+1 < len(flags) {
			issueURL = flags[i+1]
			i++
		}
	}
	filePath, err := findContributionFile(target, query)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute mark-submitted: %v\n", err)
		return 1
	}
	if err := updateContributionStatus(filePath, "submitted", issueURL, ""); err != nil {
		fmt.Fprintf(stderr, "pose contribute mark-submitted: updating file: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, text("Contribution marked as submitted: %s\n", "Contribuição marcada como submetida: %s\n"), filepath.Base(filePath))
	return 0
}

func cmdContributeDismiss(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	if len(flags) == 0 {
		fmt.Fprintln(stderr, text("Usage: pose contribute dismiss <filename|slug> [--reason <text>]", "Uso: pose contribute dismiss <filename|slug> [--reason <texto>]"))
		return 2
	}
	query := flags[0]
	var reason string
	for i := 1; i < len(flags); i++ {
		if flags[i] == "--reason" && i+1 < len(flags) {
			reason = flags[i+1]
			i++
		}
	}
	filePath, err := findContributionFile(target, query)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute dismiss: %v\n", err)
		return 1
	}
	if err := updateContributionStatus(filePath, "dismissed", "", reason); err != nil {
		fmt.Fprintf(stderr, "pose contribute dismiss: updating file: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, text("Contribution marked as dismissed: %s\n", "Contribuição marcada como descartada: %s\n"), filepath.Base(filePath))
	return 0
}

func cmdContributeSubmit(target string, flags []string, stdout, stderr io.Writer, commandLocale cliLocale) int {
	text := func(en, pt string) string { return cliText(commandLocale, en, pt) }
	if len(flags) == 0 {
		fmt.Fprintln(stderr, text("Usage: pose contribute submit <filename|slug>", "Uso: pose contribute submit <filename|slug>"))
		return 2
	}
	query := flags[0]
	filePath, err := findContributionFile(target, query)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute submit: %v\n", err)
		return 1
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "pose contribute submit: reading file: %v\n", err)
		return 1
	}
	c := parseStagedContribution(string(raw), filepath.Base(filePath), filePath)
	_, body := posepkg.SplitFrontmatter(string(raw))

	fmt.Fprintln(stdout, text("[INFO] Submitting contribution upstream to oseiaspereira88/pose on GitHub...", "[INFO] Submetendo contribuição ao repositório upstream oseiaspereira88/pose no GitHub..."))

	label := feedbackIssueLabel(c.Type)
	var outBuf bytes.Buffer
	cmd := exec.Command("gh", "issue", "create",
		"--repo", "oseiaspereira88/pose",
		"--title", fmt.Sprintf("[%s] %s", strings.ToUpper(c.Type), c.Title),
		"--body", strings.TrimSpace(body),
		"--label", label,
	)
	cmd.Stdout = io.MultiWriter(stdout, &outBuf)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, text("[WARN] Could not submit issue automatically via gh CLI: %v\n", "[WARN] Não foi possível submeter issue automaticamente via gh CLI: %v\n"), err)
		return 1
	}
	issueURL := strings.TrimSpace(outBuf.String())
	_ = updateContributionStatus(filePath, "submitted", issueURL, "")
	fmt.Fprintln(stdout, text("Result: SUCCESS — Contribution submitted upstream and status updated to 'submitted'!", "Resultado: SUCESSO — Contribuição submetida e status atualizado para 'submitted'!"))
	return 0
}

func findContributionFile(root, query string) (string, error) {
	contribDir := filepath.Join(root, filepath.FromSlash(contributionsDir))
	if _, err := os.Stat(query); err == nil {
		return query, nil
	}
	exact := filepath.Join(contribDir, query)
	if _, err := os.Stat(exact); err == nil {
		return exact, nil
	}
	if !strings.HasSuffix(query, ".md") {
		if _, err := os.Stat(exact + ".md"); err == nil {
			return exact + ".md", nil
		}
	}
	entries, err := os.ReadDir(contribDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), query) {
			return filepath.Join(contribDir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("contribution file not found for query %q under %s/", query, contributionsDir)
}

func updateContributionStatus(filePath, status, issueURL, reason string) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(raw)
	lines := strings.Split(content, "\n")
	var newLines []string
	inFm := false
	statusSet := false
	submittedAtSet := false
	upstreamIssueSet := false
	now := time.Now().UTC().Format(time.RFC3339)

	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "---" {
			if !inFm {
				inFm = true
				newLines = append(newLines, l)
				continue
			}
			// exiting frontmatter
			if !statusSet {
				newLines = append(newLines, "status: "+status)
			}
			if status == "submitted" && !submittedAtSet {
				newLines = append(newLines, fmt.Sprintf("submitted_at: %q", now))
			}
			if issueURL != "" && !upstreamIssueSet {
				newLines = append(newLines, fmt.Sprintf("upstream_issue: %q", issueURL))
			}
			if reason != "" {
				newLines = append(newLines, fmt.Sprintf("dismiss_reason: %q", reason))
			}
			inFm = false
			newLines = append(newLines, l)
			newLines = append(newLines, lines[i+1:]...)
			break
		}
		if inFm {
			if strings.HasPrefix(trimmed, "status:") {
				newLines = append(newLines, "status: "+status)
				statusSet = true
				continue
			}
			if strings.HasPrefix(trimmed, "submitted_at:") {
				if status == "submitted" {
					newLines = append(newLines, fmt.Sprintf("submitted_at: %q", now))
					submittedAtSet = true
				}
				continue
			}
			if strings.HasPrefix(trimmed, "upstream_issue:") {
				if issueURL != "" {
					newLines = append(newLines, fmt.Sprintf("upstream_issue: %q", issueURL))
					upstreamIssueSet = true
				}
				continue
			}
		}
		newLines = append(newLines, l)
	}
	return os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0o644)
}

func listStagedContributions(root string) ([]stagedContribution, error) {
	contribDir := filepath.Join(root, filepath.FromSlash(contributionsDir))
	entries, err := os.ReadDir(contribDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []stagedContribution
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		full := filepath.Join(contribDir, e.Name())
		raw, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		c := parseStagedContribution(string(raw), e.Name(), full)
		items = append(items, c)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func parseStagedContribution(content, filename, path string) stagedContribution {
	c := stagedContribution{
		Filename: filename,
		Path:     path,
		Status:   "staged",
		Type:     "enhancement",
		Title:    filename,
	}
	lines := strings.Split(content, "\n")
	inFm := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "---" {
			if !inFm {
				inFm = true
				continue
			}
			break
		}
		if inFm {
			if strings.HasPrefix(trimmed, "title:") {
				c.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "title:")), `"`)
			} else if strings.HasPrefix(trimmed, "type:") {
				c.Type = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "type:")), `"`)
			} else if strings.HasPrefix(trimmed, "created_at:") {
				c.CreatedAt = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "created_at:")), `"`)
			} else if strings.HasPrefix(trimmed, "status:") {
				c.Status = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "status:")), `"`)
			} else if strings.HasPrefix(trimmed, "submitted_at:") {
				c.SubmittedAt = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "submitted_at:")), `"`)
			} else if strings.HasPrefix(trimmed, "upstream_issue:") {
				c.UpstreamIssue = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "upstream_issue:")), `"`)
			} else if strings.HasPrefix(trimmed, "dismiss_reason:") {
				c.DismissReason = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "dismiss_reason:")), `"`)
			}
		}
	}
	return c
}

func injectContributorDocs(target, loc string) error {
	agentsPath := filepath.Join(target, "AGENTS.md")
	posePath := filepath.Join(target, "POSE.md")

	enAgentsSection := `## Open-Source POSE Contributor Mode

<!-- pose:contributor-mode -->

**Contributor Mode is ACTIVE.** When executing tasks, if you encounter POSE engine defects, tool frictions, missing stack rules, diagnostic false-positives, or clear improvement opportunities:

1. **Continuous Opportunity Scouting**: On every issue resolution or task execution, proactively scout for related improvements, edge cases, root-level parity gaps, or complementary corrections across adjacent tools and commands.
2. **User Consent & Explicit Adjudication**: The decision to create/stage feedback or submit contributions upstream MUST ALWAYS be made by the user. When observing a friction or opportunity, the agent must inform the user and ask for confirmation before staging a draft under ` + "`.pose/contributions/<timestamp>-<slug>.md`" + ` or submitting an upstream GitHub issue.
3. **Strict Privacy Invariant**: NEVER include proprietary business logic, internal hostnames/domains, customer data, API keys, credentials, or private source code in staged contributions. All examples must use generic, synthetic reproductions.
4. **Lifecycle & Status Integrity**: Contribution and feedback artifacts maintain an explicit status lifecycle (` + "`staged`" + ` -> ` + "`submitted`" + ` / ` + "`dismissed`" + `) with tracking metadata (` + "`submitted_at`" + `, ` + "`upstream_issue`" + `). Always check artifact status so subsequent sessions never mix up submitted vs pending drafts or re-submit already handled items.`

	ptAgentsSection := `## Modo Contribuidor Open-Source do POSE

<!-- pose:contributor-mode -->

**Modo Contribuidor está ATIVO.** Ao executar tarefas, se você encontrar defeitos no motor POSE, atritos de ferramentas, regras de stack ausentes, falsos positivos de diagnóstico ou oportunidades de melhoria:

1. **Prospecção Contínua de Oportunidades**: A cada resolução de issue ou execução de tarefa, faça prospecção proativa de melhorias relacionadas, casos de borda, lacunas de paridade na raiz ou correções complementares em ferramentas e comandos adjacentes.
2. **Decisão do Usuário e Adjudicação Interativa**: A decisão de registrar rascunhos de feedback ou submeter contribuições upstream DEVE SEMPRE ser do usuário. Ao observar um atrito ou oportunidade, o agente deve informar o usuário e solicitar confirmação antes de registrar rascunhos sob ` + "`.pose/contributions/<timestamp>-<slug>.md`" + ` ou abrir issues upstream no GitHub.
3. **Invariante Rígida de Privacidade**: NUNCA inclua regras de negócio proprietárias, hostnames/domínios internos, dados de clientes, chaves de API, credenciais ou código-fonte privado nas contribuições. Todos os exemplos devem utilizar código sintético e genérico.
4. **Integridade de Ciclo de Vida e Status**: Artefatos de contribuição e feedback mantêm um ciclo de vida explícito de status (` + "`staged`" + ` -> ` + "`submitted`" + ` / ` + "`dismissed`" + `) com metadados de rastreio (` + "`submitted_at`" + `, ` + "`upstream_issue`" + `). Consulte sempre o status dos artefatos para evitar reenvio ou confusão entre itens pendentes e submetidos.`

	enPoseSection := `## Open-Source POSE Contributor Mode

<!-- pose:contributor-mode -->

This repository has enabled POSE Contributor Mode. Executing AI agents will automatically stage feedback, bug reports, and stack extension proposals under ` + "`.pose/contributions/`" + ` when encountering workflow friction.

- Check staged contributions: ` + "`pose contribute list`" + `
- Check contributor status: ` + "`pose contribute status`" + `
- Stage a contribution manually: ` + "`pose contribute stage --type <bug|enhancement> --title <title>`" + `
- Disable contributor mode: ` + "`pose contribute disable`" + `
- Privacy guarantee: staged reports strictly isolate POSE engine mechanics and never leak proprietary code.`

	ptPoseSection := `## Modo Contribuidor Open-Source do POSE

<!-- pose:contributor-mode -->

Este repositório habilitou o Modo Contribuidor POSE. Agentes de IA em execução registrarão automaticamente relatórios de feedback, bugs e propostas de extensões sob ` + "`.pose/contributions/`" + ` ao encontrar atritos de execução.

- Ver contribuições em rascunho: ` + "`pose contribute list`" + `
- Ver status de contribuidor: ` + "`pose contribute status`" + `
- Registrar contribuição manualmente: ` + "`pose contribute stage --type <bug|enhancement> --title <título>`" + `
- Desativar modo contribuidor: ` + "`pose contribute disable`" + `
- Garantia de privacidade: relatórios em rascunho isolam a mecânica do POSE e nunca vazam código proprietário.`

	agentsSec := enAgentsSection
	poseSec := enPoseSection
	if loc == "pt-BR" {
		agentsSec = ptAgentsSection
		poseSec = ptPoseSection
	}

	if err := upsertDocSection(agentsPath, agentsSec); err != nil {
		return err
	}
	return upsertDocSection(posePath, poseSec)
}

func removeContributorDocs(target string) error {
	agentsPath := filepath.Join(target, "AGENTS.md")
	posePath := filepath.Join(target, "POSE.md")
	_ = removeDocSectionByMarker(agentsPath, contributorModeMarker)
	_ = removeDocSectionByMarker(posePath, contributorModeMarker)
	return nil
}

func upsertDocSection(docPath, sectionText string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return nil // if file does not exist, nothing to update
	}
	content := string(raw)
	// Remove any existing contributor section first
	content = removeDocSectionContent(content, contributorModeMarker)
	content = strings.TrimRight(content, "\n") + "\n\n" + strings.TrimSpace(sectionText) + "\n"
	return os.WriteFile(docPath, []byte(content), 0o644)
}

func removeDocSectionByMarker(docPath, marker string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return nil
	}
	cleaned := removeDocSectionContent(string(raw), marker)
	return os.WriteFile(docPath, []byte(cleaned), 0o644)
}

func removeDocSectionContent(content, marker string) string {
	if !strings.Contains(content, marker) {
		return content
	}
	preamble, sections := splitDocSections(content)
	var keptSections []docSection
	for _, sec := range sections {
		isContrib := false
		for _, line := range sec.Body {
			if strings.Contains(line, marker) {
				isContrib = true
				break
			}
		}
		if !isContrib {
			keptSections = append(keptSections, sec)
		}
	}
	out := append([]string{}, preamble...)
	for _, section := range keptSections {
		out = append(out, section.Heading)
		out = append(out, section.Body...)
	}
	return strings.Join(out, "\n")
}

// IsContributorModeActive checks if contributor mode is enabled in the repository.
func IsContributorModeActive(root string) bool {
	state, err := loadContributorState(root)
	return err == nil && state != nil && state.Active
}

// PrintContributorFailureHint emits an actionable hint when a check/lint/close failure occurs and contributor mode is active.
func PrintContributorFailureHint(root string, w io.Writer, loc cliLocale) {
	if !IsContributorModeActive(root) {
		return
	}
	text := func(en, pt string) string { return cliText(loc, en, pt) }
	fmt.Fprintf(w, "\n%s\n   %s\n",
		text("💡 [Contributor Mode ACTIVE] If this failure stems from a POSE engine defect, false-positive, or missing stack rule, stage a synthetic reproduction locally with:",
			"💡 [Modo Contribuidor ATIVO] Se esta falha for causada por defeito no motor POSE, falso-positivo ou regra de stack ausente, registre uma reprodução sintética local com:"),
		"pose contribute stage --type bug --title \"<brief-summary>\"",
	)
}

// PrintContributorDoctorHint emits an informational contributor mode status in pose doctor.
func PrintContributorDoctorHint(root string, w io.Writer, loc cliLocale) {
	if !IsContributorModeActive(root) {
		return
	}
	text := func(en, pt string) string { return cliText(loc, en, pt) }
	fmt.Fprintln(w, text(
		"[INFO] Contributor Mode: ACTIVE (.pose/contributions/) — Encountered engine or tooling friction? Stage feedback with 'pose contribute stage'.",
		"[INFO] Modo Contribuidor: ATIVO (.pose/contributions/) — Encontrou atrito de ambiente ou motor? Registre feedback com 'pose contribute stage'.",
	))
}


package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var scaffoldSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func scaffoldSlugify(value string) string {
	slug := strings.ToLower(value)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "task"
	}
	return slug
}

// cmdNewSpec is the native parity implementation of pose-new-spec.sh.
func cmdNewSpec(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	if len(args) != 1 || !scaffoldSlug.MatchString(args[0]) {
		fmt.Fprintln(stderr, cliText(locale, "Usage: pose new-spec <feature-slug>", "Uso: pose new-spec <feature-slug>"))
		return 2
	}
	slug := args[0]
	templatePath := filepath.Join(root, ".pose", "templates", "spec.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: template not found: %s\n", "Erro: template ausente: %s\n"), templatePath)
		return 2
	}
	dir := filepath.Join(root, ".pose", "specs", slug)
	if _, err := os.Stat(dir); err == nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: spec already exists: %s\n", "Erro: spec já existe: %s\n"), dir)
		return 1
	}
	content := strings.ReplaceAll(string(template), "<feature-slug>", slug)
	content = strings.ReplaceAll(content, "<YYYY-MM-DD>", time.Now().UTC().Format("2006-01-02"))
	content = strings.ReplaceAll(content, "<created_at>", time.Now().UTC().Format("2006-01-02"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: creating spec: %v\n", "Erro: criar spec: %v\n"), err)
		return 1
	}
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing spec: %v\n", "Erro: escrever spec: %v\n"), err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Spec created: %s (status: draft)\n", "Spec criada: %s (status: draft)\n"), path)
	return 0
}

func cmdNewADR(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		fmt.Fprintln(stderr, cliText(locale, "Usage: pose new-adr <title>", "Uso: pose new-adr <título>"))
		return 2
	}
	slug := scaffoldSlugify(title)
	path := filepath.Join(root, ".pose", "adr", time.Now().Format("2006-01-02")+"-"+slug+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: ADR already exists: %s\n", "Erro: ADR já existe: %s\n"), path)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: creating ADR: %v\n", "Erro: criar ADR: %v\n"), err)
		return 1
	}
	content := fmt.Sprintf("# ADR: %s\n\n## Status\nProposed\n\n## Context\n\n## Decision\n\n## Consequences\n", title)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing ADR: %v\n", "Erro: escrever ADR: %v\n"), err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "ADR created: %s\n", "ADR criada: %s\n"), path)
	return 0
}

func cmdNewKnowledge(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	owner, sensitivity, ttl := "@pose-maintainers", "public-internal", 30
	positionals := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--owner":
			if i+1 >= len(args) || args[i+1] == "" {
				fmt.Fprintln(stderr, cliText(locale, "Error: --owner requires a value.", "Erro: --owner exige um valor."))
				return 2
			}
			i++
			owner = args[i]
		case "--ttl-days":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --ttl-days requires an integer greater than zero.", "Erro: --ttl-days exige inteiro > 0."))
				return 2
			}
			i++
			if _, err := fmt.Sscanf(args[i], "%d", &ttl); err != nil || ttl < 1 || ttl > 90 {
				fmt.Fprintln(stderr, cliText(locale, "Error: --ttl-days is outside the allowed range (1..90).", "Erro: --ttl-days fora do intervalo permitido (1..90)."))
				return 2
			}
		case "--restricted":
			sensitivity = "restricted"
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(stderr, cliText(locale, "Error: unknown option: %s\n", "Erro: opção desconhecida: %s\n"), args[i])
				return 2
			}
			positionals = append(positionals, args[i])
		}
	}
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, cliText(locale, "Usage: pose new-knowledge <type> <slug> [--owner @owner] [--ttl-days N] [--restricted]", "Uso: pose new-knowledge <type> <slug> [--owner @owner] [--ttl-days N] [--restricted]"))
		return 2
	}
	kind, slug := positionals[0], scaffoldSlugify(positionals[1])
	if kind != "handoff" && kind != "note" && kind != "decision-log" {
		fmt.Fprintln(stderr, cliText(locale, "Error: invalid <type>: use handoff|note|decision-log.", "Erro: <type> inválido: use handoff|note|decision-log."))
		return 2
	}
	templatePath := filepath.Join(root, ".pose", "templates", "knowledge.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: template not found: %s\n", "Erro: template ausente: %s\n"), templatePath)
		return 2
	}
	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	path := filepath.Join(root, ".pose", "knowledge", date+"-"+kind+"-"+slug+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: artifact already exists: %s\n", "Erro: artefato já existe: %s\n"), path)
		return 1
	}
	replacements := map[string]string{"<type>": kind, "<slug>": slug, "<owner>": owner, "<sensitivity>": sensitivity, "<created_at>": date, "<last_reviewed_at>": date, "<expires_at>": now.AddDate(0, 0, ttl).Format("2006-01-02")}
	content := string(template)
	for from, to := range replacements {
		content = strings.ReplaceAll(content, from, to)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: creating knowledge artifact: %v\n", "Erro: criar knowledge: %v\n"), err)
		return 1
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing knowledge artifact: %v\n", "Erro: escrever knowledge: %v\n"), err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Knowledge artifact created: %s\n", "Artefato de knowledge criado: %s\n"), path)
	return 0
}

// cmdNewRoadmap is the native parity implementation of pose-new-roadmap.sh.
func cmdNewRoadmap(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	if len(args) != 1 || !scaffoldSlug.MatchString(args[0]) {
		fmt.Fprintln(stderr, cliText(locale, "Usage: pose new-roadmap <roadmap-slug>", "Uso: pose new-roadmap <roadmap-slug>"))
		return 2
	}
	slug := args[0]
	templatePath := filepath.Join(root, ".pose", "templates", "roadmap.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: template not found: %s\n", "Erro: template ausente: %s\n"), templatePath)
		return 2
	}
	path := filepath.Join(root, ".pose", "roadmaps", slug+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: roadmap already exists: %s\n", "Erro: roadmap já existe: %s\n"), path)
		return 1
	}
	content := strings.ReplaceAll(string(template), "<roadmap-slug>", slug)
	content = strings.ReplaceAll(content, "<YYYY-MM-DD>", time.Now().UTC().Format("2006-01-02"))
	content = strings.ReplaceAll(content, "<created_at>", time.Now().UTC().Format("2006-01-02"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: creating roadmap: %v\n", "Erro: criar roadmap: %v\n"), err)
		return 1
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing roadmap: %v\n", "Erro: escrever roadmap: %v\n"), err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Roadmap created: %s (status: draft)\n", "Roadmap criado: %s (status: draft)\n"), path)
	return 0
}

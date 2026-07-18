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
	if len(args) != 1 || !scaffoldSlug.MatchString(args[0]) {
		fmt.Fprintln(stderr, "Uso: pose new-spec <feature-slug>")
		return 2
	}
	slug := args[0]
	templatePath := filepath.Join(root, ".pose", "templates", "spec.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: template ausente: %s\n", templatePath)
		return 2
	}
	dir := filepath.Join(root, ".pose", "specs", slug)
	if _, err := os.Stat(dir); err == nil {
		fmt.Fprintf(stderr, "Erro: spec já existe: %s\n", dir)
		return 1
	}
	content := strings.ReplaceAll(string(template), "<feature-slug>", slug)
	content = strings.ReplaceAll(content, "<YYYY-MM-DD>", time.Now().UTC().Format("2006-01-02"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(stderr, "Erro: criar spec: %v\n", err)
		return 1
	}
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "Erro: escrever spec: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Spec criada: %s (status: draft)\n", path)
	return 0
}

func cmdNewADR(root string, args []string, stdout, stderr io.Writer) int {
	title := strings.TrimSpace(strings.Join(args, " "))
	if title == "" {
		fmt.Fprintln(stderr, "Uso: pose new-adr <título>")
		return 2
	}
	slug := scaffoldSlugify(title)
	path := filepath.Join(root, ".pose", "adr", time.Now().Format("2006-01-02")+"-"+slug+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "Erro: ADR já existe: %s\n", path)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "Erro: criar ADR: %v\n", err)
		return 1
	}
	content := fmt.Sprintf("# ADR: %s\n\n## Status\nProposed\n\n## Context\n\n## Decision\n\n## Consequences\n", title)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "Erro: escrever ADR: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ADR criada: %s\n", path)
	return 0
}

func cmdNewKnowledge(root string, args []string, stdout, stderr io.Writer) int {
	owner, sensitivity, ttl := "@pose-maintainers", "public-internal", 30
	positionals := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--owner":
			if i+1 >= len(args) || args[i+1] == "" {
				fmt.Fprintln(stderr, "Erro: --owner exige um valor.")
				return 2
			}
			i++
			owner = args[i]
		case "--ttl-days":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Erro: --ttl-days exige inteiro > 0.")
				return 2
			}
			i++
			if _, err := fmt.Sscanf(args[i], "%d", &ttl); err != nil || ttl < 1 || ttl > 90 {
				fmt.Fprintln(stderr, "Erro: --ttl-days fora do intervalo permitido (1..90).")
				return 2
			}
		case "--restricted":
			sensitivity = "restricted"
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(stderr, "Erro: opção desconhecida: %s\n", args[i])
				return 2
			}
			positionals = append(positionals, args[i])
		}
	}
	if len(positionals) != 2 {
		fmt.Fprintln(stderr, "Uso: pose new-knowledge <type> <slug> [--owner @owner] [--ttl-days N] [--restricted]")
		return 2
	}
	kind, slug := positionals[0], scaffoldSlugify(positionals[1])
	if kind != "handoff" && kind != "note" && kind != "decision-log" {
		fmt.Fprintln(stderr, "Erro: <type> inválido: use handoff|note|decision-log.")
		return 2
	}
	templatePath := filepath.Join(root, ".pose", "templates", "knowledge.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: template ausente: %s\n", templatePath)
		return 2
	}
	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	path := filepath.Join(root, ".pose", "knowledge", date+"-"+kind+"-"+slug+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "Erro: artefato já existe: %s\n", path)
		return 1
	}
	replacements := map[string]string{"<type>": kind, "<slug>": slug, "<owner>": owner, "<sensitivity>": sensitivity, "<created_at>": date, "<last_reviewed_at>": date, "<expires_at>": now.AddDate(0, 0, ttl).Format("2006-01-02")}
	content := string(template)
	for from, to := range replacements {
		content = strings.ReplaceAll(content, from, to)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "Erro: criar knowledge: %v\n", err)
		return 1
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "Erro: escrever knowledge: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Artefato de knowledge criado: %s\n", path)
	return 0
}

// cmdNewRoadmap is the native parity implementation of pose-new-roadmap.sh.
func cmdNewRoadmap(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || !scaffoldSlug.MatchString(args[0]) {
		fmt.Fprintln(stderr, "Uso: pose new-roadmap <roadmap-slug>")
		return 2
	}
	slug := args[0]
	templatePath := filepath.Join(root, ".pose", "templates", "roadmap.md")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: template ausente: %s\n", templatePath)
		return 2
	}
	path := filepath.Join(root, ".pose", "roadmaps", slug+".md")
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "Erro: roadmap já existe: %s\n", path)
		return 1
	}
	content := strings.ReplaceAll(string(template), "<roadmap-slug>", slug)
	content = strings.ReplaceAll(content, "<YYYY-MM-DD>", time.Now().UTC().Format("2006-01-02"))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "Erro: criar roadmap: %v\n", err)
		return 1
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "Erro: escrever roadmap: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Roadmap criado: %s (status: draft)\n", path)
	return 0
}

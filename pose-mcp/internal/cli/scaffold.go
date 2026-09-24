package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	"github.com/harne8/pose-mcp/internal/pose"
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
	if len(args) == 0 {
		render(io.Discard, stderr).Usage(cliText(locale, "Usage: pose new-spec <feature-slug> [--folder] [--legacy] [--task <artifact-ref>] [--expect-context <digest>]", "Uso: pose new-spec <feature-slug> [--folder] [--legacy] [--task <artifact-ref>] [--expect-context <digest>]"))
		return 2
	}
	slug := ""
	taskRef := ""
	expectedContext := ""
	isFolder := false
	isLegacy := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--folder", "--dated", "--dir":
			isFolder = true
		case "--legacy":
			isLegacy = true
		case "--flat":
			// flat is default
		case "--task", "--expect-context":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				render(io.Discard, stderr).Failure(cliText(locale, "task/context option requires a value", "a opção de tarefa/contexto exige um valor"))
				return 2
			}
			i++
			if a == "--task" {
				if taskRef != "" {
					render(io.Discard, stderr).Failure("duplicate --task")
					return 2
				}
				taskRef = args[i]
			} else {
				if expectedContext != "" {
					render(io.Discard, stderr).Failure("duplicate --expect-context")
					return 2
				}
				expectedContext = args[i]
			}
		default:
			if strings.HasPrefix(a, "-") {
				render(io.Discard, stderr).Failure("unknown option: " + a)
				return 2
			}
			if slug == "" {
				slug = a
			} else {
				render(io.Discard, stderr).Failure(cliText(locale, "only one spec slug is allowed", "informe apenas um slug de spec"))
				return 2
			}
		}
	}
	if slug == "" || !scaffoldSlug.MatchString(slug) {
		render(io.Discard, stderr).Usage(cliText(locale, "Usage: pose new-spec <feature-slug> [--folder] [--legacy] [--task <artifact-ref>] [--expect-context <digest>]", "Uso: pose new-spec <feature-slug> [--folder] [--legacy] [--task <artifact-ref>] [--expect-context <digest>]"))
		return 2
	}
	if taskRef == "" {
		taskRef = "spec:" + slug
	}
	requestedTask, err := pose.ParseArtifactRef(taskRef)
	if err != nil || requestedTask.Kind != "spec" || requestedTask.Slug != slug {
		render(io.Discard, stderr).Failure(cliText(locale, "--task must identify this spec slug", "--task deve identificar este slug de spec"))
		return 2
	}
	context, resolver, currentProject, err := cliAgentContext(root, taskRef)
	if err != nil {
		render(io.Discard, stderr).Failure("project context unavailable: " + err.Error())
		return 1
	}
	if context.TaskResolution == nil {
		render(io.Discard, stderr).Failure("task context unavailable")
		return 1
	}
	resolution := context.TaskResolution
	if resolution.Resolved {
		canonical := context.Authority.String()
		render(stdout, stderr).Table(cliout.Table{Header: []string{"Canonical spec", "Status"}, Rows: [][]string{{canonical, resolution.Status}}})
		return 0
	}
	if resolution.State != "unknown-spec" {
		render(io.Discard, stderr).Failure("task authority cannot be created: " + resolution.State)
		return 1
	}
	if context.Authority == nil {
		render(io.Discard, stderr).Failure("task authority is unresolved")
		return 1
	}
	store, crossProject, err := cliAuthorizedWriteStore(root, context.Authority.Project, resolver, currentProject)
	if err != nil {
		render(io.Discard, stderr).Failure(err.Error())
		return 1
	}
	if crossProject && expectedContext == "" {
		render(io.Discard, stderr).Failure("cross-project write requires --expect-context from pose context")
		return 1
	}
	if expectedContext != "" && context.ContextRevision != expectedContext {
		render(io.Discard, stderr).Failure("stale-context; rerun pose context and retry")
		return 1
	}
	templatePath := filepath.Join(root, ".pose", "templates", "spec.md")
	if crossProject {
		templatePath = filepath.Join(store.Root, ".pose", "templates", "spec.md")
	}
	template, err := os.ReadFile(templatePath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: template not found: %s\n", "Erro: template ausente: %s\n"), templatePath)
		return 2
	}
	today := time.Now().UTC().Format("2006-01-02")
	content := strings.ReplaceAll(string(template), "<feature-slug>", slug)
	content = strings.ReplaceAll(content, "<YYYY-MM-DD>", today)
	content = strings.ReplaceAll(content, "<created_at>", today)

	var targetPath string
	if isFolder {
		targetPath = filepath.Join(store.Root, ".pose", "specs", today+"-"+slug, "spec.md")
	} else if isLegacy {
		targetPath = filepath.Join(store.Root, ".pose", "specs", slug, "spec.md")
	} else {
		// Default: modern dated flat file (conforming format)
		targetPath = filepath.Join(store.Root, ".pose", "specs", today+"-"+slug+".md")
	}

	if expectedContext != "" {
		latest, _, _, err := cliAgentContext(root, taskRef)
		if err != nil || latest.ContextRevision != expectedContext {
			render(io.Discard, stderr).Failure("stale-context; rerun pose context and retry")
			return 1
		}
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: creating spec dir: %v\n", "Erro: criar diretório de spec: %v\n"), err)
		return 1
	}

	if expectedContext != "" {
		latest, _, _, err := cliAgentContext(root, taskRef)
		if err != nil || latest.ContextRevision != expectedContext {
			render(io.Discard, stderr).Failure("stale-context; rerun pose context and retry")
			return 1
		}
	}
	file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing spec: %v\n", "Erro: escrever spec: %v\n"), err)
		return 1
	}
	if _, err := file.Write([]byte(content)); err != nil {
		_ = file.Close()
		fmt.Fprintf(stderr, cliText(locale, "Error: writing spec: %v\n", "Erro: escrever spec: %v\n"), err)
		return 1
	}
	if err := file.Close(); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: writing spec: %v\n", "Erro: escrever spec: %v\n"), err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Spec created: %s (status: draft)\n", "Spec criada: %s (status: draft)\n"), targetPath)
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

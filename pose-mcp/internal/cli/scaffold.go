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

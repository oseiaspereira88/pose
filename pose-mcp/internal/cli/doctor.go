package cli

// pose doctor — installation and instance diagnostics (spec pose-doctor).
// Read-only: every check reports error/warn/ok with an actionable hint;
// doctor never fixes anything itself.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type doctorFinding struct {
	Check    string `json:"check"`
	Level    string `json:"level"` // ok | warn | error
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "-h", "--help":
			fmt.Fprintln(stdout, "Uso: pose doctor [--json] — diagnóstico read-only da instalação POSE")
			return 0
		default:
			fmt.Fprintf(stderr, "Erro: argumento inválido: %s\n", a)
			return 2
		}
	}

	var findings []doctorFinding
	add := func(check, level, message, hint string) {
		findings = append(findings, doctorFinding{check, level, message, hint})
	}

	// 1. Binary + toolchain deps.
	add("binary", "ok", fmt.Sprintf("pose %s", Version), "")
	for _, dep := range []struct{ name, hint string }{
		{"git", "instale git — o POSE resolve o root do projeto por ele"},
		{"bash", "necessário para os gates delegados ao motor de scripts"},
		{"python3", "necessário para validate/check e scripts do motor"},
	} {
		if _, err := exec.LookPath(dep.name); err != nil {
			add("deps."+dep.name, "error", dep.name+" não encontrado no PATH", dep.hint)
		} else {
			add("deps."+dep.name, "ok", dep.name+" disponível", "")
		}
	}
	if _, err := exec.LookPath("go"); err != nil {
		add("deps.go", "warn", "go não encontrado (opcional: só para rebuild do MCP)", "")
	} else {
		add("deps.go", "ok", "go disponível", "")
	}

	// 2. Instance.
	root, err := projectRoot()
	if err != nil {
		add("instance.root", "error", fmt.Sprintf("root não resolvido: %v", err), "")
		return doctorReport(findings, jsonOut, stdout)
	}
	add("instance.root", "ok", root, "")
	poseDir := filepath.Join(root, ".pose")
	if fi, err := os.Stat(poseDir); err != nil || !fi.IsDir() {
		add("instance.pose-dir", "error", ".pose/ ausente — este repo não tem POSE instalado",
			"rode o install.sh da distribuição POSE")
		return doctorReport(findings, jsonOut, stdout)
	}
	add("instance.pose-dir", "ok", ".pose/ presente", "")

	// 3. Engine + dispatcher.
	if _, err := os.Stat(filepath.Join(poseDir, "scripts", "pose-lib.sh")); err != nil {
		add("engine.scripts", "error", "motor de scripts ausente (.pose/scripts/pose-lib.sh)",
			"re-rode o install.sh — comandos delegados não funcionarão")
	} else {
		add("engine.scripts", "ok", "motor de scripts presente", "")
	}
	if _, err := os.Stat(filepath.Join(root, "pose")); err != nil {
		add("engine.dispatcher", "warn", "dispatcher ./pose ausente na raiz",
			"opcional com o binário unificado, mas o CI/hooks padrão o usam")
	} else {
		add("engine.dispatcher", "ok", "dispatcher ./pose presente", "")
	}

	// 4. Schema version.
	svPath := filepath.Join(poseDir, "schema-version")
	engineVersion := engineSchemaVersion(root)
	if b, err := os.ReadFile(svPath); err != nil {
		add("schema.version", "warn", "instância sem .pose/schema-version",
			"rode 'pose upgrade'")
	} else {
		instance := strings.TrimSpace(string(b))
		n, convErr := strconv.Atoi(instance)
		switch {
		case convErr != nil:
			add("schema.version", "error", fmt.Sprintf("schema-version inválido: %q", instance),
				"rode 'pose upgrade'")
		case engineVersion > 0 && n > engineVersion:
			add("schema.version", "error",
				fmt.Sprintf("instância v%d é mais nova que o motor v%d", n, engineVersion),
				"atualize o motor POSE (não há downgrade)")
		case engineVersion > 0 && n < engineVersion:
			add("schema.version", "warn",
				fmt.Sprintf("instância v%d atrás do motor v%d", n, engineVersion),
				"rode 'pose upgrade'")
		default:
			add("schema.version", "ok", "schema v"+instance, "")
		}
	}

	// 5. Skills symlinks (.claude/skills → .agents/skills).
	claudeSkills := filepath.Join(root, ".claude", "skills")
	if entries, err := os.ReadDir(claudeSkills); err == nil {
		broken := 0
		for _, e := range entries {
			link := filepath.Join(claudeSkills, e.Name())
			if fi, err := os.Lstat(link); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				if _, err := os.Stat(link); err != nil {
					broken++
				}
			}
		}
		if broken > 0 {
			add("skills.symlinks", "error", fmt.Sprintf("%d symlink(s) quebrado(s) em .claude/skills", broken),
				"re-rode o install.sh para recriar os symlinks")
		} else {
			add("skills.symlinks", "ok", "symlinks de skills íntegros", "")
		}
	}

	// 6. MCP wrapper.
	wrapper := filepath.Join(poseDir, "bin", "pose-mcp-claude")
	if b, err := os.ReadFile(wrapper); err != nil {
		add("mcp.wrapper", "warn", "wrapper MCP ausente (.pose/bin/pose-mcp-claude)",
			"opcional; instale com install.sh ou --mcp-binary")
	} else {
		content := string(b)
		if strings.Contains(content, `POSE_PROJECT_ROOT="$(cd`) {
			add("mcp.wrapper", "ok", "wrapper deriva o root dinamicamente", "")
		} else {
			add("mcp.wrapper", "error", "wrapper com POSE_PROJECT_ROOT hardcoded",
				"regenere com o install.sh atual (bug do formato antigo)")
		}
		if _, err := os.Stat(filepath.Join(poseDir, "bin", "pose-mcp")); err != nil {
			add("mcp.binary", "warn", "binário pose-mcp ausente ao lado do wrapper",
				"compile com go build ou passe --mcp-binary no install.sh")
		} else {
			add("mcp.binary", "ok", "binário pose-mcp presente", "")
		}
	}

	// 7. Git hooks.
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Lstat(hook); err != nil {
		add("hooks.pre-commit", "warn", "pre-commit não instalado",
			"rode './pose hooks install' para gate automático no commit")
	} else {
		add("hooks.pre-commit", "ok", "pre-commit instalado", "")
	}

	return doctorReport(findings, jsonOut, stdout)
}

// engineSchemaVersion parses POSE_SCHEMA_VERSION from the installed engine.
func engineSchemaVersion(root string) int {
	b, err := os.ReadFile(filepath.Join(root, ".pose", "scripts", "pose-lib.sh"))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "POSE_SCHEMA_VERSION="); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				return n
			}
		}
	}
	return 0
}

func doctorReport(findings []doctorFinding, jsonOut bool, stdout io.Writer) int {
	errors, warns := 0, 0
	for _, f := range findings {
		switch f.Level {
		case "error":
			errors++
		case "warn":
			warns++
		}
	}
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"findings": findings,
			"errors":   errors,
			"warnings": warns,
		})
	} else {
		for _, f := range findings {
			icon := map[string]string{"ok": "✓", "warn": "!", "error": "✗"}[f.Level]
			fmt.Fprintf(stdout, "[%s] %-18s %s\n", icon, f.Check, f.Message)
			if f.Hint != "" {
				fmt.Fprintf(stdout, "      ↳ %s\n", f.Hint)
			}
		}
		fmt.Fprintf(stdout, "\ndoctor: %d erro(s), %d aviso(s)\n", errors, warns)
	}
	if errors > 0 {
		return 1
	}
	return 0
}

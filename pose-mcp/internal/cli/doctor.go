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
	Check   string `json:"check"`
	Level   string `json:"level"` // ok | warn | error
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	text := func(english, portuguese string) string { return cliText(locale, english, portuguese) }
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "-h", "--help":
			fmt.Fprintln(stdout, text("Usage: pose doctor [--json] — read-only POSE installation diagnostics", "Uso: pose doctor [--json] — diagnóstico read-only da instalação POSE"))
			return 0
		default:
			fmt.Fprintf(stderr, text("Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), a)
			return 2
		}
	}

	var findings []doctorFinding
	add := func(check, level, message, hint string) {
		findings = append(findings, doctorFinding{check, level, message, hint})
	}

	// 1. Binary + toolchain deps.
	add("binary", "ok", fmt.Sprintf("pose %s", Version), "")
	if _, err := exec.LookPath("git"); err != nil {
		add("deps.git", "error", text("git not found in PATH", "git não encontrado no PATH"), text("install git; POSE uses it to resolve the project root", "instale git — o POSE resolve o root do projeto por ele"))
	} else {
		add("deps.git", "ok", text("git available", "git disponível"), "")
	}
	if _, err := exec.LookPath("go"); err != nil {
		add("deps.go", "warn", text("go not found (optional; needed only to rebuild MCP)", "go não encontrado (opcional: só para rebuild do MCP)"), "")
	} else {
		add("deps.go", "ok", text("go available", "go disponível"), "")
	}

	// 2. Instance.
	root, err := projectRoot()
	if err != nil {
		add("instance.root", "error", fmt.Sprintf(text("could not resolve root: %v", "root não resolvido: %v"), err), "")
		return doctorReport(findings, jsonOut, stdout, locale)
	}
	add("instance.root", "ok", root, "")
	poseDir := filepath.Join(root, ".pose")
	if fi, err := os.Stat(poseDir); err != nil || !fi.IsDir() {
		add("instance.pose-dir", "error", text(".pose/ not found — POSE is not installed in this repository", ".pose/ ausente — este repo não tem POSE instalado"),
			text("run the POSE distribution install.sh", "rode o install.sh da distribuição POSE"))
		return doctorReport(findings, jsonOut, stdout, locale)
	}
	add("instance.pose-dir", "ok", text(".pose/ present", ".pose/ presente"), "")

	// 3. Native engine contract.
	add("engine.native", "ok", text("native Go engine active; no script runtime required", "motor Go nativo ativo; runtime de scripts não é necessário"), "")

	// 4. Schema version.
	svPath := filepath.Join(poseDir, "schema-version")
	engineVersion := engineSchemaVersion(root)
	if b, err := os.ReadFile(svPath); err != nil {
		add("schema.version", "warn", text("instance has no .pose/schema-version", "instância sem .pose/schema-version"), text("run 'pose upgrade'", "rode 'pose upgrade'"))
	} else {
		instance := strings.TrimSpace(string(b))
		n, convErr := strconv.Atoi(instance)
		switch {
		case convErr != nil:
			add("schema.version", "error", fmt.Sprintf(text("invalid schema-version: %q", "schema-version inválido: %q"), instance), text("run 'pose upgrade'", "rode 'pose upgrade'"))
		case engineVersion > 0 && n > engineVersion:
			add("schema.version", "error",
				fmt.Sprintf(text("instance v%d is newer than engine v%d", "instância v%d é mais nova que o motor v%d"), n, engineVersion),
				text("update the POSE engine; downgrade is unsupported", "atualize o motor POSE (não há downgrade)"))
		case engineVersion > 0 && n < engineVersion:
			add("schema.version", "warn",
				fmt.Sprintf(text("instance v%d is behind engine v%d", "instância v%d atrás do motor v%d"), n, engineVersion), text("run 'pose upgrade'", "rode 'pose upgrade'"))
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
			add("skills.symlinks", "error", fmt.Sprintf(text("%d broken symlink(s) under .claude/skills", "%d symlink(s) quebrado(s) em .claude/skills"), broken), text("rerun install.sh to recreate symlinks", "re-rode o install.sh para recriar os symlinks"))
		} else {
			add("skills.symlinks", "ok", text("skill symlinks are healthy", "symlinks de skills íntegros"), "")
		}
	}

	// 6. MCP uses the same native binary directly.
	if b, err := os.ReadFile(filepath.Join(root, ".mcp.json")); err != nil {
		add("mcp.config", "warn", text(".mcp.json not found", ".mcp.json ausente"), text("run 'pose install' to seed the native server configuration", "rode 'pose install' para criar a configuração do servidor nativo"))
	} else if strings.Contains(string(b), `"command": "pose"`) {
		add("mcp.config", "ok", text("MCP points to the native pose binary", "MCP aponta para o binário pose nativo"), "")
	} else {
		add("mcp.config", "warn", text("MCP configuration does not use the native pose command", "configuração MCP não usa o comando pose nativo"), text("regenerate .mcp.json with 'pose install'", "regenere .mcp.json com 'pose install'"))
	}

	// 7. Git hooks.
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Lstat(hook); err != nil {
		add("hooks.pre-commit", "warn", text("pre-commit hook not installed", "pre-commit não instalado"), text("run 'pose hooks install' for an automatic commit gate", "rode 'pose hooks install' para gate automático no commit"))
	} else {
		add("hooks.pre-commit", "ok", text("pre-commit hook installed", "pre-commit instalado"), "")
	}

	return doctorReport(findings, jsonOut, stdout, locale)
}

func engineSchemaVersion(root string) int {
	return nativeSchemaVersion
}

func doctorReport(findings []doctorFinding, jsonOut bool, stdout io.Writer, locale cliLocale) int {
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
		fmt.Fprintf(stdout, cliText(locale, "\ndoctor: %d error(s), %d warning(s)\n", "\ndoctor: %d erro(s), %d aviso(s)\n"), errors, warns)
	}
	if errors > 0 {
		return 1
	}
	return 0
}

// Package cli implements the unified `pose` binary (spec
// pose-cli-go-unification, Corte 1 — strangler pattern):
//
//   - native subcommands: version, init, serve-mcp, help
//   - every other known subcommand is DELEGATED to the bash engine at
//     .pose/scripts/pose-<cmd>.sh, preserving args and exit code, so the
//     binary is a drop-in replacement for the `pose` shell dispatcher.
//
// The bash engine remains the source of truth for delegated gates; future
// phases port them natively one by one with parity tests.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version is stamped via -ldflags at release time (pose-release-pipeline).
var Version = "0.9.0-dev"

// delegated maps subcommands to their bash engine scripts.
var delegated = map[string]string{
	"upgrade":                "pose-upgrade.sh",
	"new-spec":               "pose-new-spec.sh",
	"new-roadmap":            "pose-new-roadmap.sh",
	"new-adr":                "pose-new-adr.sh",
	"new-knowledge":          "pose-new-knowledge.sh",
	"check":                  "pose-check.sh",
	"validate":               "pose-validate.sh",
	"index":                  "pose-index.sh",
	"report":                 "pose-report.sh",
	"knowledge-check":        "pose-knowledge-check.sh",
	"knowledge-housekeeping": "pose-knowledge-housekeeping.sh",
	"reports-housekeeping":   "pose-reports-housekeeping.sh",
	"recurrence-check":       "pose-recurrence-check.sh",
	"hooks":                  "pose-hooks.sh",
	"followups":              "pose-followups.sh",
	"suggest":                "pose-suggest.sh",
	"stats":                  "pose-stats.sh",
}

// Main is the entrypoint used by cmd/pose. It returns the process exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleFor(stderr)
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "version", "--version", "-v":
		return cmdVersion(stdout)
	case "help", "-h", "--help":
		return cmdHelp(stdout)
	case "init":
		// --wizard delega ao onboarding assistido do motor de scripts.
		if len(args) > 0 && args[0] == "--wizard" {
			return delegate("pose-init-wizard.sh", args[1:], stdout, stderr)
		}
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose init: %v\n", err)
			return 1
		}
		return cmdInit(root, stdout, stderr)
	case "new-spec":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-spec: %v\n", err)
			return 1
		}
		return cmdNewSpec(root, args, stdout, stderr)
	case "new-roadmap":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-roadmap: %v\n", err)
			return 1
		}
		return cmdNewRoadmap(root, args, stdout, stderr)
	case "new-adr":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-adr: %v\n", err)
			return 1
		}
		return cmdNewADR(root, args, stdout, stderr)
	case "new-knowledge":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-knowledge: %v\n", err)
			return 1
		}
		return cmdNewKnowledge(root, args, stdout, stderr)
	case "followups":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose followups: %v\n", err)
			return 1
		}
		return cmdFollowups(root, args, stdout, stderr)
	case "report":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose report: %v\n", err)
			return 1
		}
		return cmdReport(root, args, stdout, stderr)
	case "validate":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose validate: %v\n", err)
			return 1
		}
		return cmdValidate(root, args, stdout, stderr)
	case "check":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose check: %v\n", err)
			return 1
		}
		return cmdCheck(root, args, stdout, stderr)
	case "install":
		return cmdInstall(args, stdout, stderr)
	case "doctor":
		return cmdDoctor(args, stdout, stderr)
	case "import":
		defer emitTelemetry("import")
		return cmdImport(args, stdout, stderr)
	case "lint-spec":
		defer emitTelemetry("lint-spec")
		return cmdLintSpec(args, stdout, stderr)
	case "history-check":
		defer emitTelemetry("history-check")
		return cmdHistoryCheck(args, stdout, stderr)
	case "telemetry":
		return cmdTelemetry(args, stdout, stderr)
	case "serve-mcp":
		// Blocking; wiring lives in internal/bootstrap (shared with cmd/pose-mcp).
		runServeMCP(args)
		return 0
	}

	defer emitTelemetry(cmd)
	script, known := delegated[cmd]
	if !known {
		fmt.Fprintf(stderr, "%s: %s\n", cliText(locale, "Unknown command", "Comando desconhecido"), cmd)
		fmt.Fprintln(stderr, cliText(locale, "Run 'pose help' to see available commands.", "Execute 'pose help' para ver os comandos disponíveis."))
		return 2
	}
	return delegate(script, args, stdout, stderr)
}

// projectRoot resolves the repository root the same way the bash dispatcher
// does: git toplevel, falling back to the current directory.
func projectRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}

// delegate runs the bash engine script for cmd, streaming stdio and
// propagating the exit code. Scripts are resolved ONLY under
// <root>/.pose/scripts — never via PATH lookup.
func delegate(script string, args []string, stdout, stderr io.Writer) int {
	fmt.Fprintf(stderr, "[WARN] pose: %s\n", cliText(cliLocaleFor(stderr), "delegating to the deprecated script engine", "delegando para o motor de scripts descontinuado"))
	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose: %v\n", err)
		return 1
	}
	path := filepath.Join(root, ".pose", "scripts", script)
	if _, err := os.Stat(path); err != nil {
		locale := cliLocaleFor(stderr)
		fmt.Fprintf(stderr, "pose: %s %s\n", cliText(locale, "script engine not found at", "motor de scripts não encontrado em"), path)
		fmt.Fprintln(stderr, "pose:", cliText(locale, "is this directory a POSE installation? Run the distribution install.sh or 'pose init' in an installed repository.", "este diretório tem uma instalação POSE? Rode o install.sh da distribuição ou 'pose init' num repo já instalado."))
		return 1
	}
	c := exec.Command("bash", append([]string{path}, args...)...)
	c.Dir = root
	c.Stdin = os.Stdin
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintf(stderr, "pose: %v\n", err)
		return 1
	}
	return 0
}

func cmdVersion(w io.Writer) int {
	fmt.Fprintf(w, "pose %s\n", Version)
	if root, err := projectRoot(); err == nil {
		sv := filepath.Join(root, ".pose", "schema-version")
		if b, err := os.ReadFile(sv); err == nil {
			fmt.Fprintf(w, "schema: %s\n", strings.TrimSpace(string(b)))
		} else if _, err := os.Stat(filepath.Join(root, ".pose")); err == nil {
			fmt.Fprintf(w, "schema: unversioned (run 'pose upgrade')\n")
		}
	}
	return 0
}

func cmdHelp(w io.Writer) int {
	fmt.Fprint(w, helpText)
	return 0
}

const helpText = `POSE - Project Operating Standard for Engineering

Uso: pose <comando> [opções]

Nativos (binário):
  version                             Versão do binário + schema da instância
  init                                Garante a estrutura mínima do POSE no repo
  serve-mcp [--stdio]                 Sobe o servidor MCP do POSE (env POSE_*)
  doctor [--json]                     Diagnóstico da instalação/instância
  install <dir> [--locale tag] [...]  Instala o POSE embutido num repo (sem clone)
  import <spec-kit|openspec> <path>   Importa specs externas [--dry-run]
  telemetry <enable|disable|status>   Telemetria anônima OPT-IN (nada é enviado
                                      sem opt-in E POSE_TELEMETRY_URL)

Scaffold:
  new-spec <slug>                     Cria scaffold de spec por feature
  new-roadmap <slug>                  Cria roadmap governado em .pose/roadmaps/
  new-adr "<título>"                  Cria ADR com template padrão
  new-knowledge <type> <slug>         Cria handoff/note/decision-log

Gates determinísticos:
  check | validate | knowledge-check | recurrence-check | lint-spec |
  followups | history-check

Descoberta e métricas:
  suggest | stats

Geração de artefatos:
  index | report

Manutenção:
  upgrade [--dry-run] | knowledge-housekeeping | reports-housekeeping | hooks

Comandos não-nativos são delegados ao motor em .pose/scripts/ (mesma
interface do dispatcher shell). 'pose help' completo por comando: consulte
POSE.md da instância.
`

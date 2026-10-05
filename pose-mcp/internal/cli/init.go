package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// instanceDirs is the native instance contract.
var instanceDirs = []string{
	".pose/workflows",
	".pose/templates",
	".pose/rules",
	".pose/specs",
	".pose/adr",
	".pose/indexes",
	".pose/reports",
	".pose/reports/history",
	".pose/knowledge",
	".pose/roadmaps",
	".pose/changelogs/unreleased",
	".pose/feedback",
	".pose/assessments",
	".pose/state",
	".agents/skills",
}

// cmdInit creates the minimal POSE directory structure, idempotently —
// native parity of pose-init.sh.
func cmdInit(root string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	created := 0
	for _, rel := range instanceDirs {
		dir := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(dir); err == nil {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(stderr, "[%s] %s %s: %v\n", cliText(locale, "ERROR", "ERRO"), cliText(locale, "creating", "criando"), rel, err)
			return 1
		}
		fmt.Fprintf(stdout, cliText(locale, "[OK] created: %s\n", "[OK] criado: %s\n"), rel)
		created++
	}
	if created == 0 {
		fmt.Fprintf(stdout, "[INFO] %s\n", cliText(locale, "POSE structure already present. Run: pose check", "estrutura POSE já presente. Execute: pose check"))
	} else {
		fmt.Fprintf(stdout, cliText(locale, "[INFO] %d directory(ies) created. Run: pose check\n", "[INFO] %d diretório(s) criado(s). Execute: pose check\n"), created)
	}
	return 0
}

// cmdInitCommand is `pose init` (spec pose-init-is-install). Where no POSE
// instance exists it runs the full installation, because an instance made of
// empty directories is one `pose check` rejects and `pose new-spec` cannot use;
// on an installed instance it only ensures the structure and says how to
// refresh it. The installer's flags pass through.
func cmdInitCommand(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	const usage = "Usage: pose init [--wizard [--yes]] [--locale <tag>] [--project-name <name>] [--project-id <id>] [--skip-mcp] [--allow-non-git]"
	wizard, yes := false, false
	installArgs := []string{root}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--wizard":
			wizard = true
		case "--yes":
			yes = true
		case "--skip-mcp", "--allow-non-git":
			installArgs = append(installArgs, args[i])
		case "--locale", "--project-name", "--project-id":
			if i+1 >= len(args) {
				return usageError(stderr, usage)
			}
			installArgs = append(installArgs, args[i], args[i+1])
			i++
		default:
			return usageError(stderr, usage)
		}
	}
	if yes && !wizard {
		return usageError(stderr, usage)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "schema-version")); err != nil {
		if code := cmdInstall(installArgs, stdout, stderr); code != 0 {
			return code
		}
	} else {
		if code := cmdInit(root, io.Discard, stderr); code != 0 {
			return code
		}
		render(stdout, stderr).ContractLine(cliText(locale,
			"[INFO] POSE is already installed here; `pose update` refreshes the machinery and nothing else was written.",
			"[INFO] o POSE já está instalado aqui; `pose update` atualiza o maquinário e nada mais foi gravado."))
	}
	if wizard {
		wizardArgs := []string{}
		if yes {
			wizardArgs = append(wizardArgs, "--yes")
		}
		return cmdInitWizard(root, wizardArgs, stdout, stderr)
	}
	return 0
}

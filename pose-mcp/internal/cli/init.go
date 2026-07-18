package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// instanceDirs is the native mirror of .pose/scripts/pose-init.sh
// (REQUIRED_DIRS). The parity test in init_test.go fails if the two lists
// drift, keeping the bash engine authoritative during the strangler phase.
var instanceDirs = []string{
	".pose/workflows",
	".pose/templates",
	".pose/rules",
	".pose/scripts",
	".pose/specs",
	".pose/adr",
	".pose/indexes",
	".pose/reports",
	".pose/reports/history",
	".pose/knowledge",
	".pose/roadmaps",
	".pose/changelogs/unreleased",
	".agents/skills",
}

// cmdInit creates the minimal POSE directory structure, idempotently —
// native parity of pose-init.sh.
func cmdInit(root string, stdout, stderr io.Writer) int {
	locale := cliLocaleFor(stderr)
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
		fmt.Fprintf(stdout, "[OK] criado: %s\n", rel)
		created++
	}
	if created == 0 {
		fmt.Fprintf(stdout, "[INFO] %s\n", cliText(locale, "POSE structure already present. Run: pose check", "estrutura POSE já presente. Execute: pose check"))
	} else {
		fmt.Fprintf(stdout, "[INFO] %d diretório(s) criado(s). Execute: pose check\n", created)
	}
	return 0
}

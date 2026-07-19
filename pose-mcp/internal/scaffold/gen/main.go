// gen syncs the POSE distribution (pose-dist/) into internal/scaffold/dist
// so it can be embedded in the unified binary (spec pose-cli-embed-standalone).
//
// go:embed cannot reference files outside the module nor symlinks, so this
// generator materializes a copy: everything except .claude/ (symlinks —
// recreated programmatically at install time) and binaries. The drift test in
// scaffold_test.go fails whenever pose-dist and the embedded copy diverge —
// run `go generate ./internal/scaffold` after touching pose-dist/.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Module root = cwd when invoked via `go generate ./internal/scaffold`
	// (go generate runs in the package dir).
	pkgDir, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	src := filepath.Clean(filepath.Join(pkgDir, "..", "..", "..", "pose-dist"))
	if _, err := os.Stat(filepath.Join(src, "install.sh")); err != nil {
		// Standalone repo: the dist IS the module parent root.
		src = filepath.Clean(filepath.Join(pkgDir, "..", "..", ".."))
	}
	dst := filepath.Join(pkgDir, "dist")

	if _, err := os.Stat(filepath.Join(src, "install.sh")); err != nil {
		fatal(fmt.Errorf("pose dist not found (monorepo pose-dist/ or standalone root): %w", err))
	}
	if err := os.RemoveAll(dst); err != nil {
		fatal(err)
	}

	copied := 0
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return nil
		}
		if skip(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, _ := d.Info()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, b, info.Mode().Perm()); err != nil {
			return err
		}
		copied++
		return nil
	})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("scaffold: %d files synced from pose-dist\n", copied)
}

// skip mirrors the exclusion rules of the drift test: .claude (symlinks),
// stray binaries, git noise.
func skip(rel string, d fs.DirEntry) bool {
	if strings.HasPrefix(rel, ".claude") {
		return true
	}
	// Dual-home: no repo standalone do POSE a raiz do dist também contém o
	// código do produto — nada disso entra no scaffold embutido.
	top := strings.SplitN(rel, string(filepath.Separator), 2)[0]
	switch top {
	case ".git", ".github", ".gitignore", ".docs-site-build", ".idea", "pose-mcp", "mcp-enforce", "pose-action",
		"docs-site", "tests", "examples", ".goreleaser.yaml", ".gitleaks.toml", "dist-release",
		"compatibility.json", "compatibility-report.md":
		return true
	}
	// Append-only evidence is instance state, not scaffold: embedding it would
	// make every `pose validate --report` run drift the embed it was tested by.
	if rel == filepath.Join(".pose", "reports") || strings.HasPrefix(rel, filepath.Join(".pose", "reports")+string(filepath.Separator)) {
		return true
	}
	base := filepath.Base(rel)
	if base == "pose-mcp" || base == "pose-mcp-claude" {
		return true // binaries and legacy launchers are never embedded
	}
	if d.Type()&fs.ModeSymlink != 0 {
		return true
	}
	return false
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen:", err)
	os.Exit(1)
}

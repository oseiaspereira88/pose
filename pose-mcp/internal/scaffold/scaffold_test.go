package scaffold

// Drift guard (spec pose-cli-embed-standalone): the embedded dist/ must be
// byte-identical to pose-dist/ (minus .claude symlinks and binaries). When
// this fails, run `go generate ./internal/scaffold` and commit the sync.

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func poseDistDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "pose-dist"))
	if _, err := os.Stat(filepath.Join(dir, "install.sh")); err != nil {
		dir = filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	}
	if _, err := os.Stat(filepath.Join(dir, "install.sh")); err != nil {
		t.Skipf("pose dist not available at %s", dir)
	}
	return dir
}

func listSource(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, ".claude") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		top := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		switch top {
		case ".git", ".github", ".gitignore", "pose-mcp", "mcp-enforce", "pose-action", "docs-site",
			"tests", ".goreleaser.yaml", "dist-release":
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(rel)
		if base == "pose-mcp" || base == "pose-mcp-claude" || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestEmbeddedDistMatchesPoseDist(t *testing.T) {
	src := listSource(t, poseDistDir(t))
	embedded := map[string][]byte{}
	err := fs.WalkDir(Dist(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(Dist(), path)
		if err != nil {
			return err
		}
		embedded[path] = b
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var missing, extra, differs []string
	for rel := range src {
		if _, ok := embedded[rel]; !ok {
			missing = append(missing, rel)
		} else if !bytes.Equal(src[rel], embedded[rel]) {
			differs = append(differs, rel)
		}
	}
	for rel := range embedded {
		if _, ok := src[rel]; !ok {
			extra = append(extra, rel)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(differs)
	if len(missing)+len(extra)+len(differs) > 0 {
		t.Fatalf("embedded dist drifted from pose-dist — run `go generate ./internal/scaffold`\nmissing: %v\nextra: %v\ndiffers: %v",
			missing, extra, differs)
	}
	if len(embedded) < 50 {
		t.Fatalf("embedded dist suspiciously small: %d files", len(embedded))
	}
}

func TestClaudeSkillLinksMatchAgentsSkills(t *testing.T) {
	var fromEmbed []string
	entries, err := fs.ReadDir(Dist(), ".agents/skills")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			fromEmbed = append(fromEmbed, e.Name())
		}
	}
	sort.Strings(fromEmbed)
	var fromMap []string
	for name := range ClaudeSkillLinks {
		fromMap = append(fromMap, name)
	}
	sort.Strings(fromMap)
	if strings.Join(fromEmbed, ",") != strings.Join(fromMap, ",") {
		t.Fatalf("ClaudeSkillLinks drifted from .agents/skills:\nembed: %v\nmap:   %v", fromEmbed, fromMap)
	}
}

func TestEditorialDefaultsAreEnglishAndPtBROverlayIsComplete(t *testing.T) {
	root := poseDistDir(t)
	portugueseAccent := regexp.MustCompile(`[áéíóúãõâêôçÁÉÍÓÚÃÕÂÊÔÇ]`)
	prefixes := []string{".pose/workflows/", ".pose/rules/", ".agents/skills/"}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		inEditorialScope := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(rel, prefix) {
				inEditorialScope = true
				break
			}
		}
		if !inEditorialScope || filepath.Ext(path) != ".md" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if portugueseAccent.Match(content) {
			t.Errorf("English-default editorial artifact still contains Portuguese text: %s", rel)
		}
		localized := filepath.Join(root, "locales", "pt-BR", filepath.FromSlash(rel))
		if _, statErr := os.Stat(localized); statErr != nil {
			t.Errorf("pt-BR overlay missing for %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

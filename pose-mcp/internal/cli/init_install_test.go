package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-init-is-install.

func TestInitIsInstallInABareRepository(t *testing.T) {
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	repo := newGitRepo(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"init", "--skip-mcp", "--project-id", "proj.bare"}, &out, &errB); code != 0 {
			t.Fatalf("init exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if id := readProjectFileID(t, repo); id != "proj.bare" {
			t.Fatalf("init did not pass the installer flags through: project id %q", id)
		}
		out.Reset()
		errB.Reset()
		if code := Main([]string{"check", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("check --strict after init: code=%d out=%s", code, out.String())
		}
		out.Reset()
		if code := Main([]string{"new-spec", "first"}, &out, &errB); code != 0 {
			t.Fatalf("new-spec after init: %s", errB.String())
		}
		if _, err := os.Stat(filepath.Join(repo, ".mcp.json")); err == nil {
			t.Fatal("--skip-mcp was not passed through")
		}
	})
}

func TestInitOnAnInstalledInstanceWritesNothingElse(t *testing.T) {
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	repo, _ := installedInstance(t)
	agents := filepath.Join(repo, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"init"}, &out, &errB); code != 0 {
			t.Fatalf("init on an installed instance: %s", errB.String())
		}
		if !strings.Contains(out.String(), "already installed") || !strings.Contains(out.String(), "pose update") {
			t.Fatalf("init does not name the next step:\n%s", out.String())
		}
	})
	if raw, _ := os.ReadFile(agents); string(raw) != "# custom\n" {
		t.Fatal("init rewrote an installed instance")
	}
}

func TestInitIsInstallWithTheWizard(t *testing.T) {
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	repo := newGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.test/app\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"init", "--wizard", "--yes", "--skip-mcp"}, &out, &errB); code != 0 {
			t.Fatalf("init --wizard on a bare repository: code=%d out=%s err=%s", code, out.String(), errB.String())
		}
		out.Reset()
		if code := Main([]string{"check", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("check --strict after the wizard: %s", out.String())
		}
	})
}

func TestInitIsInstallIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md", "../../../docs-site/docs/cli.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "installer") && !strings.Contains(string(raw), "instalador") {
			t.Fatalf("%s does not describe init as installing", rel)
		}
	}
	if !strings.Contains(commandHelpCatalog["init"].DescriptionEN, "full installation") {
		t.Fatal("pose init help does not say it installs")
	}
}

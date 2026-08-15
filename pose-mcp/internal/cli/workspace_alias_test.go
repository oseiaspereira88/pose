package cli

// --root-only / --workspace <name> (spec pose-monorepo-validation-advisory,
// R2/R3): documented sugar over the existing --module <path> selector.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func workspaceAliasFixture(t *testing.T) string {
	t.Helper()
	repo := newGitRepo(t)
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.test/root\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(repo, "web", "package.json"), `{"name":"my-web-app"}`)
	mustWrite(t, filepath.Join(repo, "worker", "Cargo.toml"), "[package]\nname = \"my-worker\"\nversion = \"0.1.0\"\n")

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	matrix := fmt.Sprintf(`{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"self","program":%q,"args":["-test.run=^$"],"severity":"required"}]},"node":{"checks":[{"name":"self","program":%q,"args":["-test.run=^$"],"severity":"required"}]},"rust":{"checks":[{"name":"self","program":%q,"args":["-test.run=^$"],"severity":"required"}]}},"moduleOverrides":{}}`, executable, executable, executable)
	mustWrite(t, filepath.Join(repo, ".pose", "indexes", "validation-matrix.json"), matrix)
	return repo
}

func TestValidateRootOnlySelectsRootModule(t *testing.T) {
	repo := workspaceAliasFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--root-only"}, &out, &errB); code != 0 {
			t.Fatalf("validate --root-only exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "[module] .") {
			t.Errorf("expected only the root module to run, got: %s", out.String())
		}
		if strings.Contains(out.String(), "[module] web") || strings.Contains(out.String(), "[module] worker") {
			t.Errorf("--root-only ran a non-root module: %s", out.String())
		}
	})
}

func TestValidateWorkspaceResolvesNodePackageName(t *testing.T) {
	repo := workspaceAliasFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--workspace", "my-web-app"}, &out, &errB); code != 0 {
			t.Fatalf("validate --workspace exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "[module] web") {
			t.Errorf("expected the web module to run, got: %s", out.String())
		}
	})
}

func TestValidateWorkspaceResolvesCargoPackageName(t *testing.T) {
	repo := workspaceAliasFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--workspace", "my-worker"}, &out, &errB); code != 0 {
			t.Fatalf("validate --workspace exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "[module] worker") {
			t.Errorf("expected the worker module to run, got: %s", out.String())
		}
	})
}

func TestValidateWorkspaceUnknownNameFails(t *testing.T) {
	repo := workspaceAliasFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--workspace", "does-not-exist"}, &out, &errB); code != 2 {
			t.Fatalf("validate --workspace <unknown> exit=%d, want 2; err=%s", code, errB.String())
		}
		if !strings.Contains(errB.String(), "does-not-exist") {
			t.Errorf("error does not name the unresolved workspace: %s", errB.String())
		}
	})
}

func TestValidateRootOnlyAndModuleAreMutuallyExclusive(t *testing.T) {
	repo := workspaceAliasFixture(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--root-only", "--module", "web"}, &out, &errB); code != 2 {
			t.Fatalf("exit=%d, want 2", code)
		}
	})
}

package cli

// Locale-aware extension install (spec pose-rule-extension-locale-parity):
// `pose extension install` now resolves a `locales/<tag>/files/` overlay in
// the package, the same way core machinery already resolves a locale for
// AGENTS.md/POSE.md — content changes, the installed target path never
// does.

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeExtPkgWithLocale(t *testing.T, dir, id, version, kind string, files map[string]string, locale string, localizedFiles map[string]string) string {
	t.Helper()
	pkg := writeExtPkg(t, dir, id, version, kind, files, nil)
	for target, content := range localizedFiles {
		fp := filepath.Join(pkg, "locales", locale, "files", filepath.FromSlash(target))
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pkg
}

func TestLocalizedExtensionSourcePrefersOverlayWhenPresent(t *testing.T) {
	pkg := t.TempDir()
	base := filepath.Join(pkg, "files", ".pose", "rules", "x.md")
	overlay := filepath.Join(pkg, "locales", "pt-BR", "files", ".pose", "rules", "x.md")
	for _, p := range []string{base, overlay} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := localizedExtensionSource(pkg, "pt-BR", ".pose/rules/x.md"); got != overlay {
		t.Errorf("got %q, want overlay %q", got, overlay)
	}
	if got := localizedExtensionSource(pkg, "fr", ".pose/rules/x.md"); got != base {
		t.Errorf("no fr overlay: got %q, want base %q", got, base)
	}
	if got := localizedExtensionSource(pkg, "", ".pose/rules/x.md"); got != base {
		t.Errorf("empty locale: got %q, want base %q", got, base)
	}
	if got := localizedExtensionSource(pkg, "en", ".pose/rules/x.md"); got != base {
		t.Errorf("en locale: got %q, want base %q", got, base)
	}
}

func TestExtensionInstallUsesExplicitLocaleOverlay(t *testing.T) {
	fakeSignedInstall(t)
	root := newGitRepo(t)
	dir := t.TempDir()
	pkg := writeExtPkgWithLocale(t, dir, "pose-rule-x", "1.0.0", "rule",
		map[string]string{".pose/rules/x.md": "English content"},
		"pt-BR",
		map[string]string{".pose/rules/x.md": "Conteúdo em português"},
	)

	code, out := runExt(t, root, "install", pkg, "--yes", "--locale", "pt-BR")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	installed, err := os.ReadFile(filepath.Join(root, ".pose", "rules", "x.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "Conteúdo em português" {
		t.Errorf("installed content = %q, want the pt-BR overlay", installed)
	}
}

func TestExtensionInstallFallsBackToBaseWhenLocaleOverlayAbsent(t *testing.T) {
	fakeSignedInstall(t)
	root := newGitRepo(t)
	dir := t.TempDir()
	pkg := writeExtPkg(t, dir, "pose-rule-y", "1.0.0", "rule",
		map[string]string{".pose/rules/y.md": "English content"}, nil)

	// No locales/fr/ overlay exists in this package — must not error, must
	// fall back to the base content unchanged.
	code, out := runExt(t, root, "install", pkg, "--yes", "--locale", "fr")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	installed, err := os.ReadFile(filepath.Join(root, ".pose", "rules", "y.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "English content" {
		t.Errorf("installed content = %q, want the base fallback", installed)
	}
}

func TestExtensionInstallAutoDetectsTargetLocaleWithoutFlag(t *testing.T) {
	fakeSignedInstall(t)
	root := newGitRepo(t)
	// Real pt-BR core install so the target genuinely carries a pt-BR
	// POSE.md — the same detection surface machineryLocale already reads
	// for core machinery reinstalls (TestInstallReinstallDetectsExistingLocaleWithoutFlag).
	if code := cmdInstall([]string{root, "--skip-mcp", "--locale", "pt-BR"}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("pt-BR core install failed, exit=%d", code)
	}

	dir := t.TempDir()
	pkg := writeExtPkgWithLocale(t, dir, "pose-rule-z", "1.0.0", "rule",
		map[string]string{".pose/rules/z.md": "English content"},
		"pt-BR",
		map[string]string{".pose/rules/z.md": "Conteúdo em português"},
	)

	// No --locale flag: must auto-detect pt-BR from the target.
	code, out := runExt(t, root, "install", pkg, "--yes")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	installed, err := os.ReadFile(filepath.Join(root, ".pose", "rules", "z.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "Conteúdo em português" {
		t.Errorf("installed content = %q, want the auto-detected pt-BR overlay", installed)
	}
}

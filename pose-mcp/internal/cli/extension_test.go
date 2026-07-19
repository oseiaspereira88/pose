package cli

// Extension catalog lifecycle behavior (spec pose-extension-catalog-lifecycle):
// manifest validation (contents/compatibility/permissions/conflicts/
// provenance), dry-run and transactional install/remove with rollback,
// user-modification preservation, signature verification and revocation.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeExtPkg(t *testing.T, dir, id, version, kind string, files map[string]string, extra map[string]any) string {
	t.Helper()
	pkg := filepath.Join(dir, id+"-pkg")
	perms := []string{}
	for f := range files {
		p := filepath.Dir(f) + "/"
		found := false
		for _, existing := range perms {
			if existing == p {
				found = true
			}
		}
		if !found {
			perms = append(perms, p)
		}
	}
	manifest := map[string]any{
		"schema_version":    1,
		"id":                id,
		"version":           version,
		"kind":              kind,
		"description":       "test extension",
		"pose_schema_range": "1-1",
		"files":             keysOf(files),
		"permissions":       perms,
		"provenance":        map[string]any{"source": "https://example.com/repo"},
	}
	for k, v := range extra {
		manifest[k] = v
	}
	raw, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "extension.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	for target, content := range files {
		fp := filepath.Join(pkg, "files", filepath.FromSlash(target))
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pkg
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// fakeSignedInstall stubs signature verification to always succeed —
// isolates lifecycle tests from needing a real cosign binary.
func fakeSignedInstall(t *testing.T) {
	t.Helper()
	orig := verifyExtensionSignature
	verifyExtensionSignature = func(pkgDir string, m *extensionManifest) error { return nil }
	t.Cleanup(func() { verifyExtensionSignature = orig })
}

func runExt(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := cmdExtension(root, args, &out, &errB)
	return code, out.String() + errB.String()
}

func TestExtensionManifestValidation(t *testing.T) {
	root := t.TempDir()
	dir := t.TempDir()

	// Missing declared file.
	pkg := writeExtPkg(t, dir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)
	os.Remove(filepath.Join(pkg, "files", ".agents", "skills", "acme-skill", "SKILL.md"))
	if _, err := loadExtensionManifest(pkg); err == nil {
		t.Fatal("missing declared file must fail validation")
	}

	// Target outside the whitelist.
	pkg2 := writeExtPkg(t, dir, "acme-evil", "1.0.0", "skill", map[string]string{"evil/path.md": "x"}, nil)
	if _, err := loadExtensionManifest(pkg2); err == nil {
		t.Fatal("target outside extensionWhitelist must fail validation")
	}

	// Path escape.
	pkg3 := writeExtPkg(t, dir, "acme-escape", "1.0.0", "skill", map[string]string{".agents/skills/../../../../etc/passwd": "x"}, nil)
	if _, err := loadExtensionManifest(pkg3); err == nil {
		t.Fatal("path escape must fail validation")
	}

	// Revoked.
	pkg4 := writeExtPkg(t, dir, "acme-revoked", "1.0.0", "skill", map[string]string{".agents/skills/acme-revoked/SKILL.md": "x"}, map[string]any{"revoked": true, "revoked_reason": "known vulnerability"})
	_, err := loadExtensionManifest(pkg4)
	if err == nil {
		t.Fatal("revoked extension must fail closed")
	}
	_ = root
}

func TestExtensionInstallDryRunAppliesNothing(t *testing.T) {
	fakeSignedInstall(t)
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)

	code, out := runExt(t, root, "install", pkg, "--dry-run")
	if code != 0 {
		t.Fatalf("dry-run exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents/skills/acme-skill/SKILL.md")); err == nil {
		t.Fatal("dry-run must not write any file")
	}
	if _, err := os.Stat(extensionLockPath(root)); err == nil {
		t.Fatal("dry-run must not write the lock file")
	}
}

func TestExtensionInstallRequiresConsent(t *testing.T) {
	fakeSignedInstall(t)
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)
	code, out := runExt(t, root, "install", pkg)
	if code != 2 {
		t.Fatalf("install without --yes/--dry-run must require consent, exit=%d out=%s", code, out)
	}
}

func TestExtensionInstallAndListAndDigest(t *testing.T) {
	fakeSignedInstall(t)
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)

	code, out := runExt(t, root, "install", pkg, "--yes")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	content, err := os.ReadFile(filepath.Join(root, ".agents/skills/acme-skill/SKILL.md"))
	if err != nil || string(content) != "body" {
		t.Fatalf("file not installed correctly: %v %q", err, content)
	}
	_, listOut := runExt(t, root, "list")
	if !bytesContains(listOut, "acme-skill@1.0.0") {
		t.Errorf("list must show the installed extension: %s", listOut)
	}
}

func TestExtensionDigestIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	pkgA := writeExtPkg(t, dir, "same", "1.0.0", "skill", map[string]string{
		".agents/skills/same/SKILL.md": "one", ".agents/skills/same/notes.md": "two",
	}, nil)
	pkgB := writeExtPkg(t, filepath.Join(dir, "b"), "same", "1.0.0", "skill", map[string]string{
		".agents/skills/same/notes.md": "two", ".agents/skills/same/SKILL.md": "one",
	}, nil)
	mA, err := loadExtensionManifest(pkgA)
	if err != nil {
		t.Fatal(err)
	}
	mB, err := loadExtensionManifest(pkgB)
	if err != nil {
		t.Fatal(err)
	}
	dA, err := packageDigest(pkgA, mA.Files)
	if err != nil {
		t.Fatal(err)
	}
	dB, err := packageDigest(pkgB, mB.Files)
	if err != nil {
		t.Fatal(err)
	}
	if dA != dB {
		t.Errorf("digest must be order-independent: %s != %s", dA, dB)
	}
}

func TestExtensionConflictWithExistingUntrackedFile(t *testing.T) {
	fakeSignedInstall(t)
	root := t.TempDir()
	target := filepath.Join(root, ".agents/skills/acme-skill/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("pre-existing, untracked"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "new content"}, nil)

	code, out := runExt(t, root, "install", pkg, "--yes")
	if code == 0 {
		t.Fatal("installing over an untracked existing file must be rejected without --force")
	}
	if !bytesContains(out, "conflict") {
		t.Errorf("expected a conflict diagnostic: %s", out)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "pre-existing, untracked" {
		t.Error("the untracked file must be left completely unmodified")
	}
}

func TestExtensionConflictBetweenTwoExtensions(t *testing.T) {
	fakeSignedInstall(t)
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkgA := writeExtPkg(t, pkgDir, "ext-a", "1.0.0", "skill", map[string]string{".agents/skills/shared/SKILL.md": "from a"}, nil)
	if code, out := runExt(t, root, "install", pkgA, "--yes"); code != 0 {
		t.Fatalf("install a failed: %s", out)
	}
	pkgB := writeExtPkg(t, filepath.Join(pkgDir, "b"), "ext-b", "1.0.0", "skill", map[string]string{".agents/skills/shared/SKILL.md": "from b"}, nil)
	code, out := runExt(t, root, "install", pkgB, "--yes")
	if code == 0 {
		t.Fatal("installing over a file owned by a different extension must be rejected without --force")
	}
	if !bytesContains(out, "ext-a") {
		t.Errorf("conflict must name the owning extension: %s", out)
	}
}

// Rollback: a multi-file install where the second file's write is forced to
// fail must leave zero trace of the first file (transactional, R2). The
// failure is injected at the destination (an unwritable target directory)
// so it happens during apply, after planning/signature checks succeeded —
// exercising the actual rollback path, not an earlier validation error.
func TestExtensionInstallRollsBackOnFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory write permission bits")
	}
	fakeSignedInstall(t)
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := filepath.Join(pkgDir, "acme-multi-pkg")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	// Explicit, ordered files list — the map-based helper's key order is
	// nondeterministic and this test depends on file-1-then-file-2 order.
	manifest := map[string]any{
		"schema_version": 1, "id": "acme-multi", "version": "1.0.0", "kind": "skill",
		"description": "test", "pose_schema_range": "1-1",
		"files":       []string{".agents/skills/acme-ok/SKILL.md", ".agents/skills/acme-blocked/SKILL.md"},
		"permissions": []string{".agents/skills/"},
		"provenance":  map[string]any{"source": "https://example.com/repo"},
	}
	raw, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(pkg, "extension.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{".agents/skills/acme-ok/SKILL.md", ".agents/skills/acme-blocked/SKILL.md"} {
		fp := filepath.Join(pkg, "files", filepath.FromSlash(target))
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fp, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Pre-create the second file's destination directory read-only, so the
	// second write fails at apply time, after the first has already landed.
	blockedDir := filepath.Join(root, ".agents/skills/acme-blocked")
	if err := os.MkdirAll(blockedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blockedDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedDir, 0o755) })

	code, out := runExt(t, root, "install", pkg, "--yes")
	if code == 0 {
		t.Fatalf("install must fail when the second file cannot be written: %s", out)
	}
	if !bytesContains(out, "rolled back") {
		t.Errorf("expected a rollback diagnostic: %s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents/skills/acme-ok/SKILL.md")); err == nil {
		t.Fatal("the first file must be rolled back when the second write fails")
	}
	if _, err := os.Stat(extensionLockPath(root)); err == nil {
		t.Fatal("a failed transaction must never write the lock file")
	}
}

func TestExtensionRemovePreservesUserModifications(t *testing.T) {
	fakeSignedInstall(t)
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "original"}, nil)
	if code, out := runExt(t, root, "install", pkg, "--yes"); code != 0 {
		t.Fatalf("install failed: %s", out)
	}
	target := filepath.Join(root, ".agents/skills/acme-skill/SKILL.md")
	if err := os.WriteFile(target, []byte("user edited this"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runExt(t, root, "remove", "acme-skill", "--yes")
	if code == 0 {
		t.Fatal("removing a user-modified file must be rejected without --force")
	}
	if !bytesContains(out, "modified since install") {
		t.Errorf("expected a modification diagnostic: %s", out)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("the user-modified file must survive a rejected remove")
	}
	// --force overrides.
	code, _ = runExt(t, root, "remove", "acme-skill", "--yes", "--force")
	if code != 0 {
		t.Fatal("remove --force must succeed even with local modifications")
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("the file must be gone after a forced remove")
	}
}

func TestExtensionUnsignedRejectedByDefault(t *testing.T) {
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)
	code, out := runExt(t, root, "install", pkg, "--yes")
	if code == 0 {
		t.Fatal("an unsigned extension must be rejected by default")
	}
	if !bytesContains(out, "unsigned") && !bytesContains(out, "signature") {
		t.Errorf("expected a signature diagnostic: %s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents/skills/acme-skill/SKILL.md")); err == nil {
		t.Fatal("nothing must be written when signature verification fails")
	}
}

func TestExtensionAllowUnsignedOptOut(t *testing.T) {
	root := t.TempDir()
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)
	code, out := runExt(t, root, "install", pkg, "--yes", "--allow-unsigned")
	if code != 0 {
		t.Fatalf("--allow-unsigned must permit an explicit opt-out: %s", out)
	}
}

func TestExtensionVerifyCommand(t *testing.T) {
	pkgDir := t.TempDir()
	pkg := writeExtPkg(t, pkgDir, "acme-skill", "1.0.0", "skill", map[string]string{".agents/skills/acme-skill/SKILL.md": "body"}, nil)
	code, out := runExt(t, "", "verify", pkg, "--allow-unsigned")
	if code != 0 {
		t.Fatalf("verify --allow-unsigned exit=%d out=%s", code, out)
	}
	if !bytesContains(out, "digest=") {
		t.Errorf("expected a digest in the verify output: %s", out)
	}
}

func bytesContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

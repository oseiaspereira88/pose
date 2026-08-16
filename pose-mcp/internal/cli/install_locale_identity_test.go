package cli

// Install auto-detect message clarity and project-identity preservation
// (specs pose-post-install-gate-locale, pose-install-identity-preservation):
// two small operator-facing correctness gaps found auditing `pose install`/
// `pose update` across real repositories ahead of the v1.4.0 release.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallDoesNotWarnOnAutoDetectedLocale: a fresh install with no
// --locale flag auto-detects English and must not print "locale ” not
// available" — "" is auto-detection's own spelling of English, not a
// rejected explicit request.
func TestInstallDoesNotWarnOnAutoDetectedLocale(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	if strings.Contains(out.String(), "locale '' not available") || strings.Contains(out.String(), "locale '' indisponível") {
		t.Errorf("auto-detected locale must not read as a rejected explicit request:\n%s", out.String())
	}
}

// TestForcedUpdatePreservesProjectIdentityAcrossRename: `pose update --force`
// (which delegates to cmdInstall) must recover the project's already-declared
// name/id from its existing AGENTS.md before ever falling back to the
// current directory's basename — otherwise cloning or renaming a repository
// silently renamed its declared identity on the next forced update.
func TestForcedUpdatePreservesProjectIdentityAcrossRename(t *testing.T) {
	parent := t.TempDir()
	original := filepath.Join(parent, "original-name")
	if err := os.Mkdir(original, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", original, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{original, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	before, err := os.ReadFile(filepath.Join(original, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(before), "# AGENTS.md — original-name") {
		t.Fatalf("fixture did not render the expected project name:\n%s", before)
	}

	renamed := filepath.Join(parent, "renamed-dir")
	if err := os.Rename(original, renamed); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errB.Reset()
	if code := cmdUpdate(renamed, []string{"--force", "--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update --force exit=%d err=%s out=%s", code, errB.String(), out.String())
	}
	after, err := os.ReadFile(filepath.Join(renamed, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "# AGENTS.md — original-name") {
		t.Errorf("project identity did not survive a directory rename under --force:\n%s", after)
	}
	if strings.Contains(string(after), "# AGENTS.md — renamed-dir") {
		t.Errorf("project identity was silently replaced by the current directory name:\n%s", after)
	}
}

// TestInstallWarnsWhenTargetAlreadyFailsBeforeAnyWrite regression-covers the
// honest-failure-message half of spec
// pose-install-gate-failure-recovery-notice's own follow-up ("run before
// delivery so the failure is honest about what did not happen"): a target
// that already fails `pose check --strict` before this run touches anything
// must say so up front, so a post-install gate failure caused by
// pre-existing debt does not read as something this command broke.
func TestInstallWarnsWhenTargetAlreadyFailsBeforeAnyWrite(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	// Break the instance in a way unrelated to anything a rerun would touch.
	matrixPath := filepath.Join(repo, ".pose", "indexes", "validation-matrix.json")
	if err := os.WriteFile(matrixPath, []byte(`{"defaults":{"mode":"strict"},"stacks":{},"unknownKey":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errB.Reset()
	_ = cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB)
	if !strings.Contains(out.String(), "already fails") {
		t.Errorf("rerun over an already-broken target must warn about it before writing anything:\n%s", out.String())
	}
}

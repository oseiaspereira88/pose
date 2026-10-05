package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-progressive-spec-surface.
func TestMinimalSurfaceKeepsGovernanceAndDropsRitual(t *testing.T) {
	root := t.TempDir()
	tmpl, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "templates", "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose/templates/spec.md"), string(tmpl))
	if code, out := runPose(t, root, "new-spec", "tiny-fix", "--surface", "minimal"); code != 0 {
		t.Fatalf("new-spec: %s", out)
	}
	files, _ := filepath.Glob(filepath.Join(root, ".pose/specs/*-tiny-fix.md"))
	if len(files) != 1 {
		t.Fatalf("spec not created: %v", files)
	}
	raw, _ := os.ReadFile(files[0])
	text := string(raw)
	for _, want := range []string{"surface: minimal", "## 1. Intent", "## 2. Requirements", "### Artifacts", "### Requirement trace", "### Follow-ups", "pose specs facts"} {
		if !strings.Contains(text, want) {
			t.Errorf("minimal surface lacks %q", want)
		}
	}
	if n := strings.Count(text, "\nsurface:"); n != 1 {
		t.Errorf("frontmatter carries %d surface keys, want 1", n)
	}
	for _, gone := range []string{"## 4. Tasks", "### Files and modules changed", "### Validation executed"} {
		if strings.Contains(text, gone) {
			t.Errorf("minimal surface still carries %q", gone)
		}
	}
}

func TestLintAppliesTheSurfaceButNotToLifecycleGates(t *testing.T) {
	root := t.TempDir()
	body := "---\nslug: small\nstatus: done\ncreated_at: 2026-10-04\ncompleted_at: 2026-10-04\nsurface: minimal\n---\n\n# Spec: small\n\n## 1. Intent\nFix a typo.\n\n## 2. Requirements\n- R1: The word is spelled right.\n\n## 3. Technical Plan\n### Artifacts\n- modified: README.md\n\n## 6. Validation\n### Requirement trace\n- R1 [satisfied] check:spelling\n\n## 7. Final Report\nDone.\n"
	path := filepath.Join(root, ".pose/specs/2026-10-04-small.md")
	mustWrite(t, path, body)
	code, out := runPose(t, root, "lint-spec", "small")
	if code != 0 || strings.Contains(out, "required section missing: Tasks") {
		t.Fatalf("a minimal spec without Tasks must pass: %d %s", code, out)
	}
	mustWrite(t, path, strings.Replace(body, "- R1 [satisfied] check:spelling\n", "", 1))
	if code, out := runPose(t, root, "lint-spec", "small", "--strict"); code == 0 {
		t.Fatalf("a minimal surface relaxed the requirement trace gate: %s", out)
	}
	mustWrite(t, path, strings.Replace(body, "surface: minimal\n", "", 1))
	if code, out := runPose(t, root, "lint-spec", "small"); code == 0 {
		t.Fatalf("without the minimal surface, Tasks stays required: %s", out)
	}
}

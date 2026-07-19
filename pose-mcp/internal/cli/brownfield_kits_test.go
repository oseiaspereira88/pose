package cli

// Brownfield reference kits (spec pose-brownfield-reference-kits): the real,
// checked-in fixtures under examples/brownfield-kits/ are exercised here
// end to end — direct adoption, Spec Kit import and OpenSpec import — each
// proving preservation of pre-existing content, surfaced curation warnings,
// DoR readiness of the generated artifact, and that rollback is always a
// plain git revert (nothing pre-existing is ever modified).

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixtureTree copies a real on-disk directory (a checked-in example
// kit fixture) into dst, which must already exist.
func copyFixtureTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatalf("copying fixture %s: %v", src, err)
	}
}

// brownfieldKitFixture locates examples/brownfield-kits/<kit>/fixture in
// this checkout, copies it into a fresh git repo and commits the
// pre-existing content as the adoption baseline. Skips if the example
// tree isn't reachable (e.g. running from a stripped source tarball).
func brownfieldKitFixture(t *testing.T, kit string) string {
	t.Helper()
	repoRoot, err := repoRootForTest()
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	src := filepath.Join(repoRoot, "examples", "brownfield-kits", kit, "fixture")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("example kit fixture not found: %v", err)
	}
	dst := newGitRepo(t)
	copyFixtureTree(t, src, dst)
	if out, err := exec.Command("git", "-C", dst, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", dst, "-c", "user.email=test@test", "-c", "user.name=test", "commit", "-q", "-m", "baseline").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	return dst
}

// gitPorcelain returns `git status --porcelain` lines, used to prove
// rollback safety: pre-existing tracked files must show zero modification
// after adoption — only new, untracked paths are ever added.
func gitPorcelain(t *testing.T, repo string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatalf("git status: %v: %s", err, out)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestBrownfieldDirectAdoptionKit(t *testing.T) {
	repo := brownfieldKitFixture(t, "direct-adoption")
	original := map[string]string{}
	for _, rel := range []string{"README.md", "service/go.mod", "service/main.go"} {
		b, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			t.Fatal(err)
		}
		original[rel] = string(b)
	}

	// Stage 1 — visibility: doctor reports POSE is not installed yet.
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"doctor"}, &out, &errB); code != 1 {
			t.Fatalf("doctor on a brownfield repo before adoption should fail visibly, exit=%d", code)
		}
	})

	// Stage 2 — adoption: install already runs check --strict as its own
	// exit gate, so a successful install is already a passing blocking
	// gate for POSE's own structure.
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, installOut.String(), installErr.String())
	}

	for rel, want := range original {
		got, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil || string(got) != want {
			t.Errorf("pre-existing %s was not preserved byte-for-byte: err=%v", rel, err)
		}
	}

	inDir(t, repo, func() {
		// Stage 3 — register the detected Go module, visibility first
		// (tolerant), then the blocking gate (strict).
		var out, errB bytes.Buffer
		if code := Main([]string{"init", "--wizard", "--yes"}, &out, &errB); code != 0 {
			t.Fatalf("init --wizard exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := Main([]string{"validate", "--tolerant"}, &out, &errB); code != 0 {
			t.Fatalf("validate --tolerant (visibility) exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := Main([]string{"validate", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("validate --strict (blocking gate) exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := Main([]string{"check", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("check --strict (blocking gate) exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
	})

	// Rollback safety: pre-existing tracked files show zero modification;
	// everything POSE added is untracked, so `git clean -fdx` (or simply
	// never committing) fully reverts adoption.
	for _, line := range gitPorcelain(t, repo) {
		for rel := range original {
			if strings.HasSuffix(line, rel) && !strings.HasPrefix(line, "??") {
				t.Errorf("pre-existing file was modified by adoption: %s", line)
			}
		}
	}
}

func TestBrownfieldSpecKitImportKit(t *testing.T) {
	repo := brownfieldKitFixture(t, "spec-kit-import")
	original := map[string]string{}
	for _, rel := range []string{"README.md", "src/notify.py"} {
		b, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			t.Fatal(err)
		}
		original[rel] = string(b)
	}

	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, installOut.String(), installErr.String())
	}

	const slug = "user-notifications"
	inDir(t, repo, func() {
		// Stage 1 — visibility: dry-run writes nothing.
		var out, errB bytes.Buffer
		if code := Main([]string{"import", "spec-kit", ".specify/specs", "--dry-run"}, &out, &errB); code != 0 {
			t.Fatalf("dry-run exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "dry_run=true") {
			t.Errorf("expected a dry-run marker: %s", out.String())
		}
		if _, err := os.Stat(filepath.Join(repo, ".pose", "specs", slug)); !os.IsNotExist(err) {
			t.Fatalf("dry-run must not write the destination: %v", err)
		}

		// Stage 2 — import: warnings for the intentionally-missing plan.md
		// (a realistic brownfield gap) must surface, not be silently eaten.
		out.Reset()
		errB.Reset()
		if code := Main([]string{"import", "spec-kit", ".specify/specs"}, &out, &errB); code != 0 {
			t.Fatalf("import exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "warning=") || !strings.Contains(out.String(), "plan.md not found") {
			t.Errorf("expected the plan.md curation warning to surface: %s", out.String())
		}

		// Stage 3 — readiness: the generated spec already clears the DoR
		// gate structurally, independent of open curation notes.
		out.Reset()
		errB.Reset()
		if code := Main([]string{"lint-spec", slug, "--ready-check"}, &out, &errB); code != 0 {
			t.Fatalf("generated spec is not ready: exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "spec.ready=true") {
			t.Errorf("expected spec.ready=true: %s", out.String())
		}

		// The curation warning is tracked as an open follow-up, not lost.
		out.Reset()
		errB.Reset()
		if code := Main([]string{"followups", "--open"}, &out, &errB); code != 0 {
			t.Fatalf("followups --open exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), slug) {
			t.Errorf("expected an open follow-up for %s: %s", slug, out.String())
		}
	})

	for rel, want := range original {
		got, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil || string(got) != want {
			t.Errorf("pre-existing %s was not preserved byte-for-byte: err=%v", rel, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(repo, ".pose", "specs", slug, "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Format: `spec-kit`", "001-user-notifications/spec.md",
		"R1: FR-001", "plan.md not found",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("generated spec missing %q", want)
		}
	}
}

func TestBrownfieldOpenSpecImportKit(t *testing.T) {
	repo := brownfieldKitFixture(t, "openspec-import")
	original := map[string]string{}
	for _, rel := range []string{"README.md", "src/auth.py"} {
		b, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			t.Fatal(err)
		}
		original[rel] = string(b)
	}

	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install exit=%d out=%s err=%s", code, installOut.String(), installErr.String())
	}

	change := filepath.Join("openspec", "changes", "add-notifications")

	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"import", "openspec", change, "--dry-run"}, &out, &errB); code != 0 {
			t.Fatalf("dry-run exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "dry_run=true") {
			t.Errorf("expected a dry-run marker: %s", out.String())
		}

		out.Reset()
		errB.Reset()
		if code := Main([]string{"import", "openspec", change}, &out, &errB); code != 0 {
			t.Fatalf("import exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "warning=") || !strings.Contains(out.String(), "design.md not found") {
			t.Errorf("expected the design.md curation warning to surface: %s", out.String())
		}
	})

	const slug = "add-notifications-notifications"
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"lint-spec", slug, "--ready-check"}, &out, &errB); code != 0 {
			t.Fatalf("generated spec is not ready: exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "spec.ready=true") {
			t.Errorf("expected spec.ready=true: %s", out.String())
		}
	})

	for rel, want := range original {
		got, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil || string(got) != want {
			t.Errorf("pre-existing %s was not preserved byte-for-byte: err=%v", rel, err)
		}
	}
	content, err := os.ReadFile(filepath.Join(repo, ".pose", "specs", slug, "spec.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[ADDED] Shipment notification", "Scenario: Order ships", "design.md not found",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("generated spec missing %q", want)
		}
	}
}

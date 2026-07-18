package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// newGitRepo creates a temp dir initialized as a git repository and returns
// its path. Tests chdir into it so projectRoot() resolves there.
func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

func inDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	fn()
}

func TestVersionWorksOutsideRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo, no .pose
	inDir(t, dir, func() {
		var out, errB bytes.Buffer
		code := Main([]string{"version"}, &out, &errB)
		if code != 0 {
			t.Fatalf("version exit=%d stderr=%s", code, errB.String())
		}
		if !strings.Contains(out.String(), "pose ") {
			t.Fatalf("version output missing binary version: %q", out.String())
		}
	})
}

func TestUnknownCommandExit2(t *testing.T) {
	var out, errB bytes.Buffer
	code := Main([]string{"definitely-not-a-command"}, &out, &errB)
	if code != 2 {
		t.Fatalf("unknown command exit=%d, want 2", code)
	}
	if !strings.Contains(errB.String(), "Unknown command") {
		t.Fatalf("missing error message: %q", errB.String())
	}
}

func TestDelegationPropagatesArgsAndExitCode(t *testing.T) {
	repo := newGitRepo(t)
	scripts := filepath.Join(repo, ".pose", "scripts")
	if err := os.MkdirAll(scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	stub := "#!/usr/bin/env bash\necho \"args:$*\"\nexit 3\n"
	if err := os.WriteFile(filepath.Join(scripts, "pose-check.sh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		code := Main([]string{"check", "--strict", "extra"}, &out, &errB)
		if code != 3 {
			t.Fatalf("delegated exit=%d, want 3 (stderr=%s)", code, errB.String())
		}
		if !strings.Contains(out.String(), "args:--strict extra") {
			t.Fatalf("args not propagated: %q", out.String())
		}
	})
}

func TestDelegationMissingEngineIsActionable(t *testing.T) {
	repo := newGitRepo(t) // no .pose/scripts
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		code := Main([]string{"check"}, &out, &errB)
		if code != 1 {
			t.Fatalf("exit=%d, want 1", code)
		}
		if !strings.Contains(errB.String(), "script engine not found") {
			t.Fatalf("missing actionable message: %q", errB.String())
		}
	})
}

func TestInitNativeCreatesStructure(t *testing.T) {
	repo := newGitRepo(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"init"}, &out, &errB); code != 0 {
			t.Fatalf("init exit=%d stderr=%s", code, errB.String())
		}
		for _, rel := range instanceDirs {
			if _, err := os.Stat(filepath.Join(repo, rel)); err != nil {
				t.Errorf("missing dir after init: %s", rel)
			}
		}
		// Idempotent second run.
		out.Reset()
		if code := Main([]string{"init"}, &out, &errB); code != 0 {
			t.Fatalf("second init exit=%d", code)
		}
		if !strings.Contains(out.String(), "already present") {
			t.Fatalf("second init not idempotent: %q", out.String())
		}
	})
}

func TestNewSpecNativeCreatesTemplateAndRejectsInvalidInput(t *testing.T) {
	repo := newGitRepo(t)
	templateDir := filepath.Join(repo, ".pose", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	template := "slug: <feature-slug>\ncreated_at: <YYYY-MM-DD>\n"
	if err := os.WriteFile(filepath.Join(templateDir, "spec.md"), []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"new-spec", "user-auth"}, &out, &errB); code != 0 {
			t.Fatalf("new-spec exit=%d stderr=%s", code, errB.String())
		}
		content, err := os.ReadFile(filepath.Join(repo, ".pose", "specs", "user-auth", "spec.md"))
		if err != nil || !strings.Contains(string(content), "slug: user-auth") || strings.Contains(string(content), "<YYYY-MM-DD>") {
			t.Fatalf("template not materialized: %q err=%v", content, err)
		}
		if code := Main([]string{"new-spec", "user-auth"}, &out, &errB); code != 1 {
			t.Fatalf("duplicate exit=%d, want 1", code)
		}
		if code := Main([]string{"new-spec", "../escape"}, &out, &errB); code != 2 {
			t.Fatalf("invalid slug exit=%d, want 2", code)
		}
	})
}

func TestFollowupsNativeOpenAllAndJSON(t *testing.T) {
	repo := newGitRepo(t)
	for slug, body := range map[string]string{
		"one": "## 7. Final Report\n\n### Follow-ups\n- [open] investigate cache\n- [done] shipped\n",
		"two": "## 7. Final Report\n\n### Follow-ups\n- untriaged item\n",
	} {
		path := filepath.Join(repo, ".pose", "specs", slug)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "spec.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"followups"}, &out, &errB); code != 0 || strings.Contains(out.String(), "shipped") || !strings.Contains(out.String(), "investigate cache") {
			t.Fatalf("open output=%q code=%d err=%q", out.String(), code, errB.String())
		}
		out.Reset()
		Main([]string{"followups", "--all"}, &out, &errB)
		if !strings.Contains(out.String(), "shipped") {
			t.Fatalf("all output=%q", out.String())
		}
		out.Reset()
		Main([]string{"followups", "--json"}, &out, &errB)
		if !strings.Contains(out.String(), `"total":2`) {
			t.Fatalf("json output=%q", out.String())
		}
		if code := Main([]string{"followups", "--similarity", "101"}, &out, &errB); code != 2 {
			t.Fatalf("invalid similarity exit=%d, want 2", code)
		}
	})
}

func TestReportNativeCreatesMarkdownAndValidatesArgs(t *testing.T) {
	repo := newGitRepo(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"report", "--task", "native report", "--outcome", "pass"}, &out, &errB); code != 0 {
			t.Fatalf("report exit=%d stderr=%s", code, errB.String())
		}
		matches, err := filepath.Glob(filepath.Join(repo, ".pose", "reports", "*-standard-native-report.md"))
		if err != nil || len(matches) != 1 {
			t.Fatalf("report path matches=%v err=%v", matches, err)
		}
		content, _ := os.ReadFile(matches[0])
		if !strings.Contains(string(content), "Outcome: pass") {
			t.Fatalf("report content=%q", content)
		}
		if code := Main([]string{"report", "--task", "x", "--outcome", "bad"}, &out, &errB); code != 2 {
			t.Fatalf("invalid outcome exit=%d", code)
		}
	})
}

func TestCLILocaleSelectionAndFallback(t *testing.T) {
	old := os.Getenv("POSE_LOCALE")
	t.Cleanup(func() { _ = os.Setenv("POSE_LOCALE", old) })
	for _, tc := range []struct {
		locale, want string
	}{
		{"en", "Unknown command"},
		{"pt-BR", "Comando desconhecido"},
		{"fr", "Unknown command"},
	} {
		_ = os.Setenv("POSE_LOCALE", tc.locale)
		var out, errB bytes.Buffer
		if code := Main([]string{"not-a-command"}, &out, &errB); code != 2 || !strings.Contains(errB.String(), tc.want) {
			t.Fatalf("locale=%s code=%d stderr=%q", tc.locale, code, errB.String())
		}
	}
}

// TestInitParityWithBashEngine guards against drift between the native list
// and pose-init.sh REQUIRED_DIRS (R5): it parses the script and compares.
func TestInitParityWithBashEngine(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "scripts", "pose-init.sh"))
	if err != nil {
		t.Skipf("bash engine not available: %v", err)
	}
	block := regexp.MustCompile(`(?s)REQUIRED_DIRS=\((.*?)\)`).FindStringSubmatch(string(script))
	if block == nil {
		t.Fatal("REQUIRED_DIRS array not found in pose-init.sh")
	}
	re := regexp.MustCompile(`"\$(?:POSE_DIR|ROOT_DIR)(/[^"]+)"`)
	var fromBash []string
	for _, m := range re.FindAllStringSubmatch(block[1], -1) {
		p := m[1]
		if strings.HasPrefix(m[0], `"$POSE_DIR`) {
			p = ".pose" + p
		} else {
			p = strings.TrimPrefix(p, "/")
		}
		fromBash = append(fromBash, strings.TrimPrefix(p, "/"))
	}
	native := append([]string(nil), instanceDirs...)
	sort.Strings(fromBash)
	sort.Strings(native)
	if strings.Join(fromBash, ",") != strings.Join(native, ",") {
		t.Fatalf("init parity drift:\n bash:   %v\n native: %v", fromBash, native)
	}
}

func TestTelemetryOptInLifecycle(t *testing.T) {
	repo := newGitRepo(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"telemetry", "status"}, &out, &errB); code != 0 {
			t.Fatalf("status exit=%d", code)
		}
		if !strings.Contains(out.String(), "disabled") {
			t.Fatalf("default must be disabled: %q", out.String())
		}
		out.Reset()
		if code := Main([]string{"telemetry", "enable"}, &out, &errB); code != 0 {
			t.Fatalf("enable exit=%d stderr=%s", code, errB.String())
		}
		out.Reset()
		Main([]string{"telemetry", "status"}, &out, &errB)
		if !strings.Contains(out.String(), "enabled") || !strings.Contains(out.String(), "anon_id") {
			t.Fatalf("expected enabled with anon_id: %q", out.String())
		}
		// emit é no-op sem POSE_TELEMETRY_URL — não deve panicar nem demorar.
		emitTelemetry("check")
		out.Reset()
		Main([]string{"telemetry", "disable"}, &out, &errB)
		out.Reset()
		Main([]string{"telemetry", "status"}, &out, &errB)
		if !strings.Contains(out.String(), "disabled") {
			t.Fatalf("expected disabled after disable: %q", out.String())
		}
	})
}

func TestDoctorHealthyAndBrokenInstance(t *testing.T) {
	repo := newGitRepo(t)
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		// Sem .pose/: erro com hint de install.
		code := Main([]string{"doctor"}, &out, &errB)
		if code != 1 {
			t.Fatalf("doctor sem .pose deve falhar: exit=%d out=%s", code, out.String())
		}
		if !strings.Contains(out.String(), "install.sh") {
			t.Fatalf("hint de install ausente: %q", out.String())
		}
		// Instância mínima com motor + schema: melhora o quadro.
		if err := os.MkdirAll(filepath.Join(repo, ".pose", "scripts"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, ".pose", "scripts", "pose-lib.sh"),
			[]byte("POSE_SCHEMA_VERSION=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, ".pose", "schema-version"), []byte("1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		code = Main([]string{"doctor", "--json"}, &out, &errB)
		if code != 0 {
			t.Fatalf("doctor com instância mínima: exit=%d out=%s", code, out.String())
		}
		if !strings.Contains(out.String(), `"errors": 0`) {
			t.Fatalf("esperado errors=0 no JSON: %q", out.String())
		}
		// Instância mais nova que o motor: erro.
		if err := os.WriteFile(filepath.Join(repo, ".pose", "schema-version"), []byte("99\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		if code = Main([]string{"doctor"}, &out, &errB); code != 1 {
			t.Fatalf("instância v99 > motor v1 deve falhar: exit=%d", code)
		}
	})
}

func TestInstallEmbeddedFreshAndIdempotent(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}
	for _, rel := range []string{".pose/scripts/pose-lib.sh", ".pose/schema-version", "AGENTS.md", "pose", ".pose/LICENSE"} {
		if _, err := os.Stat(filepath.Join(repo, rel)); err != nil {
			t.Errorf("missing after install: %s", rel)
		}
	}
	// Conteúdo de usuário + rule custom sobrevivem ao re-run.
	if err := os.MkdirAll(filepath.Join(repo, ".pose", "specs", "user-spec"), 0o755); err != nil {
		t.Fatal(err)
	}
	userSpec := "---\nslug: user-spec\nstatus: draft\ncreated_at: 2026-07-17\n---\n\n# Spec: user-spec\n\nuser\n"
	os.WriteFile(filepath.Join(repo, ".pose", "specs", "user-spec", "spec.md"), []byte(userSpec), 0o644)
	os.WriteFile(filepath.Join(repo, ".pose", "rules", "my-rule.md"), []byte("custom"), 0o644)
	agents, _ := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	os.WriteFile(filepath.Join(repo, "AGENTS.md"), append(agents, []byte("\nEDITED\n")...), 0o644)
	out.Reset()
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("re-run exit=%d err=%s", code, errB.String())
	}
	if b, _ := os.ReadFile(filepath.Join(repo, ".pose", "specs", "user-spec", "spec.md")); !strings.Contains(string(b), "user") {
		t.Error("user spec content lost")
	}
	if _, err := os.Stat(filepath.Join(repo, ".pose", "rules", "my-rule.md")); err != nil {
		t.Error("custom rule deleted")
	}
	if b, _ := os.ReadFile(filepath.Join(repo, "AGENTS.md")); !strings.Contains(string(b), "EDITED") {
		t.Error("edited AGENTS.md overwritten without --force")
	}
	// Locale pt-BR num alvo novo.
	repo2 := newGitRepo(t)
	out.Reset()
	if code := Main([]string{"install", repo2, "--skip-mcp", "--locale", "pt-BR"}, &out, &errB); code != 0 {
		t.Fatalf("locale install exit=%d err=%s", code, errB.String())
	}
	if b, _ := os.ReadFile(filepath.Join(repo2, "AGENTS.md")); !strings.Contains(string(b), "Precedência") {
		t.Error("pt-BR locale not applied")
	}
}

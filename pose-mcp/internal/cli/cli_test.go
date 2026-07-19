package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func TestInstallMigratesOnlyLegacyMCPEntries(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".mcp.json")
	legacy := `{"mcpServers":{"harne8-pose":{"type":"stdio","command":"/project/.pose/bin/pose-mcp-claude","env":{"KEEP":"yes"}},"custom":{"command":"custom-mcp","args":["serve"]},"custom-same-basename":{"command":"/company/pose-mcp","args":["proxy"]}}}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	action, err := configureMCP(path, root, "proj.test")
	if err != nil || action != "migrated" {
		t.Fatalf("configureMCP action=%q err=%v", action, err)
	}
	var payload map[string]any
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, &payload) != nil {
		t.Fatalf("reading migrated config: %v", err)
	}
	servers := payload["mcpServers"].(map[string]any)
	poseEntry := servers["harne8-pose"].(map[string]any)
	if poseEntry["command"] != "pose" {
		t.Fatalf("pose command = %v", poseEntry["command"])
	}
	args := poseEntry["args"].([]any)
	if len(args) != 2 || args[0] != "serve-mcp" || args[1] != "--stdio" {
		t.Fatalf("pose args = %v", args)
	}
	env := poseEntry["env"].(map[string]any)
	if env["KEEP"] != "yes" || env["POSE_PROJECT_ROOT"] != root || env["POSE_DEFAULT_PROJECT_ID"] != "proj.test" {
		t.Fatalf("pose env = %v", env)
	}
	if servers["custom"].(map[string]any)["command"] != "custom-mcp" {
		t.Fatalf("custom server changed: %v", servers["custom"])
	}
	if servers["custom-same-basename"].(map[string]any)["command"] != "/company/pose-mcp" {
		t.Fatalf("custom same-basename server changed: %v", servers["custom-same-basename"])
	}
}

func TestInstallPreservesCustomMCPConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".mcp.json")
	custom := []byte(`{"mcpServers":{"pose":{"command":"company-pose-proxy","args":["run"]}}}`)
	if err := os.WriteFile(path, custom, 0o644); err != nil {
		t.Fatal(err)
	}
	action, err := configureMCP(path, root, "proj.test")
	if err != nil || action != "preserved" {
		t.Fatalf("configureMCP action=%q err=%v", action, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != string(custom) {
		t.Fatalf("custom config changed: %q err=%v", raw, err)
	}
}

func TestNativeScaffoldsCreateContractArtifacts(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install fixture exit=%d err=%s", code, errB.String())
	}
	inDir(t, repo, func() {
		out.Reset()
		if code := Main([]string{"new-roadmap", "adoption-v3"}, &out, &errB); code != 0 {
			t.Fatalf("new-roadmap exit=%d err=%s", code, errB.String())
		}
		roadmap, err := os.ReadFile(filepath.Join(repo, ".pose", "roadmaps", "adoption-v3.md"))
		if err != nil || !strings.Contains(string(roadmap), "slug: adoption-v3") || strings.Contains(string(roadmap), "<YYYY-MM-DD>") || strings.Contains(string(roadmap), "<created_at>") {
			t.Fatalf("roadmap artifact=%q err=%v", roadmap, err)
		}

		out.Reset()
		if code := Main([]string{"new-adr", "Structured Validation"}, &out, &errB); code != 0 {
			t.Fatalf("new-adr exit=%d err=%s", code, errB.String())
		}
		adrs, _ := filepath.Glob(filepath.Join(repo, ".pose", "adr", "*-structured-validation.md"))
		if len(adrs) != 1 {
			t.Fatalf("ADR artifact=%v", adrs)
		}

		out.Reset()
		if code := Main([]string{"new-knowledge", "handoff", "Release Readiness", "--ttl-days", "7", "--restricted"}, &out, &errB); code != 0 {
			t.Fatalf("new-knowledge exit=%d err=%s", code, errB.String())
		}
		knowledge, _ := filepath.Glob(filepath.Join(repo, ".pose", "knowledge", "*-handoff-release-readiness.md"))
		if len(knowledge) != 1 {
			t.Fatalf("knowledge artifact=%v", knowledge)
		}
		content, _ := os.ReadFile(knowledge[0])
		if !strings.Contains(string(content), "sensitivity: restricted") || strings.Contains(string(content), "<expires_at>") {
			t.Fatalf("knowledge content=%s", content)
		}
	})
}

func TestFollowupsNativeOpenAllAndJSON(t *testing.T) {
	repo := newGitRepo(t)
	for slug, body := range map[string]string{
		"one": "## 7. Final Report\n\n### Follow-ups\n- [open] investigate cache\n- [done] shipped\n",
		"two": "---\nstatus: in-progress\n---\n## 7. Final Report\n\n### Follow-ups\n- untriaged item\n- [covered: cache-spec] investigate cache behavior\n",
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
		if !strings.Contains(out.String(), `"total":4`) || !strings.Contains(out.String(), `"open":2`) || !strings.Contains(out.String(), "near_duplicate_candidates") {
			t.Fatalf("json output=%q", out.String())
		}
		out.Reset()
		Main([]string{"followups", "--all", "--json", "--similarity", "80"}, &out, &errB)
		if !strings.Contains(out.String(), `"target":"cache-spec"`) || !strings.Contains(out.String(), `"spec_status":"in-progress"`) || !strings.Contains(out.String(), `"total":4`) {
			t.Fatalf("all JSON missing parity fields: %q", out.String())
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
		history, err := os.ReadFile(filepath.Join(repo, ".pose", "reports", "history", "standard-native-report.jsonl"))
		if err != nil || !strings.Contains(string(history), `"outcome":"pass"`) {
			t.Fatalf("history=%q err=%v", history, err)
		}
		if code := Main([]string{"report", "--task", "x", "--outcome", "bad"}, &out, &errB); code != 2 {
			t.Fatalf("invalid outcome exit=%d", code)
		}
		if code := Main([]string{"report", "--task", "x", "--type", "../../escape"}, &out, &errB); code != 2 {
			t.Fatalf("unsafe report type exit=%d", code)
		}
		if code := Main([]string{"report", "--task", "x", "--validate-output", "../escape.log"}, &out, &errB); code != 2 {
			t.Fatalf("unsafe validation log exit=%d", code)
		}
		if code := Main([]string{"report", "--task", "x", "--unknown", "value"}, &out, &errB); code != 2 {
			t.Fatalf("unknown flag exit=%d", code)
		}
		logPath := filepath.Join(repo, "validate.log")
		if err := os.WriteFile(logPath, []byte("  -> go test ./...\nResult: FAILURE_TOLERATED\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		if code := Main([]string{"report", "--task", "native report", "--validate-output", logPath}, &out, &errB); code != 0 {
			t.Fatalf("derived report exit=%d err=%s", code, errB.String())
		}
		history, _ = os.ReadFile(filepath.Join(repo, ".pose", "reports", "history", "standard-native-report.jsonl"))
		if !strings.Contains(string(history), `"sequence":2`) || !strings.Contains(string(history), `"outcome":"partial"`) || !strings.Contains(string(history), `"outcome_source":"derived"`) {
			t.Fatalf("derived history=%s", history)
		}
	})
}

func TestParseStructuredChecks(t *testing.T) {
	checks, err := parseStructuredChecks([]byte(`{"stacks":{"go":{"checks":[{"name":"test","program":"go","args":["test","./..."],"env":{"GOCACHE":"/tmp/cache"}}]}}}`))
	if err != nil || len(checks) != 1 || checks[0].Program != "go" || checks[0].Env["GOCACHE"] != "/tmp/cache" {
		t.Fatalf("checks=%+v err=%v", checks, err)
	}
	if _, err := parseStructuredChecks([]byte(`{`)); err == nil {
		t.Fatal("malformed JSON accepted")
	}
}

func TestValidateNativeRunsStructuredChecksWithoutShell(t *testing.T) {
	repo := newGitRepo(t)
	module := filepath.Join(repo, "service")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.test/service\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	matrixDir := filepath.Join(repo, ".pose", "indexes")
	if err := os.MkdirAll(matrixDir, 0o755); err != nil {
		t.Fatal(err)
	}
	matrix := fmt.Sprintf(`{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"self","program":%q,"args":["-test.run=^$"] ,"severity":"required"}]},"contract":{"checks":[]}},"moduleOverrides":{"contracts":{"stack":"contract","checks":[{"name":"self","program":%q,"args":["-test.run=^$"],"severity":"required"}]}}}`, executable, executable)
	if err := os.WriteFile(filepath.Join(matrixDir, "validation-matrix.json"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"validate", "--module", "service"}, &out, &errB); code != 0 {
			t.Fatalf("validate exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if !strings.Contains(out.String(), "Result: SUCCESS") || strings.Contains(errB.String(), "deprecated script engine") {
			t.Fatalf("validate not native: out=%q err=%q", out.String(), errB.String())
		}
		out.Reset()
		if code := Main([]string{"validate", "--module", "contracts"}, &out, &errB); code != 0 || !strings.Contains(out.String(), "[module] contracts") {
			t.Fatalf("override-only module not discovered: exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if code := Main([]string{"validate", "--module", "../escape"}, &out, &errB); code != 2 {
			t.Fatalf("unsafe module exit=%d", code)
		}
		unsafeMatrix := fmt.Sprintf(`{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"unsafe","program":%q,"args":[],"severity":"required","when":{"fileExists":"../../secret"}}]}},"moduleOverrides":{}}`, executable)
		if err := os.WriteFile(filepath.Join(matrixDir, "validation-matrix.json"), []byte(unsafeMatrix), 0o644); err != nil {
			t.Fatal(err)
		}
		if code := Main([]string{"validate", "--module", "service"}, &out, &errB); code != 2 || !strings.Contains(errB.String(), "must remain inside") {
			t.Fatalf("unsafe when path accepted: exit=%d err=%s", code, errB.String())
		}
	})
}

func TestCheckNativeParityAndSchemaFailures(t *testing.T) {
	repo := newGitRepo(t)
	var installOut, installErr bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &installOut, &installErr); code != 0 {
		t.Fatalf("install fixture exit=%d out=%s err=%s", code, installOut.String(), installErr.String())
	}
	inDir(t, repo, func() {
		var nativeOut, nativeErr bytes.Buffer
		if code := Main([]string{"check", "--strict"}, &nativeOut, &nativeErr); code != 0 {
			t.Fatalf("native healthy exit=%d out=%s err=%s", code, nativeOut.String(), nativeErr.String())
		}
		if !strings.Contains(nativeOut.String(), "Resultado: SUCESSO") || strings.Contains(nativeErr.String(), "deprecated script engine") {
			t.Fatalf("check was not native/compatible: out=%q err=%q", nativeOut.String(), nativeErr.String())
		}
		matrixPath := filepath.Join(repo, ".pose", "indexes", "validation-matrix.json")
		raw, err := os.ReadFile(matrixPath)
		if err != nil {
			t.Fatal(err)
		}
		var matrix map[string]any
		if err := json.Unmarshal(raw, &matrix); err != nil {
			t.Fatal(err)
		}
		matrix["severty"] = "required"
		broken, _ := json.Marshal(matrix)
		if err := os.WriteFile(matrixPath, broken, 0o644); err != nil {
			t.Fatal(err)
		}
		nativeOut.Reset()
		if code := Main([]string{"check", "--strict"}, &nativeOut, &nativeErr); code != 1 || !strings.Contains(nativeOut.String(), "unknown key") {
			t.Fatalf("unknown matrix key accepted: exit=%d out=%s", code, nativeOut.String())
		}

		if err := os.Remove(filepath.Join(repo, ".pose", "schema-version")); err != nil {
			t.Fatal(err)
		}
		nativeOut.Reset()
		if code := Main([]string{"check", "--tolerant"}, &nativeOut, &nativeErr); code != 0 || !strings.Contains(nativeOut.String(), "[AVISO] schema:") {
			t.Fatalf("tolerant schema behavior: exit=%d out=%s", code, nativeOut.String())
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

func TestNativeCommandLocaleMessagesAndStableAnchors(t *testing.T) {
	old := os.Getenv("POSE_LOCALE")
	t.Cleanup(func() { _ = os.Setenv("POSE_LOCALE", old) })
	for _, tc := range []struct {
		locale, wantUsage, wantError, wantHelp, wantReport string
	}{
		{"en", "Usage: pose new-spec", "Error: provide <slug> or --all", "Usage: pose <command>", "Error: --task is required."},
		{"pt-BR", "Uso: pose new-spec", "Erro: informe <slug> ou --all", "Uso: pose <comando>", "Erro: --task é obrigatório."},
	} {
		_ = os.Setenv("POSE_LOCALE", tc.locale)
		var out, errB bytes.Buffer
		if code := cmdNewSpec(t.TempDir(), nil, &out, &errB); code != 2 || !strings.Contains(errB.String(), tc.wantUsage) {
			t.Fatalf("locale=%s scaffold exit=%d stderr=%q", tc.locale, code, errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := cmdLintSpec(nil, &out, &errB); code != 2 || !strings.Contains(errB.String(), tc.wantError) {
			t.Fatalf("locale=%s lint exit=%d stderr=%q", tc.locale, code, errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := Main([]string{"help"}, &out, &errB); code != 0 || !strings.Contains(out.String(), tc.wantHelp) {
			t.Fatalf("locale=%s help exit=%d out=%q", tc.locale, code, out.String())
		}
		out.Reset()
		errB.Reset()
		if code := cmdReport(t.TempDir(), nil, &out, &errB); code != 2 || !strings.Contains(errB.String(), tc.wantReport) {
			t.Fatalf("locale=%s report exit=%d stderr=%q", tc.locale, code, errB.String())
		}
		out.Reset()
		errB.Reset()
		if code := cmdCheck(t.TempDir(), []string{"--tolerant"}, &out, &errB); code != 1 || !strings.Contains(out.String(), "Resultado: FALHA") {
			t.Fatalf("locale=%s check anchor exit=%d out=%q", tc.locale, code, out.String())
		}
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
		// Minimal native instance + schema improves the result.
		if err := os.MkdirAll(filepath.Join(repo, ".pose"), 0o755); err != nil {
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
	for _, rel := range []string{".pose/schema-version", "AGENTS.md", ".pose/LICENSE"} {
		if _, err := os.Stat(filepath.Join(repo, rel)); err != nil {
			t.Errorf("missing after install: %s", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".pose", "scripts")); !os.IsNotExist(err) {
		t.Error("legacy .pose/scripts must not be installed")
	}
	if _, err := os.Stat(filepath.Join(repo, "pose")); !os.IsNotExist(err) {
		t.Error("legacy root dispatcher must not be installed")
	}
	if b, _ := os.ReadFile(filepath.Join(repo, ".agents", "skills", "pose-feature", "SKILL.md")); !strings.Contains(string(b), "Use to implement a non-trivial feature") {
		t.Error("default English skill not installed")
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
	if b, _ := os.ReadFile(filepath.Join(repo2, ".agents", "skills", "pose-feature", "SKILL.md")); !strings.Contains(string(b), "Use ao implementar") {
		t.Error("pt-BR skill overlay not applied")
	}
}

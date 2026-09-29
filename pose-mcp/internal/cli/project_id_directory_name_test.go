package cli

// A project that declares no id takes one from its directory name (spec
// project-id-from-any-directory-name). Test roots come from t.TempDir(), whose
// basenames ("001") are always valid slugs, so every existing test passed while
// `pose install` and `pose index` failed for any checkout named like "MyApp".
// These tests name the directory the way people do.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newNamedGitRepo(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

func unsetProjectEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"POSE_PROJECT_ROOT", "POSE_PROJECT_ROOTS", "POSE_DEFAULT_PROJECT_ID", "HARNE8_PROJECTS_DIR"} {
		old, had := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		if had {
			t.Cleanup(func() { _ = os.Setenv(key, old) })
		}
	}
}

func TestInstallAndIndexAcceptAnyDirectoryName(t *testing.T) {
	unsetProjectEnv(t)
	for name, want := range map[string]string{
		"MyApp":       "proj.myapp",
		"Acme Portal": "proj.acme-portal",
		"tmp.9Sz2E9":  "proj.tmp.9sz2e9",
		"my-app":      "proj.my-app",
	} {
		t.Run(name, func(t *testing.T) {
			root := newNamedGitRepo(t, name)
			var out, errB bytes.Buffer
			if code := cmdInstall([]string{root}, &out, &errB); code != 0 {
				t.Fatalf("install exit=%d out=%s err=%s", code, out.String(), errB.String())
			}
			raw, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
			if err != nil {
				t.Fatal(err)
			}
			if got := declaredMCPProjectID(raw); got != want {
				t.Fatalf(".mcp.json declares %q, want %q", got, want)
			}
			out.Reset()
			errB.Reset()
			if code := cmdIndex(root, nil, &out, &errB); code != 0 {
				t.Fatalf("index exit=%d out=%s err=%s", code, out.String(), errB.String())
			}
		})
	}
}

func TestIndexNamesWhyADeclaredProjectIDIsRefused(t *testing.T) {
	unsetProjectEnv(t)
	root := newNamedGitRepo(t, "MyApp")
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{root, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	// A declared id is never rewritten silently; the refusal says what to declare.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.MyApp")
	out.Reset()
	errB.Reset()
	if code := cmdIndex(root, nil, &out, &errB); code == 0 {
		t.Fatal("index accepted an invalid declared project id")
	}
	msg := out.String() + errB.String()
	for _, part := range []string{"POSE_DEFAULT_PROJECT_ID", `"proj.MyApp"`, `"proj.myapp"`} {
		if !strings.Contains(msg, part) {
			t.Fatalf("refusal does not name %s: %s", part, msg)
		}
	}
}

func TestDoctorRepairsAnInvalidStampedProjectID(t *testing.T) {
	unsetProjectEnv(t)
	root := newNamedGitRepo(t, "MyApp")
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{root}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	// What an engine before this fix stamped for a checkout named MyApp.
	path := filepath.Join(root, ".mcp.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := strings.Replace(string(raw), `"proj.myapp"`, `"proj.MyApp"`, 1)
	if stale == string(raw) {
		t.Fatalf("fixture did not stamp the legacy id: %s", raw)
	}
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	finding := doctorMCPFinding(t, root)
	if finding.Level != "warn" || finding.RemediationClass != remediationFixable || !strings.Contains(finding.Message, "proj.MyApp") {
		t.Fatalf("doctor did not flag the invalid id: %+v", finding)
	}
	if err := doctorFixRegistry["mcp.config"].apply(root); err != nil {
		t.Fatal(err)
	}
	fixed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := declaredMCPProjectID(fixed); got != "proj.myapp" {
		t.Fatalf("fix declared %q", got)
	}
	if finding := doctorMCPFinding(t, root); finding.Level != "ok" {
		t.Fatalf("doctor still flags the repaired id: %+v", finding)
	}
}

func doctorMCPFinding(t *testing.T, root string) doctorFinding {
	t.Helper()
	var report struct {
		Findings []doctorFinding `json:"findings"`
	}
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		_ = cmdDoctor([]string{"--json"}, &out, &errB)
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("doctor json: %v: %s %s", err, out.String(), errB.String())
		}
	})
	for _, f := range report.Findings {
		if f.Check == "mcp.config" {
			return f
		}
	}
	t.Fatal("doctor reported no mcp.config finding")
	return doctorFinding{}
}

func TestInstallRefusesAnInvalidExplicitProjectID(t *testing.T) {
	unsetProjectEnv(t)
	root := newNamedGitRepo(t, "app")
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{root, "--project-id", "proj.MyApp"}, &out, &errB); code != 2 {
		t.Fatalf("install exit=%d, want 2: %s %s", code, out.String(), errB.String())
	}
	if !strings.Contains(errB.String(), `"proj.myapp"`) {
		t.Fatalf("refusal does not name the valid form: %s", errB.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".pose")); !os.IsNotExist(err) {
		t.Fatalf("refused install wrote .pose/: %v", err)
	}
}

func TestReinstallReplacesAnIDThatNeverResolved(t *testing.T) {
	unsetProjectEnv(t)
	root := newNamedGitRepo(t, "MyApp")
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{root}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	// What an engine before this fix stamped for a checkout named MyApp.
	for _, name := range []string{".mcp.json", "AGENTS.md"} {
		path := filepath.Join(root, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(raw), "proj.myapp", "proj.MyApp")), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out.Reset()
	errB.Reset()
	if code := cmdInstall([]string{root, "--force"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall exit=%d err=%s", code, errB.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := declaredMCPProjectID(raw); got != "proj.myapp" {
		t.Fatalf("reinstall kept %q", got)
	}
	out.Reset()
	errB.Reset()
	if code := cmdIndex(root, nil, &out, &errB); code != 0 {
		t.Fatalf("index after reinstall: %s %s", out.String(), errB.String())
	}
}

func TestReinstallKeepsAValidDeclaredProjectID(t *testing.T) {
	unsetProjectEnv(t)
	root := newNamedGitRepo(t, "MyApp")
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{root, "--project-id", "proj.acme-core"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	out.Reset()
	errB.Reset()
	if code := cmdInstall([]string{root, "--force"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall exit=%d err=%s", code, errB.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := declaredMCPProjectID(raw); got != "proj.acme-core" {
		t.Fatalf("reinstall rewrote a valid declared id to %q", got)
	}
}

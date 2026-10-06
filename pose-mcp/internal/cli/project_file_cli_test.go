package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-project-identity-file.

func readProjectFileID(t *testing.T, repo string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repo, ".pose", "project.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.ProjectID
}

func TestProjectFileIsWrittenByInstallAndSeededByUpdate(t *testing.T) {
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	// The MCP binding is written, as in a normal install: it is where an
	// instance installed before the file existed declared its id.
	if code := cmdInstall([]string{repo, "--project-id", "proj.declared"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	if id := readProjectFileID(t, repo); id != "proj.declared" {
		t.Fatalf("install wrote project id %q", id)
	}

	// An instance installed before the file existed keeps the identity it
	// declared in .mcp.json: update recovers it instead of the directory name.
	if err := os.Remove(filepath.Join(repo, ".pose", "project.json")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errB.Reset()
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update exit=%d err=%s", code, errB.String())
	}
	if id := readProjectFileID(t, repo); id != "proj.declared" {
		t.Fatalf("update seeded project id %q, want the one AGENTS.md declares", id)
	}
	if !strings.Contains(out.String(), "proj.declared") {
		t.Fatalf("update did not say which identity it recorded:\n%s", out.String())
	}

	// Never overwritten.
	if err := os.WriteFile(filepath.Join(repo, ".pose", "project.json"), []byte(`{"schema_version":1,"project_id":"proj.kept"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("second update exit=%d", code)
	}
	if id := readProjectFileID(t, repo); id != "proj.kept" {
		t.Fatalf("update overwrote the declared identity with %q", id)
	}
}

func TestProjectFileIsReportedByDoctor(t *testing.T) {
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp", "--project-id", "proj.declared"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	finding, found := findDoctorFinding(runDoctorJSON(t, repo), "project.identity")
	if !found || finding.Level != "ok" || !strings.Contains(finding.Message, "proj.declared") {
		t.Fatalf("doctor does not report the declared identity: found=%v %+v", found, finding)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.elsewhere")
	finding, _ = findDoctorFinding(runDoctorJSON(t, repo), "project.identity")
	if finding.Level != "warn" || !strings.Contains(finding.Message, "proj.elsewhere") {
		t.Fatalf("doctor does not name a disagreeing binding: %+v", finding)
	}
}

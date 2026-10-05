package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-project-identity-file.

func projectFileRoot(t *testing.T, content string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "checkout-under-another-name")
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if content != "" {
		if err := os.WriteFile(filepath.Join(root, ".pose", "project.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	return root
}

func TestProjectFileDeclaresTheIdentity(t *testing.T) {
	root := projectFileRoot(t, `{"schema_version":1,"project_id":"proj.real","name":"Real"}`)
	if got := DefaultProjectID(root); got != "proj.real" {
		t.Fatalf("DefaultProjectID = %q, want the declared proj.real", got)
	}
	if !ProjectIdentityDeclared(root) {
		t.Fatal("an identity declared in .pose/project.json is not counted as declared")
	}
	_, id, err := EnvironmentArtifactResolver(root, "")
	if err != nil || id != "proj.real" {
		t.Fatalf("resolver id = %q err = %v", id, err)
	}

	undeclared := projectFileRoot(t, "")
	if DefaultProjectID(undeclared) != "proj.checkout-under-another-name" || ProjectIdentityDeclared(undeclared) {
		t.Fatal("without the file the directory name must still be the (undeclared) fallback")
	}
}

func TestProjectFileRefusesAConflictingBindingAndAMalformedFile(t *testing.T) {
	root := projectFileRoot(t, `{"schema_version":1,"project_id":"proj.real"}`)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.real")
	if _, _, err := EnvironmentArtifactResolver(root, ""); err != nil {
		t.Fatalf("an agreeing environment binding was refused: %v", err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.other")
	if _, _, err := EnvironmentArtifactResolver(root, ""); err == nil || !strings.Contains(err.Error(), "conflicting-project-binding") || !strings.Contains(err.Error(), "proj.other") || !strings.Contains(err.Error(), "proj.real") {
		t.Fatalf("a disagreeing environment binding was not refused with both values: %v", err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", `{"proj.bound":"`+strings.ReplaceAll(root, `\`, `\\`)+`"}`)
	if _, _, err := EnvironmentArtifactResolver(root, ""); err == nil || !strings.Contains(err.Error(), "conflicting-project-binding") {
		t.Fatalf("a disagreeing roots binding was not refused: %v", err)
	}

	for _, malformed := range []string{`{"schema_version":1,"project_id":"Not A Slug"}`, `{not json`, `{"schema_version":2,"project_id":"proj.real"}`} {
		bad := projectFileRoot(t, malformed)
		if _, _, err := EnvironmentArtifactResolver(bad, ""); err == nil || !strings.Contains(err.Error(), "invalid-project-id") {
			t.Fatalf("malformed %s was accepted: %v", malformed, err)
		}
	}
}

func TestProjectFileWriteNeverOverwrites(t *testing.T) {
	root := projectFileRoot(t, "")
	written, err := WriteProjectFile(root, "proj.first", "First")
	if err != nil || !written {
		t.Fatalf("first write: written=%v err=%v", written, err)
	}
	written, err = WriteProjectFile(root, "proj.second", "Second")
	if err != nil || written {
		t.Fatalf("a second write replaced the declared identity: written=%v err=%v", written, err)
	}
	if id, ok, _ := ReadProjectFile(root); !ok || id != "proj.first" {
		t.Fatalf("declared identity = %q", id)
	}
	if _, err := WriteProjectFile(projectFileRoot(t, ""), "Bad Id", "x"); err == nil {
		t.Fatal("an invalid id was written")
	}
}

func TestProjectFileRemovesTheDirectoryNameLimitation(t *testing.T) {
	root := projectFileRoot(t, `{"schema_version":1,"project_id":"proj.real"}`)
	snap := Store{Root: root}.CurrentObligationSnapshot()
	for _, limitation := range snap.Limitations {
		if strings.Contains(limitation, "directory name") {
			t.Fatalf("a declared identity still reports the directory-name fallback: %v", snap.Limitations)
		}
	}
	if snap.Project != "proj.real" {
		t.Fatalf("snapshot project = %q", snap.Project)
	}
	undeclared := projectFileRoot(t, "")
	snap = Store{Root: undeclared}.CurrentObligationSnapshot()
	found := false
	for _, limitation := range snap.Limitations {
		found = found || strings.Contains(limitation, ".pose/project.json")
	}
	if !found {
		t.Fatalf("the fallback limitation does not name the remedy: %v", snap.Limitations)
	}
}

func TestProjectFileIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "`.pose/project.json`") {
			t.Fatalf("%s does not document .pose/project.json", rel)
		}
	}
}

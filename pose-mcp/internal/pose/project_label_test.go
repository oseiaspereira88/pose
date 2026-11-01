package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func projectLabelOf(t *testing.T, dir, projectFile string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), dir)
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if projectFile != "" {
		if err := os.WriteFile(filepath.Join(root, ".pose", "project.json"), []byte(projectFile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Store{Root: root}.projectLabel()
}

func TestProjectLabelIsTheDeclaredName(t *testing.T) {
	for name, c := range map[string]struct{ file, want string }{
		"declared name":   {`{"schema_version":1,"project_id":"proj.pose-dist","name":"pose-dist"}`, "pose-dist"},
		"id without name": {`{"schema_version":1,"project_id":"proj.harne8"}`, "harne8"},
		"plain id":        {`{"schema_version":1,"project_id":"harne8"}`, "harne8"},
		"no file":         {"", "work-clone"},
		"malformed":       {`{not json`, "work-clone"},
		"other schema":    {`{"schema_version":9,"project_id":"proj.pose-dist","name":"pose-dist"}`, "work-clone"},
	} {
		if got := projectLabelOf(t, "work-clone", c.file); got != c.want {
			t.Errorf("%s: label %q, want %q", name, got, c.want)
		}
	}
}

// Every report that names the project, generated from a checkout under another name.
func TestReportsDoNotNameTheCheckout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "spec-worktree-1234")
	writeReviewFixture(t, root, ".pose/project.json", `{"schema_version":1,"project_id":"proj.real","name":"real"}`)
	writeReviewFixture(t, root, "api/go.mod", "module api\n\ngo 1.22\n")
	writeReviewFixture(t, root, "api/server.go", "package api\n")
	s := Store{Root: root}
	states, err := s.DiscoverAllComponents()
	if err != nil {
		t.Fatal(err)
	}
	if len(states) == 0 {
		t.Fatal("the fixture has no component, so no report names the project")
	}
	if err := s.GenerateAssessmentsIndex(states); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(s.AssessmentsDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(s.AssessmentsDir(), entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "spec-worktree-1234") {
			t.Errorf("%s names the checkout", entry.Name())
		}
	}
}

// The reports of this repository, as the engine regenerated them at closeout.
func TestCommittedAssessmentsNameTheProject(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	for _, rel := range []string{"README.md", "consolidated.md", "pose-mcp.md"} {
		raw, err := os.ReadFile(filepath.Join(root, ".pose", "assessments", rel))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "pose-delegation-roadmaps") || !strings.Contains(string(raw), "pose-dist") {
			t.Errorf("%s does not name the project", rel)
		}
	}
}

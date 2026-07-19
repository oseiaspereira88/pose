package cli

// Monorepo validation recipes (spec pose-monorepo-validation-recipes):
// docs-as-tests over pinned fixtures. Each test builds exactly the layout
// documented in docs-site/docs/monorepo-recipes.md and executes exactly the
// commands shown there — the recipe doc and this test must never drift.
// POSE composes with native build graphs via declared metadata; it does not
// implement a new orchestrator.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func recipeGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func recipeWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- Recipe 1: JavaScript workspace -----------------------------------------
// Root package.json declares npm/yarn workspaces; packages/app depends on
// packages/core. A change in core widens to its dependent app; a root-level
// change (e.g. the workspace manifest itself) runs the whole workspace.

func recipe1Fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	recipeWrite(t, root, "package.json", `{"name":"workspace-root","private":true,"workspaces":["packages/*"]}`)
	recipeWrite(t, root, "packages/core/package.json", `{"name":"core"}`)
	recipeWrite(t, root, "packages/app/package.json", `{"name":"app"}`)
	recipeWrite(t, root, ".pose/indexes/module-metadata.json", `{"schemaVersion":1,"modules":{
    "packages/app": {"dependsOn": ["packages/core"]}
  }}`)
	recipeWrite(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"stacks":{"node":{"checks":[
    {"name":"test","program":"true","severity":"required"}
  ]}}}`)
	recipeGit(t, root, "init", "-q")
	recipeGit(t, root, "add", "-A")
	recipeGit(t, root, "commit", "-qm", "base")
	return root
}

func TestRecipeJSWorkspaceDependencyWidening(t *testing.T) {
	root := recipe1Fixture(t)
	if err := os.WriteFile(filepath.Join(root, "packages/core/index.js"), []byte("module.exports = {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out := runValidate(t, root, "--changed-from", "HEAD", "--explain")
	if !strings.Contains(out, "+ packages/core: contains changed file") {
		t.Errorf("core must be directly selected: %s", out)
	}
	if !strings.Contains(out, "+ packages/app: depends on selected module packages/core") {
		t.Errorf("app must widen from core's dependency edge: %s", out)
	}
}

func TestRecipeJSWorkspaceRootManifestChangeRunsEverything(t *testing.T) {
	root := recipe1Fixture(t)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"workspace-root","private":true,"workspaces":["packages/*"],"version":"1.0.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out := runValidate(t, root, "--changed-from", "HEAD", "--explain")
	if !strings.Contains(out, "+ packages/app: root-level change") || !strings.Contains(out, "+ packages/core: root-level change") {
		t.Errorf("root manifest change must widen to the whole workspace: %s", out)
	}
}

// --- Recipe 2: declared dependency graph (Bazel-style fine-grained edges) --
// POSE does not read BUILD files; it composes with any build graph — real
// Bazel or otherwise — through declared dependsOn edges. A 3-hop chain
// (leaf -> mid -> base) proves transitive widening beyond one level.

func recipe2Fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, m := range []string{"base", "mid", "leaf"} {
		recipeWrite(t, root, m+"/go.mod", "module "+m+"\n")
	}
	recipeWrite(t, root, ".pose/indexes/module-metadata.json", `{"schemaVersion":1,"modules":{
    "mid": {"dependsOn": ["base"]},
    "leaf": {"dependsOn": ["mid"]}
  }}`)
	recipeWrite(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[
    {"name":"test","program":"true","severity":"required"}
  ]}}}`)
	recipeGit(t, root, "init", "-q")
	recipeGit(t, root, "add", "-A")
	recipeGit(t, root, "commit", "-qm", "base")
	return root
}

func TestRecipeDeclaredGraphTransitiveWidening(t *testing.T) {
	root := recipe2Fixture(t)
	if err := os.WriteFile(filepath.Join(root, "base/changed.go"), []byte("package base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runValidate(t, root, "--changed-from", "HEAD", "--explain", "--json", "result.json")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	for _, want := range []string{
		"+ base: contains changed file",
		"+ mid: depends on selected module base",
		"+ leaf: depends on selected module mid",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in transitive widening: %s", want, out)
		}
	}
	var run validationRunResult
	raw, _ := os.ReadFile(filepath.Join(root, "result.json"))
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	if run.Counts.Executed != 3 || run.Outcome != "pass" {
		t.Errorf("all three modules in the chain must run: %+v", run.Counts)
	}
}

// --- Recipe 3: mixed-language monorepo with a shared dependency ------------
// go + node + python modules; "shared" is criticality:high, so policy always
// includes it regardless of changed scope. Severity composes across stacks
// in one structured result (required go test vs optional node lint).

func recipe3Fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	recipeWrite(t, root, "services/api/go.mod", "module api\n")
	recipeWrite(t, root, "services/web/package.json", `{"name":"web"}`)
	recipeWrite(t, root, "services/worker/requirements.txt", "")
	recipeWrite(t, root, "shared/go.mod", "module shared\n")
	recipeWrite(t, root, ".pose/indexes/module-metadata.json", `{"schemaVersion":1,"modules":{
    "shared": {"criticality": "high"}
  }}`)
	recipeWrite(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"stacks":{
    "go": {"checks": [{"name":"test","program":"true","severity":"required"}]},
    "node": {"checks": [{"name":"lint","program":"true","severity":"optional"}]},
    "python": {"checks": [{"name":"pip-test","program":"true","severity":"required","when":{"fileExists":"requirements.txt"}}]}
  }}`)
	recipeGit(t, root, "init", "-q")
	recipeGit(t, root, "add", "-A")
	recipeGit(t, root, "commit", "-qm", "base")
	return root
}

func TestRecipeMixedLanguageSharedDependencyAlwaysIncluded(t *testing.T) {
	root := recipe3Fixture(t)
	// Only the web service changes; shared has no dependsOn edge to it, but
	// its declared criticality:high always includes it (policy widening).
	if err := os.WriteFile(filepath.Join(root, "services/web/index.js"), []byte("console.log(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runValidate(t, root, "--changed-from", "HEAD", "--explain", "--json", "result.json")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "+ services/web: contains changed file") {
		t.Errorf("web must be directly selected: %s", out)
	}
	if !strings.Contains(out, "+ shared: policy: criticality high always runs") {
		t.Errorf("shared dependency must always run by policy: %s", out)
	}
	if !strings.Contains(out, "- services/api: not affected") || !strings.Contains(out, "- services/worker: not affected") {
		t.Errorf("unrelated language modules must be skipped with a reason: %s", out)
	}
	var run validationRunResult
	raw, _ := os.ReadFile(filepath.Join(root, "result.json"))
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	byModule := map[string]checkResult{}
	for _, c := range run.Checks {
		byModule[c.Module] = c
	}
	if byModule["services/web"].Severity != "optional" || byModule["shared"].Severity != "required" {
		t.Errorf("severity must compose across stacks: web=%+v shared=%+v", byModule["services/web"], byModule["shared"])
	}
	if run.Counts.Skipped < 2 {
		t.Errorf("api and worker checks must be recorded as skipped, counts=%+v", run.Counts)
	}
}

func TestRecipeMixedLanguageStacksDetection(t *testing.T) {
	root := recipe3Fixture(t)
	for _, m := range []struct{ path, stack string }{
		{"services/api", "go"}, {"services/web", "node"}, {"services/worker", "python"}, {"shared", "go"},
	} {
		_, out := runStacksCmd(t, root, m.path)
		if !strings.Contains(out, "# "+m.stack) {
			t.Errorf("pose stacks --path %s should detect %s, got: %s", m.path, m.stack, out)
		}
	}
}

func runStacksCmd(t *testing.T, root, path string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := cmdStacks(root, []string{"--path", path}, &out, &errB)
	return code, out.String() + errB.String()
}

package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRuleExtensionGoAlwaysMatches(t *testing.T) {
	id, ok := resolveRuleExtension(t.TempDir(), "backend", "go")
	if !ok || id != "pose-rule-backend-go" {
		t.Fatalf("resolveRuleExtension(go) = (%q, %v), want (pose-rule-backend-go, true)", id, ok)
	}
}

func TestResolveRuleExtensionNodeWithReactMatches(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "web", "package.json"), `{"name":"web","dependencies":{"react":"^18.0.0"}}`)
	id, ok := resolveRuleExtension(root, "web", "node")
	if !ok || id != "pose-rule-frontend-react" {
		t.Fatalf("resolveRuleExtension(node+react) = (%q, %v), want (pose-rule-frontend-react, true)", id, ok)
	}
}

func TestResolveRuleExtensionNodeWithReactInDevDependenciesMatches(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "web", "package.json"), `{"name":"web","devDependencies":{"react":"^18.0.0"}}`)
	if _, ok := resolveRuleExtension(root, "web", "node"); !ok {
		t.Fatal("expected a match when react is only a devDependency")
	}
}

func TestResolveRuleExtensionNodeWithReactDoesNotMatch(t *testing.T) {
	root := t.TempDir()
	// A real Node.js backend — express, no react/vue/svelte — must never be
	// recommended the frontend rule.
	mustWrite(t, filepath.Join(root, "api", "package.json"), `{"name":"api","dependencies":{"express":"^4.0.0"}}`)
	if id, ok := resolveRuleExtension(root, "api", "node"); ok {
		t.Fatalf("resolveRuleExtension(node, no frontend) = (%q, true), want no match", id)
	}
}

func TestResolveRuleExtensionNodeWithVueMatches(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "web-vue", "package.json"), `{"name":"web-vue","dependencies":{"vue":"^3.4.0"}}`)
	id, ok := resolveRuleExtension(root, "web-vue", "node")
	if !ok || id != "pose-rule-frontend-vue" {
		t.Fatalf("resolveRuleExtension(node+vue) = (%q, %v), want (pose-rule-frontend-vue, true)", id, ok)
	}
}

func TestResolveRuleExtensionNodeWithSvelteMatches(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "web-svelte", "package.json"), `{"name":"web-svelte","devDependencies":{"@sveltejs/kit":"^2.0.0"}}`)
	id, ok := resolveRuleExtension(root, "web-svelte", "node")
	if !ok || id != "pose-rule-frontend-svelte" {
		t.Fatalf("resolveRuleExtension(node+svelte) = (%q, %v), want (pose-rule-frontend-svelte, true)", id, ok)
	}
}

func TestResolveRuleExtensionAllStacks(t *testing.T) {
	root := t.TempDir()
	expected := map[string]string{
		"go":                 "pose-rule-backend-go",
		"python":             "pose-rule-backend-python",
		"rust":               "pose-rule-backend-rust",
		"java":               "pose-rule-backend-java",
		"dotnet":             "pose-rule-backend-dotnet",
		"cloudflare-workers": "pose-rule-serverless-cloudflare",
	}
	for stack, wantID := range expected {
		id, ok := resolveRuleExtension(root, "mod", stack)
		if !ok || id != wantID {
			t.Errorf("resolveRuleExtension(%s) = (%q, %v), want (%q, true)", stack, id, ok, wantID)
		}
	}
}

func TestResolveRuleExtensionUnmappedStacksNeverMatch(t *testing.T) {
	root := t.TempDir()
	for _, stack := range []string{"ruby", "php", "elixir", "unknown-stack"} {
		if id, ok := resolveRuleExtension(root, "mod", stack); ok {
			t.Errorf("resolveRuleExtension(%s) = (%q, true), want no match — no extension authored for this stack yet", stack, id)
		}
	}
}

func TestDoctorRecommendsUnmatchedStackExtension(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "module-metadata.json"),
		`{"schemaVersion":1,"defaults":{},"modules":{"backend":{"domain":"go","criticality":"medium","validationProfile":"baseline"}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "rules.stack-extension-available")
	if !ok {
		t.Fatal("expected a rules.stack-extension-available finding")
	}
	if f.Level != "warn" {
		t.Errorf("rules.stack-extension-available level=%q, want warn", f.Level)
	}
	if !strings.Contains(f.Message, "pose-rule-backend-go") {
		t.Errorf("message does not name the matching extension: %q", f.Message)
	}
}

func TestDoctorSilentWhenRuleExtensionAlreadyInstalled(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "module-metadata.json"),
		`{"schemaVersion":1,"defaults":{},"modules":{"backend":{"domain":"go","criticality":"medium","validationProfile":"baseline"}}}`)
	mustWrite(t, filepath.Join(root, ".pose", "rules", "backend-go.md"), "# Rule: Backend Go\n")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "rules.stack-extension-available")
	if !ok {
		t.Fatal("expected a rules.stack-extension-available finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("rules.stack-extension-available level=%q, want ok — the rule is already installed", f.Level)
	}
}

func TestDoctorSilentForUnmappedStack(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "module-metadata.json"),
		`{"schemaVersion":1,"defaults":{},"modules":{"worker":{"domain":"ruby","criticality":"medium","validationProfile":"baseline"}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "rules.stack-extension-available")
	if !ok {
		t.Fatal("expected a rules.stack-extension-available finding (ok level)")
	}
	if f.Level != "ok" {
		t.Errorf("rules.stack-extension-available level=%q, want ok — no extension exists for ruby yet", f.Level)
	}
}

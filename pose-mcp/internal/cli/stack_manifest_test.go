package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStackForManifestFileRecognizesEveryMarker(t *testing.T) {
	cases := map[string]string{
		"package.json":     "node",
		"go.mod":           "go",
		"Cargo.toml":       "rust",
		"pom.xml":          "java",
		"build.gradle":     "java",
		"build.gradle.kts": "java",
		"pyproject.toml":   "python",
		"requirements.txt": "python",
		"Pipfile":          "python",
		"poetry.lock":      "python",
		"setup.py":         "python",
		"app.sln":          "dotnet",
		"App.csproj":       "dotnet",
		"App.fsproj":       "dotnet",
		"App.vbproj":       "dotnet",
		"wrangler.toml":    "cloudflare-workers",
		"wrangler.json":    "cloudflare-workers",
		"wrangler.jsonc":   "cloudflare-workers",
		"README.md":        "",
		"Makefile":         "",
		"Dockerfile":       "",
		"":                 "",
	}
	for name, want := range cases {
		if got := stackForManifestFile(name); got != want {
			t.Errorf("stackForManifestFile(%q) = %q, want %q", name, got, want)
		}
	}
}

// scanModules (pose index) and discoverValidationModules (pose validate/
// install/init) both classify markers through stackForManifestFile now
// (spec pose-validation-scanner-consolidation) — these tests exercise each
// public entry point directly so a regression in either caller is caught
// independently of the shared function's own unit test above.

func TestScanModulesDetectsSharedStacksAndTranslatesNodeLabel(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "web", "package.json"), `{"name":"web"}`)
	mustWrite(t, filepath.Join(root, "worker", "Cargo.toml"), "[package]\nname=\"worker\"\n")
	mustWrite(t, filepath.Join(root, "api", "pyproject.toml"), "[project]\nname=\"api\"\n")
	mustWrite(t, filepath.Join(root, "legacy", "pom.xml"), "<project></project>")
	mustWrite(t, filepath.Join(root, "modern-java", "build.gradle"), "")
	mustWrite(t, filepath.Join(root, "desktop", "App.csproj"), "<Project></Project>")
	mustWrite(t, filepath.Join(root, "edge", "wrangler.toml"), "name = \"edge\"\n")

	mods, _, _, _, _ := scanModules(root)
	byPath := map[string]string{}
	for _, m := range mods {
		byPath[m.Path] = m.Language
	}
	want := map[string]string{
		"web":         "javascript", // "node" translated for repo-map compatibility
		"worker":      "rust",
		"api":         "python",
		"legacy":      "java",
		"modern-java": "java",
		"desktop":     "dotnet",
		"edge":        "cloudflare-workers",
	}
	for path, lang := range want {
		if byPath[path] != lang {
			t.Errorf("module %s: language = %q, want %q (all: %+v)", path, byPath[path], lang, byPath)
		}
	}
}

func TestDiscoverValidationModulesDetectsCloudflareWorkers(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "edge-toml", "wrangler.toml"), "name = \"edge\"\n")
	mustWrite(t, filepath.Join(root, "edge-json", "wrangler.json"), `{"name":"edge"}`)
	mustWrite(t, filepath.Join(root, "edge-jsonc", "wrangler.jsonc"), `{"name":"edge"}`)

	mods, err := discoverValidationModules(root)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]string{}
	for _, m := range mods {
		byPath[m.Rel] = m.Stack
	}
	for _, path := range []string{"edge-toml", "edge-json", "edge-jsonc"} {
		if byPath[path] != "cloudflare-workers" {
			t.Errorf("module %s: stack = %q, want cloudflare-workers (all: %+v)", path, byPath[path], byPath)
		}
	}
}

func TestInstallSeedsModuleMetadataForCloudflareWorkersModule(t *testing.T) {
	repo := newGitRepo(t)
	mustWrite(t, filepath.Join(repo, "edge", "wrangler.toml"), "name = \"edge\"\n")

	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}

	raw, err := os.ReadFile(filepath.Join(repo, ".pose", "indexes", "module-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Modules map[string]map[string]string `json:"modules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	entry, ok := doc.Modules["edge"]
	if !ok {
		t.Fatalf("edge module not seeded: %+v", doc.Modules)
	}
	if entry["domain"] != "cloudflare-workers" {
		t.Errorf("edge module domain = %q, want cloudflare-workers", entry["domain"])
	}
	raw2, err := os.ReadFile(filepath.Join(repo, ".pose", "indexes", "validation-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		Stacks map[string]any `json:"stacks"`
	}
	if err := json.Unmarshal(raw2, &matrix); err != nil {
		t.Fatal(err)
	}
	if _, ok := matrix.Stacks["cloudflare-workers"]; !ok {
		t.Errorf("seeded validation-matrix.json has no cloudflare-workers stack entry: %+v", matrix.Stacks)
	}
}

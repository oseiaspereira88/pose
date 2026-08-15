package cli

// Module-metadata seeding from discovered stacks (spec
// pose-stack-detection-consolidation, github.com/oseiaspereira88/pose
// issue #21). A fresh install previously left module-metadata.json's
// modules map empty forever; nothing ever connected the engine's own
// stack discovery to it.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallSeedsModuleMetadataFromRealBrownfieldStacks(t *testing.T) {
	repo := newGitRepo(t)
	// A brownfield repo with three non-Go modules, exercising R3's
	// no-regression-vs-union-of-scanners requirement with real fixtures,
	// not synthetic single-file manifests.
	mustWrite(t, filepath.Join(repo, "web", "package.json"), `{"name":"web"}`)
	mustWrite(t, filepath.Join(repo, "worker", "Cargo.toml"), "[package]\nname=\"worker\"\n")
	mustWrite(t, filepath.Join(repo, "api", "pyproject.toml"), "[project]\nname=\"api\"\n")
	mustWrite(t, filepath.Join(repo, "legacy", "pom.xml"), "<project></project>")

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
	want := map[string]string{"web": "node", "worker": "rust", "api": "python", "legacy": "java"}
	for path, stack := range want {
		entry, ok := doc.Modules[path]
		if !ok {
			t.Errorf("module-metadata.json missing discovered module %q", path)
			continue
		}
		if entry["domain"] != stack {
			t.Errorf("module %q domain=%q, want %q", path, entry["domain"], stack)
		}
		if entry["criticality"] == "" || entry["validationProfile"] == "" {
			t.Errorf("module %q missing default criticality/validationProfile: %+v", path, entry)
		}
	}
	if len(doc.Modules) != len(want) {
		t.Errorf("module-metadata.json has %d modules, want exactly %d (no phantom entries): %+v", len(doc.Modules), len(want), doc.Modules)
	}
}

func TestInstallNeverOverwritesExistingModuleMetadataEntry(t *testing.T) {
	repo := newGitRepo(t)
	mustWrite(t, filepath.Join(repo, "web", "package.json"), `{"name":"web"}`)

	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}

	// Hand-edit the discovered entry, exactly as an operator would.
	metaPath := filepath.Join(repo, ".pose", "indexes", "module-metadata.json")
	raw, _ := os.ReadFile(metaPath)
	var doc struct {
		Modules map[string]map[string]string `json:"modules"`
	}
	json.Unmarshal(raw, &doc)
	doc.Modules["web"]["owner"] = "@web-team"
	doc.Modules["web"]["criticality"] = "high"
	edited, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(metaPath, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	// A second install run (e.g. pose update re-running install) must not
	// clobber the hand-authored fields.
	out.Reset()
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("re-run exit=%d\nout=%s\nerr=%s", code, out.String(), errB.String())
	}
	raw, _ = os.ReadFile(metaPath)
	doc.Modules = nil
	json.Unmarshal(raw, &doc)
	if doc.Modules["web"]["owner"] != "@web-team" || doc.Modules["web"]["criticality"] != "high" {
		t.Errorf("re-run overwrote hand-authored module-metadata entry: %+v", doc.Modules["web"])
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

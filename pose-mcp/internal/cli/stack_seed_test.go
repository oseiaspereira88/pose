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
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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

// TestUpdateSeedsComputedIndexesWithTargetOwnStateNotPoseDists regression-
// covers spec pose-derived-index-self-referential-leak: a plain `pose
// update` (no --force) that has to seed the cmdIndex-computed indexes
// (repo-map.json, spec-graph.json, ...) must end up with the *target's*
// own real state, not pose-dist's own baked graph the embedded scaffold
// used to ship verbatim in these files.
func TestUpdateSeedsComputedIndexesWithTargetOwnStateNotPoseDists(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	// Simulate an old instance: computed indexes and policy/review-profiles
	// absent, same shape TestDoctorFlagsIncompleteInstanceConfig uses.
	for _, rel := range []string{
		filepath.Join(".pose", "indexes", "repo-map.json"),
		filepath.Join(".pose", "indexes", "spec-graph.json"),
		filepath.Join(".pose", "indexes", "delivery-integrity.json"),
	} {
		if err := os.Remove(filepath.Join(repo, rel)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(filepath.Join(repo, ".pose", "policy")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".pose", "review-profiles")); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errB.Reset()
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update exit=%d err=%s out=%s", code, errB.String(), out.String())
	}

	raw, err := os.ReadFile(filepath.Join(repo, ".pose", "indexes", "spec-graph.json"))
	if err != nil {
		t.Fatal(err)
	}
	var graph struct {
		Specs map[string]any `json:"specs"`
	}
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	// This fixture repo has zero specs of its own — anything else means the
	// seeded content came from pose-dist's own graph instead of being
	// recomputed for this target.
	if len(graph.Specs) != 0 {
		t.Errorf("spec-graph.json has %d specs after a plain update on a spec-less fixture, want 0 (self-referential leak): %v", len(graph.Specs), graph.Specs)
	}
}

// TestUpdateSeedsEveryInstanceDirectoryNotOnlyAHandpickedFour regression-
// covers a live defect found updating a real old instance to v1.4.1: a
// plain `pose update` (no --force) only ever created 4 of the 14
// directories cmdInit's own instanceDirs contract requires (missing
// .pose/assessments among others), so an instance whose freshly-refreshed
// AGENTS.md already references .pose/assessments reported "Result:
// SUCCESS" and then failed its own very next `pose check --strict` with a
// broken reference.
func TestUpdateSeedsEveryInstanceDirectoryNotOnlyAHandpickedFour(t *testing.T) {
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	if err := os.RemoveAll(filepath.Join(repo, ".pose", "assessments")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".pose", "adr")); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errB.Reset()
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update exit=%d err=%s out=%s", code, errB.String(), out.String())
	}
	for _, rel := range []string{".pose/assessments", ".pose/adr"} {
		if fi, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel))); err != nil || !fi.IsDir() {
			t.Errorf("%s not recreated by a plain update: %v", rel, err)
		}
	}

	out.Reset()
	errB.Reset()
	if code := cmdCheck(repo, []string{"--strict"}, &out, &errB); code != 0 {
		t.Errorf("check --strict after update exit=%d out=%s", code, out.String())
	}
}

// TestModuleMetadataDiscoveryDoesNotDuplicateAnAliasedRoot regression-covers
// a live defect found updating a real external repository: a pre-existing
// module-metadata.json entry keyed by the project's own directory name
// (e.g. "acme" for a repo at .../acme), the common hand-curation
// convention for aliasing the project root, got a second, duplicate "."
// entry added alongside it the moment a go.mod (or any manifest) appeared
// at the root and discovery ran.
func TestModuleMetadataDiscoveryDoesNotDuplicateAnAliasedRoot(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "acme")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var out, errB bytes.Buffer
	if code := Main([]string{"install", repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}

	metaPath := filepath.Join(repo, ".pose", "indexes", "module-metadata.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Modules map[string]map[string]string `json:"modules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc.Modules["acme"] = map[string]string{"criticality": "medium", "domain": "go", "validationProfile": "baseline"}
	edited, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/acme\n\ngo 1.22\n")

	out.Reset()
	errB.Reset()
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update exit=%d err=%s out=%s", code, errB.String(), out.String())
	}

	raw, err = os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	doc.Modules = nil
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, dup := doc.Modules["."]; dup {
		t.Errorf("discovery added a duplicate \".\" entry alongside the pre-existing \"acme\" root alias: %+v", doc.Modules)
	}
	if _, kept := doc.Modules["acme"]; !kept {
		t.Error("the pre-existing \"acme\" root alias was removed, not just left alone")
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

func TestStampContractAdoptionRecordsTodayOnlyWhenAbsent(t *testing.T) {
	// The stamp exists because `.pose/policy/` is not machinery: an update
	// delivers a stricter engine and never a statement of when the instance
	// received it, so every closeout recorded under the previous contract
	// fails. Today is the honest date — work reviewed between adopting POSE
	// and receiving a contract was reviewed under the previous one.
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	path := func(root string) string { return filepath.Join(root, ".pose", "policy", "review.json") }
	adoptions := func(t *testing.T, root string) map[string]any {
		t.Helper()
		raw, err := os.ReadFile(path(root))
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		got, _ := doc["contract_adoptions"].(map[string]any)
		return got
	}
	write := func(t *testing.T, root, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path(root), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("every registered contract is stamped with today", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, `{"schema_version":2,"adopted_at":"2026-08-23"}`)
		stampContractAdoption(root, now, nil)
		got := adoptions(t, root)
		for _, contract := range posemodel.ReviewContracts() {
			// Not adopted_at: reviews recorded between adopting POSE and
			// receiving the contract were judged by the previous one.
			if got[contract.ID] != "2026-09-08" {
				t.Errorf("contract %s stamped %v, want today", contract.ID, got[contract.ID])
			}
		}
	})

	t.Run("a legacy field still counts as recorded", func(t *testing.T) {
		// A policy written before the registry keeps working and is never
		// rewritten into the map behind the project's back.
		root := t.TempDir()
		write(t, root, `{"schema_version":2,"component_aware":true,"component_aware_adopted_at":"2026-01-01"}`)
		stampContractAdoption(root, now, nil)
		if got := adoptions(t, root)["component-aware"]; got != nil {
			t.Errorf("re-stamped a contract the legacy field already records: %v", got)
		}
	})

	t.Run("an existing date is left alone", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, `{"schema_version":2,"contract_adoptions":{"evidence-vocabulary":"2026-01-01"}}`)
		stampContractAdoption(root, now, nil)
		if got := adoptions(t, root)["evidence-vocabulary"]; got != "2026-01-01" {
			t.Errorf("overwrote the instance's own date with %v", got)
		}
	})

	t.Run("an explicitly cleared date stays cleared", func(t *testing.T) {
		// A project that wants its whole history judged by the current
		// contract says so by emptying the value. Re-stamping would undo a
		// deliberate choice on every update.
		root := t.TempDir()
		write(t, root, `{"schema_version":2,"contract_adoptions":{"evidence-vocabulary":""}}`)
		stampContractAdoption(root, now, nil)
		if got := adoptions(t, root)["evidence-vocabulary"]; got != "" {
			t.Errorf("re-stamped a deliberately cleared date with %v", got)
		}
	})

	t.Run("keys the engine does not model survive", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, `{"schema_version":2,"acme_local_setting":"keep me"}`)
		stampContractAdoption(root, now, nil)
		raw, _ := os.ReadFile(path(root))
		var doc map[string]any
		if json.Unmarshal(raw, &doc) != nil || doc["acme_local_setting"] != "keep me" {
			t.Errorf("round-tripping dropped an unmodelled key: %s", raw)
		}
	})

	t.Run("a malformed policy is not rewritten", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, `{"schema_version":2,`)
		stampContractAdoption(root, now, nil)
		raw, _ := os.ReadFile(path(root))
		if string(raw) != `{"schema_version":2,` {
			t.Errorf("rewrote a file it could not parse: %s", raw)
		}
	})
}

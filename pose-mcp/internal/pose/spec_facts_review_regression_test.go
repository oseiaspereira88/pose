package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the second independent review of pose-spec-facts-evidence-binding.

const factsResultsPath = ".pose/results/delivery-validation.json"

// factsWithResultsFile builds an index whose results were read from the results file, as
// `pose index` does, and returns the store and a way to rewrite the file.
func factsWithResultsFile(t *testing.T, indexed []DeliveryValidationResult) (Store, func(edit func(run map[string]any))) {
	t.Helper()
	for i := range indexed {
		indexed[i].Report = factsResultsPath
		indexed[i].GitHead, indexed[i].GeneratedAt, indexed[i].ProvenanceDigest = "h1", "2026-10-18T00:00:00Z", "sha256:"+strings.Repeat("a", 64)
		if indexed[i].Severity == "" {
			indexed[i].Severity = "required"
		}
	}
	store := factsFixtureWith(t, "api", nil, false, []ChangeSet{{ID: "cs-alpha", Spec: "alpha", Head: "h1", ResolvedHead: "h1", Commits: []string{"c1"}, Paths: []ObservedPath{{Action: "modified", Path: "api/server.go"}}}})
	root := store.Root
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/policy/delivery.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-04","results_path":"`+factsResultsPath+`","roots":[]}`)
	graph := DeliveryIntegrityGraph{SchemaVersion: 1, ValidationResults: indexed,
		ChangeSets: []ChangeSet{{ID: "cs-alpha", Spec: "alpha", Head: "h1", ResolvedHead: "h1", Commits: []string{"c1"}, Paths: []ObservedPath{{Action: "modified", Path: "api/server.go"}}}}}
	scoped := ScopedDeliveryProvenanceDigest(graph, "alpha")
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].ScopeProvenance = map[string]string{"alpha": scoped}
	}
	rewrite := func(edit func(run map[string]any)) {
		checks := []map[string]any{}
		for _, r := range graph.ValidationResults {
			checks = append(checks, map[string]any{"id": r.ID, "module": r.Module, "name": r.Check, "severity": r.Severity, "evidence_class": r.EvidenceClass, "outcome": r.Outcome})
		}
		run := map[string]any{"schema_version": 1, "generated_at": "2026-10-18T00:00:00Z", "git_head": "h1", "provenance_digest": "sha256:" + strings.Repeat("a", 64),
			"scope_provenance": map[string]string{"alpha": scoped}, "checks": checks}
		if edit != nil {
			edit(run)
		}
		raw, _ := json.Marshal(run)
		write(factsResultsPath, string(raw))
	}
	rewrite(nil)
	raw, _ := json.Marshal(graph)
	for _, args := range [][]string{{"init", "-q"}, {"commit", "-q", "--allow-empty", "-m", "start"}} {
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	outputs := map[string][]byte{}
	for _, name := range IndexGenerationFiles {
		outputs[name] = []byte("{}\n")
	}
	outputs["delivery-integrity.json"] = raw
	if _, err := store.PublishIndexGeneration(outputs, IndexInputBindings(root)); err != nil {
		t.Fatal(err)
	}
	return store, rewrite
}

func statusOf(t *testing.T, store Store) SpecFactsFreshness {
	t.Helper()
	facts, err := store.DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	return facts.Freshness
}

// F01: the results file is compared with the results the index holds in every field they keep.
func TestFactsTheResultsFileIsComparedInEveryField(t *testing.T) {
	results := func() []DeliveryValidationResult {
		return []DeliveryValidationResult{{ID: "api/go/test", Module: "api", Check: "test", EvidenceClass: "unit", Outcome: "pass"}}
	}
	store, rewrite := factsWithResultsFile(t, results())
	if f := statusOf(t, store); f.Status != SpecFactsFresh {
		t.Fatalf("baseline: %+v", f)
	}
	for name, edit := range map[string]func(run map[string]any){
		"git head": func(run map[string]any) { run["git_head"] = "h2" },
		"scope": func(run map[string]any) {
			run["scope_provenance"] = map[string]string{"alpha": "sha256:" + strings.Repeat("b", 64)}
		},
		"schema":     func(run map[string]any) { run["schema_version"] = 999 },
		"module":     func(run map[string]any) { run["checks"].([]map[string]any)[0]["module"] = "web" },
		"check name": func(run map[string]any) { run["checks"].([]map[string]any)[0]["name"] = "other" },
		"class":      func(run map[string]any) { run["checks"].([]map[string]any)[0]["evidence_class"] = "integration" },
		"severity":   func(run map[string]any) { run["checks"].([]map[string]any)[0]["severity"] = "optional" },
		"extra result": func(run map[string]any) {
			run["checks"] = append(run["checks"].([]map[string]any), map[string]any{"id": "api/go/vet", "module": "api", "name": "vet", "severity": "required", "evidence_class": "lint", "outcome": "pass"})
		},
	} {
		rewrite(edit)
		if f := statusOf(t, store); f.Status == SpecFactsFresh {
			t.Errorf("%s: a results file that moved was called fresh", name)
		}
	}
	rewrite(nil)
	if f := statusOf(t, store); f.Status != SpecFactsFresh {
		t.Fatalf("restored: %+v", f)
	}
	// An index with no results and a results file that now has one.
	empty, rewriteEmpty := factsWithResultsFile(t, nil)
	rewriteEmpty(func(run map[string]any) {
		run["checks"] = []map[string]any{{"id": "api/go/test", "module": "api", "name": "test", "severity": "required", "evidence_class": "unit", "outcome": "pass"}}
	})
	if f := statusOf(t, empty); f.Status == SpecFactsFresh {
		t.Error("a result that appeared after an index with none was called fresh")
	}
	// A policy that cannot be loaded.
	os.WriteFile(filepath.Join(store.Root, ".pose", "policy", "delivery.json"), []byte("{"), 0o644)
	if f := statusOf(t, store); f.Status == SpecFactsFresh {
		t.Error("a policy that cannot be loaded was called fresh")
	}
}

// F02: the digest of nothing binds nothing.
func TestFactsTheDigestOfNothingIsNotAnAssociation(t *testing.T) {
	probe := []ChangeSet{}
	empty := DeliveryIntegrityGraph{SchemaVersion: 1, ChangeSets: probe}
	digest := ScopedDeliveryProvenanceDigest(empty, "alpha")
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "test", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": digest}}}
	facts, err := factsFixtureWith(t, "api", results, false, probe).DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Checks) != 0 || len(facts.Candidates) != 1 || !factsHasText(facts.Candidates[0].Limits, "no resolved change set") {
		t.Fatalf("the scope provenance of nothing was promoted to evidence: %+v", facts)
	}
	unresolved := []ChangeSet{{ID: "cs", Spec: "alpha", Commits: []string{"c"}}}
	graph := DeliveryIntegrityGraph{SchemaVersion: 1, ChangeSets: unresolved}
	results[0].ScopeProvenance["alpha"] = ScopedDeliveryProvenanceDigest(graph, "alpha")
	facts, _ = factsFixtureWith(t, "api", results, false, unresolved).DeriveSpecFacts("alpha")
	if len(facts.Checks) != 0 {
		t.Fatalf("a change set that is not resolved bound a result: %+v", facts.Checks)
	}
}

// F03: an omitted module is not the root of the repository.
func TestFactsAnOmittedModuleIsNotTheRoot(t *testing.T) {
	scoped := map[string]string{"alpha": "@scoped"}
	results := []DeliveryValidationResult{
		{ID: "none", Module: "", Check: "t", Outcome: "pass", ScopeProvenance: scoped},
		{ID: "blank", Module: "  ", Check: "t", Outcome: "pass", ScopeProvenance: scoped},
		{ID: "abs", Module: "/", Check: "t", Outcome: "pass", ScopeProvenance: scoped},
		{ID: "out", Module: "../x", Check: "t", Outcome: "pass", ScopeProvenance: scoped},
		{ID: "root", Module: ".", Check: "t", Outcome: "pass", ScopeProvenance: scoped},
	}
	for _, components := range []string{"api", "."} {
		facts, err := factsFixture(t, components, results, false).DeriveSpecFacts("alpha")
		if err != nil {
			t.Fatal(err)
		}
		associated := map[string]bool{}
		for _, c := range facts.Checks {
			associated[c.ID] = true
		}
		if !associated["root"] || associated["none"] || associated["blank"] || associated["abs"] || associated["out"] {
			t.Fatalf("components %q: associated = %v", components, associated)
		}
	}
}

// F04: results that differ in any field are two results, and the order of the index does not choose.
func TestFactsKeepResultsThatDifferInAnyField(t *testing.T) {
	good := map[string]string{"alpha": "@scoped"}
	for _, order := range []bool{false, true} {
		a := DeliveryValidationResult{ID: "api/go/t", Module: "api", Check: "t", EvidenceClass: "unit", Outcome: "pass", GitHead: "h1", ScopeProvenance: good}
		b := a
		b.ScopeProvenance = map[string]string{"alpha": "sha256:" + strings.Repeat("0", 64)}
		list := []DeliveryValidationResult{a, b}
		if order {
			list = []DeliveryValidationResult{b, a}
		}
		facts, err := factsFixture(t, "api", list, false).DeriveSpecFacts("alpha")
		if err != nil {
			t.Fatal(err)
		}
		if len(facts.Checks) != 1 || len(facts.Candidates) != 1 {
			t.Fatalf("order %v: one result was lost to the other: checks=%d candidates=%d", order, len(facts.Checks), len(facts.Candidates))
		}
	}
	for field, edit := range map[string]func(r *DeliveryValidationResult){
		"generated_at": func(r *DeliveryValidationResult) { r.GeneratedAt = "2026-10-19T00:00:00Z" },
		"class":        func(r *DeliveryValidationResult) { r.EvidenceClass = "integration" },
		"provenance":   func(r *DeliveryValidationResult) { r.ProvenanceDigest = "sha256:" + strings.Repeat("c", 64) },
	} {
		a := DeliveryValidationResult{ID: "api/go/t", Module: "api", Check: "t", EvidenceClass: "unit", Outcome: "pass", GitHead: "h1", ScopeProvenance: good}
		b := a
		edit(&b)
		facts, _ := factsFixture(t, "api", []DeliveryValidationResult{a, b}, false).DeriveSpecFacts("alpha")
		if len(facts.Checks) != 2 {
			t.Errorf("%s: two results that differ in it were collapsed: %d", field, len(facts.Checks))
		}
	}
}

// F05: the history is read the way the attribution reads it.
func TestFactsReadTheHistoryLikeTheAttribution(t *testing.T) {
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "t", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}}}
	for name, commit := range map[string]func(t *testing.T, root string){
		"extra spaces": func(t *testing.T, root string) {
			factsGit(t, root, "commit", "-q", "--allow-empty", "-m", "later", "-m", "POSE-Spec:   alpha  ")
		},
		"another ref": func(t *testing.T, root string) {
			factsGit(t, root, "checkout", "-q", "-b", "side")
			factsGit(t, root, "commit", "-q", "--allow-empty", "-m", "side", "-m", "POSE-Spec: alpha")
			factsGit(t, root, "checkout", "-q", "-")
		},
	} {
		store := factsFixture(t, "api", results, true)
		commit(t, store.Root)
		f, err := store.DeriveSpecFacts("alpha")
		if err != nil || f.Freshness.Status != SpecFactsStale || !strings.Contains(strings.Join(f.Freshness.StaleInputs, " "), "spec-commits") {
			t.Errorf("%s: a commit the attribution reads was missed: %+v %v", name, f.Freshness, err)
		}
	}
	// A project with no readable history is unknown, not fresh.
	store := factsFixture(t, "api", results, true)
	os.RemoveAll(filepath.Join(store.Root, ".git"))
	if f, err := store.DeriveSpecFacts("alpha"); err != nil || f.Freshness.Status != SpecFactsUnknown {
		t.Errorf("no history was called %+v %v", f.Freshness, err)
	}
}

// F06: a rename is both of its ends.
func TestFactsARenameIsBothOfItsEnds(t *testing.T) {
	sets := []ChangeSet{{ID: "cs", Spec: "alpha", Head: "h1", ResolvedHead: "h1", Commits: []string{"c"}, Paths: []ObservedPath{
		{Action: "renamed", OldPath: "api/a.go", NewPath: "web/a.go"},
		{Action: "renamed", OldPath: "web/b.go", NewPath: "docs/b.go"},
	}}}
	results := []DeliveryValidationResult{
		{ID: "api", Module: "api", Check: "t", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
		{ID: "web", Module: "web", Check: "t", Outcome: "fail", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
	}
	store := factsFixtureWith(t, "", results, false, sets)
	facts, err := store.DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Paths) != 2 {
		t.Fatalf("two renames collapsed: %+v", facts.Paths)
	}
	associated := map[string]bool{}
	for _, c := range facts.Checks {
		associated[c.ID] = true
	}
	if !associated["api"] || !associated["web"] {
		t.Fatalf("a check of either end of a rename was demoted: %v", associated)
	}
}

func factsGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func factsHasText(list []string, text string) bool {
	for _, item := range list {
		if strings.Contains(item, text) {
			return true
		}
	}
	return false
}

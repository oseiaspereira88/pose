package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-spec-facts-evidence-binding: facts separate what ties a result to a spec from what
// the index merely holds, and say how fresh the index is.

func factsFixture(t *testing.T, components string, results []DeliveryValidationResult, publish bool) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	comp := ""
	if components != "" {
		comp = "components: " + components + "\n"
	}
	write(".pose/specs/2026-10-01-alpha.md", "---\nslug: alpha\nstatus: in-progress\n"+comp+"---\n\n# Spec: alpha\n")
	graph := DeliveryIntegrityGraph{SchemaVersion: 1,
		ChangeSets:        []ChangeSet{{ID: "cs-alpha", Spec: "alpha", Head: "h1", ResolvedHead: "h1", Commits: []string{"c1"}, Paths: []ObservedPath{{Action: "modified", Path: "api/server.go"}}}},
		ValidationResults: results}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Root: root}
	if publish {
		outputs := map[string][]byte{}
		for _, name := range IndexGenerationFiles {
			outputs[name] = []byte("{}\n")
		}
		outputs["delivery-integrity.json"] = raw
		if _, err := store.PublishIndexGeneration(outputs, IndexInputBindings(root)); err != nil {
			t.Fatal(err)
		}
	} else {
		write(".pose/indexes/delivery-integrity.json", string(raw))
	}
	return store
}

func factNames(list []SpecFactCheck) []string {
	var out []string
	for _, c := range list {
		out = append(out, c.Module+"/"+c.Check)
	}
	return out
}

// V1 and V2: a result is evidence of the spec only if the index ties it to the spec.
func TestFactsSeparateAssociatedEvidenceFromCandidates(t *testing.T) {
	results := []DeliveryValidationResult{
		{ID: "1", Module: "api", Check: "go-test", EvidenceClass: "unit", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "change-set:cs-alpha"}},
		{ID: "2", Module: "web", Check: "lint", EvidenceClass: "lint", Outcome: "pass", GitHead: "h1"},
		{ID: "3", Module: "api", Check: "vet", EvidenceClass: "lint", Outcome: "pass", GitHead: "h1"},
		{ID: "4", Module: "other", Check: "secret-scan", EvidenceClass: "security", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"beta": "change-set:cs-beta"}},
	}
	// components empty: the module of a result says nothing about the spec.
	facts, err := factsFixture(t, "", results, false).DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factNames(facts.Checks), ","); got != "api/go-test" {
		t.Fatalf("associated = %s", got)
	}
	if got := strings.Join(factNames(facts.Candidates), ","); got != "api/vet,web/lint" {
		t.Fatalf("candidates = %s", got)
	}
	for _, c := range facts.Candidates {
		if c.Classification != SpecFactCandidate || c.Basis != "none" || len(c.Limits) == 0 {
			t.Errorf("a candidate must carry its limits: %+v", c)
		}
	}
	if !strings.Contains(strings.Join(facts.Limitations, " "), "left out") {
		t.Errorf("a result scoped to another spec is accounted for: %v", facts.Limitations)
	}
	// components declared: they name a module, they do not make its results evidence.
	facts, err = factsFixture(t, "api", results, false).DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	var vet SpecFactCheck
	for _, c := range facts.Candidates {
		if c.Check == "vet" {
			vet = c
		}
	}
	if vet.Basis != "components" || !strings.Contains(strings.Join(vet.Limits, " "), "declared by the spec") {
		t.Errorf("a declared module without provenance is a candidate with that limit: %+v", vet)
	}
}

// V4: a result that ran against another head is kept and labelled, not dropped or re-run.
func TestFactsKeepAResultOfAnotherHeadWithItsSource(t *testing.T) {
	results := []DeliveryValidationResult{
		{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "x"}},
		{ID: "2", Module: "api", Check: "integration", Outcome: "pass", GitHead: "h0", ProvenanceDigest: "sha256:aa", ScopeProvenance: map[string]string{"alpha": "x"}},
		{ID: "3", Module: "api", Check: "build", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "x"}},
	}
	facts, err := factsFixture(t, "", results, false).DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	heads := map[string]string{}
	for _, c := range facts.Checks {
		heads[c.Check] = c.Binding.Head
	}
	if heads["go-test"] != "same-head" || heads["integration"] != "other-head" || heads["build"] != "unknown-head" {
		t.Fatalf("heads = %v", heads)
	}
	for _, c := range facts.Checks {
		if c.Check == "integration" && (c.Binding.GitHead != "h0" || c.Binding.ProvenanceDigest != "sha256:aa" || len(c.Limits) == 0) {
			t.Errorf("the original source and the limit must be kept: %+v", c)
		}
	}
}

// V3: the index may be fresh, stale, unpublished or unreadable; none of these is a complete answer by default.
func TestFactsSayHowFreshTheIndexIs(t *testing.T) {
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "x"}}}
	fresh := factsFixture(t, "", results, true)
	facts, err := fresh.DeriveSpecFacts("alpha")
	if err != nil || facts.Freshness.Status != SpecFactsFresh || facts.Freshness.Generation == "" {
		t.Fatalf("fresh: %+v %v", facts.Freshness, err)
	}
	// An input of the index changes after it was published.
	if err := os.WriteFile(filepath.Join(fresh.Root, ".pose", "specs", "2026-10-02-beta.md"), []byte("---\nslug: beta\nstatus: draft\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	facts, err = fresh.DeriveSpecFacts("alpha")
	if err != nil || facts.Freshness.Status != SpecFactsStale || len(facts.Freshness.StaleInputs) == 0 || !strings.Contains(strings.Join(facts.Limitations, " "), "stale") || !strings.Contains(strings.Join(facts.Limitations, " "), "run `pose index`") {
		t.Fatalf("stale: %+v %v %v", facts.Freshness, err, facts.Limitations)
	}
	// Never published as a set.
	facts, err = factsFixture(t, "", results, false).DeriveSpecFacts("alpha")
	if err != nil || facts.Freshness.Status != SpecFactsUnknown || !strings.Contains(strings.Join(facts.Limitations, " "), "freshness is unknown") {
		t.Fatalf("unknown: %+v %v", facts.Freshness, err)
	}
	// A damaged published set is unavailable, not an empty delivery.
	damaged := factsFixture(t, "", results, true)
	gens, _ := filepath.Glob(filepath.Join(damaged.Root, ".pose", "indexes", "generations", "gen-*", "delivery-integrity.json"))
	if len(gens) != 1 {
		t.Fatalf("generation copies = %v", gens)
	}
	for _, copyPath := range []string{gens[0], filepath.Join(damaged.Root, ".pose", "indexes", "delivery-integrity.json")} {
		if err := os.WriteFile(copyPath, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if f, ferr := damaged.DeriveSpecFacts("alpha"); ferr == nil {
		t.Fatalf("a damaged index answered as if it were whole: %+v", f.Freshness)
	}
}

// V6: the facts are stable and reading them writes nothing.
func TestFactsAreStableAndReadOnly(t *testing.T) {
	results := []DeliveryValidationResult{
		{ID: "2", Module: "web", Check: "lint", Outcome: "pass"},
		{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "x"}},
		{ID: "3", Module: "api", Check: "build", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "x"}},
	}
	store := factsFixture(t, "", results, true)
	before := reviewFixtureSnapshot(t, store.Root)
	first, err := store.DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := store.DeriveSpecFacts("alpha")
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("two reads of the same index differ")
	}
	if got := strings.Join(factNames(first.Checks), ","); got != "api/build,api/go-test" {
		t.Fatalf("order = %s", got)
	}
	if after := reviewFixtureSnapshot(t, store.Root); len(after) != len(before) {
		t.Fatal("reading the facts changed the repository")
	}
	// R5: a passing associated check does not become a conclusion.
	raw, _ := json.Marshal(first)
	for _, forbidden := range []string{"satisfied", "approved", "rationale", "judgment", "accepted_risk"} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("the facts contain %q: a conclusion nobody drew", forbidden)
		}
	}
}

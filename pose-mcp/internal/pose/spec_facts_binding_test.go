package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-spec-facts-evidence-binding: facts separate what ties a result to a spec from what
// the index merely holds, and say how fresh the index is.

func factsFixture(t *testing.T, components string, results []DeliveryValidationResult, publish bool) Store {
	t.Helper()
	return factsFixtureWith(t, components, results, publish, []ChangeSet{{ID: "cs-alpha", Spec: "alpha", Head: "h1", ResolvedBase: "b0", ResolvedHead: "h1", DiffDigest: "sha256:d", Commits: []string{"c1"}, Paths: []ObservedPath{{Action: "modified", Path: "api/server.go"}}}})
}

// factsFixtureWith is factsFixture with the change sets the index holds for the spec.
func factsFixtureWith(t *testing.T, components string, results []DeliveryValidationResult, publish bool, changeSets []ChangeSet) Store {
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
		ChangeSets:        changeSets,
		ValidationResults: results}
	// "@scoped" stands for the scope provenance the spec's claims and change sets give.
	scoped := ScopedDeliveryProvenanceDigest(graph, "alpha")
	for i := range graph.ValidationResults {
		for slug, value := range graph.ValidationResults[i].ScopeProvenance {
			if value == "@scoped" {
				graph.ValidationResults[i].ScopeProvenance[slug] = scoped
			}
		}
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Root: root}
	if publish {
		// The history is an axis of freshness: a repository with a commit, as a project has.
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
		{ID: "1", Module: "api", Check: "go-test", EvidenceClass: "unit", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
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
		{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
		{ID: "2", Module: "api", Check: "integration", Outcome: "pass", GitHead: "h0", ProvenanceDigest: "sha256:aa", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
		{ID: "3", Module: "api", Check: "build", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
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
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "@scoped"}}}
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
	unpublished := factsFixture(t, "", results, false)
	for _, args := range [][]string{{"init", "-q"}, {"commit", "-q", "--allow-empty", "-m", "start"}} {
		cmd := exec.Command("git", append([]string{"-C", unpublished.Root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, gerr := cmd.CombinedOutput(); gerr != nil {
			t.Fatalf("git %v: %v %s", args, gerr, out)
		}
	}
	facts, err = unpublished.DeriveSpecFacts("alpha")
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
		{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
		{ID: "3", Module: "api", Check: "build", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
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

func factResult(facts SpecFacts, id string) (SpecFactCheck, bool) {
	for _, list := range [][]SpecFactCheck{facts.Checks, facts.Candidates} {
		for _, c := range list {
			if c.ID == id {
				return c, true
			}
		}
	}
	return SpecFactCheck{}, false
}

// F01: the key of a spec in a result's scope provenance is not the association; the digest
// it holds must be the one the spec's claims and change sets give.
func TestFactsAssociateOnlyAScopeProvenanceThatMatches(t *testing.T) {
	results := []DeliveryValidationResult{
		{ID: "ok", Module: "api", Check: "a", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "@scoped"}},
		{ID: "empty", Module: "api", Check: "b", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": ""}},
		{ID: "wrong", Module: "api", Check: "c", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "sha256:wrong"}},
		{ID: "zeros", Module: "api", Check: "d", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "sha256:" + strings.Repeat("0", 64)}},
	}
	facts, err := factsFixture(t, "api", results, false).DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factNames(facts.Checks), ","); got != "api/a" {
		t.Fatalf("associated = %s", got)
	}
	for _, id := range []string{"empty", "wrong", "zeros"} {
		c, ok := factResult(facts, id)
		if !ok || c.Classification != SpecFactCandidate || len(c.Limits) == 0 || c.Binding.ExpectedScopedDigest == "" {
			t.Errorf("%s must stay a candidate with its limit and the digest it was compared with: %+v", id, c)
		}
	}
}

// F02: a result of a module the spec does not cover is not associated, however its
// provenance reads: siblings, descendants, prefixes and aliases of paths.
func TestFactsAssociateOnlyAModuleThatCoversTheSpec(t *testing.T) {
	scoped := func() map[string]string { return map[string]string{"alpha": "@scoped"} }
	results := []DeliveryValidationResult{
		{ID: "same", Module: "api", Check: "t", Outcome: "pass", ScopeProvenance: scoped()},
		{ID: "web", Module: "web", Check: "t", Outcome: "pass", ScopeProvenance: scoped()},
		{ID: "sub", Module: "api/sub", Check: "t", Outcome: "pass", ScopeProvenance: scoped()},
		{ID: "prefix", Module: "api2", Check: "t", Outcome: "pass", ScopeProvenance: scoped()},
		{ID: "alias", Module: "api/../web", Check: "t", Outcome: "pass", ScopeProvenance: scoped()},
		{ID: "root", Module: ".", Check: "t", Outcome: "pass", ScopeProvenance: scoped()},
	}
	facts, err := factsFixture(t, "api", results, false).DeriveSpecFacts("alpha")
	if err != nil {
		t.Fatal(err)
	}
	associated := map[string]bool{}
	for _, c := range facts.Checks {
		associated[c.ID] = true
	}
	if !associated["same"] || !associated["root"] {
		t.Errorf("the covering modules must be associated: %v", associated)
	}
	for _, id := range []string{"web", "sub", "prefix", "alias"} {
		if associated[id] {
			t.Errorf("%s does not cover the spec and was associated", id)
		}
	}
}

// F04: two results that share a module and a check name are two results.
func TestFactsKeepEveryDistinctResult(t *testing.T) {
	scoped := func() map[string]string { return map[string]string{"alpha": "@scoped"} }
	for _, order := range [][]string{{"pass", "fail"}, {"fail", "pass"}} {
		var results []DeliveryValidationResult
		for i, outcome := range order {
			results = append(results, DeliveryValidationResult{ID: "api/s" + string(rune('a'+i)) + "/t", Module: "api", Check: "t", Outcome: outcome, ScopeProvenance: scoped()})
		}
		facts, err := factsFixture(t, "api", results, false).DeriveSpecFacts("alpha")
		if err != nil {
			t.Fatal(err)
		}
		outcomes := map[string]bool{}
		for _, c := range facts.Checks {
			outcomes[c.Outcome] = true
		}
		if len(facts.Checks) != 2 || !outcomes["fail"] || !outcomes["pass"] {
			t.Errorf("order %v lost a result: %+v", order, facts.Checks)
		}
	}
}

// F03: fresh is said only of what the generation binds and the index files are the ones it
// published; the results file it was built from is checked, and an unbound generation is unknown.
func TestFactsAreFreshOnlyForWhatIsVerified(t *testing.T) {
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", GitHead: "h1", ScopeProvenance: map[string]string{"alpha": "@scoped"}}}
	store := factsFixture(t, "api", results, true)
	if f, _ := store.DeriveSpecFacts("alpha"); f.Freshness.Status != SpecFactsFresh {
		t.Fatalf("baseline: %+v", f.Freshness)
	}
	// A legacy index file that differs from the generation it belongs to.
	legacy := filepath.Join(store.Root, ".pose", "indexes", "delivery-integrity.json")
	raw, _ := os.ReadFile(legacy)
	os.WriteFile(legacy, append(raw, ' '), 0o644)
	if f, err := store.DeriveSpecFacts("alpha"); err == nil && f.Freshness.Status == SpecFactsFresh {
		t.Errorf("a diverged index file was called fresh: %+v", f.Freshness)
	}
	os.WriteFile(legacy, raw, 0o644)
	// The results file the index was built from is replaced.
	policy := filepath.Join(store.Root, ".pose", "policy", "delivery.json")
	os.MkdirAll(filepath.Dir(policy), 0o755)
	os.WriteFile(policy, []byte(`{"schema_version":1,"enabled":true,"adopted_at":"2026-08-04","results_path":".pose/results/delivery-validation.json","roots":[]}`), 0o644)
	os.MkdirAll(filepath.Join(store.Root, ".pose", "results"), 0o755)
	os.WriteFile(filepath.Join(store.Root, ".pose", "results", "delivery-validation.json"), []byte(`{"schema_version":1,"generated_at":"2026-10-18T00:00:00Z","checks":[{"id":"1","outcome":"fail","evidence_class":"unit"}]}`), 0o644)
	// The policy is itself a bound input: take it out of the comparison by republishing.
	outputs := map[string][]byte{}
	for _, name := range IndexGenerationFiles {
		b, _ := os.ReadFile(filepath.Join(store.Root, ".pose", "indexes", name))
		outputs[name] = b
	}
	if _, err := store.PublishIndexGeneration(outputs, IndexInputBindings(store.Root)); err != nil {
		t.Fatal(err)
	}
	f, err := store.DeriveSpecFacts("alpha")
	if err != nil || f.Freshness.Status != SpecFactsStale || !strings.Contains(strings.Join(f.Freshness.StaleInputs, " "), "validation-results") {
		t.Errorf("a results file that disagrees with the index was called fresh: %+v %v", f.Freshness, err)
	}
}

// F03: an index generation that binds none of its inputs says nothing about freshness.
func TestFactsAGenerationThatBindsNothingIsUnknown(t *testing.T) {
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}}}
	store := factsFixture(t, "api", results, false)
	outputs := map[string][]byte{}
	for _, name := range IndexGenerationFiles {
		b, _ := os.ReadFile(filepath.Join(store.Root, ".pose", "indexes", name))
		if len(b) == 0 {
			b = []byte("{}\n")
		}
		outputs[name] = b
	}
	if _, err := store.PublishIndexGeneration(outputs, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if f, err := store.DeriveSpecFacts("alpha"); err != nil || f.Freshness.Status != SpecFactsUnknown || len(f.Freshness.Unbound) == 0 {
		t.Errorf("a generation without bindings was called fresh: %+v %v", f.Freshness, err)
	}
}

// F03: a commit that carries the spec's trailer after the index was built is not in the facts.
func TestFactsSeeACommitOfTheSpecThatTheIndexDoesNotHold(t *testing.T) {
	results := []DeliveryValidationResult{{ID: "1", Module: "api", Check: "go-test", Outcome: "pass", ScopeProvenance: map[string]string{"alpha": "@scoped"}}}
	store := factsFixture(t, "api", results, true)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", store.Root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	os.WriteFile(filepath.Join(store.Root, ".gitignore"), []byte(".pose/\n"), 0o644)
	git("add", ".gitignore")
	git("commit", "-q", "-m", "later work", "-m", "POSE-Spec: alpha")
	f, err := store.DeriveSpecFacts("alpha")
	if err != nil || f.Freshness.Status != SpecFactsStale || !strings.Contains(strings.Join(f.Freshness.StaleInputs, " "), "spec-commits") {
		t.Fatalf("a commit of the spec that the index lacks was called fresh: %+v %v", f.Freshness, err)
	}
}

package pose

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// attributionFixture commits, after a base, three changes to the paths of spec
// backend: its own dependency addition, another spec's addition to the same
// go.mod in between, and an ADR in a commit both specs claim. The change set
// attributes backend's two commits, as a trailer change set does.
func attributionFixture(t *testing.T) (string, Store) {
	t.Helper()
	root, store := structuralFixture(t, structuralBaseManifest, structuralBaseManifest)
	commit := func(message string, files ...string) string {
		for i := 0; i+1 < len(files); i += 2 {
			writeReviewFixture(t, root, files[i], files[i+1])
		}
		designDeltaGit(t, root, "add", "--", ".")
		designDeltaGit(t, root, "commit", "-q", "-m", message)
		return strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	}
	base := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	own := commit("add the backend dependency\n\nPOSE-Spec: backend",
		"api/go.mod", "module example.test/api\n\ngo 1.24\n\nrequire (\n\tgithub.com/kept/dep v1.0.0\n\tgithub.com/new/dep v2.0.0\n)\n")
	commit("another spec adds its dependency\n\nPOSE-Spec: other",
		"api/go.mod", "module example.test/api\n\ngo 1.24\n\nrequire (\n\tgithub.com/kept/dep v1.0.0\n\tgithub.com/new/dep v2.0.0\n\tgithub.com/other/dep v3.0.0\n)\n")
	shared := commit("a batch commit for two specs\n\nPOSE-Spec: backend\nPOSE-Spec: other",
		".pose/adr/2026-10-08-batch.md", "# ADR\n\nCarried by a batch commit.\n")
	graph := DeliveryIntegrityGraph{
		SchemaVersion: DeliveryIntegritySchemaVersion,
		ChangeSets: []ChangeSet{{
			ID: "cs-backend", Spec: "backend", Selector: "trailers:backend",
			ResolvedBase: base, ResolvedHead: shared, Commits: []string{own, shared},
			Paths: []ObservedPath{{Action: "modified", Path: "api/go.mod"}, {Action: "added", Path: ".pose/adr/2026-10-08-batch.md"}},
		}},
		Reverse: map[string][]string{"api/go.mod": {"backend"}},
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "index")
	return root, store
}

func attributionSubject(t *testing.T, store Store) ReviewBundleSubject {
	t.Helper()
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	subject, _, blockers, err := store.reviewBundleSubject(ScopeRef{Kind: "spec", Slug: "backend"}, []ReviewPlanComponent{{ID: "api", Path: "api"}}, graph, nil)
	if err != nil || len(blockers) > 0 {
		t.Fatalf("subject: %v %v", err, blockers)
	}
	return subject
}

func materialSubjects(t *testing.T, root string, subject ReviewBundleSubject) map[string]StructuralDelta {
	t.Helper()
	report, err := AssessDesignDelta(root, subject, "spec:backend", DesignDeltaOptions{})
	if err != nil {
		t.Fatal(err)
	}
	facts := map[string]StructuralDelta{}
	for _, fact := range report.Deltas {
		if StructuralDeltaIsMaterial(fact) {
			facts[fact.Kind+"/"+fact.Action+"/"+fact.Subject] = fact
		}
	}
	return facts
}

// R1: the structural comparison covers what the scope's own commits changed.
// Without the attribution the same subject reports the other spec's dependency,
// which is the false positive the field pilot measured.
func TestABMStructuralAttributionKeepsOtherSpecsOut(t *testing.T) {
	root, store := attributionFixture(t)
	subject := attributionSubject(t, store)
	if len(subject.Attribution) == 0 {
		t.Fatal("the sealed subject records no attribution")
	}
	scoped := materialSubjects(t, root, subject)
	if _, ok := scoped["dependency/added/go:github.com/new/dep"]; !ok {
		t.Fatalf("the scope's own dependency is not material: %v", keysOf(scoped))
	}
	if _, ok := scoped["dependency/added/go:github.com/other/dep"]; ok {
		t.Fatalf("another spec's dependency is reported as this scope's: %v", keysOf(scoped))
	}

	legacy := subject
	legacy.Attribution = nil
	ranged := materialSubjects(t, root, legacy)
	if _, ok := ranged["dependency/added/go:github.com/other/dep"]; !ok {
		t.Fatalf("the range comparison no longer shows what the fix removes, so this test proves nothing: %v", keysOf(ranged))
	}
}

// R2: a fact from a commit other specs also claim names them, the plan asks the
// reviewer to confirm it, and a fact from the scope's own commits does not.
func TestABMStructuralAttributionMarksSharedCommits(t *testing.T) {
	root, store := attributionFixture(t)
	facts := materialSubjects(t, root, attributionSubject(t, store))
	adr := StructuralDelta{}
	for key, fact := range facts {
		if strings.HasPrefix(key, "governance-contract/") {
			adr = fact
		}
	}
	if strings.Join(adr.SharedWith, ",") != "other" {
		t.Fatalf("the shared commit's fact does not name the other spec: %+v", facts)
	}
	if own := facts["dependency/added/go:github.com/new/dep"]; len(own.SharedWith) > 0 {
		t.Fatalf("a fact from the scope's own commit is marked shared: %+v", own)
	}

	adoptStructuralMateriality(t, store)
	plan := progressivePlan(t, store)
	if plan.Structure == nil {
		t.Fatal("the plan observed no structure")
	}
	marked := false
	for _, fact := range plan.Structure.Material {
		if fact.ID == adr.DisplayID && strings.Join(fact.SharedWith, ",") == "other" {
			marked = true
		}
	}
	warned := false
	for _, warning := range plan.Warnings {
		if strings.Contains(warning, adr.DisplayID) && strings.Contains(warning, "also attributed to other") {
			warned = true
		}
	}
	if !marked || !warned {
		t.Fatalf("the plan does not surface the shared fact (marked=%v warned=%v): %+v %v", marked, warned, plan.Structure.Material, plan.Warnings)
	}
}

// R3: the attribution names blobs, never commits, and leaves the
// implementation identity alone; a subject without it is compared as before.
func TestABMStructuralAttributionIsContentIdentity(t *testing.T) {
	root, store := attributionFixture(t)
	subject := attributionSubject(t, store)
	raw, err := json.Marshal(subject.Attribution)
	if err != nil {
		t.Fatal(err)
	}
	graph, _ := store.GetDeliveryIntegrity("")
	for _, commit := range graph.ChangeSets[0].Commits {
		if strings.Contains(string(raw), commit) {
			t.Fatalf("the attribution records commit %s, a provider ref: %s", commit, raw)
		}
	}
	blob := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", graph.ChangeSets[0].Commits[0]+":api/go.mod")))
	if !strings.Contains(string(raw), blob) {
		t.Fatalf("the own commit's go.mod blob %s is not a segment end: %s", blob, raw)
	}
	without := subject
	without.Attribution = nil
	if reviewImplementationDigest(without) != subject.ImplementationDigest {
		t.Fatal("the attribution changed the implementation digest")
	}
}

func keysOf[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// R4: the sealed bundle carries the attribution, and the replay counts the
// material facts that came from a commit other specs also claim.
func TestGovernanceReplayCountsSharedCommitFacts(t *testing.T) {
	_, store := attributionFixture(t)
	adoptStructuralMateriality(t, store)
	bundle, err := store.SealReviewBundle("spec:backend", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Payload.Subject.Attribution) == 0 {
		t.Fatal("the sealed subject records no attribution")
	}
	report, err := store.GovernanceReplay(0)
	if err != nil {
		t.Fatal(err)
	}
	if report.SharedCommitFacts != 1 || report.StructuralFacts < 2 {
		t.Fatalf("shared=%d structural=%d", report.SharedCommitFacts, report.StructuralFacts)
	}
}

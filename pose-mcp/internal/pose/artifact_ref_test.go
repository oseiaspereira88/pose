package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func qualifiedFile(t *testing.T, root, path, body string) {
	t.Helper()
	path = filepath.Join(root, ".pose", path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestQualifiedArtifactGrammar(t *testing.T) {
	for _, ref := range []string{"work", "spec:work", "roadmap:program", "milestone:program/first", "xref:proj.engine/work", "xref:proj.engine/spec:work", "xref:proj.engine/roadmap:program", "xref:proj.engine/milestone:program/first"} {
		parsed, err := ParseArtifactRef(ref)
		if err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
		roundtrip, err := ParseArtifactRef(parsed.String())
		if err != nil || roundtrip != parsed {
			t.Fatalf("roundtrip %s: %+v %v", ref, roundtrip, err)
		}
	}
	for _, ref := range []string{"", "../work", "xref:../work", "xref:engine/../../work", "xref:engine/spec:work/extra", "xref:engine/milestone:p/../bad", "xref:engine/alien:work", " work", "xref:engine/", strings.Repeat("a", 1025)} {
		if _, err := ParseArtifactRef(ref); err == nil {
			t.Errorf("accepted %q", ref)
		}
	}
}

func TestQualifiedArtifactLayoutsIdentityAndDigest(t *testing.T) {
	root := t.TempDir()
	for slug, path := range map[string]string{"flat": "flat.md", "dated": "2026-09-21-dated.md", "folder": "folder/spec.md", "dated-folder": "2026-09-21-dated-folder/spec.md"} {
		qualifiedFile(t, root, "specs/"+path, "---\nslug: "+slug+"\nstatus: done\n---\n# Work\n")
	}
	qualifiedFile(t, root, "specs/split/intent.md", "# Intent\nWork\n")
	qualifiedFile(t, root, "specs/split/STATUS.md", "done\n")
	qualifiedFile(t, root, "specs/2026-09-21-dated-split/intent.md", "# Intent\nWork\n")
	qualifiedFile(t, root, "specs/2026-09-21-dated-split/STATUS.md", "done\n")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %s %v", out, err)
	}
	if out, err := exec.Command("git", "-C", root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "fixture").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %s %v", out, err)
	}
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"proj.stable": root}})}
	list, err := (Store{Root: root}).ListSpecs("", "")
	if err != nil || len(list) != 6 {
		t.Fatalf("listing: %+v %v", list, err)
	}
	for _, sp := range list {
		resolved := r.Resolve("proj.stable", sp.Slug)
		if !resolved.Resolved || resolved.Status != "done" || len(resolved.Revision) != 40 || len(resolved.Digest) != 64 {
			t.Fatalf("%s: %+v", sp.Slug, resolved)
		}
		if resolved.Identity.String() != "xref:proj.stable/spec:"+sp.Slug {
			t.Fatal(resolved.Identity)
		}
	}
	before := r.Resolve("", "xref:proj.stable/flat")
	qualifiedFile(t, root, "specs/flat.md", "---\nslug: flat\nstatus: draft\n---\n# Work\n")
	after := r.Resolve("", "xref:proj.stable/flat")
	if before.Digest == after.Digest {
		t.Fatal("lifecycle edit did not change source digest")
	}
	newRoot := root + "-renamed"
	if err := os.Rename(root, newRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Rename(newRoot, root) })
	r.Roots = NewRoots(RootsConfig{Explicit: map[string]string{"proj.stable": newRoot}})
	if got := r.Resolve("", "xref:proj.stable/flat"); !got.Resolved || got.Identity != after.Identity || got.Digest != after.Digest {
		t.Fatalf("relocation: %+v", got)
	}
}

func TestQualifiedArtifactAuthorizationNoFallbackAndConfinement(t *testing.T) {
	parent, other := t.TempDir(), t.TempDir()
	qualifiedFile(t, parent, "specs/same.md", "---\nslug: same\nstatus: done\n---\n")
	qualifiedFile(t, other, "specs/same.md", "---\nslug: same\nstatus: draft\n---\n")
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: parent, DefaultProjectID: "parent", Explicit: map[string]string{"child": other, "gone": filepath.Join(other, "absent")}})}
	for ref, state := range map[string]string{"xref:unknown/same": "unknown-project", "xref:gone/same": "unavailable-project", "xref:child/missing": "unknown-spec"} {
		if got := r.Resolve("parent", ref); got.Resolved || got.State != state {
			t.Fatalf("%s: %+v", ref, got)
		}
	}
	r.Authorize = func(id string) bool { return id == "parent" }
	for _, id := range []string{"child", "unknown"} {
		got := r.Resolve("parent", "xref:"+id+"/same")
		raw, _ := json.Marshal(got)
		if got.State != "unauthorized-project" || got.Status != "" || strings.Contains(string(raw), other) {
			t.Fatalf("disclosure: %s", raw)
		}
	}
	if err := os.Symlink(filepath.Join(other, ".pose", "specs", "same.md"), filepath.Join(parent, ".pose", "specs", "escape.md")); err != nil {
		t.Fatal(err)
	}
	if got := r.Resolve("parent", "escape"); got.Resolved {
		t.Fatalf("escaped root: %+v", got)
	}
	qualifiedFile(t, parent, "specs/2026-09-21-same.md", "---\nslug: same\nstatus: draft\n---\n")
	if got := r.Resolve("parent", "same"); got.State != "conflicting-artifact-identity" {
		t.Fatalf("duplicate: %+v", got)
	}
}

func TestQualifiedArtifactGraphAndReadiness(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	qualifiedFile(t, a, "specs/consumer.md", "---\nslug: consumer\nstatus: draft\ndepends_on: xref:b/spec:producer\n---\n")
	qualifiedFile(t, b, "specs/producer.md", "---\nslug: producer\nstatus: done\n---\n")
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"a": a, "b": b}})}
	store := Store{Root: a}
	if got, err := store.SpecReadinessWithResolver("consumer", "a", r); err != nil || !got.Ready {
		t.Fatalf("ready: %+v %v", got, err)
	}
	qualifiedFile(t, b, "specs/producer.md", "---\nslug: producer\nstatus: done\ndepends_on: xref:a/consumer\n---\n")
	if reason := r.ValidateGraph("a", "consumer"); reason != "dependency-cycle" {
		t.Fatalf("cycle: %s", reason)
	}
	if got, err := store.SpecReadinessWithResolver("consumer", "a", r); err != nil || got.Ready || got.WaitingOn[0].Reason != "dependency-cycle" {
		t.Fatalf("cycle readiness: %+v %v", got, err)
	}
	for i := 0; i < 70; i++ {
		qualifiedFile(t, a, fmt.Sprintf("specs/n%d.md", i), fmt.Sprintf("---\nslug: n%d\nstatus: done\ndepends_on: n%d\n---\n", i, i+1))
	}
	if reason := r.ValidateGraph("a", "n0"); reason != "resolution-limit" {
		t.Fatalf("limit: %s", reason)
	}
}

// Synthetic graphs use the production parser/resolver, not a corpus parser.
func artifactGraphFixture(t *testing.T, graph map[string][]string) ArtifactResolver {
	t.Helper()
	roots := map[string]string{}
	for raw, deps := range graph {
		ref, err := ParseArtifactRef(raw)
		if err != nil || ref.Project == "" || ref.Kind != "spec" {
			t.Fatalf("invalid fixture node %q: %v", raw, err)
		}
		if roots[ref.Project] == "" {
			roots[ref.Project] = t.TempDir()
		}
		qualifiedFile(t, roots[ref.Project], "specs/"+ref.Slug+".md",
			fmt.Sprintf("---\nslug: %s\nstatus: done\ndepends_on: %s\n---\n", ref.Slug, strings.Join(deps, ", ")))
	}
	return (ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: roots}), governanceDepthLimit: 8}).WithSpecSnapshots()
}

func artifactGraphChain(edges int, project func(int) string) (map[string][]string, []string) {
	names := make([]string, edges+1)
	for i := range names {
		names[i] = fmt.Sprintf("xref:%s/spec:n%d", project(i), i)
	}
	graph := map[string][]string{}
	for i, name := range names {
		graph[name] = nil
		if i < edges {
			graph[name] = []string{names[i+1]}
		}
	}
	return graph, names
}

func TestArtifactGraphGovernanceLocalAndFederatedDepth(t *testing.T) {
	local := func(int) string { return "a" }
	crossing := func(i int) string { return []string{"a", "b"}[i%2] }
	for _, tc := range []struct {
		name    string
		edges   int
		project func(int) string
		caller  string
		want    string
	}{
		{"local-over-eight", 12, local, "a", ""},
		{"local-boundary", 64, local, "a", ""},
		{"local-excess", 65, local, "a", "resolution-limit"},
		{"federated-boundary", 8, crossing, "a", ""},
		{"federated-excess", 9, crossing, "a", "resolution-limit"},
		{"initial-crossing-boundary", 7, crossing, "b", ""},
		{"initial-crossing-excess", 8, crossing, "b", "resolution-limit"},
		{"local-segments-between-crossings", 32, func(i int) string { return crossing(i / 4) }, "a", ""},
		{"excess-crossings-with-local-segments", 36, func(i int) string { return crossing(i / 4) }, "a", "resolution-limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph, names := artifactGraphChain(tc.edges, tc.project)
			r := artifactGraphFixture(t, graph)
			if got := r.ValidateGraph(tc.caller, names[0]); got != tc.want {
				t.Fatalf("ValidateGraph = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestArtifactGraphSharedDAGChecksLongestPath(t *testing.T) {
	for _, federated := range []bool{false, true} {
		for _, excess := range []bool{false, true} {
			t.Run(fmt.Sprintf("federated=%t/excess=%t", federated, excess), func(t *testing.T) {
				edges := 64
				project := func(int) string { return "a" }
				if federated {
					edges = 8
					project = func(i int) string { return []string{"a", "b"}[i%2] }
				}
				if excess {
					edges++
				}
				graph, names := artifactGraphChain(edges, project)
				// Resolve the shared suffix via a shortcut before the longer path.
				graph[names[0]] = []string{names[4], names[1]}
				r := artifactGraphFixture(t, graph)
				want := ""
				if excess {
					want = "resolution-limit"
				}
				if got := r.ValidateGraph("a", names[0]); got != want {
					t.Fatalf("shared suffix concealed longest path: got %q, want %q", got, want)
				}
			})
		}
	}
}

func TestArtifactGraphDAGUniqueNodeBudgetAndCycles(t *testing.T) {
	for _, count := range []int{256, 257} {
		t.Run(fmt.Sprintf("nodes=%d", count), func(t *testing.T) {
			graph, names := artifactGraphChain(count-1, func(int) string { return "a" })
			graph[names[0]] = names[1:]
			for _, name := range names[1:] {
				graph[name] = nil
				if name != names[len(names)-1] {
					graph[name] = []string{names[len(names)-1]}
				}
			}
			r := artifactGraphFixture(t, graph)
			calls := 0
			r.Authorize = func(string) bool {
				calls++
				return true
			}
			want := ""
			if count > 256 {
				want = "resolution-limit"
			}
			if got := r.ValidateGraph("a", names[0]); got != want {
				t.Fatalf("ValidateGraph = %q, want %q", got, want)
			}
			if calls != 256 {
				t.Fatalf("resolved %d nodes; shared DAG must resolve exactly 256 unique nodes before success/overflow", calls)
			}
		})
	}
	for _, federated := range []bool{false, true} {
		t.Run(fmt.Sprintf("cycle/federated=%t", federated), func(t *testing.T) {
			project := func(int) string { return "a" }
			if federated {
				project = func(i int) string { return []string{"a", "b"}[i%2] }
			}
			graph, names := artifactGraphChain(4, project)
			graph[names[4]] = []string{names[0]}
			if got := artifactGraphFixture(t, graph).ValidateGraph("a", names[0]); got != "dependency-cycle" {
				t.Fatalf("cycle = %q", got)
			}
		})
	}
}

func TestArtifactGraphLayeredDAGResolvesEachNodeOnce(t *testing.T) {
	graph, names := artifactGraphChain(48, func(int) string { return "a" })
	graph[names[0]] = names[1:3]
	for i := 1; i < len(names); i++ {
		next := ((i-1)/2+1)*2 + 1
		graph[names[i]] = nil
		if next < len(names) {
			graph[names[i]] = names[next : next+2]
		}
	}
	r := artifactGraphFixture(t, graph)
	calls := 0
	r.Authorize = func(string) bool {
		calls++
		return true
	}
	if got := r.ValidateGraph("a", names[0]); got != "" || calls != len(names) {
		t.Fatalf("layered DAG = %q, resolved %d nodes, want %d", got, calls, len(names))
	}
}

func TestArtifactGraphLegacyFederatedDepth(t *testing.T) {
	graph, names := artifactGraphChain(12, func(i int) string { return []string{"a", "b"}[i%2] })
	r := artifactGraphFixture(t, graph)
	r.governanceDepthLimit = 0
	if got := r.ValidateGraph("a", names[0]); got != "" {
		t.Fatalf("legacy resolver acquired B34's additional bound: %q", got)
	}
}

func TestArtifactGraphDependencyListBudget(t *testing.T) {
	graph := map[string][]string{
		"xref:a/spec:root": make([]string, 257),
		"xref:a/spec:leaf": nil,
	}
	for i := range graph["xref:a/spec:root"] {
		graph["xref:a/spec:root"][i] = "xref:a/spec:leaf"
	}
	// Repeated references still consume the per-artifact input edge budget.
	if got := artifactGraphFixture(t, graph).ValidateGraph("a", "root"); got != "resolution-limit" {
		t.Fatalf("oversized dependency list = %q", got)
	}
}

func TestQualifiedArtifactProjectBindingConflicts(t *testing.T) {
	if _, err := ParseRootsJSON(`{"p":"/one","p":"/two"}`); err == nil {
		t.Fatal("duplicate bindings accepted")
	}
	if _, err := ParseRootsJSON(`{"p":"relative"}`); err == nil {
		t.Fatal("relative root accepted")
	}
	base, defaultRoot := t.TempDir(), t.TempDir()
	qualifiedFile(t, filepath.Join(base, "p"), "specs/work.md", "---\nslug: work\nstatus: done\n---\n")
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: defaultRoot, DefaultProjectID: "p", ProjectsDir: base})}
	if got := r.Resolve("p", "work"); got.State != "conflicting-project-binding" {
		t.Fatal(got)
	}
}

func TestQualifiedArtifactTypedRoadmapsAndContractNegotiation(t *testing.T) {
	root := t.TempDir()
	qualifiedFile(t, root, "specs/work.md", "---\nslug: work\nstatus: done\n---\n")
	qualifiedFile(t, root, "roadmaps/program.md", "---\nslug: program\nstatus: done\n---\n## Milestone: first\n- specs: work\n")
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"owner": root}})}
	for _, ref := range []string{"xref:owner/roadmap:program", "xref:owner/milestone:program/first"} {
		if got := r.Resolve("", ref); !got.Resolved || got.Status != "done" || got.Digest == "" {
			t.Fatalf("%s: %+v", ref, got)
		}
	}
	qualifiedFile(t, root, "specs/work.md", "---\nslug: work\nstatus: done\ndepends_on: roadmap:program\n---\n")
	if reason := r.ValidateGraph("owner", "roadmap:program"); reason != "dependency-cycle" {
		t.Fatalf("mixed spec/milestone/roadmap cycle: %s", reason)
	}
	qualifiedFile(t, root, "specs/work.md", "---\nslug: work\nstatus: done\n---\n")
	store := Store{Root: root}
	for _, policy := range []string{`{"schema_version":2,"qualified_artifact_refs_version":1}`, `{"schema_version":3,"qualified_artifact_refs_version":2}`, `{"schema_version":3}`} {
		if _, err := store.parseReviewPolicy([]byte(policy)); err == nil {
			t.Fatalf("unsupported policy accepted: %s", policy)
		}
		qualifiedFile(t, root, "policy/review.json", policy)
		if got := r.Resolve("owner", "work"); got.State != "unsupported-artifact-contract" {
			t.Fatalf("resolver ignored unsupported metadata: %+v", got)
		}
	}
	policy := `{"schema_version":3,"qualified_artifact_refs_version":1}`
	if _, err := store.parseReviewPolicy([]byte(policy)); err != nil {
		t.Fatal(err)
	}
	qualifiedFile(t, root, "policy/review.json", policy)
	if got := r.Resolve("owner", "work"); !got.Resolved {
		t.Fatalf("adopted contract: %+v", got)
	}
	// The existing schema-1/2 consumer's version guard must reject the adoption.
	var oldHeader struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal([]byte(policy), &oldHeader); err != nil {
		t.Fatal(err)
	}
	if oldHeader.SchemaVersion == ReviewSchemaVersion || oldHeader.SchemaVersion == ReviewPolicySchemaVersion {
		t.Fatal("adoption could be silently accepted by the old consumer")
	}
}

func TestSpecAuthorityTransferPolicySchemaIsExplicitAndBackwardCompatible(t *testing.T) {
	store := Store{Root: t.TempDir()}
	for _, raw := range []string{
		`{"schema_version":1}`,
		`{"schema_version":2}`,
		`{"schema_version":3,"qualified_artifact_refs_version":1}`,
		`{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1}`,
	} {
		if _, err := store.parseReviewPolicy([]byte(raw)); err != nil {
			t.Errorf("compatible policy rejected %s: %v", raw, err)
		}
	}
	for _, raw := range []string{
		`{"schema_version":3,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":1}`,
		`{"schema_version":4,"qualified_artifact_refs_version":1}`,
		`{"schema_version":4,"qualified_artifact_refs_version":2,"spec_authority_transfer_version":1}`,
		`{"schema_version":4,"qualified_artifact_refs_version":1,"spec_authority_transfer_version":2}`,
	} {
		if _, err := store.parseReviewPolicy([]byte(raw)); err == nil {
			t.Errorf("unsupported transfer metadata accepted: %s", raw)
		}
	}
}

func TestFederatedRoadmapQualifiedMembershipAndOutcomeConsumption(t *testing.T) {
	coordinator, source := t.TempDir(), t.TempDir()
	qualifiedFile(t, coordinator, "roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:source/roadmap:component\n---\n## Milestone: build\n- specs: xref:source/spec:engine\n")
	qualifiedFile(t, source, "specs/engine.md", "---\nslug: engine\nstatus: done\n---\n# Engine\n")
	qualifiedFile(t, source, "roadmaps/component.md", "---\nslug: component\nstatus: done\n---\n## Milestone: released\n- specs: xref:source/spec:engine\n")
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"coordinator": coordinator, "source": source}})}
	roadmap := r.Resolve("coordinator", "roadmap:program")
	if !roadmap.Resolved || roadmap.Status != "active" {
		t.Fatalf("coordinator roadmap: %+v", roadmap)
	}
	if len(roadmap.Dependencies) != 2 || roadmap.Dependencies[0].Project != "source" || roadmap.Dependencies[1].Kind != "milestone" {
		t.Fatalf("qualified member and consumed roadmap were not preserved: %+v", roadmap.Dependencies)
	}
	milestone := r.Resolve("coordinator", "milestone:program/build")
	if !milestone.Resolved || milestone.Status != "done" {
		t.Fatalf("qualified milestone membership did not resolve its source status: %+v", milestone)
	}
	if reason := r.ValidateGraph("coordinator", "roadmap:program"); reason != "" {
		t.Fatalf("acyclic qualified graph failed validation: %s", reason)
	}
}

func TestQualifiedArtifactNestedSubmoduleDoesNotAcquireAuthority(t *testing.T) {
	parent, source := t.TempDir(), t.TempDir()
	git := func(root string, args ...string) {
		t.Helper()
		args = append([]string{"-C", root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "protocol.file.allow=always"}, args...)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	for _, root := range []string{parent, source} {
		git(root, "init", "-q")
	}
	qualifiedFile(t, parent, "specs/shared.md", "---\nslug: shared\nstatus: draft\n---\n")
	qualifiedFile(t, source, "specs/shared.md", "---\nslug: shared\nstatus: done\n---\n")
	git(source, "add", ".")
	git(source, "commit", "-qm", "fixture")
	git(parent, "submodule", "add", "-q", source, "executor")
	git(parent, "add", ".")
	git(parent, "commit", "-qm", "fixture")
	child := filepath.Join(parent, "executor")
	r := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: parent, DefaultProjectID: "parent"})}
	if got := r.Resolve("parent", "xref:child/shared"); got.State != "unknown-project" {
		t.Fatalf("implicit ancestry: %+v", got)
	}
	r.Roots = NewRoots(RootsConfig{DefaultRoot: parent, DefaultProjectID: "parent", Explicit: map[string]string{"child": child}})
	got := r.Resolve("parent", "xref:child/shared")
	if !got.Resolved || got.Status != "done" || got.Revision == "" {
		t.Fatalf("submodule: %+v", got)
	}
	if local := r.Resolve("parent", "shared"); local.Status != "draft" || local.Identity.Project == got.Identity.Project {
		t.Fatalf("merged authorities: %+v %+v", local, got)
	}
}

func TestArtifactGraphTransferMetadataPreservesIdentityAndProofs(t *testing.T) {
	graph, names := artifactGraphChain(2, func(i int) string { return []string{"a", "b"}[i%2] })
	redirects := map[string]string{names[0]: names[1], names[1]: names[2]}
	graph[names[0]], graph[names[1]] = nil, nil
	resolver := artifactGraphTransferFixture(t, graph, redirects)
	resolved := resolver.Resolve("a", names[0])
	canonical := resolver.Resolve("a", names[2])
	original, _ := ParseArtifactRef(names[0])
	if !resolved.Resolved || resolved.Identity != original || resolved.CanonicalIdentity == nil || *resolved.CanonicalIdentity != canonical.Identity || resolved.redirectCrossings != 2 || resolved.Digest != canonical.Digest {
		t.Fatalf("returning redirect lost historical identity or cost: %+v", resolved)
	}
	withCost, err := json.Marshal(resolved)
	if err != nil {
		t.Fatal(err)
	}
	resolved.redirectCrossings = 0
	withoutCost, err := json.Marshal(resolved)
	if err != nil || string(withCost) != string(withoutCost) {
		t.Fatalf("private traversal cost changed serialized resolution: %s vs %s (%v)", withCost, withoutCost, err)
	}
	resolver.Authorize = func(project string) bool { return project == "a" }
	if got := resolver.ValidateGraph("a", names[0]); !strings.HasSuffix(got, "unauthorized-project") {
		t.Fatalf("intermediate redirect authorization bypassed: %q", got)
	}
	resolver.Authorize = nil
	source, err := resolver.Roots.StoreFor("a")
	if err != nil {
		t.Fatal(err)
	}
	redirect, found, err := readSpecTransferRedirect(NativeFiles(), source, original)
	if err != nil || !found {
		t.Fatalf("fixture transfer proof invalid: %v, found=%t", err, found)
	}
	if err := os.Remove(transferPlanFile(source.Root, redirect.OperationID)); err != nil {
		t.Fatal(err)
	}
	if got := resolver.ValidateGraph("a", names[0]); got != "transfer-in-progress" {
		t.Fatalf("missing transfer plan did not retain unknown transfer state: %q", got)
	}
}

func TestArtifactGraphTransferRedirectChainBound(t *testing.T) {
	for _, redirects := range []int{8, 9} {
		t.Run(fmt.Sprintf("redirects=%d", redirects), func(t *testing.T) {
			graph, names := artifactGraphChain(redirects, func(i int) string { return []string{"a", "b"}[i%2] })
			targets := map[string]string{}
			for i := 0; i < redirects; i++ {
				targets[names[i]] = names[i+1]
				graph[names[i]] = nil
			}
			resolver := artifactGraphTransferFixture(t, graph, targets)
			got := resolver.ValidateGraph("a", names[0])
			if redirects == 8 && got != "" || redirects == 9 && !strings.HasSuffix(got, "resolution-limit") {
				t.Fatalf("governed redirect chain bound: %q", got)
			}
			resolver.governanceDepthLimit = 0
			if got := resolver.ValidateGraph("a", names[0]); got != "" {
				t.Fatalf("legacy redirect chain below 16 refused: %q", got)
			}
		})
	}
}

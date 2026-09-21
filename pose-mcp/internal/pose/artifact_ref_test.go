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

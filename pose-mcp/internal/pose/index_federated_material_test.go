package pose

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIndexFederatedMaterialBindsReferencedSemanticsOnly(t *testing.T) {
	local, foreign, unrelated := t.TempDir(), t.TempDir(), t.TempDir()
	for root, project := range map[string]string{local: "proj.local", foreign: "proj.foreign", unrelated: "proj.unrelated"} {
		if err := os.MkdirAll(filepath.Join(root, ".pose", "specs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, ".pose", "roadmaps"), 0o755); err != nil {
			t.Fatal(err)
		}
		qualifiedFile(t, root, "project.json", `{"schema_version":1,"project_id":"`+project+`"}`)
	}
	qualifiedFile(t, local, "specs/consumer.md", "---\nslug: consumer\nstatus: in-progress\ndepends_on: xref:proj.foreign/spec:engine\n---\n# Consumer\n")
	qualifiedFile(t, local, "roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:proj.foreign/spec:engine\n---\n")
	qualifiedFile(t, foreign, "specs/engine.md", "---\nslug: engine\nstatus: done\n---\n# Engine v1\n")
	qualifiedFile(t, unrelated, "specs/other.md", "---\nslug: other\nstatus: done\n---\n# Other v1\n")

	bindings, err := json.Marshal(map[string]string{
		"proj.local": local, "proj.foreign": foreign, "proj.unrelated": unrelated,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", string(bindings))
	resolver, project, err := EnvironmentArtifactResolverIn(NativeFiles(), local, "")
	if err != nil || project != "proj.local" {
		t.Fatalf("resolver = %q, %v", project, err)
	}
	store := Store{Root: local, FS: NativeFiles(), FederatedProjectID: project, FederatedResolver: &resolver}
	before, err := BuildIndexFederatedMaterialIn(store)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Projects, []string{"proj.foreign"}) {
		t.Fatalf("material roots = %v, want only the referenced foreign project", before.Projects)
	}
	if len(before.Projection.Specs) != 1 || len(before.Projection.Roadmaps) != 1 {
		t.Fatalf("federated projection omitted an output input: specs=%d roadmaps=%d", len(before.Projection.Specs), len(before.Projection.Roadmaps))
	}
	beforeDigest, known := IndexFederatedMaterialDigest(before.Projection)
	if !known || beforeDigest == "" {
		t.Fatal("semantic material digest was unknown")
	}
	bound, known := IndexInputBindingIn(NativeFiles(), local, "federated-material")
	if !known || bound != beforeDigest {
		t.Fatalf("input binding did not use the shared projection: binding=%q projection=%q known=%v", bound, beforeDigest, known)
	}

	qualifiedFile(t, unrelated, "specs/other.md", "---\nslug: other\nstatus: draft\n---\n# Other unrelated edit\n")
	afterUnrelated, err := BuildIndexFederatedMaterialIn(store)
	if err != nil {
		t.Fatal(err)
	}
	unrelatedDigest, known := IndexFederatedMaterialDigest(afterUnrelated.Projection)
	if !known || unrelatedDigest != beforeDigest {
		t.Fatalf("unrelated root edit changed federated material digest: %q -> %q", beforeDigest, unrelatedDigest)
	}

	qualifiedFile(t, foreign, "specs/engine.md", "---\nslug: engine\nstatus: done\n---\n# Engine v2\n")
	afterForeign, err := BuildIndexFederatedMaterialIn(store)
	if err != nil {
		t.Fatal(err)
	}
	foreignDigest, known := IndexFederatedMaterialDigest(afterForeign.Projection)
	if !known || foreignDigest == beforeDigest {
		t.Fatalf("referenced foreign edit did not stale federated material: %q -> %q", beforeDigest, foreignDigest)
	}
	bound, known = IndexInputBindingIn(NativeFiles(), local, "federated-material")
	if !known || bound != foreignDigest {
		t.Fatalf("foreign edit binding disagrees with builder projection: binding=%q projection=%q known=%v", bound, foreignDigest, known)
	}
	if before.SpecDependencyResolutions["consumer"][0].Digest == afterForeign.SpecDependencyResolutions["consumer"][0].Digest {
		t.Fatal("foreign spec dependency digest did not track the changed content")
	}
	if before.RoadmapAcceptances["program"].Manifest.Digest == afterForeign.RoadmapAcceptances["program"].Manifest.Digest {
		t.Fatal("federated roadmap acceptance digest did not track the changed dependency")
	}
}

func TestIndexFederatedMaterialIgnoresUnrelatedCoordinatorCommit(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{".pose/specs", ".pose/roadmaps", ".pose/policy"} {
		if err := os.MkdirAll(filepath.Join(root, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	qualifiedFile(t, root, "project.json", `{"schema_version":1,"project_id":"proj.coordinator"}`)
	qualifiedFile(t, root, "specs/consumer.md", "---\nslug: consumer\nstatus: in-progress\ndepends_on: xref:proj.coordinator/spec:dependency\n---\n# Consumer\n")
	qualifiedFile(t, root, "specs/dependency.md", "---\nslug: dependency\nstatus: done\n---\n# Dependency\n")
	qualifiedFile(t, root, "roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:proj.coordinator/spec:dependency\n---\n")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", out, err)
	}
	for _, args := range [][]string{
		{"config", "user.name", "POSE fixture"},
		{"config", "user.email", "pose@example.invalid"},
		{"add", "--all"},
		{"commit", "-q", "-m", "coordinator baseline"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: root, DefaultProjectID: "proj.coordinator"}), FS: NativeFiles()}
	store := Store{Root: root, FS: NativeFiles(), FederatedProjectID: "proj.coordinator", FederatedResolver: &resolver}
	before, err := BuildIndexFederatedMaterialIn(store)
	if err != nil {
		t.Fatal(err)
	}
	beforeDigest, known := IndexFederatedMaterialDigest(before.Projection)
	if !known {
		t.Fatal("initial coordinator material digest is unknown")
	}

	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("routine evidence\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "unrelated.txt"}, {"commit", "-q", "-m", "unrelated coordinator evidence"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	after, err := BuildIndexFederatedMaterialIn(store)
	if err != nil {
		t.Fatal(err)
	}
	afterDigest, known := IndexFederatedMaterialDigest(after.Projection)
	if !known || afterDigest != beforeDigest {
		t.Fatalf("unrelated coordinator commit changed federated material digest: %q -> %q", beforeDigest, afterDigest)
	}
	if !reflect.DeepEqual(after.Projection, before.Projection) {
		t.Fatalf("unrelated coordinator commit changed the indexed semantic projection:\nbefore=%+v\nafter=%+v", before.Projection, after.Projection)
	}
	if got := after.Projection.Roadmaps[0].Acceptance.Manifest.CoordinatorRevision; got != "" {
		t.Fatalf("semantic input projection retained volatile coordinator revision %q", got)
	}
	if got := after.RoadmapAcceptances["program"].Manifest.CoordinatorRevision; got == "" {
		t.Fatal("generated output dropped the raw coordinator revision")
	}
	beforeResolution := before.SpecDependencyResolutions["consumer"][0]
	afterResolution := after.SpecDependencyResolutions["consumer"][0]
	if beforeResolution.Revision == "" || afterResolution.Revision == "" || beforeResolution.Revision == afterResolution.Revision {
		t.Fatalf("generated output did not preserve observed local source revisions across a coordinator commit: before=%q after=%q", beforeResolution.Revision, afterResolution.Revision)
	}
	for _, resolution := range after.Projection.Specs[0].DependencyResolutions {
		if resolution.Identity.Project == "proj.coordinator" && resolution.Revision != "" {
			t.Fatalf("semantic input projection retained volatile local source revision %q", resolution.Revision)
		}
	}
}

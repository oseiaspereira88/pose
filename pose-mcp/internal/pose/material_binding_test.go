package pose

import "testing"

func TestIsAdministrativePathIsClosedAndPathExact(t *testing.T) {
	for p, want := range map[string]bool{
		".pose/indexes/spec-graph.json":          true,
		".pose/results/delivery-validation.json": true,
		".pose/state/history.jsonl":              true,
		".pose/assessments/pose-mcp.md":          true,
		".pose/review-bundles/rvb-1.json":        true,
		".pose/indexes/validation-matrix.json":   false,
		".pose/policy/delivery.json":             false,
		".pose/rules/security.md":                false,
		".pose/specs/2026-01-01-x.md":            false,
		"pose-mcp/internal/pose/a.go":            false,
		".pose/indexes/../policy/delivery.json":  false,
		"../.pose/indexes/x.json":                false,
		".pose/review-x":                         false,
		".pose/indexesx/a.json":                  false,
		"":                                       false,
	} {
		if got := IsAdministrativePath(p); got != want {
			t.Errorf("%q: got %v want %v", p, got, want)
		}
	}
}

func obs(action, p string) ObservedPath { return ObservedPath{Action: action, Path: p} }

func TestCommitIsAdministrative(t *testing.T) {
	specStatus := []byte("diff --git a/.pose/specs/a.md b/.pose/specs/a.md\n--- a/.pose/specs/a.md\n+++ b/.pose/specs/a.md\n@@ -1 +1 @@\n-status: draft\n+status: done\n")
	specBody := []byte("diff --git a/.pose/specs/a.md b/.pose/specs/a.md\n--- a/.pose/specs/a.md\n+++ b/.pose/specs/a.md\n@@ -1 +1,2 @@\n-status: draft\n+status: done\n+new requirement\n")
	symlink := []byte("diff --git a/.pose/results/x b/.pose/results/x\nnew file mode 120000\n")
	cases := []struct {
		name  string
		paths []ObservedPath
		diff  []byte
		want  bool
	}{
		{"indexes and results", []ObservedPath{obs("M", ".pose/indexes/a.json"), obs("M", ".pose/results/b.json")}, nil, true},
		{"mixed with source", []ObservedPath{obs("M", ".pose/indexes/a.json"), obs("M", "a.go")}, nil, false},
		{"spec status only", []ObservedPath{obs("M", ".pose/specs/a.md")}, specStatus, true},
		{"spec body", []ObservedPath{obs("M", ".pose/specs/a.md")}, specBody, false},
		{"symlink", []ObservedPath{obs("A", ".pose/results/x")}, symlink, false},
		{"rename into class", []ObservedPath{{Action: "R", OldPath: "a.go", NewPath: ".pose/results/a.go"}}, nil, false},
		{"matrix", []ObservedPath{obs("M", ".pose/indexes/validation-matrix.json")}, nil, false},
		{"no paths", nil, nil, false},
	}
	for _, c := range cases {
		if got := CommitIsAdministrative(c.paths, c.diff); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func materialGraph(head, digest string) DeliveryIntegrityGraph {
	return DeliveryIntegrityGraph{
		Claims:     []ArtifactClaim{{Spec: "s", Path: "a.go", Action: "modified"}},
		ChangeSets: []ChangeSet{{ID: "cs-" + head, Spec: "s", Head: head, ResolvedHead: head, Commits: []string{head}, MaterialDigest: digest}},
	}
}

func TestMaterialEvidenceSurvivesAdministrativeHeadMoveButNotMaterialChange(t *testing.T) {
	validated := materialGraph("h1", MaterialDiffDigest([][]byte{[]byte("code")}, []bool{true}))
	bound := ScopedMaterialDigest(validated, "s")
	if bound == "" {
		t.Fatal("no material digest")
	}
	result := DeliveryValidationResult{GitHead: "h1", MaterialProvenance: map[string]string{"s": bound}}

	// Same material, head moved by an administrative commit.
	moved := materialGraph("h2", MaterialDiffDigest([][]byte{[]byte("code"), []byte("index")}, []bool{true, false}))
	if !deliveryEvidenceCurrent(result, "s", "in-progress", moved, moved.ChangeSets) {
		t.Fatal("administrative-only head move invalidated evidence")
	}
	// A material commit changes the digest.
	changed := materialGraph("h3", MaterialDiffDigest([][]byte{[]byte("code"), []byte("more")}, []bool{true, true}))
	if deliveryEvidenceCurrent(result, "s", "in-progress", changed, changed.ChangeSets) {
		t.Fatal("material change kept stale evidence current")
	}
	// A changed matrix is stale.
	matrix := materialGraph("h4", MaterialDiffDigest([][]byte{[]byte("code"), []byte("matrix")}, []bool{true, true}))
	if deliveryEvidenceCurrent(result, "s", "in-progress", matrix, matrix.ChangeSets) {
		t.Fatal("matrix change kept evidence current")
	}
}

func TestMaterialBindingNeedsEveryChangeSetDigest(t *testing.T) {
	g := materialGraph("h1", "")
	if ScopedMaterialDigest(g, "s") != "" {
		t.Fatal("a set without a material digest must not bind")
	}
	legacy := DeliveryValidationResult{GitHead: "h1", MaterialProvenance: map[string]string{"s": "sha256:x"}}
	if deliveryEvidenceCurrent(legacy, "s", "in-progress", g, g.ChangeSets) != true {
		t.Fatal("falls back to the git head rule")
	}
	if deliveryEvidenceCurrent(DeliveryValidationResult{GitHead: "zz"}, "s", "in-progress", g, g.ChangeSets) {
		t.Fatal("unrelated head accepted")
	}
}

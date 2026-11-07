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

func regularEntry(blob string) TreeEntry { return TreeEntry{Mode: "100644", Blob: blob} }

func TestMaterialSetDigestIgnoresAdministrativeRecordsOnly(t *testing.T) {
	paths := []ObservedPath{obs("M", "a.go"), obs("M", ".pose/indexes/spec-graph.json")}
	base := map[string]TreeEntry{"a.go": regularEntry("b1"), ".pose/indexes/spec-graph.json": regularEntry("i1"), ".pose/indexes/validation-matrix.json": regularEntry("m1")}
	want := MaterialSetDigest(paths, base)

	admin := map[string]TreeEntry{"a.go": regularEntry("b1"), ".pose/indexes/spec-graph.json": regularEntry("i2"), ".pose/indexes/validation-matrix.json": regularEntry("m1")}
	if MaterialSetDigest(paths, admin) != want {
		t.Fatal("an administrative record changed the digest")
	}
	for name, mutate := range map[string]func(m map[string]TreeEntry){
		"code":    func(m map[string]TreeEntry) { m["a.go"] = regularEntry("b2") },
		"matrix":  func(m map[string]TreeEntry) { m[".pose/indexes/validation-matrix.json"] = regularEntry("m2") },
		"deleted": func(m map[string]TreeEntry) { delete(m, "a.go") },
		"symlink": func(m map[string]TreeEntry) {
			m[".pose/indexes/spec-graph.json"] = TreeEntry{Mode: "120000", Blob: "i1"}
		},
		"symlink target": func(m map[string]TreeEntry) {
			m[".pose/indexes/spec-graph.json"] = TreeEntry{Mode: "120000", Blob: "i9"}
		},
		"mode": func(m map[string]TreeEntry) { m["a.go"] = TreeEntry{Mode: "100755", Blob: "b1"} },
	} {
		m := map[string]TreeEntry{}
		for k, v := range base {
			m[k] = v
		}
		mutate(m)
		if MaterialSetDigest(paths, m) == want {
			t.Errorf("%s: digest unchanged", name)
		}
	}
}

func TestMaterialSetDigestTreatsRenamesOutOfAClassAsMaterial(t *testing.T) {
	paths := []ObservedPath{{Action: "R", OldPath: "a.go", NewPath: ".pose/results/a.go"}}
	moved := map[string]TreeEntry{".pose/results/a.go": regularEntry("b1")}
	kept := map[string]TreeEntry{".pose/results/a.go": regularEntry("b1"), "a.go": regularEntry("b1")}
	if MaterialSetDigest(paths, moved) == MaterialSetDigest(paths, kept) {
		t.Fatal("the vanished source path must count")
	}
}

func materialGraph(head, digest string) DeliveryIntegrityGraph {
	return DeliveryIntegrityGraph{
		Claims:     []ArtifactClaim{{Spec: "s", Path: "a.go", Action: "modified"}},
		ChangeSets: []ChangeSet{{ID: "cs-" + head, Spec: "s", Head: head, ResolvedHead: head, Commits: []string{head}, MaterialDigest: digest}},
	}
}

func TestMaterialEvidenceSurvivesHeadMoveButNotMaterialChange(t *testing.T) {
	validated := materialGraph("h1", "sha256:material-1")
	result := DeliveryValidationResult{GitHead: "h1", MaterialProvenance: map[string]string{"s": ScopedMaterialDigest(validated, "s")}}

	moved := materialGraph("h2", "sha256:material-1")
	if !deliveryEvidenceCurrent(result, "s", "in-progress", moved, moved.ChangeSets) {
		t.Fatal("a head move with the same material invalidated evidence")
	}
	changed := materialGraph("h3", "sha256:material-2")
	if deliveryEvidenceCurrent(result, "s", "in-progress", changed, changed.ChangeSets) {
		t.Fatal("material change kept stale evidence current")
	}
}

func TestMaterialBindingGovernsAndNeverFallsThrough(t *testing.T) {
	g := materialGraph("h1", "sha256:material-1")
	// The legacy rules would accept this result (same git head, same scoped digest);
	// its material binding is from another engine, so it is stale.
	result := DeliveryValidationResult{GitHead: "h1", ScopeProvenance: map[string]string{"s": ScopedDeliveryProvenanceDigest(g, "s")},
		MaterialProvenance: map[string]string{"s": "sha256:other-engine"}}
	if deliveryEvidenceCurrent(result, "s", "in-progress", g, g.ChangeSets) {
		t.Fatal("a mismatched material binding fell through to the commit-bound rules")
	}
	if !deliveryEvidenceCurrent(DeliveryValidationResult{GitHead: "h1"}, "s", "in-progress", g, g.ChangeSets) {
		t.Fatal("a result without a material binding keeps the existing rule")
	}
	result.GeneratedAt = "2026-11-07T00:00:00Z"
	if !deliveryEvidenceCurrent(result, "s", "done", g, g.ChangeSets) {
		t.Fatal("a closed spec keeps its exemption")
	}
}

func TestMaterialDigestDoesNotChangeTheCommitBoundDigests(t *testing.T) {
	with := materialGraph("h1", "sha256:material-1")
	without := materialGraph("h1", "")
	if ScopedDeliveryProvenanceDigest(with, "s") != ScopedDeliveryProvenanceDigest(without, "s") {
		t.Fatal("the scoped provenance digest moved with the material digest")
	}
	if ScopedMaterialDigest(without, "s") != "" {
		t.Fatal("a set without a material digest must not bind")
	}
}

func TestUnreadableMaterialNeverBindsAndIsRecordedAsUnavailable(t *testing.T) {
	g := materialGraph("h1", MaterialDigestUnreadable)
	if ScopedMaterialDigest(g, "s") != "" {
		t.Fatal("an unreadable set produced a digest")
	}
	bound, ok := MaterialBinding(g, "s")
	if !ok || bound != MaterialBindingUnavailable {
		t.Fatalf("got %q %v", bound, ok)
	}
	// Recorded as unavailable, the result is stale although the legacy rules would accept it.
	result := DeliveryValidationResult{GitHead: "h1", ScopeProvenance: map[string]string{"s": ScopedDeliveryProvenanceDigest(g, "s")},
		MaterialProvenance: map[string]string{"s": bound}}
	if deliveryEvidenceCurrent(result, "s", "in-progress", g, g.ChangeSets) {
		t.Fatal("unreadable material let the legacy rule accept the result")
	}
	if _, ok := MaterialBinding(materialGraph("h1", ""), "s"); ok {
		t.Fatal("a set that never computed a digest must record nothing")
	}
}

func TestRenameIntoAnAdministrativeClassStaysMaterial(t *testing.T) {
	paths := []ObservedPath{{Action: "R", OldPath: "a.go", NewPath: ".pose/results/a.go"}}
	first := MaterialSetDigest(paths, map[string]TreeEntry{".pose/results/a.go": regularEntry("b1")})
	edited := MaterialSetDigest(paths, map[string]TreeEntry{".pose/results/a.go": regularEntry("b2")})
	if first == edited {
		t.Fatal("editing the moved file inside the class kept the digest")
	}
}

func TestFactResultKeySeparatesMaterialBindings(t *testing.T) {
	a := DeliveryValidationResult{ID: "x", MaterialProvenance: map[string]string{"s": "sha256:1"}}
	b := DeliveryValidationResult{ID: "x", MaterialProvenance: map[string]string{"s": "sha256:2"}}
	if factResultKey(a) == factResultKey(b) {
		t.Fatal("two material bindings share one identity")
	}
}

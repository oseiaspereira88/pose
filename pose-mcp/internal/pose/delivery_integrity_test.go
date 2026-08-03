package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactClaimsClosedGrammarAndContradictions(t *testing.T) {
	spec := Spec{Slug: "alpha", Body: "## 3. Technical Plan\n\n### Artifacts\n- created: cmd/new.go\n- modified: internal/core.go\n- renamed: old/name.go -> new/name.go\n- removed: obsolete.go\n\n### Risks\nnone\n"}
	claims, found, err := ParseArtifactClaims(spec, ArtifactPolicy{})
	if err != nil || !found || len(claims) != 4 || claims[2].OldPath != "old/name.go" {
		t.Fatalf("claims=%+v found=%t err=%v", claims, found, err)
	}
	for _, body := range []string{
		"### Artifacts\n- created: ../escape\n",
		"### Artifacts\n- created: *.go\n",
		"### Artifacts\n- none: analysis\n- modified: x.go\n",
		"### Artifacts\n- modified: x.go\n- removed: x.go\n",
		"### Artifacts\n- none: invented\n",
	} {
		if _, _, err := ParseArtifactClaims(Spec{Slug: "bad", Body: body}, ArtifactPolicy{}); err == nil {
			t.Fatalf("invalid body accepted: %s", body)
		}
	}
}

func TestArtifactPathRejectsSymlinkEscapeAndDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifactPath(root, "escape", false); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("symlink escape accepted: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifactPath(root, "dir", false); err == nil {
		t.Fatal("directory accepted as exact artifact")
	}
}

func TestDeliveryIntegrityKeepsClaimsObservationsAndReverseSeparate(t *testing.T) {
	policy := ArtifactPolicy{SchemaVersion: 1, Enabled: true, GovernedRoots: []string{"internal"}, Exclusions: []string{"internal/testdata"}, Severities: map[string]string{"action-mismatch": "error", "undeclared": "error", "orphan": "warning"}}
	specs := []Spec{{Slug: "alpha", Status: "done"}}
	claims := []ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "internal/core.go"}}
	sets := []ChangeSet{{ID: "cs-1", Spec: "alpha", Selector: "range:a..b", Paths: []ObservedPath{{Action: "modified", Path: "internal/core.go"}, {Action: "created", Path: "internal/extra.go"}}}}
	graph := BuildDeliveryIntegrity(specs, claims, sets, []string{"internal/core.go", "internal/orphan.go", "internal/testdata/fixture.go"}, policy)
	if got := graph.Reverse["internal/core.go"]; len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("reverse=%v", graph.Reverse)
	}
	codes := map[string]bool{}
	for _, finding := range graph.Findings {
		codes[finding.Code+":"+finding.Path] = true
	}
	if !codes["undeclared:internal/extra.go"] || !codes["orphan:internal/orphan.go"] || codes["orphan:internal/testdata/fixture.go"] {
		t.Fatalf("unexpected findings: %+v", graph.Findings)
	}
	if len(graph.Edges) == 0 || graph.InputDigest == "" {
		t.Fatalf("graph lacks deterministic identity: %+v", graph)
	}
}

func TestDeliveryIntegrityFindingIDsAndInputDigestAreStable(t *testing.T) {
	policy := ArtifactPolicy{GovernedRoots: []string{"cmd"}}
	args := func() DeliveryIntegrityGraph {
		return BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, nil, nil, []string{"cmd/a.go"}, policy)
	}
	a, b := args(), args()
	if a.InputDigest != b.InputDigest || len(a.Findings) != len(b.Findings) {
		t.Fatalf("unstable graph: a=%+v b=%+v", a, b)
	}
	for i := range a.Findings {
		if a.Findings[i].ID != b.Findings[i].ID {
			t.Fatalf("unstable finding ID: %s != %s", a.Findings[i].ID, b.Findings[i].ID)
		}
	}
}

func TestDeliveryIntegrityReconcilesEvolvingClaimsAcrossImmutableChangeSets(t *testing.T) {
	claims := []ArtifactClaim{
		{Spec: "alpha", Action: "modified", Path: "internal/first.go"},
		{Spec: "alpha", Action: "created", Path: "internal/later.go"},
	}
	sets := []ChangeSet{
		{ID: "cs-early", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "internal/first.go"}}},
		{ID: "cs-final", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "internal/first.go"}, {Action: "created", Path: "internal/later.go"}}},
	}
	graph := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, claims, sets, []string{"internal/first.go", "internal/later.go"}, ArtifactPolicy{})
	for _, finding := range graph.Findings {
		if finding.Code == "action-mismatch" {
			t.Fatalf("an intermediate immutable change set invalidated final coverage: %+v", graph.Findings)
		}
	}
}

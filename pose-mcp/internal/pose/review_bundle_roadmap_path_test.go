package pose

// Regression test: reviewBundleScopeProjection's "roadmap" case used to key
// its semantic Section on rm.Path — Store.Root joined with the roadmap
// file, an absolute filesystem path. Two checkouts of the exact same
// commit at different paths (any two clones — a developer's machine and
// CI, or CI and a fresh clone) therefore computed different
// ChangedSections keys for identical content, so a roadmap sealed on one
// machine always read as "changed" (state: superseded) everywhere else.
// Discovered publishing v1.3.0: the roadmap sealed locally failed CI's
// `pose check --strict` with "resolve closeout blockers", reproduced with
// a plain `git clone` of the exact same commit to a different path.

import (
	"strings"
	"testing"
)

func TestReviewBundleRoadmapScopeProjectionIsPathIndependent(t *testing.T) {
	roadmapFile := "---\nslug: sample\nstatus: draft\ncreated_at: 2026-08-15\n---\n\n# Roadmap: Sample\n\n**Outcome:** test.\n\n## Milestone: m1\n- after:\n- specs: alpha\n"
	rootA := t.TempDir()
	rootB := t.TempDir()
	for _, root := range []string{rootA, rootB} {
		writeReviewFixture(t, root, ".pose/roadmaps/sample.md", roadmapFile)
	}
	scope, err := ParseScopeRef("roadmap:sample")
	if err != nil {
		t.Fatal(err)
	}
	projA, _, err := Store{Root: rootA}.reviewBundleScopeProjection(scope)
	if err != nil {
		t.Fatal(err)
	}
	projB, _, err := Store{Root: rootB}.reviewBundleScopeProjection(scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(projA.Sections) != 1 || len(projB.Sections) != 1 {
		t.Fatalf("expected exactly one roadmap section each: A=%d B=%d", len(projA.Sections), len(projB.Sections))
	}
	pathA, pathB := projA.Sections[0].Path, projB.Sections[0].Path
	if strings.Contains(pathA, rootA) || strings.Contains(pathB, rootB) {
		t.Errorf("roadmap scope Path leaks the absolute checkout root: A=%q (root=%q) B=%q (root=%q)", pathA, rootA, pathB, rootB)
	}
	if pathA != pathB {
		t.Errorf("roadmap scope Path differs across checkouts of identical content: A=%q B=%q — sealing on one machine and verifying on another would always report the roadmap as changed", pathA, pathB)
	}
	if projA.Sections[0].Digest != projB.Sections[0].Digest {
		t.Errorf("roadmap content digest differs across checkouts of identical content: A=%q B=%q", projA.Sections[0].Digest, projB.Sections[0].Digest)
	}
}

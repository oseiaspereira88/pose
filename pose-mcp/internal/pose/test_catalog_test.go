package pose

import (
	"strings"
	"testing"
)

// A test: ref resolves to what each ecosystem names a test, to a tracked test
// file, or to a go test -run pattern that selects a test; an invented name
// resolves to nothing (spec pose-trace-test-refs-resolve).
func TestTraceTestRefsResolveAgainstTheTrackedTests(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeReviewFixture(t, root, "api/digest_test.go", "package api\n\nfunc TestDigestStable(t *testing.T) {\n\tt.Run(\"empty input\", func(t *testing.T) {})\n}\n")
	writeReviewFixture(t, root, "web/src/Panel.test.tsx", "it('shows the open obligations', () => {});\n")
	writeReviewFixture(t, root, "tools/test_render.py", "def test_render_plain():\n    pass\n")
	writeReviewFixture(t, root, "core/src/lib.rs", "#[test]\nfn parses_header() {}\n")
	writeReviewFixture(t, root, "api/digest.go", "package api\n\nfunc TestLooksLikeATestButIsNot() {}\n")
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "fixture")
	catalog := LoadTestCatalog(root)
	if catalog == nil {
		t.Fatal("no catalog for a Git tree")
	}
	for _, ref := range []string{
		"TestDigestStable", "TestDigestStable/empty_input", "Digest", "shows the open obligations",
		"shows-the-open-obligations", "test_render_plain", "parses_header", "api/digest_test.go",
	} {
		if !catalog.Resolves(ref) {
			t.Errorf("%q did not resolve", ref)
		}
	}
	for _, ref := range []string{"TestDigestStableForever", "TestLooksLikeATestButIsNot", "missing/file_test.go", ""} {
		if catalog.Resolves(ref) {
			t.Errorf("%q resolved but names no test", ref)
		}
	}
	trace := ParseRequirementTrace("## 2. Requirements\n- R1: a\n- R2: b\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestDigestStable\n- R2 [satisfied] test:TestABMDesignBasisDigestStable\n")
	if got := strings.Join(UnresolvedTestRefs(trace, catalog), ","); got != "R2 test:TestABMDesignBasisDigestStable" {
		t.Fatalf("unresolved = %q", got)
	}
	if UnresolvedTestRefs(trace, nil) != nil {
		t.Fatal("without Git nothing can be resolved, so nothing is reported")
	}
}

// A nested repository the validation matrix registers as a module, neither
// tracked nor a submodule, still contributes its tests.
func TestTraceTestRefsResolveInNestedModuleRepositories(t *testing.T) {
	root := t.TempDir()
	writeReviewFixture(t, root, "web/src/Roadmap.test.tsx", "it('shows the backlog', () => {});\n")
	for _, dir := range []string{root, root + "/web"} {
		designDeltaGit(t, dir, "init", "-q")
		designDeltaGit(t, dir, "config", "user.email", "pose@example.test")
		designDeltaGit(t, dir, "config", "user.name", "POSE Test")
	}
	designDeltaGit(t, root+"/web", "add", "--", ".")
	designDeltaGit(t, root+"/web", "commit", "-q", "-m", "web")
	writeReviewFixture(t, root, ".gitignore", "web/\n")
	writeReviewFixture(t, root, ".pose/indexes/validation-matrix.json", `{"moduleOverrides":{"web":{"checks":[]}}}`)
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "outer")
	if catalog := buildTestCatalog(root); catalog == nil || !catalog.Resolves("shows the backlog") || !catalog.Resolves("web/src/Roadmap.test.tsx") {
		t.Fatalf("a registered nested module repository's tests did not resolve: nested=%v", nestedModuleRepositories(root))
	}
	writeReviewFixture(t, root, ".pose/indexes/validation-matrix.json", `{"moduleOverrides":{}}`)
	if catalog := buildTestCatalog(root); catalog == nil || catalog.Resolves("shows the backlog") {
		t.Fatal("an unregistered nested repository was read")
	}
}

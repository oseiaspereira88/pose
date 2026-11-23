package cli

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// sealReadinessRoot is the dispatch fixture with the delivery policy enabled
// and the index holding the finding `pose close` would report, unsealed.
func sealReadinessRoot(t *testing.T, severity string) string {
	t.Helper()
	root := newDetachedFixtureRoot(t)
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14"}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/policy/delivery.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-01","results_path":".pose/results/current.json","roots":[{"path":"pose-mcp","kind":"capability","profile":"api-contract","entrypoint":"pose-mcp/lib.go"}],"severities":{"undeclared-delivery":"error"}}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n")
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\n")
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "alpha")
	head := git("rev-parse", "HEAD")
	provenance := "sha256:" + strings.Repeat("1", 64)
	graph := posemodel.DeliveryIntegrityGraph{
		SchemaVersion: 1, ProvenanceDigest: provenance,
		ChangeSets: []posemodel.ChangeSet{{ID: "cs-alpha", Spec: "alpha", Selector: "range:" + head + ".." + head, Base: head, Head: head, ResolvedBase: head, ResolvedHead: head,
			Paths: []posemodel.ObservedPath{{Action: "modified", Path: "pose-mcp/lib.go"}}, DiffDigest: "sha256:" + strings.Repeat("2", 64)}},
		ValidationResults: []posemodel.DeliveryValidationResult{{ID: "val-alpha", Module: "pose-mcp", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: head, ProvenanceDigest: provenance}},
		Reverse:           map[string][]string{"pose-mcp/lib.go": {"alpha"}},
		Findings:          []posemodel.DeliveryIntegrityFinding{posemodel.NewDeliveryIntegrityFinding("undeclared-delivery", severity, "alpha", "pose-mcp/lib.go", "cs-alpha", "changed delivery root pose-mcp has no delivery declaration", "declare a typed delivery target or narrow the attributed change set")},
	}
	raw, _ := json.Marshal(graph)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	return root
}

// Spec pose-review-readiness-before-seal R1: a spec scope whose change
// touches a delivery root without a declaration is refused at seal, with the
// rule `pose close` uses, instead of after an approved review.
func TestSealRefusesAnUndeclaredDelivery(t *testing.T) {
	root := sealReadinessRoot(t, "error")
	code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal")
	if code == 0 || !strings.Contains(out, "undeclared-delivery [pose-mcp/lib.go]") || !strings.Contains(out, "pose close refuses it") {
		t.Fatalf("an undeclared delivery was sealed: %d %s", code, out)
	}
	// A warning, as for a spec created before the delivery policy, seals.
	root = sealReadinessRoot(t, "warning")
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatalf("a warning blocked the seal: %d %s", code, out)
	}
}

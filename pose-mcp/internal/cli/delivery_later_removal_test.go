package cli

import (
	"encoding/json"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// laterRemovalFixture is a repository whose history is: alpha creates
// internal/x.go and internal/y.go; beta removes x.go and declares it; gamma
// declares x.go modified but recreates it after beta's removal, and an
// unattributed commit deletes it again; eps declares y.go removed, but y.go
// is deleted by an unattributed commit.
func laterRemovalFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "config", "user.name", "POSE Tests")
	spec := func(slug, artifacts string) {
		writeArtifactTestFile(t, root, ".pose/specs/"+slug+"/spec.md", "---\nslug: "+slug+"\nstatus: done\ncreated_at: 2026-10-30\n---\n\n# Spec: "+slug+"\n\n## 3. Technical Plan\n\n### Artifacts\n"+artifacts+"\n## 4. Tasks\nwork\n")
	}
	commit := func(msg, trailer string) {
		artifactGit(t, root, "add", "-A", "--", ".")
		args := []string{"commit", "-q", "-m", msg}
		if trailer != "" {
			args = append(args, "-m", "POSE-Spec: "+trailer)
		}
		artifactGit(t, root, args...)
	}
	writeArtifactTestFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["internal"],"severities":{"action-mismatch":"error","undeclared":"error","existence":"error"}}`)
	writeArtifactTestFile(t, root, "README.md", "fixture\n")
	commit("baseline", "")
	spec("alpha", "- created: internal/x.go\n- created: internal/y.go\n")
	writeArtifactTestFile(t, root, "internal/x.go", "package internal\n")
	writeArtifactTestFile(t, root, "internal/y.go", "package internal\n")
	commit("alpha", "alpha")
	spec("beta", "- removed: internal/x.go\n")
	artifactGit(t, root, "rm", "-q", "--", "internal/x.go")
	commit("beta", "beta")
	spec("gamma", "- modified: internal/x.go\n")
	writeArtifactTestFile(t, root, "internal/x.go", "package internal\n// back\n")
	commit("gamma", "gamma")
	artifactGit(t, root, "rm", "-q", "--", "internal/x.go")
	commit("an unattributed deletion", "")
	spec("eps", "- removed: internal/y.go\n")
	commit("eps declares", "eps")
	artifactGit(t, root, "rm", "-q", "--", "internal/y.go")
	commit("another unattributed deletion", "")
	return root
}

func existenceFindings(graph posemodel.DeliveryIntegrityGraph) map[string]string {
	out := map[string]string{}
	for _, f := range graph.Findings {
		if f.Code == "existence" {
			out[f.Spec+" "+f.Path] = f.Message
		}
	}
	return out
}

// Spec pose-delivery-later-removal R1–R4: a later spec's observed removal
// satisfies the existence of a path an earlier spec created; a claim whose
// change follows the removal, and a removal its spec never performed, do
// not; the graph records the removal; artifact-check reads the same rule.
func TestALaterRemovalSatisfiesOnlyWhatPrecedesIt(t *testing.T) {
	root := laterRemovalFixture(t)
	graph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	findings := existenceFindings(graph)
	if msg, ok := findings["alpha internal/x.go"]; ok {
		t.Fatalf("alpha's x.go, removed later by beta, is still a finding: %s", msg)
	}
	if msg := findings["gamma internal/x.go"]; !strings.Contains(msg, "spec beta removed the path at") || !strings.Contains(msg, "does not precede the removal") {
		t.Fatalf("gamma, which changed x.go after beta's removal: %q", msg)
	}
	if msg := findings["alpha internal/y.go"]; !strings.Contains(msg, "spec eps declares the path removed, but its change set deletes nothing there") {
		t.Fatalf("a removal eps never performed: %q", msg)
	}
	removedBy := false
	for _, e := range graph.Edges {
		if e.Type == "removed-by" && e.From == "artifact:internal/x.go" && e.To == "spec:beta" {
			removedBy = true
		}
	}
	if !removedBy || len(graph.Removals) != 1 || graph.Removals[0].Spec != "beta" || graph.Removals[0].Commit == "" {
		t.Fatalf("the graph does not record beta's removal: %v %+v", removedBy, graph.Removals)
	}
	for slug, wantFinding := range map[string]bool{"alpha": false, "gamma": true} {
		var out, errOut strings.Builder
		cmdArtifactCheck(root, []string{"--spec", slug, "--json"}, &out, &errOut)
		var checked posemodel.DeliveryIntegrityGraph
		if err := json.Unmarshal([]byte(out.String()), &checked); err != nil {
			t.Fatalf("artifact-check %s: %v %s %s", slug, err, out.String(), errOut.String())
		}
		_, got := existenceFindings(checked)[slug+" internal/x.go"]
		if got != wantFinding {
			t.Fatalf("artifact-check %s: existence finding %v, want %v: %+v", slug, got, wantFinding, checked.Findings)
		}
	}
}

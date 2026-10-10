package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// dispatchInstance is a git repository with a sealed bundle whose sealed
// commit exists, as dispatch needs.
func dispatchInstance(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14"}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 6. Validation\n\n### Requirement trace\n- R1 [satisfied] test:TestFixture\n")
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\n")
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "alpha")
	head := git("rev-parse", "HEAD")
	graph := posemodel.DeliveryIntegrityGraph{
		SchemaVersion:    1,
		ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ChangeSets: []posemodel.ChangeSet{{
			ID: "cs-alpha", Spec: "alpha", Selector: "range:" + head + ".." + head, Base: head, Head: head, ResolvedBase: head, ResolvedHead: head,
			Paths:      []posemodel.ObservedPath{{Action: "modified", Path: "pose-mcp/lib.go"}},
			DiffDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		}},
		Deliveries:        []posemodel.DeliveryTarget{{Spec: "alpha", Ref: "contract:alpha-api", Kind: "contract", ID: "alpha-api", Module: "pose-mcp", Profile: "api-contract", Entrypoint: "pose-mcp/lib.go"}},
		ValidationResults: []posemodel.DeliveryValidationResult{{ID: "val-alpha", Module: "pose-mcp", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: head, ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}},
		Reverse:           map[string][]string{"pose-mcp/lib.go": {"alpha"}},
	}
	rawGraph, _ := json.Marshal(graph)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", string(rawGraph))
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal"); code != 0 {
		t.Fatalf("seal: %s", out)
	}
	return root
}

func writeReviewers(t *testing.T, root string, adapters map[string]posemodel.ReviewerAdapter) {
	t.Helper()
	raw, _ := json.Marshal(posemodel.ReviewersPolicy{SchemaVersion: 1, Adapters: adapters})
	writeCloseoutCLIFile(t, root, ".pose/policy/reviewers.json", string(raw))
}

// Spec pose-delegated-review-dispatch: a run reads the brief on stdin in a
// disposable worktree at the sealed commit, is recorded, and is failed when it
// edits files, times out, exits non-zero or exceeds its budget; dispatch never
// records an attestation and never touches the working tree.
func TestReviewDispatchRunsAndRecordsAdapters(t *testing.T) {
	root := dispatchInstance(t)

	// R6: no adapter → the brief and the configuration needed.
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "codex", "--apply"); code == 0 || !strings.Contains(out, "reviewers.json") || !strings.Contains(out, "# Independent review of spec:alpha") {
		t.Fatalf("no adapter: %d %s", code, out)
	}

	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"ok":     {Command: []string{"sh", "-c", "grep -q 'Independent review of spec:alpha' && git rev-parse HEAD > {output} && echo decision: approved >> {output}"}, Vendor: "test", Model: "ok-1", Timeout: "30s", MaxRunsPerBundle: 2},
		"fail":   {Command: []string{"sh", "-c", "cat >/dev/null; exit 3"}, Vendor: "test", Model: "fail-1", Timeout: "30s"},
		"slow":   {Command: []string{"sh", "-c", "cat >/dev/null; sleep 5"}, Vendor: "test", Model: "slow-1", Timeout: "300ms"},
		"editor": {Command: []string{"sh", "-c", "cat >/dev/null; echo changed >> pose-mcp/lib.go"}, Vendor: "test", Model: "edit-1", Timeout: "30s"},
		// Review tools regenerate derived state; that is not an edit.
		"tools": {Command: []string{"sh", "-c", "cat >/dev/null; mkdir -p .pose/assessments .pose/state && echo x > .pose/assessments/technical-debt.md && echo {} > .pose/state/technical-debt.json && echo decision: approved"}, Vendor: "test", Model: "tools-1", Timeout: "30s"},
	})
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "ok"); code != 0 || !strings.Contains(out, "review_dispatch.apply=false") {
		t.Fatalf("preview: %s", out)
	}
	before, _ := hashTree(root, true)

	code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "ok", "--apply")
	if code != 0 || !strings.Contains(out, "review_dispatch.status=completed") || !strings.Contains(out, "attestation=not recorded") {
		t.Fatalf("ok run: %d %s", code, out)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 1 || runs[0].Vendor != "test" || runs[0].Model != "ok-1" || !strings.HasPrefix(runs[0].BriefDigest, "sha256:") || !strings.HasPrefix(runs[0].TranscriptDigest, "sha256:") {
		t.Fatalf("run record: %+v", runs)
	}
	if !strings.Contains(runs[0].Output, runs[0].SealedCommit) || !strings.Contains(runs[0].Output, "decision: approved") {
		t.Fatalf("the reviewer did not run at the sealed commit with the brief: %q", runs[0].Output)
	}

	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "tools", "--apply"); code != 0 || !strings.Contains(out, "review_dispatch.status=completed") {
		t.Fatalf("a run that only regenerated derived state failed: %d %s", code, out)
	}
	for adapter, want := range map[string]string{"fail": "the reviewer command failed", "slow": "timeout", "editor": "changed files in the sealed worktree"} {
		code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", adapter, "--apply")
		if code == 0 || !strings.Contains(out, "review_dispatch.status=failed") || !strings.Contains(out, want) {
			t.Fatalf("%s: %d %s", adapter, code, out)
		}
	}
	// Budget: ok allows two runs on this bundle.
	runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "ok", "--apply")
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "ok", "--apply"); code == 0 || !strings.Contains(out, "budget exhausted") {
		t.Fatalf("budget: %d %s", code, out)
	}

	after, _ := hashTree(root, true)
	for path := range after {
		if strings.HasPrefix(path, ".pose/review-runs/") {
			delete(after, path)
		}
	}
	if changes := diffTrees(before, after); len(changes) != 0 {
		t.Fatalf("dispatch touched the working tree: %+v", changes)
	}
	if atts, _ := (posemodel.Store{Root: root}).ListReviewAttestations(""); len(atts) != 0 {
		t.Fatal("dispatch recorded an attestation")
	}
	if out, _ := exec.Command("git", "-C", root, "worktree", "list").Output(); strings.Count(string(out), "\n") != 1 {
		t.Fatalf("a review worktree was left behind: %s", out)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "review-runs")); len(entries) != 14 {
		t.Fatalf("expected 7 runs with transcripts, found %d files", len(entries))
	}
}

// R7: every brief kind dispatches through the same run record.
func TestReviewDispatchRecordsEveryBriefKind(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"smoke": {Command: []string{"sh", "-c", "grep -q '## Surfaces to run' && echo decision: approved"}, Vendor: "test", Model: "smoke-1", Timeout: "30s", MaxRunsPerBundle: 3},
		"adj":   {Command: []string{"sh", "-c", "grep -q 'Independent adjudication' && echo decision: approved"}, Vendor: "test", Model: "adj-1", Timeout: "30s"},
	})
	for adapter, kind := range map[string]string{"smoke": "smoke", "adj": "adjudication"} {
		if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", adapter, "--kind", kind, "--apply"); code != 0 || !strings.Contains(out, "review_dispatch.kind="+kind) || !strings.Contains(out, "review_dispatch.status=completed") {
			t.Fatalf("%s dispatch: %d %s", kind, code, out)
		}
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	kinds := map[string]bool{}
	for _, run := range runs {
		kinds[run.Kind] = true
	}
	if len(runs) != 2 || !kinds["smoke"] || !kinds["adjudication"] {
		t.Fatalf("runs: %+v", runs)
	}
}

// R4: the timeout holds when the adapter's child keeps running and holds its
// output — the shape of codex exec and claude -p (found in review).
func TestReviewDispatchTimeoutKillsTheAdaptersChildren(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"spawner": {Command: []string{"sh", "-c", "cat >/dev/null; sleep 30; true"}, Vendor: "test", Model: "spawn-1", Timeout: "300ms"},
	})
	start := time.Now()
	code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "spawner", "--apply")
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("the timeout did not hold: %s", elapsed)
	}
	if code == 0 || !strings.Contains(out, "review_dispatch.status=failed") || !strings.Contains(out, "timeout") {
		t.Fatalf("spawner: %d %s", code, out)
	}
}

// R4: a run that exits 0 leaving a child on its output completes, and a run
// that renames sealed content into derived state fails (found in review).
func TestReviewDispatchJudgesTheRunNotItsLeftovers(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{
		"lingers": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved; sleep 8 & exit 0"}, Vendor: "test", Model: "linger-1", Timeout: "60s"},
		"mover":   {Command: []string{"sh", "-c", "cat >/dev/null; mkdir -p .pose/state && git mv pose-mcp/lib.go .pose/state/lib.go"}, Vendor: "test", Model: "move-1", Timeout: "30s"},
	})
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "lingers", "--apply"); code != 0 || !strings.Contains(out, "review_dispatch.status=completed") {
		t.Fatalf("a finished run with a lingering child: %d %s", code, out)
	}
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "mover", "--apply"); code == 0 || !strings.Contains(out, "review_dispatch.status=failed") || !strings.Contains(out, "pose-mcp/lib.go") {
		t.Fatalf("a rename out of the sealed content: %d %s", code, out)
	}
}

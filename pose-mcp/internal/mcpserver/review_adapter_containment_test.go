package mcpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

// Exercise the existing MCP projection with an immutable, canonically identified
// historical run. Real execution and native signed binding are qualified by the
// domain/CLI fixtures, rather than fabricating an MCP dispatch tool.
func TestAdapterContainmentMCPReportsFailedCleanupWithoutApproval(t *testing.T) {
	root := t.TempDir()
	write := func(path string, data []byte) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/alpha.md", []byte("---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n# Alpha\n"))
	write(".pose/policy/review.json", []byte(`{"schema_version":1,"enabled":false,"profiles":{"spec":"spec-closeout@1"}}`))
	write(".pose/review-profiles/spec-closeout.json", []byte(`{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`))
	safety, err := pose.ResolveReviewerContainment(pose.ReviewerAdapter{Command: []string{"sh"}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}})
	if err != nil {
		t.Fatal(err)
	}
	safety.CaptureComplete = true
	safety.Operation = pose.NewOperationResult("review.dispatch.cleanup", "spec:alpha")
	safety.Operation.Fail(pose.OutcomeFailed, "cleanup", "cleanup-incomplete", "synthetic owned resources remain")
	output := "decision: approved\n"
	hash := sha256.Sum256([]byte(output))
	run := pose.ReviewRun{SchemaVersion: 1, Safety: safety, Scope: "spec:alpha", BundleID: "rvb-fixture", Adapter: "synthetic", StartedAt: "2026-10-26T00:00:00Z", EndedAt: "2026-10-26T00:00:01Z", Status: pose.ReviewRunCompleted, Output: output, TranscriptDigest: "sha256:" + hex.EncodeToString(hash[:])}
	raw, _ := json.Marshal(run)
	hash = sha256.Sum256(raw)
	run.RunID = "rrn-" + hex.EncodeToString(hash[:])[:16]
	run.TranscriptPath = ".pose/review-runs/" + run.RunID + ".transcript"
	raw, _ = json.Marshal(run)
	write(".pose/review-runs/"+run.RunID+".json", raw)
	write(run.TranscriptPath, []byte(output))
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	defer ts.Close()
	_, response := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_review_bundle","arguments":{"scope":"spec:alpha"}}}`)
	if response.Error != nil || response.Result["isError"] != false {
		t.Fatal(response)
	}
	raw, _ = json.Marshal(response.Result["structuredContent"])
	var v pose.ReviewBundleVerification
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.DelegatedRuns) != 1 || v.DelegatedRuns[0].Safety == nil || v.DelegatedRuns[0].Safety.Operation.Outcome != pose.OutcomeFailed || v.DelegatedRuns[0].DraftDecision != "unknown" {
		t.Fatal(string(raw))
	}
	_, response = post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_insights","arguments":{"group_by":"workflow","since_days":0}}}`)
	raw, _ = json.Marshal(response.Result["structuredContent"])
	var metrics pose.InsightsResult
	if err := json.Unmarshal(raw, &metrics); err != nil {
		t.Fatal(err)
	}
	if len(metrics.DelegatedReview.ByBundle) != 1 || metrics.DelegatedReview.ByBundle[0].ApprovedDrafts != 0 {
		t.Fatal(string(raw))
	}
}

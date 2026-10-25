package mcpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	pose "github.com/harne8/pose-mcp/internal/pose"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAttemptLedgerMCPDisclosesPriorRejectionAndDeclaredMetrics(t *testing.T) {
	root := t.TempDir()
	write := func(path, data string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/alpha.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n# Alpha\n")
	write(".pose/policy/review.json", `{"schema_version":1,"enabled":false,"profiles":{"spec":"spec-closeout@1"}}`)
	write(".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	for i, decision := range []string{"rejected", "approved"} {
		output := "decision: " + decision
		sum := sha256.Sum256([]byte(output))
		run := pose.ReviewRun{SchemaVersion: 1, Scope: "spec:alpha", BundleID: "rvb-" + strconv.Itoa(i), Adapter: "synthetic", StartedAt: "2026-10-25T00:00:0" + strconv.Itoa(i) + "Z", EndedAt: "2026-10-25T00:00:03Z", Status: pose.ReviewRunCompleted, Output: output, TranscriptDigest: "sha256:" + hex.EncodeToString(sum[:])}
		identity, _ := json.Marshal(run)
		sum = sha256.Sum256(identity)
		run.RunID = "rrn-" + hex.EncodeToString(sum[:])[:16]
		run.TranscriptPath = ".pose/review-runs/" + run.RunID + ".transcript"
		raw, _ := json.Marshal(run)
		write(".pose/review-runs/"+run.RunID+".json", string(raw))
		write(run.TranscriptPath, output)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_review_bundle","arguments":{"scope":"spec:alpha"}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("verify %+v", out)
	}
	raw, _ := json.Marshal(out.Result["structuredContent"])
	var verification pose.ReviewBundleVerification
	if err := json.Unmarshal(raw, &verification); err != nil {
		t.Fatal(err)
	}
	if len(verification.DelegatedRuns) != 2 || verification.DelegatedRuns[0].DraftDecision != "rejected" || verification.DelegatedRuns[1].Assurance != "declared-draft-not-attestation" {
		t.Fatalf("disclosure %s", raw)
	}
	_, out = post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_insights","arguments":{"group_by":"workflow","since_days":0}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("insights %+v", out)
	}
	raw, _ = json.Marshal(out.Result["structuredContent"])
	var metrics pose.InsightsResult
	if err := json.Unmarshal(raw, &metrics); err != nil {
		t.Fatal(err)
	}
	if metrics.DelegatedReview == nil || len(metrics.DelegatedReview.ByBundle) != 2 || metrics.DelegatedReview.ByAdapter[0].ApprovalRate == nil || *metrics.DelegatedReview.ByAdapter[0].ApprovalRate != 0.5 {
		t.Fatalf("metrics %s", raw)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/review-run-ledger")); !os.IsNotExist(err) {
		t.Fatal("legacy read invented witnesses")
	}
}

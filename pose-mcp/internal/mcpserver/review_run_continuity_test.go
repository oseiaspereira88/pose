package mcpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func TestRunContinuityMCPPartialNeverBecomesVerdict(t *testing.T) {
	root := t.TempDir()
	write := func(path, data string) {
		t.Helper()
		full := filepath.Join(root, path)
		if e := os.MkdirAll(filepath.Dir(full), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(full, []byte(data), 0644); e != nil {
			t.Fatal(e)
		}
	}
	write(".pose/specs/alpha.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n# Alpha\n")
	write(".pose/policy/review.json", `{"schema_version":1,"enabled":false,"profiles":{"spec":"spec-closeout@1"}}`)
	write(".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	output := "decision: rejected\n[private-session]"
	sum := sha256.Sum256([]byte(output))
	digest := "sha256:" + hex.EncodeToString(sum[:])
	run := pose.ReviewRun{SchemaVersion: 1, Scope: "spec:alpha", BundleID: "rvb-fixture", Adapter: "synthetic", StartedAt: "2026-10-25T00:00:00Z", EndedAt: "2026-10-25T00:00:01Z", Status: pose.ReviewRunInterrupted, Output: output, PartialDigest: digest, TranscriptDigest: digest, Interruption: &pose.ReviewInterruption{Cause: "usage-limit", Resumable: false, ResetAt: "2030-01-01T00:00:00Z"}}
	raw, _ := json.Marshal(run)
	sum = sha256.Sum256(raw)
	run.RunID = "rrn-" + hex.EncodeToString(sum[:])[:16]
	run.TranscriptPath = ".pose/review-runs/" + run.RunID + ".transcript"
	raw, _ = json.Marshal(run)
	write(".pose/review-runs/"+run.RunID+".json", string(raw))
	write(run.TranscriptPath, output)
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, response := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_review_bundle","arguments":{"scope":"spec:alpha"}}}`)
	if response.Error != nil || response.Result["isError"] != false {
		t.Fatal(response)
	}
	raw, _ = json.Marshal(response.Result["structuredContent"])
	var verification pose.ReviewBundleVerification
	if e := json.Unmarshal(raw, &verification); e != nil {
		t.Fatal(e)
	}
	if len(verification.DelegatedRuns) != 1 || !verification.DelegatedRuns[0].Partial || verification.DelegatedRuns[0].DraftDecision != "unknown" || verification.DelegatedRuns[0].Interruption.Cause != "usage-limit" || strings.Contains(string(raw), "session_digest") {
		t.Fatal(string(raw))
	}
	_, response = post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_insights","arguments":{"group_by":"workflow","since_days":0}}}`)
	if response.Error != nil || response.Result["isError"] != false {
		t.Fatal(response)
	}
	raw, _ = json.Marshal(response.Result["structuredContent"])
	var metrics pose.InsightsResult
	if e := json.Unmarshal(raw, &metrics); e != nil {
		t.Fatal(e)
	}
	row := metrics.DelegatedReview.ByBundle[0]
	if row.RejectingDrafts != 0 || row.ApprovedDrafts != 0 || row.UnknownDrafts != 1 || row.Resumes != 0 || row.Attempts != 1 {
		t.Fatal(row)
	}
}

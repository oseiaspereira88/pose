package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// MCP parity of spec pose-closeout-review-policy-by-cost-benefit: the MCP
// closeout state renders the same closeout-review@1 decision as the store the
// CLI reads, and reading writes nothing.
func TestCloseoutReviewPolicyParity(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/project.json", `{"schema_version":1,"project_id":"proj.fixture","name":"fixture"}`)
	write(".pose/policy/review.json", `{"schema_version":5,"enabled":true,"profiles":{"spec":"spec-closeout@1","milestone":"milestone-integration@1","roadmap":"roadmap-outcome@1"},"milestone_review_required":"when-justified","roadmap_review_required":"when-justified"}`)
	write(".pose/review-profiles/milestone-integration.json", `{"schema_version":1,"id":"milestone-integration","version":1,"scope":"milestone","criteria":[{"id":"integration","description":"The milestone is integrated.","kind":"judgment","evidence_classes":["integration"]}]}`)
	write(".pose/review-profiles/roadmap-outcome.json", `{"schema_version":1,"id":"roadmap-outcome","version":1,"scope":"roadmap","criteria":[{"id":"outcome","description":"The roadmap outcome is verified.","kind":"judgment","evidence_classes":["integration"]}]}`)
	write(".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"The spec is correct.","kind":"judgment","evidence_classes":["integration"]}]}`)
	for _, slug := range []string{"a", "b", "c"} {
		write(".pose/specs/"+slug+".md", "---\nslug: "+slug+"\nstatus: done\ncreated_at: 2026-11-01\n---\n\n# Spec: "+slug+"\n")
	}
	write(".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: small\n- specs: a, spec:a\n\n## Milestone: large\n- specs: a, b, c\n")
	store := pose.Store{Root: root}
	server := New(store)
	ts := httptest.NewServer(server.Handler("", ""))
	t.Cleanup(ts.Close)
	for _, ref := range []string{"milestone:program/small", "milestone:program/large", "roadmap:program"} {
		t.Run(ref, func(t *testing.T) {
			want, err := store.EvaluateCloseoutReview(ref)
			if err != nil {
				t.Fatal(err)
			}
			_, response := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"`+ref+`"}}}`)
			if response.Error != nil || response.Result["isError"] == true {
				t.Fatalf("error=%+v result=%v", response.Error, response.Result)
			}
			structured, _ := response.Result["structuredContent"].(map[string]any)
			review, _ := structured["review"].(map[string]any)
			raw, _ := json.Marshal(review["closeout_review"])
			var got pose.CloseoutReviewDecision
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("no decision in %v", review)
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("MCP decision %+v differs from the store's %+v", got, want)
			}
			if review["required"] != want.Required {
				t.Fatalf("required %v, want %v", review["required"], want.Required)
			}
			if structured["terminal"] == true {
				t.Fatalf("an unclosed scope reads as terminal over MCP: %v", structured)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "review-closeouts")); !os.IsNotExist(err) {
		t.Fatalf("an MCP read wrote closeout records: %v", err)
	}
}

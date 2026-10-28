package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func actionFulfilCall(t *testing.T, ts *httptest.Server, args map[string]any) rpcResult {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_action_fulfil", "arguments": args}})
	_, out := post(t, ts, string(raw))
	return out
}

// Spec pose-delegated-review-action-fulfillment R9, R11 over MCP: the same
// envelope as the CLI and a CI job, verified by the same engine, recorded
// on the mcp channel; a preview records nothing.
func TestToolsCall_ActionFulfil(t *testing.T) {
	_, s, key, bundle, run := mcpRunFixture(t)
	m, err := s.PrepareDelegatedReviewMaterial("spec:alpha", pose.DelegationPurposeReview, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := s.OpenActionRequest(pose.ActionRequest{Origin: "spec:alpha", Kind: pose.ActionDelegatedReview, Question: "Review spec:alpha",
		RequestedBy: pose.ActionPrincipal{Principal: "agent:implementer"}, Targets: []pose.NodeRef{{}},
		Effects: []pose.ObligationEffect{{Phase: pose.PhaseCloseout, Mode: pose.EffectBlock}}, Review: &m}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if c, err := s.ConcludeReviewRun(run.RunID, key.Name, true, time.Now()); err != nil || c.State != pose.ReviewConclusionRecorded {
		t.Fatalf("conclude: %+v %v (bundle %s)", c, err, bundle.BundleID)
	}
	ts := httptest.NewServer(New(s).Handler("", ""))
	t.Cleanup(ts.Close)
	args := map[string]any{"id": opened.Request.ID, "run": run.RunID, "runner": "agent:conductor", "request_digest": opened.Request.RequestDigest,
		"expected_revision": opened.Revision, "idempotency_key": "mcp-1"}
	preview := actionFulfilCall(t, ts, args)
	if preview.Error != nil || preview.Result["isError"] != false || !strings.Contains(toolText(preview), `"preview":true`) {
		t.Fatalf("preview: %+v", preview)
	}
	if v, _ := s.LoadActionRequest(opened.Request.ID); v.State != pose.ActionStateOpen {
		t.Fatalf("a preview recorded: %s", v.State)
	}
	args["apply"] = true
	applied := actionFulfilCall(t, ts, args)
	if applied.Error != nil || applied.Result["isError"] != false {
		t.Fatalf("apply: %+v", applied)
	}
	v, err := s.LoadActionRequest(opened.Request.ID)
	if err != nil || v.State != pose.ActionStateFulfilled || v.Events[len(v.Events)-1].Channel != "mcp" {
		t.Fatalf("after: %+v %v", v, err)
	}
	// The canonical envelope, as the Conductor writes it, replays the same
	// fulfilment (review round 1, F08).
	envelope := actionFulfilCall(t, ts, map[string]any{"apply": true, "envelope": map[string]any{"request_id": opened.Request.ID, "request_digest": opened.Request.RequestDigest,
		"expected_revision": opened.Revision, "idempotency_key": "mcp-1", "run_id": run.RunID, "runner": "agent:conductor", "channel": "mcp"}})
	if envelope.Error != nil || envelope.Result["isError"] != false {
		t.Fatalf("the canonical envelope: %+v", envelope)
	}
	args["runner"] = "human:ada"
	args["idempotency_key"] = "mcp-2"
	if refused := actionFulfilCall(t, ts, args); refused.Result["isError"] != true {
		t.Fatalf("a person was recorded as a runner over MCP: %+v", refused)
	}
}

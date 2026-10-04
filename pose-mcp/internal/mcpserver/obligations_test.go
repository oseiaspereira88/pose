package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-state-attention: the MCP tool returns the same projection as the
// CLI for the same tree, with coverage and attention.
func TestToolsCall_Obligations_ReturnsReportWithCoverage(t *testing.T) {
	root := t.TempDir()
	for rel, body := range map[string]string{
		".pose/specs/2026-10-01-base.md":     "---\nslug: base\nstatus: draft\n---\n\n# Spec: base\n",
		".pose/specs/2026-10-02-consumer.md": "---\nslug: consumer\nstatus: draft\ndepends_on: base\n---\n\n# Spec: consumer\n",
	} {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_obligations","arguments":{"kind":"dependency"}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	obligations, _ := sc["obligations"].([]any)
	coverage, _ := sc["coverage"].([]any)
	if len(obligations) != 1 || len(coverage) == 0 || sc["complete"] != false || sc["attention"] == nil {
		t.Fatalf("pose_obligations: %+v", sc)
	}
	direct, err := pose.Store{Root: root}.ProjectObligations(pose.ObligationQuery{Category: pose.ObligationDependency})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := obligations[0].(map[string]any)
	if len(direct.Obligations) != 1 || first["id"] != direct.Obligations[0].ID {
		t.Fatalf("MCP and the domain disagree on ids: %+v vs %+v", first, direct.Obligations)
	}
}

// Spec pose-phase-scoped-readiness: phases are an opt-in block beside the
// legacy `ready`.
func TestToolsCall_SpecReadiness_PhasesBesideLegacyReady(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, ".pose/specs/2026-10-01-base.md")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("---\nslug: base\nstatus: draft\n---\n\n# Spec: base\n"), 0o644)
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, plain := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_spec_readiness","arguments":{"slug":"base"}}}`)
	sc, _ := plain.Result["structuredContent"].(map[string]any)
	if _, has := sc["phases"]; has || sc["ready"] != true {
		t.Fatalf("plain readiness: %+v", sc)
	}
	_, withPhases := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_spec_readiness","arguments":{"slug":"base","phases":true}}}`)
	sc, _ = withPhases.Result["structuredContent"].(map[string]any)
	phases, _ := sc["phases"].([]any)
	if sc["ready"] != true || len(phases) != 5 {
		t.Fatalf("readiness with phases: %+v", sc)
	}
}

// Spec pose-governed-effect-enforcement: MCP closeout state carries the same
// refusal cause as the CLI, from the same domain function.
func TestToolsCall_CloseoutState_CarriesTheGovernedEffect(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(body), 0o644)
	}
	write(".pose/specs/2026-10-04-storage.md", "---\nslug: storage\nstatus: in-progress\n---\n\n# Spec: storage\n\n- R4: x\n")
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"},"agency_readiness_version":1}`)
	store := pose.Store{Root: root}
	if _, err := store.OpenActionRequest(pose.ActionRequest{Origin: "spec:storage", Kind: pose.ActionApproval, Question: "Close?", RequestedBy: pose.ActionPrincipal{Principal: "agent:a"},
		Targets: []pose.NodeRef{{Artifact: "self"}}, Effects: []pose.ObligationEffect{{Phase: pose.PhaseCloseout, Mode: pose.EffectBlock}}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(store).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:storage"}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	diagnostics, _ := sc["diagnostics"].([]any)
	found := false
	for _, raw := range diagnostics {
		d, _ := raw.(map[string]any)
		found = found || d["code"] == "action-request-pending"
	}
	if !found || sc["terminal"] != false {
		t.Fatalf("MCP closeout state lacks the governed effect: %+v", sc)
	}
}

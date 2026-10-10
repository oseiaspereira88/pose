package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-mcp-action-resolve-signed-only: an agent relays an answer over
// MCP only with the principal's proof.

func actionResolveFixture(t *testing.T, actionsPolicy string) (string, *httptest.Server, pose.ActionRequestView) {
	t.Helper()
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.storage-test")
	root := t.TempDir()
	for rel, body := range map[string]string{
		".pose/specs/2026-10-04-storage.md": "---\nslug: storage\nstatus: in-progress\n---\n\n# Spec: storage\n\n## 2. Requirements\n\n- R4: Keep reading v1.\n",
		".pose/policy/actions.json":         actionsPolicy,
	} {
		p := filepath.Join(root, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := pose.Store{Root: root}
	view, err := store.OpenActionRequest(pose.ActionRequest{Origin: "spec:storage", Kind: pose.ActionDecision, Question: "Keep reading v1?",
		RequestedBy: pose.ActionPrincipal{Principal: "agent:impl"}, Recipient: pose.ObligationActor{Role: "maintainer"},
		Options: []pose.ActionOption{{ID: "preserve-v1", Consequence: "Keep"}, {ID: "break-v1", Consequence: "Migrate"}},
		Targets: []pose.NodeRef{{Artifact: "self", Kind: "requirement", ID: "R4"}}, Effects: []pose.ObligationEffect{{Phase: pose.PhaseCloseout, Mode: pose.EffectBlock}}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(store).Handler("", ""))
	t.Cleanup(ts.Close)
	return root, ts, view
}

func resolveCall(t *testing.T, ts *httptest.Server, view pose.ActionRequestView, extra map[string]any) rpcResult {
	t.Helper()
	args := map[string]any{"id": view.Request.ID, "actor": "human:maintainer", "answer": "preserve-v1", "request_digest": view.Request.RequestDigest,
		"expected_revision": view.Revision, "idempotency_key": "k1"}
	for k, v := range extra {
		args[k] = v
	}
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_action_resolve", "arguments": args}})
	_, out := post(t, ts, string(raw))
	return out
}

func TestToolsCall_ActionResolve_RefusesAnAnswerWithoutProof(t *testing.T) {
	// Even under declared assurance, where the CLI accepts a declaration.
	root, ts, view := actionResolveFixture(t, `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"identity_assurance":"declared"}`)
	preview := resolveCall(t, ts, view, nil)
	sc, _ := preview.Result["structuredContent"].(map[string]any)
	statement, _ := sc["statement"].(string)
	if sc["preview"] != true || !strings.Contains(statement, view.Request.RequestDigest) || !strings.Contains(sc["sign_command"].(string), "-n pose-action-answer") {
		t.Fatalf("the preview does not give the statement to sign: %+v", preview)
	}
	want := string(pose.AnswerStatement(view.Request, "human:maintainer", "preserve-v1", "k1", "").Canonical())
	if statement != want {
		t.Fatalf("preview statement differs from the CLI's:\n%s\n%s", statement, want)
	}
	out := resolveCall(t, ts, view, map[string]any{"apply": true})
	if out.Result["isError"] != true || !strings.Contains(toolText(out), "proof") {
		t.Fatalf("a declared answer was recorded over MCP: %+v", out)
	}
	out = resolveCall(t, ts, view, map[string]any{"apply": true, "signature": "-----BEGIN SSH SIGNATURE-----\nAAAA\n-----END SSH SIGNATURE-----\n"})
	if out.Result["isError"] != true {
		t.Fatalf("a bad signature was recorded: %+v", out)
	}
	after, _ := pose.Store{Root: root}.LoadActionRequest(view.Request.ID)
	if after.State != pose.ActionStateOpen || after.Revision != 1 {
		t.Fatalf("a refused answer was recorded: %+v", after)
	}
}

func TestToolsCall_ActionResolve_RecordsASignedAnswer(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen is not installed")
	}
	key := filepath.Join(t.TempDir(), "id_ed25519")
	if out, err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v %s", err, out)
	}
	public, _ := os.ReadFile(key + ".pub")
	policy, _ := json.Marshal(map[string]any{"schema_version": 1, "roles": map[string][]string{"maintainer": {"human:maintainer"}}, "identity_assurance": "verified",
		"keys": map[string][]map[string]string{"human:maintainer": {{"key": strings.TrimSpace(string(public))}}}})
	root, ts, view := actionResolveFixture(t, string(policy))
	sc, _ := resolveCall(t, ts, view, nil).Result["structuredContent"].(map[string]any)
	sign := exec.Command(keygen, "-Y", "sign", "-f", key, "-n", pose.ActionAnswerNamespace)
	sign.Stdin = strings.NewReader(sc["statement"].(string))
	signature, err := sign.Output()
	if err != nil {
		t.Fatal(err)
	}
	out := resolveCall(t, ts, view, map[string]any{"apply": true, "signature": string(signature)})
	if out.Result["isError"] == true {
		t.Fatalf("a signed answer was refused: %+v", out)
	}
	after, _ := pose.Store{Root: root}.LoadActionRequest(view.Request.ID)
	if after.State != pose.ActionStateAnswered || after.Assurance != pose.ReviewIdentityAssuranceVerified {
		t.Fatalf("not recorded as a verified answer: %+v", after)
	}
	last := after.Events[len(after.Events)-1]
	if last.Channel != "mcp" || last.SSHSignature == nil || pose.VerifyRecordedActionSignature(after.Request, last) != nil {
		t.Fatalf("the recorded answer does not carry a re-verifiable signature: %+v", last)
	}
}

func toolText(out rpcResult) string {
	raw, _ := json.Marshal(out.Result)
	return string(raw)
}

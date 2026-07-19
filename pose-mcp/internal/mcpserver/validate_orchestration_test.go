package mcpserver

// Safe validation orchestration behavior (spec pose-safe-validate-orchestration):
// plan resolution, digest-bound approval, identity-mandatory authorization,
// idempotent submission, and the threat-test scenarios the spec calls out —
// substitution, replay, cancellation and unconfigured-harness result spoofing.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpenforce "github.com/harne8/mcp-enforce"
	"github.com/harne8/pose-mcp/internal/pose"
)

func orchFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	matrix := `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"test","program":"true","severity":"required"}]}}}`
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// stubExecutor counts Submit calls and returns a fixed execution id, so
// idempotency (no double-submit) is directly observable.
type stubExecutor struct {
	calls int
	id    string
	err   error
}

func (s *stubExecutor) Submit(_ context.Context, _ ApprovedValidationRequest) (string, error) {
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	return s.id, nil
}

func TestOrchestrationRequestResolvesImmutablePlan(t *testing.T) {
	root := orchFixture(t)
	o := newOrchestrator()
	req, err := o.request(root, "proj.a", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if req.State != statePendingApproval || req.Plan.Digest == "" || req.Plan.MatrixSHA256 == "" {
		t.Fatalf("request = %+v", req)
	}
	// Same inputs, same tree -> same digest (deterministic, R1).
	req2, err := o.request(root, "proj.a", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if req2.Plan.Digest != req.Plan.Digest {
		t.Error("identical inputs must produce identical plan digests")
	}
	if req2.ID == req.ID {
		t.Error("each request call must mint a fresh request_id even with an identical plan")
	}
}

func TestOrchestrationApprovalRequiresMatchingDigest_SubstitutionRejected(t *testing.T) {
	root := orchFixture(t)
	o := newOrchestrator()
	req, _ := o.request(root, "proj.a", "", "", "", "")
	_, err := o.approve(req.ID, "wrong-digest", "approve", "looks fine", "run.1", nil)
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("substitution (wrong digest) must be rejected, got: %v", err)
	}
	got, _ := o.status(req.ID)
	if got.State != statePendingApproval {
		t.Errorf("a rejected approval attempt must not change state, got %s", got.State)
	}
}

func TestOrchestrationApprovalNoReplayOfDecidedRequest(t *testing.T) {
	root := orchFixture(t)
	o := newOrchestrator()
	req, _ := o.request(root, "proj.a", "", "", "", "")
	if _, err := o.approve(req.ID, req.Plan.Digest, "approve", "ok", "run.1", nil); err != nil {
		t.Fatal(err)
	}
	// Replay: approving (or rejecting) an already-decided request must fail.
	if _, err := o.approve(req.ID, req.Plan.Digest, "approve", "ok again", "run.2", nil); err == nil {
		t.Fatal("re-approving an already-approved request must be rejected (no replay)")
	}
	if _, err := o.approve(req.ID, req.Plan.Digest, "reject", "changed my mind", "run.2", nil); err == nil {
		t.Fatal("rejecting an already-approved request must be rejected (no replay)")
	}
}

func TestOrchestrationSubmitRequiresApprovalAndIsIdempotent(t *testing.T) {
	root := orchFixture(t)
	o := newOrchestrator()
	req, _ := o.request(root, "proj.a", "", "", "", "")
	exec := &stubExecutor{id: "exec-123"}

	if _, err := o.submit(context.Background(), req.ID, exec); err == nil {
		t.Fatal("submitting a non-approved request must fail")
	}
	if exec.calls != 0 {
		t.Fatalf("executor must not be invoked before approval, calls=%d", exec.calls)
	}

	if _, err := o.approve(req.ID, req.Plan.Digest, "approve", "ok", "run.1", []string{"validate:submit"}); err != nil {
		t.Fatal(err)
	}
	got, err := o.submit(context.Background(), req.ID, exec)
	if err != nil || got.ExecutionID != "exec-123" || got.State != stateSubmitted {
		t.Fatalf("submit = %+v, err=%v", got, err)
	}
	// Idempotency: resubmitting must not call the executor again.
	got2, err := o.submit(context.Background(), req.ID, exec)
	if err != nil || got2.ExecutionID != "exec-123" {
		t.Fatalf("idempotent resubmit = %+v, err=%v", got2, err)
	}
	if exec.calls != 1 {
		t.Errorf("executor.Submit called %d times, want exactly 1 (idempotency)", exec.calls)
	}
}

func TestOrchestrationRejectedRequestCannotBeSubmitted(t *testing.T) {
	root := orchFixture(t)
	o := newOrchestrator()
	req, _ := o.request(root, "proj.a", "", "", "", "")
	if _, err := o.approve(req.ID, req.Plan.Digest, "reject", "no", "run.1", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := o.submit(context.Background(), req.ID, &stubExecutor{id: "x"}); err == nil {
		t.Fatal("a rejected request must never be submittable")
	}
}

func TestOrchestrationCancellation(t *testing.T) {
	root := orchFixture(t)
	o := newOrchestrator()
	req, _ := o.request(root, "proj.a", "", "", "", "")
	got, err := o.cancel(req.ID, "no longer needed")
	if err != nil || got.State != stateCancelled {
		t.Fatalf("cancel = %+v, err=%v", got, err)
	}
	// Cancelling an already-terminal request is rejected, not silently reapplied.
	if _, err := o.cancel(req.ID, "again"); err == nil {
		t.Fatal("cancelling an already-cancelled request must fail")
	}
	if _, err := o.approve(req.ID, req.Plan.Digest, "approve", "too late", "run.1", nil); err == nil {
		t.Fatal("a cancelled request must not become approvable")
	}
}

func TestOrchestrationUnknownRequestID(t *testing.T) {
	o := newOrchestrator()
	if _, err := o.status("ghost"); err != errValidationRequestNotFound {
		t.Fatalf("status of unknown id: %v", err)
	}
	if _, err := o.approve("ghost", "d", "approve", "r", "run.1", nil); err != errValidationRequestNotFound {
		t.Fatalf("approve of unknown id: %v", err)
	}
}

// --- MCP-level tests: identity enforcement, harness config, catalog --------

func orchServer(t *testing.T, secret []byte) (*Server, string) {
	t.Helper()
	root := orchFixture(t)
	s := New(pose.Store{Root: root}).WithIdentitySecret(secret)
	return s, root
}

func TestApproveDeniesAnonymousCaller(t *testing.T) {
	s, _ := orchServer(t, nil) // no identity secret configured at all
	out := postToServer(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_validate_request","arguments":{}}}`)
	requestID, digest := extractPlan(t, out)
	out2 := postToServer(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_validate_approve","arguments":{"request_id":`+quote(requestID)+`,"plan_digest":`+quote(digest)+`,"decision":"approve"}}}`)
	isErr, _ := out2.Result["isError"].(bool)
	if !isErr {
		t.Fatal("approval without any bound Execution Identity must be denied")
	}
	content, _ := out2.Result["content"].([]any)
	text, _ := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "bound Execution Identity") {
		t.Errorf("expected identity-required diagnostic, got: %s", text)
	}
}

func TestApproveWithValidIdentitySucceeds(t *testing.T) {
	secret := []byte("test-secret-32-bytes-minimum!!!!")
	s, _ := orchServer(t, secret)
	ts := newHTTPTestServer(t, s)

	reqOut := postHTTP(t, ts, "", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_validate_request","arguments":{}}}`)
	requestID, digest := extractPlan(t, reqOut)

	tok, err := mcpenforce.MintToken(mcpenforce.Identity{RunID: "run.42", ProjectID: "", Scopes: []string{"validate:approve"}}, secret)
	if err != nil {
		t.Fatal(err)
	}
	approveOut := postHTTP(t, ts, tok, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_validate_approve","arguments":{"request_id":`+quote(requestID)+`,"plan_digest":`+quote(digest)+`,"decision":"approve"}}}`)
	isErr, _ := approveOut.Result["isError"].(bool)
	if isErr {
		content, _ := approveOut.Result["content"].([]any)
		t.Fatalf("approval with a valid identity must succeed: %v", content)
	}
	sc, _ := approveOut.Result["structuredContent"].(map[string]any)
	if sc["state"] != "approved" || sc["approver_run_id"] != "run.42" {
		t.Errorf("structuredContent = %+v", sc)
	}
}

func TestSubmitWithoutHarnessConfiguredIsAConfigError(t *testing.T) {
	secret := []byte("test-secret-32-bytes-minimum!!!!")
	s, _ := orchServer(t, secret)
	ts := newHTTPTestServer(t, s) // no WithHarnessExecutor call

	reqOut := postHTTP(t, ts, "", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_validate_request","arguments":{}}}`)
	requestID, digest := extractPlan(t, reqOut)
	tok, _ := mcpenforce.MintToken(mcpenforce.Identity{RunID: "run.1", Scopes: []string{"validate:approve"}}, secret)
	postHTTP(t, ts, tok, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_validate_approve","arguments":{"request_id":`+quote(requestID)+`,"plan_digest":`+quote(digest)+`,"decision":"approve"}}}`)

	submitOut := postHTTP(t, ts, "", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"pose_validate_submit","arguments":{"request_id":`+quote(requestID)+`}}}`)
	isErr, _ := submitOut.Result["isError"].(bool)
	if !isErr {
		t.Fatal("submit without a configured Harness executor must be a tool error, never a silent no-op success")
	}
	content, _ := submitOut.Result["content"].([]any)
	text, _ := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "not configured") {
		t.Errorf("expected 'not configured' diagnostic, got: %s", text)
	}
}

func TestValidateOrchestrationToolsInCatalog(t *testing.T) {
	want := map[string]bool{
		"pose_validate_request": true, "pose_validate_approve": true,
		"pose_validate_submit": true, "pose_validate_status": true, "pose_validate_cancel": true,
	}
	found := map[string]bool{}
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		if want[name] {
			found[name] = true
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("tool %q missing from catalog", name)
		}
	}
}

// --- small helpers -----------------------------------------------------

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func extractPlan(t *testing.T, out rpcResult) (requestID, digest string) {
	t.Helper()
	sc, _ := out.Result["structuredContent"].(map[string]any)
	requestID, _ = sc["request_id"].(string)
	plan, _ := sc["plan"].(map[string]any)
	digest, _ = plan["digest"].(string)
	if requestID == "" || digest == "" {
		t.Fatalf("could not extract plan from result: %+v", out.Result)
	}
	return
}

func newHTTPTestServer(t *testing.T, s *Server) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(s.Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

// postHTTP posts body to ts, optionally carrying an Execution Identity token
// (ADR-007) — the mechanism is HTTP-header-only, so orchestration approval
// tests must go through a real HTTP transport rather than the stdio path.
func postHTTP(t *testing.T, ts *httptest.Server, identityToken, body string) rpcResult {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if identityToken != "" {
		req.Header.Set(mcpenforce.IdentityHeader, identityToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	var out rpcResult
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
	}
	return out
}

func postToServer(t *testing.T, s *Server, body string) rpcResult {
	t.Helper()
	ts := newHTTPTestServer(t, s)
	return postHTTP(t, ts, "", body)
}

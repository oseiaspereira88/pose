package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// The PolicyGate / PolicyConfig / PolicyInput / audit logic now lives in the
// shared mcp-enforce module and is unit-tested there. What remains here is the
// WIRING: proof that the server actually routes tools/call through the gate and
// surfaces a denial as a -32004 JSON-RPC error carrying the policy metadata.

// opaFixture starts a minimal OPA-compatible HTTP stub that returns body
// (JSON-encoded) with statusCode on any POST /v1/data/... request.
func opaFixture(t *testing.T, statusCode int, body any) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

// TestPolicyGate_Integration_DeniedReturns32004 exercises the full server path:
// OPA denies → -32004 error with policy metadata in error.data.
func TestPolicyGate_Integration_DeniedReturns32004(t *testing.T) {
	stub := opaFixture(t, http.StatusOK, map[string]any{
		"result": map[string]any{
			"allow":      false,
			"violations": []string{"principal_not_authorized"},
		},
	})
	g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub.Client()})

	tmpRoot := t.TempDir()
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: tmpRoot})
	ts := httptest.NewServer(NewWithRootsAndPolicy(roots, g).Handler("", ""))
	t.Cleanup(ts.Close)

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":99,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{}}}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MCP-Principal", "eve")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out struct {
		Error *struct {
			Code int    `json:"code"`
			Msg  string `json:"message"`
			Data any    `json:"data"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error == nil {
		t.Fatal("expected error response, got nil")
	}
	if out.Error.Code != -32004 {
		t.Errorf("error.code = %d, want -32004", out.Error.Code)
	}
	if out.Error.Data == nil {
		t.Error("error.data must not be nil for policy denial")
	}
}

package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mcpenforce "github.com/harne8/mcp-enforce"

	"github.com/harne8/pose-mcp/internal/pose"
)

// TestToolsCall_Identity_Enforcement exercises the full HTTP path with identity
// binding: RequireIdentity denies anonymous and invalid tokens, and accepts a
// valid, unexpired run-bound identity.
func TestToolsCall_Identity_Enforcement(t *testing.T) {
	secret := []byte("sek")
	frozen := time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC)
	gate := NewPolicyGate(PolicyConfig{RequireIdentity: true, Clock: func() time.Time { return frozen }})
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: t.TempDir()})
	srv := NewWithRootsAndPolicy(roots, gate).WithIdentitySecret(secret)
	ts := httptest.NewServer(srv.Handler("", ""))
	t.Cleanup(ts.Close)

	call := func(token string) int {
		body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{}}}`)
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-MCP-Principal", "svc.worker")
		if token != "" {
			req.Header.Set(mcpenforce.IdentityHeader, token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct {
			Error *struct {
				Code int `json:"code"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out.Error != nil {
			return out.Error.Code
		}
		return 0
	}

	if code := call(""); code != -32004 {
		t.Errorf("no identity with RequireIdentity: code = %d, want -32004", code)
	}
	if code := call("garbage.sig"); code != -32004 {
		t.Errorf("invalid identity: code = %d, want -32004", code)
	}
	tok, err := mcpenforce.MintToken(mcpenforce.Identity{
		RunID: "run-1", ProjectID: "proj.a", ExpiresAt: frozen.Add(time.Hour),
	}, secret)
	if err != nil {
		t.Fatal(err)
	}
	if code := call(tok); code == -32004 {
		t.Error("valid run-bound identity was policy-denied, want allow")
	}
}

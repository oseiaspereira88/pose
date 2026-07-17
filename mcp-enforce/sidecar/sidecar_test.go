package sidecar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	mcpenforce "github.com/crisol/mcp-enforce"
)

// capturingRecorder collects exchanges for assertions.
type capturingRecorder struct {
	mu        sync.Mutex
	exchanges []Exchange
}

func (c *capturingRecorder) Record(_ context.Context, ex Exchange) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.exchanges = append(c.exchanges, ex)
}

func (c *capturingRecorder) all() []Exchange {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Exchange(nil), c.exchanges...)
}

// fakeUpstream records whether it was hit and serves a canned JSON response.
type fakeUpstream struct {
	hit          bool
	gotPrincipal string
	server       *httptest.Server
}

func newFakeUpstream(t *testing.T) *fakeUpstream {
	t.Helper()
	fu := &fakeUpstream{}
	fu.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fu.hit = true
		fu.gotPrincipal = r.Header.Get("X-MCP-Principal")
		w.Header().Set("Mcp-Session-Id", "sess-xyz")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{"ok": true}})
	}))
	t.Cleanup(fu.server.Close)
	return fu
}

func newSidecar(t *testing.T, gate *mcpenforce.PolicyGate, upstreamURL string) *Sidecar {
	t.Helper()
	u, err := url.Parse(upstreamURL)
	if err != nil {
		t.Fatal(err)
	}
	return New(Config{Gate: gate, Upstream: u})
}

func post(t *testing.T, h http.Handler, body, principal string) *http.Response {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if principal != "" {
		req.Header.Set("X-MCP-Principal", principal)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeRPCError(t *testing.T, resp *http.Response) (code int, hasData bool) {
	t.Helper()
	var out struct {
		Error *struct {
			Code int `json:"code"`
			Data any `json:"data"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Error == nil {
		return 0, false
	}
	return out.Error.Code, out.Error.Data != nil
}

func TestSidecar_AllowsAndForwards_PreservingPrincipalAndSession(t *testing.T) {
	up := newFakeUpstream(t)
	sc := newSidecar(t, mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{}), up.server.URL)

	resp := post(t, sc, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_query","arguments":{"project_id":"proj.a"}}}`, "svc.worker")
	defer resp.Body.Close()

	if !up.hit {
		t.Fatal("allowed tools/call was not forwarded upstream")
	}
	if up.gotPrincipal != "svc.worker" {
		t.Errorf("upstream principal = %q, want forwarded \"svc.worker\"", up.gotPrincipal)
	}
	if got := resp.Header.Get("Mcp-Session-Id"); got != "sess-xyz" {
		t.Errorf("session header = %q, want passthrough \"sess-xyz\"", got)
	}
}

func TestSidecar_ForwardsAggregateProjectScopeToOPA(t *testing.T) {
	var captured []string
	opa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input struct {
				ProjectIDs []string `json:"project_ids"`
			} `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		captured = body.Input.ProjectIDs
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"allow": true, "violations": []string{}},
		})
	}))
	t.Cleanup(opa.Close)

	up := newFakeUpstream(t)
	gate := mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{OPAURL: opa.URL, HTTPClient: opa.Client()})
	sc := newSidecar(t, gate, up.server.URL)
	resp := post(t, sc, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"analytics_query","arguments":{"project_ids":["proj.b","proj.a","proj.b"]}}}`, "svc.portal")
	resp.Body.Close()

	want := []string{"proj.b", "proj.a"}
	if !up.hit || len(captured) != len(want) || captured[0] != want[0] || captured[1] != want[1] {
		t.Fatalf("forwarded=%v OPA project_ids=%#v, want %#v", up.hit, captured, want)
	}
}

func TestSidecar_RequirePrincipal_DeniesAndDoesNotForward(t *testing.T) {
	up := newFakeUpstream(t)
	sc := newSidecar(t, mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{RequirePrincipal: true}), up.server.URL)

	resp := post(t, sc, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"graph_query","arguments":{}}}`, "")
	defer resp.Body.Close()

	code, hasData := decodeRPCError(t, resp)
	if code != -32004 {
		t.Errorf("error.code = %d, want -32004", code)
	}
	if !hasData {
		t.Error("policy denial must carry error.data")
	}
	if up.hit {
		t.Error("denied request must NOT reach upstream")
	}
}

func TestSidecar_NonToolsCall_PassesThrough(t *testing.T) {
	up := newFakeUpstream(t)
	// RequirePrincipal is on, but a non-tools/call request is not gated.
	sc := newSidecar(t, mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{RequirePrincipal: true}), up.server.URL)

	resp := post(t, sc, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, "")
	defer resp.Body.Close()

	if !up.hit {
		t.Error("non-tools/call request should pass through to upstream")
	}
}

func TestSidecar_BatchToolsCall_Denied(t *testing.T) {
	up := newFakeUpstream(t)
	sc := newSidecar(t, mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{}), up.server.URL)

	resp := post(t, sc, `[{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_query"}}]`, "svc.worker")
	defer resp.Body.Close()

	code, _ := decodeRPCError(t, resp)
	if code != -32004 {
		t.Errorf("batch tools/call error.code = %d, want -32004", code)
	}
	if up.hit {
		t.Error("batched tools/call must NOT reach upstream")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// postIdentity posts a tools/call carrying an Execution Identity token.
func postIdentity(t *testing.T, h http.Handler, body, token string) *http.Response {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(mcpenforce.IdentityHeader, token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

const toolsCallBody = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_query"}}`

func TestSidecar_Identity_ValidForwards(t *testing.T) {
	secret := []byte("sek")
	up := newFakeUpstream(t)
	gate := mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{
		RequireIdentity: true,
		Clock:           func() time.Time { return time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC) },
	})
	sc := New(Config{Gate: gate, Upstream: mustURL(t, up.server.URL), IdentitySecret: secret})

	tok, _ := mcpenforce.MintToken(mcpenforce.Identity{
		RunID: "run-1", ProjectID: "proj.a", Scopes: []string{"repo:read"},
		ExpiresAt: time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
	}, secret)
	resp := postIdentity(t, sc, toolsCallBody, tok)
	defer resp.Body.Close()
	if !up.hit {
		t.Error("valid run-bound identity should be forwarded")
	}
}

func TestSidecar_Identity_InvalidDenied(t *testing.T) {
	up := newFakeUpstream(t)
	sc := New(Config{Gate: mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{}), Upstream: mustURL(t, up.server.URL), IdentitySecret: []byte("sek")})
	resp := postIdentity(t, sc, toolsCallBody, "garbage.sig")
	defer resp.Body.Close()
	if code, _ := decodeRPCError(t, resp); code != -32004 {
		t.Errorf("invalid identity error.code = %d, want -32004", code)
	}
	if up.hit {
		t.Error("invalid identity must NOT reach upstream")
	}
}

func TestSidecar_RequireIdentity_NoTokenDenied(t *testing.T) {
	up := newFakeUpstream(t)
	gate := mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{RequireIdentity: true})
	sc := New(Config{Gate: gate, Upstream: mustURL(t, up.server.URL), IdentitySecret: []byte("sek")})
	resp := postIdentity(t, sc, toolsCallBody, "") // no identity header
	defer resp.Body.Close()
	if code, _ := decodeRPCError(t, resp); code != -32004 {
		t.Errorf("missing identity error.code = %d, want -32004", code)
	}
	if up.hit {
		t.Error("missing required identity must NOT reach upstream")
	}
}

func TestSidecar_Recorder_CapturesAllowedExchange(t *testing.T) {
	secret := []byte("sek")
	up := newFakeUpstream(t) // responds application/json
	rec := &capturingRecorder{}
	gate := mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{
		Clock: func() time.Time { return time.Date(2026, 6, 25, 11, 0, 0, 0, time.UTC) },
	})
	sc := New(Config{Gate: gate, Upstream: mustURL(t, up.server.URL), IdentitySecret: secret, Recorder: rec})

	tok, _ := mcpenforce.MintToken(mcpenforce.Identity{
		RunID: "run-1", ProjectID: "proj.a", Scopes: []string{"repo:read"},
		ExpiresAt: time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC),
	}, secret)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_query","arguments":{"q":"x"}}}`
	resp := postIdentity(t, sc, body, tok)
	resp.Body.Close()

	all := rec.all()
	if len(all) != 1 {
		t.Fatalf("exchanges = %d, want 1", len(all))
	}
	ex := all[0]
	if ex.RunID != "run-1" || ex.Tool != "graph_query" || !ex.Allowed {
		t.Errorf("exchange = %+v", ex)
	}
	if len(ex.RequestArgs) == 0 || !bytes.Contains(ex.RequestArgs, []byte(`"q":"x"`)) {
		t.Errorf("request args not captured: %s", ex.RequestArgs)
	}
	if len(ex.ResponseBody) == 0 {
		t.Error("JSON response body not captured")
	}
}

func TestSidecar_Recorder_CapturesDenied(t *testing.T) {
	up := newFakeUpstream(t)
	rec := &capturingRecorder{}
	gate := mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{RequireIdentity: true})
	sc := New(Config{Gate: gate, Upstream: mustURL(t, up.server.URL), IdentitySecret: []byte("sek"), Recorder: rec})

	resp := postIdentity(t, sc, toolsCallBody, "") // no identity → denied
	resp.Body.Close()

	all := rec.all()
	if len(all) != 1 {
		t.Fatalf("exchanges = %d, want 1", len(all))
	}
	if all[0].Allowed {
		t.Error("denied exchange marked allowed")
	}
	if len(all[0].ResponseBody) != 0 {
		t.Error("denied exchange must not carry a response body")
	}
	if up.hit {
		t.Error("denied request must not reach upstream")
	}
}

func TestJSONLRecorder(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONLRecorder(&buf)
	r.Record(context.Background(), Exchange{RunID: "run-1", Tool: "graph_query", Allowed: true})
	r.Record(context.Background(), Exchange{RunID: "run-1", Tool: "graph_delete", Allowed: false})
	lines := strings.Count(buf.String(), "\n")
	if lines != 2 {
		t.Errorf("lines = %d, want 2", lines)
	}
	if !strings.Contains(buf.String(), `"run_id":"run-1"`) {
		t.Errorf("run_id missing from JSONL: %s", buf.String())
	}
}

func TestSidecar_SSEResponse_StreamsThrough(t *testing.T) {
	// Upstream emits a Server-Sent Events stream; the sidecar must stream it back.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			_, _ = io.WriteString(w, "event: message\ndata: {\"chunk\":1}\n\n")
			f.Flush()
		}
	}))
	defer upstream.Close()

	sc := newSidecar(t, mcpenforce.NewPolicyGate(mcpenforce.PolicyConfig{}), upstream.URL)
	resp := post(t, sc, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_query"}}`, "svc.worker")
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "event: message") {
		t.Errorf("first SSE line = %q err=%v, want event: message", line, err)
	}
}

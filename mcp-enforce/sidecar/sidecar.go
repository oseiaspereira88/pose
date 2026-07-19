// Package sidecar is an MCP enforcement reverse proxy. It applies the shared
// mcp-enforce policy gate and audit in front of an MCP server that cannot host
// enforcement in-process — e.g. graphforge/mcp-server, a foreign repo that must
// stay independent (ADR-001, ADR-021). It gates tools/call requests and forwards
// everything else transparently, preserving Streamable HTTP session headers and
// SSE response streams.
package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	mcpenforce "github.com/harne8/mcp-enforce"
)

// maxPeekBytes bounds how much of a request body is buffered to inspect the
// JSON-RPC envelope. tools/call requests from agents are small.
const maxPeekBytes = 1 << 20 // 1 MiB

// Config configures a Sidecar.
type Config struct {
	Gate     *mcpenforce.PolicyGate // required
	Auditor  mcpenforce.Auditor     // defaults to NopAuditor when nil
	Upstream *url.URL               // MCP server to forward allowed requests to
	// IdentitySecret verifies the X-MCP-Execution-Identity token (ADR-007). Empty
	// disables identity binding (the header is ignored).
	IdentitySecret []byte
	// Recorder receives every gated live MCP exchange (ADR-022/ADR-017). Defaults
	// to NopRecorder.
	Recorder ExchangeRecorder
}

// Sidecar is an http.Handler that enforces policy in front of an upstream MCP
// server.
type Sidecar struct {
	gate           *mcpenforce.PolicyGate
	auditor        mcpenforce.Auditor
	proxy          *httputil.ReverseProxy
	identitySecret []byte
	recorder       ExchangeRecorder
}

// exchangeCtxKey carries the pending Exchange from ServeHTTP to the proxy's
// ModifyResponse hook so the response can be attached before recording.
type exchangeCtxKey struct{}

// New builds a Sidecar forwarding allowed requests to cfg.Upstream.
func New(cfg Config) *Sidecar {
	auditor := cfg.Auditor
	if auditor == nil {
		auditor = mcpenforce.NopAuditor{}
	}
	recorder := cfg.Recorder
	if recorder == nil {
		recorder = NopRecorder{}
	}
	s := &Sidecar{
		gate:           cfg.Gate,
		auditor:        auditor,
		proxy:          httputil.NewSingleHostReverseProxy(cfg.Upstream),
		identitySecret: cfg.IdentitySecret,
		recorder:       recorder,
	}
	s.proxy.ModifyResponse = s.captureResponse
	return s
}

// captureResponse attaches the upstream response to the pending Exchange and
// records it. JSON responses are captured for replay; streamed (SSE) responses
// are recorded without a body to avoid blocking the stream.
func (s *Sidecar) captureResponse(resp *http.Response) error {
	ex, _ := resp.Request.Context().Value(exchangeCtxKey{}).(*Exchange)
	if ex == nil {
		return nil
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewReader(body))
			if json.Valid(body) {
				ex.ResponseBody = body
			}
		}
	} else {
		ex.Streamed = true
	}
	s.recorder.Record(resp.Request.Context(), *ex)
	return nil
}

// rpcPeek is the minimal JSON-RPC envelope the gate needs.
type rpcPeek struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"params"`
}

// ServeHTTP gates POST tools/call requests and proxies everything else.
func (s *Sidecar) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only POST carries JSON-RPC requests; GET (SSE stream) and others pass through.
	if r.Method != http.MethodPost || r.Body == nil {
		s.proxy.ServeHTTP(w, r)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPeekBytes))
	_ = r.Body.Close()
	if err != nil {
		writeRPCError(w, nil, -32700, "parse error", nil)
		return
	}

	decision, id, args, gated := s.decide(r, body)
	if gated {
		s.auditor.Record(r.Context(), *decision)
		ex := Exchange{
			At: time.Now().UTC(), RunID: decision.RunID, Principal: decision.Principal,
			ProjectID: decision.ProjectID, Tool: decision.ToolName, Allowed: decision.Allow,
			Violations: decision.Violations, RequestArgs: args,
		}
		if !decision.Allow {
			s.recorder.Record(r.Context(), ex)
			writeRPCError(w, id, -32004, "policy denied", decision.Metadata())
			return
		}
		// Allowed: carry the exchange to captureResponse, which attaches the
		// upstream response and records it.
		exCopy := ex
		r = r.WithContext(context.WithValue(r.Context(), exchangeCtxKey{}, &exCopy))
	}

	// Restore the consumed body so the proxy can forward it upstream.
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))
	s.proxy.ServeHTTP(w, r)
}

// decide returns (decision, requestID, requestArgs, gated). gated=false means the
// request is not a tools/call and passes through unevaluated.
func (s *Sidecar) decide(r *http.Request, body []byte) (*mcpenforce.PolicyDecision, json.RawMessage, json.RawMessage, bool) {
	var single rpcPeek
	if err := json.Unmarshal(body, &single); err == nil {
		if single.Method != "tools/call" {
			return nil, single.ID, nil, false
		}
		input := s.inputFrom(r, single)
		// Bind the Execution Identity (ADR-007) when a secret is configured: a
		// present-but-invalid token is denied; a valid one populates the scope
		// fields before evaluation. No secret = identity binding disabled.
		if len(s.identitySecret) > 0 {
			if id, err := mcpenforce.IdentityFromHeader(r.Header, s.identitySecret); err != nil {
				d := mcpenforce.DenyDecision(input, "invalid_identity")
				return &d, single.ID, single.Params.Arguments, true
			} else if id != nil {
				input = id.Apply(input)
			}
		}
		d, evErr := s.gate.Evaluate(r.Context(), input)
		if evErr != nil {
			d = mcpenforce.DenyDecision(input, "policy_error")
		}
		return &d, single.ID, single.Params.Arguments, true
	}
	// Not a single object: likely a JSON-RPC batch array. Batched tools/call is
	// unsupported by this sidecar — deny rather than forward unevaluated.
	var batch []rpcPeek
	if err := json.Unmarshal(body, &batch); err == nil {
		for _, el := range batch {
			if el.Method == "tools/call" {
				d := mcpenforce.DenyDecision(s.inputFrom(r, el), "batch_tools_call_unsupported")
				return &d, nil, el.Params.Arguments, true
			}
		}
	}
	return nil, nil, nil, false
}

func (s *Sidecar) inputFrom(r *http.Request, peek rpcPeek) mcpenforce.PolicyInput {
	projectID, projectIDs, invalidProjectScope := mcpenforce.ProjectScopeFromArguments(peek.Params.Arguments)
	if projectID == "" {
		projectID = mcpenforce.HeaderValue(r.Header, "X-MCP-Project", "X-Project-Id", "X-Project-ID")
	}
	return mcpenforce.PolicyInput{
		Principal:           mcpenforce.PrincipalFromHeader(r.Header),
		ProjectID:           projectID,
		ProjectIDs:          projectIDs,
		InvalidProjectScope: invalidProjectScope,
		Method:              "tools/call",
		ToolName:            peek.Params.Name,
	}
}

// writeRPCError emits a JSON-RPC 2.0 error inside a 200 envelope (the JSON-RPC
// convention), echoing the request id when known.
func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string, data any) {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	errObj := map[string]any{"code": code, "message": msg}
	if data != nil {
		errObj["data"] = data
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   errObj,
	})
}

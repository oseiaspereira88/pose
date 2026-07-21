// Package mcpserver implements a minimal MCP server over Streamable HTTP
// (POST /mcp, JSON-RPC 2.0) per ADR-012, exposing the read-only POSE tool
// surface defined by ADR-003. A stdio transport is also available for
// claude-native (subprocess) deployments.
package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	mcpenforce "github.com/harne8/mcp-enforce"
	"github.com/harne8/pose-mcp/internal/observability"
	"github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// Reporter is the interface for emitting Conductor run events (external-run-reporters).
// When nil, conductor_run_* tools return a configuration error instead of calling out.
type Reporter interface {
	// OpenRun opens an observed external run and returns the assigned run_id and task_id.
	OpenRun(ctx context.Context, title, specSlug, adapter, origin string) (runID, taskID string, err error)
	// PostEvent appends an event to an open external run.
	PostEvent(ctx context.Context, runID string, evtType string, payload map[string]any, costUSD float64) error
}

// ConductorClient implements Reporter via the Conductor run reporter HTTP API.
type ConductorClient struct {
	base      string
	projectID string
	token     string
	http      *http.Client
}

// NewConductorClient returns a Reporter that calls the Conductor API at baseURL
// for the given project and bearer token.
func NewConductorClient(baseURL, projectID, token string) *ConductorClient {
	return &ConductorClient{
		base:      strings.TrimRight(baseURL, "/"),
		projectID: projectID,
		token:     token,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ConductorClient) OpenRun(ctx context.Context, title, specSlug, adapter, origin string) (string, string, error) {
	body, _ := json.Marshal(map[string]string{
		"title":     title,
		"spec_slug": specSlug,
		"adapter":   adapter,
		"origin":    origin,
	})
	resp, err := c.conductorPost(ctx, "/api/v1/projects/"+c.projectID+"/runs", body)
	if err != nil {
		return "", "", err
	}
	return resp["run_id"], resp["task_id"], nil
}

func (c *ConductorClient) PostEvent(ctx context.Context, runID string, evtType string, payload map[string]any, costUSD float64) error {
	body, _ := json.Marshal(map[string]any{
		"type":     evtType,
		"payload":  payload,
		"cost_usd": costUSD,
	})
	_, err := c.conductorPost(ctx, "/api/v1/projects/"+c.projectID+"/runs/"+runID+"/events", body)
	return err
}

func (c *ConductorClient) conductorPost(ctx context.Context, path string, body []byte) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("conductor: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("conductor: %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("conductor: %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out map[string]string
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

const (
	protocolVersion = "2025-03-26"
	serverName      = "harne8-pose-mcp"
)

// serverVersion follows the authoritative binary version instead of an
// MCP-local one (spec pose-version-contract): serverInfo.version must match
// `pose version` and the release metadata for the same build.
var serverVersion = version.Version

// Server dispatches MCP requests against a POSE store resolved per project_id
// (pose-mcp-multi-project). With a single root it behaves as before.
type Server struct {
	roots          *pose.Roots
	policy         *PolicyGate
	auditor        mcpenforce.Auditor // records allow+deny decisions
	identitySecret []byte             // verifies X-MCP-Execution-Identity (ADR-007); empty = disabled
	reporter       Reporter           // nil = conductor_run_* tools return config error
	harness        HarnessExecutor    // nil = pose_validate_submit returns config error
	orch           *orchestrator      // safe validation orchestration state (spec pose-safe-validate-orchestration)
	obs            *observability.Provider
}

// WithIdentitySecret sets the HMAC secret used to verify the Execution Identity
// token (ADR-007). Empty disables identity binding (the header is ignored).
func (s *Server) WithIdentitySecret(secret []byte) *Server {
	s.identitySecret = secret
	return s
}

// WithReporter enables the Conductor run reporter tools (conductor_run_open,
// conductor_run_event, conductor_run_close). Returns the server for chaining.
func (s *Server) WithReporter(r Reporter) *Server {
	s.reporter = r
	return s
}

// WithHarnessExecutor enables pose_validate_submit by wiring a real Harness
// client (spec pose-safe-validate-orchestration). Without it, an approved
// request can be resolved and approved but never submitted — the same
// "optional tool, clear config error" pattern as WithReporter.
func (s *Server) WithHarnessExecutor(h HarnessExecutor) *Server {
	s.harness = h
	return s
}

// WithObservability wires the opt-in OpenTelemetry provider (spec
// pose-otel-observability). Every server has a working (no-op by default)
// obs field from construction, so this is optional — callers that never
// call it get zero-cost, zero-network tracing/metrics/logging.
func (s *Server) WithObservability(p *observability.Provider) *Server {
	if p != nil {
		s.obs = p
	}
	return s
}

func defaultObservability() *observability.Provider {
	p, _ := observability.Init(context.Background(), observability.Config{})
	return p
}

// observability returns s.obs, falling back to a shared no-op instance for
// a Server constructed as a bare struct literal (test fixtures) rather
// than via New/NewWithRoots — never nil, so call sites need no guard.
func (s *Server) observability() *observability.Provider {
	if s.obs != nil {
		return s.obs
	}
	return sharedNoopObservability
}

var sharedNoopObservability = defaultObservability()

// New builds a single-root server (legacy / dev): every request resolves to this
// store regardless of project_id only when project_id is empty.
func New(store pose.Store) *Server {
	return &Server{roots: pose.NewRoots(pose.RootsConfig{DefaultRoot: store.Root}), policy: NewPolicyGate(PolicyConfig{}), auditor: defaultAuditor, orch: newOrchestrator(), obs: defaultObservability()}
}

// NewWithRoots builds a project-aware server backed by a roots registry.
func NewWithRoots(roots *pose.Roots) *Server {
	return NewWithRootsAndPolicy(roots, NewPolicyGate(PolicyConfig{}))
}

// NewWithRootsAndPolicy builds a project-aware server with an explicit policy gate.
func NewWithRootsAndPolicy(roots *pose.Roots, policy *PolicyGate) *Server {
	if policy == nil {
		policy = NewPolicyGate(PolicyConfig{})
	}
	return &Server{roots: roots, policy: policy, auditor: defaultAuditor, orch: newOrchestrator(), obs: defaultObservability()}
}

// TokenAuth wraps next with Bearer token authentication. When token is empty
// the middleware is a no-op (dev mode). /healthz is always exempt.
func TokenAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminTokenAuth wraps next with Bearer token authentication for the admin
// surface (portal-workspace-scale). When token is empty the middleware is a
// no-op (dev mode), same shape as TokenAuth but a separate secret so the
// refresh signal isn't gated by whatever token MCP clients carry.
func AdminTokenAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleAdminRefresh forces an immediate roots rescan (portal-workspace-scale),
// so a project onboarded/reindexed by the Conductor is discoverable without
// waiting for the next on-miss rescan.
func (s *Server) handleAdminRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.roots.Refresh()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]int{"projects": len(s.roots.Projects())})
}

// Handler returns the HTTP surface: POST /mcp request/response, GET /mcp SSE
// endpoint, /healthz, and POST /admin/refresh (gated by its own adminToken,
// independent of the general MCP bearer token).
func (s *Server) Handler(token, adminToken string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/admin/refresh", AdminTokenAuth(adminToken, http.HandlerFunc(s.handleAdminRefresh)))
	return TokenAuth(token, mux)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleSSE(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
		return
	}
	// Notifications (no id) are acknowledged without a body.
	if len(req.ID) == 0 || string(req.ID) == "null" {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	switch req.Method {
	case "initialize":
		w.Header().Set("Mcp-Session-Id", fmt.Sprintf("session_%d", time.Now().UnixNano()))
		writeRPC(w, result(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
		}))
	case "ping":
		writeRPC(w, result(req.ID, map[string]any{}))
	case "tools/list":
		writeRPC(w, result(req.ID, map[string]any{"tools": toolDefinitions()}))
	case "tools/call":
		writeRPC(w, s.callTool(r.Context(), r, req))
	default:
		writeRPC(w, errorResp(req.ID, -32601, fmt.Sprintf("method %q not found", req.Method)))
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		http.Error(w, "GET /mcp requires Accept: text/event-stream", http.StatusNotAcceptable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeRPC(w, rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32603, Message: "streaming not supported"}})
		return
	}
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Mcp-Session-Id", sessionID)
	w.WriteHeader(http.StatusOK)
	writeSSE(w, "endpoint", map[string]any{"uri": "/mcp", "session_id": sessionID})
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			writeSSE(w, "ping", map[string]any{"session_id": sessionID, "ts": time.Now().UTC().Format(time.RFC3339Nano)})
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// callTool is the HTTP-aware entry point for tools/call: reads principal and
// project_id from headers before delegating to callToolCtx.
func (s *Server) callTool(ctx context.Context, r *http.Request, req rpcRequest) rpcResponse {
	return s.callToolCtx(ctx,
		headerValue(r, "X-MCP-Principal", "X-Principal"),
		headerValue(r, "X-MCP-Project", "X-Project-Id", "X-Project-ID"),
		headerValue(r, mcpenforce.IdentityHeader),
		req,
	)
}

// callToolCtx is the transport-agnostic entry point for tools/call.
func (s *Server) callToolCtx(ctx context.Context, principal, projectIDFromHeader, identityToken string, req rpcRequest) rpcResponse {
	var p toolCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil || p.Name == "" {
		return errorResp(req.ID, -32602, "invalid tools/call params")
	}

	// Opt-in OpenTelemetry signals (spec pose-otel-observability, R1/R2):
	// one span + one duration/outcome measurement per tool call, tagged
	// only with the tool name and its catalog risk class — both fixed,
	// low-cardinality values, never an argument, path or repo name.
	obs := s.observability()
	riskClass := string(catalogGovernance[p.Name].Risk)
	callAttrs := metric.WithAttributes(attribute.String("tool", p.Name))
	ctx, span := obs.Tracer.Start(ctx, p.Name, trace.WithAttributes(
		attribute.String("pose.mcp.tool", p.Name),
		attribute.String("pose.mcp.risk_class", riskClass),
	))
	obs.Instr.InFlight.Add(ctx, 1, callAttrs)
	start := time.Now()
	outcome, logMsg := "ok", "ok"
	defer func() {
		obs.Instr.InFlight.Add(ctx, -1, callAttrs)
		obs.Instr.CallDuration.Record(ctx, float64(time.Since(start).Microseconds())/1000,
			metric.WithAttributes(attribute.String("tool", p.Name), attribute.String("risk_class", riskClass), attribute.String("outcome", outcome)))
		if outcome != "ok" {
			span.SetStatus(codes.Error, outcome)
		}
		obs.Log.Emit(ctx, p.Name, riskClass, outcome, logMsg)
		span.End()
	}()

	projectID := projectIDFromArguments(p.Arguments)
	if projectID == "" {
		projectID = projectIDFromHeader
	}
	policyInput := PolicyInput{
		Principal: principal,
		ProjectID: projectID,
		Method:    req.Method,
		ToolName:  p.Name,
	}
	// Bind the Execution Identity (ADR-007) when a secret is configured: a
	// present-but-invalid token is denied; a valid one populates the scope fields.
	if len(s.identitySecret) > 0 && identityToken != "" {
		id, ierr := mcpenforce.ParseToken(identityToken, s.identitySecret)
		if ierr != nil {
			outcome, logMsg = "policy_denied", "invalid_identity"
			obs.Instr.PolicyDenials.Add(ctx, 1, callAttrs)
			decision := mcpenforce.DenyDecision(policyInput, "invalid_identity")
			s.auditor.Record(ctx, decision)
			return errorRespData(req.ID, -32004, "policy denied", decision.Metadata())
		}
		policyInput = id.Apply(policyInput)
	}
	decision, err := s.policy.Evaluate(ctx, policyInput)
	if err != nil {
		decision = mcpenforce.DenyDecision(policyInput, "policy_error")
	}
	// Auditable trail of every decision — allow and deny — through the shared
	// auditor (pose-mcp-enterprise-hardening): a complete record of who invoked
	// what, with no payload content.
	s.auditor.Record(ctx, decision)
	if !decision.Allow {
		outcome, logMsg = "policy_denied", "policy denied"
		obs.Instr.PolicyDenials.Add(ctx, 1, callAttrs)
		return errorRespData(req.ID, -32004, "policy denied", decision.Metadata())
	}
	// Carry the verified Execution Identity (if any) to dispatch so tools
	// that require explicit authorization beyond the server's default
	// policy mode — pose_validate_approve/submit (spec
	// pose-safe-validate-orchestration R2) — can enforce it per-call. By
	// this point any presented identity is guaranteed valid and unexpired
	// (PolicyGate.Evaluate denies otherwise); RunID is empty when no
	// identity was presented or identity binding is unconfigured.
	ctx = withCallerIdentity(ctx, policyInput.RunID, policyInput.Scopes)
	out, err := s.dispatch(ctx, p.Name, p.Arguments)
	if err != nil {
		outcome = "error"
		logMsg = observability.Message(err.Error())
		span.RecordError(err)
		var unknown unknownToolError
		if errors.As(err, &unknown) {
			return errorResp(req.ID, -32602, err.Error())
		}
		// Project selection failures (spec pose-mcp-project-scope-contract
		// R2/R3): distinct, machine-readable error codes carrying only the
		// caller-supplied logical project_id — never the resolved filesystem
		// root — so a client can tell "retry with a known id" apart from
		// "pass project_id, the deployment has more than one project."
		var unknownProj pose.ProjectUnknownError
		var ambiguousProj pose.ProjectAmbiguousError
		switch {
		case errors.As(err, &unknownProj):
			return result(req.ID, map[string]any{
				"content":           []map[string]any{{"type": "text", "text": err.Error()}},
				"structuredContent": map[string]any{"error_code": "project_unknown", "project_id": unknownProj.ProjectID},
				"isError":           true,
			})
		case errors.As(err, &ambiguousProj):
			return result(req.ID, map[string]any{
				"content":           []map[string]any{{"type": "text", "text": err.Error()}},
				"structuredContent": map[string]any{"error_code": "project_ambiguous", "reason": ambiguousProj.Reason},
				"isError":           true,
			})
		}
		// Tool execution failures are results with isError=true (MCP spec),
		// so the model can read the cause and adapt.
		return result(req.ID, map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		})
	}
	pretty, merr := json.MarshalIndent(out, "", "  ")
	if merr != nil {
		outcome, logMsg = "error", "internal error encoding tool result"
		return errorResp(req.ID, -32603, "internal error encoding tool result")
	}
	return result(req.ID, map[string]any{
		"content":           []map[string]any{{"type": "text", "text": string(pretty)}},
		"structuredContent": out,
		"isError":           false,
	})
}

// dispatchRPC handles one JSON-RPC request without any HTTP context.
// Used by ServeStdio for the stdio transport.
func (s *Server) dispatchRPC(ctx context.Context, req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return result(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
		})
	case "ping":
		return result(req.ID, map[string]any{})
	case "tools/list":
		return result(req.ID, map[string]any{"tools": toolDefinitions()})
	case "tools/call":
		return s.callToolCtx(ctx, "", "", "", req)
	default:
		return errorResp(req.ID, -32601, fmt.Sprintf("method %q not found", req.Method))
	}
}

// ServeStdio runs the MCP server over stdin/stdout (one JSON-RPC message per
// line). All diagnostic output goes to stderr. This is the transport for
// claude-native (subprocess) deployments — no HTTP daemon needed.
func (s *Server) ServeStdio(ctx context.Context) error {
	enc := json.NewEncoder(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		// Notifications (no id) require no response per JSON-RPC spec.
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue
		}
		_ = enc.Encode(s.dispatchRPC(ctx, req))
	}
	return scanner.Err()
}

type unknownToolError struct{ name string }

func (e unknownToolError) Error() string { return fmt.Sprintf("unknown tool %q", e.name) }

func (s *Server) dispatch(ctx context.Context, name string, args json.RawMessage) (any, error) {
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	// Conductor reporter tools don't need a POSE store — handle them first.
	switch name {
	case "conductor_run_open", "conductor_run_event", "conductor_run_close":
		return s.dispatchReporter(ctx, name, args)
	}
	// Safe validation orchestration (spec pose-safe-validate-orchestration):
	// approve/submit/status/cancel act on an already-resolved request_id and
	// need no POSE store of their own — only pose_validate_request does.
	switch name {
	case "pose_validate_approve", "pose_validate_submit", "pose_validate_status", "pose_validate_cancel":
		return s.dispatchValidateOrchestration(ctx, name, args)
	}
	// All other tools resolve their store from the optional project_id (multi-project).
	var sel struct {
		ProjectID string `json:"project_id"`
	}
	_ = json.Unmarshal(args, &sel)
	store, err := s.roots.StoreFor(sel.ProjectID)
	if err != nil {
		return nil, err
	}
	switch name {
	case "pose_get_spec":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_get_spec: required argument %q missing", "slug")
		}
		return store.GetSpec(a.Slug)
	case "pose_requirement_trace":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_requirement_trace: required argument %q missing", "slug")
		}
		spec, err := store.GetSpec(a.Slug)
		if err != nil {
			return nil, err
		}
		trace := pose.ParseRequirementTrace(spec.Body)
		return map[string]any{"slug": spec.Slug, "status": spec.Status, "trace": trace}, nil
	case "pose_capability_state":
		assessment, err := store.LoadCapabilityAssessment()
		if err != nil {
			return nil, fmt.Errorf("pose_capability_state: %v", err)
		}
		issues := store.ValidateCapabilityEvidence(assessment)
		if issues == nil {
			issues = []string{}
		}
		ageDays := -1
		if assessed, err := time.Parse("2006-01-02", assessment.AssessedAt); err == nil {
			ageDays = int(time.Since(assessed).Hours() / 24)
		}
		return map[string]any{
			"schema_version":  assessment.SchemaVersion,
			"assessed_at":     assessment.AssessedAt,
			"baseline_commit": assessment.BaselineCommit,
			"method":          assessment.Method,
			"mechanisms":      assessment.Mechanisms,
			"evidence_issues": issues,
			"age_days":        ageDays,
		}, nil
	case "pose_capability_history":
		var a struct {
			Cursor string `json:"cursor"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_capability_history: invalid arguments")
		}
		events, err := pose.LoadCapabilityHistory(store.CapabilityHistoryPath())
		if err != nil {
			return nil, fmt.Errorf("pose_capability_history: %v", err)
		}
		effective := pose.EffectiveSnapshots(events)
		if effective == nil {
			effective = []pose.CapabilitySnapshot{}
		}
		after, err := decodePageCursor(a.Cursor)
		if err != nil {
			return nil, fmt.Errorf("pose_capability_history: %w", err)
		}
		page, next := paginatePage(effective, after, a.Limit)
		return map[string]any{"snapshots": page, "count": len(page), "next_cursor": next}, nil
	case "pose_spec_amendments":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_spec_amendments: required argument %q missing", "slug")
		}
		spec, err := store.GetSpec(a.Slug)
		if err != nil {
			return nil, err
		}
		events, err := pose.LoadAmendments(pose.AmendmentsPath(spec.Path))
		if err != nil {
			return nil, fmt.Errorf("pose_spec_amendments: %v", err)
		}
		pending := pose.UnacknowledgedChanges(spec.Body, events)
		if events == nil {
			events = []pose.Amendment{}
		}
		if pending == nil {
			pending = []string{}
		}
		return map[string]any{"slug": spec.Slug, "status": spec.Status, "events": events, "unacknowledged": pending}, nil
	case "pose_list_specs":
		var a struct {
			Status string `json:"status"`
			Cursor string `json:"cursor"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_list_specs: invalid arguments")
		}
		specs, err := store.ListSpecs(a.Status)
		if err != nil {
			return nil, err
		}
		after, err := decodePageCursor(a.Cursor)
		if err != nil {
			return nil, fmt.Errorf("pose_list_specs: %w", err)
		}
		page, next := paginatePage(specs, after, a.Limit)
		return map[string]any{"specs": page, "count": len(page), "next_cursor": next}, nil
	case "pose_spec_readiness":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_spec_readiness: required argument %q missing", "slug")
		}
		return store.SpecReadiness(a.Slug)
	case "pose_get_changelog":
		var a struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_get_changelog: invalid arguments")
		}
		return store.GetChangelog(a.Version)
	case "pose_list_roadmaps":
		var a struct {
			Cursor string `json:"cursor"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_list_roadmaps: invalid arguments")
		}
		roadmaps, err := store.ListRoadmaps()
		if err != nil {
			return nil, err
		}
		after, err := decodePageCursor(a.Cursor)
		if err != nil {
			return nil, fmt.Errorf("pose_list_roadmaps: %w", err)
		}
		page, next := paginatePage(roadmaps, after, a.Limit)
		return map[string]any{"roadmaps": page, "count": len(page), "next_cursor": next}, nil
	case "pose_get_roadmap":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_get_roadmap: required argument %q missing", "slug")
		}
		return store.GetRoadmap(a.Slug)
	case "pose_suggest":
		var a struct {
			TaskType string `json:"task_type"`
			Domain   string `json:"domain"`
			Path     string `json:"path"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.TaskType == "" {
			return nil, fmt.Errorf("pose_suggest: required argument %q missing", "task_type")
		}
		return store.Suggest(ctx, a.TaskType, a.Domain, a.Path)
	case "pose_get_workflow":
		var a struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_get_workflow: invalid arguments")
		}
		if a.Name == "" {
			items, err := store.ListWorkflows()
			if err != nil {
				return nil, err
			}
			return map[string]any{"workflows": items, "count": len(items)}, nil
		}
		return store.GetWorkflow(a.Name)
	case "pose_get_rules":
		var a struct {
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_get_rules: invalid arguments")
		}
		if a.Domain == "" {
			items, err := store.ListRules()
			if err != nil {
				return nil, err
			}
			return map[string]any{"rules": items, "count": len(items)}, nil
		}
		return store.GetRule(a.Domain)
	case "pose_insights":
		var a struct {
			GroupBy   string `json:"group_by"`
			SinceDays int    `json:"since_days"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_insights: invalid arguments")
		}
		return store.Insights(a.GroupBy, a.SinceDays)
	case "pose_get_followups":
		var a struct {
			All bool `json:"all"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_get_followups: invalid arguments")
		}
		return store.Followups(ctx, a.All)
	case "pose_check":
		var a struct {
			Strict *bool `json:"strict"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_check: invalid arguments")
		}
		return store.Check(ctx, a.Strict == nil || *a.Strict)
	case "pose_extension_list":
		items, err := store.ListExtensions()
		if err != nil {
			return nil, err
		}
		return map[string]any{"extensions": items, "count": len(items)}, nil
	case "pose_skills_check":
		var a struct {
			Strict *bool `json:"strict"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_skills_check: invalid arguments")
		}
		return store.SkillsCheck(ctx, a.Strict == nil || *a.Strict)
	case "pose_lint_spec":
		var a struct {
			Slug   string `json:"slug"`
			Strict *bool  `json:"strict"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_lint_spec: invalid arguments")
		}
		return store.LintSpec(ctx, a.Slug, a.Strict == nil || *a.Strict)
	case "pose_list_knowledge":
		var a struct {
			Cursor string `json:"cursor"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_list_knowledge: invalid arguments")
		}
		items, err := store.ListKnowledge()
		if err != nil {
			return nil, err
		}
		after, err := decodePageCursor(a.Cursor)
		if err != nil {
			return nil, fmt.Errorf("pose_list_knowledge: %w", err)
		}
		page, next := paginatePage(items, after, a.Limit)
		return map[string]any{"entries": page, "count": len(page), "next_cursor": next}, nil
	case "pose_get_knowledge":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_get_knowledge: required argument %q missing", "slug")
		}
		return store.GetKnowledge(a.Slug)
	case "pose_list_reports":
		var a struct {
			Cursor string `json:"cursor"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_list_reports: invalid arguments")
		}
		reports, err := store.ListReports()
		if err != nil {
			return nil, err
		}
		after, err := decodePageCursor(a.Cursor)
		if err != nil {
			return nil, fmt.Errorf("pose_list_reports: %w", err)
		}
		page, next := paginatePage(reports, after, a.Limit)
		return map[string]any{"reports": page, "count": len(page), "next_cursor": next}, nil
	case "pose_get_report":
		var a struct {
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Filename == "" {
			return nil, fmt.Errorf("pose_get_report: required argument %q missing", "filename")
		}
		return store.GetReport(a.Filename)
	case "pose_get_skill":
		var a struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_get_skill: invalid arguments")
		}
		if a.Name == "" {
			items, err := store.ListSkills()
			if err != nil {
				return nil, err
			}
			return map[string]any{"skills": items, "count": len(items)}, nil
		}
		return store.GetSkill(a.Name)
	case "pose_validate_request":
		var a struct {
			ProjectID    string `json:"project_id"`
			StackFilter  string `json:"stack_filter"`
			ModuleFilter string `json:"module_filter"`
			ChangedFrom  string `json:"changed_from"`
			ChangedTo    string `json:"changed_to"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_validate_request: invalid arguments")
		}
		req, err := s.orch.request(store.Root, sel.ProjectID, a.StackFilter, a.ModuleFilter, a.ChangedFrom, a.ChangedTo)
		if err != nil {
			return nil, fmt.Errorf("pose_validate_request: %w", err)
		}
		return req, nil
	default:
		return nil, unknownToolError{name}
	}
}

// dispatchValidateOrchestration handles the request-id-scoped orchestration
// tools (spec pose-safe-validate-orchestration): approve, submit, status,
// cancel. None resolves a POSE store — they act purely on the in-process
// request registry created by pose_validate_request.
func (s *Server) dispatchValidateOrchestration(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "pose_validate_approve":
		var a struct {
			RequestID  string `json:"request_id"`
			PlanDigest string `json:"plan_digest"`
			Decision   string `json:"decision"`
			Rationale  string `json:"rationale"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.RequestID == "" || a.PlanDigest == "" || a.Decision == "" {
			return nil, fmt.Errorf("pose_validate_approve: required arguments %q, %q and %q missing", "request_id", "plan_digest", "decision")
		}
		// R2: explicit authorization — a bound, verified Execution Identity
		// is mandatory for this tool specifically, independent of the
		// server's default policy mode (dev/allow-all still gates approval).
		caller := callerIdentityFromContext(ctx)
		if caller.RunID == "" {
			return nil, fmt.Errorf("pose_validate_approve: requires a bound Execution Identity (X-MCP-Execution-Identity) — approval cannot be anonymous")
		}
		return s.orch.approve(a.RequestID, a.PlanDigest, a.Decision, a.Rationale, caller.RunID, caller.Scopes)
	case "pose_validate_submit":
		var a struct {
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.RequestID == "" {
			return nil, fmt.Errorf("pose_validate_submit: required argument %q missing", "request_id")
		}
		if s.harness == nil {
			return nil, fmt.Errorf("harness executor not configured — set up a Harness client via WithHarnessExecutor")
		}
		return s.orch.submit(ctx, a.RequestID, s.harness)
	case "pose_validate_status":
		var a struct {
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.RequestID == "" {
			return nil, fmt.Errorf("pose_validate_status: required argument %q missing", "request_id")
		}
		return s.orch.status(a.RequestID)
	case "pose_validate_cancel":
		var a struct {
			RequestID string `json:"request_id"`
			Reason    string `json:"reason"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.RequestID == "" {
			return nil, fmt.Errorf("pose_validate_cancel: required argument %q missing", "request_id")
		}
		return s.orch.cancel(a.RequestID, a.Reason)
	default:
		return nil, unknownToolError{name}
	}
}

// dispatchReporter handles the conductor_run_* tools (external-run-reporters).
func (s *Server) dispatchReporter(ctx context.Context, name string, args json.RawMessage) (any, error) {
	if s.reporter == nil {
		return nil, fmt.Errorf("conductor reporter not configured — set CONDUCTOR_URL, CONDUCTOR_RUN_TOKEN, CONDUCTOR_PROJECT_ID")
	}
	switch name {
	case "conductor_run_open":
		var a struct {
			Title    string `json:"title"`
			SpecSlug string `json:"spec_slug"`
			Adapter  string `json:"adapter"`
			Origin   string `json:"origin"`
		}
		_ = json.Unmarshal(args, &a)
		adapter := a.Adapter
		if adapter == "" {
			adapter = "mcp"
		}
		runID, taskID, err := s.reporter.OpenRun(ctx, a.Title, a.SpecSlug, adapter, a.Origin)
		if err != nil {
			return nil, fmt.Errorf("conductor_run_open: %w", err)
		}
		return map[string]string{"run_id": runID, "task_id": taskID, "status": "open"}, nil

	case "conductor_run_event":
		var a struct {
			RunID   string         `json:"run_id"`
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
			CostUSD float64        `json:"cost_usd"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.RunID == "" {
			return nil, fmt.Errorf("conductor_run_event: required argument \"run_id\" missing")
		}
		evtType := a.Type
		if evtType == "" {
			evtType = "run.event"
		}
		if err := s.reporter.PostEvent(ctx, a.RunID, evtType, a.Payload, a.CostUSD); err != nil {
			return nil, fmt.Errorf("conductor_run_event: %w", err)
		}
		return map[string]string{"run_id": a.RunID, "status": "recorded"}, nil

	case "conductor_run_close":
		var a struct {
			RunID   string  `json:"run_id"`
			Outcome string  `json:"outcome"`
			CostUSD float64 `json:"cost_usd"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.RunID == "" {
			return nil, fmt.Errorf("conductor_run_close: required argument \"run_id\" missing")
		}
		evtType := terminalEventType(a.Outcome)
		if err := s.reporter.PostEvent(ctx, a.RunID, evtType, nil, a.CostUSD); err != nil {
			return nil, fmt.Errorf("conductor_run_close: %w", err)
		}
		return map[string]string{"run_id": a.RunID, "status": "closed", "event": evtType}, nil

	default:
		return nil, unknownToolError{name}
	}
}

// terminalEventType maps an outcome string to the canonical Conductor terminal event.
func terminalEventType(outcome string) string {
	switch strings.ToLower(outcome) {
	case "succeeded", "success", "completed", "finished":
		return "run.succeeded"
	case "failed", "failure", "error":
		return "run.failed"
	case "canceled", "cancelled":
		return "run.canceled"
	default:
		if outcome == "" {
			return "run.succeeded"
		}
		return "run." + outcome
	}
}

func toolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name": "pose_get_spec",
			"description": "Read one POSE spec by slug: lifecycle frontmatter (status, " +
				"created_at, completed_at, supersedes, depends_on, priority) plus the " +
				"full markdown body.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Spec slug, e.g. \"semql-entity-aliases\"",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"slug"},
			},
		},
		{
			"name": "pose_requirement_trace",
			"description": "Bidirectional requirement-to-evidence trace of one POSE spec: every " +
				"declared R-ID with its trace disposition (satisfied, waived, withdrawn), " +
				"evidence text and structured refs (check:, test:, report:, commit:), plus the " +
				"reverse evidence→requirements index, missing and orphaned entries.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Spec slug whose requirement trace to project",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"slug"},
			},
		},
		{
			"name": "pose_capability_state",
			"description": "Current capability assessment of the project: mechanisms with scores, " +
				"targets, typed evidence references and named gaps, plus evidence-resolution issues " +
				"and the assessment's age in days. Scores are human judgment; this projection never computes one.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_capability_history",
			"description": "Append-only capability-assessment snapshots (score vectors by timestamp, " +
				"supersede-aware), paginated — the mechanical basis for score diffs between dates or releases.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cursor": map[string]any{
						"type":        "string",
						"description": "Opaque pagination cursor from a previous call's next_cursor",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum snapshots to return; 0 or omitted returns all remaining",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_spec_amendments",
			"description": "Append-only amendment history of one POSE spec: material requirement " +
				"changes with affected R-IDs, rationale, author/reviewer aliases and timestamps, " +
				"plus any current requirement state not yet acknowledged by an amendment event.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Spec slug whose amendment history to read",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"slug"},
			},
		},
		{
			"name": "pose_list_specs",
			"description": "List every POSE spec of the project with its lifecycle frontmatter " +
				"(no body). Optionally filter by status: draft, in-progress, done, blocked, " +
				"superseded or abandoned.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type":        "string",
						"description": "Optional lifecycle filter",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
					"cursor": map[string]any{
						"type":        "string",
						"description": sharedCursorDescription,
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": sharedLimitDescription,
					},
				},
			},
		},
		{
			"name": "pose_spec_readiness",
			"description": "Answer whether a POSE spec is eligible for work/execution: not in a " +
				"terminal status and every depends_on ref satisfied (spec refs need status done; " +
				"milestone:/roadmap: refs are fail-closed until roadmaps are projected). Returns " +
				"{ready, status, waiting_on: [{ref, reason}], reason}. Consult before acting on a spec.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Spec slug to evaluate",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"slug"},
			},
		},
		{
			"name": "pose_get_changelog",
			"description": "Read the changelog state (pose-release-changelog): pending unreleased " +
				"fragments (one per delivered spec) and consolidated release versions; pass a " +
				"version to load its full body.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"version": map[string]any{
						"type":        "string",
						"description": "Optional release version (e.g. \"v0.2.0\") to load its consolidated changelog",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_list_roadmaps",
			"description": "List every governed roadmap (pose-roadmap-artifact): status, depends_on " +
				"and milestones (id, after, target dates, specs). The order of N specs lives here.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
					"cursor": map[string]any{
						"type":        "string",
						"description": sharedCursorDescription,
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": sharedLimitDescription,
					},
				},
			},
		},
		{
			"name": "pose_get_roadmap",
			"description": "Read one governed roadmap by slug: frontmatter, milestones (with planned " +
				"dates and member specs) and the markdown body.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Roadmap slug, e.g. \"sdd-serie\"",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"slug"},
			},
		},
		{
			"name": "pose_suggest",
			"description": "Canonical POSE trail for a task type (workflow + skill + cumulative " +
				"rules + validation command), straight from the deterministic CLI " +
				"(pose suggest --json). Use before starting any task to know which workflow " +
				"and rules apply. Optional domain or repo-relative path refine the rule set.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_type": map[string]any{
						"type": "string",
						"description": "Task type, e.g. feature, bugfix, refactor, documentation, " +
							"adr, knowledge, recurrence-escalation",
					},
					"domain": map[string]any{
						"type":        "string",
						"description": "Optional domain for extra rules (e.g. frontend, backend-go, k8s)",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Optional repo-relative path to infer the domain from",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"task_type"},
			},
		},
		{
			"name": "pose_get_workflow",
			"description": "Read one POSE workflow (.pose/workflows/<name>.md) with its full " +
				"checklist and execution modes. Without a name, lists every available workflow.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Workflow name, e.g. feature, bugfix, review; omit to list all",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_get_rules",
			"description": "Read one POSE domain rule (.pose/rules/<domain>.md). Without a " +
				"domain, lists every available rule. Rules apply cumulatively; in conflict the " +
				"most restrictive (usually security) prevails.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"domain": map[string]any{
						"type":        "string",
						"description": "Rule domain, e.g. security, backend-go, frontend-react; omit to list all",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_insights",
			"description": "Aggregate local POSE report history into deterministic outcome insights. " +
				"Returns the same structured contract as pose stats --json, without network access or writes.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"group_by": map[string]any{
						"type":        "string",
						"enum":        []string{"workflow", "task", "context"},
						"default":     "workflow",
						"description": "Dimension used to group outcomes",
					},
					"since_days": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"default":     0,
						"description": "Optional rolling window in days; zero includes all history",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_get_followups",
			"description": "Live backlog of spec follow-ups (pose followups --json): open " +
				"items with their source spec, disposition and lexical near-duplicate " +
				"candidates. Input for planning the next specs.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"all": map[string]any{
						"type":        "boolean",
						"description": "true = every follow-up (any disposition); default false = open backlog only",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_check",
			"description": "Evaluate the POSE structural integrity gate (pose check) in " +
				"read-only mode. Returns the verdict (passed/exit_code) plus the full output " +
				"as evidence — a failing gate is a result, not an error.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"strict": map[string]any{
						"type":        "boolean",
						"description": "Strict mode (default true); tolerant turns failures into warnings",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_extension_list",
			"description": "List installed POSE extensions (skills, workflows, rules or import " +
				"adapters installed via `pose extension install`): id, version, kind, installed_at, " +
				"digest, managed files and whether signature verification passed at install time. " +
				"Read-only — installing or removing an extension is a local CLI operation " +
				"(`pose extension install/remove`), never exposed as an MCP write.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_skills_check",
			"description": "Evaluate the Agent Skills conformance gate (pose skills-check) in " +
				"read-only mode: required metadata (name/description/when_to_use plus POSE's " +
				"pose_schema_range/clients/capabilities), linked-resource resolution, an offline " +
				"unsafe-instruction/secret-shaped-content scan, and claude-code client cross-check. " +
				"Returns the verdict (passed/exit_code) plus the full output as evidence.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"strict": map[string]any{
						"type":        "boolean",
						"description": "Strict mode (default true); tolerant turns failures into warnings",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_lint_spec",
			"description": "Evaluate the spec content + lifecycle gate (pose lint-spec) in " +
				"read-only mode: skeletal sections, done-without-completed_at, follow-ups " +
				"without disposition. Without a slug, evaluates every spec.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Spec slug; omit to lint all specs",
					},
					"strict": map[string]any{
						"type":        "boolean",
						"description": "Strict mode (default true)",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		{
			"name": "pose_list_knowledge",
			"description": "List POSE knowledge entries (handoffs, decision-logs, notes) " +
				"from .pose/knowledge/. Excludes sensitivity:restricted entries. " +
				"Returns metadata only; use pose_get_knowledge for the full body.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
					"cursor": map[string]any{
						"type":        "string",
						"description": sharedCursorDescription,
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": sharedLimitDescription,
					},
				},
			},
		},
		{
			"name": "pose_get_knowledge",
			"description": "Get one POSE knowledge entry by slug (full body). " +
				"Returns error if the entry has sensitivity:restricted.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug": map[string]any{
						"type":        "string",
						"description": "Knowledge slug",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"slug"},
			},
		},
		{
			"name": "pose_list_reports",
			"description": "List all historical POSE compliance and validation reports from " +
				".pose/reports/history/*.jsonl. Returns structured execution metadata, " +
				"excluding full report bodies.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
					"cursor": map[string]any{
						"type":        "string",
						"description": sharedCursorDescription,
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": sharedLimitDescription,
					},
				},
			},
		},
		{
			"name": "pose_get_report",
			"description": "Get the full markdown content of a POSE validation report by its filename " +
				"from .pose/reports/.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filename": map[string]any{
						"type":        "string",
						"description": "The report filename, e.g. \"2026-06-11-standard-semql-entity-aliases.md\"",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
				"required": []string{"filename"},
			},
		},
		{
			"name": "pose_get_skill",
			"description": "Read one agent skill by name (.agents/skills/<name>/SKILL.md): " +
				"full SKILL.md body with trigger keywords, steps, and output requirements. " +
				"Without a name, lists every available skill. Use before executing a skill-driven " +
				"task (e.g. pose-investigate) to get the exact execution protocol.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Skill name, e.g. \"pose-investigate\"; omit to list all",
					},
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
				},
			},
		},
		// Safe validation orchestration (external-side-effect, spec
		// pose-safe-validate-orchestration): request an immutable check plan,
		// require explicit approval bound to that plan's digest, and hand the
		// approved plan to a pluggable Harness executor. pose-mcp never runs
		// the plan itself on tools/call — only Submit reaches outside this
		// process, and only after project scope, policy allow and a bound
		// Execution Identity all pass.
		{
			"name": "pose_validate_request",
			"description": "Resolve an immutable, digest-pinned validation check plan (mirrors " +
				"pose validate's stack/module/changed-scope filters) without executing anything. " +
				"Returns {request_id, plan, state: \"pending_approval\"}. The plan's digest binds " +
				"every subsequent step — approve with the exact digest returned here.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Optional project to scope the .pose root (multi-project); omit for the default root",
					},
					"stack_filter": map[string]any{
						"type":        "string",
						"description": "Optional stack filter, same values as pose validate --stack",
					},
					"module_filter": map[string]any{
						"type":        "string",
						"description": "Optional module filter, same as pose validate --module",
					},
					"changed_from": map[string]any{
						"type":        "string",
						"description": "Optional changed-scope base revision, same as pose validate --changed-from",
					},
					"changed_to": map[string]any{
						"type":        "string",
						"description": "Optional changed-scope head revision, same as pose validate --changed-to",
					},
				},
			},
		},
		{
			"name": "pose_validate_approve",
			"description": "Approve or reject a pending validation request. Requires a bound " +
				"Execution Identity (X-MCP-Execution-Identity) — approval can never be anonymous, " +
				"regardless of the server's default policy mode. plan_digest must equal the exact " +
				"digest pose_validate_request returned; a mismatch is rejected as plan substitution.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"request_id": map[string]any{
						"type":        "string",
						"description": "Request id returned by pose_validate_request",
					},
					"plan_digest": map[string]any{
						"type":        "string",
						"description": "The exact plan.digest returned by pose_validate_request — binds the approval to that immutable plan",
					},
					"decision": map[string]any{
						"type":        "string",
						"enum":        []string{"approve", "reject"},
						"description": "Approval decision",
					},
					"rationale": map[string]any{
						"type":        "string",
						"description": "Why this decision was made; recorded on the request",
					},
				},
				"required": []string{"request_id", "plan_digest", "decision"},
			},
		},
		{
			"name": "pose_validate_submit",
			"description": "Hand an approved validation request to the configured Harness executor. " +
				"Only valid from state \"approved\"; idempotent — resubmitting an already-submitted " +
				"request returns the same execution_id without re-invoking the Harness. Requires a " +
				"Harness executor to be configured; otherwise returns a configuration error.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"request_id": map[string]any{
						"type":        "string",
						"description": "Request id to submit",
					},
				},
				"required": []string{"request_id"},
			},
		},
		{
			"name": "pose_validate_status",
			"description": "Read the current state of a validation request: pending_approval, " +
				"approved, rejected, submitted or cancelled, plus its plan, approver and execution_id " +
				"when present.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"request_id": map[string]any{
						"type":        "string",
						"description": "Request id to inspect",
					},
				},
				"required": []string{"request_id"},
			},
		},
		{
			"name": "pose_validate_cancel",
			"description": "Cancel a validation request that has not reached a terminal state. " +
				"Cancelling a submitted request marks it locally; propagating cancellation to a " +
				"running Harness execution is the executor's own responsibility.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"request_id": map[string]any{
						"type":        "string",
						"description": "Request id to cancel",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Why this request is being cancelled",
					},
				},
				"required": []string{"request_id"},
			},
		},
		// Conductor run reporter tools (external-run-reporters): open, append events to,
		// and close an observed external run on the Conductor board. Requires
		// CONDUCTOR_URL / CONDUCTOR_RUN_TOKEN / CONDUCTOR_PROJECT_ID to be configured.
		{
			"name": "conductor_run_open",
			"description": "Open an observed external run on the Conductor board. Call this at the " +
				"start of a session to create a task visible in the board. Returns run_id and " +
				"task_id to pass to subsequent conductor_run_event / conductor_run_close calls. " +
				"Requires CONDUCTOR_URL, CONDUCTOR_PROJECT_ID (and optionally CONDUCTOR_RUN_TOKEN) " +
				"to be set when pose-mcp was started.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Human-readable title for the run (e.g. 'Fix auth bug in session-store')",
					},
					"spec_slug": map[string]any{
						"type":        "string",
						"description": "POSE spec slug this run is associated with, e.g. 'portal-session-principal'",
					},
					"adapter": map[string]any{
						"type":        "string",
						"description": "Reporter adapter identifier, e.g. 'claude-code', 'codex'. Defaults to 'mcp'",
					},
					"origin": map[string]any{
						"type":        "string",
						"description": "Free-form origin label, e.g. 'ide', 'cli'",
					},
				},
			},
		},
		{
			"name": "conductor_run_event",
			"description": "Append an event to an open external run on the Conductor board. Use to " +
				"report intermediate progress (tool calls, cost increments, checkpoints). Terminal " +
				"event types (run.succeeded, run.failed, run.canceled) also close the run — prefer " +
				"conductor_run_close for that.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{
						"type":        "string",
						"description": "run_id returned by conductor_run_open",
					},
					"type": map[string]any{
						"type":        "string",
						"description": "Event type, e.g. 'run.event', 'run.tool_call', 'run.checkpoint'. Defaults to 'run.event'",
					},
					"payload": map[string]any{
						"type":        "object",
						"description": "Arbitrary JSON payload for the event (will be redacted server-side)",
					},
					"cost_usd": map[string]any{
						"type":        "number",
						"description": "Incremental cost in USD for this event (optional)",
					},
				},
				"required": []string{"run_id"},
			},
		},
		{
			"name": "conductor_run_close",
			"description": "Close an observed external run on the Conductor board by emitting a " +
				"terminal event. The run's task moves to CLOSED / FAILED / CANCELED. Call this at " +
				"the end of a session.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{
						"type":        "string",
						"description": "run_id returned by conductor_run_open",
					},
					"outcome": map[string]any{
						"type":        "string",
						"description": "Outcome of the run: 'succeeded' (default), 'failed', or 'canceled'",
					},
					"cost_usd": map[string]any{
						"type":        "number",
						"description": "Total cost in USD for the run (optional)",
					},
				},
				"required": []string{"run_id"},
			},
		},
	}
}

func result(id json.RawMessage, res any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: res}
}

func errorResp(id json.RawMessage, code int, msg string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
}

func errorRespData(id json.RawMessage, code int, msg string, data any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg, Data: data}}
}

func projectIDFromArguments(args json.RawMessage) string {
	var sel struct {
		ProjectID string `json:"project_id"`
	}
	_ = json.Unmarshal(args, &sel)
	return sel.ProjectID
}

func headerValue(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(r.Header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func writeRPC(w http.ResponseWriter, resp rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("pose-mcp: writing response: %v", err)
	}
}

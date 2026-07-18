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

	mcpenforce "github.com/crisol/mcp-enforce"
	"github.com/crisol/pose-mcp/internal/pose"
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
	serverName      = "crisol-pose-mcp"
	serverVersion   = "0.1.0"
)

// Server dispatches MCP requests against a POSE store resolved per project_id
// (pose-mcp-multi-project). With a single root it behaves as before.
type Server struct {
	roots          *pose.Roots
	policy         *PolicyGate
	auditor        mcpenforce.Auditor // records allow+deny decisions
	identitySecret []byte             // verifies X-MCP-Execution-Identity (ADR-007); empty = disabled
	reporter       Reporter           // nil = conductor_run_* tools return config error
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

// New builds a single-root server (legacy / dev): every request resolves to this
// store regardless of project_id only when project_id is empty.
func New(store pose.Store) *Server {
	return &Server{roots: pose.NewRoots(pose.RootsConfig{DefaultRoot: store.Root}), policy: NewPolicyGate(PolicyConfig{}), auditor: defaultAuditor}
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
	return &Server{roots: roots, policy: policy, auditor: defaultAuditor}
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
		return errorRespData(req.ID, -32004, "policy denied", decision.Metadata())
	}
	out, err := s.dispatch(ctx, p.Name, p.Arguments)
	if err != nil {
		var unknown unknownToolError
		if errors.As(err, &unknown) {
			return errorResp(req.ID, -32602, err.Error())
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
	case "pose_list_specs":
		var a struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("pose_list_specs: invalid arguments")
		}
		specs, err := store.ListSpecs(a.Status)
		if err != nil {
			return nil, err
		}
		return map[string]any{"specs": specs, "count": len(specs)}, nil
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
		roadmaps, err := store.ListRoadmaps()
		if err != nil {
			return nil, err
		}
		return map[string]any{"roadmaps": roadmaps, "count": len(roadmaps)}, nil
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
		items, err := store.ListKnowledge()
		if err != nil {
			return nil, err
		}
		return map[string]any{"entries": items, "count": len(items)}, nil
	case "pose_get_knowledge":
		var a struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(args, &a); err != nil || a.Slug == "" {
			return nil, fmt.Errorf("pose_get_knowledge: required argument %q missing", "slug")
		}
		return store.GetKnowledge(a.Slug)
	case "pose_list_reports":
		reports, err := store.ListReports()
		if err != nil {
			return nil, err
		}
		return map[string]any{"reports": reports, "count": len(reports)}, nil
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
				},
			},
		},
		{
			"name": "pose_list_knowledge",
			"description": "List POSE knowledge entries (handoffs, decision-logs, notes) " +
				"from .pose/knowledge/. Excludes sensitivity:restricted entries. " +
				"Returns metadata only; use pose_get_knowledge for the full body.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
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
				},
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

package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

// fakeReporter implements Reporter for testing the conductor_run_* tools.
type fakeReporter struct {
	runID  string
	taskID string
	events []string // event types posted
	err    error
}

func (f *fakeReporter) OpenRun(_ context.Context, _, _, _, _ string) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.runID, f.taskID, nil
}

func (f *fakeReporter) PostEvent(_ context.Context, _ string, evtType string, _ map[string]any, _ float64) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, evtType)
	return nil
}

func newTestServerWithReporter(t *testing.T, r Reporter) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	ts := httptest.NewServer(New(pose.Store{Root: root}).WithReporter(r).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

func newTestServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n\n# Spec: alpha\n\nBody.\n")
	write(".pose/specs/beta/spec.md", "---\nslug: beta\nstatus: draft\ncreated_at: 2026-06-03\n---\n\n# Spec: beta\n")
	write(".pose/workflows/feature.md", "# Workflow: Feature\n\nChecklist.\n")
	write(".pose/rules/security.md", "# Rule: Security\n")
	write(".pose/rules/backend-go.md", "# Rule: Backend Go\n")
	write(".pose/knowledge/handbook.md", "---\nslug: handbook\ntype: handoff\nowner: @platform\nsensitivity: public-internal\ncreated_at: 2026-06-01\n---\n\n# Handbook\n\nTeam processes.\n")
	write(".pose/reports/history/standard-feature.jsonl", `{"generated_at":"2026-06-11T12:00:00Z","task":"feature","task_slug":"alpha","workflow":"feature","context":"ci","report_path":"/abs/path/to/2026-06-11-standard-feature.md","outcome":"pass"}`)
	write(".pose/reports/2026-06-11-standard-feature.md", "# Report Feature\nPassed standard validation.")

	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler(token, ""))
	t.Cleanup(ts.Close)
	return ts
}

type rpcResult struct {
	Result map[string]any `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func post(t *testing.T, ts *httptest.Server, body string) (*http.Response, rpcResult) {
	t.Helper()
	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	var out rpcResult
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
	}
	return resp, out
}

func TestInitialize(t *testing.T) {
	ts := newTestServer(t, "")
	resp, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get("Mcp-Session-Id") == "" {
		t.Error("missing Mcp-Session-Id header")
	}
	if out.Result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v", out.Result["protocolVersion"])
	}
	info, _ := out.Result["serverInfo"].(map[string]any)
	if info["name"] != "harne8-pose-mcp" {
		t.Errorf("serverInfo.name = %v", info["name"])
	}
	// spec pose-version-contract R1: serverInfo.version follows the
	// authoritative binary version on every transport.
	if info["version"] != version.Version {
		t.Errorf("serverInfo.version = %v, want %v", info["version"], version.Version)
	}
}

func TestInitializeStdioVersionMatchesAuthority(t *testing.T) {
	s := &Server{}
	resp := s.dispatchRPC(context.Background(), rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize"})
	res, _ := resp.Result.(map[string]any)
	info, _ := res["serverInfo"].(map[string]any)
	if info["version"] != version.Version {
		t.Errorf("stdio serverInfo.version = %v, want %v", info["version"], version.Version)
	}
}

func TestNotificationAccepted(t *testing.T) {
	ts := newTestServer(t, "")
	resp, _ := post(t, ts, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 202", resp.StatusCode)
	}
}

func TestToolsList(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	tools, _ := out.Result["tools"].([]any)
	if len(tools) != 32 {
		t.Fatalf("tools = %d, want 32", len(tools))
	}
	names := map[string]bool{}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		names[tool["name"].(string)] = true
		if tool["inputSchema"] == nil {
			t.Errorf("tool %v missing inputSchema", tool["name"])
		}
	}
	for _, want := range []string{"pose_get_spec", "pose_list_specs", "pose_spec_readiness",
		"pose_list_roadmaps", "pose_get_roadmap", "pose_get_changelog",
		"pose_suggest", "pose_get_workflow", "pose_get_rules", "pose_insights", "pose_get_followups", "pose_check",
		"pose_lint_spec", "pose_list_knowledge", "pose_get_knowledge", "pose_list_reports",
		"pose_get_report"} {
		if !names[want] {
			t.Errorf("missing tool %s (got %v)", want, names)
		}
	}
}

func TestToolsCall_GetSpec(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected error: %+v", out.Error)
	}
	if out.Result["isError"] != false {
		t.Errorf("isError = %v", out.Result["isError"])
	}
	content := out.Result["content"].([]any)[0].(map[string]any)
	if !strings.Contains(content["text"].(string), `"status": "done"`) {
		t.Errorf("content text missing status: %v", content["text"])
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["slug"] != "alpha" {
		t.Errorf("structuredContent.slug = %v", structured["slug"])
	}
}

func TestToolsCall_ListSpecsFilter(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"status":"draft"}}}`)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", structured["count"])
	}
}

func TestToolsCall_Insights(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":31,"method":"tools/call","params":{"name":"pose_insights","arguments":{"group_by":"task","since_days":0}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("insights failed: error=%+v result=%v", out.Error, out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["group_by"] != "task" || structured["records_scanned"] != float64(1) {
		t.Fatalf("unexpected insights: %v", structured)
	}
	rows, _ := structured["rows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["key"] != "alpha" {
		t.Fatalf("unexpected rows: %v", rows)
	}
}

func TestToolsCall_InsightsRejectsInvalidInputs(t *testing.T) {
	ts := newTestServer(t, "")
	for _, arguments := range []string{`{"group_by":"owner"}`, `{"since_days":-1}`} {
		body := `{"jsonrpc":"2.0","id":32,"method":"tools/call","params":{"name":"pose_insights","arguments":` + arguments + `}}`
		_, out := post(t, ts, body)
		if out.Error != nil || out.Result["isError"] != true {
			t.Fatalf("invalid arguments %s did not fail closed: error=%+v result=%v", arguments, out.Error, out.Result)
		}
	}
}

// newMultiProjectServer builds two project roots with distinct specs, served by
// one project-aware server (pose-mcp-multi-project).
func newMultiProjectServer(t *testing.T) *httptest.Server {
	t.Helper()
	mk := func(slug string) string {
		root := t.TempDir()
		path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nslug: " + slug + "\nstatus: done\ncreated_at: 2026-06-01\ncompleted_at: 2026-06-02\n---\n\n# Spec: " + slug + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		history := filepath.Join(root, ".pose", "reports", "history", "runs.jsonl")
		if err := os.MkdirAll(filepath.Dir(history), 0o755); err != nil {
			t.Fatal(err)
		}
		record := `{"generated_at":"2026-07-18T00:00:00Z","workflow":"feature","task_slug":"` + slug + `","outcome":"pass"}`
		if err := os.WriteFile(history, []byte(record+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}
	rootA := mk("only-in-a")
	rootB := mk("only-in-b")
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      rootA,
		DefaultProjectID: "proj.a",
		Explicit:         map[string]string{"proj.b": rootB},
	})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)
	return ts
}

func TestToolsCall_RequirePrincipal_DeniesAnonymous(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "specs", "alpha", "spec.md"),
		[]byte("---\nslug: alpha\nstatus: draft\ncreated_at: 2026-06-01\n---\n# alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: root, DefaultProjectID: "proj.a"})
	srv := NewWithRootsAndPolicy(roots, NewPolicyGate(PolicyConfig{RequirePrincipal: true}))
	ts := httptest.NewServer(srv.Handler("", ""))
	t.Cleanup(ts.Close)

	// No X-MCP-Principal header → anonymous → policy denied.
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":30,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha"}}}`)
	if out.Error == nil || out.Error.Code != -32004 {
		t.Fatalf("anonymous tools/call = %+v, want -32004 policy denied", out.Error)
	}
}

func TestToolsCall_ListSpecs_ScopedByProject(t *testing.T) {
	ts := newMultiProjectServer(t)

	_, outA := post(t, ts, `{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"project_id":"proj.a"}}}`)
	sa, _ := outA.Result["structuredContent"].(map[string]any)
	specsA, _ := sa["specs"].([]any)
	if len(specsA) != 1 || specsA[0].(map[string]any)["slug"] != "only-in-a" {
		t.Fatalf("proj.a specs = %v, want [only-in-a]", sa["specs"])
	}

	_, outB := post(t, ts, `{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"project_id":"proj.b"}}}`)
	sb, _ := outB.Result["structuredContent"].(map[string]any)
	specsB, _ := sb["specs"].([]any)
	if len(specsB) != 1 || specsB[0].(map[string]any)["slug"] != "only-in-b" {
		t.Fatalf("proj.b specs = %v, want [only-in-b]", sb["specs"])
	}
}

func TestToolsCall_InsightsProject(t *testing.T) {
	ts := newMultiProjectServer(t)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":23,"method":"tools/call","params":{"name":"pose_insights","arguments":{"project_id":"proj.b","group_by":"task"}}}`)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	rows, _ := structured["rows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["key"] != "only-in-b" {
		t.Fatalf("proj.b insights = %v, want only-in-b", structured)
	}
}

func TestToolsCall_UnknownProjectIsToolError(t *testing.T) {
	ts := newMultiProjectServer(t)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":22,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"project_id":"proj.ghost"}}}`)
	if out.Result["isError"] != true {
		t.Fatalf("expected isError=true for unknown project, got %v", out.Result)
	}
}

func TestToolsCall_GetWorkflow(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"pose_get_workflow","arguments":{"name":"feature"}}}`)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["title"] != "Workflow: Feature" {
		t.Errorf("title = %v", structured["title"])
	}
	if !strings.Contains(structured["body"].(string), "Checklist.") {
		t.Error("workflow body missing")
	}
}

func TestToolsCall_ListRules(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"pose_get_rules","arguments":{}}}`)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["count"].(float64) != 2 {
		t.Errorf("rules count = %v, want 2", structured["count"])
	}
}

func TestToolsCall_CheckWithoutCLIIsToolError(t *testing.T) {
	// A Go test process is not a runnable native pose executable: failure must surface as a
	// tool-level error (isError=true), never as a crash or protocol error.
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"pose_check","arguments":{}}}`)
	if out.Error != nil {
		t.Fatalf("expected tool-level error, got protocol error: %+v", out.Error)
	}
	if out.Result["isError"] != true {
		t.Errorf("isError = %v, want true", out.Result["isError"])
	}
}

func TestToolsCall_UnknownTool(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)
	if out.Error == nil || out.Error.Code != -32602 {
		t.Errorf("error = %+v, want -32602", out.Error)
	}
}

func TestToolsCall_MissingSlugIsToolError(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{}}}`)
	if out.Error != nil {
		t.Fatalf("expected tool-level error, got protocol error: %+v", out.Error)
	}
	if out.Result["isError"] != true {
		t.Errorf("isError = %v, want true", out.Result["isError"])
	}
}

func TestUnknownMethod(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":7,"method":"resources/list","params":{}}`)
	if out.Error == nil || out.Error.Code != -32601 {
		t.Errorf("error = %+v, want -32601", out.Error)
	}
}

func TestGetMethodNotAllowed(t *testing.T) {
	ts := newTestServer(t, "")
	resp, err := http.Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Errorf("GET status = %d, want 406 without SSE Accept", resp.StatusCode)
	}
}

func TestGetSSE(t *testing.T) {
	ts := newTestServer(t, "")
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /mcp: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if resp.Header.Get("Mcp-Session-Id") == "" {
		t.Fatal("missing Mcp-Session-Id")
	}
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil {
		t.Fatalf("read SSE event line: %v", err)
	}
	if strings.TrimSpace(line) != "event: endpoint" {
		t.Fatalf("first SSE line = %q, want endpoint event", line)
	}
}

func TestToolsCall_ListKnowledge(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"pose_list_knowledge","arguments":{}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected error: %+v", out.Error)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", structured["count"])
	}
	entries, _ := structured["entries"].([]any)
	if len(entries) != 1 {
		t.Errorf("entries length = %d, want 1", len(entries))
	}
}

func TestToolsCall_GetKnowledge(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"pose_get_knowledge","arguments":{"slug":"handbook"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected error: %+v", out.Error)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["slug"] != "handbook" {
		t.Errorf("slug = %v", structured["slug"])
	}
	if !strings.Contains(structured["body"].(string), "Team processes") {
		t.Errorf("body missing expected content: %q", structured["body"])
	}
}

func TestToolsCall_ListReports(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"pose_list_reports","arguments":{}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected error: %+v", out.Error)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", structured["count"])
	}
	reports, _ := structured["reports"].([]any)
	if len(reports) != 1 {
		t.Errorf("reports length = %d, want 1", len(reports))
	}
	r := reports[0].(map[string]any)
	if r["task"] != "feature" || r["outcome"] != "pass" || r["filename"] != "2026-06-11-standard-feature.md" {
		t.Errorf("unexpected report content: %+v", r)
	}
}

func TestToolsCall_GetReport(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"pose_get_report","arguments":{"filename":"2026-06-11-standard-feature.md"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected error: %+v", out.Error)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["filename"] != "2026-06-11-standard-feature.md" {
		t.Errorf("filename = %v", structured["filename"])
	}
	if !strings.Contains(structured["body"].(string), "Passed standard validation.") {
		t.Errorf("body missing expected content: %q", structured["body"])
	}
}

func TestTokenAuthNoTokenAllowsAll(t *testing.T) {
	ts := newTestServer(t, "")
	resp, _ := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestTokenAuthRejectsNoHeader(t *testing.T) {
	ts := newTestServer(t, "secret")
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTokenAuthRejectsGetSSEWithoutHeader(t *testing.T) {
	ts := newTestServer(t, "secret")
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTokenAuthAcceptsGetSSEWithCorrectToken(t *testing.T) {
	ts := newTestServer(t, "secret")
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestTokenAuthRejectsWrongToken(t *testing.T) {
	ts := newTestServer(t, "secret")
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrongtoken")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestTokenAuthAcceptsCorrectToken(t *testing.T) {
	ts := newTestServer(t, "secret")
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestTokenAuthHealthzExempt(t *testing.T) {
	ts := newTestServer(t, "secret")
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestConductorRunOpen_ReporterNotConfigured(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"conductor_run_open","arguments":{"title":"test"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", out.Error)
	}
	// Tool execution failure is returned as isError=true in result content (MCP spec).
	content, _ := out.Result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("expected content in result")
	}
	text, _ := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "not configured") {
		t.Errorf("expected 'not configured' error, got: %s", text)
	}
	isErr, _ := out.Result["isError"].(bool)
	if !isErr {
		t.Error("expected isError=true when reporter not configured")
	}
}

func TestConductorRunOpen(t *testing.T) {
	r := &fakeReporter{runID: "run.ext.123", taskID: "task.ext.123"}
	ts := newTestServerWithReporter(t, r)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"conductor_run_open","arguments":{"title":"Fix auth","spec_slug":"portal-auth","adapter":"claude-code"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", out.Error)
	}
	isErr, _ := out.Result["isError"].(bool)
	if isErr {
		content, _ := out.Result["content"].([]any)
		text, _ := content[0].(map[string]any)["text"].(string)
		t.Fatalf("isError=true: %s", text)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["run_id"] != "run.ext.123" {
		t.Errorf("run_id = %v, want run.ext.123", structured["run_id"])
	}
	if structured["task_id"] != "task.ext.123" {
		t.Errorf("task_id = %v, want task.ext.123", structured["task_id"])
	}
	if structured["status"] != "open" {
		t.Errorf("status = %v, want open", structured["status"])
	}
}

func TestConductorRunEvent(t *testing.T) {
	r := &fakeReporter{runID: "run.ext.456", taskID: "task.ext.456"}
	ts := newTestServerWithReporter(t, r)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"conductor_run_event","arguments":{"run_id":"run.ext.456","type":"run.checkpoint","cost_usd":0.05}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", out.Error)
	}
	isErr, _ := out.Result["isError"].(bool)
	if isErr {
		t.Fatal("unexpected isError=true")
	}
	if len(r.events) != 1 || r.events[0] != "run.checkpoint" {
		t.Errorf("events = %v, want [run.checkpoint]", r.events)
	}
}

func TestConductorRunEvent_MissingRunID(t *testing.T) {
	r := &fakeReporter{runID: "run.ext.789", taskID: "task.ext.789"}
	ts := newTestServerWithReporter(t, r)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"conductor_run_event","arguments":{"type":"run.event"}}}`)
	isErr, _ := out.Result["isError"].(bool)
	if !isErr {
		t.Error("expected isError=true for missing run_id")
	}
}

func TestConductorRunClose(t *testing.T) {
	r := &fakeReporter{runID: "run.ext.close", taskID: "task.ext.close"}
	ts := newTestServerWithReporter(t, r)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"conductor_run_close","arguments":{"run_id":"run.ext.close","outcome":"succeeded","cost_usd":1.23}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", out.Error)
	}
	isErr, _ := out.Result["isError"].(bool)
	if isErr {
		t.Fatal("unexpected isError=true")
	}
	if len(r.events) != 1 || r.events[0] != "run.succeeded" {
		t.Errorf("events = %v, want [run.succeeded]", r.events)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["status"] != "closed" {
		t.Errorf("status = %v, want closed", structured["status"])
	}
	if structured["event"] != "run.succeeded" {
		t.Errorf("event = %v, want run.succeeded", structured["event"])
	}
}

func TestConductorRunClose_DefaultOutcomeSucceeded(t *testing.T) {
	r := &fakeReporter{runID: "run.ext.def", taskID: "task.ext.def"}
	ts := newTestServerWithReporter(t, r)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"conductor_run_close","arguments":{"run_id":"run.ext.def"}}}`)
	isErr, _ := out.Result["isError"].(bool)
	if isErr {
		t.Fatal("unexpected isError=true")
	}
	if len(r.events) != 1 || r.events[0] != "run.succeeded" {
		t.Errorf("events = %v, want [run.succeeded] for empty outcome", r.events)
	}
}

func TestConductorRunOpen_InToolList(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{}}`)
	tools, _ := out.Result["tools"].([]any)
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		if m, ok := tool.(map[string]any); ok {
			names = append(names, m["name"].(string))
		}
	}
	for _, want := range []string{"conductor_run_open", "conductor_run_event", "conductor_run_close"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tool %q not found in tools/list: %v", want, names)
		}
	}
}

func TestAdminRefresh_DiscoversNewProjectWithoutWaitingForThrottle(t *testing.T) {
	base := t.TempDir()
	roots := pose.NewRoots(pose.RootsConfig{ProjectsDir: base})
	if got := len(roots.Projects()); got != 0 {
		t.Fatalf("initial projects = %d, want 0", got)
	}
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	// Materialize a new project after the server started, like onboarding does.
	if err := os.MkdirAll(filepath.Join(base, "proj.new", ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["projects"] != 1 {
		t.Fatalf("projects in response = %d, want 1", body["projects"])
	}
	if got := roots.Projects(); len(got) != 1 || got[0] != "proj.new" {
		t.Fatalf("roots.Projects() = %v, want [proj.new]", got)
	}
}

func TestAdminRefresh_RequiresCorrectAdminToken(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{ProjectsDir: t.TempDir()})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", "admin-secret"))
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAdminRefresh_AcceptsCorrectAdminToken(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{ProjectsDir: t.TempDir()})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", "admin-secret"))
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer admin-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestAdminRefresh_MethodNotAllowed(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{ProjectsDir: t.TempDir()})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("", ""))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/admin/refresh")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

// TestAdminRefresh_IndependentFromGeneralToken: the general MCP bearer token
// (TokenAuth) wraps the whole mux, but with no admin token configured,
// /admin/refresh needs nothing beyond it — the two secrets are independent.
func TestAdminRefresh_IndependentFromGeneralToken(t *testing.T) {
	roots := pose.NewRoots(pose.RootsConfig{ProjectsDir: t.TempDir()})
	ts := httptest.NewServer(NewWithRoots(roots).Handler("mcp-secret", ""))
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/admin/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer mcp-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

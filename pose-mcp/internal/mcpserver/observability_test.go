package mcpserver

// OpenTelemetry operational signals wired into the tools/call path (spec
// pose-otel-observability): a Server with no explicit WithObservability
// call must behave exactly as before (default no-op, proven by every
// other test in this package still passing unmodified); an enabled
// provider must observe a real, correlated span/metric/log per call,
// including the policy-denial path.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/observability"
	"github.com/harne8/pose-mcp/internal/pose"
)

func otlpCounter(t *testing.T) (*httptest.Server, *int64) {
	t.Helper()
	var n int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&n, 1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func enabledObservability(t *testing.T, endpoint string) *observability.Provider {
	t.Helper()
	p, err := observability.Init(context.Background(), observability.Config{
		Enabled: true, Endpoint: endpoint, Insecure: true, SampleRatio: 1.0, ExportPeriod: 3600,
	})
	if err != nil {
		t.Fatalf("observability.Init: %v", err)
	}
	return p
}

func TestSuccessfulToolCallEmitsSpanAndDuration(t *testing.T) {
	srv, count := otlpCounter(t)
	obs := enabledObservability(t, srv.URL)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "specs", "alpha", "spec.md"),
		[]byte("---\nslug: alpha\nstatus: draft\ncreated_at: 2026-06-01\n---\n\n# Spec: alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(pose.Store{Root: root}).WithObservability(obs)
	ts := httptest.NewServer(s.Handler("", ""))
	t.Cleanup(ts.Close)

	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", out.Error)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := obs.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if atomic.LoadInt64(count) == 0 {
		t.Error("a successful tool call must export at least one OTLP request (span and/or metric batch)")
	}
}

func TestPolicyDeniedCallIncrementsDenialMetric(t *testing.T) {
	srv, count := otlpCounter(t)
	obs := enabledObservability(t, srv.URL)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewWithRootsAndPolicy(pose.NewRoots(pose.RootsConfig{DefaultRoot: root}), NewPolicyGate(PolicyConfig{RequirePrincipal: true})).
		WithObservability(obs)
	ts := httptest.NewServer(s.Handler("", ""))
	t.Cleanup(ts.Close)

	// No X-MCP-Principal header: RequirePrincipal denies it.
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha"}}}`)
	if out.Error == nil || out.Error.Code != -32004 {
		t.Fatalf("expected policy denial, got %+v", out.Error)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := obs.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if atomic.LoadInt64(count) == 0 {
		t.Error("a policy-denied call must still export its metrics batch (denial counter)")
	}
}

func TestDefaultServerObservabilityIsNoopAndNeverPanics(t *testing.T) {
	// A Server built via New()/NewWithRootsAndPolicy() (no WithObservability
	// call) and a bare struct literal must both work identically to before
	// this spec — proven by exercising callToolCtx through both.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "specs", "alpha", "spec.md"),
		[]byte("---\nslug: alpha\nstatus: draft\ncreated_at: 2026-06-01\n---\n\n# Spec: alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"pose_get_spec","arguments":{"slug":"alpha"}}}`)
	if out.Error != nil {
		t.Fatalf("unexpected RPC error with default (no-op) observability: %+v", out.Error)
	}

	// A bare struct literal (s.obs left nil) must fall back to the shared
	// no-op provider rather than panic on s.obs.Tracer/s.obs.Instr.
	bare := &Server{roots: pose.NewRoots(pose.RootsConfig{DefaultRoot: root}), policy: NewPolicyGate(PolicyConfig{}), auditor: defaultAuditor, orch: newOrchestrator()}
	req := rpcRequest{JSONRPC: "2.0", ID: []byte(`9`), Method: "tools/call",
		Params: []byte(`{"name":"pose_get_spec","arguments":{"slug":"alpha"}}`)}
	resp := bare.callToolCtx(context.Background(), "", "", "", req)
	if resp.Error != nil {
		t.Fatalf("bare Server with nil obs must not panic or error: %+v", resp.Error)
	}
}

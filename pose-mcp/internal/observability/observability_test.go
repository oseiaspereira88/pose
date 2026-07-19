package observability

// OpenTelemetry operational signals (spec pose-otel-observability): opt-in
// gating (R1/R2/R3 all depend on nothing leaving the process unless both
// env gates are set), redaction of paths/tokens (R3), correlated logging
// (R3), and real OTLP export against a local HTTP receiver, including
// graceful handling of an unreachable endpoint.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func clearOTelEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"POSE_OTEL_ENABLED", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_INSECURE",
		"OTEL_EXPORTER_OTLP_HEADERS", "OTEL_TRACES_SAMPLER_ARG", "OTEL_METRIC_EXPORT_INTERVAL",
	} {
		old, had := os.LookupEnv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, old)
			}
		})
	}
}

func TestFromEnvRequiresBothGates(t *testing.T) {
	cases := []struct {
		name, enabled, endpoint string
		want                    bool
	}{
		{"neither", "", "", false},
		{"only-enabled", "1", "", false},
		{"only-endpoint", "", "http://127.0.0.1:4318", false},
		{"both", "1", "http://127.0.0.1:4318", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clearOTelEnv(t)
			if c.enabled != "" {
				t.Setenv("POSE_OTEL_ENABLED", c.enabled)
			}
			if c.endpoint != "" {
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", c.endpoint)
			}
			cfg := FromEnv()
			if cfg.Enabled != c.want {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, c.want)
			}
		})
	}
}

func TestFromEnvParsesOptionalSettings(t *testing.T) {
	clearOTelEnv(t)
	t.Setenv("POSE_OTEL_ENABLED", "1")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "x-api-key=abc123,x-team=pose")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.25")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "5000")

	cfg := FromEnv()
	if !cfg.Enabled || cfg.Endpoint != "http://collector:4318" || !cfg.Insecure {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.Headers["x-api-key"] != "abc123" || cfg.Headers["x-team"] != "pose" {
		t.Errorf("headers not parsed: %+v", cfg.Headers)
	}
	if cfg.SampleRatio != 0.25 {
		t.Errorf("SampleRatio = %v, want 0.25", cfg.SampleRatio)
	}
	if cfg.ExportPeriod != 5 {
		t.Errorf("ExportPeriod = %v, want 5", cfg.ExportPeriod)
	}
}

func TestInitDisabledIsInertNoop(t *testing.T) {
	p, err := Init(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Init(disabled) error: %v", err)
	}
	if p.Tracer == nil || p.Meter == nil || p.Instr == nil || p.Log == nil {
		t.Fatal("disabled provider must still be fully usable (no-op)")
	}
	// Every operation must be safe and silent.
	_, span := p.Tracer.Start(context.Background(), "x")
	span.End()
	p.Instr.CallDuration.Record(context.Background(), 1.0)
	p.Instr.PolicyDenials.Add(context.Background(), 1)
	p.Instr.InFlight.Add(context.Background(), 1)
	p.Log.Emit(context.Background(), "tool", "read", "ok", "message")
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown(disabled) error: %v", err)
	}
}

func TestNilProviderShutdownIsSafe(t *testing.T) {
	var p *Provider
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("nil Provider.Shutdown must be a no-op, got: %v", err)
	}
}

func TestSecretsRedaction(t *testing.T) {
	fakeAWSKeyShapedFixture := "AKIA" + "ABCDEFGHIJKLMNOP"
	for _, s := range []string{
		"key: " + fakeAWSKeyShapedFixture,
		"-----BEGIN RSA PRIVATE KEY-----\nMIIB...\n-----END RSA PRIVATE KEY-----",
		"token ghp_" + strings.Repeat("a", 30),
		"Authorization: Bearer " + strings.Repeat("x", 24),
	} {
		if got := Secrets(s); strings.Contains(got, "AKIA") || strings.Contains(got, "BEGIN RSA") || strings.Contains(got, "ghp_") || strings.Contains(got, "Bearer "+strings.Repeat("x", 24)) {
			t.Errorf("Secrets(%q) = %q, secret shape survived", s, got)
		}
	}
	if !strings.Contains(Secrets("no secrets here"), "no secrets here") {
		t.Error("Secrets must not alter ordinary text")
	}
}

func TestPathsRedaction(t *testing.T) {
	msg := "open /home/dev/projects/acme-corp/.pose/specs/secret-launch/spec.md: no such file"
	got := Paths(msg)
	if strings.Contains(got, "acme-corp") || strings.Contains(got, "secret-launch") {
		t.Errorf("Paths() leaked repository layout: %q", got)
	}
	if !strings.Contains(got, "[PATH]") {
		t.Errorf("expected a [PATH] placeholder: %q", got)
	}
	if !strings.Contains(got, "no such file") {
		t.Errorf("Paths() must preserve surrounding prose: %q", got)
	}
}

func TestLoggerCorrelatesTraceContextAndRedacts(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf)

	traceID, _ := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	fakeAWSKeyShapedFixture := "AKIA" + "ABCDEFGHIJKLMNOP"
	l.Emit(ctx, "pose_check", "gate", "error", "failed near "+fakeAWSKeyShapedFixture)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON log line: %v\n%s", err, buf.String())
	}
	if rec["trace_id"] != traceID.String() || rec["span_id"] != spanID.String() {
		t.Errorf("log record not correlated to the active span: %+v", rec)
	}
	if rec["tool"] != "pose_check" || rec["risk_class"] != "gate" || rec["outcome"] != "error" {
		t.Errorf("log record missing low-cardinality fields: %+v", rec)
	}
	if strings.Contains(rec["message"].(string), "AKIA") {
		t.Errorf("log message leaked a secret-shaped fixture: %v", rec["message"])
	}
}

func TestLoggerWithoutSpanOmitsCorrelationFields(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf)
	l.Emit(context.Background(), "pose_check", "gate", "ok", "ok")
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := rec["trace_id"]; ok {
		t.Errorf("no active span: trace_id must be omitted, got %+v", rec)
	}
}

// otlpReceiver is a minimal test double: it just counts POST requests to
// any path, enough to prove the exporter actually sent something.
func otlpReceiver(t *testing.T) (*httptest.Server, *int64) {
	t.Helper()
	var count int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &count
}

func TestInitEnabledExportsSpansAndMetricsToConfiguredEndpoint(t *testing.T) {
	srv, count := otlpReceiver(t)

	ctx := context.Background()
	p, err := Init(ctx, Config{Enabled: true, Endpoint: srv.URL, Insecure: true, SampleRatio: 1.0, ExportPeriod: 3600})
	if err != nil {
		t.Fatalf("Init(enabled) error: %v", err)
	}
	_, span := p.Tracer.Start(ctx, "pose_get_spec", trace.WithAttributes(attribute.String("pose.mcp.tool", "pose_get_spec")))
	span.End()
	p.Instr.CallDuration.Record(ctx, 12.5)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
	if atomic.LoadInt64(count) == 0 {
		t.Error("enabled provider must export to the configured endpoint on shutdown, got zero requests")
	}
}

func TestInitEnabledDoesNotBlockOnUnreachableEndpoint(t *testing.T) {
	srv, _ := otlpReceiver(t)
	srv.Close() // now guaranteed unreachable

	ctx := context.Background()
	p, err := Init(ctx, Config{Enabled: true, Endpoint: srv.URL, Insecure: true, SampleRatio: 1.0, ExportPeriod: 3600})
	if err != nil {
		t.Fatalf("Init must not fail eagerly for an unreachable endpoint: %v", err)
	}
	_, span := p.Tracer.Start(ctx, "pose_check")
	span.End()

	done := make(chan error, 1)
	go func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		done <- p.Shutdown(shutdownCtx)
	}()
	select {
	case <-done:
		// Shutdown returned (with or without an export error) within the
		// bounded timeout — the caller (MCP server startup/shutdown) is
		// never blocked by an unreachable collector.
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown against an unreachable endpoint did not respect its context deadline")
	}
}

package observability

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"

	"github.com/harne8/pose-mcp/internal/version"
)

// Provider bundles the tracer/meter this process uses for the lifetime of
// a `pose serve-mcp` run, plus the trace-correlated logger (R3) and a
// Shutdown that flushes and tears everything down. When Config.Enabled is
// false, every field is the OTel no-op implementation: real code paths
// that call Tracer()/Meter() never need an `if enabled` branch of their
// own, and pay zero cost (no network, no background goroutine).
type Provider struct {
	Tracer   trace.Tracer
	Meter    metric.Meter
	Instr    *Instruments
	Log      *Logger
	shutdown func(context.Context) error
}

// Instruments are created once per Provider and reused for the life of
// the process — R2's three signal categories (latency, outcome/denial,
// saturation), all deliberately free of any user/repo/path attribute.
type Instruments struct {
	CallDuration  metric.Float64Histogram   // ms, attrs: tool, risk_class, outcome
	PolicyDenials metric.Int64Counter       // attrs: tool
	InFlight      metric.Int64UpDownCounter // attrs: tool — current concurrency (saturation)
}

func newInstruments(m metric.Meter) (*Instruments, error) {
	dur, err := m.Float64Histogram("pose.mcp.tool.call.duration",
		metric.WithDescription("MCP tool call duration"), metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	denials, err := m.Int64Counter("pose.mcp.policy.denial.count",
		metric.WithDescription("MCP tool calls denied by policy"))
	if err != nil {
		return nil, err
	}
	inflight, err := m.Int64UpDownCounter("pose.mcp.tool.call.inflight",
		metric.WithDescription("MCP tool calls currently executing"))
	if err != nil {
		return nil, err
	}
	return &Instruments{CallDuration: dur, PolicyDenials: denials, InFlight: inflight}, nil
}

// noopProvider is what every disabled or misconfigured run gets — safe,
// inert, and indistinguishable in caller code from the real thing.
func noopProvider() *Provider {
	m := noopmetric.NewMeterProvider().Meter("pose-mcp")
	instr, _ := newInstruments(m) // no-op meter never errors
	return &Provider{
		Tracer:   nooptrace.NewTracerProvider().Tracer("pose-mcp"),
		Meter:    m,
		Instr:    instr,
		Log:      newLogger(nil),
		shutdown: func(context.Context) error { return nil },
	}
}

// Init builds the real OTLP-exporting Provider when cfg.Enabled, or the
// no-op Provider otherwise. It never returns an error for "disabled" —
// only for a genuine misconfiguration while actually enabled (e.g. an
// unparseable endpoint), since a transient exporter/network failure must
// never block MCP server startup (Technical risk: bounded
// buffering/backpressure, never a hard dependency).
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return noopProvider(), nil
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName("pose-mcp"),
		semconv.ServiceVersion(version.Version),
	))
	if err != nil {
		return nil, fmt.Errorf("observability: resource: %w", err)
	}

	traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(cfg.Endpoint)}
	if cfg.Insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		traceOpts = append(traceOpts, otlptracehttp.WithHeaders(cfg.Headers))
	}
	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("observability: trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithBatcher(traceExporter),
	)

	metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpointURL(cfg.Endpoint)}
	if cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		metricOpts = append(metricOpts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}
	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, fmt.Errorf("observability: metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(secondsToDuration(cfg.ExportPeriod)))),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	meter := mp.Meter("pose-mcp")
	instr, err := newInstruments(meter)
	if err != nil {
		return nil, fmt.Errorf("observability: instruments: %w", err)
	}

	return &Provider{
		Tracer: tp.Tracer("pose-mcp"),
		Meter:  meter,
		Instr:  instr,
		Log:    newLogger(os.Stderr),
		shutdown: func(ctx context.Context) error {
			tErr := tp.Shutdown(ctx)
			mErr := mp.Shutdown(ctx)
			if tErr != nil {
				return tErr
			}
			return mErr
		},
	}, nil
}

// Shutdown flushes and tears down the provider. Safe to call on a no-op
// Provider (returns nil immediately).
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}

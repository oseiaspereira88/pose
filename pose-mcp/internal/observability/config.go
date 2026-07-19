// Package observability implements the opt-in OpenTelemetry operational
// signals for the POSE MCP server (spec pose-otel-observability): traces
// and metrics for MCP tool calls and validation-plan orchestration, plus a
// trace-correlated, redacted structured logger. Everything here is inert —
// zero allocation beyond a config read, zero network — unless explicitly
// enabled; POSE keeps working fully offline by default (Constraint).
package observability

import (
	"os"
	"strconv"
	"strings"
)

// Config is the resolved opt-in observability configuration. Two
// independent gates must both be satisfied before anything leaves the
// process: POSE_OTEL_ENABLED (POSE's own opt-in) and
// OTEL_EXPORTER_OTLP_ENDPOINT (the standard OTel env var — no default
// endpoint is ever baked into the binary). Either absent means Enabled is
// false and Init becomes a pure no-op (Compatibility: no exporter
// configuration means no network transmission).
type Config struct {
	Enabled      bool
	Endpoint     string
	Insecure     bool
	Headers      map[string]string
	SampleRatio  float64
	ExportPeriod int // metric export interval, seconds
}

// FromEnv resolves Config from environment variables, using the same
// standard OTel env var names wherever one exists so POSE composes with
// any existing OTel-aware tooling/documentation an operator already knows.
func FromEnv() Config {
	cfg := Config{
		SampleRatio:  1.0,
		ExportPeriod: 15,
	}
	poseEnabled := os.Getenv("POSE_OTEL_ENABLED") == "1" || strings.EqualFold(os.Getenv("POSE_OTEL_ENABLED"), "true")
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	cfg.Enabled = poseEnabled && endpoint != ""
	if !cfg.Enabled {
		return cfg
	}
	cfg.Endpoint = endpoint
	cfg.Insecure = strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), "true")
	cfg.Headers = parseHeaders(os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil && ratio >= 0 && ratio <= 1 {
			cfg.SampleRatio = ratio
		}
	}
	if v := os.Getenv("OTEL_METRIC_EXPORT_INTERVAL"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.ExportPeriod = ms / 1000
			if cfg.ExportPeriod < 1 {
				cfg.ExportPeriod = 1
			}
		}
	}
	return cfg
}

// parseHeaders parses the OTLP-standard "key1=value1,key2=value2" header
// list format (OTEL_EXPORTER_OTLP_HEADERS).
func parseHeaders(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, pair := range strings.Split(raw, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

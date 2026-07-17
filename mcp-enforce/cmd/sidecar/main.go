// Command sidecar runs the MCP enforcement reverse proxy (mcp-enforce-sidecar):
// it applies the shared mcp-enforce policy gate + audit in front of a foreign
// MCP server (e.g. graphforge/mcp-server) on the composition network, per
// ADR-021. The upstream server stays unmodified and reachable only through this
// proxy in the composed platform.
package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	mcpenforce "github.com/crisol/mcp-enforce"
	"github.com/crisol/mcp-enforce/sidecar"
)

func main() {
	addr := envOr("GF_SIDECAR_ADDR", ":8770")
	upstreamRaw := envOr("GF_SIDECAR_UPSTREAM_URL", "http://mcp-server:8765")

	upstream, err := url.Parse(upstreamRaw)
	if err != nil {
		log.Fatalf("mcp-enforce-sidecar: invalid GF_SIDECAR_UPSTREAM_URL %q: %v", upstreamRaw, err)
	}

	// Policy/audit config from the GF_SIDECAR_ env prefix (OPA_URL, OPA_PATH,
	// OPA_TIMEOUT, REQUIRE_PRINCIPAL). Empty OPA_URL = dev allow-all.
	// Exchange recorder (ADR-022/ADR-017): when GF_SIDECAR_EXCHANGE_LOG is set,
	// append each gated live MCP exchange as JSON lines for post-hoc replay/audit.
	var recorder sidecar.ExchangeRecorder
	if logPath := os.Getenv("GF_SIDECAR_EXCHANGE_LOG"); logPath != "" {
		f, ferr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if ferr != nil {
			log.Fatalf("mcp-enforce-sidecar: open exchange log %q: %v", logPath, ferr)
		}
		defer f.Close()
		recorder = sidecar.NewJSONLRecorder(f)
	}

	cfg := mcpenforce.ConfigFromEnv("GF_SIDECAR_", "graphforge/mcp/allow")
	sc := sidecar.New(sidecar.Config{
		Gate:           mcpenforce.NewPolicyGate(cfg),
		Auditor:        mcpenforce.NewSlogAuditor(nil, "graphforge-mcp-sidecar"),
		Upstream:       upstream,
		IdentitySecret: []byte(os.Getenv("GF_SIDECAR_IDENTITY_SECRET")),
		Recorder:       recorder,
	})

	policyMode := "allow-all"
	if cfg.OPAURL != "" {
		policyMode = "opa:" + cfg.OPAURL
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           sc,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("crisol mcp-enforce-sidecar listening addr=%s upstream=%s policy=%s require_principal=%v",
		addr, upstreamRaw, policyMode, cfg.RequirePrincipal)
	log.Fatal(srv.ListenAndServe())
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

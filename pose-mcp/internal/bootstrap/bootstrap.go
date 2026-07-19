// Package bootstrap wires the POSE MCP server from environment configuration.
// It is invoked by the unified `pose serve-mcp` command.
// (unified CLI, spec pose-cli-go-unification).
package bootstrap

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	mcpenforce "github.com/harne8/mcp-enforce"
	"github.com/harne8/pose-mcp/internal/mcpserver"
	"github.com/harne8/pose-mcp/internal/observability"
	"github.com/harne8/pose-mcp/internal/pose"
)

// opaConfigFromEnv reads OPA integration settings from the POSE_MCP_ env prefix
// via the shared mcp-enforce module (ADR-021):
//
//   - POSE_MCP_OPA_URL   Base URL of the OPA server (empty = dev allow-all).
//   - POSE_MCP_OPA_PATH  OPA policy path (default: "pose/mcp/allow").
//   - POSE_MCP_OPA_TIMEOUT  Evaluation timeout in seconds (default: "2").
//   - POSE_MCP_REQUIRE_PRINCIPAL  "1"/"true" denies anonymous tools/call even
//     without OPA (strict authz; pose-mcp-enterprise-hardening).
func opaConfigFromEnv() mcpserver.PolicyConfig {
	return mcpenforce.ConfigFromEnv("POSE_MCP_", "pose/mcp/allow")
}

// Run starts the MCP server using environment configuration. args are the
// remaining command-line arguments (used only to detect --stdio). It blocks
// until the server exits.
func Run(args []string) {
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	// Opt-in OpenTelemetry signals (spec pose-otel-observability): inert
	// unless both POSE_OTEL_ENABLED=1 and OTEL_EXPORTER_OTLP_ENDPOINT are
	// set. A misconfiguration while enabled must never block startup — log
	// and fall back to the no-op provider instead of failing the process.
	obs, obsErr := observability.Init(ctx, observability.FromEnv())
	if obsErr != nil {
		log.Printf("pose-mcp: observability disabled (init error): %v", obsErr)
		obs, _ = observability.Init(ctx, observability.Config{})
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = obs.Shutdown(shutdownCtx)
	}()

	// Empty default root = empty start: no pre-wired default project; projects are
	// resolved only after onboarding (empty-start-no-default-project). When
	// POSE_PROJECT_ROOT is set it must have a .pose/ and becomes the default.
	root := os.Getenv("POSE_PROJECT_ROOT")
	addr := envOr("POSE_MCP_ADDR", ":8790")
	token := envOr("POSE_MCP_TOKEN", "")
	// portal-workspace-scale: separate secret gating POST /admin/refresh, the
	// Conductor's push signal after onboarding/reindex. Empty = disabled (dev).
	adminToken := envOr("POSE_MCP_ADMIN_TOKEN", "")
	// Multi-project (pose-mcp-multi-project): additional projects are resolved by
	// project_id from a scan of HARNE8_PROJECTS_DIR (dirname == project_id) plus an
	// explicit override.
	defaultProjectID := os.Getenv("POSE_DEFAULT_PROJECT_ID")
	projectsDir := envOr("HARNE8_PROJECTS_DIR", "")
	// Default prefix is empty: the clone dir name IS the project_id
	// (HARNE8_PROJECTS_DIR/<project_id>), the canonical onboarding convention.
	projectIDPrefix := os.Getenv("HARNE8_PROJECT_ID_PREFIX")

	if root != "" {
		if _, err := os.Stat(filepath.Join(root, ".pose")); err != nil {
			log.Fatalf("pose-mcp: no .pose/ under project root %q: %v", root, err)
		}
		if defaultProjectID == "" {
			defaultProjectID = "proj." + filepath.Base(filepath.Clean(root))
		}
	} else {
		defaultProjectID = "" // no root -> no default project
	}

	explicit, err := pose.ParseRootsJSON(os.Getenv("POSE_PROJECT_ROOTS"))
	if err != nil {
		log.Fatalf("pose-mcp: %v", err)
	}
	// pose-mcp-project-scope-contract: opt-in fail-closed mode for a
	// deployment that has onboarded more than one project — an empty
	// project_id becomes a structured project_ambiguous error instead of
	// silently resolving to the default root. Off by default: existing
	// single-project stdio deployments are always unaffected regardless of
	// this flag (strict mode only trips when >1 project is registered).
	strictSelection := envOr("POSE_MCP_STRICT_PROJECT_SELECTION", "") != ""
	roots := pose.NewRoots(pose.RootsConfig{
		DefaultRoot:      root,
		DefaultProjectID: defaultProjectID,
		ProjectsDir:      projectsDir,
		ProjectIDPrefix:  projectIDPrefix,
		Explicit:         explicit,
		StrictSelection:  strictSelection,
	})

	authMode := "off"
	if token != "" {
		authMode = "bearer"
	}

	opaCfg := opaConfigFromEnv()
	policy := mcpserver.NewPolicyGate(opaCfg)
	policyMode := "allow-all"
	if opaCfg.OPAURL != "" {
		policyMode = "opa:" + opaCfg.OPAURL
	}

	// Identity binding (ADR-007): POSE_MCP_IDENTITY_SECRET verifies the
	// X-MCP-Execution-Identity token; POSE_MCP_REQUIRE_IDENTITY (via opaConfigFromEnv)
	// denies calls without a run-bound identity. Empty secret = binding disabled.
	server := mcpserver.NewWithRootsAndPolicy(roots, policy).
		WithIdentitySecret([]byte(os.Getenv("POSE_MCP_IDENTITY_SECRET"))).
		WithObservability(obs)

	// Conductor run reporter (external-run-reporters): enable conductor_run_* tools
	// when CONDUCTOR_URL and CONDUCTOR_PROJECT_ID are set.
	conductorURL := os.Getenv("CONDUCTOR_URL")
	conductorProjectID := os.Getenv("CONDUCTOR_PROJECT_ID")
	conductorRunToken := os.Getenv("CONDUCTOR_RUN_TOKEN")
	if conductorURL != "" && conductorProjectID != "" {
		server.WithReporter(mcpserver.NewConductorClient(conductorURL, conductorProjectID, conductorRunToken))
		log.Printf("pose-mcp conductor_reporter=enabled project=%s", conductorProjectID)
	}

	// Stdio transport: spawn-safe for claude-native (subprocess) deployments.
	// Activated by --stdio flag or POSE_MCP_STDIO=1 env var.
	if stdioMode(args) {
		log.SetOutput(os.Stderr)
		log.Printf("pose-mcp default_root=%s projects=%v transport=stdio policy=%s", root, roots.Projects(), policyMode)
		if err := server.ServeStdio(ctx); err != nil {
			log.Fatal(err)
		}
		return
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.Handler(token, adminToken),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	log.Printf("pose-mcp listening addr=%s default_root=%s projects=%v transport=streamable-http auth=%s policy=%s", addr, root, roots.Projects(), authMode, policyMode)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func stdioMode(args []string) bool {
	if os.Getenv("POSE_MCP_STDIO") == "1" {
		return true
	}
	for _, arg := range args {
		if arg == "--stdio" {
			return true
		}
	}
	return false
}

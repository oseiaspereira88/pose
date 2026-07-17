// pose-mcp is the official MCP adapter of POSE (ADR-003): a project-scoped,
// read-only Streamable HTTP server exposing POSE specs, workflows and rules
// to agents and orchestrators.
//
// Compatibility alias: the unified CLI (`pose serve-mcp`, spec
// pose-cli-go-unification) is the canonical entrypoint; this binary keeps
// existing deployments (compose, wrappers) working unchanged.
package main

import (
	"os"

	"github.com/crisol/pose-mcp/internal/bootstrap"
)

func main() {
	bootstrap.Run(os.Args[1:])
}

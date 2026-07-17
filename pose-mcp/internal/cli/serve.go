package cli

import "github.com/crisol/pose-mcp/internal/bootstrap"

// runServeMCP starts the MCP server (blocking). Split into its own file so
// tests of the dispatcher don't need the server wiring.
func runServeMCP(args []string) {
	bootstrap.Run(args)
}

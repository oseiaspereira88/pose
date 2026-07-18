// pose is the native-only unified POSE CLI and MCP server.
package main

import (
	"os"
	"path/filepath"

	"github.com/crisol/pose-mcp/internal/cli"
)

func main() {
	if base := filepath.Base(os.Args[0]); base == "pre-commit" || base == "post-merge" {
		os.Exit(cli.HookMain(base, os.Stdout, os.Stderr))
	}
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}

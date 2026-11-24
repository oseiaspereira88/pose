// pose is the native-only unified POSE CLI and MCP server.
package main

import (
	"os"
	"path/filepath"

	"github.com/harne8/pose-mcp/internal/cli"
	"github.com/harne8/pose-mcp/internal/pose"
)

func main() {
	pose.ReviewShimAvailable = true
	if base := filepath.Base(os.Args[0]); base == "pre-commit" || base == "post-merge" {
		os.Exit(cli.HookMain(base, os.Stdout, os.Stderr))
	}
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}

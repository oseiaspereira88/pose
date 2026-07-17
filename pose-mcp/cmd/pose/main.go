// pose is the unified POSE CLI (spec pose-cli-go-unification): native
// subcommands (version, init, serve-mcp) plus transparent delegation to the
// bash engine in .pose/scripts/ for everything not yet ported.
package main

import (
	"os"

	"github.com/crisol/pose-mcp/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}

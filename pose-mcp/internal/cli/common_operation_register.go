package cli

import (
	"io"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// The read-only, side-effect-free commands the MCP wrappers of the domain package
// run in process instead of spawning the binary (spec
// pose-common-domain-and-cli-mcp-handoff R1). The gates (check, skills-check,
// lint-spec) keep the process boundary: they are long, can be killed on a deadline,
// and their verdict must never be a half-finished run.
func init() {
	posemodel.RegisterInProcessCommand("suggest", func(root string, args []string, stdout, stderr io.Writer) int {
		return cmdSuggest(root, args, stdout, stderr)
	})
	posemodel.RegisterInProcessCommand("followups", func(root string, args []string, stdout, stderr io.Writer) int {
		return cmdFollowups(root, args, stdout, stderr)
	})
}

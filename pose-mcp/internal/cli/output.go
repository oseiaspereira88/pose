package cli

// How a command reaches the renderer (spec pose-cli-output-rendering-system).
//
// Commands receive two writers and nothing else, so this is where they become a
// renderer with a capability profile per stream. Colour and animation are
// resolved from the environment here; the flags that override them
// (--color, --quiet, --verbose) are parsed by each command and applied on top.

import (
	"io"
	"os"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
)

// render builds the renderer for one invocation.
func render(stdout, stderr io.Writer) *cliout.Renderer {
	return cliout.Resolve(stdout, stderr, cliout.ColorAuto, os.Getenv)
}

// renderWithColor builds the renderer with an explicit --color choice.
func renderWithColor(stdout, stderr io.Writer, mode cliout.ColorMode) *cliout.Renderer {
	return cliout.Resolve(stdout, stderr, mode, os.Getenv)
}

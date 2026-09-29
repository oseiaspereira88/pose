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

// outputFlags are the result-channel flags every gate accepts (spec
// pose-cli-output-machine-channel, R1).
type outputFlags struct {
	JSON  bool
	Quiet bool
	Color cliout.ColorMode
}

// splitOutputFlags removes --json, --quiet and --color <mode> from args and
// returns the rest for the command's own parser. A malformed --color is
// reported as the usage error the caller should print.
func splitOutputFlags(args []string) ([]string, outputFlags, string) {
	flags := outputFlags{Color: cliout.ColorAuto}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.JSON = true
		case "--quiet":
			flags.Quiet = true
		case "--color":
			if i+1 >= len(args) {
				return nil, flags, "--color requires a value (auto, always or never)"
			}
			i++
			mode, ok := cliout.ParseColorMode(args[i])
			if !ok {
				return nil, flags, "--color must be auto, always or never, not " + args[i]
			}
			flags.Color = mode
		default:
			rest = append(rest, args[i])
		}
	}
	return rest, flags, ""
}

// gateOutput is one gate's result channel: findings and fields go through the
// renderer, and the verdict keeps its pinned human line while the machine
// document records the same decision.
type gateOutput struct {
	r     *cliout.Renderer
	flags outputFlags
	// result is stdout for a full human run and io.Discard otherwise, for the
	// prose a gate prints around its findings.
	result io.Writer
}

func newGateOutput(command string, flags outputFlags, stdout, stderr io.Writer) *gateOutput {
	r := renderWithColor(stdout, stderr, flags.Color)
	r.SetQuiet(flags.Quiet)
	if flags.JSON {
		r.RecordJSON(command)
	}
	result := stdout
	if flags.JSON || flags.Quiet {
		result = io.Discard
	}
	return &gateOutput{r: r, flags: flags, result: result}
}

// Field writes a contract `name=value` line; --quiet omits it, --json records it.
func (g *gateOutput) Field(name, value string) {
	if g.flags.Quiet && !g.flags.JSON {
		return
	}
	g.r.Field(name, value)
}

// Verdict prints the gate's pinned verdict line, or records state and text
// under --json. It prints under --quiet: the verdict is what --quiet keeps.
func (g *gateOutput) Verdict(state cliout.State, line, text string) {
	g.VerdictWord(state, "", line, text)
}

// VerdictWord is Verdict for a decision the default SUCCESS/FAILURE words do
// not name, such as a tolerated failure or a stale state. The word is the
// untranslated one a machine keys on.
func (g *gateOutput) VerdictWord(state cliout.State, word, line, text string) {
	if g.flags.JSON {
		g.r.Verdict(cliout.Verdict{State: state, Word: word, Text: text})
		return
	}
	g.r.ContractLine(line)
}

// RecordNote adds a machine-only field that has no contract line of its own.
func (g *gateOutput) RecordNote(name, value string) { g.r.RecordField(name, value) }

// Close flushes the machine document, if any.
func (g *gateOutput) Close() { _ = g.r.FlushJSON() }

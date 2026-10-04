package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/harne8/pose-mcp/internal/pose"
)

// cmdStateGovernance renders the live effective-governance projection (spec
// pose-effective-governance-projection): what is in force here, not what the
// engine ships. Read-only.
func cmdStateGovernance(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose state --governance [--scope <spec:slug|milestone:roadmap/id|roadmap:slug>] [--json]"
	scope, jsonOutput := "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--governance":
		case "--json":
			jsonOutput = true
		case "--scope":
			if i+1 >= len(args) {
				return usageError(stderr, usage)
			}
			i++
			scope = args[i]
		default:
			return usageError(stderr, usage)
		}
	}
	projection, err := pose.Store{Root: root}.EffectiveGovernance(scope)
	if err != nil {
		render(stdout, stderr).Failure("pose state --governance: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, projection)
	}
	out := render(stdout, stderr)
	for _, entry := range projection.Entries {
		out.Field("governance."+entry.ID, fmt.Sprintf("kind=%s supported=%t configured=%t applicable=%t effective=%t%s — %s",
			entry.Kind, entry.Supported, entry.Configured, entry.Applicable, entry.Effective, governanceReasons(entry.Reasons), entry.Explanation))
	}
	if ctx := projection.ScopeBundle; ctx != nil {
		switch {
		case ctx.BundleID == "":
			out.Field("governance.scope", projection.Scope+" has no sealed bundle")
		case ctx.Unstamped:
			out.Field("governance.scope", fmt.Sprintf("%s bundle %s carries no governing_contracts stamp: it is read by the dated adoption rule", projection.Scope, ctx.BundleID))
		default:
			line := fmt.Sprintf("%s bundle %s stamped %s; assurance %s", projection.Scope, ctx.BundleID, strings.Join(ctx.StampedContracts, ","), ctx.IdentityAssurance)
			if len(ctx.NotStamped) > 0 {
				line += "; sealed before " + strings.Join(ctx.NotStamped, ",") + " existed and never held to them"
			}
			out.Field("governance.scope", line)
		}
	}
	return 0
}

func governanceReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return " reasons=" + strings.Join(reasons, ",")
}

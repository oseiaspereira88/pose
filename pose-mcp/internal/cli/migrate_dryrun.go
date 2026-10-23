package cli

import (
	"fmt"
	"io"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdMigrate is `pose migrate v7 --dry-run [--json]` (spec
// pose-v7-legacy-cleanup-plan): the inventory a future major would convert,
// keep or refuse, measured on this repository. No apply is implemented, and none is promised
// here: a later apply would be announced when it exists.
func cmdMigrate(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose migrate v7 --dry-run [--json]"
	if len(args) == 0 || args[0] != "v7" {
		return usageError(stderr, usage)
	}
	dryRun, jsonOutput := false, false
	for _, arg := range args[1:] {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--json":
			jsonOutput = true
		case "--apply":
			render(stdout, stderr).Failure("pose migrate: no apply is available in this engine; the only mode is `pose migrate v7 --dry-run`, which reads the repository, writes nothing and reports what a migration would convert, keep or refuse")
			return 2
		default:
			return usageError(stderr, usage)
		}
	}
	if !dryRun {
		return usageError(stderr, usage)
	}
	inv, err := (posemodel.Store{Root: root}).InventoryLegacy()
	if err != nil {
		render(stdout, stderr).Failure("pose migrate: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, inv)
	}
	out := render(stdout, stderr)
	out.Field("migrate.revision", inv.Revision)
	out.Field("migrate.writes", "none (dry-run)")
	for _, class := range inv.Classes {
		out.Field("migrate."+class.ID, fmt.Sprintf("%d — %s; removal: %s", class.Count, class.Action, class.Removal))
		if details := inv.Details[class.ID]; len(details) > 0 {
			shown := details
			if len(shown) > 10 {
				shown = append(append([]string{}, details[:10]...), fmt.Sprintf("… %d more", len(details)-10))
			}
			out.Field("  "+class.ID, strings.Join(shown, ", "))
		}
	}
	for _, risk := range inv.Risks {
		out.Field("migrate.risk", risk)
	}
	for _, l := range inv.Limitations {
		out.Field("migrate.limitation", l)
	}
	return 0
}

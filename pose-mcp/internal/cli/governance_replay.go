package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

func cmdGovernanceReplay(root string, args []string, stdout, stderr io.Writer) int {
	limit := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
		case "--limit":
			if i+1 >= len(args) {
				return usageError(stderr, "pose stats replay: --limit needs a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 1 || n > 20000 {
				return usageError(stderr, "pose stats replay: limit must be between 1 and 20000")
			}
			limit = n
			i++
		default:
			return usageError(stderr, "Usage: pose stats replay [--limit N] [--json]")
		}
	}
	report, err := (posepkg.Store{Root: root}).GovernanceReplay(limit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

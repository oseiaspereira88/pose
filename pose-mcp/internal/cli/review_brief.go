package cli

// pose review brief (spec pose-delegated-review-brief).

import (
	"encoding/json"
	"io"
	"os"
	"strings"
)

const reviewBriefUsage = "Usage: pose review brief <bundle-id|scope> [--kind review|adjudication|smoke] [--note-file <path>] [--json]"

func cmdReviewBrief(root string, args []string, stdout, stderr io.Writer) int {
	var target, kind, noteFile string
	asJSON := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--kind", "--note-file":
			if i+1 >= len(args) {
				return usageError(stderr, reviewBriefUsage)
			}
			if args[i] == "--kind" {
				kind = args[i+1]
			} else {
				noteFile = args[i+1]
			}
			i++
		case "--json":
			asJSON = true
		default:
			if strings.HasPrefix(args[i], "-") || target != "" {
				return usageError(stderr, reviewBriefUsage)
			}
			target = args[i]
		}
	}
	if target == "" {
		return usageError(stderr, reviewBriefUsage)
	}
	notes := ""
	if noteFile != "" {
		raw, err := os.ReadFile(noteFile)
		if err != nil {
			render(stdout, stderr).Failure("pose review brief: " + err.Error())
			return 1
		}
		notes = string(raw)
	}
	store := cliGovernedStore(root)
	ref := target
	if !strings.HasPrefix(target, "rvb-") {
		resolved, err := resolveReviewBundleScope(store, target)
		if err != nil {
			render(stdout, stderr).Failure("pose review brief: " + strings.TrimPrefix(err.Error(), "pose: "))
			return 1
		}
		ref = resolved
	}
	brief, err := store.RenderReviewBrief(ref, kind, notes)
	if err != nil {
		render(stdout, stderr).Failure("pose review brief: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	if asJSON {
		raw, _ := json.MarshalIndent(brief, "", "  ")
		render(stdout, stderr).ContractLine(string(raw))
		return 0
	}
	render(stdout, stderr).ContractLine(strings.TrimRight(brief.Text, "\n"))
	return 0
}

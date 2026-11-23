package cli

import (
	"io"
	"os"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const reviewDraftCheckUsage = "pose review draft-check [<bundle-id>] --draft <file> [--json]"

// cmdReviewDraftCheck reports the reasons a draft would be refused at
// conclusion, writing nothing (spec pose-review-readiness-before-seal R3).
func cmdReviewDraftCheck(root string, args []string, stdout, stderr io.Writer) int {
	var bundleID, draftPath string
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--draft":
			if i+1 >= len(args) {
				return usageError(stderr, "pose review draft-check: --draft needs a file\nUsage: "+reviewDraftCheckUsage)
			}
			i++
			draftPath = args[i]
		default:
			if strings.HasPrefix(args[i], "-") || bundleID != "" {
				return usageError(stderr, "pose review draft-check: unexpected argument "+args[i]+"\nUsage: "+reviewDraftCheckUsage)
			}
			bundleID = args[i]
		}
	}
	if draftPath == "" {
		return usageError(stderr, "pose review draft-check: name the draft with --draft\nUsage: "+reviewDraftCheckUsage)
	}
	var raw []byte
	var err error
	if draftPath == "-" {
		raw, err = io.ReadAll(io.LimitReader(os.Stdin, 4<<20))
	} else {
		raw, err = os.ReadFile(draftPath)
	}
	if err != nil {
		render(stdout, stderr).Failure("pose review draft-check: " + err.Error())
		return 1
	}
	store := cliGovernedStore(root)
	var reasons []string
	contextPath := os.Getenv(posemodel.ReviewDraftContextEnv)
	switch {
	case bundleID == "" && contextPath == "":
		return usageError(stderr, "pose review draft-check: name the bundle; only a delegated review provides it\nUsage: "+reviewDraftCheckUsage)
	case contextPath != "" && bundleID == "":
		ctx, err := posemodel.ReadReviewDraftContext(contextPath)
		if err != nil {
			render(stdout, stderr).Failure("pose review draft-check: " + strings.TrimPrefix(err.Error(), "pose: "))
			return 1
		}
		reasons = store.CheckReviewDraftIn(ctx, string(raw), time.Now())
	default:
		reasons, err = store.CheckReviewDraft(bundleID, string(raw), time.Now())
		if err != nil {
			render(stdout, stderr).Failure("pose review draft-check: " + strings.TrimPrefix(err.Error(), "pose: "))
			return 1
		}
	}
	if jsonOut {
		writeJSON(stdout, map[string]any{"ok": len(reasons) == 0, "reasons": nonNilStrings(reasons)})
	} else if len(reasons) == 0 {
		render(stdout, stderr).ContractLine("Draft check: the conclusion step would accept this draft.")
	} else {
		out := render(stdout, stderr)
		out.ContractLine("Draft check: the conclusion step would refuse this draft:")
		for _, r := range reasons {
			out.ContractLine("  - " + r)
		}
	}
	if len(reasons) > 0 {
		return 1
	}
	return 0
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

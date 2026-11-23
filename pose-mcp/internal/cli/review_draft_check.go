package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const reviewDraftCheckUsage = "pose review draft-check [<bundle-id|run-id>] --draft <file> [--json]"

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
	raw, err := readReviewDraftInput(draftPath)
	if err != nil {
		render(stdout, stderr).Failure("pose review draft-check: " + err.Error())
		return 1
	}
	store := cliGovernedStore(root)
	var reasons []string
	var ctx *posemodel.ReviewDraftContext
	if contextPath := os.Getenv(posemodel.ReviewDraftContextEnv); contextPath != "" {
		read, err := posemodel.ReadReviewDraftContext(contextPath)
		if err != nil {
			render(stdout, stderr).Failure("pose review draft-check: " + strings.TrimPrefix(err.Error(), "pose: "))
			return 1
		}
		ctx = &read
	}
	switch {
	case bundleID == "" && ctx == nil:
		return usageError(stderr, "pose review draft-check: name the bundle; only a delegated review provides it\nUsage: "+reviewDraftCheckUsage)
	case ctx != nil && (bundleID == "" || bundleID == ctx.Bundle.BundleID):
		// Inside a delegated review the sealed checkout holds no bundle: the
		// run's own bundle, named or not, is judged from the context the
		// dispatch wrote (review round 2, N5).
		reasons = store.CheckReviewDraftIn(*ctx, string(raw), time.Now())
	default:
		reasons, err = store.CheckReviewDraft(bundleID, string(raw), time.Now())
		if err != nil {
			msg := strings.TrimPrefix(err.Error(), "pose: ")
			if ctx != nil {
				msg += "; this run's draft context holds bundle " + ctx.Bundle.BundleID + ", which is checked when it is named or no reference is given"
			}
			render(stdout, stderr).Failure("pose review draft-check: " + msg)
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

// reviewDraftInputLimit bounds the draft read; a larger input is refused,
// never judged by its prefix (review round 2, N4).
const reviewDraftInputLimit = 4 << 20

func readReviewDraftInput(path string) ([]byte, error) {
	var in io.Reader
	if path == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		in = f
	}
	raw, err := io.ReadAll(io.LimitReader(in, reviewDraftInputLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > reviewDraftInputLimit {
		return nil, fmt.Errorf("the draft is larger than %d bytes; a run's final message that large is not checked by a prefix", reviewDraftInputLimit)
	}
	return raw, nil
}

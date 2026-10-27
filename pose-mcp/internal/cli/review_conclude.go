package cli

// pose review conclude <run>: record the conclusion of a delegated review run
// (spec pose-delegated-review-conclusions R1, R3, R4).

import (
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const reviewConcludeUsage = "pose review conclude <run> [--issuer <name>] [--apply] [--json]"

func cmdReviewConclude(root string, args []string, stdout, stderr io.Writer) int {
	var run, issuer string
	apply, jsonOut := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			apply = true
		case "--json":
			jsonOut = true
		case "--issuer":
			if i+1 >= len(args) {
				return usageError(stderr, "pose review conclude: --issuer needs a name\nUsage: "+reviewConcludeUsage)
			}
			i++
			issuer = args[i]
		default:
			if strings.HasPrefix(args[i], "-") || run != "" {
				return usageError(stderr, "pose review conclude: unexpected argument "+args[i]+"\nUsage: "+reviewConcludeUsage)
			}
			run = args[i]
		}
	}
	if run == "" {
		return usageError(stderr, "pose review conclude: name the run\nUsage: "+reviewConcludeUsage)
	}
	store := cliGovernedStore(root)
	if issuer == "" {
		policy, _, _, err := store.LoadDelegationPolicy()
		if err != nil {
			render(stdout, stderr).Failure("pose review conclude: " + strings.TrimPrefix(err.Error(), "pose: "))
			return 1
		}
		issuer = policy.ConclusionsIssuer()
	}
	if issuer == "" {
		return usageError(stderr, "pose review conclude: name the issuer that signs it with --issuer, or set conclusions.issuer in .pose/policy/delegation.json")
	}
	conclusion, err := store.ConcludeReviewRun(run, issuer, apply, time.Now())
	if err != nil {
		render(stdout, stderr).Failure("pose review conclude: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	return printReviewConclusion(stdout, stderr, jsonOut, conclusion, apply)
}

func printReviewConclusion(stdout, stderr io.Writer, jsonOut bool, c posemodel.ReviewConclusion, apply bool) int {
	code := 0
	if c.State != posemodel.ReviewConclusionRecorded {
		code = 1
	}
	if jsonOut {
		writeJSON(stdout, c)
		return code
	}
	out := render(stdout, stderr)
	out.Field("review_conclusion.run", c.RunID)
	out.Field("review_conclusion.state", c.State)
	if c.Decision != "" {
		out.Field("review_conclusion.decision", c.Decision)
	}
	if c.Attestation != "" {
		out.Field("review_conclusion.attestation", c.Attestation)
	}
	if c.ProofID != "" {
		out.Field("review_conclusion.proof", c.ProofID)
	}
	for _, reason := range c.Reasons {
		out.Field("review_conclusion.reason", reason)
	}
	if c.Replayed {
		out.Field("review_conclusion.replayed", "true (recorded before; nothing new)")
	}
	out.Field("review_conclusion.apply", fmt.Sprint(apply))
	return code
}

// delegateAutoConclude records the conclusion of a review run that just
// completed, when the instance turned the capability on, and says what it did.
func delegateAutoConclude(store posemodel.Store, run posemodel.ReviewRun, err error, stdout, stderr io.Writer) {
	if err != nil {
		return
	}
	conclusion, cerr := store.AutoConcludeReviewRun(run, time.Now())
	if cerr != nil {
		render(stdout, stderr).Hint("pose delegate: the conclusion was not recorded: " + strings.TrimPrefix(cerr.Error(), "pose: "))
		return
	}
	if conclusion == nil {
		return
	}
	line := "pose delegate: conclusion " + conclusion.State
	if conclusion.Attestation != "" {
		line += ": attestation " + conclusion.Attestation
	}
	if len(conclusion.Reasons) > 0 {
		line += ": " + strings.Join(conclusion.Reasons, "; ")
	}
	render(stdout, stderr).Hint(line)
}

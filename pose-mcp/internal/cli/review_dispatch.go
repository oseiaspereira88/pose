package cli

// pose review dispatch (spec pose-delegated-review-dispatch).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const reviewDispatchUsage = "Usage: pose review dispatch <bundle-id|scope> --via <adapter> [--kind review|adjudication|smoke] [--note-file <path>] [--reason <text>] [--apply]"

func cmdReviewDispatch(root string, args []string, stdout, stderr io.Writer) int {
	var target, adapter, kind, noteFile, reason string
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--via", "--kind", "--note-file", "--reason":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return usageError(stderr, reviewDispatchUsage)
			}
			switch args[i] {
			case "--via":
				adapter = args[i+1]
			case "--kind":
				kind = args[i+1]
			case "--reason":
				reason = args[i+1]
			default:
				noteFile = args[i+1]
			}
			i++
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(args[i], "-") || target != "" {
				return usageError(stderr, reviewDispatchUsage)
			}
			target = args[i]
		}
	}
	if target == "" {
		return usageError(stderr, reviewDispatchUsage)
	}
	out := render(stdout, stderr)
	notes := ""
	if noteFile != "" {
		raw, err := posemodel.ReadReviewAdapterFile(noteFile, posemodel.ReviewNotesMaxBytes)
		if err != nil {
			out.Failure("pose review dispatch: " + err.Error())
			return 1
		}
		notes = string(raw)
	}
	store := cliGovernedStore(root)
	ref := target
	if !strings.HasPrefix(target, "rvb-") {
		resolved, err := resolveReviewBundleScope(store, target)
		if err != nil {
			out.Failure("pose review dispatch: " + strings.TrimPrefix(err.Error(), "pose: "))
			return 1
		}
		ref = resolved
	}
	plan, err := store.PlanReviewDispatch(ref, kind, notes, adapter, reason)
	if errors.Is(err, posemodel.ErrNoReviewerAdapter) {
		// Not silent: hand over the brief and say what to configure.
		example, _ := json.MarshalIndent(posemodel.ExampleReviewersPolicy(), "", "  ")
		out.Failure("pose review dispatch: no reviewer adapter is configured; add .pose/policy/reviewers.json, for example:\n" + string(example) + "\nThe brief a reviewer would receive follows.")
		out.ContractLine(strings.TrimRight(plan.Brief.Text, "\n"))
		return 1
	}
	if err != nil {
		out.Failure("pose review dispatch: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	if adapter == "" {
		return usageError(stderr, reviewDispatchUsage)
	}
	for _, warning := range plan.Warnings {
		out.Hint("pose review dispatch: " + warning)
	}
	if plan.SameVendorWarning != "" {
		out.Hint("pose review dispatch: " + plan.SameVendorWarning)
	}
	out.Field("review_dispatch.scope", plan.Brief.Scope)
	out.Field("review_dispatch.bundle", plan.Bundle.BundleID)
	out.Field("review_dispatch.kind", plan.Brief.Kind)
	out.Field("review_dispatch.brief_digest", plan.Brief.Digest)
	out.Field("review_dispatch.adapter", fmt.Sprintf("%s (%s %s)", plan.Adapter, plan.Config.Vendor, plan.Config.Model))
	out.Field("review_dispatch.sealed_commit", plan.SealedCommit)
	out.Field("review_dispatch.prior_runs", fmt.Sprint(plan.PriorRuns))
	safety, _ := posemodel.ResolveReviewerContainment(plan.Config)
	if safety != nil {
		raw, _ := json.Marshal(safety)
		out.Field("review_dispatch.safety", string(raw))
	}
	if !apply {
		out.Field("review_dispatch.apply", "false")
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	run, err := store.RunReviewDispatchContext(ctx, plan, time.Now)
	if run.Safety != nil {
		raw, _ := json.Marshal(run.Safety)
		out.Field("review_dispatch.safety", string(raw))
		out.Field("review_dispatch.run", run.RunID)
	}
	if err != nil {
		out.Failure("pose review dispatch: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	out.Field("review_dispatch.attempt", run.AttemptID)
	out.Field("review_dispatch.reason", run.Reason)
	out.Field("review_dispatch.limit_of", run.LimitOf)
	out.Field("review_dispatch.partial", fmt.Sprint(run.Status == posemodel.ReviewRunInterrupted))
	out.Field("review_dispatch.run", run.RunID)
	out.Field("review_dispatch.status", run.Status)
	if run.Failure != "" {
		out.Field("review_dispatch.failure", run.Failure)
	}
	out.Field("review_dispatch.record", run.Path)
	out.Field("review_dispatch.attestation", "not recorded: a run's conclusion is recorded only by the engine's verified step")
	if run.Status != posemodel.ReviewRunCompleted {
		return 1
	}
	return 0
}

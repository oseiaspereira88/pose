package cli

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func cmdReviewResume(root string, args []string, stdout, stderr io.Writer) int {
	usage := "Usage: pose review resume <run-id> [--wait-until-reset[=<max-duration>]] [--apply]"
	id := ""
	apply, wait := false, false
	max := 24 * time.Hour
	for _, arg := range args {
		switch {
		case arg == "--apply":
			apply = true
		case arg == "--wait-until-reset":
			wait = true
		case strings.HasPrefix(arg, "--wait-until-reset="):
			wait = true
			var e error
			max, e = time.ParseDuration(strings.TrimPrefix(arg, "--wait-until-reset="))
			if e != nil || max <= 0 || max > 24*time.Hour {
				return usageError(stderr, usage)
			}
		case strings.HasPrefix(arg, "-") || id != "":
			return usageError(stderr, usage)
		default:
			id = arg
		}
	}
	if id == "" {
		return usageError(stderr, usage)
	}
	store := cliGovernedStore(root)
	out := render(stdout, stderr)
	plan, e := store.PlanReviewResume(id)
	if e != nil {
		out.Failure("pose review resume: " + e.Error())
		return 1
	}
	if pending := plan.PendingContinuation; pending != nil {
		out.Field("review_resume.continuation", pending.ID)
		out.Field("review_resume.attempt", pending.AttemptID)
		out.Field("review_resume.status", "pending")
		return 0
	}
	out.Field("review_resume.predecessor", id)
	out.Field("review_resume.bundle", plan.Brief.BundleID)
	if wait {
		runs, e := store.ListReviewRuns("")
		if e != nil {
			out.Failure(e.Error())
			return 1
		}
		var prior posemodel.ReviewRun
		for _, r := range runs {
			if r.RunID == id {
				prior = r
			}
		}
		var seed [1]byte
		if _, e = rand.Read(seed[:]); e != nil {
			out.Failure(e.Error())
			return 1
		}
		delay, e := posemodel.ReviewResumeDelay(prior, time.Now(), max, time.Duration(seed[0])*time.Second/255)
		if e != nil {
			out.Failure(e.Error())
			return 1
		}
		out.Field("review_resume.wait", delay.String())
		if apply && plan.Existing == nil {
			time.Sleep(delay)
			plan, e = store.PlanReviewResume(id)
			if e != nil {
				out.Failure(e.Error())
				return 1
			}
		}
	}
	if !apply {
		out.Field("review_resume.apply", "false")
		return 0
	}
	run, e := store.RunReviewDispatch(plan, time.Now)
	if e != nil {
		out.Failure("pose review resume: " + e.Error())
		return 1
	}
	out.Field("review_resume.run", run.RunID)
	out.Field("review_resume.attempt", run.AttemptID)
	out.Field("review_resume.status", run.Status)
	if run.Status != posemodel.ReviewRunCompleted {
		return 1
	}
	return 0
}
func cmdReviewStatus(root string, args []string, stdout, stderr io.Writer) int {
	jsonOut := false
	if len(args) > 0 {
		if len(args) != 1 || args[0] != "--json" {
			return usageError(stderr, "Usage: pose review status [--json]")
		}
		jsonOut = true
	}
	rows, lost := (posemodel.Store{Root: root}).ReviewInterruptedRuns()
	type statusRow struct {
		posemodel.ReviewRunSummary
		TimeLeft string `json:"time_left"`
	}
	statusRows := []statusRow{}
	for _, r := range rows {
		left := "unknown"
		if r.Interruption.ResetAt != "" {
			t, _ := time.Parse(time.RFC3339Nano, r.Interruption.ResetAt)
			d := time.Until(t)
			if d < 0 {
				d = 0
			}
			left = d.Round(time.Second).String()
		}
		statusRows = append(statusRows, statusRow{r, left})
	}
	if jsonOut {
		if e := json.NewEncoder(stdout).Encode(struct {
			Runs   []statusRow `json:"interrupted_runs"`
			Losses []string    `json:"losses"`
		}{statusRows, lost}); e != nil {
			return 1
		}
	} else {
		out := render(stdout, stderr)
		for _, r := range rows {
			reset, left := "unknown", "unknown"
			if r.Interruption.ResetAt != "" {
				reset = r.Interruption.ResetAt
				t, _ := time.Parse(time.RFC3339Nano, reset)
				d := time.Until(t)
				if d < 0 {
					d = 0
				}
				left = d.Round(time.Second).String()
			}
			out.Field("review_status."+r.RunID, fmt.Sprintf("attempt=%s adapter=%s cause=%s resumable=%t reset_at=%s time_left=%s partial=true", r.AttemptID, r.Adapter, r.Interruption.Cause, r.Interruption.Resumable, reset, left))
		}
		for _, loss := range lost {
			out.Failure("review history incomplete: " + loss)
		}
	}
	if len(lost) > 0 {
		return 1
	}
	return 0
}
func diagnoseReviewContinuity(root string, add func(string, string, string, string)) {
	rows, lost := (posemodel.Store{Root: root}).PendingReviewInterruptions()
	for _, r := range rows {
		add("review.interrupted", "warn", r.RunID+" is interrupted ("+r.Interruption.Cause+") and not resumed", "pose review status; inspect before pose review resume "+r.RunID)
	}
	for _, loss := range lost {
		add("review.history", "warn", loss, "inspect immutable delegated review evidence")
	}
}

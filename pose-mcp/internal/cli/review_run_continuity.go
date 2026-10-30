package cli

import (
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func diagnoseReviewContinuity(root string, add func(string, string, string, string)) {
	rows, lost := (posemodel.Store{Root: root}).PendingReviewInterruptions()
	for _, r := range rows {
		add("review.interrupted", "warn", r.RunID+" is interrupted ("+r.Interruption.Cause+") and not resumed", "pose delegate ls; inspect before pose delegate resume "+r.RunID)
	}
	for _, loss := range lost {
		add("review.history", "warn", loss, "inspect immutable delegated review evidence")
	}
}

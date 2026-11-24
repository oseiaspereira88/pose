package pose

// The review run shim (spec pose-review-run-reaps-lingering-reviewer-processes,
// review rounds 6 and 8): a descendant of the reviewer that leaves the
// adapter's process group, starts its own session and clears its environment
// matches neither the group kill nor the sealed-checkout entry. Making the
// dispatcher a child subreaper caught it, but a subreaper is process-wide:
// in the dispatching process it also adopted and ended children other code
// started there (a detached delegation), and reaped processes another
// goroutine was waiting for. The dispatcher therefore starts the adapter
// through a shim, the same executable re-entered with ReviewShimArg, which
// makes itself the subreaper, runs the adapter, and, once it exits, ends
// every process left in its subtree. The shim has no other children, so
// everything it adopts is the run's.

// ReviewShimArg is the first argument that runs the executable as a review
// run shim instead of the CLI.
const ReviewShimArg = "__pose-review-run-shim"

// ReviewShimReportEnv names the file the shim writes its report to.
const ReviewShimReportEnv = "POSE_REVIEW_SHIM_REPORT"

// ReviewShimAvailable says the running executable handles ReviewShimArg. It
// is set by the pose command and by the test binaries that dispatch; any
// other embedder starts the adapter directly, without the shim.
var ReviewShimAvailable = false

// reviewShimReport is what the shim found after the adapter exited.
type reviewShimReport struct {
	Ended []string `json:"ended"`
	Error string   `json:"error,omitempty"`
}
